package services

import (
	"context"
	"log/slog"

	"github.com/vido/api/internal/ai"
	"github.com/vido/api/internal/metadata"
)

// KeywordServiceInterface defines the contract for keyword generation services.
type KeywordServiceInterface interface {
	// GenerateKeywords generates alternative search keywords for a media title.
	// Implements metadata.KeywordGenerator interface.
	GenerateKeywords(ctx context.Context, title string) (*metadata.KeywordVariants, error)

	// IsConfigured returns true if the service is properly configured.
	IsConfigured() bool
}

// KeywordService provides keyword generation functionality.
// It wraps the AI service and implements metadata.KeywordGenerator interface.
type KeywordService struct {
	aiService AIServiceInterface
}

// Compile-time interface verification.
var _ KeywordServiceInterface = (*KeywordService)(nil)
var _ metadata.KeywordGenerator = (*KeywordService)(nil)

// NewKeywordService creates a new keyword service.
func NewKeywordService(aiService AIServiceInterface) *KeywordService {
	return &KeywordService{
		aiService: aiService,
	}
}

// GenerateKeywords generates alternative search keywords for a media title.
// It calls the AI service to generate keyword variants and converts them
// to the metadata package's KeywordVariants type.
// Uses automatic language detection to provide better hints to the AI.
func (s *KeywordService) GenerateKeywords(ctx context.Context, title string) (*metadata.KeywordVariants, error) {
	if s.aiService == nil {
		slog.Warn("Keyword service called but AI service is not configured")
		return nil, ai.ErrAINotConfigured
	}

	// Detect language for better AI prompt
	detectedLang := ai.DetectLanguage(title)
	language := detectedLang.String()

	slog.Debug("Detected language for keyword generation",
		"title", title,
		"language", language,
	)

	// Generate keywords using AI service with detected language
	aiVariants, err := s.aiService.GenerateKeywords(ctx, title, language)
	if err != nil {
		slog.Error("Failed to generate keywords",
			"error", err,
			"title", title,
			"language", language,
		)
		return nil, err
	}

	// Convert to metadata package type
	result := convertToMetadataVariants(aiVariants)

	slog.Debug("Generated keywords for title",
		"title", title,
		"language", language,
		"keyword_count", len(result.GetPrioritizedList()),
	)

	return result, nil
}

// IsConfigured returns true if the AI service is configured.
func (s *KeywordService) IsConfigured() bool {
	return s.aiService != nil && s.aiService.IsConfigured()
}

// convertToMetadataVariants converts AI package KeywordVariants to metadata package KeywordVariants.
// This is needed because we have two KeywordVariants types to avoid circular imports.
func convertToMetadataVariants(aiVariants *ai.KeywordVariants) *metadata.KeywordVariants {
	if aiVariants == nil {
		return nil
	}

	// Copy alternative spellings
	altSpellings := make([]string, len(aiVariants.AlternativeSpellings))
	copy(altSpellings, aiVariants.AlternativeSpellings)

	// Copy common aliases
	aliases := make([]string, len(aiVariants.CommonAliases))
	copy(aliases, aiVariants.CommonAliases)

	return &metadata.KeywordVariants{
		Original:             aiVariants.Original,
		SimplifiedChinese:    aiVariants.SimplifiedChinese,
		TraditionalChinese:   aiVariants.TraditionalChinese,
		English:              aiVariants.English,
		Romaji:               aiVariants.Romaji,
		Pinyin:               aiVariants.Pinyin,
		AlternativeSpellings: altSpellings,
		CommonAliases:        aliases,
	}
}
