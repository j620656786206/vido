package handlers

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/vido/api/internal/services"
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
	GetMovieVideos(ctx context.Context, movieID int) (*tmdb.VideosResponse, error)
	GetTVShowVideos(ctx context.Context, tvID int) (*tmdb.VideosResponse, error)
	GetWatchProviders(ctx context.Context, mediaType string, id int, region string) (*tmdb.WatchProvidersResponse, error)
}

// RecommendationServiceInterface defines the contract for related-content lookups
// (Story 12-3). Kept separate from TMDbServiceInterface because it owns the
// cross-domain TMDb + ownership-repo join.
type RecommendationServiceInterface interface {
	GetMovieRecommendations(ctx context.Context, tmdbID int) (*services.RecommendationResult, error)
	GetTVRecommendations(ctx context.Context, tmdbID int) (*services.RecommendationResult, error)
}

// TMDbHandler handles HTTP requests for TMDb operations.
// It uses TMDbServiceInterface for business logic, following the
// Handler → Service → Repository → Database architecture.
type TMDbHandler struct {
	service     TMDbServiceInterface
	recsService RecommendationServiceInterface
}

// NewTMDbHandler creates a new TMDbHandler with the given service.
func NewTMDbHandler(service TMDbServiceInterface) *TMDbHandler {
	return &TMDbHandler{
		service: service,
	}
}

