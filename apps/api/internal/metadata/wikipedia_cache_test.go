package metadata

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func setupTestWikipediaCache(t *testing.T) (*WikipediaCache, *sql.DB) {
	// Create in-memory database
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)

	// Create the wikipedia_cache table
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS wikipedia_cache (
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
		)
	`)
	require.NoError(t, err)

	// Create indexes
	_, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_wikipedia_cache_query ON wikipedia_cache(query)`)
	require.NoError(t, err)
	_, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_wikipedia_cache_page_title ON wikipedia_cache(page_title)`)
	require.NoError(t, err)
	_, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_wikipedia_cache_expires_at ON wikipedia_cache(expires_at)`)
	require.NoError(t, err)

	cache := NewWikipediaCache(db, nil)
	return cache, db
}

func TestWikipediaCache_SetAndGet(t *testing.T) {
	cache, db := setupTestWikipediaCache(t)
	defer db.Close()

	ctx := context.Background()

	t.Run("stores and retrieves item", func(t *testing.T) {
		item := &MetadataItem{
			ID:            "12345",
			Title:         "寄生上流",
			TitleZhTW:     "寄生上流",
			OriginalTitle: "기생충",
			Year:          2019,
			Overview:      "一部韓國電影",
			Genres:        []string{"驚悚", "黑色喜劇"},
			PosterURL:     "https://upload.wikimedia.org/poster.jpg",
			MediaType:     MediaTypeMovie,
			Confidence:    0.95,
		}

		err := cache.Set(ctx, "寄生上流", item)
		require.NoError(t, err)

		retrieved, err := cache.Get(ctx, "寄生上流")
		require.NoError(t, err)
		require.NotNil(t, retrieved)

		assert.Equal(t, "12345", retrieved.ID)
		assert.Equal(t, "寄生上流", retrieved.Title)
		assert.Equal(t, "기생충", retrieved.OriginalTitle)
		assert.Equal(t, 2019, retrieved.Year)
		assert.Equal(t, "一部韓國電影", retrieved.Overview)
		assert.Equal(t, []string{"驚悚", "黑色喜劇"}, retrieved.Genres)
		assert.Equal(t, "https://upload.wikimedia.org/poster.jpg", retrieved.PosterURL)
		assert.Equal(t, MediaTypeMovie, retrieved.MediaType)
		assert.Equal(t, 0.95, retrieved.Confidence)
	})

	t.Run("returns nil for cache miss", func(t *testing.T) {
		retrieved, err := cache.Get(ctx, "nonexistent query")
		require.NoError(t, err)
		assert.Nil(t, retrieved)
	})

	t.Run("handles nil item gracefully", func(t *testing.T) {
		err := cache.Set(ctx, "test", nil)
		require.NoError(t, err)
	})
}

func TestWikipediaCache_Expiration(t *testing.T) {
	cache, db := setupTestWikipediaCache(t)
	defer db.Close()

	ctx := context.Background()

	// Insert an expired entry directly
	_, err := db.Exec(`
		INSERT INTO wikipedia_cache (id, query, page_title, title, media_type, confidence, expires_at)
		VALUES ('expired-id', 'expired query', '12345', 'Expired Title', 'movie', 0.5, datetime('now', '-1 day'))
	`)
	require.NoError(t, err)

	// Should not retrieve expired entries
	retrieved, err := cache.Get(ctx, "expired query")
	require.NoError(t, err)
	assert.Nil(t, retrieved)
}

func TestWikipediaCache_GetByPageTitle(t *testing.T) {
	cache, db := setupTestWikipediaCache(t)
	defer db.Close()

	ctx := context.Background()

	item := &MetadataItem{
		ID:         "67890",
		Title:      "Test Movie",
		MediaType:  MediaTypeMovie,
		Confidence: 0.8,
	}

	err := cache.Set(ctx, "test query", item)
	require.NoError(t, err)

	// Get by page title (which is the ID)
	retrieved, err := cache.GetByPageTitle(ctx, "67890")
	require.NoError(t, err)
	require.NotNil(t, retrieved)
	assert.Equal(t, "Test Movie", retrieved.Title)
}

func TestWikipediaCache_Delete(t *testing.T) {
	cache, db := setupTestWikipediaCache(t)
	defer db.Close()

	ctx := context.Background()

	item := &MetadataItem{
		ID:         "12345",
		Title:      "To Delete",
		MediaType:  MediaTypeMovie,
		Confidence: 0.8,
	}

	err := cache.Set(ctx, "delete me", item)
	require.NoError(t, err)

	// Verify it exists
	retrieved, _ := cache.Get(ctx, "delete me")
	assert.NotNil(t, retrieved)

	// Delete it
	err = cache.Delete(ctx, "delete me")
	require.NoError(t, err)

	// Verify it's gone
	retrieved, _ = cache.Get(ctx, "delete me")
	assert.Nil(t, retrieved)
}

