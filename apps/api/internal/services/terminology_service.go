package services

import (
	"context"
	"log/slog"
	"time"

	"github.com/vido/api/internal/ai"
	"github.com/vido/api/internal/ai/prompts"
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

// TerminologyCorrectionService uses AI to fix cross-strait Chinese terminology.
type TerminologyCorrectionService struct {
	provider ai.TextCompleter
}

// Compile-time interface verification.
var _ TerminologyCorrectionServiceInterface = (*TerminologyCorrectionService)(nil)

// NewTerminologyCorrectionService creates a new terminology correction service.
// Returns nil if provider is nil (graceful degradation per AC #2).
// The caller is responsible for creating the provider only when an API key is configured.
func NewTerminologyCorrectionService(provider ai.TextCompleter) *TerminologyCorrectionService {
	if provider == nil {
		slog.Info("Terminology correction service not configured - no AI provider")
		return nil
	}

	slog.Info("Terminology correction service initialized")
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
	if s == nil || s.provider == nil {
		return false
	}
	// A provider may be a LAZY holder whose key is resolved per call (sub-2-1a
	// AC #2). Since sub-2-1a these services are constructed UNCONDITIONALLY, so
	// "the provider object exists" no longer implies "a key is configured" —
	// without this probe a keyless install would report configured, attempt the
	// work, and fail with ErrAINotConfigured where it used to skip cleanly.
	// A plain provider (Gemini, a direct Claude client) has no probe and stays
	// configured, exactly as before.
	if probe, ok := s.provider.(interface {
		IsConfigured(ctx context.Context) bool
	}); ok {
		return probe.IsConfigured(context.Background())
	}
	return true
}
