package handlers

import (
	"errors"
	"log/slog"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/vido/api/internal/services"
)

// BackupHandler handles HTTP requests for backup management
type BackupHandler struct {
	backupService services.BackupServiceInterface
}

// NewBackupHandler creates a new BackupHandler
func NewBackupHandler(backupService services.BackupServiceInterface) *BackupHandler {
	return &BackupHandler{backupService: backupService}
}

// RegisterRoutes registers backup management routes
func (h *BackupHandler) RegisterRoutes(rg *gin.RouterGroup) {
	backups := rg.Group("/settings/backups")
	{
		backups.POST("", h.CreateBackup)
		backups.GET("", h.ListBackups)
		backups.GET("/:id", h.GetBackup)
		backups.DELETE("/:id", h.DeleteBackup)
		backups.GET("/:id/download", h.DownloadBackup)
	}
}

// CreateBackup handles POST /api/v1/settings/backups
func (h *BackupHandler) CreateBackup(c *gin.Context) {
	backup, err := h.backupService.CreateBackup(c.Request.Context())
	if err != nil {
		if errors.Is(err, services.ErrBackupInProgress) {
			ErrorResponse(c, 409, "BACKUP_IN_PROGRESS", "Another backup is already running", "Please wait for the current backup to complete.")
			return
		}
		slog.Error("Failed to create backup", "error", err)
		ErrorResponse(c, 500, "BACKUP_CREATE_FAILED", "Failed to create backup", "Please try again later.")
		return
	}

	SuccessResponse(c, backup)
}

// ListBackups handles GET /api/v1/settings/backups
func (h *BackupHandler) ListBackups(c *gin.Context) {
	result, err := h.backupService.ListBackups(c.Request.Context())
	if err != nil {
		slog.Error("Failed to list backups", "error", err)
		InternalServerError(c, "Failed to retrieve backups")
		return
	}

	SuccessResponse(c, result)
}

// GetBackup handles GET /api/v1/settings/backups/:id
func (h *BackupHandler) GetBackup(c *gin.Context) {
	id := c.Param("id")

	backup, err := h.backupService.GetBackup(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, services.ErrBackupNotFound) {
			BadRequestError(c, "BACKUP_NOT_FOUND", "Backup not found: "+id)
			return
		}
		slog.Error("Failed to get backup", "error", err, "id", id)
		InternalServerError(c, "Failed to retrieve backup")
		return
	}

	SuccessResponse(c, backup)
}

// DeleteBackup handles DELETE /api/v1/settings/backups/:id
func (h *BackupHandler) DeleteBackup(c *gin.Context) {
	id := c.Param("id")

	if err := h.backupService.DeleteBackup(c.Request.Context(), id); err != nil {
		if errors.Is(err, services.ErrBackupNotFound) {
			BadRequestError(c, "BACKUP_NOT_FOUND", "Backup not found: "+id)
			return
		}
		slog.Error("Failed to delete backup", "error", err, "id", id)
		InternalServerError(c, "Failed to delete backup")
		return
	}

	SuccessResponse(c, gin.H{"deleted": true})
}

// DownloadBackup handles GET /api/v1/settings/backups/:id/download
func (h *BackupHandler) DownloadBackup(c *gin.Context) {
	id := c.Param("id")

	filePath, err := h.backupService.GetBackupFilePath(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, services.ErrBackupNotFound) {
			BadRequestError(c, "BACKUP_NOT_FOUND", "Backup not found: "+id)
			return
		}
		slog.Error("Failed to get backup file path", "error", err, "id", id)
		InternalServerError(c, "Failed to download backup")
		return
	}

	filename := filepath.Base(filePath)
	c.Header("Content-Disposition", "attachment; filename="+filename)
	c.Header("Content-Type", "application/gzip")
	c.File(filePath)
}
