package services

import (
	"context"
	"database/sql"
	"log/slog"

	"github.com/vido/api/internal/ai"
	"github.com/vido/api/internal/config"
)

// AIServiceInterface defines the contract for AI parsing services.
type AIServiceInterface interface {
	// ParseFilename sends a filename to AI for parsing.
	// Uses cache-first strategy with 30-day TTL.
	ParseFilename(ctx context.Context, filename string) (*ai.ParseResponse, error)

	// ClearCache removes all cached AI parsing results.
	ClearCache(ctx context.Context) (int64, error)

	// ClearExpiredCache removes expired cached entries.
	ClearExpiredCache(ctx context.Context) (int64, error)

	// GetCacheStats returns cache statistics.
	GetCacheStats(ctx context.Context) (*ai.CacheStats, error)

	// IsConfigured returns true if an AI provider is configured.
	IsConfigured() bool
}

// AIService orchestrates AI parsing with caching.
type AIService struct {
	provider ai.Provider
	cache    *ai.Cache
	cfg      *config.Config
}

// Compile-time interface verification.
var _ AIServiceInterface = (*AIService)(nil)

// NewAIService creates a new AI service with the configured provider and cache.
// Returns nil if no AI provider is configured.
func NewAIService(cfg *config.Config, db *sql.DB) (*AIService, error) {
	if !cfg.HasAIProvider() {
		slog.Info("AI service not configured - no API keys set")
		return nil, nil
	}

	// Create provider using factory
	factoryCfg := ai.FactoryConfig{
		ProviderName: cfg.GetAIProvider(),
		GeminiAPIKey: cfg.GetGeminiAPIKey(),
		ClaudeAPIKey: cfg.GetClaudeAPIKey(),
	}

	provider, err := ai.NewProvider(factoryCfg)
	if err != nil {
		return nil, err
	}

	// Create cache
	cache := ai.NewCache(db)

	return &AIService{
		provider: provider,
		cache:    cache,
		cfg:      cfg,
	}, nil
}

// NewAIServiceWithProvider creates an AI service with a specific provider (for testing).
func NewAIServiceWithProvider(provider ai.Provider, cache *ai.Cache) *AIService {
	return &AIService{
		provider: provider,
		cache:    cache,
	}
}

// ParseFilename parses a filename using AI with cache-first strategy.
func (s *AIService) ParseFilename(ctx context.Context, filename string) (*ai.ParseResponse, error) {
	if s.provider == nil {
		return nil, ai.ErrAINotConfigured
	}

	// Check cache first
	cached, err := s.cache.Get(ctx, filename)
	if err != nil {
		slog.Warn("Cache lookup failed, proceeding with AI",
			"error", err,
			"filename", filename,
		)
		// Continue to AI parsing
	} else if cached != nil {
		slog.Debug("Cache hit for AI parsing",
			"filename", filename,
			"title", cached.Title,
		)
		return cached, nil
	}

	// Parse with AI
	req := &ai.ParseRequest{
		Filename: filename,
	}

	response, err := s.provider.Parse(ctx, req)
	if err != nil {
		return nil, err
	}

	// Cache the result
	if err := s.cache.Set(ctx, filename, s.provider.Name(), "", response); err != nil {
		slog.Warn("Failed to cache AI response",
			"error", err,
			"filename", filename,
		)
		// Don't fail the request if caching fails
	}

	return response, nil
}

// ClearCache removes all cached AI parsing results.
func (s *AIService) ClearCache(ctx context.Context) (int64, error) {
	return s.cache.ClearAll(ctx)
}

// ClearExpiredCache removes expired cached entries.
func (s *AIService) ClearExpiredCache(ctx context.Context) (int64, error) {
	return s.cache.ClearExpired(ctx)
}

// GetCacheStats returns cache statistics.
func (s *AIService) GetCacheStats(ctx context.Context) (*ai.CacheStats, error) {
	return s.cache.Stats(ctx)
}

// IsConfigured returns true if an AI provider is configured.
func (s *AIService) IsConfigured() bool {
	return s.provider != nil
}

// GetProviderName returns the name of the configured provider.
func (s *AIService) GetProviderName() string {
	if s.provider == nil {
		return ""
	}
	return string(s.provider.Name())
}
