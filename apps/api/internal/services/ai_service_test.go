package services

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

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

// Tests for ParseFansubFilename (Story 3.2)

func TestAIService_ParseFansubFilename_Success(t *testing.T) {
	db := setupTestAIDB(t)
	defer db.Close()

	expectedResponse := &ai.ParseResponse{
		Title:       "我的英雄學院",
		Season:      1,
		Episode:     1,
		MediaType:   "tv",
		Quality:     "1080p",
		FansubGroup: "幻櫻字幕組",
		Language:    "Traditional Chinese",
		Confidence:  0.95,
	}

	var capturedPrompt string
	provider := &mockProvider{
		name: ai.ProviderGemini,
		parseFunc: func(ctx context.Context, req *ai.ParseRequest) (*ai.ParseResponse, error) {
			capturedPrompt = req.Prompt
			return expectedResponse, nil
		},
	}
	cache := ai.NewCache(db)
	service := NewAIServiceWithProvider(provider, cache)

	ctx := context.Background()
	filename := "【幻櫻字幕組】【4月新番】我的英雄學院 第01話 1080P【繁體】.mp4"

	result, err := service.ParseFansubFilename(ctx, filename)

	require.NoError(t, err)
	assert.Equal(t, expectedResponse.Title, result.Title)
	assert.Equal(t, expectedResponse.FansubGroup, result.FansubGroup)
	assert.Equal(t, expectedResponse.Language, result.Language)
	assert.True(t, provider.parseCalled)
	// Verify fansub prompt was used
	assert.Contains(t, capturedPrompt, "fansub releases")
	assert.Contains(t, capturedPrompt, filename)
}

