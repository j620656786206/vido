// Package douban provides a web scraper client for fetching metadata from Douban.
// This package implements polite scraping with rate limiting, exponential backoff,
// and Traditional Chinese conversion for AC compliance.
package douban

import (
	"time"
)

// SearchResult represents a single search result from Douban
type SearchResult struct {
	// ID is the Douban subject ID (e.g., "1292052")
	ID string
	// Title is the title from search results
	Title string
	// URL is the full URL to the detail page
	URL string
	// Year is the release year (if available from search)
	Year int
	// Rating is the Douban rating (if available from search)
	Rating float64
	// Type indicates if this is a movie or TV show
	Type MediaType
}

// MediaType represents the type of media
type MediaType string

const (
	// MediaTypeMovie represents a movie
	MediaTypeMovie MediaType = "movie"
	// MediaTypeTV represents a TV show/series
	MediaTypeTV MediaType = "tv"
)

// DetailResult represents scraped metadata from a Douban detail page
type DetailResult struct {
	// ID is the Douban subject ID
	ID string
	// Title is the Chinese title (may be Simplified)
	Title string
	// TitleTraditional is the Traditional Chinese title (converted if needed)
	TitleTraditional string
	// OriginalTitle is the original language title (if different)
	OriginalTitle string
	// Year is the release year
	Year int
	// Rating is the Douban rating (0-10 scale)
	Rating float64
	// RatingCount is the number of ratings
	RatingCount int
	// Director is the director's name(s)
	Director string
	// Cast is a list of main cast members
	Cast []string
	// Genres is a list of genres
	Genres []string
	// Summary is the plot summary
	Summary string
	// SummaryTraditional is the Traditional Chinese summary
	SummaryTraditional string
	// PosterURL is the URL to the poster image
	PosterURL string
	// Type indicates movie or TV
	Type MediaType
	// Runtime is the duration in minutes (movies only)
	Runtime int
	// Episodes is the number of episodes (TV only)
	Episodes int
	// Countries is a list of production countries
	Countries []string
	// Languages is a list of languages
	Languages []string
	// ReleaseDate is the release date string
	ReleaseDate string
	// IMDbID is the IMDb ID if available
	IMDbID string
	// ScrapedAt is when this data was scraped
	ScrapedAt time.Time
}

// ClientConfig holds configuration for the Douban client
type ClientConfig struct {
	// RequestsPerSecond is the rate limit (default: 0.5 = 1 req per 2 seconds)
	RequestsPerSecond float64
	// Timeout is the HTTP request timeout
	Timeout time.Duration
	// MaxRetries is the maximum number of retry attempts
	MaxRetries int
	// InitialBackoff is the initial backoff duration for retries
	InitialBackoff time.Duration
	// MaxBackoff is the maximum backoff duration
	MaxBackoff time.Duration
	// BackoffMultiplier is the multiplier for exponential backoff
	BackoffMultiplier float64
	// JitterMin is the minimum random jitter to add to delays
	JitterMin time.Duration
	// JitterMax is the maximum random jitter to add to delays
	JitterMax time.Duration
	// Enabled controls whether the client is active
	Enabled bool
}

// DefaultConfig returns the default client configuration
// Following AC4: 1 request per 2 seconds minimum
func DefaultConfig() ClientConfig {
	return ClientConfig{
		RequestsPerSecond: 0.5, // 1 request per 2 seconds per AC4
		Timeout:           30 * time.Second,
		MaxRetries:        5,
		InitialBackoff:    1 * time.Second,
		MaxBackoff:        16 * time.Second,
		BackoffMultiplier: 2.0,
		JitterMin:         100 * time.Millisecond,
		JitterMax:         500 * time.Millisecond,
		Enabled:           true,
	}
}

// BlockedError represents an error when anti-scraping measures are detected
type BlockedError struct {
	// StatusCode is the HTTP status code (e.g., 403)
	StatusCode int
	// Reason describes why the request was blocked
	Reason string
	// RetryAfter suggests when to retry (if available)
	RetryAfter time.Duration
}

func (e *BlockedError) Error() string {
	return "douban: blocked - " + e.Reason
}

// IsBlocked returns true if this is a blocking error
func (e *BlockedError) IsBlocked() bool {
	return true
}

// ParseError represents an error during HTML parsing
type ParseError struct {
	// Field is the field that failed to parse
	Field string
	// Reason describes why parsing failed
	Reason string
	// HTML is a snippet of the problematic HTML (for debugging)
	HTML string
}

func (e *ParseError) Error() string {
	return "douban: parse error for " + e.Field + " - " + e.Reason
}

// NotFoundError represents a resource not found error
type NotFoundError struct {
	// Query is what was searched for
	Query string
	// ID is the Douban ID if looking up by ID
	ID string
}

func (e *NotFoundError) Error() string {
	if e.ID != "" {
		return "douban: subject not found: " + e.ID
	}
	return "douban: no results for query: " + e.Query
}

// Error codes for Douban scraper (per project-context.md Rule 7)
const (
	ErrCodeBlocked     = "DOUBAN_BLOCKED"
	ErrCodeNotFound    = "DOUBAN_NOT_FOUND"
	ErrCodeParseError  = "DOUBAN_PARSE_ERROR"
	ErrCodeRateLimited = "DOUBAN_RATE_LIMITED"
	ErrCodeTimeout     = "DOUBAN_TIMEOUT"
)
