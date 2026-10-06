package handlers

// disc-2026-10-single-generate-ignores-embedded-english-a — in pipeline mode
// the detail-page 生成字幕 click goes through the SoloGenerator port (the
// subtitle.SoloRunner) instead of TranscriptionService. Without the port the
// handler is byte-for-byte the legacy one — the pre-existing tests in this
// package prove that; these prove the routed half.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/services"
	"github.com/vido/api/internal/subtitle"
)

type fakeSoloGenerator struct {
	jobID      string
	startErr   error
	started    []subtitle.MediaRef
	models     []string
	inProgress bool
	statusJob  string
	statusAsk  []subtitle.MediaRef
}

func (f *fakeSoloGenerator) Start(_ context.Context, ref subtitle.MediaRef, modelID string) (string, error) {
	f.started = append(f.started, ref)
	f.models = append(f.models, modelID)
	return f.jobID, f.startErr
}

func (f *fakeSoloGenerator) Status(ref subtitle.MediaRef) (bool, string) {
	f.statusAsk = append(f.statusAsk, ref)
	return f.inProgress, f.statusJob
}

func soloMovieHandler(t *testing.T, solo *fakeSoloGenerator, svc *mockTranscriptionService) *TranscriptionHandler {
	t.Helper()
	movie := &models.Movie{FilePath: models.NewNullString(createTempMediaFile(t))}
	movie.ID = testMovieUUID
	h := NewTranscriptionHandler(&mockTranscriptionMovieGetter{movie: movie}, nil, svc)
	h.SetSoloGenerator(solo)
	return h
}

func soloEpisodeHandler(t *testing.T, solo *fakeSoloGenerator, svc *mockTranscriptionService) *TranscriptionHandler {
	t.Helper()
	h := newEpisodeHandler(episodeWithFile(t), nil, svc)
	h.SetSoloGenerator(solo)
	return h
}

func postTranscribe(t *testing.T, h *TranscriptionHandler, path string) (*httptest.ResponseRecorder, APIResponse) {
	t.Helper()
	r := setupTranscriptionRouter(h)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, path, nil)
	r.ServeHTTP(w, req)
	var resp APIResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	return w, resp
}

// ─── AC #1: both routes drive the solo generator, legacy gates do not apply ─

func TestTranscribeMovie_WithSoloGeneratorRoutesThroughThePipeline(t *testing.T) {
	solo := &fakeSoloGenerator{jobID: "solo-job-1"}
	// ASR unavailable and no resume: the legacy gate would 503. The routed
	// click lets the runner decide (a Chinese or English track needs no ASR).
	svc := &mockTranscriptionService{available: false, canResume: false, jobID: "legacy-job"}
	h := soloMovieHandler(t, solo, svc)

	w, resp := postTranscribe(t, h, "/api/v1/movies/"+testMovieUUID+"/transcribe?translate=true")

	require.Equal(t, http.StatusAccepted, w.Code, w.Body.String())
	assert.True(t, resp.Success)
	data := resp.Data.(map[string]interface{})
	assert.Equal(t, "solo-job-1", data["job_id"])
	assert.NotEmpty(t, data["message"])

	require.Len(t, solo.started, 1)
	assert.Equal(t, subtitle.MediaRef{ID: testMovieUUID, MediaType: models.SubtitleRunMediaMovie}, solo.started[0])
	assert.Equal(t, "", solo.models[0], "no per-run model yet — the deployment default applies")
	assert.Empty(t, svc.receivedMediaID, "TranscriptionService.StartTranscription must not be called")
}

func TestTranscribeEpisode_WithSoloGeneratorRoutesThroughThePipeline(t *testing.T) {
	solo := &fakeSoloGenerator{jobID: "solo-job-2"}
	svc := &mockTranscriptionService{available: false, canResumeEpisode: false}
	h := soloEpisodeHandler(t, solo, svc)

	w, resp := postTranscribe(t, h, "/api/v1/episodes/"+testEpisodeUUID+"/transcribe")

	require.Equal(t, http.StatusAccepted, w.Code, w.Body.String())
	assert.Equal(t, "solo-job-2", resp.Data.(map[string]interface{})["job_id"])
	require.Len(t, solo.started, 1)
	assert.Equal(t, subtitle.MediaRef{ID: testEpisodeUUID, MediaType: models.SubtitleRunMediaEpisode}, solo.started[0])
	assert.Empty(t, svc.receivedMediaID)
}

func TestTranscribeMovie_WithSoloGeneratorStillValidatesTheFile(t *testing.T) {
	solo := &fakeSoloGenerator{jobID: "x"}
	movie := &models.Movie{FilePath: models.NewNullString("/nonexistent/path.mkv")}
	movie.ID = testMovieUUID
	h := NewTranscriptionHandler(&mockTranscriptionMovieGetter{movie: movie}, nil, &mockTranscriptionService{available: true})
	h.SetSoloGenerator(solo)

	w, resp := postTranscribe(t, h, "/api/v1/movies/"+testMovieUUID+"/transcribe")

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, "VALIDATION_REQUIRED_FIELD", resp.Error.Code)
	assert.Empty(t, solo.started, "a missing file is answered before the runner is asked")
}

// ─── AC #3 / #4 / #5: the runner's sentinels keep their wire codes ─────────

