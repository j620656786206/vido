package repository

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	_ "modernc.org/sqlite"
)

// MockCacheRepository is a mock implementation of CacheRepositoryInterface for testing
type MockCacheRepository struct {
	mock.Mock
}

func (m *MockCacheRepository) Get(ctx context.Context, key string) (*CacheEntry, error) {
	args := m.Called(ctx, key)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*CacheEntry), args.Error(1)
}

func (m *MockCacheRepository) Set(ctx context.Context, key string, value string, cacheType string, ttl time.Duration) error {
	args := m.Called(ctx, key, value, cacheType, ttl)
	return args.Error(0)
}

func (m *MockCacheRepository) Delete(ctx context.Context, key string) error {
	args := m.Called(ctx, key)
	return args.Error(0)
}

func (m *MockCacheRepository) Clear(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockCacheRepository) ClearExpired(ctx context.Context) (int64, error) {
	args := m.Called(ctx)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockCacheRepository) ClearByType(ctx context.Context, cacheType string) (int64, error) {
	args := m.Called(ctx, cacheType)
	return args.Get(0).(int64), args.Error(1)
}

// Verify mock implements interface
var _ CacheRepositoryInterface = (*MockCacheRepository)(nil)

func TestCacheRepository_ValidationErrors(t *testing.T) {
	repo := NewCacheRepository(nil)
	ctx := context.Background()

	t.Run("Get with empty key returns error", func(t *testing.T) {
		_, err := repo.Get(ctx, "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cache key cannot be empty")
	})

	t.Run("Set with empty key returns error", func(t *testing.T) {
		err := repo.Set(ctx, "", "value", "type", time.Hour)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cache key cannot be empty")
	})

	t.Run("Set with zero TTL returns error", func(t *testing.T) {
		err := repo.Set(ctx, "key", "value", "type", 0)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cache TTL must be positive")
	})

	t.Run("Set with negative TTL returns error", func(t *testing.T) {
		err := repo.Set(ctx, "key", "value", "type", -time.Hour)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cache TTL must be positive")
	})

	t.Run("Delete with empty key returns error", func(t *testing.T) {
		err := repo.Delete(ctx, "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cache key cannot be empty")
	})

	t.Run("ClearByType with empty type returns error", func(t *testing.T) {
		_, err := repo.ClearByType(ctx, "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cache type cannot be empty")
	})
}

func TestMockCacheRepository_Interface(t *testing.T) {
	mockRepo := new(MockCacheRepository)
	ctx := context.Background()

	t.Run("Set and Get", func(t *testing.T) {
		mockRepo.On("Set", ctx, "test-key", "test-value", "tmdb", time.Hour).Return(nil)
		mockRepo.On("Get", ctx, "test-key").Return(&CacheEntry{
			Key:   "test-key",
			Value: "test-value",
			Type:  "tmdb",
		}, nil)

		err := mockRepo.Set(ctx, "test-key", "test-value", "tmdb", time.Hour)
		assert.NoError(t, err)

		entry, err := mockRepo.Get(ctx, "test-key")
		assert.NoError(t, err)
		assert.NotNil(t, entry)
		assert.Equal(t, "test-value", entry.Value)
	})

	t.Run("Delete", func(t *testing.T) {
		mockRepo.On("Delete", ctx, "delete-key").Return(nil)

		err := mockRepo.Delete(ctx, "delete-key")
		assert.NoError(t, err)
	})

	t.Run("Clear", func(t *testing.T) {
		mockRepo.On("Clear", ctx).Return(nil)

		err := mockRepo.Clear(ctx)
		assert.NoError(t, err)
	})

	t.Run("ClearExpired", func(t *testing.T) {
		mockRepo.On("ClearExpired", ctx).Return(int64(5), nil)

		count, err := mockRepo.ClearExpired(ctx)
		assert.NoError(t, err)
		assert.Equal(t, int64(5), count)
	})

	t.Run("ClearByType", func(t *testing.T) {
		mockRepo.On("ClearByType", ctx, "tmdb").Return(int64(10), nil)

		count, err := mockRepo.ClearByType(ctx, "tmdb")
		assert.NoError(t, err)
		assert.Equal(t, int64(10), count)
	})

	mockRepo.AssertExpectations(t)
}

