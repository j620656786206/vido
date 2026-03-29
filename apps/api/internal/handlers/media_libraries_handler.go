package handlers

import (
	"errors"
	"log/slog"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/vido/api/internal/repository"
	"github.com/vido/api/internal/services"
)

// MediaLibrariesHandler handles HTTP requests for media library CRUD management.
type MediaLibrariesHandler struct {
	service services.MediaLibraryServiceInterface
}

// NewMediaLibrariesHandler creates a new MediaLibrariesHandler.
func NewMediaLibrariesHandler(service services.MediaLibraryServiceInterface) *MediaLibrariesHandler {
	return &MediaLibrariesHandler{service: service}
}

// RegisterRoutes registers media library management routes on the given router group.
func (h *MediaLibrariesHandler) RegisterRoutes(rg *gin.RouterGroup) {
	libs := rg.Group("/libraries")
	{
		libs.GET("", h.ListLibraries)
		libs.POST("", h.CreateLibrary)
		libs.PUT("/:id", h.UpdateLibrary)
		libs.DELETE("/:id", h.DeleteLibrary)
		libs.POST("/:id/paths", h.AddPath)
		libs.DELETE("/:id/paths/:pathId", h.RemovePath)
		libs.POST("/:id/paths/refresh", h.RefreshPaths)
	}
}

// ListLibraries handles GET /api/v1/libraries
func (h *MediaLibrariesHandler) ListLibraries(c *gin.Context) {
	libraries, err := h.service.GetAllLibraries(c.Request.Context())
	if err != nil {
		slog.Error("Failed to list media libraries", "error", err)
		InternalServerError(c, "Failed to list libraries")
		return
	}
	SuccessResponse(c, gin.H{"libraries": libraries})
}

// CreateLibrary handles POST /api/v1/libraries
func (h *MediaLibrariesHandler) CreateLibrary(c *gin.Context) {
	var req services.CreateLibraryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ValidationError(c, "Invalid request body: "+err.Error())
		return
	}

	library, err := h.service.CreateLibrary(c.Request.Context(), req)
	if err != nil {
		slog.Error("Failed to create library", "error", err)
		handleMediaLibraryError(c, err)
		return
	}

	CreatedResponse(c, library)
}

// UpdateLibrary handles PUT /api/v1/libraries/:id
func (h *MediaLibrariesHandler) UpdateLibrary(c *gin.Context) {
	id := c.Param("id")

	var req services.UpdateLibraryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ValidationError(c, "Invalid request body: "+err.Error())
		return
	}

	library, err := h.service.UpdateLibrary(c.Request.Context(), id, req)
	if err != nil {
		slog.Error("Failed to update library", "id", id, "error", err)
		handleMediaLibraryError(c, err)
		return
	}

	SuccessResponse(c, library)
}

// DeleteLibrary handles DELETE /api/v1/libraries/:id
func (h *MediaLibrariesHandler) DeleteLibrary(c *gin.Context) {
	id := c.Param("id")
	removeMedia := c.Query("remove_media") == "true"

	if err := h.service.DeleteLibrary(c.Request.Context(), id, removeMedia); err != nil {
		slog.Error("Failed to delete library", "id", id, "error", err)
		handleMediaLibraryError(c, err)
		return
	}

	SuccessResponse(c, gin.H{"deleted": true})
}

// AddPath handles POST /api/v1/libraries/:id/paths
func (h *MediaLibrariesHandler) AddPath(c *gin.Context) {
	libraryID := c.Param("id")

	var req struct {
		Path string `json:"path" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		ValidationError(c, "Invalid request body: path is required")
		return
	}

	path, err := h.service.AddPath(c.Request.Context(), libraryID, req.Path)
	if err != nil {
		slog.Error("Failed to add path", "library_id", libraryID, "error", err)
		handleMediaLibraryError(c, err)
		return
	}

	CreatedResponse(c, path)
}

// RemovePath handles DELETE /api/v1/libraries/:id/paths/:pathId
func (h *MediaLibrariesHandler) RemovePath(c *gin.Context) {
	pathID := c.Param("pathId")

	if err := h.service.RemovePath(c.Request.Context(), pathID); err != nil {
		slog.Error("Failed to remove path", "path_id", pathID, "error", err)
		handleMediaLibraryError(c, err)
		return
	}

	SuccessResponse(c, gin.H{"deleted": true})
}

// RefreshPaths handles POST /api/v1/libraries/:id/paths/refresh
func (h *MediaLibrariesHandler) RefreshPaths(c *gin.Context) {
	libraryID := c.Param("id")

	paths, err := h.service.RefreshPathStatuses(c.Request.Context(), libraryID)
	if err != nil {
		slog.Error("Failed to refresh paths", "library_id", libraryID, "error", err)
		handleMediaLibraryError(c, err)
		return
	}

	SuccessResponse(c, gin.H{"paths": paths})
}

// handleMediaLibraryError maps service errors to appropriate HTTP responses.
func handleMediaLibraryError(c *gin.Context, err error) {
	if errors.Is(err, repository.ErrLibraryNotFound) {
		NotFoundError(c, "library")
		return
	}
	if errors.Is(err, repository.ErrLibraryPathNotFound) {
		NotFoundError(c, "library path")
		return
	}
	if errors.Is(err, repository.ErrLibraryPathDuplicate) {
		ErrorResponse(c, 409, "LIBRARY_DUPLICATE_PATH", "Path already exists in a library", "Use a different path")
		return
	}
	errMsg := err.Error()
	if strings.Contains(errMsg, "validation:") || strings.Contains(errMsg, "is required") || strings.Contains(errMsg, "must be") {
		BadRequestError(c, "VALIDATION_FAILED", errMsg)
		return
	}
	InternalServerError(c, errMsg)
}
