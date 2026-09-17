package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"path/filepath"
	"runtime/debug"
	"sync"

	"github.com/google/uuid"

	"github.com/vido/api/internal/ai"
	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/repository"
	"github.com/vido/api/internal/sse"
)

// Story 9R-16: Route C batch-generation orchestrator. Mirrors the fetch-batch
// (subtitle/batch.go) single-flight shape but lives in services because it
// drives TranscriptionService (Rule 19: services must not import subtitle; the
// orchestrator needs no subtitle-engine types). In-memory state only — resume
// is re-enumeration: the AC 12 writeback makes completed items self-exclude
// from the missing-zh-Hant scope, so 下次繼續 = start a new scope=missing batch.

// ErrGenerationBatchRunning is returned when a generation batch is already in
// progress (409 scenario — TRANSCRIPTION_BATCH_RUNNING).
var ErrGenerationBatchRunning = errors.New("generation batch already running")

// ErrGenerationSelectionInvalid marks a scope=selected media id that cannot run
// generation (unknown id, or a row without a media file). The handler maps it
// to 400 (AC 8 ruling: REJECT, not filter — sub-4-2: the consented list IS the
// amount the user confirmed on F16; silently dropping items would execute a
// batch nobody consented to). Since sub-4-2 ids may be movies OR episodes.
// A REAL lookup failure (DB error, cancelled ctx) is deliberately NOT this
// sentinel — it propagates unwrapped so the handler answers 500, not "your
// selection is invalid" (CR sub-4-2 M2).
var ErrGenerationSelectionInvalid = errors.New("selected media cannot run generation")

// ErrGenerationItemSkipped marks an item the pipeline deliberately routed out
// (skipped run — e.g. only non-target text tracks, or no text source with no
// live ASR). The batch counts it as a FAILURE, not a success: on a consented
// batch, success_count must mean "a subtitle now exists" — crediting skips
// would let a keyless deployment report N successes, $0 spent, and zero
// subtitles (CR sub-4-2 H1). Legacy mode already fails these items via
// ErrTranscriptionDisabled; this sentinel restores the same honesty for the
// pipeline engine.
var ErrGenerationItemSkipped = errors.New("generation item skipped by pipeline")

// Generation-batch terminal/live statuses (AC 9 vocabulary, [@contract-v2] —
// media ids are UUID STRINGS since 9R-18).
const (
	GenerationBatchStatusRunning       = "running"
	GenerationBatchStatusComplete      = "complete"
	GenerationBatchStatusCancelled     = "cancelled"
	GenerationBatchStatusError         = "error"
	GenerationBatchStatusBudgetCeiling = "budget_ceiling"
)

// GenerationBatchItemStatus is one queue entry's state (dsr-6d-a AC #1
// [@contract-v1]). Its own type on purpose: the batch statuses above also use
// "running" and "cancelled", and the two vocabularies must not be mixed up.
type GenerationBatchItemStatus string

const (
	GenerationBatchItemQueued    GenerationBatchItemStatus = "queued"
	GenerationBatchItemRunning   GenerationBatchItemStatus = "running"
	GenerationBatchItemDone      GenerationBatchItemStatus = "done"
	GenerationBatchItemFailed    GenerationBatchItemStatus = "failed"
	GenerationBatchItemPaused    GenerationBatchItemStatus = "paused"
	GenerationBatchItemCancelled GenerationBatchItemStatus = "cancelled"
)

// GenerationBatchItemReason says why an item failed; empty for every other
// status (dsr-6d-a AC #1 [@contract-v1]). fail_count keeps counting all three
// (sub-4-2 CR H1: success must mean a subtitle now exists).
type GenerationBatchItemReason string

const (
	GenerationBatchItemReasonNone GenerationBatchItemReason = ""
	// Skipped: the item had nothing usable to generate from — the pipeline
	// routed it out, or (legacy mode) generation is not configured. One name
	// for both modes: it is the same situation seen from two engines.
	GenerationBatchItemReasonSkipped GenerationBatchItemReason = "skipped"
	// BusyElsewhere: the media was already being processed by the pool, a
	// worker or the detail dialog.
	GenerationBatchItemReasonBusyElsewhere GenerationBatchItemReason = "busy_elsewhere"
	GenerationBatchItemReasonError         GenerationBatchItemReason = "error"
)

