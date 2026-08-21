package handlers

import (
	"context"
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/services"
)

// NFOLocalizerMovieGetter is the movie lookup the localizer handler needs.
type NFOLocalizerMovieGetter interface {
	GetByID(ctx context.Context, id string) (*models.Movie, error)
}

// NFOLocalizerSeriesGetter is the series lookup the TV routes need (9R-13a).
// Narrow on purpose (Rule 11), a sibling of NFOLocalizerMovieGetter rather than
// a widening of it — the two return different row types.
type NFOLocalizerSeriesGetter interface {
	GetByID(ctx context.Context, id string) (*models.Series, error)
}

// NFOLocalizerEpisodeGetter is the episode lookup for the single-episode route
// (9R-13a). Mirrors TranscriptionEpisodeGetter; *repository.EpisodeRepository
// satisfies it.
type NFOLocalizerEpisodeGetter interface {
	FindByID(ctx context.Context, id string) (*models.Episode, error)
}

// NFOLocalizerServiceInterface is the localizer contract (Story 9R-13, extended
// by 9R-13a with the TV surface).
type NFOLocalizerServiceInterface interface {
	IsAvailable() bool
	LocalizeMovieNFO(ctx context.Context, movie models.Movie) (*services.NFOLocalizeResult, error)
	LocalizeTVShowNFO(ctx context.Context, series models.Series) (*services.NFOLocalizeResult, error)
	LocalizeSeriesNFOWithEpisodes(ctx context.Context, series models.Series) (*services.NFOSeriesLocalizeResult, error)
	LocalizeEpisodeNFO(ctx context.Context, episode models.Episode, showTitle string) (*services.NFOLocalizeResult, error)
}

// replaceConfirmation is the body both TV routes require.
//
// TV .nfo names are single-slot (spike 9R-S1): there is no free filename to
// write beside the user's original the way the movie path has, so localizing a
// show REPLACES a file the user may have curated by hand. The original is
// backed up to `.nfo.orig` first, but a backup is not consent — the caller has
// to say so.
type replaceConfirmation struct {
	ConfirmReplace bool `json:"confirm_replace"`
}

// NFOLocalizerHandler serves the .nfo localization endpoints.
type NFOLocalizerHandler struct {
	movieService   NFOLocalizerMovieGetter
	seriesService  NFOLocalizerSeriesGetter
	episodeService NFOLocalizerEpisodeGetter
	localizer      NFOLocalizerServiceInterface
}

// NewNFOLocalizerHandler creates a new NFOLocalizerHandler. seriesService and
// episodeService are optional (9R-13a): when nil, the TV routes are not
// registered at all, exactly as main.go already omits every route when the
// localizer itself is absent.
func NewNFOLocalizerHandler(
	movieService NFOLocalizerMovieGetter,
	seriesService NFOLocalizerSeriesGetter,
	episodeService NFOLocalizerEpisodeGetter,
	localizer NFOLocalizerServiceInterface,
) *NFOLocalizerHandler {
	return &NFOLocalizerHandler{
		movieService:   movieService,
		seriesService:  seriesService,
		episodeService: episodeService,
		localizer:      localizer,
	}
}

// RegisterRoutes registers the localizer routes.
func (h *NFOLocalizerHandler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.POST("/movies/:id/localize-nfo", h.LocalizeMovie)
	if h.seriesService != nil {
		rg.POST("/series/:id/localize-nfo", h.LocalizeSeries)
	}
	if h.episodeService != nil {
		rg.POST("/episodes/:id/localize-nfo", h.LocalizeEpisode)
	}
}

// requireReplaceConfirmation is the 9R-13a opt-in gate.
//
// It runs BEFORE any lookup or localization call on purpose: rejecting after
// translating would still have spent the user's Claude budget on a request the
// server was always going to refuse.
func requireReplaceConfirmation(c *gin.Context) bool {
	var body replaceConfirmation
	// A missing or malformed body is treated as "not confirmed" rather than a
	// 400: the caller's intent is the same either way, and one clear message
	// beats two.
	_ = c.ShouldBindJSON(&body)
	if body.ConfirmReplace {
		return true
	}
	ErrorResponse(c, http.StatusConflict, "NFO_REPLACE_NOT_CONFIRMED",
		"影集的 .nfo 只有單一檔名可用，在地化會覆寫既有檔案",
		"原始檔會先備份為 .nfo.orig。確認後請在請求內容加上 \"confirm_replace\": true。")
	return false
}

