package handlers

import (
	"context"
	"errors"
	"log/slog"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/vido/api/internal/services"
)

// SubtitleRunHistoryReader is the narrow service surface this handler needs.
type SubtitleRunHistoryReader interface {
	History(ctx context.Context, q services.RunHistoryQuery) (*services.SubtitleRunHistory, error)
}

// SubtitleRunHistoryHandler serves the per-run spend history
// (disc-2026-11-spend-history-b-api).
type SubtitleRunHistoryHandler struct {
	history SubtitleRunHistoryReader
	logger  *slog.Logger
}

// NewSubtitleRunHistoryHandler creates the handler.
func NewSubtitleRunHistoryHandler(history SubtitleRunHistoryReader, logger *slog.Logger) *SubtitleRunHistoryHandler {
	if logger == nil {
		logger = slog.Default()
	}
	return &SubtitleRunHistoryHandler{history: history, logger: logger}
}

// RegisterRoutes mounts GET /api/v1/subtitles/runs — a static path beside
// /subtitles/spend; nothing under /subtitles is a wildcard, so nothing
// shadows it.
func (h *SubtitleRunHistoryHandler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET("/subtitles/runs", h.ListRuns)
}

// ListRuns handles GET /api/v1/subtitles/runs.
// @Summary One month of subtitle runs: what each cost, what it was estimated at, and which models ran
// @Description [@contract-v1] Lists the completed, failed (stopped runs included — they cost money too) and skipped subtitle runs that STARTED in one calendar month (server zone), newest first. `summary` and `series` always cover the whole month; `category` filters `items` and `total` only. `summary.spent_usd` sums every recorded spend; `unpriced_runs` counts runs whose spend was never recorded (never reported as $0); `estimate_usd` sums every recorded estimate (null when none) and `estimated_runs` counts them; `comparable_*` covers only runs that recorded both spend and estimate, and the two sums are null when there are none. `series` has one entry per day (through today for the current month, none for a future month); a day without runs is a real 0, and the four `by_category` values add up to the day's `spent_usd` (no_subtitle runs are not charted). Per item, any value that was not recorded is null: `spent_usd`, `estimate_usd`, `route`, `source`, `asr_provider`, `asr_model`, `batch_id`, `triggered_by`; `translation_model` is set only on the translate and asr routes; `media.title` is null when the item has left the library; `failure_reason` is the activity record's code for a failed run and "" otherwise. Category by route: deliver_direct / convert_then_deliver → chinese_track, translate → translate_english, asr → asr, unrecorded → legacy, anything else → no_subtitle.
// @Tags subtitles
// @Produce json
// @Param month query string false "YYYY-MM in the server's zone; omitted = this month"
// @Param category query string false "chinese_track | translate_english | asr | legacy | no_subtitle; omitted = all (filters items only)"
// @Param page query int false "1-based page, default 1"
// @Param page_size query int false "items per page, default 50, max 200"
// @Success 200 {object} APIResponse "{month, summary:{spent_usd, runs, unpriced_runs, estimate_usd|null, estimated_runs, comparable_runs, comparable_spent_usd|null, comparable_estimate_usd|null}, series:[{date, spent_usd, runs, by_category:{chinese_track, translate_english, asr, legacy}}], items:[{id, started_at, completed_at|null, status, failure_reason, media:{id, type, title|null, season|null, episode|null}, route|null, source|null, category, spent_usd|null, estimate_usd|null, translation_model|null, asr_provider|null, asr_model|null, batch_id|null, triggered_by|null}], page, page_size, total}"
// @Failure 400 {object} APIResponse "VALIDATION_INVALID_FORMAT — month is not YYYY-MM, category is unknown, or page / page_size is out of range"
// @Router /api/v1/subtitles/runs [get]
func (h *SubtitleRunHistoryHandler) ListRuns(c *gin.Context) {
	page, ok := positiveIntQuery(c, "page")
	if !ok {
		BadRequestError(c, "VALIDATION_INVALID_FORMAT", "page must be a positive integer")
		return
	}
	pageSize, ok := positiveIntQuery(c, "page_size")
	if !ok {
		BadRequestError(c, "VALIDATION_INVALID_FORMAT", "page_size must be a positive integer")
		return
	}
	history, err := h.history.History(c.Request.Context(), services.RunHistoryQuery{
		Month:    c.Query("month"),
		Category: c.Query("category"),
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		if errors.Is(err, services.ErrInvalidRunHistoryQuery) {
			BadRequestError(c, "VALIDATION_INVALID_FORMAT", strings.TrimPrefix(err.Error(), services.ErrInvalidRunHistoryQuery.Error()+": "))
			return
		}
		h.logger.Error("subtitle run history failed", "error", err)
		InternalServerError(c, "無法讀取字幕花費明細，請稍後再試。")
		return
	}
	SuccessResponse(c, history)
}

// positiveIntQuery reads an optional positive integer query value: absent is
// (0, true) — the service's default — and anything else that is not >= 1 is
// (0, false).
func positiveIntQuery(c *gin.Context, key string) (int, bool) {
	raw, present := c.GetQuery(key)
	if !present {
		return 0, true
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 1 {
		return 0, false
	}
	return n, true
}
