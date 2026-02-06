package cache

import (
	"context"
	"database/sql"
	"log/slog"
	"time"
)

// OfflineCache provides cached data access when external services are unavailable.
// It uses SQLite for persistent storage and keeps stale data available for offline mode.
type OfflineCache struct {
	db     *sql.DB
	logger *slog.Logger
}

// NewOfflineCache creates a new OfflineCache with the given database connection.
func NewOfflineCache(db *sql.DB) *OfflineCache {
	return &OfflineCache{
		db:     db,
		logger: slog.Default(),
	}
}

// InitSchema creates the cache table if it doesn't exist.
func (c *OfflineCache) InitSchema(ctx context.Context) error {
	_, err := c.db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS offline_cache (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL,
			data_type TEXT NOT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			accessed_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			expires_at DATETIME,
			is_stale BOOLEAN NOT NULL DEFAULT 0
		)
	`)
	if err != nil {
		return err
	}

	// Create index for faster expiration queries
	_, err = c.db.ExecContext(ctx, `
		CREATE INDEX IF NOT EXISTS idx_offline_cache_expires ON offline_cache(expires_at)
	`)
	return err
}

// Set stores a value in the cache with the given TTL.
func (c *OfflineCache) Set(ctx context.Context, key, value, dataType string, ttl time.Duration) error {
	expiresAt := time.Now().Add(ttl)

	_, err := c.db.ExecContext(ctx, `
		INSERT INTO offline_cache (key, value, data_type, created_at, accessed_at, expires_at, is_stale)
		VALUES (?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, ?, 0)
		ON CONFLICT(key) DO UPDATE SET
			value = excluded.value,
			data_type = excluded.data_type,
			accessed_at = CURRENT_TIMESTAMP,
			expires_at = excluded.expires_at,
			is_stale = 0
	`, key, value, dataType, expiresAt)

	if err != nil {
		c.logger.Error("Failed to set cache",
			"key", key,
			"error", err,
		)
		return err
	}

	c.logger.Debug("Cache set",
		"key", key,
		"data_type", dataType,
		"ttl", ttl,
	)

	return nil
}

// Get retrieves a value from the cache.
// Returns the value, whether it's stale, and any error.
// Stale data is still returned (for offline mode) but flagged as stale.
func (c *OfflineCache) Get(ctx context.Context, key string) (string, bool, error) {
	var value string
	var isStale bool
	var expiresAt sql.NullTime

	err := c.db.QueryRowContext(ctx, `
		SELECT value, is_stale, expires_at FROM offline_cache WHERE key = ?
	`, key).Scan(&value, &isStale, &expiresAt)

	if err == sql.ErrNoRows {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}

	// Check if expired (but still return the data marked as stale)
	if expiresAt.Valid && time.Now().After(expiresAt.Time) {
		isStale = true
	}

	// Update accessed_at
	go func() {
		_, _ = c.db.Exec(`
			UPDATE offline_cache SET accessed_at = CURRENT_TIMESTAMP WHERE key = ?
		`, key)
	}()

	return value, isStale, nil
}

// Delete removes a value from the cache.
func (c *OfflineCache) Delete(ctx context.Context, key string) error {
	_, err := c.db.ExecContext(ctx, `DELETE FROM offline_cache WHERE key = ?`, key)
	return err
}

// MarkStale marks a cache entry as stale.
// The data is kept but flagged for refresh when connectivity is restored.
func (c *OfflineCache) MarkStale(ctx context.Context, key string) error {
	_, err := c.db.ExecContext(ctx, `
		UPDATE offline_cache SET is_stale = 1 WHERE key = ?
	`, key)
	return err
}

// MarkAllStale marks all cache entries for a data type as stale.
func (c *OfflineCache) MarkAllStale(ctx context.Context, dataType string) error {
	_, err := c.db.ExecContext(ctx, `
		UPDATE offline_cache SET is_stale = 1 WHERE data_type = ?
	`, dataType)
	return err
}

// GetStaleKeys returns all keys that are marked as stale.
// Used for cache invalidation when services recover.
func (c *OfflineCache) GetStaleKeys(ctx context.Context) ([]string, error) {
	rows, err := c.db.QueryContext(ctx, `
		SELECT key FROM offline_cache WHERE is_stale = 1
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var keys []string
	for rows.Next() {
		var key string
		if err := rows.Scan(&key); err != nil {
			return nil, err
		}
		keys = append(keys, key)
	}

	return keys, rows.Err()
}

// ClearOldEntries removes cache entries older than the specified duration.
func (c *OfflineCache) ClearOldEntries(ctx context.Context, maxAge time.Duration) (int64, error) {
	cutoff := time.Now().Add(-maxAge)

	result, err := c.db.ExecContext(ctx, `
		DELETE FROM offline_cache WHERE created_at < ?
	`, cutoff)
	if err != nil {
		return 0, err
	}

	return result.RowsAffected()
}

// ClearExpired removes all expired cache entries.
// Unlike GetStaleKeys, this completely removes the data.
func (c *OfflineCache) ClearExpired(ctx context.Context) (int64, error) {
	result, err := c.db.ExecContext(ctx, `
		DELETE FROM offline_cache WHERE expires_at < CURRENT_TIMESTAMP AND is_stale = 1
	`)
	if err != nil {
		return 0, err
	}

	return result.RowsAffected()
}

// Refresh updates a stale entry with fresh data and clears the stale flag.
func (c *OfflineCache) Refresh(ctx context.Context, key, value string, ttl time.Duration) error {
	expiresAt := time.Now().Add(ttl)

	_, err := c.db.ExecContext(ctx, `
		UPDATE offline_cache
		SET value = ?, accessed_at = CURRENT_TIMESTAMP, expires_at = ?, is_stale = 0
		WHERE key = ?
	`, value, expiresAt, key)

	return err
}

// Stats returns cache statistics.
type CacheStats struct {
	TotalEntries  int64 `json:"totalEntries"`
	StaleEntries  int64 `json:"staleEntries"`
	ExpiredCount  int64 `json:"expiredCount"`
	TotalSizeKB   int64 `json:"totalSizeKb"`
}

// GetStats returns cache statistics.
func (c *OfflineCache) GetStats(ctx context.Context) (*CacheStats, error) {
	stats := &CacheStats{}

	// Get total entries
	err := c.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM offline_cache`).Scan(&stats.TotalEntries)
	if err != nil {
		return nil, err
	}

	// Get stale entries
	err = c.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM offline_cache WHERE is_stale = 1`).Scan(&stats.StaleEntries)
	if err != nil {
		return nil, err
	}

	// Get expired entries
	err = c.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM offline_cache WHERE expires_at < CURRENT_TIMESTAMP
	`).Scan(&stats.ExpiredCount)
	if err != nil {
		return nil, err
	}

	// Get approximate total size
	err = c.db.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(LENGTH(value)), 0) / 1024 FROM offline_cache
	`).Scan(&stats.TotalSizeKB)
	if err != nil {
		return nil, err
	}

	return stats, nil
}
