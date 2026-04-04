package services

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/vido/api/internal/repository"
	"github.com/vido/api/internal/tmdb"
)

// TMDbConfig holds configuration for the TMDb service
type TMDbConfig struct {
	APIKey            string
	DefaultLanguage   string
	FallbackLanguages []string
	CacheTTLHours     int
}

// TMDbServiceInterface defines the contract for TMDb operations
type TMDbServiceInterface interface {
	// SearchMovies searches for movies by query
	SearchMovies(ctx context.Context, query string, page int) (*tmdb.SearchResultMovies, error)
	// SearchTVShows searches for TV shows by query
	SearchTVShows(ctx context.Context, query string, page int) (*tmdb.SearchResultTVShows, error)
	// GetMovieDetails retrieves movie details by ID
	GetMovieDetails(ctx context.Context, movieID int) (*tmdb.MovieDetails, error)
	// GetTVShowDetails retrieves TV show details by ID
	GetTVShowDetails(ctx context.Context, tvID int) (*tmdb.TVShowDetails, error)
	// FindByExternalID finds movies/TV shows by an external ID (e.g., IMDB)
	FindByExternalID(ctx context.Context, externalID string, externalSource string) (*tmdb.FindByExternalIDResponse, error)
}

// TMDbService implements TMDbServiceInterface
type TMDbService struct {
	cacheService tmdb.CacheServiceInterface
	client       tmdb.ClientInterface
}

// Compile-time interface verification
var _ TMDbServiceInterface = (*TMDbService)(nil)

// NewTMDbService creates a new TMDb service with all layers wired together
// Architecture: TMDbService → CacheService → LanguageFallbackClient → Client → TMDb API
func NewTMDbService(cfg TMDbConfig, cacheRepo repository.CacheRepositoryInterface) *TMDbService {
	// Build the client layer
	client := tmdb.NewClient(tmdb.ClientConfig{
		APIKey:   cfg.APIKey,
		Language: cfg.DefaultLanguage,
	})

	// Build the language fallback layer
	languages := cfg.FallbackLanguages
	if len(languages) == 0 {
		// Default fallback chain: zh-TW → zh-CN → en
		languages = tmdb.DefaultFallbackLanguages
	}
	fallbackClient := tmdb.NewLanguageFallbackClient(client, languages)

	// Build the cache layer
	ttl := time.Duration(cfg.CacheTTLHours) * time.Hour
	if ttl == 0 {
		ttl = tmdb.DefaultCacheTTL
	}
	cacheService := tmdb.NewCacheService(fallbackClient, cacheRepo, tmdb.CacheServiceConfig{
		TTL: ttl,
	})

	slog.Info("TMDb service initialized",
		"default_language", cfg.DefaultLanguage,
		"fallback_languages", strings.Join(languages, ","),
		"cache_ttl_hours", int(ttl.Hours()),
	)

	return &TMDbService{
		cacheService: cacheService,
		client:       client,
	}
}

// VideosProvider returns the TMDb client as a TMDbVideosProvider for on-demand video fetching
func (s *TMDbService) VideosProvider() TMDbVideosProvider {
	return s.client
}

// NewTMDbServiceWithCacheService creates a TMDb service with a custom cache service
// This is useful for testing with mock dependencies
func NewTMDbServiceWithCacheService(cacheService tmdb.CacheServiceInterface) *TMDbService {
	return &TMDbService{
		cacheService: cacheService,
	}
}

// SearchMovies searches for movies by query
func (s *TMDbService) SearchMovies(ctx context.Context, query string, page int) (*tmdb.SearchResultMovies, error) {
	if query == "" {
		return nil, tmdb.NewBadRequestError("search query cannot be empty")
	}

	if page < 1 {
		page = 1
	}

	slog.Debug("Searching movies",
		"query", query,
		"page", page,
	)

	result, err := s.cacheService.SearchMovies(ctx, query, page)
	if err != nil {
		slog.Error("Failed to search movies",
			"query", query,
			"page", page,
			"error", err,
		)
		return nil, err
	}

	slog.Debug("Movie search completed",
		"query", query,
		"results", result.TotalResults,
	)

	return result, nil
}

// SearchTVShows searches for TV shows by query
func (s *TMDbService) SearchTVShows(ctx context.Context, query string, page int) (*tmdb.SearchResultTVShows, error) {
	if query == "" {
		return nil, tmdb.NewBadRequestError("search query cannot be empty")
	}

	if page < 1 {
		page = 1
	}

	slog.Debug("Searching TV shows",
		"query", query,
		"page", page,
	)

	result, err := s.cacheService.SearchTVShows(ctx, query, page)
	if err != nil {
		slog.Error("Failed to search TV shows",
			"query", query,
			"page", page,
			"error", err,
		)
		return nil, err
	}

	slog.Debug("TV show search completed",
		"query", query,
		"results", result.TotalResults,
	)

	return result, nil
}

// GetMovieDetails retrieves movie details by ID
func (s *TMDbService) GetMovieDetails(ctx context.Context, movieID int) (*tmdb.MovieDetails, error) {
	if movieID <= 0 {
		return nil, tmdb.NewBadRequestError("movie ID must be greater than 0")
	}

	slog.Debug("Getting movie details",
		"movie_id", movieID,
	)

	result, err := s.cacheService.GetMovieDetails(ctx, movieID)
	if err != nil {
		slog.Error("Failed to get movie details",
			"movie_id", movieID,
			"error", err,
		)
		return nil, err
	}

	slog.Debug("Movie details retrieved",
		"movie_id", movieID,
		"title", result.Title,
	)

	return result, nil
}

// GetTVShowDetails retrieves TV show details by ID
func (s *TMDbService) GetTVShowDetails(ctx context.Context, tvID int) (*tmdb.TVShowDetails, error) {
	if tvID <= 0 {
		return nil, tmdb.NewBadRequestError("TV show ID must be greater than 0")
	}

	slog.Debug("Getting TV show details",
		"tv_id", tvID,
	)

	result, err := s.cacheService.GetTVShowDetails(ctx, tvID)
	if err != nil {
		slog.Error("Failed to get TV show details",
			"tv_id", tvID,
			"error", err,
		)
		return nil, err
	}

	slog.Debug("TV show details retrieved",
		"tv_id", tvID,
		"name", result.Name,
	)

	return result, nil
}

// FindByExternalID finds movies/TV shows by an external ID (e.g., IMDB).
// This bypasses the cache layer and calls the client directly since find results are not cacheable.
func (s *TMDbService) FindByExternalID(ctx context.Context, externalID string, externalSource string) (*tmdb.FindByExternalIDResponse, error) {
	if externalID == "" {
		return nil, tmdb.NewBadRequestError("external ID cannot be empty")
	}

	slog.Debug("Finding by external ID",
		"external_id", externalID,
		"source", externalSource,
	)

	result, err := s.client.FindByExternalID(ctx, externalID, externalSource)
	if err != nil {
		slog.Error("Failed to find by external ID",
			"external_id", externalID,
			"error", err,
		)
		return nil, err
	}

	return result, nil
}

// Ping checks if the TMDb API is accessible.
// Implements health.Pingable interface for health monitoring.
func (s *TMDbService) Ping(ctx context.Context) error {
	// Use a simple search query to verify API connectivity
	_, err := s.cacheService.SearchMovies(ctx, "test", 1)
	if err != nil {
		slog.Debug("TMDb ping failed", "error", err)
		return err
	}
	return nil
}
