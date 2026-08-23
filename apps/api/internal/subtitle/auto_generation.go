package subtitle

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/vido/api/internal/models"
)

// AutoGenerationMaxPerRun bounds one auto-trigger round.
//
// Deliberately a package constant rather than a per-library setting, the
// PipelineConcurrencyM1 precedent: the bound exists to stop a first scan of a
// large library from spending an evening of NAS CPU in one burst, and an
// operator-tunable knob would let someone remove the very bound it exists to
// provide. Items beyond the cap are not lost — the next scan re-enumerates
// them, and the P5 pre-flight makes re-enumeration cheap for anything already
// finished.
const AutoGenerationMaxPerRun = 20

// AutoGenerationItemTimeout bounds ONE item of a free-lane round
// (bugfix-autogenerator-no-timeout-or-shutdown D2).
//
// Per item, not per round: a round is already bounded by maxPerRun × this, and
// a round-level deadline would fail item 18 because items 1–17 were slow —
// writing a `failed` row on a file that did nothing wrong. The value is
// defaultExtractTimeout (10 min) + the ffprobe timeout (10 s) + slack for the
// DB / OpenCC / placement steps, so on the common path the subprocess deadlines
// still fire first and this one only catches the NON-subprocess hang that
// nothing else bounds — with one gap: failItem's cleanup writes run on
// context.WithoutCancel (process_item.go), so a database that locks up INSIDE
// that cleanup (the 2026-08-01 FUSE incident class) is bounded by nothing here;
// Stop() then waits on it until the container's stop grace period expires.
const AutoGenerationItemTimeout = 15 * time.Minute

// autoFailureAttemptLimit bounds how many times the free lane retries one item
// that fails outright.
//
// 補審 M1. The CR H1 exclusion covers items parked at the PAID threshold — but
// an item that FAILS (unreadable container, ffprobe absent, a share that went
// away mid-extract) writes a `failed` run and puts the media row back at
// `not_searched` (process_item.go failItem), so it is re-enumerated,
// re-attempted and re-recorded on EVERY scan, forever. That is H1's starvation
// with a broken file in place of a paid one: twenty such files at the head of
// the alphabet spend the whole per-run budget, and the free items behind them
// are never reached.
//
// Three, not one: a single failure is very often the NAS rather than the file
// (a sleeping disk, a share that reconnected a second late), and parking an
// item on the first transient error would be its own silent bug.
//
// Parking is not a verdict on the item. The manual paths — POST
// /subtitles/pipeline/run and the generate-subtitles list — still process it,
// and a run that finally delivers zh-Hant removes it from this enumeration
// altogether.
const autoFailureAttemptLimit = 3

// autoEligibleStatuses is the status set the trigger will act on.
//
// Everything else is deliberately excluded: `found` is done, `skipped` and
// `no_text_source` are terminal verdicts the pipeline already reached, and the
// in-flight states (`searching`/`probing`/`extracting`/`translating`) belong to
// a run that is either still going or died mid-way — re-entering those from a
// background trigger would race a live run.
var autoEligibleStatuses = map[models.SubtitleStatus]struct{}{
	models.SubtitleStatusNotSearched:  {},
	models.SubtitleStatusNotFound:     {},
	models.SubtitleStatusUntranslated: {},
}

// AutoLibraryPolicy answers "which libraries opted in to free auto-generation".
//
// A SET rather than a per-id predicate (Rule 11, narrow but not chatty): the
// trigger needs the whole answer once per round, and asking per item would
// issue one query per candidate on a NAS-grade SQLite file.
type AutoLibraryPolicy interface {
	AutoSubtitleLibraryIDs(ctx context.Context) (map[string]struct{}, error)
}

