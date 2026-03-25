package subtitle

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"

	"golang.org/x/sync/errgroup"

	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/sse"
	"github.com/vido/api/internal/subtitle/providers"
)

// SubtitleStatusUpdater is the minimal interface the engine needs for DB updates.
// Both MovieRepository and SeriesRepository implement this via UpdateSubtitleStatus.
type SubtitleStatusUpdater interface {
	UpdateSubtitleStatus(ctx context.Context, id string, status models.SubtitleStatus, path, language string, score float64) error
}

// PipelineStage represents the current stage of subtitle processing.
type PipelineStage string

const (
	StageSearching   PipelineStage = "searching"
	StageScoring     PipelineStage = "scoring"
	StageDownloading PipelineStage = "downloading"
	StageConverting  PipelineStage = "converting"
	StagePlacing     PipelineStage = "placing"
	StageComplete    PipelineStage = "complete"
	StageFailed      PipelineStage = "failed"
)

// ErrAllDownloadsFailed indicates all subtitle download attempts were exhausted.
var ErrAllDownloadsFailed = errors.New("subtitle: all download attempts failed")

// ErrNoResults indicates no subtitle search results were found.
var ErrNoResults = errors.New("subtitle: no search results found")

// EngineResult contains the outcome of a subtitle pipeline execution.
type EngineResult struct {
	Success      bool
	SubtitlePath string
	Language     string
	Score        float64
	Error        error
	ProviderUsed string
}

// Engine orchestrates the subtitle pipeline: Search → Score → Download → Convert → Place.
type Engine struct {
	providers  []providers.SubtitleProvider
	scorer     *Scorer
	converter  *Converter
	placer     *Placer
	sseHub     *sse.Hub
	movieRepo  SubtitleStatusUpdater
	seriesRepo SubtitleStatusUpdater
}

// NewEngine creates a subtitle pipeline engine with all dependencies injected.
func NewEngine(
	providerList []providers.SubtitleProvider,
	scorer *Scorer,
	converter *Converter,
	placer *Placer,
	sseHub *sse.Hub,
	movieRepo SubtitleStatusUpdater,
	seriesRepo SubtitleStatusUpdater,
) *Engine {
	return &Engine{
		providers:  providerList,
		scorer:     scorer,
		converter:  converter,
		placer:     placer,
		sseHub:     sseHub,
		movieRepo:  movieRepo,
		seriesRepo: seriesRepo,
	}
}

// ConversionPolicy controls how the engine handles simplified→traditional conversion.
type ConversionPolicy int

const (
	// ConvertAuto is the default: convert simplified Chinese to traditional.
	ConvertAuto ConversionPolicy = iota
	// ConvertAlways forces conversion regardless of detection result.
	ConvertAlways
	// ConvertNever skips conversion (for CN mainland content).
	ConvertNever
)

// ProcessOptions contains optional parameters for the subtitle pipeline.
type ProcessOptions struct {
	// ProductionCountry is the media's production country code (e.g., "CN").
	// When "CN", the default ConversionPolicy becomes ConvertNever.
	ProductionCountry string
	// ConversionOverride allows the caller to override the derived policy.
	// nil = use default based on ProductionCountry.
	ConversionOverride *ConversionPolicy
}

// deriveConversionPolicy returns the effective ConversionPolicy based on options.
func deriveConversionPolicy(opts *ProcessOptions) ConversionPolicy {
	if opts != nil && opts.ConversionOverride != nil {
		return *opts.ConversionOverride
	}
	if opts != nil && strings.Contains(opts.ProductionCountry, "CN") {
		return ConvertNever
	}
	return ConvertAuto
}

