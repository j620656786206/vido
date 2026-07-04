package handlers

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/repository"
	"github.com/vido/api/internal/services"
	"github.com/vido/api/internal/tmdb"
)

// Error codes (Rule 7 — REQUEST_ is the 15th registered prefix, Story 13-1a
// AC #6; list extension + CR-workflow sync ship in the same change).
const (
	errCodeRequestDuplicate        = "REQUEST_DUPLICATE"
	errCodeRequestAlreadyInLibrary = "REQUEST_ALREADY_IN_LIBRARY"
)

// RequestHandler handles HTTP requests for the media request system.
// Story 13-1a (G-1/P3-001, Epic 13). [@contract-v1] on the create/list shape.
type RequestHandler struct {
	service services.RequestServiceInterface
}

// NewRequestHandler builds a new handler.
func NewRequestHandler(service services.RequestServiceInterface) *RequestHandler {
	return &RequestHandler{service: service}
}

// RegisterRoutes mounts the request routes under the provided API group.
func (h *RequestHandler) RegisterRoutes(rg *gin.RouterGroup) {
	requests := rg.Group("/requests")
	{
		requests.GET("", h.ListRequests)
		requests.POST("", h.CreateRequest)
	}
}

// ListRequests handles GET /api/v1/requests
// @Summary List media requests (newest first)
// @Tags requests
// @Produce json
// @Success 200 {object} APIResponse{data=object}
// @Failure 500 {object} APIResponse "INTERNAL_ERROR"
// @Router /api/v1/requests [get]
func (h *RequestHandler) ListRequests(c *gin.Context) {
	requests, err := h.service.ListRequests(c.Request.Context())
	if err != nil {
		slog.Error("Failed to list requests", "error", err)
		InternalServerError(c, "無法載入請求清單")
		return
	}
	// Never send null — the UI expects an array ([@contract-v1] AC #3).
	if requests == nil {
		requests = []models.Request{}
	}
	SuccessResponse(c, gin.H{"requests": requests})
}

// CreateRequest handles POST /api/v1/requests
// @Summary Create a media request (one-click 想要)
// @Tags requests
// @Accept json
// @Produce json
// @Success 201 {object} APIResponse{data=models.Request}
// @Failure 400 {object} APIResponse "VALIDATION_REQUIRED_FIELD | VALIDATION_INVALID_FORMAT"
// @Failure 404 {object} APIResponse "TMDB_NOT_FOUND — unknown tmdb_id"
// @Failure 409 {object} APIResponse "REQUEST_DUPLICATE | REQUEST_ALREADY_IN_LIBRARY"
// @Router /api/v1/requests [post]
func (h *RequestHandler) CreateRequest(c *gin.Context) {
	var req services.CreateMediaRequestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ErrorResponse(c, http.StatusBadRequest, "VALIDATION_INVALID_FORMAT",
			"請求格式錯誤："+err.Error(),
			"請確認 tmdb_id 與 media_type 欄位格式。")
		return
	}

	request, err := h.service.CreateRequest(c.Request.Context(), req)
	if err != nil {
		handleRequestError(c, err)
		return
	}
	CreatedResponse(c, request)
}

// handleRequestError maps service errors to HTTP responses (Rule 3 envelope,
// zh-TW messages per story AC #2). Expected 4xx flows (duplicate, owned,
// unknown tmdb_id, validation) log at Debug — they are normal user behavior,
// not system faults (CR M2); only unexpected failures log at Error.
func handleRequestError(c *gin.Context, err error) {
	if errors.Is(err, repository.ErrRequestDuplicate) {
		slog.Debug("Request rejected: active duplicate", "error", err)
		ErrorResponse(c, http.StatusConflict, errCodeRequestDuplicate,
			"已有進行中的請求",
			"請至想要清單查看該請求的進度。")
		return
	}
	if errors.Is(err, services.ErrRequestAlreadyInLibrary) {
		slog.Debug("Request rejected: already in library", "error", err)
		ErrorResponse(c, http.StatusConflict, errCodeRequestAlreadyInLibrary,
			"此片已在媒體庫中",
			"請直接在媒體庫中觀看。")
		return
	}
	// TMDb resolve failures pass through typed (TMDB_NOT_FOUND on a bad
	// tmdb_id arrives with its own status code + zh-TW-ready messaging).
	var tmdbErr *tmdb.TMDbError
	if errors.As(err, &tmdbErr) {
		slog.Debug("Request rejected: tmdb resolve failed", "error_code", tmdbErr.Code, "error", err)
		status := tmdbErr.StatusCode
		if status == 0 {
			status = http.StatusBadGateway
		}
		ErrorResponse(c, status, tmdbErr.Code, tmdbErr.Message, tmdbErr.Suggestion)
		return
	}
	var validationErr *models.ValidationError
	if errors.As(err, &validationErr) {
		slog.Debug("Request rejected: validation", "field", validationErr.Field, "error", err)
		code := "VALIDATION_INVALID_FORMAT"
		if validationErr.Field == "tmdb_id" {
			// A zero/absent tmdb_id is a missing required field, not a format issue.
			code = "VALIDATION_REQUIRED_FIELD"
		}
		BadRequestError(c, code, validationErr.Message)
		return
	}
	slog.Error("Failed to create request", "error", err)
	InternalServerError(c, "建立請求失敗，請稍後再試")
}
