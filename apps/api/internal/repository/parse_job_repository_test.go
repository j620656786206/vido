package repository

import (
	"context"
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vido/api/internal/models"
	_ "modernc.org/sqlite"
)

func setupParseJobTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)

	_, err = db.Exec(`
		CREATE TABLE parse_jobs (
			id TEXT PRIMARY KEY,
			torrent_hash TEXT NOT NULL,
			file_path TEXT NOT NULL,
			file_name TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'pending',
			media_id TEXT,
			error_message TEXT,
			retry_count INTEGER DEFAULT 0,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			completed_at TIMESTAMP,
			UNIQUE(torrent_hash)
		);
		CREATE INDEX idx_parse_jobs_status ON parse_jobs(status);
		CREATE INDEX idx_parse_jobs_torrent_hash ON parse_jobs(torrent_hash);
	`)
	require.NoError(t, err)

	return db
}

func createTestParseJob(id, hash, filePath, fileName string) *models.ParseJob {
	return &models.ParseJob{
		ID:          id,
		TorrentHash: hash,
		FilePath:    filePath,
		FileName:    fileName,
		Status:      models.ParseJobPending,
	}
}

func TestParseJobRepository_Create(t *testing.T) {
	db := setupParseJobTestDB(t)
	defer db.Close()

	repo := NewParseJobRepository(db)
	ctx := context.Background()

	job := createTestParseJob("job-1", "hash1", "/downloads/movie.mkv", "movie.mkv")
	err := repo.Create(ctx, job)
	require.NoError(t, err)

	found, err := repo.GetByID(ctx, "job-1")
	require.NoError(t, err)
	assert.Equal(t, "job-1", found.ID)
	assert.Equal(t, "hash1", found.TorrentHash)
	assert.Equal(t, models.ParseJobPending, found.Status)
}

