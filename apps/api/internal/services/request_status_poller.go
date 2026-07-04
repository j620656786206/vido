// Package services — RequestStatusPoller (Story 13-3a, G-3/P3-003).
//
// The always-on reconciler that makes each request row track reality:
// 搜尋中 → 下載中 x% → 已入庫 / 失敗. Clones the DownloadProgressBroadcaster
// anatomy with ONE recorded deviation: the ClientCount()==0 gate moves from
// the POLL to the BROADCAST — this loop has SSE-independent server duties
// (DB status truth, scan triggering, pending retry, the 13-5 completed seam),
// so it always reconciles; the cheap idle-gate is ListActive()==0 instead.
// Status derivation semantics + the request_progress payload carry
// [@contract-v1] (13-3a AC #2/#4 — consumers 13-3b/13-5/13-2a).
package services

import (
	"context"
	"errors"
	"log/slog"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/plugins"
	"github.com/vido/api/internal/qbittorrent"
	"github.com/vido/api/internal/repository"
	"github.com/vido/api/internal/sse"
)

// defaultRequestStatusInterval comfortably beats the <30s freshness criterion;
// every source is LAN/local and the idle-gate removes all cost on a quiet NAS.
const defaultRequestStatusInterval = 15 * time.Second

// requestScanDebounce bounds poller-initiated library scans: a burst of
// completions inside one window shares a single scan (AC #3).
const requestScanDebounce = 2 * time.Minute

// requestQueueSource is the NARROW plugins.Manager surface the poller needs
// (registered names, health gate, client access). *plugins.Manager satisfies it.
type requestQueueSource interface {
	RegisteredPlugins() []string
	Health(name string) plugins.PluginHealth
	GetClient(ctx context.Context, name string) (plugins.DVRPlugin, error)
}

// torrentSource is the one DownloadService method the qBT progress-refinement
// join needs — narrower than DownloadServiceInterface so the unit fake stays
// one method (the broadcaster's wide-interface choice predates this need).
type torrentSource interface {
	GetAllDownloads(ctx context.Context, filter string, sortField string, order string) ([]qbittorrent.Torrent, error)
}

// scanTrigger is the one ScannerService method the import-window trigger needs.
type scanTrigger interface {
	StartScan(ctx context.Context) (*ScanResult, error)
}

// RequestStatusPollerInterface is the lifecycle contract (broadcaster mirror).
type RequestStatusPollerInterface interface {
	Start(ctx context.Context)
	Stop()
}

// queueDerivedState is the qBT/queue evidence verdict for one request row.
type queueDerivedState int

const (
	queueStateDownloading queueDerivedState = iota
	queueStateImportWindow
	queueStateFailed
)

// mapTorrentToQueueState maps a joined qBT torrent status onto the request
// derivation (AC #2 table; MapQBState style — smaller vocabulary):
// downloading/queued/checking/stalled → downloading; completed/seeding/paused
// → import window (the *arr import hasn't landed the file yet); error → failed.
func mapTorrentToQueueState(status qbittorrent.TorrentStatus) queueDerivedState {
	switch status {
	case qbittorrent.StatusError:
		return queueStateFailed
	case qbittorrent.StatusCompleted, qbittorrent.StatusSeeding, qbittorrent.StatusPaused:
		return queueStateImportWindow
	default:
		return queueStateDownloading
	}
}

// requestProgressItem is one request_progress payload element: the 13-1a
// request resource verbatim + the ephemeral progress (0–1, present only when
// meaningful — never persisted; AC #4 [@contract-v1]).
type requestProgressItem struct {
	models.Request
	Progress *float64 `json:"progress,omitempty"`
}

// queueEvidence is one plugin's queue snapshot for a tick; ok=false means the
// source was unavailable and rules 2–4 must skip queue evidence (fail-soft).
type queueEvidence struct {
	items map[int64]plugins.QueueItem
	ok    bool
}

