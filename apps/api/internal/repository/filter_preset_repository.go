package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/vido/api/internal/models"
)

// ErrFilterPresetNotFound is returned when a filter preset lookup finds no matching record.
var ErrFilterPresetNotFound = errors.New("filter preset not found")

// FilterPresetRepositoryInterface defines the contract for filter preset data access.
type FilterPresetRepositoryInterface interface {
	Create(ctx context.Context, preset *models.FilterPreset) error
	GetAll(ctx context.Context) ([]models.FilterPreset, error)
	Delete(ctx context.Context, id string) error
	Count(ctx context.Context) (int, error)
}

// FilterPresetRepository provides SQLite data access for filter presets.
type FilterPresetRepository struct {
	db *sql.DB
}

// NewFilterPresetRepository creates a new FilterPresetRepository.
func NewFilterPresetRepository(db *sql.DB) *FilterPresetRepository {
	return &FilterPresetRepository{db: db}
}

// Compile-time interface verification.
var _ FilterPresetRepositoryInterface = (*FilterPresetRepository)(nil)

func (r *FilterPresetRepository) Create(ctx context.Context, preset *models.FilterPreset) error {
	if preset == nil {
		return fmt.Errorf("preset cannot be nil")
	}
	if preset.ID == "" {
		preset.ID = uuid.New().String()
	}
	preset.CreatedAt = time.Now()

	query := `
		INSERT INTO filter_presets (id, name, filters, sort_order, created_at)
		VALUES (?, ?, ?, ?, ?)
	`
	_, err := r.db.ExecContext(ctx, query,
		preset.ID, preset.Name, preset.Filters, preset.SortOrder, preset.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create filter preset: %w", err)
	}
	return nil
}

func (r *FilterPresetRepository) GetAll(ctx context.Context) ([]models.FilterPreset, error) {
	query := `
		SELECT id, name, filters, sort_order, created_at
		FROM filter_presets
		ORDER BY sort_order, created_at
	`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list filter presets: %w", err)
	}
	defer rows.Close()

	var presets []models.FilterPreset
	for rows.Next() {
		var p models.FilterPreset
		if err := rows.Scan(&p.ID, &p.Name, &p.Filters, &p.SortOrder, &p.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan filter preset: %w", err)
		}
		presets = append(presets, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating filter presets: %w", err)
	}
	return presets, nil
}

func (r *FilterPresetRepository) Delete(ctx context.Context, id string) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM filter_presets WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("failed to delete filter preset: %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("filter preset with id %s: %w", id, ErrFilterPresetNotFound)
	}
	return nil
}

func (r *FilterPresetRepository) Count(ctx context.Context) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM filter_presets`).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count filter presets: %w", err)
	}
	return count, nil
}
