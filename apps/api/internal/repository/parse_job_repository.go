package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/vido/api/internal/models"
)

// ParseJobRepository provides data access operations for parse jobs.
type ParseJobRepository struct {
	db *sql.DB
}

// NewParseJobRepository creates a new ParseJobRepository.
func NewParseJobRepository(db *sql.DB) *ParseJobRepository {
	return &ParseJobRepository{db: db}
}

// Create inserts a new parse job into the database.
// Timestamps are UTC since ux3-1-6/migration 032: the driver stores time.Time
// as offset-bearing text and time-window queries compare it lexicographically,
// which is only correct when every row shares one offset.
func (r *ParseJobRepository) Create(ctx context.Context, job *models.ParseJob) error {
	if job == nil {
		return fmt.Errorf("parse job cannot be nil")
	}

	now := time.Now().UTC()
	job.CreatedAt = now
	job.UpdatedAt = now

	query := `
		INSERT INTO parse_jobs (
			id, torrent_hash, file_path, file_name, status,
			media_id, error_message, retry_count,
			created_at, updated_at, completed_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := r.db.ExecContext(ctx, query,
		job.ID,
		job.TorrentHash,
		job.FilePath,
		job.FileName,
		string(job.Status),
		job.MediaID,
		job.ErrorMessage,
		job.RetryCount,
		job.CreatedAt,
		job.UpdatedAt,
		job.CompletedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create parse job: %w", err)
	}

	return nil
}

// GetByID retrieves a parse job by its primary key.
func (r *ParseJobRepository) GetByID(ctx context.Context, id string) (*models.ParseJob, error) {
	query := `
		SELECT id, torrent_hash, file_path, file_name, status,
			media_id, error_message, retry_count,
			created_at, updated_at, completed_at
		FROM parse_jobs
		WHERE id = ?
	`

	job := &models.ParseJob{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&job.ID, &job.TorrentHash, &job.FilePath, &job.FileName, &job.Status,
		&job.MediaID, &job.ErrorMessage, &job.RetryCount,
		&job.CreatedAt, &job.UpdatedAt, &job.CompletedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("parse job with id %s not found", id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find parse job: %w", err)
	}

	return job, nil
}

// GetByTorrentHash retrieves a parse job by torrent hash.
func (r *ParseJobRepository) GetByTorrentHash(ctx context.Context, hash string) (*models.ParseJob, error) {
	query := `
		SELECT id, torrent_hash, file_path, file_name, status,
			media_id, error_message, retry_count,
			created_at, updated_at, completed_at
		FROM parse_jobs
		WHERE torrent_hash = ?
	`

	job := &models.ParseJob{}
	err := r.db.QueryRowContext(ctx, query, hash).Scan(
		&job.ID, &job.TorrentHash, &job.FilePath, &job.FileName, &job.Status,
		&job.MediaID, &job.ErrorMessage, &job.RetryCount,
		&job.CreatedAt, &job.UpdatedAt, &job.CompletedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find parse job by hash: %w", err)
	}

	return job, nil
}

// GetPending retrieves pending parse jobs ordered by creation time.
func (r *ParseJobRepository) GetPending(ctx context.Context, limit int) ([]*models.ParseJob, error) {
	query := `
		SELECT id, torrent_hash, file_path, file_name, status,
			media_id, error_message, retry_count,
			created_at, updated_at, completed_at
		FROM parse_jobs
		WHERE status = ?
		ORDER BY created_at ASC
		LIMIT ?
	`

	rows, err := r.db.QueryContext(ctx, query, string(models.ParseJobPending), limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get pending parse jobs: %w", err)
	}
	defer rows.Close()

	var jobs []*models.ParseJob
	for rows.Next() {
		job := &models.ParseJob{}
		err := rows.Scan(
			&job.ID, &job.TorrentHash, &job.FilePath, &job.FileName, &job.Status,
			&job.MediaID, &job.ErrorMessage, &job.RetryCount,
			&job.CreatedAt, &job.UpdatedAt, &job.CompletedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan parse job: %w", err)
		}
		jobs = append(jobs, job)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating parse jobs: %w", err)
	}

	return jobs, nil
}

// UpdateStatus updates a parse job's status and optional error message.
// UTC per the Create doc comment.
func (r *ParseJobRepository) UpdateStatus(ctx context.Context, id string, status models.ParseJobStatus, errMsg string) error {
	now := time.Now().UTC()

	var completedAt *time.Time
	if status == models.ParseJobCompleted || status == models.ParseJobFailed || status == models.ParseJobSkipped {
		completedAt = &now
	}

	var errorMessage *string
	if errMsg != "" {
		errorMessage = &errMsg
	}

	query := `
		UPDATE parse_jobs
		SET status = ?, error_message = ?, updated_at = ?, completed_at = ?
		WHERE id = ?
	`

	result, err := r.db.ExecContext(ctx, query, string(status), errorMessage, now, completedAt, id)
	if err != nil {
		return fmt.Errorf("failed to update parse job status: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("parse job with id %s not found", id)
	}

	return nil
}

// Update modifies an existing parse job in the database.
func (r *ParseJobRepository) Update(ctx context.Context, job *models.ParseJob) error {
	if job == nil {
		return fmt.Errorf("parse job cannot be nil")
	}

	job.UpdatedAt = time.Now().UTC()

	query := `
		UPDATE parse_jobs
		SET torrent_hash = ?, file_path = ?, file_name = ?, status = ?,
			media_id = ?, error_message = ?, retry_count = ?,
			updated_at = ?, completed_at = ?
		WHERE id = ?
	`

	result, err := r.db.ExecContext(ctx, query,
		job.TorrentHash, job.FilePath, job.FileName, string(job.Status),
		job.MediaID, job.ErrorMessage, job.RetryCount,
		job.UpdatedAt, job.CompletedAt,
		job.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update parse job: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("parse job with id %s not found", job.ID)
	}

	return nil
}

// Delete removes a parse job from the database by ID.
func (r *ParseJobRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM parse_jobs WHERE id = ?`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete parse job: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("parse job with id %s not found", id)
	}

	return nil
}

