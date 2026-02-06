package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vido/api/internal/retry"
	_ "modernc.org/sqlite"
)

func setupRetryTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)

	// Create retry_queue table
	_, err = db.Exec(`
		CREATE TABLE retry_queue (
			id TEXT PRIMARY KEY,
			task_id TEXT NOT NULL,
			task_type TEXT NOT NULL,
			payload TEXT NOT NULL,
			attempt_count INTEGER DEFAULT 0,
			max_attempts INTEGER DEFAULT 4,
			last_error TEXT,
			next_attempt_at TIMESTAMP NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			CONSTRAINT unique_task_id UNIQUE (task_id)
		);
		CREATE INDEX idx_retry_queue_next_attempt ON retry_queue(next_attempt_at);
		CREATE INDEX idx_retry_queue_task_type ON retry_queue(task_type);
	`)
	require.NoError(t, err)

	return db
}

func createTestRetryItem(id, taskID, taskType string) *retry.RetryItem {
	now := time.Now().Truncate(time.Second)
	payload, _ := json.Marshal(map[string]string{"filename": "test.mkv"})

	return &retry.RetryItem{
		ID:            id,
		TaskID:        taskID,
		TaskType:      taskType,
		Payload:       payload,
		AttemptCount:  0,
		MaxAttempts:   4,
		LastError:     "",
		NextAttemptAt: now.Add(1 * time.Second),
		CreatedAt:     now,
		UpdatedAt:     now,
	}
}

func TestRetryRepository_Add(t *testing.T) {
	db := setupRetryTestDB(t)
	defer db.Close()

	repo := NewRetryRepository(db)
	ctx := context.Background()

	item := createTestRetryItem("retry-1", "task-1", retry.TaskTypeParse)
	err := repo.Add(ctx, item)
	require.NoError(t, err)

	// Verify it was added
	found, err := repo.FindByID(ctx, "retry-1")
	require.NoError(t, err)
	assert.NotNil(t, found)
	assert.Equal(t, "retry-1", found.ID)
	assert.Equal(t, "task-1", found.TaskID)
	assert.Equal(t, retry.TaskTypeParse, found.TaskType)
}