func TestWikipediaCache_DeleteExpired(t *testing.T) {
	cache, db := setupTestWikipediaCache(t)
	defer db.Close()

	ctx := context.Background()

	// Insert expired entries directly
	_, err := db.Exec(`
		INSERT INTO wikipedia_cache (id, query, page_title, title, media_type, confidence, expires_at)
		VALUES ('expired1', 'q1', 'p1', 'Title1', 'movie', 0.5, datetime('now', '-1 day'))
	`)
	require.NoError(t, err)

	_, err = db.Exec(`
		INSERT INTO wikipedia_cache (id, query, page_title, title, media_type, confidence, expires_at)
		VALUES ('expired2', 'q2', 'p2', 'Title2', 'movie', 0.5, datetime('now', '-2 days'))
	`)
	require.NoError(t, err)

	// Insert a valid entry
	item := &MetadataItem{
		ID:         "valid",
		Title:      "Valid Title",
		MediaType:  MediaTypeMovie,
		Confidence: 0.8,
	}
	err = cache.Set(ctx, "valid query", item)
	require.NoError(t, err)

	// Count before cleanup
	var countBefore int
	db.QueryRow("SELECT COUNT(*) FROM wikipedia_cache").Scan(&countBefore)
	assert.Equal(t, 3, countBefore)

	// Delete expired
	err = cache.DeleteExpired(ctx)
	require.NoError(t, err)

	// Count after cleanup
	var countAfter int
	db.QueryRow("SELECT COUNT(*) FROM wikipedia_cache").Scan(&countAfter)
	assert.Equal(t, 1, countAfter)

	// The valid entry should still exist
	retrieved, _ := cache.Get(ctx, "valid query")
	assert.NotNil(t, retrieved)
}

func TestWikipediaCache_Clear(t *testing.T) {
	cache, db := setupTestWikipediaCache(t)
	defer db.Close()

	ctx := context.Background()

	// Add some entries
	for i := 0; i < 5; i++ {
		item := &MetadataItem{
			ID:         string(rune('A' + i)),
			Title:      "Title",
			MediaType:  MediaTypeMovie,
			Confidence: 0.5,
		}
		cache.Set(ctx, string(rune('a'+i)), item)
	}

	// Clear all
	err := cache.Clear(ctx)
	require.NoError(t, err)

	// Verify empty
	var count int
	db.QueryRow("SELECT COUNT(*) FROM wikipedia_cache").Scan(&count)
	assert.Equal(t, 0, count)
}

func TestWikipediaCache_Stats(t *testing.T) {
	cache, db := setupTestWikipediaCache(t)
	defer db.Close()

	ctx := context.Background()

	// Add valid entries
	for i := 0; i < 3; i++ {
		item := &MetadataItem{
			ID:         string(rune('A' + i)),
			Title:      "Title",
			MediaType:  MediaTypeMovie,
			Confidence: 0.5,
		}
		cache.Set(ctx, string(rune('a'+i)), item)
	}

	// Add expired entry directly
	_, err := db.Exec(`
		INSERT INTO wikipedia_cache (id, query, page_title, title, media_type, confidence, expires_at)
		VALUES ('expired', 'expired', 'exp', 'Expired', 'movie', 0.5, datetime('now', '-1 day'))
	`)
	require.NoError(t, err)

	stats, err := cache.Stats(ctx)
	require.NoError(t, err)
	require.NotNil(t, stats)

	assert.Equal(t, 4, stats.TotalEntries)
	assert.Equal(t, 3, stats.ActiveEntries)
	assert.Equal(t, 1, stats.ExpiredEntries)
	assert.Equal(t, 7*24*time.Hour, stats.TTL)
}

func TestWikipediaCacheTTL(t *testing.T) {
	assert.Equal(t, 7*24*time.Hour, WikipediaCacheTTL)
}

func TestNewWikipediaCache_NilDB(t *testing.T) {
	cache := NewWikipediaCache(nil, nil)
	require.NotNil(t, cache)

	// Operations should be no-ops with nil db
	ctx := context.Background()

	result, err := cache.Get(ctx, "test")
	assert.NoError(t, err)
	assert.Nil(t, result)

	err = cache.Set(ctx, "test", &MetadataItem{ID: "1", Title: "Test"})
	assert.NoError(t, err)

	err = cache.Delete(ctx, "test")
	assert.NoError(t, err)

	err = cache.DeleteExpired(ctx)
	assert.NoError(t, err)

	err = cache.Clear(ctx)
	assert.NoError(t, err)

	stats, err := cache.Stats(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, stats)
	assert.Equal(t, 0, stats.TotalEntries)
}
