package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/vido/api/internal/repository"
	"github.com/vido/api/internal/retry"
)

// RetryServiceInterface defines the contract for retry operations
type RetryServiceInterface interface {
	QueueRetry(ctx context.Context, taskID, taskType string, payload interface{}, err error) error
	CancelRetry(ctx context.Context, itemID string) error
	TriggerImmediate(ctx context.Context, itemID string) error
	GetPendingRetries(ctx context.Context) ([]*retry.RetryItem, error)
	GetRetryStats(ctx context.Context) (*retry.RetryStats, error)
	GetByID(ctx context.Context, id string) (*retry.RetryItem, error)
	IsRetryableError(err error) bool
	StartScheduler(ctx context.Context) error
	StopScheduler()
}

// RetryService manages retry operations
type RetryService struct {
	repo      repository.RetryRepositoryInterface
	scheduler *retry.RetryScheduler
	backoff   *retry.BackoffCalculator
	logger    *slog.Logger
	executor  retry.TaskExecutor
}

// NewRetryService creates a new RetryService instance
func NewRetryService(
	repo repository.RetryRepositoryInterface,
	executor retry.TaskExecutor,
	logger *slog.Logger,
) *RetryService {
	if logger == nil {
		logger = slog.Default()
	}

	backoff := retry.NewBackoffCalculator()

	// Create scheduler with the repository adapter
	repoAdapter := &repositoryAdapter{repo: repo}
	scheduler := retry.NewRetryScheduler(repoAdapter, executor, logger, nil)

	return &RetryService{
		repo:      repo,
		scheduler: scheduler,
		backoff:   backoff,
		logger:    logger,
		executor:  executor,
	}
}

// repositoryAdapter adapts RetryRepositoryInterface to retry.RepositoryInterface
type repositoryAdapter struct {
	repo repository.RetryRepositoryInterface
}

func (a *repositoryAdapter) GetPending(ctx context.Context, now time.Time) ([]*retry.RetryItem, error) {
	return a.repo.GetPending(ctx, now)
}

func (a *repositoryAdapter) Update(ctx context.Context, item *retry.RetryItem) error {
	return a.repo.Update(ctx, item)
}

func (a *repositoryAdapter) Delete(ctx context.Context, id string) error {
	return a.repo.Delete(ctx, id)
}

// SetEventHandler sets the event handler for retry lifecycle events
func (s *RetryService) SetEventHandler(handler retry.EventHandler) {
	s.scheduler.SetEventHandler(handler)
}

// StartScheduler starts the retry scheduler background process
func (s *RetryService) StartScheduler(ctx context.Context) error {
	return s.scheduler.Start(ctx)
}

// StopScheduler stops the retry scheduler
func (s *RetryService) StopScheduler() {
	s.scheduler.Stop()
}

