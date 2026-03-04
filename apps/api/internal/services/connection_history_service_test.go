package services

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vido/api/internal/models"
)

// mockConnectionHistoryRepo is a test double for ConnectionHistoryRepositoryInterface
type mockConnectionHistoryRepo struct {
	events       []models.ConnectionEvent
	createCalled int
	createErr    error
	getErr       error
}

func (m *mockConnectionHistoryRepo) Create(_ context.Context, event *models.ConnectionEvent) error {
	m.createCalled++
	if m.createErr != nil {
		return m.createErr
	}
	m.events = append(m.events, *event)
	return nil
}

func (m *mockConnectionHistoryRepo) GetHistory(_ context.Context, service string, limit int) ([]models.ConnectionEvent, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	var result []models.ConnectionEvent
	for _, e := range m.events {
		if e.Service == service {
			result = append(result, e)
		}
	}
	if limit > 0 && len(result) > limit {
		result = result[:limit]
	}
	return result, nil
}

func TestConnectionHistoryService_RecordEvent(t *testing.T) {
	t.Run("successfully records event", func(t *testing.T) {
		repo := &mockConnectionHistoryRepo{}
		svc := NewConnectionHistoryService(repo)

		err := svc.RecordEvent(context.Background(), "qbittorrent", models.EventDisconnected, models.ServiceStatusDown, "connection refused")
		require.NoError(t, err)

		assert.Equal(t, 1, repo.createCalled)
		assert.Len(t, repo.events, 1)
		assert.Equal(t, "qbittorrent", repo.events[0].Service)
		assert.Equal(t, models.EventDisconnected, repo.events[0].EventType)
		assert.Equal(t, models.ServiceStatusDown, repo.events[0].Status)
		assert.Equal(t, "connection refused", repo.events[0].Message)
		assert.NotEmpty(t, repo.events[0].ID)
	})

	t.Run("returns error when repo fails", func(t *testing.T) {
		repo := &mockConnectionHistoryRepo{createErr: fmt.Errorf("db error")}
		svc := NewConnectionHistoryService(repo)

		err := svc.RecordEvent(context.Background(), "qbittorrent", models.EventError, models.ServiceStatusDegraded, "timeout")
		assert.Error(t, err)
		assert.Equal(t, 1, repo.createCalled)
	})
}

func TestConnectionHistoryService_GetHistory(t *testing.T) {
	t.Run("returns events for service", func(t *testing.T) {
		repo := &mockConnectionHistoryRepo{
			events: []models.ConnectionEvent{
				{ID: "1", Service: "qbittorrent", EventType: models.EventConnected, Status: models.ServiceStatusHealthy},
				{ID: "2", Service: "tmdb", EventType: models.EventError, Status: models.ServiceStatusDegraded},
				{ID: "3", Service: "qbittorrent", EventType: models.EventDisconnected, Status: models.ServiceStatusDown},
			},
		}
		svc := NewConnectionHistoryService(repo)

		events, err := svc.GetHistory(context.Background(), "qbittorrent", 20)
		require.NoError(t, err)
		assert.Len(t, events, 2)
	})

	t.Run("returns error when repo fails", func(t *testing.T) {
		repo := &mockConnectionHistoryRepo{getErr: fmt.Errorf("db error")}
		svc := NewConnectionHistoryService(repo)

		_, err := svc.GetHistory(context.Background(), "qbittorrent", 20)
		assert.Error(t, err)
	})
}

func TestIsValidServiceName(t *testing.T) {
	tests := []struct {
		name  string
		valid bool
	}{
		{"qbittorrent", true},
		{"tmdb", true},
		{"douban", true},
		{"wikipedia", true},
		{"ai", true},
		{"unknown", false},
		{"", false},
		{"QBITTORRENT", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.valid, IsValidServiceName(tt.name))
		})
	}
}