func TestAIService_ParseFansubFilename_JapaneseFansub(t *testing.T) {
	db := setupTestAIDB(t)
	defer db.Close()

	expectedResponse := &ai.ParseResponse{
		Title:       "Kimetsu no Yaiba",
		Season:      1,
		Episode:     26,
		MediaType:   "tv",
		Quality:     "1080p",
		Source:      "BD",
		Codec:       "x264",
		FansubGroup: "Leopard-Raws",
		Confidence:  0.9,
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
	filename := "[Leopard-Raws] Kimetsu no Yaiba - 26 (BD 1920x1080 x264 FLAC).mkv"

	result, err := service.ParseFansubFilename(ctx, filename)

	require.NoError(t, err)
	assert.Equal(t, "Kimetsu no Yaiba", result.Title)
	assert.Equal(t, 26, result.Episode)
	assert.Equal(t, "BD", result.Source)
	assert.Equal(t, "Leopard-Raws", result.FansubGroup)
}

func TestAIService_ParseFansubFilename_CacheHit(t *testing.T) {
	db := setupTestAIDB(t)
	defer db.Close()

	cachedResponse := &ai.ParseResponse{
		Title:       "Cached Fansub Title",
		MediaType:   "tv",
		FansubGroup: "CachedGroup",
	}

	provider := &mockProvider{name: ai.ProviderGemini}
	cache := ai.NewCache(db)
	service := NewAIServiceWithProvider(provider, cache)

	ctx := context.Background()
	filename := "[Test] Anime - 01.mkv"

	// Pre-populate cache with fansub prefix
	cacheKey := "fansub:" + filename
	err := cache.Set(ctx, cacheKey, ai.ProviderGemini, "fansub", cachedResponse)
	require.NoError(t, err)

	// Parse should hit cache
	result, err := service.ParseFansubFilename(ctx, filename)

	require.NoError(t, err)
	assert.Equal(t, "Cached Fansub Title", result.Title)
	assert.Equal(t, "CachedGroup", result.FansubGroup)
	assert.False(t, provider.parseCalled) // Should not call provider
}

func TestAIService_ParseFansubFilename_CachesResult(t *testing.T) {
	db := setupTestAIDB(t)
	defer db.Close()

	callCount := 0
	provider := &mockProvider{
		name: ai.ProviderGemini,
		parseFunc: func(ctx context.Context, req *ai.ParseRequest) (*ai.ParseResponse, error) {
			callCount++
			return &ai.ParseResponse{
				Title:       "Parsed Fansub",
				MediaType:   "tv",
				FansubGroup: "TestGroup",
			}, nil
		},
	}
	cache := ai.NewCache(db)
	service := NewAIServiceWithProvider(provider, cache)

	ctx := context.Background()
	filename := "[TestGroup] Show - 01.mkv"

	// First call - should parse and cache
	_, err := service.ParseFansubFilename(ctx, filename)
	require.NoError(t, err)
	assert.Equal(t, 1, callCount)

	// Create new service with same cache
	provider2 := &mockProvider{name: ai.ProviderGemini}
	service2 := NewAIServiceWithProvider(provider2, cache)

	// Second call - should hit cache
	result, err := service2.ParseFansubFilename(ctx, filename)
	require.NoError(t, err)
	assert.Equal(t, "Parsed Fansub", result.Title)
	assert.False(t, provider2.parseCalled) // Should hit cache
}

func TestAIService_ParseFansubFilename_LowConfidence(t *testing.T) {
	db := setupTestAIDB(t)
	defer db.Close()

	// Returns low confidence result
	lowConfidenceResponse := &ai.ParseResponse{
		Title:      "Uncertain Title",
		MediaType:  "tv",
		Confidence: 0.2, // Below 0.3 threshold
	}

	provider := &mockProvider{
		name: ai.ProviderGemini,
		parseFunc: func(ctx context.Context, req *ai.ParseRequest) (*ai.ParseResponse, error) {
			return lowConfidenceResponse, nil
		},
	}
	cache := ai.NewCache(db)
	service := NewAIServiceWithProvider(provider, cache)

	ctx := context.Background()
	filename := "ambiguous_filename.mkv"

	// Should still return result (with warning logged)
	result, err := service.ParseFansubFilename(ctx, filename)

	require.NoError(t, err)
	assert.Equal(t, "Uncertain Title", result.Title)
	assert.Equal(t, 0.2, result.Confidence)
}

func TestAIService_ParseFansubFilename_ProviderError(t *testing.T) {
	db := setupTestAIDB(t)
	defer db.Close()

	expectedErr := errors.New("AI provider error")
	provider := &mockProvider{
		name: ai.ProviderGemini,
		parseFunc: func(ctx context.Context, req *ai.ParseRequest) (*ai.ParseResponse, error) {
			return nil, expectedErr
		},
	}
	cache := ai.NewCache(db)
	service := NewAIServiceWithProvider(provider, cache)

	ctx := context.Background()

	_, err := service.ParseFansubFilename(ctx, "[Group] Test - 01.mkv")

	assert.Error(t, err)
	assert.Equal(t, expectedErr, err)
}

func TestAIService_ParseFansubFilename_NotConfigured(t *testing.T) {
	service := &AIService{
		provider: nil,
	}

	_, err := service.ParseFansubFilename(context.Background(), "[Test] Anime - 01.mkv")

	assert.ErrorIs(t, err, ai.ErrAINotConfigured)
}

func TestAIService_ParseFansubFilename_UsesSeparateCacheFromGeneric(t *testing.T) {
	db := setupTestAIDB(t)
	defer db.Close()

	// Generic parse returns different result
	genericResponse := &ai.ParseResponse{
		Title:     "Generic Parse Result",
		MediaType: "movie",
	}

	// Fansub parse returns different result
	fansubResponse := &ai.ParseResponse{
		Title:       "Fansub Parse Result",
		MediaType:   "tv",
		FansubGroup: "SubGroup",
	}

	callNum := 0
	provider := &mockProvider{
		name: ai.ProviderGemini,
		parseFunc: func(ctx context.Context, req *ai.ParseRequest) (*ai.ParseResponse, error) {
			callNum++
			if req.Prompt != "" {
				// Fansub prompt provided
				return fansubResponse, nil
			}
			return genericResponse, nil
		},
	}
	cache := ai.NewCache(db)
	service := NewAIServiceWithProvider(provider, cache)

	ctx := context.Background()
	filename := "[Test] Show - 01.mkv"

	// Call generic parse
	result1, err := service.ParseFilename(ctx, filename)
	require.NoError(t, err)
	assert.Equal(t, "Generic Parse Result", result1.Title)

	// Call fansub parse - should not hit generic cache
	result2, err := service.ParseFansubFilename(ctx, filename)
	require.NoError(t, err)
	assert.Equal(t, "Fansub Parse Result", result2.Title)
	assert.Equal(t, "SubGroup", result2.FansubGroup)

	// Verify both were called (separate cache keys)
	assert.Equal(t, 2, callNum)
}

func TestAIService_ParseFansubFilename_Timeout(t *testing.T) {
	db := setupTestAIDB(t)
	defer db.Close()

	// Provider that takes longer than timeout
	provider := &mockProvider{
		name: ai.ProviderGemini,
		parseFunc: func(ctx context.Context, req *ai.ParseRequest) (*ai.ParseResponse, error) {
			// Check if context has deadline
			deadline, ok := ctx.Deadline()
			if !ok {
				t.Error("Expected context to have deadline for fansub parsing")
			}

			// Verify timeout is approximately 10 seconds
			timeout := time.Until(deadline)
			assert.True(t, timeout > 9*time.Second && timeout <= 10*time.Second,
				"Timeout should be ~10 seconds for fansub parsing, got %v", timeout)

			return &ai.ParseResponse{
				Title:     "Test",
				MediaType: "tv",
			}, nil
		},
	}
	cache := ai.NewCache(db)
	service := NewAIServiceWithProvider(provider, cache)

	ctx := context.Background()
	_, err := service.ParseFansubFilename(ctx, "[Test] Anime - 01.mkv")

	require.NoError(t, err)
}

func TestFansubParsingTimeout_Constant(t *testing.T) {
	// Verify the timeout constant matches NFR-P14 requirement
	assert.Equal(t, 10*time.Second, FansubParsingTimeout, "Fansub parsing timeout should be 10 seconds per NFR-P14")
}

// Tests for GenerateKeywords (Story 3.6)

func TestAIService_GenerateKeywords_Success(t *testing.T) {
	db := setupTestAIDB(t)
	defer db.Close()

	// Mock provider that returns keyword generation response
	provider := &mockProvider{
		name: ai.ProviderGemini,
		parseFunc: func(ctx context.Context, req *ai.ParseRequest) (*ai.ParseResponse, error) {
			// Return a response with RawResponse containing keyword JSON
			return &ai.ParseResponse{
				Title:     "鬼滅之刃",
				MediaType: "tv",
				RawResponse: `{
					"original": "鬼滅之刃",
					"simplified_chinese": "鬼灭之刃",
					"traditional_chinese": "鬼滅之刃",
					"english": "Demon Slayer",
					"romaji": "Kimetsu no Yaiba",
					"alternative_spellings": ["Demon Slayer: Kimetsu no Yaiba"],
					"common_aliases": ["鬼滅"]
				}`,
			}, nil
		},
	}
	cache := ai.NewCache(db)
	service := NewAIServiceWithProvider(provider, cache)

	ctx := context.Background()

	result, err := service.GenerateKeywords(ctx, "鬼滅之刃", "Traditional Chinese")

	require.NoError(t, err)
	assert.Equal(t, "鬼滅之刃", result.Original)
	assert.Equal(t, "鬼灭之刃", result.SimplifiedChinese)
	assert.Equal(t, "Demon Slayer", result.English)
	assert.Equal(t, "Kimetsu no Yaiba", result.Romaji)
	assert.Len(t, result.AlternativeSpellings, 1)
	assert.Len(t, result.CommonAliases, 1)
}

func TestAIService_GenerateKeywords_CacheHit(t *testing.T) {
	db := setupTestAIDB(t)
	defer db.Close()

	cachedKeywords := &ai.KeywordVariants{
		Original:          "Test",
		English:           "Test English",
		SimplifiedChinese: "测试",
	}

	// Serialize to response for cache
	cachedResponse := &ai.ParseResponse{
		Title:       "Test",
		MediaType:   "tv",
		RawResponse: `{"original":"Test","english":"Test English","simplified_chinese":"测试"}`,
	}

	provider := &mockProvider{name: ai.ProviderGemini}
	cache := ai.NewCache(db)
	service := NewAIServiceWithProvider(provider, cache)

	ctx := context.Background()
	title := "Test"

	// Pre-populate cache with keyword prefix
	cacheKey := "keywords:" + title
	err := cache.Set(ctx, cacheKey, ai.ProviderGemini, "keywords", cachedResponse)
	require.NoError(t, err)

	// Generate should hit cache
	result, err := service.GenerateKeywords(ctx, title, "English")

	require.NoError(t, err)
	assert.Equal(t, cachedKeywords.Original, result.Original)
	assert.Equal(t, cachedKeywords.English, result.English)
	assert.False(t, provider.parseCalled) // Should not call provider
}

func TestAIService_GenerateKeywords_CachesResult(t *testing.T) {
	db := setupTestAIDB(t)
	defer db.Close()

	callCount := 0
	provider := &mockProvider{
		name: ai.ProviderGemini,
		parseFunc: func(ctx context.Context, req *ai.ParseRequest) (*ai.ParseResponse, error) {
			callCount++
			return &ai.ParseResponse{
				Title:     "Test",
				MediaType: "tv",
				RawResponse: `{
					"original": "Test",
					"english": "Test Movie"
				}`,
			}, nil
		},
	}
	cache := ai.NewCache(db)
	service := NewAIServiceWithProvider(provider, cache)

	ctx := context.Background()
	title := "Test"

	// First call - should call AI and cache
	_, err := service.GenerateKeywords(ctx, title, "English")
	require.NoError(t, err)
	assert.Equal(t, 1, callCount)

	// Create new service with same cache
	provider2 := &mockProvider{name: ai.ProviderGemini}
	service2 := NewAIServiceWithProvider(provider2, cache)

	// Second call - should hit cache
	result, err := service2.GenerateKeywords(ctx, title, "English")
	require.NoError(t, err)
	assert.Equal(t, "Test Movie", result.English)
	assert.False(t, provider2.parseCalled) // Should hit cache
}

func TestAIService_GenerateKeywords_NotConfigured(t *testing.T) {
	service := &AIService{
		provider: nil,
	}

	_, err := service.GenerateKeywords(context.Background(), "Test", "English")

	assert.ErrorIs(t, err, ai.ErrAINotConfigured)
}

func TestAIService_GenerateKeywords_ProviderError(t *testing.T) {
	db := setupTestAIDB(t)
	defer db.Close()

	expectedErr := errors.New("AI provider error")
	provider := &mockProvider{
		name: ai.ProviderGemini,
		parseFunc: func(ctx context.Context, req *ai.ParseRequest) (*ai.ParseResponse, error) {
			return nil, expectedErr
		},
	}
	cache := ai.NewCache(db)
	service := NewAIServiceWithProvider(provider, cache)

	ctx := context.Background()

	_, err := service.GenerateKeywords(ctx, "Test", "English")

	assert.Error(t, err)
	assert.Equal(t, expectedErr, err)
}

func TestAIService_GenerateKeywords_EmptyTitle(t *testing.T) {
	db := setupTestAIDB(t)
	defer db.Close()

	provider := &mockProvider{name: ai.ProviderGemini}
	cache := ai.NewCache(db)
	service := NewAIServiceWithProvider(provider, cache)

	ctx := context.Background()

	_, err := service.GenerateKeywords(ctx, "", "English")

	assert.Error(t, err)
	assert.False(t, provider.parseCalled)
}

func TestAIService_GenerateKeywords_UsesKeywordPrompt(t *testing.T) {
	db := setupTestAIDB(t)
	defer db.Close()

	var capturedPrompt string
	provider := &mockProvider{
		name: ai.ProviderGemini,
		parseFunc: func(ctx context.Context, req *ai.ParseRequest) (*ai.ParseResponse, error) {
			capturedPrompt = req.Prompt
			return &ai.ParseResponse{
				Title:       "Test",
				MediaType:   "tv",
				RawResponse: `{"original":"Test","english":"Test English"}`,
			}, nil
		},
	}
	cache := ai.NewCache(db)
	service := NewAIServiceWithProvider(provider, cache)

	ctx := context.Background()

	_, err := service.GenerateKeywords(ctx, "鬼滅之刃", "Traditional Chinese")

	require.NoError(t, err)
	// Verify keyword generation prompt was used
	assert.Contains(t, capturedPrompt, "keyword generator")
	assert.Contains(t, capturedPrompt, "鬼滅之刃")
	assert.Contains(t, capturedPrompt, "Traditional Chinese")
}

func TestKeywordGenerationTimeout_Constant(t *testing.T) {
	// Keyword generation should use same 10-second timeout
	assert.Equal(t, 10*time.Second, KeywordGenerationTimeout, "Keyword generation timeout should be 10 seconds")
}
