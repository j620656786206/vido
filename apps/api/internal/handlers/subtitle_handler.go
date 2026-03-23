package handlers

import (
	"bufio"
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/vido/api/internal/subtitle"
	"github.com/vido/api/internal/subtitle/providers"
)

// SubtitleEngineInterface defines the contract for subtitle engine operations.
type SubtitleEngineInterface interface {
	Process(ctx context.Context, mediaID, mediaType, mediaFilePath string, query providers.SubtitleQuery, mediaResolution string) subtitle.EngineResult
}

// SubtitleHandler handles HTTP requests for manual subtitle search and download.
type SubtitleHandler struct {
	engine    SubtitleEngineInterface
	providers []providers.SubtitleProvider
	scorer    *subtitle.Scorer
	converter *subtitle.Converter
	placer    *subtitle.Placer
}

// NewSubtitleHandler creates a new SubtitleHandler.
func NewSubtitleHandler(
	engine SubtitleEngineInterface,
	providerList []providers.SubtitleProvider,
	scorer *subtitle.Scorer,
	converter *subtitle.Converter,
	placer *subtitle.Placer,
) *SubtitleHandler {
	return &SubtitleHandler{
		engine:    engine,
		providers: providerList,
		scorer:    scorer,
		converter: converter,
		placer:    placer,
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

// SubtitleSearchRequest is the request body for subtitle search.
type SubtitleSearchRequest struct {
	MediaID   string   `json:"mediaId" binding:"required"`
	MediaType string   `json:"mediaType" binding:"required,oneof=movie series"`
	Providers []string `json:"providers"`
	Query     string   `json:"query"`
}

// SubtitleDownloadRequest is the request body for subtitle download.
type SubtitleDownloadRequest struct {
	MediaID       string `json:"mediaId" binding:"required"`
	MediaType     string `json:"mediaType" binding:"required,oneof=movie series"`
	MediaFilePath string `json:"mediaFilePath" binding:"required"`
	SubtitleID    string `json:"subtitleId" binding:"required"`
	Provider      string `json:"provider" binding:"required"`
	Resolution    string `json:"resolution"`
}

// SubtitlePreviewRequest is the request body for subtitle preview.
type SubtitlePreviewRequest struct {
	SubtitleID string `json:"subtitleId" binding:"required"`
	Provider   string `json:"provider" binding:"required"`
}

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
		// Default: use mediaId as query (caller should pass title)
		query.Title = req.MediaID
	}

	// Search all selected providers in parallel
	var allResults []providers.SubtitleResult
	for _, p := range activeProviders {
		results, err := p.Search(c.Request.Context(), query)
		if err != nil {
			slog.Warn("Provider search failed in manual search",
				"provider", p.Name(), "error", err)
			continue
		}
		allResults = append(allResults, results...)
	}

	// Score results
	scored := h.scorer.Score(allResults, "")

	SuccessResponse(c, scored)
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

	// Download subtitle content
	data, err := provider.Download(c.Request.Context(), req.SubtitleID)
	if err != nil {
		slog.Error("Manual subtitle download failed",
			"provider", req.Provider,
			"subtitleId", req.SubtitleID,
			"error", err)
		InternalServerError(c, "Failed to download subtitle: "+err.Error())
		return
	}

	// Detect language and convert if needed
	detection := subtitle.Detect(data)
	finalData := data
	finalLang := detection.Language

	if subtitle.NeedsConversion(detection.Language) && h.converter != nil && h.converter.IsAvailable() {
		converted, convErr := h.converter.ConvertS2TWP(data)
		if convErr != nil {
			slog.Warn("Conversion failed in manual download, using original",
				"error", convErr)
		} else {
			finalData = converted
			finalLang = subtitle.LangTraditional
		}
	}

	// Place the subtitle file
	placeResult, err := h.placer.Place(subtitle.PlaceRequest{
		MediaFilePath: req.MediaFilePath,
		SubtitleData:  finalData,
		Language:      finalLang,
		Format:        "", // auto-detect from content
	})
	if err != nil {
		InternalServerError(c, "Failed to place subtitle file: "+err.Error())
		return
	}

	SuccessResponse(c, map[string]interface{}{
		"subtitlePath": placeResult.SubtitlePath,
		"language":     placeResult.Language,
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

	// Download with timeout for preview
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
