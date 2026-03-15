package handlers

import (
	"log/slog"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/vido/api/internal/services"
)

// LibraryHandler handles HTTP requests for library operations.
type LibraryHandler struct {
	service services.LibraryServiceInterface
}

// NewLibraryHandler creates a new LibraryHandler with the given service.
func NewLibraryHandler(service services.LibraryServiceInterface) *LibraryHandler {
	return &LibraryHandler{service: service}
}

// ListLibrary handles GET /api/v1/library
// Returns a paginated list of library items (movies + series combined)
// Supports filters: genre, year_min, year_max via query params
func (h *LibraryHandler) ListLibrary(c *gin.Context) {
	params := parseListParams(c)

	// Parse type filter: all (default), movie, tv
	mediaType := c.DefaultQuery("type", "all")
	if mediaType != "all" && mediaType != "movie" && mediaType != "tv" {
		BadRequestError(c, "VALIDATION_INVALID_FORMAT", "type must be 'all', 'movie', or 'tv'")
		return
	}

	// Parse filter params
	if genres := c.Query("genres"); genres != "" {
		genreList := strings.Split(genres, ",")
		cleaned := make([]string, 0, len(genreList))
		for _, g := range genreList {
			g = strings.TrimSpace(g)
			if g != "" {
				cleaned = append(cleaned, g)
			}
		}
		if len(cleaned) > 0 {
			params.Filters["genres"] = cleaned
		}
	}
	if yearMin := c.Query("year_min"); yearMin != "" {
		if _, err := strconv.Atoi(yearMin); err != nil {
			BadRequestError(c, "VALIDATION_INVALID_FORMAT", "year_min must be a valid year number")
			return
		}
		params.Filters["year_min"] = yearMin
	}
	if yearMax := c.Query("year_max"); yearMax != "" {
		if _, err := strconv.Atoi(yearMax); err != nil {
			BadRequestError(c, "VALIDATION_INVALID_FORMAT", "year_max must be a valid year number")
			return
		}
		params.Filters["year_max"] = yearMax
	}

	result, err := h.service.ListLibrary(c.Request.Context(), params, mediaType)
	if err != nil {
		slog.Error("Failed to list library", "error", err, "type", mediaType)
		InternalServerError(c, "Failed to retrieve library")
		return
	}

	SuccessResponse(c, PaginatedResponse{
		Items:      result.Items,
		Page:       result.Pagination.Page,
		PageSize:   result.Pagination.PageSize,
		TotalItems: result.Pagination.TotalResults,
		TotalPages: result.Pagination.TotalPages,
	})
}

// GetRecentlyAdded handles GET /api/v1/library/recent
// Returns the most recently added media items sorted by created_at DESC.
func (h *LibraryHandler) GetRecentlyAdded(c *gin.Context) {
	limit := 20
	if limitStr := c.Query("limit"); limitStr != "" {
		parsed, err := strconv.Atoi(limitStr)
		if err != nil || parsed < 1 || parsed > 100 {
			BadRequestError(c, "VALIDATION_INVALID_FORMAT", "limit must be a number between 1 and 100")
			return
		}
		limit = parsed
	}

	result, err := h.service.GetRecentlyAdded(c.Request.Context(), limit)
	if err != nil {
		slog.Error("Failed to get recently added", "error", err)
		InternalServerError(c, "Failed to retrieve recently added items")
		return
	}

	SuccessResponse(c, PaginatedResponse{
		Items:      result.Items,
		Page:       result.Pagination.Page,
		PageSize:   result.Pagination.PageSize,
		TotalItems: result.Pagination.TotalResults,
		TotalPages: result.Pagination.TotalPages,
	})
}

// DeleteMovie handles DELETE /api/v1/library/movies/:id
func (h *LibraryHandler) DeleteMovie(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		BadRequestError(c, "VALIDATION_REQUIRED_FIELD", "Movie ID is required")
		return
	}

	if err := h.service.DeleteMovie(c.Request.Context(), id); err != nil {
		slog.Error("Failed to delete movie", "error", err, "movie_id", id)
		InternalServerError(c, "Failed to delete movie")
		return
	}

	NoContentResponse(c)
}

// DeleteSeries handles DELETE /api/v1/library/series/:id
func (h *LibraryHandler) DeleteSeries(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		BadRequestError(c, "VALIDATION_REQUIRED_FIELD", "Series ID is required")
		return
	}

	if err := h.service.DeleteSeries(c.Request.Context(), id); err != nil {
		slog.Error("Failed to delete series", "error", err, "series_id", id)
		InternalServerError(c, "Failed to delete series")
		return
	}

	NoContentResponse(c)
}

