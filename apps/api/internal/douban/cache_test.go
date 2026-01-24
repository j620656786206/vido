package douban

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func setupTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)

	// Create the douban_cache table
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS douban_cache (
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
		CREATE INDEX IF NOT EXISTS idx_douban_cache_douban_id ON douban_cache(douban_id);
		CREATE INDEX IF NOT EXISTS idx_douban_cache_title ON douban_cache(title);
		CREATE INDEX IF NOT EXISTS idx_douban_cache_expires_at ON douban_cache(expires_at);
	`)
	require.NoError(t, err)

	return db
}

func TestNewCache(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	config := DefaultCacheConfig()
	config.CleanupInterval = 0 // Disable cleanup for tests
	cache := NewCache(db, config, nil)

	assert.NotNil(t, cache)
	cache.Close()
}

func TestCache_SetAndGet(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	config := DefaultCacheConfig()
	config.CleanupInterval = 0
	cache := NewCache(db, config, nil)
	defer cache.Close()

	ctx := context.Background()

	// Create a test result
	result := &DetailResult{
		ID:                 "27010768",
		Title:              "寄生虫",
		TitleTraditional:   "寄生蟲",
		OriginalTitle:      "Parasite",
		Year:               2019,
		Rating:             8.7,
		RatingCount:        1234567,
		Director:           "奉俊昊",
		Cast:               []string{"宋康昊", "李善均"},
		Genres:             []string{"剧情", "喜剧"},
		Countries:          []string{"韩国"},
		Languages:          []string{"韩语"},
		PosterURL:          "https://example.com/poster.jpg",
		Summary:            "基泽一家...",
		SummaryTraditional: "基澤一家...",
		Type:               MediaTypeMovie,
		Runtime:            132,
		ReleaseDate:        "2019-05-21",
		IMDbID:             "tt6751668",
		ScrapedAt:          time.Now(),
	}

	// Set the cache
	err := cache.Set(ctx, result)
	require.NoError(t, err)

	// Get from cache
	cached, err := cache.Get(ctx, "27010768")
	require.NoError(t, err)
	require.NotNil(t, cached)

	// Verify all fields
	assert.Equal(t, "27010768", cached.ID)
	assert.Equal(t, "寄生虫", cached.Title)
	assert.Equal(t, "寄生蟲", cached.TitleTraditional)
	assert.Equal(t, "Parasite", cached.OriginalTitle)
	assert.Equal(t, 2019, cached.Year)
	assert.Equal(t, 8.7, cached.Rating)
	assert.Equal(t, 1234567, cached.RatingCount)
	assert.Equal(t, "奉俊昊", cached.Director)
	assert.Equal(t, []string{"宋康昊", "李善均"}, cached.Cast)
	assert.Equal(t, []string{"剧情", "喜剧"}, cached.Genres)
	assert.Equal(t, []string{"韩国"}, cached.Countries)
	assert.Equal(t, []string{"韩语"}, cached.Languages)
	assert.Equal(t, "https://example.com/poster.jpg", cached.PosterURL)
	assert.Equal(t, "基泽一家...", cached.Summary)
	assert.Equal(t, "基澤一家...", cached.SummaryTraditional)
	assert.Equal(t, MediaTypeMovie, cached.Type)
	assert.Equal(t, 132, cached.Runtime)
	assert.Equal(t, "2019-05-21", cached.ReleaseDate)
	assert.Equal(t, "tt6751668", cached.IMDbID)
}

func TestCache_GetMiss(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	config := DefaultCacheConfig()
	config.CleanupInterval = 0
	cache := NewCache(db, config, nil)
	defer cache.Close()

	ctx := context.Background()

	// Get non-existent entry
	cached, err := cache.Get(ctx, "nonexistent")
	require.NoError(t, err)
	assert.Nil(t, cached)
}

func TestCache_GetExpired(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	config := DefaultCacheConfig()
	config.CleanupInterval = 0
	cache := NewCache(db, config, nil)
	defer cache.Close()

	ctx := context.Background()

	// Insert an expired entry directly with a past expiration time
	_, err := db.Exec(`
		INSERT INTO douban_cache (id, douban_id, title, media_type, scraped_at, expires_at)
		VALUES ('test-id', '12345', 'Test Movie', 'movie', datetime('now'), datetime('now', '-1 hour'))
	`)
	require.NoError(t, err)

	// Should not return expired entry
	cached, err := cache.Get(ctx, "12345")
	require.NoError(t, err)
	assert.Nil(t, cached)
}

func TestCache_GetByTitle(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	config := DefaultCacheConfig()
	config.CleanupInterval = 0
	cache := NewCache(db, config, nil)
	defer cache.Close()

	ctx := context.Background()

	// Add multiple entries
	entries := []DetailResult{
		{ID: "1", Title: "寄生虫", Type: MediaTypeMovie, Rating: 8.7, ScrapedAt: time.Now()},
		{ID: "2", Title: "寄生兽", Type: MediaTypeTV, Rating: 8.0, ScrapedAt: time.Now()},
		{ID: "3", Title: "其他电影", Type: MediaTypeMovie, Rating: 7.0, ScrapedAt: time.Now()},
	}

	for _, e := range entries {
		err := cache.Set(ctx, &e)
		require.NoError(t, err)
	}

	// Search by title
	results, err := cache.GetByTitle(ctx, "寄生")
	require.NoError(t, err)
	assert.Len(t, results, 2)

	// Should be ordered by rating
	assert.Equal(t, "1", results[0].ID)
	assert.Equal(t, "2", results[1].ID)
}

func TestCache_Delete(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	config := DefaultCacheConfig()
	config.CleanupInterval = 0
	cache := NewCache(db, config, nil)
	defer cache.Close()

	ctx := context.Background()

	result := &DetailResult{
		ID:        "12345",
		Title:     "Test Movie",
		Type:      MediaTypeMovie,
		ScrapedAt: time.Now(),
	}

	// Set and verify
	err := cache.Set(ctx, result)
	require.NoError(t, err)

	cached, err := cache.Get(ctx, "12345")
	require.NoError(t, err)
	require.NotNil(t, cached)

	// Delete
	err = cache.Delete(ctx, "12345")
	require.NoError(t, err)

	// Verify deleted
	cached, err = cache.Get(ctx, "12345")
	require.NoError(t, err)
	assert.Nil(t, cached)
}

func TestCache_DeleteExpired(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	config := DefaultCacheConfig()
	config.CleanupInterval = 0
	cache := NewCache(db, config, nil)
	defer cache.Close()

	ctx := context.Background()

	// Insert an expired entry directly
	_, err := db.Exec(`
		INSERT INTO douban_cache (id, douban_id, title, media_type, expires_at)
		VALUES ('test-id', 'expired-1', 'Expired Movie', 'movie', datetime('now', '-1 day'))
	`)
	require.NoError(t, err)

	// Insert a valid entry
	result := &DetailResult{
		ID:        "valid-1",
		Title:     "Valid Movie",
		Type:      MediaTypeMovie,
		ScrapedAt: time.Now(),
	}
	err = cache.Set(ctx, result)
	require.NoError(t, err)

	// Delete expired
	err = cache.DeleteExpired(ctx)
	require.NoError(t, err)

	// Verify expired is gone
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM douban_cache WHERE douban_id = 'expired-1'").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 0, count)

	// Verify valid still exists
	cached, err := cache.Get(ctx, "valid-1")
	require.NoError(t, err)
	assert.NotNil(t, cached)
}

func TestCache_Clear(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	config := DefaultCacheConfig()
	config.CleanupInterval = 0
	cache := NewCache(db, config, nil)
	defer cache.Close()

	ctx := context.Background()

	// Add entries
	for i := 0; i < 5; i++ {
		result := &DetailResult{
			ID:        string(rune('A' + i)),
			Title:     "Movie " + string(rune('A'+i)),
			Type:      MediaTypeMovie,
			ScrapedAt: time.Now(),
		}
		err := cache.Set(ctx, result)
		require.NoError(t, err)
	}

	// Clear
	err := cache.Clear(ctx)
	require.NoError(t, err)

	// Verify empty
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM douban_cache").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 0, count)
}

func TestCache_Stats(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	config := DefaultCacheConfig()
	config.CleanupInterval = 0
	cache := NewCache(db, config, nil)
	defer cache.Close()

	ctx := context.Background()

	// Add valid entries
	for i := 0; i < 3; i++ {
		result := &DetailResult{
			ID:        string(rune('A' + i)),
			Title:     "Movie " + string(rune('A'+i)),
			Type:      MediaTypeMovie,
			ScrapedAt: time.Now(),
		}
		err := cache.Set(ctx, result)
		require.NoError(t, err)
	}

	// Add expired entry directly
	_, err := db.Exec(`
		INSERT INTO douban_cache (id, douban_id, title, media_type, expires_at)
		VALUES ('test-expired', 'expired', 'Expired', 'movie', datetime('now', '-1 day'))
	`)
	require.NoError(t, err)

	// Get stats
	stats, err := cache.Stats(ctx)
	require.NoError(t, err)

	assert.Equal(t, 4, stats.TotalEntries)
	assert.Equal(t, 3, stats.ActiveEntries)
	assert.Equal(t, 1, stats.ExpiredEntries)
	assert.Equal(t, config.DefaultTTL, stats.TTL)
}

func TestCache_Disabled(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	config := DefaultCacheConfig()
	config.Enabled = false
	config.CleanupInterval = 0
	cache := NewCache(db, config, nil)
	defer cache.Close()

	ctx := context.Background()

	result := &DetailResult{
		ID:        "12345",
		Title:     "Test Movie",
		Type:      MediaTypeMovie,
		ScrapedAt: time.Now(),
	}

	// Set should be no-op
	err := cache.Set(ctx, result)
	require.NoError(t, err)

	// Get should return nil
	cached, err := cache.Get(ctx, "12345")
	require.NoError(t, err)
	assert.Nil(t, cached)
}

func TestCache_NilDB(t *testing.T) {
	config := DefaultCacheConfig()
	config.CleanupInterval = 0
	cache := NewCache(nil, config, nil)
	defer cache.Close()

	ctx := context.Background()

	// All operations should handle nil DB gracefully
	result := &DetailResult{ID: "12345", Title: "Test", Type: MediaTypeMovie, ScrapedAt: time.Now()}

	err := cache.Set(ctx, result)
	assert.NoError(t, err)

	cached, err := cache.Get(ctx, "12345")
	assert.NoError(t, err)
	assert.Nil(t, cached)

	err = cache.Delete(ctx, "12345")
	assert.NoError(t, err)

	err = cache.DeleteExpired(ctx)
	assert.NoError(t, err)

	err = cache.Clear(ctx)
	assert.NoError(t, err)

	stats, err := cache.Stats(ctx)
	assert.NoError(t, err)
	assert.Equal(t, 0, stats.TotalEntries)
}

func TestDefaultCacheConfig(t *testing.T) {
	config := DefaultCacheConfig()

	assert.Equal(t, 7*24*time.Hour, config.DefaultTTL)
	assert.Equal(t, 1*time.Hour, config.CleanupInterval)
	assert.True(t, config.Enabled)
}

func TestCache_CleanupLoop(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Use a very short cleanup interval for testing
	config := CacheConfig{
		DefaultTTL:      100 * time.Millisecond,
		CleanupInterval: 50 * time.Millisecond, // Very short for test
		Enabled:         true,
	}
	cache := NewCache(db, config, nil)

	ctx := context.Background()

	// Insert an entry that will expire in the past (already expired)
	_, err := db.Exec(`
		INSERT INTO douban_cache (id, douban_id, title, media_type, scraped_at, expires_at)
		VALUES ('test-id', 'to-expire', 'Expiring Movie', 'movie', datetime('now'), datetime('now', '-1 second'))
	`)
	require.NoError(t, err)

	// Verify entry exists
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM douban_cache WHERE douban_id = 'to-expire'").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 1, count)

	// Wait for cleanup loop to run (should trigger within ~100ms)
	time.Sleep(150 * time.Millisecond)

	// Close cache to stop cleanup loop
	cache.Close()

	// Verify entry was cleaned up
	err = db.QueryRow("SELECT COUNT(*) FROM douban_cache WHERE douban_id = 'to-expire'").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 0, count, "expired entry should have been cleaned up by background loop")

	_ = ctx // ctx used for future assertions if needed
}

func TestCache_CleanupLoop_Stop(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	config := CacheConfig{
		DefaultTTL:      1 * time.Hour,
		CleanupInterval: 10 * time.Millisecond,
		Enabled:         true,
	}
	cache := NewCache(db, config, nil)

	// Close should stop the cleanup goroutine gracefully
	cache.Close()

	// If we reach here without hanging, the test passes
	// The cleanup goroutine should have stopped
}
