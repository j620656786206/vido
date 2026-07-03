package services

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/vido/api/internal/qbittorrent"
	"github.com/vido/api/internal/sse"
)

// defaultDownloadProgressInterval is the gated poll cadence. ~2s balances Epic 14's "<1s-ish"
// freshness target against qBittorrent rate-friendliness for a single-user NAS (ux3-4-2b decision
// #2; tunable). No settings key is added yet — the ClientCount gate already removes idle load, so a
// per-instance override is YAGNI until felt latency proves otherwise.
const defaultDownloadProgressInterval = 2 * time.Second

// progressSink is the NARROW hub interface the broadcaster depends on (Rule 11 / AC6): only
// Broadcast + ClientCount, so a unit test injects a fake with no real hub/HTTP. *sse.Hub satisfies it.
type progressSink interface {
	Broadcast(event sse.Event)
	ClientCount() int
}

// Compile-time proof the real hub satisfies the narrow sink (AC6).
var _ progressSink = (*sse.Hub)(nil)

// DownloadProgressBroadcasterInterface is the lifecycle contract (mirrors CacheSweepSchedulerInterface).
type DownloadProgressBroadcasterInterface interface {
	Start(ctx context.Context)
	Stop()
}

// DownloadProgressBroadcaster polls qBittorrent on a ticker and fans the snapshot out to every
// connected SSE client as a single download_progress event — moving polling from N browsers to ONE
// gated server poll (Epic 14 H-1 / P3-012 / ux3-4-2b). The poll is GATED on connected clients: when
// ClientCount()==0 nobody is on the Downloads page, so qBittorrent is not touched at all (Rule 14
// bounded work; this is why the pattern REMOVES idle load rather than relocating it). qBittorrent
// exposes no live-progress webhook, so a server-side poll is the only source — doing it once, gated,
// is the win.
//
// Lifecycle mirrors CacheSweepScheduler: Start(ctx) blocks in a ticker loop honoring BOTH ctx.Done()
// and Stop(); Stop() is idempotent; defer ticker.Stop(). A poll error is logged and the loop
// CONTINUES (Rule 13 — never panics the goroutine, never broadcasts a stale/empty snapshot on
// error); an expected *qbittorrent.ConnectionError while qBT is unconfigured/unreachable is logged at
// DEBUG, not spammed at ERROR.
type DownloadProgressBroadcaster struct {
	downloadSvc DownloadServiceInterface
	sink        progressSink
	interval    time.Duration

	// lastPollErr dedups the WARN log for a persistent unexpected poll error: at the ~2s cadence an
	// error that recurs every tick would otherwise emit ~30 WARN/min. It is touched ONLY by the single
	// run goroutine's tick() (never by Stop), so it needs no lock. Reset to "" on a successful poll.
	lastPollErr string

	mu      sync.Mutex
	stopCh  chan struct{}
	stopped bool
}

// Compile-time interface verification.
var _ DownloadProgressBroadcasterInterface = (*DownloadProgressBroadcaster)(nil)

// NewDownloadProgressBroadcaster constructs a broadcaster with the default ~2s cadence.
func NewDownloadProgressBroadcaster(downloadSvc DownloadServiceInterface, sink progressSink) *DownloadProgressBroadcaster {
	return &DownloadProgressBroadcaster{
		downloadSvc: downloadSvc,
		sink:        sink,
		interval:    defaultDownloadProgressInterval,
		stopCh:      make(chan struct{}),
	}
}

// Start runs the gated poll→broadcast loop until ctx is cancelled or Stop() is called. It blocks;
// callers run it in a dedicated goroutine (main.go).
func (b *DownloadProgressBroadcaster) Start(ctx context.Context) {
	slog.Info("Download progress broadcaster started", "interval", b.interval)
	b.run(ctx, b.interval)
}

// run owns the ticker loop, split out from Start so unit tests drive it directly with a short
// injected interval. Unlike CacheSweepScheduler there is deliberately NO cold-start poll: at boot
// ClientCount()==0 so the gate would skip it anyway, and the first useful push is only after a client
// connects — by then the client's own initial GET /downloads has already seeded the page, so SSE only
// needs to push subsequent deltas.
//
// Like the sibling CacheSweepScheduler this assumes a SINGLE Start per instance (main.go wires exactly
// one `go Start(ctx)` + one Stop()), so it carries no restart/double-start guard — precedent-aligned.
func (b *DownloadProgressBroadcaster) run(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("Download progress broadcaster stopped (context cancelled)")
			return
		case <-b.stopCh:
			slog.Info("Download progress broadcaster stopped (stop signal)")
			return
		case <-ticker.C:
			b.tick(ctx)
		}
	}
}

