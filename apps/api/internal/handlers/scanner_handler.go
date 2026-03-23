package handlers

import (
	"context"
	"net/http"
	"strings"

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

// ScannerHandler handles HTTP requests for scanner operations.
type ScannerHandler struct {
	scannerService ScannerServiceInterface
}

// NewScannerHandler creates a new ScannerHandler with the given service.
func NewScannerHandler(scannerService ScannerServiceInterface) *ScannerHandler {
	return &ScannerHandler{
		scannerService: scannerService,
	}
}

// RegisterRoutes registers scanner routes on the given router group.
func (h *ScannerHandler) RegisterRoutes(rg *gin.RouterGroup) {
	scanner := rg.Group("/scanner")
	{
		scanner.POST("/scan", h.TriggerScan)
		scanner.GET("/status", h.GetStatus)
		scanner.POST("/cancel", h.CancelScan)
	}
}

// TriggerScan handles POST /api/v1/scanner/scan
// Starts a new scan in the background. Returns 409 if a scan is already running.
func (h *ScannerHandler) TriggerScan(c *gin.Context) {
	if h.scannerService.IsScanActive() {
		ErrorResponse(c, http.StatusConflict, "SCANNER_ALREADY_RUNNING",
			"A scan is already in progress",
			"Wait for the current scan to complete or cancel it first.")
		return
	}

	// Start scan in a goroutine (non-blocking)
	go func() {
		_, _ = h.scannerService.StartScan(c.Request.Context())
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

// CancelScan handles POST /api/v1/scanner/cancel
// Cancels the currently active scan.
func (h *ScannerHandler) CancelScan(c *gin.Context) {
	err := h.scannerService.CancelScan()
	if err != nil {
		if strings.Contains(err.Error(), "no scan is currently active") {
			BadRequestError(c, "SCANNER_NOT_ACTIVE", "No scan is currently active")
			return
		}
		InternalServerError(c, "Failed to cancel scan")
		return
	}

	SuccessResponse(c, gin.H{"message": "Scan cancelled"})
}
