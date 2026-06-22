package handlers

import (
	"github.com/gin-gonic/gin"
	"github.com/vido/api/internal/services"
)

// ActivityHandler serves the background-job activity aggregate for the v2 Activity hub
// (UX Redesign D4-1 / ux3-2-1).
type ActivityHandler struct {
	svc *services.ActivityService
}

// NewActivityHandler creates a new ActivityHandler.
func NewActivityHandler(svc *services.ActivityService) *ActivityHandler {
	return &ActivityHandler{svc: svc}
}

// RegisterRoutes registers GET /api/v1/activity.
func (h *ActivityHandler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET("/activity", h.GetActivity)
}

// GetActivity handles GET /api/v1/activity. The service is fail-soft per section, so this
// never returns an error envelope — a degraded section reports its own "unavailable"
// status within the payload (ADR B1/F3).
func (h *ActivityHandler) GetActivity(c *gin.Context) {
	SuccessResponse(c, h.svc.GetActivity(c.Request.Context()))
}
