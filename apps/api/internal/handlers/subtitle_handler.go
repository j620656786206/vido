package handlers

import (
	"bufio"
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/sse"
	"github.com/vido/api/internal/subtitle"
	"github.com/vido/api/internal/subtitle/providers"
)

// SubtitleStatusUpdater is the interface for updating subtitle status in the DB.
type SubtitleStatusUpdater interface {
	UpdateSubtitleStatus(ctx context.Context, id string, status models.SubtitleStatus, path, language string, score float64) error
}

// SubtitleHandler handles HTTP requests for manual subtitle search and download.
type SubtitleHandler struct {
	providers  []providers.SubtitleProvider
	scorer     *subtitle.Scorer
	converter  *subtitle.Converter
	placer     *subtitle.Placer
	sseHub     *sse.Hub
	movieRepo  SubtitleStatusUpdater
	seriesRepo SubtitleStatusUpdater
}

// NewSubtitleHandler creates a new SubtitleHandler.
func NewSubtitleHandler(
	providerList []providers.SubtitleProvider,
	scorer *subtitle.Scorer,
	converter *subtitle.Converter,
	placer *subtitle.Placer,
	sseHub *sse.Hub,
	movieRepo SubtitleStatusUpdater,
	seriesRepo SubtitleStatusUpdater,
) *SubtitleHandler {
	return &SubtitleHandler{
		providers:  providerList,
		scorer:     scorer,
		converter:  converter,
		placer:     placer,
		sseHub:     sseHub,
		movieRepo:  movieRepo,
		seriesRepo: seriesRepo,
	}
}

// RegisterRoutes registers subtitle routes on the given router group.
func (h *SubtitleHandler) RegisterRoutes(rg *gin.RouterGroup) {
	subtitles := rg.Group("/subtitles")
	{
		subtitles.POST("/search", h.SearchSubtitles)
		subtitles.POST("/download", h.DownloadSubtitle)
		subtitles.POST("/preview", h.PreviewSubtitle)
	}
}

// --- Request / Response DTOs (snake_case JSON per Rule 6) ---

// SubtitleSearchRequest is the request body for subtitle search.
type SubtitleSearchRequest struct {
	MediaID   string   `json:"media_id" binding:"required"`
	MediaType string   `json:"media_type" binding:"required,oneof=movie series"`
	Providers []string `json:"providers"`
	Query     string   `json:"query"`
}

// SubtitleSearchResultDTO is the snake_case JSON response for a scored result.
type SubtitleSearchResultDTO struct {
	ID             string                     `json:"id"`
	Source         string                     `json:"source"`
	Filename       string                     `json:"filename"`
	Language       string                     `json:"language"`
	DownloadURL    string                     `json:"download_url"`
	Downloads      int                        `json:"downloads"`
	Group          string                     `json:"group"`
	Resolution     string                     `json:"resolution"`
	Format         string                     `json:"format"`
	Score          float64                    `json:"score"`
	ScoreBreakdown *SubtitleScoreBreakdownDTO `json:"score_breakdown"`
}

// SubtitleScoreBreakdownDTO is the snake_case JSON score breakdown.
type SubtitleScoreBreakdownDTO struct {
	Language    float64 `json:"language"`
	Resolution  float64 `json:"resolution"`
	SourceTrust float64 `json:"source_trust"`
	Group       float64 `json:"group"`
	Downloads   float64 `json:"downloads"`
}

// SubtitleDownloadRequest is the request body for subtitle download.
type SubtitleDownloadRequest struct {
	MediaID              string `json:"media_id" binding:"required"`
	MediaType            string `json:"media_type" binding:"required,oneof=movie series"`
	MediaFilePath        string `json:"media_file_path" binding:"required"`
	SubtitleID           string `json:"subtitle_id" binding:"required"`
	Provider             string `json:"provider" binding:"required"`
	Resolution           string `json:"resolution"`
	ConvertToTraditional *bool  `json:"convert_to_traditional"`
}

// SubtitlePreviewRequest is the request body for subtitle preview.
type SubtitlePreviewRequest struct {
	SubtitleID string `json:"subtitle_id" binding:"required"`
	Provider   string `json:"provider" binding:"required"`
}

// --- Handlers ---

