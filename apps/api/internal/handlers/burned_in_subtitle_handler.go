package handlers

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/vido/api/internal/ai"
	"github.com/vido/api/internal/services"
)

// BurnedInSeasonChecker is the service surface this handler needs.
type BurnedInSeasonChecker interface {
	CheckSeason(ctx context.Context, seriesID string, season int) (services.BurnedInCheckResult, error)
}

// BurnedInSubtitleHandler triggers the burned-in subtitle check by hand
// (disc-2026-11-burned-in-subs-b1 AC #8) — for -b2's automation and for
// testing against the real library.
type BurnedInSubtitleHandler struct {
	checker BurnedInSeasonChecker
	logger  *slog.Logger
}

// NewBurnedInSubtitleHandler creates the handler.
func NewBurnedInSubtitleHandler(checker BurnedInSeasonChecker, logger *slog.Logger) *BurnedInSubtitleHandler {
	if logger == nil {
		logger = slog.Default()
	}
	return &BurnedInSubtitleHandler{checker: checker, logger: logger}
}

// RegisterRoutes mounts POST /api/v1/subtitles/burned-in/check.
func (h *BurnedInSubtitleHandler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.POST("/subtitles/burned-in/check", h.CheckSeason)
}

// burnedInCheckRequest is the body. Season is a pointer so a missing season
// is rejected rather than read as season 0 (specials).
type burnedInCheckRequest struct {
	SeriesID string `json:"series_id"`
	Season   *int   `json:"season"`
}

// CheckSeason handles POST /api/v1/subtitles/burned-in/check.
// @Summary Check one season for Chinese subtitles burned into the picture
// @Description Samples frames from the first episode of the season whose file has no subtitle track (subtitle_tracks = '[]'), asks Claude (claude-sonnet-5, thinking off) whether Chinese is burned in, tells Simplified from Traditional with OpenCC, and writes the answer onto every such episode of the season. A "none" needs a second opinion: when the season has another such episode, the middle one of them is checked too and its answer is the season's (undecidable → nothing written); a season with one such episode accepts a single none. `checked_episode_ids` lists the episodes read; `cost_usd` sums both calls. `result` is zh-Hans / zh-Hant / zh (Chinese, script untold) / none, or null when nothing was decided — `reason` then says why (no_episode_to_check, frames_unavailable, reply_unreadable, script_check_failed) and nothing is written. Each model call is recorded in ai_side_spend as kind burned_in_check.
// @Tags subtitles
// @Accept json
// @Produce json
// @Param body body burnedInCheckRequest true "{series_id, season}"
// @Success 200 {object} APIResponse "{result: string|null, reason?, episodes_updated, cost_usd, checked_episode_ids: string[]}"
// @Failure 400 {object} APIResponse "VALIDATION_ERROR — bad body; AI_NOT_CONFIGURED — no Claude API key"
// @Failure 502 {object} APIResponse "AI_PROVIDER_ERROR — the model call failed"
// @Router /api/v1/subtitles/burned-in/check [post]
func (h *BurnedInSubtitleHandler) CheckSeason(c *gin.Context) {
	var req burnedInCheckRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ValidationError(c, "請求格式不正確：需要 series_id 和 season。")
		return
	}
	req.SeriesID = strings.TrimSpace(req.SeriesID)
	if req.SeriesID == "" || req.Season == nil || *req.Season < 0 {
		ValidationError(c, "需要 series_id 和 season（0 以上的整數）。")
		return
	}

	res, err := h.checker.CheckSeason(c.Request.Context(), req.SeriesID, *req.Season)
	switch {
	case err == nil:
		SuccessResponse(c, res)
	case errors.Is(err, ai.ErrAINotConfigured):
		ErrorResponse(c, http.StatusBadRequest, "AI_NOT_CONFIGURED",
			"還沒設定 Claude 金鑰，沒辦法檢查字幕是不是燒在畫面上。",
			"到「設定 → 金鑰設定」填入 Claude 金鑰後再試一次。")
	default:
		h.logger.Error("burned-in subtitle check failed", "error", err, "series_id", req.SeriesID, "season", *req.Season)
		if errors.Is(err, services.ErrBurnedInModelCall) {
			ErrorResponse(c, http.StatusBadGateway, "AI_PROVIDER_ERROR",
				"問 Claude 的時候出了問題，這一季先不判斷。", "稍後再試一次；如果一直失敗，到「設定 → 金鑰設定」測試 Claude 金鑰。")
			return
		}
		InternalServerError(c, "檢查燒字幕時出錯，請稍後再試。")
	}
}
