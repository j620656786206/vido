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
	scheduler     services.BackupSchedulerInterface
}

// NewBackupHandler creates a new BackupHandler
func NewBackupHandler(backupService services.BackupServiceInterface) *BackupHandler {
	return &BackupHandler{backupService: backupService}
}

// SetScheduler sets the backup scheduler for schedule endpoints
func (h *BackupHandler) SetScheduler(scheduler services.BackupSchedulerInterface) {
	h.scheduler = scheduler
}

// RegisterRoutes registers backup management routes
func (h *BackupHandler) RegisterRoutes(rg *gin.RouterGroup) {
	backups := rg.Group("/settings/backups")
	{
		backups.POST("", h.CreateBackup)
		backups.GET("", h.ListBackups)
		// Schedule routes MUST be registered before /:id to avoid Gin radix tree conflict
		backups.GET("/schedule", h.GetSchedule)
		backups.PUT("/schedule", h.UpdateSchedule)
		backups.GET("/:id", h.GetBackup)
		backups.DELETE("/:id", h.DeleteBackup)
		backups.GET("/:id/download", h.DownloadBackup)
		backups.POST("/:id/verify", h.VerifyBackup)
		backups.POST("/:id/restore", h.RestoreBackup)
	}
	rg.GET("/settings/restore/status", h.GetRestoreStatus)
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

// RestoreBackup handles POST /api/v1/settings/backups/:id/restore
func (h *BackupHandler) RestoreBackup(c *gin.Context) {
	id := c.Param("id")

	result, err := h.backupService.RestoreBackup(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, services.ErrBackupNotFound) {
			BadRequestError(c, "BACKUP_NOT_FOUND", "Backup not found: "+id)
			return
		}
		if errors.Is(err, services.ErrRestoreInProgress) {
			ErrorResponse(c, 409, "RESTORE_IN_PROGRESS", "Another restore operation is already running", "Please wait for the current restore to complete.")
			return
		}
		slog.Error("Failed to restore backup", "error", err, "id", id)
		ErrorResponse(c, 500, "RESTORE_FAILED", "Failed to restore backup", "Please try again later.")
		return
	}

	SuccessResponse(c, result)
}

// GetRestoreStatus handles GET /api/v1/settings/restore/status
func (h *BackupHandler) GetRestoreStatus(c *gin.Context) {
	result, err := h.backupService.GetRestoreStatus(c.Request.Context())
	if err != nil {
		slog.Error("Failed to get restore status", "error", err)
		InternalServerError(c, "Failed to get restore status")
		return
	}

	SuccessResponse(c, result)
}

// GetSchedule handles GET /api/v1/settings/backups/schedule
func (h *BackupHandler) GetSchedule(c *gin.Context) {
	if h.scheduler == nil {
		InternalServerError(c, "Scheduler not configured")
		return
	}

	resp, err := h.scheduler.GetSchedule(c.Request.Context())
	if err != nil {
		slog.Error("Failed to get backup schedule", "error", err)
		InternalServerError(c, "Failed to get backup schedule")
		return
	}

	SuccessResponse(c, resp)
}

// UpdateSchedule handles PUT /api/v1/settings/backups/schedule
func (h *BackupHandler) UpdateSchedule(c *gin.Context) {
	if h.scheduler == nil {
		InternalServerError(c, "Scheduler not configured")
		return
	}

	var schedule services.BackupSchedule
	if err := c.ShouldBindJSON(&schedule); err != nil {
		BadRequestError(c, "SCHEDULE_INVALID", "Invalid schedule configuration: "+err.Error())
		return
	}

	if err := h.scheduler.SetSchedule(c.Request.Context(), schedule); err != nil {
		slog.Error("Failed to update backup schedule", "error", err)
		BadRequestError(c, "SCHEDULE_INVALID", err.Error())
		return
	}

	resp, err := h.scheduler.GetSchedule(c.Request.Context())
	if err != nil {
		slog.Error("Failed to get updated schedule", "error", err)
		InternalServerError(c, "Schedule updated but failed to retrieve")
		return
	}

	SuccessResponse(c, resp)
}

// VerifyBackup handles POST /api/v1/settings/backups/:id/verify
func (h *BackupHandler) VerifyBackup(c *gin.Context) {
	id := c.Param("id")

	result, err := h.backupService.VerifyBackup(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, services.ErrBackupNotFound) {
			BadRequestError(c, "BACKUP_NOT_FOUND", "Backup not found: "+id)
			return
		}
		slog.Error("Failed to verify backup", "error", err, "id", id)
		ErrorResponse(c, 500, "BACKUP_VERIFY_FAILED", "Failed to verify backup", "Please try again later.")
		return
	}

	SuccessResponse(c, result)
}
