package repository

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vido/api/internal/database/migrations"
	"github.com/vido/api/internal/models"
	_ "modernc.org/sqlite"
)

// setupRequestsDB creates an in-memory DB and applies the REAL migration
// chain (incl. 027) via the production runner, so this test can never drift
// from the shipped schema (CR M1 — no hand-copied schema literals).
func setupRequestsDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })

	runner, err := migrations.NewRunner(db)
	require.NoError(t, err)
	require.NoError(t, runner.RegisterAll(migrations.GetAll()))
	require.NoError(t, runner.Up(context.Background()))
	return db
}

func TestRequestRepository_Create(t *testing.T) {
	repo := NewRequestRepository(setupRequestsDB(t))
	ctx := context.Background()

	req := &models.Request{TMDbID: 550, MediaType: models.RequestMediaTypeMovie, Title: "鬥陣俱樂部"}
	require.NoError(t, repo.Create(ctx, req))

	assert.NotEmpty(t, req.ID, "Create must assign a uuid")
	assert.Equal(t, models.RequestStatusPending, req.Status, "rows are born pending")
	assert.False(t, req.RequestedAt.IsZero())
	assert.False(t, req.FulfilmentSource.Valid, "fulfilment_source stays NULL until 13-4")

	t.Run("nil request rejected", func(t *testing.T) {
		assert.Error(t, repo.Create(ctx, nil))
	})

	t.Run("active duplicate maps to ErrRequestDuplicate", func(t *testing.T) {
		dup := &models.Request{TMDbID: 550, MediaType: models.RequestMediaTypeMovie, Title: "x"}
		err := repo.Create(ctx, dup)
		assert.ErrorIs(t, err, ErrRequestDuplicate, "unique-index violation must surface as the typed sentinel, not a raw error")
	})
}

func TestRequestRepository_List(t *testing.T) {
	repo := NewRequestRepository(setupRequestsDB(t))
	ctx := context.Background()

	t.Run("empty table returns empty (nil) slice without error", func(t *testing.T) {
		requests, err := repo.List(ctx)
		require.NoError(t, err)
		assert.Empty(t, requests)
	})

	first := &models.Request{TMDbID: 550, MediaType: models.RequestMediaTypeMovie, Title: "first"}
	require.NoError(t, repo.Create(ctx, first))
	second := &models.Request{TMDbID: 1399, MediaType: models.RequestMediaTypeTV, Title: "second"}
	require.NoError(t, repo.Create(ctx, second))
	// Force a strictly older timestamp on the first row so DESC ordering is
	// deterministic even when both inserts land in the same clock tick. Pass a
	// Go time.Time so the driver serializes it identically to Create's insert.
	_, err := repo.db.Exec(`UPDATE requests SET requested_at = ? WHERE id = ?`, first.RequestedAt.Add(-time.Hour), first.ID)
	require.NoError(t, err)

	requests, err := repo.List(ctx)
	require.NoError(t, err)
	require.Len(t, requests, 2)
	assert.Equal(t, "second", requests[0].Title, "List orders requested_at DESC (newest first)")
	assert.Equal(t, "first", requests[1].Title)
}

func TestRequestRepository_FindActiveByTMDbID(t *testing.T) {
	repo := NewRequestRepository(setupRequestsDB(t))
	ctx := context.Background()

	req := &models.Request{TMDbID: 550, MediaType: models.RequestMediaTypeMovie, Title: "x"}
	require.NoError(t, repo.Create(ctx, req))

	t.Run("finds an active request", func(t *testing.T) {
		found, err := repo.FindActiveByTMDbID(ctx, 550, models.RequestMediaTypeMovie)
		require.NoError(t, err)
		assert.Equal(t, req.ID, found.ID)
	})

	t.Run("not found for other media_type", func(t *testing.T) {
		_, err := repo.FindActiveByTMDbID(ctx, 550, models.RequestMediaTypeTV)
		assert.ErrorIs(t, err, ErrRequestNotFound)
	})

	t.Run("terminal rows are not active", func(t *testing.T) {
		_, err := repo.db.Exec(`UPDATE requests SET status = 'failed' WHERE id = ?`, req.ID)
		require.NoError(t, err)
		_, err = repo.FindActiveByTMDbID(ctx, 550, models.RequestMediaTypeMovie)
		assert.ErrorIs(t, err, ErrRequestNotFound, "failed rows must not count as active")
	})
}

