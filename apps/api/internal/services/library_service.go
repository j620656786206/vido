package services

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/google/uuid"
	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/repository"
	"github.com/vido/api/internal/tmdb"
)

// SearchResult represents a unified search result item
type SearchResult struct {
	Type   string      `json:"type"` // "movie" or "series"
	Movie  *models.Movie  `json:"movie,omitempty"`
	Series *models.Series `json:"series,omitempty"`
}

// LibrarySearchResults contains unified search results across movies and series
type LibrarySearchResults struct {
	Results    []SearchResult                 `json:"results"`
	Movies     *repository.PaginationResult   `json:"moviesPagination"`
	Series     *repository.PaginationResult   `json:"seriesPagination"`
	TotalCount int                            `json:"totalCount"`
}

// LibraryServiceInterface defines the contract for media library operations
type LibraryServiceInterface interface {
	// SaveMovieFromTMDb saves a movie from TMDb search/details to the database
	SaveMovieFromTMDb(ctx context.Context, tmdbMovie *tmdb.MovieDetails, filePath string) (*models.Movie, error)

	// SaveSeriesFromTMDb saves a series from TMDb search/details to the database
	SaveSeriesFromTMDb(ctx context.Context, tmdbSeries *tmdb.TVShowDetails, filePath string) (*models.Series, error)

	// SearchLibrary performs unified FTS search across movies and series
	SearchLibrary(ctx context.Context, query string, params repository.ListParams) (*LibrarySearchResults, error)

	// GetMovieByID retrieves a movie by its ID
	GetMovieByID(ctx context.Context, id string) (*models.Movie, error)

	// GetSeriesByID retrieves a series by its ID
	GetSeriesByID(ctx context.Context, id string) (*models.Series, error)

	// GetMovieByTMDbID retrieves a movie by TMDb ID
	GetMovieByTMDbID(ctx context.Context, tmdbID int64) (*models.Movie, error)

	// GetSeriesByTMDbID retrieves a series by TMDb ID
	GetSeriesByTMDbID(ctx context.Context, tmdbID int64) (*models.Series, error)
}

// LibraryService handles media library storage and search operations
type LibraryService struct {
	movieRepo   repository.MovieRepositoryInterface
	seriesRepo  repository.SeriesRepositoryInterface
	episodeRepo repository.EpisodeRepositoryInterface
	logger      *slog.Logger
}

// NewLibraryService creates a new LibraryService
func NewLibraryService(
	movieRepo repository.MovieRepositoryInterface,
	seriesRepo repository.SeriesRepositoryInterface,
	episodeRepo repository.EpisodeRepositoryInterface,
) *LibraryService {
	return &LibraryService{
		movieRepo:   movieRepo,
		seriesRepo:  seriesRepo,
		episodeRepo: episodeRepo,
		logger:      slog.Default().With("service", "library"),
	}
}

// SaveMovieFromTMDb converts a TMDb movie to a model and saves it to the database
func (s *LibraryService) SaveMovieFromTMDb(ctx context.Context, tmdbMovie *tmdb.MovieDetails, filePath string) (*models.Movie, error) {
	if tmdbMovie == nil {
		return nil, fmt.Errorf("tmdb movie cannot be nil")
	}

	movie := ConvertTMDbMovieToModel(tmdbMovie, filePath)

	// Generate ID if not set
	if movie.ID == "" {
		movie.ID = uuid.New().String()
	}

	// Upsert to prevent duplicates (uses TMDb ID as unique identifier)
	if err := s.movieRepo.Upsert(ctx, movie); err != nil {
		s.logger.Error("Failed to save movie",
			"tmdbId", tmdbMovie.ID,
			"title", tmdbMovie.Title,
			"error", err,
		)
		return nil, fmt.Errorf("failed to save movie: %w", err)
	}

	s.logger.Info("Movie saved successfully",
		"id", movie.ID,
		"tmdbId", tmdbMovie.ID,
		"title", movie.Title,
	)

	return movie, nil
}

