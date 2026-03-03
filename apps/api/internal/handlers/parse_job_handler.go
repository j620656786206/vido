package handlers

import (
	"errors"
	"log/slog"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/vido/api/internal/services"
)

// ParseJobHandler handles HTTP requests for parse job status operations.
type ParseJobHandler struct {
	service services.ParseQueueServiceInterface
}

// NewParseJobHandler creates a new ParseJobHandler.
func NewParseJobHandler(service services.ParseQueueServiceInterface) *ParseJobHandler {
	return &ParseJobHandler{service: service}
}

// GetParseStatus handles GET /api/v1/downloads/:hash/parse-status
// @Summary Get parse status for a download
// @Description Returns the parse job status for a specific torrent by hash
// @Tags downloads, parse-jobs
// @Produce json
// @Param hash path string true "Torrent info hash"
// @Success 200 {object} APIResponse
// @Failure 400 {object} APIResponse
// @Failure 404 {object} APIResponse
// @Failure 500 {object} APIResponse
// @Router /api/v1/downloads/{hash}/parse-status [get]
func (h *ParseJobHandler) GetParseStatus(c *gin.Context) {
	hash := c.Param("hash")
	if hash == "" {
		ValidationError(c, "torrent hash is required")
		return
	}

	job, err := h.service.GetJobStatus(c.Request.Context(), hash)
	if err != nil {
		slog.Error("Failed to get parse status", "error", err, "hash", hash)
		InternalServerError(c, "Failed to retrieve parse status")
		return
	}

	if job == nil {
		NotFoundError(c, "parse job")
		return
	}

	SuccessResponse(c, job)
}

// ListJobs handles GET /api/v1/parse-jobs
// @Summary List all parse jobs
// @Description Returns all parse jobs ordered by creation time descending
// @Tags parse-jobs
// @Produce json
// @Param limit query int false "Maximum number of jobs to return" default(50)
// @Success 200 {object} APIResponse
// @Failure 500 {object} APIResponse
// @Router /api/v1/parse-jobs [get]
func (h *ParseJobHandler) ListJobs(c *gin.Context) {
	limit := 50
	if limitStr := c.Query("limit"); limitStr != "" {
		if parsed, err := strconv.Atoi(limitStr); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	jobs, err := h.service.ListJobs(c.Request.Context(), limit)
	if err != nil {
		slog.Error("Failed to list parse jobs", "error", err)
		InternalServerError(c, "Failed to retrieve parse jobs")
		return
	}

	SuccessResponse(c, jobs)
}

// RetryParseJob handles POST /api/v1/parse-jobs/:id/retry
// @Summary Retry a failed parse job
// @Description Resets a failed parse job to pending for re-processing
// @Tags parse-jobs
// @Produce json
// @Param id path string true "Parse job ID"
// @Success 200 {object} APIResponse
// @Failure 400 {object} APIResponse
// @Failure 500 {object} APIResponse
// @Router /api/v1/parse-jobs/{id}/retry [post]
func (h *ParseJobHandler) RetryParseJob(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		ValidationError(c, "parse job ID is required")
		return
	}

	err := h.service.RetryJob(c.Request.Context(), id)
	if err != nil {
		slog.Error("Failed to retry parse job", "error", err, "id", id)

		if errors.Is(err, services.ErrJobNotRetryable) || errors.Is(err, services.ErrMaxRetriesReached) {
			BadRequestError(c, "PARSE_JOB_NOT_RETRYABLE", err.Error())
			return
		}
		if strings.Contains(err.Error(), "not found") {
			NotFoundError(c, "parse job")
			return
		}

		InternalServerError(c, "Failed to retry parse job")
		return
	}

	SuccessResponse(c, map[string]interface{}{
		"id":      id,
		"message": "Parse job queued for retry",
	})
}

// RegisterRoutes registers parse job routes.
func (h *ParseJobHandler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET("/downloads/:hash/parse-status", h.GetParseStatus)

	parseJobs := rg.Group("/parse-jobs")
	{
		parseJobs.GET("", h.ListJobs)
		parseJobs.POST("/:id/retry", h.RetryParseJob)
	}
}
