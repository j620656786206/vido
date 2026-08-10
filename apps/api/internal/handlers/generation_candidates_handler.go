package handlers

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/vido/api/internal/services"
)

// GenerationCandidateAnalyzerInterface is the narrow surface this handler
// drives (Rule 11, mirroring GenerationBatchProcessorInterface).
type GenerationCandidateAnalyzerInterface interface {
	Analyze(ctx context.Context, progress services.AnalysisProgress) (*services.GenerationCandidateResult, error)
}

// GenerationCandidatesHandler serves the cost-preview list (story sub-4-1):
// which items would need PAID speech recognition, and roughly what the whole
// selection would cost — answered WITHOUT spending anything.
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

// RegisterRoutes mounts the candidates route.
func (h *GenerationCandidatesHandler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET("/subtitles/generation-candidates", h.ListCandidates)
}

// ListCandidates godoc
// @Summary      List subtitle-generation candidates with cost estimates
// @Description  Enumerates every movie and episode missing a zh-Hant subtitle, classifies each one as extract (a usable embedded text track exists) or asr (paid speech recognition), and returns a per-item plus aggregate USD ESTIMATE. Probes files that carry no scan-time track data; never extracts or transcribes, so calling this costs nothing.
// @Tags         subtitles
// @Produce      json
// @Success      200  {object}  APIResponse  "candidates + summary"
// @Failure      500  {object}  APIResponse  "DB_QUERY_FAILED"
// @Router       /api/v1/subtitles/generation-candidates [get]
func (h *GenerationCandidatesHandler) ListCandidates(c *gin.Context) {
	// Progress is nil here: this is the plain request/response shape. The
	// analysis pass can be long on a library full of episodes (one ffprobe per
	// un-enriched file), and surfacing that as SSE progress for the F14 screen
	// is wired separately — the service already emits per-item progress.
	result, err := h.analyzer.Analyze(c.Request.Context(), nil)
	if err != nil {
		ErrorResponse(c, http.StatusInternalServerError, "DB_QUERY_FAILED",
			"無法列出可產生字幕的項目："+err.Error(),
			"檢查媒體庫是否可存取，或稍後再試。")
		return
	}

	SuccessResponse(c, result)
}
