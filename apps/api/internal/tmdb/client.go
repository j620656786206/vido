package tmdb

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"sync/atomic"
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

// KeyProvider resolves the TMDb API key for ONE request. It is asked every
// time, so a key saved from the settings page — or the bundled fallback — is
// used by the very next call with no restart (sub-7-7a, closing
// backlog-tmdb-runtime-key-resolution). Returning "" means "not configured";
// the client then fails the request locally instead of sending a keyless call.
type KeyProvider func(ctx context.Context) (string, error)

// RequestObserver is told how each request ended: nil on 2xx, the *TMDbError
// on an API error, the transport error otherwise. The health monitor uses it
// to see a 429 the moment a real request gets one, instead of waiting for the
// 5-minute ping (sub-7-7a AC #4).
type RequestObserver func(endpoint string, err error)

// ClientConfig holds configuration for the TMDb client
type ClientConfig struct {
	// APIKey is the static key. Ignored when KeyProvider is set; kept for
	// tests and one-off tools that have no resolver.
	APIKey   string
	Language string
	BaseURL  string        // Optional, defaults to DefaultBaseURL
	Timeout  time.Duration // Optional, defaults to 30 seconds
	// KeyProvider, when set, supplies the key per request (see KeyProvider).
	KeyProvider KeyProvider
	// Observer, when set, is notified after every request (see RequestObserver).
	// It can also be attached later with SetObserver, because the health
	// monitor that wants it is built after the client in main.go.
	Observer RequestObserver
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
	// GetSeasonDetails retrieves a season's full episode list
	GetSeasonDetails(ctx context.Context, tvID int, seasonNumber int) (*SeasonDetails, error)
	// GetSeasonDetailsWithLanguage retrieves season details with a specific language
	GetSeasonDetailsWithLanguage(ctx context.Context, tvID int, seasonNumber int, language string) (*SeasonDetails, error)
	// GetMovieVideos retrieves videos (trailers, teasers) for a movie
	GetMovieVideos(ctx context.Context, movieID int) (*VideosResponse, error)
	// GetTVShowVideos retrieves videos (trailers, teasers) for a TV show
	GetTVShowVideos(ctx context.Context, tvID int) (*VideosResponse, error)
	// GetWatchProviders retrieves streaming/rent/buy providers for a movie or TV show,
	// optionally filtered to a single region (Story 12-4)
	GetWatchProviders(ctx context.Context, mediaType string, id int, region string) (*WatchProvidersResponse, error)
	// GetTVExternalIDs retrieves a TV show's external-service ids (tvdb/imdb) —
	// language-neutral (Story 13-4b, the Sonarr TVDB-resolution flow)
	GetTVExternalIDs(ctx context.Context, tvID int) (*TVExternalIDs, error)
	// GetMovieRecommendations retrieves recommended movies for a movie
	GetMovieRecommendations(ctx context.Context, movieID int) (*SearchResultMovies, error)
	// GetMovieRecommendationsWithLanguage retrieves recommended movies with a specific language
	GetMovieRecommendationsWithLanguage(ctx context.Context, movieID int, language string) (*SearchResultMovies, error)
	// GetMovieSimilar retrieves similar movies for a movie (recommendations fallback)
	GetMovieSimilar(ctx context.Context, movieID int) (*SearchResultMovies, error)
	// GetMovieSimilarWithLanguage retrieves similar movies with a specific language
	GetMovieSimilarWithLanguage(ctx context.Context, movieID int, language string) (*SearchResultMovies, error)
	// GetTVRecommendations retrieves recommended TV shows for a TV show
	GetTVRecommendations(ctx context.Context, tvID int) (*SearchResultTVShows, error)
	// GetTVRecommendationsWithLanguage retrieves recommended TV shows with a specific language
	GetTVRecommendationsWithLanguage(ctx context.Context, tvID int, language string) (*SearchResultTVShows, error)
	// GetTVSimilar retrieves similar TV shows for a TV show (recommendations fallback)
	GetTVSimilar(ctx context.Context, tvID int) (*SearchResultTVShows, error)
	// GetTVSimilarWithLanguage retrieves similar TV shows with a specific language
	GetTVSimilarWithLanguage(ctx context.Context, tvID int, language string) (*SearchResultTVShows, error)
	// FindByExternalID finds movies/TV shows by an external ID (e.g., IMDB)
	FindByExternalID(ctx context.Context, externalID string, externalSource string) (*FindByExternalIDResponse, error)
	// GetTrendingMovies retrieves the trending movies for a time window ("day" or "week")
	GetTrendingMovies(ctx context.Context, timeWindow string, page int) (*SearchResultMovies, error)
	// GetTrendingMoviesWithLanguage retrieves trending movies with a specific language
	GetTrendingMoviesWithLanguage(ctx context.Context, timeWindow string, language string, page int) (*SearchResultMovies, error)
	// GetTrendingTVShows retrieves the trending TV shows for a time window ("day" or "week")
	GetTrendingTVShows(ctx context.Context, timeWindow string, page int) (*SearchResultTVShows, error)
	// GetTrendingTVShowsWithLanguage retrieves trending TV shows with a specific language
	GetTrendingTVShowsWithLanguage(ctx context.Context, timeWindow string, language string, page int) (*SearchResultTVShows, error)
	// DiscoverMovies queries /discover/movie with the given filter params
	DiscoverMovies(ctx context.Context, params DiscoverParams) (*SearchResultMovies, error)
	// DiscoverTVShows queries /discover/tv with the given filter params
	DiscoverTVShows(ctx context.Context, params DiscoverParams) (*SearchResultTVShows, error)
}

