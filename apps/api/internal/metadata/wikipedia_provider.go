package metadata

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"

	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/wikipedia"
)

// WikipediaProviderConfig holds configuration for the Wikipedia provider
type WikipediaProviderConfig struct {
	// Enabled controls whether the provider is active
	Enabled bool
	// ClientConfig holds configuration for the Wikipedia HTTP client
	ClientConfig wikipedia.ClientConfig
	// CircuitBreakerConfig holds configuration for the circuit breaker
	CircuitBreakerConfig CircuitBreakerConfig
	// CacheConfig holds configuration for the result cache
	CacheConfig WikipediaCacheConfig
}

// WikipediaCacheConfig holds cache configuration for the Wikipedia provider
type WikipediaCacheConfig struct {
	// Enabled controls whether caching is enabled
	Enabled bool
}

// DefaultWikipediaProviderConfig returns a default configuration for the Wikipedia provider
func DefaultWikipediaProviderConfig() WikipediaProviderConfig {
	return WikipediaProviderConfig{
		Enabled:      true,
		ClientConfig: wikipedia.DefaultConfig(),
		CircuitBreakerConfig: CircuitBreakerConfig{
			FailureThreshold: 5,  // Open after 5 failures (more lenient for Wikipedia)
			SuccessThreshold: 2,  // Close after 2 successes
			Timeout:          30, // 30 seconds timeout for half-open state
		},
		CacheConfig: WikipediaCacheConfig{
			Enabled: true,
		},
	}
}

// WikipediaProvider implements MetadataProvider for Wikipedia
type WikipediaProvider struct {
	config           WikipediaProviderConfig
	client           *wikipedia.Client
	searcher         *wikipedia.Searcher
	infoboxParser    *wikipedia.InfoboxParser
	contentExtractor *wikipedia.ContentExtractor
	imageExtractor   *wikipedia.ImageExtractor
	cache            *WikipediaCache
	circuitBreaker   *CircuitBreaker
	logger           *slog.Logger

	mu      sync.RWMutex
	enabled bool
}

// NewWikipediaProvider creates a new Wikipedia provider
func NewWikipediaProvider(config WikipediaProviderConfig) *WikipediaProvider {
	return NewWikipediaProviderWithLogger(config, nil, nil)
}

// NewWikipediaProviderWithDB creates a new Wikipedia provider with database connection for caching
func NewWikipediaProviderWithDB(config WikipediaProviderConfig, db *sql.DB) *WikipediaProvider {
	return NewWikipediaProviderWithLogger(config, nil, db)
}

// NewWikipediaProviderWithLogger creates a new Wikipedia provider with a custom logger and optional database
func NewWikipediaProviderWithLogger(config WikipediaProviderConfig, logger *slog.Logger, db *sql.DB) *WikipediaProvider {
	if logger == nil {
		logger = slog.Default()
	}

	// Create the Wikipedia client
	client := wikipedia.NewClient(config.ClientConfig, logger)

	// Create the circuit breaker
	cb := NewCircuitBreaker("wikipedia", config.CircuitBreakerConfig)

	// Create cache if database connection is provided
	var cache *WikipediaCache
	if db != nil && config.CacheConfig.Enabled {
		cache = NewWikipediaCache(db, logger)
		logger.Info("Wikipedia cache enabled")
	}

	return &WikipediaProvider{
		config:           config,
		client:           client,
		searcher:         wikipedia.NewSearcher(client, logger),
		infoboxParser:    wikipedia.NewInfoboxParser(),
		contentExtractor: wikipedia.NewContentExtractor(),
		imageExtractor:   wikipedia.NewImageExtractor(client, logger),
		cache:            cache,
		circuitBreaker:   cb,
		logger:           logger,
		enabled:          config.Enabled,
	}
}

// SetCache sets the cache for the provider (useful for deferred initialization)
func (p *WikipediaProvider) SetCache(db *sql.DB) {
	if db != nil && p.config.CacheConfig.Enabled {
		p.cache = NewWikipediaCache(db, p.logger)
		p.logger.Info("Wikipedia cache initialized")
	}
}

// Close closes the provider and releases resources
func (p *WikipediaProvider) Close() {
	// No resources to release for Wikipedia provider
}

// Name returns the provider name
func (p *WikipediaProvider) Name() string {
	return "Wikipedia"
}

// Source returns the metadata source
func (p *WikipediaProvider) Source() models.MetadataSource {
	return models.MetadataSourceWikipedia
}

// IsAvailable returns whether the provider is available
func (p *WikipediaProvider) IsAvailable() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if !p.enabled {
		return false
	}

	// Also check circuit breaker state
	return p.circuitBreaker.State() != CircuitStateOpen
}