func TestRequestRepository_UpdateFulfilment(t *testing.T) {
	repo := NewRequestRepository(setupRequestsDB(t))
	ctx := context.Background()

	req := &models.Request{TMDbID: 550, MediaType: models.RequestMediaTypeMovie, Title: "x"}
	require.NoError(t, repo.Create(ctx, req))
	createdAt := req.UpdatedAt

	t.Run("success transition writes all fulfilment fields", func(t *testing.T) {
		time.Sleep(5 * time.Millisecond) // ensure updated_at moves
		writtenAt, err := repo.UpdateFulfilment(ctx, req.ID, models.RequestStatusSearching,
			models.NewNullString(models.RequestFulfilmentSourceArr),
			models.NewNullString("42"), models.NullString{})
		require.NoError(t, err)

		found, err := repo.FindActiveByTMDbID(ctx, 550, models.RequestMediaTypeMovie)
		require.NoError(t, err)
		assert.Equal(t, models.RequestStatusSearching, found.Status)
		assert.Equal(t, "arr", found.FulfilmentSource.String)
		assert.Equal(t, "42", found.ExternalID.String)
		assert.False(t, found.ErrorMessage.Valid, "success transition clears error_message")
		assert.True(t, found.UpdatedAt.After(createdAt), "updated_at must be bumped")
		assert.WithinDuration(t, writtenAt, found.UpdatedAt, time.Second,
			"returned timestamp must match the stored updated_at (CR M1)")
	})

	t.Run("failure annotation keeps status and sets zh-TW reason", func(t *testing.T) {
		req2 := &models.Request{TMDbID: 551, MediaType: models.RequestMediaTypeMovie, Title: "y"}
		require.NoError(t, repo.Create(ctx, req2))

		_, err := repo.UpdateFulfilment(ctx, req2.ID, models.RequestStatusPending,
			models.NullString{}, models.NullString{}, models.NewNullString("Radarr 未設定"))
		require.NoError(t, err)

		found, err := repo.FindActiveByTMDbID(ctx, 551, models.RequestMediaTypeMovie)
		require.NoError(t, err)
		assert.Equal(t, models.RequestStatusPending, found.Status)
		assert.Equal(t, "Radarr 未設定", found.ErrorMessage.String)
		assert.False(t, found.FulfilmentSource.Valid)
	})

	t.Run("unknown id returns ErrRequestNotFound", func(t *testing.T) {
		_, err := repo.UpdateFulfilment(ctx, "no-such-id", models.RequestStatusSearching,
			models.NullString{}, models.NullString{}, models.NullString{})
		assert.ErrorIs(t, err, ErrRequestNotFound)
	})
}

func TestRequestRepository_ListActive(t *testing.T) {
	repo := NewRequestRepository(setupRequestsDB(t))
	ctx := context.Background()

	t.Run("empty table returns empty slice", func(t *testing.T) {
		active, err := repo.ListActive(ctx)
		require.NoError(t, err)
		assert.Empty(t, active)
	})

	// Seed one row per status; only pending/searching/downloading are active.
	statuses := []string{
		models.RequestStatusPending, models.RequestStatusSearching, models.RequestStatusDownloading,
		models.RequestStatusCompleted, models.RequestStatusFailed,
	}
	for i, status := range statuses {
		req := &models.Request{TMDbID: int64(1000 + i), MediaType: models.RequestMediaTypeMovie, Title: status}
		require.NoError(t, repo.Create(ctx, req))
		if status != models.RequestStatusPending {
			_, err := repo.UpdateFulfilment(ctx, req.ID, status, models.NullString{}, models.NullString{}, models.NullString{})
			require.NoError(t, err)
		}
	}

	active, err := repo.ListActive(ctx)
	require.NoError(t, err)
	require.Len(t, active, 3, "only pending/searching/downloading are active")
	for _, r := range active {
		assert.Contains(t, []string{
			models.RequestStatusPending, models.RequestStatusSearching, models.RequestStatusDownloading,
		}, r.Status)
	}
	// Oldest-first so the reconciler treats rows fairly across ticks.
	assert.True(t, !active[0].RequestedAt.After(active[1].RequestedAt))
}

func TestRequestRepository_UpdateStatus(t *testing.T) {
	repo := NewRequestRepository(setupRequestsDB(t))
	ctx := context.Background()

	req := &models.Request{TMDbID: 550, MediaType: models.RequestMediaTypeMovie, Title: "x"}
	require.NoError(t, repo.Create(ctx, req))

	t.Run("status transition with error cleared", func(t *testing.T) {
		// Seed an error, then a clean transition must NULL it.
		_, err := repo.UpdateFulfilment(ctx, req.ID, models.RequestStatusPending,
			models.NullString{}, models.NullString{}, models.NewNullString("Radarr 連線失敗"))
		require.NoError(t, err)

		updatedAt, err := repo.UpdateStatus(ctx, req.ID, models.RequestStatusDownloading, "")
		require.NoError(t, err)

		found, err := repo.FindActiveByTMDbID(ctx, 550, models.RequestMediaTypeMovie)
		require.NoError(t, err)
		assert.Equal(t, models.RequestStatusDownloading, found.Status)
		assert.False(t, found.ErrorMessage.Valid, "empty errMsg clears error_message")
		assert.WithinDuration(t, updatedAt, found.UpdatedAt, time.Second)
	})

	t.Run("failed transition records zh-TW reason", func(t *testing.T) {
		_, err := repo.UpdateStatus(ctx, req.ID, models.RequestStatusFailed, "下載發生錯誤")
		require.NoError(t, err)

		requests, err := repo.List(ctx)
		require.NoError(t, err)
		require.Len(t, requests, 1)
		assert.Equal(t, models.RequestStatusFailed, requests[0].Status)
		assert.Equal(t, "下載發生錯誤", requests[0].ErrorMessage.String)
	})

	t.Run("unknown id returns ErrRequestNotFound", func(t *testing.T) {
		_, err := repo.UpdateStatus(ctx, "no-such-id", models.RequestStatusCompleted, "")
		assert.ErrorIs(t, err, ErrRequestNotFound)
	})
}
