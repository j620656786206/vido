package handlers

import (
	"fmt"
	"log/slog"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/vido/api/internal/services"
)

// availabilityCheckOwnedMaxIDs caps the request size to keep the SQL IN-list
// bounded. SQLite's SQLITE_MAX_VARIABLE_NUMBER defaults to 32766 (modernc.org
// driver inherits this), but a homepage view never needs more than a few
// hundred cards. Reject clearly above that so the bad caller gets a useful
// error instead of a truncated query.
const availabilityCheckOwnedMaxIDs = 500

// AvailabilityHandler serves /media/check-owned — the batch ownership lookup
// used by the homepage to render 已有 / 已請求 badges (Story 10-4).
type AvailabilityHandler struct {
	service services.AvailabilityServiceInterface
}

// NewAvailabilityHandler constructs a handler bound to the given service.
func NewAvailabilityHandler(service services.AvailabilityServiceInterface) *AvailabilityHandler {
	return &AvailabilityHandler{service: service}
}

// CheckOwnedRequest is the POST body for /media/check-owned.
// Example: {"tmdb_ids": [603, 157336, 1396]}
type CheckOwnedRequest struct {
	TMDbIDs []int64 `json:"tmdb_ids" binding:"required"`
}

// CheckOwnedResponse carries the subset of input IDs that are already owned.
// Example: {"owned_ids": [603, 1396]}
type CheckOwnedResponse struct {
	OwnedIDs []int64 `json:"owned_ids"`
}

// CheckOwned handles POST /api/v1/media/check-owned.
// @Summary Batch check owned TMDb IDs across movies and series
// @Description Returns the subset of the supplied TMDb IDs that exist as
// @Description non-removed records in the local library. Used by homepage
// @Description availability badges to avoid N+1 queries.
// @Tags media
// @Accept json
// @Produce json
// @Param request body CheckOwnedRequest true "TMDb IDs to check"
// @Success 200 {object} APIResponse{data=CheckOwnedResponse}
// @Failure 400 {object} APIResponse "Invalid request body or too many IDs"
// @Router /api/v1/media/check-owned [post]
func (h *AvailabilityHandler) CheckOwned(c *gin.Context) {
	var req CheckOwnedRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequestError(c, "VALIDATION_INVALID_FORMAT", "Invalid request body: "+err.Error())
		return
	}

	if len(req.TMDbIDs) > availabilityCheckOwnedMaxIDs {
		BadRequestError(c, "VALIDATION_INVALID_FORMAT",
			fmt.Sprintf("Too many TMDb IDs — max %d per request", availabilityCheckOwnedMaxIDs))
		return
	}

	owned, err := h.service.CheckOwned(c.Request.Context(), req.TMDbIDs)
	if err != nil {
		slog.Error("Failed to check owned media", "error", err, "id_count", len(req.TMDbIDs))
		InternalServerError(c, "Failed to check owned media")
		return
	}

	// Never send null — the frontend expects an array so an empty set still
	// normalises to `{"owned_ids": []}`.
	if owned == nil {
		owned = []int64{}
	}

	SuccessResponse(c, CheckOwnedResponse{OwnedIDs: owned})
}

// LibraryIDResponse is the library id of an owned title, or null.
// Example: {"id": "511b6c7c-fada-4233-a322-e8f563fa27bd"}
type LibraryIDResponse struct {
	ID *string `json:"id"`
}

// LibraryID handles GET /api/v1/media/library-id?type=movie|tv&tmdb_id=N.
// @Summary Find the library id of an owned TMDb title
// @Description The TMDb detail page sends an owned title to its library page
// @Description (seasons, episodes, subtitles). id is null when the library
// @Description does not have this title (disc-2026-11-tmdb-detail-owned-no-link-to-library).
// @Tags media
// @Produce json
// @Param type query string true "movie or tv"
// @Param tmdb_id query int true "TMDb id"
// @Success 200 {object} APIResponse{data=LibraryIDResponse}
// @Failure 400 {object} APIResponse "Invalid type or tmdb_id"
// @Router /api/v1/media/library-id [get]
func (h *AvailabilityHandler) LibraryID(c *gin.Context) {
	mediaType := c.Query("type")
	if mediaType != "movie" && mediaType != "tv" {
		BadRequestError(c, "VALIDATION_INVALID_FORMAT", "type must be movie or tv")
		return
	}
	tmdbID, err := strconv.ParseInt(c.Query("tmdb_id"), 10, 64)
	if err != nil || tmdbID <= 0 {
		BadRequestError(c, "VALIDATION_INVALID_FORMAT", "tmdb_id must be a positive integer")
		return
	}

	id, err := h.service.LibraryID(c.Request.Context(), mediaType, tmdbID)
	if err != nil {
		slog.Error("Failed to look up library id", "error", err, "type", mediaType, "tmdb_id", tmdbID)
		InternalServerError(c, "Failed to look up library id")
		return
	}
	if id == "" {
		SuccessResponse(c, LibraryIDResponse{})
		return
	}
	SuccessResponse(c, LibraryIDResponse{ID: &id})
}

// RegisterRoutes mounts /media/check-owned and /media/library-id under the
// given API group.
func (h *AvailabilityHandler) RegisterRoutes(rg *gin.RouterGroup) {
	media := rg.Group("/media")
	{
		media.POST("/check-owned", h.CheckOwned)
		media.GET("/library-id", h.LibraryID)
	}
}
