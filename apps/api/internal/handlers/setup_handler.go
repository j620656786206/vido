package handlers

import (
	"context"
	"errors"
	"log/slog"

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
		BadRequestError(c, "SETUP_VALIDATION_FAILED", err.Error())
		return
	}

	SuccessResponse(c, gin.H{
		"valid": true,
	})
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
