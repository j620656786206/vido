package services

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/vido/api/internal/metadata"
	"github.com/vido/api/internal/models"
)

// MetadataServiceConfig holds configuration for the metadata service
type MetadataServiceConfig struct {
	// TMDbImageBaseURL is the base URL for TMDb images
	TMDbImageBaseURL string
	// EnableDouban enables the Douban provider
	EnableDouban bool
	// EnableWikipedia enables the Wikipedia provider
	EnableWikipedia bool
	// EnableCircuitBreaker enables circuit breakers for providers
	EnableCircuitBreaker bool
	// FallbackDelayMs is the delay between provider attempts in milliseconds
	FallbackDelayMs int
	// CircuitBreakerFailureThreshold is the number of failures before circuit opens
	CircuitBreakerFailureThreshold int
	// CircuitBreakerTimeoutSeconds is the timeout before circuit enters half-open
	CircuitBreakerTimeoutSeconds int
}

// ProviderInfo contains information about a metadata provider
type ProviderInfo struct {
	Name      string                  `json:"name"`
	Source    models.MetadataSource   `json:"source"`
	Available bool                    `json:"available"`
	Status    metadata.ProviderStatus `json:"status"`
}

// SearchMetadataRequest represents a request to search for metadata
type SearchMetadataRequest struct {
	Query     string `json:"query"`
	MediaType string `json:"mediaType"` // "movie" or "tv"
	Year      int    `json:"year,omitempty"`
	Page      int    `json:"page,omitempty"`
	Language  string `json:"language,omitempty"`
}

// ToMetadataRequest converts to the internal metadata.SearchRequest
func (r *SearchMetadataRequest) ToMetadataRequest() *metadata.SearchRequest {
	mediaType := metadata.MediaTypeMovie
	if r.MediaType == "tv" {
		mediaType = metadata.MediaTypeTV
	}

	return &metadata.SearchRequest{
		Query:     r.Query,
		MediaType: mediaType,
		Year:      r.Year,
		Page:      r.Page,
		Language:  r.Language,
	}
}

// Validate validates the search request
func (r *SearchMetadataRequest) Validate() error {
	if r.Query == "" {
		return errors.New("query is required")
	}
	return nil
}

// MetadataServiceInterface defines the contract for metadata operations
type MetadataServiceInterface interface {
	// SearchMetadata searches for metadata using the fallback chain
	SearchMetadata(ctx context.Context, req *SearchMetadataRequest) (*metadata.SearchResult, *metadata.FallbackStatus, error)
	// GetProviders returns information about registered providers
	GetProviders() []ProviderInfo
}

// MetadataService implements MetadataServiceInterface
type MetadataService struct {
	orchestrator *metadata.Orchestrator
	tmdbProvider *metadata.TMDbProvider
}

// Compile-time interface verification
var _ MetadataServiceInterface = (*MetadataService)(nil)

// NewMetadataService creates a new metadata service with configured providers
func NewMetadataService(cfg MetadataServiceConfig, tmdbSearcher metadata.TMDbSearcher) *MetadataService {
	// Build orchestrator config
	fallbackDelay := time.Duration(cfg.FallbackDelayMs) * time.Millisecond
	if fallbackDelay <= 0 {
		fallbackDelay = 100 * time.Millisecond
	}

	orchConfig := metadata.OrchestratorConfig{
		FallbackDelay:        fallbackDelay,
		EnableCircuitBreaker: cfg.EnableCircuitBreaker,
	}

	if cfg.EnableCircuitBreaker {
		failureThreshold := cfg.CircuitBreakerFailureThreshold
		if failureThreshold <= 0 {
			failureThreshold = 5
		}
		timeoutSeconds := cfg.CircuitBreakerTimeoutSeconds
		if timeoutSeconds <= 0 {
			timeoutSeconds = 30
		}

		orchConfig.CircuitBreakerConfig = metadata.CircuitBreakerConfig{
			FailureThreshold: failureThreshold,
			SuccessThreshold: 2,
			Timeout:          time.Duration(timeoutSeconds) * time.Second,
		}
	}

	orch := metadata.NewOrchestrator(orchConfig)

	// Create and register TMDb provider
	tmdbConfig := metadata.TMDbProviderConfig{
		ImageBaseURL: cfg.TMDbImageBaseURL,
	}
	if tmdbConfig.ImageBaseURL == "" {
		tmdbConfig.ImageBaseURL = "https://image.tmdb.org/t/p/w500"
	}

	tmdbProvider := metadata.NewTMDbProvider(tmdbSearcher, tmdbConfig)
	orch.RegisterProvider(tmdbProvider)

	// Register Douban provider if enabled
	if cfg.EnableDouban {
		doubanProvider := metadata.NewDoubanProvider(metadata.DoubanProviderConfig{
			Enabled: true,
		})
		orch.RegisterProvider(doubanProvider)
	}

	// Register Wikipedia provider if enabled
	if cfg.EnableWikipedia {
		wikipediaProvider := metadata.NewWikipediaProvider(metadata.WikipediaProviderConfig{
			Enabled: true,
		})
		orch.RegisterProvider(wikipediaProvider)
	}

	slog.Info("Metadata service initialized",
		"tmdb_enabled", true,
		"douban_enabled", cfg.EnableDouban,
		"wikipedia_enabled", cfg.EnableWikipedia,
		"circuit_breaker_enabled", cfg.EnableCircuitBreaker,
		"fallback_delay_ms", fallbackDelay.Milliseconds(),
	)

	return &MetadataService{
		orchestrator: orch,
		tmdbProvider: tmdbProvider,
	}
}

// SearchMetadata searches for metadata using the fallback chain
func (s *MetadataService) SearchMetadata(ctx context.Context, req *SearchMetadataRequest) (*metadata.SearchResult, *metadata.FallbackStatus, error) {
	if err := req.Validate(); err != nil {
		return nil, nil, err
	}

	metaReq := req.ToMetadataRequest()
	result, status := s.orchestrator.Search(ctx, metaReq)

	return result, status, nil
}

// GetProviders returns information about registered providers
func (s *MetadataService) GetProviders() []ProviderInfo {
	providers := s.orchestrator.Providers()
	infos := make([]ProviderInfo, len(providers))

	for i, p := range providers {
		infos[i] = ProviderInfo{
			Name:      p.Name(),
			Source:    p.Source(),
			Available: p.IsAvailable(),
			Status:    p.Status(),
		}
	}

	return infos
}

// SetKeywordGenerator sets the AI keyword generator for retry phase (Story 3.6)
func (s *MetadataService) SetKeywordGenerator(generator metadata.KeywordGenerator) {
	s.orchestrator.SetKeywordGenerator(generator)
	slog.Info("AI keyword generator configured for metadata search")
}
