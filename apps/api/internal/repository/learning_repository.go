package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/vido/api/internal/models"
)

// LearningRepository provides data access operations for filename pattern mappings
type LearningRepository struct {
	db *sql.DB
}

// NewLearningRepository creates a new instance of LearningRepository
func NewLearningRepository(db *sql.DB) *LearningRepository {
	return &LearningRepository{
		db: db,
	}
}

// Save inserts a new filename mapping into the database
func (r *LearningRepository) Save(ctx context.Context, mapping *models.FilenameMapping) error {
	if mapping == nil {
		return fmt.Errorf("mapping cannot be nil")
	}

	query := `
		INSERT INTO filename_mappings (
			id, pattern, pattern_type, pattern_regex, fansub_group,
			title_pattern, metadata_type, metadata_id, tmdb_id,
			confidence, use_count, created_at, last_used_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := r.db.ExecContext(ctx, query,
		mapping.ID,
		mapping.Pattern,
		mapping.PatternType,
		nullString(mapping.PatternRegex),
		nullString(mapping.FansubGroup),
		nullString(mapping.TitlePattern),
		mapping.MetadataType,
		mapping.MetadataID,
		nullInt(mapping.TmdbID),
		mapping.Confidence,
		mapping.UseCount,
		mapping.CreatedAt,
		nullTime(mapping.LastUsedAt),
	)

	if err != nil {
		return fmt.Errorf("failed to save mapping: %w", err)
	}

	return nil
}

// FindByID retrieves a mapping by its primary key
func (r *LearningRepository) FindByID(ctx context.Context, id string) (*models.FilenameMapping, error) {
	query := `
		SELECT
			id, pattern, pattern_type, pattern_regex, fansub_group,
			title_pattern, metadata_type, metadata_id, tmdb_id,
			confidence, use_count, created_at, last_used_at
		FROM filename_mappings
		WHERE id = ?
	`

	mapping := &models.FilenameMapping{}
	var patternRegex, fansubGroup, titlePattern sql.NullString
	var tmdbID sql.NullInt64
	var lastUsedAt sql.NullTime

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&mapping.ID,
		&mapping.Pattern,
		&mapping.PatternType,
		&patternRegex,
		&fansubGroup,
		&titlePattern,
		&mapping.MetadataType,
		&mapping.MetadataID,
		&tmdbID,
		&mapping.Confidence,
		&mapping.UseCount,
		&mapping.CreatedAt,
		&lastUsedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find mapping: %w", err)
	}

	// Handle nullable fields
	mapping.PatternRegex = patternRegex.String
	mapping.FansubGroup = fansubGroup.String
	mapping.TitlePattern = titlePattern.String
	if tmdbID.Valid {
		mapping.TmdbID = int(tmdbID.Int64)
	}
	if lastUsedAt.Valid {
		mapping.LastUsedAt = &lastUsedAt.Time
	}

	return mapping, nil
}

// FindByExactPattern retrieves a mapping by exact pattern match
func (r *LearningRepository) FindByExactPattern(ctx context.Context, pattern string) (*models.FilenameMapping, error) {
	query := `
		SELECT
			id, pattern, pattern_type, pattern_regex, fansub_group,
			title_pattern, metadata_type, metadata_id, tmdb_id,
			confidence, use_count, created_at, last_used_at
		FROM filename_mappings
		WHERE pattern = ?
	`

	mapping := &models.FilenameMapping{}
	var patternRegex, fansubGroup, titlePattern sql.NullString
	var tmdbID sql.NullInt64
	var lastUsedAt sql.NullTime

	err := r.db.QueryRowContext(ctx, query, pattern).Scan(
		&mapping.ID,
		&mapping.Pattern,
		&mapping.PatternType,
		&patternRegex,
		&fansubGroup,
		&titlePattern,
		&mapping.MetadataType,
		&mapping.MetadataID,
		&tmdbID,
		&mapping.Confidence,
		&mapping.UseCount,
		&mapping.CreatedAt,
		&lastUsedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find mapping by pattern: %w", err)
	}

	// Handle nullable fields
	mapping.PatternRegex = patternRegex.String
	mapping.FansubGroup = fansubGroup.String
	mapping.TitlePattern = titlePattern.String
	if tmdbID.Valid {
		mapping.TmdbID = int(tmdbID.Int64)
	}
	if lastUsedAt.Valid {
		mapping.LastUsedAt = &lastUsedAt.Time
	}

	return mapping, nil
}

// FindByFansubAndTitle retrieves mappings by fansub group and title pattern
func (r *LearningRepository) FindByFansubAndTitle(ctx context.Context, fansubGroup, titlePattern string) ([]*models.FilenameMapping, error) {
	query := `
		SELECT
			id, pattern, pattern_type, pattern_regex, fansub_group,
			title_pattern, metadata_type, metadata_id, tmdb_id,
			confidence, use_count, created_at, last_used_at
		FROM filename_mappings
		WHERE fansub_group = ? AND title_pattern = ?
	`

	rows, err := r.db.QueryContext(ctx, query, fansubGroup, titlePattern)
	if err != nil {
		return nil, fmt.Errorf("failed to find mappings: %w", err)
	}
	defer rows.Close()

	return scanMappings(rows)
}

// ListWithRegex retrieves all mappings that have a regex pattern
func (r *LearningRepository) ListWithRegex(ctx context.Context) ([]*models.FilenameMapping, error) {
	query := `
		SELECT
			id, pattern, pattern_type, pattern_regex, fansub_group,
			title_pattern, metadata_type, metadata_id, tmdb_id,
			confidence, use_count, created_at, last_used_at
		FROM filename_mappings
		WHERE pattern_regex IS NOT NULL AND pattern_regex != ''
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list mappings with regex: %w", err)
	}
	defer rows.Close()

	return scanMappings(rows)
}

