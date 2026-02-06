package services

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vido/api/internal/health"
	"github.com/vido/api/internal/metadata"
	"github.com/vido/api/internal/models"
)

// MockHealthChecker for testing
type MockHealthChecker struct{}

func (m *MockHealthChecker) CheckTMDb(ctx context.Context) error      { return nil }
func (m *MockHealthChecker) CheckDouban(ctx context.Context) error    { return nil }
func (m *MockHealthChecker) CheckWikipedia(ctx context.Context) error { return nil }
func (m *MockHealthChecker) CheckAI(ctx context.Context) error        { return nil }

func TestNewDegradationService(t *testing.T) {
	checker := &MockHealthChecker{}
	monitor := health.NewHealthMonitor(checker)
	service := NewDegradationService(monitor)

	require.NotNil(t, service)
	assert.NotNil(t, service.monitor)
	assert.NotNil(t, service.partialHandler)
}

func TestDegradationService_GetCurrentLevel(t *testing.T) {
	checker := &MockHealthChecker{}
	monitor := health.NewHealthMonitor(checker)
	service := NewDegradationService(monitor)

	level := service.GetCurrentLevel()
	assert.Equal(t, models.DegradationNormal, level)
}

func TestDegradationService_GetServiceHealth(t *testing.T) {
	checker := &MockHealthChecker{}
	monitor := health.NewHealthMonitor(checker)
	service := NewDegradationService(monitor)

	health := service.GetServiceHealth(models.ServiceNameTMDb)
	require.NotNil(t, health)
	assert.Equal(t, "tmdb", health.Name)
	assert.Equal(t, models.ServiceStatusHealthy, health.Status)
}

func TestDegradationService_GetServiceHealth_Unknown(t *testing.T) {
	checker := &MockHealthChecker{}
	monitor := health.NewHealthMonitor(checker)
	service := NewDegradationService(monitor)

	health := service.GetServiceHealth(models.ServiceName("unknown"))
	assert.Nil(t, health)
}

func TestDegradationService_GetDegradedResult(t *testing.T) {
	checker := &MockHealthChecker{}
	monitor := health.NewHealthMonitor(checker)
	service := NewDegradationService(monitor)

	results := []*metadata.MetadataResult{
		{
			Source:    "tmdb",
			Title:     "The Matrix",
			Year:      1999,
			Overview:  "A computer hacker learns about the true nature of reality.",
			PosterURL: "https://image.tmdb.org/t/p/w500/matrix.jpg",
		},
	}

	degraded := service.GetDegradedResult(results)

	require.NotNil(t, degraded)
	assert.Equal(t, models.DegradationNormal, degraded.DegradationLevel)
}

func TestDegradationService_GetHealthStatus(t *testing.T) {
	checker := &MockHealthChecker{}
	monitor := health.NewHealthMonitor(checker)
	service := NewDegradationService(monitor)

	status := service.GetHealthStatus()

	require.NotNil(t, status)
	assert.Equal(t, models.DegradationNormal, status.DegradationLevel)
	assert.NotNil(t, status.Services)
}

func TestDegradationServiceInterface(t *testing.T) {
	checker := &MockHealthChecker{}
	monitor := health.NewHealthMonitor(checker)

	var _ DegradationServiceInterface = NewDegradationService(monitor)
}
