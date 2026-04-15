package handlers

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/vido/api/internal/tmdb"
)

// TMDbServiceInterface defines the contract for TMDb operations.
// This interface enables testing handlers with mock services.
type TMDbServiceInterface interface {
	SearchMovies(ctx context.Context, query string, page int) (*tmdb.SearchResultMovies, error)
	SearchTVShows(ctx context.Context, query string, page int) (*tmdb.SearchResultTVShows, error)
	GetMovieDetails(ctx context.Context, movieID int) (*tmdb.MovieDetails, error)
	GetTVShowDetails(ctx context.Context, tvID int) (*tmdb.TVShowDetails, error)
	GetTrendingMovies(ctx context.Context, timeWindow string, page int) (*tmdb.SearchResultMovies, error)
	GetTrendingTVShows(ctx context.Context, timeWindow string, page int) (*tmdb.SearchResultTVShows, error)
	DiscoverMovies(ctx context.Context, params tmdb.DiscoverParams) (*tmdb.SearchResultMovies, error)
	DiscoverTVShows(ctx context.Context, params tmdb.DiscoverParams) (*tmdb.SearchResultTVShows, error)
}

// TMDbHandler handles HTTP requests for TMDb operations.
// It uses TMDbServiceInterface for business logic, following the
// Handler → Service → Repository → Database architecture.
type TMDbHandler struct {
	service TMDbServiceInterface
}

// NewTMDbHandler creates a new TMDbHandler with the given service.
func NewTMDbHandler(service TMDbServiceInterface) *TMDbHandler {
	return &TMDbHandler{
		service: service,
	}
}

// SearchMovies handles GET /api/v1/tmdb/search/movies
// Searches for movies on TMDb by query
// @Summary Search movies on TMDb
// @Description Search for movies using TMDb API with zh-TW language priority
// @Tags tmdb
// @Accept json
// @Produce json
// @Param query query string true "Search query"
// @Param page query int false "Page number" default(1)
// @Success 200 {object} APIResponse{data=tmdb.SearchResultMovies}
// @Failure 400 {object} APIResponse{error=APIError}
// @Failure 500 {object} APIResponse{error=APIError}
// @Router /api/v1/tmdb/search/movies [get]
func (h *TMDbHandler) SearchMovies(c *gin.Context) {
	query := c.Query("query")
	if query == "" {
		ErrorResponse(c, http.StatusBadRequest, tmdb.ErrCodeBadRequest,
			"Search query is required",
			"Please provide a 'query' parameter")
		return
	}

	page := 1
	if pageStr := c.Query("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	result, err := h.service.SearchMovies(c.Request.Context(), query, page)
	if err != nil {
		handleTMDbError(c, err, "search movies", slog.String("query", query))
		return
	}

	SuccessResponse(c, result)
}

// SearchTVShows handles GET /api/v1/tmdb/search/tv
// Searches for TV shows on TMDb by query
// @Summary Search TV shows on TMDb
// @Description Search for TV shows using TMDb API with zh-TW language priority
// @Tags tmdb
// @Accept json
// @Produce json
// @Param query query string true "Search query"
// @Param page query int false "Page number" default(1)
// @Success 200 {object} APIResponse{data=tmdb.SearchResultTVShows}
// @Failure 400 {object} APIResponse{error=APIError}
// @Failure 500 {object} APIResponse{error=APIError}
// @Router /api/v1/tmdb/search/tv [get]
func (h *TMDbHandler) SearchTVShows(c *gin.Context) {
	query := c.Query("query")
	if query == "" {
		ErrorResponse(c, http.StatusBadRequest, tmdb.ErrCodeBadRequest,
			"Search query is required",
			"Please provide a 'query' parameter")
		return
	}

	page := 1
	if pageStr := c.Query("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	result, err := h.service.SearchTVShows(c.Request.Context(), query, page)
	if err != nil {
		handleTMDbError(c, err, "search TV shows", slog.String("query", query))
		return
	}

	SuccessResponse(c, result)
}

// GetMovieDetails handles GET /api/v1/tmdb/movies/:id
// Gets movie details from TMDb by ID
// @Summary Get movie details from TMDb
// @Description Get detailed movie information from TMDb API
// @Tags tmdb
// @Accept json
// @Produce json
// @Param id path int true "TMDb Movie ID"
// @Success 200 {object} APIResponse{data=tmdb.MovieDetails}
// @Failure 400 {object} APIResponse{error=APIError}
// @Failure 404 {object} APIResponse{error=APIError}
// @Failure 500 {object} APIResponse{error=APIError}
// @Router /api/v1/tmdb/movies/{id} [get]
func (h *TMDbHandler) GetMovieDetails(c *gin.Context) {
	idStr := c.Param("id")
	if idStr == "" {
		ErrorResponse(c, http.StatusBadRequest, tmdb.ErrCodeBadRequest,
			"Movie ID is required",
			"Please provide a movie ID in the URL path")
		return
	}

	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		ErrorResponse(c, http.StatusBadRequest, tmdb.ErrCodeBadRequest,
			"Invalid movie ID",
			"Movie ID must be a positive integer")
		return
	}

	result, err := h.service.GetMovieDetails(c.Request.Context(), id)
	if err != nil {
		handleTMDbError(c, err, "get movie details", slog.Int("movie_id", id))
		return
	}

	SuccessResponse(c, result)
}

// GetTVShowDetails handles GET /api/v1/tmdb/tv/:id
// Gets TV show details from TMDb by ID
// @Summary Get TV show details from TMDb
// @Description Get detailed TV show information from TMDb API
// @Tags tmdb
// @Accept json
// @Produce json
// @Param id path int true "TMDb TV Show ID"
// @Success 200 {object} APIResponse{data=tmdb.TVShowDetails}
// @Failure 400 {object} APIResponse{error=APIError}
// @Failure 404 {object} APIResponse{error=APIError}
// @Failure 500 {object} APIResponse{error=APIError}
// @Router /api/v1/tmdb/tv/{id} [get]
func (h *TMDbHandler) GetTVShowDetails(c *gin.Context) {
	idStr := c.Param("id")
	if idStr == "" {
		ErrorResponse(c, http.StatusBadRequest, tmdb.ErrCodeBadRequest,
			"TV show ID is required",
			"Please provide a TV show ID in the URL path")
		return
	}

	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		ErrorResponse(c, http.StatusBadRequest, tmdb.ErrCodeBadRequest,
			"Invalid TV show ID",
			"TV show ID must be a positive integer")
		return
	}

	result, err := h.service.GetTVShowDetails(c.Request.Context(), id)
	if err != nil {
		handleTMDbError(c, err, "get TV show details", slog.Int("tv_id", id))
		return
	}

	SuccessResponse(c, result)
}

