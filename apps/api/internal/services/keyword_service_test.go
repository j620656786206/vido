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

// [P1] Tests GenerateKeywords returns ErrAINotConfigured when AI service is nil
func TestKeywordService_GenerateKeywords_NilAIService(t *testing.T) {
	// GIVEN: A keyword service with nil AI service
	service := NewKeywordService(nil)
	ctx := context.Background()

	// WHEN: Generating keywords
	result, err := service.GenerateKeywords(ctx, "Test Title")

	// THEN: Should return ErrAINotConfigured error
	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Equal(t, ai.ErrAINotConfigured, err)
}

// [P2] Tests convertToMetadataVariants handles nil input
func TestKeywordService_ConvertToMetadataVariants_Nil(t *testing.T) {
	// GIVEN: A nil KeywordVariants input
	var aiVariants *ai.KeywordVariants = nil

	// WHEN: Converting to metadata variants
	result := convertToMetadataVariants(aiVariants)

	// THEN: Should return nil without panic
	assert.Nil(t, result)
}

// [P2] Tests convertToMetadataVariants handles empty slices
func TestKeywordService_ConvertToMetadataVariants_EmptySlices(t *testing.T) {
	// GIVEN: KeywordVariants with empty slices
	aiVariants := &ai.KeywordVariants{
		Original:             "Test",
		English:              "Test English",
		AlternativeSpellings: []string{},
		CommonAliases:        []string{},
	}

	// WHEN: Converting to metadata variants
	metaVariants := convertToMetadataVariants(aiVariants)

	// THEN: Should preserve empty slices
	assert.NotNil(t, metaVariants)
	assert.Equal(t, "Test", metaVariants.Original)
	assert.Equal(t, "Test English", metaVariants.English)
	assert.Empty(t, metaVariants.AlternativeSpellings)
	assert.Empty(t, metaVariants.CommonAliases)
}

// [P2] Tests GenerateKeywords passes correct language to AI service
func TestKeywordService_GenerateKeywords_LanguageDetection(t *testing.T) {
	// GIVEN: A mock AI service that captures the language parameter
	var capturedLanguage string
	aiService := &mockAIServiceForKeywords{
		generateFunc: func(ctx context.Context, title, language string) (*ai.KeywordVariants, error) {
			capturedLanguage = language
			return &ai.KeywordVariants{Original: title}, nil
		},
	}

	service := NewKeywordService(aiService)
	ctx := context.Background()

	// WHEN: Generating keywords for a Traditional Chinese title
	_, err := service.GenerateKeywords(ctx, "鬼滅之刃")

	// THEN: Should detect and pass Traditional Chinese language
	require.NoError(t, err)
	assert.Equal(t, "Traditional Chinese", capturedLanguage)
}

// [P2] Tests GenerateKeywords with Japanese title
func TestKeywordService_GenerateKeywords_JapaneseTitle(t *testing.T) {
	// GIVEN: A mock AI service that captures the language parameter
	var capturedLanguage string
	aiService := &mockAIServiceForKeywords{
		generateFunc: func(ctx context.Context, title, language string) (*ai.KeywordVariants, error) {
			capturedLanguage = language
			return &ai.KeywordVariants{Original: title, Romaji: "tesuto"}, nil
		},
	}

	service := NewKeywordService(aiService)
	ctx := context.Background()

	// WHEN: Generating keywords for a Japanese title (with hiragana)
	_, err := service.GenerateKeywords(ctx, "テスト")

	// THEN: Should detect and pass Japanese language
	require.NoError(t, err)
	assert.Equal(t, "Japanese", capturedLanguage)
}