// AutoDeferredRunLister reads back the runs the free lane must not spend this
// round's budget on, so the trigger can stop re-probing them.
//
// WHY THIS PORT EXISTS (CR H1). The pipeline's pre-flight gate is "does an
// acceptable sidecar already exist" — and a deferred item never gets one. Left
// alone, a library whose alphabetically-first items all need paid work would
// spend its ENTIRE per-run budget re-extracting those same items on every
// scan, forever, while the free items further down the list were never
// reached: the feature would burn CPU and appear to do nothing. Excluding
// already-deferred items is what makes the budget move down the list.
//
// It reads TWO statuses, not one (補審 M1): `skipped` carries the deferral
// marker, and `failed` carries the item that cannot be processed at all. Both
// re-occupy the budget on every scan for exactly the same reason — neither
// leaves a sidecar behind for the pre-flight to find.
//
// *repository.SubtitleRunRepository satisfies it.
type AutoDeferredRunLister interface {
	ListByStatus(ctx context.Context, status models.SubtitleRunStatus, limit int) ([]models.SubtitleRun, error)
}

// AutoSeriesLibraryResolver resolves an episode's owning library.
//
// It exists because `episodes` has NO library_id column — only `movies` and
// `series` carry one. An episode's opt-in therefore has to be read through its
// parent series, and an episode whose series cannot be resolved is SKIPPED
// rather than guessed: opting in is a spending-adjacent decision, and inferring
// consent from a missing row is exactly the class of assumption the 2026-08-07
// incident was made of.
type AutoSeriesLibraryResolver interface {
	FindByID(ctx context.Context, id string) (*models.Series, error)
}

// AutoGenerator runs the free lane of the pipeline over items that arrived
// missing zh-Hant subtitles, for libraries whose owner opted in.
//
// WHAT IT IS NOT: this is not the library-wide sweep that caused the 2026-08-07
// incident. Every item is processed with ProcessItemOptions.FreeOnly, so the
// two paid routes stop at the threshold and are left for the consent screen.
// `internal/cost_consent_test.go` still holds — WorkerPool.EnqueueMissing has
// no production caller, and this type does not call it.
type AutoGenerator struct {
	item      ItemProcessor
	policy    AutoLibraryPolicy
	movies    MovieGenerationFinder
	episodes  EpisodeGenerationFinder
	series    AutoSeriesLibraryResolver
	runs      AutoDeferredRunLister
	logger    *slog.Logger
	maxPerRun int

	// itemTimeout is the per-item deadline (AutoGenerationItemTimeout unless a
	// test overrides it).
	itemTimeout time.Duration

	// lifetime is the parent of every round's ctx; Stop cancels it. Owned here
	// rather than injected (D5) so main.go's wiring is one Stop() call.
	lifetime context.Context
	cancel   context.CancelFunc
	// wg counts in-flight round goroutines so Stop can drain them — the
	// WorkerPool.Stop shape. wg.Add happens under mu, in the same critical
	// section as the stopped check, or Stop could slip between the two.
	wg      sync.WaitGroup
	stopped bool

	// running is the single-flight guard (CR M1). Every sibling in this
	// codebase has one — GenerationCandidateService returns ErrAnalysisRunning,
	// GenerationBatchProcessor exposes IsRunning — and this one needs it for the
	// same reason: a manual scan finishing next to a scheduled one would put two
	// rounds over the same item list, racing two ProcessItem calls onto one
	// sidecar path.
	mu      sync.Mutex
	running bool
	// pending records a trigger that landed while a round was in flight
	// (bugfix-autogenerator-dropped-round-not-deferred). A FLAG, not a
	// counter: every round re-enumerates the whole eligible set, so one
	// follow-up covers any number of merged triggers. Without it the trigger
	// was dropped — and scanner_service.go only fires scan-complete when a
	// scan created or updated files, so "the next scan picks it up" could be
	// days away on a quiet library.
	pending bool
}

// AutoGeneratorOption injects one optional port.
type AutoGeneratorOption func(*AutoGenerator)

// WithAutoCandidateFinders supplies the enumeration ports. Both are optional:
// a nil finder contributes no candidates rather than failing the round.
func WithAutoCandidateFinders(movies MovieGenerationFinder, episodes EpisodeGenerationFinder) AutoGeneratorOption {
	return func(g *AutoGenerator) { g.movies, g.episodes = movies, episodes }
}

// WithAutoSeriesResolver supplies the episode→library bridge. Without it,
// episodes are skipped entirely (their opt-in cannot be established).
func WithAutoSeriesResolver(series AutoSeriesLibraryResolver) AutoGeneratorOption {
	return func(g *AutoGenerator) { g.series = series }
}

