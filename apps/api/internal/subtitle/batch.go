package subtitle

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/sse"
	"github.com/vido/api/internal/subtitle/providers"
)

// ErrBatchAlreadyRunning is returned when a batch is already in progress.
var ErrBatchAlreadyRunning = errors.New("batch already running")

// BatchScope defines the scope of batch subtitle processing.
type BatchScope string

const (
	// ScopeSeason processes all episodes in a specific season.
	ScopeSeason BatchScope = "season"
	// ScopeLibrary processes all media items needing subtitles.
	ScopeLibrary BatchScope = "library"
)

// BatchRequest is the input for starting a batch subtitle operation.
type BatchRequest struct {
	Scope    BatchScope `json:"scope"`
	SeasonID *string    `json:"season_id,omitempty"`
}

// BatchProgress reports the current state of a running batch.
type BatchProgress struct {
	BatchID      string `json:"batch_id"`
	TotalItems   int    `json:"total_items"`
	CurrentIndex int    `json:"current_index"`
	CurrentItem  string `json:"current_item"`
	SuccessCount int    `json:"success_count"`
	FailCount    int    `json:"fail_count"`
	Status       string `json:"status"` // "running", "complete", "cancelled", "error"
}

// FailedItem records a single item that failed during batch processing.
type FailedItem struct {
	MediaID   string `json:"media_id"`
	MediaType string `json:"media_type"`
	Title     string `json:"title"`
	Error     string `json:"error"`
}

// BatchItem is a media item to process in a batch.
type BatchItem struct {
	MediaID           string
	MediaType         string
	MediaFilePath     string
	Title             string
	Resolution        string
	ProductionCountry string // comma-separated ISO codes
}

// BatchConfig holds configurable parameters for batch processing.
type BatchConfig struct {
	// DelayBetweenItems is the pause between processing each item (default 3s).
	DelayBetweenItems time.Duration
}

// DefaultBatchConfig returns the default batch configuration.
func DefaultBatchConfig() BatchConfig {
	return BatchConfig{
		DelayBetweenItems: 3 * time.Second,
	}
}

// BatchItemCollector defines the interface for collecting items needing subtitles.
type BatchItemCollector interface {
	CollectMoviesNeedingSubtitles(ctx context.Context) ([]BatchItem, error)
	CollectSeriesNeedingSubtitles(ctx context.Context) ([]BatchItem, error)
	CollectEpisodesBySeasonID(ctx context.Context, seasonID string) ([]BatchItem, error)
}

// batchEngine is the narrow port over *Engine that the batch loop calls. It
// exists so the D5 flag seam can be proven with a spy: "legacy is byte-identical"
// has to be an assertion over captured arguments, not a claim about a diff.
type batchEngine interface {
	Process(ctx context.Context, mediaID, mediaType, mediaFilePath string,
		query providers.SubtitleQuery, mediaResolution string, opts ...ProcessOptions) EngineResult
}

// ItemProcessor is the D5 seam (sub-1-6 AC #1): the generation pipeline's
// per-item entry point. *Pipeline satisfies it.
//
// A nil ItemProcessor IS legacy mode. The flag itself never reaches this
// package — main.go reads it once at startup and either wires this option or
// does not, which is what keeps the flag out of the stages, the handlers and
// the scanner path (D5's ban on a flag that spreads).
type ItemProcessor interface {
	ProcessItem(ctx context.Context, ref MediaRef, opts ProcessItemOptions) (*ProcessOutcome, error)
}

// BatchProcessor manages batch subtitle processing with concurrency control.
type BatchProcessor struct {
	engine  batchEngine
	sseHub  *sse.Hub
	config  BatchConfig
	collect BatchItemCollector

	// item is the generation pipeline when the D5 flag is on, nil in legacy
	// mode. It is the ONE conditional the flag produces.
	item ItemProcessor

	mu           sync.Mutex
	activeBatch  *BatchProgress
	activeCancel context.CancelFunc
}

// BatchProcessorOption configures optional batch dependencies. Variadic so the
// existing four-argument construction keeps working unchanged.
type BatchProcessorOption func(*BatchProcessor)

