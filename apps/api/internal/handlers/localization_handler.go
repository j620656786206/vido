package handlers

import (
	"errors"
	"log/slog"

	"github.com/gin-gonic/gin"
	"github.com/vido/api/internal/ai/prompts"
	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/services"
)

// LocalizationHandler serves the sub-7-4 localization dial:
//
//	GET /api/v1/subtitles/localization  → {level, source, levels[]}
//	PUT /api/v1/subtitles/localization  {level} → same shape
//
// The level is a TASTE setting (literal | standard | ott) — the settings page
// shows a three-way radio over it; the pipeline reads it per run.
type LocalizationHandler struct {
	service *services.LocalizationSettingsService
}

// NewLocalizationHandler builds a LocalizationHandler.
func NewLocalizationHandler(service *services.LocalizationSettingsService) *LocalizationHandler {
	return &LocalizationHandler{service: service}
}

// RegisterRoutes mounts the routes under /subtitles.
func (h *LocalizationHandler) RegisterRoutes(rg *gin.RouterGroup) {
	g := rg.Group("/subtitles/localization")
	{
		g.GET("", h.Get)
		g.PUT("", h.Put)
	}
}

// LocalizationResponse is the wire shape (snake_case per Rule 6).
type LocalizationResponse struct {
	Level  prompts.LocalizationLevel   `json:"level"`
	Source services.LocalizationSource `json:"source"`
	Levels []prompts.LocalizationLevel `json:"levels"`
}

// LocalizationUpdateRequest is the PUT body.
type LocalizationUpdateRequest struct {
	Level string `json:"level" binding:"required"`
}

func toLocalizationResponse(s services.LocalizationSettings) LocalizationResponse {
	return LocalizationResponse{Level: s.Level, Source: s.Source, Levels: prompts.LocalizationLevels}
}

// Get handles GET /api/v1/subtitles/localization.
func (h *LocalizationHandler) Get(c *gin.Context) {
	SuccessResponse(c, toLocalizationResponse(h.service.Resolve(c.Request.Context())))
}

// Put handles PUT /api/v1/subtitles/localization.
func (h *LocalizationHandler) Put(c *gin.Context) {
	var req LocalizationUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ValidationError(c, "level is required")
		return
	}
	saved, err := h.service.Set(c.Request.Context(), req.Level)
	if err != nil {
		var ve *models.ValidationError
		if errors.As(err, &ve) {
			ValidationError(c, ve.Error())
			return
		}
		slog.Error("localization handler error", "op", "set", "error", err)
		InternalServerError(c, "在地化程度儲存失敗")
		return
	}
	SuccessResponse(c, toLocalizationResponse(saved))
}
