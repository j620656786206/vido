package handlers

import (
	"context"
	"log/slog"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/repository"
)

// MovieServiceInterface defines the contract for movie business operations.
// This interface enables testing handlers with mock services.
type MovieServiceInterface interface {
	Create(ctx context.Context, movie *models.Movie) error
	GetByID(ctx context.Context, id string) (*models.Movie, error)
	GetByTMDbID(ctx context.Context, tmdbID int64) (*models.Movie, error)
	Update(ctx context.Context, movie *models.Movie) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, params repository.ListParams) ([]models.Movie, *repository.PaginationResult, error)
	SearchByTitle(ctx context.Context, title string, params repository.ListParams) ([]models.Movie, *repository.PaginationResult, error)
}

// MovieHandler handles HTTP requests for movie operations.
// It uses MovieServiceInterface for business logic, following the
// Handler → Service → Repository → Database architecture.
type MovieHandler struct {
	service MovieServiceInterface
}

// NewMovieHandler creates a new MovieHandler with the given service.
func NewMovieHandler(service MovieServiceInterface) *MovieHandler {
	return &MovieHandler{
		service: service,
	}
}

// CreateMovieRequest represents the request body for creating a movie
type CreateMovieRequest struct {
	Title         string   `json:"title" binding:"required"`
	OriginalTitle string   `json:"originalTitle,omitempty"`
	ReleaseDate   string   `json:"releaseDate" binding:"required"`
	Genres        []string `json:"genres,omitempty"`
	Overview      string   `json:"overview,omitempty"`
	PosterPath    string   `json:"posterPath,omitempty"`
	TMDbID        int64    `json:"tmdbId,omitempty"`
	IMDbID        string   `json:"imdbId,omitempty"`
}

// UpdateMovieRequest represents the request body for updating a movie
type UpdateMovieRequest struct {
	Title         string   `json:"title,omitempty"`
	OriginalTitle string   `json:"originalTitle,omitempty"`
	ReleaseDate   string   `json:"releaseDate,omitempty"`
	Genres        []string `json:"genres,omitempty"`
	Overview      string   `json:"overview,omitempty"`
	PosterPath    string   `json:"posterPath,omitempty"`
	Rating        float64  `json:"rating,omitempty"`
	Runtime       int64    `json:"runtime,omitempty"`
	Status        string   `json:"status,omitempty"`
}

// List handles GET /api/v1/movies
// Returns a paginated list of movies
func (h *MovieHandler) List(c *gin.Context) {
	params := parseListParams(c)

	movies, pagination, err := h.service.List(c.Request.Context(), params)
	if err != nil {
		slog.Error("Failed to list movies", "error", err)
		InternalServerError(c, "Failed to retrieve movies")
		return
	}

	SuccessResponse(c, PaginatedResponse{
		Items:      movies,
		Page:       pagination.Page,
		PageSize:   pagination.PageSize,
		TotalItems: pagination.TotalResults,
		TotalPages: pagination.TotalPages,
	})
}

// GetByID handles GET /api/v1/movies/:id
// Returns a single movie by ID
func (h *MovieHandler) GetByID(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		BadRequestError(c, "VALIDATION_REQUIRED_FIELD", "Movie ID is required")
		return
	}

	movie, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		slog.Error("Failed to get movie", "error", err, "movie_id", id)
		NotFoundError(c, "Movie")
		return
	}

	SuccessResponse(c, movie)
}

// Create handles POST /api/v1/movies
// Creates a new movie
func (h *MovieHandler) Create(c *gin.Context) {
	var req CreateMovieRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ValidationError(c, "Invalid request body: "+err.Error())
		return
	}

	movie := &models.Movie{
		Title:       req.Title,
		ReleaseDate: req.ReleaseDate,
		Genres:      req.Genres,
	}

	// Set optional fields
	if req.OriginalTitle != "" {
		movie.OriginalTitle.String = req.OriginalTitle
		movie.OriginalTitle.Valid = true
	}
	if req.Overview != "" {
		movie.Overview.String = req.Overview
		movie.Overview.Valid = true
	}
	if req.PosterPath != "" {
		movie.PosterPath.String = req.PosterPath
		movie.PosterPath.Valid = true
	}
	if req.TMDbID != 0 {
		movie.TMDbID.Int64 = req.TMDbID
		movie.TMDbID.Valid = true
	}
	if req.IMDbID != "" {
		movie.IMDbID.String = req.IMDbID
		movie.IMDbID.Valid = true
	}

	if err := h.service.Create(c.Request.Context(), movie); err != nil {
		slog.Error("Failed to create movie", "error", err, "title", req.Title)
		InternalServerError(c, "Failed to create movie")
		return
	}

	CreatedResponse(c, movie)
}

