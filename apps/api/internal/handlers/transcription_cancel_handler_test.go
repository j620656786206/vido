package handlers

// disc-2026-10-solo-generation-cancel-api — POST …/transcribe/cancel, the
// "still stopping" 409, and the status route's `cancelling`.

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/subtitle"
)

// AC #1: the payload, exact field names, for each answer the runner gives.
func TestCancelTranscription_PipelineModePayload(t *testing.T) {
	cases := []struct {
		name   string
		result subtitle.SoloCancelResult
		want   map[string]interface{}
	}{
		{"stopped our job", subtitle.SoloCancelResult{Cancelled: true, Running: true, Owner: subtitle.SoloOwnerSolo},
			map[string]interface{}{"cancelled": true, "running": true, "owner": "solo"}},
		{"batch owns it", subtitle.SoloCancelResult{Running: true, Owner: subtitle.SoloOwnerOther},
			map[string]interface{}{"cancelled": false, "running": true, "owner": "other"}},
		{"nothing runs", subtitle.SoloCancelResult{},
			map[string]interface{}{"cancelled": false, "running": false}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			solo := &fakeSoloGenerator{cancel: tc.result}
			h := soloMovieHandler(t, solo, &mockTranscriptionService{})

			w, resp := postTranscribe(t, h, "/api/v1/movies/"+testMovieUUID+"/transcribe/cancel")

			require.Equal(t, http.StatusOK, w.Code, w.Body.String())
			assert.True(t, resp.Success)
			assert.Equal(t, tc.want, resp.Data, "owner is absent when nothing runs")
			require.Len(t, solo.cancelAsk, 1)
			assert.Equal(t, subtitle.MediaRef{ID: testMovieUUID, MediaType: models.SubtitleRunMediaMovie}, solo.cancelAsk[0])
		})
	}
}

func TestCancelTranscription_EpisodeRouteAsksAboutTheEpisodeRef(t *testing.T) {
	solo := &fakeSoloGenerator{cancel: subtitle.SoloCancelResult{Cancelled: true, Running: true, Owner: subtitle.SoloOwnerSolo}}
	h := soloEpisodeHandler(t, solo, &mockTranscriptionService{})

	w, _ := postTranscribe(t, h, "/api/v1/episodes/"+testEpisodeUUID+"/transcribe/cancel")

	require.Equal(t, http.StatusOK, w.Code)
	require.Len(t, solo.cancelAsk, 1)
	assert.Equal(t, subtitle.MediaRef{ID: testEpisodeUUID, MediaType: models.SubtitleRunMediaEpisode}, solo.cancelAsk[0])
}

func TestCancelTranscription_EpisodeRouteNotMountedWithoutAnEpisodeGetter(t *testing.T) {
	h := soloMovieHandler(t, &fakeSoloGenerator{}, &mockTranscriptionService{})
	r := setupTranscriptionRouter(h)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/episodes/"+testEpisodeUUID+"/transcribe/cancel", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

// AC #7: legacy mode never stops anything and says it cannot.
func TestCancelTranscription_LegacyModeIsNotSupported(t *testing.T) {
	for _, running := range []bool{true, false} {
		svc := &mockTranscriptionService{inProgress: running}
		movie := &models.Movie{FilePath: models.NewNullString(createTempMediaFile(t))}
		movie.ID = testMovieUUID
		h := NewTranscriptionHandler(&mockTranscriptionMovieGetter{movie: movie}, nil, svc)

		w, resp := postTranscribe(t, h, "/api/v1/movies/"+testMovieUUID+"/transcribe/cancel")

		require.Equal(t, http.StatusOK, w.Code)
		want := map[string]interface{}{"cancelled": false, "running": running}
		if running {
			want["owner"] = "legacy"
		}
		assert.Equal(t, want, resp.Data)
	}
}

// AC #6: a click while the previous run is still stopping — same code,
// different words.
func TestTranscribeMovie_WhileStoppingSays409StillStopping(t *testing.T) {
	solo := &fakeSoloGenerator{startErr: subtitle.ErrSoloGenerationCancelling}
	h := soloMovieHandler(t, solo, &mockTranscriptionService{})

	w, resp := postTranscribe(t, h, "/api/v1/movies/"+testMovieUUID+"/transcribe")

	require.Equal(t, http.StatusConflict, w.Code)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "TRANSCRIPTION_IN_PROGRESS", resp.Error.Code, "the dialog's parsing is unchanged")
	assert.Contains(t, resp.Error.Message, "正在停止")
}

// AC #6: the status route reads cancelling only while the run is in flight.
func TestTranscriptionStatus_CarriesCancelling(t *testing.T) {
	for _, tc := range []struct {
		inProgress, cancelling bool
		want                   interface{}
	}{
		{true, true, true},
		{true, false, nil},
		{false, true, nil},
	} {
		solo := &fakeSoloGenerator{inProgress: tc.inProgress, statusJob: "solo-1", cancelling: tc.cancelling}
		h := soloMovieHandler(t, solo, &mockTranscriptionService{})
		r := setupTranscriptionRouter(h)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/movies/"+testMovieUUID+"/transcribe/status", nil)
		r.ServeHTTP(w, req)
		require.Equal(t, http.StatusOK, w.Code)

		var resp APIResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Equal(t, tc.want, resp.Data.(map[string]interface{})["cancelling"], "%+v", tc)
	}
}