// setupCacheTestDB creates an in-memory database with cache_entries table
func setupCacheTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}

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

	// Create index on type
	_, err = db.Exec(`CREATE INDEX idx_cache_type ON cache_entries(type)`)
	if err != nil {
		t.Fatalf("Failed to create type index: %v", err)
	}

	// Create index on expires_at
	_, err = db.Exec(`CREATE INDEX idx_cache_expires ON cache_entries(expires_at)`)
	if err != nil {
		t.Fatalf("Failed to create expires_at index: %v", err)
	}

	return db
}

// TestCacheRepository_SetAndGet verifies basic set and get operations
func TestCacheRepository_SetAndGet(t *testing.T) {
	db := setupCacheTestDB(t)
	defer db.Close()

	repo := NewCacheRepository(db)
	ctx := context.Background()

	// Set a cache entry
	err := repo.Set(ctx, "test-key", `{"data": "value"}`, "tmdb", time.Hour)
	assert.NoError(t, err)

	// Get the cache entry
	entry, err := repo.Get(ctx, "test-key")
	assert.NoError(t, err)
	assert.NotNil(t, entry)
	assert.Equal(t, "test-key", entry.Key)
	assert.Equal(t, `{"data": "value"}`, entry.Value)
	assert.Equal(t, "tmdb", entry.Type)
}

// TestCacheRepository_GetNotFound verifies nil returned for non-existent key
func TestCacheRepository_GetNotFound(t *testing.T) {
	db := setupCacheTestDB(t)
	defer db.Close()

	repo := NewCacheRepository(db)
	ctx := context.Background()

	entry, err := repo.Get(ctx, "non-existent-key")
	assert.NoError(t, err)
	assert.Nil(t, entry)
}

// TestCacheRepository_GetExpired verifies expired entries return nil
func TestCacheRepository_GetExpired(t *testing.T) {
	db := setupCacheTestDB(t)
	defer db.Close()

	repo := NewCacheRepository(db)
	ctx := context.Background()

	// Manually insert an expired entry (1 hour in the past)
	_, err := db.Exec(`
		INSERT INTO cache_entries (key, value, type, expires_at, created_at, updated_at)
		VALUES (?, ?, ?, datetime('now', '-1 hour'), datetime('now'), datetime('now'))
	`, "expiring-key", "value", "test")
	assert.NoError(t, err)

	// Get should return nil for expired entry
	entry, err := repo.Get(ctx, "expiring-key")
	assert.NoError(t, err)
	assert.Nil(t, entry)
}

// TestCacheRepository_SetUpsert verifies set updates existing entry
func TestCacheRepository_SetUpsert(t *testing.T) {
	db := setupCacheTestDB(t)
	defer db.Close()

	repo := NewCacheRepository(db)
	ctx := context.Background()

	// Set initial value
	err := repo.Set(ctx, "upsert-key", "original-value", "test", time.Hour)
	assert.NoError(t, err)

	// Update with new value
	err = repo.Set(ctx, "upsert-key", "updated-value", "test", time.Hour)
	assert.NoError(t, err)

	// Verify updated value
	entry, err := repo.Get(ctx, "upsert-key")
	assert.NoError(t, err)
	assert.NotNil(t, entry)
	assert.Equal(t, "updated-value", entry.Value)

	// Verify only one entry exists
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM cache_entries WHERE key = ?", "upsert-key").Scan(&count)
	assert.NoError(t, err)
	assert.Equal(t, 1, count)
}

