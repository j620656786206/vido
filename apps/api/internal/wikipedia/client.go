package wikipedia

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

const (
	// BaseURL is the base URL for zh.wikipedia.org MediaWiki API
	BaseURL = "https://zh.wikipedia.org/w/api.php"
)

// Client is a MediaWiki API client for zh.wikipedia.org with rate limiting
// and proper User-Agent headers (NFR-I13, NFR-I14).
type Client struct {
	httpClient  *http.Client
	rateLimiter *rate.Limiter
	config      ClientConfig
	logger      *slog.Logger

	// Enabled state with mutex protection
	enabled   bool
	enabledMu sync.RWMutex

	// Metrics for monitoring
	metrics   *ClientMetrics
	metricsMu sync.RWMutex
}

// ClientMetrics tracks API performance and issues
type ClientMetrics struct {
	TotalRequests      int64
	SuccessfulRequests int64
	FailedRequests     int64
	RateLimitedCount   int64
	LastRequestTime    time.Time
}

// NewClient creates a new Wikipedia client with the given configuration
func NewClient(config ClientConfig, logger *slog.Logger) *Client {
	if logger == nil {
		logger = slog.Default()
	}

	// Apply defaults for zero values
	if config.RequestsPerSecond <= 0 {
		config.RequestsPerSecond = 1.0 // 1 request per second per NFR-I14
	}
	if config.Timeout <= 0 {
		config.Timeout = 10 * time.Second
	}
	if config.MaxRetries <= 0 {
		config.MaxRetries = 3
	}
	if config.UserAgent == "" {
		config.UserAgent = "Vido/1.0"
	}

	return &Client{
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
		rateLimiter: rate.NewLimiter(rate.Limit(config.RequestsPerSecond), 1),
		config:      config,
		logger:      logger,
		enabled:     config.Enabled,
		metrics:     &ClientMetrics{},
	}
}

// buildUserAgent creates the User-Agent header per NFR-I13
// Format: ApplicationName/Version (Contact; Description)
func (c *Client) buildUserAgent() string {
	contact := c.config.ContactEmail
	if contact == "" {
		contact = "contact@example.com"
	}
	return fmt.Sprintf("%s (https://github.com/vido; %s) Go-http-client/1.1", c.config.UserAgent, contact)
}

// doRequest performs an HTTP request with rate limiting
func (c *Client) doRequest(ctx context.Context, params url.Values) ([]byte, error) {
	// Check if client is enabled
	if !c.IsEnabled() {
		return nil, fmt.Errorf("wikipedia client is disabled")
	}

	// Wait for rate limiter (NFR-I14: 1 req/s)
	if err := c.rateLimiter.Wait(ctx); err != nil {
		c.metricsMu.Lock()
		c.metrics.RateLimitedCount++
		c.metricsMu.Unlock()
		return nil, fmt.Errorf("rate limiter: %w", err)
	}

	// Build request URL
	reqURL := BaseURL + "?" + params.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	// Set headers per NFR-I13
	req.Header.Set("User-Agent", c.buildUserAgent())
	req.Header.Set("Accept", "application/json")

	c.metricsMu.Lock()
	c.metrics.TotalRequests++
	c.metrics.LastRequestTime = time.Now()
	c.metricsMu.Unlock()

	// Execute request with retries
	var lastErr error
	for attempt := 0; attempt <= c.config.MaxRetries; attempt++ {
		if attempt > 0 {
			// Wait before retry
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(time.Duration(attempt) * time.Second):
			}

			c.logger.Info("Retrying Wikipedia API request",
				"attempt", attempt,
				"url", reqURL,
			)

			// Wait for rate limiter again
			if err := c.rateLimiter.Wait(ctx); err != nil {
				return nil, fmt.Errorf("rate limiter on retry: %w", err)
			}
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = err
			c.logger.Warn("Wikipedia API request failed",
				"attempt", attempt,
				"error", err,
			)
			continue
		}
		defer resp.Body.Close()

		// Check for rate limiting
		if resp.StatusCode == http.StatusTooManyRequests {
			c.metricsMu.Lock()
			c.metrics.RateLimitedCount++
			c.metricsMu.Unlock()
			lastErr = fmt.Errorf("rate limited (429)")
			continue
		}

		// Check for other errors
		if resp.StatusCode != http.StatusOK {
			lastErr = fmt.Errorf("unexpected status code: %d", resp.StatusCode)
			continue
		}

		// Read response body
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			lastErr = fmt.Errorf("read body: %w", err)
			continue
		}

		// Check for API errors in response
		var apiResp struct {
			Error *struct {
				Code string `json:"code"`
				Info string `json:"info"`
			} `json:"error"`
		}
		if err := json.Unmarshal(body, &apiResp); err == nil && apiResp.Error != nil {
			c.metricsMu.Lock()
			c.metrics.FailedRequests++
			c.metricsMu.Unlock()
			return nil, &APIError{
				Code: apiResp.Error.Code,
				Info: apiResp.Error.Info,
			}
		}

		c.metricsMu.Lock()
		c.metrics.SuccessfulRequests++
		c.metricsMu.Unlock()

		return body, nil
	}

	c.metricsMu.Lock()
	c.metrics.FailedRequests++
	c.metricsMu.Unlock()

	if lastErr != nil {
		return nil, fmt.Errorf("all %d retries failed: %w", c.config.MaxRetries, lastErr)
	}
	return nil, fmt.Errorf("all %d retries failed", c.config.MaxRetries)
}

