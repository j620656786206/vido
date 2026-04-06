package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"

	"github.com/vido/api/internal/models"
)

func createBackupTestDB(t *testing.T, withTable bool) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)

	if withTable {
		_, err = db.Exec(`CREATE TABLE backups (
			id TEXT PRIMARY KEY,
			filename TEXT NOT NULL,
			size_bytes INTEGER NOT NULL DEFAULT 0,
			schema_version INTEGER NOT NULL,
			checksum TEXT NOT NULL DEFAULT '',
			status TEXT NOT NULL DEFAULT 'pending',
			error_message TEXT NOT NULL DEFAULT '',
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`)
		require.NoError(t, err)
	}

	return db
}

func TestBackupRepository_List_MissingTable(t *testing.T) {
	t.Run("returns ErrTableMissing when backups table does not exist", func(t *testing.T) {
		db := createBackupTestDB(t, false)
		defer db.Close()

		repo := NewBackupRepository(db)
		ctx := context.Background()

		result, err := repo.List(ctx)
		assert.Nil(t, result)
		assert.Error(t, err)
		assert.True(t, errors.Is(err, ErrTableMissing), "error should wrap ErrTableMissing, got: %v", err)
	})
}

func TestBackupRepository_TotalSizeBytes_MissingTable(t *testing.T) {
	t.Run("returns ErrTableMissing when backups table does not exist", func(t *testing.T) {
		db := createBackupTestDB(t, false)
		defer db.Close()

		repo := NewBackupRepository(db)
		ctx := context.Background()

		total, err := repo.TotalSizeBytes(ctx)
		assert.Equal(t, int64(0), total)
		assert.Error(t, err)
		assert.True(t, errors.Is(err, ErrTableMissing), "error should wrap ErrTableMissing, got: %v", err)
	})
}

func TestBackupRepository_List_EmptyTable(t *testing.T) {
	t.Run("returns empty slice when table exists but has no rows", func(t *testing.T) {
		db := createBackupTestDB(t, true)
		defer db.Close()

		repo := NewBackupRepository(db)
		ctx := context.Background()

		result, err := repo.List(ctx)
		assert.NoError(t, err)
		assert.Empty(t, result)
	})
}

func TestBackupRepository_TotalSizeBytes_EmptyTable(t *testing.T) {
	t.Run("returns 0 when table exists but has no completed backups", func(t *testing.T) {
		db := createBackupTestDB(t, true)
		defer db.Close()

		repo := NewBackupRepository(db)
		ctx := context.Background()

		total, err := repo.TotalSizeBytes(ctx)
		assert.NoError(t, err)
		assert.Equal(t, int64(0), total)
	})
}

func TestBackupRepository_CRUD(t *testing.T) {
	ctx := context.Background()

	t.Run("Create and List round-trip", func(t *testing.T) {
		db := createBackupTestDB(t, true)
		defer db.Close()

		repo := NewBackupRepository(db)

		backup := &models.Backup{
			ID:            "b1",
			Filename:      "test-backup.tar.gz",
			SizeBytes:     1024,
			SchemaVersion: 17,
			Checksum:      "abc123",
			Status:        models.BackupStatusCompleted,
			CreatedAt:     time.Now(),
		}
		require.NoError(t, repo.Create(ctx, backup))

		backups, err := repo.List(ctx)
		assert.NoError(t, err)
		assert.Len(t, backups, 1)
		assert.Equal(t, "b1", backups[0].ID)
		assert.Equal(t, "test-backup.tar.gz", backups[0].Filename)
	})

	t.Run("TotalSizeBytes sums completed backups only", func(t *testing.T) {
		db := createBackupTestDB(t, true)
		defer db.Close()

		repo := NewBackupRepository(db)

		// Completed backup
		require.NoError(t, repo.Create(ctx, &models.Backup{
			ID: "b1", Filename: "a.tar.gz", SizeBytes: 1000,
			SchemaVersion: 17, Status: models.BackupStatusCompleted, CreatedAt: time.Now(),
		}))
		// Failed backup — should NOT be counted
		require.NoError(t, repo.Create(ctx, &models.Backup{
			ID: "b2", Filename: "b.tar.gz", SizeBytes: 2000,
			SchemaVersion: 17, Status: models.BackupStatusFailed, CreatedAt: time.Now(),
		}))

		total, err := repo.TotalSizeBytes(ctx)
		assert.NoError(t, err)
		assert.Equal(t, int64(1000), total)
	})

	t.Run("GetByID returns nil for non-existent backup", func(t *testing.T) {
		db := createBackupTestDB(t, true)
		defer db.Close()

		repo := NewBackupRepository(db)

		result, err := repo.GetByID(ctx, "nonexistent")
		assert.NoError(t, err)
		assert.Nil(t, result)
	})

	t.Run("Delete removes backup record", func(t *testing.T) {
		db := createBackupTestDB(t, true)
		defer db.Close()

		repo := NewBackupRepository(db)

		require.NoError(t, repo.Create(ctx, &models.Backup{
			ID: "b1", Filename: "a.tar.gz", SchemaVersion: 17, Status: "completed", CreatedAt: time.Now(),
		}))

		require.NoError(t, repo.Delete(ctx, "b1"))

		result, err := repo.GetByID(ctx, "b1")
		assert.NoError(t, err)
		assert.Nil(t, result)
	})
}

func TestIsTableMissing(t *testing.T) {
	t.Run("detects SQLite no such table error", func(t *testing.T) {
		err := errors.New("no such table: backups")
		assert.True(t, isTableMissing(err))
	})

	t.Run("detects wrapped no such table error", func(t *testing.T) {
		inner := errors.New("no such table: backups")
		wrapped := fmt.Errorf("query failed: %w", inner)
		assert.True(t, isTableMissing(wrapped))
	})

	t.Run("returns false for nil error", func(t *testing.T) {
		assert.False(t, isTableMissing(nil))
	})

	t.Run("returns false for unrelated error", func(t *testing.T) {
		err := errors.New("connection refused")
		assert.False(t, isTableMissing(err))
	})

	t.Run("returns false for similar but different error", func(t *testing.T) {
		err := errors.New("no such column: backups")
		assert.False(t, isTableMissing(err))
	})
}
