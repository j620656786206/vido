package services

import (
	"context"
	"log/slog"
	"time"

	"github.com/vido/api/internal/ai"
	"github.com/vido/api/internal/ai/prompts"
	"github.com/vido/api/internal/config"
)

const (
	// TerminologyCorrectionTimeout is the maximum time for AI terminology correction (AC #6).
	TerminologyCorrectionTimeout = 30 * time.Second
	// TerminologyCorrectionMaxTokens is the max response tokens for subtitle correction.
	TerminologyCorrectionMaxTokens = 2048
)

// TerminologyCorrectionServiceInterface defines the contract for AI terminology correction.
type TerminologyCorrectionServiceInterface interface {
	// Correct sends subtitle content to Claude for cross-strait terminology correction.
	// On error or timeout, returns the original content unchanged.
	Correct(ctx context.Context, subtitleContent string) (string, error)

	// IsConfigured returns true if a Claude API key is available.
	IsConfigured() bool
}

// TerminologyCorrectionService uses Claude to fix cross-strait Chinese terminology.
type TerminologyCorrectionService struct {
	provider *ai.ClaudeProvider
	cfg      *config.Config
}

// Compile-time interface verification.
var _ TerminologyCorrectionServiceInterface = (*TerminologyCorrectionService)(nil)

// NewTerminologyCorrectionService creates a new terminology correction service.
// Returns nil if no Claude API key is configured (graceful degradation per AC #2).
func NewTerminologyCorrectionService(cfg *config.Config) *TerminologyCorrectionService {
	if !cfg.HasClaudeKey() {
		slog.Info("Terminology correction service not configured - no Claude API key set")
		return nil
	}

	provider := ai.NewClaudeProvider(cfg.GetClaudeAPIKey())

	slog.Info("Terminology correction service initialized")
	return &TerminologyCorrectionService{
		provider: provider,
		cfg:      cfg,
	}
}

// NewTerminologyCorrectionServiceWithProvider creates a service with a specific provider (for testing).
func NewTerminologyCorrectionServiceWithProvider(provider *ai.ClaudeProvider) *TerminologyCorrectionService {
	return &TerminologyCorrectionService{
		provider: provider,
	}
}

// Correct sends subtitle content to Claude for terminology correction.
// Applies a 30-second timeout (AC #6). On any error, returns the original content
// unchanged and logs a warning (AC #4 — graceful degradation).
func (s *TerminologyCorrectionService) Correct(ctx context.Context, subtitleContent string) (string, error) {
	if s.provider == nil {
		return subtitleContent, nil
	}

	if subtitleContent == "" {
		return subtitleContent, nil
	}

	start := time.Now()

	// Apply 30-second timeout per AC #6
	ctx, cancel := context.WithTimeout(ctx, TerminologyCorrectionTimeout)
	defer cancel()

	userPrompt := prompts.BuildTerminologyCorrectorPrompt(subtitleContent)

	corrected, err := s.provider.CompleteText(
		ctx,
		prompts.TerminologyCorrectorSystemPrompt,
		userPrompt,
		TerminologyCorrectionMaxTokens,
	)
	if err != nil {
		duration := time.Since(start)
		slog.Warn("AI terminology correction failed — using original content",
			"error", err,
			"duration_ms", duration.Milliseconds(),
			"content_length", len(subtitleContent),
		)
		// AC #4: fall back to original content on error
		return subtitleContent, err
	}

	duration := time.Since(start)
	slog.Info("Terminology correction completed",
		"duration_ms", duration.Milliseconds(),
		"original_length", len(subtitleContent),
		"corrected_length", len(corrected),
	)

	return corrected, nil
}

// IsConfigured returns true if a Claude API key is available for terminology correction.
func (s *TerminologyCorrectionService) IsConfigured() bool {
	return s != nil && s.provider != nil
}
