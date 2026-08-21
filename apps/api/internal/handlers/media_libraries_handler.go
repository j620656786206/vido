package handlers

import (
	"errors"
	"log/slog"

	"github.com/gin-gonic/gin"
	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/repository"
	"github.com/vido/api/internal/services"
)

// MediaLibrariesHandler handles HTTP requests for media library CRUD management.
type MediaLibrariesHandler struct {
	service services.MediaLibraryServiceInterface
	// autoSubtitleSupported answers "does THIS deployment run the free
	// auto-generation lane at all" (補審 M4).
	//
	// The `auto_subtitle` column is writable in every mode, but the trigger
	// that honours it is built only inside main.go's
	// `if cfg.SubtitlePipelineEnabled()` block — and the shipped default is
	// `legacy` (config.go, docs/deployment.md). Without this the settings page
	// offers an opt-in that, on a default install, can never do anything and
	// says nothing about it: the user ticks the box, saves successfully, and
	// waits forever. Capability honor, the transcription-handler precedent: a
	// deployment that cannot do the thing does not offer it.
	autoSubtitleSupported func() bool
}

// MediaLibrariesHandlerOption configures optional handler capabilities.
type MediaLibrariesHandlerOption func(*MediaLibrariesHandler)

// WithAutoSubtitleSupport supplies the predicate behind
// `auto_subtitle_supported` in the list response. Wired from
// cfg.SubtitlePipelineEnabled.
func WithAutoSubtitleSupport(supported func() bool) MediaLibrariesHandlerOption {
	return func(h *MediaLibrariesHandler) {
		if supported != nil {
			h.autoSubtitleSupported = supported
		}
	}
}

// NewMediaLibrariesHandler creates a new MediaLibrariesHandler.
//
// Unwired, the capability reports FALSE: a caller that never declared the
// pipeline does not get to advertise a lane it may not have.
func NewMediaLibrariesHandler(service services.MediaLibraryServiceInterface, opts ...MediaLibrariesHandlerOption) *MediaLibrariesHandler {
	h := &MediaLibrariesHandler{
		service:               service,
		autoSubtitleSupported: func() bool { return false },
	}
	for _, opt := range opts {
		opt(h)
	}
	return h
}

// RegisterRoutes registers media library management routes on the given router group.
func (h *MediaLibrariesHandler) RegisterRoutes(rg *gin.RouterGroup) {
	libs := rg.Group("/libraries")
	{
		libs.GET("", h.ListLibraries)
		libs.GET("/:id", h.GetLibrary)
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
	// `auto_subtitle_supported` rides the LIST response rather than a new
	// endpoint: the settings page that hosts the opt-in already fetches this
	// exact payload, so the capability costs zero extra requests and zero new
	// wire surface (additive on v1).
	SuccessResponse(c, gin.H{
		"libraries":               libraries,
		"auto_subtitle_supported": h.autoSubtitleSupported(),
	})
}

// GetLibrary handles GET /api/v1/libraries/:id
func (h *MediaLibrariesHandler) GetLibrary(c *gin.Context) {
	id := c.Param("id")

	library, err := h.service.GetLibrary(c.Request.Context(), id)
	if err != nil {
		slog.Error("Failed to get library", "id", id, "error", err)
		handleMediaLibraryError(c, err)
		return
	}

	SuccessResponse(c, library)
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
	libraryID := c.Param("id")
	pathID := c.Param("pathId")

	if err := h.service.RemovePath(c.Request.Context(), libraryID, pathID); err != nil {
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
	var validationErr *models.ValidationError
	if errors.As(err, &validationErr) {
		BadRequestError(c, "VALIDATION_FAILED", err.Error())
		return
	}
	InternalServerError(c, err.Error())
}
