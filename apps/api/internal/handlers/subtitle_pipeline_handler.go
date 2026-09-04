package handlers

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/vido/api/internal/subtitle"
)

// SubtitlePipelineQueue is the narrow surface the handler drives on the
// generation worker pool (*subtitle.WorkerPool satisfies it). A nil queue is
// the shipped legacy mode — VIDO_SUBTITLE_PIPELINE_MODE left at `legacy` means
// main.go builds no pool at all.
type SubtitlePipelineQueue interface {
	EnqueueItem(ref subtitle.MediaRef, opts subtitle.ProcessItemOptions) subtitle.EnqueueOutcome
	IsRunning() bool
}

// SubtitleMediaLookup resolves a MediaRef to a media row so the endpoint can
// answer 404 for an id that does not exist instead of silently queueing work
// that ProcessItem would fail on minutes later. subtitle.MediaStore satisfies it.
type SubtitleMediaLookup interface {
	Load(ctx context.Context, ref subtitle.MediaRef) (*subtitle.MediaItem, error)
}

// SubtitlePipelineHandler serves the FR12 manual-trigger endpoint
// (Story sub-1-6 AC #4). It is a NEW handler on purpose: subtitle_handler.go
// owns the manual provider-search path and stays untouched (D3).
type SubtitlePipelineHandler struct {
	queue SubtitlePipelineQueue
	media SubtitleMediaLookup
	// models validates a requested model_id (sub-6-8a AC #4). nil = accept
	// anything, the pre-story behaviour.
	models ModelValidator
	// configured is the FR23 capability gate (AC #5) — ONE predicate, wired
	// from cfg.HasClaudeKey, shared verbatim with the worker pool's enqueue
	// sweep and the batch seam.
	configured func() bool
}

// NewSubtitlePipelineHandler creates the handler. queue may be nil (legacy
// mode); configured may be nil, which reads as "configured".
func NewSubtitlePipelineHandler(queue SubtitlePipelineQueue, media SubtitleMediaLookup, configured func() bool, models ModelValidator) *SubtitlePipelineHandler {
	return &SubtitlePipelineHandler{queue: queue, media: media, configured: configured, models: models}
}

// RegisterRoutes registers the pipeline routes (Rule 10 — the group is already
// /api/v1).
func (h *SubtitlePipelineHandler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.POST("/subtitles/pipeline/run", h.RunPipeline)
}

// SubtitlePipelineRunRequest is the FR12 request body (snake_case, Rule 6).
//
// media_type uses the pipeline's INTERNAL vocabulary — movie|series|episode
// (sub-1-2 AC #1) — deliberately NOT the TMDB movie|tv pair that
// requests.media_type carries: `tv` would enqueue an id that resolves to nothing.
type SubtitlePipelineRunRequest struct {
	MediaID   string `json:"media_id" binding:"required"`
	MediaType string `json:"media_type" binding:"required,oneof=movie series episode"`
	// Force is FR32's re-run switch: bypass the P5 pre-flight and the segment
	// cache READS for this one item.
	Force bool `json:"force"`
	// ModelID pins the translation model for this one item (sub-6-8a AC #4).
	// Optional; empty = the deployment's effective default. Must be one of
	// GET /api/v1/settings/models.
	ModelID string `json:"model_id"`
}

// SubtitlePipelineRunResponse is the 202 payload.
type SubtitlePipelineRunResponse struct {
	// Status is "queued" (accepted now) or "already_queued" (an identical item
	// is queued or running — the caller's intent is already satisfied).
	Status  string `json:"status"`
	MediaID string `json:"media_id"`
}

