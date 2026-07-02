package handlers

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/qbittorrent"
	"github.com/vido/api/internal/services"
)

// SetupRequiredMarker is the substring appended to QBITTORRENT_NOT_CONFIGURED
// suggestion fields so the frontend can branch programmatically without parsing
// zh-TW copy. See bugfix-10-2 [@contract-v1] AC #3.
const SetupRequiredMarker = "SETUP_REQUIRED"

// qbtErrorToHTTPStatus maps a qBittorrent ConnectionError.Code to the
// semantically correct HTTP status. Used by the three GET endpoints in this
// file to keep the contract single-sourced (bugfix-10-2 [@contract-v1]).
// Unknown codes fall through to 502 with a warn log so a future qBT error
// code addition surfaces in observability rather than silently 502'ing.
func qbtErrorToHTTPStatus(code string) int {
	switch code {
	case qbittorrent.ErrCodeNotConfigured:
		return http.StatusServiceUnavailable
	case qbittorrent.ErrCodeTimeout:
		return http.StatusGatewayTimeout
	case qbittorrent.ErrCodeAuthFailed, qbittorrent.ErrCodeConnectionFailed:
		return http.StatusBadGateway
	default:
		slog.Warn("unknown qBT error code mapped to 502", "code", code)
		return http.StatusBadGateway
	}
}

// DownloadItem extends Torrent with parse status information.
type DownloadItem struct {
	qbittorrent.Torrent
	ParseStatus *DownloadParseStatus `json:"parse_status,omitempty"`
}

// DownloadParseStatus represents the parse status for a download.
type DownloadParseStatus struct {
	Status       models.ParseJobStatus `json:"status"`
	ErrorMessage *string               `json:"error_message,omitempty"`
	MediaID      *string               `json:"media_id,omitempty"`
}

// DownloadHandler handles HTTP requests for download monitoring.
type DownloadHandler struct {
	service       services.DownloadServiceInterface
	parseQueueSvc services.ParseQueueServiceInterface
}

// NewDownloadHandler creates a new DownloadHandler.
func NewDownloadHandler(service services.DownloadServiceInterface, parseQueueSvc ...services.ParseQueueServiceInterface) *DownloadHandler {
	h := &DownloadHandler{service: service}
	if len(parseQueueSvc) > 0 && parseQueueSvc[0] != nil {
		h.parseQueueSvc = parseQueueSvc[0]
	}
	return h
}

// ListDownloads handles GET /api/v1/downloads
// @Summary List downloads with pagination
// @Description Retrieves torrents from qBittorrent with filtering, sorting, and pagination
// @Tags downloads
// @Accept json
// @Produce json
// @Param filter query string false "Filter by status (all, downloading, paused, completed, seeding, error)" default(all)
// @Param sort query string false "Sort field (added_on, name, progress, size)" default(added_on)
// @Param order query string false "Sort order (asc, desc)" default(desc)
// @Param page query int false "Page number (1-based)" default(1)
// @Param pageSize query int false "Items per page (50, 100, 200, 500)" default(100)
// @Success 200 {object} APIResponse
// @Failure 502 {object} APIResponse
// @Failure 503 {object} APIResponse
// @Failure 504 {object} APIResponse
// @Failure 500 {object} APIResponse
// @Router /api/v1/downloads [get]
func (h *DownloadHandler) ListDownloads(c *gin.Context) {
	filter := c.DefaultQuery("filter", "all")
	sort := c.DefaultQuery("sort", "added_on")
	order := c.DefaultQuery("order", "desc")
	page := parseIntQuery(c, "page", 1)
	pageSize := parseIntQuery(c, "pageSize", 100)

	// Clamp pageSize to allowed values
	if pageSize < 1 {
		pageSize = 100
	} else if pageSize > 500 {
		pageSize = 500
	}
	if page < 1 {
		page = 1
	}

	torrents, err := h.service.GetAllDownloads(c.Request.Context(), filter, sort, order)
	if err != nil {
		slog.Error("Failed to list downloads", "error", err)

		var connErr *qbittorrent.ConnectionError
		if errors.As(err, &connErr) {
			status := qbtErrorToHTTPStatus(connErr.Code)
			switch connErr.Code {
			case qbittorrent.ErrCodeNotConfigured:
				ErrorResponse(c, status, connErr.Code, "qBittorrent 尚未設定", "請先設定 qBittorrent 連線。"+SetupRequiredMarker)
			case qbittorrent.ErrCodeAuthFailed:
				ErrorResponse(c, status, connErr.Code, "qBittorrent 認證失敗", "請檢查帳號密碼是否正確。")
			default:
				ErrorResponse(c, status, connErr.Code, "無法連線到 qBittorrent", connErr.Error())
			}
			return
		}

		InternalServerError(c, "Failed to retrieve downloads")
		return
	}

	// Enrich with parse status if service is available
	var allItems []DownloadItem
	if h.parseQueueSvc != nil {
		allItems = make([]DownloadItem, len(torrents))
		for i, t := range torrents {
			allItems[i] = DownloadItem{Torrent: t}
			if t.Status == qbittorrent.StatusCompleted || t.Status == qbittorrent.StatusSeeding {
				if job, err := h.parseQueueSvc.GetJobStatus(c.Request.Context(), t.Hash); err == nil && job != nil {
					allItems[i].ParseStatus = &DownloadParseStatus{
						Status:       job.Status,
						ErrorMessage: job.ErrorMessage,
						MediaID:      job.MediaID,
					}
				}
			}
		}
	} else {
		allItems = make([]DownloadItem, len(torrents))
		for i, t := range torrents {
			allItems[i] = DownloadItem{Torrent: t}
		}
	}

	// Apply pagination
	total := len(allItems)
	totalPages := (total + pageSize - 1) / pageSize
	start := (page - 1) * pageSize
	end := start + pageSize
	if start > total {
		start = total
	}
	if end > total {
		end = total
	}

	SuccessResponse(c, PaginatedResponse{
		Items:      allItems[start:end],
		Page:       page,
		PageSize:   pageSize,
		TotalItems: total,
		TotalPages: totalPages,
	})
}

