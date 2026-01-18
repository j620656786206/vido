package services

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vido/api/internal/ai"
	_ "modernc.org/sqlite"
)

// mockProvider implements ai.Provider for testing.
type mockProvider struct {
	name        ai.ProviderName
	parseFunc   func(ctx context.Context, req *ai.ParseRequest) (*ai.ParseResponse, error)
	parseCalled bool
}

func (m *mockProvider) Parse(ctx context.Context, req *ai.ParseRequest) (*ai.ParseResponse, error) {
	m.parseCalled = true
	if m.parseFunc != nil {
		return m.parseFunc(ctx, req)
	}
	return &ai.ParseResponse{
		Title:      "Default Title",
		MediaType:  "movie",
		Confidence: 0.9,
	}, nil
}

func (m *mockProvider) Name() ai.ProviderName {
	return m.name
}

func setupTestAIDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)

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

func TestNewAIServiceWithProvider(t *testing.T) {
	db := setupTestAIDB(t)
	defer db.Close()

	provider := &mockProvider{name: ai.ProviderGemini}
	cache := ai.NewCache(db)

	service := NewAIServiceWithProvider(provider, cache)

	assert.NotNil(t, service)
	assert.True(t, service.IsConfigured())
	assert.Equal(t, "gemini", service.GetProviderName())
}

func TestAIService_ParseFilename_Success(t *testing.T) {
	db := setupTestAIDB(t)
	defer db.Close()

	expectedResponse := &ai.ParseResponse{
		Title:       "Attack on Titan",
		Year:        2013,
		Season:      1,
		Episode:     1,
		MediaType:   "tv",
		Quality:     "1080p",
		FansubGroup: "SubsPlease",
		Confidence:  0.95,
	}

	provider := &mockProvider{
		name: ai.ProviderGemini,
		parseFunc: func(ctx context.Context, req *ai.ParseRequest) (*ai.ParseResponse, error) {
			return expectedResponse, nil
		},
	}
	cache := ai.NewCache(db)
	service := NewAIServiceWithProvider(provider, cache)

	ctx := context.Background()
	filename := "[SubsPlease] Shingeki no Kyojin - 01 [1080p].mkv"

	result, err := service.ParseFilename(ctx, filename)

	require.NoError(t, err)
	assert.Equal(t, expectedResponse.Title, result.Title)
	assert.Equal(t, expectedResponse.Season, result.Season)
	assert.Equal(t, expectedResponse.Episode, result.Episode)
	assert.True(t, provider.parseCalled)
}

func TestAIService_ParseFilename_CacheHit(t *testing.T) {
	db := setupTestAIDB(t)
	defer db.Close()

	cachedResponse := &ai.ParseResponse{
		Title:     "Cached Title",
		MediaType: "movie",
	}

	provider := &mockProvider{name: ai.ProviderGemini}
	cache := ai.NewCache(db)
	service := NewAIServiceWithProvider(provider, cache)

	ctx := context.Background()
	filename := "test.mkv"

	// Pre-populate cache
	err := cache.Set(ctx, filename, ai.ProviderGemini, "test", cachedResponse)
	require.NoError(t, err)

	// Parse should hit cache
	result, err := service.ParseFilename(ctx, filename)

	require.NoError(t, err)
	assert.Equal(t, "Cached Title", result.Title)
	assert.False(t, provider.parseCalled) // Should not call provider
}

func TestAIService_ParseFilename_CachesResult(t *testing.T) {
	db := setupTestAIDB(t)
	defer db.Close()

	callCount := 0
	provider := &mockProvider{
		name: ai.ProviderGemini,
		parseFunc: func(ctx context.Context, req *ai.ParseRequest) (*ai.ParseResponse, error) {
			callCount++
			return &ai.ParseResponse{
				Title:     "Parsed Title",
				MediaType: "movie",
			}, nil
		},
	}
	cache := ai.NewCache(db)
	service := NewAIServiceWithProvider(provider, cache)

	ctx := context.Background()
	filename := "test.mkv"

	// First call - should parse and cache
	_, err := service.ParseFilename(ctx, filename)
	require.NoError(t, err)
	assert.Equal(t, 1, callCount)

	// Create new service with same cache
	provider2 := &mockProvider{name: ai.ProviderGemini}
	service2 := NewAIServiceWithProvider(provider2, cache)

	// Second call - should hit cache
	result, err := service2.ParseFilename(ctx, filename)
	require.NoError(t, err)
	assert.Equal(t, "Parsed Title", result.Title)
	assert.False(t, provider2.parseCalled) // Should hit cache
}

