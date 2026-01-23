package douban

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"math/rand"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// Client is a Douban web scraper client with rate limiting and anti-scraping measures
type Client struct {
	httpClient  *http.Client
	rateLimiter *rate.Limiter
	config      ClientConfig
	logger      *slog.Logger

	// User agent rotation
	userAgents []string
	uaIndex    int
	uaMu       sync.Mutex

	// Metrics for monitoring
	metrics *ClientMetrics
	metricsMu sync.RWMutex
}

// ClientMetrics tracks scraper performance and issues
type ClientMetrics struct {
	TotalRequests    int64
	SuccessfulRequests int64
	BlockedRequests    int64
	TimeoutRequests    int64
	RetryCount         int64
	LastRequestTime    time.Time
	LastBlockedTime    time.Time
}

// Common User-Agent strings for rotation (per Dev Notes)
var defaultUserAgents = []string{
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:121.0) Gecko/20100101 Firefox/121.0",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.2 Safari/605.1.15",
	"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
}

// NewClient creates a new Douban client with the given configuration
func NewClient(config ClientConfig, logger *slog.Logger) *Client {
	if logger == nil {
		logger = slog.Default()
	}

	// Apply defaults for zero values
	if config.RequestsPerSecond <= 0 {
		config.RequestsPerSecond = 0.5 // 1 request per 2 seconds
	}
	if config.Timeout <= 0 {
		config.Timeout = 30 * time.Second
	}
	if config.MaxRetries <= 0 {
		config.MaxRetries = 5
	}
	if config.InitialBackoff <= 0 {
		config.InitialBackoff = 1 * time.Second
	}
	if config.MaxBackoff <= 0 {
		config.MaxBackoff = 16 * time.Second
	}
	if config.BackoffMultiplier <= 0 {
		config.BackoffMultiplier = 2.0
	}
	if config.JitterMin <= 0 {
		config.JitterMin = 100 * time.Millisecond
	}
	if config.JitterMax <= 0 {
		config.JitterMax = 500 * time.Millisecond
	}

	return &Client{
		httpClient: &http.Client{
			Timeout: config.Timeout,
			// Don't follow redirects automatically to detect blocks
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= 3 {
					return fmt.Errorf("stopped after 3 redirects")
				}
				return nil
			},
		},
		rateLimiter: rate.NewLimiter(rate.Limit(config.RequestsPerSecond), 1),
		config:      config,
		logger:      logger,
		userAgents:  defaultUserAgents,
		uaIndex:     0,
		metrics:     &ClientMetrics{},
	}
}

// getNextUserAgent returns the next User-Agent in rotation
func (c *Client) getNextUserAgent() string {
	c.uaMu.Lock()
	defer c.uaMu.Unlock()

	ua := c.userAgents[c.uaIndex]
	c.uaIndex = (c.uaIndex + 1) % len(c.userAgents)
	return ua
}

// addJitter adds random jitter to a duration
func (c *Client) addJitter(d time.Duration) time.Duration {
	jitterRange := c.config.JitterMax - c.config.JitterMin
	if jitterRange <= 0 {
		return d
	}
	jitter := c.config.JitterMin + time.Duration(rand.Int63n(int64(jitterRange)))
	return d + jitter
}