// WithItemProcessor turns on the D5 pipeline seam. Wired by main.go only when
// VIDO_SUBTITLE_PIPELINE_MODE=pipeline.
func WithItemProcessor(item ItemProcessor) BatchProcessorOption {
	return func(bp *BatchProcessor) { bp.item = item }
}

// NewBatchProcessor creates a new BatchProcessor.
func NewBatchProcessor(
	engine batchEngine,
	sseHub *sse.Hub,
	collector BatchItemCollector,
	config BatchConfig,
	opts ...BatchProcessorOption,
) *BatchProcessor {
	bp := &BatchProcessor{
		engine:  engine,
		sseHub:  sseHub,
		config:  config,
		collect: collector,
	}
	for _, opt := range opts {
		opt(bp)
	}
	return bp
}

// Cancel stops the active batch processing, if any.
func (bp *BatchProcessor) Cancel() {
	bp.mu.Lock()
	defer bp.mu.Unlock()
	if bp.activeCancel != nil {
		bp.activeCancel()
	}
}

// IsRunning returns true if a batch is currently active.
func (bp *BatchProcessor) IsRunning() bool {
	bp.mu.Lock()
	defer bp.mu.Unlock()
	return bp.activeBatch != nil
}

// GetProgress returns the current batch progress, or nil if no batch is running.
func (bp *BatchProcessor) GetProgress() *BatchProgress {
	bp.mu.Lock()
	defer bp.mu.Unlock()
	if bp.activeBatch == nil {
		return nil
	}
	// Return a copy to avoid race
	p := *bp.activeBatch
	return &p
}

// ActivityProgress reports the active batch as primitives for the /activity aggregate
// (ux3-2-1). Kept primitive on purpose: internal/services must NOT import this package
// (subtitle already imports services — a shared type crossing here would import-cycle).
// Returns active=false when no batch is running.
func (bp *BatchProcessor) ActivityProgress() (active bool, percentDone, current, total int, currentItem string) {
	p := bp.GetProgress()
	if p == nil || p.Status != "running" {
		return false, 0, 0, 0, ""
	}
	if p.TotalItems > 0 {
		percentDone = p.CurrentIndex * 100 / p.TotalItems
	}
	return true, percentDone, p.CurrentIndex, p.TotalItems, p.CurrentItem
}

// Start begins batch processing in the background. Returns the batch ID and item count,
// or ErrBatchAlreadyRunning if a batch is already in progress (409 scenario).
func (bp *BatchProcessor) Start(ctx context.Context, req BatchRequest) (string, int, error) {
	// Quick check — release lock before DB queries (H1 fix)
	bp.mu.Lock()
	if bp.activeBatch != nil {
		bp.mu.Unlock()
		return "", 0, ErrBatchAlreadyRunning
	}
	bp.mu.Unlock()

	// Collect items WITHOUT holding the lock (H1 fix: avoid blocking IsRunning/GetProgress)
	items, err := bp.collectItems(ctx, req)
	if err != nil {
		return "", 0, fmt.Errorf("collect items: %w", err)
	}

	if len(items) == 0 {
		batchID := uuid.New().String()
		bp.broadcastComplete(batchID, 0, 0, 0)
		return batchID, 0, nil
	}

	// Re-acquire lock and double-check (another Start may have raced)
	bp.mu.Lock()
	if bp.activeBatch != nil {
		bp.mu.Unlock()
		return "", 0, ErrBatchAlreadyRunning
	}

	batchID := uuid.New().String()
	// Use background context so batch outlives the HTTP request (C1 fix)
	processCtx, processCancel := context.WithCancel(context.Background())
	bp.activeBatch = &BatchProgress{
		BatchID:    batchID,
		TotalItems: len(items),
		Status:     "running",
	}
	bp.activeCancel = processCancel
	bp.mu.Unlock()

	go bp.process(processCtx, batchID, items)

	return batchID, len(items), nil
}