// SearchSubtitles handles POST /api/v1/subtitles/search
func (h *SubtitleHandler) SearchSubtitles(c *gin.Context) {
	var req SubtitleSearchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ValidationError(c, "Invalid request: "+err.Error())
		return
	}

	// Filter providers based on request
	activeProviders := h.filterProviders(req.Providers)
	if len(activeProviders) == 0 {
		SuccessResponse(c, []interface{}{})
		return
	}

	// Build subtitle query
	query := providers.SubtitleQuery{
		Title: req.Query,
	}
	if query.Title == "" {
		query.Title = req.MediaID
	}

	// Search all selected providers in parallel (AC #2)
	var (
		mu         sync.Mutex
		allResults []providers.SubtitleResult
	)

	var wg sync.WaitGroup
	for _, p := range activeProviders {
		p := p // capture loop variable
		wg.Add(1)
		go func() {
			defer wg.Done()
			results, err := p.Search(c.Request.Context(), query)
			if err != nil {
				slog.Warn("Provider search failed in manual search",
					"provider", p.Name(), "error", err)
				return
			}
			mu.Lock()
			allResults = append(allResults, results...)
			mu.Unlock()
		}()
	}
	wg.Wait()

	// Score results
	scored := h.scorer.Score(allResults, "")

	// Convert to DTOs with snake_case JSON (L1 fix)
	dtos := make([]SubtitleSearchResultDTO, 0, len(scored))
	for _, s := range scored {
		dto := SubtitleSearchResultDTO{
			ID:          s.ID,
			Source:      s.Source,
			Filename:    s.Filename,
			Language:    s.Language,
			DownloadURL: s.DownloadURL,
			Downloads:   s.Downloads,
			Group:       s.Group,
			Resolution:  s.Resolution,
			Format:      s.Format,
			Score:       s.Score,
			ScoreBreakdown: &SubtitleScoreBreakdownDTO{
				Language:    s.ScoreBreakdown.Language,
				Resolution:  s.ScoreBreakdown.Resolution,
				SourceTrust: s.ScoreBreakdown.SourceTrust,
				Group:       s.ScoreBreakdown.Group,
				Downloads:   s.ScoreBreakdown.Downloads,
			},
		}
		dtos = append(dtos, dto)
	}

	SuccessResponse(c, dtos)
}

// DownloadSubtitle handles POST /api/v1/subtitles/download
func (h *SubtitleHandler) DownloadSubtitle(c *gin.Context) {
	var req SubtitleDownloadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ValidationError(c, "Invalid request: "+err.Error())
		return
	}

	// Find the requested provider
	provider := h.findProvider(req.Provider)
	if provider == nil {
		BadRequestError(c, "SUBTITLE_PROVIDER_NOT_FOUND",
			"Provider '"+req.Provider+"' is not configured or not available")
		return
	}

	// Broadcast SSE: downloading (AC #6)
	h.broadcastStatus(req.MediaID, req.MediaType, "downloading",
		fmt.Sprintf("Downloading subtitle from %s...", req.Provider))

	// Download subtitle content
	data, err := provider.Download(c.Request.Context(), req.SubtitleID)
	if err != nil {
		slog.Error("Manual subtitle download failed",
			"provider", req.Provider,
			"subtitle_id", req.SubtitleID,
			"error", err)
		h.broadcastStatus(req.MediaID, req.MediaType, "failed",
			"Download failed: "+err.Error())
		InternalServerError(c, "Failed to download subtitle: "+err.Error())
		return
	}

	// Detect language and apply conversion policy (AC #9, #10, #11)
	detection := subtitle.Detect(data)
	finalData := data
	finalLang := detection.Language

	shouldConvert := h.shouldConvert(detection.Language, req.ConvertToTraditional)

	if shouldConvert && h.converter != nil && h.converter.IsAvailable() {
		h.broadcastStatus(req.MediaID, req.MediaType, "converting",
			"Converting simplified → traditional...")
		converted, convErr := h.converter.ConvertS2TWP(data)
		if convErr != nil {
			slog.Warn("Conversion failed in manual download, using original",
				"error", convErr)
		} else {
			finalData = converted
			finalLang = subtitle.LangTraditional
		}
	}

	// Place the subtitle file (AC #5, #8)
	h.broadcastStatus(req.MediaID, req.MediaType, "placing", "Placing subtitle file...")

	placeResult, err := h.placer.Place(subtitle.PlaceRequest{
		MediaFilePath: req.MediaFilePath,
		SubtitleData:  finalData,
		Language:      finalLang,
		Format:        "",
	})
	if err != nil {
		h.broadcastStatus(req.MediaID, req.MediaType, "failed",
			"Failed to place subtitle: "+err.Error())
		InternalServerError(c, "Failed to place subtitle file: "+err.Error())
		return
	}

	// Update DB subtitle status (AC #8 — M1 fix)
	if err := h.updateSubtitleDB(c.Request.Context(), req.MediaID, req.MediaType,
		placeResult.SubtitlePath, placeResult.Language, 0); err != nil {
		slog.Error("Failed to update subtitle DB status",
			"media_id", req.MediaID,
			"error", err)
		// Non-fatal: file is placed, DB update failed — log but still return success
	}

	h.broadcastStatus(req.MediaID, req.MediaType, "complete", "Subtitle downloaded successfully!")

	SuccessResponse(c, map[string]interface{}{
		"subtitle_path": placeResult.SubtitlePath,
		"language":      placeResult.Language,
		"score":         0.0,
	})
}

