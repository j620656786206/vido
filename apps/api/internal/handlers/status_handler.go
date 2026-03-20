package handlers

import (
	"errors"
	"log/slog"

	"github.com/gin-gonic/gin"
	"github.com/vido/api/internal/services"
)

// StatusHandler handles HTTP requests for service connection status dashboard
type StatusHandler struct {
	statusService services.ServiceStatusServiceInterface
}

// NewStatusHandler creates a new StatusHandler
func NewStatusHandler(statusService services.ServiceStatusServiceInterface) *StatusHandler {
	return &StatusHandler{
		statusService: statusService,
	}
}

// RegisterRoutes registers service status routes
func (h *StatusHandler) RegisterRoutes(rg *gin.RouterGroup) {
	status := rg.Group("/settings/services")
	{
		status.GET("", h.GetAllServiceStatuses)
		status.POST("/:name/test", h.TestServiceConnection)
	}
}

// GetAllServiceStatuses handles GET /api/v1/settings/services
// Returns connection status for all external services
func (h *StatusHandler) GetAllServiceStatuses(c *gin.Context) {
	statuses, err := h.statusService.GetAllStatuses(c.Request.Context())
	if err != nil {
		slog.Error("Failed to get service statuses", "error", err)
		InternalServerError(c, "Failed to retrieve service statuses")
		return
	}

	SuccessResponse(c, gin.H{
		"services": statuses,
	})
}

// TestServiceConnection handles POST /api/v1/settings/services/:name/test
// Manually tests connectivity for a specific service
func (h *StatusHandler) TestServiceConnection(c *gin.Context) {
	name := c.Param("name")
	if name == "" {
		BadRequestError(c, "VALIDATION_REQUIRED_FIELD", "Service name is required")
		return
	}

	status, err := h.statusService.TestService(c.Request.Context(), name)
	if err != nil {
		if errors.Is(err, services.ErrServiceNotFound) {
			BadRequestError(c, "SERVICE_NOT_FOUND", "Unknown service name: "+name)
			return
		}
		slog.Error("Failed to test service connection", "service", name, "error", err)
		InternalServerError(c, "Failed to test service connection")
		return
	}

	SuccessResponse(c, status)
}
