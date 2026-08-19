package handlers

// Story 9R-10a — the per-episode manual transcribe entry point.
//
// Deliberately a SEPARATE file from transcription_handler_test.go: the movie
// route and its tests are explicitly out of scope for 9R-10a (story red line
// 2 — "this story is an addition, not a refactor"), so nothing here edits a
// movie assertion.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/repository"
	"github.com/vido/api/internal/services"
)

// Media-id fixture convention (9R-18 AC 7): media ids are UUID STRINGS.
const testEpisodeUUID = "9c3b7e21-4d5a-4b8c-9f01-2e3d4c5b6a7f"

type mockTranscriptionEpisodeGetter struct {
	episode *models.Episode
	err     error
}

func (m *mockTranscriptionEpisodeGetter) FindByID(_ context.Context, _ string) (*models.Episode, error) {
	return m.episode, m.err
}

// setupEpisodeTranscriptionRouter mounts the handler with BOTH getters wired.
func setupEpisodeTranscriptionRouter(h *TranscriptionHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h.RegisterRoutes(r.Group("/api/v1"))
	return r
}

func newEpisodeHandler(ep *models.Episode, epErr error, svc *mockTranscriptionService) *TranscriptionHandler {
	return NewTranscriptionHandler(
		&mockTranscriptionMovieGetter{},
		&mockTranscriptionEpisodeGetter{episode: ep, err: epErr},
		svc,
	)
}

func episodeWithFile(t *testing.T) *models.Episode {
	t.Helper()
	return &models.Episode{
		ID:            testEpisodeUUID,
		SeriesID:      "series-1",
		SeasonNumber:  1,
		EpisodeNumber: 3,
		FilePath:      models.NewNullString(createTempMediaFile(t)),
	}
}

func postEpisodeTranscribe(h *TranscriptionHandler, id string) *httptest.ResponseRecorder {
	r := setupEpisodeTranscriptionRouter(h)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/episodes/"+id+"/transcribe", nil)
	r.ServeHTTP(w, req)
	return w
}

// ── AC #5 example 1 — happy path ──────────────────────────────────────────

func TestTranscribeEpisode_Success(t *testing.T) {
	mockSvc := &mockTranscriptionService{available: true, jobID: "job-ep-1"}
	w := postEpisodeTranscribe(newEpisodeHandler(episodeWithFile(t), nil, mockSvc), testEpisodeUUID)

	assert.Equal(t, http.StatusAccepted, w.Code)

	var resp APIResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.True(t, resp.Success)
	assert.Equal(t, "job-ep-1", resp.Data.(map[string]interface{})["job_id"])

	// The UUID reaches the service untouched (9R-18).
	assert.Equal(t, testEpisodeUUID, mockSvc.receivedMediaID)
}

// ── AC #5 examples 2 & 3 — the options actually sent ──────────────────────

// TestTranscribeEpisode_SendsEpisodeMediaTypeAndTranslation is the data-corruption
// tripwire. WithMediaType silently defaults to MOVIE
// (transcription_service.go newTranscriptionConfig), so a handler that forgets
// it writes an episode's subtitle_status into the MOVIES table — 0 rows, no
// error, and the user sees "it finished but the badge never flipped" while the
// next batch re-runs (and re-pays for) the same episode.
// generation_batch_runner.go:17 carries the same warning for the batch path.
func TestTranscribeEpisode_SendsEpisodeMediaTypeAndTranslation(t *testing.T) {
	mockSvc := &mockTranscriptionService{available: true, jobID: "job-ep-2"}
	w := postEpisodeTranscribe(newEpisodeHandler(episodeWithFile(t), nil, mockSvc), testEpisodeUUID)
	require.Equal(t, http.StatusAccepted, w.Code)

	translate, mediaType := services.TranscriptionOptionsFor(mockSvc.receivedOpts)
	assert.Equal(t, models.SubtitleRunMediaEpisode, mediaType,
		"WithMediaType(episode) missing — the run would write into the movies table")
	assert.True(t, translate,
		"WithTranslation() missing — the run would stop at an English-only SRT")
}

// ── AC #5 examples 4 & 5 — file preconditions ─────────────────────────────

func TestTranscribeEpisode_NoFilePath(t *testing.T) {
	ep := &models.Episode{ID: testEpisodeUUID, SeriesID: "series-1", SeasonNumber: 1, EpisodeNumber: 3}
	mockSvc := &mockTranscriptionService{available: true}
	w := postEpisodeTranscribe(newEpisodeHandler(ep, nil, mockSvc), testEpisodeUUID)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Empty(t, mockSvc.receivedMediaID, "must not start a run without a file")
}

func TestTranscribeEpisode_FileNotAccessible(t *testing.T) {
	ep := &models.Episode{
		ID: testEpisodeUUID, SeriesID: "series-1", SeasonNumber: 1, EpisodeNumber: 3,
		FilePath: models.NewNullString("/nonexistent/path/S01E03.mkv"),
	}
	mockSvc := &mockTranscriptionService{available: true}
	w := postEpisodeTranscribe(newEpisodeHandler(ep, nil, mockSvc), testEpisodeUUID)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Empty(t, mockSvc.receivedMediaID)
}

// ── AC #5 examples 6 & 7 — the resume-aware availability gate ─────────────

