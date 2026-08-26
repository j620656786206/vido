package handlers

import (
	"github.com/gin-gonic/gin"
	"github.com/vido/api/internal/services"
)

// HomeSummaryHandler serves the Home v3 readout-band aggregate
// (Story ux3-1-6, tech-spec D1).
type HomeSummaryHandler struct {
	svc *services.HomeSummaryService
}

// NewHomeSummaryHandler creates a new HomeSummaryHandler.
func NewHomeSummaryHandler(svc *services.HomeSummaryService) *HomeSummaryHandler {
	return &HomeSummaryHandler{svc: svc}
}

// RegisterRoutes registers GET /api/v1/home-summary.
func (h *HomeSummaryHandler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET("/home-summary", h.GetSummary)
}

// GetSummary handles GET /api/v1/home-summary. The service is fail-soft per
// cell, so this never returns an error envelope — a degraded cell reports its
// own "unavailable" status within the payload (ADR B1/F3).
func (h *HomeSummaryHandler) GetSummary(c *gin.Context) {
	SuccessResponse(c, h.svc.GetSummary(c.Request.Context()))
}
