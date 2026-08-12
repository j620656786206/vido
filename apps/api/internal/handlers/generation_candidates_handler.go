package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/vido/api/internal/services"
)

// GenerationCandidateAnalyzerInterface is the narrow surface this handler
// drives (Rule 11, mirroring GenerationBatchProcessorInterface).
type GenerationCandidateAnalyzerInterface interface {
	StartAnalysis() error
	CancelAnalysis()
	Snapshot() services.AnalysisSnapshot
}

// GenerationCandidatesHandler serves the cost-preview list (story sub-4-1;
// AC #7 [@contract-v1] — the GET state envelope {status, analyzed, total,
// result?, analyzed_at?, error?}; additive default_budget_usd since sub-5-1,
// no bump — FE consumer sub-4-3 consent flow): which items would need PAID
// speech recognition, and roughly what the whole selection would cost —
// answered WITHOUT spending anything.
//
// It is a sibling of, not a replacement for, the existing
// `/subtitles/generation-batch/preview` endpoint, which still returns a bare
// count and still has live frontend consumers.
type GenerationCandidatesHandler struct {
	analyzer GenerationCandidateAnalyzerInterface
}

func NewGenerationCandidatesHandler(analyzer GenerationCandidateAnalyzerInterface) *GenerationCandidatesHandler {
	return &GenerationCandidatesHandler{analyzer: analyzer}
}

// RegisterRoutes mounts the candidates routes.
func (h *GenerationCandidatesHandler) RegisterRoutes(rg *gin.RouterGroup) {
	gc := rg.Group("/subtitles/generation-candidates")
	{
		gc.GET("", h.ListCandidates)
		gc.POST("/analyze", h.StartAnalysis)
		gc.POST("/analyze/cancel", h.CancelAnalysis)
	}
}

// StartAnalysis godoc
// @Summary      Start the subtitle-generation cost analysis
// @Description  Kicks off the probe sweep that classifies every candidate as extract or paid speech recognition. Returns immediately; progress arrives on the generation_candidates_progress SSE event and the result via GET. Probing is local and free — this endpoint spends nothing.
// @Tags         subtitles
// @Produce      json
// @Success      202  {object}  APIResponse  "{started:true}"
// @Failure      409  {object}  APIResponse  "TRANSCRIPTION_ANALYSIS_RUNNING — an analysis is already in flight"
// @Router       /api/v1/subtitles/generation-candidates/analyze [post]
func (h *GenerationCandidatesHandler) StartAnalysis(c *gin.Context) {
	if err := h.analyzer.StartAnalysis(); err != nil {
		if errors.Is(err, services.ErrAnalysisRunning) {
			// Its own code, not a reuse of TRANSCRIPTION_BATCH_RUNNING (CR
			// sub-4-1 M1): that code means "a Route C generation batch is
			// running" — paid work in flight. This 409 means the FREE analysis
			// sweep is running. A client keying UI copy off the code would
			// warn about spending that is not happening.
			ErrorResponse(c, http.StatusConflict, "TRANSCRIPTION_ANALYSIS_RUNNING",
				"分析正在進行中。", "等待目前的分析結束，或先取消它。")
			return
		}
		ErrorResponse(c, http.StatusInternalServerError, "DB_QUERY_FAILED",
			"無法開始分析："+err.Error(), "請稍後再試。")
		return
	}
	c.JSON(http.StatusAccepted, APIResponse{Success: true, Data: map[string]interface{}{"started": true}})
}

// CancelAnalysis godoc
// @Summary      Cancel the running cost analysis
// @Description  Stops an in-flight probe sweep. Any partial classification is discarded — a fraction of the library priced as if it were all of it would be a misleading quote.
// @Tags         subtitles
// @Produce      json
// @Success      200  {object}  APIResponse  "{cancelled:true}"
// @Router       /api/v1/subtitles/generation-candidates/analyze/cancel [post]
func (h *GenerationCandidatesHandler) CancelAnalysis(c *gin.Context) {
	h.analyzer.CancelAnalysis()
	SuccessResponse(c, map[string]interface{}{"cancelled": true})
}

// ListCandidates godoc
// @Summary      Get subtitle-generation candidates and their cost estimates
// @Description  Returns the current state of the cost analysis: its status, how far it has got, and — once ready — every candidate with its route (extract vs paid speech recognition) and USD estimate, plus the aggregate. Reading this never triggers work; POST /analyze does.
// @Tags         subtitles
// @Produce      json
// @Success      200  {object}  APIResponse  "{status, analyzed, total, result?, analyzed_at?, error?}"
// @Router       /api/v1/subtitles/generation-candidates [get]
func (h *GenerationCandidatesHandler) ListCandidates(c *gin.Context) {
	// Always 200 with a state envelope rather than 404/425 for "not analyzed
	// yet": the client renders the analysis screen (F14) and the list (F15)
	// from the same payload, so a status field is what it actually needs.
	SuccessResponse(c, h.analyzer.Snapshot())
}
