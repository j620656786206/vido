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
	MediaID            string
	MediaType          string
	MediaFilePath      string
	Title              string
	Resolution         string
	ProductionCountry  string // comma-separated ISO codes
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

// BatchProcessor manages batch subtitle processing with concurrency control.
type BatchProcessor struct {
	engine  *Engine
	sseHub  *sse.Hub
	config  BatchConfig
	collect BatchItemCollector

	mu           sync.Mutex
	activeBatch  *BatchProgress
	activeCancel context.CancelFunc
}

// NewBatchProcessor creates a new BatchProcessor.
func NewBatchProcessor(
	engine *Engine,
	sseHub *sse.Hub,
	collector BatchItemCollector,
	config BatchConfig,
) *BatchProcessor {
	return &BatchProcessor{
		engine:  engine,
		sseHub:  sseHub,
		config:  config,
		collect: collector,
	}
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
		failedItems  []FailedItem
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

		// Process the item
		query := providers.SubtitleQuery{Title: item.Title}
		result := bp.engine.Process(ctx, item.MediaID, item.MediaType, item.MediaFilePath,
			query, item.Resolution, opts)

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
			failedItems = append(failedItems, FailedItem{
				MediaID:   item.MediaID,
				MediaType: item.MediaType,
				Title:     item.Title,
				Error:     errMsg,
			})
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
