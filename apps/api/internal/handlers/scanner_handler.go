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
