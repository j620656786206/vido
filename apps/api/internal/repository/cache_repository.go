package repository

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"
)

// CacheRepository provides data access operations for cache entries.
// It implements CacheRepositoryInterface for TTL-based caching with SQLite persistence.
type CacheRepository struct {
	db *sql.DB
}

// NewCacheRepository creates a new instance of CacheRepository.
func NewCacheRepository(db *sql.DB) *CacheRepository {
	return &CacheRepository{
		db: db,
	}
}

// Get retrieves a cache entry by key, returns nil if not found or expired.
func (r *CacheRepository) Get(ctx context.Context, key string) (*CacheEntry, error) {
	if key == "" {
		return nil, fmt.Errorf("cache key cannot be empty")
	}

	query := `
		SELECT key, value, type, expires_at, created_at, updated_at
		FROM cache_entries
		WHERE key = ? AND expires_at > datetime('now')
	`

	entry := &CacheEntry{}
	err := r.db.QueryRowContext(ctx, query, key).Scan(
		&entry.Key,
		&entry.Value,
		&entry.Type,
		&entry.ExpiresAt,
		&entry.CreatedAt,
		&entry.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil // Not found or expired
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get cache entry: %w", err)
	}

	return entry, nil
}

// Set creates or updates a cache entry with the specified TTL.
func (r *CacheRepository) Set(ctx context.Context, key string, value string, cacheType string, ttl time.Duration) error {
	if key == "" {
		return fmt.Errorf("cache key cannot be empty")
	}

	if ttl <= 0 {
		return fmt.Errorf("cache TTL must be positive")
	}

	now := time.Now()
	expiresAt := now.Add(ttl)

	query := `
		INSERT INTO cache_entries (key, value, type, expires_at, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(key) DO UPDATE SET
			value = excluded.value,
			type = excluded.type,
			expires_at = excluded.expires_at,
			updated_at = excluded.updated_at
	`

	_, err := r.db.ExecContext(ctx, query, key, value, cacheType, expiresAt, now, now)
	if err != nil {
		return fmt.Errorf("failed to set cache entry: %w", err)
	}

	slog.Debug("Cache entry set", "key", key, "type", cacheType, "ttl", ttl)
	return nil
}

// Delete removes a cache entry by key.
func (r *CacheRepository) Delete(ctx context.Context, key string) error {
	if key == "" {
		return fmt.Errorf("cache key cannot be empty")
	}

	query := `DELETE FROM cache_entries WHERE key = ?`

	result, err := r.db.ExecContext(ctx, query, key)
	if err != nil {
		return fmt.Errorf("failed to delete cache entry: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	slog.Debug("Cache entry deleted", "key", key, "deleted", rowsAffected > 0)

	return nil
}

// Clear removes all cache entries.
func (r *CacheRepository) Clear(ctx context.Context) error {
	query := `DELETE FROM cache_entries`

	result, err := r.db.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to clear cache: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	slog.Info("Cache cleared", "entries_deleted", rowsAffected)

	return nil
}

// ClearExpired removes all expired cache entries.
func (r *CacheRepository) ClearExpired(ctx context.Context) (int64, error) {
	query := `DELETE FROM cache_entries WHERE expires_at <= datetime('now')`

	result, err := r.db.ExecContext(ctx, query)
	if err != nil {
		return 0, fmt.Errorf("failed to clear expired cache entries: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected > 0 {
		slog.Info("Expired cache entries cleared", "entries_deleted", rowsAffected)
	}

	return rowsAffected, nil
}

// ClearByType removes all cache entries of a specific type.
func (r *CacheRepository) ClearByType(ctx context.Context, cacheType string) (int64, error) {
	if cacheType == "" {
		return 0, fmt.Errorf("cache type cannot be empty")
	}

	query := `DELETE FROM cache_entries WHERE type = ?`

	result, err := r.db.ExecContext(ctx, query, cacheType)
	if err != nil {
		return 0, fmt.Errorf("failed to clear cache entries by type: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	slog.Info("Cache entries cleared by type", "type", cacheType, "entries_deleted", rowsAffected)

	return rowsAffected, nil
}

// Compile-time interface verification
var _ CacheRepositoryInterface = (*CacheRepository)(nil)
