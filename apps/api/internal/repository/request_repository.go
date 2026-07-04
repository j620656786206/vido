package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/vido/api/internal/models"
)

// ErrRequestNotFound is returned when a request lookup finds no matching record.
var ErrRequestNotFound = errors.New("request not found")

// ErrRequestDuplicate is returned when an ACTIVE request (pending/searching/
// downloading) already exists for the same (tmdb_id, media_type) — surfaced
// either by the service pre-check or by the partial unique index on a race
// (Story 13-1a AC #4).
var ErrRequestDuplicate = errors.New("active request already exists")

// RequestRepositoryInterface defines the contract for request data access.
type RequestRepositoryInterface interface {
	Create(ctx context.Context, request *models.Request) error
	List(ctx context.Context) ([]models.Request, error)
	FindActiveByTMDbID(ctx context.Context, tmdbID int64, mediaType string) (*models.Request, error)
}

// RequestRepository provides SQLite data access for media requests.
type RequestRepository struct {
	db *sql.DB
}

// NewRequestRepository creates a new RequestRepository.
func NewRequestRepository(db *sql.DB) *RequestRepository {
	return &RequestRepository{db: db}
}

// Compile-time interface verification.
var _ RequestRepositoryInterface = (*RequestRepository)(nil)

// requestColumns is the canonical column list — INSERT, SELECT, and scan stay
// in sync through it (Rule 15 DB Column Sync).
const requestColumns = `id, tmdb_id, media_type, title, status, fulfilment_source, external_id, seasons, episodes, error_message, requested_at, updated_at`

func scanRequest(scanner interface{ Scan(dest ...any) error }) (models.Request, error) {
	var r models.Request
	err := scanner.Scan(
		&r.ID, &r.TMDbID, &r.MediaType, &r.Title, &r.Status,
		&r.FulfilmentSource, &r.ExternalID, &r.Seasons, &r.Episodes,
		&r.ErrorMessage, &r.RequestedAt, &r.UpdatedAt,
	)
	return r, err
}

func (r *RequestRepository) Create(ctx context.Context, request *models.Request) error {
	if request == nil {
		return fmt.Errorf("request cannot be nil")
	}
	if request.ID == "" {
		request.ID = uuid.New().String()
	}
	if request.Status == "" {
		request.Status = models.RequestStatusPending
	}
	now := time.Now()
	request.RequestedAt = now
	request.UpdatedAt = now

	query := `INSERT INTO requests (` + requestColumns + `) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	_, err := r.db.ExecContext(ctx, query,
		request.ID, request.TMDbID, request.MediaType, request.Title, request.Status,
		request.FulfilmentSource, request.ExternalID, request.Seasons, request.Episodes,
		request.ErrorMessage, request.RequestedAt, request.UpdatedAt,
	)
	if err != nil {
		// The partial unique index (idx_requests_active_unique) rejects a second
		// ACTIVE request for the same (tmdb_id, media_type) on a race the service
		// pre-check missed; map it to the typed sentinel, never a raw 500.
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return fmt.Errorf("request for tmdb_id %d (%s): %w", request.TMDbID, request.MediaType, ErrRequestDuplicate)
		}
		return fmt.Errorf("failed to create request: %w", err)
	}
	return nil
}

func (r *RequestRepository) List(ctx context.Context) ([]models.Request, error) {
	query := `SELECT ` + requestColumns + ` FROM requests ORDER BY requested_at DESC`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list requests: %w", err)
	}
	defer rows.Close()

	var requests []models.Request
	for rows.Next() {
		req, err := scanRequest(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan request: %w", err)
		}
		requests = append(requests, req)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating requests: %w", err)
	}
	return requests, nil
}

func (r *RequestRepository) FindActiveByTMDbID(ctx context.Context, tmdbID int64, mediaType string) (*models.Request, error) {
	query := `SELECT ` + requestColumns + ` FROM requests
		WHERE tmdb_id = ? AND media_type = ? AND status IN ('pending', 'searching', 'downloading')
		LIMIT 1`
	row := r.db.QueryRowContext(ctx, query, tmdbID, mediaType)
	req, err := scanRequest(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("active request for tmdb_id %d (%s): %w", tmdbID, mediaType, ErrRequestNotFound)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find active request: %w", err)
	}
	return &req, nil
}
