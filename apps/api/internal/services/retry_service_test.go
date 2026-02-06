package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vido/api/internal/retry"
	_ "modernc.org/sqlite"
)

// MockRetryRepository is a mock implementation for testing
type MockRetryRepository struct {
	items map[string]*retry.RetryItem
}

func NewMockRetryRepository() *MockRetryRepository {
	return &MockRetryRepository{
		items: make(map[string]*retry.RetryItem),
	}
}

func (m *MockRetryRepository) Add(ctx context.Context, item *retry.RetryItem) error {
	m.items[item.ID] = item
	return nil
}

func (m *MockRetryRepository) FindByID(ctx context.Context, id string) (*retry.RetryItem, error) {
	return m.items[id], nil
}

func (m *MockRetryRepository) FindByTaskID(ctx context.Context, taskID string) (*retry.RetryItem, error) {
	for _, item := range m.items {
		if item.TaskID == taskID {
			return item, nil
		}
	}
	return nil, nil
}

func (m *MockRetryRepository) GetPending(ctx context.Context, now time.Time) ([]*retry.RetryItem, error) {
	var pending []*retry.RetryItem
	for _, item := range m.items {
		if !item.NextAttemptAt.After(now) {
			pending = append(pending, item)
		}
	}
	return pending, nil
}

func (m *MockRetryRepository) GetAll(ctx context.Context) ([]*retry.RetryItem, error) {
	var all []*retry.RetryItem
	for _, item := range m.items {
		all = append(all, item)
	}
	return all, nil
}

func (m *MockRetryRepository) Update(ctx context.Context, item *retry.RetryItem) error {
	if _, exists := m.items[item.ID]; !exists {
		return errors.New("not found")
	}
	m.items[item.ID] = item
	return nil
}

func (m *MockRetryRepository) Delete(ctx context.Context, id string) error {
	if _, exists := m.items[id]; !exists {
		return errors.New("not found")
	}
	delete(m.items, id)
	return nil
}

func (m *MockRetryRepository) DeleteByTaskID(ctx context.Context, taskID string) error {
	for id, item := range m.items {
		if item.TaskID == taskID {
			delete(m.items, id)
			return nil
		}
	}
	return nil
}

func (m *MockRetryRepository) Count(ctx context.Context) (int, error) {
	return len(m.items), nil
}

func (m *MockRetryRepository) CountByTaskType(ctx context.Context, taskType string) (int, error) {
	count := 0
	for _, item := range m.items {
		if item.TaskType == taskType {
			count++
		}
	}
	return count, nil
}

func (m *MockRetryRepository) ClearAll(ctx context.Context) error {
	m.items = make(map[string]*retry.RetryItem)
	return nil
}

// Stats methods for Story 3.11
func (m *MockRetryRepository) IncrementQueued(ctx context.Context, taskType string) error {
	return nil
}

func (m *MockRetryRepository) IncrementSucceeded(ctx context.Context, taskType string) error {
	return nil
}

func (m *MockRetryRepository) IncrementFailed(ctx context.Context, taskType string) error {
	return nil
}

func (m *MockRetryRepository) IncrementExhausted(ctx context.Context, taskType string) error {
	return nil
}

func (m *MockRetryRepository) GetStats(ctx context.Context) (*retry.RetryStats, error) {
	return &retry.RetryStats{
		TotalPending:   len(m.items),
		TotalSucceeded: 0,
		TotalFailed:    0,
	}, nil
}

// MockTaskExecutor is a mock executor for testing
type MockTaskExecutor struct {
	shouldFail bool
}

func (m *MockTaskExecutor) Execute(ctx context.Context, item *retry.RetryItem) error {
	if m.shouldFail {
		return errors.New("execution failed")
	}
	return nil
}

