package handlers

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/repository"
	"github.com/vido/api/internal/services"
	"github.com/vido/api/internal/subtitle"
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
	// estimator prices the single-item run (story dsr-6a). nil = the estimate
	// routes are not mounted.
	estimator TranscriptionEstimator
	// solo routes the click through the subtitle pipeline
	// (disc-2026-10-single-generate-ignores-embedded-english-a). nil = legacy
	// mode: the click goes to TranscriptionService exactly as before.
	solo SoloGenerator
}

// SoloGenerator is the pipeline-mode entry for one click on 生成字幕: route
// first (Chinese track → deliver, English track → translate), speech
// recognition only when the file has no usable text track. *subtitle.SoloRunner
// satisfies it. Errors come back as the services sentinels this handler
// already maps (in-progress → 409, disabled → 503) plus the pipeline's
// not-writable sentinel (409).
type SoloGenerator interface {
	Start(ctx context.Context, ref subtitle.MediaRef, modelID string) (jobID string, err error)
	Status(ref subtitle.MediaRef) (inProgress bool, jobID string)
}

// TranscriptionEstimator prices what a click on 生成字幕 would start (story
// dsr-6a AC #2). *services.TranscriptionEstimateService satisfies it.
type TranscriptionEstimator interface {
	Estimate(ctx context.Context, target services.TranscriptionEstimateTarget) services.TranscriptionEstimate
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

// SetEstimator wires the single-item price (story dsr-6a AC #2). A setter, not a
// constructor parameter, so the handler's many existing call sites and fakes
// stay valid; call it BEFORE RegisterRoutes — an unwired estimator leaves the
// estimate routes unmounted (404), the same capability honor the episode route
// uses.
func (h *TranscriptionHandler) SetEstimator(e TranscriptionEstimator) {
	h.estimator = e
}

// SetSoloGenerator switches the transcribe and status routes onto the subtitle
// pipeline (pipeline mode). A setter for the same reason SetEstimator is one:
// the constructor's many call sites and fakes stay valid, and an unwired
// handler keeps the legacy behaviour byte-for-byte. Call it BEFORE
// RegisterRoutes.
func (h *TranscriptionHandler) SetSoloGenerator(g SoloGenerator) {
	h.solo = g
}

// RegisterRoutes registers transcription routes on the given router group.
//
// The per-episode route (story 9R-10a) is mounted ONLY when an episode getter
// is wired. Capability honor: an unwired deployment answers 404 ("this route
// does not exist here") instead of panicking on a nil lookup or inventing a
// 503 that would blame the ASR configuration for a wiring mistake.
func (h *TranscriptionHandler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.POST("/movies/:id/transcribe", h.TranscribeMovie)
	rg.GET("/movies/:id/transcribe/status", h.transcriptionStatusFor(models.SubtitleRunMediaMovie))
	if h.episodeService != nil {
		rg.POST("/episodes/:id/transcribe", h.TranscribeEpisode)
		rg.GET("/episodes/:id/transcribe/status", h.transcriptionStatusFor(models.SubtitleRunMediaEpisode))
	}
	if h.estimator != nil {
		rg.GET("/movies/:id/transcribe/estimate", h.EstimateMovie)
		if h.episodeService != nil {
			rg.GET("/episodes/:id/transcribe/estimate", h.EstimateEpisode)
		}
	}
}

// TranscribeMovie triggers transcription for a movie.
// POST /api/v1/movies/:id/transcribe
// Returns 202 Accepted with job ID; 409 SUBTITLE_TARGET_NOT_WRITABLE when the
// movie folder refuses a write probe (refused before any paid call).
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
	// disk proceeds even when FFmpeg/ASR are gone. In pipeline mode the
	// SoloRunner judges this itself — a file with a Chinese or English track
	// needs no ASR at all — so the legacy gate is skipped.
	if h.solo == nil && !h.transcriptionService.IsAvailable() &&
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

	movie, ok := h.lookupMovieFile(c, id)
	if !ok {
		return
	}

	// Pipeline mode: the click routes like the batch does. `?translate=true`
	// is ignored on purpose — the pipeline always ends in a zh-Hant sidecar,
	// and the dialog has always sent translate=true anyway.
	if h.solo != nil {
		h.startSolo(c, subtitle.MediaRef{ID: id, MediaType: models.SubtitleRunMediaMovie}, "movie")
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
		if errors.Is(err, services.ErrTranscriptionTargetNotWritable) {
			targetNotWritableError(c, mediaDir)
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
// @Failure      409  {object}  APIResponse  "TRANSCRIPTION_IN_PROGRESS — a run for this episode is already in flight / SUBTITLE_TARGET_NOT_WRITABLE — the episode folder refused a write probe; refused before any paid call"
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
	if h.solo == nil && !h.transcriptionService.IsAvailable() &&
		!h.transcriptionService.CanResumeEpisodeTranslateOnly(c.Request.Context(), id) {
		ErrorResponse(c, http.StatusServiceUnavailable, "TRANSCRIPTION_DISABLED",
			"語音辨識尚未設定",
			"生成字幕需要雲端語音辨識（ASR）金鑰。請至金鑰設定（/settings/keys）儲存雲端 ASR 金鑰，儲存後立即生效。")
		return
	}

	episode, ok := h.lookupEpisodeFile(c, id)
	if !ok {
		return
	}

	if h.solo != nil {
		h.startSolo(c, subtitle.MediaRef{ID: id, MediaType: models.SubtitleRunMediaEpisode}, "episode")
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
		if errors.Is(err, services.ErrTranscriptionTargetNotWritable) {
			targetNotWritableError(c, mediaDir)
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

// startSolo is the pipeline-mode body shared by both transcribe routes. The
// media row was already resolved and its file checked on disk by the caller;
// the runner loads it again itself (it needs the path and the title). Wire
// codes are the legacy ones so the dialog's outcome parsing
// (transcriptionService.ts parseTranscribeResponse) needs no change.
func (h *TranscriptionHandler) startSolo(c *gin.Context, ref subtitle.MediaRef, noun string) {
	jobID, err := h.solo.Start(c.Request.Context(), ref, "")
	if err != nil {
		switch {
		case errors.Is(err, services.ErrTranscriptionInProgress):
			subject := "這部影片"
			if ref.MediaType == models.SubtitleRunMediaEpisode {
				subject = "這一集"
			}
			ErrorResponse(c, http.StatusConflict, "TRANSCRIPTION_IN_PROGRESS",
				subject+"的字幕生成已在執行中",
				"請等待目前的生成完成。")
		case errors.Is(err, services.ErrTranscriptionDisabled):
			ErrorResponse(c, http.StatusServiceUnavailable, "TRANSCRIPTION_DISABLED",
				"語音辨識尚未設定",
				"這部影片沒有可用的片內字幕，要生成字幕需要雲端語音辨識（ASR）金鑰。請至金鑰設定（/settings/keys）儲存雲端 ASR 金鑰，儲存後立即生效。")
		case errors.Is(err, services.ErrTranscriptionTargetNotWritable), errors.Is(err, subtitle.ErrSubtitleTargetNotWritable):
			// The runner does not know the folder; name it from the error's
			// own detail would leak the absolute path, so the generic line is
			// composed without a base name here.
			ErrorResponse(c, http.StatusConflict, "SUBTITLE_TARGET_NOT_WRITABLE",
				"影片所在的資料夾寫不進字幕，這次沒有花到錢。請到 NAS 確認 Vido 能寫入這個資料夾，再按重試",
				"確認這個資料夾不是唯讀掛載，而且 Vido 容器的使用者有寫入權限。")
		default:
			slog.Error("Failed to start subtitle generation", noun+"_id", ref.ID, "error", err)
			InternalServerError(c, "Failed to start transcription")
		}
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

// targetNotWritableError answers a run refused because the subtitle could
// not be written next to the video (disc-2026-09-solo-run-unwritable-folder-pays-asr).
// Nothing was charged. The dialog shows only the message, after
// 「無法開始生成：」, so the message carries the fix itself; it names the folder
// by its base name only — never the absolute path — as the consent list's
// 資料夾無法寫入 row does.
func targetNotWritableError(c *gin.Context, mediaDir string) {
	ErrorResponse(c, http.StatusConflict, "SUBTITLE_TARGET_NOT_WRITABLE",
		fmt.Sprintf("影片所在的資料夾「%s」寫不進字幕，這次沒有花到錢。請到 NAS 確認 Vido 能寫入這個資料夾，再按重試", filepath.Base(mediaDir)),
		"確認這個資料夾不是唯讀掛載，而且 Vido 容器的使用者有寫入權限。")
}

// lookupMovieFile resolves the movie and checks its file is on disk, writing the
// error response itself (ok=false means the response is already written).
// Shared by the trigger and the estimate so the two answer a missing file with
// the same status and the same words.
func (h *TranscriptionHandler) lookupMovieFile(c *gin.Context, id string) (*models.Movie, bool) {
	movie, err := h.movieService.GetByID(c.Request.Context(), id)
	if err != nil {
		slog.Error("Failed to get movie for transcription", "id", id, "error", err)
		NotFoundError(c, "Movie")
		return nil, false
	}

	// Validate file_path exists
	if !movie.FilePath.Valid || movie.FilePath.String == "" {
		BadRequestError(c, "VALIDATION_REQUIRED_FIELD", "Movie has no file path — scan the media library first")
		return nil, false
	}

	// Validate file is accessible on disk (AC #1, task 4.3)
	if _, err := os.Stat(movie.FilePath.String); err != nil {
		BadRequestError(c, "VALIDATION_REQUIRED_FIELD", "Movie file not accessible — check if the file exists on disk")
		return nil, false
	}
	return movie, true
}

// lookupEpisodeFile is the episode counterpart of lookupMovieFile.
func (h *TranscriptionHandler) lookupEpisodeFile(c *gin.Context, id string) (*models.Episode, bool) {
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
		return nil, false
	case err != nil:
		slog.Error("Failed to look up episode for transcription", "episode_id", id, "error", err)
		InternalServerError(c, "Failed to look up episode")
		return nil, false
	}

	if !episode.FilePath.Valid || episode.FilePath.String == "" {
		BadRequestError(c, "VALIDATION_REQUIRED_FIELD", "這一集沒有媒體檔案路徑——請先掃描媒體庫")
		return nil, false
	}

	if _, err := os.Stat(episode.FilePath.String); err != nil {
		BadRequestError(c, "VALIDATION_REQUIRED_FIELD", "找不到這一集的媒體檔案——請確認檔案仍在磁碟上")
		return nil, false
	}
	return episode, true
}

// TranscriptionStatusResponse is the GET …/transcribe/status payload.
type TranscriptionStatusResponse struct {
	InProgress bool `json:"in_progress"`
	// JobID is the running solo job's id when the run was started by a click
	// in pipeline mode — the dialog attaches to THAT job's terminal event.
	// Absent for a batch / pool run and in legacy mode (additive,
	// disc-2026-10-single-generate-ignores-embedded-english-a).
	JobID string `json:"job_id,omitempty"`
}

// TranscriptionStatus answers whether a subtitle generation is running for one
// movie or episode right now (story bugfix-dialog-reopen-shows-idle-during-run).
// The 管理字幕 dialog asks on open, so a reopen during a run shows the progress
// instead of a paid 生成字幕 button. It reads the same single-flight table the
// two POST routes answer 409 from — solo and batch runs alike — and touches no
// database: an unknown id is simply "not running".
//
// @Summary      Is subtitle generation running for this movie or episode?
// @Description  Reads the in-memory single-flight table behind the 409 TRANSCRIPTION_IN_PROGRESS of POST /movies/{id}/transcribe and POST /episodes/{id}/transcribe. Spends nothing, starts nothing, does not look the media up.
// @Tags         subtitles
// @Produce      json
// @Param        id path string true "Movie or episode ID (UUID)"
// @Success      200 {object} APIResponse "data: {in_progress: bool, job_id?: string} — job_id only for a running solo click in pipeline mode"
// @Router       /api/v1/movies/{id}/transcribe/status [get]
// @Router       /api/v1/episodes/{id}/transcribe/status [get]
func (h *TranscriptionHandler) TranscriptionStatus(c *gin.Context) {
	h.transcriptionStatusFor(models.SubtitleRunMediaMovie)(c)
}

// transcriptionStatusFor binds the status route to its media type: in pipeline
// mode the in-flight set is keyed by (type, id), so the movie and episode
// routes must ask about different refs even when the ids collide.
func (h *TranscriptionHandler) transcriptionStatusFor(mediaType string) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if h.solo != nil {
			// Pipeline mode: the pool's in-flight set is the truth (solo click,
			// batch and workers all reserve there); the legacy table only sees
			// the ASR leg and would read idle while a track is being translated.
			inProgress, jobID := h.solo.Status(subtitle.MediaRef{ID: id, MediaType: mediaType})
			c.JSON(http.StatusOK, APIResponse{
				Success: true,
				Data:    TranscriptionStatusResponse{InProgress: inProgress, JobID: jobID},
			})
			return
		}
		c.JSON(http.StatusOK, APIResponse{
			Success: true,
			Data:    TranscriptionStatusResponse{InProgress: h.transcriptionService.IsInProgress(id)},
		})
	}
}

// EstimateMovie prices a click on 生成字幕 for one movie.
//
// @Summary      Estimate the cost of generating subtitles for one movie
// @Description  Prices what POST /movies/{id}/transcribe?translate=true would actually do — speech recognition + translation, or translation only when an untranslated English SRT can be resumed — so the 管理字幕 dialog can show the amount on the button before anything is spent (story dsr-6a). Spends nothing and starts no job. It is NOT gated on speech-recognition availability: asr_available=false still returns the price, and the client disables the button. A movie with no stored duration is probed with ffprobe (bounded by the shared ffprobe limit), then falls back to the TMDb runtime and finally a stated 45-minute assumption (runtime_source=fallback).
// @Tags         subtitles
// @Produce      json
// @Param        id path string true "Movie ID (UUID)"
// @Success      200 {object} APIResponse "data: {media_id, media_type, plan: full|translate_only|extract, route?: extract|asr|skip (pipeline mode only), asr_available, self_hosted_asr, translation_configured, model_id, runtime_minutes, runtime_known, runtime_source: ffprobe|tmdb|fallback, estimated_usd}"
// @Failure      400 {object} APIResponse "VALIDATION_REQUIRED_FIELD — the movie has no file path, or the file is not on disk"
// @Failure      404 {object} APIResponse "DB_NOT_FOUND — no such movie"
// @Router       /api/v1/movies/{id}/transcribe/estimate [get]
func (h *TranscriptionHandler) EstimateMovie(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		BadRequestError(c, "VALIDATION_INVALID_FORMAT", "Invalid movie ID")
		return
	}
	movie, ok := h.lookupMovieFile(c, id)
	if !ok {
		return
	}
	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data: h.estimator.Estimate(c.Request.Context(), services.TranscriptionEstimateTarget{
			MediaID:            id,
			MediaType:          models.SubtitleRunMediaMovie,
			FilePath:           movie.FilePath.String,
			DurationSeconds:    movie.DurationSeconds,
			Runtime:            movie.Runtime,
			SubtitleTracksJSON: movie.SubtitleTracks.String,
		}),
	})
}

// EstimateEpisode prices a click on 生成字幕 for one episode.
//
// @Summary      Estimate the cost of generating subtitles for one episode
// @Description  The episode counterpart of the movie estimate (story dsr-6a): prices what POST /episodes/{id}/transcribe would actually do. Episodes rarely have a stored duration, so this usually probes the file with ffprobe and remembers the measured length for next time. Spends nothing and starts no job.
// @Tags         subtitles
// @Produce      json
// @Param        id path string true "Episode ID (UUID)"
// @Success      200 {object} APIResponse "data: same shape as the movie estimate, media_type=episode"
// @Failure      400 {object} APIResponse "VALIDATION_REQUIRED_FIELD — the episode has no file path, or the file is not on disk"
// @Failure      404 {object} APIResponse "DB_NOT_FOUND — no such episode"
// @Failure      500 {object} APIResponse "INTERNAL_ERROR — the episode lookup itself failed"
// @Router       /api/v1/episodes/{id}/transcribe/estimate [get]
func (h *TranscriptionHandler) EstimateEpisode(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		BadRequestError(c, "VALIDATION_INVALID_FORMAT", "影集單集 ID 無效")
		return
	}
	episode, ok := h.lookupEpisodeFile(c, id)
	if !ok {
		return
	}
	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data: h.estimator.Estimate(c.Request.Context(), services.TranscriptionEstimateTarget{
			MediaID:         id,
			MediaType:       models.SubtitleRunMediaEpisode,
			FilePath:        episode.FilePath.String,
			DurationSeconds: episode.DurationSeconds,
			Runtime:         episode.Runtime,
		}),
	})
}
