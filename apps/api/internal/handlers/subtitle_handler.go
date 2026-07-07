package handlers

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"errors"

	"github.com/gin-gonic/gin"
	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/sse"
	"github.com/vido/api/internal/subtitle"
	"github.com/vido/api/internal/subtitle/providers"
)

// SubtitleHandler handles HTTP requests for manual subtitle search and download.
type SubtitleHandler struct {
	providers      []providers.SubtitleProvider
	scorer         *subtitle.Scorer
	converter      *subtitle.Converter
	placer         *subtitle.Placer
	sseHub         *sse.Hub
	movieRepo      subtitle.SubtitleStatusUpdater
	seriesRepo     subtitle.SubtitleStatusUpdater
	batchProcessor *subtitle.BatchProcessor
}

// NewSubtitleHandler creates a new SubtitleHandler.
func NewSubtitleHandler(
	providerList []providers.SubtitleProvider,
	scorer *subtitle.Scorer,
	converter *subtitle.Converter,
	placer *subtitle.Placer,
	sseHub *sse.Hub,
	movieRepo subtitle.SubtitleStatusUpdater,
	seriesRepo subtitle.SubtitleStatusUpdater,
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

// SetBatchProcessor sets the batch processor for batch subtitle operations (Story 8-9).
func (h *SubtitleHandler) SetBatchProcessor(bp *subtitle.BatchProcessor) {
	h.batchProcessor = bp
}

// RegisterRoutes registers subtitle routes on the given router group.
func (h *SubtitleHandler) RegisterRoutes(rg *gin.RouterGroup) {
	subtitles := rg.Group("/subtitles")
	{
		subtitles.POST("/search", h.SearchSubtitles)
		subtitles.POST("/download", h.DownloadSubtitle)
		subtitles.POST("/preview", h.PreviewSubtitle)
		subtitles.POST("/convert", h.ConvertSubtitle)
		subtitles.POST("/batch", h.StartBatch)
		subtitles.GET("/batch/status", h.GetBatchStatus)
		subtitles.POST("/batch/cancel", h.CancelBatch)
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
	MediaID              string  `json:"media_id" binding:"required"`
	MediaType            string  `json:"media_type" binding:"required,oneof=movie series"`
	MediaFilePath        string  `json:"media_file_path" binding:"required"`
	SubtitleID           string  `json:"subtitle_id" binding:"required"`
	Provider             string  `json:"provider" binding:"required"`
	Resolution           string  `json:"resolution"`
	ConvertToTraditional *bool   `json:"convert_to_traditional"`
	Score                float64 `json:"score"`
}

// SubtitlePreviewRequest is the request body for subtitle preview.
type SubtitlePreviewRequest struct {
	SubtitleID string `json:"subtitle_id" binding:"required"`
	Provider   string `json:"provider" binding:"required"`
}

// SubtitleConvertRequest is the request body for converting an EXISTING local
// simplified-Chinese subtitle sidecar to Traditional Chinese (OpenCC s2twp).
// SourceLanguage is optional and defaults to zh-Hans.
type SubtitleConvertRequest struct {
	MediaID        string `json:"media_id" binding:"required"`
	MediaType      string `json:"media_type" binding:"required,oneof=movie series"`
	MediaFilePath  string `json:"media_file_path" binding:"required"`
	SourceLanguage string `json:"source_language"`
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
		ErrorResponse(c, 500, "SUBTITLE_DOWNLOAD_FAILED",
			"Failed to download subtitle: "+err.Error(),
			"Try a different provider or subtitle.")
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
		ErrorResponse(c, 500, "SUBTITLE_PLACE_FAILED",
			"Failed to place subtitle file: "+err.Error(),
			"Check file permissions and disk space.")
		return
	}

	// Update DB subtitle status (AC #8 — M1 fix)
	if err := h.updateSubtitleDB(c.Request.Context(), req.MediaID, req.MediaType,
		placeResult.SubtitlePath, placeResult.Language, req.Score); err != nil {
		slog.Error("Failed to update subtitle DB status",
			"media_id", req.MediaID,
			"error", err)
		// Non-fatal: file is placed, DB update failed — log but still return success
	}

	h.broadcastStatus(req.MediaID, req.MediaType, "complete", "Subtitle downloaded successfully!")

	SuccessResponse(c, map[string]interface{}{
		"subtitle_path": placeResult.SubtitlePath,
		"language":      placeResult.Language,
		"score":         req.Score,
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
		ErrorResponse(c, 500, "SUBTITLE_PREVIEW_FAILED",
			"Failed to download subtitle for preview: "+err.Error(),
			"Try a different subtitle or provider.")
		return
	}

	// Extract first 10 lines
	lines := extractFirstLines(data, 10)

	SuccessResponse(c, map[string]interface{}{
		"lines":    lines,
		"language": subtitle.Detect(data).Language,
	})
}

// ConvertSubtitle handles POST /api/v1/subtitles/convert.
//
// It converts an EXISTING local simplified-Chinese subtitle sidecar (e.g.
// Movie.zh-Hans.srt) to Traditional Chinese via OpenCC s2twp, writing a new
// Movie.zh-Hant.srt next to the media file. Synchronous; the source file is left
// in place (non-destructive). An explicit convert request IS the user's intent,
// so the §9b CN skip policy is deliberately NOT applied here.
func (h *SubtitleHandler) ConvertSubtitle(c *gin.Context) {
	var req SubtitleConvertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ValidationError(c, "Invalid request: "+err.Error())
		return
	}

	// OpenCC must be available to convert.
	if h.converter == nil || !h.converter.IsAvailable() {
		ErrorResponse(c, 503, "SUBTITLE_CONVERT_UNAVAILABLE",
			"繁簡轉換服務目前無法使用",
			"請確認伺服器已正確初始化 OpenCC 轉換器。")
		return
	}

	// Resolve + guard the media file path (must be absolute; mirrors placer).
	cleanMediaPath := filepath.Clean(req.MediaFilePath)
	if !filepath.IsAbs(cleanMediaPath) {
		BadRequestError(c, "VALIDATION_INVALID_FORMAT", "media_file_path must be an absolute path")
		return
	}

	// Default to simplified; NormalizeLanguageTag canonicalizes AND collapses unsafe
	// input to "und", so it doubles as a path-traversal guard for the filename segment.
	sourceLang := req.SourceLanguage
	if sourceLang == "" {
		sourceLang = subtitle.LangSimplified
	}
	sourceLang = subtitle.NormalizeLanguageTag(sourceLang)

	// Locate the source sidecar next to the media file: {name}.{lang}.srt then .ass.
	var sourcePath, sourceExt string
	for _, ext := range []string{"srt", "ass"} {
		candidate := subtitle.BuildSubtitleFilename(cleanMediaPath, sourceLang, ext)
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			sourcePath = candidate
			sourceExt = ext
			break
		}
	}
	if sourcePath == "" {
		ErrorResponse(c, 404, "SUBTITLE_NOT_FOUND",
			"找不到該語言的字幕檔",
			"請確認該媒體旁存在對應語言的字幕檔（例如 movie.zh-Hans.srt）。")
		return
	}

	// Read the source content.
	data, err := os.ReadFile(sourcePath)
	if err != nil {
		slog.Error("Failed to read source subtitle for convert", "path", sourcePath, "error", err)
		ErrorResponse(c, 500, "SUBTITLE_CONVERT_FAILED",
			"無法讀取來源字幕檔",
			"請確認檔案權限與磁碟狀態。")
		return
	}

	// Content-based language decision — never trust the filename tag (§9b detector).
	switch subtitle.Detect(data).Language {
	case subtitle.LangTraditional:
		ErrorResponse(c, 409, "SUBTITLE_ALREADY_TRADITIONAL",
			"字幕已是繁體中文，無需轉換", "")
		return
	case subtitle.LangSimplified, subtitle.LangAmbiguous:
		// proceed with conversion
	default:
		ErrorResponse(c, 400, "SUBTITLE_NOT_SIMPLIFIED",
			"該字幕不是簡體中文，無法轉換",
			"僅支援將簡體中文字幕轉為繁體中文。")
		return
	}

	// Convert simplified → traditional (OpenCC s2twp).
	converted, err := h.converter.ConvertS2TWP(data)
	if err != nil {
		slog.Error("OpenCC conversion failed", "path", sourcePath, "error", err)
		ErrorResponse(c, 500, "SUBTITLE_CONVERT_FAILED",
			"字幕轉換失敗", "請稍後再試。")
		return
	}

	// Place the converted track as {name}.zh-Hant.{ext} (non-destructive: the source
	// zh-Hans file stays; the placer backs up any pre-existing zh-Hant sidecar).
	placeResult, err := h.placer.Place(subtitle.PlaceRequest{
		MediaFilePath: cleanMediaPath,
		SubtitleData:  converted,
		Language:      subtitle.LangTraditional,
		Format:        sourceExt,
	})
	if err != nil {
		slog.Error("Failed to place converted subtitle", "media_file_path", cleanMediaPath, "error", err)
		ErrorResponse(c, 500, "SUBTITLE_PLACE_FAILED",
			"無法寫入轉換後的字幕檔："+err.Error(),
			"請確認檔案權限與磁碟空間。")
		return
	}

	// Persist status (pointer flip only; non-fatal on failure — the file is placed).
	if err := h.updateSubtitleDB(c.Request.Context(), req.MediaID, req.MediaType,
		placeResult.SubtitlePath, placeResult.Language, 1.0); err != nil {
		slog.Error("Failed to update subtitle DB after convert",
			"media_id", req.MediaID, "error", err)
	}

	SuccessResponse(c, map[string]interface{}{
		"subtitle_path": placeResult.SubtitlePath,
		"language":      placeResult.Language,
		"source_path":   sourcePath,
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

// --- Batch Handlers (Story 8-9) ---

// BatchStartRequest is the request body for starting a batch.
type BatchStartRequest struct {
	Scope    string  `json:"scope" binding:"required,oneof=season library"`
	SeasonID *string `json:"season_id"`
}

// StartBatch handles POST /api/v1/subtitles/batch
func (h *SubtitleHandler) StartBatch(c *gin.Context) {
	if h.batchProcessor == nil {
		ErrorResponse(c, 500, "SUBTITLE_BATCH_NOT_CONFIGURED",
			"Batch processing not configured",
			"Check server configuration for batch subtitle processing.")
		return
	}

	var req BatchStartRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ValidationError(c, "Invalid request: "+err.Error())
		return
	}

	// Validate season scope requires season_id
	if req.Scope == "season" && req.SeasonID == nil {
		BadRequestError(c, "VALIDATION_REQUIRED_FIELD", "season_id is required for season scope")
		return
	}

	batchReq := subtitle.BatchRequest{
		Scope:    subtitle.BatchScope(req.Scope),
		SeasonID: req.SeasonID,
	}

	// Start atomically checks for running batch and launches processing (H2 fix: no TOCTOU)
	batchID, totalItems, err := h.batchProcessor.Start(c.Request.Context(), batchReq)
	if err != nil {
		if errors.Is(err, subtitle.ErrBatchAlreadyRunning) {
			progress := h.batchProcessor.GetProgress()
			c.JSON(409, APIResponse{
				Success: false,
				Error: &APIError{
					Code:       "SUBTITLE_BATCH_RUNNING",
					Message:    "A batch is already in progress",
					Suggestion: "Wait for the current batch to complete before starting a new one.",
				},
				Data: progress,
			})
			return
		}
		ErrorResponse(c, 500, "SUBTITLE_BATCH_START_FAILED",
			"Failed to start batch: "+err.Error(),
			"Check that media items exist and providers are configured.")
		return
	}

	// Return 202 Accepted (AC #4)
	c.JSON(202, APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"batch_id":    batchID,
			"total_items": totalItems,
		},
	})
}