func TestParseJobRepository_Create_NilJob(t *testing.T) {
	db := setupParseJobTestDB(t)
	defer db.Close()

	repo := NewParseJobRepository(db)
	err := repo.Create(context.Background(), nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot be nil")
}

func TestParseJobRepository_Create_DuplicateHash(t *testing.T) {
	db := setupParseJobTestDB(t)
	defer db.Close()

	repo := NewParseJobRepository(db)
	ctx := context.Background()

	job1 := createTestParseJob("job-1", "hash1", "/a", "a.mkv")
	err := repo.Create(ctx, job1)
	require.NoError(t, err)

	job2 := createTestParseJob("job-2", "hash1", "/b", "b.mkv")
	err = repo.Create(ctx, job2)
	assert.Error(t, err, "duplicate torrent_hash should fail")
}

func TestParseJobRepository_GetByID_NotFound(t *testing.T) {
	db := setupParseJobTestDB(t)
	defer db.Close()

	repo := NewParseJobRepository(db)
	_, err := repo.GetByID(context.Background(), "nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestParseJobRepository_GetByTorrentHash(t *testing.T) {
	db := setupParseJobTestDB(t)
	defer db.Close()

	repo := NewParseJobRepository(db)
	ctx := context.Background()

	job := createTestParseJob("job-1", "abc123", "/downloads/movie.mkv", "movie.mkv")
	err := repo.Create(ctx, job)
	require.NoError(t, err)

	found, err := repo.GetByTorrentHash(ctx, "abc123")
	require.NoError(t, err)
	require.NotNil(t, found)
	assert.Equal(t, "job-1", found.ID)
}

func TestParseJobRepository_GetByTorrentHash_NotFound(t *testing.T) {
	db := setupParseJobTestDB(t)
	defer db.Close()

	repo := NewParseJobRepository(db)
	found, err := repo.GetByTorrentHash(context.Background(), "nonexistent")
	require.NoError(t, err)
	assert.Nil(t, found)
}

func TestParseJobRepository_GetPending(t *testing.T) {
	db := setupParseJobTestDB(t)
	defer db.Close()

	repo := NewParseJobRepository(db)
	ctx := context.Background()

	// Create 3 pending and 1 processing job
	for i, status := range []models.ParseJobStatus{
		models.ParseJobPending, models.ParseJobPending, models.ParseJobPending, models.ParseJobProcessing,
	} {
		job := createTestParseJob(
			"job-"+string(rune('1'+i)),
			"hash"+string(rune('1'+i)),
			"/downloads/file"+string(rune('1'+i)),
			"file"+string(rune('1'+i)),
		)
		job.Status = status
		err := repo.Create(ctx, job)
		require.NoError(t, err)
	}

	pending, err := repo.GetPending(ctx, 10)
	require.NoError(t, err)
	assert.Len(t, pending, 3)

	// Test limit
	pending2, err := repo.GetPending(ctx, 2)
	require.NoError(t, err)
	assert.Len(t, pending2, 2)
}

func TestParseJobRepository_UpdateStatus(t *testing.T) {
	db := setupParseJobTestDB(t)
	defer db.Close()

	repo := NewParseJobRepository(db)
	ctx := context.Background()

	job := createTestParseJob("job-1", "hash1", "/downloads/movie.mkv", "movie.mkv")
	err := repo.Create(ctx, job)
	require.NoError(t, err)

	// Update to processing
	err = repo.UpdateStatus(ctx, "job-1", models.ParseJobProcessing, "")
	require.NoError(t, err)

	found, err := repo.GetByID(ctx, "job-1")
	require.NoError(t, err)
	assert.Equal(t, models.ParseJobProcessing, found.Status)
	assert.Nil(t, found.ErrorMessage)

	// Update to failed with error
	err = repo.UpdateStatus(ctx, "job-1", models.ParseJobFailed, "parse error")
	require.NoError(t, err)

	found, err = repo.GetByID(ctx, "job-1")
	require.NoError(t, err)
	assert.Equal(t, models.ParseJobFailed, found.Status)
	require.NotNil(t, found.ErrorMessage)
	assert.Equal(t, "parse error", *found.ErrorMessage)
	assert.NotNil(t, found.CompletedAt)
}

func TestParseJobRepository_UpdateStatus_NotFound(t *testing.T) {
	db := setupParseJobTestDB(t)
	defer db.Close()

	repo := NewParseJobRepository(db)
	err := repo.UpdateStatus(context.Background(), "nonexistent", models.ParseJobCompleted, "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestParseJobRepository_Update(t *testing.T) {
	db := setupParseJobTestDB(t)
	defer db.Close()

	repo := NewParseJobRepository(db)
	ctx := context.Background()

	job := createTestParseJob("job-1", "hash1", "/downloads/movie.mkv", "movie.mkv")
	err := repo.Create(ctx, job)
	require.NoError(t, err)

	// Update the job
	job.Status = models.ParseJobCompleted
	mediaID := "media-123"
	job.MediaID = &mediaID
	job.RetryCount = 2

	err = repo.Update(ctx, job)
	require.NoError(t, err)

	found, err := repo.GetByID(ctx, "job-1")
	require.NoError(t, err)
	assert.Equal(t, models.ParseJobCompleted, found.Status)
	require.NotNil(t, found.MediaID)
	assert.Equal(t, "media-123", *found.MediaID)
	assert.Equal(t, 2, found.RetryCount)
}

func TestParseJobRepository_Delete(t *testing.T) {
	db := setupParseJobTestDB(t)
	defer db.Close()

	repo := NewParseJobRepository(db)
	ctx := context.Background()

	job := createTestParseJob("job-1", "hash1", "/downloads/movie.mkv", "movie.mkv")
	err := repo.Create(ctx, job)
	require.NoError(t, err)

	err = repo.Delete(ctx, "job-1")
	require.NoError(t, err)

	_, err = repo.GetByID(ctx, "job-1")
	assert.Error(t, err)
}

func TestParseJobRepository_Delete_NotFound(t *testing.T) {
	db := setupParseJobTestDB(t)
	defer db.Close()

	repo := NewParseJobRepository(db)
	err := repo.Delete(context.Background(), "nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}
