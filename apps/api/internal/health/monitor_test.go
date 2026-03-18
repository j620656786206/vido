package health

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vido/api/internal/models"
)

// MockHealthChecker is a mock implementation of HealthChecker for testing
type MockHealthChecker struct {
	mu             sync.RWMutex
	tmdbErr        error
	doubanErr      error
	wikipediaErr   error
	aiErr          error
	qbittorrentErr error
}

func (m *MockHealthChecker) CheckTMDb(ctx context.Context) error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.tmdbErr
}

func (m *MockHealthChecker) CheckDouban(ctx context.Context) error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.doubanErr
}

func (m *MockHealthChecker) CheckWikipedia(ctx context.Context) error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.wikipediaErr
}

func (m *MockHealthChecker) CheckAI(ctx context.Context) error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.aiErr
}

func (m *MockHealthChecker) CheckQBittorrent(ctx context.Context) error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.qbittorrentErr
}

func (m *MockHealthChecker) SetTMDbError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.tmdbErr = err
}

func (m *MockHealthChecker) SetDoubanError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.doubanErr = err
}

func (m *MockHealthChecker) SetWikipediaError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.wikipediaErr = err
}

func (m *MockHealthChecker) SetAIError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.aiErr = err
}

func (m *MockHealthChecker) SetQBittorrentError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.qbittorrentErr = err
}

func TestNewHealthMonitor(t *testing.T) {
	checker := &MockHealthChecker{}
	monitor := NewHealthMonitor(checker)

	assert.NotNil(t, monitor)
	assert.NotNil(t, monitor.services)
	assert.Equal(t, models.DegradationNormal, monitor.GetDegradationLevel())
}

func TestHealthMonitor_GetDegradationLevel_AllHealthy(t *testing.T) {
	checker := &MockHealthChecker{}
	monitor := NewHealthMonitor(checker)

	level := monitor.GetDegradationLevel()
	assert.Equal(t, models.DegradationNormal, level)
}

func TestHealthMonitor_GetDegradationLevel_OneDegraded(t *testing.T) {
	checker := &MockHealthChecker{}
	monitor := NewHealthMonitor(checker)

	// Simulate one service degraded (1 error)
	monitor.services.TMDb.RecordError("timeout")

	level := monitor.GetDegradationLevel()
	assert.Equal(t, models.DegradationPartial, level)
}

func TestHealthMonitor_GetDegradationLevel_OneDown(t *testing.T) {
	checker := &MockHealthChecker{}
	monitor := NewHealthMonitor(checker)

	// Simulate one service down (3 errors)
	monitor.services.TMDb.RecordError("error 1")
	monitor.services.TMDb.RecordError("error 2")
	monitor.services.TMDb.RecordError("error 3")

	level := monitor.GetDegradationLevel()
	assert.Equal(t, models.DegradationPartial, level)
}

func TestHealthMonitor_GetDegradationLevel_MultipleDown(t *testing.T) {
	checker := &MockHealthChecker{}
	monitor := NewHealthMonitor(checker)

	// Simulate 3 services down (more than half of 5)
	for i := 0; i < 3; i++ {
		monitor.services.TMDb.RecordError("error")
		monitor.services.Douban.RecordError("error")
		monitor.services.Wikipedia.RecordError("error")
	}

	level := monitor.GetDegradationLevel()
	assert.Equal(t, models.DegradationMinimal, level)
}

func TestHealthMonitor_GetDegradationLevel_AllDown(t *testing.T) {
	checker := &MockHealthChecker{}
	monitor := NewHealthMonitor(checker)

	// Simulate all services down
	for i := 0; i < 3; i++ {
		monitor.services.TMDb.RecordError("error")
		monitor.services.Douban.RecordError("error")
		monitor.services.Wikipedia.RecordError("error")
		monitor.services.AI.RecordError("error")
		monitor.services.QBittorrent.RecordError("error")
	}

	level := monitor.GetDegradationLevel()
	assert.Equal(t, models.DegradationOffline, level)
}

func TestHealthMonitor_GetServiceHealth(t *testing.T) {
	checker := &MockHealthChecker{}
	monitor := NewHealthMonitor(checker)

	health := monitor.GetServiceHealth(models.ServiceNameTMDb)
	require.NotNil(t, health)
	assert.Equal(t, "tmdb", health.Name)
	assert.Equal(t, models.ServiceStatusHealthy, health.Status)
}

func TestHealthMonitor_GetServiceHealth_NotFound(t *testing.T) {
	checker := &MockHealthChecker{}
	monitor := NewHealthMonitor(checker)

	health := monitor.GetServiceHealth(models.ServiceName("unknown"))
	assert.Nil(t, health)
}

