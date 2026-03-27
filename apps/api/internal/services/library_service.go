package services

import (
	"context"
	"errors"
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

// ErrNotFound is returned when a requested resource does not exist
var ErrNotFound = errors.New("not found")

// SearchResult represents a unified search result item
type SearchResult struct {
	Type   string         `json:"type"` // "movie" or "series"
	Movie  *models.Movie  `json:"movie,omitempty"`
	Series *models.Series `json:"series,omitempty"`
}

// LibrarySearchResults contains unified search results across movies and series
type LibrarySearchResults struct {
	Results    []SearchResult               `json:"results"`
	Movies     *repository.PaginationResult `json:"movies_pagination"`
	Series     *repository.PaginationResult `json:"series_pagination"`
	TotalCount int                          `json:"total_count"`
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

// TMDbVideosProvider provides access to TMDb video data for on-demand fetching
type TMDbVideosProvider interface {
	GetMovieVideos(ctx context.Context, movieID int) (*tmdb.VideosResponse, error)
	GetTVShowVideos(ctx context.Context, tvID int) (*tmdb.VideosResponse, error)
}

// LibraryServiceInterface defines the contract for media library operations
type LibraryServiceInterface interface {
	// SaveMovieFromTMDb saves a movie from TMDb search/details to the database
	SaveMovieFromTMDb(ctx context.Context, tmdbMovie *tmdb.MovieDetails, filePath string) (*models.Movie, error)

	// SaveSeriesFromTMDb saves a series from TMDb search/details to the database
	SaveSeriesFromTMDb(ctx context.Context, tmdbSeries *tmdb.TVShowDetails, filePath string) (*models.Series, error)

	// SearchLibrary performs unified FTS search across movies and series.
	// mediaType filters results: "all" (default), "movie", or "tv".
	SearchLibrary(ctx context.Context, query string, params repository.ListParams, mediaType string) (*LibrarySearchResults, error)

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

	// GetDistinctGenres returns all unique genres across movies and series
	GetDistinctGenres(ctx context.Context) ([]string, error)

	// GetLibraryStats returns library statistics including year range and counts
	GetLibraryStats(ctx context.Context) (*LibraryStats, error)

	// GetMovieVideos retrieves videos for a library movie by looking up its TMDb ID
	GetMovieVideos(ctx context.Context, id string) (*tmdb.VideosResponse, error)

	// GetSeriesVideos retrieves videos for a library series by looking up its TMDb ID
	GetSeriesVideos(ctx context.Context, id string) (*tmdb.VideosResponse, error)

	// BatchDelete deletes multiple items by IDs and type
	BatchDelete(ctx context.Context, ids []string, mediaType string) (*BatchResult, error)

	// BatchReparse resets parse_status to "pending" for multiple items
	BatchReparse(ctx context.Context, ids []string, mediaType string) (*BatchResult, error)

	// BatchExport returns metadata for multiple items
	BatchExport(ctx context.Context, ids []string, mediaType string) ([]interface{}, error)
}

// LibraryService handles media library storage and search operations
type LibraryService struct {
	movieRepo      repository.MovieRepositoryInterface
	seriesRepo     repository.SeriesRepositoryInterface
	episodeRepo    repository.EpisodeRepositoryInterface
	tmdbVideos     TMDbVideosProvider
	logger         *slog.Logger
}

// NewLibraryService creates a new LibraryService
func NewLibraryService(
	movieRepo repository.MovieRepositoryInterface,
	seriesRepo repository.SeriesRepositoryInterface,
	episodeRepo repository.EpisodeRepositoryInterface,
	opts ...LibraryServiceOption,
) *LibraryService {
	s := &LibraryService{
		movieRepo:   movieRepo,
		seriesRepo:  seriesRepo,
		episodeRepo: episodeRepo,
		logger:      slog.Default().With("service", "library"),
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// LibraryServiceOption configures optional dependencies for LibraryService
type LibraryServiceOption func(*LibraryService)

// WithTMDbVideos sets the TMDb videos provider for on-demand video fetching
func WithTMDbVideos(provider TMDbVideosProvider) LibraryServiceOption {
	return func(s *LibraryService) {
		s.tmdbVideos = provider
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

// SearchLibrary performs unified FTS search across movies and series.
// mediaType filters results: "all" (default), "movie", or "tv".
// Results are returned within 500ms as per NFR-SC8.
func (s *LibraryService) SearchLibrary(ctx context.Context, query string, params repository.ListParams, mediaType string) (*LibrarySearchResults, error) {
	params.Validate()

	searchMovies := mediaType == "" || mediaType == "all" || mediaType == "movie"
	searchSeries := mediaType == "" || mediaType == "all" || mediaType == "tv"

	var wg sync.WaitGroup
	var moviesErr, seriesErr error
	var movies []models.Movie
	var series []models.Series
	var moviesPagination, seriesPagination *repository.PaginationResult

	// Search movies and series in parallel for performance
	if searchMovies {
		wg.Add(1)
		go func() {
			defer wg.Done()
			movies, moviesPagination, moviesErr = s.movieRepo.FullTextSearch(ctx, query, params)
		}()
	}

	if searchSeries {
		wg.Add(1)
		go func() {
			defer wg.Done()
			series, seriesPagination, seriesErr = s.seriesRepo.FullTextSearch(ctx, query, params)
		}()
	}

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
	// To correctly interleave movies+series across pages, we must fetch enough
	// items from both repos to cover the requested page window after merging.
	// Override each repo query to fetch page=1 with limit=page*pageSize.
	fetchParams := params
	fetchParams.Page = 1
	fetchParams.PageSize = params.Page * params.PageSize

	var wg sync.WaitGroup
	var moviesErr, seriesErr error
	var movies []models.Movie
	var series []models.Series
	var moviesPagination, seriesPagination *repository.PaginationResult

	wg.Add(2)

	go func() {
		defer wg.Done()
		movies, moviesPagination, moviesErr = s.movieRepo.List(ctx, fetchParams)
	}()

	go func() {
		defer wg.Done()
		series, seriesPagination, seriesErr = s.seriesRepo.List(ctx, fetchParams)
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

	// Combine items and sort to interleave correctly
	allItems := make([]LibraryItem, 0, len(movies)+len(series))
	for i := range movies {
		allItems = append(allItems, LibraryItem{Type: "movie", Movie: &movies[i]})
	}
	for i := range series {
		allItems = append(allItems, LibraryItem{Type: "series", Series: &series[i]})
	}

	// Sort combined items by the requested field to ensure correct interleaved ordering
	sort.Slice(allItems, func(i, j int) bool {
		return compareLibraryItems(allItems[i], allItems[j], params.SortBy, params.SortOrder)
	})

	// Slice to the correct page window from the merged result
	start := (params.Page - 1) * params.PageSize
	end := start + params.PageSize
	if start > len(allItems) {
		start = len(allItems)
	}
	if end > len(allItems) {
		end = len(allItems)
	}
	items := allItems[start:end]

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

// compareLibraryItems compares two library items by the given sort field and order.
func compareLibraryItems(a, b LibraryItem, sortBy, sortOrder string) bool {
	asc := sortOrder == "asc"

	switch sortBy {
	case "title":
		at := getTitle(a)
		bt := getTitle(b)
		if asc {
			return at < bt
		}
		return at > bt

	case "release_date":
		at := getReleaseDate(a)
		bt := getReleaseDate(b)
		if asc {
			return at < bt
		}
		return at > bt

	case "rating", "vote_average":
		ar := getVoteAverage(a)
		br := getVoteAverage(b)
		if asc {
			return ar < br
		}
		return ar > br

	default: // created_at
		at := getCreatedAt(a)
		bt := getCreatedAt(b)
		if asc {
			return at.Before(bt)
		}
		return at.After(bt)
	}
}

// getCreatedAt extracts the created_at timestamp from a LibraryItem.
func getCreatedAt(item LibraryItem) time.Time {
	if item.Movie != nil {
		return item.Movie.CreatedAt
	}
	if item.Series != nil {
		return item.Series.CreatedAt
	}
	return time.Time{}
}

// getTitle extracts the title from a LibraryItem.
func getTitle(item LibraryItem) string {
	if item.Movie != nil {
		return item.Movie.Title
	}
	if item.Series != nil {
		return item.Series.Title
	}
	return ""
}

// getReleaseDate extracts the release date string from a LibraryItem.
func getReleaseDate(item LibraryItem) string {
	if item.Movie != nil {
		return item.Movie.ReleaseDate
	}
	if item.Series != nil {
		return item.Series.FirstAirDate
	}
	return ""
}

// getVoteAverage extracts the vote average from a LibraryItem.
func getVoteAverage(item LibraryItem) float64 {
	if item.Movie != nil && item.Movie.VoteAverage.Valid {
		return item.Movie.VoteAverage.Float64
	}
	if item.Series != nil && item.Series.VoteAverage.Valid {
		return item.Series.VoteAverage.Float64
	}
	return 0
}

// LibraryStats contains library statistics
type LibraryStats struct {
	YearMin    int `json:"year_min"`
	YearMax    int `json:"year_max"`
	MovieCount int `json:"movie_count"`
	TvCount    int `json:"tv_count"`
	TotalCount int `json:"total_count"`
}

// GetDistinctGenres returns all unique genres across movies and series
func (s *LibraryService) GetDistinctGenres(ctx context.Context) ([]string, error) {
	var wg sync.WaitGroup
	var movieGenres, seriesGenres []string
	var movieErr, seriesErr error

	wg.Add(2)
	go func() {
		defer wg.Done()
		movieGenres, movieErr = s.movieRepo.GetDistinctGenres(ctx)
	}()
	go func() {
		defer wg.Done()
		seriesGenres, seriesErr = s.seriesRepo.GetDistinctGenres(ctx)
	}()
	wg.Wait()

	if movieErr != nil {
		return nil, fmt.Errorf("failed to get movie genres: %w", movieErr)
	}
	if seriesErr != nil {
		return nil, fmt.Errorf("failed to get series genres: %w", seriesErr)
	}

	// Merge and deduplicate
	genreSet := make(map[string]struct{})
	for _, g := range movieGenres {
		genreSet[g] = struct{}{}
	}
	for _, g := range seriesGenres {
		genreSet[g] = struct{}{}
	}

	genres := make([]string, 0, len(genreSet))
	for g := range genreSet {
		genres = append(genres, g)
	}

	sort.Strings(genres)
	return genres, nil
}

// GetLibraryStats returns library statistics including year range and counts
func (s *LibraryService) GetLibraryStats(ctx context.Context) (*LibraryStats, error) {
	var wg sync.WaitGroup
	var movieMinYear, movieMaxYear, seriesMinYear, seriesMaxYear int
	var movieCount, seriesCount int
	var movieYearErr, seriesYearErr, movieCountErr, seriesCountErr error

	wg.Add(4)
	go func() {
		defer wg.Done()
		movieMinYear, movieMaxYear, movieYearErr = s.movieRepo.GetYearRange(ctx)
	}()
	go func() {
		defer wg.Done()
		seriesMinYear, seriesMaxYear, seriesYearErr = s.seriesRepo.GetYearRange(ctx)
	}()
	go func() {
		defer wg.Done()
		movieCount, movieCountErr = s.movieRepo.Count(ctx)
	}()
	go func() {
		defer wg.Done()
		seriesCount, seriesCountErr = s.seriesRepo.Count(ctx)
	}()
	wg.Wait()

	if movieYearErr != nil {
		return nil, fmt.Errorf("failed to get movie year range: %w", movieYearErr)
	}
	if seriesYearErr != nil {
		return nil, fmt.Errorf("failed to get series year range: %w", seriesYearErr)
	}
	if movieCountErr != nil {
		return nil, fmt.Errorf("failed to count movies: %w", movieCountErr)
	}
	if seriesCountErr != nil {
		return nil, fmt.Errorf("failed to count series: %w", seriesCountErr)
	}

	// Calculate overall min/max year
	minYear := movieMinYear
	if seriesMinYear > 0 && (minYear == 0 || seriesMinYear < minYear) {
		minYear = seriesMinYear
	}
	maxYear := movieMaxYear
	if seriesMaxYear > maxYear {
		maxYear = seriesMaxYear
	}

	return &LibraryStats{
		YearMin:    minYear,
		YearMax:    maxYear,
		MovieCount: movieCount,
		TvCount:    seriesCount,
		TotalCount: movieCount + seriesCount,
	}, nil
}

// GetMovieVideos retrieves videos for a library movie by looking up its TMDb ID
func (s *LibraryService) GetMovieVideos(ctx context.Context, id string) (*tmdb.VideosResponse, error) {
	if s.tmdbVideos == nil {
		return &tmdb.VideosResponse{Results: []tmdb.Video{}}, nil
	}

	movie, err := s.movieRepo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("%w: movie %s", ErrNotFound, id)
	}

	if !movie.TMDbID.Valid || movie.TMDbID.Int64 <= 0 {
		return &tmdb.VideosResponse{Results: []tmdb.Video{}}, nil
	}

	videos, err := s.tmdbVideos.GetMovieVideos(ctx, int(movie.TMDbID.Int64))
	if err != nil {
		slog.Error("Failed to fetch movie videos from TMDb", "error", err, "movie_id", id, "tmdb_id", movie.TMDbID.Int64)
		return nil, fmt.Errorf("failed to fetch videos: %w", err)
	}

	return videos, nil
}

// GetSeriesVideos retrieves videos for a library series by looking up its TMDb ID
func (s *LibraryService) GetSeriesVideos(ctx context.Context, id string) (*tmdb.VideosResponse, error) {
	if s.tmdbVideos == nil {
		return &tmdb.VideosResponse{Results: []tmdb.Video{}}, nil
	}

	series, err := s.seriesRepo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("%w: series %s", ErrNotFound, id)
	}

	if !series.TMDbID.Valid || series.TMDbID.Int64 <= 0 {
		return &tmdb.VideosResponse{Results: []tmdb.Video{}}, nil
	}

	videos, err := s.tmdbVideos.GetTVShowVideos(ctx, int(series.TMDbID.Int64))
	if err != nil {
		slog.Error("Failed to fetch series videos from TMDb", "error", err, "series_id", id, "tmdb_id", series.TMDbID.Int64)
		return nil, fmt.Errorf("failed to fetch videos: %w", err)
	}

	return videos, nil
}

// BatchResult contains the result of a batch operation
type BatchResult struct {
	SuccessCount int          `json:"success_count"`
	FailedCount  int          `json:"failed_count"`
	Errors       []BatchError `json:"errors,omitempty"`
}

// BatchError represents a single item failure in a batch operation
type BatchError struct {
	ID      string `json:"id"`
	Message string `json:"message"`
}

// BatchDelete deletes multiple items by IDs and type
func (s *LibraryService) BatchDelete(ctx context.Context, ids []string, mediaType string) (*BatchResult, error) {
	if len(ids) == 0 {
		return nil, fmt.Errorf("no IDs provided")
	}

	result := &BatchResult{}
	for _, id := range ids {
		var err error
		if mediaType == "movie" {
			err = s.movieRepo.Delete(ctx, id)
		} else {
			err = s.seriesRepo.Delete(ctx, id)
		}
		if err != nil {
			result.FailedCount++
			result.Errors = append(result.Errors, BatchError{ID: id, Message: err.Error()})
			s.logger.Error("Batch delete failed for item", "id", id, "type", mediaType, "error", err)
		} else {
			result.SuccessCount++
		}
	}

	s.logger.Info("Batch delete completed", "type", mediaType, "success", result.SuccessCount, "failed", result.FailedCount)
	return result, nil
}

// BatchReparse resets parse_status to "pending" for multiple items
func (s *LibraryService) BatchReparse(ctx context.Context, ids []string, mediaType string) (*BatchResult, error) {
	if len(ids) == 0 {
		return nil, fmt.Errorf("no IDs provided")
	}

	result := &BatchResult{}
	for _, id := range ids {
		var err error
		if mediaType == "movie" {
			movie, findErr := s.movieRepo.FindByID(ctx, id)
			if findErr != nil {
				err = findErr
			} else {
				movie.ParseStatus = "pending"
				err = s.movieRepo.Update(ctx, movie)
			}
		} else {
			series, findErr := s.seriesRepo.FindByID(ctx, id)
			if findErr != nil {
				err = findErr
			} else {
				series.ParseStatus = "pending"
				err = s.seriesRepo.Update(ctx, series)
			}
		}
		if err != nil {
			result.FailedCount++
			result.Errors = append(result.Errors, BatchError{ID: id, Message: err.Error()})
			s.logger.Error("Batch reparse failed for item", "id", id, "type", mediaType, "error", err)
		} else {
			result.SuccessCount++
		}
	}

	s.logger.Info("Batch reparse completed", "type", mediaType, "success", result.SuccessCount, "failed", result.FailedCount)
	return result, nil
}

// BatchExportResult contains exported items and any errors
type BatchExportResult struct {
	Items  []interface{} `json:"items"`
	Errors []BatchError  `json:"errors,omitempty"`
}

// BatchExport returns metadata for multiple items
func (s *LibraryService) BatchExport(ctx context.Context, ids []string, mediaType string) ([]interface{}, error) {
	if len(ids) == 0 {
		return nil, fmt.Errorf("no IDs provided")
	}

	results := make([]interface{}, 0, len(ids))
	for _, id := range ids {
		if mediaType == "movie" {
			movie, err := s.movieRepo.FindByID(ctx, id)
			if err != nil {
				s.logger.Warn("Batch export: movie not found", "id", id, "error", err)
				continue
			}
			results = append(results, movie)
		} else {
			series, err := s.seriesRepo.FindByID(ctx, id)
			if err != nil {
				s.logger.Warn("Batch export: series not found", "id", id, "error", err)
				continue
			}
			results = append(results, series)
		}
	}

	if len(results) < len(ids) {
		s.logger.Warn("Batch export: some items not found",
			"requested", len(ids), "found", len(results), "type", mediaType)
	}

	return results, nil
}

// Compile-time interface verification
var _ LibraryServiceInterface = (*LibraryService)(nil)
