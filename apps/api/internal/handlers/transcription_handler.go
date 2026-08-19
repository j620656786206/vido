package handlers

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/repository"
	"github.com/vido/api/internal/services"
)

// TranscriptionMovieGetter defines the movie lookup needed by the transcription handler.
type TranscriptionMovieGetter interface {
	GetByID(ctx context.Context, id string) (*models.Movie, error)
}

// TranscriptionEpisodeGetter defines the episode lookup needed by the
// per-episode transcribe route (story 9R-10a AC #2). Narrow on purpose
// (Rule 11), a sibling of TranscriptionMovieGetter rather than a widening of
// it — the two repositories expose different method names.
// *repository.EpisodeRepository satisfies it.
type TranscriptionEpisodeGetter interface {
	FindByID(ctx context.Context, id string) (*models.Episode, error)
}

// TranscriptionServiceInterface defines the contract for transcription operations.
type TranscriptionServiceInterface interface {
	IsAvailable() bool
	// CanResumeTranslateOnly reports whether this media would resume
	// translate-only (CR sub-2-2a M2) — such a run needs no ASR, so the
	// availability gate must not 503 it.
	CanResumeTranslateOnly(ctx context.Context, mediaID string) bool
	// CanResumeEpisodeTranslateOnly is the EPISODE counterpart (story 9R-10a
	// AC #3). Separate method, not a mediaType parameter on the one above:
	// widening that signature would churn every movie-side call site and fake
	// for no gain (Rule 11 — same call the sub-3-2 writer/reader pair made).
	CanResumeEpisodeTranslateOnly(ctx context.Context, episodeID string) bool
	IsInProgress(mediaID string) bool
	StartTranscription(ctx context.Context, mediaID string, filePath string, mediaDir string, opts ...services.TranscriptionOption) (string, error)
}

// TranscriptionHandler handles transcription API requests.
type TranscriptionHandler struct {
	movieService         TranscriptionMovieGetter
	episodeService       TranscriptionEpisodeGetter
	transcriptionService TranscriptionServiceInterface
}

// NewTranscriptionHandler creates a new TranscriptionHandler. episodeService
// may be nil — the per-episode route is then simply not mounted (see
// RegisterRoutes), which keeps movie-only callers and their fakes valid.
func NewTranscriptionHandler(movieService TranscriptionMovieGetter, episodeService TranscriptionEpisodeGetter, transcriptionService TranscriptionServiceInterface) *TranscriptionHandler {
	return &TranscriptionHandler{
		movieService:         movieService,
		episodeService:       episodeService,
		transcriptionService: transcriptionService,
	}
}

// RegisterRoutes registers transcription routes on the given router group.
//
// The per-episode route (story 9R-10a) is mounted ONLY when an episode getter
// is wired. Capability honor: an unwired deployment answers 404 ("this route
// does not exist here") instead of panicking on a nil lookup or inventing a
// 503 that would blame the ASR configuration for a wiring mistake.
func (h *TranscriptionHandler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.POST("/movies/:id/transcribe", h.TranscribeMovie)
	if h.episodeService != nil {
		rg.POST("/episodes/:id/transcribe", h.TranscribeEpisode)
	}
}