// GetBatchStatus handles GET /api/v1/subtitles/batch/status
func (h *SubtitleHandler) GetBatchStatus(c *gin.Context) {
	if h.batchProcessor == nil {
		SuccessResponse(c, map[string]interface{}{
			"running": false,
		})
		return
	}

	progress := h.batchProcessor.GetProgress()
	if progress == nil {
		SuccessResponse(c, map[string]interface{}{
			"running": false,
		})
		return
	}

	SuccessResponse(c, map[string]interface{}{
		"running":  true,
		"progress": progress,
	})
}

// CancelBatch handles POST /api/v1/subtitles/batch/cancel.
// Story 8-11 (frontend trigger): exposes the existing BatchProcessor.Cancel()
// (context-cancellation path) over HTTP so the UI can stop an in-flight batch.
// Idempotent: cancelling when no batch is running returns 200 with cancelled=false.
func (h *SubtitleHandler) CancelBatch(c *gin.Context) {
	if h.batchProcessor == nil {
		ErrorResponse(c, 500, "SUBTITLE_BATCH_NOT_CONFIGURED",
			"Batch processing not configured",
			"Check server configuration for batch subtitle processing.")
		return
	}

	if !h.batchProcessor.IsRunning() {
		SuccessResponse(c, map[string]interface{}{
			"cancelled": false,
			"running":   false,
		})
		return
	}

	h.batchProcessor.Cancel()
	SuccessResponse(c, map[string]interface{}{
		"cancelled": true,
	})
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
	scanner := bufio.NewScanner(bytes.NewReader(data))
	var lines []string
	for scanner.Scan() && len(lines) < n {
		line := scanner.Text()
		if strings.TrimSpace(line) != "" {
			lines = append(lines, line)
		}
	}
	return lines
}
