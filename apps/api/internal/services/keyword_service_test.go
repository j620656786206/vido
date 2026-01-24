package services

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vido/api/internal/ai"
	"github.com/vido/api/internal/metadata"
)

// mockAIServiceForKeywords implements AIServiceInterface for keyword testing
type mockAIServiceForKeywords struct {
	generateFunc func(ctx context.Context, title, language string) (*ai.KeywordVariants, error)
}

func (m *mockAIServiceForKeywords) ParseFilename(ctx context.Context, filename string) (*ai.ParseResponse, error) {
	return nil, nil
}

func (m *mockAIServiceForKeywords) ParseFansubFilename(ctx context.Context, filename string) (*ai.ParseResponse, error) {
	return nil, nil
}

func (m *mockAIServiceForKeywords) GenerateKeywords(ctx context.Context, title, language string) (*ai.KeywordVariants, error) {
	if m.generateFunc != nil {
		return m.generateFunc(ctx, title, language)
	}
	return &ai.KeywordVariants{Original: title}, nil
}

func (m *mockAIServiceForKeywords) ClearCache(ctx context.Context) (int64, error) {
	return 0, nil
}

func (m *mockAIServiceForKeywords) ClearExpiredCache(ctx context.Context) (int64, error) {
	return 0, nil
}

func (m *mockAIServiceForKeywords) GetCacheStats(ctx context.Context) (*ai.CacheStats, error) {
	return &ai.CacheStats{}, nil
}

func (m *mockAIServiceForKeywords) IsConfigured() bool {
	return true
}

func (m *mockAIServiceForKeywords) GetProviderName() string {
	return "mock"
}

func TestNewKeywordService(t *testing.T) {
	aiService := &mockAIServiceForKeywords{}
	service := NewKeywordService(aiService)

	require.NotNil(t, service)
}

func TestKeywordService_GenerateKeywords_Success(t *testing.T) {
	expectedVariants := &ai.KeywordVariants{
		Original:          "鬼滅之刃",
		English:           "Demon Slayer",
		SimplifiedChinese: "鬼灭之刃",
		Romaji:            "Kimetsu no Yaiba",
	}

	aiService := &mockAIServiceForKeywords{
		generateFunc: func(ctx context.Context, title, language string) (*ai.KeywordVariants, error) {
			return expectedVariants, nil
		},
	}

	service := NewKeywordService(aiService)
	ctx := context.Background()

	result, err := service.GenerateKeywords(ctx, "鬼滅之刃")

	require.NoError(t, err)
	assert.Equal(t, "鬼滅之刃", result.Original)
	assert.Equal(t, "Demon Slayer", result.English)
	assert.Equal(t, "鬼灭之刃", result.SimplifiedChinese)
}

func TestKeywordService_GenerateKeywords_Error(t *testing.T) {
	expectedErr := errors.New("AI service error")
	aiService := &mockAIServiceForKeywords{
		generateFunc: func(ctx context.Context, title, language string) (*ai.KeywordVariants, error) {
			return nil, expectedErr
		},
	}

	service := NewKeywordService(aiService)
	ctx := context.Background()

	_, err := service.GenerateKeywords(ctx, "Test")

	assert.Error(t, err)
	assert.Equal(t, expectedErr, err)
}

func TestKeywordService_ImplementsKeywordGenerator(t *testing.T) {
	aiService := &mockAIServiceForKeywords{}
	service := NewKeywordService(aiService)

	// Verify KeywordService implements metadata.KeywordGenerator
	var _ metadata.KeywordGenerator = service
}

func TestKeywordService_IsConfigured(t *testing.T) {
	t.Run("configured", func(t *testing.T) {
		aiService := &mockAIServiceForKeywords{}
		service := NewKeywordService(aiService)

		assert.True(t, service.IsConfigured())
	})

	t.Run("not configured - nil ai service", func(t *testing.T) {
		service := NewKeywordService(nil)

		assert.False(t, service.IsConfigured())
	})
}

func TestKeywordService_ConvertToMetadataVariants(t *testing.T) {
	aiVariants := &ai.KeywordVariants{
		Original:           "Test",
		English:            "Test English",
		SimplifiedChinese:  "测试",
		TraditionalChinese: "測試",
		Romaji:             "tesuto",
		AlternativeSpellings: []string{"alt1", "alt2"},
		CommonAliases:      []string{"alias1"},
	}

	metaVariants := convertToMetadataVariants(aiVariants)

	assert.Equal(t, aiVariants.Original, metaVariants.Original)
	assert.Equal(t, aiVariants.English, metaVariants.English)
	assert.Equal(t, aiVariants.SimplifiedChinese, metaVariants.SimplifiedChinese)
	assert.Equal(t, aiVariants.TraditionalChinese, metaVariants.TraditionalChinese)
	assert.Equal(t, aiVariants.Romaji, metaVariants.Romaji)
	assert.Equal(t, aiVariants.AlternativeSpellings, metaVariants.AlternativeSpellings)
	assert.Equal(t, aiVariants.CommonAliases, metaVariants.CommonAliases)
}
