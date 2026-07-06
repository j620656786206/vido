package services

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"path/filepath"
	"sync"

	"github.com/google/uuid"

	"github.com/vido/api/internal/ai"
	"github.com/vido/api/internal/models"
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
// Route C generation (unknown / non-movie id, or a movie without a media file).
// The handler maps it to 400 (AC 8 ruling: REJECT, not filter — the FE knows
// its selection and excludes series ids client-side).
var ErrGenerationSelectionInvalid = errors.New("selected media cannot run generation")

// Generation-batch terminal/live statuses (AC 9 vocabulary, [@contract-v2] —
// media ids are UUID STRINGS since 9R-18).
const (
	GenerationBatchStatusRunning       = "running"
	GenerationBatchStatusComplete      = "complete"
	GenerationBatchStatusCancelled     = "cancelled"
	GenerationBatchStatusError         = "error"
	GenerationBatchStatusBudgetCeiling = "budget_ceiling"
)

// GenerationBatchProgress reports the current state of a running generation
// batch (snake_case wire shape for GET .../status, AC 2).
type GenerationBatchProgress struct {
	BatchID        string  `json:"batch_id"`
	TotalItems     int     `json:"total_items"`
	CurrentIndex   int     `json:"current_index"`
	CurrentMediaID string  `json:"current_media_id"`
	CurrentItem    string  `json:"current_item"`
	SuccessCount   int     `json:"success_count"`
	FailCount      int     `json:"fail_count"`
	PausedCount    int     `json:"paused_count"`
	Status         string  `json:"status"`
	SpentUSD       float64 `json:"spent_usd"`
	BudgetUSD      float64 `json:"budget_usd"`
}

// GenerationBatchItem is one enumerated queue entry. The exported fields are
// the 202-response items[] shape (AC 1); file locations stay internal.
type GenerationBatchItem struct {
	MediaID string `json:"media_id"`
	Title   string `json:"title"`

	filePath string
	mediaDir string
}

// generationRunner is the narrow TranscriptionService surface the orchestrator
// needs (Rule 11 — enables test fakes without the full pipeline).
type generationRunner interface {
	IsAvailable() bool
	RunTranscription(ctx context.Context, mediaID string, filePath string, mediaDir string, opts ...TranscriptionOption) error
}

// generationCandidateFinder is the narrow movie-repo surface for enumeration
// (AC 4) + selected-id resolution. *repository.MovieRepository satisfies it.
type generationCandidateFinder interface {
	FindMissingZhHantSubtitle(ctx context.Context) ([]models.Movie, error)
	CountMissingZhHantSubtitle(ctx context.Context) (int, error)
	FindByID(ctx context.Context, id string) (*models.Movie, error)
}

// GenerationBatchProcessor runs the Route C generation pipeline sequentially
// over an enumerated queue under ONE shared ai.Budget (AC 5/6/7). Global
// single-flight: at most one generation batch at a time (independent from the
// Epic 8 fetch-batch — they share no state).
type GenerationBatchProcessor struct {
	runner    generationRunner
	finder    generationCandidateFinder
	sseHub    *sse.Hub
	budgetUSD float64 // shared batch ceiling (AI_RUN_BUDGET_USD; 0 = unlimited)
	logger    *slog.Logger

	mu           sync.Mutex
	activeBatch  *GenerationBatchProgress
	activeCancel context.CancelFunc
	activeBudget *ai.Budget
}

// NewGenerationBatchProcessor wires the orchestrator. budgetUSD is the shared
// batch cost ceiling (cfg.AIRunBudgetUSD).
func NewGenerationBatchProcessor(
	runner generationRunner,
	finder generationCandidateFinder,
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
		sseHub:    sseHub,
		budgetUSD: budgetUSD,
		logger:    logger.With("service", "generation_batch"),
	}
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

// GetProgress returns a copy of the current progress with live cost figures,
// or nil when no batch is running.
func (p *GenerationBatchProcessor) GetProgress() *GenerationBatchProgress {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.activeBatch == nil {
		return nil
	}
	prog := *p.activeBatch
	prog.SpentUSD = p.activeBudget.SpentUSD()
	return &prog
}