// Process runs the full subtitle pipeline for a single media item.
func (e *Engine) Process(ctx context.Context, mediaID, mediaType, mediaFilePath string, query providers.SubtitleQuery, mediaResolution string, opts ...ProcessOptions) EngineResult {
	var processOpts *ProcessOptions
	if len(opts) > 0 {
		processOpts = &opts[0]
	}
	conversionPolicy := deriveConversionPolicy(processOpts)
	// Stage 1: Search
	e.broadcastStatus(mediaID, mediaType, StageSearching, "Searching subtitle providers...")
	e.updateStatus(ctx, mediaID, mediaType, models.SubtitleStatusSearching)

	results, err := e.search(ctx, query)
	if err != nil {
		return e.handleFailure(ctx, mediaID, mediaType, fmt.Errorf("search: %w", err))
	}
	if len(results) == 0 {
		return e.handleFailure(ctx, mediaID, mediaType, ErrNoResults)
	}

	slog.Info("Subtitle search completed", "results", len(results), "mediaID", mediaID)

	// Stage 2: Score
	e.broadcastStatus(mediaID, mediaType, StageScoring, fmt.Sprintf("Scoring %d results...", len(results)))
	scored := e.scorer.Score(results, mediaResolution)

	// Stage 3: Download (with fallback)
	e.broadcastStatus(mediaID, mediaType, StageDownloading, "Downloading best match...")
	data, match, err := e.downloadBestMatch(ctx, scored)
	if err != nil {
		return e.handleFailure(ctx, mediaID, mediaType, err)
	}

	slog.Info("Subtitle downloaded", "provider", match.Source, "score", match.Score, "mediaID", mediaID)

	// Stage 4: Convert if needed (respects CN conversion policy)
	e.broadcastStatus(mediaID, mediaType, StageConverting, "Checking language...")
	convertedData, finalLang, err := e.convertIfNeeded(data, conversionPolicy)
	if err != nil {
		slog.Warn("Conversion failed, using original", "error", err, "mediaID", mediaID)
		// convertIfNeeded already returns original data on failure;
		// keep finalLang from detector (not provider metadata, which is often inaccurate).
	}

	// Stage 5: Place
	e.broadcastStatus(mediaID, mediaType, StagePlacing, "Placing subtitle file...")
	placeResult, err := e.placer.Place(PlaceRequest{
		MediaFilePath: mediaFilePath,
		SubtitleData:  convertedData,
		Language:      finalLang,
		Format:        match.Format,
		Score:         match.Score,
	})
	if err != nil {
		return e.handleFailure(ctx, mediaID, mediaType, fmt.Errorf("place: %w", err))
	}

	// Stage 6: Update DB
	e.updateSubtitleFound(ctx, mediaID, mediaType, placeResult.SubtitlePath, placeResult.Language, match.Score)
	e.broadcastStatus(mediaID, mediaType, StageComplete, "Subtitle found and placed!")

	return EngineResult{
		Success:      true,
		SubtitlePath: placeResult.SubtitlePath,
		Language:     placeResult.Language,
		Score:        match.Score,
		ProviderUsed: match.Source,
	}
}

// search queries all providers in parallel and merges results.
func (e *Engine) search(ctx context.Context, query providers.SubtitleQuery) ([]providers.SubtitleResult, error) {
	var (
		mu      sync.Mutex
		results []providers.SubtitleResult
		errs    []error
	)

	var g errgroup.Group

	for _, p := range e.providers {
		p := p // capture loop variable
		g.Go(func() error {
			res, err := p.Search(ctx, query)
			mu.Lock()
			defer mu.Unlock()

			if err != nil {
				slog.Warn("Provider search failed", "provider", p.Name(), "error", err)
				errs = append(errs, fmt.Errorf("%s: %w", p.Name(), err))
				return nil // Don't abort other providers
			}

			results = append(results, res...)
			return nil
		})
	}

	// Wait for all providers (errgroup doesn't return error since we return nil above)
	_ = g.Wait()

	// Only error if ALL providers failed
	if len(results) == 0 && len(errs) > 0 {
		return nil, fmt.Errorf("all providers failed: %v", errs)
	}

	return results, nil
}

// downloadBestMatch tries to download from scored results, best to worst.
func (e *Engine) downloadBestMatch(ctx context.Context, scored []ScoredResult) ([]byte, *ScoredResult, error) {
	for i := range scored {
		result := &scored[i]

		// Find the provider that owns this result
		provider := e.findProvider(result.Source)
		if provider == nil {
			slog.Warn("No provider found for source", "source", result.Source)
			continue
		}

		data, err := provider.Download(ctx, result.ID)
		if err != nil {
			slog.Warn("Download failed, trying next",
				"provider", result.Source,
				"id", result.ID,
				"error", err,
				"attempt", i+1,
				"total", len(scored),
			)
			continue
		}

		return data, result, nil
	}

	return nil, nil, ErrAllDownloadsFailed
}

