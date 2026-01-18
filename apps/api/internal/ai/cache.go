package ai

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
)

const (
	// DefaultCacheTTL is the default cache TTL of 30 days per NFR-I10.
	DefaultCacheTTL = 30 * 24 * time.Hour
)

// CacheEntry represents a cached AI parsing result.
type CacheEntry struct {
	ID           string
	FilenameHash string
	Provider     string
	RequestPrompt string
	ResponseJSON string
	CreatedAt    time.Time
	ExpiresAt    time.Time
}

// Cache provides caching functionality for AI parsing results.
type Cache struct {
	db  *sql.DB
	ttl time.Duration
}

// CacheOption is a functional option for configuring Cache.
type CacheOption func(*Cache)

// WithCacheTTL sets a custom TTL for cache entries.
func WithCacheTTL(ttl time.Duration) CacheOption {
	return func(c *Cache) {
		c.ttl = ttl
	}
}

// NewCache creates a new AI cache.
func NewCache(db *sql.DB, opts ...CacheOption) *Cache {
	c := &Cache{
		db:  db,
		ttl: DefaultCacheTTL,
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

// HashFilename generates a SHA-256 hash of the filename for use as cache key.
func HashFilename(filename string) string {
	hash := sha256.Sum256([]byte(filename))
	return hex.EncodeToString(hash[:])
}

// Get retrieves a cached parsing result by filename.
// Returns nil if not found or expired.
func (c *Cache) Get(ctx context.Context, filename string) (*ParseResponse, error) {
	hash := HashFilename(filename)

	query := `
		SELECT response_json, expires_at
		FROM ai_cache
		WHERE filename_hash = ?
		LIMIT 1
	`

	var responseJSON string
	var expiresAtStr string

	err := c.db.QueryRowContext(ctx, query, hash).Scan(&responseJSON, &expiresAtStr)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil // Cache miss
		}
		return nil, fmt.Errorf("failed to query cache: %w", err)
	}

	// Parse expires_at
	expiresAt, err := time.Parse(time.RFC3339, expiresAtStr)
	if err != nil {
		slog.Error("Failed to parse expires_at",
			"error", err,
			"expires_at", expiresAtStr,
		)
		return nil, fmt.Errorf("failed to parse expires_at: %w", err)
	}

	// Check if expired
	if time.Now().After(expiresAt) {
		slog.Debug("Cache entry expired",
			"filename_hash", hash[:16]+"...",
			"expired_at", expiresAt,
		)
		// Delete expired entry
		go c.deleteExpired(context.Background(), hash)
		return nil, nil
	}

	// Parse response
	var response ParseResponse
	if err := json.Unmarshal([]byte(responseJSON), &response); err != nil {
		slog.Error("Failed to unmarshal cached response",
			"error", err,
			"filename_hash", hash[:16]+"...",
		)
		return nil, fmt.Errorf("failed to unmarshal cached response: %w", err)
	}

	slog.Debug("Cache hit",
		"filename_hash", hash[:16]+"...",
		"title", response.Title,
	)

	return &response, nil
}

// Set stores a parsing result in the cache.
func (c *Cache) Set(ctx context.Context, filename string, provider ProviderName, prompt string, response *ParseResponse) error {
	hash := HashFilename(filename)
	id := uuid.New().String()

	responseJSON, err := json.Marshal(response)
	if err != nil {
		return fmt.Errorf("failed to marshal response: %w", err)
	}

	expiresAt := time.Now().Add(c.ttl).Format(time.RFC3339)
	createdAt := time.Now().Format(time.RFC3339)

	query := `
		INSERT INTO ai_cache (id, filename_hash, provider, request_prompt, response_json, created_at, expires_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(filename_hash) DO UPDATE SET
			provider = excluded.provider,
			request_prompt = excluded.request_prompt,
			response_json = excluded.response_json,
			expires_at = excluded.expires_at
	`

	_, err = c.db.ExecContext(ctx, query, id, hash, string(provider), prompt, string(responseJSON), createdAt, expiresAt)
	if err != nil {
		return fmt.Errorf("failed to insert cache entry: %w", err)
	}

	slog.Debug("Cached AI response",
		"filename_hash", hash[:16]+"...",
		"provider", provider,
		"expires_at", expiresAt,
	)

	return nil
}

// Delete removes a cache entry by filename.
func (c *Cache) Delete(ctx context.Context, filename string) error {
	hash := HashFilename(filename)
	return c.deleteExpired(ctx, hash)
}

// deleteExpired deletes a specific cache entry by hash.
func (c *Cache) deleteExpired(ctx context.Context, hash string) error {
	query := `DELETE FROM ai_cache WHERE filename_hash = ?`
	_, err := c.db.ExecContext(ctx, query, hash)
	if err != nil {
		slog.Error("Failed to delete cache entry",
			"error", err,
			"filename_hash", hash[:16]+"...",
		)
		return err
	}
	return nil
}

// ClearExpired removes all expired cache entries.
func (c *Cache) ClearExpired(ctx context.Context) (int64, error) {
	now := time.Now().Format(time.RFC3339)
	query := `DELETE FROM ai_cache WHERE expires_at < ?`

	result, err := c.db.ExecContext(ctx, query, now)
	if err != nil {
		return 0, fmt.Errorf("failed to clear expired entries: %w", err)
	}

	count, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get affected rows: %w", err)
	}

	if count > 0 {
		slog.Info("Cleared expired AI cache entries",
			"count", count,
		)
	}

	return count, nil
}

// ClearAll removes all cache entries.
func (c *Cache) ClearAll(ctx context.Context) (int64, error) {
	query := `DELETE FROM ai_cache`

	result, err := c.db.ExecContext(ctx, query)
	if err != nil {
		return 0, fmt.Errorf("failed to clear cache: %w", err)
	}

	count, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get affected rows: %w", err)
	}

	slog.Info("Cleared all AI cache entries",
		"count", count,
	)

	return count, nil
}

// Stats returns cache statistics.
func (c *Cache) Stats(ctx context.Context) (*CacheStats, error) {
	now := time.Now().Format(time.RFC3339)
	query := `
		SELECT
			COUNT(*) as total,
			COALESCE(SUM(CASE WHEN expires_at > ? THEN 1 ELSE 0 END), 0) as valid,
			COALESCE(SUM(CASE WHEN expires_at <= ? THEN 1 ELSE 0 END), 0) as expired
		FROM ai_cache
	`

	var stats CacheStats
	err := c.db.QueryRowContext(ctx, query, now, now).Scan(&stats.TotalEntries, &stats.ValidEntries, &stats.ExpiredEntries)
	if err != nil {
		return nil, fmt.Errorf("failed to get cache stats: %w", err)
	}

	return &stats, nil
}

// CacheStats contains cache statistics.
type CacheStats struct {
	TotalEntries   int64
	ValidEntries   int64
	ExpiredEntries int64
}
