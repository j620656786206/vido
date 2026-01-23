package metadata

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"sync"
	"time"

	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/tmdb"
)

// TMDbSearcher defines the interface for TMDb search operations
// This interface is implemented by services.TMDbService
type TMDbSearcher interface {
	// SearchMovies searches for movies by query
	SearchMovies(ctx context.Context, query string, page int) (*tmdb.SearchResultMovies, error)
	// SearchTVShows searches for TV shows by query
	SearchTVShows(ctx context.Context, query string, page int) (*tmdb.SearchResultTVShows, error)
}

// TMDbProviderConfig holds configuration for the TMDb provider
type TMDbProviderConfig struct {
	// ImageBaseURL is the base URL for TMDb images (e.g., "https://image.tmdb.org/t/p/w500")
	ImageBaseURL string
}

// TMDbProvider wraps TMDbService to implement MetadataProvider
type TMDbProvider struct {
	service   TMDbSearcher
	config    TMDbProviderConfig
	mu        sync.RWMutex
	available bool
	status    ProviderStatus
}

// NewTMDbProvider creates a new TMDb provider
func NewTMDbProvider(service TMDbSearcher, config TMDbProviderConfig) *TMDbProvider {
	// Apply defaults
	if config.ImageBaseURL == "" {
		config.ImageBaseURL = "https://image.tmdb.org/t/p/w500"
	}

	return &TMDbProvider{
		service:   service,
		config:    config,
		available: true,
		status:    ProviderStatusAvailable,
	}
}

// Name returns the provider name
func (p *TMDbProvider) Name() string {
	return "TMDb"
}

// Source returns the metadata source
func (p *TMDbProvider) Source() models.MetadataSource {
	return models.MetadataSourceTMDb
}

// IsAvailable returns whether the provider is available
func (p *TMDbProvider) IsAvailable() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.available
}

// Status returns the current provider status
func (p *TMDbProvider) Status() ProviderStatus {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.status
}

// SetAvailable sets the availability status
func (p *TMDbProvider) SetAvailable(available bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.available = available
	if available {
		p.status = ProviderStatusAvailable
	} else {
		p.status = ProviderStatusUnavailable
	}
}

// Search performs a metadata search
func (p *TMDbProvider) Search(ctx context.Context, req *SearchRequest) (*SearchResult, error) {
	if err := req.Validate(); err != nil {
		return nil, NewProviderError(p.Name(), p.Source(), ErrCodeInvalidRequest, err.Error(), err)
	}

	startTime := time.Now()

	var result *SearchResult
	var err error

	switch req.MediaType {
	case MediaTypeMovie:
		result, err = p.searchMovies(ctx, req)
	case MediaTypeTV:
		result, err = p.searchTVShows(ctx, req)
	default:
		// Default to movie search
		result, err = p.searchMovies(ctx, req)
	}

	if err != nil {
		slog.Error("TMDb search failed",
			"query", req.Query,
			"media_type", req.MediaType,
			"error", err,
			"duration", time.Since(startTime),
		)
		return nil, err
	}

	slog.Debug("TMDb search completed",
		"query", req.Query,
		"media_type", req.MediaType,
		"results", result.TotalCount,
		"duration", time.Since(startTime),
	)

	return result, nil
}

// searchMovies searches for movies
func (p *TMDbProvider) searchMovies(ctx context.Context, req *SearchRequest) (*SearchResult, error) {
	page := req.Page
	if page < 1 {
		page = 1
	}

	tmdbResult, err := p.service.SearchMovies(ctx, req.Query, page)
	if err != nil {
		return nil, NewProviderError(p.Name(), p.Source(), ErrCodeUnavailable, "failed to search movies", err)
	}

	return p.convertMovieResults(tmdbResult), nil
}

// searchTVShows searches for TV shows
func (p *TMDbProvider) searchTVShows(ctx context.Context, req *SearchRequest) (*SearchResult, error) {
	page := req.Page
	if page < 1 {
		page = 1
	}

	tmdbResult, err := p.service.SearchTVShows(ctx, req.Query, page)
	if err != nil {
		return nil, NewProviderError(p.Name(), p.Source(), ErrCodeUnavailable, "failed to search TV shows", err)
	}

	return p.convertTVShowResults(tmdbResult), nil
}

// convertMovieResults converts TMDb movie results to common format
func (p *TMDbProvider) convertMovieResults(tmdbResult *tmdb.SearchResultMovies) *SearchResult {
	items := make([]MetadataItem, 0, len(tmdbResult.Results))

	for _, movie := range tmdbResult.Results {
		item := MetadataItem{
			ID:            strconv.Itoa(movie.ID),
			Title:         movie.Title,
			OriginalTitle: movie.OriginalTitle,
			Year:          extractYear(movie.ReleaseDate),
			ReleaseDate:   movie.ReleaseDate,
			Overview:      movie.Overview,
			PosterURL:     buildImageURL(p.config.ImageBaseURL, movie.PosterPath),
			BackdropURL:   buildImageURL(p.config.ImageBaseURL, movie.BackdropPath),
			MediaType:     MediaTypeMovie,
			Rating:        movie.VoteAverage,
			VoteCount:     movie.VoteCount,
			Popularity:    movie.Popularity,
			RawData:       movie,
		}

		items = append(items, item)
	}

	return &SearchResult{
		Items:      items,
		Source:     models.MetadataSourceTMDb,
		TotalCount: tmdbResult.TotalResults,
		Page:       tmdbResult.Page,
		TotalPages: tmdbResult.TotalPages,
	}
}

// convertTVShowResults converts TMDb TV show results to common format
func (p *TMDbProvider) convertTVShowResults(tmdbResult *tmdb.SearchResultTVShows) *SearchResult {
	items := make([]MetadataItem, 0, len(tmdbResult.Results))

	for _, show := range tmdbResult.Results {
		item := MetadataItem{
			ID:            strconv.Itoa(show.ID),
			Title:         show.Name,
			OriginalTitle: show.OriginalName,
			Year:          extractYear(show.FirstAirDate),
			ReleaseDate:   show.FirstAirDate,
			Overview:      show.Overview,
			PosterURL:     buildImageURL(p.config.ImageBaseURL, show.PosterPath),
			BackdropURL:   buildImageURL(p.config.ImageBaseURL, show.BackdropPath),
			MediaType:     MediaTypeTV,
			Rating:        show.VoteAverage,
			VoteCount:     show.VoteCount,
			Popularity:    show.Popularity,
			RawData:       show,
		}

		items = append(items, item)
	}

	return &SearchResult{
		Items:      items,
		Source:     models.MetadataSourceTMDb,
		TotalCount: tmdbResult.TotalResults,
		Page:       tmdbResult.Page,
		TotalPages: tmdbResult.TotalPages,
	}
}

// extractYear extracts the year from a date string in YYYY-MM-DD format
func extractYear(date string) int {
	if len(date) < 4 {
		return 0
	}

	// Check for proper YYYY-MM-DD format
	if len(date) >= 10 && date[4] == '-' {
		year, err := strconv.Atoi(date[:4])
		if err != nil {
			return 0
		}
		return year
	}

	return 0
}

// buildImageURL builds a full image URL from the base URL and path
func buildImageURL(baseURL string, path *string) string {
	if baseURL == "" || path == nil || *path == "" {
		return ""
	}
	return fmt.Sprintf("%s%s", baseURL, *path)
}

// Compile-time interface verification
var _ MetadataProvider = (*TMDbProvider)(nil)