// GenerationBatchProgress reports the state of a generation batch (snake_case
// wire shape shared by GET .../status progress and last, the 409 body, the 202
// progress and the generation_batch_progress SSE event — 9R-16 AC 2/9).
//
// Items (dsr-6d-a AC #1) is the whole queue with each entry's state. Snapshots
// handed out by the processor always carry it; the SSE event sends it only on
// the terminal broadcast and a single changed_item while running (AC #2).
type GenerationBatchProgress struct {
	BatchID        string                     `json:"batch_id"`
	TotalItems     int                        `json:"total_items"`
	CurrentIndex   int                        `json:"current_index"`
	CurrentMediaID string                     `json:"current_media_id"`
	CurrentItem    string                     `json:"current_item"`
	SuccessCount   int                        `json:"success_count"`
	FailCount      int                        `json:"fail_count"`
	PausedCount    int                        `json:"paused_count"`
	Status         string                     `json:"status"`
	SpentUSD       float64                    `json:"spent_usd"`
	BudgetUSD      float64                    `json:"budget_usd"`
	Items          []GenerationBatchItemState `json:"items"`
}

// GenerationBatchItem is one enumerated queue entry. The exported fields are
// the 202-response items[] shape (AC 1; media_type additive since sub-4-2;
// series_title additive since dsr-6d-a AC #7 — "" for movies and whenever the
// series lookup degrades); file locations stay internal. MediaType uses the
// internal vocabulary (models.SubtitleRunMediaMovie|Episode — movie|episode,
// NOT TMDB movie|tv).
type GenerationBatchItem struct {
	MediaID     string `json:"media_id"`
	Title       string `json:"title"`
	MediaType   string `json:"media_type"`
	SeriesTitle string `json:"series_title"`

	filePath string
	mediaDir string
}

// GenerationBatchItemState is a queue entry plus its state (dsr-6d-a AC #1
// [@contract-v1]). Every key is always present — reason is "" unless failed.
type GenerationBatchItemState struct {
	GenerationBatchItem
	Status GenerationBatchItemStatus `json:"status"`
	Reason GenerationBatchItemReason `json:"reason"`
}

// GenerationRunner executes one consented batch item. Implementations are
// selected by cmd/api per pipeline mode (sub-4-2): legacy mode keeps the Route
// C transcription engine (RouteCGenerationRunner below); pipeline mode injects
// a cmd/api adapter over subtitle.Pipeline.ProcessItem so an item with an
// extractable embedded subtitle takes the free extract→translate route instead
// of unconditionally paying for ASR — the F15 quote must match what actually
// runs. mediaType carries the internal movie|episode vocabulary; the mirror of
// subtitle.MediaRef's two fields (services ↛ subtitle — see project-context.md
// Rule 19).
type GenerationRunner interface {
	IsAvailable() bool
	ExecuteGeneration(ctx context.Context, mediaID, mediaType, filePath, mediaDir string) error
}

// generationCandidateFinder is the narrow movie-repo surface for enumeration
// (AC 4) + selected-id resolution. *repository.MovieRepository satisfies it.
type generationCandidateFinder interface {
	FindMissingZhHantSubtitle(ctx context.Context) ([]models.Movie, error)
	CountMissingZhHantSubtitle(ctx context.Context) (int, error)
	FindByID(ctx context.Context, id string) (*models.Movie, error)
}

// generationEpisodeFinder is the narrow episode-repo surface for scope=selected
// id resolution (sub-4-2 D1 ruling: mixed movie+episode batches) and the
// episode half of the preview count (sub-5-1 AC #7).
// *repository.EpisodeRepository satisfies it. A nil finder degrades to the
// pre-sub-4-2 movies-only behavior.
type generationEpisodeFinder interface {
	FindByID(ctx context.Context, id string) (*models.Episode, error)
	CountMissingZhHantSubtitle(ctx context.Context) (int, error)
}

// GenerationBatchProcessor runs the Route C generation pipeline sequentially
// over an enumerated queue under ONE shared ai.Budget (AC 5/6/7). Global
// single-flight: at most one generation batch at a time (independent from the
// Epic 8 fetch-batch — they share no state).
type GenerationBatchProcessor struct {
	runner   GenerationRunner
	finder   generationCandidateFinder
	episodes generationEpisodeFinder
	sseHub   *sse.Hub
	// budgetUSD is the DEFAULT ceiling (AI_RUN_BUDGET_USD; <=0 = unlimited),
	// used only when Start receives no user-approved ceiling (sub-4-2 AC #1).
	budgetUSD float64
	logger    *slog.Logger

	// series resolves episode series titles for the queue (dsr-6d-a AC #7).
	// nil degrades every series_title to "".
	series CandidateSeriesTitleResolver

	mu           sync.Mutex
	activeBatch  *GenerationBatchProgress
	activeCancel context.CancelFunc
	activeBudget *ai.Budget
	// lastBatch is the most recent terminal snapshot (dsr-6d-a AC #3): kept in
	// memory until the next batch actually starts or DismissLast, gone on
	// restart. nil while a batch runs.
	lastBatch *GenerationBatchProgress
}

