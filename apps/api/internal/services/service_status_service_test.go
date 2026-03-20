package services

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vido/api/internal/health"
	"github.com/vido/api/internal/models"
)

func TestServiceStatusService_GetAllStatuses(t *testing.T) {
	checker := health.NewStubHealthChecker()
	monitor := health.NewHealthMonitor(checker)
	svc := NewServiceStatusService(monitor, checker)

	statuses, err := svc.GetAllStatuses(context.Background())
	require.NoError(t, err)
	assert.Len(t, statuses, 5)

	// All should be connected (stub checker returns healthy)
	for _, s := range statuses {
		assert.Equal(t, models.StatusConnected, s.Status, "service %s should be connected", s.Name)
	}
}

func TestServiceStatusService_GetAllStatuses_MixedStates(t *testing.T) {
	checker := health.NewStubHealthChecker()
	monitor := health.NewHealthMonitor(checker)
	svc := NewServiceStatusService(monitor, checker)

	// Simulate some failures
	monitor.UpdateServiceHealth(models.ServiceNameAI, errors.New("quota exceeded"))
	monitor.UpdateServiceHealth(models.ServiceNameQBittorrent, errors.New("connection refused"))
	monitor.UpdateServiceHealth(models.ServiceNameQBittorrent, errors.New("connection refused"))
	monitor.UpdateServiceHealth(models.ServiceNameQBittorrent, errors.New("connection refused"))

	statuses, err := svc.GetAllStatuses(context.Background())
	require.NoError(t, err)
	assert.Len(t, statuses, 5)

	statusMap := make(map[string]string)
	for _, s := range statuses {
		statusMap[s.Name] = s.Status
	}

	assert.Equal(t, models.StatusConnected, statusMap["tmdb"])
	assert.Equal(t, models.StatusError, statusMap["ai"])
	assert.Equal(t, models.StatusDisconnected, statusMap["qbittorrent"])
}

func TestServiceStatusService_TestService_Success(t *testing.T) {
	checker := health.NewStubHealthChecker()
	monitor := health.NewHealthMonitor(checker)
	svc := NewServiceStatusService(monitor, checker)

	status, err := svc.TestService(context.Background(), "tmdb")
	require.NoError(t, err)
	require.NotNil(t, status)
	assert.Equal(t, "tmdb", status.Name)
	assert.Equal(t, models.StatusConnected, status.Status)
	assert.GreaterOrEqual(t, status.ResponseTimeMs, int64(0))
}

func TestServiceStatusService_TestService_AllServices(t *testing.T) {
	checker := health.NewStubHealthChecker()
	monitor := health.NewHealthMonitor(checker)
	svc := NewServiceStatusService(monitor, checker)

	serviceNames := []string{"tmdb", "douban", "wikipedia", "ai", "qbittorrent"}
	for _, name := range serviceNames {
		t.Run(name, func(t *testing.T) {
			status, err := svc.TestService(context.Background(), name)
			require.NoError(t, err)
			require.NotNil(t, status)
			assert.Equal(t, name, status.Name)
			assert.Equal(t, models.StatusConnected, status.Status)
		})
	}
}

func TestServiceStatusService_TestService_InvalidService(t *testing.T) {
	checker := health.NewStubHealthChecker()
	monitor := health.NewHealthMonitor(checker)
	svc := NewServiceStatusService(monitor, checker)

	status, err := svc.TestService(context.Background(), "unknown")
	assert.Error(t, err)
	assert.Nil(t, status)
	assert.ErrorIs(t, err, ErrServiceNotFound)
}

func TestServiceStatusService_TestService_TracksResponseTime(t *testing.T) {
	checker := health.NewStubHealthChecker()
	monitor := health.NewHealthMonitor(checker)
	svc := NewServiceStatusService(monitor, checker)

	status, err := svc.TestService(context.Background(), "tmdb")
	require.NoError(t, err)
	assert.GreaterOrEqual(t, status.ResponseTimeMs, int64(0))
}