// TranscribeMovie triggers transcription for a movie.
// POST /api/v1/movies/:id/transcribe
// Returns 202 Accepted with job ID.
func (h *TranscriptionHandler) TranscribeMovie(c *gin.Context) {
	// Validate movie ID — an opaque STRING (movie PKs are UUIDs, 9R-18);
	// non-empty is the only format constraint. Parsed BEFORE the availability
	// gate so the gate can consult the per-media resume eligibility.
	id := c.Param("id")
	if id == "" {
		BadRequestError(c, "VALIDATION_INVALID_FORMAT", "Invalid movie ID")
		return
	}

	// Availability gate, resume-aware (CR sub-2-2a M2): a translate-only
	// resume needs no ASR, so an `untranslated` row with its English SRT on
	// disk proceeds even when FFmpeg/ASR are gone.
	if !h.transcriptionService.IsAvailable() &&
		!h.transcriptionService.CanResumeTranslateOnly(c.Request.Context(), id) {
		// sub-2-2d AC #3: the γ-ratified zh-TW envelope (this body was English —
		// a Rule 3 gap). sub-5-2 AC #4 retired the restart clause: the ASR client
		// is no longer boot-built — ASRProviderHolder resolves the key per call,
		// so a key saved on the settings page takes effect immediately. Telling
		// the user to restart their NAS is now a false instruction.
		ErrorResponse(c, http.StatusServiceUnavailable, "TRANSCRIPTION_DISABLED",
			"語音辨識尚未設定",
			"生成字幕需要雲端語音辨識（ASR）金鑰。請至金鑰設定（/settings/keys）儲存雲端 ASR 金鑰，儲存後立即生效。")
		return
	}

	// Fetch movie
	movie, err := h.movieService.GetByID(c.Request.Context(), id)
	if err != nil {
		slog.Error("Failed to get movie for transcription", "id", id, "error", err)
		NotFoundError(c, "Movie")
		return
	}

	// Validate file_path exists
	if !movie.FilePath.Valid || movie.FilePath.String == "" {
		BadRequestError(c, "VALIDATION_REQUIRED_FIELD", "Movie has no file path — scan the media library first")
		return
	}

	// Validate file is accessible on disk (AC #1, task 4.3)
	if _, err := os.Stat(movie.FilePath.String); err != nil {
		BadRequestError(c, "VALIDATION_REQUIRED_FIELD", "Movie file not accessible — check if the file exists on disk")
		return
	}

	// Check if transcription is already running
	if h.transcriptionService.IsInProgress(id) {
		ErrorResponse(c, http.StatusConflict, "TRANSCRIPTION_IN_PROGRESS",
			"Transcription is already running for this movie",
			"Wait for the current transcription to complete.")
		return
	}

	// Check for translate=true query param (Story 9-2b)
	var opts []services.TranscriptionOption
	if c.Query("translate") == "true" {
		opts = append(opts, services.WithTranslation())
	}

	// Start async transcription
	mediaDir := filepath.Dir(movie.FilePath.String)
	jobID, err := h.transcriptionService.StartTranscription(c.Request.Context(), id, movie.FilePath.String, mediaDir, opts...)
	if err != nil {
		if errors.Is(err, services.ErrTranscriptionInProgress) {
			ErrorResponse(c, http.StatusConflict, "TRANSCRIPTION_IN_PROGRESS",
				"Transcription is already running for this movie",
				"Wait for the current transcription to complete.")
			return
		}
		slog.Error("Failed to start transcription", "movie_id", id, "error", err)
		InternalServerError(c, "Failed to start transcription")
		return
	}

	// Return 202 Accepted
	c.JSON(http.StatusAccepted, APIResponse{
		Success: true,
		Data: map[string]string{
			"job_id":  jobID,
			"message": "Transcription started. Listen to SSE events for progress.",
		},
	})
}

