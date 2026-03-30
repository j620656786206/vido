package handlers

import (
	"errors"
	"log/slog"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/qbittorrent"
	"github.com/vido/api/internal/services"
)

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
// @Failure 400 {object} APIResponse
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
			switch connErr.Code {
			case qbittorrent.ErrCodeNotConfigured:
				ErrorResponse(c, 400, connErr.Code, "qBittorrent 尚未設定", "請先設定 qBittorrent 連線。")
			case qbittorrent.ErrCodeAuthFailed:
				ErrorResponse(c, 400, connErr.Code, "qBittorrent 認證失敗", "請檢查帳號密碼是否正確。")
			default:
				ErrorResponse(c, 400, connErr.Code, "無法連線到 qBittorrent", connErr.Error())
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
// @Failure 400 {object} APIResponse
// @Failure 404 {object} APIResponse
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
			switch connErr.Code {
			case qbittorrent.ErrCodeTorrentNotFound:
				NotFoundError(c, "torrent")
			case qbittorrent.ErrCodeNotConfigured:
				ErrorResponse(c, 400, connErr.Code, "qBittorrent 尚未設定", "請先設定 qBittorrent 連線。")
			default:
				ErrorResponse(c, 400, connErr.Code, "無法連線到 qBittorrent", connErr.Error())
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
// @Failure 400 {object} APIResponse
// @Failure 500 {object} APIResponse
// @Router /api/v1/downloads/counts [get]
func (h *DownloadHandler) GetDownloadCounts(c *gin.Context) {
	counts, err := h.service.GetDownloadCounts(c.Request.Context())
	if err != nil {
		slog.Error("Failed to get download counts", "error", err)

		var connErr *qbittorrent.ConnectionError
		if errors.As(err, &connErr) {
			switch connErr.Code {
			case qbittorrent.ErrCodeNotConfigured:
				ErrorResponse(c, 400, connErr.Code, "qBittorrent 尚未設定", "請先設定 qBittorrent 連線。")
			case qbittorrent.ErrCodeAuthFailed:
				ErrorResponse(c, 400, connErr.Code, "qBittorrent 認證失敗", "請檢查帳號密碼是否正確。")
			default:
				ErrorResponse(c, 400, connErr.Code, "無法連線到 qBittorrent", connErr.Error())
			}
			return
		}

		InternalServerError(c, "Failed to retrieve download counts")
		return
	}

	SuccessResponse(c, counts)
}

// RegisterRoutes registers download monitoring routes.
func (h *DownloadHandler) RegisterRoutes(rg *gin.RouterGroup) {
	downloads := rg.Group("/downloads")
	{
		downloads.GET("", h.ListDownloads)
		downloads.GET("/counts", h.GetDownloadCounts)
		downloads.GET("/:hash", h.GetDownloadDetails)
	}
}