// Update handles PUT /api/v1/movies/:id
// Updates an existing movie
func (h *MovieHandler) Update(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		BadRequestError(c, "VALIDATION_REQUIRED_FIELD", "Movie ID is required")
		return
	}

	// Get existing movie
	movie, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		NotFoundError(c, "Movie")
		return
	}

	var req UpdateMovieRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ValidationError(c, "Invalid request body: "+err.Error())
		return
	}

	// Update fields if provided
	if req.Title != "" {
		movie.Title = req.Title
	}
	if req.OriginalTitle != "" {
		movie.OriginalTitle.String = req.OriginalTitle
		movie.OriginalTitle.Valid = true
	}
	if req.ReleaseDate != "" {
		movie.ReleaseDate = req.ReleaseDate
	}
	if req.Genres != nil {
		movie.Genres = req.Genres
	}
	if req.Overview != "" {
		movie.Overview.String = req.Overview
		movie.Overview.Valid = true
	}
	if req.PosterPath != "" {
		movie.PosterPath.String = req.PosterPath
		movie.PosterPath.Valid = true
	}
	if req.Rating != 0 {
		movie.Rating.Float64 = req.Rating
		movie.Rating.Valid = true
	}
	if req.Runtime != 0 {
		movie.Runtime.Int64 = req.Runtime
		movie.Runtime.Valid = true
	}
	if req.Status != "" {
		movie.Status.String = req.Status
		movie.Status.Valid = true
	}

	if err := h.service.Update(c.Request.Context(), movie); err != nil {
		slog.Error("Failed to update movie", "error", err, "movie_id", id)
		InternalServerError(c, "Failed to update movie")
		return
	}

	SuccessResponse(c, movie)
}

// Delete handles DELETE /api/v1/movies/:id
// Deletes a movie by ID
func (h *MovieHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		BadRequestError(c, "VALIDATION_REQUIRED_FIELD", "Movie ID is required")
		return
	}

	if err := h.service.Delete(c.Request.Context(), id); err != nil {
		slog.Error("Failed to delete movie", "error", err, "movie_id", id)
		InternalServerError(c, "Failed to delete movie")
		return
	}

	NoContentResponse(c)
}

// Search handles GET /api/v1/movies/search
// Searches movies by title
func (h *MovieHandler) Search(c *gin.Context) {
	query := c.Query("q")
	if query == "" {
		BadRequestError(c, "VALIDATION_REQUIRED_FIELD", "Search query 'q' is required")
		return
	}

	params := parseListParams(c)

	movies, pagination, err := h.service.SearchByTitle(c.Request.Context(), query, params)
	if err != nil {
		slog.Error("Failed to search movies", "error", err, "query", query)
		InternalServerError(c, "Failed to search movies")
		return
	}

	SuccessResponse(c, PaginatedResponse{
		Items:      movies,
		Page:       pagination.Page,
		PageSize:   pagination.PageSize,
		TotalItems: pagination.TotalResults,
		TotalPages: pagination.TotalPages,
	})
}

// RegisterRoutes registers all movie routes on the given router group
func (h *MovieHandler) RegisterRoutes(rg *gin.RouterGroup) {
	movies := rg.Group("/movies")
	{
		movies.GET("", h.List)
		movies.GET("/search", h.Search)
		movies.GET("/:id", h.GetByID)
		movies.POST("", h.Create)
		movies.PUT("/:id", h.Update)
		movies.DELETE("/:id", h.Delete)
	}
}

// parseListParams extracts pagination parameters from query string
func parseListParams(c *gin.Context) repository.ListParams {
	params := repository.NewListParams()

	if page := c.Query("page"); page != "" {
		if p, err := strconv.Atoi(page); err == nil && p > 0 {
			params.Page = p
		}
	}

	if pageSize := c.Query("page_size"); pageSize != "" {
		if ps, err := strconv.Atoi(pageSize); err == nil && ps > 0 && ps <= 100 {
			params.PageSize = ps
		}
	}

	if sortBy := c.Query("sort_by"); sortBy != "" {
		params.SortBy = sortBy
	}

	if sortOrder := c.Query("sort_order"); sortOrder != "" {
		params.SortOrder = sortOrder
	}

	return params
}