// Status returns the current provider status
func (p *WikipediaProvider) Status() ProviderStatus {
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
func (p *WikipediaProvider) SetEnabled(enabled bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.enabled = enabled
}

// Search performs a metadata search using Wikipedia MediaWiki API
func (p *WikipediaProvider) Search(ctx context.Context, req *SearchRequest) (*SearchResult, error) {
	p.mu.RLock()
	enabled := p.enabled
	p.mu.RUnlock()

	if !enabled {
		return nil, NewProviderError(
			p.Name(),
			p.Source(),
			ErrCodeUnavailable,
			"Wikipedia provider is disabled",
			nil,
		)
	}

	// Check circuit breaker
	if p.circuitBreaker.State() == CircuitStateOpen {
		return nil, NewProviderError(
			p.Name(),
			p.Source(),
			ErrCodeCircuitOpen,
			"Wikipedia provider circuit breaker is open",
			ErrCircuitOpen,
		)
	}

	// Check cache first
	if p.cache != nil {
		cached, cacheErr := p.cache.Get(ctx, req.Query)
		if cacheErr != nil {
			p.logger.Warn("Cache lookup failed",
				"query", req.Query,
				"error", cacheErr,
			)
		} else if cached != nil {
			p.logger.Debug("Cache hit for Wikipedia search",
				"query", req.Query,
				"title", cached.Title,
			)
			return &SearchResult{
				Items:      []MetadataItem{*cached},
				Source:     p.Source(),
				TotalCount: 1,
				Page:       req.Page,
				TotalPages: 1,
			}, nil
		}
	}

	// Determine media type for search optimization
	var mediaType wikipedia.MediaType
	switch req.MediaType {
	case MediaTypeMovie:
		mediaType = wikipedia.MediaTypeMovie
	case MediaTypeTV:
		mediaType = wikipedia.MediaTypeTV
	default:
		mediaType = "" // No media type filter
	}

	// Build search options
	opts := wikipedia.SearchOptions{
		Limit:                    5,
		MediaType:                mediaType,
		PreferTraditionalChinese: true,
	}

	// Execute search with circuit breaker
	var searchResults []wikipedia.RankedResult
	err := p.circuitBreaker.Execute(func() error {
		var searchErr error
		searchResults, searchErr = p.searcher.Search(ctx, req.Query, opts)
		return searchErr
	})

	if err != nil {
		return p.handleSearchError(err, req)
	}

	if len(searchResults) == 0 {
		return &SearchResult{
			Items:      []MetadataItem{},
			Source:     p.Source(),
			TotalCount: 0,
			Page:       req.Page,
			TotalPages: 0,
		}, nil
	}

	// Get the top result and fetch full metadata
	topResult := searchResults[0]
	item, err := p.fetchFullMetadata(ctx, topResult)
	if err != nil {
		p.logger.Warn("Failed to fetch Wikipedia metadata",
			"pageTitle", topResult.Title,
			"error", err,
		)
		// Return empty result instead of error - Wikipedia is a fallback
		return &SearchResult{
			Items:      []MetadataItem{},
			Source:     p.Source(),
			TotalCount: 0,
			Page:       req.Page,
			TotalPages: 0,
		}, nil
	}

	// Store in cache
	if p.cache != nil {
		if cacheErr := p.cache.Set(ctx, req.Query, item); cacheErr != nil {
			p.logger.Warn("Failed to cache Wikipedia result",
				"query", req.Query,
				"error", cacheErr,
			)
		}
	}

	return &SearchResult{
		Items:      []MetadataItem{*item},
		Source:     p.Source(),
		TotalCount: 1,
		Page:       req.Page,
		TotalPages: 1,
	}, nil
}

// fetchFullMetadata fetches complete metadata for a Wikipedia page
func (p *WikipediaProvider) fetchFullMetadata(ctx context.Context, result wikipedia.RankedResult) (*MetadataItem, error) {
	// Get page content with circuit breaker
	var content *wikipedia.PageContent
	err := p.circuitBreaker.Execute(func() error {
		var fetchErr error
		content, fetchErr = p.client.GetPageContent(ctx, result.Title)
		return fetchErr
	})
	if err != nil {
		return nil, err
	}

	// Parse Infobox
	infobox, infoboxErr := p.infoboxParser.Parse(content.Wikitext)
	if infoboxErr != nil {
		p.logger.Debug("Infobox parsing failed",
			"pageTitle", result.Title,
			"error", infoboxErr,
		)
		// Continue without Infobox data
	}

	// Extract summary
	summary := p.contentExtractor.ExtractSummary(content)
	if len(summary) > 500 {
		summary = p.contentExtractor.TruncateSummary(summary, 500)
	}

	// Extract image
	var imageResult *wikipedia.ImageResult
	if infobox != nil {
		imageResult = p.imageExtractor.ExtractFromInfobox(ctx, infobox)
	}
	if imageResult == nil || !imageResult.HasImage {
		imageResult = p.imageExtractor.ExtractFromPage(ctx, content)
	}

	// Build metadata item
	item := p.buildMetadataItem(result, content, infobox, summary, imageResult)
	return item, nil
}

// buildMetadataItem creates a MetadataItem from Wikipedia data
func (p *WikipediaProvider) buildMetadataItem(
	result wikipedia.RankedResult,
	content *wikipedia.PageContent,
	infobox *wikipedia.InfoboxData,
	summary string,
	imageResult *wikipedia.ImageResult,
) *MetadataItem {
	item := &MetadataItem{
		ID:         fmt.Sprintf("%d", content.PageID),
		Confidence: result.Confidence,
		MediaType:  MediaTypeMovie, // Default
	}

	// Set title - prefer Infobox title (Name field)
	if infobox != nil && infobox.Name != "" {
		item.Title = infobox.Name
		item.TitleZhTW = infobox.Name // Wikipedia zh is Traditional Chinese
	} else {
		item.Title = result.Title
		item.TitleZhTW = result.Title
	}

	// Set original title
	if infobox != nil && infobox.OriginalName != "" {
		item.OriginalTitle = infobox.OriginalName
	}

	// Set year
	if infobox != nil && infobox.Year > 0 {
		item.Year = infobox.Year
	}

	// Set overview
	item.Overview = summary
	item.OverviewZhTW = summary // Wikipedia zh content is in Traditional Chinese

	// Set genres
	if infobox != nil && len(infobox.Genre) > 0 {
		item.Genres = infobox.Genre
	}

	// Set poster URL
	if imageResult != nil && imageResult.HasImage {
		item.PosterURL = imageResult.URL
	}

	// Determine media type from Infobox type string
	if infobox != nil {
		item.MediaType = p.determineMediaType(infobox.Type)
	}

	// Store raw data for debugging
	item.RawData = map[string]interface{}{
		"pageTitle":   result.Title,
		"pageID":      content.PageID,
		"infobox":     infobox,
		"imageResult": imageResult,
	}

	return item
}

// determineMediaType maps Infobox type strings to MediaType
func (p *WikipediaProvider) determineMediaType(infoboxType string) MediaType {
	infoboxLower := strings.ToLower(infoboxType)

	// Film templates
	if strings.Contains(infoboxLower, "film") ||
		strings.Contains(infoboxLower, "movie") ||
		strings.Contains(infoboxLower, "電影") {
		return MediaTypeMovie
	}

	// TV templates
	if strings.Contains(infoboxLower, "television") ||
		strings.Contains(infoboxLower, "tv") ||
		strings.Contains(infoboxLower, "電視") {
		return MediaTypeTV
	}

	// Anime templates - default to TV for series
	if strings.Contains(infoboxLower, "animanga") ||
		strings.Contains(infoboxLower, "anime") ||
		strings.Contains(infoboxLower, "動畫") {
		return MediaTypeTV
	}

	return MediaTypeMovie // Default
}

// handleSearchError handles errors from Wikipedia search
func (p *WikipediaProvider) handleSearchError(err error, req *SearchRequest) (*SearchResult, error) {
	// Check for not found
	var notFoundErr *wikipedia.NotFoundError
	if errors.As(err, &notFoundErr) {
		return &SearchResult{
			Items:      []MetadataItem{},
			Source:     p.Source(),
			TotalCount: 0,
			Page:       req.Page,
			TotalPages: 0,
		}, nil
	}

	// Check for parse errors
	var parseErr *wikipedia.ParseError
	if errors.As(err, &parseErr) {
		return nil, NewProviderError(
			p.Name(),
			p.Source(),
			wikipedia.ErrCodeParseError,
			"Wikipedia parse error: "+parseErr.Error(),
			err,
		)
	}

	// Check for API errors
	var apiErr *wikipedia.APIError
	if errors.As(err, &apiErr) {
		return nil, NewProviderError(
			p.Name(),
			p.Source(),
			wikipedia.ErrCodeAPIError,
			"Wikipedia API error: "+apiErr.Error(),
			err,
		)
	}

	// Check for context timeout/deadline
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return nil, NewProviderError(
			p.Name(),
			p.Source(),
			wikipedia.ErrCodeTimeout,
			"Wikipedia request timeout",
			err,
		)
	}

	// Generic error fallback
	return nil, NewProviderError(
		p.Name(),
		p.Source(),
		ErrCodeUnavailable,
		"Wikipedia search failed: "+err.Error(),
		err,
	)
}

// GetCircuitBreakerStats returns the circuit breaker statistics
func (p *WikipediaProvider) GetCircuitBreakerStats() CircuitBreakerStats {
	return p.circuitBreaker.Stats()
}

// ResetCircuitBreaker resets the circuit breaker to closed state
func (p *WikipediaProvider) ResetCircuitBreaker() {
	p.circuitBreaker.Reset()
}

// GetCacheStats returns the cache statistics (nil if cache is not enabled)
func (p *WikipediaProvider) GetCacheStats(ctx context.Context) (*WikipediaCacheStats, error) {
	if p.cache == nil {
		return nil, nil
	}
	return p.cache.Stats(ctx)
}

// Compile-time interface verification
var _ MetadataProvider = (*WikipediaProvider)(nil)
