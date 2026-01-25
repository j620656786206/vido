package metadata

import (
	"context"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/vido/api/internal/ai"
	"github.com/vido/api/internal/models"
)

// OrchestratorConfig holds configuration for the orchestrator
type OrchestratorConfig struct {
	// FallbackDelay is the delay between provider attempts (default: 100ms, must be <1s per NFR-R3)
	FallbackDelay time.Duration
	// EnableCircuitBreaker enables circuit breaker for providers
	EnableCircuitBreaker bool
	// CircuitBreakerConfig is the configuration for circuit breakers
	CircuitBreakerConfig CircuitBreakerConfig
}

// SourceAttempt represents a single provider attempt result
type SourceAttempt struct {
	// Source is the metadata source that was attempted
	Source models.MetadataSource `json:"source"`
	// Success indicates if the attempt was successful
	Success bool `json:"success"`
	// Skipped indicates if the provider was skipped
	Skipped bool `json:"skipped,omitempty"`
	// SkipReason is the reason for skipping
	SkipReason string `json:"skipReason,omitempty"`
	// Error is the error if the attempt failed
	Error error `json:"-"`
	// Duration is how long the attempt took
	Duration time.Duration `json:"duration"`
}

// KeywordAttempt represents a single keyword retry attempt
type KeywordAttempt struct {
	// Keyword is the alternative keyword that was tried
	Keyword string `json:"keyword"`
	// Success indicates if the attempt found results
	Success bool `json:"success"`
}

// FallbackStatus represents the status of a fallback chain execution
type FallbackStatus struct {
	// Attempts contains all provider attempts in order
	Attempts []SourceAttempt `json:"attempts"`
	// TotalDuration is the total time spent in the fallback chain
	TotalDuration time.Duration `json:"totalDuration"`
	// Cancelled indicates if the search was cancelled
	Cancelled bool `json:"cancelled,omitempty"`
	// KeywordAttempts contains the AI-generated keyword retry attempts (Story 3.6)
	KeywordAttempts []KeywordAttempt `json:"keywordAttempts,omitempty"`
	// SuccessfulKeyword is the keyword that succeeded (if any)
	SuccessfulKeyword string `json:"successfulKeyword,omitempty"`
	// KeywordError captures keyword-related errors (Story 3.6)
	KeywordError error `json:"-"`
}

// AllFailed returns true if all attempts failed or were skipped
func (s *FallbackStatus) AllFailed() bool {
	if len(s.Attempts) == 0 {
		return true
	}
	for _, attempt := range s.Attempts {
		if attempt.Success {
			return false
		}
	}
	return true
}

// StatusString returns a human-readable status string (e.g., "TMDb ❌ → Douban ✓")
func (s *FallbackStatus) StatusString() string {
	if len(s.Attempts) == 0 {
		return ""
	}

	var parts []string
	for _, attempt := range s.Attempts {
		var symbol string
		switch {
		case attempt.Success:
			symbol = "✓"
		case attempt.Skipped:
			symbol = "⏭"
		default:
			symbol = "❌"
		}
		parts = append(parts, sourceDisplayName(attempt.Source)+" "+symbol)
	}

	result := strings.Join(parts, " → ")

	// If all failed, add manual search suggestion
	if s.AllFailed() {
		result += " → Manual search"
	}

	return result
}

// sourceDisplayName returns the display name for a metadata source
func sourceDisplayName(source models.MetadataSource) string {
	switch source {
	case models.MetadataSourceTMDb:
		return "TMDb"
	case models.MetadataSourceDouban:
		return "Douban"
	case models.MetadataSourceWikipedia:
		return "Wikipedia"
	case models.MetadataSourceManual:
		return "Manual"
	default:
		return string(source)
	}
}

// KeywordVariants holds alternative search terms for a media title.
// This is a local copy to avoid circular imports with the ai package.
type KeywordVariants struct {
	Original             string   `json:"original"`
	SimplifiedChinese    string   `json:"simplified_chinese,omitempty"`
	TraditionalChinese   string   `json:"traditional_chinese,omitempty"`
	English              string   `json:"english,omitempty"`
	Romaji               string   `json:"romaji,omitempty"`
	Pinyin               string   `json:"pinyin,omitempty"`
	AlternativeSpellings []string `json:"alternative_spellings,omitempty"`
	CommonAliases        []string `json:"common_aliases,omitempty"`
}

