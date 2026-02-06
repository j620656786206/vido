package retry

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

// TaskExecutor defines the interface for executing retry tasks
type TaskExecutor interface {
	// Execute runs the task and returns an error if it fails
	Execute(ctx context.Context, item *RetryItem) error
}

// SchedulerConfig contains configuration for the RetryScheduler
type SchedulerConfig struct {
	// TickInterval is how often to check for pending retries
	TickInterval time.Duration
	// MaxConcurrent is the maximum number of concurrent retry executions
	MaxConcurrent int
}

// DefaultSchedulerConfig returns the default scheduler configuration
func DefaultSchedulerConfig() *SchedulerConfig {
	return &SchedulerConfig{
		TickInterval:  1 * time.Second,
		MaxConcurrent: 5,
	}
}

// RepositoryInterface defines the repository methods needed by the scheduler
type RepositoryInterface interface {
	GetPending(ctx context.Context, now time.Time) ([]*RetryItem, error)
	Update(ctx context.Context, item *RetryItem) error
	Delete(ctx context.Context, id string) error
}

// EventType represents the type of retry event
type EventType string

const (
	EventRetryScheduled EventType = "retry_scheduled"
	EventRetryStarted   EventType = "retry_started"
	EventRetrySuccess   EventType = "retry_success"
	EventRetryFailed    EventType = "retry_failed"
	EventRetryExhausted EventType = "retry_exhausted"
)

// Event represents a retry lifecycle event
type Event struct {
	Type     EventType
	Item     *RetryItem
	Error    error
	Metadata map[string]interface{}
}

// EventHandler is called when retry events occur
type EventHandler func(event Event)

// RetryScheduler manages the retry background process
type RetryScheduler struct {
	repo         RepositoryInterface
	backoff      *BackoffCalculator
	executor     TaskExecutor
	logger       *slog.Logger
	config       *SchedulerConfig
	stopCh       chan struct{}
	wg           sync.WaitGroup
	running      bool
	mu           sync.RWMutex
	eventHandler EventHandler
	semaphore    chan struct{}
}

// NewRetryScheduler creates a new RetryScheduler instance
func NewRetryScheduler(
	repo RepositoryInterface,
	executor TaskExecutor,
	logger *slog.Logger,
	config *SchedulerConfig,
) *RetryScheduler {
	if config == nil {
		config = DefaultSchedulerConfig()
	}
	if logger == nil {
		logger = slog.Default()
	}

	return &RetryScheduler{
		repo:      repo,
		backoff:   NewBackoffCalculator(),
		executor:  executor,
		logger:    logger,
		config:    config,
		stopCh:    make(chan struct{}),
		semaphore: make(chan struct{}, config.MaxConcurrent),
	}
}

// SetEventHandler sets the event handler for retry lifecycle events
func (s *RetryScheduler) SetEventHandler(handler EventHandler) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.eventHandler = handler
}

// Start begins the scheduler's background process
func (s *RetryScheduler) Start(ctx context.Context) error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return nil
	}
	s.running = true
	s.stopCh = make(chan struct{})
	s.mu.Unlock()

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		s.run(ctx)
	}()

	s.logger.Info("Retry scheduler started",
		"tick_interval", s.config.TickInterval,
		"max_concurrent", s.config.MaxConcurrent,
	)

	return nil
}

// Stop gracefully stops the scheduler
func (s *RetryScheduler) Stop() {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return
	}
	s.running = false
	close(s.stopCh)
	s.mu.Unlock()

	s.wg.Wait()
	s.logger.Info("Retry scheduler stopped")
}

// IsRunning returns whether the scheduler is currently running
func (s *RetryScheduler) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.running
}

// run is the main scheduler loop
func (s *RetryScheduler) run(ctx context.Context) {
	ticker := time.NewTicker(s.config.TickInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopCh:
			return
		case <-ticker.C:
			s.processPendingRetries(ctx)
		}
	}
}

