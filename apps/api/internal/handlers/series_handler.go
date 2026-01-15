package handlers

import (
	"context"
	"log/slog"

	"github.com/gin-gonic/gin"
	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/repository"
)

// SeriesServiceInterface defines the contract for series business operations.
// This interface enables testing handlers with mock services.
type SeriesServiceInterface interface {
	Create(ctx context.Context, series *models.Series) error
	GetByID(ctx context.Context, id string) (*models.Series, error)
	GetByTMDbID(ctx context.Context, tmdbID int64) (*models.Series, error)
	Update(ctx context.Context, series *models.Series) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, params repository.ListParams) ([]models.Series, *repository.PaginationResult, error)
	SearchByTitle(ctx context.Context, title string, params repository.ListParams) ([]models.Series, *repository.PaginationResult, error)
}

// SeriesHandler handles HTTP requests for TV series operations.
// It uses SeriesServiceInterface for business logic, following the
// Handler → Service → Repository → Database architecture.
type SeriesHandler struct {
	service SeriesServiceInterface
}

// NewSeriesHandler creates a new SeriesHandler with the given service.
func NewSeriesHandler(service SeriesServiceInterface) *SeriesHandler {
	return &SeriesHandler{
		service: service,
	}
}

// CreateSeriesRequest represents the request body for creating a series
type CreateSeriesRequest struct {
	Title            string   `json:"title" binding:"required"`
	OriginalTitle    string   `json:"originalTitle,omitempty"`
	FirstAirDate     string   `json:"firstAirDate" binding:"required"`
	Genres           []string `json:"genres,omitempty"`
	Overview         string   `json:"overview,omitempty"`
	PosterPath       string   `json:"posterPath,omitempty"`
	NumberOfSeasons  int64    `json:"numberOfSeasons,omitempty"`
	NumberOfEpisodes int64    `json:"numberOfEpisodes,omitempty"`
	TMDbID           int64    `json:"tmdbId,omitempty"`
	IMDbID           string   `json:"imdbId,omitempty"`
}

// UpdateSeriesRequest represents the request body for updating a series
type UpdateSeriesRequest struct {
	Title            string   `json:"title,omitempty"`
	OriginalTitle    string   `json:"originalTitle,omitempty"`
	FirstAirDate     string   `json:"firstAirDate,omitempty"`
	LastAirDate      string   `json:"lastAirDate,omitempty"`
	Genres           []string `json:"genres,omitempty"`
	Overview         string   `json:"overview,omitempty"`
	PosterPath       string   `json:"posterPath,omitempty"`
	Rating           float64  `json:"rating,omitempty"`
	NumberOfSeasons  int64    `json:"numberOfSeasons,omitempty"`
	NumberOfEpisodes int64    `json:"numberOfEpisodes,omitempty"`
	Status           string   `json:"status,omitempty"`
	InProduction     *bool    `json:"inProduction,omitempty"`
}

// List handles GET /api/v1/series
// Returns a paginated list of series
func (h *SeriesHandler) List(c *gin.Context) {
	params := parseListParams(c)

	series, pagination, err := h.service.List(c.Request.Context(), params)
	if err != nil {
		slog.Error("Failed to list series", "error", err)
		InternalServerError(c, "Failed to retrieve series")
		return
	}

	SuccessResponse(c, PaginatedResponse{
		Items:      series,
		Page:       pagination.Page,
		PageSize:   pagination.PageSize,
		TotalItems: pagination.TotalResults,
		TotalPages: pagination.TotalPages,
	})
}

// GetByID handles GET /api/v1/series/:id
// Returns a single series by ID
func (h *SeriesHandler) GetByID(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		BadRequestError(c, "VALIDATION_REQUIRED_FIELD", "Series ID is required")
		return
	}

	series, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		slog.Error("Failed to get series", "error", err, "series_id", id)
		NotFoundError(c, "Series")
		return
	}

	SuccessResponse(c, series)
}