// PreviewSubtitle handles POST /api/v1/subtitles/preview
func (h *SubtitleHandler) PreviewSubtitle(c *gin.Context) {
	var req SubtitlePreviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ValidationError(c, "Invalid request: "+err.Error())
		return
	}

	provider := h.findProvider(req.Provider)
	if provider == nil {
		BadRequestError(c, "SUBTITLE_PROVIDER_NOT_FOUND",
			"Provider '"+req.Provider+"' is not configured")
		return
	}

	// Download with timeout for preview (AC #4)
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	data, err := provider.Download(ctx, req.SubtitleID)
	if err != nil {
		InternalServerError(c, "Failed to download subtitle for preview: "+err.Error())
		return
	}

	// Extract first 10 lines
	lines := extractFirstLines(data, 10)

	SuccessResponse(c, map[string]interface{}{
		"lines":    lines,
		"language": subtitle.Detect(data).Language,
	})
}

// --- CN Conversion Policy (AC #9, #10, #11) ---

// shouldConvert determines whether to apply S→T conversion.
// If the user explicitly set convert_to_traditional, that takes priority (AC #11).
// Otherwise, simplified Chinese is always converted (AC #10).
// The frontend is responsible for defaulting the toggle based on
// production_countries (AC #9 — OFF for CN, ON for non-CN).
func (h *SubtitleHandler) shouldConvert(detectedLang string, userOverride *bool) bool {
	if !subtitle.NeedsConversion(detectedLang) {
		return false
	}
	if userOverride != nil {
		return *userOverride
	}
	// Default: convert simplified to traditional
	return true
}

// --- SSE Broadcasting ---

// broadcastStatus sends an SSE event for subtitle processing progress.
func (h *SubtitleHandler) broadcastStatus(mediaID, mediaType, stage, message string) {
	if h.sseHub == nil {
		return
	}
	h.sseHub.Broadcast(sse.Event{
		Type: sse.EventSubtitleProgress,
		Data: map[string]string{
			"media_id":   mediaID,
			"media_type": mediaType,
			"stage":      stage,
			"message":    message,
		},
	})
}

// --- DB Update ---

// updateSubtitleDB updates the subtitle status in the database for the given media.
func (h *SubtitleHandler) updateSubtitleDB(ctx context.Context, mediaID, mediaType, path, language string, score float64) error {
	switch mediaType {
	case "movie":
		if h.movieRepo != nil {
			return h.movieRepo.UpdateSubtitleStatus(ctx, mediaID, models.SubtitleStatusFound, path, language, score)
		}
	case "series":
		if h.seriesRepo != nil {
			return h.seriesRepo.UpdateSubtitleStatus(ctx, mediaID, models.SubtitleStatusFound, path, language, score)
		}
	}
	return nil
}

// --- Provider Helpers ---

// filterProviders returns providers matching the requested names, or all if empty.
func (h *SubtitleHandler) filterProviders(names []string) []providers.SubtitleProvider {
	if len(names) == 0 {
		return h.providers
	}

	nameSet := make(map[string]bool, len(names))
	for _, n := range names {
		nameSet[strings.ToLower(n)] = true
	}

	var filtered []providers.SubtitleProvider
	for _, p := range h.providers {
		if nameSet[strings.ToLower(p.Name())] {
			filtered = append(filtered, p)
		}
	}
	return filtered
}

// findProvider returns the provider matching the name, or nil.
func (h *SubtitleHandler) findProvider(name string) providers.SubtitleProvider {
	for _, p := range h.providers {
		if strings.EqualFold(p.Name(), name) {
			return p
		}
	}
	return nil
}

// extractFirstLines returns the first N non-empty lines from subtitle content.
func extractFirstLines(data []byte, n int) []string {
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	var lines []string
	for scanner.Scan() && len(lines) < n {
		line := scanner.Text()
		if strings.TrimSpace(line) != "" {
			lines = append(lines, line)
		}
	}
	return lines
}
