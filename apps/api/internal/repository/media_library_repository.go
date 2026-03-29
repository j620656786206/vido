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

// ErrLibraryNotFound is returned when a library lookup finds no matching record.
var ErrLibraryNotFound = errors.New("library not found")

// ErrLibraryPathNotFound is returned when a library path lookup finds no matching record.
var ErrLibraryPathNotFound = errors.New("library path not found")

// ErrLibraryPathDuplicate is returned when a path already exists in another library.
var ErrLibraryPathDuplicate = errors.New("path already exists in a library")

// MediaLibraryRepositoryInterface defines the contract for media library data access.
type MediaLibraryRepositoryInterface interface {
	Create(ctx context.Context, library *models.MediaLibrary) error
	GetByID(ctx context.Context, id string) (*models.MediaLibrary, error)
	GetAll(ctx context.Context) ([]models.MediaLibrary, error)
	GetAllWithPathsAndCounts(ctx context.Context) ([]models.MediaLibraryWithPaths, error)
	Update(ctx context.Context, library *models.MediaLibrary) error
	Delete(ctx context.Context, id string) error

	AddPath(ctx context.Context, path *models.MediaLibraryPath) error
	RemovePath(ctx context.Context, pathID string) error
	GetPathsByLibraryID(ctx context.Context, libraryID string) ([]models.MediaLibraryPath, error)
	GetAllPaths(ctx context.Context) ([]models.MediaLibraryPath, error)
	UpdatePathStatus(ctx context.Context, pathID string, status models.MediaLibraryPathStatus) error
}

// MediaLibraryRepository provides SQLite data access for media libraries.
type MediaLibraryRepository struct {
	db *sql.DB
}

// NewMediaLibraryRepository creates a new instance of MediaLibraryRepository.
func NewMediaLibraryRepository(db *sql.DB) *MediaLibraryRepository {
	return &MediaLibraryRepository{db: db}
}

func (r *MediaLibraryRepository) Create(ctx context.Context, library *models.MediaLibrary) error {
	if library == nil {
		return fmt.Errorf("library cannot be nil")
	}
	if library.ID == "" {
		library.ID = uuid.New().String()
	}

	now := time.Now()
	library.CreatedAt = now
	library.UpdatedAt = now

	query := `
		INSERT INTO media_libraries (id, name, content_type, auto_detect, sort_order, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`
	_, err := r.db.ExecContext(ctx, query,
		library.ID, library.Name, library.ContentType,
		library.AutoDetect, library.SortOrder,
		library.CreatedAt, library.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create library: %w", err)
	}
	return nil
}

func (r *MediaLibraryRepository) GetByID(ctx context.Context, id string) (*models.MediaLibrary, error) {
	query := `
		SELECT id, name, content_type, auto_detect, sort_order, created_at, updated_at
		FROM media_libraries WHERE id = ?
	`
	lib := &models.MediaLibrary{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&lib.ID, &lib.Name, &lib.ContentType,
		&lib.AutoDetect, &lib.SortOrder,
		&lib.CreatedAt, &lib.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("library with id %s: %w", id, ErrLibraryNotFound)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find library: %w", err)
	}
	return lib, nil
}

func (r *MediaLibraryRepository) GetAll(ctx context.Context) ([]models.MediaLibrary, error) {
	query := `
		SELECT id, name, content_type, auto_detect, sort_order, created_at, updated_at
		FROM media_libraries ORDER BY sort_order, created_at
	`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list libraries: %w", err)
	}
	defer rows.Close()

	var libraries []models.MediaLibrary
	for rows.Next() {
		var lib models.MediaLibrary
		if err := rows.Scan(
			&lib.ID, &lib.Name, &lib.ContentType,
			&lib.AutoDetect, &lib.SortOrder,
			&lib.CreatedAt, &lib.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan library: %w", err)
		}
		libraries = append(libraries, lib)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating libraries: %w", err)
	}
	return libraries, nil
}

func (r *MediaLibraryRepository) GetAllWithPathsAndCounts(ctx context.Context) ([]models.MediaLibraryWithPaths, error) {
	libraries, err := r.GetAll(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]models.MediaLibraryWithPaths, 0, len(libraries))
	for _, lib := range libraries {
		paths, err := r.GetPathsByLibraryID(ctx, lib.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to get paths for library %s: %w", lib.ID, err)
		}

		count, err := r.getMediaCount(ctx, lib.ID, lib.ContentType)
		if err != nil {
			return nil, fmt.Errorf("failed to get media count for library %s: %w", lib.ID, err)
		}

		result = append(result, models.MediaLibraryWithPaths{
			MediaLibrary: lib,
			Paths:        paths,
			MediaCount:   count,
		})
	}
	return result, nil
}

