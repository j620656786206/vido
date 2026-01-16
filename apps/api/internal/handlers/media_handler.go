package handlers

import (
	"log/slog"

	"github.com/gin-gonic/gin"
	"github.com/vido/api/internal/services"
)

// MediaHandler handles HTTP requests for media directory configuration.
// It uses services.MediaServiceInterface for business logic, following the
// Handler -> Service architecture (no repository needed as config is from env).
type MediaHandler struct {
	service services.MediaServiceInterface
}

// NewMediaHandler creates a new MediaHandler with the given service.
func NewMediaHandler(service services.MediaServiceInterface) *MediaHandler {
	return &MediaHandler{
		service: service,
	}
}

// GetMediaDirectories handles GET /api/v1/settings/media-directories
// Returns the list of all configured media directories with their status.
func (h *MediaHandler) GetMediaDirectories(c *gin.Context) {
	config := h.service.GetConfig()
	slog.Info("Retrieved media directories",
		"total", config.TotalCount,
		"valid", config.ValidCount,
		"search_only_mode", config.SearchOnlyMode)
	SuccessResponse(c, config)
}

// RefreshMediaDirectories handles POST /api/v1/settings/media-directories/refresh
// Re-validates all configured directories and returns the updated status.
// Useful when directories may have been mounted/unmounted at runtime.
func (h *MediaHandler) RefreshMediaDirectories(c *gin.Context) {
	slog.Info("Refreshing media directory status")
	config := h.service.RefreshDirectoryStatus()
	SuccessResponse(c, config)
}

// RegisterRoutes registers all media directory routes on the given router group.
// Routes are registered under /settings/media-directories.
func (h *MediaHandler) RegisterRoutes(rg *gin.RouterGroup) {
	media := rg.Group("/settings/media-directories")
	{
		media.GET("", h.GetMediaDirectories)
		media.POST("/refresh", h.RefreshMediaDirectories)
	}
}
