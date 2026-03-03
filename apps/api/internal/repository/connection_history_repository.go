package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/vido/api/internal/models"
)

// ConnectionHistoryRepositoryInterface defines the contract for connection history data access
type ConnectionHistoryRepositoryInterface interface {
	Create(ctx context.Context, event *models.ConnectionEvent) error
	GetHistory(ctx context.Context, service string, limit int) ([]models.ConnectionEvent, error)
}

// ConnectionHistoryRepository implements ConnectionHistoryRepositoryInterface with SQLite
type ConnectionHistoryRepository struct {
	db *sql.DB
}

// NewConnectionHistoryRepository creates a new ConnectionHistoryRepository
func NewConnectionHistoryRepository(db *sql.DB) *ConnectionHistoryRepository {
	return &ConnectionHistoryRepository{db: db}
}

// Compile-time interface verification
var _ ConnectionHistoryRepositoryInterface = (*ConnectionHistoryRepository)(nil)

// Create inserts a new connection event
func (r *ConnectionHistoryRepository) Create(ctx context.Context, event *models.ConnectionEvent) error {
	query := `
		INSERT INTO connection_history (id, service, event_type, status, message, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`

	_, err := r.db.ExecContext(ctx, query,
		event.ID,
		event.Service,
		string(event.EventType),
		event.Status,
		event.Message,
		event.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to insert connection event: %w", err)
	}

	return nil
}

// GetHistory retrieves recent connection events for a service
func (r *ConnectionHistoryRepository) GetHistory(ctx context.Context, service string, limit int) ([]models.ConnectionEvent, error) {
	if limit <= 0 {
		limit = 20
	}

	query := `
		SELECT id, service, event_type, status, COALESCE(message, ''), created_at
		FROM connection_history
		WHERE service = ?
		ORDER BY created_at DESC
		LIMIT ?
	`

	rows, err := r.db.QueryContext(ctx, query, service, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query connection history: %w", err)
	}
	defer rows.Close()

	var events []models.ConnectionEvent
	for rows.Next() {
		var event models.ConnectionEvent
		var eventType string
		if err := rows.Scan(
			&event.ID,
			&event.Service,
			&eventType,
			&event.Status,
			&event.Message,
			&event.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan connection event: %w", err)
		}
		event.EventType = models.ConnectionEventType(eventType)
		events = append(events, event)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating connection history: %w", err)
	}

	return events, nil
}
