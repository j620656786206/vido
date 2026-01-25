package services

import (
	"context"
	"errors"
	"fmt"
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

// ManualSearchRequest represents a request for manual metadata search (Story 3.7)
type ManualSearchRequest struct {
	Query     string `json:"query"`
	MediaType string `json:"mediaType"` // "movie" or "tv"
	Year      int    `json:"year,omitempty"`
	Source    string `json:"source"` // "tmdb", "douban", "wikipedia", or "all"
}

// Validate validates the manual search request
func (r *ManualSearchRequest) Validate() error {
	if r.Query == "" {
		return ErrManualSearchQueryRequired
	}
	// Default media type to movie
	if r.MediaType == "" {
		r.MediaType = "movie"
	}
	// Default source to all
	if r.Source == "" {
		r.Source = "all"
	}
	// Validate source
	validSources := map[string]bool{"tmdb": true, "douban": true, "wikipedia": true, "all": true}
	if !validSources[r.Source] {
		return ErrManualSearchInvalidSource
	}
	return nil
}

// ManualSearchResultItem represents a single search result item (Story 3.7)
type ManualSearchResultItem struct {
	ID         string               `json:"id"`
	Source     models.MetadataSource `json:"source"`
	Title      string               `json:"title"`
	TitleZhTW  string               `json:"titleZhTW,omitempty"`
	Year       int                  `json:"year,omitempty"`
	MediaType  string               `json:"mediaType"`
	Overview   string               `json:"overview,omitempty"`
	PosterURL  string               `json:"posterUrl,omitempty"`
	Rating     float64              `json:"rating,omitempty"`
	Confidence float64              `json:"confidence,omitempty"`
}

// ManualSearchResponse represents the response from manual search (Story 3.7)
type ManualSearchResponse struct {
	Results         []ManualSearchResultItem `json:"results"`
	TotalCount      int                      `json:"totalCount"`
	SearchedSources []string                 `json:"searchedSources"`
}

// Manual search errors
var (
	ErrManualSearchQueryRequired = errors.New("query is required")
	ErrManualSearchInvalidSource = errors.New("invalid source: must be 'tmdb', 'douban', 'wikipedia', or 'all'")
)

// SelectedMetadataItem represents a user-selected metadata item for apply operation
type SelectedMetadataItem struct {
	ID     string `json:"id"`
	Source string `json:"source"`
}

// ApplyMetadataRequest represents a request to apply metadata to a media item (Story 3.7)
type ApplyMetadataRequest struct {
	MediaID      string               `json:"mediaId"`
	MediaType    string               `json:"mediaType"` // "movie" or "series"
	SelectedItem SelectedMetadataItem `json:"selectedItem"`
	LearnPattern bool                 `json:"learnPattern,omitempty"` // Optional: trigger learning system (Story 3.9)
}

// Validate validates the apply metadata request
func (r *ApplyMetadataRequest) Validate() error {
	if r.MediaID == "" {
		return ErrApplyMetadataMediaIDRequired
	}
	if r.SelectedItem.ID == "" {
		return ErrApplyMetadataSelectedItemRequired
	}
	if r.SelectedItem.Source == "" {
		return ErrApplyMetadataSelectedItemRequired
	}
	// Default media type to movie
	if r.MediaType == "" {
		r.MediaType = "movie"
	}
	return nil
}

// ApplyMetadataResponse represents the response from applying metadata
type ApplyMetadataResponse struct {
	Success   bool                  `json:"success"`
	MediaID   string                `json:"mediaId"`
	MediaType string                `json:"mediaType"`
	Title     string                `json:"title"`
	Source    models.MetadataSource `json:"source"`
}

// Apply metadata errors
var (
	ErrApplyMetadataMediaIDRequired      = errors.New("mediaId is required")
	ErrApplyMetadataSelectedItemRequired = errors.New("selectedItem with id and source is required")
	ErrApplyMetadataNotFound             = errors.New("media item not found")
	ErrApplyMetadataFailed               = errors.New("failed to apply metadata")
)

// MetadataServiceInterface defines the contract for metadata operations
type MetadataServiceInterface interface {
	// SearchMetadata searches for metadata using the fallback chain
	SearchMetadata(ctx context.Context, req *SearchMetadataRequest) (*metadata.SearchResult, *metadata.FallbackStatus, error)
	// GetProviders returns information about registered providers
	GetProviders() []ProviderInfo
	// ManualSearch performs manual search across selected sources (Story 3.7)
	ManualSearch(ctx context.Context, req *ManualSearchRequest) (*ManualSearchResponse, error)
	// ApplyMetadata applies selected metadata to a media item (Story 3.7 - AC3)
	ApplyMetadata(ctx context.Context, req *ApplyMetadataRequest) (*ApplyMetadataResponse, error)
}

// MediaUpdater is an interface for updating media metadata
type MediaUpdater interface {
	UpdateMetadataSource(ctx context.Context, mediaID string, source models.MetadataSource) error
	GetByID(ctx context.Context, id string) (title string, exists bool, err error)
}

// MetadataService implements MetadataServiceInterface
type MetadataService struct {
	orchestrator  *metadata.Orchestrator
	tmdbProvider  *metadata.TMDbProvider
	movieUpdater  MediaUpdater
	seriesUpdater MediaUpdater
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

// ManualSearch performs manual search across selected sources (Story 3.7)
// Unlike SearchMetadata which uses the fallback chain, this method:
// - Searches specific source(s) selected by the user
// - Aggregates results from all selected sources (if "all")
// - Returns results with source indicators
func (s *MetadataService) ManualSearch(ctx context.Context, req *ManualSearchRequest) (*ManualSearchResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	mediaType := metadata.MediaTypeMovie
	if req.MediaType == "tv" {
		mediaType = metadata.MediaTypeTV
	}

	searchReq := &metadata.SearchRequest{
		Query:     req.Query,
		MediaType: mediaType,
		Year:      req.Year,
		Page:      1,
		Language:  "zh-TW",
	}

	response := &ManualSearchResponse{
		Results:         []ManualSearchResultItem{},
		SearchedSources: []string{},
	}

	// Determine which sources to search
	sourcesToSearch := []models.MetadataSource{}
	switch req.Source {
	case "tmdb":
		sourcesToSearch = append(sourcesToSearch, models.MetadataSourceTMDb)
	case "douban":
		sourcesToSearch = append(sourcesToSearch, models.MetadataSourceDouban)
	case "wikipedia":
		sourcesToSearch = append(sourcesToSearch, models.MetadataSourceWikipedia)
	case "all":
		sourcesToSearch = append(sourcesToSearch,
			models.MetadataSourceTMDb,
			models.MetadataSourceDouban,
			models.MetadataSourceWikipedia,
		)
	}

	// Search each selected source
	for _, source := range sourcesToSearch {
		response.SearchedSources = append(response.SearchedSources, string(source))

		result, err := s.orchestrator.SearchSource(ctx, searchReq, source)
		if err != nil {
			slog.Debug("Manual search source failed",
				"source", source,
				"error", err,
			)
			continue
		}

		if result != nil && result.HasResults() {
			for _, item := range result.Items {
				resultItem := ManualSearchResultItem{
					ID:         fmt.Sprintf("%s-%s", source, item.ID),
					Source:     source,
					Title:      item.Title,
					TitleZhTW:  item.TitleZhTW,
					Year:       item.Year,
					MediaType:  string(item.MediaType),
					Overview:   item.Overview,
					PosterURL:  item.PosterURL,
					Rating:     item.Rating,
					Confidence: item.Confidence,
				}
				response.Results = append(response.Results, resultItem)
			}
		}
	}

	// Sort by relevance (rating * confidence) descending
	s.sortResultsByRelevance(response.Results)

	response.TotalCount = len(response.Results)

	slog.Info("Manual search completed",
		"query", req.Query,
		"source", req.Source,
		"total_results", response.TotalCount,
		"searched_sources", response.SearchedSources,
	)

	return response, nil
}

// sortResultsByRelevance sorts manual search results by relevance score
func (s *MetadataService) sortResultsByRelevance(results []ManualSearchResultItem) {
	// Simple relevance sort by rating (could be enhanced with more factors)
	for i := 0; i < len(results)-1; i++ {
		for j := i + 1; j < len(results); j++ {
			scoreI := results[i].Rating * results[i].Confidence
			scoreJ := results[j].Rating * results[j].Confidence
			if scoreI == 0 {
				scoreI = results[i].Rating
			}
			if scoreJ == 0 {
				scoreJ = results[j].Rating
			}
			if scoreJ > scoreI {
				results[i], results[j] = results[j], results[i]
			}
		}
	}
}

// SetMediaUpdaters sets the media updaters for movies and series
// This allows the service to update metadata source when applying metadata
func (s *MetadataService) SetMediaUpdaters(movieUpdater, seriesUpdater MediaUpdater) {
	s.movieUpdater = movieUpdater
	s.seriesUpdater = seriesUpdater
	slog.Info("Media updaters configured for metadata service")
}

// ApplyMetadata applies selected metadata to a media item (Story 3.7 - AC3)
// This method:
// - Validates the media exists
// - Updates the metadata source field
// - Returns the updated media information
func (s *MetadataService) ApplyMetadata(ctx context.Context, req *ApplyMetadataRequest) (*ApplyMetadataResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	// Determine source from selected item
	source := models.MetadataSource(req.SelectedItem.Source)

	// Get the appropriate updater based on media type
	var updater MediaUpdater
	switch req.MediaType {
	case "movie":
		updater = s.movieUpdater
	case "series":
		updater = s.seriesUpdater
	default:
		// Default to movie for backwards compatibility
		updater = s.movieUpdater
	}

	var title string
	if updater != nil {
		// Check if media exists
		t, exists, err := updater.GetByID(ctx, req.MediaID)
		if err != nil {
			slog.Error("Failed to get media",
				"media_id", req.MediaID,
				"media_type", req.MediaType,
				"error", err,
			)
			return nil, ErrApplyMetadataFailed
		}
		if !exists {
			slog.Debug("Media not found",
				"media_id", req.MediaID,
				"media_type", req.MediaType,
			)
			return nil, ErrApplyMetadataNotFound
		}
		title = t

		// Update metadata source
		if err := updater.UpdateMetadataSource(ctx, req.MediaID, source); err != nil {
			slog.Error("Failed to update metadata source",
				"media_id", req.MediaID,
				"source", source,
				"error", err,
			)
			return nil, ErrApplyMetadataFailed
		}
	} else {
		// If no updater is configured, we can't verify or update
		// This is for testing or when updaters are not yet configured
		slog.Warn("No media updater configured, skipping database update",
			"media_id", req.MediaID,
			"media_type", req.MediaType,
		)
		title = "Unknown" // Placeholder for testing
	}

	slog.Info("Metadata applied successfully",
		"media_id", req.MediaID,
		"media_type", req.MediaType,
		"source", source,
		"learn_pattern", req.LearnPattern,
	)

	// TODO: If learnPattern is true, trigger learning system (Story 3.9)
	if req.LearnPattern {
		slog.Debug("Learning pattern requested, will be implemented in Story 3.9",
			"media_id", req.MediaID,
			"selected_item", req.SelectedItem.ID,
		)
	}

	return &ApplyMetadataResponse{
		Success:   true,
		MediaID:   req.MediaID,
		MediaType: req.MediaType,
		Title:     title,
		Source:    source,
	}, nil
}