// process runs the batch sequentially with delays.
func (bp *BatchProcessor) process(ctx context.Context, batchID string, items []BatchItem) {
	startTime := time.Now()
	var (
		successCount int
		failCount    int
	)

	for i, item := range items {
		// Check cancellation
		select {
		case <-ctx.Done():
			slog.Info("Batch cancelled", "batch_id", batchID, "processed", i, "total", len(items))
			bp.mu.Lock()
			if bp.activeBatch != nil {
				bp.activeBatch.Status = "cancelled"
			}
			bp.mu.Unlock()
			bp.broadcastProgress(batchID, len(items), i, item.Title, successCount, failCount, "cancelled")
			bp.clearActiveBatch()
			return
		default:
		}

		// Update progress
		bp.mu.Lock()
		if bp.activeBatch != nil {
			bp.activeBatch.CurrentIndex = i + 1
			bp.activeBatch.CurrentItem = item.Title
			bp.activeBatch.SuccessCount = successCount
			bp.activeBatch.FailCount = failCount
		}
		bp.mu.Unlock()

		// Build ProcessOptions with CN policy
		opts := ProcessOptions{
			ProductionCountry: item.ProductionCountry,
		}

		// ── D5 flag seam (sub-1-6 AC #1) ────────────────────────────────────
		// THE one conditional the feature flag produces. Everything above and
		// below is shared bookkeeping, so a batch reports identically whichever
		// backend ran it.
		var result EngineResult
		if bp.item != nil {
			result = bp.processViaPipeline(ctx, item)
		} else {
			query := providers.SubtitleQuery{Title: item.Title}
			result = bp.engine.Process(ctx, item.MediaID, item.MediaType, item.MediaFilePath,
				query, item.Resolution, opts)
		}

		if result.Success {
			successCount++
			slog.Info("Batch item succeeded",
				"batch_id", batchID,
				"index", i+1,
				"total", len(items),
				"media_id", item.MediaID,
				"title", item.Title,
			)
		} else {
			failCount++
			errMsg := "unknown error"
			if result.Error != nil {
				errMsg = result.Error.Error()
			}
			slog.Warn("Batch item failed",
				"batch_id", batchID,
				"index", i+1,
				"total", len(items),
				"media_id", item.MediaID,
				"title", item.Title,
				"error", errMsg,
			)
		}

		// Broadcast per-item progress
		bp.broadcastProgress(batchID, len(items), i+1, item.Title, successCount, failCount, "running")

		// Delay between items (except last)
		if i < len(items)-1 {
			select {
			case <-ctx.Done():
				slog.Info("Batch cancelled during delay", "batch_id", batchID, "processed", i+1, "total", len(items))
				bp.mu.Lock()
				if bp.activeBatch != nil {
					bp.activeBatch.Status = "cancelled"
				}
				bp.mu.Unlock()
				bp.broadcastProgress(batchID, len(items), i+1, item.Title, successCount, failCount, "cancelled")
				bp.clearActiveBatch()
				return
			case <-time.After(bp.config.DelayBetweenItems):
			}
		}
	}

	duration := time.Since(startTime)
	slog.Info("Batch complete",
		"batch_id", batchID,
		"total", len(items),
		"success", successCount,
		"fail", failCount,
		"duration", duration,
	)

	bp.broadcastComplete(batchID, len(items), successCount, failCount)
	bp.clearActiveBatch()
}

// collectItems gathers media items based on the batch scope.
func (bp *BatchProcessor) collectItems(ctx context.Context, req BatchRequest) ([]BatchItem, error) {
	switch req.Scope {
	case ScopeLibrary:
		return bp.collectLibraryItems(ctx)
	case ScopeSeason:
		if req.SeasonID == nil {
			return nil, fmt.Errorf("season_id required for season scope")
		}
		return bp.collectSeasonItems(ctx, *req.SeasonID)
	default:
		return nil, fmt.Errorf("unknown batch scope: %s", req.Scope)
	}
}

// collectLibraryItems gathers all movies and series needing subtitle search.
func (bp *BatchProcessor) collectLibraryItems(ctx context.Context) ([]BatchItem, error) {
	var items []BatchItem

	movies, err := bp.collect.CollectMoviesNeedingSubtitles(ctx)
	if err != nil {
		return nil, fmt.Errorf("collect movies: %w", err)
	}
	items = append(items, movies...)

	series, err := bp.collect.CollectSeriesNeedingSubtitles(ctx)
	if err != nil {
		return nil, fmt.Errorf("collect series: %w", err)
	}
	items = append(items, series...)

	return items, nil
}

