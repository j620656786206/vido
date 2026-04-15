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

// ErrExploreBlockNotFound is returned when an explore block lookup finds no matching record.
var ErrExploreBlockNotFound = errors.New("explore block not found")

// ExploreBlockRepositoryInterface defines the contract for explore block data access.
type ExploreBlockRepositoryInterface interface {
	Create(ctx context.Context, block *models.ExploreBlock) error
	GetByID(ctx context.Context, id string) (*models.ExploreBlock, error)
	GetAll(ctx context.Context) ([]models.ExploreBlock, error)
	Update(ctx context.Context, block *models.ExploreBlock) error
	Delete(ctx context.Context, id string) error
	Reorder(ctx context.Context, orderedIDs []string) error
	Count(ctx context.Context) (int, error)
}

// ExploreBlockRepository provides SQLite data access for explore blocks.
type ExploreBlockRepository struct {
	db *sql.DB
}

// NewExploreBlockRepository creates a new ExploreBlockRepository.
func NewExploreBlockRepository(db *sql.DB) *ExploreBlockRepository {
	return &ExploreBlockRepository{db: db}
}

// Compile-time interface verification.
var _ ExploreBlockRepositoryInterface = (*ExploreBlockRepository)(nil)

func (r *ExploreBlockRepository) Create(ctx context.Context, block *models.ExploreBlock) error {
	if block == nil {
		return fmt.Errorf("block cannot be nil")
	}
	if block.ID == "" {
		block.ID = uuid.New().String()
	}
	now := time.Now()
	block.CreatedAt = now
	block.UpdatedAt = now

	query := `
		INSERT INTO explore_blocks
			(id, name, content_type, genre_ids, language, region, sort_by, max_items, sort_order, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err := r.db.ExecContext(ctx, query,
		block.ID, block.Name, block.ContentType,
		block.GenreIDs, block.Language, block.Region, block.SortBy,
		block.MaxItems, block.SortOrder,
		block.CreatedAt, block.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create explore block: %w", err)
	}
	return nil
}

func (r *ExploreBlockRepository) GetByID(ctx context.Context, id string) (*models.ExploreBlock, error) {
	query := `
		SELECT id, name, content_type, genre_ids, language, region, sort_by, max_items, sort_order, created_at, updated_at
		FROM explore_blocks WHERE id = ?
	`
	block := &models.ExploreBlock{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&block.ID, &block.Name, &block.ContentType,
		&block.GenreIDs, &block.Language, &block.Region, &block.SortBy,
		&block.MaxItems, &block.SortOrder,
		&block.CreatedAt, &block.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("explore block with id %s: %w", id, ErrExploreBlockNotFound)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find explore block: %w", err)
	}
	return block, nil
}

func (r *ExploreBlockRepository) GetAll(ctx context.Context) ([]models.ExploreBlock, error) {
	query := `
		SELECT id, name, content_type, genre_ids, language, region, sort_by, max_items, sort_order, created_at, updated_at
		FROM explore_blocks
		ORDER BY sort_order, created_at
	`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list explore blocks: %w", err)
	}
	defer rows.Close()

	var blocks []models.ExploreBlock
	for rows.Next() {
		var b models.ExploreBlock
		if err := rows.Scan(
			&b.ID, &b.Name, &b.ContentType,
			&b.GenreIDs, &b.Language, &b.Region, &b.SortBy,
			&b.MaxItems, &b.SortOrder,
			&b.CreatedAt, &b.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan explore block: %w", err)
		}
		blocks = append(blocks, b)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating explore blocks: %w", err)
	}
	return blocks, nil
}

func (r *ExploreBlockRepository) Update(ctx context.Context, block *models.ExploreBlock) error {
	if block == nil {
		return fmt.Errorf("block cannot be nil")
	}
	block.UpdatedAt = time.Now()
	query := `
		UPDATE explore_blocks
		SET name = ?, content_type = ?, genre_ids = ?, language = ?, region = ?, sort_by = ?,
			max_items = ?, sort_order = ?, updated_at = ?
		WHERE id = ?
	`
	result, err := r.db.ExecContext(ctx, query,
		block.Name, block.ContentType, block.GenreIDs, block.Language, block.Region, block.SortBy,
		block.MaxItems, block.SortOrder, block.UpdatedAt, block.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update explore block: %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("explore block with id %s: %w", block.ID, ErrExploreBlockNotFound)
	}
	return nil
}

func (r *ExploreBlockRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM explore_blocks WHERE id = ?`
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete explore block: %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("explore block with id %s: %w", id, ErrExploreBlockNotFound)
	}
	return nil
}

// Reorder updates sort_order for the given IDs in a single transaction.
// Each ID in orderedIDs receives sort_order = index. IDs not present in the
// slice are left untouched. Missing IDs cause the whole transaction to roll back.
func (r *ExploreBlockRepository) Reorder(ctx context.Context, orderedIDs []string) error {
	if len(orderedIDs) == 0 {
		return nil
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin reorder tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	now := time.Now()
	stmt, err := tx.PrepareContext(ctx, `UPDATE explore_blocks SET sort_order = ?, updated_at = ? WHERE id = ?`)
	if err != nil {
		return fmt.Errorf("failed to prepare reorder statement: %w", err)
	}
	defer stmt.Close()

	for idx, id := range orderedIDs {
		result, err := stmt.ExecContext(ctx, idx, now, id)
		if err != nil {
			return fmt.Errorf("failed to reorder block %s: %w", id, err)
		}
		rowsAffected, err := result.RowsAffected()
		if err != nil {
			return fmt.Errorf("failed to get rows affected during reorder: %w", err)
		}
		if rowsAffected == 0 {
			return fmt.Errorf("explore block with id %s: %w", id, ErrExploreBlockNotFound)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit reorder: %w", err)
	}
	return nil
}

func (r *ExploreBlockRepository) Count(ctx context.Context) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM explore_blocks`).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count explore blocks: %w", err)
	}
	return count, nil
}
