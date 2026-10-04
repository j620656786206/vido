package handlers

import (
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/vido/api/internal/services"
)

// UsageReportHandler serves the opt-in anonymous usage report switch
// (infra-optin-usage-report-a2, PRD P1-040):
//
//	GET /api/v1/settings/usage-report            → status
//	PUT /api/v1/settings/usage-report {enabled}  → status
//
// Registered BEFORE settingsHandler, or /settings/:key would take
// "usage-report" as a setting key.
type UsageReportHandler struct {
	service services.UsageReportServiceInterface
}

// NewUsageReportHandler builds a UsageReportHandler.
func NewUsageReportHandler(service services.UsageReportServiceInterface) *UsageReportHandler {
	return &UsageReportHandler{service: service}
}

// RegisterRoutes mounts the routes under /settings/usage-report.
func (h *UsageReportHandler) RegisterRoutes(rg *gin.RouterGroup) {
	g := rg.Group("/settings/usage-report")
	{
		g.GET("", h.Get)
		g.PUT("", h.Put)
	}
}

// UsageReportResponse is the wire shape (snake_case, Rule 6).
type UsageReportResponse struct {
	// Available is false when this build has no receiver (local / fork build).
	Available bool `json:"available"`
	Enabled   bool `json:"enabled"`
	// LastSentAt / LastPayload describe the last SUCCESSFUL send; null = never.
	// LastPayload is the exact body that was sent, verbatim (P1-040-3).
	LastSentAt  *time.Time `json:"last_sent_at"`
	LastPayload *string    `json:"last_payload"`
}

// UsageReportUpdateRequest is the PUT body. A pointer so a missing field is a
// validation error, not a silent "false".
type UsageReportUpdateRequest struct {
	Enabled *bool `json:"enabled"`
}

func toUsageReportResponse(s services.UsageReportStatus) UsageReportResponse {
	return UsageReportResponse{
		Available:   s.Available,
		Enabled:     s.Enabled,
		LastSentAt:  s.LastSentAt,
		LastPayload: s.LastPayload,
	}
}

// Get handles GET /api/v1/settings/usage-report.
// @Summary      Opt-in anonymous usage report status
// @Description  Whether the report is available in this build and turned on, and the last successfully sent body (verbatim) with its time.
// @Tags         settings
// @Produce      json
// @Success      200  {object}  APIResponse{data=UsageReportResponse}
// @Failure      500  {object}  APIResponse
// @Router       /api/v1/settings/usage-report [get]
func (h *UsageReportHandler) Get(c *gin.Context) {
	st, err := h.service.Status(c.Request.Context())
	if err != nil {
		slog.Error("usage report handler error", "op", "status", "error", err)
		InternalServerError(c, "匿名使用回報的狀態讀取失敗")
		return
	}
	SuccessResponse(c, toUsageReportResponse(st))
}

// Put handles PUT /api/v1/settings/usage-report.
// @Summary      Turn the opt-in anonymous usage report on or off
// @Tags         settings
// @Accept       json
// @Produce      json
// @Param        body  body      UsageReportUpdateRequest  true  "enabled"
// @Success      200   {object}  APIResponse{data=UsageReportResponse}
// @Failure      400   {object}  APIResponse
// @Failure      500   {object}  APIResponse
// @Router       /api/v1/settings/usage-report [put]
func (h *UsageReportHandler) Put(c *gin.Context) {
	var req UsageReportUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Enabled == nil {
		ValidationError(c, "enabled is required")
		return
	}
	st, err := h.service.SetEnabled(c.Request.Context(), *req.Enabled)
	if err != nil {
		slog.Error("usage report handler error", "op", "set", "error", err)
		InternalServerError(c, "匿名使用回報的設定儲存失敗")
		return
	}
	SuccessResponse(c, toUsageReportResponse(st))
}