// Create handles POST /api/v1/series
// Creates a new series
func (h *SeriesHandler) Create(c *gin.Context) {
	var req CreateSeriesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ValidationError(c, "Invalid request body: "+err.Error())
		return
	}

	series := &models.Series{
		Title:        req.Title,
		FirstAirDate: req.FirstAirDate,
		Genres:       req.Genres,
	}

	// Set optional fields
	if req.OriginalTitle != "" {
		series.OriginalTitle.String = req.OriginalTitle
		series.OriginalTitle.Valid = true
	}
	if req.Overview != "" {
		series.Overview.String = req.Overview
		series.Overview.Valid = true
	}
	if req.PosterPath != "" {
		series.PosterPath.String = req.PosterPath
		series.PosterPath.Valid = true
	}
	if req.NumberOfSeasons != 0 {
		series.NumberOfSeasons.Int64 = req.NumberOfSeasons
		series.NumberOfSeasons.Valid = true
	}
	if req.NumberOfEpisodes != 0 {
		series.NumberOfEpisodes.Int64 = req.NumberOfEpisodes
		series.NumberOfEpisodes.Valid = true
	}
	if req.TMDbID != 0 {
		series.TMDbID.Int64 = req.TMDbID
		series.TMDbID.Valid = true
	}
	if req.IMDbID != "" {
		series.IMDbID.String = req.IMDbID
		series.IMDbID.Valid = true
	}

	if err := h.service.Create(c.Request.Context(), series); err != nil {
		slog.Error("Failed to create series", "error", err, "title", req.Title)
		InternalServerError(c, "Failed to create series")
		return
	}

	CreatedResponse(c, series)
}

// Update handles PUT /api/v1/series/:id
// Updates an existing series
func (h *SeriesHandler) Update(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		BadRequestError(c, "VALIDATION_REQUIRED_FIELD", "Series ID is required")
		return
	}

	// Get existing series
	series, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		NotFoundError(c, "Series")
		return
	}

	var req UpdateSeriesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ValidationError(c, "Invalid request body: "+err.Error())
		return
	}

	// Update fields if provided
	if req.Title != "" {
		series.Title = req.Title
	}
	if req.OriginalTitle != "" {
		series.OriginalTitle.String = req.OriginalTitle
		series.OriginalTitle.Valid = true
	}
	if req.FirstAirDate != "" {
		series.FirstAirDate = req.FirstAirDate
	}
	if req.LastAirDate != "" {
		series.LastAirDate.String = req.LastAirDate
		series.LastAirDate.Valid = true
	}
	if req.Genres != nil {
		series.Genres = req.Genres
	}
	if req.Overview != "" {
		series.Overview.String = req.Overview
		series.Overview.Valid = true
	}
	if req.PosterPath != "" {
		series.PosterPath.String = req.PosterPath
		series.PosterPath.Valid = true
	}
	if req.Rating != 0 {
		series.Rating.Float64 = req.Rating
		series.Rating.Valid = true
	}
	if req.NumberOfSeasons != 0 {
		series.NumberOfSeasons.Int64 = req.NumberOfSeasons
		series.NumberOfSeasons.Valid = true
	}
	if req.NumberOfEpisodes != 0 {
		series.NumberOfEpisodes.Int64 = req.NumberOfEpisodes
		series.NumberOfEpisodes.Valid = true
	}
	if req.Status != "" {
		series.Status.String = req.Status
		series.Status.Valid = true
	}
	if req.InProduction != nil {
		series.InProduction.Bool = *req.InProduction
		series.InProduction.Valid = true
	}

	if err := h.service.Update(c.Request.Context(), series); err != nil {
		slog.Error("Failed to update series", "error", err, "series_id", id)
		InternalServerError(c, "Failed to update series")
		return
	}

	SuccessResponse(c, series)
}

// Delete handles DELETE /api/v1/series/:id
// Deletes a series by ID
func (h *SeriesHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		BadRequestError(c, "VALIDATION_REQUIRED_FIELD", "Series ID is required")
		return
	}

	if err := h.service.Delete(c.Request.Context(), id); err != nil {
		slog.Error("Failed to delete series", "error", err, "series_id", id)
		InternalServerError(c, "Failed to delete series")
		return
	}

	NoContentResponse(c)
}

// Search handles GET /api/v1/series/search
// Searches series by title
func (h *SeriesHandler) Search(c *gin.Context) {
	query := c.Query("q")
	if query == "" {
		BadRequestError(c, "VALIDATION_REQUIRED_FIELD", "Search query 'q' is required")
		return
	}

	params := parseListParams(c)

	series, pagination, err := h.service.SearchByTitle(c.Request.Context(), query, params)
	if err != nil {
		slog.Error("Failed to search series", "error", err, "query", query)
		InternalServerError(c, "Failed to search series")
		return
	}

	SuccessResponse(c, PaginatedResponse{
		Items:      series,
		Page:       pagination.Page,
		PageSize:   pagination.PageSize,
		TotalItems: pagination.TotalResults,
		TotalPages: pagination.TotalPages,
	})
}

// RegisterRoutes registers all series routes on the given router group
func (h *SeriesHandler) RegisterRoutes(rg *gin.RouterGroup) {
	series := rg.Group("/series")
	{
		series.GET("", h.List)
		series.GET("/search", h.Search)
		series.GET("/:id", h.GetByID)
		series.POST("", h.Create)
		series.PUT("/:id", h.Update)
		series.DELETE("/:id", h.Delete)
	}
}