func TestTranscribeMovie_SoloGeneratorErrorsMapToTheExistingCodes(t *testing.T) {
	cases := []struct {
		name     string
		err      error
		wantHTTP int
		wantCode string
	}{
		{"in flight elsewhere", services.ErrTranscriptionInProgress, http.StatusConflict, "TRANSCRIPTION_IN_PROGRESS"},
		{"would need ASR and none configured", services.ErrTranscriptionDisabled, http.StatusServiceUnavailable, "TRANSCRIPTION_DISABLED"},
		{"folder not writable (pipeline sentinel)", fmt.Errorf("%w: read-only", subtitle.ErrSubtitleTargetNotWritable), http.StatusConflict, "SUBTITLE_TARGET_NOT_WRITABLE"},
		{"folder not writable (service sentinel)", services.ErrTranscriptionTargetNotWritable, http.StatusConflict, "SUBTITLE_TARGET_NOT_WRITABLE"},
		{"anything else", errors.New("load failed"), http.StatusInternalServerError, "INTERNAL_ERROR"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			solo := &fakeSoloGenerator{startErr: tc.err}
			h := soloMovieHandler(t, solo, &mockTranscriptionService{available: true})

			w, resp := postTranscribe(t, h, "/api/v1/movies/"+testMovieUUID+"/transcribe")

			assert.Equal(t, tc.wantHTTP, w.Code, w.Body.String())
			require.NotNil(t, resp.Error)
			assert.Equal(t, tc.wantCode, resp.Error.Code)
		})
	}
}

func TestTranscribeEpisode_SoloGeneratorInProgressNamesTheEpisode(t *testing.T) {
	solo := &fakeSoloGenerator{startErr: services.ErrTranscriptionInProgress}
	h := soloEpisodeHandler(t, solo, &mockTranscriptionService{available: true})

	w, resp := postTranscribe(t, h, "/api/v1/episodes/"+testEpisodeUUID+"/transcribe")

	assert.Equal(t, http.StatusConflict, w.Code)
	assert.Equal(t, "TRANSCRIPTION_IN_PROGRESS", resp.Error.Code)
	assert.Equal(t, "這一集的字幕生成已在執行中", resp.Error.Message)
}

func TestTranscribeEpisode_SoloGeneratorDisabledIs503WithTheSameWords(t *testing.T) {
	solo := &fakeSoloGenerator{startErr: services.ErrTranscriptionDisabled}
	h := soloEpisodeHandler(t, solo, &mockTranscriptionService{available: true})

	w, resp := postTranscribe(t, h, "/api/v1/episodes/"+testEpisodeUUID+"/transcribe")

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	assert.Equal(t, "TRANSCRIPTION_DISABLED", resp.Error.Code)
	assert.Equal(t, "語音辨識尚未設定", resp.Error.Message)
}

// ─── AC #9: status comes from the runner and carries the job id ─────────────

type soloStatusBody struct {
	Success bool `json:"success"`
	Data    struct {
		InProgress *bool   `json:"in_progress"`
		JobID      *string `json:"job_id"`
	} `json:"data"`
}

func TestTranscriptionStatus_WithSoloGenerator(t *testing.T) {
	cases := []struct {
		name       string
		inProgress bool
		jobID      string
	}{
		{"our job running", true, "solo-job-9"},
		{"batch or pool running it", true, ""},
		{"idle", false, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			solo := &fakeSoloGenerator{inProgress: tc.inProgress, statusJob: tc.jobID}
			svc := &mockTranscriptionService{inProgress: !tc.inProgress} // the legacy table must be ignored
			h := soloEpisodeHandler(t, solo, svc)

			r := setupTranscriptionRouter(h)
			w := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodGet, "/api/v1/episodes/"+testEpisodeUUID+"/transcribe/status", nil)
			r.ServeHTTP(w, req)

			require.Equal(t, http.StatusOK, w.Code)
			var body soloStatusBody
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
			require.NotNil(t, body.Data.InProgress)
			assert.Equal(t, tc.inProgress, *body.Data.InProgress)
			if tc.jobID == "" {
				assert.Nil(t, body.Data.JobID, "no job id → the key is absent, never an empty string")
			} else {
				require.NotNil(t, body.Data.JobID)
				assert.Equal(t, tc.jobID, *body.Data.JobID)
			}
			require.Len(t, solo.statusAsk, 1)
			assert.Equal(t, subtitle.MediaRef{ID: testEpisodeUUID, MediaType: models.SubtitleRunMediaEpisode}, solo.statusAsk[0])
			assert.Empty(t, svc.inProgressAsked, "TranscriptionService.IsInProgress is not consulted in pipeline mode")
		})
	}
}

func TestTranscriptionStatus_WithSoloGeneratorMovieRefIsMovie(t *testing.T) {
	solo := &fakeSoloGenerator{}
	h := soloMovieHandler(t, solo, &mockTranscriptionService{})

	r := setupTranscriptionRouter(h)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/movies/"+testMovieUUID+"/transcribe/status", nil)
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Len(t, solo.statusAsk, 1)
	assert.Equal(t, subtitle.MediaRef{ID: testMovieUUID, MediaType: models.SubtitleRunMediaMovie}, solo.statusAsk[0])
}
