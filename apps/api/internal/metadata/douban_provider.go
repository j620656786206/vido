package metadata

import (
	"context"
	"sync"

	"github.com/vido/api/internal/models"
)

// DoubanProviderConfig holds configuration for the Douban provider
type DoubanProviderConfig struct {
	// Enabled controls whether the provider is active
	Enabled bool
}

// DoubanProvider is a stub implementation for Douban metadata
// Full implementation will be in Story 3.4
type DoubanProvider struct {
	config  DoubanProviderConfig
	mu      sync.RWMutex
	enabled bool
}

// NewDoubanProvider creates a new Douban provider stub
func NewDoubanProvider(config DoubanProviderConfig) *DoubanProvider {
	return &DoubanProvider{
		config:  config,
		enabled: config.Enabled,
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
	return p.enabled
}

// Status returns the current provider status
func (p *DoubanProvider) Status() ProviderStatus {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if p.enabled {
		return ProviderStatusAvailable
	}
	return ProviderStatusUnavailable
}

// SetEnabled sets the enabled status
func (p *DoubanProvider) SetEnabled(enabled bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.enabled = enabled
}

// Search performs a metadata search
// This is a stub implementation - full implementation in Story 3.4
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

	// Stub: Return not implemented error
	// Full implementation will be added in Story 3.4 (Douban Web Scraper)
	return nil, NewProviderError(
		p.Name(),
		p.Source(),
		ErrCodeUnavailable,
		"Douban provider not implemented - see Story 3.4",
		nil,
	)
}

// Compile-time interface verification
var _ MetadataProvider = (*DoubanProvider)(nil)
