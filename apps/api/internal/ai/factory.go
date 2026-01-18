package ai

import (
	"fmt"
	"log/slog"
	"strings"
)

// FactoryConfig contains configuration for creating AI providers.
type FactoryConfig struct {
	// ProviderName is the name of the AI provider to use ("gemini" or "claude").
	ProviderName string
	// GeminiAPIKey is the API key for Gemini.
	GeminiAPIKey string
	// ClaudeAPIKey is the API key for Claude.
	ClaudeAPIKey string
}

// NewProvider creates an AI provider based on the configuration.
// It returns the appropriate provider based on ProviderName, or an error if
// the provider is not configured or the name is invalid.
func NewProvider(cfg FactoryConfig) (Provider, error) {
	providerName := ProviderName(strings.ToLower(cfg.ProviderName))

	if !providerName.IsValid() {
		slog.Warn("Invalid AI provider name",
			"provider", cfg.ProviderName,
			"valid_providers", []string{string(ProviderGemini), string(ProviderClaude)},
		)
		return nil, fmt.Errorf("%w: invalid provider name '%s'", ErrAINotConfigured, cfg.ProviderName)
	}

	switch providerName {
	case ProviderGemini:
		if cfg.GeminiAPIKey == "" {
			slog.Error("Gemini provider selected but GEMINI_API_KEY not set")
			return nil, fmt.Errorf("%w: GEMINI_API_KEY not configured", ErrAINotConfigured)
		}
		slog.Info("Creating Gemini AI provider")
		return NewGeminiProvider(cfg.GeminiAPIKey), nil

	case ProviderClaude:
		if cfg.ClaudeAPIKey == "" {
			slog.Error("Claude provider selected but CLAUDE_API_KEY not set")
			return nil, fmt.Errorf("%w: CLAUDE_API_KEY not configured", ErrAINotConfigured)
		}
		slog.Info("Creating Claude AI provider")
		return NewClaudeProvider(cfg.ClaudeAPIKey), nil

	default:
		return nil, fmt.Errorf("%w: unknown provider '%s'", ErrAINotConfigured, providerName)
	}
}

// NewProviderWithFallback creates an AI provider with automatic fallback.
// It first tries the primary provider, and if that fails, tries the secondary.
// Returns the first successfully created provider, or an error if both fail.
func NewProviderWithFallback(primary, secondary FactoryConfig) (Provider, error) {
	// Try primary provider
	provider, err := NewProvider(primary)
	if err == nil {
		return provider, nil
	}

	slog.Warn("Primary AI provider failed, trying fallback",
		"primary", primary.ProviderName,
		"secondary", secondary.ProviderName,
		"error", err,
	)

	// Try secondary provider
	return NewProvider(secondary)
}

// MustNewProvider creates an AI provider and panics if it fails.
// This should only be used during application startup.
func MustNewProvider(cfg FactoryConfig) Provider {
	provider, err := NewProvider(cfg)
	if err != nil {
		panic(fmt.Sprintf("failed to create AI provider: %v", err))
	}
	return provider
}
