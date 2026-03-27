package handlers

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/vido/api/internal/services"
)

// ScannerServiceInterface defines the contract for scanner operations.
// This interface enables testing handlers with mock services.
type ScannerServiceInterface interface {
	IsScanActive() bool
	StartScan(ctx context.Context) (*services.ScanResult, error)
	CancelScan() error
	GetProgress() services.ScanProgress
}

// EnrichmentServiceInterface defines the contract for enrichment operations.
type EnrichmentServiceInterface interface {
	StartEnrichment(ctx context.Context) (*services.EnrichmentResult, error)
	CancelEnrichment() error
	IsEnrichmentActive() bool
	GetProgress() services.EnrichmentProgress
}

// ScannerHandler handles HTTP requests for scanner operations.
type ScannerHandler struct {
	scannerService    ScannerServiceInterface
	scanScheduler     services.ScanSchedulerInterface
	enrichmentService EnrichmentServiceInterface
}

// NewScannerHandler creates a new ScannerHandler with the given service.
func NewScannerHandler(scannerService ScannerServiceInterface) *ScannerHandler {
	return &ScannerHandler{
		scannerService: scannerService,
	}
}

// SetScheduler sets the scan scheduler for schedule management endpoints.
func (h *ScannerHandler) SetScheduler(scheduler services.ScanSchedulerInterface) {
	h.scanScheduler = scheduler
}

// SetEnrichmentService sets the enrichment service for metadata enrichment endpoints.
func (h *ScannerHandler) SetEnrichmentService(svc EnrichmentServiceInterface) {
	h.enrichmentService = svc
}

// RegisterRoutes registers scanner routes on the given router group.
func (h *ScannerHandler) RegisterRoutes(rg *gin.RouterGroup) {
	scanner := rg.Group("/scanner")
	{
		scanner.POST("/scan", h.TriggerScan)
		scanner.GET("/status", h.GetStatus)
		scanner.POST("/cancel", h.CancelScan)
		scanner.GET("/schedule", h.GetSchedule)
		scanner.PUT("/schedule", h.SetSchedule)
		scanner.POST("/enrich", h.TriggerEnrich)
		scanner.GET("/enrich/status", h.GetEnrichStatus)
		scanner.POST("/enrich/cancel", h.CancelEnrich)
	}
}

// TriggerScan handles POST /api/v1/scanner/scan
// Starts a new scan in the background. Returns 409 if a scan is already running.
// Uses context.Background() for the goroutine since the scan outlives the HTTP request.
// The StartScan mutex is the single gate for concurrency — no pre-check race condition.
func (h *ScannerHandler) TriggerScan(c *gin.Context) {
	// Start scan in a goroutine with background context (not request context,
	// which would be cancelled when the HTTP response is sent).
	// StartScan's internal mutex handles concurrent request protection — if two
	// requests arrive simultaneously, the second will get SCANNER_ALREADY_RUNNING.
	go func() {
		result, err := h.scannerService.StartScan(context.Background())
		if err != nil {
			slog.Error("scan failed", "error", err)
			return
		}
		if result != nil {
			slog.Info("scan completed in background",
				"files_found", result.FilesFound,
				"files_created", result.FilesCreated,
				"duration", result.Duration,
			)
		}
	}()

	c.JSON(http.StatusAccepted, APIResponse{
		Success: true,
		Data:    gin.H{"message": "Scan started"},
	})
}

// GetStatus handles GET /api/v1/scanner/status
// Returns the current scan progress.
func (h *ScannerHandler) GetStatus(c *gin.Context) {
	progress := h.scannerService.GetProgress()
	SuccessResponse(c, progress)
}

// scheduleRequest represents the request body for setting scan schedule
type scheduleRequest struct {
	Interval string `json:"interval" binding:"required"`
}

// GetSchedule handles GET /api/v1/scanner/schedule
// Returns the current scan schedule configuration.
func (h *ScannerHandler) GetSchedule(c *gin.Context) {
	if h.scanScheduler == nil {
		InternalServerError(c, "Scan scheduler not configured")
		return
	}

	interval := h.scanScheduler.GetInterval()
	SuccessResponse(c, gin.H{"interval": string(interval)})
}

// SetSchedule handles PUT /api/v1/scanner/schedule
// Updates the scan schedule interval and reconfigures the scheduler.
func (h *ScannerHandler) SetSchedule(c *gin.Context) {
	if h.scanScheduler == nil {
		InternalServerError(c, "Scan scheduler not configured")
		return
	}

	var req scheduleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequestError(c, "SCANNER_SCHEDULE_INVALID", "Request body must contain an 'interval' field")
		return
	}

	interval := services.ScanScheduleInterval(req.Interval)
	if !services.ValidScanScheduleIntervals[interval] {
		BadRequestError(c, "SCANNER_SCHEDULE_INVALID",
			"Invalid schedule interval. Must be one of: manual, hourly, daily")
		return
	}

	if err := h.scanScheduler.Reconfigure(interval); err != nil {
		InternalServerError(c, "Failed to update scan schedule")
		return
	}

	SuccessResponse(c, gin.H{"interval": req.Interval})
}

// CancelScan handles POST /api/v1/scanner/cancel
// Cancels the currently active scan.
func (h *ScannerHandler) CancelScan(c *gin.Context) {
	err := h.scannerService.CancelScan()
	if err != nil {
		if errors.Is(err, services.ErrScanNotActive) {
			BadRequestError(c, "SCANNER_NOT_ACTIVE", "No scan is currently active")
			return
		}
		InternalServerError(c, "Failed to cancel scan")
		return
	}

	SuccessResponse(c, gin.H{"message": "Scan cancelled"})
}

// TriggerEnrich handles POST /api/v1/scanner/enrich
// Starts metadata enrichment for unenriched movies in the background.
func (h *ScannerHandler) TriggerEnrich(c *gin.Context) {
	if h.enrichmentService == nil {
		InternalServerError(c, "Enrichment service not configured")
		return
	}

	if h.enrichmentService.IsEnrichmentActive() {
		c.JSON(http.StatusConflict, APIResponse{
			Success: false,
			Error:   &APIError{Code: "ENRICHMENT_ALREADY_RUNNING", Message: "An enrichment is already in progress"},
		})
		return
	}

	go func() {
		result, err := h.enrichmentService.StartEnrichment(context.Background())
		if err != nil {
			slog.Error("enrichment failed", "error", err)
			return
		}
		if result != nil {
			slog.Info("enrichment completed in background",
				"succeeded", result.Succeeded,
				"failed", result.Failed,
				"duration", result.Duration,
			)
		}
	}()

	c.JSON(http.StatusAccepted, APIResponse{
		Success: true,
		Data:    gin.H{"message": "Enrichment started"},
	})
}

// GetEnrichStatus handles GET /api/v1/scanner/enrich/status
func (h *ScannerHandler) GetEnrichStatus(c *gin.Context) {
	if h.enrichmentService == nil {
		InternalServerError(c, "Enrichment service not configured")
		return
	}
	SuccessResponse(c, h.enrichmentService.GetProgress())
}

// CancelEnrich handles POST /api/v1/scanner/enrich/cancel
func (h *ScannerHandler) CancelEnrich(c *gin.Context) {
	if h.enrichmentService == nil {
		InternalServerError(c, "Enrichment service not configured")
		return
	}
	if err := h.enrichmentService.CancelEnrichment(); err != nil {
		BadRequestError(c, "ENRICHMENT_NOT_ACTIVE", "No enrichment is currently active")
		return
	}
	SuccessResponse(c, gin.H{"message": "Enrichment cancelled"})
}
