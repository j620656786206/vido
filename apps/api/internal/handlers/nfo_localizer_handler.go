package handlers

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/services"
)

// NFOLocalizerMovieGetter is the movie lookup the localizer handler needs.
type NFOLocalizerMovieGetter interface {
	GetByID(ctx context.Context, id string) (*models.Movie, error)
}

// NFOLocalizerServiceInterface is the localizer contract (Story 9R-13).
type NFOLocalizerServiceInterface interface {
	IsAvailable() bool
	LocalizeMovieNFO(ctx context.Context, movie models.Movie) (*services.NFOLocalizeResult, error)
}

// NFOLocalizerHandler serves the .nfo localization endpoint.
type NFOLocalizerHandler struct {
	movieService NFOLocalizerMovieGetter
	localizer    NFOLocalizerServiceInterface
}

// NewNFOLocalizerHandler creates a new NFOLocalizerHandler.
func NewNFOLocalizerHandler(movieService NFOLocalizerMovieGetter, localizer NFOLocalizerServiceInterface) *NFOLocalizerHandler {
	return &NFOLocalizerHandler{movieService: movieService, localizer: localizer}
}

// RegisterRoutes registers the localizer route.
func (h *NFOLocalizerHandler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.POST("/movies/:id/localize-nfo", h.LocalizeMovie)
}

// LocalizeMovie writes an additive zh-TW .nfo for a movie (Story 9R-13).
// POST /api/v1/movies/:id/localize-nfo
func (h *NFOLocalizerHandler) LocalizeMovie(c *gin.Context) {
	if h.localizer == nil || !h.localizer.IsAvailable() {
		ErrorResponse(c, http.StatusServiceUnavailable, "NFO_LOCALIZE_DISABLED",
			"Metadata localization is not available",
			"Ensure a translation provider (CLAUDE_API_KEY) is configured.")
		return
	}

	idStr := c.Param("id")
	movie, err := h.movieService.GetByID(c.Request.Context(), idStr)
	if err != nil {
		slog.Error("Failed to get movie for nfo localization", "id", idStr, "error", err)
		NotFoundError(c, "Movie")
		return
	}
	if !movie.FilePath.Valid || movie.FilePath.String == "" {
		BadRequestError(c, "VALIDATION_REQUIRED_FIELD", "Movie has no file path — scan the media library first")
		return
	}

	res, err := h.localizer.LocalizeMovieNFO(c.Request.Context(), *movie)
	if err != nil {
		slog.Error("nfo localization failed", "id", idStr, "error", err)
		ErrorResponse(c, http.StatusInternalServerError, "NFO_LOCALIZE_FAILED",
			"Failed to localize metadata", "Check server logs for details.")
		return
	}
	SuccessResponse(c, res)
}
