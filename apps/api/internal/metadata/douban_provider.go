package metadata

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	"github.com/vido/api/internal/douban"
	"github.com/vido/api/internal/models"
)

// DoubanProviderConfig holds configuration for the Douban provider
type DoubanProviderConfig struct {
	// Enabled controls whether the provider is active
	Enabled bool
	// ClientConfig holds configuration for the Douban HTTP client
	ClientConfig douban.ClientConfig
	// CircuitBreakerConfig holds configuration for the circuit breaker
	CircuitBreakerConfig CircuitBreakerConfig
}

// DefaultDoubanProviderConfig returns a default configuration for the Douban provider
func DefaultDoubanProviderConfig() DoubanProviderConfig {
	return DoubanProviderConfig{
		Enabled:      true,
		ClientConfig: douban.DefaultConfig(),
		CircuitBreakerConfig: CircuitBreakerConfig{
			FailureThreshold: 3,              // Open after 3 failures (more sensitive for scraping)
			SuccessThreshold: 2,              // Close after 2 successes
			Timeout:          60 * time.Second, // Longer timeout for anti-scraping recovery
		},
	}
}

// DoubanProvider implements MetadataProvider for Douban
type DoubanProvider struct {
	config         DoubanProviderConfig
	client         *douban.Client
	searcher       *douban.Searcher
	scraper        *douban.Scraper
	circuitBreaker *CircuitBreaker
	logger         *slog.Logger

	mu      sync.RWMutex
	enabled bool
}

// NewDoubanProvider creates a new Douban provider
func NewDoubanProvider(config DoubanProviderConfig) *DoubanProvider {
	return NewDoubanProviderWithLogger(config, nil)
}

// NewDoubanProviderWithLogger creates a new Douban provider with a custom logger
func NewDoubanProviderWithLogger(config DoubanProviderConfig, logger *slog.Logger) *DoubanProvider {
	if logger == nil {
		logger = slog.Default()
	}

	// Create the Douban client
	client := douban.NewClient(config.ClientConfig, logger)

	// Create the circuit breaker
	cb := NewCircuitBreaker("douban", config.CircuitBreakerConfig)

	return &DoubanProvider{
		config:         config,
		client:         client,
		searcher:       douban.NewSearcher(client, logger),
		scraper:        douban.NewScraper(client, logger),
		circuitBreaker: cb,
		logger:         logger,
		enabled:        config.Enabled,
	}
}

// Name returns the provider name
func (p *DoubanProvider) Name() string {
	return "Douban"
}

// Source returns the metadata source
func (p *DoubanProvider) Source() models.MetadataSource {
	return models.MetadataSourceDouban
}

// IsAvailable returns whether the provider is available
func (p *DoubanProvider) IsAvailable() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if !p.enabled {
		return false
	}

	// Also check circuit breaker state
	return p.circuitBreaker.State() != CircuitStateOpen
}

// Status returns the current provider status
func (p *DoubanProvider) Status() ProviderStatus {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if !p.enabled {
		return ProviderStatusUnavailable
	}

	switch p.circuitBreaker.State() {
	case CircuitStateOpen:
		return ProviderStatusRateLimited
	case CircuitStateHalfOpen:
		return ProviderStatusAvailable // Allow some requests
	default:
		return ProviderStatusAvailable
	}
}

// SetEnabled sets the enabled status
func (p *DoubanProvider) SetEnabled(enabled bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.enabled = enabled
	p.client.SetEnabled(enabled)
}

