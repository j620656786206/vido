package handlers

import (
	"errors"
	"log/slog"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/services"
)

// LogHandler handles HTTP requests for system log viewing
type LogHandler struct {
	service services.LogServiceInterface
}

// NewLogHandler creates a new LogHandler
func NewLogHandler(service services.LogServiceInterface) *LogHandler {
	return &LogHandler{service: service}
}

// GetLogs handles GET /api/v1/settings/logs
func (h *LogHandler) GetLogs(c *gin.Context) {
	filter := models.LogFilter{
		Level:   models.LogLevel(c.Query("level")),
		Keyword: c.Query("keyword"),
	}

	if pageStr := c.Query("page"); pageStr != "" {
		if page, err := strconv.Atoi(pageStr); err == nil {
			filter.Page = page
		}
	}

	if perPageStr := c.Query("per_page"); perPageStr != "" {
		if perPage, err := strconv.Atoi(perPageStr); err == nil {
			filter.PerPage = perPage
		}
	}

	result, err := h.service.GetLogs(c.Request.Context(), filter)
	if err != nil {
		if errors.Is(err, services.ErrInvalidLogLevel) {
			BadRequestError(c, "VALIDATION_INVALID_FORMAT", err.Error())
			return
		}
		slog.Error("Failed to get system logs", "error", err)
		InternalServerError(c, "Failed to retrieve system logs")
		return
	}

	SuccessResponse(c, result)
}

// ClearLogs handles DELETE /api/v1/settings/logs
func (h *LogHandler) ClearLogs(c *gin.Context) {
	daysStr := c.Query("older_than_days")
	if daysStr == "" {
		BadRequestError(c, "VALIDATION_REQUIRED_FIELD", "older_than_days query parameter is required")
		return
	}

	days, err := strconv.Atoi(daysStr)
	if err != nil || days <= 0 {
		BadRequestError(c, "VALIDATION_INVALID_FORMAT", "older_than_days must be a positive integer")
		return
	}

	result, err := h.service.ClearLogs(c.Request.Context(), days)
	if err != nil {
		slog.Error("Failed to clear system logs", "error", err, "days", days)
		InternalServerError(c, "Failed to clear system logs")
		return
	}

	SuccessResponse(c, result)
}

// RegisterRoutes registers log viewer routes
func (h *LogHandler) RegisterRoutes(rg *gin.RouterGroup) {
	logs := rg.Group("/settings/logs")
	{
		logs.GET("", h.GetLogs)
		logs.DELETE("", h.ClearLogs)
	}
}