// RequestStatusPoller reconciles active request rows against library
// ownership, *arr queues and qBittorrent every tick, persists transitions,
// triggers the import-window scan, retries stranded pending rows, and fans a
// request_progress snapshot out to connected SSE clients.
type RequestStatusPoller struct {
	repo         repository.RequestRepositoryInterface
	availability AvailabilityServiceInterface
	queues       requestQueueSource
	torrents     torrentSource
	scanner      scanTrigger
	fulfilment   FulfilmentServiceInterface // nil-safe (13-4a semantics)
	sink         progressSink
	interval     time.Duration
	now          func() time.Time // injectable clock for the debounce tests

	// OnRequestCompleted is the 13-5 seam: invoked exactly once per request
	// transition INTO completed (idempotence lives on the transition edge —
	// a completed row leaves ListActive, so re-ticks cannot re-fire). Nil-safe;
	// THIS story wires nothing into it (capability-honor, AC #6).
	OnRequestCompleted func(ctx context.Context, req models.Request)

	// Tick-goroutine-only state (single run loop — no lock needed, the
	// broadcaster lastPollErr precedent).
	inImportWindow  map[string]bool
	lastScanTrigger time.Time
	lastErrBySource map[string]string

	mu      sync.Mutex
	stopCh  chan struct{}
	stopped bool
}

// Compile-time interface verification.
var _ RequestStatusPollerInterface = (*RequestStatusPoller)(nil)

// NewRequestStatusPoller constructs the reconciler with the default 15s cadence.
func NewRequestStatusPoller(
	repo repository.RequestRepositoryInterface,
	availability AvailabilityServiceInterface,
	queues requestQueueSource,
	torrents torrentSource,
	scanner scanTrigger,
	fulfilment FulfilmentServiceInterface,
	sink progressSink,
) *RequestStatusPoller {
	return &RequestStatusPoller{
		repo:            repo,
		availability:    availability,
		queues:          queues,
		torrents:        torrents,
		scanner:         scanner,
		fulfilment:      fulfilment,
		sink:            sink,
		interval:        defaultRequestStatusInterval,
		now:             time.Now,
		inImportWindow:  map[string]bool{},
		lastErrBySource: map[string]string{},
		stopCh:          make(chan struct{}),
	}
}

// Start runs the reconcile loop until ctx is cancelled or Stop() is called.
// It blocks; main.go runs it in a dedicated goroutine (broadcaster mirror).
func (p *RequestStatusPoller) Start(ctx context.Context) {
	slog.Info("Request status poller started", "interval", p.interval)
	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("Request status poller stopped (context cancelled)")
			return
		case <-p.stopCh:
			slog.Info("Request status poller stopped (stop signal)")
			return
		case <-ticker.C:
			p.tick(ctx)
		}
	}
}

// Stop gracefully stops the poller. Idempotent (broadcaster mirror).
func (p *RequestStatusPoller) Stop() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if !p.stopped {
		p.stopped = true
		close(p.stopCh)
	}
}

// tick performs one reconcile pass. The idle-gate: zero active rows → zero
// external calls (an idle NAS does no *arr/qBT/DB-join traffic). Ticks are
// independent — every source failure degrades only its own evidence column
// and the loop always survives (AC #5, Rule 13).
func (p *RequestStatusPoller) tick(ctx context.Context) {
	active, err := p.repo.ListActive(ctx)
	if err != nil {
		p.logSourceErr("list_active", "Request status poll failed to list active requests", err)
		return
	}
	p.clearSourceErr("list_active")

	if len(active) == 0 {
		// Idle gate — also reset window bookkeeping so stale entries never leak.
		if len(p.inImportWindow) > 0 {
			p.inImportWindow = map[string]bool{}
		}
		return
	}

	ownedSet, ownedOK := p.fetchOwned(ctx, active)
	queues := p.fetchQueues(ctx)
	torrents := p.fetchTorrents(ctx)

	snapshot := make([]requestProgressItem, 0, len(active))
	for i := range active {
		snapshot = append(snapshot, p.reconcile(ctx, &active[i], ownedSet, ownedOK, queues, torrents))
	}

	p.broadcast(snapshot)
}

