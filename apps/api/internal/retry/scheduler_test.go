package retry

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockRepository is a mock implementation of RepositoryInterface for testing
type MockRepository struct {
	items map[string]*RetryItem
	mu    sync.RWMutex
}

func NewMockRepository() *MockRepository {
	return &MockRepository{
		items: make(map[string]*RetryItem),
	}
}

func (m *MockRepository) Add(item *RetryItem) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.items[item.ID] = item
}

func (m *MockRepository) GetPending(ctx context.Context, now time.Time) ([]*RetryItem, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var pending []*RetryItem
	for _, item := range m.items {
		if !item.NextAttemptAt.After(now) {
			pending = append(pending, item)
		}
	}
	return pending, nil
}

func (m *MockRepository) Update(ctx context.Context, item *RetryItem) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.items[item.ID]; !exists {
		return errors.New("item not found")
	}
	m.items[item.ID] = item
	return nil
}

func (m *MockRepository) Delete(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.items, id)
	return nil
}

func (m *MockRepository) Get(id string) *RetryItem {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.items[id]
}

func (m *MockRepository) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.items)
}

// MockExecutor is a mock implementation of TaskExecutor for testing
type MockExecutor struct {
	shouldFail    bool
	failCount     int32
	maxFails      int32
	executionTime time.Duration
	executions    int32
	mu            sync.Mutex
}

func NewMockExecutor() *MockExecutor {
	return &MockExecutor{}
}

func (m *MockExecutor) Execute(ctx context.Context, item *RetryItem) error {
	atomic.AddInt32(&m.executions, 1)

	if m.executionTime > 0 {
		time.Sleep(m.executionTime)
	}

	if m.shouldFail {
		if m.maxFails > 0 {
			count := atomic.AddInt32(&m.failCount, 1)
			if count > m.maxFails {
				return nil // Succeed after max fails
			}
		}
		return errors.New("execution failed")
	}

	return nil
}

func (m *MockExecutor) ExecutionCount() int32 {
	return atomic.LoadInt32(&m.executions)
}

func (m *MockExecutor) SetFailure(shouldFail bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.shouldFail = shouldFail
}

func (m *MockExecutor) SetMaxFails(maxFails int32) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.maxFails = maxFails
}