// convertIfNeeded detects language and converts simplified → traditional if needed.
// Respects ConversionPolicy: ConvertNever skips conversion (CN content),
// ConvertAlways forces it, ConvertAuto uses detection-based logic.
func (e *Engine) convertIfNeeded(data []byte, policy ConversionPolicy) ([]byte, string, error) {
	detection := Detect(data)

	// ConvertNever: skip conversion entirely (CN mainland content)
	if policy == ConvertNever {
		return data, detection.Language, nil
	}

	switch detection.Language {
	case LangTraditional:
		if policy == ConvertAlways {
			// Already traditional — no conversion needed even if forced
			return data, LangTraditional, nil
		}
		return data, LangTraditional, nil
	case LangSimplified, LangAmbiguous:
		if e.converter != nil && e.converter.IsAvailable() {
			converted, err := e.converter.ConvertS2TWP(data)
			if err != nil {
				// Graceful degradation: return original with warning
				return data, detection.Language, fmt.Errorf("conversion failed: %w", err)
			}
			return converted, LangTraditional, nil
		}
		return data, detection.Language, nil
	default:
		// Non-Chinese or undetermined — pass through
		return data, detection.Language, nil
	}
}

// findProvider returns the provider matching the given source name.
func (e *Engine) findProvider(source string) providers.SubtitleProvider {
	for _, p := range e.providers {
		if p.Name() == source {
			return p
		}
	}
	return nil
}

// handleFailure updates status to not_found and broadcasts failure.
func (e *Engine) handleFailure(ctx context.Context, mediaID, mediaType string, err error) EngineResult {
	e.updateStatus(ctx, mediaID, mediaType, models.SubtitleStatusNotFound)
	e.broadcastStatus(mediaID, mediaType, StageFailed, err.Error())

	return EngineResult{
		Success: false,
		Error:   err,
	}
}

// updateStatus updates the subtitle status in the database.
func (e *Engine) updateStatus(ctx context.Context, mediaID, mediaType string, status models.SubtitleStatus) {
	var err error
	switch mediaType {
	case "movie":
		err = e.movieRepo.UpdateSubtitleStatus(ctx, mediaID, status, "", "", 0)
	case "series":
		err = e.seriesRepo.UpdateSubtitleStatus(ctx, mediaID, status, "", "", 0)
	default:
		slog.Error("Unknown mediaType in updateStatus", "mediaType", mediaType, "mediaID", mediaID)
		return
	}
	if err != nil {
		slog.Warn("Failed to update subtitle status", "mediaID", mediaID, "status", status, "error", err)
	}
}

// updateSubtitleFound updates the DB with the found subtitle details.
func (e *Engine) updateSubtitleFound(ctx context.Context, mediaID, mediaType, path, lang string, score float64) {
	var err error
	switch mediaType {
	case "movie":
		err = e.movieRepo.UpdateSubtitleStatus(ctx, mediaID, models.SubtitleStatusFound, path, lang, score)
	case "series":
		err = e.seriesRepo.UpdateSubtitleStatus(ctx, mediaID, models.SubtitleStatusFound, path, lang, score)
	default:
		slog.Error("Unknown mediaType in updateSubtitleFound", "mediaType", mediaType, "mediaID", mediaID)
		return
	}
	if err != nil {
		slog.Warn("Failed to update subtitle found status", "mediaID", mediaID, "error", err)
	}
}

// broadcastStatus sends an SSE event for the current pipeline stage.
func (e *Engine) broadcastStatus(mediaID, mediaType string, stage PipelineStage, message string) {
	if e.sseHub == nil {
		return
	}

	e.sseHub.Broadcast(sse.Event{
		Type: sse.EventSubtitleProgress,
		Data: map[string]interface{}{
			"mediaId":   mediaID,
			"mediaType": mediaType,
			"stage":     string(stage),
			"message":   message,
		},
	})
}
