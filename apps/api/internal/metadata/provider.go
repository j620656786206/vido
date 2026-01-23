// Package metadata provides the metadata provider interface and common types
// for the multi-source metadata fallback chain.
package metadata

import (
	"context"
	"errors"
	"strings"

	"github.com/vido/api/internal/models"
)

// ProviderStatus represents the current status of a metadata provider
type ProviderStatus string

const (
	// ProviderStatusAvailable indicates the provider is ready to handle requests
	ProviderStatusAvailable ProviderStatus = "available"
	// ProviderStatusUnavailable indicates the provider cannot handle requests
	ProviderStatusUnavailable ProviderStatus = "unavailable"
	// ProviderStatusRateLimited indicates the provider has hit rate limits
	ProviderStatusRateLimited ProviderStatus = "rate_limited"
)

// String returns the string representation of ProviderStatus
func (s ProviderStatus) String() string {
	return string(s)
}

// IsValid checks if the ProviderStatus is a valid value
func (s ProviderStatus) IsValid() bool {
	switch s {
	case ProviderStatusAvailable, ProviderStatusUnavailable, ProviderStatusRateLimited:
		return true
	default:
		return false
	}
}

// MediaType represents the type of media being searched
type MediaType string

const (
	// MediaTypeMovie represents a movie
	MediaTypeMovie MediaType = "movie"
	// MediaTypeTV represents a TV show
	MediaTypeTV MediaType = "tv"
)

// IsValid checks if the MediaType is a valid value
func (m MediaType) IsValid() bool {
	switch m {
	case MediaTypeMovie, MediaTypeTV:
		return true
	default:
		return false
	}
}

// SearchRequest represents a common request format for all metadata providers
type SearchRequest struct {
	// Query is the search term (required)
	Query string
	// Year is an optional filter for release year
	Year int
	// MediaType is the type of media to search for ("movie" or "tv")
	MediaType MediaType
	// Language is the preferred language for results (e.g., "zh-TW")
	Language string
	// Page is the page number for pagination (1-indexed)
	Page int
}

// Validate validates the search request
func (r *SearchRequest) Validate() error {
	if strings.TrimSpace(r.Query) == "" {
		return errors.New("query is required")
	}

	// Default media type to movie if not specified
	if r.MediaType == "" {
		r.MediaType = MediaTypeMovie
	}

	if !r.MediaType.IsValid() {
		return errors.New("invalid media type: must be 'movie' or 'tv'")
	}

	// Default page to 1 if not specified
	if r.Page < 1 {
		r.Page = 1
	}

	return nil
}

// SearchResult represents the common result format from all metadata providers
type SearchResult struct {
	// Items contains the search results
	Items []MetadataItem
	// Source indicates which provider returned these results
	Source models.MetadataSource
	// TotalCount is the total number of results available
	TotalCount int
	// Page is the current page number
	Page int
	// TotalPages is the total number of pages available
	TotalPages int
}

// HasResults returns true if the search result contains any items
func (r *SearchResult) HasResults() bool {
	return len(r.Items) > 0
}

// MetadataItem represents a normalized metadata item from any provider
type MetadataItem struct {
	// ID is the provider-specific ID
	ID string
	// Title is the title in the default language
	Title string
	// TitleZhTW is the Traditional Chinese title if available
	TitleZhTW string
	// OriginalTitle is the original language title
	OriginalTitle string
	// Year is the release year
	Year int
	// ReleaseDate is the full release date (YYYY-MM-DD format)
	ReleaseDate string
	// Overview is the description/synopsis
	Overview string
	// OverviewZhTW is the Traditional Chinese overview if available
	OverviewZhTW string
	// PosterURL is the URL to the poster image
	PosterURL string
	// BackdropURL is the URL to the backdrop image
	BackdropURL string
	// MediaType indicates if this is a movie or TV show
	MediaType MediaType
	// Genres is a list of genre names
	Genres []string
	// Rating is the average rating (0-10 scale)
	Rating float64
	// VoteCount is the number of votes
	VoteCount int
	// Popularity is the popularity score
	Popularity float64
	// Confidence is the match confidence (0-1 scale)
	Confidence float64
	// RawData contains the original provider response for debugging
	RawData interface{}
}

// HasTitle returns true if the item has any title set
func (i *MetadataItem) HasTitle() bool {
	return i.Title != "" || i.TitleZhTW != ""
}

// GetDisplayTitle returns the appropriate title based on language preference
func (i *MetadataItem) GetDisplayTitle(lang string) string {
	// If zh-TW is preferred and available, use it
	if strings.HasPrefix(lang, "zh") && i.TitleZhTW != "" {
		return i.TitleZhTW
	}

	// Otherwise prefer the default title
	if i.Title != "" {
		return i.Title
	}

	// Fallback to zh-TW title if that's all we have
	return i.TitleZhTW
}

// MetadataProvider defines the interface for all metadata sources
type MetadataProvider interface {
	// Name returns the human-readable name of the provider
	Name() string

	// Source returns the MetadataSource enum value for this provider
	Source() models.MetadataSource

	// Search performs a metadata search and returns results
	Search(ctx context.Context, req *SearchRequest) (*SearchResult, error)

	// IsAvailable returns true if the provider is currently available
	IsAvailable() bool

	// Status returns the current status of the provider
	Status() ProviderStatus
}

// ProviderError represents an error from a metadata provider
type ProviderError struct {
	// Provider is the name of the provider that encountered the error
	Provider string
	// Source is the MetadataSource of the provider
	Source models.MetadataSource
	// Code is the error code
	Code string
	// Message is the error message
	Message string
	// Err is the underlying error if any
	Err error
}

// Error implements the error interface
func (e *ProviderError) Error() string {
	if e.Err != nil {
		return e.Provider + ": " + e.Message + ": " + e.Err.Error()
	}
	return e.Provider + ": " + e.Message
}

// Unwrap returns the underlying error
func (e *ProviderError) Unwrap() error {
	return e.Err
}

// NewProviderError creates a new ProviderError
func NewProviderError(provider string, source models.MetadataSource, code, message string, err error) *ProviderError {
	return &ProviderError{
		Provider: provider,
		Source:   source,
		Code:     code,
		Message:  message,
		Err:      err,
	}
}

// Common error codes for metadata providers
const (
	// ErrCodeNoResults indicates no results were found
	ErrCodeNoResults = "METADATA_NO_RESULTS"
	// ErrCodeTimeout indicates the request timed out
	ErrCodeTimeout = "METADATA_TIMEOUT"
	// ErrCodeRateLimited indicates rate limiting was hit
	ErrCodeRateLimited = "METADATA_RATE_LIMITED"
	// ErrCodeUnavailable indicates the provider is unavailable
	ErrCodeUnavailable = "METADATA_UNAVAILABLE"
	// ErrCodeInvalidRequest indicates the request was invalid
	ErrCodeInvalidRequest = "METADATA_INVALID_REQUEST"
)
