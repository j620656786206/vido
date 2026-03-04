package health

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/repository"
)

// HealthChecker defines the interface for checking external service health
type HealthChecker interface {
	CheckTMDb(ctx context.Context) error
	CheckDouban(ctx context.Context) error
	CheckWikipedia(ctx context.Context) error
	CheckAI(ctx context.Context) error
	CheckQBittorrent(ctx context.Context) error
}

// HealthMonitor tracks the health of external services
type HealthMonitor struct {
	mu          sync.RWMutex
	services    *models.ServicesHealth
	checker     HealthChecker
	historyRepo repository.ConnectionHistoryRepositoryInterface
	logger      *slog.Logger
}

// NewHealthMonitor creates a new HealthMonitor
func NewHealthMonitor(checker HealthChecker) *HealthMonitor {
	return &HealthMonitor{
		services: models.NewServicesHealth(),
		checker:  checker,
		logger:   slog.Default(),
	}
}

// SetHistoryRepo sets the connection history repository for event persistence.
func (m *HealthMonitor) SetHistoryRepo(repo repository.ConnectionHistoryRepositoryInterface) {
	m.historyRepo = repo
}

// GetDegradationLevel returns the current degradation level based on service health
func (m *HealthMonitor) GetDegradationLevel() models.DegradationLevel {
	m.mu.RLock()
	defer m.mu.RUnlock()

	downCount := 0
	degradedCount := 0
	totalServices := 5 // TMDb, Douban, Wikipedia, AI, qBittorrent

	for _, svc := range m.services.AllServices() {
		if svc.IsDown() {
			downCount++
		} else if svc.IsDegraded() {
			degradedCount++
		}
	}

	// All services down
	if downCount == totalServices {
		return models.DegradationOffline
	}

	// More than half down
	if downCount > totalServices/2 {
		return models.DegradationMinimal
	}

	// Any service degraded or down
	if downCount > 0 || degradedCount > 0 {
		return models.DegradationPartial
	}

	return models.DegradationNormal
}

// GetServiceHealth returns the health status of a specific service
func (m *HealthMonitor) GetServiceHealth(name models.ServiceName) *models.ServiceHealth {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.services.GetService(name)
}

// GetAllServices returns all service health statuses
func (m *HealthMonitor) GetAllServices() *models.ServicesHealth {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.services
}

// CheckAllServices checks all external services and updates their health status
func (m *HealthMonitor) CheckAllServices(ctx context.Context) {
	var wg sync.WaitGroup

	checks := []struct {
		name    models.ServiceName
		checker func(context.Context) error
	}{
		{models.ServiceNameTMDb, m.checker.CheckTMDb},
		{models.ServiceNameDouban, m.checker.CheckDouban},
		{models.ServiceNameWikipedia, m.checker.CheckWikipedia},
		{models.ServiceNameAI, m.checker.CheckAI},
		{models.ServiceNameQBittorrent, m.checker.CheckQBittorrent},
	}

	for _, check := range checks {
		wg.Add(1)
		go func(name models.ServiceName, fn func(context.Context) error) {
			defer wg.Done()
			err := fn(ctx)
			m.UpdateServiceHealth(name, err)
		}(check.name, check.checker)
	}

	wg.Wait()
}