// WithAutoDeferredRuns supplies the deferred-run reader (CR H1). Without it the
// trigger still works, but re-probes previously deferred items on every scan.
func WithAutoDeferredRuns(runs AutoDeferredRunLister) AutoGeneratorOption {
	return func(g *AutoGenerator) { g.runs = runs }
}

// WithAutoMaxPerRun overrides the per-round cap. Tests use it; production does
// not.
func WithAutoMaxPerRun(n int) AutoGeneratorOption {
	return func(g *AutoGenerator) {
		if n > 0 {
			g.maxPerRun = n
		}
	}
}

// WithAutoItemTimeout overrides the per-item deadline. Tests use it; production
// does not. Non-positive values are ignored.
func WithAutoItemTimeout(d time.Duration) AutoGeneratorOption {
	return func(g *AutoGenerator) {
		if d > 0 {
			g.itemTimeout = d
		}
	}
}

// NewAutoGenerator builds the trigger. A nil logger falls back to the default.
func NewAutoGenerator(item ItemProcessor, policy AutoLibraryPolicy, logger *slog.Logger, opts ...AutoGeneratorOption) *AutoGenerator {
	if logger == nil {
		logger = slog.Default()
	}
	lifetime, cancel := context.WithCancel(context.Background())
	g := &AutoGenerator{
		item: item, policy: policy, logger: logger,
		maxPerRun:   AutoGenerationMaxPerRun,
		itemTimeout: AutoGenerationItemTimeout,
		lifetime:    lifetime,
		cancel:      cancel,
	}
	for _, opt := range opts {
		opt(g)
	}
	return g
}

// ScanCallback adapts Run for the scan-complete hook: it returns immediately
// and does the work on its own goroutine, the postScanEnrichment shape
// (cmd/api/main.go). The scanner's completion path must not wait on minutes of
// ffmpeg.
func (g *AutoGenerator) ScanCallback() func() {
	return func() { g.spawnRound(spawnReasonScan) }
}

// spawnRound starts one round on its own goroutine under the generator's
// lifetime ctx, or does nothing once Stop has been called (D4: a scan completing
// inside the shutdown window must not start work behind Stop's back). Reports
// whether a round was started. `reason` is logged; the two callers are the
// scan-complete hook and the deferred follow-up (bugfix-autogenerator-dropped-
// round-not-deferred).
//
// WaitGroup safety is the `mu` ordering, nothing else: the stopped check and
// the wg.Add sit in ONE critical section, and Stop sets stopped under the same
// lock BEFORE it calls wg.Wait — so an Add either happens-before the Wait or
// is refused. This matters because the Add here CAN be from zero (a follow-up
// spawned at the end of a round that was entered by a direct Run, not by a
// goroutine of ours). Do not hoist the stopped check out of the lock.
func (g *AutoGenerator) spawnRound(reason string) bool {
	g.mu.Lock()
	if g.stopped {
		g.mu.Unlock()
		g.logger.Debug("subtitle auto-generation: stopped — round not started", "reason", reason)
		return false
	}
	g.wg.Add(1)
	g.mu.Unlock()

	if reason == spawnReasonDeferred {
		g.logger.Info("subtitle auto-generation: starting deferred follow-up round", "reason", reason)
	}
	go func() {
		defer g.wg.Done()
		g.Run(g.lifetime)
	}()
	return true
}

const (
	spawnReasonScan     = "scan_complete"
	spawnReasonDeferred = "deferred_trigger"
)

// Stop cancels the in-flight round (if any) and blocks until its goroutine has
// returned, so failItem's cleanup writes land while the database is still open
// (D1). Idempotent. Must be called from main.go's shutdown block BEFORE
// db.Close().
//
// Deliberately unbounded, like WorkerPool.Stop: a timeout here would hand the
// round back to exactly the closed-DB write this method exists to prevent.
// The per-item deadline is what bounds the wait.
func (g *AutoGenerator) Stop() {
	g.mu.Lock()
	first := !g.stopped
	g.stopped = true
	g.pending = false // a queued follow-up is dropped: shutdown is not the time to start work
	g.mu.Unlock()

	// cancel is safe to call repeatedly; the wait sits OUTSIDE the lock so a
	// draining Run can still take mu for its own single-flight bookkeeping.
	g.cancel()
	g.wg.Wait()
	if first {
		g.logger.Info("subtitle auto-generation stopped")
	}
}

