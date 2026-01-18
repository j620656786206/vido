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
	// The context should include a timeout of 15 seconds per NFR-I12.
	Parse(ctx context.Context, req *ParseRequest) (*ParseResponse, error)

	// Name returns the provider name for logging and caching purposes.
	Name() ProviderName
}

// ProviderConfig contains common configuration for AI providers.
type ProviderConfig struct {
	// APIKey is the authentication key for the provider.
	APIKey string
	// Timeout is the request timeout in seconds (default: 15s per NFR-I12).
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
