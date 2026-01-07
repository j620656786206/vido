package tmdb

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/alexyu/vido/internal/config"
	"github.com/rs/zerolog/log"
	"golang.org/x/time/rate"
)

const (
	// DefaultBaseURL is the base URL for TMDb API v3
	DefaultBaseURL = "https://api.themoviedb.org/3"

	// TMDb API rate limit: 40 requests per 10 seconds
	requestsPerInterval = 40
	rateLimitInterval   = 10 * time.Second
)

// Client represents a TMDb API client
type Client struct {
	baseURL    string
	apiKey     string
	language   string
	httpClient *http.Client
	limiter    *rate.Limiter
}

// NewClient creates a new TMDb API client with rate limiting
func NewClient(cfg *config.Config) *Client {
	// Create rate limiter: 40 requests per 10 seconds
	// Using rate.Every to calculate the rate: 10s / 40 requests = 250ms per request
	limiter := rate.NewLimiter(rate.Every(rateLimitInterval/requestsPerInterval), requestsPerInterval)

	return &Client{
		baseURL:  DefaultBaseURL,
		apiKey:   cfg.TMDbAPIKey,
		language: cfg.TMDbDefaultLanguage,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
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

	// Log request
	log.Debug().
		Str("method", method).
		Str("url", reqURL).
		Msg("TMDb API request")

	// Execute request
	resp, err := c.httpClient.Do(req)
	if err != nil {
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
		// Parse TMDb error response and return appropriate AppError
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
	queryParams.Set("language", c.language)

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