// tick performs one gated poll→broadcast. When no clients are connected it skips the qBittorrent call
// entirely (the gate — AC3 / Rule 14). A poll error is logged and swallowed so the loop survives
// (Rule 13); an expected ConnectionError (qBT unconfigured/unreachable) or a shutdown-race context
// error is logged quietly at DEBUG so a fresh/idle NAS never emits false ERROR/WARN spam.
func (b *DownloadProgressBroadcaster) tick(ctx context.Context) {
	if b.sink.ClientCount() == 0 {
		return // nobody watching → no poll → no broadcast (removes idle qBT load)
	}

	// "all" + added_on/desc mirrors the GET /downloads defaults, so the SSE snapshot carries the same
	// ordering and item shape the page first loaded with.
	torrents, err := b.downloadSvc.GetAllDownloads(ctx, "all", "added_on", "desc")
	if err != nil {
		var connErr *qbittorrent.ConnectionError
		switch {
		case errors.As(err, &connErr):
			// qBT unconfigured/unreachable is an expected steady state on a fresh NAS — DEBUG, not spam.
			slog.Debug("Download progress poll skipped (qBittorrent unavailable)", "error", err)
		case errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded):
			// A tick racing graceful shutdown; expected, not a real failure.
			slog.Debug("Download progress poll interrupted by context", "error", err)
		case err.Error() == b.lastPollErr:
			// Same unexpected error as the previous tick — throttle to DEBUG so a persistent fault does
			// not emit ~30 WARN/min at the ~2s cadence (the first occurrence already WARNed below).
			slog.Debug("Download progress poll still failing (repeat)", "error", err)
		default:
			slog.Warn("Download progress poll failed", "error", err)
			b.lastPollErr = err.Error()
		}
		return
	}
	// A successful poll clears the throttle so the NEXT distinct failure WARNs again.
	b.lastPollErr = ""

	// AC2 [@contract-v1]: broadcast the raw []qbittorrent.Torrent snapshot. Its JSON tags are already
	// snake_case (hash, name, progress, download_speed, upload_speed, eta, size, status, added_on, …) —
	// the SAME per-item keys GET /downloads exposes (handlers.DownloadItem embeds this Torrent), so the
	// FE (ux3-4-3) applies its usual snakeToCamel and reads one item shape.
	//
	// download_progress DELIBERATELY DIVERGES from GET /downloads in three ways the FE consumer MUST
	// reconcile when it setQueryData (spelled out here + in AC2 so the ux3-4-3 ack is meaningful):
	//   1. NO parse_status — that is a handler-layer enrichment (needs parseQueueSvc; importing the
	//      handlers pkg here would be an import cycle). Left ABSENT (omitempty); the FE must MERGE, not
	//      replace, so completed-download parse badges survive the ~2s push.
	//   2. BARE ARRAY, not the paginated envelope — GET returns {items, page, pageSize, totalItems,
	//      totalPages}; this event's Data is just the item array. The FE maps it into .items.
	//   3. FULL, UNPAGINATED list — every torrent, not one page. The FE must not let a push overwrite
	//      the current page's window.
	// Normalize nil→empty so the payload is ALWAYS a JSON array (never null), matching the array contract.
	if torrents == nil {
		torrents = []qbittorrent.Torrent{}
	}
	b.sink.Broadcast(sse.Event{
		ID:   uuid.New().String(),
		Type: sse.EventDownloadProgress,
		Data: torrents,
	})
}

// Stop gracefully stops the broadcaster. It is idempotent — safe to call when never started or when
// called twice (mirrors CacheSweepScheduler.Stop).
func (b *DownloadProgressBroadcaster) Stop() {
	b.mu.Lock()
	defer b.mu.Unlock()

	if !b.stopped {
		b.stopped = true
		close(b.stopCh)
	}
}