// SetRecommendationService wires the related-content service (Story 12-3).
// Optional dependency: the recommendations routes require it, but the rest of
// the handler functions without it (mirrors TMDbService.SetContentFilter).
func (h *TMDbHandler) SetRecommendationService(s RecommendationServiceInterface) {
	h.recsService = s
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
// @Summary Discover movies on TMDb
// @Description Discover movies using TMDb Discover API with zh-TW language priority
// @Tags tmdb
// @Accept json
// @Produce json
// @Param genre query string false "Comma-separated genre IDs"
// @Param year_gte query int false "Minimum release year (0 = unlimited)"
// @Param year_lte query int false "Maximum release year (0 = unlimited)"
// @Param region query string false "ISO 3166-1 region code"
// @Param vote_gte query number false "Minimum TMDb rating 0-10 (0 = unlimited)"
// @Param vote_lte query number false "Maximum TMDb rating 0-10 (0 = unlimited)"
// @Param watch_providers query string false "Comma-separated TMDb watch-provider IDs (e.g. 8 for Netflix)"
// @Param watch_region query string false "ISO 3166-1 watch region (defaults to region, then TW)"
// @Param language query string false "BCP 47 language code"
// @Param sort query string false "TMDb-native sort key (e.g. popularity.desc, vote_average.desc); local-only keys like date_added are rejected with 400 TMDB_UNSUPPORTED_SORT"
// @Param page query int false "Page number" default(1)
// @Success 200 {object} APIResponse{data=tmdb.SearchResultMovies}
// @Failure 400 {object} APIResponse{error=APIError} "TMDB_INVALID_YEAR_RANGE (year_gte > year_lte, Story 10-1a), TMDB_INVALID_VOTE_RANGE (vote_gte > vote_lte), or TMDB_UNSUPPORTED_SORT (local-only sort key e.g. date_added) — Story 11-1"
// @Failure 500 {object} APIResponse{error=APIError}
// @Router /api/v1/tmdb/discover/movies [get]
func (h *TMDbHandler) DiscoverMovies(c *gin.Context) {
	params, err := parseDiscoverParams(c)
	if err != nil {
		handleValidationError(c, err, "discover movies", slog.Any("params", params))
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
// @Summary Discover TV shows on TMDb
// @Description Discover TV shows using TMDb Discover API with zh-TW language priority
// @Tags tmdb
// @Accept json
// @Produce json
// @Param genre query string false "Comma-separated genre IDs"
// @Param year_gte query int false "Minimum first-air year (0 = unlimited)"
// @Param year_lte query int false "Maximum first-air year (0 = unlimited)"
// @Param region query string false "ISO 3166-1 region code"
// @Param vote_gte query number false "Minimum TMDb rating 0-10 (0 = unlimited)"
// @Param vote_lte query number false "Maximum TMDb rating 0-10 (0 = unlimited)"
// @Param watch_providers query string false "Comma-separated TMDb watch-provider IDs (e.g. 8 for Netflix)"
// @Param watch_region query string false "ISO 3166-1 watch region (defaults to region, then TW)"
// @Param language query string false "BCP 47 language code"
// @Param sort query string false "TMDb-native sort key (e.g. popularity.desc, vote_average.desc); local-only keys like date_added are rejected with 400 TMDB_UNSUPPORTED_SORT"
// @Param page query int false "Page number" default(1)
// @Success 200 {object} APIResponse{data=tmdb.SearchResultTVShows}
// @Failure 400 {object} APIResponse{error=APIError} "TMDB_INVALID_YEAR_RANGE (year_gte > year_lte, Story 10-1a), TMDB_INVALID_VOTE_RANGE (vote_gte > vote_lte), or TMDB_UNSUPPORTED_SORT (local-only sort key e.g. date_added) — Story 11-1"
// @Failure 500 {object} APIResponse{error=APIError}
// @Router /api/v1/tmdb/discover/tv [get]
func (h *TMDbHandler) DiscoverTVShows(c *gin.Context) {
	params, err := parseDiscoverParams(c)
	if err != nil {
		handleValidationError(c, err, "discover TV shows", slog.Any("params", params))
		return
	}
	result, err := h.service.DiscoverTVShows(c.Request.Context(), params)
	if err != nil {
		handleTMDbError(c, err, "discover TV shows", slog.Any("params", params))
		return
	}
	SuccessResponse(c, result)
}

// GetMovieVideos handles GET /api/v1/tmdb/movies/:id/videos.
// Returns trailers/teasers from TMDb for a movie (Story 10-2 AC #6).
// @Summary Get movie videos from TMDb
// @Description Retrieve trailers, teasers, and clips for a movie via TMDb videos endpoint
// @Tags tmdb
// @Accept json
// @Produce json
// @Param id path int true "TMDb movie ID"
// @Success 200 {object} APIResponse{data=tmdb.VideosResponse}
// @Failure 400 {object} APIResponse{error=APIError}
// @Failure 404 {object} APIResponse{error=APIError}
// @Failure 500 {object} APIResponse{error=APIError}
// @Router /api/v1/tmdb/movies/{id}/videos [get]
func (h *TMDbHandler) GetMovieVideos(c *gin.Context) {
	movieID, err := strconv.Atoi(c.Param("id"))
	if err != nil || movieID <= 0 {
		ErrorResponse(c, http.StatusBadRequest, tmdb.ErrCodeBadRequest,
			"Invalid movie ID",
			"Movie ID must be a positive integer")
		return
	}
	result, err := h.service.GetMovieVideos(c.Request.Context(), movieID)
	if err != nil {
		handleTMDbError(c, err, "get movie videos", slog.Int("movie_id", movieID))
		return
	}
	SuccessResponse(c, result)
}

// GetTVShowVideos handles GET /api/v1/tmdb/tv/:id/videos.
// @Summary Get TV show videos from TMDb
// @Description Retrieve trailers, teasers, and clips for a TV show via TMDb videos endpoint
// @Tags tmdb
// @Accept json
// @Produce json
// @Param id path int true "TMDb TV show ID"
// @Success 200 {object} APIResponse{data=tmdb.VideosResponse}
// @Failure 400 {object} APIResponse{error=APIError}
// @Failure 404 {object} APIResponse{error=APIError}
// @Failure 500 {object} APIResponse{error=APIError}
// @Router /api/v1/tmdb/tv/{id}/videos [get]
func (h *TMDbHandler) GetTVShowVideos(c *gin.Context) {
	tvID, err := strconv.Atoi(c.Param("id"))
	if err != nil || tvID <= 0 {
		ErrorResponse(c, http.StatusBadRequest, tmdb.ErrCodeBadRequest,
			"Invalid TV show ID",
			"TV show ID must be a positive integer")
		return
	}
	result, err := h.service.GetTVShowVideos(c.Request.Context(), tvID)
	if err != nil {
		handleTMDbError(c, err, "get TV show videos", slog.Int("tv_id", tvID))
		return
	}
	SuccessResponse(c, result)
}

// GetMovieRecommendations handles GET /api/v1/tmdb/movies/:id/recommendations.
// Returns related movies (TMDb /recommendations, falling back to /similar) with
// an "已有" ownership flag per tile (Story 12-3 AC #1, #3, #4).
// @Summary Get related movies from TMDb
// @Description Retrieve recommended/similar movies for a movie, annotated with local-library ownership
// @Tags tmdb
// @Accept json
// @Produce json
// @Param id path int true "TMDb movie ID"
// @Success 200 {object} APIResponse{data=services.RecommendationResult}
// @Failure 400 {object} APIResponse{error=APIError}
// @Failure 404 {object} APIResponse{error=APIError}
// @Failure 500 {object} APIResponse{error=APIError}
// @Router /api/v1/tmdb/movies/{id}/recommendations [get]
func (h *TMDbHandler) GetMovieRecommendations(c *gin.Context) {
	movieID, err := strconv.Atoi(c.Param("id"))
	if err != nil || movieID <= 0 {
		ErrorResponse(c, http.StatusBadRequest, tmdb.ErrCodeBadRequest,
			"Invalid movie ID",
			"Movie ID must be a positive integer")
		return
	}
	if h.recsService == nil {
		ErrorResponse(c, http.StatusInternalServerError, tmdb.ErrCodeServerError,
			"Recommendations unavailable",
			"The recommendation service is not configured")
		return
	}
	result, err := h.recsService.GetMovieRecommendations(c.Request.Context(), movieID)
	if err != nil {
		handleTMDbError(c, err, "get movie recommendations", slog.Int("movie_id", movieID))
		return
	}
	SuccessResponse(c, result)
}

// GetTVRecommendations handles GET /api/v1/tmdb/tv/:id/recommendations.
// @Summary Get related TV shows from TMDb
// @Description Retrieve recommended/similar TV shows for a TV show, annotated with local-library ownership
// @Tags tmdb
// @Accept json
// @Produce json
// @Param id path int true "TMDb TV show ID"
// @Success 200 {object} APIResponse{data=services.RecommendationResult}
// @Failure 400 {object} APIResponse{error=APIError}
// @Failure 404 {object} APIResponse{error=APIError}
// @Failure 500 {object} APIResponse{error=APIError}
// @Router /api/v1/tmdb/tv/{id}/recommendations [get]
func (h *TMDbHandler) GetTVRecommendations(c *gin.Context) {
	tvID, err := strconv.Atoi(c.Param("id"))
	if err != nil || tvID <= 0 {
		ErrorResponse(c, http.StatusBadRequest, tmdb.ErrCodeBadRequest,
			"Invalid TV show ID",
			"TV show ID must be a positive integer")
		return
	}
	if h.recsService == nil {
		ErrorResponse(c, http.StatusInternalServerError, tmdb.ErrCodeServerError,
			"Recommendations unavailable",
			"The recommendation service is not configured")
		return
	}
	result, err := h.recsService.GetTVRecommendations(c.Request.Context(), tvID)
	if err != nil {
		handleTMDbError(c, err, "get TV recommendations", slog.Int("tv_id", tvID))
		return
	}
	SuccessResponse(c, result)
}

// GetMovieWatchProviders handles GET /api/v1/tmdb/movies/:id/watch/providers.
// Returns the streaming/rent/buy providers for a movie in a region (default TW),
// sourced from TMDb's JustWatch-backed watch/providers endpoint (Story 12-4).
// @Summary Get movie streaming-platform availability from TMDb
// @Description Retrieve watch providers (flatrate/rent/buy) for a movie, filtered to a region (default TW). Data sourced from JustWatch via TMDb.
// @Tags tmdb
// @Accept json
// @Produce json
// @Param id path int true "TMDb movie ID"
// @Param region query string false "ISO 3166-1 region code" default(TW)
// @Success 200 {object} APIResponse{data=tmdb.WatchProvidersResponse}
// @Failure 400 {object} APIResponse{error=APIError}
// @Failure 404 {object} APIResponse{error=APIError}
// @Failure 500 {object} APIResponse{error=APIError}
// @Router /api/v1/tmdb/movies/{id}/watch/providers [get]
func (h *TMDbHandler) GetMovieWatchProviders(c *gin.Context) {
	movieID, err := strconv.Atoi(c.Param("id"))
	if err != nil || movieID <= 0 {
		ErrorResponse(c, http.StatusBadRequest, tmdb.ErrCodeBadRequest,
			"Invalid movie ID",
			"Movie ID must be a positive integer")
		return
	}
	region := c.Query("region")
	result, err := h.service.GetWatchProviders(c.Request.Context(), "movie", movieID, region)
	if err != nil {
		handleTMDbError(c, err, "get movie watch providers", slog.Int("movie_id", movieID), slog.String("region", region))
		return
	}
	SuccessResponse(c, result)
}

// GetTVWatchProviders handles GET /api/v1/tmdb/tv/:id/watch/providers.
// @Summary Get TV streaming-platform availability from TMDb
// @Description Retrieve watch providers (flatrate/rent/buy) for a TV show, filtered to a region (default TW). Data sourced from JustWatch via TMDb.
// @Tags tmdb
// @Accept json
// @Produce json
// @Param id path int true "TMDb TV show ID"
// @Param region query string false "ISO 3166-1 region code" default(TW)
// @Success 200 {object} APIResponse{data=tmdb.WatchProvidersResponse}
// @Failure 400 {object} APIResponse{error=APIError}
// @Failure 404 {object} APIResponse{error=APIError}
// @Failure 500 {object} APIResponse{error=APIError}
// @Router /api/v1/tmdb/tv/{id}/watch/providers [get]
func (h *TMDbHandler) GetTVWatchProviders(c *gin.Context) {
	tvID, err := strconv.Atoi(c.Param("id"))
	if err != nil || tvID <= 0 {
		ErrorResponse(c, http.StatusBadRequest, tmdb.ErrCodeBadRequest,
			"Invalid TV show ID",
			"TV show ID must be a positive integer")
		return
	}
	region := c.Query("region")
	result, err := h.service.GetWatchProviders(c.Request.Context(), "tv", tvID, region)
	if err != nil {
		handleTMDbError(c, err, "get TV watch providers", slog.Int("tv_id", tvID), slog.String("region", region))
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
//   - genre           → DiscoverParams.GenreIDs (comma-separated IDs, e.g. "28,12")
//   - year_gte        → DiscoverParams.YearGte
//   - year_lte        → DiscoverParams.YearLte
//   - region          → DiscoverParams.Region
//   - vote_gte        → DiscoverParams.VoteAverageGte (min rating, 0–10)
//   - vote_lte        → DiscoverParams.VoteAverageLte (max rating, 0–10)
//   - watch_providers → DiscoverParams.WatchProviders (comma-separated provider IDs)
//   - watch_region    → DiscoverParams.WatchRegion (defaults to region, then TW)
//   - language        → DiscoverParams.Language
//   - sort            → DiscoverParams.SortBy
//   - page            → DiscoverParams.Page
//
// Returns a non-nil *tmdb.TMDbError when both year bounds are non-zero
// and year_gte > year_lte (Story 10-1a), or when both vote bounds are
// non-zero and vote_gte > vote_lte (Story 11-1). Zero values for either
// bound retain the "unlimited/unbounded" semantics and skip validation.
func parseDiscoverParams(c *gin.Context) (tmdb.DiscoverParams, error) {
	p := tmdb.DiscoverParams{
		GenreIDs:       tmdb.ParseIntCSV(c.Query("genre")),
		Region:         c.Query("region"),
		WatchProviders: tmdb.ParseIntCSV(c.Query("watch_providers")),
		WatchRegion:    c.Query("watch_region"),
		Language:       c.Query("language"),
		SortBy:         c.Query("sort"),
		Page:           parsePageQuery(c.Query("page")),
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
	if v := c.Query("vote_gte"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil && f > 0 {
			p.VoteAverageGte = f
		}
	}
	if v := c.Query("vote_lte"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil && f > 0 {
			p.VoteAverageLte = f
		}
	}
	if p.YearGte > 0 && p.YearLte > 0 && p.YearGte > p.YearLte {
		return p, tmdb.NewInvalidYearRangeError()
	}
	if p.VoteAverageGte > 0 && p.VoteAverageLte > 0 && p.VoteAverageGte > p.VoteAverageLte {
		return p, tmdb.NewInvalidVoteRangeError()
	}
	if tmdb.IsLocalSortKey(p.SortBy) {
		// date_added and friends are local-library sorts the discover endpoint
		// cannot honor — reject explicitly rather than silently ignoring the key
		// (Story 11-1 AC #3; real date-added ordering lives in Story 5-4).
		return p, tmdb.NewUnsupportedSortError(p.SortBy)
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

		// Videos endpoints (Story 10-2 AC #6)
		tmdbGroup.GET("/movies/:id/videos", h.GetMovieVideos)
		tmdbGroup.GET("/tv/:id/videos", h.GetTVShowVideos)

		// Recommendations endpoints (Story 12-3 — related content)
		tmdbGroup.GET("/movies/:id/recommendations", h.GetMovieRecommendations)
		tmdbGroup.GET("/tv/:id/recommendations", h.GetTVRecommendations)

		// Watch-providers endpoints (Story 12-4 — streaming-platform availability)
		tmdbGroup.GET("/movies/:id/watch/providers", h.GetMovieWatchProviders)
		tmdbGroup.GET("/tv/:id/watch/providers", h.GetTVWatchProviders)
	}
}

// handleValidationError handles request-parsing errors produced inside the
// handler layer (validation only — TMDb was never called). Logged at WARN
// level with a non-"TMDb ... failed" message so log-based TMDb failure
// alerts stay accurate (Story 10-1a code-review M2). The HTTP response
// mirrors handleTMDbError so AC #5's ApiResponse<T> envelope is preserved.
func handleValidationError(c *gin.Context, err error, operation string, attrs ...any) {
	if tmdbErr, ok := err.(*tmdb.TMDbError); ok {
		logAttrs := append([]any{
			"error_code", tmdbErr.Code,
			"error", tmdbErr.Message,
		}, attrs...)
		slog.Warn(operation+" request validation failed", logAttrs...)

		ErrorResponse(c, tmdbErr.StatusCode, tmdbErr.Code, tmdbErr.Message, tmdbErr.Suggestion)
		return
	}

	// Defensive fallback — parseDiscoverParams currently only returns *tmdb.TMDbError.
	logAttrs := append([]any{"error", err}, attrs...)
	slog.Warn(operation+" request validation failed", logAttrs...)

	ErrorResponse(c, http.StatusBadRequest, tmdb.ErrCodeBadRequest, err.Error(), "Please check your request parameters")
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
