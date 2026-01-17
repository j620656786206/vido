package repository

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	_ "modernc.org/sqlite"
)

func TestNewRepositories(t *testing.T) {
	// Test with nil db - this is just to verify the factory function works
	// In real usage, a valid *sql.DB would be passed
	repos := NewRepositories(nil)

	assert.NotNil(t, repos)
	assert.NotNil(t, repos.Movies)
	assert.NotNil(t, repos.Series)
	assert.NotNil(t, repos.Settings)
	assert.Nil(t, repos.Cache) // Cache is not initialized yet
	assert.NotNil(t, repos.Secrets)
}

func TestRepositories_SetCacheRepository(t *testing.T) {
	repos := NewRepositories(nil)

	// Initially cache is nil
	assert.Nil(t, repos.Cache)

	// Create a mock cache repository
	mockCache := new(MockCacheRepository)

	// Set cache repository
	repos.SetCacheRepository(mockCache)

	// Verify cache is now set
	assert.NotNil(t, repos.Cache)
	assert.Equal(t, mockCache, repos.Cache)
}

func TestRepositories_SetCacheRepository_WithRealDB(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer db.Close()

	repos := NewRepositories(db)

	// Initially cache is nil
	assert.Nil(t, repos.Cache)

	// Create a real cache repository
	cacheRepo := NewCacheRepository(db)

	// Set cache repository
	repos.SetCacheRepository(cacheRepo)

	// Verify cache is now set
	assert.NotNil(t, repos.Cache)
}

func TestNewRepositoriesWithCache(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer db.Close()

	repos := NewRepositoriesWithCache(db)

	assert.NotNil(t, repos)
	assert.NotNil(t, repos.Movies)
	assert.NotNil(t, repos.Series)
	assert.NotNil(t, repos.Settings)
	assert.NotNil(t, repos.Cache) // Cache is initialized
	assert.NotNil(t, repos.Secrets)
}

func TestNewRepositoriesWithCache_CacheOperations(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer db.Close()

	// Create cache_entries table
	_, err = db.Exec(`
		CREATE TABLE cache_entries (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL,
			type TEXT NOT NULL,
			expires_at TIMESTAMP NOT NULL,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create cache_entries table: %v", err)
	}

	repos := NewRepositoriesWithCache(db)
	ctx := context.Background()

	// Test cache operations through the repository
	err = repos.Cache.Set(ctx, "test-key", "test-value", "test", time.Hour)
	assert.NoError(t, err)

	entry, err := repos.Cache.Get(ctx, "test-key")
	assert.NoError(t, err)
	assert.NotNil(t, entry)
	assert.Equal(t, "test-value", entry.Value)
}