// parseIntQuery parses an integer query parameter with a default value.
func parseIntQuery(c *gin.Context, key string, defaultVal int) int {
	val := c.Query(key)
	if val == "" {
		return defaultVal
	}
	n, err := strconv.Atoi(val)
	if err != nil {
		return defaultVal
	}
	return n
}

// GetDownloadDetails handles GET /api/v1/downloads/:hash
// @Summary Get download details
// @Description Retrieves detailed information for a specific torrent by hash
// @Tags downloads
// @Accept json
// @Produce json
// @Param hash path string true "Torrent info hash"
// @Success 200 {object} APIResponse
// @Failure 404 {object} APIResponse
// @Failure 502 {object} APIResponse
// @Failure 503 {object} APIResponse
// @Failure 504 {object} APIResponse
// @Failure 500 {object} APIResponse
// @Router /api/v1/downloads/{hash} [get]
func (h *DownloadHandler) GetDownloadDetails(c *gin.Context) {
	hash := c.Param("hash")
	if hash == "" {
		ValidationError(c, "torrent hash is required")
		return
	}

	details, err := h.service.GetDownloadDetails(c.Request.Context(), hash)
	if err != nil {
		slog.Error("Failed to get download details", "error", err, "hash", hash)

		var connErr *qbittorrent.ConnectionError
		if errors.As(err, &connErr) {
			status := qbtErrorToHTTPStatus(connErr.Code)
			switch connErr.Code {
			case qbittorrent.ErrCodeTorrentNotFound:
				NotFoundError(c, "torrent")
			case qbittorrent.ErrCodeNotConfigured:
				ErrorResponse(c, status, connErr.Code, "qBittorrent 尚未設定", "請先設定 qBittorrent 連線。"+SetupRequiredMarker)
			case qbittorrent.ErrCodeAuthFailed:
				ErrorResponse(c, status, connErr.Code, "qBittorrent 認證失敗", "請檢查帳號密碼是否正確。")
			default:
				ErrorResponse(c, status, connErr.Code, "無法連線到 qBittorrent", connErr.Error())
			}
			return
		}

		InternalServerError(c, "Failed to retrieve download details")
		return
	}

	SuccessResponse(c, details)
}

// GetDownloadCounts handles GET /api/v1/downloads/counts
// @Summary Get download counts by status
// @Description Returns the count of torrents grouped by status
// @Tags downloads
// @Produce json
// @Success 200 {object} APIResponse
// @Failure 502 {object} APIResponse
// @Failure 503 {object} APIResponse
// @Failure 504 {object} APIResponse
// @Failure 500 {object} APIResponse
// @Router /api/v1/downloads/counts [get]
func (h *DownloadHandler) GetDownloadCounts(c *gin.Context) {
	counts, err := h.service.GetDownloadCounts(c.Request.Context())
	if err != nil {
		slog.Error("Failed to get download counts", "error", err)

		var connErr *qbittorrent.ConnectionError
		if errors.As(err, &connErr) {
			status := qbtErrorToHTTPStatus(connErr.Code)
			switch connErr.Code {
			case qbittorrent.ErrCodeNotConfigured:
				ErrorResponse(c, status, connErr.Code, "qBittorrent 尚未設定", "請先設定 qBittorrent 連線。"+SetupRequiredMarker)
			case qbittorrent.ErrCodeAuthFailed:
				ErrorResponse(c, status, connErr.Code, "qBittorrent 認證失敗", "請檢查帳號密碼是否正確。")
			default:
				ErrorResponse(c, status, connErr.Code, "無法連線到 qBittorrent", connErr.Error())
			}
			return
		}

		InternalServerError(c, "Failed to retrieve download counts")
		return
	}

	SuccessResponse(c, counts)
}

