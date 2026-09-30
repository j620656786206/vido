package handlers

import (
	"context"
	"log/slog"

	"github.com/gin-gonic/gin"

	"github.com/vido/api/internal/services"
)

// SubtitleSpendReader is the narrow service surface this handler needs.
type SubtitleSpendReader interface {
	MonthSummary(ctx context.Context, batchID string) (*services.SubtitleSpendSummary, error)
}

// SubtitleSpendHandler serves the monthly AI spend summary (sub-7-6b).
type SubtitleSpendHandler struct {
	spend  SubtitleSpendReader
	logger *slog.Logger
}

// NewSubtitleSpendHandler creates the handler.
func NewSubtitleSpendHandler(spend SubtitleSpendReader, logger *slog.Logger) *SubtitleSpendHandler {
	if logger == nil {
		logger = slog.Default()
	}
	return &SubtitleSpendHandler{spend: spend, logger: logger}
}

// RegisterRoutes mounts GET /api/v1/subtitles/spend. Its own group on the
// same prefix as SubtitleHandler — gin merges them; there is no wildcard on
// /subtitles so nothing shadows this path.
func (h *SubtitleSpendHandler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET("/subtitles/spend", h.GetSpend)
}

// GetSpend handles GET /api/v1/subtitles/spend.
// @Summary This month's subtitle AI spend, and what the free lanes and the cache saved
// @Description Aggregates the subtitle_runs ledger for the current calendar month (server zone): translated/asr run counts and USD, a per-model table, the count of items delivered from an existing Chinese track with an ESTIMATE of the translation they avoided (`skipped_saved_runtime_assumed` flags when an item's runtime had to be assumed), and the cache's estimated saving (null when no run measured its split). Runs recorded before the ledger columns existed are reported apart as `unrouted_*`; completed runs without a recorded spend are counted in `unpriced_runs`, never as $0. `period` must be `month`. An optional `batch_id` adds `by_batch`, the receipt for one consent batch (any status, any month).
// @Tags subtitles
// @Produce json
// @Param period query string true "must be 'month'"
// @Param batch_id query string false "consent batch id — adds by_batch"
// @Success 200 {object} APIResponse "{period, from, to, translated_runs, translated_usd, asr_runs, asr_usd, unpriced_runs, unrouted_runs, unrouted_usd, skipped_deliver_count, skipped_saved_usd_estimate, skipped_saved_runtime_assumed, cache_hit_cues, cache_measured_runs, cache_saved_usd_estimate|null, by_model:[{model_id,runs,usd}], by_batch?:{batch_id,runs,completed_runs,usd,cue_count,cache_hit_cues|null,cache_measured_runs,model_id,model_ids?}}"
// @Failure 400 {object} APIResponse "VALIDATION_INVALID_FORMAT — period is not 'month'"
// @Router /api/v1/subtitles/spend [get]
func (h *SubtitleSpendHandler) GetSpend(c *gin.Context) {
	if period := c.Query("period"); period != "month" {
		BadRequestError(c, "VALIDATION_INVALID_FORMAT", "period must be 'month'")
		return
	}
	summary, err := h.spend.MonthSummary(c.Request.Context(), c.Query("batch_id"))
	if err != nil {
		h.logger.Error("subtitle spend summary failed", "error", err)
		InternalServerError(c, "無法讀取字幕花費紀錄，請稍後再試。")
		return
	}
	SuccessResponse(c, summary)
}