func (r *MediaLibraryRepository) getMediaCount(ctx context.Context, libraryID string, contentType models.MediaLibraryContentType) (int, error) {
	var query string
	switch contentType {
	case models.ContentTypeMovie:
		query = `SELECT COUNT(*) FROM movies WHERE library_id = ? AND is_removed = 0`
	case models.ContentTypeSeries:
		query = `SELECT COUNT(*) FROM series WHERE library_id = ? AND is_removed = 0`
	default:
		return 0, nil
	}

	var count int
	if err := r.db.QueryRowContext(ctx, query, libraryID).Scan(&count); err != nil {
		return 0, fmt.Errorf("failed to count media: %w", err)
	}
	return count, nil
}

func (r *MediaLibraryRepository) Update(ctx context.Context, library *models.MediaLibrary) error {
	if library == nil {
		return fmt.Errorf("library cannot be nil")
	}
	library.UpdatedAt = time.Now()

	query := `
		UPDATE media_libraries
		SET name = ?, content_type = ?, auto_detect = ?, sort_order = ?, updated_at = ?
		WHERE id = ?
	`
	result, err := r.db.ExecContext(ctx, query,
		library.Name, library.ContentType, library.AutoDetect,
		library.SortOrder, library.UpdatedAt, library.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update library: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("library with id %s: %w", library.ID, ErrLibraryNotFound)
	}
	return nil
}

func (r *MediaLibraryRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM media_libraries WHERE id = ?`
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete library: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("library with id %s: %w", id, ErrLibraryNotFound)
	}
	return nil
}

// --- Path operations ---

func (r *MediaLibraryRepository) AddPath(ctx context.Context, path *models.MediaLibraryPath) error {
	if path == nil {
		return fmt.Errorf("path cannot be nil")
	}
	if path.ID == "" {
		path.ID = uuid.New().String()
	}
	path.CreatedAt = time.Now()

	query := `
		INSERT INTO media_library_paths (id, library_id, path, status, created_at)
		VALUES (?, ?, ?, ?, ?)
	`
	_, err := r.db.ExecContext(ctx, query,
		path.ID, path.LibraryID, path.Path, path.Status, path.CreatedAt,
	)
	if err != nil {
		if isUniqueConstraintError(err) {
			return fmt.Errorf("path %s: %w", path.Path, ErrLibraryPathDuplicate)
		}
		return fmt.Errorf("failed to add path: %w", err)
	}
	return nil
}

func (r *MediaLibraryRepository) RemovePath(ctx context.Context, pathID string) error {
	query := `DELETE FROM media_library_paths WHERE id = ?`
	result, err := r.db.ExecContext(ctx, query, pathID)
	if err != nil {
		return fmt.Errorf("failed to remove path: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("path with id %s: %w", pathID, ErrLibraryPathNotFound)
	}
	return nil
}

func (r *MediaLibraryRepository) GetPathsByLibraryID(ctx context.Context, libraryID string) ([]models.MediaLibraryPath, error) {
	query := `
		SELECT id, library_id, path, status, last_checked_at, created_at
		FROM media_library_paths WHERE library_id = ? ORDER BY created_at
	`
	rows, err := r.db.QueryContext(ctx, query, libraryID)
	if err != nil {
		return nil, fmt.Errorf("failed to get paths: %w", err)
	}
	defer rows.Close()

	var paths []models.MediaLibraryPath
	for rows.Next() {
		var p models.MediaLibraryPath
		if err := rows.Scan(&p.ID, &p.LibraryID, &p.Path, &p.Status, &p.LastCheckedAt, &p.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan path: %w", err)
		}
		paths = append(paths, p)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating paths: %w", err)
	}
	return paths, nil
}

func (r *MediaLibraryRepository) GetAllPaths(ctx context.Context) ([]models.MediaLibraryPath, error) {
	query := `
		SELECT id, library_id, path, status, last_checked_at, created_at
		FROM media_library_paths ORDER BY created_at
	`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get all paths: %w", err)
	}
	defer rows.Close()

	var paths []models.MediaLibraryPath
	for rows.Next() {
		var p models.MediaLibraryPath
		if err := rows.Scan(&p.ID, &p.LibraryID, &p.Path, &p.Status, &p.LastCheckedAt, &p.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan path: %w", err)
		}
		paths = append(paths, p)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating paths: %w", err)
	}
	return paths, nil
}

func (r *MediaLibraryRepository) UpdatePathStatus(ctx context.Context, pathID string, status models.MediaLibraryPathStatus) error {
	now := time.Now()
	query := `UPDATE media_library_paths SET status = ?, last_checked_at = ? WHERE id = ?`
	result, err := r.db.ExecContext(ctx, query, status, now, pathID)
	if err != nil {
		return fmt.Errorf("failed to update path status: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("path with id %s: %w", pathID, ErrLibraryPathNotFound)
	}
	return nil
}

// isUniqueConstraintError checks if the error is a UNIQUE constraint violation.
func isUniqueConstraintError(err error) bool {
	return err != nil && (errors.Is(err, sql.ErrNoRows) == false) &&
		(contains(err.Error(), "UNIQUE constraint failed") || contains(err.Error(), "unique constraint"))
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