// collectSeasonItems gathers episodes for a specific season needing subtitle search.
func (bp *BatchProcessor) collectSeasonItems(ctx context.Context, seasonID string) ([]BatchItem, error) {
	items, err := bp.collect.CollectEpisodesBySeasonID(ctx, seasonID)
	if err != nil {
		return nil, fmt.Errorf("collect episodes by season: %w", err)
	}
	return items, nil
}

// clearActiveBatch resets the active batch state and cancels the processing context.
func (bp *BatchProcessor) clearActiveBatch() {
	bp.mu.Lock()
	bp.activeBatch = nil
	if bp.activeCancel != nil {
		bp.activeCancel()
		bp.activeCancel = nil
	}
	bp.mu.Unlock()
}

// broadcastProgress sends a batch progress SSE event.
func (bp *BatchProcessor) broadcastProgress(batchID string, total, current int, currentItem string, success, fail int, status string) {
	if bp.sseHub == nil {
		return
	}
	bp.sseHub.Broadcast(sse.Event{
		Type: sse.EventSubtitleBatchProgress,
		Data: map[string]interface{}{
			"batch_id":      batchID,
			"total_items":   total,
			"current_index": current,
			"current_item":  currentItem,
			"success_count": success,
			"fail_count":    fail,
			"status":        status,
		},
	})
}

// broadcastComplete sends a batch completion SSE event.
func (bp *BatchProcessor) broadcastComplete(batchID string, total, success, fail int) {
	if bp.sseHub == nil {
		return
	}
	bp.sseHub.Broadcast(sse.Event{
		Type: sse.EventSubtitleBatchProgress,
		Data: map[string]interface{}{
			"batch_id":      batchID,
			"total_items":   total,
			"current_index": total,
			"success_count": success,
			"fail_count":    fail,
			"status":        "complete",
		},
	})
}

// processViaPipeline adapts one ProcessItem call onto the batch loop's
// EngineResult bookkeeping (sub-1-6 AC #1).
//
// Mapping notes:
//   - Score / ProviderUsed stay zero: a GENERATED subtitle was never scored
//     against provider results, and claiming a provider would make it
//     indistinguishable from a search hit (sub-1-5b AC #6.3).
//   - A nil-Run outcome is the P5 pre-flight early-exit — the item already had
//     an acceptable sidecar. That is the desired result of a re-scan, so it
//     counts as a SUCCESS, not a failure.
func (bp *BatchProcessor) processViaPipeline(ctx context.Context, item BatchItem) EngineResult {
	outcome, err := bp.item.ProcessItem(ctx,
		MediaRef{ID: item.MediaID, MediaType: item.MediaType},
		// The batch never forces: force is the operator's lever on the manual
		// endpoint (AC #4). An auto-run that re-translated everything on every
		// scan would burn the budget the P5 pre-flight exists to protect.
		ProcessItemOptions{})
	if err != nil {
		return EngineResult{Error: err}
	}
	if outcome == nil {
		return EngineResult{Error: fmt.Errorf("subtitle pipeline returned no outcome for %s %s",
			item.MediaType, item.MediaID)}
	}
	return EngineResult{
		Success:      true,
		SubtitlePath: outcome.SubtitlePath,
		Language:     deliveredLanguage,
	}
}

// --- Default Collector Implementation ---

// RepoCollector implements BatchItemCollector using repository interfaces.
type RepoCollector struct {
	movieRepo   MovieSubtitleFinder
	seriesRepo  SeriesSubtitleFinder
	episodeRepo EpisodeSeasonFinder
}

// MovieSubtitleFinder is the interface for finding movies needing subtitles.
type MovieSubtitleFinder interface {
	FindBySubtitleStatus(ctx context.Context, status models.SubtitleStatus) ([]models.Movie, error)
}