// ReparseMovie handles POST /api/v1/library/movies/:id/reparse
func (h *LibraryHandler) ReparseMovie(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		BadRequestError(c, "VALIDATION_REQUIRED_FIELD", "Movie ID is required")
		return
	}

	// Verify the movie exists
	movie, err := h.service.GetMovieByID(c.Request.Context(), id)
	if err != nil {
		NotFoundError(c, "Movie")
		return
	}

	// TODO: Trigger re-parse via metadata service (Story 5.6)
	SuccessResponse(c, map[string]interface{}{
		"id":     movie.ID,
		"status": "reparse_queued",
	})
}

// ReparseSeries handles POST /api/v1/library/series/:id/reparse
func (h *LibraryHandler) ReparseSeries(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		BadRequestError(c, "VALIDATION_REQUIRED_FIELD", "Series ID is required")
		return
	}

	series, err := h.service.GetSeriesByID(c.Request.Context(), id)
	if err != nil {
		NotFoundError(c, "Series")
		return
	}

	// TODO: Trigger re-parse via metadata service (Story 5.6)
	SuccessResponse(c, map[string]interface{}{
		"id":     series.ID,
		"status": "reparse_queued",
	})
}

// ExportMovie handles POST /api/v1/library/movies/:id/export
func (h *LibraryHandler) ExportMovie(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		BadRequestError(c, "VALIDATION_REQUIRED_FIELD", "Movie ID is required")
		return
	}

	movie, err := h.service.GetMovieByID(c.Request.Context(), id)
	if err != nil {
		NotFoundError(c, "Movie")
		return
	}

	// Return movie metadata as exportable JSON
	SuccessResponse(c, movie)
}

// ExportSeries handles POST /api/v1/library/series/:id/export
func (h *LibraryHandler) ExportSeries(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		BadRequestError(c, "VALIDATION_REQUIRED_FIELD", "Series ID is required")
		return
	}

	series, err := h.service.GetSeriesByID(c.Request.Context(), id)
	if err != nil {
		NotFoundError(c, "Series")
		return
	}

	// Return series metadata as exportable JSON
	SuccessResponse(c, series)
}

// GetGenres handles GET /api/v1/library/genres
// Returns distinct genres across all library items
func (h *LibraryHandler) GetGenres(c *gin.Context) {
	genres, err := h.service.GetDistinctGenres(c.Request.Context())
	if err != nil {
		slog.Error("Failed to get genres", "error", err)
		InternalServerError(c, "Failed to retrieve genres")
		return
	}

	SuccessResponse(c, genres)
}

// GetStats handles GET /api/v1/library/stats
// Returns library statistics including year range and counts
func (h *LibraryHandler) GetStats(c *gin.Context) {
	stats, err := h.service.GetLibraryStats(c.Request.Context())
	if err != nil {
		slog.Error("Failed to get library stats", "error", err)
		InternalServerError(c, "Failed to retrieve library stats")
		return
	}

	SuccessResponse(c, stats)
}

// SearchLibrary handles GET /api/v1/library/search?q=X&page=1&page_size=20&type=all
// Performs FTS5 full-text search across movies and series in the library.
func (h *LibraryHandler) SearchLibrary(c *gin.Context) {
	query := c.Query("q")
	if len(query) < 2 {
		BadRequestError(c, "VALIDATION_REQUIRED_FIELD", "Search query (q) must be at least 2 characters")
		return
	}

	// Parse type filter: all (default), movie, tv
	mediaType := c.DefaultQuery("type", "all")
	if mediaType != "all" && mediaType != "movie" && mediaType != "tv" {
		BadRequestError(c, "VALIDATION_INVALID_FORMAT", "type must be 'all', 'movie', or 'tv'")
		return
	}

	params := parseListParams(c)

	result, err := h.service.SearchLibrary(c.Request.Context(), query, params, mediaType)
	if err != nil {
		slog.Error("Failed to search library", "error", err, "query", query, "type", mediaType)
		InternalServerError(c, "Failed to search library")
		return
	}

	SuccessResponse(c, result)
}

// RegisterRoutes registers all library routes on the given router group
func (h *LibraryHandler) RegisterRoutes(rg *gin.RouterGroup) {
	library := rg.Group("/library")
	{
		library.GET("", h.ListLibrary)
		library.GET("/search", h.SearchLibrary)
		library.GET("/recent", h.GetRecentlyAdded)
		library.GET("/genres", h.GetGenres)
		library.GET("/stats", h.GetStats)

		movies := library.Group("/movies")
		{
			movies.POST("/:id/reparse", h.ReparseMovie)
			movies.POST("/:id/export", h.ExportMovie)
			movies.DELETE("/:id", h.DeleteMovie)
		}

		series := library.Group("/series")
		{
			series.POST("/:id/reparse", h.ReparseSeries)
			series.POST("/:id/export", h.ExportSeries)
			series.DELETE("/:id", h.DeleteSeries)
		}
	}
}