// RunPipeline handles POST /api/v1/subtitles/pipeline/run.
// @Summary Queue one media item for subtitle generation
// @Description Runs the extract → (translate|convert) → place pipeline for ONE item. Always asynchronous: a translate run takes minutes (AC #7b), so the endpoint answers 202 immediately and progress arrives over SSE `subtitle_progress` (stages probing/extracting/translating/placing/complete). Requires VIDO_SUBTITLE_PIPELINE_MODE=pipeline and a configured translation key; either missing is a 409. A duplicate submit for an item already queued or running is also 202, with status=already_queued.
// @Tags subtitles
// @Accept json
// @Produce json
// @Param request body SubtitlePipelineRunRequest true "media_id + media_type (movie|series|episode); force re-runs an item that already has a sidecar; model_id optional (must be one of GET /settings/models)"
// @Success 202 {object} APIResponse "queued: {status: queued|already_queued, media_id}"
// @Failure 400 {object} APIResponse "VALIDATION_INVALID_FORMAT — missing media_id, unknown media_type, or unsupported model_id"
// @Failure 404 {object} APIResponse "DB_NOT_FOUND — no such media row"
// @Failure 409 {object} APIResponse "AI_NOT_CONFIGURED — no translation key, or the pipeline is not enabled"
// @Failure 500 {object} APIResponse "DB_QUERY_FAILED — the media lookup itself failed"
// @Failure 503 {object} APIResponse "SUBTITLE_QUEUE_FULL — the generation queue is saturated; retry later"
// @Router /api/v1/subtitles/pipeline/run [post]
func (h *SubtitlePipelineHandler) RunPipeline(c *gin.Context) {
	var req SubtitlePipelineRunRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequestError(c, "VALIDATION_INVALID_FORMAT",
			"請求格式錯誤：media_id 為必填，media_type 必須是 movie、series 或 episode")
		return
	}

	// ── AC #5: the capability gate, checked BEFORE any DB work ──────────────
	// An unconfigured install must not pay a query per request, and must never
	// end up with queued work that could only ever fail.
	if h.configured != nil && !h.configured() {
		ErrorResponse(c, http.StatusConflict, "AI_NOT_CONFIGURED",
			"尚未設定翻譯服務金鑰",
			// sub-2-1a: keys hot-reload now — saving one takes effect immediately,
			// so this suggestion must NOT tell the user to restart.
			"請於金鑰設定儲存 Claude API 金鑰（PUT /api/v1/settings/keys），或設定 CLAUDE_API_KEY 環境變數。")
		return
	}
	if h.queue == nil || !h.queue.IsRunning() {
		// Same Rule 7 code by design — this story adds none — but a different
		// message, because the operator's next action is a different knob.
		ErrorResponse(c, http.StatusConflict, "AI_NOT_CONFIGURED",
			"字幕生成管線尚未啟用",
			"請將 VIDO_SUBTITLE_PIPELINE_MODE 設為 pipeline 後重啟伺服器。")
		return
	}

	if h.models != nil && !h.models.Supports(c.Request.Context(), req.ModelID) {
		BadRequestError(c, "VALIDATION_INVALID_FORMAT",
			"不支援的模型：請從 GET /api/v1/settings/models 提供的清單中選擇，或省略 model_id 以使用預設模型")
		return
	}

	ref := subtitle.MediaRef{ID: req.MediaID, MediaType: req.MediaType}

	if h.media != nil {
		if _, err := h.media.Load(c.Request.Context(), ref); err != nil {
			if errors.Is(err, subtitle.ErrMediaNotFound) {
				NotFoundError(c, "Media")
				return
			}
			slog.Error("subtitle pipeline run: media lookup failed",
				"media_id", req.MediaID, "media_type", req.MediaType, "error", err)
			ErrorResponse(c, http.StatusInternalServerError, "DB_QUERY_FAILED",
				"無法讀取媒體資料",
				"請稍後再試；若持續發生請檢查伺服器日誌。")
			return
		}
	}

	// The four outcomes are answered DISTINCTLY (CR M1): "already_queued" is a
	// promise that the work is happening, so it is reserved for the genuine
	// duplicate — a dropped or refused item must never wear it.
	var status string
	switch h.queue.EnqueueItem(ref, subtitle.ProcessItemOptions{Force: req.Force, ModelID: req.ModelID}) {
	case subtitle.EnqueueAccepted:
		status = "queued"
	case subtitle.EnqueueDuplicate:
		status = "already_queued"
	case subtitle.EnqueueQueueFull:
		// Inline literal under the registered SUBTITLE_ prefix, following the
		// subtitle_handler.go precedent (SUBTITLE_PROVIDER_NOT_FOUND etc.) —
		// no registry change.
		ErrorResponse(c, http.StatusServiceUnavailable, "SUBTITLE_QUEUE_FULL",
			"字幕生成佇列已滿",
			"請稍後再試；下次掃描也會自動重新排入。")
		return
	default: // EnqueueStopped — the pool stopped between IsRunning and the send.
		ErrorResponse(c, http.StatusConflict, "AI_NOT_CONFIGURED",
			"字幕生成管線尚未啟用",
			"請將 VIDO_SUBTITLE_PIPELINE_MODE 設為 pipeline 後重啟伺服器。")
		return
	}

	// 202, never 200-with-result: holding the request open for a multi-minute
	// translate invites proxy timeouts and duplicate submits (J2 — SSE is the
	// progress channel, the subtitle_runs row is the record).
	c.JSON(http.StatusAccepted, APIResponse{
		Success: true,
		Data:    SubtitlePipelineRunResponse{Status: status, MediaID: req.MediaID},
	})
}