// GetPrioritizedList returns keywords in search priority order.
func (k *KeywordVariants) GetPrioritizedList() []string {
	seen := make(map[string]bool)
	var keywords []string

	add := func(keyword string) {
		if keyword != "" && keyword != k.Original && !seen[keyword] {
			seen[keyword] = true
			keywords = append(keywords, keyword)
		}
	}

	add(k.English)
	add(k.Romaji)
	add(k.SimplifiedChinese)
	add(k.TraditionalChinese)

	for _, spelling := range k.AlternativeSpellings {
		add(spelling)
	}

	for _, alias := range k.CommonAliases {
		add(alias)
	}

	return keywords
}

// KeywordGenerator generates alternative search keywords using AI.
type KeywordGenerator interface {
	GenerateKeywords(ctx context.Context, title string) (*KeywordVariants, error)
}

// ProgressCallback is called for each provider attempt
type ProgressCallback func(attempt SourceAttempt)

// SearchOption is a functional option for Search
type SearchOption func(*searchOptions)

type searchOptions struct {
	progressCallback ProgressCallback
}

// WithProgressCallback sets a callback for progress events
func WithProgressCallback(cb ProgressCallback) SearchOption {
	return func(o *searchOptions) {
		o.progressCallback = cb
	}
}

// Orchestrator manages the fallback chain of metadata providers
type Orchestrator struct {
	config           OrchestratorConfig
	providers        []MetadataProvider
	circuitBreakers  map[string]*CircuitBreaker
	keywordGenerator KeywordGenerator
	mu               sync.RWMutex
}

// NewOrchestrator creates a new orchestrator
func NewOrchestrator(config OrchestratorConfig) *Orchestrator {
	// Apply defaults
	if config.FallbackDelay <= 0 {
		config.FallbackDelay = 100 * time.Millisecond
	}
	// Enforce NFR-R3: <1 second fallback delay
	if config.FallbackDelay >= time.Second {
		config.FallbackDelay = 900 * time.Millisecond
	}

	return &Orchestrator{
		config:          config,
		providers:       []MetadataProvider{},
		circuitBreakers: make(map[string]*CircuitBreaker),
	}
}

// RegisterProvider adds a provider to the fallback chain
func (o *Orchestrator) RegisterProvider(provider MetadataProvider) {
	o.mu.Lock()
	defer o.mu.Unlock()

	o.providers = append(o.providers, provider)

	// Create circuit breaker if enabled
	if o.config.EnableCircuitBreaker {
		cbConfig := o.config.CircuitBreakerConfig
		cbConfig.OnStateChange = func(name string, from, to CircuitState) {
			slog.Info("Provider circuit breaker state changed",
				"provider", name,
				"from", from.String(),
				"to", to.String(),
			)
		}
		o.circuitBreakers[provider.Name()] = NewCircuitBreaker(provider.Name(), cbConfig)
	}

	slog.Info("Registered metadata provider",
		"provider", provider.Name(),
		"source", provider.Source(),
		"circuit_breaker", o.config.EnableCircuitBreaker,
	)
}

// Providers returns the list of registered providers
func (o *Orchestrator) Providers() []MetadataProvider {
	o.mu.RLock()
	defer o.mu.RUnlock()

	result := make([]MetadataProvider, len(o.providers))
	copy(result, o.providers)
	return result
}

// SetKeywordGenerator sets the AI keyword generator for the retry phase (Story 3.6)
func (o *Orchestrator) SetKeywordGenerator(generator KeywordGenerator) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.keywordGenerator = generator
}