// Search performs a search on zh.wikipedia.org
// Uses MediaWiki API: action=query&list=search
func (c *Client) Search(ctx context.Context, query string, limit int) ([]SearchResult, error) {
	if limit <= 0 || limit > 10 {
		limit = 5
	}

	params := url.Values{
		"action":   {"query"},
		"list":     {"search"},
		"srsearch": {query},
		"srlimit":  {fmt.Sprintf("%d", limit)},
		"format":   {"json"},
		"utf8":     {"1"},
	}

	body, err := c.doRequest(ctx, params)
	if err != nil {
		return nil, err
	}

	var response searchResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("parse search response: %w", err)
	}

	results := make([]SearchResult, 0, len(response.Query.Search))
	for _, s := range response.Query.Search {
		results = append(results, SearchResult{
			PageID:    s.PageID,
			Title:     s.Title,
			Snippet:   s.Snippet,
			WordCount: s.WordCount,
			Timestamp: s.Timestamp,
		})
	}

	if len(results) == 0 {
		return nil, &NotFoundError{Query: query}
	}

	c.logger.Debug("Wikipedia search completed",
		"query", query,
		"results", len(results),
	)

	return results, nil
}

// GetPageContent retrieves the content of a Wikipedia page by title
// Uses MediaWiki API: action=query&prop=revisions&rvprop=content
func (c *Client) GetPageContent(ctx context.Context, title string) (*PageContent, error) {
	params := url.Values{
		"action":  {"query"},
		"titles":  {title},
		"prop":    {"revisions|extracts"},
		"rvprop":  {"content"},
		"rvslots": {"main"},
		"exintro": {"1"},           // Only get intro section
		"explaintext": {"1"},       // Plain text extract
		"format":  {"json"},
		"utf8":    {"1"},
	}

	body, err := c.doRequest(ctx, params)
	if err != nil {
		return nil, err
	}

	var response pageContentResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("parse page content response: %w", err)
	}

	// Get the first (and only) page from the response
	for _, page := range response.Query.Pages {
		// Check for missing page
		if page.Missing {
			return nil, &NotFoundError{PageTitle: title}
		}

		content := &PageContent{
			PageID:  page.PageID,
			Title:   page.Title,
			Extract: page.Extract,
		}

		// Extract wikitext from revisions
		if len(page.Revisions) > 0 && page.Revisions[0].Slots.Main.Content != "" {
			content.Wikitext = page.Revisions[0].Slots.Main.Content
		}

		c.logger.Debug("Wikipedia page content retrieved",
			"title", title,
			"page_id", page.PageID,
			"wikitext_length", len(content.Wikitext),
		)

		return content, nil
	}

	return nil, &NotFoundError{PageTitle: title}
}