// ActivityProgress reports the active batch as primitives for the /activity
// aggregate (AC 10 — mirrors subtitle.BatchProcessor.ActivityProgress so the
// ActivityService source interface is shared). active=false when idle.
func (p *GenerationBatchProcessor) ActivityProgress() (active bool, percentDone, current, total int, currentItem string) {
	prog := p.GetProgress()
	if prog == nil || prog.Status != GenerationBatchStatusRunning {
		return false, 0, 0, 0, ""
	}
	if prog.TotalItems > 0 {
		percentDone = prog.CurrentIndex * 100 / prog.TotalItems
	}
	return true, percentDone, prog.CurrentIndex, prog.TotalItems, prog.CurrentItem
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
// (AC 3 — the F8 idle-dialog count) without starting anything.
func (p *GenerationBatchProcessor) PreviewMissing(ctx context.Context) (int, error) {
	return p.finder.CountMissingZhHantSubtitle(ctx)
}

// Start enumerates the queue and begins background processing. Returns the
// batch ID and the enumerated items (the 202 items[] list, in run order).
// A scope=missing resolving to 0 items returns ("", empty, nil) WITHOUT
// starting a batch — nothing to do is not an error (AC 1).
// Errors: ErrGenerationBatchRunning (409), ErrGenerationSelectionInvalid (400).
func (p *GenerationBatchProcessor) Start(ctx context.Context, scope string, mediaIDs []string) (string, []GenerationBatchItem, error) {
	// Quick check — release the lock before DB queries (fetch-batch H1 fix).
	p.mu.Lock()
	if p.activeBatch != nil {
		p.mu.Unlock()
		return "", nil, ErrGenerationBatchRunning
	}
	p.mu.Unlock()

	items, err := p.collectItems(ctx, scope, mediaIDs)
	if err != nil {
		return "", nil, err
	}

	if len(items) == 0 {
		return "", []GenerationBatchItem{}, nil
	}

	// Re-acquire and double-check (another Start may have raced).
	p.mu.Lock()
	if p.activeBatch != nil {
		p.mu.Unlock()
		return "", nil, ErrGenerationBatchRunning
	}

	batchID := uuid.New().String()
	// Detached from the HTTP request so the batch outlives it; ONE shared
	// Budget rides the batch ctx into every item's pipeline (AC 6b).
	budget := ai.NewBudget(p.budgetUSD)
	processCtx, processCancel := context.WithCancel(context.Background())
	processCtx = ai.WithBudget(processCtx, budget)

	p.activeBatch = &GenerationBatchProgress{
		BatchID:    batchID,
		TotalItems: len(items),
		Status:     GenerationBatchStatusRunning,
		BudgetUSD:  p.budgetUSD,
	}
	p.activeCancel = processCancel
	p.activeBudget = budget
	p.mu.Unlock()

	go p.process(processCtx, batchID, items, budget)

	return batchID, items, nil
}

// collectItems resolves the scope into the run-order queue (movies only, AC 8).
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
		for _, id := range mediaIDs {
			movie, err := p.finder.FindByID(ctx, id)
			if err != nil || movie == nil {
				// Unknown id — likely a series id or a stale selection (AC 8: reject).
				return nil, fmt.Errorf("media_id %s 不是可生成字幕的電影: %w", id, ErrGenerationSelectionInvalid)
			}
			if !movie.FilePath.Valid || movie.FilePath.String == "" {
				return nil, fmt.Errorf("media_id %s 沒有媒體檔案: %w", id, ErrGenerationSelectionInvalid)
			}
			item, ok := p.toItem(*movie)
			if !ok {
				return nil, fmt.Errorf("media_id %s 不是可生成字幕的電影: %w", id, ErrGenerationSelectionInvalid)
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
		MediaID:  m.ID,
		Title:    m.Title,
		filePath: m.FilePath.String,
		mediaDir: filepath.Dir(m.FilePath.String),
	}, true
}

// process runs the queue sequentially (one 轉錄中, rest 排隊中 — the shared
// ai.Governor is the real AI throttle). Per-item failures continue the loop;
// the budget ceiling pauses the remainder (AC 5/7).
func (p *GenerationBatchProcessor) process(ctx context.Context, batchID string, items []GenerationBatchItem, budget *ai.Budget) {
	var successCount, failCount int

	for i, item := range items {
		// Cancellation check before starting the next item (AC 2).
		select {
		case <-ctx.Done():
			p.finish(batchID, GenerationBatchStatusCancelled, len(items), i, item, successCount, failCount, 0, budget)
			return
		default:
		}

		// AC 7: budget pre-check — an exhausted envelope pauses this item and
		// everything queued behind it (paused, NOT failed).
		if budget.Exceeded() {
			paused := len(items) - i
			p.finish(batchID, GenerationBatchStatusBudgetCeiling, len(items), i, item, successCount, failCount, paused, budget)
			return
		}

		p.mu.Lock()
		if p.activeBatch != nil {
			p.activeBatch.CurrentIndex = i + 1
			p.activeBatch.CurrentMediaID = item.MediaID
			p.activeBatch.CurrentItem = item.Title
			p.activeBatch.SuccessCount = successCount
			p.activeBatch.FailCount = failCount
		}
		p.mu.Unlock()
		p.broadcast(batchID, len(items), i+1, item, successCount, failCount, 0, GenerationBatchStatusRunning, budget)

		err := p.runner.RunTranscription(ctx, item.MediaID, item.filePath, item.mediaDir, WithTranslation())
		switch {
		case err == nil:
			successCount++
			p.logger.Info("generation batch item succeeded",
				"batch_id", batchID, "index", i+1, "total", len(items),
				"media_id", item.MediaID, "title", item.Title)
		case ctx.Err() != nil:
			// The in-flight item died because the batch was cancelled — report
			// cancelled, not failed (AC 2).
			p.finish(batchID, GenerationBatchStatusCancelled, len(items), i+1, item, successCount, failCount, 0, budget)
			return
		case errors.Is(err, ai.ErrBudgetExceeded):
			// AC 7: mid-item ceiling hit — this item and all remaining are paused.
			paused := len(items) - i
			p.logger.Info("generation batch hit budget ceiling",
				"batch_id", batchID, "index", i+1, "total", len(items),
				"media_id", item.MediaID, "spent_usd", budget.SpentUSD())
			p.finish(batchID, GenerationBatchStatusBudgetCeiling, len(items), i+1, item, successCount, failCount, paused, budget)
			return
		default:
			// Per-item tolerance (AC 5) — includes ErrTranscriptionInProgress
			// (user ran that item from the detail dialog mid-batch): count the
			// failure, keep going.
			failCount++
			p.logger.Warn("generation batch item failed — continuing",
				"batch_id", batchID, "index", i+1, "total", len(items),
				"media_id", item.MediaID, "title", item.Title, "error", err)
		}

		p.mu.Lock()
		if p.activeBatch != nil {
			p.activeBatch.SuccessCount = successCount
			p.activeBatch.FailCount = failCount
		}
		p.mu.Unlock()
		p.broadcast(batchID, len(items), i+1, item, successCount, failCount, 0, GenerationBatchStatusRunning, budget)
	}

	p.logger.Info("generation batch complete",
		"batch_id", batchID, "total", len(items),
		"success", successCount, "fail", failCount,
		"spent_usd", budget.SpentUSD(), "budget_usd", p.budgetUSD)
	last := GenerationBatchItem{}
	if len(items) > 0 {
		last = items[len(items)-1]
	}
	p.finish(batchID, GenerationBatchStatusComplete, len(items), len(items), last, successCount, failCount, 0, budget)
}

// finish records the terminal status, broadcasts it, and clears the active
// batch (cancels the process ctx to release any derived resources).
func (p *GenerationBatchProcessor) finish(batchID, status string, total, currentIndex int, current GenerationBatchItem, success, fail, paused int, budget *ai.Budget) {
	p.mu.Lock()
	if p.activeCancel != nil {
		p.activeCancel()
		p.activeCancel = nil
	}
	// activeBatch is cleared (not status-stamped) at terminal — the fetch-batch
	// precedent: GET .../status reports {running:false, progress:null} after a
	// terminal state; the terminal snapshot reaches clients via the broadcast
	// below (dead-store on the cleared struct removed in 9R-16 CR).
	p.activeBatch = nil
	p.activeBudget = nil
	p.mu.Unlock()

	p.broadcast(batchID, total, currentIndex, current, success, fail, paused, status, budget)
}

// broadcast emits the generation_batch_progress SSE event (AC 9,
// [@contract-v2] — current_media_id is a UUID STRING since 9R-18). The payload
// map is built by hand — ai.BudgetSnapshot has no json tags on purpose.
func (p *GenerationBatchProcessor) broadcast(batchID string, total, currentIndex int, current GenerationBatchItem, success, fail, paused int, status string, budget *ai.Budget) {
	if p.sseHub == nil {
		return
	}
	snap := budget.Snapshot()
	p.sseHub.Broadcast(sse.Event{
		Type: sse.EventGenerationBatchProgress,
		Data: map[string]interface{}{
			"batch_id":         batchID,
			"total_items":      total,
			"current_index":    currentIndex,
			"current_media_id": current.MediaID,
			"current_item":     current.Title,
			"success_count":    success,
			"fail_count":       fail,
			"paused_count":     paused,
			"status":           status,
			"spent_usd":        snap.SpentUSD,
			"budget_usd":       snap.BudgetUSD,
		},
	})
}