// Search executes the fallback chain and returns the first successful result
func (o *Orchestrator) Search(ctx context.Context, req *SearchRequest, opts ...SearchOption) (*SearchResult, *FallbackStatus) {
	options := &searchOptions{}
	for _, opt := range opts {
		opt(options)
	}

	o.mu.RLock()
	providers := make([]MetadataProvider, len(o.providers))
	copy(providers, o.providers)
	o.mu.RUnlock()

	status := &FallbackStatus{
		Attempts: make([]SourceAttempt, 0, len(providers)),
	}

	startTime := time.Now()

	for i, provider := range providers {
		// Check context cancellation
		select {
		case <-ctx.Done():
			status.Cancelled = true
			status.TotalDuration = time.Since(startTime)
			slog.Debug("Search cancelled",
				"reason", ctx.Err(),
				"attempts", len(status.Attempts),
			)
			return nil, status
		default:
		}

		attempt := o.tryProvider(ctx, provider, req)
		status.Attempts = append(status.Attempts, attempt)

		// Check for context cancellation in the attempt
		if ctx.Err() != nil {
			status.Cancelled = true
			status.TotalDuration = time.Since(startTime)
			slog.Debug("Search cancelled during provider attempt",
				"provider", provider.Name(),
				"reason", ctx.Err(),
			)
			return nil, status
		}

		// Notify progress callback
		if options.progressCallback != nil {
			options.progressCallback(attempt)
		}

		// If successful, return immediately
		if attempt.Success {
			status.TotalDuration = time.Since(startTime)
			slog.Info("Metadata search successful",
				"provider", provider.Name(),
				"query", req.Query,
				"attempts", len(status.Attempts),
				"duration", status.TotalDuration,
			)

			// Execute the actual search since tryProvider only checks availability
			result, err := o.executeSearch(ctx, provider, req)
			if err != nil {
				// Mark as failed if actual execution fails
				status.Attempts[len(status.Attempts)-1].Success = false
				status.Attempts[len(status.Attempts)-1].Error = err
				continue
			}
			return result, status
		}

		// Apply fallback delay before next provider (except for last)
		if i < len(providers)-1 && o.config.FallbackDelay > 0 {
			select {
			case <-ctx.Done():
				status.Cancelled = true
				status.TotalDuration = time.Since(startTime)
				return nil, status
			case <-time.After(o.config.FallbackDelay):
			}
		}
	}

	// Story 3.6: AI Keyword Retry Phase
	// If all providers failed, try AI-generated alternative keywords
	o.mu.RLock()
	keywordGen := o.keywordGenerator
	o.mu.RUnlock()

	if keywordGen != nil {
		slog.Debug("Starting AI keyword retry phase",
			"original_query", req.Query,
		)

		variants, err := keywordGen.GenerateKeywords(ctx, req.Query)
		if err != nil {
			slog.Warn("AI keyword generation failed",
				"error", err,
				"query", req.Query,
			)
			status.KeywordError = ai.ErrKeywordGenerationFailed
		} else if variants != nil {
			keywords := variants.GetPrioritizedList()
			if len(keywords) == 0 {
				slog.Warn("No alternative keywords generated",
					"query", req.Query,
				)
				status.KeywordError = ai.ErrKeywordNoAlternatives
			} else {
				for _, keyword := range keywords {
					// Check context cancellation
					select {
					case <-ctx.Done():
						status.Cancelled = true
						status.TotalDuration = time.Since(startTime)
						return nil, status
					default:
					}

					// Record keyword attempt
					keywordAttempt := KeywordAttempt{Keyword: keyword}

					// Try all providers with the alternative keyword
					altReq := &SearchRequest{
						Query:     keyword,
						Year:      req.Year,
						MediaType: req.MediaType,
						Language:  req.Language,
						Page:      req.Page,
					}

					for _, provider := range providers {
						result, err := o.executeSearchWithCircuitBreaker(ctx, provider, altReq)
						if err == nil && result != nil && result.HasResults() {
							keywordAttempt.Success = true
							status.KeywordAttempts = append(status.KeywordAttempts, keywordAttempt)
							status.SuccessfulKeyword = keyword
							status.TotalDuration = time.Since(startTime)

							slog.Info("Metadata search successful with alternative keyword",
								"original_query", req.Query,
								"successful_keyword", keyword,
								"provider", provider.Name(),
								"duration", status.TotalDuration,
							)
							return result, status
						}
					}

					status.KeywordAttempts = append(status.KeywordAttempts, keywordAttempt)
				}
				// If we tried all keywords and none succeeded
				if len(status.KeywordAttempts) > 0 && status.SuccessfulKeyword == "" {
					status.KeywordError = ai.ErrKeywordAllFailed
				}
			}
		}
	}

	status.TotalDuration = time.Since(startTime)
	slog.Warn("All metadata providers failed",
		"query", req.Query,
		"attempts", len(status.Attempts),
		"keyword_attempts", len(status.KeywordAttempts),
		"duration", status.TotalDuration,
	)

	return nil, status
}