// TestCacheRepository_Delete verifies cache entry deletion
func TestCacheRepository_Delete(t *testing.T) {
	db := setupCacheTestDB(t)
	defer db.Close()

	repo := NewCacheRepository(db)
	ctx := context.Background()

	// Set a cache entry
	err := repo.Set(ctx, "delete-key", "value", "test", time.Hour)
	assert.NoError(t, err)

	// Verify entry exists
	entry, err := repo.Get(ctx, "delete-key")
	assert.NoError(t, err)
	assert.NotNil(t, entry)

	// Delete the entry
	err = repo.Delete(ctx, "delete-key")
	assert.NoError(t, err)

	// Verify entry is deleted
	entry, err = repo.Get(ctx, "delete-key")
	assert.NoError(t, err)
	assert.Nil(t, entry)
}

// TestCacheRepository_DeleteNonExistent verifies no error for non-existent key
func TestCacheRepository_DeleteNonExistent(t *testing.T) {
	db := setupCacheTestDB(t)
	defer db.Close()

	repo := NewCacheRepository(db)
	ctx := context.Background()

	// Delete non-existent key should not error
	err := repo.Delete(ctx, "non-existent-key")
	assert.NoError(t, err)
}

// TestCacheRepository_Clear verifies all entries are removed
func TestCacheRepository_Clear(t *testing.T) {
	db := setupCacheTestDB(t)
	defer db.Close()

	repo := NewCacheRepository(db)
	ctx := context.Background()

	// Set multiple entries
	err := repo.Set(ctx, "key1", "value1", "type1", time.Hour)
	assert.NoError(t, err)
	err = repo.Set(ctx, "key2", "value2", "type2", time.Hour)
	assert.NoError(t, err)
	err = repo.Set(ctx, "key3", "value3", "type1", time.Hour)
	assert.NoError(t, err)

	// Verify entries exist
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM cache_entries").Scan(&count)
	assert.NoError(t, err)
	assert.Equal(t, 3, count)

	// Clear all entries
	err = repo.Clear(ctx)
	assert.NoError(t, err)

	// Verify all entries are removed
	err = db.QueryRow("SELECT COUNT(*) FROM cache_entries").Scan(&count)
	assert.NoError(t, err)
	assert.Equal(t, 0, count)
}

// TestCacheRepository_ClearExpired verifies only expired entries are removed
func TestCacheRepository_ClearExpired(t *testing.T) {
	db := setupCacheTestDB(t)
	defer db.Close()

	repo := NewCacheRepository(db)
	ctx := context.Background()

	// Manually insert an expired entry (1 hour in the past)
	_, err := db.Exec(`
		INSERT INTO cache_entries (key, value, type, expires_at, created_at, updated_at)
		VALUES (?, ?, ?, datetime('now', '-1 hour'), datetime('now'), datetime('now'))
	`, "expired-entry", "value", "test")
	assert.NoError(t, err)

	// Set a long-lived entry using the repo
	err = repo.Set(ctx, "long-lived", "value", "test", time.Hour)
	assert.NoError(t, err)

	// Clear expired entries
	deleted, err := repo.ClearExpired(ctx)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), deleted)

	// Verify expired is gone
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM cache_entries WHERE key = ?", "expired-entry").Scan(&count)
	assert.NoError(t, err)
	assert.Equal(t, 0, count)

	// Verify long-lived remains
	entry, err := repo.Get(ctx, "long-lived")
	assert.NoError(t, err)
	assert.NotNil(t, entry)
}

// TestCacheRepository_ClearExpiredNone verifies zero returned when no expired entries
func TestCacheRepository_ClearExpiredNone(t *testing.T) {
	db := setupCacheTestDB(t)
	defer db.Close()

	repo := NewCacheRepository(db)
	ctx := context.Background()

	// Set only long-lived entries
	err := repo.Set(ctx, "key1", "value", "test", time.Hour)
	assert.NoError(t, err)

	// Clear expired (should be none)
	deleted, err := repo.ClearExpired(ctx)
	assert.NoError(t, err)
	assert.Equal(t, int64(0), deleted)
}

