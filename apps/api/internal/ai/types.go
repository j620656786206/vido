// Package ai provides AI provider abstraction for filename parsing.
// It supports multiple AI providers (Gemini, Claude) through a unified interface.
package ai

import (
	"errors"
	"regexp"
	"strings"
)

// Error codes for AI operations (following project-context.md Rule 7)
var (
	ErrAITimeout         = errors.New("AI_TIMEOUT: AI parsing timeout exceeded")
	ErrAIQuotaExceeded   = errors.New("AI_QUOTA_EXCEEDED: API quota exhausted")
	ErrAIInvalidResponse = errors.New("AI_INVALID_RESPONSE: Cannot parse AI response")
	ErrAIProviderError   = errors.New("AI_PROVIDER_ERROR: Provider returned an error")
	ErrAINotConfigured   = errors.New("AI_NOT_CONFIGURED: No AI provider configured")

	// Keyword generation error codes (Story 3.6)
	ErrKeywordGenerationFailed = errors.New("KEYWORD_GENERATION_FAILED: AI failed to generate keywords")
	ErrKeywordNoAlternatives   = errors.New("KEYWORD_NO_ALTERNATIVES: No alternative keywords generated")
	ErrKeywordAllFailed        = errors.New("KEYWORD_ALL_FAILED: All keyword variants failed to find results")
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
	// TitleRomanized is the romanized version of CJK titles.
	TitleRomanized string `json:"title_romanized,omitempty"`
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
	// Source is the release source (e.g., "BD", "WEB", "TV").
	Source string `json:"source,omitempty"`
	// Codec is the video codec (e.g., "x264", "x265", "HEVC").
	Codec string `json:"codec,omitempty"`
	// FansubGroup is the release/fansub group name.
	FansubGroup string `json:"fansub_group,omitempty"`
	// Language is the subtitle/dub language (e.g., "Traditional Chinese").
	Language string `json:"language,omitempty"`
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

// markdownCodeBlockPattern matches markdown code blocks: ```json ... ``` or ``` ... ```
var markdownCodeBlockPattern = regexp.MustCompile("(?s)^\\s*```(?:json)?\\s*\\n?(.*?)\\n?```\\s*$")

// CleanJSONResponse strips markdown code blocks from AI responses.
// AI models often return JSON wrapped in markdown code blocks like:
// ```json
// {"title": "..."}
// ```
// This function extracts the raw JSON for proper parsing.
func CleanJSONResponse(response string) string {
	response = strings.TrimSpace(response)

	// Try to extract content from markdown code block
	if matches := markdownCodeBlockPattern.FindStringSubmatch(response); len(matches) > 1 {
		return strings.TrimSpace(matches[1])
	}

	// If no code block found, return as-is (already clean JSON)
	return response
}