// ListAll retrieves all parse jobs ordered by creation time descending.
func (r *ParseJobRepository) ListAll(ctx context.Context, limit int) ([]*models.ParseJob, error) {
	query := `
		SELECT id, torrent_hash, file_path, file_name, status,
			media_id, error_message, retry_count,
			created_at, updated_at, completed_at
		FROM parse_jobs
		ORDER BY created_at DESC
		LIMIT ?
	`

	rows, err := r.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list parse jobs: %w", err)
	}
	defer rows.Close()

	var jobs []*models.ParseJob
	for rows.Next() {
		job := &models.ParseJob{}
		err := rows.Scan(
			&job.ID, &job.TorrentHash, &job.FilePath, &job.FileName, &job.Status,
			&job.MediaID, &job.ErrorMessage, &job.RetryCount,
			&job.CreatedAt, &job.UpdatedAt, &job.CompletedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan parse job: %w", err)
		}
		jobs = append(jobs, job)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating parse jobs: %w", err)
	}

	return jobs, nil
}

// CountByStatus counts parse jobs in the given state (ux3-1-6 attention cell).
// A real COUNT(*) on purpose — NOT a capped list length like /activity's
// pending count (tech-spec D2: do not replicate that defect).
func (r *ParseJobRepository) CountByStatus(ctx context.Context, status models.ParseJobStatus) (int, error) {
	var count int
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM parse_jobs WHERE status = ?`, string(status)).Scan(&count); err != nil {
		return 0, fmt.Errorf("failed to count parse jobs by status: %w", err)
	}
	return count, nil
}

// CompletedMediaIDsSince returns the distinct media ids of jobs completed
// at/after `since` (ux3-1-6 processed-today cell). Jobs without a media id are
// excluded — they created nothing countable.
//
// The comparison is lexicographic over the driver's time text, which is only
// correct when every row carries the same offset: migration 032 normalized the
// historical local-offset rows to UTC and this repository writes UTC since the
// same story, so the UTC-normalized parameter compares exactly (SQLite's
// datetime() cannot parse the driver's Go String() format — do NOT "fix" this
// by wrapping in datetime(), it returns NULL and matches nothing). The day
// window bounds the result set — no LIMIT by design (tech-spec D2).
func (r *ParseJobRepository) CompletedMediaIDsSince(ctx context.Context, since time.Time) ([]string, error) {
	query := `SELECT DISTINCT media_id FROM parse_jobs
		WHERE status = ? AND completed_at IS NOT NULL AND completed_at >= ?
		AND media_id IS NOT NULL AND media_id != ''`

	rows, err := r.db.QueryContext(ctx, query, string(models.ParseJobCompleted), since.UTC())
	if err != nil {
		return nil, fmt.Errorf("failed to query completed parse jobs since: %w", err)
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("failed to scan completed parse job media id: %w", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating completed parse job media ids: %w", err)
	}
	return ids, nil
}

// Compile-time interface verification
var _ ParseJobRepositoryInterface = (*ParseJobRepository)(nil)