func TestHealthMonitor_GetAllServices(t *testing.T) {
	checker := &MockHealthChecker{}
	monitor := NewHealthMonitor(checker)

	services := monitor.GetAllServices()
	assert.NotNil(t, services)
	assert.NotNil(t, services.TMDb)
	assert.NotNil(t, services.Douban)
	assert.NotNil(t, services.Wikipedia)
	assert.NotNil(t, services.AI)
	assert.NotNil(t, services.QBittorrent)
}

func TestHealthMonitor_CheckAllServices(t *testing.T) {
	checker := &MockHealthChecker{}
	monitor := NewHealthMonitor(checker)

	ctx := context.Background()
	monitor.CheckAllServices(ctx)

	// All should be healthy
	assert.Equal(t, models.ServiceStatusHealthy, monitor.services.TMDb.Status)
	assert.Equal(t, models.ServiceStatusHealthy, monitor.services.Douban.Status)
	assert.Equal(t, models.ServiceStatusHealthy, monitor.services.Wikipedia.Status)
	assert.Equal(t, models.ServiceStatusHealthy, monitor.services.AI.Status)
	assert.Equal(t, models.ServiceStatusHealthy, monitor.services.QBittorrent.Status)
}

func TestHealthMonitor_CheckAllServices_WithErrors(t *testing.T) {
	checker := &MockHealthChecker{}
	checker.SetTMDbError(errors.New("connection refused"))
	checker.SetAIError(errors.New("quota exceeded"))

	monitor := NewHealthMonitor(checker)

	ctx := context.Background()
	monitor.CheckAllServices(ctx)

	// Wait a bit for goroutines to complete
	time.Sleep(100 * time.Millisecond)

	assert.Equal(t, models.ServiceStatusDegraded, monitor.services.TMDb.Status)
	assert.Equal(t, models.ServiceStatusHealthy, monitor.services.Douban.Status)
	assert.Equal(t, models.ServiceStatusHealthy, monitor.services.Wikipedia.Status)
	assert.Equal(t, models.ServiceStatusDegraded, monitor.services.AI.Status)
}

func TestHealthMonitor_UpdateServiceHealth_Success(t *testing.T) {
	checker := &MockHealthChecker{}
	monitor := NewHealthMonitor(checker)

	// First mark as degraded
	monitor.services.TMDb.RecordError("timeout")
	assert.Equal(t, models.ServiceStatusDegraded, monitor.services.TMDb.Status)

	// Then record success
	monitor.UpdateServiceHealth(models.ServiceNameTMDb, nil)
	assert.Equal(t, models.ServiceStatusHealthy, monitor.services.TMDb.Status)
}

func TestHealthMonitor_UpdateServiceHealth_Error(t *testing.T) {
	checker := &MockHealthChecker{}
	monitor := NewHealthMonitor(checker)

	monitor.UpdateServiceHealth(models.ServiceNameTMDb, errors.New("connection refused"))
	assert.Equal(t, models.ServiceStatusDegraded, monitor.services.TMDb.Status)
	assert.Equal(t, "connection refused", monitor.services.TMDb.Message)
}

func TestHealthMonitor_GetHealthStatus(t *testing.T) {
	checker := &MockHealthChecker{}
	monitor := NewHealthMonitor(checker)

	status := monitor.GetHealthStatus()
	assert.Equal(t, models.DegradationNormal, status.DegradationLevel)
	assert.NotNil(t, status.Services)
}

func TestHealthMonitor_GetHealthStatus_Degraded(t *testing.T) {
	checker := &MockHealthChecker{}
	monitor := NewHealthMonitor(checker)

	// Simulate degradation
	monitor.services.AI.RecordError("quota exceeded")

	status := monitor.GetHealthStatus()
	assert.Equal(t, models.DegradationPartial, status.DegradationLevel)
	assert.Contains(t, status.Message, "AI")
}

func TestHealthMonitor_GenerateStatusMessage(t *testing.T) {
	checker := &MockHealthChecker{}
	monitor := NewHealthMonitor(checker)

	tests := []struct {
		name          string
		setup         func()
		expectedLevel models.DegradationLevel
		containsText  string
	}{
		{
			name:          "all healthy",
			setup:         func() {},
			expectedLevel: models.DegradationNormal,
			containsText:  "",
		},
		{
			name: "ai degraded",
			setup: func() {
				monitor.services.AI.RecordError("quota exceeded")
			},
			expectedLevel: models.DegradationPartial,
			containsText:  "AI",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset monitor
			monitor = NewHealthMonitor(checker)
			tt.setup()

			status := monitor.GetHealthStatus()
			assert.Equal(t, tt.expectedLevel, status.DegradationLevel)
			if tt.containsText != "" {
				assert.Contains(t, status.Message, tt.containsText)
			}
		})
	}
}