// NewGenerationBatchProcessor wires the orchestrator. budgetUSD is the default
// batch cost ceiling (cfg.AIRunBudgetUSD) applied when a Start call carries no
// user-approved ceiling. episodes may be nil in movie-only tests.
func NewGenerationBatchProcessor(
	runner GenerationRunner,
	finder generationCandidateFinder,
	episodes generationEpisodeFinder,
	sseHub *sse.Hub,
	budgetUSD float64,
	logger *slog.Logger,
) *GenerationBatchProcessor {
	if logger == nil {
		logger = slog.Default()
	}
	return &GenerationBatchProcessor{
		runner:    runner,
		finder:    finder,
		episodes:  episodes,
		sseHub:    sseHub,
		budgetUSD: budgetUSD,
		logger:    logger.With("service", "generation_batch"),
	}
}

// SetSeriesTitleResolver wires the episode series-title lookup (dsr-6d-a AC
// #7). A setter rather than a constructor parameter, the SetSSEHub /
// SetModelCatalog precedent: the title is display-only, and every existing
// constructor call site stays valid.
//
// ⚠️ Wiring only: call it during startup, before the server serves. It is the
// one field on this struct not guarded by p.mu, so setting it while a batch
// enumerates would be a data race.
func (p *GenerationBatchProcessor) SetSeriesTitleResolver(r CandidateSeriesTitleResolver) {
	p.series = r
}

// withLock runs fn under p.mu with a deferred unlock, so a panic inside can
// never leave the mutex held — the recover in process would otherwise block
// forever in finish (dsr-6d-a AC #4).
func (p *GenerationBatchProcessor) withLock(fn func()) {
	p.mu.Lock()
	defer p.mu.Unlock()
	fn()
}

// copyItems deep-copies a queue so no snapshot shares the processor's backing
// array (dsr-6d-a AC #1).
func copyItems(items []GenerationBatchItemState) []GenerationBatchItemState {
	if items == nil {
		return nil
	}
	out := make([]GenerationBatchItemState, len(items))
	copy(out, items) // the element type holds only value fields
	return out
}

// snapshotLocked copies a progress struct including its queue. withSpent
// replaces SpentUSD with the live budget figure (active batches only). Caller
// holds p.mu.
func (p *GenerationBatchProcessor) snapshotLocked(src *GenerationBatchProgress, withSpent bool) *GenerationBatchProgress {
	if src == nil {
		return nil
	}
	out := *src
	out.Items = copyItems(src.Items)
	if withSpent && p.activeBudget != nil {
		out.SpentUSD = p.activeBudget.SpentUSD()
	}
	return &out
}

// IsAvailable reports whether the underlying generation pipeline can run
// (FFmpeg + ASR configured) — the handler's 503 TRANSCRIPTION_DISABLED gate.
func (p *GenerationBatchProcessor) IsAvailable() bool {
	return p.runner != nil && p.runner.IsAvailable()
}

// IsRunning returns true if a generation batch is currently active.
func (p *GenerationBatchProcessor) IsRunning() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.activeBatch != nil
}

// GetProgress returns a deep copy of the current progress (queue included)
// with live cost figures, or nil when no batch is running.
func (p *GenerationBatchProcessor) GetProgress() *GenerationBatchProgress {
	var out *GenerationBatchProgress
	p.withLock(func() { out = p.snapshotLocked(p.activeBatch, true) })
	return out
}

// Snapshot returns the running batch (nil when idle) and the last terminal
// snapshot (nil while running or when none is kept) under ONE lock, so a
// status response can never show a batch as both running and finished
// (dsr-6d-a AC #3).
func (p *GenerationBatchProcessor) Snapshot() (progress, last *GenerationBatchProgress) {
	p.withLock(func() {
		progress = p.snapshotLocked(p.activeBatch, true)
		last = p.snapshotLocked(p.lastBatch, false)
	})
	return progress, last
}

// SnapshotFor returns the batch with this id: the running one, else the last
// terminal snapshot when a very short batch already finished, else nil — the
// batch was dismissed or replaced within milliseconds (dsr-6d-a AC #5).
func (p *GenerationBatchProcessor) SnapshotFor(batchID string) *GenerationBatchProgress {
	var out *GenerationBatchProgress
	p.withLock(func() {
		switch {
		case p.activeBatch != nil && p.activeBatch.BatchID == batchID:
			out = p.snapshotLocked(p.activeBatch, true)
		case p.lastBatch != nil && p.lastBatch.BatchID == batchID:
			out = p.snapshotLocked(p.lastBatch, false)
		}
	})
	return out
}