// reconcile applies the AC #2 derivation table to one row IN ORDER and
// returns its snapshot item (with the row mutated to its post-tick state).
func (p *RequestStatusPoller) reconcile(
	ctx context.Context,
	row *models.Request,
	ownedSet map[int64]bool,
	ownedOK bool,
	queues map[string]queueEvidence,
	torrents map[string]qbittorrent.Torrent,
) requestProgressItem {
	// Rule 1 — Vido's own library is the truth for 已入庫 (terminal).
	if ownedOK && ownedSet[row.TMDbID] {
		p.completeRequest(ctx, row)
		return requestProgressItem{Request: *row}
	}

	// Rules 2–4 need an external id.
	if row.ExternalID.Valid {
		return p.reconcileExternal(ctx, row, queues, torrents)
	}

	// Rule 5 — no external id: stays pending; retry fulfilment (the absorbed
	// 13-4a handoff). FulfilRequest is nil-safe-by-absence, mutates + persists.
	if p.fulfilment != nil {
		p.fulfilment.FulfilRequest(ctx, row)
	}
	return requestProgressItem{Request: *row}
}

// reconcileExternal runs derivation rows 2–4 for a row with an external id.
func (p *RequestStatusPoller) reconcileExternal(
	ctx context.Context,
	row *models.Request,
	queues map[string]queueEvidence,
	torrents map[string]qbittorrent.Torrent,
) requestProgressItem {
	pluginName := dvrMoviePlugin
	if row.MediaType == models.RequestMediaTypeTV {
		pluginName = dvrSeriesPlugin
	}

	evidence := queues[pluginName]
	if !evidence.ok {
		// Fail-soft: no queue evidence this tick → hold whatever status the
		// row has; never regress, never fake an import window (AC #5).
		return requestProgressItem{Request: *row}
	}

	externalID, err := strconv.ParseInt(row.ExternalID.String, 10, 64)
	if err != nil {
		// A malformed external_id can only come from a bug upstream — log and
		// hold rather than guessing (defensive; not a reachable state today).
		p.logSourceErr("external_id:"+row.ID, "Request has malformed external_id", err)
		return requestProgressItem{Request: *row}
	}

	if item, found := evidence.items[externalID]; found {
		return p.reconcileQueued(ctx, row, item, torrents)
	}

	// Not in the queue.
	if row.Status == models.RequestStatusDownloading {
		// Rule 3 — import window: the download vanished but the file has not
		// landed in the library yet. HOLD downloading (no 6th enum value) and
		// trigger the debounced scan on window ENTRY (AC #3).
		p.enterImportWindow(ctx, row.ID)
		return requestProgressItem{Request: *row}
	}

	// Rule 4 — *arr is still hunting.
	if row.Status != models.RequestStatusSearching {
		p.persistStatus(ctx, row, models.RequestStatusSearching, "")
	}
	return requestProgressItem{Request: *row}
}

// reconcileQueued runs rule 2 for a row with a live queue record: derive
// downloading/failed/import-window + the ephemeral progress, refined by the
// joined qBT torrent when the hash matches.
func (p *RequestStatusPoller) reconcileQueued(
	ctx context.Context,
	row *models.Request,
	item plugins.QueueItem,
	torrents map[string]qbittorrent.Torrent,
) requestProgressItem {
	progress := queueProgress(item)
	state := queueStateDownloading

	// *arr-reported terminal failure.
	if strings.EqualFold(item.Status, "failed") {
		state = queueStateFailed
	}

	// qBT refinement by torrent hash (case-normalized).
	if torrents != nil && item.DownloadID != "" {
		if torrent, ok := torrents[strings.ToLower(item.DownloadID)]; ok {
			progress = torrent.Progress
			if mapped := mapTorrentToQueueState(torrent.Status); mapped > state {
				state = mapped
			} else if mapped == queueStateFailed {
				state = queueStateFailed
			}
		}
	}

	switch state {
	case queueStateFailed:
		p.persistStatus(ctx, row, models.RequestStatusFailed, "下載發生錯誤，請重試或檢查下載器")
		delete(p.inImportWindow, row.ID)
		return requestProgressItem{Request: *row}
	case queueStateImportWindow:
		// Finished in qBT, still listed by *arr → same hold-and-scan as a
		// vanished queue record (AC #2 mapping table → rule 3).
		p.enterImportWindow(ctx, row.ID)
		return requestProgressItem{Request: *row}
	default:
		if row.Status != models.RequestStatusDownloading {
			p.persistStatus(ctx, row, models.RequestStatusDownloading, "")
		}
		delete(p.inImportWindow, row.ID)
		return requestProgressItem{Request: *row, Progress: &progress}
	}
}

