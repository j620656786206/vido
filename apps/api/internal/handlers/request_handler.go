package handlers

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"

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
	// 13-2a AC #2: a malformed / unknown-to-TMDB / owned-overlapping partial
	// selection (code-list extension under the existing REQUEST_ prefix —
	// prefix count stays 16, no CR-workflow sync needed).
	errCodeRequestInvalidSelection = "REQUEST_INVALID_SELECTION"
	// 13-7a AC #6: cancel / retry on a row in the wrong state (code-list
	// extension under REQUEST_ — no new prefix, no CR-workflow sync).
	errCodeRequestNotCancellable = "REQUEST_NOT_CANCELLABLE"
	errCodeRequestNotRetryable   = "REQUEST_NOT_RETRYABLE"
	// The broken download could not be removed from Radarr/Sonarr, so the
	// retry did nothing and the row stays failed (13-7a CR).
	errCodeRequestRetryCleanupFailed = "REQUEST_RETRY_CLEANUP_FAILED"
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
		requests.GET("/tv/:tmdb_id/coverage", h.TVCoverage)
		requests.DELETE("/:id", h.CancelRequest)
		requests.POST("/:id/retry", h.RetryRequest)
	}
}

// CancelRequest handles DELETE /api/v1/requests/:id (13-7a AC #1
// [@contract-v1]) — only a pending request can be cancelled; it is deleted.
// @Summary Cancel a pending media request
// @Tags requests
// @Param id path string true "Request id"
// @Success 204 "Cancelled (row deleted)"
// @Failure 404 {object} APIResponse "DB_NOT_FOUND"
// @Failure 409 {object} APIResponse "REQUEST_NOT_CANCELLABLE — already being processed"
// @Failure 500 {object} APIResponse "INTERNAL_ERROR"
// @Router /api/v1/requests/{id} [delete]
func (h *RequestHandler) CancelRequest(c *gin.Context) {
	id := c.Param("id")
	if err := h.service.CancelRequest(c.Request.Context(), id); err != nil {
		switch {
		case errors.Is(err, repository.ErrRequestNotFound):
			requestNotFound(c)
		case errors.Is(err, services.ErrRequestNotCancellable):
			slog.Debug("Cancel rejected: request not pending", "request_id", id, "error", err)
			ErrorResponse(c, http.StatusConflict, errCodeRequestNotCancellable,
				"此請求已在處理中，無法取消",
				"重新整理想要清單查看最新狀態。")
		default:
			slog.Error("Failed to cancel request", "request_id", id, "error", err)
			InternalServerError(c, "取消請求失敗，請稍後再試")
		}
		return
	}
	NoContentResponse(c)
}

// RetryRequest handles POST /api/v1/requests/:id/retry (13-7a AC #2
// [@contract-v1]) — only a failed request can be retried; the response is
// the updated request resource (13-1a shape).
// @Summary Retry a failed media request
// @Tags requests
// @Produce json
// @Param id path string true "Request id"
// @Success 200 {object} APIResponse{data=models.Request}
// @Failure 404 {object} APIResponse "DB_NOT_FOUND"
// @Failure 409 {object} APIResponse "REQUEST_NOT_RETRYABLE | REQUEST_DUPLICATE"
// @Failure 502 {object} APIResponse "REQUEST_RETRY_CLEANUP_FAILED — the broken download could not be removed; nothing changed"
// @Failure 500 {object} APIResponse "INTERNAL_ERROR"
// @Router /api/v1/requests/{id}/retry [post]
func (h *RequestHandler) RetryRequest(c *gin.Context) {
	id := c.Param("id")
	request, err := h.service.RetryRequest(c.Request.Context(), id)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrRequestNotFound):
			requestNotFound(c)
		case errors.Is(err, services.ErrRequestNotRetryable):
			slog.Debug("Retry rejected: request not failed", "request_id", id, "error", err)
			ErrorResponse(c, http.StatusConflict, errCodeRequestNotRetryable,
				"這個請求沒有失敗，不需要重試",
				"重新整理想要清單查看最新狀態。")
		case errors.Is(err, repository.ErrRequestDuplicate):
			slog.Debug("Retry rejected: newer active request for the title", "request_id", id, "error", err)
			ErrorResponse(c, http.StatusConflict, errCodeRequestDuplicate,
				"這部片已有進行中的請求",
				"請至想要清單查看該請求的進度。")
		case errors.Is(err, services.ErrRetryCleanupFailed):
			slog.Warn("Retry rejected: failed download could not be removed", "request_id", id, "error", err)
			ErrorResponse(c, http.StatusBadGateway, errCodeRequestRetryCleanupFailed,
				"沒能清掉壞掉的下載，這次重試沒有生效",
				"確認 Radarr／Sonarr 連線正常後再按一次重試。")
		default:
			slog.Error("Failed to retry request", "request_id", id, "error", err)
			InternalServerError(c, "重試請求失敗，請稍後再試")
		}
		return
	}
	SuccessResponse(c, request)
}

// requestNotFound is the 404 for an unknown request id (DB_NOT_FOUND, the
// glossary/movie precedent) with a zh-TW message.
func requestNotFound(c *gin.Context) {
	ErrorResponse(c, http.StatusNotFound, "DB_NOT_FOUND",
		"找不到這筆請求",
		"它可能已被取消；重新整理想要清單。")
}

// TVCoverage handles GET /api/v1/requests/tv/:tmdb_id/coverage (13-2a AC #5
// [@contract-v1] — the 13-2b tree's owned/requested reflection).
// @Summary Owned/requested coverage for a TV show's season-episode tree
// @Tags requests
// @Produce json
// @Param tmdb_id path int true "TMDB TV id"
// @Success 200 {object} APIResponse{data=services.RequestCoverage}
// @Failure 400 {object} APIResponse "VALIDATION_INVALID_FORMAT"
// @Failure 500 {object} APIResponse "INTERNAL_ERROR"
// @Router /api/v1/requests/tv/{tmdb_id}/coverage [get]
func (h *RequestHandler) TVCoverage(c *gin.Context) {
	tmdbID, err := strconv.ParseInt(c.Param("tmdb_id"), 10, 64)
	if err != nil || tmdbID <= 0 {
		ErrorResponse(c, http.StatusBadRequest, "VALIDATION_INVALID_FORMAT",
			"tmdb_id 必須是正整數", "請確認網址中的 tmdb_id。")
		return
	}
	coverage, err := h.service.TVCoverage(c.Request.Context(), tmdbID)
	if err != nil {
		slog.Error("Failed to load request coverage", "tmdb_id", tmdbID, "error", err)
		InternalServerError(c, "無法載入請求範圍資訊")
		return
	}
	SuccessResponse(c, coverage)
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
	var selectionErr *services.InvalidSelectionError
	if errors.As(err, &selectionErr) {
		slog.Debug("Request rejected: invalid selection", "error", err)
		// Reason is the clean zh-TW half — the English sentinel diagnostic
		// stays out of the Rule 3 envelope.
		ErrorResponse(c, http.StatusBadRequest, errCodeRequestInvalidSelection,
			selectionErr.Reason,
			"請重新勾選要請求的季或集後再試一次。")
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
