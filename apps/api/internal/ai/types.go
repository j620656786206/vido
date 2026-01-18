// Package ai provides AI provider abstraction for filename parsing.
// It supports multiple AI providers (Gemini, Claude) through a unified interface.
package ai

import (
	"errors"
)

// Error codes for AI operations (following project-context.md Rule 7)
var (
	ErrAITimeout         = errors.New("AI_TIMEOUT: AI parsing timeout exceeded 15s")
	ErrAIQuotaExceeded   = errors.New("AI_QUOTA_EXCEEDED: API quota exhausted")
	ErrAIInvalidResponse = errors.New("AI_INVALID_RESPONSE: Cannot parse AI response")
	ErrAIProviderError   = errors.New("AI_PROVIDER_ERROR: Provider returned an error")
	ErrAINotConfigured   = errors.New("AI_NOT_CONFIGURED: No AI provider configured")
)

// ParseRequest contains the input for AI parsing operations.
type ParseRequest struct {
	// Filename is the media filename to parse.
	Filename string `json:"filename"`
	// Prompt is an optional custom prompt for parsing.
	// If empty, a default prompt will be used.
	Prompt string `json:"prompt,omitempty"`
}

// ParseResponse contains normalized parsing results from any AI provider.
type ParseResponse struct {
	// Title is the extracted media title.
	Title string `json:"title"`
	// Year is the release year (optional).
	Year int `json:"year,omitempty"`
	// Season is the TV show season number (optional).
	Season int `json:"season,omitempty"`
	// Episode is the TV show episode number (optional).
	Episode int `json:"episode,omitempty"`
	// MediaType is either "movie" or "tv".
	MediaType string `json:"media_type"`
	// Quality is the video resolution (e.g., "1080p", "2160p").
	Quality string `json:"quality,omitempty"`
	// FansubGroup is the release/fansub group name.
	FansubGroup string `json:"fansub_group,omitempty"`
	// Confidence is a score from 0.0 to 1.0 indicating reliability.
	Confidence float64 `json:"confidence"`
	// RawResponse is the original AI response for debugging.
	RawResponse string `json:"raw_response"`
}

// Validate checks if the ParseRequest has required fields.
func (r *ParseRequest) Validate() error {
	if r.Filename == "" {
		return errors.New("filename is required")
	}
	return nil
}

// IsMovie returns true if the media type is movie.
func (r *ParseResponse) IsMovie() bool {
	return r.MediaType == "movie"
}

// IsTVShow returns true if the media type is TV show.
func (r *ParseResponse) IsTVShow() bool {
	return r.MediaType == "tv"
}
