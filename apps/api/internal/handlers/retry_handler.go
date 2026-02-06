package handlers

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/vido/api/internal/retry"
	"github.com/vido/api/internal/services"
)

// RetryHandler handles HTTP requests for retry queue operations.
// It uses services.RetryServiceInterface for business logic, following the
// Handler → Service → Repository → Database architecture.
type RetryHandler struct {
	service services.RetryServiceInterface
}

// NewRetryHandler creates a new RetryHandler with the given service.
func NewRetryHandler(service services.RetryServiceInterface) *RetryHandler {
	return &RetryHandler{
		service: service,
	}
}

// PendingRetriesResponse represents the response for GET /api/v1/retry/pending
type PendingRetriesResponse struct {
	Items []*retry.RetryItemResponse `json:"items"`
	Stats *retry.RetryStats          `json:"stats"`
}

// GetPending handles GET /api/v1/retry/pending
// Returns all pending retry items with countdown timers
func (h *RetryHandler) GetPending(c *gin.Context) {
	items, err := h.service.GetPendingRetries(c.Request.Context())
	if err != nil {
		slog.Error("Failed to get pending retries", "error", err)
		InternalServerError(c, "無法取得重試隊列")
		return
	}

	stats, err := h.service.GetRetryStats(c.Request.Context())
	if err != nil {
		slog.Error("Failed to get retry stats", "error", err)
		// Continue with nil stats
		stats = &retry.RetryStats{}
	}

	// Convert items to response format
	responseItems := make([]*retry.RetryItemResponse, len(items))
	for i, item := range items {
		responseItems[i] = item.ToResponse()
	}

	SuccessResponse(c, PendingRetriesResponse{
		Items: responseItems,
		Stats: stats,
	})
}

// TriggerImmediate handles POST /api/v1/retry/:id/trigger
// Triggers an immediate retry for the specified item
func (h *RetryHandler) TriggerImmediate(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		BadRequestError(c, "RETRY_INVALID_ID", "Retry ID is required")
		return
	}

	slog.Info("Triggering immediate retry", "id", id)

	err := h.service.TriggerImmediate(c.Request.Context(), id)
	if err != nil {
		slog.Error("Failed to trigger immediate retry", "error", err, "id", id)

		// Check for not found
		if containsString(err.Error(), "not found") {
			ErrorResponse(c, http.StatusNotFound, "RETRY_NOT_FOUND",
				"找不到此重試項目",
				"重試項目可能已完成或被取消。")
			return
		}

		InternalServerError(c, "無法觸發立即重試")
		return
	}

	slog.Info("Immediate retry triggered", "id", id)

	SuccessResponse(c, map[string]interface{}{
		"id":      id,
		"message": "已觸發立即重試",
	})
}

// Cancel handles DELETE /api/v1/retry/:id
// Cancels a pending retry item
func (h *RetryHandler) Cancel(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		BadRequestError(c, "RETRY_INVALID_ID", "Retry ID is required")
		return
	}

	slog.Info("Cancelling retry", "id", id)

	err := h.service.CancelRetry(c.Request.Context(), id)
	if err != nil {
		slog.Error("Failed to cancel retry", "error", err, "id", id)

		// Check for not found
		if containsString(err.Error(), "not found") {
			ErrorResponse(c, http.StatusNotFound, "RETRY_NOT_FOUND",
				"找不到此重試項目",
				"重試項目可能已完成或被取消。")
			return
		}

		InternalServerError(c, "無法取消重試")
		return
	}

	slog.Info("Retry cancelled", "id", id)

	NoContentResponse(c)
}

// GetByID handles GET /api/v1/retry/:id
// Returns details of a specific retry item
func (h *RetryHandler) GetByID(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		BadRequestError(c, "RETRY_INVALID_ID", "Retry ID is required")
		return
	}

	item, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		slog.Error("Failed to get retry item", "error", err, "id", id)
		InternalServerError(c, "無法取得重試項目")
		return
	}

	if item == nil {
		ErrorResponse(c, http.StatusNotFound, "RETRY_NOT_FOUND",
			"找不到此重試項目",
			"重試項目可能已完成或被取消。")
		return
	}

	SuccessResponse(c, item.ToResponse())
}

// helper function to check if string contains substring
func containsString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// RegisterRetryRoutes registers the retry routes on a Gin engine
func RegisterRetryRoutes(router *gin.RouterGroup, handler *RetryHandler) {
	retry := router.Group("/retry")
	{
		retry.GET("/pending", handler.GetPending)
		retry.GET("/:id", handler.GetByID)
		retry.POST("/:id/trigger", handler.TriggerImmediate)
		retry.DELETE("/:id", handler.Cancel)
	}
}