// ListAll retrieves all filename mappings
func (r *LearningRepository) ListAll(ctx context.Context) ([]*models.FilenameMapping, error) {
	query := `
		SELECT
			id, pattern, pattern_type, pattern_regex, fansub_group,
			title_pattern, metadata_type, metadata_id, tmdb_id,
			confidence, use_count, created_at, last_used_at
		FROM filename_mappings
		ORDER BY use_count DESC, created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list all mappings: %w", err)
	}
	defer rows.Close()

	return scanMappings(rows)
}

// Delete removes a filename mapping by ID
func (r *LearningRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM filename_mappings WHERE id = ?`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete mapping: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("mapping with id %s not found", id)
	}

	return nil
}

// IncrementUseCount increments the use count and updates last_used_at
func (r *LearningRepository) IncrementUseCount(ctx context.Context, id string) error {
	query := `
		UPDATE filename_mappings
		SET use_count = use_count + 1, last_used_at = ?
		WHERE id = ?
	`

	result, err := r.db.ExecContext(ctx, query, time.Now(), id)
	if err != nil {
		return fmt.Errorf("failed to increment use count: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("mapping with id %s not found", id)
	}

	return nil
}

// Count returns the total number of filename mappings
func (r *LearningRepository) Count(ctx context.Context) (int, error) {
	query := `SELECT COUNT(*) FROM filename_mappings`

	var count int
	err := r.db.QueryRowContext(ctx, query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count mappings: %w", err)
	}

	return count, nil
}

// Update modifies an existing filename mapping
func (r *LearningRepository) Update(ctx context.Context, mapping *models.FilenameMapping) error {
	if mapping == nil {
		return fmt.Errorf("mapping cannot be nil")
	}

	query := `
		UPDATE filename_mappings
		SET
			pattern = ?,
			pattern_type = ?,
			pattern_regex = ?,
			fansub_group = ?,
			title_pattern = ?,
			metadata_type = ?,
			metadata_id = ?,
			tmdb_id = ?,
			confidence = ?,
			use_count = ?,
			last_used_at = ?
		WHERE id = ?
	`

	result, err := r.db.ExecContext(ctx, query,
		mapping.Pattern,
		mapping.PatternType,
		nullString(mapping.PatternRegex),
		nullString(mapping.FansubGroup),
		nullString(mapping.TitlePattern),
		mapping.MetadataType,
		mapping.MetadataID,
		nullInt(mapping.TmdbID),
		mapping.Confidence,
		mapping.UseCount,
		nullTime(mapping.LastUsedAt),
		mapping.ID,
	)

	if err != nil {
		return fmt.Errorf("failed to update mapping: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("mapping with id %s not found", mapping.ID)
	}

	return nil
}

// Helper functions

func nullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{Valid: false}
	}
	return sql.NullString{String: s, Valid: true}
}

func nullInt(i int) sql.NullInt64 {
	if i == 0 {
		return sql.NullInt64{Valid: false}
	}
	return sql.NullInt64{Int64: int64(i), Valid: true}
}

func nullTime(t *time.Time) sql.NullTime {
	if t == nil {
		return sql.NullTime{Valid: false}
	}
	return sql.NullTime{Time: *t, Valid: true}
}

func scanMappings(rows *sql.Rows) ([]*models.FilenameMapping, error) {
	var mappings []*models.FilenameMapping

	for rows.Next() {
		mapping := &models.FilenameMapping{}
		var patternRegex, fansubGroup, titlePattern sql.NullString
		var tmdbID sql.NullInt64
		var lastUsedAt sql.NullTime

		err := rows.Scan(
			&mapping.ID,
			&mapping.Pattern,
			&mapping.PatternType,
			&patternRegex,
			&fansubGroup,
			&titlePattern,
			&mapping.MetadataType,
			&mapping.MetadataID,
			&tmdbID,
			&mapping.Confidence,
			&mapping.UseCount,
			&mapping.CreatedAt,
			&lastUsedAt,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan mapping: %w", err)
		}

		// Handle nullable fields
		mapping.PatternRegex = patternRegex.String
		mapping.FansubGroup = fansubGroup.String
		mapping.TitlePattern = titlePattern.String
		if tmdbID.Valid {
			mapping.TmdbID = int(tmdbID.Int64)
		}
		if lastUsedAt.Valid {
			mapping.LastUsedAt = &lastUsedAt.Time
		}

		mappings = append(mappings, mapping)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating mappings: %w", err)
	}

	return mappings, nil
}
