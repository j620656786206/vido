package handlers

import (
	"errors"
	"log/slog"

	"github.com/gin-gonic/gin"
	"github.com/vido/api/internal/qbittorrent"
	"github.com/vido/api/internal/services"
)

// DownloadHandler handles HTTP requests for download monitoring.
type DownloadHandler struct {
	service services.DownloadServiceInterface
}

// NewDownloadHandler creates a new DownloadHandler.
func NewDownloadHandler(service services.DownloadServiceInterface) *DownloadHandler {
	return &DownloadHandler{service: service}
}

// ListDownloads handles GET /api/v1/downloads
// @Summary List all downloads
// @Description Retrieves all torrents from qBittorrent with optional filtering and sorting
// @Tags downloads
// @Accept json
// @Produce json
// @Param filter query string false "Filter by status (all, downloading, paused, completed, seeding, error)" default(all)
// @Param sort query string false "Sort field (added_on, name, progress, size)" default(added_on)
// @Param order query string false "Sort order (asc, desc)" default(desc)
// @Success 200 {object} APIResponse
// @Failure 400 {object} APIResponse
// @Failure 500 {object} APIResponse
// @Router /api/v1/downloads [get]
func (h *DownloadHandler) ListDownloads(c *gin.Context) {
	filter := c.DefaultQuery("filter", "all")
	sort := c.DefaultQuery("sort", "added_on")
	order := c.DefaultQuery("order", "desc")

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

	SuccessResponse(c, torrents)
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