// completeRequest persists the terminal completed transition and fires the
// 13-5 seam exactly once (on the successful transition write).
func (p *RequestStatusPoller) completeRequest(ctx context.Context, row *models.Request) {
	if !p.persistStatus(ctx, row, models.RequestStatusCompleted, "") {
		return // persist failed → row keeps its old status; a later tick retries
	}
	delete(p.inImportWindow, row.ID)
	slog.Info("Request completed — media landed in library",
		"request_id", row.ID, "tmdb_id", row.TMDbID, "title", row.Title)
	if p.OnRequestCompleted != nil {
		p.OnRequestCompleted(ctx, *row)
	}
}

// persistStatus writes a transition + mirrors it onto the in-memory row so
// the snapshot carries the post-tick truth. Returns false when the write failed.
func (p *RequestStatusPoller) persistStatus(ctx context.Context, row *models.Request, status, errMsg string) bool {
	updatedAt, err := p.repo.UpdateStatus(ctx, row.ID, status, errMsg)
	if err != nil {
		slog.Error("Request status transition write failed",
			"request_id", row.ID, "from", row.Status, "to", status, "error", err)
		return false
	}
	row.Status = status
	row.UpdatedAt = updatedAt
	if errMsg == "" {
		row.ErrorMessage = models.NullString{}
	} else {
		row.ErrorMessage = models.NewNullString(errMsg)
	}
	return true
}

// enterImportWindow marks a row as waiting for the *arr import and triggers
// the debounced library scan on the ENTRY edge only.
func (p *RequestStatusPoller) enterImportWindow(ctx context.Context, id string) {
	if p.inImportWindow[id] {
		return
	}
	p.inImportWindow[id] = true
	p.maybeTriggerScan(ctx)
}

// maybeTriggerScan starts ONE library scan, debounced to at most one
// poller-initiated scan per requestScanDebounce, skipping when a scan is
// already running (AC #3 — product-critical: default installs scan manually,
// so without this the pipeline dead-ends at 下載中 100%).
func (p *RequestStatusPoller) maybeTriggerScan(ctx context.Context) {
	if p.scanner == nil {
		return
	}
	now := p.now()
	if !p.lastScanTrigger.IsZero() && now.Sub(p.lastScanTrigger) < requestScanDebounce {
		return
	}
	p.lastScanTrigger = now

	// StartScan runs the whole scan synchronously — never block the tick.
	go func() {
		if _, err := p.scanner.StartScan(ctx); err != nil {
			if strings.Contains(err.Error(), "SCANNER_ALREADY_RUNNING") {
				slog.Debug("Import-window scan skipped (scan already running)", "error", err)
				return
			}
			// Never fatal (Rule 13) — ownership catches up on a later scan.
			slog.Warn("Import-window scan failed to start", "error", err)
		}
	}()
}

// fetchOwned bulk-checks library ownership for every active row (one call per
// tick — Story 10-4 service reuse). ok=false skips completion detection.
func (p *RequestStatusPoller) fetchOwned(ctx context.Context, active []models.Request) (map[int64]bool, bool) {
	ids := make([]int64, 0, len(active))
	seen := make(map[int64]struct{}, len(active))
	for _, row := range active {
		if _, dup := seen[row.TMDbID]; dup {
			continue
		}
		seen[row.TMDbID] = struct{}{}
		ids = append(ids, row.TMDbID)
	}

	owned, err := p.availability.CheckOwned(ctx, ids)
	if err != nil {
		p.logSourceErr("check_owned", "Request status poll failed ownership check", err)
		return nil, false
	}
	p.clearSourceErr("check_owned")

	set := make(map[int64]bool, len(owned))
	for _, id := range owned {
		set[id] = true
	}
	return set, true
}