func TestTranscribeEpisode_Unavailable_NoResume_503(t *testing.T) {
	mockSvc := &mockTranscriptionService{available: false, canResumeEpisode: false}
	w := postEpisodeTranscribe(newEpisodeHandler(episodeWithFile(t), nil, mockSvc), testEpisodeUUID)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)

	var resp APIResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.NotNil(t, resp.Error)
	assert.Equal(t, "TRANSCRIPTION_DISABLED", resp.Error.Code)
}

// TestTranscribeEpisode_Unavailable_WithResume_Proceeds is story red line 3.
// An `untranslated` episode whose English SRT is still on disk resumes
// TRANSLATE-ONLY — it needs no speech recognition at all, so an ASR-less
// deployment must not 503 it. Reusing the MOVIE resume check here would query
// the movies table with an episode id, return false forever, and reproduce the
// exact bug CR sub-2-2a M2 fixed for movies.
func TestTranscribeEpisode_Unavailable_WithResume_Proceeds(t *testing.T) {
	mockSvc := &mockTranscriptionService{available: false, canResumeEpisode: true, jobID: "job-ep-resume"}
	w := postEpisodeTranscribe(newEpisodeHandler(episodeWithFile(t), nil, mockSvc), testEpisodeUUID)

	assert.Equal(t, http.StatusAccepted, w.Code)
	assert.Equal(t, testEpisodeUUID, mockSvc.receivedMediaID)
}

// TestTranscribeEpisode_ResumeGateIsEpisodeScoped pins that the episode route
// consults the EPISODE resume check, not the movie one — the two mock flags are
// set to opposite values so a wrong-gate implementation flips the outcome.
func TestTranscribeEpisode_ResumeGateIsEpisodeScoped(t *testing.T) {
	mockSvc := &mockTranscriptionService{available: false, canResume: true, canResumeEpisode: false}
	w := postEpisodeTranscribe(newEpisodeHandler(episodeWithFile(t), nil, mockSvc), testEpisodeUUID)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code,
		"episode route must consult CanResumeEpisodeTranslateOnly, not the movie check")
}

// ── remaining gates ───────────────────────────────────────────────────────

// TestTranscribeEpisode_NotFound — the repository's not-found sentinel is the
// ONLY lookup failure that means "this episode does not exist".
func TestTranscribeEpisode_NotFound(t *testing.T) {
	mockSvc := &mockTranscriptionService{available: true}
	notFound := fmt.Errorf("episode with id %s: %w", testEpisodeUUID, repository.ErrEpisodeNotFound)
	w := postEpisodeTranscribe(newEpisodeHandler(nil, notFound, mockSvc), testEpisodeUUID)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// TestTranscribeEpisode_NilWithoutError covers the (nil, nil) shape the narrow
// port permits even though *repository.EpisodeRepository never returns it
// (CR L3) — a future implementation of the port must not crash the handler.
func TestTranscribeEpisode_NilWithoutError(t *testing.T) {
	mockSvc := &mockTranscriptionService{available: true}
	w := postEpisodeTranscribe(newEpisodeHandler(nil, nil, mockSvc), testEpisodeUUID)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// TestTranscribeEpisode_LookupFailureIs500 is CR M1's regression nail. A locked
// or unreachable database is NOT "episode not found": answering 404 sends the
// user hunting for a file that never moved, and hides an infrastructure fault
// behind a routine-looking response.
func TestTranscribeEpisode_LookupFailureIs500(t *testing.T) {
	mockSvc := &mockTranscriptionService{available: true}
	w := postEpisodeTranscribe(newEpisodeHandler(nil, errors.New("database is locked"), mockSvc), testEpisodeUUID)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Empty(t, mockSvc.receivedMediaID, "must not start a run when the lookup failed")
}

// TestTranscribeEpisode_EmptyID — CR M2. `:id` is an opaque STRING (9R-18), so
// the only 400-able format problem is an EMPTY id. gin never routes to it, so
// the branch is exercised through a hand-built context, exactly as the movie
// route's twin does.
func TestTranscribeEpisode_EmptyID(t *testing.T) {
	h := newEpisodeHandler(nil, nil, &mockTranscriptionService{available: true})

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest(http.MethodPost, "/api/v1/episodes//transcribe", nil)
	c.Params = gin.Params{{Key: "id", Value: ""}}

	h.TranscribeEpisode(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp APIResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.NotNil(t, resp.Error)
	assert.Equal(t, "VALIDATION_INVALID_FORMAT", resp.Error.Code)
}

func TestTranscribeEpisode_AlreadyInProgress(t *testing.T) {
	mockSvc := &mockTranscriptionService{available: true, inProgress: true}
	w := postEpisodeTranscribe(newEpisodeHandler(episodeWithFile(t), nil, mockSvc), testEpisodeUUID)

	assert.Equal(t, http.StatusConflict, w.Code)

	var resp APIResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.NotNil(t, resp.Error)
	assert.Equal(t, "TRANSCRIPTION_IN_PROGRESS", resp.Error.Code)
}

// TestTranscribeEpisode_RouteAbsentWithoutGetter — capability honor: a handler
// built without an episode getter simply does not expose the route (404 from
// the router) rather than panicking on a nil lookup.
func TestTranscribeEpisode_RouteAbsentWithoutGetter(t *testing.T) {
	h := NewTranscriptionHandler(&mockTranscriptionMovieGetter{}, nil, &mockTranscriptionService{available: true})
	w := postEpisodeTranscribe(h, testEpisodeUUID)

	assert.Equal(t, http.StatusNotFound, w.Code)
}