// tryProvider attempts to use a provider and returns the attempt result
func (o *Orchestrator) tryProvider(ctx context.Context, provider MetadataProvider, req *SearchRequest) SourceAttempt {
	attempt := SourceAttempt{
		Source: provider.Source(),
	}

	startTime := time.Now()

	// Check if provider is available
	if !provider.IsAvailable() {
		attempt.Skipped = true
		attempt.SkipReason = "provider unavailable"
		attempt.Duration = time.Since(startTime)
		slog.Debug("Skipping unavailable provider",
			"provider", provider.Name(),
		)
		return attempt
	}

	// Check circuit breaker if enabled
	if o.config.EnableCircuitBreaker {
		o.mu.RLock()
		cb, exists := o.circuitBreakers[provider.Name()]
		o.mu.RUnlock()

		if exists && cb.State() == CircuitStateOpen {
			attempt.Skipped = true
			attempt.SkipReason = "circuit breaker open"
			attempt.Duration = time.Since(startTime)
			slog.Debug("Skipping provider due to circuit breaker",
				"provider", provider.Name(),
			)
			return attempt
		}
	}

	// Execute search
	result, err := o.executeSearchWithCircuitBreaker(ctx, provider, req)
	attempt.Duration = time.Since(startTime)

	if err != nil {
		attempt.Success = false
		attempt.Error = err
		// Check if this was a context cancellation
		if ctx.Err() != nil {
			attempt.Error = ctx.Err()
		}
		slog.Debug("Provider search failed",
			"provider", provider.Name(),
			"error", err,
			"duration", attempt.Duration,
		)
		return attempt
	}

	// Check if we got results
	if result == nil || !result.HasResults() {
		attempt.Success = false
		slog.Debug("Provider returned no results",
			"provider", provider.Name(),
			"duration", attempt.Duration,
		)
		return attempt
	}

	attempt.Success = true
	return attempt
}

// executeSearch executes a search on a provider
func (o *Orchestrator) executeSearch(ctx context.Context, provider MetadataProvider, req *SearchRequest) (*SearchResult, error) {
	return provider.Search(ctx, req)
}

// executeSearchWithCircuitBreaker executes a search with circuit breaker protection
func (o *Orchestrator) executeSearchWithCircuitBreaker(ctx context.Context, provider MetadataProvider, req *SearchRequest) (*SearchResult, error) {
	if !o.config.EnableCircuitBreaker {
		return o.executeSearch(ctx, provider, req)
	}

	o.mu.RLock()
	cb, exists := o.circuitBreakers[provider.Name()]
	o.mu.RUnlock()

	if !exists {
		return o.executeSearch(ctx, provider, req)
	}

	var result *SearchResult
	var searchErr error

	err := cb.Execute(func() error {
		r, e := provider.Search(ctx, req)
		result = r
		searchErr = e
		return e
	})

	if err == ErrCircuitOpen {
		return nil, err
	}

	return result, searchErr
}

// GetCircuitBreakerState returns the circuit breaker state for a provider
func (o *Orchestrator) GetCircuitBreakerState(providerName string) (CircuitState, bool) {
	o.mu.RLock()
	defer o.mu.RUnlock()

	cb, exists := o.circuitBreakers[providerName]
	if !exists {
		return CircuitStateClosed, false
	}
	return cb.State(), true
}

// ResetCircuitBreaker resets the circuit breaker for a provider
func (o *Orchestrator) ResetCircuitBreaker(providerName string) {
	o.mu.RLock()
	cb, exists := o.circuitBreakers[providerName]
	o.mu.RUnlock()

	if exists {
		cb.Reset()
	}
}

// SearchSource searches a specific metadata source directly (Story 3.7)
// Unlike Search which uses the fallback chain, this method:
// - Searches only the specified source
// - Does not fall back to other sources
// - Returns nil if source not found or unavailable
func (o *Orchestrator) SearchSource(ctx context.Context, req *SearchRequest, source models.MetadataSource) (*SearchResult, error) {
	o.mu.RLock()
	providers := make([]MetadataProvider, len(o.providers))
	copy(providers, o.providers)
	o.mu.RUnlock()

	// Find the provider for the requested source
	var targetProvider MetadataProvider
	for _, p := range providers {
		if p.Source() == source {
			targetProvider = p
			break
		}
	}

	if targetProvider == nil {
		slog.Debug("Source provider not registered",
			"source", source,
		)
		return nil, nil
	}

	if !targetProvider.IsAvailable() {
		slog.Debug("Source provider unavailable",
			"source", source,
		)
		return nil, nil
	}

	// Check circuit breaker
	if o.config.EnableCircuitBreaker {
		o.mu.RLock()
		cb, exists := o.circuitBreakers[targetProvider.Name()]
		o.mu.RUnlock()

		if exists && cb.State() == CircuitStateOpen {
			slog.Debug("Source provider circuit breaker open",
				"source", source,
			)
			return nil, ErrCircuitOpen
		}
	}

	result, err := o.executeSearchWithCircuitBreaker(ctx, targetProvider, req)
	if err != nil {
		slog.Debug("Source search failed",
			"source", source,
			"error", err,
		)
		return nil, err
	}

	return result, nil
}
