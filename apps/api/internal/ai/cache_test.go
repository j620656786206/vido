package ai

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

	// Create ai_cache table
	_, err = db.Exec(`
		CREATE TABLE ai_cache (
			id TEXT PRIMARY KEY,
			filename_hash TEXT UNIQUE NOT NULL,
			provider TEXT NOT NULL,
			request_prompt TEXT NOT NULL,
			response_json TEXT NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			expires_at TIMESTAMP NOT NULL
		);
		CREATE INDEX idx_ai_cache_filename_hash ON ai_cache(filename_hash);
		CREATE INDEX idx_ai_cache_expires_at ON ai_cache(expires_at);
	`)
	require.NoError(t, err)

	return db
}

func TestHashFilename(t *testing.T) {
	tests := []struct {
		name     string
		filename string
	}{
		{"simple filename", "movie.mkv"},
		{"complex filename", "[SubsPlease] Shingeki no Kyojin - 01 [1080p].mkv"},
		{"unicode filename", "進擊的巨人 第一季 01.mkv"},
		{"empty filename", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash := HashFilename(tt.filename)

			// SHA-256 produces 64 hex characters
			assert.Len(t, hash, 64)

			// Same input should produce same hash
			hash2 := HashFilename(tt.filename)
			assert.Equal(t, hash, hash2)
		})
	}

	// Different inputs should produce different hashes
	hash1 := HashFilename("file1.mkv")
	hash2 := HashFilename("file2.mkv")
	assert.NotEqual(t, hash1, hash2)
}

func TestNewCache(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	t.Run("default TTL", func(t *testing.T) {
		cache := NewCache(db)
		assert.Equal(t, DefaultCacheTTL, cache.ttl)
	})

	t.Run("custom TTL", func(t *testing.T) {
		customTTL := 7 * 24 * time.Hour
		cache := NewCache(db, WithCacheTTL(customTTL))
		assert.Equal(t, customTTL, cache.ttl)
	})
}

func TestCache_SetAndGet(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	cache := NewCache(db)
	ctx := context.Background()

	filename := "[SubsPlease] Attack on Titan - 01 [1080p].mkv"
	response := &ParseResponse{
		Title:       "Attack on Titan",
		Year:        2013,
		Season:      1,
		Episode:     1,
		MediaType:   "tv",
		Quality:     "1080p",
		FansubGroup: "SubsPlease",
		Confidence:  0.95,
	}

	// Set cache
	err := cache.Set(ctx, filename, ProviderGemini, "test prompt", response)
	require.NoError(t, err)

	// Get cache
	cached, err := cache.Get(ctx, filename)
	require.NoError(t, err)
	require.NotNil(t, cached)

	assert.Equal(t, response.Title, cached.Title)
	assert.Equal(t, response.Year, cached.Year)
	assert.Equal(t, response.Season, cached.Season)
	assert.Equal(t, response.Episode, cached.Episode)
	assert.Equal(t, response.MediaType, cached.MediaType)
	assert.Equal(t, response.Confidence, cached.Confidence)
}

func TestCache_Get_CacheMiss(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	cache := NewCache(db)
	ctx := context.Background()

	// Get non-existent entry
	cached, err := cache.Get(ctx, "nonexistent.mkv")
	require.NoError(t, err)
	assert.Nil(t, cached)
}

func TestCache_Get_Expired(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Use very short TTL for testing
	cache := NewCache(db, WithCacheTTL(1*time.Millisecond))
	ctx := context.Background()

	filename := "test.mkv"
	response := &ParseResponse{
		Title:     "Test",
		MediaType: "movie",
	}

	// Set cache
	err := cache.Set(ctx, filename, ProviderGemini, "test", response)
	require.NoError(t, err)

	// Wait for expiry
	time.Sleep(10 * time.Millisecond)

	// Get should return nil for expired entry
	cached, err := cache.Get(ctx, filename)
	require.NoError(t, err)
	assert.Nil(t, cached)
}