func TestRetryRepository_Add_NilItem(t *testing.T) {
	db := setupRetryTestDB(t)
	defer db.Close()

	repo := NewRetryRepository(db)
	ctx := context.Background()

	err := repo.Add(ctx, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot be nil")
}

func TestRetryRepository_Add_DuplicateTaskID(t *testing.T) {
	db := setupRetryTestDB(t)
	defer db.Close()

	repo := NewRetryRepository(db)
	ctx := context.Background()

	item1 := createTestRetryItem("retry-1", "task-1", retry.TaskTypeParse)
	err := repo.Add(ctx, item1)
	require.NoError(t, err)

	// Try to add another item with the same task_id
	item2 := createTestRetryItem("retry-2", "task-1", retry.TaskTypeParse)
	err = repo.Add(ctx, item2)
	assert.Error(t, err) // Should fail due to unique constraint
}

func TestRetryRepository_FindByID(t *testing.T) {
	db := setupRetryTestDB(t)
	defer db.Close()

	repo := NewRetryRepository(db)
	ctx := context.Background()

	item := createTestRetryItem("retry-1", "task-1", retry.TaskTypeParse)
	item.LastError = "TMDb timeout"
	err := repo.Add(ctx, item)
	require.NoError(t, err)

	// Find by ID
	found, err := repo.FindByID(ctx, "retry-1")
	require.NoError(t, err)
	assert.NotNil(t, found)
	assert.Equal(t, "retry-1", found.ID)
	assert.Equal(t, "TMDb timeout", found.LastError)
}

func TestRetryRepository_FindByID_NotFound(t *testing.T) {
	db := setupRetryTestDB(t)
	defer db.Close()

	repo := NewRetryRepository(db)
	ctx := context.Background()

	found, err := repo.FindByID(ctx, "nonexistent")
	require.NoError(t, err)
	assert.Nil(t, found)
}

func TestRetryRepository_FindByTaskID(t *testing.T) {
	db := setupRetryTestDB(t)
	defer db.Close()

	repo := NewRetryRepository(db)
	ctx := context.Background()

	item := createTestRetryItem("retry-1", "task-123", retry.TaskTypeMetadataFetch)
	err := repo.Add(ctx, item)
	require.NoError(t, err)

	// Find by task ID
	found, err := repo.FindByTaskID(ctx, "task-123")
	require.NoError(t, err)
	assert.NotNil(t, found)
	assert.Equal(t, "task-123", found.TaskID)
	assert.Equal(t, retry.TaskTypeMetadataFetch, found.TaskType)
}

func TestRetryRepository_FindByTaskID_NotFound(t *testing.T) {
	db := setupRetryTestDB(t)
	defer db.Close()

	repo := NewRetryRepository(db)
	ctx := context.Background()

	found, err := repo.FindByTaskID(ctx, "nonexistent")
	require.NoError(t, err)
	assert.Nil(t, found)
}

func TestRetryRepository_GetPending(t *testing.T) {
	db := setupRetryTestDB(t)
	defer db.Close()

	repo := NewRetryRepository(db)
	ctx := context.Background()
	now := time.Now().Truncate(time.Second)

	// Add items with different next_attempt_at times
	item1 := createTestRetryItem("retry-1", "task-1", retry.TaskTypeParse)
	item1.NextAttemptAt = now.Add(-1 * time.Second) // Ready for retry
	err := repo.Add(ctx, item1)
	require.NoError(t, err)

	item2 := createTestRetryItem("retry-2", "task-2", retry.TaskTypeParse)
	item2.NextAttemptAt = now.Add(-2 * time.Second) // Ready for retry
	err = repo.Add(ctx, item2)
	require.NoError(t, err)

	item3 := createTestRetryItem("retry-3", "task-3", retry.TaskTypeParse)
	item3.NextAttemptAt = now.Add(10 * time.Second) // Not ready yet
	err = repo.Add(ctx, item3)
	require.NoError(t, err)

	// Get pending items
	pending, err := repo.GetPending(ctx, now)
	require.NoError(t, err)
	assert.Len(t, pending, 2)

	// Should be ordered by next_attempt_at
	assert.Equal(t, "retry-2", pending[0].ID) // Earlier time first
	assert.Equal(t, "retry-1", pending[1].ID)
}

func TestRetryRepository_GetAll(t *testing.T) {
	db := setupRetryTestDB(t)
	defer db.Close()

	repo := NewRetryRepository(db)
	ctx := context.Background()

	// Add multiple items
	for i := 1; i <= 3; i++ {
		item := createTestRetryItem(
			"retry-"+string(rune('0'+i)),
			"task-"+string(rune('0'+i)),
			retry.TaskTypeParse,
		)
		err := repo.Add(ctx, item)
		require.NoError(t, err)
	}

	// Get all items
	all, err := repo.GetAll(ctx)
	require.NoError(t, err)
	assert.Len(t, all, 3)
}

func TestRetryRepository_Update(t *testing.T) {
	db := setupRetryTestDB(t)
	defer db.Close()

	repo := NewRetryRepository(db)
	ctx := context.Background()
	now := time.Now().Truncate(time.Second)

	item := createTestRetryItem("retry-1", "task-1", retry.TaskTypeParse)
	err := repo.Add(ctx, item)
	require.NoError(t, err)

	// Update the item
	item.AttemptCount = 2
	item.LastError = "Rate limit exceeded"
	item.NextAttemptAt = now.Add(4 * time.Second)

	err = repo.Update(ctx, item)
	require.NoError(t, err)

	// Verify update
	found, err := repo.FindByID(ctx, "retry-1")
	require.NoError(t, err)
	assert.Equal(t, 2, found.AttemptCount)
	assert.Equal(t, "Rate limit exceeded", found.LastError)
}

func TestRetryRepository_Update_NotFound(t *testing.T) {
	db := setupRetryTestDB(t)
	defer db.Close()

	repo := NewRetryRepository(db)
	ctx := context.Background()

	item := createTestRetryItem("retry-nonexistent", "task-1", retry.TaskTypeParse)
	err := repo.Update(ctx, item)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestRetryRepository_Update_NilItem(t *testing.T) {
	db := setupRetryTestDB(t)
	defer db.Close()

	repo := NewRetryRepository(db)
	ctx := context.Background()

	err := repo.Update(ctx, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot be nil")
}

func TestRetryRepository_Delete(t *testing.T) {
	db := setupRetryTestDB(t)
	defer db.Close()

	repo := NewRetryRepository(db)
	ctx := context.Background()

	item := createTestRetryItem("retry-1", "task-1", retry.TaskTypeParse)
	err := repo.Add(ctx, item)
	require.NoError(t, err)

	// Delete the item
	err = repo.Delete(ctx, "retry-1")
	require.NoError(t, err)

	// Verify deletion
	found, err := repo.FindByID(ctx, "retry-1")
	require.NoError(t, err)
	assert.Nil(t, found)
}

func TestRetryRepository_Delete_NotFound(t *testing.T) {
	db := setupRetryTestDB(t)
	defer db.Close()

	repo := NewRetryRepository(db)
	ctx := context.Background()

	err := repo.Delete(ctx, "nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestRetryRepository_DeleteByTaskID(t *testing.T) {
	db := setupRetryTestDB(t)
	defer db.Close()

	repo := NewRetryRepository(db)
	ctx := context.Background()

	item := createTestRetryItem("retry-1", "task-123", retry.TaskTypeParse)
	err := repo.Add(ctx, item)
	require.NoError(t, err)

	// Delete by task ID
	err = repo.DeleteByTaskID(ctx, "task-123")
	require.NoError(t, err)

	// Verify deletion
	found, err := repo.FindByTaskID(ctx, "task-123")
	require.NoError(t, err)
	assert.Nil(t, found)
}

func TestRetryRepository_Count(t *testing.T) {
	db := setupRetryTestDB(t)
	defer db.Close()

	repo := NewRetryRepository(db)
	ctx := context.Background()

	// Initially empty
	count, err := repo.Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, 0, count)

	// Add items
	for i := 1; i <= 5; i++ {
		item := createTestRetryItem(
			"retry-"+string(rune('0'+i)),
			"task-"+string(rune('0'+i)),
			retry.TaskTypeParse,
		)
		err := repo.Add(ctx, item)
		require.NoError(t, err)
	}

	count, err = repo.Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, 5, count)
}

func TestRetryRepository_CountByTaskType(t *testing.T) {
	db := setupRetryTestDB(t)
	defer db.Close()

	repo := NewRetryRepository(db)
	ctx := context.Background()

	// Add items with different types
	item1 := createTestRetryItem("retry-1", "task-1", retry.TaskTypeParse)
	err := repo.Add(ctx, item1)
	require.NoError(t, err)

	item2 := createTestRetryItem("retry-2", "task-2", retry.TaskTypeParse)
	err = repo.Add(ctx, item2)
	require.NoError(t, err)

	item3 := createTestRetryItem("retry-3", "task-3", retry.TaskTypeMetadataFetch)
	err = repo.Add(ctx, item3)
	require.NoError(t, err)

	// Count by type
	parseCount, err := repo.CountByTaskType(ctx, retry.TaskTypeParse)
	require.NoError(t, err)
	assert.Equal(t, 2, parseCount)

	fetchCount, err := repo.CountByTaskType(ctx, retry.TaskTypeMetadataFetch)
	require.NoError(t, err)
	assert.Equal(t, 1, fetchCount)
}

func TestRetryRepository_ClearAll(t *testing.T) {
	db := setupRetryTestDB(t)
	defer db.Close()

	repo := NewRetryRepository(db)
	ctx := context.Background()

	// Add items
	for i := 1; i <= 3; i++ {
		item := createTestRetryItem(
			"retry-"+string(rune('0'+i)),
			"task-"+string(rune('0'+i)),
			retry.TaskTypeParse,
		)
		err := repo.Add(ctx, item)
		require.NoError(t, err)
	}

	// Clear all
	err := repo.ClearAll(ctx)
	require.NoError(t, err)

	// Verify all cleared
	count, err := repo.Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, 0, count)
}

func TestRetryRepository_PayloadPersistence(t *testing.T) {
	db := setupRetryTestDB(t)
	defer db.Close()

	repo := NewRetryRepository(db)
	ctx := context.Background()

	// Create item with complex payload
	payload := map[string]interface{}{
		"filename": "test.mkv",
		"mediaId":  "media-123",
		"attempt":  1,
	}
	payloadJSON, _ := json.Marshal(payload)

	item := createTestRetryItem("retry-1", "task-1", retry.TaskTypeParse)
	item.Payload = payloadJSON
	err := repo.Add(ctx, item)
	require.NoError(t, err)

	// Retrieve and verify payload
	found, err := repo.FindByID(ctx, "retry-1")
	require.NoError(t, err)

	var retrievedPayload map[string]interface{}
	err = json.Unmarshal(found.Payload, &retrievedPayload)
	require.NoError(t, err)

	assert.Equal(t, "test.mkv", retrievedPayload["filename"])
	assert.Equal(t, "media-123", retrievedPayload["mediaId"])
}
