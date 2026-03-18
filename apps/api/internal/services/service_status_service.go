package services

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/vido/api/internal/health"
	"github.com/vido/api/internal/models"
)

// ErrServiceNotFound is returned when a service name is not recognized
var ErrServiceNotFound = errors.New("service not found")

// ServiceStatusServiceInterface defines the contract for service status operations
type ServiceStatusServiceInterface interface {
	// GetAllStatuses returns connection status for all external services
	GetAllStatuses(ctx context.Context) ([]models.ServiceStatus, error)

	// TestService manually tests connectivity for a specific service
	TestService(ctx context.Context, serviceName string) (*models.ServiceStatus, error)
}

// ServiceStatusService provides service connection status for the settings dashboard
type ServiceStatusService struct {
	monitor *health.HealthMonitor
	checker health.HealthChecker
	logger  *slog.Logger
}

// Compile-time interface verification
var _ ServiceStatusServiceInterface = (*ServiceStatusService)(nil)

// NewServiceStatusService creates a new ServiceStatusService
func NewServiceStatusService(monitor *health.HealthMonitor, checker health.HealthChecker) *ServiceStatusService {
	return &ServiceStatusService{
		monitor: monitor,
		checker: checker,
		logger:  slog.Default(),
	}
}

// GetAllStatuses returns connection status for all external services
func (s *ServiceStatusService) GetAllStatuses(ctx context.Context) ([]models.ServiceStatus, error) {
	return s.monitor.GetAllServiceStatuses(), nil
}

// TestService manually tests connectivity for a specific service and returns updated status
func (s *ServiceStatusService) TestService(ctx context.Context, serviceName string) (*models.ServiceStatus, error) {
	name := models.ServiceName(serviceName)

	checkFn, err := s.getCheckFunc(name)
	if err != nil {
		return nil, err
	}

	start := time.Now()
	checkErr := checkFn(ctx)
	elapsed := time.Since(start).Milliseconds()

	s.monitor.UpdateServiceHealthWithTime(name, checkErr, elapsed)

	svc := s.monitor.GetServiceHealth(name)
	if svc == nil {
		return nil, ErrServiceNotFound
	}

	status := svc.ToServiceStatus()
	return &status, nil
}

// getCheckFunc returns the appropriate health check function for a service
func (s *ServiceStatusService) getCheckFunc(name models.ServiceName) (func(context.Context) error, error) {
	switch name {
	case models.ServiceNameTMDb:
		return s.checker.CheckTMDb, nil
	case models.ServiceNameDouban:
		return s.checker.CheckDouban, nil
	case models.ServiceNameWikipedia:
		return s.checker.CheckWikipedia, nil
	case models.ServiceNameAI:
		return s.checker.CheckAI, nil
	case models.ServiceNameQBittorrent:
		return s.checker.CheckQBittorrent, nil
	default:
		return nil, fmt.Errorf("%w: %s", ErrServiceNotFound, name)
	}
}
