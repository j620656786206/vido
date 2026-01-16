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