// Search performs a metadata search using Douban web scraping
func (p *DoubanProvider) Search(ctx context.Context, req *SearchRequest) (*SearchResult, error) {
	p.mu.RLock()
	enabled := p.enabled
	p.mu.RUnlock()

	if !enabled {
		return nil, NewProviderError(
			p.Name(),
			p.Source(),
			ErrCodeUnavailable,
			"Douban provider is disabled",
			nil,
		)
	}

	// Check circuit breaker
	if p.circuitBreaker.State() == CircuitStateOpen {
		return nil, NewProviderError(
			p.Name(),
			p.Source(),
			ErrCodeCircuitOpen,
			"Douban provider circuit breaker is open",
			ErrCircuitOpen,
		)
	}

	// Convert media type
	mediaType := douban.MediaTypeMovie
	if req.MediaType == MediaTypeTV {
		mediaType = douban.MediaTypeTV
	}

	var searchResults []douban.SearchResult
	var items []MetadataItem

	// Execute search with circuit breaker
	err := p.circuitBreaker.Execute(func() error {
		var searchErr error
		searchResults, searchErr = p.searcher.Search(ctx, req.Query, mediaType)
		return searchErr
	})

	if err != nil {
		// Check if this is a blocking error
		var blockedErr *douban.BlockedError
		if errors.As(err, &blockedErr) {
			return nil, NewProviderError(
				p.Name(),
				p.Source(),
				douban.ErrCodeBlocked,
				"Douban blocked the request: "+blockedErr.Reason,
				err,
			)
		}

		// Check for not found
		var notFoundErr *douban.NotFoundError
		if errors.As(err, &notFoundErr) {
			return &SearchResult{
				Items:      []MetadataItem{},
				Source:     p.Source(),
				TotalCount: 0,
				Page:       req.Page,
				TotalPages: 0,
			}, nil
		}

		return nil, NewProviderError(
			p.Name(),
			p.Source(),
			ErrCodeUnavailable,
			"Douban search failed",
			err,
		)
	}

	// For each search result, scrape the detail page to get full metadata
	for _, sr := range searchResults {
		// Rate limit: only scrape first 5 results to be polite
		if len(items) >= 5 {
			break
		}

		var detail *douban.DetailResult
		err := p.circuitBreaker.Execute(func() error {
			var scrapeErr error
			detail, scrapeErr = p.scraper.ScrapeDetail(ctx, sr.ID)
			return scrapeErr
		})

		if err != nil {
			p.logger.Warn("Failed to scrape Douban detail",
				"id", sr.ID,
				"error", err,
			)
			// Continue with next result instead of failing completely
			continue
		}

		items = append(items, p.convertToMetadataItem(detail))
	}

	return &SearchResult{
		Items:      items,
		Source:     p.Source(),
		TotalCount: len(items),
		Page:       req.Page,
		TotalPages: 1, // Douban search doesn't provide pagination info
	}, nil
}

// convertToMetadataItem converts a Douban detail result to a normalized MetadataItem
func (p *DoubanProvider) convertToMetadataItem(detail *douban.DetailResult) MetadataItem {
	item := MetadataItem{
		ID:            detail.ID,
		Title:         detail.Title,
		OriginalTitle: detail.OriginalTitle,
		Year:          detail.Year,
		ReleaseDate:   detail.ReleaseDate,
		Overview:      detail.Summary,
		PosterURL:     detail.PosterURL,
		Rating:        detail.Rating,
		VoteCount:     detail.RatingCount,
		Genres:        detail.Genres,
		MediaType:     MediaTypeMovie,
		Confidence:    0.8, // Default confidence for Douban results
	}

	// Set Traditional Chinese fields
	if detail.TitleTraditional != "" {
		item.TitleZhTW = detail.TitleTraditional
	}
	if detail.SummaryTraditional != "" {
		item.OverviewZhTW = detail.SummaryTraditional
	}

	// Set media type
	if detail.Type == douban.MediaTypeTV {
		item.MediaType = MediaTypeTV
	}

	// Store raw data for debugging
	item.RawData = detail

	return item
}

// GetCircuitBreakerStats returns the circuit breaker statistics
func (p *DoubanProvider) GetCircuitBreakerStats() CircuitBreakerStats {
	return p.circuitBreaker.Stats()
}

// ResetCircuitBreaker resets the circuit breaker to closed state
func (p *DoubanProvider) ResetCircuitBreaker() {
	p.circuitBreaker.Reset()
}

// GetClientMetrics returns the HTTP client metrics
func (p *DoubanProvider) GetClientMetrics() douban.ClientMetrics {
	return p.client.GetMetrics()
}

// Compile-time interface verification
var _ MetadataProvider = (*DoubanProvider)(nil)
