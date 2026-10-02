package handlers

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/vido/api/internal/subtitle/miner"
)

// OfficialSubtitleMiner is the narrow surface the handler drives (Rule 11);
// *miner.OfficialSubtitleMiner satisfies it.
type OfficialSubtitleMiner interface {
	MineSeries(ctx context.Context, seriesID string) (miner.MineResult, error)
	StartPartial(ctx context.Context) error
	Status() miner.MineStatus
}

// GlossaryMineHandler serves 「重新從官方字幕學習」 (story sub-7-5b AC #3).
type GlossaryMineHandler struct {
	miner OfficialSubtitleMiner
}

// NewGlossaryMineHandler creates the handler.
func NewGlossaryMineHandler(m OfficialSubtitleMiner) *GlossaryMineHandler {
	return &GlossaryMineHandler{miner: m}
}

// RegisterRoutes mounts the routes under /subtitles/glossary/mine.
func (h *GlossaryMineHandler) RegisterRoutes(rg *gin.RouterGroup) {
	g := rg.Group("/subtitles/glossary/mine")
	{
		g.GET("", h.Status)
		g.POST("", h.Start)
	}
}

// GlossaryMineRequest is the POST body. series_id empty = sweep every
// partial show in the library (what the settings-page button does).
type GlossaryMineRequest struct {
	SeriesID string `json:"series_id"`
}

// Status handles GET /api/v1/subtitles/glossary/mine.
//
// @Summary Last official-subtitle mining run and whether one is in flight
// @Tags subtitles
// @Produce json
// @Success 200 {object} APIResponse "miner.MineStatus [@contract-v1]"
// @Router /subtitles/glossary/mine [get]
func (h *GlossaryMineHandler) Status(c *gin.Context) {
	SuccessResponse(c, h.miner.Status())
}

// Start handles POST /api/v1/subtitles/glossary/mine.
//
// @Summary Learn a show's proper-noun renderings from its official zh-Hant subtitles
// @Description With `series_id`: runs that one show synchronously and returns its MineResult. Without: starts a background sweep over every partial show (202; poll GET for results). $0 — no model is called. 409 GLOSSARY_MINE_RUNNING while a run is in flight.
// @Tags subtitles
// @Accept json
// @Produce json
// @Param request body GlossaryMineRequest false "{series_id?}"
// @Success 200 {object} APIResponse "miner.MineResult [@contract-v1] (single show)"
// @Success 202 {object} APIResponse "{started:true} (library sweep)"
// @Router /subtitles/glossary/mine [post]
func (h *GlossaryMineHandler) Start(c *gin.Context) {
	var req GlossaryMineRequest
	if c.Request.ContentLength > 0 {
		if err := c.ShouldBindJSON(&req); err != nil {
			ValidationError(c, "Invalid request: "+err.Error())
			return
		}
	}
	if q := strings.TrimSpace(c.Query("series_id")); q != "" && req.SeriesID == "" {
		req.SeriesID = q
	}

	if id := strings.TrimSpace(req.SeriesID); id != "" {
		res, err := h.miner.MineSeries(c.Request.Context(), id)
		switch {
		case errors.Is(err, miner.ErrMinerBusy):
			ErrorResponse(c, http.StatusConflict, "GLOSSARY_MINE_RUNNING",
				"正在從官方字幕學習中。", "等這一輪跑完再按。")
		case err != nil:
			// The run itself failed (series not found, episodes unreadable).
			// The result carries the reason; the status endpoint keeps it.
			ErrorResponse(c, http.StatusInternalServerError, "GLOSSARY_MINE_FAILED",
				"這部劇沒學成："+res.Error, "看看它的集數有沒有 file_path，或稍後再試。")
		default:
			SuccessResponse(c, res)
		}
		return
	}

	// Detached from the request: the sweep outlives the HTTP call. The miner
	// claims itself before StartPartial returns, so the very next GET already
	// says running=true.
	if err := h.miner.StartPartial(context.Background()); err != nil {
		if errors.Is(err, miner.ErrMinerBusy) {
			ErrorResponse(c, http.StatusConflict, "GLOSSARY_MINE_RUNNING",
				"正在從官方字幕學習中。", "等這一輪跑完再按。")
			return
		}
		ErrorResponse(c, http.StatusInternalServerError, "GLOSSARY_MINE_FAILED", "無法開始："+err.Error(), "稍後再試。")
		return
	}
	c.JSON(http.StatusAccepted, APIResponse{Success: true, Data: map[string]interface{}{"started": true}})
}