func TestHealthMonitor_GetServiceHealth_QBittorrent(t *testing.T) {
	checker := &MockHealthChecker{}
	monitor := NewHealthMonitor(checker)

	health := monitor.GetServiceHealth(models.ServiceNameQBittorrent)
	require.NotNil(t, health)
	assert.Equal(t, "qbittorrent", health.Name)
	assert.Equal(t, models.ServiceStatusHealthy, health.Status)
}

func TestHealthMonitor_UpdateServiceHealth_QBittorrent(t *testing.T) {
	checker := &MockHealthChecker{}
	monitor := NewHealthMonitor(checker)

	// Record an error
	monitor.UpdateServiceHealth(models.ServiceNameQBittorrent, errors.New("connection refused"))
	assert.Equal(t, models.ServiceStatusDegraded, monitor.services.QBittorrent.Status)
	assert.Equal(t, "connection refused", monitor.services.QBittorrent.Message)

	// Record success - should recover
	monitor.UpdateServiceHealth(models.ServiceNameQBittorrent, nil)
	assert.Equal(t, models.ServiceStatusHealthy, monitor.services.QBittorrent.Status)
}

func TestHealthMonitor_GetAllServiceStatuses(t *testing.T) {
	checker := &MockHealthChecker{}
	monitor := NewHealthMonitor(checker)

	// Set up some service states
	monitor.UpdateServiceHealth(models.ServiceNameTMDb, nil)
	monitor.UpdateServiceHealth(models.ServiceNameAI, errors.New("quota exceeded"))

	statuses := monitor.GetAllServiceStatuses()
	assert.Len(t, statuses, 5)

	// TMDb should be connected
	assert.Equal(t, models.StatusConnected, statuses[0].Status)

	// AI should be error
	found := false
	for _, s := range statuses {
		if s.Name == "ai" {
			assert.Equal(t, models.StatusError, s.Status)
			found = true
		}
	}
	assert.True(t, found, "AI status should be in results")
}

func TestHealthMonitor_UpdateServiceHealthWithTime(t *testing.T) {
	checker := &MockHealthChecker{}
	monitor := NewHealthMonitor(checker)

	monitor.UpdateServiceHealthWithTime(models.ServiceNameTMDb, nil, 150)

	svc := monitor.GetServiceHealth(models.ServiceNameTMDb)
	require.NotNil(t, svc)
	assert.Equal(t, int64(150), svc.ResponseTimeMs)
	assert.Equal(t, models.ServiceStatusHealthy, svc.Status)
}

func TestHealthMonitor_UpdateServiceHealthWithTime_Error(t *testing.T) {
	checker := &MockHealthChecker{}
	monitor := NewHealthMonitor(checker)

	monitor.UpdateServiceHealthWithTime(models.ServiceNameTMDb, errors.New("timeout"), 5000)

	svc := monitor.GetServiceHealth(models.ServiceNameTMDb)
	require.NotNil(t, svc)
	assert.Equal(t, int64(5000), svc.ResponseTimeMs)
	assert.Equal(t, models.ServiceStatusDegraded, svc.Status)
}

func TestHealthMonitor_CheckAllServices_TracksResponseTime(t *testing.T) {
	checker := &MockHealthChecker{}
	monitor := NewHealthMonitor(checker)

	ctx := context.Background()
	monitor.CheckAllServices(ctx)

	// Wait for goroutines
	time.Sleep(100 * time.Millisecond)

	// All services should have non-negative response times
	for _, svc := range monitor.services.AllServices() {
		assert.GreaterOrEqual(t, svc.ResponseTimeMs, int64(0))
	}
}

func TestHealthMonitor_CheckAllServices_QBittorrentError(t *testing.T) {
	checker := &MockHealthChecker{}
	checker.SetQBittorrentError(errors.New("qBittorrent unreachable"))

	monitor := NewHealthMonitor(checker)

	ctx := context.Background()
	monitor.CheckAllServices(ctx)

	// Wait for goroutines to complete
	time.Sleep(100 * time.Millisecond)

	assert.Equal(t, models.ServiceStatusDegraded, monitor.services.QBittorrent.Status)
	assert.Equal(t, models.ServiceStatusHealthy, monitor.services.TMDb.Status)
}
