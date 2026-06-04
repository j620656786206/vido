package handlers

import (
	"context"
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/vido/api/internal/services"
	"github.com/vido/api/internal/tmdb"
)

// UnifiedSearchServiceInterface defines the contract for the unified
// instant-search service (Story 11-3). Declared in the handlers package so
// the handler can be tested with a mock without importing the concrete service.
type UnifiedSearchServiceInterface interface {
	Search(ctx context.Context, query string, page int) (*services.UnifiedSearchResult, error)
}

// SearchHandler handles the unified instant-search endpoint, aggregating
// movies, TV shows, and people into a single response (AC #1).
type SearchHandler struct {
	service UnifiedSearchServiceInterface
}

// NewSearchHandler creates a SearchHandler with the given service.
func NewSearchHandler(service UnifiedSearchServiceInterface) *SearchHandler {
	return &SearchHandler{service: service}
}

// Search handles GET /api/v1/search?q={query}&page=1
// Returns unified, dual-language (zh-TW + en) results across movies, TV shows,
// and people with zh-TW title matches boosted to the top of each media list.
// @Summary Unified instant search
// @Description Dual-language (zh-TW + en) search across movies, TV shows, and people with zh-TW boost ranking (Story 11-3)
// @Tags search
// @Accept json
// @Produce json
// @Param q query string true "Search query"
// @Param page query int false "Page number" default(1)
// @Success 200 {object} APIResponse{data=services.UnifiedSearchResult}
// @Failure 400 {object} APIResponse{error=APIError}
// @Failure 500 {object} APIResponse{error=APIError}
// @Router /api/v1/search [get]
func (h *SearchHandler) Search(c *gin.Context) {
	query := strings.TrimSpace(c.Query("q"))
	if query == "" {
		ErrorResponse(c, http.StatusBadRequest, tmdb.ErrCodeBadRequest,
			"Search query is required",
			"Please provide a 'q' parameter")
		return
	}

	page := parsePageQuery(c.Query("page"))

	result, err := h.service.Search(c.Request.Context(), query, page)
	if err != nil {
		handleTMDbError(c, err, "unified search", slog.String("query", query))
		return
	}

	SuccessResponse(c, result)
}

// RegisterRoutes registers the unified search route on the given router group.
// The endpoint is top-level (/api/v1/search) rather than nested under /tmdb,
// keeping the existing per-type search routes intact for backward compatibility
// (AC #6).
func (h *SearchHandler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET("/search", h.Search)
}
