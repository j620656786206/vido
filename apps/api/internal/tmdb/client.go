package tmdb

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"golang.org/x/time/rate"
)

const (
	// DefaultBaseURL is the base URL for TMDb API v3
	DefaultBaseURL = "https://api.themoviedb.org/3"

	// TMDb API rate limit: 40 requests per 10 seconds
	requestsPerInterval = 40
	rateLimitInterval   = 10 * time.Second
)

// ClientConfig holds configuration for the TMDb client
type ClientConfig struct {
	APIKey   string
	Language string
	BaseURL  string // Optional, defaults to DefaultBaseURL
	Timeout  time.Duration // Optional, defaults to 30 seconds
}

// ClientInterface defines the contract for TMDb API operations
// This allows for mocking in tests and implementing caching layers
type ClientInterface interface {
	// SearchMovies searches for movies by title
	SearchMovies(ctx context.Context, query string, page int) (*SearchResultMovies, error)
	// SearchMoviesWithLanguage searches for movies with a specific language
	SearchMoviesWithLanguage(ctx context.Context, query string, language string, page int) (*SearchResultMovies, error)
	// GetMovieDetails retrieves complete movie information
	GetMovieDetails(ctx context.Context, movieID int) (*MovieDetails, error)
	// GetMovieDetailsWithLanguage retrieves movie details with a specific language
	GetMovieDetailsWithLanguage(ctx context.Context, movieID int, language string) (*MovieDetails, error)
	// SearchTVShows searches for TV shows by name
	SearchTVShows(ctx context.Context, query string, page int) (*SearchResultTVShows, error)
	// SearchTVShowsWithLanguage searches for TV shows with a specific language
	SearchTVShowsWithLanguage(ctx context.Context, query string, language string, page int) (*SearchResultTVShows, error)
	// GetTVShowDetails retrieves complete TV show information
	GetTVShowDetails(ctx context.Context, tvID int) (*TVShowDetails, error)
	// GetTVShowDetailsWithLanguage retrieves TV show details with a specific language
	GetTVShowDetailsWithLanguage(ctx context.Context, tvID int, language string) (*TVShowDetails, error)
}

// Client represents a TMDb API client
type Client struct {
	baseURL    string
	apiKey     string
	language   string
	httpClient *http.Client
	limiter    *rate.Limiter
}

// Compile-time interface verification
var _ ClientInterface = (*Client)(nil)

// NewClient creates a new TMDb API client with rate limiting
func NewClient(cfg ClientConfig) *Client {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}

	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	language := cfg.Language
	if language == "" {
		language = "zh-TW"
	}

	// Create rate limiter: 40 requests per 10 seconds
	// Using rate.Every to calculate the rate: 10s / 40 requests = 250ms per request
	limiter := rate.NewLimiter(rate.Every(rateLimitInterval/requestsPerInterval), requestsPerInterval)

	return &Client{
		baseURL:  baseURL,
		apiKey:   cfg.APIKey,
		language: language,
		httpClient: &http.Client{
			Timeout: timeout,
		},
		limiter: limiter,
	}
}

// doRequest performs an HTTP request with rate limiting and common error handling
func (c *Client) doRequest(ctx context.Context, method, endpoint string, queryParams url.Values) ([]byte, error) {
	// Wait for rate limiter
	if err := c.limiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limiter error: %w", err)
	}

	// Build URL with query parameters
	reqURL, err := c.buildURL(endpoint, queryParams)
	if err != nil {
		return nil, fmt.Errorf("failed to build URL: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, method, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Vido/1.0")

	// Log request (using slog instead of zerolog)
	slog.Debug("TMDb API request",
		"method", method,
		"endpoint", endpoint,
	)

	// Execute request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		slog.Error("TMDb API request failed",
			"error", err,
			"endpoint", endpoint,
		)
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Check for HTTP errors
	if resp.StatusCode != http.StatusOK {
		slog.Warn("TMDb API returned error",
			"status_code", resp.StatusCode,
			"endpoint", endpoint,
		)
		// Parse TMDb error response and return appropriate TMDbError
		return nil, ParseAPIError(resp.StatusCode, body)
	}

	return body, nil
}

// buildURL constructs the full URL with query parameters
func (c *Client) buildURL(endpoint string, queryParams url.Values) (string, error) {
	// Parse base URL
	u, err := url.Parse(c.baseURL + endpoint)
	if err != nil {
		return "", err
	}

	// Add API key and language to query parameters
	if queryParams == nil {
		queryParams = url.Values{}
	}
	queryParams.Set("api_key", c.apiKey)

	// Only set language if not already specified in queryParams
	if queryParams.Get("language") == "" {
		queryParams.Set("language", c.language)
	}

	u.RawQuery = queryParams.Encode()
	return u.String(), nil
}

// Get performs a GET request to the TMDb API
func (c *Client) Get(ctx context.Context, endpoint string, queryParams url.Values, result interface{}) error {
	body, err := c.doRequest(ctx, http.MethodGet, endpoint, queryParams)
	if err != nil {
		return err
	}

	// Unmarshal response
	if err := json.Unmarshal(body, result); err != nil {
		return fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return nil
}