// UpdateServiceHealth updates the health status of a specific service
func (m *HealthMonitor) UpdateServiceHealth(name models.ServiceName, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	svc := m.services.GetService(name)
	if svc == nil {
		return
	}

	previousStatus := svc.Status

	if err == nil {
		svc.RecordSuccess()
		if previousStatus != models.ServiceStatusHealthy {
			m.logger.Info("Service recovered",
				"service", name,
				"previous_status", previousStatus,
			)
		}
	} else {
		svc.RecordError(err.Error())
		if previousStatus != svc.Status {
			m.logger.Warn("Service health changed",
				"service", name,
				"previous_status", previousStatus,
				"new_status", svc.Status,
				"error", err.Error(),
			)
		}
	}

	// Record status change events to connection history
	if previousStatus != svc.Status && previousStatus != "" && m.historyRepo != nil {
		var eventType models.ConnectionEventType
		switch {
		case svc.Status == models.ServiceStatusHealthy:
			eventType = models.EventRecovered
		case svc.Status == models.ServiceStatusDown:
			eventType = models.EventDisconnected
		default:
			eventType = models.EventError
		}

		event := &models.ConnectionEvent{
			ID:        uuid.New().String(),
			Service:   string(name),
			EventType: eventType,
			Status:    svc.Status,
			Message:   svc.Message,
			CreatedAt: time.Now(),
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if repoErr := m.historyRepo.Create(ctx, event); repoErr != nil {
			m.logger.Error("Failed to record connection event",
				"service", name,
				"event_type", eventType,
				"error", repoErr,
			)
		}
	}
}

// GetHealthStatus returns the complete health status response
func (m *HealthMonitor) GetHealthStatus() *models.HealthStatusResponse {
	m.mu.RLock()
	defer m.mu.RUnlock()

	level := m.getDegradationLevelUnlocked()
	message := m.generateStatusMessage(level)

	return &models.HealthStatusResponse{
		DegradationLevel: level,
		Services:         m.services,
		Message:          message,
	}
}

// getDegradationLevelUnlocked is the internal version without locking
func (m *HealthMonitor) getDegradationLevelUnlocked() models.DegradationLevel {
	downCount := 0
	degradedCount := 0
	totalServices := 5 // TMDb, Douban, Wikipedia, AI, qBittorrent

	for _, svc := range m.services.AllServices() {
		if svc.IsDown() {
			downCount++
		} else if svc.IsDegraded() {
			degradedCount++
		}
	}

	if downCount == totalServices {
		return models.DegradationOffline
	}

	if downCount > totalServices/2 {
		return models.DegradationMinimal
	}

	if downCount > 0 || degradedCount > 0 {
		return models.DegradationPartial
	}

	return models.DegradationNormal
}

// generateStatusMessage generates a user-friendly status message
func (m *HealthMonitor) generateStatusMessage(level models.DegradationLevel) string {
	if level == models.DegradationNormal {
		return ""
	}

	affectedServices := make([]string, 0)
	for _, svc := range m.services.AllServices() {
		if !svc.IsHealthy() {
			affectedServices = append(affectedServices, svc.DisplayName)
		}
	}

	if len(affectedServices) == 0 {
		return ""
	}

	switch level {
	case models.DegradationOffline:
		return "所有外部服務無法使用，僅能存取本地快取"
	case models.DegradationMinimal:
		return fmt.Sprintf("多項服務無法使用：%s", strings.Join(affectedServices, "、"))
	case models.DegradationPartial:
		return fmt.Sprintf("部分服務降級中：%s", strings.Join(affectedServices, "、"))
	default:
		return ""
	}
}

// StartMonitoring starts background health monitoring
func (m *HealthMonitor) StartMonitoring(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Perform initial check
	m.CheckAllServices(ctx)

	for {
		select {
		case <-ctx.Done():
			m.logger.Info("Health monitoring stopped")
			return
		case <-ticker.C:
			m.CheckAllServices(ctx)
		}
	}
}

// StartQBMonitoring starts a dedicated monitor for qBittorrent with 30s interval (NFR-R6)
func (m *HealthMonitor) StartQBMonitoring(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	// Perform initial check
	err := m.checker.CheckQBittorrent(ctx)
	m.UpdateServiceHealth(models.ServiceNameQBittorrent, err)

	for {
		select {
		case <-ctx.Done():
			m.logger.Info("qBittorrent health monitoring stopped")
			return
		case <-ticker.C:
			err := m.checker.CheckQBittorrent(ctx)
			m.UpdateServiceHealth(models.ServiceNameQBittorrent, err)
		}
	}
}