// GetImageInfo retrieves information about a Wikipedia image
// Uses MediaWiki API: action=query&titles=File:{filename}&prop=imageinfo
func (c *Client) GetImageInfo(ctx context.Context, filename string) (*ImageInfo, error) {
	// Ensure filename has File: prefix
	fileTitle := filename
	if len(filename) >= 5 && filename[:5] != "File:" {
		fileTitle = "File:" + filename
	}

	params := url.Values{
		"action":  {"query"},
		"titles":  {fileTitle},
		"prop":    {"imageinfo"},
		"iiprop":  {"url|size|mime"},
		"format":  {"json"},
		"utf8":    {"1"},
	}

	body, err := c.doRequest(ctx, params)
	if err != nil {
		return nil, err
	}

	var response imageInfoResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("parse image info response: %w", err)
	}

	// Get the first (and only) page from the response
	for _, page := range response.Query.Pages {
		// Check for missing page
		if page.Missing {
			return nil, &NotFoundError{PageTitle: fileTitle}
		}

		if len(page.ImageInfo) == 0 {
			return nil, &NotFoundError{PageTitle: fileTitle}
		}

		ii := page.ImageInfo[0]
		info := &ImageInfo{
			URL:            ii.URL,
			DescriptionURL: ii.DescriptionURL,
			Width:          ii.Width,
			Height:         ii.Height,
			Size:           ii.Size,
			MimeType:       ii.Mime,
		}

		c.logger.Debug("Wikipedia image info retrieved",
			"filename", filename,
			"url", info.URL,
		)

		return info, nil
	}

	return nil, &NotFoundError{PageTitle: fileTitle}
}

// IsEnabled returns whether the client is enabled (thread-safe)
func (c *Client) IsEnabled() bool {
	c.enabledMu.RLock()
	defer c.enabledMu.RUnlock()
	return c.enabled
}

// SetEnabled enables or disables the client (thread-safe)
func (c *Client) SetEnabled(enabled bool) {
	c.enabledMu.Lock()
	defer c.enabledMu.Unlock()
	c.enabled = enabled
}

// GetMetrics returns a copy of the current metrics
func (c *Client) GetMetrics() ClientMetrics {
	c.metricsMu.RLock()
	defer c.metricsMu.RUnlock()
	return *c.metrics
}

// API response structures

type searchResponse struct {
	Query struct {
		Search []struct {
			PageID    int64  `json:"pageid"`
			Title     string `json:"title"`
			Snippet   string `json:"snippet"`
			WordCount int    `json:"wordcount"`
			Timestamp string `json:"timestamp"`
		} `json:"search"`
	} `json:"query"`
}

type pageContentResponse struct {
	Query struct {
		Pages map[string]struct {
			PageID    int64  `json:"pageid"`
			Title     string `json:"title"`
			Missing   bool   `json:"missing,omitempty"`
			Extract   string `json:"extract"`
			Revisions []struct {
				Slots struct {
					Main struct {
						Content string `json:"content"`
					} `json:"main"`
				} `json:"slots"`
			} `json:"revisions"`
		} `json:"pages"`
	} `json:"query"`
}

type imageInfoResponse struct {
	Query struct {
		Pages map[string]struct {
			PageID    int64 `json:"pageid"`
			Title     string `json:"title"`
			Missing   bool   `json:"missing,omitempty"`
			ImageInfo []struct {
				URL            string `json:"url"`
				DescriptionURL string `json:"descriptionurl"`
				Width          int    `json:"width"`
				Height         int    `json:"height"`
				Size           int64  `json:"size"`
				Mime           string `json:"mime"`
			} `json:"imageinfo"`
		} `json:"pages"`
	} `json:"query"`
}