// TranscribeEpisode triggers subtitle generation for a single episode
// (story 9R-10a AC #2 [@contract-v1]).
//
// @Summary      Generate subtitles for one episode
// @Description  Runs the Route C pipeline (extract → speech recognition → glossary-aware translation → OpenCC → place) for a SINGLE episode, the per-item counterpart of the movies route. Translation is ALWAYS on: an English-only SRT has no consumer and would contradict the UI's "語音辨識＋AI 翻譯" promise, so this route takes no translate query param. The run is media-type-aware, so its status writeback lands on the EPISODES table. Asynchronous — progress arrives on the transcription_* SSE events keyed by this episode id.
// @Tags         subtitles
// @Produce      json
// @Param        id   path      string  true  "Episode row id (UUID string, 9R-18)"
// @Success      202  {object}  APIResponse  "{job_id, message}"
// @Failure      400  {object}  APIResponse  "VALIDATION_INVALID_FORMAT (empty id) / VALIDATION_REQUIRED_FIELD (no file path, or file missing on disk)"
// @Failure      404  {object}  APIResponse  "episode not found"
// @Failure      409  {object}  APIResponse  "TRANSCRIPTION_IN_PROGRESS — a run for this episode is already in flight"
// @Failure      500  {object}  APIResponse  "failed to start"
// @Failure      503  {object}  APIResponse  "TRANSCRIPTION_DISABLED — no ASR capability AND this episode cannot resume translate-only"
// @Router       /api/v1/episodes/{id}/transcribe [post]
func (h *TranscriptionHandler) TranscribeEpisode(c *gin.Context) {
	// Episode PKs are opaque UUID STRINGS (9R-18) — non-empty is the only
	// format constraint. Parsed BEFORE the availability gate so the gate can
	// consult this episode's resume eligibility.
	id := c.Param("id")
	if id == "" {
		BadRequestError(c, "VALIDATION_INVALID_FORMAT", "影集單集 ID 無效")
		return
	}

	// Availability gate, resume-aware — and EPISODE-scoped (story red line 3).
	// CanResumeTranslateOnly hard-codes the MOVIE table, so using it here would
	// return false for every episode and 503 an `untranslated` episode whose
	// English SRT is already on disk — a run that needs no ASR at all.
	if !h.transcriptionService.IsAvailable() &&
		!h.transcriptionService.CanResumeEpisodeTranslateOnly(c.Request.Context(), id) {
		ErrorResponse(c, http.StatusServiceUnavailable, "TRANSCRIPTION_DISABLED",
			"語音辨識尚未設定",
			"生成字幕需要雲端語音辨識（ASR）金鑰。請至金鑰設定（/settings/keys）儲存雲端 ASR 金鑰，儲存後立即生效。")
		return
	}

	// CR M1/L2: classify the lookup failure. A blanket 404 told a user whose
	// SQLite was locked that the episode does not exist — sending them to hunt
	// for a file that never moved. Only the not-found sentinel (and the
	// interface-permitted nil,nil) is a 404; anything else is infrastructure.
	episode, err := h.episodeService.FindByID(c.Request.Context(), id)
	switch {
	case errors.Is(err, repository.ErrEpisodeNotFound), err == nil && episode == nil:
		// A stale id from a bookmark or a re-scanned library is routine, not an
		// incident — Warn, not Error (the movie route's Error level here is
		// deliberately left alone: story red line 2).
		slog.Warn("Episode not found for transcription", "episode_id", id)
		NotFoundError(c, "Episode")
		return
	case err != nil:
		slog.Error("Failed to look up episode for transcription", "episode_id", id, "error", err)
		InternalServerError(c, "Failed to look up episode")
		return
	}

	if !episode.FilePath.Valid || episode.FilePath.String == "" {
		BadRequestError(c, "VALIDATION_REQUIRED_FIELD", "這一集沒有媒體檔案路徑——請先掃描媒體庫")
		return
	}

	if _, err := os.Stat(episode.FilePath.String); err != nil {
		BadRequestError(c, "VALIDATION_REQUIRED_FIELD", "找不到這一集的媒體檔案——請確認檔案仍在磁碟上")
		return
	}

	if h.transcriptionService.IsInProgress(id) {
		ErrorResponse(c, http.StatusConflict, "TRANSCRIPTION_IN_PROGRESS",
			"這一集的字幕生成已在執行中",
			"請等待目前的生成完成。")
		return
	}

	// WithMediaType(episode) is NOT optional: it defaults to movie, and a movie
	// default would write this episode's subtitle_status into the movies table
	// (0 rows, no error) — the badge would never flip and the next batch would
	// re-run, and re-pay for, the same episode.
	// WithTranslation is unconditional here (see the @Description note).
	mediaDir := filepath.Dir(episode.FilePath.String)
	jobID, err := h.transcriptionService.StartTranscription(
		c.Request.Context(), id, episode.FilePath.String, mediaDir,
		services.WithTranslation(),
		services.WithMediaType(models.SubtitleRunMediaEpisode),
	)
	if err != nil {
		if errors.Is(err, services.ErrTranscriptionInProgress) {
			ErrorResponse(c, http.StatusConflict, "TRANSCRIPTION_IN_PROGRESS",
				"這一集的字幕生成已在執行中",
				"請等待目前的生成完成。")
			return
		}
		slog.Error("Failed to start episode transcription", "episode_id", id, "error", err)
		InternalServerError(c, "Failed to start transcription")
		return
	}

	c.JSON(http.StatusAccepted, APIResponse{
		Success: true,
		Data: map[string]string{
			"job_id":  jobID,
			"message": "Transcription started. Listen to SSE events for progress.",
		},
	})
}
