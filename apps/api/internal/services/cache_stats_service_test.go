package services

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupCacheTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)

	// Create cache tables matching production schema
	_, err = db.Exec(`
		CREATE TABLE cache_entries (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL,
			type TEXT NOT NULL,
			expires_at TIMESTAMP NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
		CREATE TABLE ai_cache (
			id TEXT PRIMARY KEY,
			filename_hash TEXT UNIQUE NOT NULL,
			provider TEXT NOT NULL,
			request_prompt TEXT NOT NULL,
			response_json TEXT NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			expires_at TIMESTAMP NOT NULL
		);
		CREATE TABLE douban_cache (
			id TEXT PRIMARY KEY,
			douban_id TEXT UNIQUE NOT NULL,
			title TEXT NOT NULL,
			title_traditional TEXT,
			original_title TEXT,
			year INTEGER,
			rating REAL,
			rating_count INTEGER,
			director TEXT,
			cast_json TEXT,
			genres_json TEXT,
			countries_json TEXT,
			languages_json TEXT,
			poster_url TEXT,
			summary TEXT,
			summary_traditional TEXT,
			media_type TEXT DEFAULT 'movie',
			runtime INTEGER,
			episodes INTEGER,
			release_date TEXT,
			imdb_id TEXT,
			scraped_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			expires_at TIMESTAMP NOT NULL
		);
		CREATE TABLE wikipedia_cache (
			id TEXT PRIMARY KEY,
			query TEXT NOT NULL,
			page_title TEXT NOT NULL,
			title TEXT NOT NULL,
			original_title TEXT,
			year INTEGER,
			director TEXT,
			cast_json TEXT,
			genres_json TEXT,
			summary TEXT,
			image_url TEXT,
			media_type TEXT DEFAULT 'movie',
			confidence REAL DEFAULT 0.5,
			fetched_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			expires_at TIMESTAMP NOT NULL
		);
	`)
	require.NoError(t, err)

	t.Cleanup(func() { db.Close() })
	return db
}

func TestCacheStatsService_GetCacheStats_EmptyTables(t *testing.T) {
	db := setupCacheTestDB(t)
	svc := NewCacheStatsService(db, "")

	stats, err := svc.GetCacheStats(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, stats)
	assert.Len(t, stats.CacheTypes, 5)
	assert.Equal(t, int64(0), stats.TotalSizeBytes)

	// Verify all types present
	types := make(map[string]bool)
	for _, ct := range stats.CacheTypes {
		types[ct.Type] = true
		assert.Equal(t, int64(0), ct.EntryCount)
	}
	assert.True(t, types["image"])
	assert.True(t, types["ai"])
	assert.True(t, types["metadata"])
	assert.True(t, types["douban"])
	assert.True(t, types["wikipedia"])
}

func TestCacheStatsService_GetCacheStats_WithData(t *testing.T) {
	db := setupCacheTestDB(t)
	svc := NewCacheStatsService(db, "")

	// Insert test data
	_, err := db.Exec(`INSERT INTO cache_entries (key, value, type, expires_at) VALUES ('k1', 'v1', 'tmdb', '2099-01-01')`)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO cache_entries (key, value, type, expires_at) VALUES ('k2', 'v2', 'tmdb', '2099-01-01')`)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO ai_cache (id, filename_hash, provider, request_prompt, response_json, expires_at) VALUES ('a1', 'h1', 'openai', 'prompt', '{}', '2099-01-01')`)
	require.NoError(t, err)

	stats, err := svc.GetCacheStats(context.Background())
	require.NoError(t, err)

	// Find metadata and ai cache types
	for _, ct := range stats.CacheTypes {
		switch ct.Type {
		case "metadata":
			assert.Equal(t, int64(2), ct.EntryCount)
		case "ai":
			assert.Equal(t, int64(1), ct.EntryCount)
		case "douban":
			assert.Equal(t, int64(0), ct.EntryCount)
		case "wikipedia":
			assert.Equal(t, int64(0), ct.EntryCount)
		}
	}
}

func TestCacheStatsService_GetCacheStats_Labels(t *testing.T) {
	db := setupCacheTestDB(t)
	svc := NewCacheStatsService(db, "")

	stats, err := svc.GetCacheStats(context.Background())
	require.NoError(t, err)

	expectedLabels := map[string]string{
		"image":     "圖片快取",
		"ai":        "AI 解析快取",
		"metadata":  "TMDb 中繼資料",
		"douban":    "豆瓣快取",
		"wikipedia": "維基百科快取",
	}

	for _, ct := range stats.CacheTypes {
		assert.Equal(t, expectedLabels[ct.Type], ct.Label, "label mismatch for type %s", ct.Type)
	}
}

func TestCacheStatsService_GetImageCacheSize_EmptyDir(t *testing.T) {
	svc := NewCacheStatsService(nil, "")
	size, err := svc.GetImageCacheSize(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, int64(0), size)
}

func TestCacheStatsService_GetImageCacheSize_NonExistentDir(t *testing.T) {
	svc := NewCacheStatsService(nil, "/nonexistent/path")
	size, err := svc.GetImageCacheSize(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, int64(0), size)
}

func TestCacheStatsService_GetImageCacheSize_WithFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test image files
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "img1.jpg"), make([]byte, 1024), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "img2.png"), make([]byte, 2048), 0644))

	// Create subdirectory with file
	subDir := filepath.Join(tmpDir, "sub")
	require.NoError(t, os.Mkdir(subDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(subDir, "img3.jpg"), make([]byte, 512), 0644))

	svc := NewCacheStatsService(nil, tmpDir)
	size, err := svc.GetImageCacheSize(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, int64(3584), size) // 1024 + 2048 + 512
}

func TestCacheStatsService_GetCacheStats_TotalSizeCalculation(t *testing.T) {
	db := setupCacheTestDB(t)

	tmpDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "img.jpg"), make([]byte, 100), 0644))

	svc := NewCacheStatsService(db, tmpDir)

	// Add some DB entries so DB caches have non-zero size
	_, err := db.Exec(`INSERT INTO ai_cache (id, filename_hash, provider, request_prompt, response_json, expires_at) VALUES ('a1', 'h1', 'openai', 'prompt', '{}', '2099-01-01')`)
	require.NoError(t, err)

	stats, err := svc.GetCacheStats(context.Background())
	require.NoError(t, err)

	// Total should be sum of all cache types
	var expectedTotal int64
	for _, ct := range stats.CacheTypes {
		expectedTotal += ct.SizeBytes
	}
	assert.Equal(t, expectedTotal, stats.TotalSizeBytes)
	assert.Greater(t, stats.TotalSizeBytes, int64(0))
}

func TestCacheStatsService_ImageCacheWithDBStats(t *testing.T) {
	db := setupCacheTestDB(t)
	tmpDir := t.TempDir()

	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "a.jpg"), make([]byte, 10), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "b.jpg"), make([]byte, 10), 0644))

	svc := NewCacheStatsService(db, tmpDir)
	stats, err := svc.GetCacheStats(context.Background())
	require.NoError(t, err)

	// Find image cache type
	for _, ct := range stats.CacheTypes {
		if ct.Type == "image" {
			assert.Equal(t, int64(2), ct.EntryCount)
			assert.Equal(t, int64(20), ct.SizeBytes)
			return
		}
	}
	t.Fatal("image cache type not found")
}

func TestCacheStatsService_InterfaceCompliance(t *testing.T) {
	var _ CacheStatsServiceInterface = (*CacheStatsService)(nil)
}