// SaveSeriesFromTMDb converts a TMDb series to a model and saves it to the database
func (s *LibraryService) SaveSeriesFromTMDb(ctx context.Context, tmdbSeries *tmdb.TVShowDetails, filePath string) (*models.Series, error) {
	if tmdbSeries == nil {
		return nil, fmt.Errorf("tmdb series cannot be nil")
	}

	series := ConvertTMDbSeriesToModel(tmdbSeries, filePath)

	// Generate ID if not set
	if series.ID == "" {
		series.ID = uuid.New().String()
	}

	// Upsert to prevent duplicates (uses TMDb ID as unique identifier)
	if err := s.seriesRepo.Upsert(ctx, series); err != nil {
		s.logger.Error("Failed to save series",
			"tmdbId", tmdbSeries.ID,
			"title", tmdbSeries.Name,
			"error", err,
		)
		return nil, fmt.Errorf("failed to save series: %w", err)
	}

	s.logger.Info("Series saved successfully",
		"id", series.ID,
		"tmdbId", tmdbSeries.ID,
		"title", series.Title,
	)

	return series, nil
}

// SearchLibrary performs unified FTS search across movies and series
// Results are returned within 500ms as per NFR-SC8
func (s *LibraryService) SearchLibrary(ctx context.Context, query string, params repository.ListParams) (*LibrarySearchResults, error) {
	params.Validate()

	var wg sync.WaitGroup
	var moviesErr, seriesErr error
	var movies []models.Movie
	var series []models.Series
	var moviesPagination, seriesPagination *repository.PaginationResult

	// Search movies and series in parallel for performance
	wg.Add(2)

	go func() {
		defer wg.Done()
		movies, moviesPagination, moviesErr = s.movieRepo.FullTextSearch(ctx, query, params)
	}()

	go func() {
		defer wg.Done()
		series, seriesPagination, seriesErr = s.seriesRepo.FullTextSearch(ctx, query, params)
	}()

	wg.Wait()

	// Check for errors
	if moviesErr != nil {
		s.logger.Error("Failed to search movies", "query", query, "error", moviesErr)
		return nil, fmt.Errorf("failed to search movies: %w", moviesErr)
	}
	if seriesErr != nil {
		s.logger.Error("Failed to search series", "query", query, "error", seriesErr)
		return nil, fmt.Errorf("failed to search series: %w", seriesErr)
	}

	// Merge results
	results := make([]SearchResult, 0, len(movies)+len(series))

	for i := range movies {
		results = append(results, SearchResult{
			Type:  "movie",
			Movie: &movies[i],
		})
	}

	for i := range series {
		results = append(results, SearchResult{
			Type:   "series",
			Series: &series[i],
		})
	}

	totalCount := 0
	if moviesPagination != nil {
		totalCount += moviesPagination.TotalResults
	}
	if seriesPagination != nil {
		totalCount += seriesPagination.TotalResults
	}

	return &LibrarySearchResults{
		Results:    results,
		Movies:     moviesPagination,
		Series:     seriesPagination,
		TotalCount: totalCount,
	}, nil
}

// GetMovieByID retrieves a movie by its ID
func (s *LibraryService) GetMovieByID(ctx context.Context, id string) (*models.Movie, error) {
	if id == "" {
		return nil, fmt.Errorf("movie ID cannot be empty")
	}
	return s.movieRepo.FindByID(ctx, id)
}

// GetSeriesByID retrieves a series by its ID
func (s *LibraryService) GetSeriesByID(ctx context.Context, id string) (*models.Series, error) {
	if id == "" {
		return nil, fmt.Errorf("series ID cannot be empty")
	}
	return s.seriesRepo.FindByID(ctx, id)
}

// GetMovieByTMDbID retrieves a movie by TMDb ID
func (s *LibraryService) GetMovieByTMDbID(ctx context.Context, tmdbID int64) (*models.Movie, error) {
	return s.movieRepo.FindByTMDbID(ctx, tmdbID)
}

// GetSeriesByTMDbID retrieves a series by TMDb ID
func (s *LibraryService) GetSeriesByTMDbID(ctx context.Context, tmdbID int64) (*models.Series, error) {
	return s.seriesRepo.FindByTMDbID(ctx, tmdbID)
}

// Compile-time interface verification
var _ LibraryServiceInterface = (*LibraryService)(nil)
