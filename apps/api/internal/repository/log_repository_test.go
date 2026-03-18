package repository

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vido/api/internal/models"
)

func setupLogTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })

	_, err = db.Exec(`
		CREATE TABLE system_logs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			level TEXT NOT NULL,
			message TEXT NOT NULL,
			source TEXT,
			context_json TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
		CREATE INDEX idx_system_logs_level ON system_logs(level);
		CREATE INDEX idx_system_logs_created_at ON system_logs(created_at DESC);
	`)
	require.NoError(t, err)
	return db
}

func TestLogRepository_CreateLog(t *testing.T) {
	db := setupLogTestDB(t)
	repo := NewLogRepository(db)
	ctx := context.Background()

	tests := []struct {
		name    string
		log     *models.SystemLog
		wantErr bool
	}{
		{
			name: "success",
			log: &models.SystemLog{
				Level:       models.LogLevelInfo,
				Message:     "Test message",
				Source:      "test",
				ContextJSON: `{"key": "value"}`,
				CreatedAt:   time.Now(),
			},
		},
		{
			name:    "nil log",
			log:     nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := repo.CreateLog(ctx, tt.log)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestLogRepository_GetLogs(t *testing.T) {
	db := setupLogTestDB(t)
	repo := NewLogRepository(db)
	ctx := context.Background()

	// Seed test data
	now := time.Now()
	logs := []models.SystemLog{
		{Level: models.LogLevelError, Message: "Error occurred", Source: "tmdb", ContextJSON: `{"code": "TMDB_TIMEOUT"}`, CreatedAt: now.Add(-3 * time.Hour)},
		{Level: models.LogLevelWarn, Message: "Warning message", Source: "cache", CreatedAt: now.Add(-2 * time.Hour)},
		{Level: models.LogLevelInfo, Message: "Info message", Source: "api", CreatedAt: now.Add(-1 * time.Hour)},
		{Level: models.LogLevelDebug, Message: "Debug detail", Source: "parser", CreatedAt: now},
	}

	for i := range logs {
		require.NoError(t, repo.CreateLog(ctx, &logs[i]))
	}

	t.Run("all logs default pagination", func(t *testing.T) {
		result, total, err := repo.GetLogs(ctx, models.LogFilter{})
		require.NoError(t, err)
		assert.Equal(t, 4, total)
		assert.Len(t, result, 4)
		// Newest first
		assert.Equal(t, models.LogLevelDebug, result[0].Level)
		assert.Equal(t, models.LogLevelError, result[3].Level)
	})

	t.Run("filter by level", func(t *testing.T) {
		result, total, err := repo.GetLogs(ctx, models.LogFilter{Level: models.LogLevelError})
		require.NoError(t, err)
		assert.Equal(t, 1, total)
		assert.Len(t, result, 1)
		assert.Equal(t, "Error occurred", result[0].Message)
	})

	t.Run("filter by keyword", func(t *testing.T) {
		result, total, err := repo.GetLogs(ctx, models.LogFilter{Keyword: "tmdb"})
		require.NoError(t, err)
		assert.Equal(t, 1, total)
		assert.Len(t, result, 1)
		assert.Equal(t, "Error occurred", result[0].Message)
	})

	t.Run("keyword in message", func(t *testing.T) {
		result, total, err := repo.GetLogs(ctx, models.LogFilter{Keyword: "Warning"})
		require.NoError(t, err)
		assert.Equal(t, 1, total)
		assert.Len(t, result, 1)
	})

	t.Run("combined filters", func(t *testing.T) {
		result, total, err := repo.GetLogs(ctx, models.LogFilter{Level: models.LogLevelError, Keyword: "Error"})
		require.NoError(t, err)
		assert.Equal(t, 1, total)
		assert.Len(t, result, 1)
	})

	t.Run("pagination", func(t *testing.T) {
		result, total, err := repo.GetLogs(ctx, models.LogFilter{Page: 1, PerPage: 2})
		require.NoError(t, err)
		assert.Equal(t, 4, total)
		assert.Len(t, result, 2)
	})

	t.Run("page 2", func(t *testing.T) {
		result, total, err := repo.GetLogs(ctx, models.LogFilter{Page: 2, PerPage: 2})
		require.NoError(t, err)
		assert.Equal(t, 4, total)
		assert.Len(t, result, 2)
	})

	t.Run("no results", func(t *testing.T) {
		result, total, err := repo.GetLogs(ctx, models.LogFilter{Keyword: "nonexistent"})
		require.NoError(t, err)
		assert.Equal(t, 0, total)
		assert.Nil(t, result)
	})

	t.Run("defaults for invalid pagination", func(t *testing.T) {
		result, total, err := repo.GetLogs(ctx, models.LogFilter{Page: -1, PerPage: -1})
		require.NoError(t, err)
		assert.Equal(t, 4, total)
		assert.Len(t, result, 4)
	})
}

func TestLogRepository_CreateLogBatch(t *testing.T) {
	db := setupLogTestDB(t)
	repo := NewLogRepository(db)
	ctx := context.Background()

	t.Run("empty batch", func(t *testing.T) {
		err := repo.CreateLogBatch(ctx, nil)
		assert.NoError(t, err)
	})

	t.Run("batch insert", func(t *testing.T) {
		now := time.Now()
		batch := []models.SystemLog{
			{Level: models.LogLevelInfo, Message: "Batch 1", CreatedAt: now},
			{Level: models.LogLevelWarn, Message: "Batch 2", CreatedAt: now},
			{Level: models.LogLevelError, Message: "Batch 3", CreatedAt: now},
		}

		err := repo.CreateLogBatch(ctx, batch)
		require.NoError(t, err)

		result, total, err := repo.GetLogs(ctx, models.LogFilter{})
		require.NoError(t, err)
		assert.Equal(t, 3, total)
		assert.Len(t, result, 3)
	})
}

func TestLogRepository_DeleteOlderThan(t *testing.T) {
	db := setupLogTestDB(t)
	repo := NewLogRepository(db)
	ctx := context.Background()

	// Insert logs with different timestamps
	now := time.Now()
	logs := []models.SystemLog{
		{Level: models.LogLevelInfo, Message: "Recent", CreatedAt: now},
		{Level: models.LogLevelInfo, Message: "Old", CreatedAt: now.Add(-40 * 24 * time.Hour)},
	}
	for i := range logs {
		require.NoError(t, repo.CreateLog(ctx, &logs[i]))
	}

	t.Run("invalid days", func(t *testing.T) {
		_, err := repo.DeleteOlderThan(ctx, 0)
		assert.Error(t, err)

		_, err = repo.DeleteOlderThan(ctx, -1)
		assert.Error(t, err)
	})

	t.Run("delete old logs", func(t *testing.T) {
		deleted, err := repo.DeleteOlderThan(ctx, 30)
		require.NoError(t, err)
		assert.Equal(t, int64(1), deleted)

		// Only recent log remains
		result, total, err := repo.GetLogs(ctx, models.LogFilter{})
		require.NoError(t, err)
		assert.Equal(t, 1, total)
		assert.Equal(t, "Recent", result[0].Message)
	})
}