func TestAIService_ParseFilename_ProviderError(t *testing.T) {
	db := setupTestAIDB(t)
	defer db.Close()

	expectedErr := errors.New("provider error")
	provider := &mockProvider{
		name: ai.ProviderGemini,
		parseFunc: func(ctx context.Context, req *ai.ParseRequest) (*ai.ParseResponse, error) {
			return nil, expectedErr
		},
	}
	cache := ai.NewCache(db)
	service := NewAIServiceWithProvider(provider, cache)

	ctx := context.Background()

	_, err := service.ParseFilename(ctx, "test.mkv")

	assert.Error(t, err)
	assert.Equal(t, expectedErr, err)
}

func TestAIService_ParseFilename_NotConfigured(t *testing.T) {
	service := &AIService{
		provider: nil,
	}

	_, err := service.ParseFilename(context.Background(), "test.mkv")

	assert.ErrorIs(t, err, ai.ErrAINotConfigured)
}

func TestAIService_ClearCache(t *testing.T) {
	db := setupTestAIDB(t)
	defer db.Close()

	provider := &mockProvider{name: ai.ProviderGemini}
	cache := ai.NewCache(db)
	service := NewAIServiceWithProvider(provider, cache)

	ctx := context.Background()

	// Add some cache entries
	for i := 0; i < 3; i++ {
		err := cache.Set(ctx, "file"+string(rune('0'+i))+".mkv", ai.ProviderGemini, "test",
			&ai.ParseResponse{Title: "Test", MediaType: "movie"})
		require.NoError(t, err)
	}

	// Clear cache
	count, err := service.ClearCache(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(3), count)

	// Verify cache is empty
	stats, err := service.GetCacheStats(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(0), stats.TotalEntries)
}

func TestAIService_GetCacheStats(t *testing.T) {
	db := setupTestAIDB(t)
	defer db.Close()

	provider := &mockProvider{name: ai.ProviderGemini}
	cache := ai.NewCache(db)
	service := NewAIServiceWithProvider(provider, cache)

	ctx := context.Background()

	// Add cache entries
	for i := 0; i < 5; i++ {
		err := cache.Set(ctx, "file"+string(rune('0'+i))+".mkv", ai.ProviderGemini, "test",
			&ai.ParseResponse{Title: "Test", MediaType: "movie"})
		require.NoError(t, err)
	}

	stats, err := service.GetCacheStats(ctx)

	require.NoError(t, err)
	assert.Equal(t, int64(5), stats.TotalEntries)
	assert.Equal(t, int64(5), stats.ValidEntries)
	assert.Equal(t, int64(0), stats.ExpiredEntries)
}

func TestAIService_IsConfigured(t *testing.T) {
	t.Run("configured", func(t *testing.T) {
		db := setupTestAIDB(t)
		defer db.Close()

		provider := &mockProvider{name: ai.ProviderGemini}
		cache := ai.NewCache(db)
		service := NewAIServiceWithProvider(provider, cache)

		assert.True(t, service.IsConfigured())
	})

	t.Run("not configured", func(t *testing.T) {
		service := &AIService{provider: nil}
		assert.False(t, service.IsConfigured())
	})
}

func TestAIService_GetProviderName(t *testing.T) {
	t.Run("gemini", func(t *testing.T) {
		db := setupTestAIDB(t)
		defer db.Close()

		provider := &mockProvider{name: ai.ProviderGemini}
		service := NewAIServiceWithProvider(provider, ai.NewCache(db))

		assert.Equal(t, "gemini", service.GetProviderName())
	})

	t.Run("claude", func(t *testing.T) {
		db := setupTestAIDB(t)
		defer db.Close()

		provider := &mockProvider{name: ai.ProviderClaude}
		service := NewAIServiceWithProvider(provider, ai.NewCache(db))

		assert.Equal(t, "claude", service.GetProviderName())
	})

	t.Run("not configured", func(t *testing.T) {
		service := &AIService{provider: nil}
		assert.Equal(t, "", service.GetProviderName())
	})
}
