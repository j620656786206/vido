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

// --- Story 13-7a: cancel / retry writers ---

func seedRequest(t *testing.T, repo *RequestRepository, tmdbID int64, status string, external string) *models.Request {
	t.Helper()
	ctx := context.Background()
	req := &models.Request{TMDbID: tmdbID, MediaType: models.RequestMediaTypeMovie, Title: "片"}
	require.NoError(t, repo.Create(ctx, req))
	if status != models.RequestStatusPending || external != "" {
		var src, ext models.NullString
		if external != "" {
			src = models.NewNullString(models.RequestFulfilmentSourceArr)
			ext = models.NewNullString(external)
		}
		msg := models.NullString{}
		if status == models.RequestStatusFailed {
			msg = models.NewNullString("下載發生錯誤，請重試或檢查下載器")
		}
		_, err := repo.UpdateFulfilment(ctx, req.ID, status, src, ext, msg)
		require.NoError(t, err)
	}
	got, err := repo.FindByID(ctx, req.ID)
	require.NoError(t, err)
	return got
}

func TestRequestRepository_FindByID(t *testing.T) {
	repo := NewRequestRepository(setupRequestsDB(t))
	req := seedRequest(t, repo, 550, models.RequestStatusPending, "")
	assert.Equal(t, int64(550), req.TMDbID)

	_, err := repo.FindByID(context.Background(), "nope")
	assert.ErrorIs(t, err, ErrRequestNotFound)
}

func TestRequestRepository_DeleteIfPending(t *testing.T) {
	repo := NewRequestRepository(setupRequestsDB(t))
	ctx := context.Background()

	pending := seedRequest(t, repo, 1, models.RequestStatusPending, "")
	n, err := repo.DeleteIfPending(ctx, pending.ID)
	require.NoError(t, err)
	assert.EqualValues(t, 1, n)
	_, err = repo.FindByID(ctx, pending.ID)
	assert.ErrorIs(t, err, ErrRequestNotFound, "cancel is a hard delete")

	// Deleting frees the active-unique index: the same title is re-requestable.
	require.NoError(t, repo.Create(ctx, &models.Request{TMDbID: 1, MediaType: models.RequestMediaTypeMovie, Title: "片"}))

	for i, status := range []string{models.RequestStatusSearching, models.RequestStatusDownloading, models.RequestStatusCompleted, models.RequestStatusFailed} {
		row := seedRequest(t, repo, 100+int64(i), status, "9")
		n, err := repo.DeleteIfPending(ctx, row.ID)
		require.NoError(t, err)
		assert.Zero(t, n, "a %s row is not deleted", status)
		_, err = repo.FindByID(ctx, row.ID)
		assert.NoError(t, err, "the %s row is still there", status)
	}

	n, err = repo.DeleteIfPending(ctx, "unknown")
	require.NoError(t, err)
	assert.Zero(t, n)
}

func TestRequestRepository_ResetForRetry(t *testing.T) {
	repo := NewRequestRepository(setupRequestsDB(t))
	ctx := context.Background()

	t.Run("failed with external id → searching, keeps the *arr link", func(t *testing.T) {
		row := seedRequest(t, repo, 10, models.RequestStatusFailed, "77")
		got, err := repo.ResetForRetry(ctx, row.ID, models.RequestStatusSearching)
		require.NoError(t, err)
		assert.Equal(t, models.RequestStatusSearching, got.Status)
		assert.False(t, got.ErrorMessage.Valid, "error cleared")
		assert.Equal(t, "77", got.ExternalID.String)
		assert.Equal(t, models.RequestFulfilmentSourceArr, got.FulfilmentSource.String)
		assert.True(t, !got.UpdatedAt.Before(row.UpdatedAt))
	})

	t.Run("terminal failed → pending", func(t *testing.T) {
		row := seedRequest(t, repo, 11, models.RequestStatusFailed, "")
		got, err := repo.ResetForRetry(ctx, row.ID, models.RequestStatusPending)
		require.NoError(t, err)
		assert.Equal(t, models.RequestStatusPending, got.Status)
		assert.False(t, got.ExternalID.Valid)
	})

	t.Run("only failed rows move", func(t *testing.T) {
		row := seedRequest(t, repo, 12, models.RequestStatusDownloading, "5")
		_, err := repo.ResetForRetry(ctx, row.ID, models.RequestStatusSearching)
		assert.ErrorIs(t, err, ErrRequestNotFound)
		again, _ := repo.FindByID(ctx, row.ID)
		assert.Equal(t, models.RequestStatusDownloading, again.Status)

		_, err = repo.ResetForRetry(ctx, "unknown", models.RequestStatusPending)
		assert.ErrorIs(t, err, ErrRequestNotFound)
	})

	t.Run("a newer active request for the same title blocks the reset", func(t *testing.T) {
		row := seedRequest(t, repo, 13, models.RequestStatusFailed, "")
		require.NoError(t, repo.Create(ctx, &models.Request{TMDbID: 13, MediaType: models.RequestMediaTypeMovie, Title: "片"}))
		_, err := repo.ResetForRetry(ctx, row.ID, models.RequestStatusPending)
		assert.ErrorIs(t, err, ErrRequestDuplicate)
	})
}