// DismissLast forgets the last terminal snapshot (dsr-6d-a AC #6). While a
// batch runs it clears nothing and reports running=true. Idempotent.
func (p *GenerationBatchProcessor) DismissLast() (dismissed, running bool) {
	p.withLock(func() {
		if p.activeBatch != nil {
			running = true
			return
		}
		dismissed = p.lastBatch != nil
		p.lastBatch = nil
	})
	return dismissed, running
}

// ActivityProgress reports the active batch as primitives for the /activity
// aggregate (AC 10 — mirrors subtitle.BatchProcessor.ActivityProgress so the
// ActivityService source interface is shared). active=false when idle. Reads
// the counters directly: /activity polls this, and it has no use for a copy
// of the queue (dsr-6d-a AC #1).
func (p *GenerationBatchProcessor) ActivityProgress() (active bool, percentDone, current, total int, currentItem string) {
	p.withLock(func() {
		b := p.activeBatch
		if b == nil || b.Status != GenerationBatchStatusRunning {
			return
		}
		active, current, total, currentItem = true, b.CurrentIndex, b.TotalItems, b.CurrentItem
	})
	if active && total > 0 {
		percentDone = current * 100 / total
	}
	return active, percentDone, current, total, currentItem
}

// Cancel stops the active generation batch, if any (AC 2). Idempotent: the
// in-flight item's pipeline ctx is cancelled, queued items never start.
func (p *GenerationBatchProcessor) Cancel() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.activeCancel != nil {
		p.activeCancel()
	}
}

// PreviewMissing returns how many movies a scope=missing batch would enumerate
// (AC 3 — the F8 idle-dialog count; semantics FROZEN per sub-4-2 AC #3: the
// batch itself stays movies-only) AND a library-wide missing count including
// episodes (sub-5-1 AC #7 — an upper bound on the consent list, which
// additionally filters skipped/unprobeable items). Two numbers, each honest
// for its own consumer.
//
// The episode leg DEGRADES rather than fails (sub-5-1 CR M2): total_items is
// a frozen pre-existing key whose availability must not start depending on
// the episodes table — an `episodes` lock (a recurring condition on the NAS
// target) would otherwise 500 a response whose movie half was already
// computed. Degraded includingEpisodes == movies is the pre-sub-5-1 toast
// behavior: an undercount, direction-safe. Same degrade for a nil finder.
func (p *GenerationBatchProcessor) PreviewMissing(ctx context.Context) (movies, includingEpisodes int, err error) {
	movies, err = p.finder.CountMissingZhHantSubtitle(ctx)
	if err != nil {
		return 0, 0, err
	}
	includingEpisodes = movies
	if p.episodes != nil {
		episodes, err := p.episodes.CountMissingZhHantSubtitle(ctx)
		if err != nil {
			p.logger.Warn("episode preview count failed — degrading to the movies-only count",
				"error", err)
			return movies, movies, nil
		}
		includingEpisodes += episodes
	}
	return movies, includingEpisodes, nil
}

// Start enumerates the queue and begins background processing. Returns the
// batch ID and the enumerated items (the 202 items[] list, in run order).
// A scope=missing resolving to 0 items returns ("", empty, nil) WITHOUT
// starting a batch — nothing to do is not an error (AC 1).
//
// budgetUSD is the user-approved ceiling from the request (sub-4-2 AC #1);
// 0 means "not provided → use the configured default". The handler validates
// user input to be strictly > 0, so 0 can only mean absent — user input is
// NEVER mapped onto ai.NewBudget's <=0 = unlimited semantic.
// Errors: ErrGenerationBatchRunning (409), ErrGenerationSelectionInvalid (400).
func (p *GenerationBatchProcessor) Start(ctx context.Context, scope string, mediaIDs []string, budgetUSD float64, modelID string) (string, []GenerationBatchItem, error) {
	// Quick check — release the lock before DB queries (fetch-batch H1 fix).
	busy := false
	p.withLock(func() { busy = p.activeBatch != nil })
	if busy {
		return "", nil, ErrGenerationBatchRunning
	}

	items, err := p.collectItems(ctx, scope, mediaIDs)
	if err != nil {
		return "", nil, err
	}

	if len(items) == 0 {
		return "", []GenerationBatchItem{}, nil
	}

	batchID := uuid.New().String()
	// Detached from the HTTP request so the batch outlives it; ONE shared
	// Budget rides the batch ctx into every item's pipeline (AC 6b).
	ceiling := p.budgetUSD
	if budgetUSD > 0 {
		ceiling = budgetUSD
	}
	budget := ai.NewBudget(ceiling)
	processCtx, processCancel := context.WithCancel(context.Background())
	processCtx = ai.WithBudget(processCtx, budget)
	// sub-6-8a AC #4: the model the user picked (and saw priced) rides the
	// batch ctx exactly as the shared Budget does, so every item's provider,
	// run row and segment-cache key name it without threading a parameter
	// through the runner port. Empty leaves the ctx untouched = the
	// deployment default.
	processCtx = ai.WithModelID(processCtx, modelID)

	// Re-acquire and double-check (another Start may have raced).
	conflict := false
	p.withLock(func() {
		if p.activeBatch != nil {
			conflict = true
			return
		}
		queue := make([]GenerationBatchItemState, len(items))
		for i, item := range items {
			queue[i] = GenerationBatchItemState{GenerationBatchItem: item, Status: GenerationBatchItemQueued}
		}
		// The queue exists before the goroutine runs, so the 202 progress can
		// never carry an empty one (dsr-6d-a AC #1).
		p.activeBatch = &GenerationBatchProgress{
			BatchID:    batchID,
			TotalItems: len(items),
			Status:     GenerationBatchStatusRunning,
			BudgetUSD:  ceiling,
			Items:      queue,
		}
		p.activeCancel = processCancel
		p.activeBudget = budget
		// A batch really starts only here, so only here does the previous
		// result stop being "the last one" — a 409 or an empty scope keeps it
		// (dsr-6d-a AC #3).
		p.lastBatch = nil
	})
	if conflict {
		processCancel()
		return "", nil, ErrGenerationBatchRunning
	}

	go p.process(processCtx, batchID, items, budget)

	return batchID, items, nil
}