// Run executes one round: read the policy, enumerate, filter, process.
//
// Synchronous for the round itself — but if a trigger landed while this round
// was in flight, Run leaves ONE follow-up goroutine behind (under `lifetime`,
// never under the caller's ctx) on return; callers that need quiescence call
// Stop to drain it. spawnRound is the only production entry.
func (g *AutoGenerator) Run(ctx context.Context) {
	if g.item == nil || g.policy == nil {
		return
	}

	// A trigger whose ctx is already dead is not a trigger: it must neither
	// start a round nor queue a follow-up (a callback that raced Stop is the
	// production case — the policy read would fail and log at Error level on
	// every shutdown for no reason).
	if ctx.Err() != nil {
		g.logger.Debug("subtitle auto-generation: round cancelled before it started")
		return
	}

	// CR M1: one round at a time — a trigger landing mid-round never starts a
	// CONCURRENT pass (two rounds would race two ProcessItem calls onto one
	// sidecar path). It is queued as one follow-up round instead of dropped
	// (bugfix-autogenerator-dropped-round-not-deferred).
	//
	// Also refuses to start after Stop (review L5): spawnRound already checks
	// this, but a direct caller must not be able to run outside Stop's drain.
	g.mu.Lock()
	if g.stopped {
		g.mu.Unlock()
		g.logger.Debug("subtitle auto-generation: stopped — round refused")
		return
	}
	if g.running {
		g.pending = true
		g.mu.Unlock()
		g.logger.Debug("subtitle auto-generation: a round is already in flight — follow-up round queued")
		return
	}
	g.running = true
	g.mu.Unlock()
	defer func() {
		g.mu.Lock()
		g.running = false
		rerun := g.pending
		g.pending = false
		g.mu.Unlock()
		if rerun {
			// OUTSIDE mu: spawnRound takes the lock itself, and it refuses once
			// Stop has run (see its comment for why that ordering is the whole
			// WaitGroup-safety argument).
			g.spawnRound(spawnReasonDeferred)
		}
	}()

	// 9R-10a CR M1: a failed read is NOT an empty answer. On the NAS this is a
	// locked SQLite file, and treating that as "nobody opted in" would make the
	// feature silently stop working with nothing in the log to show for it.
	enabled, err := g.policy.AutoSubtitleLibraryIDs(ctx)
	if err != nil {
		g.logger.Error("subtitle auto-generation: cannot read library policy — round aborted", "error", err)
		return
	}
	if len(enabled) == 0 {
		// Routine, not exceptional: the feature ships OFF for every library.
		g.logger.Debug("subtitle auto-generation: no library opted in")
		return
	}

	refs, err := g.collect(ctx, enabled)
	if err != nil {
		g.logger.Error("subtitle auto-generation: enumeration failed — round aborted", "error", err)
		return
	}
	if len(refs) == 0 {
		g.logger.Debug("subtitle auto-generation: nothing eligible", "libraries", len(enabled))
		return
	}

	g.logger.Info("subtitle auto-generation started",
		"libraries", len(enabled), "considered", len(refs), "max_per_run", g.maxPerRun)

	var processed, deferredPaid, failed, remaining int
	var cancelled bool
	for i, ref := range refs {
		if cancelled {
			break
		}
		// AC #2: once cancelled, every remaining item would fail instantly and
		// write a `failed` row each — stop at the cancellation point instead.
		if ctx.Err() != nil {
			remaining = len(refs) - i
			break
		}
		// AC #4: one deadline per item, released as soon as the item returns
		// (not deferred — a deferred cancel inside the loop would hold up to
		// maxPerRun timers until the round ends).
		itemCtx, cancelItem := context.WithTimeout(ctx, g.itemTimeout)
		outcome, err := g.item.ProcessItem(itemCtx, ref, ProcessItemOptions{FreeOnly: true})
		cancelItem()
		switch {
		case err != nil && ctx.Err() != nil:
			// CR M1: the item that was mid-flight when the round was cancelled
			// is not a failure of the FILE — failItem has already marked its
			// row (CancelledRunPrefix) so it will not count toward parking, and
			// the counters here must agree. It is folded into `remaining`: it
			// was not finished, and the next round will re-enumerate it.
			remaining = len(refs) - i
			g.logger.Info("subtitle auto-generation: item cancelled mid-flight",
				"media_id", ref.ID, "media_type", ref.MediaType)
			// Nothing after this point can start (the loop head would break on
			// the same ctx); break here so `remaining` keeps counting THIS item.
			cancelled = true
		case err != nil:
			// Per-item failure is already recorded on the item's run row by the
			// pipeline; stopping the round here would strand every later item
			// behind one unreadable file.
			failed++
			g.logger.Warn("subtitle auto-generation: item failed",
				"media_id", ref.ID, "media_type", ref.MediaType, "error", err)
		case isDeferredOutcome(outcome):
			// CR L2: keyed on the run row's deferral marker, not on RouteKind.
			// Kind alone is only unambiguous because FreeOnly is always true
			// here — a SUCCESSFUL translate run carries RouteTranslate too.
			deferredPaid++
		default:
			processed++
		}
	}

	if ctx.Err() != nil {
		g.logger.Warn("subtitle auto-generation cancelled",
			"libraries", len(enabled), "considered", len(refs),
			"processed", processed, "deferred_paid", deferredPaid, "failed", failed,
			"remaining", remaining)
		return
	}
	g.logger.Info("subtitle auto-generation finished",
		"libraries", len(enabled), "considered", len(refs),
		"processed", processed, "deferred_paid", deferredPaid, "failed", failed)
}