// doRequest performs an HTTP request with rate limiting and retries
func (c *Client) doRequest(ctx context.Context, req *http.Request) (*http.Response, error) {
	// Wait for rate limiter
	if err := c.rateLimiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limiter: %w", err)
	}

	// Add common headers
	req.Header.Set("User-Agent", c.getNextUserAgent())
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Language", "zh-TW,zh;q=0.9,en-US;q=0.8,en;q=0.7")
	req.Header.Set("Accept-Encoding", "gzip, deflate, br")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Upgrade-Insecure-Requests", "1")

	var lastErr error
	backoff := c.config.InitialBackoff

	for attempt := 0; attempt <= c.config.MaxRetries; attempt++ {
		if attempt > 0 {
			// Apply exponential backoff with jitter
			sleepDuration := c.addJitter(backoff)
			c.logger.Info("Retrying request",
				"attempt", attempt,
				"backoff", sleepDuration,
				"url", req.URL.String(),
			)

			c.metricsMu.Lock()
			c.metrics.RetryCount++
			c.metricsMu.Unlock()

			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(sleepDuration):
			}

			// Increase backoff for next attempt
			backoff = time.Duration(float64(backoff) * c.config.BackoffMultiplier)
			if backoff > c.config.MaxBackoff {
				backoff = c.config.MaxBackoff
			}

			// Wait for rate limiter again
			if err := c.rateLimiter.Wait(ctx); err != nil {
				return nil, fmt.Errorf("rate limiter on retry: %w", err)
			}

			// Rotate User-Agent on retry
			req.Header.Set("User-Agent", c.getNextUserAgent())
		}

		c.metricsMu.Lock()
		c.metrics.TotalRequests++
		c.metrics.LastRequestTime = time.Now()
		c.metricsMu.Unlock()

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = err
			c.logger.Warn("Request failed",
				"attempt", attempt,
				"error", err,
				"url", req.URL.String(),
			)
			continue
		}

		// Check for blocking responses
		if blocked, blockErr := c.isBlocked(resp); blocked {
			resp.Body.Close()
			lastErr = blockErr

			c.metricsMu.Lock()
			c.metrics.BlockedRequests++
			c.metrics.LastBlockedTime = time.Now()
			c.metricsMu.Unlock()

			c.logger.Warn("Request blocked by anti-scraping",
				"attempt", attempt,
				"status", resp.StatusCode,
				"url", req.URL.String(),
			)
			continue
		}

		// Success
		c.metricsMu.Lock()
		c.metrics.SuccessfulRequests++
		c.metricsMu.Unlock()

		return resp, nil
	}

	// All retries exhausted
	if lastErr != nil {
		return nil, fmt.Errorf("all %d retries failed: %w", c.config.MaxRetries, lastErr)
	}
	return nil, fmt.Errorf("all %d retries failed", c.config.MaxRetries)
}

// isBlocked checks if a response indicates we've been blocked
func (c *Client) isBlocked(resp *http.Response) (bool, *BlockedError) {
	// Check status codes
	switch resp.StatusCode {
	case http.StatusForbidden: // 403
		return true, &BlockedError{
			StatusCode: resp.StatusCode,
			Reason:     "forbidden (403)",
		}
	case http.StatusTooManyRequests: // 429
		retryAfter := time.Duration(0)
		if ra := resp.Header.Get("Retry-After"); ra != "" {
			if seconds, err := time.ParseDuration(ra + "s"); err == nil {
				retryAfter = seconds
			}
		}
		return true, &BlockedError{
			StatusCode: resp.StatusCode,
			Reason:     "rate limited (429)",
			RetryAfter: retryAfter,
		}
	case http.StatusServiceUnavailable: // 503
		return true, &BlockedError{
			StatusCode: resp.StatusCode,
			Reason:     "service unavailable (503)",
		}
	}

	// Check for CAPTCHA or other blocking indicators in response
	// We'll do a lightweight check by looking at Content-Type
	contentType := resp.Header.Get("Content-Type")
	if !strings.Contains(contentType, "text/html") && resp.StatusCode == http.StatusOK {
		// Might be a CAPTCHA page or redirect
		return true, &BlockedError{
			StatusCode: resp.StatusCode,
			Reason:     "unexpected content type: " + contentType,
		}
	}

	return false, nil
}

// Get performs a GET request to the given URL
func (c *Client) Get(ctx context.Context, urlStr string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, urlStr, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	return c.doRequest(ctx, req)
}

// GetBody performs a GET request and returns the body as a string
func (c *Client) GetBody(ctx context.Context, urlStr string) (string, error) {
	resp, err := c.Get(ctx, urlStr)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read body: %w", err)
	}

	return string(body), nil
}

// SearchURL builds a Douban search URL
func SearchURL(query string, mediaType MediaType) string {
	// Douban search URL pattern from Dev Notes
	baseURL := "https://search.douban.com/movie/subject_search"
	params := url.Values{}
	params.Set("search_text", query)
	params.Set("cat", "1002") // Movies/TV category

	return baseURL + "?" + params.Encode()
}

// DetailURL builds a Douban detail page URL
func DetailURL(id string) string {
	return fmt.Sprintf("https://movie.douban.com/subject/%s/", id)
}

// GetMetrics returns a copy of the current metrics
func (c *Client) GetMetrics() ClientMetrics {
	c.metricsMu.RLock()
	defer c.metricsMu.RUnlock()
	return *c.metrics
}

// IsEnabled returns whether the client is enabled
func (c *Client) IsEnabled() bool {
	return c.config.Enabled
}

// SetEnabled enables or disables the client
func (c *Client) SetEnabled(enabled bool) {
	c.config.Enabled = enabled
}