func TestNewRetryService(t *testing.T) {
	repo := NewMockRetryRepository()
	executor := &MockTaskExecutor{}

	service := NewRetryService(repo, executor, nil)

	assert.NotNil(t, service)
	assert.NotNil(t, service.repo)
	assert.NotNil(t, service.scheduler)
	assert.NotNil(t, service.backoff)
	assert.NotNil(t, service.logger)
}

func TestRetryService_QueueRetry(t *testing.T) {
	repo := NewMockRetryRepository()
	executor := &MockTaskExecutor{}
	service := NewRetryService(repo, executor, nil)
	ctx := context.Background()

	// Queue with retryable error
	err := service.QueueRetry(ctx, "task-1", retry.TaskTypeParse, map[string]string{"file": "test.mkv"}, retry.ErrTimeout)
	require.NoError(t, err)

	// Verify item was added
	item, err := repo.FindByTaskID(ctx, "task-1")
	require.NoError(t, err)
	assert.NotNil(t, item)
	assert.Equal(t, "task-1", item.TaskID)
	assert.Equal(t, retry.TaskTypeParse, item.TaskType)
	assert.Equal(t, 0, item.AttemptCount)
	assert.Equal(t, 4, item.MaxAttempts)
}

func TestRetryService_QueueRetry_NonRetryableError(t *testing.T) {
	repo := NewMockRetryRepository()
	executor := &MockTaskExecutor{}
	service := NewRetryService(repo, executor, nil)
	ctx := context.Background()

	// Queue with non-retryable error
	err := service.QueueRetry(ctx, "task-1", retry.TaskTypeParse, nil, retry.ErrNotFound)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not retryable")
}

func TestRetryService_QueueRetry_AlreadyQueued(t *testing.T) {
	repo := NewMockRetryRepository()
	executor := &MockTaskExecutor{}
	service := NewRetryService(repo, executor, nil)
	ctx := context.Background()

	// Queue first time
	err := service.QueueRetry(ctx, "task-1", retry.TaskTypeParse, nil, retry.ErrTimeout)
	require.NoError(t, err)

	// Queue same task again - should not error
	err = service.QueueRetry(ctx, "task-1", retry.TaskTypeParse, nil, retry.ErrTimeout)
	require.NoError(t, err)

	// Only one item should exist
	count, _ := repo.Count(ctx)
	assert.Equal(t, 1, count)
}

func TestRetryService_CancelRetry(t *testing.T) {
	repo := NewMockRetryRepository()
	executor := &MockTaskExecutor{}
	service := NewRetryService(repo, executor, nil)
	ctx := context.Background()

	// Queue an item
	err := service.QueueRetry(ctx, "task-1", retry.TaskTypeParse, nil, retry.ErrTimeout)
	require.NoError(t, err)

	// Get the item ID
	items, _ := repo.GetAll(ctx)
	require.Len(t, items, 1)
	itemID := items[0].ID

	// Cancel it
	err = service.CancelRetry(ctx, itemID)
	require.NoError(t, err)

	// Verify it's gone
	count, _ := repo.Count(ctx)
	assert.Equal(t, 0, count)
}

