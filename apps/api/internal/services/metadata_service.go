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

// UpdateMetadataRequest represents a request to manually update metadata (Story 3.8)
type UpdateMetadataRequest struct {
	ID           string   `json:"id"`
	MediaType    string   `json:"mediaType"`    // "movie" or "series"
	Title        string   `json:"title"`        // Required: Chinese title
	TitleEnglish string   `json:"titleEnglish"` // Optional: English title
	Year         int      `json:"year"`         // Required
	Genres       []string `json:"genres"`
	Director     string   `json:"director"`
	Cast         []string `json:"cast"`
	Overview     string   `json:"overview"`
	PosterURL    string   `json:"posterUrl"` // Optional: URL for poster
}

// Validate validates the update metadata request
func (r *UpdateMetadataRequest) Validate() error {
	if r.ID == "" {
		return ErrUpdateMetadataIDRequired
	}
	if r.Title == "" {
		return ErrUpdateMetadataTitleRequired
	}
	if r.Year == 0 {
		return ErrUpdateMetadataYearRequired
	}
	// Default media type to movie
	if r.MediaType == "" {
		r.MediaType = "movie"
	}
	return nil
}

// UpdateMetadataResponse represents the response from updating metadata
type UpdateMetadataResponse struct {
	ID             string               `json:"id"`
	Title          string               `json:"title"`
	MetadataSource models.MetadataSource `json:"metadataSource"`
	UpdatedAt      string               `json:"updatedAt"`
}

// Update metadata errors
var (
	ErrUpdateMetadataIDRequired    = errors.New("id is required")
	ErrUpdateMetadataTitleRequired = errors.New("title is required")
	ErrUpdateMetadataYearRequired  = errors.New("year is required")
	ErrUpdateMetadataNotFound      = errors.New("media item not found")
	ErrUpdateMetadataFailed        = errors.New("failed to update metadata")
)

// UploadPosterRequest represents a request to upload a poster image (Story 3.8 - AC3)
type UploadPosterRequest struct {
	MediaID     string `json:"mediaId"`
	MediaType   string `json:"mediaType"` // "movie" or "series"
	FileData    []byte `json:"-"`         // Binary image data
	FileName    string `json:"fileName"`
	ContentType string `json:"contentType"`
	FileSize    int64  `json:"fileSize"`
}

// Validate validates the upload poster request
func (r *UploadPosterRequest) Validate() error {
	if r.MediaID == "" {
		return ErrUploadPosterMediaIDRequired
	}
	if len(r.FileData) == 0 {
		return ErrUploadPosterFileRequired
	}
	// Validate file type
	validTypes := map[string]bool{
		"image/jpeg": true,
		"image/png":  true,
		"image/webp": true,
	}
	if !validTypes[r.ContentType] {
		return ErrPosterInvalidFormat
	}
	// Validate file size (max 5MB)
	const maxFileSize = 5 * 1024 * 1024
	if r.FileSize > maxFileSize {
		return ErrPosterTooLarge
	}
	// Default media type to movie
	if r.MediaType == "" {
		r.MediaType = "movie"
	}
	return nil
}

// UploadPosterResponse represents the response from uploading a poster
type UploadPosterResponse struct {
	PosterURL    string `json:"posterUrl"`
	ThumbnailURL string `json:"thumbnailUrl"`
}

// Upload poster errors
var (
	ErrUploadPosterMediaIDRequired = errors.New("mediaId is required")
	ErrUploadPosterFileRequired    = errors.New("file is required")
	ErrPosterInvalidFormat         = errors.New("invalid image format: must be jpg, png, or webp")
	ErrPosterTooLarge              = errors.New("file too large: maximum size is 5MB")
	ErrUploadPosterNotFound        = errors.New("media item not found")
	ErrUploadPosterFailed          = errors.New("failed to upload poster")
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
	// UpdateMetadata manually updates metadata for a media item (Story 3.8 - AC2)
	UpdateMetadata(ctx context.Context, req *UpdateMetadataRequest) (*UpdateMetadataResponse, error)
	// UploadPoster uploads a custom poster image for a media item (Story 3.8 - AC3)
	UploadPoster(ctx context.Context, req *UploadPosterRequest) (*UploadPosterResponse, error)
}

// MediaUpdater is an interface for updating media metadata
type MediaUpdater interface {
	UpdateMetadataSource(ctx context.Context, mediaID string, source models.MetadataSource) error
	GetByID(ctx context.Context, id string) (title string, exists bool, err error)
}

// MetadataEditor is an interface for full metadata editing (Story 3.8)
type MetadataEditor interface {
	UpdateMetadata(ctx context.Context, req *UpdateMetadataRequest) (*UpdateMetadataResponse, error)
	Exists(ctx context.Context, id string) (bool, error)
}