// collectItems resolves the scope into the run-order queue.
//
// scope=missing stays DELIBERATELY movies-only (sub-4-2 AC #3): the frozen
// preview endpoint counts missing movies, and widening the batch without
// widening the count would make the legacy F8 dialog contradict itself.
// Episodes enter a batch exclusively via an explicit scope=selected id list
// (the F15 consent screen submits per-item selections).
//
// scope=selected resolves each id against BOTH sources (sub-4-2 AC #2, D1
// ruling): movies first, then episodes. Any id that resolves against neither
// rejects the WHOLE batch — the consented list is the amount confirmed on F16.
func (p *GenerationBatchProcessor) collectItems(ctx context.Context, scope string, mediaIDs []string) ([]GenerationBatchItem, error) {
	switch scope {
	case "missing":
		movies, err := p.finder.FindMissingZhHantSubtitle(ctx)
		if err != nil {
			return nil, fmt.Errorf("enumerate missing zh-Hant movies: %w", err)
		}
		items := make([]GenerationBatchItem, 0, len(movies))
		for _, m := range movies {
			item, ok := p.toItem(m)
			if !ok {
				continue // fail-soft: skip malformed rows, logged in toItem
			}
			items = append(items, item)
		}
		return items, nil
	case "selected":
		items := make([]GenerationBatchItem, 0, len(mediaIDs))
		// One series lookup per show within this batch (dsr-6d-a AC #7): a
		// select-all over a large library would otherwise read the series row
		// once per episode before the 202 returns.
		seriesTitles := map[string]string{}
		for _, id := range mediaIDs {
			movie, err := p.finder.FindByID(ctx, id)
			if err != nil && !errors.Is(err, sql.ErrNoRows) {
				// A real lookup failure (locked DB, cancelled ctx) is NOT "your
				// selection is invalid" — propagate for the handler's 500 (CR M2).
				return nil, fmt.Errorf("resolve media_id %s: %w", id, err)
			}
			if err == nil && movie != nil {
				if !movie.FilePath.Valid || movie.FilePath.String == "" {
					return nil, fmt.Errorf("media_id %s 沒有媒體檔案: %w", id, ErrGenerationSelectionInvalid)
				}
				item, ok := p.toItem(*movie)
				if !ok {
					return nil, fmt.Errorf("media_id %s 不是可生成字幕的項目: %w", id, ErrGenerationSelectionInvalid)
				}
				items = append(items, item)
				continue
			}
			item, ok, err := p.toEpisodeItem(ctx, id, seriesTitles)
			if err != nil {
				return nil, err
			}
			if !ok {
				// Unknown in both tables — a series id or a stale selection (reject).
				return nil, fmt.Errorf("media_id %s 不是可生成字幕的項目: %w", id, ErrGenerationSelectionInvalid)
			}
			items = append(items, item)
		}
		return items, nil
	default:
		return nil, fmt.Errorf("unknown generation batch scope: %s", scope)
	}
}

// toItem converts a movie row into a queue item. The row id (a UUID string)
// IS the wire media_id — no conversion (9R-18: the previous ParseInt here
// silently dropped every UUID-keyed movie from the batch).
func (p *GenerationBatchProcessor) toItem(m models.Movie) (GenerationBatchItem, bool) {
	if !m.FilePath.Valid || m.FilePath.String == "" {
		p.logger.Warn("skipping movie without file path in generation batch",
			"movie_id", m.ID, "title", m.Title)
		return GenerationBatchItem{}, false
	}
	return GenerationBatchItem{
		MediaID:   m.ID,
		Title:     m.Title,
		MediaType: models.SubtitleRunMediaMovie,
		filePath:  m.FilePath.String,
		mediaDir:  filepath.Dir(m.FilePath.String),
	}, true
}

