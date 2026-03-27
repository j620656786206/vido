package handlers

import (
	"context"
	"log/slog"
	"sort"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/repository"
)

// RecentMediaServiceInterface defines the contract for recent media operations.
type RecentMediaServiceInterface interface {
	GetRecentMedia(ctx context.Context, limit int) ([]RecentMediaItem, error)
}

// RecentMediaItem represents a unified media item (movie or series) for the dashboard.
type RecentMediaItem struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Year      int       `json:"year,omitempty"`
	PosterURL string    `json:"poster_url,omitempty"`
	MediaType string    `json:"media_type"` // "movie" or "tv"
	JustAdded bool      `json:"just_added"`
	AddedAt   time.Time `json:"added_at"`
}

// RecentMediaHandler handles HTTP requests for the dashboard recent media endpoint.
type RecentMediaHandler struct {
	movieService  MovieServiceInterface
	seriesService SeriesServiceInterface
}

// NewRecentMediaHandler creates a new RecentMediaHandler.
func NewRecentMediaHandler(movieService MovieServiceInterface, seriesService SeriesServiceInterface) *RecentMediaHandler {
	return &RecentMediaHandler{
		movieService:  movieService,
		seriesService: seriesService,
	}
}

// GetRecentMedia handles GET /api/v1/media/recent
// @Summary Get recently added media
// @Description Returns recently added media items (movies and series combined), sorted by creation date descending
// @Tags media
// @Accept json
// @Produce json
// @Param limit query int false "Number of items to return" default(10) minimum(1) maximum(50)
// @Success 200 {object} APIResponse{data=[]RecentMediaItem} "List of recent media items"
// @Failure 500 {object} APIResponse "Internal server error"
// @Router /api/v1/media/recent [get]
func (h *RecentMediaHandler) GetRecentMedia(c *gin.Context) {
	limit := 10
	if l, err := strconv.Atoi(c.Query("limit")); err == nil && l > 0 && l <= 50 {
		limit = l
	}

	// Fetch recent movies and series in parallel approach (sequential for simplicity)
	params := repository.ListParams{
		Page:      1,
		PageSize:  limit,
		SortBy:    "created_at",
		SortOrder: "desc",
	}

	movies, _, err := h.movieService.List(c.Request.Context(), params)
	if err != nil {
		slog.Error("Failed to fetch recent movies", "error", err)
		InternalServerError(c, "Failed to fetch recent media")
		return
	}

	series, _, err := h.seriesService.List(c.Request.Context(), params)
	if err != nil {
		slog.Error("Failed to fetch recent series", "error", err)
		InternalServerError(c, "Failed to fetch recent media")
		return
	}

	fiveMinutesAgo := time.Now().Add(-5 * time.Minute)

	// Combine movies and series into unified response
	items := make([]RecentMediaItem, 0, len(movies)+len(series))

	for _, m := range movies {
		year := extractYear(m.ReleaseDate)
		items = append(items, RecentMediaItem{
			ID:        m.ID,
			Title:     m.Title,
			Year:      year,
			PosterURL: nullStringValue(m.PosterPath),
			MediaType: "movie",
			JustAdded: m.CreatedAt.After(fiveMinutesAgo),
			AddedAt:   m.CreatedAt,
		})
	}

	for _, s := range series {
		year := extractYear(s.FirstAirDate)
		items = append(items, RecentMediaItem{
			ID:        s.ID,
			Title:     s.Title,
			Year:      year,
			PosterURL: nullStringValue(s.PosterPath),
			MediaType: "tv",
			JustAdded: s.CreatedAt.After(fiveMinutesAgo),
			AddedAt:   s.CreatedAt,
		})
	}

	// Sort by AddedAt descending and limit
	sort.Slice(items, func(i, j int) bool {
		return items[i].AddedAt.After(items[j].AddedAt)
	})

	if len(items) > limit {
		items = items[:limit]
	}

	SuccessResponse(c, items)
}

// RegisterRoutes registers recent media routes.
func (h *RecentMediaHandler) RegisterRoutes(rg *gin.RouterGroup) {
	media := rg.Group("/media")
	{
		media.GET("/recent", h.GetRecentMedia)
	}
}

func extractYear(dateStr string) int {
	if len(dateStr) >= 4 {
		year, err := strconv.Atoi(dateStr[:4])
		if err == nil {
			return year
		}
	}
	return 0
}

func nullStringValue(ns models.NullString) string {
	if ns.Valid {
		return ns.String
	}
	return ""
}
