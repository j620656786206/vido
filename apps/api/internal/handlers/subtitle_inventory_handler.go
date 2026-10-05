package handlers

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"os"

	"github.com/gin-gonic/gin"

	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/repository"
	"github.com/vido/api/internal/subtitle"
)

// SubtitleInventoryHandler answers "what subtitles does this title actually
// have?" for ONE movie or episode, read on demand when the 管理字幕 dialog
// opens (bugfix-subtitle-dialog-real-inventory). It is never called per
// episode on a season expand — that would probe every file (red line 3).
type SubtitleInventoryHandler struct {
	movies   inventoryMovieFinder
	episodes inventoryEpisodeFinder
	prober   subtitle.TrackProber
}

type inventoryMovieFinder interface {
	FindByID(ctx context.Context, id string) (*models.Movie, error)
}

type inventoryEpisodeFinder interface {
	FindByID(ctx context.Context, id string) (*models.Episode, error)
}

// NewSubtitleInventoryHandler wires the handler. prober may be nil (the
// embedded half then reports "unavailable").
func NewSubtitleInventoryHandler(movies inventoryMovieFinder, episodes inventoryEpisodeFinder, prober subtitle.TrackProber) *SubtitleInventoryHandler {
	return &SubtitleInventoryHandler{movies: movies, episodes: episodes, prober: prober}
}

// RegisterRoutes registers the inventory routes.
func (h *SubtitleInventoryHandler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET("/movies/:id/subtitles/inventory", h.Movie)
	rg.GET("/episodes/:id/subtitles/inventory", h.Episode)
}

// Movie godoc
// @Summary      List a movie's actual subtitles
// @Description  Reads, on demand, the subtitle files beside the movie (language of Chinese files decided by content) and its embedded subtitle tracks (ffprobe). Each half carries its own status (ok / unavailable / failed); one failing never hides the other.
// @Tags         subtitles
// @Produce      json
// @Param        id   path  string  true  "Movie ID"
// @Success      200  {object}  APIResponse  "data: {sidecars:{status,files[]}, embedded:{status,tracks[]}}"
// @Failure      400  {object}  APIResponse
// @Failure      404  {object}  APIResponse
// @Router       /api/v1/movies/{id}/subtitles/inventory [get]
func (h *SubtitleInventoryHandler) Movie(c *gin.Context) {
	id := c.Param("id")
	if h.movies == nil {
		NotFoundError(c, "Movie")
		return
	}
	movie, err := h.movies.FindByID(c.Request.Context(), id)
	switch {
	case errors.Is(err, sql.ErrNoRows), err == nil && movie == nil:
		NotFoundError(c, "Movie")
		return
	case err != nil:
		slog.Error("Failed to look up movie for subtitle inventory", "movie_id", id, "error", err)
		InternalServerError(c, "Failed to look up movie")
		return
	}
	h.respond(c, movie.FilePath, movie.SubtitlePath, "這部電影沒有媒體檔案路徑——請先掃描媒體庫", "找不到這部電影的媒體檔案——請確認檔案仍在磁碟上")
}

// Episode godoc
// @Summary      List an episode's actual subtitles
// @Description  Same as the movie route, for one episode.
// @Tags         subtitles
// @Produce      json
// @Param        id   path  string  true  "Episode ID"
// @Success      200  {object}  APIResponse  "data: {sidecars:{status,files[]}, embedded:{status,tracks[]}}"
// @Failure      400  {object}  APIResponse
// @Failure      404  {object}  APIResponse
// @Router       /api/v1/episodes/{id}/subtitles/inventory [get]
func (h *SubtitleInventoryHandler) Episode(c *gin.Context) {
	id := c.Param("id")
	if h.episodes == nil {
		NotFoundError(c, "Episode")
		return
	}
	episode, err := h.episodes.FindByID(c.Request.Context(), id)
	switch {
	case errors.Is(err, repository.ErrEpisodeNotFound), err == nil && episode == nil:
		NotFoundError(c, "Episode")
		return
	case err != nil:
		slog.Error("Failed to look up episode for subtitle inventory", "episode_id", id, "error", err)
		InternalServerError(c, "Failed to look up episode")
		return
	}
	h.respond(c, episode.FilePath, episode.SubtitlePath, "這一集沒有媒體檔案路徑——請先掃描媒體庫", "找不到這一集的媒體檔案——請確認檔案仍在磁碟上")
}

func (h *SubtitleInventoryHandler) respond(c *gin.Context, filePath, subtitlePath models.NullString, noPathMsg, missingMsg string) {
	if !filePath.Valid || filePath.String == "" {
		BadRequestError(c, "VALIDATION_REQUIRED_FIELD", noPathMsg)
		return
	}
	if _, err := os.Stat(filePath.String); err != nil {
		BadRequestError(c, "VALIDATION_REQUIRED_FIELD", missingMsg)
		return
	}
	own := ""
	if subtitlePath.Valid {
		own = subtitlePath.String
	}
	SuccessResponse(c, subtitle.BuildInventory(c.Request.Context(), filePath.String, own, h.prober))
}