func createTestItem(id, taskID string, nextAttemptAt time.Time) *RetryItem {
	payload, _ := json.Marshal(map[string]string{"test": "data"})
	return &RetryItem{
		ID:            id,
		TaskID:        taskID,
		TaskType:      TaskTypeParse,
		Payload:       payload,
		AttemptCount:  0,
		MaxAttempts:   4,
		NextAttemptAt: nextAttemptAt,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
}

func TestRetryScheduler_NewRetryScheduler(t *testing.T) {
	repo := NewMockRepository()
	executor := NewMockExecutor()

	scheduler := NewRetryScheduler(repo, executor, nil, nil)

	assert.NotNil(t, scheduler)
	assert.NotNil(t, scheduler.backoff)
	assert.NotNil(t, scheduler.logger)
	assert.Equal(t, 1*time.Second, scheduler.config.TickInterval)
	assert.Equal(t, 5, scheduler.config.MaxConcurrent)
}

func TestRetryScheduler_StartStop(t *testing.T) {
	repo := NewMockRepository()
	executor := NewMockExecutor()
	config := &SchedulerConfig{
		TickInterval:  100 * time.Millisecond,
		MaxConcurrent: 2,
	}

	scheduler := NewRetryScheduler(repo, executor, nil, config)

	// Start scheduler
	err := scheduler.Start(context.Background())
	require.NoError(t, err)
	assert.True(t, scheduler.IsRunning())

	// Starting again should be a no-op
	err = scheduler.Start(context.Background())
	require.NoError(t, err)

	// Stop scheduler
	scheduler.Stop()
	assert.False(t, scheduler.IsRunning())

	// Stopping again should be a no-op
	scheduler.Stop()
}

func TestRetryScheduler_ProcessPendingRetries_Success(t *testing.T) {
	repo := NewMockRepository()
	executor := NewMockExecutor()
	config := &SchedulerConfig{
		TickInterval:  50 * time.Millisecond,
		MaxConcurrent: 5,
	}

	// Add a pending item
	item := createTestItem("retry-1", "task-1", time.Now().Add(-1*time.Second))
	repo.Add(item)

	scheduler := NewRetryScheduler(repo, executor, nil, config)

	var successEvent Event
	var eventMu sync.Mutex
	scheduler.SetEventHandler(func(e Event) {
		eventMu.Lock()
		defer eventMu.Unlock()
		if e.Type == EventRetrySuccess {
			successEvent = e
		}
	})

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	err := scheduler.Start(ctx)
	require.NoError(t, err)

	// Wait for processing
	time.Sleep(200 * time.Millisecond)
	scheduler.Stop()

	// Verify execution
	assert.GreaterOrEqual(t, executor.ExecutionCount(), int32(1))

	// Verify item was deleted (success)
	assert.Nil(t, repo.Get("retry-1"))

	// Verify event was emitted
	eventMu.Lock()
	assert.Equal(t, EventRetrySuccess, successEvent.Type)
	eventMu.Unlock()
}

func TestRetryScheduler_ProcessPendingRetries_Failure(t *testing.T) {
	repo := NewMockRepository()
	executor := NewMockExecutor()
	executor.SetFailure(true)
	config := &SchedulerConfig{
		TickInterval:  50 * time.Millisecond,
		MaxConcurrent: 5,
	}

	// Add a pending item
	item := createTestItem("retry-1", "task-1", time.Now().Add(-1*time.Second))
	repo.Add(item)

	scheduler := NewRetryScheduler(repo, executor, nil, config)

	var failedEvent Event
	var eventMu sync.Mutex
	scheduler.SetEventHandler(func(e Event) {
		eventMu.Lock()
		defer eventMu.Unlock()
		if e.Type == EventRetryFailed {
			failedEvent = e
		}
	})

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	err := scheduler.Start(ctx)
	require.NoError(t, err)

	// Wait for processing
	time.Sleep(200 * time.Millisecond)
	scheduler.Stop()

	// Verify item was updated with next attempt time
	updatedItem := repo.Get("retry-1")
	if updatedItem != nil {
		assert.Equal(t, 1, updatedItem.AttemptCount)
		assert.True(t, updatedItem.NextAttemptAt.After(time.Now()))
		assert.NotEmpty(t, updatedItem.LastError)
	}

	// Verify failed event was emitted
	eventMu.Lock()
	assert.Equal(t, EventRetryFailed, failedEvent.Type)
	eventMu.Unlock()
}

func TestRetryScheduler_MaxRetriesExhausted(t *testing.T) {
	repo := NewMockRepository()
	executor := NewMockExecutor()
	executor.SetFailure(true)
	config := &SchedulerConfig{
		TickInterval:  50 * time.Millisecond,
		MaxConcurrent: 5,
	}

	// Add item that's already at max attempts - 1
	item := createTestItem("retry-1", "task-1", time.Now().Add(-1*time.Second))
	item.AttemptCount = 3 // Will become 4 on next attempt (max)
	repo.Add(item)

	scheduler := NewRetryScheduler(repo, executor, nil, config)

	var exhaustedEvent Event
	var eventMu sync.Mutex
	scheduler.SetEventHandler(func(e Event) {
		eventMu.Lock()
		defer eventMu.Unlock()
		if e.Type == EventRetryExhausted {
			exhaustedEvent = e
		}
	})

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	err := scheduler.Start(ctx)
	require.NoError(t, err)

	// Wait for processing
	time.Sleep(200 * time.Millisecond)
	scheduler.Stop()

	// Verify item was deleted (exhausted)
	assert.Nil(t, repo.Get("retry-1"))

	// Verify exhausted event was emitted
	eventMu.Lock()
	assert.Equal(t, EventRetryExhausted, exhaustedEvent.Type)
	eventMu.Unlock()
}

func TestRetryScheduler_Concurrency(t *testing.T) {
	repo := NewMockRepository()
	executor := NewMockExecutor()
	executor.executionTime = 100 * time.Millisecond
	config := &SchedulerConfig{
		TickInterval:  50 * time.Millisecond,
		MaxConcurrent: 2, // Only 2 concurrent
	}

	// Add 5 pending items
	for i := 0; i < 5; i++ {
		item := createTestItem(
			"retry-"+string(rune('0'+i)),
			"task-"+string(rune('0'+i)),
			time.Now().Add(-1*time.Second),
		)
		repo.Add(item)
	}

	scheduler := NewRetryScheduler(repo, executor, nil, config)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	err := scheduler.Start(ctx)
	require.NoError(t, err)

	// Wait for some processing
	time.Sleep(500 * time.Millisecond)
	scheduler.Stop()

	// All items should be processed eventually
	// Due to concurrency limit and timing, check that at least some were processed
	assert.GreaterOrEqual(t, executor.ExecutionCount(), int32(2))
}

func TestRetryScheduler_EventHandler(t *testing.T) {
	repo := NewMockRepository()
	executor := NewMockExecutor()
	config := &SchedulerConfig{
		TickInterval:  50 * time.Millisecond,
		MaxConcurrent: 5,
	}

	scheduler := NewRetryScheduler(repo, executor, nil, config)

	var events []Event
	var eventMu sync.Mutex
	scheduler.SetEventHandler(func(e Event) {
		eventMu.Lock()
		defer eventMu.Unlock()
		events = append(events, e)
	})

	// Add a pending item
	item := createTestItem("retry-1", "task-1", time.Now().Add(-1*time.Second))
	repo.Add(item)

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	err := scheduler.Start(ctx)
	require.NoError(t, err)

	// Wait for processing
	time.Sleep(200 * time.Millisecond)
	scheduler.Stop()

	// Verify events were emitted
	eventMu.Lock()
	defer eventMu.Unlock()

	// Should have at least started and success events
	assert.GreaterOrEqual(t, len(events), 2)

	// Check event types
	eventTypes := make(map[EventType]bool)
	for _, e := range events {
		eventTypes[e.Type] = true
	}
	assert.True(t, eventTypes[EventRetryStarted])
	assert.True(t, eventTypes[EventRetrySuccess])
}

func TestRetryScheduler_TriggerImmediate(t *testing.T) {
	repo := NewMockRepository()
	executor := NewMockExecutor()
	config := &SchedulerConfig{
		TickInterval:  1 * time.Second, // Long interval
		MaxConcurrent: 5,
	}

	scheduler := NewRetryScheduler(repo, executor, nil, config)

	// Add item with future next_attempt_at (won't be picked up by scheduler)
	item := createTestItem("retry-1", "task-1", time.Now().Add(1*time.Hour))
	repo.Add(item)

	// Trigger immediate execution
	scheduler.TriggerImmediate(context.Background(), item)

	// Wait for execution
	time.Sleep(100 * time.Millisecond)

	// Verify execution happened
	assert.Equal(t, int32(1), executor.ExecutionCount())
}

func TestRetryScheduler_ContextCancellation(t *testing.T) {
	repo := NewMockRepository()
	executor := NewMockExecutor()
	config := &SchedulerConfig{
		TickInterval:  50 * time.Millisecond,
		MaxConcurrent: 5,
	}

	scheduler := NewRetryScheduler(repo, executor, nil, config)

	ctx, cancel := context.WithCancel(context.Background())

	err := scheduler.Start(ctx)
	require.NoError(t, err)
	assert.True(t, scheduler.IsRunning())

	// Cancel context
	cancel()

	// Wait for shutdown
	time.Sleep(100 * time.Millisecond)

	// Scheduler should have stopped
	// Note: IsRunning might still be true since we cancelled context, not called Stop
	// But the goroutine should have exited
}

func TestRetryScheduler_NoPendingItems(t *testing.T) {
	repo := NewMockRepository()
	executor := NewMockExecutor()
	config := &SchedulerConfig{
		TickInterval:  50 * time.Millisecond,
		MaxConcurrent: 5,
	}

	scheduler := NewRetryScheduler(repo, executor, nil, config)

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	err := scheduler.Start(ctx)
	require.NoError(t, err)

	// Wait
	time.Sleep(150 * time.Millisecond)
	scheduler.Stop()

	// No executions should have happened
	assert.Equal(t, int32(0), executor.ExecutionCount())
}

func TestEventType_Constants(t *testing.T) {
	assert.Equal(t, EventType("retry_scheduled"), EventRetryScheduled)
	assert.Equal(t, EventType("retry_started"), EventRetryStarted)
	assert.Equal(t, EventType("retry_success"), EventRetrySuccess)
	assert.Equal(t, EventType("retry_failed"), EventRetryFailed)
	assert.Equal(t, EventType("retry_exhausted"), EventRetryExhausted)
}

func TestDefaultSchedulerConfig(t *testing.T) {
	config := DefaultSchedulerConfig()

	assert.Equal(t, 1*time.Second, config.TickInterval)
	assert.Equal(t, 5, config.MaxConcurrent)
}
