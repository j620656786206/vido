package handlers

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

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
	IsInProgress(mediaID int64) bool
	StartTranscription(ctx context.Context, mediaID int64, filePath string, mediaDir string) (string, error)
}

// TranscriptionHandler handles transcription API requests.
type TranscriptionHandler struct {
	movieService        TranscriptionMovieGetter
	transcriptionService TranscriptionServiceInterface
}

// NewTranscriptionHandler creates a new TranscriptionHandler.
func NewTranscriptionHandler(movieService TranscriptionMovieGetter, transcriptionService TranscriptionServiceInterface) *TranscriptionHandler {
	return &TranscriptionHandler{
		movieService:        movieService,
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
	// Check if transcription feature is available
	if !h.transcriptionService.IsAvailable() {
		ErrorResponse(c, http.StatusServiceUnavailable, "TRANSCRIPTION_DISABLED",
			"Transcription is not available",
			"Ensure FFmpeg is installed and OPENAI_API_KEY is configured.")
		return
	}

	// Validate movie ID
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		BadRequestError(c, "VALIDATION_INVALID_FORMAT", "Invalid movie ID")
		return
	}

	// Fetch movie
	movie, err := h.movieService.GetByID(c.Request.Context(), idStr)
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

	// Start async transcription
	mediaDir := filepath.Dir(movie.FilePath.String)
	jobID, err := h.transcriptionService.StartTranscription(c.Request.Context(), id, movie.FilePath.String, mediaDir)
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
