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
	// Start's budgetUSD is the user-approved ceiling; 0 = "not provided, use
	// the configured default" (sub-4-2 AC #1 — the handler guarantees user
	// input is strictly > 0, so 0 can only mean absent).
	Start(ctx context.Context, scope string, mediaIDs []string, budgetUSD float64) (string, []services.GenerationBatchItem, error)
	GetProgress() *services.GenerationBatchProgress
	Cancel()
	PreviewMissing(ctx context.Context) (movies, includingEpisodes int, err error)
}

// GenerationBatchHandler handles the generation-batch API (Story 9R-16;
// AC #1 [@contract-v3] since sub-4-2: additive budget_usd + items[].media_type,
// scope=selected accepts mixed movie/episode UUIDs; media ids are UUID STRINGS
// since 9R-18 — FE consumer ux3-subtitle-v2-batch).
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
// entries are movie OR episode row ids — UUID STRINGS (9R-18; mixed since
// sub-4-2 D1). budget_usd is the user-approved batch ceiling (sub-4-2 AC #1):
// optional, must be > 0 when present; absent falls back to AI_RUN_BUDGET_USD.
// A pointer distinguishes "absent" from a literal 0 so a zero can be rejected
// instead of silently meaning "unlimited".
type GenerationBatchStartRequest struct {
	Scope     string   `json:"scope" binding:"required,oneof=missing selected"`
	MediaIDs  []string `json:"media_ids"`
	BudgetUSD *float64 `json:"budget_usd"`
}

// StartGenerationBatch handles POST /api/v1/subtitles/generation-batch.
// @Summary Start a subtitle-generation batch
// @Description Runs the enumerated items sequentially under ONE shared AI budget. scope=missing enumerates movies lacking a zh-Hant subtitle (deliberately movies-only — the frozen preview endpoint counts movies; episodes enter via scope=selected); scope=selected runs the given media_ids, which may mix MOVIE and EPISODE row ids (sub-4-2 D1) — any id that resolves against neither table, or has no media file, REJECTS the whole batch with 400 (not filtered: the consented list is the confirmed amount). budget_usd (optional, > 0) is the user-approved batch ceiling; absent falls back to the AI_RUN_BUDGET_USD default. The ceiling is a SOFT cap: it is checked before each paid call, so the actual spend can slightly exceed it — reaching it pauses the remaining items (status budget_ceiling), completed items are kept. An empty missing scope returns 200 with total_items=0 (nothing to do is not an error).
// @Tags subtitles
// @Accept json
// @Produce json
// @Param request body GenerationBatchStartRequest true "scope: missing|selected; media_ids required iff scope=selected; budget_usd optional (> 0)"
// @Success 202 {object} APIResponse "batch started: {batch_id, total_items, items:[{media_id,title,media_type}]}"
// @Success 200 {object} APIResponse "scope=missing resolved to 0 items: {total_items:0, items:[]}"
// @Failure 400 {object} APIResponse "validation failed (bad scope / missing media_ids / unknown id / budget_usd <= 0)"
// @Failure 409 {object} APIResponse "TRANSCRIPTION_BATCH_RUNNING — current progress in error body data"
// @Failure 500 {object} APIResponse "TRANSCRIPTION_BATCH_START_FAILED"
// @Failure 503 {object} APIResponse "TRANSCRIPTION_DISABLED"
// @Router /api/v1/subtitles/generation-batch [post]
func (h *GenerationBatchHandler) StartGenerationBatch(c *gin.Context) {
	if !h.processor.IsAvailable() {
		// sub-2-2d AC #3 settings-page-first framing, made MODE-NEUTRAL by
		// sub-4-2 CR M3: in pipeline mode availability is the TRANSLATION
		// (Claude) key via the hot-reloading resolver (no restart needed);
		// in legacy mode it is the boot-time ASR key (restart needed). The
		// old copy unconditionally named the ASR key — a pipeline-mode user
		// with an ASR key but no Claude key was told to save the key they
		// already had. FFmpeg stays a deployment fact (bundled in the image).
		ErrorResponse(c, http.StatusServiceUnavailable, "TRANSCRIPTION_DISABLED",
			"字幕生成功能未啟用",
			"請至金鑰設定（/settings/keys）確認所需的 AI 金鑰已儲存（翻譯需 Claude 金鑰、語音辨識需雲端 ASR 金鑰）；若儲存後仍無法使用，請重啟伺服器。FFmpeg 已內建於 Docker 映像檔。")
		return
	}

	var req GenerationBatchStartRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// CR sub-4-2 L6: the bind can fail on ANY field — don't blame scope
		// for a malformed budget_usd or media_ids.
		BadRequestError(c, "VALIDATION_INVALID_FORMAT",
			"請求格式錯誤：請確認 scope（missing｜selected）、media_ids（字串陣列）與 budget_usd（數字）的型別")
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
	// sub-4-2 AC #1: a provided ceiling must be strictly positive — 0 or a
	// negative number must NEVER reach ai.NewBudget, whose <=0 means unlimited.
	var budgetUSD float64
	if req.BudgetUSD != nil {
		if *req.BudgetUSD <= 0 {
			BadRequestError(c, "VALIDATION_INVALID_FORMAT",
				"budget_usd 必須大於 0")
			return
		}
		budgetUSD = *req.BudgetUSD
	}

	batchID, items, err := h.processor.Start(c.Request.Context(), req.Scope, req.MediaIDs, budgetUSD)
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
				"media_ids 含無法生成字幕的項目（查無此電影或影集，或沒有媒體檔案）："+err.Error())
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
// @Description Returns how many movies scope=missing would enumerate (total_items — the F8 idle-dialog count) without starting anything, plus total_items_including_episodes: the library-wide missing count with episodes, for consumers (the F17 scan toast) whose next screen — the consent list — includes episodes. Only scope=missing is supported — a selected scope needs no preview.
// @Tags subtitles
// @Produce json
// @Param scope query string true "must be 'missing'"
// @Success 200 {object} APIResponse "{total_items, total_items_including_episodes}"
// @Failure 400 {object} APIResponse "scope missing or unsupported"
// @Failure 500 {object} APIResponse "DB_QUERY_FAILED"
// @Router /api/v1/subtitles/generation-batch/preview [get]
func (h *GenerationBatchHandler) PreviewGenerationBatch(c *gin.Context) {
	if c.Query("scope") != "missing" {
		BadRequestError(c, "VALIDATION_INVALID_FORMAT",
			"preview 僅支援 scope=missing")
		return
	}

	movies, includingEpisodes, err := h.processor.PreviewMissing(c.Request.Context())
	if err != nil {
		ErrorResponse(c, http.StatusInternalServerError, "DB_QUERY_FAILED",
			"無法取得缺字幕項目數量："+err.Error(),
			"請稍後再試。")
		return
	}

	SuccessResponse(c, map[string]interface{}{
		// total_items semantics FROZEN (sub-4-2 AC #3): what scope=missing
		// would actually run — movies only. Existing F8 consumers unchanged.
		"total_items": movies,
		// sub-5-1 AC #7 additive key (this endpoint's existing-keys-unchanged
		// precedent, no bump): the library-wide missing count INCLUDING
		// episodes — an UPPER BOUND on the consent list the F17 toast links
		// to (the list additionally filters skipped/unprobeable items and
		// items with no media file). Each number is honest for its own
		// consumer; do not merge them.
		"total_items_including_episodes": includingEpisodes,
	})
}