// GetTrendingMovies handles GET /api/v1/tmdb/trending/movies?time_window=week&page=1
// Returns trending movies with zh-TW content filtering (Story 10-1 AC #1, #3, #4, #5).
func (h *TMDbHandler) GetTrendingMovies(c *gin.Context) {
	timeWindow := parseTrendingWindow(c.Query("time_window"))
	page := parsePageQuery(c.Query("page"))

	result, err := h.service.GetTrendingMovies(c.Request.Context(), timeWindow, page)
	if err != nil {
		handleTMDbError(c, err, "get trending movies",
			slog.String("time_window", timeWindow),
			slog.Int("page", page),
		)
		return
	}
	SuccessResponse(c, result)
}

// GetTrendingTVShows handles GET /api/v1/tmdb/trending/tv?time_window=week&page=1
func (h *TMDbHandler) GetTrendingTVShows(c *gin.Context) {
	timeWindow := parseTrendingWindow(c.Query("time_window"))
	page := parsePageQuery(c.Query("page"))

	result, err := h.service.GetTrendingTVShows(c.Request.Context(), timeWindow, page)
	if err != nil {
		handleTMDbError(c, err, "get trending TV shows",
			slog.String("time_window", timeWindow),
			slog.Int("page", page),
		)
		return
	}
	SuccessResponse(c, result)
}

// DiscoverMovies handles GET /api/v1/tmdb/discover/movies with filter params
// (genre, year_gte, year_lte, region, language, sort, page).
func (h *TMDbHandler) DiscoverMovies(c *gin.Context) {
	params, err := parseDiscoverParams(c)
	if err != nil {
		handleTMDbError(c, err, "discover movies", slog.Any("params", params))
		return
	}
	result, err := h.service.DiscoverMovies(c.Request.Context(), params)
	if err != nil {
		handleTMDbError(c, err, "discover movies", slog.Any("params", params))
		return
	}
	SuccessResponse(c, result)
}

