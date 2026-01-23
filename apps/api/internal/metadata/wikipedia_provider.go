package metadata

import (
	"context"
	"sync"
	"time"

	"github.com/vido/api/internal/models"
	"golang.org/x/time/rate"
)

// WikipediaProviderConfig holds configuration for the Wikipedia provider
type WikipediaProviderConfig struct {
	// Enabled controls whether the provider is active
	Enabled bool
	// RateLimitPerSecond is the rate limit for Wikipedia API calls (default: 1 per NFR-I14)
	RateLimitPerSecond int
}

// WikipediaProvider is a stub implementation for Wikipedia metadata
// Full implementation will be in Story 3.5
type WikipediaProvider struct {
	config  WikipediaProviderConfig
	limiter *rate.Limiter
	mu      sync.RWMutex
	enabled bool
}

// NewWikipediaProvider creates a new Wikipedia provider stub
func NewWikipediaProvider(config WikipediaProviderConfig) *WikipediaProvider {
	// Apply defaults - NFR-I14 requires 1 req/s rate limit
	rateLimit := config.RateLimitPerSecond
	if rateLimit <= 0 {
		rateLimit = 1
	}

	return &WikipediaProvider{
		config:  config,
		enabled: config.Enabled,
		limiter: rate.NewLimiter(rate.Limit(rateLimit), 1), // 1 burst
	}
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
	return p.enabled
}

// Status returns the current provider status
func (p *WikipediaProvider) Status() ProviderStatus {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if p.enabled {
		return ProviderStatusAvailable
	}
	return ProviderStatusUnavailable
}

// SetEnabled sets the enabled status
func (p *WikipediaProvider) SetEnabled(enabled bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.enabled = enabled
}

// Search performs a metadata search
// This is a stub implementation - full implementation in Story 3.5
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

	// Apply rate limiting (NFR-I14: 1 req/s for Wikipedia)
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := p.limiter.Wait(ctx); err != nil {
		return nil, NewProviderError(
			p.Name(),
			p.Source(),
			ErrCodeRateLimited,
			"rate limit exceeded",
			err,
		)
	}

	// Stub: Return not implemented error
	// Full implementation will be added in Story 3.5 (Wikipedia Metadata Fallback)
	return nil, NewProviderError(
		p.Name(),
		p.Source(),
		ErrCodeUnavailable,
		"Wikipedia provider not implemented - see Story 3.5",
		nil,
	)
}

// Compile-time interface verification
var _ MetadataProvider = (*WikipediaProvider)(nil)