// toEpisodeItem resolves a scope=selected id against the episodes table
// (sub-4-2 AC #2). ok=false means "not an episode either"; a found episode
// without a media file is a hard selection error (same rule as movies); a real
// lookup failure propagates (→ handler 500, CR M2).
// The title is cosmetic (SSE current_item) and built from the episode row
// alone; the show's name rides separately as series_title (dsr-6d-a AC #7) so
// the dialog and workspace rows can tell episodes of different shows apart.
func (p *GenerationBatchProcessor) toEpisodeItem(ctx context.Context, id string, seriesTitles map[string]string) (GenerationBatchItem, bool, error) {
	if p.episodes == nil {
		return GenerationBatchItem{}, false, nil
	}
	episode, err := p.episodes.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrEpisodeNotFound) {
			return GenerationBatchItem{}, false, nil
		}
		return GenerationBatchItem{}, false, fmt.Errorf("resolve media_id %s: %w", id, err)
	}
	if episode == nil {
		return GenerationBatchItem{}, false, nil
	}
	if !episode.FilePath.Valid || episode.FilePath.String == "" {
		return GenerationBatchItem{}, false, fmt.Errorf("media_id %s 沒有媒體檔案: %w", id, ErrGenerationSelectionInvalid)
	}
	title := fmt.Sprintf("S%02dE%02d", episode.SeasonNumber, episode.EpisodeNumber)
	if episode.Title.Valid && episode.Title.String != "" {
		title = fmt.Sprintf("%s %s", title, episode.Title.String)
	}
	return GenerationBatchItem{
		MediaID:     episode.ID,
		Title:       title,
		MediaType:   models.SubtitleRunMediaEpisode,
		SeriesTitle: p.seriesTitle(ctx, episode.SeriesID, seriesTitles),
		filePath:    episode.FilePath.String,
		mediaDir:    filepath.Dir(episode.FilePath.String),
	}, true, nil
}

// seriesTitle resolves a show's name once per batch (memo). Rule 13 case 3 —
// deliberately degraded after logging: the title is display-only, so a nil
// resolver, an empty series id or a failed lookup yields "" and never fails
// the batch start (the resolveSeriesMeta precedent in generation_candidates.go).
func (p *GenerationBatchProcessor) seriesTitle(ctx context.Context, seriesID string, memo map[string]string) string {
	if p.series == nil || seriesID == "" {
		return ""
	}
	if title, ok := memo[seriesID]; ok {
		return title
	}
	title := ""
	series, err := p.series.FindByID(ctx, seriesID)
	if err != nil || series == nil {
		p.logger.Warn("series title lookup failed — queue row shows the episode title only",
			"series_id", seriesID, "error", err)
	} else {
		title = series.Title
	}
	memo[seriesID] = title
	return title
}

