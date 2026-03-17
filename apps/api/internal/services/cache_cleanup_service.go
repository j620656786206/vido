package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"
)

// ErrInvalidCacheType indicates an unknown cache type was provided
var ErrInvalidCacheType = errors.New("invalid cache type")

// CleanupResult represents the outcome of a cache cleanup operation
type CleanupResult struct {
	Type           string `json:"type"`
	EntriesRemoved int64  `json:"entriesRemoved"`
	BytesReclaimed int64  `json:"bytesReclaimed"`
}

// ValidCacheTypes lists all recognized cache type keys
var ValidCacheTypes = []string{"image", "ai", "metadata", "douban", "wikipedia"}

// CacheCleanupServiceInterface defines the contract for cache cleanup operations
type CacheCleanupServiceInterface interface {
	ClearCacheByAge(ctx context.Context, days int) (*CleanupResult, error)
	ClearCacheByType(ctx context.Context, cacheType string) (*CleanupResult, error)
}

// CacheCleanupService handles cache cleanup operations across all cache types
type CacheCleanupService struct {
	db       *sql.DB
	imageDir string
}

// NewCacheCleanupService creates a new CacheCleanupService
func NewCacheCleanupService(db *sql.DB, imageDir string) *CacheCleanupService {
	return &CacheCleanupService{
		db:       db,
		imageDir: imageDir,
	}
}

// ClearCacheByAge removes cache entries older than the specified number of days
func (s *CacheCleanupService) ClearCacheByAge(ctx context.Context, days int) (*CleanupResult, error) {
	if days <= 0 {
		return nil, fmt.Errorf("days must be positive")
	}

	cutoff := time.Now().AddDate(0, 0, -days)
	result := &CleanupResult{Type: "all"}

	// Clear old entries from each DB table
	tables := []struct {
		name      string
		dateCol   string
	}{
		{"cache_entries", "created_at"},
		{"ai_cache", "created_at"},
		{"douban_cache", "scraped_at"},
		{"wikipedia_cache", "fetched_at"},
	}

	for _, tbl := range tables {
		removed, err := s.deleteOlderThan(ctx, tbl.name, tbl.dateCol, cutoff)
		if err != nil {
			slog.Warn("Failed to clear old entries from table", "table", tbl.name, "error", err)
			continue
		}
		result.EntriesRemoved += removed
	}

	// Clear old image files
	imageRemoved, imageBytes, err := s.clearOldImages(cutoff)
	if err != nil {
		slog.Warn("Failed to clear old images", "error", err)
	} else {
		result.EntriesRemoved += imageRemoved
		result.BytesReclaimed += imageBytes
	}

	slog.Info("Cache cleared by age", "days", days, "entries_removed", result.EntriesRemoved, "bytes_reclaimed", result.BytesReclaimed)
	return result, nil
}

// ClearCacheByType removes all entries of a specific cache type
func (s *CacheCleanupService) ClearCacheByType(ctx context.Context, cacheType string) (*CleanupResult, error) {
	if !isValidCacheType(cacheType) {
		return nil, ErrInvalidCacheType
	}

	result := &CleanupResult{Type: cacheType}

	switch cacheType {
	case "image":
		removed, bytes, err := s.clearAllImages()
		if err != nil {
			return nil, fmt.Errorf("clear image cache: %w", err)
		}
		result.EntriesRemoved = removed
		result.BytesReclaimed = bytes

	case "ai":
		removed, err := s.clearTable(ctx, "ai_cache")
		if err != nil {
			return nil, fmt.Errorf("clear ai cache: %w", err)
		}
		result.EntriesRemoved = removed

	case "metadata":
		removed, err := s.clearTable(ctx, "cache_entries")
		if err != nil {
			return nil, fmt.Errorf("clear metadata cache: %w", err)
		}
		result.EntriesRemoved = removed

	case "douban":
		removed, err := s.clearTable(ctx, "douban_cache")
		if err != nil {
			return nil, fmt.Errorf("clear douban cache: %w", err)
		}
		result.EntriesRemoved = removed

	case "wikipedia":
		removed, err := s.clearTable(ctx, "wikipedia_cache")
		if err != nil {
			return nil, fmt.Errorf("clear wikipedia cache: %w", err)
		}
		result.EntriesRemoved = removed
	}

	slog.Info("Cache cleared by type", "type", cacheType, "entries_removed", result.EntriesRemoved, "bytes_reclaimed", result.BytesReclaimed)
	return result, nil
}

func (s *CacheCleanupService) deleteOlderThan(ctx context.Context, table, dateCol string, cutoff time.Time) (int64, error) {
	query := fmt.Sprintf("DELETE FROM %s WHERE %s < ?", table, dateCol)
	result, err := s.db.ExecContext(ctx, query, cutoff)
	if err != nil {
		return 0, fmt.Errorf("delete from %s: %w", table, err)
	}
	rows, _ := result.RowsAffected()
	return rows, nil
}

func (s *CacheCleanupService) clearTable(ctx context.Context, table string) (int64, error) {
	result, err := s.db.ExecContext(ctx, fmt.Sprintf("DELETE FROM %s", table))
	if err != nil {
		return 0, fmt.Errorf("clear %s: %w", table, err)
	}
	rows, _ := result.RowsAffected()
	return rows, nil
}

func (s *CacheCleanupService) clearOldImages(cutoff time.Time) (removed int64, bytes int64, err error) {
	if s.imageDir == "" {
		return 0, 0, nil
	}

	err = filepath.Walk(s.imageDir, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if info.IsDir() {
			return nil
		}
		if info.ModTime().Before(cutoff) {
			size := info.Size()
			if rmErr := os.Remove(path); rmErr != nil {
				slog.Warn("Failed to remove old image", "path", path, "error", rmErr)
				return nil
			}
			removed++
			bytes += size
		}
		return nil
	})
	if err != nil && os.IsNotExist(err) {
		return 0, 0, nil
	}
	return removed, bytes, err
}

func (s *CacheCleanupService) clearAllImages() (removed int64, bytes int64, err error) {
	if s.imageDir == "" {
		return 0, 0, nil
	}

	err = filepath.Walk(s.imageDir, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if info.IsDir() {
			return nil
		}
		size := info.Size()
		if rmErr := os.Remove(path); rmErr != nil {
			slog.Warn("Failed to remove image", "path", path, "error", rmErr)
			return nil
		}
		removed++
		bytes += size
		return nil
	})
	if err != nil && os.IsNotExist(err) {
		return 0, 0, nil
	}
	return removed, bytes, err
}

func isValidCacheType(t string) bool {
	for _, v := range ValidCacheTypes {
		if v == t {
			return true
		}
	}
	return false
}

// Compile-time interface verification
var _ CacheCleanupServiceInterface = (*CacheCleanupService)(nil)
