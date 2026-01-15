package handlers

import (
	"context"
	"log/slog"

	"github.com/gin-gonic/gin"
	"github.com/vido/api/internal/models"
)

// SettingsServiceInterface defines the contract for settings business operations.
// This interface enables testing handlers with mock services.
type SettingsServiceInterface interface {
	Set(ctx context.Context, setting *models.Setting) error
	Get(ctx context.Context, key string) (*models.Setting, error)
	GetAll(ctx context.Context) ([]models.Setting, error)
	Delete(ctx context.Context, key string) error
	GetString(ctx context.Context, key string) (string, error)
	GetInt(ctx context.Context, key string) (int, error)
	GetBool(ctx context.Context, key string) (bool, error)
	SetString(ctx context.Context, key, value string) error
	SetInt(ctx context.Context, key string, value int) error
	SetBool(ctx context.Context, key string, value bool) error
}

// SettingsHandler handles HTTP requests for application settings.
// It uses SettingsServiceInterface for business logic, following the
// Handler → Service → Repository → Database architecture.
type SettingsHandler struct {
	service SettingsServiceInterface
}

// NewSettingsHandler creates a new SettingsHandler with the given service.
func NewSettingsHandler(service SettingsServiceInterface) *SettingsHandler {
	return &SettingsHandler{
		service: service,
	}
}

// SetSettingRequest represents the request body for setting a value
type SetSettingRequest struct {
	Key   string      `json:"key" binding:"required"`
	Value interface{} `json:"value" binding:"required"`
	Type  string      `json:"type" binding:"required,oneof=string int bool"`
}

// List handles GET /api/v1/settings
// Returns all settings
func (h *SettingsHandler) List(c *gin.Context) {
	settings, err := h.service.GetAll(c.Request.Context())
	if err != nil {
		slog.Error("Failed to list settings", "error", err)
		InternalServerError(c, "Failed to retrieve settings")
		return
	}

	SuccessResponse(c, settings)
}

// Get handles GET /api/v1/settings/:key
// Returns a single setting by key
func (h *SettingsHandler) Get(c *gin.Context) {
	key := c.Param("key")
	if key == "" {
		BadRequestError(c, "VALIDATION_REQUIRED_FIELD", "Setting key is required")
		return
	}

	setting, err := h.service.Get(c.Request.Context(), key)
	if err != nil {
		slog.Error("Failed to get setting", "error", err, "key", key)
		NotFoundError(c, "Setting")
		return
	}

	SuccessResponse(c, setting)
}

// Set handles POST /api/v1/settings
// Creates or updates a setting
func (h *SettingsHandler) Set(c *gin.Context) {
	var req SetSettingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ValidationError(c, "Invalid request body: "+err.Error())
		return
	}

	var err error
	ctx := c.Request.Context()

	switch req.Type {
	case "string":
		strVal, ok := req.Value.(string)
		if !ok {
			ValidationError(c, "Value must be a string for type 'string'")
			return
		}
		err = h.service.SetString(ctx, req.Key, strVal)
	case "int":
		// JSON numbers come as float64
		floatVal, ok := req.Value.(float64)
		if !ok {
			ValidationError(c, "Value must be a number for type 'int'")
			return
		}
		err = h.service.SetInt(ctx, req.Key, int(floatVal))
	case "bool":
		boolVal, ok := req.Value.(bool)
		if !ok {
			ValidationError(c, "Value must be a boolean for type 'bool'")
			return
		}
		err = h.service.SetBool(ctx, req.Key, boolVal)
	default:
		ValidationError(c, "Invalid type: must be 'string', 'int', or 'bool'")
		return
	}

	if err != nil {
		slog.Error("Failed to set setting", "error", err, "key", req.Key)
		InternalServerError(c, "Failed to save setting")
		return
	}

	// Return the saved setting
	setting, err := h.service.Get(ctx, req.Key)
	if err != nil {
		slog.Error("Failed to retrieve saved setting", "error", err, "key", req.Key)
		InternalServerError(c, "Setting saved but failed to retrieve")
		return
	}

	SuccessResponse(c, setting)
}

// Delete handles DELETE /api/v1/settings/:key
// Deletes a setting by key
func (h *SettingsHandler) Delete(c *gin.Context) {
	key := c.Param("key")
	if key == "" {
		BadRequestError(c, "VALIDATION_REQUIRED_FIELD", "Setting key is required")
		return
	}

	if err := h.service.Delete(c.Request.Context(), key); err != nil {
		slog.Error("Failed to delete setting", "error", err, "key", key)
		InternalServerError(c, "Failed to delete setting")
		return
	}

	NoContentResponse(c)
}

// RegisterRoutes registers all settings routes on the given router group
func (h *SettingsHandler) RegisterRoutes(rg *gin.RouterGroup) {
	settings := rg.Group("/settings")
	{
		settings.GET("", h.List)
		settings.GET("/:key", h.Get)
		settings.POST("", h.Set)
		settings.DELETE("/:key", h.Delete)
	}
}