// process runs the queue sequentially (one 轉錄中, rest 排隊中 — the shared
// ai.Governor is the real AI throttle). Per-item failures continue the loop;
// the budget ceiling pauses the remainder (AC 5/7).
func (p *GenerationBatchProcessor) process(ctx context.Context, batchID string, items []GenerationBatchItem, budget *ai.Budget) {
	// dsr-6d-a AC #4: a panic ends THIS batch with status error instead of
	// taking the whole API process down. Every critical section below unlocks
	// via defer, so the mutex is free by the time this runs.
	defer func() {
		if r := recover(); r != nil {
			p.logger.Error("generation batch panicked — ending it with status error",
				"batch_id", batchID, "panic", r, "stack", string(debug.Stack()))
			p.finish(batchID, GenerationBatchStatusError, -1, GenerationBatchItem{}, budget)
		}
	}()

	for i, item := range items {
		// Cancellation check before starting the next item (AC 2).
		select {
		case <-ctx.Done():
			p.finish(batchID, GenerationBatchStatusCancelled, i, item, budget)
			return
		default:
		}

		// AC 7: budget pre-check — an exhausted envelope pauses this item and
		// everything queued behind it (paused, NOT failed).
		if budget.Exceeded() {
			p.finish(batchID, GenerationBatchStatusBudgetCeiling, i, item, budget)
			return
		}

		p.markItem(batchID, i, item, GenerationBatchItemRunning, GenerationBatchItemReasonNone, budget)

		err := p.runner.ExecuteGeneration(ctx, item.MediaID, item.MediaType, item.filePath, item.mediaDir)
		switch {
		case err == nil:
			p.logger.Info("generation batch item succeeded",
				"batch_id", batchID, "index", i+1, "total", len(items),
				"media_id", item.MediaID, "title", item.Title)
			p.markItem(batchID, i, item, GenerationBatchItemDone, GenerationBatchItemReasonNone, budget)
		case ctx.Err() != nil:
			// The in-flight item died because the batch was cancelled — report
			// cancelled, not failed (AC 2).
			p.finish(batchID, GenerationBatchStatusCancelled, i+1, item, budget)
			return
		case errors.Is(err, ai.ErrBudgetExceeded):
			// AC 7: mid-item ceiling hit — this item and all remaining are paused.
			p.logger.Info("generation batch hit budget ceiling",
				"batch_id", batchID, "index", i+1, "total", len(items),
				"media_id", item.MediaID, "spent_usd", budget.SpentUSD())
			p.finish(batchID, GenerationBatchStatusBudgetCeiling, i+1, item, budget)
			return
		case errors.Is(err, ErrGenerationItemSkipped), errors.Is(err, ErrTranscriptionDisabled):
			// CR H1: the pipeline routed the item out (or, in legacy mode,
			// generation is not configured) without producing a subtitle — an
			// honest batch counts that as a failure, never a success (a keyless
			// deployment must not report N successes and zero subtitles).
			// Distinct log so the operator sees the reason class immediately.
			p.logger.Warn("generation batch item skipped — counted as failed",
				"batch_id", batchID, "index", i+1, "total", len(items),
				"media_id", item.MediaID, "title", item.Title, "reason", err)
			p.markItem(batchID, i, item, GenerationBatchItemFailed, GenerationBatchItemReasonSkipped, budget)
		case errors.Is(err, ErrTranscriptionInProgress):
			// The user ran that item from the detail dialog mid-batch, or the
			// pool owns it right now: count the failure, keep going (AC 5).
			p.logger.Warn("generation batch item already being processed elsewhere — continuing",
				"batch_id", batchID, "index", i+1, "total", len(items),
				"media_id", item.MediaID, "title", item.Title, "error", err)
			p.markItem(batchID, i, item, GenerationBatchItemFailed, GenerationBatchItemReasonBusyElsewhere, budget)
		default:
			// Per-item tolerance (AC 5): count the failure, keep going.
			p.logger.Warn("generation batch item failed — continuing",
				"batch_id", batchID, "index", i+1, "total", len(items),
				"media_id", item.MediaID, "title", item.Title, "error", err)
			p.markItem(batchID, i, item, GenerationBatchItemFailed, GenerationBatchItemReasonError, budget)
		}
	}

	var success, fail int
	var found bool
	p.withLock(func() {
		if p.activeBatch != nil && p.activeBatch.BatchID == batchID {
			success, fail, found = p.activeBatch.SuccessCount, p.activeBatch.FailCount, true
		}
	})
	if !found {
		// Unreachable today (only finish clears activeBatch, and only this
		// goroutine calls it) — but a zeroed count in a log is a lie, so say
		// it could not be read instead of pretending it was 0/0.
		p.logger.Error("generation batch counters vanished before the completion log",
			"batch_id", batchID)
	}
	p.logger.Info("generation batch complete",
		"batch_id", batchID, "total", len(items),
		"success", success, "fail", fail,
		"spent_usd", budget.SpentUSD(), "budget_usd", budget.Snapshot().BudgetUSD)
	last := GenerationBatchItem{}
	if len(items) > 0 {
		last = items[len(items)-1]
	}
	p.finish(batchID, GenerationBatchStatusComplete, len(items), last, budget)
}

// markItem moves queue entry i to status and updates the counters in the SAME
// critical section (dsr-6d-a AC #1), then broadcasts a running event carrying
// just that entry. Running also advances current_*.
func (p *GenerationBatchProcessor) markItem(batchID string, i int, item GenerationBatchItem, status GenerationBatchItemStatus, reason GenerationBatchItemReason, budget *ai.Budget) {
	var payload map[string]interface{}
	p.withLock(func() {
		b := p.activeBatch
		if b == nil || b.BatchID != batchID || i < 0 || i >= len(b.Items) {
			return
		}
		entry := &b.Items[i]
		entry.Status = status
		entry.Reason = reason
		switch status {
		case GenerationBatchItemRunning:
			b.CurrentIndex = i + 1
			b.CurrentMediaID = item.MediaID
			b.CurrentItem = item.Title
		case GenerationBatchItemDone:
			b.SuccessCount++
		case GenerationBatchItemFailed:
			b.FailCount++
		}
		changed := *entry
		payload = batchEventData(b, budget, nil, &changed)
	})
	p.broadcastData(payload)
}