// collect enumerates both media families and applies the three filters:
// library opt-in, eligible status, and the per-round cap. The cap spans both
// families — episodes do not get a fresh budget after movies have spent it.
func (g *AutoGenerator) collect(ctx context.Context, enabled map[string]struct{}) ([]MediaRef, error) {
	refs := make([]MediaRef, 0, g.maxPerRun)

	excluded, err := g.excludedMediaIDs(ctx)
	if err != nil {
		return nil, err
	}

	if g.movies != nil {
		movies, err := g.movies.FindMissingZhHantSubtitle(ctx)
		if err != nil {
			return nil, err
		}
		for _, m := range movies {
			if len(refs) >= g.maxPerRun {
				return refs, nil
			}
			if _, skip := excluded[m.ID]; skip {
				continue
			}
			if !autoEligible(m.SubtitleStatus) || !inEnabledLibrary(m.LibraryID, enabled) {
				continue
			}
			refs = append(refs, MediaRef{ID: m.ID, MediaType: models.SubtitleRunMediaMovie})
		}
	}

	if g.episodes != nil && g.series != nil {
		episodes, err := g.episodes.FindMissingZhHantSubtitle(ctx)
		if err != nil {
			return nil, err
		}
		// One lookup per distinct series, not per episode: a season of 24
		// episodes shares one parent row.
		libraryOf := map[string]models.NullString{}
		for _, e := range episodes {
			if len(refs) >= g.maxPerRun {
				return refs, nil
			}
			if _, skip := excluded[e.ID]; skip {
				continue
			}
			if !autoEligible(e.SubtitleStatus) {
				continue
			}
			libraryID, ok := libraryOf[e.SeriesID]
			if !ok {
				series, err := g.series.FindByID(ctx, e.SeriesID)
				switch {
				case isSeriesNotFound(err):
					// CR H2: an ORPHAN episode is skipped, not fatal. The real
					// SeriesRepository.FindByID returns an ERROR wrapping
					// sql.ErrNoRows for a missing row — it does not return
					// (nil, nil) — so treating every error as fatal meant one
					// episode whose parent row is gone aborted the whole round,
					// discarding the movies already collected, on every scan.
					// A genuine failure (locked DB) still aborts, below.
					libraryID = models.NullString{}
				case err != nil:
					return nil, err
				case series != nil:
					libraryID = series.LibraryID
				}
				libraryOf[e.SeriesID] = libraryID
			}
			if !inEnabledLibrary(libraryID, enabled) {
				continue
			}
			refs = append(refs, MediaRef{ID: e.ID, MediaType: models.SubtitleRunMediaEpisode})
		}
	}

	return refs, nil
}