// writeActionError maps an error from a download action (pause/resume/remove) to
// the correct HTTP response, reusing the qbtErrorToHTTPStatus contract shared by
// the GET endpoints (bugfix-10-2 [@contract-v1]). No TorrentNotFound branch —
// qBittorrent's pause/resume/delete are idempotent and 200 even for unknown
// hashes, so there is no not-found surface for these actions.
func (h *DownloadHandler) writeActionError(c *gin.Context, err error, action string) {
	slog.Error("download action failed", "action", action, "error", err)

	var connErr *qbittorrent.ConnectionError
	if errors.As(err, &connErr) {
		status := qbtErrorToHTTPStatus(connErr.Code)
		switch connErr.Code {
		case qbittorrent.ErrCodeNotConfigured:
			ErrorResponse(c, status, connErr.Code, "qBittorrent 尚未設定", "請先設定 qBittorrent 連線。"+SetupRequiredMarker)
		case qbittorrent.ErrCodeAuthFailed:
			ErrorResponse(c, status, connErr.Code, "qBittorrent 認證失敗", "請檢查帳號密碼是否正確。")
		default:
			ErrorResponse(c, status, connErr.Code, "無法連線到 qBittorrent", connErr.Error())
		}
		return
	}

	InternalServerError(c, "Failed to "+action+" download")
}

// PauseDownload handles POST /api/v1/downloads/:hash/pause
// @Summary Pause a download
// @Description Pauses the torrent with the given hash. [@contract-v1]
// @Tags downloads
// @Produce json
// @Param hash path string true "Torrent info hash"
// @Success 200 {object} APIResponse
// @Failure 400 {object} APIResponse
// @Failure 502 {object} APIResponse
// @Failure 503 {object} APIResponse
// @Failure 504 {object} APIResponse
// @Failure 500 {object} APIResponse
// @Router /api/v1/downloads/{hash}/pause [post]
func (h *DownloadHandler) PauseDownload(c *gin.Context) {
	hash := c.Param("hash")
	if hash == "" {
		ValidationError(c, "torrent hash is required")
		return
	}

	if err := h.service.PauseDownload(c.Request.Context(), hash); err != nil {
		h.writeActionError(c, err, "pause")
		return
	}

	SuccessResponse(c, nil)
}

// ResumeDownload handles POST /api/v1/downloads/:hash/resume
// @Summary Resume a download
// @Description Resumes the torrent with the given hash. [@contract-v1]
// @Tags downloads
// @Produce json
// @Param hash path string true "Torrent info hash"
// @Success 200 {object} APIResponse
// @Failure 400 {object} APIResponse
// @Failure 502 {object} APIResponse
// @Failure 503 {object} APIResponse
// @Failure 504 {object} APIResponse
// @Failure 500 {object} APIResponse
// @Router /api/v1/downloads/{hash}/resume [post]
func (h *DownloadHandler) ResumeDownload(c *gin.Context) {
	hash := c.Param("hash")
	if hash == "" {
		ValidationError(c, "torrent hash is required")
		return
	}

	if err := h.service.ResumeDownload(c.Request.Context(), hash); err != nil {
		h.writeActionError(c, err, "resume")
		return
	}

	SuccessResponse(c, nil)
}

// RemoveDownload handles DELETE /api/v1/downloads/:hash
// @Summary Remove a download
// @Description Removes the torrent with the given hash from qBittorrent. When
// @Description deleteFiles=true the downloaded data is also deleted from disk;
// @Description when false (default) the files are kept. [@contract-v1]
// @Tags downloads
// @Produce json
// @Param hash path string true "Torrent info hash"
// @Param deleteFiles query bool false "Also delete downloaded files from disk" default(false)
// @Success 200 {object} APIResponse
// @Failure 400 {object} APIResponse
// @Failure 502 {object} APIResponse
// @Failure 503 {object} APIResponse
// @Failure 504 {object} APIResponse
// @Failure 500 {object} APIResponse
// @Router /api/v1/downloads/{hash} [delete]
func (h *DownloadHandler) RemoveDownload(c *gin.Context) {
	hash := c.Param("hash")
	if hash == "" {
		ValidationError(c, "torrent hash is required")
		return
	}

	deleteFiles, _ := strconv.ParseBool(c.DefaultQuery("deleteFiles", "false"))

	if err := h.service.RemoveDownload(c.Request.Context(), hash, deleteFiles); err != nil {
		h.writeActionError(c, err, "remove")
		return
	}

	SuccessResponse(c, nil)
}

// RegisterRoutes registers download monitoring routes.
func (h *DownloadHandler) RegisterRoutes(rg *gin.RouterGroup) {
	downloads := rg.Group("/downloads")
	{
		downloads.GET("", h.ListDownloads)
		downloads.GET("/counts", h.GetDownloadCounts)
		downloads.GET("/:hash", h.GetDownloadDetails)
		downloads.POST("/:hash/pause", h.PauseDownload)
		downloads.POST("/:hash/resume", h.ResumeDownload)
		downloads.DELETE("/:hash", h.RemoveDownload)
	}
}