// wantsEpisodes reads the ?include_episodes flag.
//
// CR M4: an earlier draft compared against the literal string "true", so
// `?include_episodes=1` and a bare `?include_episodes` silently localized the
// show file ONLY — the caller gets a 200 and assumes 24 episodes were done.
// Presence means yes unless the value explicitly says otherwise.
func wantsEpisodes(c *gin.Context) bool {
	raw, present := c.GetQuery("include_episodes")
	if !present {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "false", "0", "no":
		return false
	default:
		return true
	}
}

// LocalizeSeries localizes tvshow.nfo, optionally every episode too (9R-13a).
// POST /api/v1/series/:id/localize-nfo[?include_episodes=true]
func (h *NFOLocalizerHandler) LocalizeSeries(c *gin.Context) {
	if h.localizer == nil || !h.localizer.IsAvailable() {
		ErrorResponse(c, http.StatusServiceUnavailable, "NFO_LOCALIZE_DISABLED",
			"Metadata localization is not available",
			"Ensure a translation provider (CLAUDE_API_KEY) is configured.")
		return
	}
	if !requireReplaceConfirmation(c) {
		return
	}

	idStr := c.Param("id")
	series, err := h.seriesService.GetByID(c.Request.Context(), idStr)
	if err != nil || series == nil {
		slog.Error("Failed to get series for nfo localization", "id", idStr, "error", err)
		NotFoundError(c, "Series")
		return
	}
	if !series.FilePath.Valid || series.FilePath.String == "" {
		BadRequestError(c, "VALIDATION_REQUIRED_FIELD", "影集沒有資料夾路徑 —— 請先掃描媒體庫")
		return
	}

	if wantsEpisodes(c) {
		res, err := h.localizer.LocalizeSeriesNFOWithEpisodes(c.Request.Context(), *series)
		if err != nil {
			slog.Error("series nfo localization failed", "id", idStr, "error", err)
			ErrorResponse(c, http.StatusInternalServerError, "NFO_LOCALIZE_FAILED",
				"Failed to localize metadata", "Check server logs for details.")
			return
		}
		SuccessResponse(c, res)
		return
	}

	res, err := h.localizer.LocalizeTVShowNFO(c.Request.Context(), *series)
	if err != nil {
		slog.Error("tvshow nfo localization failed", "id", idStr, "error", err)
		ErrorResponse(c, http.StatusInternalServerError, "NFO_LOCALIZE_FAILED",
			"Failed to localize metadata", "Check server logs for details.")
		return
	}
	SuccessResponse(c, res)
}

// LocalizeEpisode localizes one episode's .nfo (9R-13a).
// POST /api/v1/episodes/:id/localize-nfo
func (h *NFOLocalizerHandler) LocalizeEpisode(c *gin.Context) {
	if h.localizer == nil || !h.localizer.IsAvailable() {
		ErrorResponse(c, http.StatusServiceUnavailable, "NFO_LOCALIZE_DISABLED",
			"Metadata localization is not available",
			"Ensure a translation provider (CLAUDE_API_KEY) is configured.")
		return
	}
	if !requireReplaceConfirmation(c) {
		return
	}

	idStr := c.Param("id")
	episode, err := h.episodeService.FindByID(c.Request.Context(), idStr)
	if err != nil || episode == nil {
		slog.Error("Failed to get episode for nfo localization", "id", idStr, "error", err)
		NotFoundError(c, "Episode")
		return
	}
	if !episode.FilePath.Valid || episode.FilePath.String == "" {
		BadRequestError(c, "VALIDATION_REQUIRED_FIELD", "這一集沒有檔案路徑 —— 請先掃描媒體庫")
		return
	}

	// <showtitle> is resolved from the parent series when one is reachable.
	// Missing it is cosmetic (the element is omitempty), so a lookup failure
	// must not fail the localization.
	showTitle := ""
	if h.seriesService != nil {
		if series, serr := h.seriesService.GetByID(c.Request.Context(), episode.SeriesID); serr == nil && series != nil {
			showTitle = series.Title
		}
	}

	res, err := h.localizer.LocalizeEpisodeNFO(c.Request.Context(), *episode, showTitle)
	if err != nil {
		slog.Error("episode nfo localization failed", "id", idStr, "error", err)
		ErrorResponse(c, http.StatusInternalServerError, "NFO_LOCALIZE_FAILED",
			"Failed to localize metadata", "Check server logs for details.")
		return
	}
	SuccessResponse(c, res)
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
	// PRE-EXISTING FIX (9R-13a, surfaced by the new handler tests): the getter
	// interface permits (nil, nil), and without the nil arm the next line
	// dereferences it — a panic and a stack-trace 500 where a clean 404
	// belongs. Same defect class as 9R-10a CR L3 on the transcribe route.
	if err != nil || movie == nil {
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
