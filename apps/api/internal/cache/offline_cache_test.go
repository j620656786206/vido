package cache

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "github.com/mattn/go-sqlite3"
)

func setupTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)

	// Create cache table
	_, err = db.Exec(`
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
	require.NoError(t, err)

	return db
}

func TestNewOfflineCache(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	cache := NewOfflineCache(db)
	require.NotNil(t, cache)
}

func TestOfflineCache_Set_Get(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	cache := NewOfflineCache(db)

	ctx := context.Background()
	key := "test-key"
	value := `{"title":"The Matrix","year":1999}`
	dataType := "metadata"

	// Set value
	err := cache.Set(ctx, key, value, dataType, time.Hour)
	require.NoError(t, err)

	// Get value
	result, isStale, err := cache.Get(ctx, key)
	require.NoError(t, err)
	assert.Equal(t, value, result)
	assert.False(t, isStale)
}

func TestOfflineCache_Get_NotFound(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	cache := NewOfflineCache(db)

	ctx := context.Background()
	result, isStale, err := cache.Get(ctx, "non-existent")
	require.NoError(t, err)
	assert.Empty(t, result)
	assert.False(t, isStale)
}

func TestOfflineCache_Get_Expired_StillReturned(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	cache := NewOfflineCache(db)

	ctx := context.Background()
	key := "expired-key"
	value := `{"title":"Old Movie"}`

	// Set with very short TTL
	err := cache.Set(ctx, key, value, "metadata", time.Millisecond)
	require.NoError(t, err)

	// Wait for expiration
	time.Sleep(10 * time.Millisecond)

	// Should still return value but mark as stale
	result, isStale, err := cache.Get(ctx, key)
	require.NoError(t, err)
	assert.Equal(t, value, result)
	assert.True(t, isStale)
}

func TestOfflineCache_Delete(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	cache := NewOfflineCache(db)

	ctx := context.Background()
	key := "delete-key"
	value := `{"title":"To Delete"}`

	// Set value
	err := cache.Set(ctx, key, value, "metadata", time.Hour)
	require.NoError(t, err)

	// Delete
	err = cache.Delete(ctx, key)
	require.NoError(t, err)

	// Verify deleted
	result, _, err := cache.Get(ctx, key)
	require.NoError(t, err)
	assert.Empty(t, result)
}

func TestOfflineCache_MarkStale(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	cache := NewOfflineCache(db)

	ctx := context.Background()
	key := "stale-key"
	value := `{"title":"Stale Data"}`

	// Set value
	err := cache.Set(ctx, key, value, "metadata", time.Hour)
	require.NoError(t, err)

	// Mark as stale
	err = cache.MarkStale(ctx, key)
	require.NoError(t, err)

	// Get should return stale
	result, isStale, err := cache.Get(ctx, key)
	require.NoError(t, err)
	assert.Equal(t, value, result)
	assert.True(t, isStale)
}

func TestOfflineCache_RefreshOnReconnection(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	cache := NewOfflineCache(db)

	ctx := context.Background()

	// Set two values - one stale, one fresh
	err := cache.Set(ctx, "fresh", "fresh-value", "metadata", time.Hour)
	require.NoError(t, err)

	err = cache.Set(ctx, "stale", "stale-value", "metadata", time.Hour)
	require.NoError(t, err)
	err = cache.MarkStale(ctx, "stale")
	require.NoError(t, err)

	// Get stale keys
	staleKeys, err := cache.GetStaleKeys(ctx)
	require.NoError(t, err)
	assert.Contains(t, staleKeys, "stale")
	assert.NotContains(t, staleKeys, "fresh")
}

func TestOfflineCache_ClearExpired(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	cache := NewOfflineCache(db)

	ctx := context.Background()

	// Set expired entry
	err := cache.Set(ctx, "expired", "old-value", "metadata", time.Millisecond)
	require.NoError(t, err)

	// Set fresh entry
	err = cache.Set(ctx, "fresh", "new-value", "metadata", time.Hour)
	require.NoError(t, err)

	time.Sleep(10 * time.Millisecond)

	// Clear expired (but in offline mode, we keep stale data)
	// This tests clearing only truly expired entries
	count, err := cache.ClearOldEntries(ctx, 24*time.Hour)
	require.NoError(t, err)
	// Both entries are newer than 24 hours
	assert.Equal(t, int64(0), count)
}
