package handlers

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/vido/api/internal/services"
)

// GenerationBatchProcessorInterface is the narrow orchestrator surface the
// handler drives (Story 9R-16; Rule 11 test-fake precedent:
// TranscriptionServiceInterface in transcription_handler.go).
type GenerationBatchProcessorInterface interface {
	IsAvailable() bool
	IsRunning() bool
	Start(ctx context.Context, scope string, mediaIDs []string) (string, []services.GenerationBatchItem, error)
	GetProgress() *services.GenerationBatchProgress
	Cancel()
	PreviewMissing(ctx context.Context) (int, error)
}

// GenerationBatchHandler handles the Route C generation-batch API
// (Story 9R-16, [@contract-v2] since 9R-18: media ids are UUID STRINGS —
// FE consumer ux3-subtitle-v2-batch).
type GenerationBatchHandler struct {
	processor GenerationBatchProcessorInterface
}

// NewGenerationBatchHandler creates a new GenerationBatchHandler.
func NewGenerationBatchHandler(processor GenerationBatchProcessorInterface) *GenerationBatchHandler {
	return &GenerationBatchHandler{processor: processor}
}

// RegisterRoutes registers generation-batch routes on the given router group.
func (h *GenerationBatchHandler) RegisterRoutes(rg *gin.RouterGroup) {
	gb := rg.Group("/subtitles/generation-batch")
	{
		gb.POST("", h.StartGenerationBatch)
		gb.GET("/status", h.GetGenerationBatchStatus)
		gb.POST("/cancel", h.CancelGenerationBatch)
		gb.GET("/preview", h.PreviewGenerationBatch)
	}
}

// GenerationBatchStartRequest is the request body for starting a generation
// batch (snake_case per Rule 6). media_ids is required iff scope=selected;
// entries are movie row ids — UUID STRINGS (9R-18).
type GenerationBatchStartRequest struct {
	Scope    string   `json:"scope" binding:"required,oneof=missing selected"`
	MediaIDs []string `json:"media_ids"`
}

// StartGenerationBatch handles POST /api/v1/subtitles/generation-batch.
// @Summary Start a Route C subtitle-generation batch
// @Description Enumerates movies (movies only — series generation does not exist yet, 9R-10a) and runs the transcribe→translate→place pipeline sequentially under one shared AI budget. scope=missing enumerates movies lacking a zh-Hant subtitle; scope=selected runs the given movie media_ids — any id that is not a movie with a media file is REJECTED with 400 (not filtered). An empty missing scope returns 200 with total_items=0 (nothing to do is not an error).
// @Tags subtitles
// @Accept json
// @Produce json
// @Param request body GenerationBatchStartRequest true "scope: missing|selected; media_ids required iff scope=selected"
// @Success 202 {object} APIResponse "batch started: {batch_id, total_items, items:[{media_id,title}]}"
// @Success 200 {object} APIResponse "scope=missing resolved to 0 items: {total_items:0, items:[]}"
// @Failure 400 {object} APIResponse "validation failed (bad scope / missing media_ids / non-movie id)"
// @Failure 409 {object} APIResponse "TRANSCRIPTION_BATCH_RUNNING — current progress in error body data"
// @Failure 500 {object} APIResponse "TRANSCRIPTION_BATCH_START_FAILED"
// @Failure 503 {object} APIResponse "TRANSCRIPTION_DISABLED"
// @Router /api/v1/subtitles/generation-batch [post]
func (h *GenerationBatchHandler) StartGenerationBatch(c *gin.Context) {
	if !h.processor.IsAvailable() {
		ErrorResponse(c, http.StatusServiceUnavailable, "TRANSCRIPTION_DISABLED",
			"字幕生成功能未啟用",
			"請確認伺服器已安裝 FFmpeg 並設定 OPENAI_API_KEY。")
		return
	}

	var req GenerationBatchStartRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequestError(c, "VALIDATION_INVALID_FORMAT",
			"請求格式錯誤：scope 必須是 missing 或 selected")
		return
	}

	if req.Scope == "selected" && len(req.MediaIDs) == 0 {
		BadRequestError(c, "VALIDATION_REQUIRED_FIELD",
			"scope 為 selected 時必須提供 media_ids")
		return
	}
	if req.Scope == "missing" && len(req.MediaIDs) > 0 {
		BadRequestError(c, "VALIDATION_INVALID_FORMAT",
			"scope 為 missing 時不可提供 media_ids")
		return
	}

	batchID, items, err := h.processor.Start(c.Request.Context(), req.Scope, req.MediaIDs)
	if err != nil {
		if errors.Is(err, services.ErrGenerationBatchRunning) {
			// Mirror SUBTITLE_BATCH_RUNNING: progress rides the error body.
			progress := h.processor.GetProgress()
			c.JSON(http.StatusConflict, APIResponse{
				Success: false,
				Error: &APIError{
					Code:       "TRANSCRIPTION_BATCH_RUNNING",
					Message:    "已有一個字幕生成批次正在執行",
					Suggestion: "請等待目前批次完成，或先取消再重新開始。",
				},
				Data: progress,
			})
			return
		}
		if errors.Is(err, services.ErrGenerationSelectionInvalid) {
			BadRequestError(c, "VALIDATION_INVALID_FORMAT",
				"media_ids 含無法生成字幕的項目（非電影或沒有媒體檔案）："+err.Error())
			return
		}
		ErrorResponse(c, http.StatusInternalServerError, "TRANSCRIPTION_BATCH_START_FAILED",
			"字幕生成批次啟動失敗："+err.Error(),
			"請確認媒體資料庫可讀取後再試一次。")
		return
	}

	// scope=missing resolving to 0 items — nothing to do is not an error (AC 1).
	if len(items) == 0 {
		SuccessResponse(c, map[string]interface{}{
			"total_items": 0,
			"items":       []services.GenerationBatchItem{},
		})
		return
	}

	c.JSON(http.StatusAccepted, APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"batch_id":    batchID,
			"total_items": len(items),
			"items":       items,
		},
	})
}