// Client represents a TMDb API client
type Client struct {
	baseURL     string
	apiKey      string
	keyProvider KeyProvider
	language    string
	httpClient  *http.Client
	limiter     *rate.Limiter
	// observer is swapped at most once, after boot, by SetObserver; atomic so
	// a scan already in flight never races the write.
	observer atomic.Pointer[RequestObserver]
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

	c := &Client{
		baseURL:     baseURL,
		apiKey:      cfg.APIKey,
		keyProvider: cfg.KeyProvider,
		language:    language,
		httpClient: &http.Client{
			Timeout: timeout,
		},
		limiter: limiter,
	}
	if cfg.Observer != nil {
		c.SetObserver(cfg.Observer)
	}
	return c
}

// SetObserver attaches (or replaces) the RequestObserver. Safe to call while
// requests are in flight.
func (c *Client) SetObserver(o RequestObserver) {
	if o == nil {
		c.observer.Store(nil)
		return
	}
	c.observer.Store(&o)
}

func (c *Client) notify(endpoint string, err error) {
	if o := c.observer.Load(); o != nil {
		(*o)(endpoint, err)
	}
}

// apiKeyFor returns the key to use for this request: the provider's answer
// when one is configured, the static key otherwise.
func (c *Client) apiKeyFor(ctx context.Context) (string, error) {
	if c.keyProvider == nil {
		return c.apiKey, nil
	}
	key, err := c.keyProvider(ctx)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(key), nil
}

// redactAPIKey scrubs the key out of a transport error. Go's *url.Error carries
// the full request URL — `?api_key=…` included — and that text used to reach
// the stdout log and the service-status page verbatim. With a bundled key that
// would print the owner's key in every user's log (sub-7-7a). The error chain
// is kept intact (errors.Is on context/net errors still works); only the URL
// string inside it is rewritten.
func redactAPIKey(err error, apiKey string) error {
	if err == nil || apiKey == "" {
		return err
	}
	var ue *url.Error
	if errors.As(err, &ue) {
		ue.URL = strings.ReplaceAll(ue.URL, url.QueryEscape(apiKey), "REDACTED")
		ue.URL = strings.ReplaceAll(ue.URL, apiKey, "REDACTED")
	}
	return err
}

// doRequest performs an HTTP request with rate limiting and common error handling
func (c *Client) doRequest(ctx context.Context, method, endpoint string, queryParams url.Values) ([]byte, error) {
	// Resolve the key BEFORE spending a rate-limiter token: an unconfigured
	// install must not burn quota on requests that cannot succeed.
	apiKey, err := c.apiKeyFor(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve API key: %w", err)
	}
	if apiKey == "" {
		// "not configured" is the wording models.isUnconfiguredError keys on,
		// so the status page says 未設定 rather than treating this as an outage.
		notConfigured := NewUnauthorizedError("TMDb API key not configured")
		c.notify(endpoint, notConfigured)
		return nil, notConfigured
	}

	// Wait for rate limiter
	if err := c.limiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limiter error: %w", err)
	}

	// Build URL with query parameters
	reqURL, err := c.buildURL(endpoint, queryParams, apiKey)
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
		err = redactAPIKey(err, apiKey)
		slog.Error("TMDb API request failed",
			"error", err,
			"endpoint", endpoint,
		)
		wrapped := fmt.Errorf("HTTP request failed: %w", err)
		c.notify(endpoint, wrapped)
		return nil, wrapped
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
		apiErr := ParseAPIError(resp.StatusCode, body)
		c.notify(endpoint, apiErr)
		return nil, apiErr
	}

	c.notify(endpoint, nil)
	return body, nil
}

// buildURL constructs the full URL with query parameters
func (c *Client) buildURL(endpoint string, queryParams url.Values, apiKey string) (string, error) {
	// Parse base URL
	u, err := url.Parse(c.baseURL + endpoint)
	if err != nil {
		return "", err
	}

	// Add API key and language to query parameters
	if queryParams == nil {
		queryParams = url.Values{}
	}
	queryParams.Set("api_key", apiKey)

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
