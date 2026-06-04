package handlers

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/repository"
	"github.com/vido/api/internal/services"
)

// Error codes (Rule 7 — {SOURCE}_{ERROR_TYPE}).
const (
	errCodeFilterPresetNotFound   = "FILTER_PRESET_NOT_FOUND"
	errCodeFilterPresetValidation = "FILTER_PRESET_VALIDATION_FAILED"
	errCodeFilterPresetLimit      = "FILTER_PRESET_LIMIT_REACHED"
)

// FilterPresetsHandler handles HTTP requests for saved filter preset CRUD.
// Story 11.4 (P2-015).
type FilterPresetsHandler struct {
	service services.FilterPresetServiceInterface
}

// NewFilterPresetsHandler builds a new handler.
func NewFilterPresetsHandler(service services.FilterPresetServiceInterface) *FilterPresetsHandler {
	return &FilterPresetsHandler{service: service}
}

// RegisterRoutes mounts the filter-preset routes under the provided API group.
func (h *FilterPresetsHandler) RegisterRoutes(rg *gin.RouterGroup) {
	presets := rg.Group("/filter-presets")
	{
		presets.GET("", h.ListPresets)
		presets.POST("", h.CreatePreset)
		presets.DELETE("/:id", h.DeletePreset)
	}
}

// ListPresets handles GET /api/v1/filter-presets
// @Summary List saved filter presets
// @Tags filter-presets
// @Produce json
// @Success 200 {object} APIResponse{data=object}
// @Router /api/v1/filter-presets [get]
func (h *FilterPresetsHandler) ListPresets(c *gin.Context) {
	presets, err := h.service.GetAllPresets(c.Request.Context())
	if err != nil {
		slog.Error("Failed to list filter presets", "error", err)
		InternalServerError(c, "Failed to list filter presets")
		return
	}
	// Never send null — the UI expects an array.
	if presets == nil {
		presets = []models.FilterPreset{}
	}
	SuccessResponse(c, gin.H{"presets": presets})
}

// CreatePreset handles POST /api/v1/filter-presets
// @Summary Create a filter preset
// @Tags filter-presets
// @Accept json
// @Produce json
// @Success 201 {object} APIResponse{data=models.FilterPreset}
// @Router /api/v1/filter-presets [post]
func (h *FilterPresetsHandler) CreatePreset(c *gin.Context) {
	var req services.CreateFilterPresetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequestError(c, errCodeFilterPresetValidation, "Invalid request body: "+err.Error())
		return
	}

	preset, err := h.service.CreatePreset(c.Request.Context(), req)
	if err != nil {
		slog.Error("Failed to create filter preset", "error", err)
		handleFilterPresetError(c, err)
		return
	}
	CreatedResponse(c, preset)
}

// DeletePreset handles DELETE /api/v1/filter-presets/:id
// @Summary Delete a filter preset
// @Tags filter-presets
// @Produce json
// @Success 200 {object} APIResponse
// @Router /api/v1/filter-presets/{id} [delete]
func (h *FilterPresetsHandler) DeletePreset(c *gin.Context) {
	id := c.Param("id")
	if err := h.service.DeletePreset(c.Request.Context(), id); err != nil {
		slog.Error("Failed to delete filter preset", "id", id, "error", err)
		handleFilterPresetError(c, err)
		return
	}
	SuccessResponse(c, gin.H{"deleted": true})
}

// handleFilterPresetError maps service errors to HTTP responses.
func handleFilterPresetError(c *gin.Context, err error) {
	if errors.Is(err, repository.ErrFilterPresetNotFound) {
		ErrorResponse(c, http.StatusNotFound, errCodeFilterPresetNotFound,
			"Filter preset not found",
			"Verify the preset ID is correct.")
		return
	}
	if errors.Is(err, services.ErrFilterPresetLimitReached) {
		ErrorResponse(c, http.StatusConflict, errCodeFilterPresetLimit,
			err.Error(),
			"Delete an existing preset before saving a new one.")
		return
	}
	var validationErr *models.ValidationError
	if errors.As(err, &validationErr) {
		BadRequestError(c, errCodeFilterPresetValidation, err.Error())
		return
	}
	InternalServerError(c, err.Error())
}
