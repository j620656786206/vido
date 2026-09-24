package services

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
)

// CacheTypeInfo represents size and count information for a single cache type
type CacheTypeInfo struct {
	Type       string `json:"type"`
	Label      string `json:"label"`
	SizeBytes  int64  `json:"size_bytes"`
	EntryCount int64  `json:"entry_count"`
}

// CacheStats represents the complete cache statistics response
type CacheStats struct {
	CacheTypes     []CacheTypeInfo `json:"cache_types"`
	TotalSizeBytes int64           `json:"total_size_bytes"`
}

// CacheStatsServiceInterface defines the contract for cache statistics operations
type CacheStatsServiceInterface interface {
	GetCacheStats(ctx context.Context) (*CacheStats, error)
}

// CacheStatsService provides cache statistics by querying the cache tables.
// (No filesystem "image cache": see ValidCacheTypes.)
type CacheStatsService struct {
	db *sql.DB
}

// NewCacheStatsService creates a new CacheStatsService
func NewCacheStatsService(db *sql.DB) *CacheStatsService {
	return &CacheStatsService{db: db}
}

// GetCacheStats returns cache size and entry count for all cache types
func (s *CacheStatsService) GetCacheStats(ctx context.Context) (*CacheStats, error) {
	stats := &CacheStats{}

	// AI parsing cache
	aiSize, aiCount, err := s.getTableStats(ctx, "ai_cache")
	if err != nil {
		slog.Warn("Failed to get AI cache stats", "error", err)
	}
	stats.CacheTypes = append(stats.CacheTypes, CacheTypeInfo{
		Type:       "ai",
		Label:      "AI 解析快取",
		SizeBytes:  aiSize,
		EntryCount: aiCount,
	})

	// TMDb metadata cache (cache_entries table)
	metadataSize, metadataCount, err := s.getTableStats(ctx, "cache_entries")
	if err != nil {
		slog.Warn("Failed to get metadata cache stats", "error", err)
	}
	stats.CacheTypes = append(stats.CacheTypes, CacheTypeInfo{
		Type:       "metadata",
		Label:      "TMDb 中繼資料",
		SizeBytes:  metadataSize,
		EntryCount: metadataCount,
	})

	// Douban cache
	doubanSize, doubanCount, err := s.getTableStats(ctx, "douban_cache")
	if err != nil {
		slog.Warn("Failed to get Douban cache stats", "error", err)
	}
	stats.CacheTypes = append(stats.CacheTypes, CacheTypeInfo{
		Type:       "douban",
		Label:      "豆瓣快取",
		SizeBytes:  doubanSize,
		EntryCount: doubanCount,
	})

	// Wikipedia cache
	wikiSize, wikiCount, err := s.getTableStats(ctx, "wikipedia_cache")
	if err != nil {
		slog.Warn("Failed to get Wikipedia cache stats", "error", err)
	}
	stats.CacheTypes = append(stats.CacheTypes, CacheTypeInfo{
		Type:       "wikipedia",
		Label:      "維基百科快取",
		SizeBytes:  wikiSize,
		EntryCount: wikiCount,
	})

	// Calculate total
	for _, ct := range stats.CacheTypes {
		stats.TotalSizeBytes += ct.SizeBytes
	}

	return stats, nil
}

// getTableStats returns the estimated size and row count for a given table.
// Uses SQLite dbstat virtual table for size; falls back to a rough estimate if unavailable.
func (s *CacheStatsService) getTableStats(ctx context.Context, tableName string) (sizeBytes int64, count int64, err error) {
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM %s", tableName)
	if err := s.db.QueryRowContext(ctx, countQuery).Scan(&count); err != nil {
		return 0, 0, fmt.Errorf("count %s: %w", tableName, err)
	}

	if count == 0 {
		return 0, 0, nil
	}

	// Try dbstat virtual table for accurate per-table size
	sizeQuery := fmt.Sprintf("SELECT COALESCE(SUM(pgsize), 0) FROM dbstat WHERE name = '%s'", tableName)
	if err := s.db.QueryRowContext(ctx, sizeQuery).Scan(&sizeBytes); err != nil {
		slog.Debug("dbstat not available, using rough estimate", "table", tableName, "error", err)
		sizeBytes = count * 512
	}

	return sizeBytes, count, nil
}

// Compile-time interface verification
var _ CacheStatsServiceInterface = (*CacheStatsService)(nil)
