package handlers

import (
	"log/slog"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/vido/api/internal/services"
)

// ExportHandler handles HTTP requests for metadata export
type ExportHandler struct {
	exportService services.ExportServiceInterface
}

// NewExportHandler creates a new ExportHandler
func NewExportHandler(exportService services.ExportServiceInterface) *ExportHandler {
	return &ExportHandler{exportService: exportService}
}

// RegisterRoutes registers export routes
func (h *ExportHandler) RegisterRoutes(rg *gin.RouterGroup) {
	exports := rg.Group("/settings/export")
	{
		exports.POST("", h.TriggerExport)
		exports.GET("/status", h.GetExportStatus)
		exports.GET("/:id/download", h.DownloadExport)
	}
}

// TriggerExport handles POST /api/v1/settings/export
func (h *ExportHandler) TriggerExport(c *gin.Context) {
	var req struct {
		Format string `json:"format" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequestError(c, "EXPORT_FORMAT_INVALID", "Format is required: json, yaml, or nfo")
		return
	}

	var result *services.ExportResult
	var err error

	switch services.ExportFormat(req.Format) {
	case services.ExportFormatJSON:
		result, err = h.exportService.ExportJSON(c.Request.Context())
	case services.ExportFormatYAML:
		result, err = h.exportService.ExportYAML(c.Request.Context())
	case services.ExportFormatNFO:
		result, err = h.exportService.ExportNFO(c.Request.Context())
	default:
		BadRequestError(c, "EXPORT_FORMAT_INVALID", "Supported formats: json, yaml, nfo")
		return
	}

	if err != nil {
		slog.Error("Failed to start export", "error", err, "format", req.Format)
		ErrorResponse(c, 500, "EXPORT_FAILED", "Failed to start export", "Please try again later.")
		return
	}

	SuccessResponse(c, result)
}

// GetExportStatus handles GET /api/v1/settings/export/status
func (h *ExportHandler) GetExportStatus(c *gin.Context) {
	result, err := h.exportService.GetExportStatus(c.Request.Context())
	if err != nil {
		slog.Error("Failed to get export status", "error", err)
		InternalServerError(c, "Failed to get export status")
		return
	}

	SuccessResponse(c, result)
}

// DownloadExport handles GET /api/v1/settings/export/:id/download
func (h *ExportHandler) DownloadExport(c *gin.Context) {
	id := c.Param("id")

	filePath, err := h.exportService.GetExportFilePath(c.Request.Context(), id)
	if err != nil {
		BadRequestError(c, "EXPORT_NOT_FOUND", err.Error())
		return
	}

	filename := filepath.Base(filePath)
	c.Header("Content-Disposition", "attachment; filename="+filename)
	c.File(filePath)
}