// GetGenerationBatchStatus handles GET /api/v1/subtitles/generation-batch/status.
// @Summary Get generation-batch status
// @Description Recovery probe: returns whether a generation batch is running and its progress (null when idle).
// @Tags subtitles
// @Produce json
// @Success 200 {object} APIResponse "{running, progress|null}"
// @Router /api/v1/subtitles/generation-batch/status [get]
func (h *GenerationBatchHandler) GetGenerationBatchStatus(c *gin.Context) {
	progress := h.processor.GetProgress()
	SuccessResponse(c, map[string]interface{}{
		"running":  progress != nil,
		"progress": progress,
	})
}

// CancelGenerationBatch handles POST /api/v1/subtitles/generation-batch/cancel.
// @Summary Cancel the running generation batch
// @Description Idempotent: the in-flight item's pipeline is cancelled and queued items never start. Cancelling when nothing runs returns cancelled=false.
// @Tags subtitles
// @Produce json
// @Success 200 {object} APIResponse "{cancelled, running}"
// @Router /api/v1/subtitles/generation-batch/cancel [post]
func (h *GenerationBatchHandler) CancelGenerationBatch(c *gin.Context) {
	if !h.processor.IsRunning() {
		SuccessResponse(c, map[string]interface{}{
			"cancelled": false,
			"running":   false,
		})
		return
	}

	h.processor.Cancel()
	SuccessResponse(c, map[string]interface{}{
		"cancelled": true,
		"running":   h.processor.IsRunning(),
	})
}

// PreviewGenerationBatch handles GET /api/v1/subtitles/generation-batch/preview.
// @Summary Preview the missing-scope generation batch size
// @Description Returns how many movies scope=missing would enumerate (the F8 idle-dialog count) without starting anything. Only scope=missing is supported — a selected scope needs no preview.
// @Tags subtitles
// @Produce json
// @Param scope query string true "must be 'missing'"
// @Success 200 {object} APIResponse "{total_items}"
// @Failure 400 {object} APIResponse "scope missing or unsupported"
// @Failure 500 {object} APIResponse "DB_QUERY_FAILED"
// @Router /api/v1/subtitles/generation-batch/preview [get]
func (h *GenerationBatchHandler) PreviewGenerationBatch(c *gin.Context) {
	if c.Query("scope") != "missing" {
		BadRequestError(c, "VALIDATION_INVALID_FORMAT",
			"preview 僅支援 scope=missing")
		return
	}

	count, err := h.processor.PreviewMissing(c.Request.Context())
	if err != nil {
		ErrorResponse(c, http.StatusInternalServerError, "DB_QUERY_FAILED",
			"無法取得缺字幕項目數量："+err.Error(),
			"請稍後再試。")
		return
	}

	SuccessResponse(c, map[string]interface{}{
		"total_items": count,
	})
}