func TestCache_Set_Upsert(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	cache := NewCache(db)
	ctx := context.Background()

	filename := "test.mkv"

	// First insert
	response1 := &ParseResponse{
		Title:     "Original Title",
		MediaType: "movie",
	}
	err := cache.Set(ctx, filename, ProviderGemini, "prompt1", response1)
	require.NoError(t, err)

	// Update with new data
	response2 := &ParseResponse{
		Title:     "Updated Title",
		MediaType: "tv",
	}
	err = cache.Set(ctx, filename, ProviderClaude, "prompt2", response2)
	require.NoError(t, err)

	// Should get updated data
	cached, err := cache.Get(ctx, filename)
	require.NoError(t, err)
	require.NotNil(t, cached)
	assert.Equal(t, "Updated Title", cached.Title)
	assert.Equal(t, "tv", cached.MediaType)
}

func TestCache_Delete(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	cache := NewCache(db)
	ctx := context.Background()

	filename := "test.mkv"
	response := &ParseResponse{
		Title:     "Test",
		MediaType: "movie",
	}

	// Set and verify
	err := cache.Set(ctx, filename, ProviderGemini, "test", response)
	require.NoError(t, err)

	cached, err := cache.Get(ctx, filename)
	require.NoError(t, err)
	require.NotNil(t, cached)

	// Delete
	err = cache.Delete(ctx, filename)
	require.NoError(t, err)

	// Should not find
	cached, err = cache.Get(ctx, filename)
	require.NoError(t, err)
	assert.Nil(t, cached)
}

func TestCache_ClearExpired(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()
	cache := NewCache(db, WithCacheTTL(24*time.Hour)) // Long TTL

	// Insert entries with manual expired timestamp directly in DB
	for i := 0; i < 5; i++ {
		// Use past time directly for expires_at
		pastTime := time.Now().Add(-1 * time.Hour).Format(time.RFC3339)
		_, err := db.ExecContext(ctx,
			`INSERT INTO ai_cache (id, filename_hash, provider, request_prompt, response_json, created_at, expires_at)
			 VALUES (?, ?, ?, ?, ?, ?, ?)`,
			"id-"+string(rune('0'+i)), "hash-"+string(rune('0'+i)), "gemini", "test",
			`{"title":"Test","media_type":"movie"}`, pastTime, pastTime)
		require.NoError(t, err)
	}

	// Verify entries exist
	stats, err := cache.Stats(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(5), stats.TotalEntries)
	assert.Equal(t, int64(5), stats.ExpiredEntries) // All expired

	// Clear expired
	count, err := cache.ClearExpired(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(5), count)

	// Stats should show 0 entries
	stats, err = cache.Stats(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(0), stats.TotalEntries)
}

func TestCache_ClearAll(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	cache := NewCache(db)
	ctx := context.Background()

	// Insert entries
	for i := 0; i < 3; i++ {
		response := &ParseResponse{Title: "Test", MediaType: "movie"}
		err := cache.Set(ctx, "file"+string(rune('0'+i))+".mkv", ProviderGemini, "test", response)
		require.NoError(t, err)
	}

	// Clear all
	count, err := cache.ClearAll(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(3), count)

	// Stats should show 0 entries
	stats, err := cache.Stats(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(0), stats.TotalEntries)
}

func TestCache_Stats(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()

	// Insert some with short TTL (will be expired)
	shortCache := NewCache(db, WithCacheTTL(1*time.Millisecond))
	for i := 0; i < 2; i++ {
		response := &ParseResponse{Title: "Expired", MediaType: "movie"}
		err := shortCache.Set(ctx, "expired"+string(rune('0'+i))+".mkv", ProviderGemini, "test", response)
		require.NoError(t, err)
	}

	// Wait for expiry
	time.Sleep(10 * time.Millisecond)

	// Insert some with long TTL (will be valid)
	longCache := NewCache(db, WithCacheTTL(24*time.Hour))
	for i := 0; i < 3; i++ {
		response := &ParseResponse{Title: "Valid", MediaType: "movie"}
		err := longCache.Set(ctx, "valid"+string(rune('0'+i))+".mkv", ProviderGemini, "test", response)
		require.NoError(t, err)
	}

	// Check stats
	stats, err := longCache.Stats(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(5), stats.TotalEntries)
	assert.Equal(t, int64(3), stats.ValidEntries)
	assert.Equal(t, int64(2), stats.ExpiredEntries)
}

func TestDefaultCacheTTL(t *testing.T) {
	// Verify default TTL is 30 days per NFR-I10
	assert.Equal(t, 30*24*time.Hour, DefaultCacheTTL)
}