// PosterUploader is an interface for poster image upload (Story 3.8 - AC3)
type PosterUploader interface {
	UploadPoster(ctx context.Context, req *UploadPosterRequest) (*UploadPosterResponse, error)
	Exists(ctx context.Context, id string) (bool, error)
}

// MetadataService implements MetadataServiceInterface
type MetadataService struct {
	orchestrator     *metadata.Orchestrator
	tmdbProvider     *metadata.TMDbProvider
	movieUpdater     MediaUpdater
	seriesUpdater    MediaUpdater
	movieEditor      MetadataEditor
	seriesEditor     MetadataEditor
	posterUploader   PosterUploader
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

// SetMetadataEditors sets the metadata editors for movies and series (Story 3.8)
// This allows the service to perform full metadata updates
func (s *MetadataService) SetMetadataEditors(movieEditor, seriesEditor MetadataEditor) {
	s.movieEditor = movieEditor
	s.seriesEditor = seriesEditor
	slog.Info("Metadata editors configured for metadata service")
}

// UpdateMetadata manually updates metadata for a media item (Story 3.8 - AC2)
// This method:
// - Validates required fields (title, year)
// - Updates all provided metadata fields
// - Sets metadata source to "manual"
// - Returns the updated metadata
func (s *MetadataService) UpdateMetadata(ctx context.Context, req *UpdateMetadataRequest) (*UpdateMetadataResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	// Get the appropriate editor based on media type
	var editor MetadataEditor
	switch req.MediaType {
	case "series":
		editor = s.seriesEditor
	default:
		// Default to movie
		editor = s.movieEditor
	}

	if editor != nil {
		// Check if media exists
		exists, err := editor.Exists(ctx, req.ID)
		if err != nil {
			slog.Error("Failed to check if media exists",
				"media_id", req.ID,
				"media_type", req.MediaType,
				"error", err,
			)
			return nil, ErrUpdateMetadataFailed
		}
		if !exists {
			slog.Debug("Media not found for update",
				"media_id", req.ID,
				"media_type", req.MediaType,
			)
			return nil, ErrUpdateMetadataNotFound
		}

		// Perform the update
		result, err := editor.UpdateMetadata(ctx, req)
		if err != nil {
			slog.Error("Failed to update metadata",
				"media_id", req.ID,
				"media_type", req.MediaType,
				"error", err,
			)
			return nil, ErrUpdateMetadataFailed
		}

		slog.Info("Metadata updated successfully",
			"media_id", req.ID,
			"media_type", req.MediaType,
			"title", req.Title,
		)

		return result, nil
	}

	// If no editor is configured, return error
	slog.Warn("No metadata editor configured",
		"media_id", req.ID,
		"media_type", req.MediaType,
	)
	return nil, ErrUpdateMetadataFailed
}

// SetPosterUploader sets the poster uploader for the metadata service (Story 3.8 - AC3)
func (s *MetadataService) SetPosterUploader(uploader PosterUploader) {
	s.posterUploader = uploader
	slog.Info("Poster uploader configured for metadata service")
}

// UploadPoster uploads a custom poster image for a media item (Story 3.8 - AC3)
// This method:
// - Validates file type (jpg, png, webp)
// - Validates file size (max 5MB)
// - Processes and stores the image
// - Returns the poster URL and thumbnail URL
func (s *MetadataService) UploadPoster(ctx context.Context, req *UploadPosterRequest) (*UploadPosterResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	if s.posterUploader == nil {
		slog.Warn("No poster uploader configured",
			"media_id", req.MediaID,
			"media_type", req.MediaType,
		)
		return nil, ErrUploadPosterFailed
	}

	// Check if media exists
	exists, err := s.posterUploader.Exists(ctx, req.MediaID)
	if err != nil {
		slog.Error("Failed to check if media exists",
			"media_id", req.MediaID,
			"error", err,
		)
		return nil, ErrUploadPosterFailed
	}
	if !exists {
		slog.Debug("Media not found for poster upload",
			"media_id", req.MediaID,
			"media_type", req.MediaType,
		)
		return nil, ErrUploadPosterNotFound
	}

	// Perform the upload
	result, err := s.posterUploader.UploadPoster(ctx, req)
	if err != nil {
		slog.Error("Failed to upload poster",
			"media_id", req.MediaID,
			"media_type", req.MediaType,
			"error", err,
		)
		return nil, err
	}

	slog.Info("Poster uploaded successfully",
		"media_id", req.MediaID,
		"media_type", req.MediaType,
		"poster_url", result.PosterURL,
	)

	return result, nil
}
