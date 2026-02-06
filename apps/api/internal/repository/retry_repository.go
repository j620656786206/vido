package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/vido/api/internal/retry"
)

// RetryRepository provides data access operations for retry queue items
type RetryRepository struct {
	db *sql.DB
}

// NewRetryRepository creates a new instance of RetryRepository
func NewRetryRepository(db *sql.DB) *RetryRepository {
	return &RetryRepository{
		db: db,
	}
}

// Add inserts a new retry item into the queue
func (r *RetryRepository) Add(ctx context.Context, item *retry.RetryItem) error {
	if item == nil {
		return fmt.Errorf("retry item cannot be nil")
	}

	query := `
		INSERT INTO retry_queue (
			id, task_id, task_type, payload, attempt_count,
			max_attempts, last_error, next_attempt_at, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := r.db.ExecContext(ctx, query,
		item.ID,
		item.TaskID,
		item.TaskType,
		string(item.Payload),
		item.AttemptCount,
		item.MaxAttempts,
		nullStringFromString(item.LastError),
		item.NextAttemptAt,
		item.CreatedAt,
		item.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to add retry item: %w", err)
	}

	return nil
}

// FindByID retrieves a retry item by its primary key
func (r *RetryRepository) FindByID(ctx context.Context, id string) (*retry.RetryItem, error) {
	query := `
		SELECT
			id, task_id, task_type, payload, attempt_count,
			max_attempts, last_error, next_attempt_at, created_at, updated_at
		FROM retry_queue
		WHERE id = ?
	`

	item := &retry.RetryItem{}
	var lastError sql.NullString
	var payload string

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&item.ID,
		&item.TaskID,
		&item.TaskType,
		&payload,
		&item.AttemptCount,
		&item.MaxAttempts,
		&lastError,
		&item.NextAttemptAt,
		&item.CreatedAt,
		&item.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find retry item: %w", err)
	}

	item.Payload = []byte(payload)
	item.LastError = lastError.String

	return item, nil
}

// FindByTaskID retrieves a retry item by its task ID
func (r *RetryRepository) FindByTaskID(ctx context.Context, taskID string) (*retry.RetryItem, error) {
	query := `
		SELECT
			id, task_id, task_type, payload, attempt_count,
			max_attempts, last_error, next_attempt_at, created_at, updated_at
		FROM retry_queue
		WHERE task_id = ?
	`

	item := &retry.RetryItem{}
	var lastError sql.NullString
	var payload string

	err := r.db.QueryRowContext(ctx, query, taskID).Scan(
		&item.ID,
		&item.TaskID,
		&item.TaskType,
		&payload,
		&item.AttemptCount,
		&item.MaxAttempts,
		&lastError,
		&item.NextAttemptAt,
		&item.CreatedAt,
		&item.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find retry item by task ID: %w", err)
	}

	item.Payload = []byte(payload)
	item.LastError = lastError.String

	return item, nil
}

// GetPending retrieves all retry items ready for processing (next_attempt_at <= now)
func (r *RetryRepository) GetPending(ctx context.Context, now time.Time) ([]*retry.RetryItem, error) {
	query := `
		SELECT
			id, task_id, task_type, payload, attempt_count,
			max_attempts, last_error, next_attempt_at, created_at, updated_at
		FROM retry_queue
		WHERE next_attempt_at <= ?
		ORDER BY next_attempt_at ASC
	`

	rows, err := r.db.QueryContext(ctx, query, now)
	if err != nil {
		return nil, fmt.Errorf("failed to get pending retries: %w", err)
	}
	defer rows.Close()

	return scanRetryItems(rows)
}

// GetAll retrieves all retry items in the queue
func (r *RetryRepository) GetAll(ctx context.Context) ([]*retry.RetryItem, error) {
	query := `
		SELECT
			id, task_id, task_type, payload, attempt_count,
			max_attempts, last_error, next_attempt_at, created_at, updated_at
		FROM retry_queue
		ORDER BY next_attempt_at ASC
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get all retries: %w", err)
	}
	defer rows.Close()

	return scanRetryItems(rows)
}