func autoEligible(status models.SubtitleStatus) bool {
	_, ok := autoEligibleStatuses[status]
	return ok
}

// inEnabledLibrary is false for a NULL library_id: an item that belongs to no
// library has no owner who could have opted it in.
func inEnabledLibrary(libraryID models.NullString, enabled map[string]struct{}) bool {
	if !libraryID.Valid || libraryID.String == "" {
		return false
	}
	_, ok := enabled[libraryID.String]
	return ok
}

// excludedMediaIDs returns the media ids this round must not spend budget on:
// items parked at the threshold of paid work (CR H1), and items that have
// failed outright autoFailureAttemptLimit times (補審 M1).
//
// TWO queries per round, not one per item. The port is optional: without it the
// trigger still runs correctly, it just re-probes parked items — so a boot that
// does not wire it degrades in cost, never in correctness.
func (g *AutoGenerator) excludedMediaIDs(ctx context.Context) (map[string]struct{}, error) {
	if g.runs == nil {
		return map[string]struct{}{}, nil
	}
	out := map[string]struct{}{}

	skipped, err := g.runs.ListByStatus(ctx, models.SubtitleRunSkipped, 0)
	if err != nil {
		return nil, err
	}
	// ListByStatus is newest-first (`ORDER BY started_at DESC`), so the FIRST
	// row seen for a media id is its latest skipped run — the one that decides
	// whether the item is currently parked awaiting consent.
	//
	// 補審 M3: every id seen is recorded, not just the deferred ones. Keyed on
	// the deferred ids alone the guard was inert — a media id whose LATEST
	// skipped run reached a different verdict would still be excluded by any
	// older deferral, which is not what this comment claimed and not what the
	// exclusion is for.
	decided := make(map[string]struct{}, len(skipped))
	for _, r := range skipped {
		if _, seen := decided[r.MediaID]; seen {
			continue
		}
		decided[r.MediaID] = struct{}{}
		if strings.HasPrefix(r.ErrorMessage, DeferredPaidRunPrefix) {
			out[r.MediaID] = struct{}{}
		}
	}

	failed, err := g.runs.ListByStatus(ctx, models.SubtitleRunFailed, 0)
	if err != nil {
		return nil, err
	}
	// Failures are COUNTED rather than latest-wins: what parks an item is a
	// RUN of bad attempts, not one bad day. An item still enumerable here has
	// by definition never had a run that delivered its zh-Hant subtitle, so
	// every failed row it carries is a free-lane attempt that came to nothing.
	//
	// EXCEPT a row written under CALLER cancellation (CancelledRunPrefix,
	// bugfix-autogenerator-no-timeout-or-shutdown AC #5) — a shutdown, a pool
	// stop, or a user cancelling a consent batch mid-item. None of those says
	// anything about the file, and counting them would let three restarts park
	// an innocent item for good. (Flip side, review M3: the same long item can
	// be re-selected first after every restart and never count — tracked with
	// bugfix-auto-exclusion-never-expires.)
	// A per-item DeadlineExceeded is NOT exempt — a file that takes longer than
	// AutoGenerationItemTimeout is exactly what the limit is for.
	attempts := make(map[string]int, len(failed))
	for _, r := range failed {
		if strings.HasPrefix(r.ErrorMessage, CancelledRunPrefix) {
			continue
		}
		attempts[r.MediaID]++
		if attempts[r.MediaID] >= autoFailureAttemptLimit {
			out[r.MediaID] = struct{}{}
		}
	}

	return out, nil
}

// isDeferredOutcome reports whether one ProcessItem call ended at the paid
// threshold, read from the run row's marker rather than inferred from the route.
func isDeferredOutcome(outcome *ProcessOutcome) bool {
	return outcome != nil && outcome.Run != nil &&
		strings.HasPrefix(outcome.Run.ErrorMessage, DeferredPaidRunPrefix)
}

// isSeriesNotFound recognises the "parent series row is gone" case across both
// shapes the repositories use: SeriesRepository wraps sql.ErrNoRows, while the
// episode repository (9R-10a CR M1) uses a typed sentinel. Matching both keeps
// this correct if the series repo later grows a sentinel of its own.
func isSeriesNotFound(err error) bool {
	return err != nil && errors.Is(err, sql.ErrNoRows)
}