// DiscoverTVShows handles GET /api/v1/tmdb/discover/tv with the same filter params.
func (h *TMDbHandler) DiscoverTVShows(c *gin.Context) {
	params, err := parseDiscoverParams(c)
	if err != nil {
		handleTMDbError(c, err, "discover TV shows", slog.Any("params", params))
		return
	}
	result, err := h.service.DiscoverTVShows(c.Request.Context(), params)
	if err != nil {
		handleTMDbError(c, err, "discover TV shows", slog.Any("params", params))
		return
	}
	SuccessResponse(c, result)
}

// parseTrendingWindow normalizes the time_window query param; unknown / empty
// values default to "week" (TMDb's most useful default for a homepage feed).
func parseTrendingWindow(raw string) string {
	switch raw {
	case "day", "week":
		return raw
	default:
		return "week"
	}
}

// parsePageQuery converts a `?page=N` string to an int; returns 1 on parse failure
// or non-positive values.
func parsePageQuery(raw string) int {
	if raw == "" {
		return 1
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 1 {
		return 1
	}
	return n
}

// parseDiscoverParams maps the handler's query-string parameters to a
// tmdb.DiscoverParams struct. Keys use snake_case per Rule 18:
//   - genre       → DiscoverParams.Genre (comma-separated IDs)
//   - year_gte    → DiscoverParams.YearGte
//   - year_lte    → DiscoverParams.YearLte
//   - region      → DiscoverParams.Region
//   - language    → DiscoverParams.Language
//   - sort        → DiscoverParams.SortBy
//   - page        → DiscoverParams.Page
//
// Returns a non-nil *tmdb.TMDbError when both year bounds are non-zero
// and year_gte > year_lte (Story 10-1a). Zero values for either bound
// retain the "unlimited" semantics from Story 10-1 and skip validation.
func parseDiscoverParams(c *gin.Context) (tmdb.DiscoverParams, error) {
	p := tmdb.DiscoverParams{
		Genre:    c.Query("genre"),
		Region:   c.Query("region"),
		Language: c.Query("language"),
		SortBy:   c.Query("sort"),
		Page:     parsePageQuery(c.Query("page")),
	}
	if v := c.Query("year_gte"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			p.YearGte = n
		}
	}
	if v := c.Query("year_lte"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			p.YearLte = n
		}
	}
	if p.YearGte > 0 && p.YearLte > 0 && p.YearGte > p.YearLte {
		return p, tmdb.NewInvalidYearRangeError()
	}
	return p, nil
}

// RegisterRoutes registers all TMDb routes on the given router group
func (h *TMDbHandler) RegisterRoutes(rg *gin.RouterGroup) {
	tmdbGroup := rg.Group("/tmdb")
	{
		// Search endpoints
		search := tmdbGroup.Group("/search")
		{
			search.GET("/movies", h.SearchMovies)
			search.GET("/tv", h.SearchTVShows)
		}

		// Trending endpoints (Story 10-1)
		trending := tmdbGroup.Group("/trending")
		{
			trending.GET("/movies", h.GetTrendingMovies)
			trending.GET("/tv", h.GetTrendingTVShows)
		}

		// Discover endpoints (Story 10-1)
		discover := tmdbGroup.Group("/discover")
		{
			discover.GET("/movies", h.DiscoverMovies)
			discover.GET("/tv", h.DiscoverTVShows)
		}

		// Details endpoints
		tmdbGroup.GET("/movies/:id", h.GetMovieDetails)
		tmdbGroup.GET("/tv/:id", h.GetTVShowDetails)
	}
}

// handleTMDbError handles TMDb-specific errors and returns appropriate HTTP responses
func handleTMDbError(c *gin.Context, err error, operation string, attrs ...any) {
	// Check if it's a TMDb-specific error
	if tmdbErr, ok := err.(*tmdb.TMDbError); ok {
		logAttrs := append([]any{
			"error_code", tmdbErr.Code,
			"error", tmdbErr.Message,
		}, attrs...)
		slog.Error("TMDb "+operation+" failed", logAttrs...)

		ErrorResponse(c, tmdbErr.StatusCode, tmdbErr.Code, tmdbErr.Message, tmdbErr.Suggestion)
		return
	}

	// Generic error
	logAttrs := append([]any{"error", err}, attrs...)
	slog.Error("TMDb "+operation+" failed", logAttrs...)

	ErrorResponse(c, http.StatusInternalServerError, "TMDB_INTERNAL_ERROR",
		"An unexpected error occurred while "+operation,
		"Please try again later")
}