// Update modifies an existing retry item
func (r *RetryRepository) Update(ctx context.Context, item *retry.RetryItem) error {
	if item == nil {
		return fmt.Errorf("retry item cannot be nil")
	}

	query := `
		UPDATE retry_queue
		SET
			attempt_count = ?,
			last_error = ?,
			next_attempt_at = ?,
			updated_at = ?
		WHERE id = ?
	`

	result, err := r.db.ExecContext(ctx, query,
		item.AttemptCount,
		nullStringFromString(item.LastError),
		item.NextAttemptAt,
		time.Now(),
		item.ID,
	)

	if err != nil {
		return fmt.Errorf("failed to update retry item: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("retry item with id %s not found", item.ID)
	}

	return nil
}

// Delete removes a retry item from the queue
func (r *RetryRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM retry_queue WHERE id = ?`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete retry item: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("retry item with id %s not found", id)
	}

	return nil
}

// DeleteByTaskID removes a retry item by its task ID
func (r *RetryRepository) DeleteByTaskID(ctx context.Context, taskID string) error {
	query := `DELETE FROM retry_queue WHERE task_id = ?`

	_, err := r.db.ExecContext(ctx, query, taskID)
	if err != nil {
		return fmt.Errorf("failed to delete retry item by task ID: %w", err)
	}

	return nil
}

// Count returns the total number of retry items in the queue
func (r *RetryRepository) Count(ctx context.Context) (int, error) {
	query := `SELECT COUNT(*) FROM retry_queue`

	var count int
	err := r.db.QueryRowContext(ctx, query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count retry items: %w", err)
	}

	return count, nil
}

// CountByTaskType returns the number of retry items for a specific task type
func (r *RetryRepository) CountByTaskType(ctx context.Context, taskType string) (int, error) {
	query := `SELECT COUNT(*) FROM retry_queue WHERE task_type = ?`

	var count int
	err := r.db.QueryRowContext(ctx, query, taskType).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count retry items by type: %w", err)
	}

	return count, nil
}

// ClearAll removes all retry items from the queue
func (r *RetryRepository) ClearAll(ctx context.Context) error {
	query := `DELETE FROM retry_queue`

	_, err := r.db.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to clear retry queue: %w", err)
	}

	return nil
}

// Stats methods for tracking historical retry data (Story 3.11)

// IncrementQueued increments the queued count for a task type
func (r *RetryRepository) IncrementQueued(ctx context.Context, taskType string) error {
	return r.incrementStat(ctx, taskType, "total_queued")
}

// IncrementSucceeded increments the succeeded count for a task type
func (r *RetryRepository) IncrementSucceeded(ctx context.Context, taskType string) error {
	return r.incrementStat(ctx, taskType, "total_succeeded")
}

// IncrementFailed increments the failed count for a task type
func (r *RetryRepository) IncrementFailed(ctx context.Context, taskType string) error {
	return r.incrementStat(ctx, taskType, "total_failed")
}

// IncrementExhausted increments the exhausted count for a task type
func (r *RetryRepository) IncrementExhausted(ctx context.Context, taskType string) error {
	return r.incrementStat(ctx, taskType, "total_exhausted")
}

// incrementStat is a helper to increment a specific stat column
func (r *RetryRepository) incrementStat(ctx context.Context, taskType, column string) error {
	today := time.Now().Format("2006-01-02")

	// Use UPSERT to create or update the stats row
	query := fmt.Sprintf(`
		INSERT INTO retry_stats (task_type, date, %s, created_at, updated_at)
		VALUES (?, ?, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		ON CONFLICT(task_type, date)
		DO UPDATE SET %s = %s + 1, updated_at = CURRENT_TIMESTAMP
	`, column, column, column)

	_, err := r.db.ExecContext(ctx, query, taskType, today)
	if err != nil {
		return fmt.Errorf("failed to increment %s stat: %w", column, err)
	}

	return nil
}

// GetStats returns aggregated retry statistics
func (r *RetryRepository) GetStats(ctx context.Context) (*retry.RetryStats, error) {
	// Get pending count from retry_queue
	pendingCount, err := r.Count(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get pending count: %w", err)
	}

	// Get historical totals from retry_stats
	query := `
		SELECT
			COALESCE(SUM(total_succeeded), 0) as succeeded,
			COALESCE(SUM(total_failed), 0) as failed
		FROM retry_stats
	`

	var succeeded, failed int
	err = r.db.QueryRowContext(ctx, query).Scan(&succeeded, &failed)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to get stats: %w", err)
	}

	return &retry.RetryStats{
		TotalPending:   pendingCount,
		TotalSucceeded: succeeded,
		TotalFailed:    failed,
	}, nil
}

// Helper functions

func nullStringFromString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{Valid: false}
	}
	return sql.NullString{String: s, Valid: true}
}

func scanRetryItems(rows *sql.Rows) ([]*retry.RetryItem, error) {
	var items []*retry.RetryItem

	for rows.Next() {
		item := &retry.RetryItem{}
		var lastError sql.NullString
		var payload string

		err := rows.Scan(
			&item.ID,
			&item.TaskID,
			&item.TaskType,
			&payload,
			&item.AttemptCount,
			&item.MaxAttempts,
			&lastError,
			&item.NextAttemptAt,
			&item.CreatedAt,
			&item.UpdatedAt,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan retry item: %w", err)
		}

		item.Payload = []byte(payload)
		item.LastError = lastError.String

		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating retry items: %w", err)
	}

	return items, nil
}
