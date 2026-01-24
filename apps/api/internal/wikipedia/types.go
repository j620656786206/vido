// Package wikipedia provides a client for fetching metadata from zh.wikipedia.org
// using the MediaWiki API. This package implements polite API access with rate limiting,
// proper User-Agent headers, and Traditional Chinese content extraction.
package wikipedia

import (
	"time"
)

// SearchResult represents a single search result from Wikipedia
type SearchResult struct {
	// PageID is the Wikipedia page ID
	PageID int64
	// Title is the page title
	Title string
	// Snippet is a text snippet from the page
	Snippet string
	// WordCount is the word count of the page
	WordCount int
	// Timestamp is when the page was last modified
	Timestamp string
}

// PageContent represents the content of a Wikipedia page
type PageContent struct {
	// PageID is the Wikipedia page ID
	PageID int64
	// Title is the page title
	Title string
	// Wikitext is the raw wikitext content
	Wikitext string
	// Extract is the plain text summary (first paragraph)
	Extract string
	// HTMLContent is the rendered HTML content (if available)
	HTMLContent string
}

// InfoboxData represents extracted data from an Infobox template
type InfoboxData struct {
	// Type indicates the Infobox type (film, television, animanga)
	Type string
	// Name is the title from the Infobox
	Name string
	// OriginalName is the original language title
	OriginalName string
	// Image is the image filename (without File: prefix)
	Image string
	// Director is the director name(s)
	Director string
	// Creator is the creator name(s) (for TV shows)
	Creator string
	// Starring is the cast list
	Starring []string
	// Producer is the producer name(s)
	Producer string
	// Writer is the writer name(s)
	Writer string
	// Music is the music composer
	Music string
	// Country is the production country
	Country string
	// Language is the language
	Language string
	// Genre is the genre(s)
	Genre []string
	// Released is the release date string
	Released string
	// FirstAired is the first air date (for TV shows)
	FirstAired string
	// Runtime is the duration string
	Runtime string
	// NumSeasons is the number of seasons (for TV shows)
	NumSeasons int
	// NumEpisodes is the number of episodes (for TV shows)
	NumEpisodes int
	// Year is the extracted year from release/first_aired
	Year int
	// Studio is the production studio (for anime)
	Studio string
}

// ImageInfo represents information about a Wikipedia image
type ImageInfo struct {
	// URL is the direct URL to the image
	URL string
	// DescriptionURL is the URL to the file description page
	DescriptionURL string
	// Width is the image width in pixels
	Width int
	// Height is the image height in pixels
	Height int
	// Size is the file size in bytes
	Size int64
	// MimeType is the MIME type of the image
	MimeType string
}

// ClientConfig holds configuration for the Wikipedia client
type ClientConfig struct {
	// RequestsPerSecond is the rate limit (default: 1 per NFR-I14)
	RequestsPerSecond float64
	// Timeout is the HTTP request timeout
	Timeout time.Duration
	// MaxRetries is the maximum number of retry attempts
	MaxRetries int
	// Enabled controls whether the client is active
	Enabled bool
	// UserAgent is the User-Agent header value (NFR-I13)
	UserAgent string
	// ContactEmail is the contact email for the User-Agent
	ContactEmail string
}

// DefaultConfig returns the default client configuration
// Following NFR-I14: 1 request per second rate limit for Wikipedia
func DefaultConfig() ClientConfig {
	return ClientConfig{
		RequestsPerSecond: 1.0, // 1 request per second per NFR-I14
		Timeout:           10 * time.Second,
		MaxRetries:        3,
		Enabled:           true,
		UserAgent:         "Vido/1.0",
		ContactEmail:      "contact@example.com",
	}
}

// MediaType represents the type of media.
// Note: In the WikipediaProvider, MediaTypeAnime is mapped to metadata.MediaTypeTV
// since anime series are treated as TV content in the metadata system.
type MediaType string

const (
	// MediaTypeMovie represents a movie/film
	MediaTypeMovie MediaType = "movie"
	// MediaTypeTV represents a TV show/series
	MediaTypeTV MediaType = "tv"
	// MediaTypeAnime represents an anime (mapped to MediaTypeTV in provider)
	MediaTypeAnime MediaType = "anime"
)

// Error codes for Wikipedia client (per project-context.md Rule 7)
const (
	ErrCodeNotFound    = "WIKIPEDIA_NOT_FOUND"
	ErrCodeNoInfobox   = "WIKIPEDIA_NO_INFOBOX"
	ErrCodeParseError  = "WIKIPEDIA_PARSE_ERROR"
	ErrCodeRateLimited = "WIKIPEDIA_RATE_LIMITED"
	ErrCodeTimeout     = "WIKIPEDIA_TIMEOUT"
	ErrCodeAPIError    = "WIKIPEDIA_API_ERROR"
)

// NotFoundError represents a resource not found error
type NotFoundError struct {
	// Query is what was searched for
	Query string
	// PageTitle is the page title if looking up by title
	PageTitle string
}

func (e *NotFoundError) Error() string {
	if e.PageTitle != "" {
		return "wikipedia: page not found: " + e.PageTitle
	}
	return "wikipedia: no results for query: " + e.Query
}

// ParseError represents an error during wikitext parsing
type ParseError struct {
	// Field is the field that failed to parse
	Field string
	// Reason describes why parsing failed
	Reason string
	// Wikitext is a snippet of the problematic wikitext (for debugging)
	Wikitext string
}

func (e *ParseError) Error() string {
	return "wikipedia: parse error for " + e.Field + " - " + e.Reason
}

// APIError represents an error from the MediaWiki API
type APIError struct {
	// Code is the API error code
	Code string
	// Info is the error description
	Info string
}

func (e *APIError) Error() string {
	return "wikipedia api error: " + e.Code + " - " + e.Info
}
