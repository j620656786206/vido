package ai

import (
	"context"
)

// ProviderName represents the name of an AI provider.
type ProviderName string

const (
	// ProviderGemini represents Google's Gemini AI.
	ProviderGemini ProviderName = "gemini"
	// ProviderClaude represents Anthropic's Claude AI.
	ProviderClaude ProviderName = "claude"
)

// String returns the string representation of the provider name.
func (p ProviderName) String() string {
	return string(p)
}

// IsValid checks if the provider name is a known provider.
func (p ProviderName) IsValid() bool {
	switch p {
	case ProviderGemini, ProviderClaude:
		return true
	default:
		return false
	}
}

// Provider defines the interface for AI parsing providers.
// This follows the Strategy pattern for provider switching.
type Provider interface {
	// Parse sends a filename to the AI for parsing and returns normalized results.
	// The provider bounds each attempt itself (RequestTimeoutFor, sub-6-2);
	// the caller's ctx needs no deadline of its own.
	Parse(ctx context.Context, req *ParseRequest) (*ParseResponse, error)

	// Name returns the provider name for logging and caching purposes.
	Name() ProviderName
}

// TextCompleter defines the interface for AI text completion (non-JSON output).
// Used by services that need raw text responses (e.g., terminology correction).
type TextCompleter interface {
	CompleteText(ctx context.Context, systemPrompt, userPrompt string, maxTokens int) (string, error)
}

// Pinger is the OPTIONAL probe a provider exposes for key validation
// (sub-6-6). Ping makes the smallest real request the API allows and reports
// ONLY the transport verdict — a 2xx is success even when the reply carries no
// text. CompleteText's empty-text = ErrAIInvalidResponse rule is right for
// callers that need words back and wrong for a probe: Sonnet 5 at max_tokens=1
// routinely returns an empty text block, which made a valid key test as
// "Cannot parse AI response" (eval-1 product problem 6).
type Pinger interface {
	Ping(ctx context.Context) error
}

// CachingCompleter is the OPTIONAL, additive counterpart to TextCompleter for
// providers that support prompt caching and can report token usage to the
// caller. It sits beside TextCompleter rather than extending it so the
// multi-provider moat is preserved: only ClaudeProvider implements it, Gemini
// deliberately does not, and consumers type-assert and degrade.
//
// [@contract-v1] (story sub-1-1 AC #5) — consumer: sub-1-5a (translate core),
// which composes the cue-grain cache key and reads the cache-token fields to
// detect a silently-inert prefix cache. Changing this shape is a Rule 20 bump
// plus a downstream stale-mark.
//
// This interface ships the CAPABILITY only. Caching POLICY — the ≥4096-token
// prefix minimum, the "disable and record" ruling, the versioned cache key, the
// per-show first-request gate — belongs to sub-1-5a/1-5b.
type CachingCompleter interface {
	CompleteTextWithUsage(ctx context.Context, req CompletionRequest) (CompletionResult, error)
}

// CacheTTL selects the cache_control lifetime for a system block.
type CacheTTL string

const (
	// CacheTTLNone marks a block as NOT a cache breakpoint (no cache_control).
	CacheTTLNone CacheTTL = ""
	// CacheTTL5m is the default ephemeral lifetime.
	CacheTTL5m CacheTTL = "5m"
	// CacheTTL1h is the season-batch lifetime: a batch spans tens of minutes, so
	// the 5-minute entry would expire mid-run.
	CacheTTL1h CacheTTL = "1h"
)

// SystemBlock is one ordered system content block. Prompt caching is a PREFIX
// match, so stable content must come first in the slice — order is semantic.
type SystemBlock struct {
	Text string
	// CacheTTL non-empty emits a cache_control breakpoint on THIS block.
	CacheTTL CacheTTL
}

// CompletionRequest is the input to CompleteTextWithUsage.
type CompletionRequest struct {
	// System blocks in prefix order (stable content first). Empty emits no
	// system field at all.
	System []SystemBlock
	// UserPrompt is the volatile per-request content.
	UserPrompt string
	// MaxTokens <= 0 falls back to ClaudeMaxTokens.
	MaxTokens int
}

// CompletionUsage carries the token counts the Messages API returns, including
// the two cache dimensions. Both cache fields reading zero across repeated
// calls means the prefix silently failed to cache (e.g. below the model's
// minimum cacheable length) — surfacing them is what lets a caller detect that.
type CompletionUsage struct {
	InputTokens              int64
	OutputTokens             int64
	CacheCreationInputTokens int64
	CacheReadInputTokens     int64
}

// CompletionResult is the output of CompleteTextWithUsage.
type CompletionResult struct {
	Text  string
	Usage CompletionUsage
}

// ProviderConfig contains common configuration for AI providers.
type ProviderConfig struct {
	// APIKey is the authentication key for the provider.
	APIKey string
	// TimeoutSeconds pins a fixed per-attempt timeout; 0 (the default) lets
	// the provider derive one per call from the model and max_tokens
	// (RequestTimeoutFor, sub-6-2).
	TimeoutSeconds int
	// BaseURL is an optional custom base URL for the API.
	BaseURL string
}

// DefaultPrompt is the default prompt template for filename parsing.
const DefaultPrompt = `Parse the following media filename and extract metadata.
Return a JSON object with these fields:
- title: The media title (required)
- year: Release year (number, optional)
- season: TV season number (number, optional)
- episode: TV episode number (number, optional)
- media_type: Either "movie" or "tv" (required)
- quality: Video quality like "1080p", "720p", "2160p" (optional)
- fansub_group: The fansub or release group name (optional)
- confidence: Your confidence in the parsing from 0.0 to 1.0 (required)

Be especially careful with Asian media filenames that may have:
- Chinese/Japanese/Korean titles
- Multiple language titles
- Fansub group names in brackets [GroupName]
- Episode notation like S01E01, EP01, 第01話, 第01集

Filename: %s

Respond ONLY with valid JSON, no markdown or explanation.`
