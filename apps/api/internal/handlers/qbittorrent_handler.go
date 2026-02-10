package handlers

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/vido/api/internal/qbittorrent"
)

// QBittorrentServiceInterface defines the contract for qBittorrent operations
// used by the handler layer.
type QBittorrentServiceInterface interface {
	GetConfig(ctx context.Context) (*qbittorrent.Config, error)
	SaveConfig(ctx context.Context, config *qbittorrent.Config) error
	TestConnection(ctx context.Context) (*qbittorrent.VersionInfo, error)
	IsConfigured(ctx context.Context) bool
}

// QBittorrentHandler handles HTTP requests for qBittorrent settings.
type QBittorrentHandler struct {
	service QBittorrentServiceInterface
}

// NewQBittorrentHandler creates a new QBittorrentHandler.
func NewQBittorrentHandler(service QBittorrentServiceInterface) *QBittorrentHandler {
	return &QBittorrentHandler{service: service}
}

// QBConfigResponse is the API response for qBittorrent configuration.
// Password is never included in the response.
type QBConfigResponse struct {
	Host       string `json:"host"`
	Username   string `json:"username"`
	BasePath   string `json:"basePath"`
	Configured bool   `json:"configured"`
}

// SaveQBConfigRequest is the request body for saving qBittorrent configuration.
type SaveQBConfigRequest struct {
	Host     string `json:"host" binding:"required"`
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
	BasePath string `json:"basePath"`
}

// GetConfig handles GET /api/v1/settings/qbittorrent
func (h *QBittorrentHandler) GetConfig(c *gin.Context) {
	config, err := h.service.GetConfig(c.Request.Context())
	if err != nil {
		slog.Error("Failed to get qBittorrent config", "error", err)
		InternalServerError(c, "Failed to retrieve qBittorrent configuration")
		return
	}

	SuccessResponse(c, QBConfigResponse{
		Host:       config.Host,
		Username:   config.Username,
		BasePath:   config.BasePath,
		Configured: h.service.IsConfigured(c.Request.Context()),
	})
}

// SaveConfig handles PUT /api/v1/settings/qbittorrent
func (h *QBittorrentHandler) SaveConfig(c *gin.Context) {
	var req SaveQBConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ValidationError(c, "Invalid request body: "+err.Error())
		return
	}

	config := &qbittorrent.Config{
		Host:     req.Host,
		Username: req.Username,
		Password: req.Password,
		BasePath: req.BasePath,
	}

	if err := h.service.SaveConfig(c.Request.Context(), config); err != nil {
		slog.Error("Failed to save qBittorrent config", "error", err)
		InternalServerError(c, "Failed to save qBittorrent configuration")
		return
	}

	SuccessResponse(c, gin.H{"message": "Configuration saved"})
}

// TestConnection handles POST /api/v1/settings/qbittorrent/test
func (h *QBittorrentHandler) TestConnection(c *gin.Context) {
	info, err := h.service.TestConnection(c.Request.Context())
	if err != nil {
		slog.Error("qBittorrent connection test failed", "error", err)
		c.JSON(http.StatusBadRequest, APIResponse{
			Success: false,
			Error: &APIError{
				Code:       "QB_CONNECTION_FAILED",
				Message:    "無法連線到 qBittorrent",
				Suggestion: err.Error(),
			},
		})
		return
	}

	SuccessResponse(c, info)
}

// RegisterRoutes registers qBittorrent settings routes.
func (h *QBittorrentHandler) RegisterRoutes(rg *gin.RouterGroup) {
	qb := rg.Group("/settings/qbittorrent")
	{
		qb.GET("", h.GetConfig)
		qb.PUT("", h.SaveConfig)
		qb.POST("/test", h.TestConnection)
	}
}