// processPendingRetries checks for and executes pending retries
func (s *RetryScheduler) processPendingRetries(ctx context.Context) {
	// Get items ready for retry (next_attempt_at <= now)
	items, err := s.repo.GetPending(ctx, time.Now())
	if err != nil {
		s.logger.Error("Failed to get pending retries", "error", err)
		return
	}

	if len(items) == 0 {
		return
	}

	s.logger.Debug("Found pending retries", "count", len(items))

	for _, item := range items {
		select {
		case <-ctx.Done():
			return
		case <-s.stopCh:
			return
		case s.semaphore <- struct{}{}: // Acquire semaphore
			go func(retryItem *RetryItem) {
				defer func() { <-s.semaphore }() // Release semaphore
				s.executeRetry(ctx, retryItem)
			}(item)
		}
	}
}

// executeRetry executes a single retry item
func (s *RetryScheduler) executeRetry(ctx context.Context, item *RetryItem) {
	s.emitEvent(Event{
		Type: EventRetryStarted,
		Item: item,
	})

	item.AttemptCount++
	item.UpdatedAt = time.Now()

	s.logger.Info("Executing retry",
		"task_id", item.TaskID,
		"task_type", item.TaskType,
		"attempt", item.AttemptCount,
		"max_attempts", item.MaxAttempts,
	)

	err := s.executor.Execute(ctx, item)
	if err != nil {
		s.handleRetryFailure(ctx, item, err)
	} else {
		s.handleRetrySuccess(ctx, item)
	}
}

// handleRetryFailure handles a failed retry attempt
func (s *RetryScheduler) handleRetryFailure(ctx context.Context, item *RetryItem, err error) {
	item.LastError = err.Error()

	if item.AttemptCount >= item.MaxAttempts {
		// Max attempts reached - mark for manual intervention
		s.logger.Warn("Retry exhausted, marking for manual intervention",
			"task_id", item.TaskID,
			"task_type", item.TaskType,
			"attempts", item.AttemptCount,
			"last_error", item.LastError,
		)

		// Delete from queue since max retries reached
		if delErr := s.repo.Delete(ctx, item.ID); delErr != nil {
			s.logger.Error("Failed to delete exhausted retry item",
				"id", item.ID,
				"error", delErr,
			)
		}

		s.emitEvent(Event{
			Type:  EventRetryExhausted,
			Item:  item,
			Error: err,
		})
		return
	}

	// Schedule next retry
	delay := s.backoff.Calculate(item.AttemptCount)
	item.NextAttemptAt = time.Now().Add(delay)

	if updateErr := s.repo.Update(ctx, item); updateErr != nil {
		s.logger.Error("Failed to update retry item",
			"id", item.ID,
			"error", updateErr,
		)
		return
	}

	s.logger.Info("Retry scheduled",
		"task_id", item.TaskID,
		"attempt", item.AttemptCount,
		"next_attempt", item.NextAttemptAt,
		"delay", delay,
	)

	s.emitEvent(Event{
		Type:  EventRetryFailed,
		Item:  item,
		Error: err,
		Metadata: map[string]interface{}{
			"next_attempt": item.NextAttemptAt,
			"delay":        delay,
		},
	})
}

// handleRetrySuccess handles a successful retry
func (s *RetryScheduler) handleRetrySuccess(ctx context.Context, item *RetryItem) {
	s.logger.Info("Retry succeeded",
		"task_id", item.TaskID,
		"task_type", item.TaskType,
		"attempts", item.AttemptCount,
	)

	// Delete from queue on success
	if err := s.repo.Delete(ctx, item.ID); err != nil {
		s.logger.Error("Failed to delete successful retry item",
			"id", item.ID,
			"error", err,
		)
	}

	s.emitEvent(Event{
		Type: EventRetrySuccess,
		Item: item,
	})
}

// emitEvent sends an event to the registered handler
func (s *RetryScheduler) emitEvent(event Event) {
	s.mu.RLock()
	handler := s.eventHandler
	s.mu.RUnlock()

	if handler != nil {
		handler(event)
	}
}

// TriggerImmediate immediately triggers a retry for the given item
func (s *RetryScheduler) TriggerImmediate(ctx context.Context, item *RetryItem) {
	if item == nil {
		return
	}

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		s.executeRetry(ctx, item)
	}()
}
