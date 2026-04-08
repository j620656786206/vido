package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/services"
)

// ─── Mock implementations ─────────────────────────────────────────────────

type mockTranscriptionMovieGetter struct {
	movie *models.Movie
	err   error
}

func (m *mockTranscriptionMovieGetter) GetByID(_ context.Context, _ string) (*models.Movie, error) {
	return m.movie, m.err
}

type mockTranscriptionService struct {
	available  bool
	inProgress bool
	jobID      string
	startErr   error
}

func (m *mockTranscriptionService) IsAvailable() bool {
	return m.available
}

func (m *mockTranscriptionService) IsInProgress(_ int64) bool {
	return m.inProgress
}

func (m *mockTranscriptionService) StartTranscription(_ context.Context, _ int64, _ string, _ string) (string, error) {
	return m.jobID, m.startErr
}

// ─── Tests ────────────────────────────────────────────────────────────────

func setupTranscriptionRouter(h *TranscriptionHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	api := r.Group("/api/v1")
	h.RegisterRoutes(api)
	return r
}

func TestTranscribeMovie_Success(t *testing.T) {
	movie := &models.Movie{
		FilePath: models.NewNullString("/media/movies/test.mkv"),
	}
	movie.ID = "42"

	h := NewTranscriptionHandler(
		&mockTranscriptionMovieGetter{movie: movie},
		&mockTranscriptionService{available: true, jobID: "job-123"},
	)

	r := setupTranscriptionRouter(h)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/movies/42/transcribe", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusAccepted, w.Code)

	var resp APIResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.True(t, resp.Success)

	data := resp.Data.(map[string]interface{})
	assert.Equal(t, "job-123", data["job_id"])
}

func TestTranscribeMovie_ServiceUnavailable(t *testing.T) {
	h := NewTranscriptionHandler(
		&mockTranscriptionMovieGetter{},
		&mockTranscriptionService{available: false},
	)

	r := setupTranscriptionRouter(h)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/movies/1/transcribe", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestTranscribeMovie_InvalidID(t *testing.T) {
	h := NewTranscriptionHandler(
		&mockTranscriptionMovieGetter{},
		&mockTranscriptionService{available: true},
	)

	r := setupTranscriptionRouter(h)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/movies/abc/transcribe", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestTranscribeMovie_MovieNotFound(t *testing.T) {
	h := NewTranscriptionHandler(
		&mockTranscriptionMovieGetter{err: errors.New("not found")},
		&mockTranscriptionService{available: true},
	)

	r := setupTranscriptionRouter(h)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/movies/1/transcribe", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestTranscribeMovie_NoFilePath(t *testing.T) {
	movie := &models.Movie{
		FilePath: models.NullString{},
	}
	movie.ID = "1"

	h := NewTranscriptionHandler(
		&mockTranscriptionMovieGetter{movie: movie},
		&mockTranscriptionService{available: true},
	)

	r := setupTranscriptionRouter(h)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/movies/1/transcribe", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestTranscribeMovie_AlreadyInProgress(t *testing.T) {
	movie := &models.Movie{
		FilePath: models.NewNullString("/media/test.mkv"),
	}
	movie.ID = "1"

	h := NewTranscriptionHandler(
		&mockTranscriptionMovieGetter{movie: movie},
		&mockTranscriptionService{available: true, inProgress: true},
	)

	r := setupTranscriptionRouter(h)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/movies/1/transcribe", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)
}

func TestTranscribeMovie_StartError(t *testing.T) {
	movie := &models.Movie{
		FilePath: models.NewNullString("/media/test.mkv"),
	}
	movie.ID = "1"

	h := NewTranscriptionHandler(
		&mockTranscriptionMovieGetter{movie: movie},
		&mockTranscriptionService{
			available: true,
			startErr:  services.ErrTranscriptionInProgress,
		},
	)

	r := setupTranscriptionRouter(h)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/movies/1/transcribe", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)
}

func TestTranscribeMovie_InternalError(t *testing.T) {
	movie := &models.Movie{
		FilePath: models.NewNullString("/media/test.mkv"),
	}
	movie.ID = "1"

	h := NewTranscriptionHandler(
		&mockTranscriptionMovieGetter{movie: movie},
		&mockTranscriptionService{
			available: true,
			startErr:  errors.New("unexpected error"),
		},
	)

	r := setupTranscriptionRouter(h)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/movies/1/transcribe", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
