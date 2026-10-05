package handlers

// Story bugfix-dialog-reopen-shows-idle-during-run — the 管理字幕 dialog asks
// "is this one generating right now?" when it opens, so a reopen during a run
// shows the progress instead of a paid 生成字幕 button.

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type transcriptionStatusBody struct {
	Success bool `json:"success"`
	Data    struct {
		InProgress *bool `json:"in_progress"`
	} `json:"data"`
}

func getTranscriptionStatus(t *testing.T, h *TranscriptionHandler, path string) (*httptest.ResponseRecorder, transcriptionStatusBody) {
	t.Helper()
	r := setupTranscriptionRouter(h)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, path, nil)
	r.ServeHTTP(w, req)
	var body transcriptionStatusBody
	if w.Code == http.StatusOK {
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	}
	return w, body
}

func TestTranscriptionStatus_Movie(t *testing.T) {
	for _, running := range []bool{true, false} {
		svc := &mockTranscriptionService{inProgress: running}
		h := NewTranscriptionHandler(&mockTranscriptionMovieGetter{}, nil, svc)

		w, body := getTranscriptionStatus(t, h, "/api/v1/movies/"+testMovieUUID+"/transcribe/status")

		require.Equal(t, http.StatusOK, w.Code, "running=%v", running)
		assert.True(t, body.Success)
		require.NotNil(t, body.Data.InProgress, "in_progress must be present even when false")
		assert.Equal(t, running, *body.Data.InProgress)
		assert.Equal(t, testMovieUUID, svc.inProgressAsked, "asks about the id in the path")
	}
}

func TestTranscriptionStatus_Episode(t *testing.T) {
	for _, running := range []bool{true, false} {
		svc := &mockTranscriptionService{inProgress: running}
		h := newEpisodeHandler(nil, nil, svc)

		w, body := getTranscriptionStatus(t, h, "/api/v1/episodes/"+testEpisodeUUID+"/transcribe/status")

		require.Equal(t, http.StatusOK, w.Code, "running=%v", running)
		assert.True(t, body.Success)
		require.NotNil(t, body.Data.InProgress, "in_progress must be present even when false")
		assert.Equal(t, running, *body.Data.InProgress)
		assert.Equal(t, testEpisodeUUID, svc.inProgressAsked, "asks about the id in the path")
	}
}

// The status route never looks the media up: answering "not running" for an
// id that does not exist is true, and a DB read here would slow every open.
func TestTranscriptionStatus_DoesNotLookUpMedia(t *testing.T) {
	svc := &mockTranscriptionService{}
	h := NewTranscriptionHandler(&mockTranscriptionMovieGetter{err: assert.AnError}, nil, svc)

	w, body := getTranscriptionStatus(t, h, "/api/v1/movies/"+testMovieUUID+"/transcribe/status")

	require.Equal(t, http.StatusOK, w.Code)
	require.NotNil(t, body.Data.InProgress)
	assert.False(t, *body.Data.InProgress)
}

// Capability honor, same as POST /episodes/:id/transcribe: no episode getter
// wired → the episode route is not mounted.
func TestTranscriptionStatus_EpisodeRouteNotMountedWithoutEpisodeGetter(t *testing.T) {
	h := NewTranscriptionHandler(&mockTranscriptionMovieGetter{}, nil, &mockTranscriptionService{inProgress: true})

	w, _ := getTranscriptionStatus(t, h, "/api/v1/episodes/"+testEpisodeUUID+"/transcribe/status")

	assert.Equal(t, http.StatusNotFound, w.Code)
}
