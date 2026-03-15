package services

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"sync"
	"time"

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

// LibraryListResult contains combined movie + series listing with pagination
type LibraryListResult struct {
	Items      []LibraryItem                `json:"items"`
	Pagination *repository.PaginationResult `json:"pagination"`
}

// LibraryItem represents a unified library item (movie or series)
type LibraryItem struct {
	Type   string         `json:"type"` // "movie" or "series"
	Movie  *models.Movie  `json:"movie,omitempty"`
	Series *models.Series `json:"series,omitempty"`
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

	// ListLibrary lists media items with pagination and optional type filtering
	ListLibrary(ctx context.Context, params repository.ListParams, mediaType string) (*LibraryListResult, error)

	// GetRecentlyAdded returns the most recently added media items
	GetRecentlyAdded(ctx context.Context, limit int) (*LibraryListResult, error)

	// DeleteMovie deletes a movie by ID
	DeleteMovie(ctx context.Context, id string) error

	// DeleteSeries deletes a series by ID
	DeleteSeries(ctx context.Context, id string) error
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

// ListLibrary lists media items with pagination and optional type filtering.
// mediaType can be "all", "movie", or "tv".
// Default sort is created_at DESC (newest first).
func (s *LibraryService) ListLibrary(ctx context.Context, params repository.ListParams, mediaType string) (*LibraryListResult, error) {
	params.Validate()

	// Default sort to created_at DESC
	if params.SortBy == "" {
		params.SortBy = "created_at"
	}
	if params.SortOrder == "" {
		params.SortOrder = "desc"
	}

	switch mediaType {
	case "movie":
		return s.listMoviesOnly(ctx, params)
	case "tv":
		return s.listSeriesOnly(ctx, params)
	default:
		return s.listAll(ctx, params)
	}
}

func (s *LibraryService) listMoviesOnly(ctx context.Context, params repository.ListParams) (*LibraryListResult, error) {
	movies, pagination, err := s.movieRepo.List(ctx, params)
	if err != nil {
		s.logger.Error("Failed to list movies", "error", err)
		return nil, fmt.Errorf("failed to list movies: %w", err)
	}

	items := make([]LibraryItem, len(movies))
	for i := range movies {
		items[i] = LibraryItem{Type: "movie", Movie: &movies[i]}
	}

	return &LibraryListResult{Items: items, Pagination: pagination}, nil
}

func (s *LibraryService) listSeriesOnly(ctx context.Context, params repository.ListParams) (*LibraryListResult, error) {
	series, pagination, err := s.seriesRepo.List(ctx, params)
	if err != nil {
		s.logger.Error("Failed to list series", "error", err)
		return nil, fmt.Errorf("failed to list series: %w", err)
	}

	items := make([]LibraryItem, len(series))
	for i := range series {
		items[i] = LibraryItem{Type: "series", Series: &series[i]}
	}

	return &LibraryListResult{Items: items, Pagination: pagination}, nil
}

func (s *LibraryService) listAll(ctx context.Context, params repository.ListParams) (*LibraryListResult, error) {
	// Fetch counts first to compute correct per-type page sizes
	var wg sync.WaitGroup
	var moviesErr, seriesErr error
	var movies []models.Movie
	var series []models.Series
	var moviesPagination, seriesPagination *repository.PaginationResult

	// Query both repos with full page size — we'll trim to the requested pageSize after merging
	wg.Add(2)

	go func() {
		defer wg.Done()
		movies, moviesPagination, moviesErr = s.movieRepo.List(ctx, params)
	}()

	go func() {
		defer wg.Done()
		series, seriesPagination, seriesErr = s.seriesRepo.List(ctx, params)
	}()

	wg.Wait()

	if moviesErr != nil {
		s.logger.Error("Failed to list movies", "error", moviesErr)
		return nil, fmt.Errorf("failed to list movies: %w", moviesErr)
	}
	if seriesErr != nil {
		s.logger.Error("Failed to list series", "error", seriesErr)
		return nil, fmt.Errorf("failed to list series: %w", seriesErr)
	}

	// Combine items and sort by created_at to interleave correctly
	allItems := make([]LibraryItem, 0, len(movies)+len(series))
	for i := range movies {
		allItems = append(allItems, LibraryItem{Type: "movie", Movie: &movies[i]})
	}
	for i := range series {
		allItems = append(allItems, LibraryItem{Type: "series", Series: &series[i]})
	}

	// Sort combined items by created_at to ensure correct interleaved ordering
	sort.Slice(allItems, func(i, j int) bool {
		ti := getCreatedAt(allItems[i])
		tj := getCreatedAt(allItems[j])
		if params.SortOrder == "asc" {
			return ti.Before(tj)
		}
		return ti.After(tj) // DESC by default
	})

	// Trim combined results to respect the requested pageSize
	items := allItems
	if len(items) > params.PageSize {
		items = items[:params.PageSize]
	}

	// Compute combined pagination from total counts
	totalResults := 0
	if moviesPagination != nil {
		totalResults += moviesPagination.TotalResults
	}
	if seriesPagination != nil {
		totalResults += seriesPagination.TotalResults
	}

	totalPages := 0
	if params.PageSize > 0 {
		totalPages = (totalResults + params.PageSize - 1) / params.PageSize
	}

	return &LibraryListResult{
		Items: items,
		Pagination: &repository.PaginationResult{
			Page:         params.Page,
			PageSize:     params.PageSize,
			TotalResults: totalResults,
			TotalPages:   totalPages,
		},
	}, nil
}

// GetRecentlyAdded returns the most recently added media items sorted by created_at DESC.
func (s *LibraryService) GetRecentlyAdded(ctx context.Context, limit int) (*LibraryListResult, error) {
	if limit <= 0 {
		limit = 20
	}
	params := repository.ListParams{
		Page:      1,
		PageSize:  limit,
		SortBy:    "created_at",
		SortOrder: "desc",
	}
	return s.ListLibrary(ctx, params, "all")
}

// DeleteMovie deletes a movie by ID
func (s *LibraryService) DeleteMovie(ctx context.Context, id string) error {
	if id == "" {
		return fmt.Errorf("movie ID cannot be empty")
	}
	if err := s.movieRepo.Delete(ctx, id); err != nil {
		s.logger.Error("Failed to delete movie", "error", err, "id", id)
		return fmt.Errorf("failed to delete movie: %w", err)
	}
	s.logger.Info("Movie deleted", "id", id)
	return nil
}

// DeleteSeries deletes a series by ID
func (s *LibraryService) DeleteSeries(ctx context.Context, id string) error {
	if id == "" {
		return fmt.Errorf("series ID cannot be empty")
	}
	if err := s.seriesRepo.Delete(ctx, id); err != nil {
		s.logger.Error("Failed to delete series", "error", err, "id", id)
		return fmt.Errorf("failed to delete series: %w", err)
	}
	s.logger.Info("Series deleted", "id", id)
	return nil
}

// getCreatedAt extracts the created_at timestamp from a LibraryItem for sorting.
func getCreatedAt(item LibraryItem) time.Time {
	if item.Movie != nil {
		return item.Movie.CreatedAt
	}
	if item.Series != nil {
		return item.Series.CreatedAt
	}
	return time.Time{}
}

// Compile-time interface verification
var _ LibraryServiceInterface = (*LibraryService)(nil)
