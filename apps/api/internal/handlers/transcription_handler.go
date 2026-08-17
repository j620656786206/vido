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
	"github.com/vido/api/internal/services"
)

// TranscriptionMovieGetter defines the movie lookup needed by the transcription handler.
type TranscriptionMovieGetter interface {
	GetByID(ctx context.Context, id string) (*models.Movie, error)
}

// TranscriptionServiceInterface defines the contract for transcription operations.
type TranscriptionServiceInterface interface {
	IsAvailable() bool
	// CanResumeTranslateOnly reports whether this media would resume
	// translate-only (CR sub-2-2a M2) — such a run needs no ASR, so the
	// availability gate must not 503 it.
	CanResumeTranslateOnly(ctx context.Context, mediaID string) bool
	IsInProgress(mediaID string) bool
	StartTranscription(ctx context.Context, mediaID string, filePath string, mediaDir string, opts ...services.TranscriptionOption) (string, error)
}

// TranscriptionHandler handles transcription API requests.
type TranscriptionHandler struct {
	movieService         TranscriptionMovieGetter
	transcriptionService TranscriptionServiceInterface
}

// NewTranscriptionHandler creates a new TranscriptionHandler.
func NewTranscriptionHandler(movieService TranscriptionMovieGetter, transcriptionService TranscriptionServiceInterface) *TranscriptionHandler {
	return &TranscriptionHandler{
		movieService:         movieService,
		transcriptionService: transcriptionService,
	}
}

// RegisterRoutes registers transcription routes on the given router group.
func (h *TranscriptionHandler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.POST("/movies/:id/transcribe", h.TranscribeMovie)
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
