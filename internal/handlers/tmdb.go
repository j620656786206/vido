package handlers

import (
	"net/http"
	"strconv"

	"github.com/alexyu/vido/internal/middleware"
	"github.com/alexyu/vido/internal/tmdb"
	"github.com/gin-gonic/gin"
)

// TMDbHandler holds the TMDb client for handling requests
type TMDbHandler struct {
	client *tmdb.Client
}

// NewTMDbHandler creates a new TMDb handler
func NewTMDbHandler(client *tmdb.Client) *TMDbHandler {
	return &TMDbHandler{
		client: client,
	}
}

// SearchMoviesHandler handles movie search requests
// @Summary      Search for movies
// @Description  Search for movies by title with zh-TW localization
// @Tags         movies
// @Produce      json
// @Param        query  query     string  true   "Search query"
// @Param        page   query     int     false  "Page number (default: 1)"
// @Success      200    {object}  tmdb.SearchResultMovies
// @Failure      400    {object}  middleware.ErrorResponse
// @Failure      429    {object}  middleware.ErrorResponse
// @Failure      500    {object}  middleware.ErrorResponse
// @Router       /api/v1/movies/search [get]
func (h *TMDbHandler) SearchMovies(c *gin.Context) {
	// Get query parameter
	query := c.Query("query")
	if query == "" {
		c.Error(middleware.NewValidationError("query parameter is required"))
		return
	}

	// Get page parameter (default to 1)
	page := 1
	if pageStr := c.Query("page"); pageStr != "" {
		var err error
		page, err = strconv.Atoi(pageStr)
		if err != nil || page < 1 {
			c.Error(middleware.NewValidationError("page must be a positive integer"))
			return
		}
	}

	// Search movies using TMDb client
	result, err := h.client.SearchMovies(c.Request.Context(), query, page)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, result)
}

// GetMovieDetailsHandler handles movie details requests
// @Summary      Get movie details
// @Description  Get detailed information for a specific movie by ID with zh-TW localization
// @Tags         movies
// @Produce      json
// @Param        id   path      int  true  "Movie ID"
// @Success      200  {object}  tmdb.MovieDetails
// @Failure      400  {object}  middleware.ErrorResponse
// @Failure      404  {object}  middleware.ErrorResponse
// @Failure      429  {object}  middleware.ErrorResponse
// @Failure      500  {object}  middleware.ErrorResponse
// @Router       /api/v1/movies/{id} [get]
func (h *TMDbHandler) GetMovieDetails(c *gin.Context) {
	// Get movie ID from path parameter
	idStr := c.Param("id")
	movieID, err := strconv.Atoi(idStr)
	if err != nil || movieID <= 0 {
		c.Error(middleware.NewValidationError("invalid movie ID"))
		return
	}

	// Get movie details using TMDb client
	result, err := h.client.GetMovieDetails(c.Request.Context(), movieID)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, result)
}

// SearchTVShowsHandler handles TV show search requests
// @Summary      Search for TV shows
// @Description  Search for TV shows by name with zh-TW localization
// @Tags         tv
// @Produce      json
// @Param        query  query     string  true   "Search query"
// @Param        page   query     int     false  "Page number (default: 1)"
// @Success      200    {object}  tmdb.SearchResultTVShows
// @Failure      400    {object}  middleware.ErrorResponse
// @Failure      429    {object}  middleware.ErrorResponse
// @Failure      500    {object}  middleware.ErrorResponse
// @Router       /api/v1/tv/search [get]
func (h *TMDbHandler) SearchTVShows(c *gin.Context) {
	// Get query parameter
	query := c.Query("query")
	if query == "" {
		c.Error(middleware.NewValidationError("query parameter is required"))
		return
	}

	// Get page parameter (default to 1)
	page := 1
	if pageStr := c.Query("page"); pageStr != "" {
		var err error
		page, err = strconv.Atoi(pageStr)
		if err != nil || page < 1 {
			c.Error(middleware.NewValidationError("page must be a positive integer"))
			return
		}
	}

	// Search TV shows using TMDb client
	result, err := h.client.SearchTVShows(c.Request.Context(), query, page)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, result)
}

// GetTVShowDetailsHandler handles TV show details requests
// @Summary      Get TV show details
// @Description  Get detailed information for a specific TV show by ID with zh-TW localization
// @Tags         tv
// @Produce      json
// @Param        id   path      int  true  "TV show ID"
// @Success      200  {object}  tmdb.TVShowDetails
// @Failure      400  {object}  middleware.ErrorResponse
// @Failure      404  {object}  middleware.ErrorResponse
// @Failure      429  {object}  middleware.ErrorResponse
// @Failure      500  {object}  middleware.ErrorResponse
// @Router       /api/v1/tv/{id} [get]
func (h *TMDbHandler) GetTVShowDetails(c *gin.Context) {
	// Get TV show ID from path parameter
	idStr := c.Param("id")
	tvID, err := strconv.Atoi(idStr)
	if err != nil || tvID <= 0 {
		c.Error(middleware.NewValidationError("invalid TV show ID"))
		return
	}

	// Get TV show details using TMDb client
	result, err := h.client.GetTVShowDetails(c.Request.Context(), tvID)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, result)
}