// fetchQueues collects each registered+healthy plugin's queue keyed by
// ExternalID. A per-plugin failure degrades only that plugin's evidence.
func (p *RequestStatusPoller) fetchQueues(ctx context.Context) map[string]queueEvidence {
	out := map[string]queueEvidence{}
	for _, name := range p.queues.RegisteredPlugins() {
		if p.queues.Health(name).Status != plugins.HealthStatusHealthy {
			out[name] = queueEvidence{}
			continue
		}
		client, err := p.queues.GetClient(ctx, name)
		if err != nil {
			p.logSourceErr("queue:"+name, "Request status poll failed to get plugin client", err)
			out[name] = queueEvidence{}
			continue
		}
		items, err := client.GetQueue(ctx)
		if err != nil {
			p.logSourceErr("queue:"+name, "Request status poll failed to fetch plugin queue", err)
			out[name] = queueEvidence{}
			continue
		}
		p.clearSourceErr("queue:" + name)

		byID := make(map[int64]plugins.QueueItem, len(items))
		for _, item := range items {
			byID[item.ExternalID] = item
		}
		out[name] = queueEvidence{items: byID, ok: true}
	}
	return out
}

// fetchTorrents returns the qBT torrent map keyed by lowercase hash, or nil
// when qBT is unavailable (progress refinement is then skipped — AC #5).
func (p *RequestStatusPoller) fetchTorrents(ctx context.Context) map[string]qbittorrent.Torrent {
	torrents, err := p.torrents.GetAllDownloads(ctx, "all", "added_on", "desc")
	if err != nil {
		var connErr *qbittorrent.ConnectionError
		if errors.As(err, &connErr) {
			// Unconfigured/unreachable qBT is an expected steady state.
			slog.Debug("Request status poll skipped qBT refinement (unavailable)", "error", err)
		} else {
			p.logSourceErr("qbittorrent", "Request status poll failed to fetch torrents", err)
		}
		return nil
	}
	p.clearSourceErr("qbittorrent")

	byHash := make(map[string]qbittorrent.Torrent, len(torrents))
	for _, t := range torrents {
		byHash[strings.ToLower(t.Hash)] = t
	}
	return byHash
}

// broadcast fans the snapshot out — gated on connected clients (the ONE
// deviation from the broadcaster template lives at this end, not the poll).
func (p *RequestStatusPoller) broadcast(snapshot []requestProgressItem) {
	if p.sink.ClientCount() == 0 {
		return
	}
	if snapshot == nil {
		snapshot = []requestProgressItem{} // bare array, never null (AC #4)
	}
	p.sink.Broadcast(sse.Event{
		ID:   uuid.New().String(),
		Type: sse.EventRequestProgress,
		Data: snapshot,
	})
}

// queueProgress estimates progress from the *arr queue record.
func queueProgress(item plugins.QueueItem) float64 {
	if item.Size <= 0 {
		return 0
	}
	progress := float64(item.Size-item.SizeLeft) / float64(item.Size)
	if progress < 0 {
		return 0
	}
	if progress > 1 {
		return 1
	}
	return progress
}

// logSourceErr WARNs the first occurrence of a source failure and throttles
// repeats to DEBUG (the broadcaster lastPollErr dedup, per source).
func (p *RequestStatusPoller) logSourceErr(source, message string, err error) {
	if p.lastErrBySource[source] == err.Error() {
		slog.Debug(message+" (repeat)", "source", source, "error", err)
		return
	}
	slog.Warn(message, "source", source, "error", err)
	p.lastErrBySource[source] = err.Error()
}

// clearSourceErr resets the dedup so the next distinct failure WARNs again.
func (p *RequestStatusPoller) clearSourceErr(source string) {
	delete(p.lastErrBySource, source)
}