// QueueRetry adds a task to the retry queue
func (s *RetryService) QueueRetry(ctx context.Context, taskID, taskType string, payload interface{}, origErr error) error {
	// Check if error is retryable
	if !s.IsRetryableError(origErr) {
		return fmt.Errorf("error is not retryable: %w", origErr)
	}

	// Check if already queued
	existing, err := s.repo.FindByTaskID(ctx, taskID)
	if err != nil {
		return fmt.Errorf("failed to check existing retry: %w", err)
	}
	if existing != nil {
		s.logger.Debug("Retry already queued", "task_id", taskID)
		return nil // Already queued
	}

	// Marshal payload
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	now := time.Now()
	item := &retry.RetryItem{
		ID:            uuid.New().String(),
		TaskID:        taskID,
		TaskType:      taskType,
		Payload:       payloadJSON,
		AttemptCount:  0,
		MaxAttempts:   retry.MaxRetryAttempts,
		LastError:     origErr.Error(),
		NextAttemptAt: now.Add(s.backoff.Calculate(0)),
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	if err := s.repo.Add(ctx, item); err != nil {
		return fmt.Errorf("failed to queue retry: %w", err)
	}

	s.logger.Info("Retry queued",
		"task_id", taskID,
		"task_type", taskType,
		"next_attempt", item.NextAttemptAt,
	)

	return nil
}

// CancelRetry removes a retry item from the queue
func (s *RetryService) CancelRetry(ctx context.Context, itemID string) error {
	if itemID == "" {
		return errors.New("item ID cannot be empty")
	}

	// Check if item exists
	item, err := s.repo.FindByID(ctx, itemID)
	if err != nil {
		return fmt.Errorf("failed to find retry item: %w", err)
	}
	if item == nil {
		return fmt.Errorf("retry item not found: %s", itemID)
	}

	if err := s.repo.Delete(ctx, itemID); err != nil {
		return fmt.Errorf("failed to cancel retry: %w", err)
	}

	s.logger.Info("Retry cancelled", "id", itemID, "task_id", item.TaskID)
	return nil
}

// TriggerImmediate triggers an immediate retry for the given item
func (s *RetryService) TriggerImmediate(ctx context.Context, itemID string) error {
	if itemID == "" {
		return errors.New("item ID cannot be empty")
	}

	item, err := s.repo.FindByID(ctx, itemID)
	if err != nil {
		return fmt.Errorf("failed to find retry item: %w", err)
	}
	if item == nil {
		return fmt.Errorf("retry item not found: %s", itemID)
	}

	// Update next attempt time to now
	item.NextAttemptAt = time.Now()
	if err := s.repo.Update(ctx, item); err != nil {
		return fmt.Errorf("failed to update retry: %w", err)
	}

	s.logger.Info("Immediate retry triggered", "id", itemID, "task_id", item.TaskID)

	// If scheduler is running, it will pick it up on next tick
	// Otherwise, trigger directly
	if !s.scheduler.IsRunning() {
		s.scheduler.TriggerImmediate(ctx, item)
	}

	return nil
}

// GetPendingRetries returns all pending retry items
func (s *RetryService) GetPendingRetries(ctx context.Context) ([]*retry.RetryItem, error) {
	return s.repo.GetAll(ctx)
}

// GetRetryStats returns statistics about the retry queue
func (s *RetryService) GetRetryStats(ctx context.Context) (*retry.RetryStats, error) {
	count, err := s.repo.Count(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get retry count: %w", err)
	}

	return &retry.RetryStats{
		TotalPending:   count,
		TotalSucceeded: 0, // Would need separate tracking for historical data
		TotalFailed:    0,
	}, nil
}

// IsRetryableError checks if an error should trigger a retry
func (s *RetryService) IsRetryableError(err error) bool {
	if err == nil {
		return false
	}

	// Check for RetryableError type
	var retryErr *retry.RetryableError
	if errors.As(err, &retryErr) {
		return retryErr.Retryable
	}

	// Check common error patterns
	errStr := strings.ToLower(err.Error())
	retryablePatterns := []string{
		"timeout",
		"rate limit",
		"network",
		"connection",
		"temporary",
		"unavailable",
		"503",
		"502",
		"504",
		"429",
		"service unavailable",
		"bad gateway",
		"gateway timeout",
		"too many requests",
		"connection refused",
		"connection reset",
		"dns lookup",
		"no such host",
		"i/o timeout",
		"context deadline exceeded",
	}

	for _, pattern := range retryablePatterns {
		if strings.Contains(errStr, pattern) {
			return true
		}
	}

	return false
}

// GetByID retrieves a retry item by ID
func (s *RetryService) GetByID(ctx context.Context, id string) (*retry.RetryItem, error) {
	if id == "" {
		return nil, errors.New("item ID cannot be empty")
	}
	return s.repo.FindByID(ctx, id)
}

// GetByTaskID retrieves a retry item by task ID
func (s *RetryService) GetByTaskID(ctx context.Context, taskID string) (*retry.RetryItem, error) {
	if taskID == "" {
		return nil, errors.New("task ID cannot be empty")
	}
	return s.repo.FindByTaskID(ctx, taskID)
}

// CancelByTaskID removes a retry item by its task ID
func (s *RetryService) CancelByTaskID(ctx context.Context, taskID string) error {
	if taskID == "" {
		return errors.New("task ID cannot be empty")
	}
	return s.repo.DeleteByTaskID(ctx, taskID)
}

// ClearAll removes all retry items from the queue
func (s *RetryService) ClearAll(ctx context.Context) error {
	return s.repo.ClearAll(ctx)
}

// Compile-time interface verification
var _ RetryServiceInterface = (*RetryService)(nil)
