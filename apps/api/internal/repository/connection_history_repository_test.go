package repository

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vido/api/internal/models"
)

func setupConnectionHistoryTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)

	_, err = db.Exec(`
		CREATE TABLE connection_history (
			id TEXT PRIMARY KEY,
			service TEXT NOT NULL,
			event_type TEXT NOT NULL,
			status TEXT NOT NULL,
			message TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
		CREATE INDEX idx_connection_history_service ON connection_history(service);
		CREATE INDEX idx_connection_history_created_at ON connection_history(created_at);
	`)
	require.NoError(t, err)

	t.Cleanup(func() { db.Close() })
	return db
}

func TestNewConnectionHistoryRepository(t *testing.T) {
	db := setupConnectionHistoryTestDB(t)
	repo := NewConnectionHistoryRepository(db)
	assert.NotNil(t, repo)
}

func TestConnectionHistoryRepository_Create(t *testing.T) {
	db := setupConnectionHistoryTestDB(t)
	repo := NewConnectionHistoryRepository(db)
	ctx := context.Background()

	event := &models.ConnectionEvent{
		ID:        "evt-1",
		Service:   "qbittorrent",
		EventType: models.EventDisconnected,
		Status:    models.ServiceStatusDown,
		Message:   "connection refused",
		CreatedAt: time.Now(),
	}

	err := repo.Create(ctx, event)
	assert.NoError(t, err)

	// Verify it was stored
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM connection_history WHERE id = ?", "evt-1").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestConnectionHistoryRepository_Create_DuplicateID(t *testing.T) {
	db := setupConnectionHistoryTestDB(t)
	repo := NewConnectionHistoryRepository(db)
	ctx := context.Background()

	event := &models.ConnectionEvent{
		ID:        "evt-dup",
		Service:   "qbittorrent",
		EventType: models.EventConnected,
		Status:    models.ServiceStatusHealthy,
		CreatedAt: time.Now(),
	}

	err := repo.Create(ctx, event)
	assert.NoError(t, err)

	// Duplicate should fail
	err = repo.Create(ctx, event)
	assert.Error(t, err)
}

func TestConnectionHistoryRepository_GetHistory(t *testing.T) {
	db := setupConnectionHistoryTestDB(t)
	repo := NewConnectionHistoryRepository(db)
	ctx := context.Background()

	now := time.Now()

	// Insert events in chronological order
	events := []*models.ConnectionEvent{
		{ID: "evt-1", Service: "qbittorrent", EventType: models.EventConnected, Status: models.ServiceStatusHealthy, CreatedAt: now.Add(-3 * time.Minute)},
		{ID: "evt-2", Service: "qbittorrent", EventType: models.EventDisconnected, Status: models.ServiceStatusDown, Message: "timeout", CreatedAt: now.Add(-2 * time.Minute)},
		{ID: "evt-3", Service: "qbittorrent", EventType: models.EventRecovered, Status: models.ServiceStatusHealthy, CreatedAt: now.Add(-1 * time.Minute)},
		{ID: "evt-4", Service: "tmdb", EventType: models.EventError, Status: models.ServiceStatusDegraded, Message: "rate limited", CreatedAt: now},
	}

	for _, e := range events {
		require.NoError(t, repo.Create(ctx, e))
	}

	// Get qbittorrent history
	history, err := repo.GetHistory(ctx, "qbittorrent", 10)
	require.NoError(t, err)
	assert.Len(t, history, 3)

	// Should be in reverse chronological order
	assert.Equal(t, "evt-3", history[0].ID)
	assert.Equal(t, models.EventRecovered, history[0].EventType)
	assert.Equal(t, "evt-2", history[1].ID)
	assert.Equal(t, "evt-1", history[2].ID)
}

func TestConnectionHistoryRepository_GetHistory_WithLimit(t *testing.T) {
	db := setupConnectionHistoryTestDB(t)
	repo := NewConnectionHistoryRepository(db)
	ctx := context.Background()

	now := time.Now()
	for i := 0; i < 5; i++ {
		require.NoError(t, repo.Create(ctx, &models.ConnectionEvent{
			ID:        fmt.Sprintf("evt-%d", i),
			Service:   "qbittorrent",
			EventType: models.EventConnected,
			Status:    models.ServiceStatusHealthy,
			CreatedAt: now.Add(time.Duration(i) * time.Minute),
		}))
	}

	history, err := repo.GetHistory(ctx, "qbittorrent", 2)
	require.NoError(t, err)
	assert.Len(t, history, 2)
}

func TestConnectionHistoryRepository_GetHistory_DefaultLimit(t *testing.T) {
	db := setupConnectionHistoryTestDB(t)
	repo := NewConnectionHistoryRepository(db)
	ctx := context.Background()

	// With limit 0, should default to 20
	history, err := repo.GetHistory(ctx, "qbittorrent", 0)
	require.NoError(t, err)
	assert.Empty(t, history)
}

func TestConnectionHistoryRepository_GetHistory_EmptyResult(t *testing.T) {
	db := setupConnectionHistoryTestDB(t)
	repo := NewConnectionHistoryRepository(db)
	ctx := context.Background()

	history, err := repo.GetHistory(ctx, "nonexistent", 10)
	assert.NoError(t, err)
	assert.Empty(t, history)
}

func TestConnectionHistoryRepository_GetHistory_EmptyMessage(t *testing.T) {
	db := setupConnectionHistoryTestDB(t)
	repo := NewConnectionHistoryRepository(db)
	ctx := context.Background()

	event := &models.ConnectionEvent{
		ID:        "evt-no-msg",
		Service:   "qbittorrent",
		EventType: models.EventConnected,
		Status:    models.ServiceStatusHealthy,
		CreatedAt: time.Now(),
	}
	require.NoError(t, repo.Create(ctx, event))

	history, err := repo.GetHistory(ctx, "qbittorrent", 10)
	require.NoError(t, err)
	assert.Len(t, history, 1)
	assert.Equal(t, "", history[0].Message)
}
