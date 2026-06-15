package handlers

import (
	"github.com/gin-gonic/gin"
	"github.com/vido/api/internal/services"
)

// StatusSummaryHandler serves the ambient NAS-status summary for the v2 sidebar-footer
// status strip (UX Redesign D4-2 / ux3-0-3).
type StatusSummaryHandler struct {
	svc *services.StatusSummaryService
}

// NewStatusSummaryHandler creates a new StatusSummaryHandler.
func NewStatusSummaryHandler(svc *services.StatusSummaryService) *StatusSummaryHandler {
	return &StatusSummaryHandler{svc: svc}
}

// RegisterRoutes registers GET /api/v1/status/summary.
func (h *StatusSummaryHandler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET("/status/summary", h.GetStatusSummary)
}

// GetStatusSummary handles GET /api/v1/status/summary. The service is fail-soft per
// section, so this never returns an error envelope — a degraded section reports its
// own "unavailable" status within the payload (ADR B1/F3).
func (h *StatusSummaryHandler) GetStatusSummary(c *gin.Context) {
	SuccessResponse(c, h.svc.GetSummary(c.Request.Context()))
}
