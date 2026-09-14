package handlers

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/services"
)

// SetupServiceInterface defines the contract for setup wizard operations.
type SetupServiceInterface interface {
	IsFirstRun(ctx context.Context) (bool, error)
	CompleteSetup(ctx context.Context, config models.SetupConfig) error
	ValidateStep(ctx context.Context, step string, data map[string]interface{}) error
}

// ValidateStepRequest is the request body for step validation.
type ValidateStepRequest struct {
	Step string                 `json:"step" binding:"required"`
	Data map[string]interface{} `json:"data" binding:"required"`
}

// SetupHandler handles HTTP requests for the setup wizard.
type SetupHandler struct {
	service SetupServiceInterface
}

// NewSetupHandler creates a new SetupHandler.
func NewSetupHandler(service SetupServiceInterface) *SetupHandler {
	return &SetupHandler{service: service}
}

// GetStatus handles GET /api/v1/setup/status
// Returns whether setup is needed.
func (h *SetupHandler) GetStatus(c *gin.Context) {
	needsSetup, err := h.service.IsFirstRun(c.Request.Context())
	if err != nil {
		slog.Error("Failed to check setup status", "error", err)
		InternalServerError(c, "Failed to check setup status")
		return
	}

	SuccessResponse(c, gin.H{
		"needsSetup": needsSetup,
	})
}

// Complete handles POST /api/v1/setup/complete
// Accepts setup config and saves all settings.
func (h *SetupHandler) Complete(c *gin.Context) {
	var config models.SetupConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		ValidationError(c, "Invalid request body: "+err.Error())
		return
	}

	if err := h.service.CompleteSetup(c.Request.Context(), config); err != nil {
		slog.Error("Failed to complete setup", "error", err)
		if errors.Is(err, services.ErrKeysNotWritable) {
			keysNotWritable(c)
			return
		}
		if errors.Is(err, services.ErrSetupAlreadyCompleted) {
			BadRequestError(c, "SETUP_ALREADY_COMPLETED", "Setup wizard has already been completed")
			return
		}
		InternalServerError(c, "Failed to complete setup")
		return
	}

	SuccessResponse(c, gin.H{
		"message": "Setup completed successfully",
	})
}

// ValidateStep handles POST /api/v1/setup/validate-step
// Validates individual step data.
func (h *SetupHandler) ValidateStep(c *gin.Context) {
	var req ValidateStepRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ValidationError(c, "Invalid request body: "+err.Error())
		return
	}

	if err := h.service.ValidateStep(c.Request.Context(), req.Step, req.Data); err != nil {
		slog.Info("Setup step validation failed", "step", req.Step, "error", err)
		if errors.Is(err, services.ErrKeysNotWritable) {
			keysNotWritable(c)
			return
		}
		BadRequestError(c, "SETUP_VALIDATION_FAILED", err.Error())
		return
	}

	SuccessResponse(c, gin.H{
		"valid": true,
	})
}

// keysNotWritable answers a wizard that was given API keys the server cannot
// store safely (no ENCRYPTION_KEY) — the same refusal the settings page gives
// (key_settings_handler.go). The wizard shows only `message`, so the message
// itself says what to do next (dsr-13).
func keysNotWritable(c *gin.Context) {
	ErrorResponse(c, http.StatusConflict, "SETUP_KEYS_NOT_WRITABLE",
		"伺服器沒有設定 ENCRYPTION_KEY，無法安全儲存 API 金鑰。請按「跳過」，設定好之後再到「設定 › API 金鑰」填寫。",
		"請設定 ENCRYPTION_KEY 環境變數後重啟伺服器。")
}

// RegisterRoutes registers all setup routes on the given router group.
func (h *SetupHandler) RegisterRoutes(rg *gin.RouterGroup) {
	setup := rg.Group("/setup")
	{
		setup.GET("/status", h.GetStatus)
		setup.POST("/complete", h.Complete)
		setup.POST("/validate-step", h.ValidateStep)
	}
}