func TestRetryService_CancelRetry_NotFound(t *testing.T) {
	repo := NewMockRetryRepository()
	executor := &MockTaskExecutor{}
	service := NewRetryService(repo, executor, nil)
	ctx := context.Background()

	err := service.CancelRetry(ctx, "nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestRetryService_CancelRetry_EmptyID(t *testing.T) {
	repo := NewMockRetryRepository()
	executor := &MockTaskExecutor{}
	service := NewRetryService(repo, executor, nil)
	ctx := context.Background()

	err := service.CancelRetry(ctx, "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot be empty")
}

func TestRetryService_TriggerImmediate(t *testing.T) {
	repo := NewMockRetryRepository()
	executor := &MockTaskExecutor{}
	service := NewRetryService(repo, executor, nil)
	ctx := context.Background()

	// Queue an item with future next_attempt_at
	err := service.QueueRetry(ctx, "task-1", retry.TaskTypeParse, nil, retry.ErrTimeout)
	require.NoError(t, err)

	items, _ := repo.GetAll(ctx)
	require.Len(t, items, 1)
	itemID := items[0].ID
	originalNextAttempt := items[0].NextAttemptAt

	// Trigger immediate
	err = service.TriggerImmediate(ctx, itemID)
	require.NoError(t, err)

	// Verify next_attempt_at was updated to now
	updated, _ := repo.FindByID(ctx, itemID)
	assert.True(t, updated.NextAttemptAt.Before(originalNextAttempt))
}

func TestRetryService_TriggerImmediate_NotFound(t *testing.T) {
	repo := NewMockRetryRepository()
	executor := &MockTaskExecutor{}
	service := NewRetryService(repo, executor, nil)
	ctx := context.Background()

	err := service.TriggerImmediate(ctx, "nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestRetryService_GetPendingRetries(t *testing.T) {
	repo := NewMockRetryRepository()
	executor := &MockTaskExecutor{}
	service := NewRetryService(repo, executor, nil)
	ctx := context.Background()

	// Queue multiple items
	for i := 1; i <= 3; i++ {
		err := service.QueueRetry(ctx, "task-"+string(rune('0'+i)), retry.TaskTypeParse, nil, retry.ErrTimeout)
		require.NoError(t, err)
	}

	// Get pending
	pending, err := service.GetPendingRetries(ctx)
	require.NoError(t, err)
	assert.Len(t, pending, 3)
}

func TestRetryService_GetRetryStats(t *testing.T) {
	repo := NewMockRetryRepository()
	executor := &MockTaskExecutor{}
	service := NewRetryService(repo, executor, nil)
	ctx := context.Background()

	// Queue some items
	for i := 1; i <= 5; i++ {
		err := service.QueueRetry(ctx, "task-"+string(rune('0'+i)), retry.TaskTypeParse, nil, retry.ErrTimeout)
		require.NoError(t, err)
	}

	// Get stats
	stats, err := service.GetRetryStats(ctx)
	require.NoError(t, err)
	assert.Equal(t, 5, stats.TotalPending)
}

func TestRetryService_IsRetryableError(t *testing.T) {
	repo := NewMockRetryRepository()
	executor := &MockTaskExecutor{}
	service := NewRetryService(repo, executor, nil)

	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "RetryableError - timeout",
			err:      retry.ErrTimeout,
			expected: true,
		},
		{
			name:     "RetryableError - rate limit",
			err:      retry.ErrRateLimit,
			expected: true,
		},
		{
			name:     "RetryableError - not found",
			err:      retry.ErrNotFound,
			expected: false,
		},
		{
			name:     "Generic error with timeout",
			err:      errors.New("request timeout after 30s"),
			expected: true,
		},
		{
			name:     "Generic error with connection",
			err:      errors.New("connection refused"),
			expected: true,
		},
		{
			name:     "Generic error with 503",
			err:      errors.New("HTTP 503 Service Unavailable"),
			expected: true,
		},
		{
			name:     "Generic error with 429",
			err:      errors.New("HTTP 429 Too Many Requests"),
			expected: true,
		},
		{
			name:     "Generic error not retryable",
			err:      errors.New("invalid input format"),
			expected: false,
		},
		{
			name:     "Context deadline exceeded",
			err:      errors.New("context deadline exceeded"),
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.IsRetryableError(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRetryService_GetByID(t *testing.T) {
	repo := NewMockRetryRepository()
	executor := &MockTaskExecutor{}
	service := NewRetryService(repo, executor, nil)
	ctx := context.Background()

	// Queue an item
	err := service.QueueRetry(ctx, "task-1", retry.TaskTypeParse, nil, retry.ErrTimeout)
	require.NoError(t, err)

	items, _ := repo.GetAll(ctx)
	require.Len(t, items, 1)

	// Get by ID
	item, err := service.GetByID(ctx, items[0].ID)
	require.NoError(t, err)
	assert.NotNil(t, item)
	assert.Equal(t, "task-1", item.TaskID)
}

func TestRetryService_GetByID_EmptyID(t *testing.T) {
	repo := NewMockRetryRepository()
	executor := &MockTaskExecutor{}
	service := NewRetryService(repo, executor, nil)
	ctx := context.Background()

	_, err := service.GetByID(ctx, "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot be empty")
}

func TestRetryService_GetByTaskID(t *testing.T) {
	repo := NewMockRetryRepository()
	executor := &MockTaskExecutor{}
	service := NewRetryService(repo, executor, nil)
	ctx := context.Background()

	// Queue an item
	err := service.QueueRetry(ctx, "task-123", retry.TaskTypeParse, nil, retry.ErrTimeout)
	require.NoError(t, err)

	// Get by task ID
	item, err := service.GetByTaskID(ctx, "task-123")
	require.NoError(t, err)
	assert.NotNil(t, item)
	assert.Equal(t, "task-123", item.TaskID)
}

func TestRetryService_CancelByTaskID(t *testing.T) {
	repo := NewMockRetryRepository()
	executor := &MockTaskExecutor{}
	service := NewRetryService(repo, executor, nil)
	ctx := context.Background()

	// Queue an item
	err := service.QueueRetry(ctx, "task-123", retry.TaskTypeParse, nil, retry.ErrTimeout)
	require.NoError(t, err)

	// Cancel by task ID
	err = service.CancelByTaskID(ctx, "task-123")
	require.NoError(t, err)

	// Verify it's gone
	item, _ := service.GetByTaskID(ctx, "task-123")
	assert.Nil(t, item)
}

func TestRetryService_ClearAll(t *testing.T) {
	repo := NewMockRetryRepository()
	executor := &MockTaskExecutor{}
	service := NewRetryService(repo, executor, nil)
	ctx := context.Background()

	// Queue multiple items
	for i := 1; i <= 5; i++ {
		err := service.QueueRetry(ctx, "task-"+string(rune('0'+i)), retry.TaskTypeParse, nil, retry.ErrTimeout)
		require.NoError(t, err)
	}

	// Clear all
	err := service.ClearAll(ctx)
	require.NoError(t, err)

	// Verify all cleared
	count, _ := repo.Count(ctx)
	assert.Equal(t, 0, count)
}

func TestRetryService_Payload(t *testing.T) {
	repo := NewMockRetryRepository()
	executor := &MockTaskExecutor{}
	service := NewRetryService(repo, executor, nil)
	ctx := context.Background()

	// Queue with payload
	payload := map[string]interface{}{
		"filename": "test.mkv",
		"mediaId":  "media-123",
	}
	err := service.QueueRetry(ctx, "task-1", retry.TaskTypeParse, payload, retry.ErrTimeout)
	require.NoError(t, err)

	// Get and verify payload
	items, _ := repo.GetAll(ctx)
	require.Len(t, items, 1)

	var retrievedPayload map[string]interface{}
	err = json.Unmarshal(items[0].Payload, &retrievedPayload)
	require.NoError(t, err)

	assert.Equal(t, "test.mkv", retrievedPayload["filename"])
	assert.Equal(t, "media-123", retrievedPayload["mediaId"])
}

func TestRetryService_StartStopScheduler(t *testing.T) {
	repo := NewMockRetryRepository()
	executor := &MockTaskExecutor{}
	service := NewRetryService(repo, executor, nil)
	ctx := context.Background()

	// Start scheduler
	err := service.StartScheduler(ctx)
	require.NoError(t, err)

	// Stop scheduler
	service.StopScheduler()
}

// Integration test with real database
func TestRetryService_Integration(t *testing.T) {
	// Skip in short mode
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	defer db.Close()

	// Create table
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
	`)
	require.NoError(t, err)

	// This would use actual repository in a full integration test
	// For now, we just verify the test setup works
}