// TestCacheRepository_ClearByType verifies entries are removed by type
func TestCacheRepository_ClearByType(t *testing.T) {
	db := setupCacheTestDB(t)
	defer db.Close()

	repo := NewCacheRepository(db)
	ctx := context.Background()

	// Set entries with different types
	err := repo.Set(ctx, "tmdb-1", "value1", "tmdb", time.Hour)
	assert.NoError(t, err)
	err = repo.Set(ctx, "tmdb-2", "value2", "tmdb", time.Hour)
	assert.NoError(t, err)
	err = repo.Set(ctx, "other-1", "value3", "other", time.Hour)
	assert.NoError(t, err)

	// Clear entries by type "tmdb"
	deleted, err := repo.ClearByType(ctx, "tmdb")
	assert.NoError(t, err)
	assert.Equal(t, int64(2), deleted)

	// Verify tmdb entries are gone
	entry, err := repo.Get(ctx, "tmdb-1")
	assert.NoError(t, err)
	assert.Nil(t, entry)

	entry, err = repo.Get(ctx, "tmdb-2")
	assert.NoError(t, err)
	assert.Nil(t, entry)

	// Verify other type remains
	entry, err = repo.Get(ctx, "other-1")
	assert.NoError(t, err)
	assert.NotNil(t, entry)
}

// TestCacheRepository_ClearByTypeNone verifies zero returned when no matching type
func TestCacheRepository_ClearByTypeNone(t *testing.T) {
	db := setupCacheTestDB(t)
	defer db.Close()

	repo := NewCacheRepository(db)
	ctx := context.Background()

	// Set entries with different type
	err := repo.Set(ctx, "key1", "value", "other", time.Hour)
	assert.NoError(t, err)

	// Clear by non-existent type
	deleted, err := repo.ClearByType(ctx, "tmdb")
	assert.NoError(t, err)
	assert.Equal(t, int64(0), deleted)
}

// TestCacheRepository_MultipleCacheTypes verifies different cache types work correctly
func TestCacheRepository_MultipleCacheTypes(t *testing.T) {
	db := setupCacheTestDB(t)
	defer db.Close()

	repo := NewCacheRepository(db)
	ctx := context.Background()

	// Set entries with different cache types
	err := repo.Set(ctx, "movie:123", `{"title":"Movie"}`, "tmdb-movie", time.Hour)
	assert.NoError(t, err)
	err = repo.Set(ctx, "series:456", `{"title":"Series"}`, "tmdb-series", time.Hour)
	assert.NoError(t, err)
	err = repo.Set(ctx, "search:action", `{"results":[]}`, "tmdb-search", time.Hour)
	assert.NoError(t, err)

	// Verify all entries can be retrieved
	entry, err := repo.Get(ctx, "movie:123")
	assert.NoError(t, err)
	assert.Equal(t, "tmdb-movie", entry.Type)

	entry, err = repo.Get(ctx, "series:456")
	assert.NoError(t, err)
	assert.Equal(t, "tmdb-series", entry.Type)

	entry, err = repo.Get(ctx, "search:action")
	assert.NoError(t, err)
	assert.Equal(t, "tmdb-search", entry.Type)
}

// TestCacheRepository_LargeValue verifies large values can be stored and retrieved
func TestCacheRepository_LargeValue(t *testing.T) {
	db := setupCacheTestDB(t)
	defer db.Close()

	repo := NewCacheRepository(db)
	ctx := context.Background()

	// Create a large value (simulate JSON response)
	largeValue := `{"results":[`
	for i := 0; i < 100; i++ {
		if i > 0 {
			largeValue += ","
		}
		largeValue += `{"id":` + string(rune('0'+i%10)) + `,"title":"Movie Title","overview":"A very long overview..."}`
	}
	largeValue += `]}`

	err := repo.Set(ctx, "large-key", largeValue, "test", time.Hour)
	assert.NoError(t, err)

	entry, err := repo.Get(ctx, "large-key")
	assert.NoError(t, err)
	assert.NotNil(t, entry)
	assert.Equal(t, largeValue, entry.Value)
}
