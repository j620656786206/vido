package handlers

import (
	"context"
	"errors"
	"log/slog"

	"github.com/gin-gonic/gin"
	"github.com/vido/api/internal/services"
)

// DoubanRatingServiceInterface defines the contract for Douban rating
// enrichment, enabling handler tests with a mock service.
type DoubanRatingServiceInterface interface {
	EnrichDoubanRating(ctx context.Context, mediaID, mediaType string) (*services.DoubanRatingResult, error)
}

// DoubanRatingHandler serves the lazy Douban rating endpoints used by the
// detail page (Story 12-1). Following Handler → Service → Repository, it holds
// no business logic — it delegates to DoubanRatingService.
type DoubanRatingHandler struct {
	service DoubanRatingServiceInterface
}

// NewDoubanRatingHandler creates a new DoubanRatingHandler.
func NewDoubanRatingHandler(service DoubanRatingServiceInterface) *DoubanRatingHandler {
	return &DoubanRatingHandler{service: service}
}

// GetMovieDoubanRating handles GET /api/v1/movies/:id/douban-rating
// Returns the Douban rating for a movie, or `data: null` when unavailable.
func (h *DoubanRatingHandler) GetMovieDoubanRating(c *gin.Context) {
	h.getRating(c, "movie")
}

// GetSeriesDoubanRating handles GET /api/v1/series/:id/douban-rating
// Returns the Douban rating for a series, or `data: null` when unavailable.
func (h *DoubanRatingHandler) GetSeriesDoubanRating(c *gin.Context) {
	h.getRating(c, "series")
}

func (h *DoubanRatingHandler) getRating(c *gin.Context, mediaType string) {
	id := c.Param("id")
	if id == "" {
		BadRequestError(c, "VALIDATION_REQUIRED_FIELD", "Media ID is required")
		return
	}

	result, err := h.service.EnrichDoubanRating(c.Request.Context(), id, mediaType)
	if err != nil {
		// Douban scrape failures degrade to a nil result (not an error); only a
		// hard failure reaches here. A missing record → 404; anything else is a
		// genuine infrastructure failure → 500 (do not mask it as not-found).
		if errors.Is(err, services.ErrMediaNotFound) {
			NotFoundError(c, "Media")
			return
		}
		slog.Error("Failed to enrich Douban rating", "error", err, "media_id", id, "media_type", mediaType)
		InternalServerError(c, "Failed to fetch Douban rating")
		return
	}

	// result may be nil — graceful degradation. SuccessResponse serializes it
	// as `data: null`, and the UI falls back to TMDb-only (AC #4).
	SuccessResponse(c, result)
}

// RegisterRoutes registers the Douban rating routes on the given router group.
// The :id param name matches the existing /movies/:id and /series/:id routes,
// so Gin's radix tree accepts the longer paths without conflict.
func (h *DoubanRatingHandler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET("/movies/:id/douban-rating", h.GetMovieDoubanRating)
	rg.GET("/series/:id/douban-rating", h.GetSeriesDoubanRating)
}