// SeriesSubtitleFinder is the interface for finding series needing subtitles.
type SeriesSubtitleFinder interface {
	FindBySubtitleStatus(ctx context.Context, status models.SubtitleStatus) ([]models.Series, error)
}

// EpisodeSeasonFinder is the interface for finding episodes by season ID.
type EpisodeSeasonFinder interface {
	FindBySeasonID(ctx context.Context, seasonID string) ([]models.Episode, error)
}

// NewRepoCollector creates a collector backed by repositories.
func NewRepoCollector(movieRepo MovieSubtitleFinder, seriesRepo SeriesSubtitleFinder, episodeRepo EpisodeSeasonFinder) *RepoCollector {
	return &RepoCollector{movieRepo: movieRepo, seriesRepo: seriesRepo, episodeRepo: episodeRepo}
}

// CollectMoviesNeedingSubtitles returns movies with not_searched or not_found status.
func (rc *RepoCollector) CollectMoviesNeedingSubtitles(ctx context.Context) ([]BatchItem, error) {
	var items []BatchItem

	for _, status := range []models.SubtitleStatus{models.SubtitleStatusNotSearched, models.SubtitleStatusNotFound} {
		movies, err := rc.movieRepo.FindBySubtitleStatus(ctx, status)
		if err != nil {
			return nil, err
		}
		for _, m := range movies {
			country := ""
			if countries, err := m.GetProductionCountries(); err == nil {
				codes := make([]string, 0, len(countries))
				for _, c := range countries {
					codes = append(codes, c.ISO3166_1)
				}
				country = strings.Join(codes, ",")
			}

			filePath := ""
			if m.FilePath.Valid {
				filePath = m.FilePath.String
			}

			items = append(items, BatchItem{
				MediaID:           m.ID,
				MediaType:         "movie",
				MediaFilePath:     filePath,
				Title:             m.Title,
				ProductionCountry: country,
			})
		}
	}

	return items, nil
}

// CollectSeriesNeedingSubtitles returns series with not_searched or not_found status.
// Note: Series model does not have production_countries — CN policy defaults to ConvertAuto.
func (rc *RepoCollector) CollectSeriesNeedingSubtitles(ctx context.Context) ([]BatchItem, error) {
	var items []BatchItem

	for _, status := range []models.SubtitleStatus{models.SubtitleStatusNotSearched, models.SubtitleStatusNotFound} {
		seriesList, err := rc.seriesRepo.FindBySubtitleStatus(ctx, status)
		if err != nil {
			return nil, err
		}
		for _, s := range seriesList {
			filePath := ""
			if s.FilePath.Valid {
				filePath = s.FilePath.String
			}

			items = append(items, BatchItem{
				MediaID:       s.ID,
				MediaType:     "series",
				MediaFilePath: filePath,
				Title:         s.Title,
				// Series model doesn't have production_countries — empty string = ConvertAuto
				ProductionCountry: "",
			})
		}
	}

	return items, nil
}

// CollectEpisodesBySeasonID returns episodes for a given season that have a file path.
func (rc *RepoCollector) CollectEpisodesBySeasonID(ctx context.Context, seasonID string) ([]BatchItem, error) {
	if rc.episodeRepo == nil {
		return nil, fmt.Errorf("episode repository not configured")
	}
	episodes, err := rc.episodeRepo.FindBySeasonID(ctx, seasonID)
	if err != nil {
		return nil, err
	}

	var items []BatchItem
	for _, ep := range episodes {
		// Only include episodes that have a media file
		if !ep.FilePath.Valid || ep.FilePath.String == "" {
			continue
		}

		title := ep.GetSeasonEpisodeCode()
		if ep.Title.Valid && ep.Title.String != "" {
			title = fmt.Sprintf("%s %s", ep.GetSeasonEpisodeCode(), ep.Title.String)
		}

		items = append(items, BatchItem{
			MediaID:       ep.ID,
			MediaType:     "episode",
			MediaFilePath: ep.FilePath.String,
			Title:         title,
			// Episodes don't have production_countries — empty string = ConvertAuto
			ProductionCountry: "",
		})
	}

	return items, nil
}