// finish records the terminal status, keeps it as the last snapshot,
// broadcasts it, and clears the active batch (cancels the process ctx to
// release any derived resources).
//
// Item transitions do not depend on currentIndex (the pre-checks pass i, the
// mid-item exits i+1): every entry still queued or running moves to the
// terminal's state, and the counters are the per-item tallies afterwards
// (dsr-6d-a AC #1). currentIndex < 0 (the panic path) keeps current_* as the
// last running item set them. A second call for the same batch — a panic
// inside finish or broadcast re-entering through the recover — is a no-op.
func (p *GenerationBatchProcessor) finish(batchID, status string, currentIndex int, current GenerationBatchItem, budget *ai.Budget) {
	var payload map[string]interface{}
	var unfinished []string
	p.withLock(func() {
		b := p.activeBatch
		if b == nil || b.BatchID != batchID {
			return
		}
		if p.activeCancel != nil {
			p.activeCancel()
			p.activeCancel = nil
		}
		for idx := range b.Items {
			entry := &b.Items[idx]
			if entry.Status != GenerationBatchItemQueued && entry.Status != GenerationBatchItemRunning {
				continue
			}
			switch status {
			case GenerationBatchStatusBudgetCeiling:
				entry.Status = GenerationBatchItemPaused
			case GenerationBatchStatusError:
				if entry.Status == GenerationBatchItemRunning {
					entry.Status = GenerationBatchItemFailed
					entry.Reason = GenerationBatchItemReasonError
					b.FailCount++
				} else {
					entry.Status = GenerationBatchItemCancelled
				}
			case GenerationBatchStatusCancelled:
				entry.Status = GenerationBatchItemCancelled
			default:
				// complete with an unfinished entry is a bug in the loop above;
				// never leave a "running" row in a snapshot someone reads later.
				// Logged after the lock is released — a slow handler must not
				// block every reader of this processor.
				unfinished = append(unfinished, entry.MediaID)
				entry.Status = GenerationBatchItemCancelled
			}
		}
		paused := 0
		for _, entry := range b.Items {
			if entry.Status == GenerationBatchItemPaused {
				paused++
			}
		}
		b.PausedCount = paused
		b.Status = status
		if currentIndex >= 0 {
			b.CurrentIndex = currentIndex
			b.CurrentMediaID = current.MediaID
			b.CurrentItem = current.Title
		}
		if budget != nil {
			b.SpentUSD = budget.SpentUSD()
		}
		// Terminal: keep the snapshot (GET .../status last, dsr-6d-a AC #3) in
		// the same critical section that clears the active batch, so a status
		// probe never sees neither.
		p.activeBatch = nil
		p.activeBudget = nil
		p.lastBatch = b
		payload = batchEventData(b, budget, copyItems(b.Items), nil)
	})
	for _, mediaID := range unfinished {
		p.logger.Error("generation batch completed with an unfinished item",
			"batch_id", batchID, "media_id", mediaID)
	}
	p.broadcastData(payload)
}

// batchEventData builds the generation_batch_progress SSE payload (AC 9
// [@contract-v2]; items + changed_item additive since dsr-6d-a AC #2). The map
// is built by hand on purpose: ai.BudgetSnapshot has no json tags, and the SSE
// tests read the payload as a map. Running events carry changed_item and a nil
// items; the terminal event carries the whole queue and a nil changed_item —
// a select-all over thousands of items must not resend the queue twice per
// item. Caller holds p.mu.
func batchEventData(b *GenerationBatchProgress, budget *ai.Budget, items []GenerationBatchItemState, changed *GenerationBatchItemState) map[string]interface{} {
	var spent, ceiling float64
	if budget != nil {
		snap := budget.Snapshot()
		spent, ceiling = snap.SpentUSD, snap.BudgetUSD
	}
	data := map[string]interface{}{
		"batch_id":         b.BatchID,
		"total_items":      b.TotalItems,
		"current_index":    b.CurrentIndex,
		"current_media_id": b.CurrentMediaID,
		"current_item":     b.CurrentItem,
		"success_count":    b.SuccessCount,
		"fail_count":       b.FailCount,
		"paused_count":     b.PausedCount,
		"status":           b.Status,
		"spent_usd":        spent,
		"budget_usd":       ceiling,
		"items":            nil,
		"changed_item":     nil,
	}
	if items != nil {
		data["items"] = items
	}
	if changed != nil {
		data["changed_item"] = *changed
	}
	return data
}

// broadcastData emits a prepared generation_batch_progress payload outside the
// lock (current_media_id is a UUID STRING since 9R-18).
func (p *GenerationBatchProcessor) broadcastData(data map[string]interface{}) {
	if p.sseHub == nil || data == nil {
		return
	}
	p.sseHub.Broadcast(sse.Event{Type: sse.EventGenerationBatchProgress, Data: data})
}
