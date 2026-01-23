package services

import (
	"context"
	"database/sql"
	"log/slog"
	"time"

	"github.com/vido/api/internal/ai"
	"github.com/vido/api/internal/ai/prompts"
	"github.com/vido/api/internal/config"
)

const (
	// FansubParsingTimeout is the maximum time for fansub AI parsing (NFR-P14).
	FansubParsingTimeout = 10 * time.Second
)

// AIServiceInterface defines the contract for AI parsing services.
type AIServiceInterface interface {
	// ParseFilename sends a filename to AI for parsing.
	// Uses cache-first strategy with 30-day TTL.
	ParseFilename(ctx context.Context, filename string) (*ai.ParseResponse, error)

	// ParseFansubFilename parses a fansub filename using a specialized prompt.
	// Optimized for Japanese and Chinese fansub naming conventions.
	// Uses cache-first strategy with 30-day TTL.
	ParseFansubFilename(ctx context.Context, filename string) (*ai.ParseResponse, error)

	// ClearCache removes all cached AI parsing results.
	ClearCache(ctx context.Context) (int64, error)

	// ClearExpiredCache removes expired cached entries.
	ClearExpiredCache(ctx context.Context) (int64, error)

	// GetCacheStats returns cache statistics.
	GetCacheStats(ctx context.Context) (*ai.CacheStats, error)

	// IsConfigured returns true if an AI provider is configured.
	IsConfigured() bool

	// GetProviderName returns the name of the configured AI provider.
	GetProviderName() string
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

// ParseFansubFilename parses a fansub filename using a specialized prompt.
// Optimized for Japanese and Chinese fansub naming conventions.
// Enforces 10-second timeout per NFR-P14.
func (s *AIService) ParseFansubFilename(ctx context.Context, filename string) (*ai.ParseResponse, error) {
	start := time.Now()

	if s.provider == nil {
		return nil, ai.ErrAINotConfigured
	}

	// Use fansub-specific cache key prefix to avoid collisions
	cacheKey := "fansub:" + filename

	// Check cache first
	cached, err := s.cache.Get(ctx, cacheKey)
	if err != nil {
		slog.Warn("Cache lookup failed for fansub parsing, proceeding with AI",
			"error", err,
			"filename", filename,
			"duration_ms", time.Since(start).Milliseconds(),
		)
		// Continue to AI parsing
	} else if cached != nil {
		slog.Debug("Cache hit for fansub AI parsing",
			"filename", filename,
			"title", cached.Title,
			"duration_ms", time.Since(start).Milliseconds(),
		)
		return cached, nil
	}

	// Apply 10-second timeout for fansub parsing (NFR-P14)
	ctx, cancel := context.WithTimeout(ctx, FansubParsingTimeout)
	defer cancel()

	// Build fansub-specific prompt
	fansubPrompt := prompts.BuildFansubPrompt(filename)

	// Parse with AI using specialized prompt
	req := &ai.ParseRequest{
		Filename: filename,
		Prompt:   fansubPrompt,
	}

	slog.Debug("Starting fansub AI parsing",
		"filename", filename,
		"timeout_seconds", FansubParsingTimeout.Seconds(),
	)

	response, err := s.provider.Parse(ctx, req)
	if err != nil {
		duration := time.Since(start)
		slog.Error("AI fansub parsing failed",
			"error", err,
			"filename", filename,
			"duration_ms", duration.Milliseconds(),
			"timeout_exceeded", ctx.Err() == context.DeadlineExceeded,
		)
		return nil, err
	}

	duration := time.Since(start)

	// Validate response confidence
	if response.Confidence < 0.3 {
		slog.Warn("Low confidence fansub parsing result",
			"filename", filename,
			"confidence", response.Confidence,
			"title", response.Title,
			"duration_ms", duration.Milliseconds(),
		)
	}

	// Cache the result with fansub prefix
	if err := s.cache.Set(ctx, cacheKey, s.provider.Name(), "fansub", response); err != nil {
		slog.Warn("Failed to cache fansub AI response",
			"error", err,
			"filename", filename,
		)
		// Don't fail the request if caching fails
	}

	slog.Info("Fansub filename parsed successfully",
		"filename", filename,
		"title", response.Title,
		"fansub_group", response.FansubGroup,
		"confidence", response.Confidence,
		"duration_ms", duration.Milliseconds(),
	)

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
