package services

import (
	"log/slog"

	"github.com/vido/api/internal/health"
	"github.com/vido/api/internal/metadata"
	"github.com/vido/api/internal/models"
)

// DegradationServiceInterface defines the contract for degradation handling.
type DegradationServiceInterface interface {
	// GetCurrentLevel returns the current system degradation level.
	GetCurrentLevel() models.DegradationLevel

	// GetServiceHealth returns the health status of a specific service.
	GetServiceHealth(name models.ServiceName) *models.ServiceHealth

	// GetDegradedResult merges partial results and returns degradation info.
	GetDegradedResult(results []*metadata.MetadataResult) *models.DegradedResult

	// GetHealthStatus returns the complete health status response.
	GetHealthStatus() *models.HealthStatusResponse
}

// DegradationService orchestrates degradation handling across the system.
type DegradationService struct {
	monitor        *health.HealthMonitor
	partialHandler *metadata.PartialResultHandler
	logger         *slog.Logger
}

// Compile-time interface verification.
var _ DegradationServiceInterface = (*DegradationService)(nil)

// NewDegradationService creates a new DegradationService.
func NewDegradationService(monitor *health.HealthMonitor) *DegradationService {
	return &DegradationService{
		monitor:        monitor,
		partialHandler: metadata.NewPartialResultHandler(),
		logger:         slog.Default(),
	}
}

// GetCurrentLevel returns the current system degradation level.
func (s *DegradationService) GetCurrentLevel() models.DegradationLevel {
	return s.monitor.GetDegradationLevel()
}

// GetServiceHealth returns the health status of a specific service.
func (s *DegradationService) GetServiceHealth(name models.ServiceName) *models.ServiceHealth {
	return s.monitor.GetServiceHealth(name)
}

// GetDegradedResult merges partial results and returns degradation info.
func (s *DegradationService) GetDegradedResult(results []*metadata.MetadataResult) *models.DegradedResult {
	return s.partialHandler.MergePartialResults(results)
}

// GetHealthStatus returns the complete health status response.
func (s *DegradationService) GetHealthStatus() *models.HealthStatusResponse {
	return s.monitor.GetHealthStatus()
}

// UpdateServiceHealth updates the health of a specific service based on an error.
func (s *DegradationService) UpdateServiceHealth(name models.ServiceName, err error) {
	s.monitor.UpdateServiceHealth(name, err)
}
