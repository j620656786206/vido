package handlers

// disc-2026-10-reattach-progress-starts-at-extracting — the status route also
// says where a running generation is, so a dialog attaching mid-run starts its
// stepper there instead of at 抽取.

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vido/api/internal/services"
	"github.com/vido/api/internal/sse"
	"github.com/vido/api/internal/subtitle"
)

type phaseStatusBody struct {
	Data struct {
		InProgress *bool    `json:"in_progress"`
		Phase      *string  `json:"phase"`
		Message    *string  `json:"message"`
		Percentage *float64 `json:"percentage"`
	} `json:"data"`
}

func getPhaseStatus(t *testing.T, h *TranscriptionHandler, path string) phaseStatusBody {
	t.Helper()
	r := setupTranscriptionRouter(h)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, path, nil)
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	var body phaseStatusBody
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.NotNil(t, body.Data.InProgress)
	return body
}

// trackerAt is a real tracker that has seen one progress event for id.
func trackerAt(id string, ev sse.EventType, extra map[string]interface{}) *subtitle.PhaseTracker {
	tr := subtitle.NewPhaseTracker()
	data := map[string]interface{}{"media_id": id, "message": "轉錄中…"}
	for k, v := range extra {
		data[k] = v
	}
	tr.Observe(sse.Event{Type: ev, Data: data})
	return tr
}

func TestTranscriptionStatus_PhaseWhileRunning(t *testing.T) {
	t.Run("pipeline mode", func(t *testing.T) {
		h := soloEpisodeHandler(t, &fakeSoloGenerator{inProgress: true, statusJob: "solo-1"}, &mockTranscriptionService{})
		h.SetPhaseReader(trackerAt(testEpisodeUUID, services.EventTranscriptionProgress, nil))

		body := getPhaseStatus(t, h, "/api/v1/episodes/"+testEpisodeUUID+"/transcribe/status")

		require.NotNil(t, body.Data.Phase)
		assert.Equal(t, "transcribing", *body.Data.Phase)
		require.NotNil(t, body.Data.Message)
		assert.Equal(t, "轉錄中…", *body.Data.Message)
		assert.Nil(t, body.Data.Percentage, "percentage only while translating")
	})

	t.Run("legacy mode, translating with percentage", func(t *testing.T) {
		h := NewTranscriptionHandler(&mockTranscriptionMovieGetter{}, nil, &mockTranscriptionService{inProgress: true})
		h.SetPhaseReader(trackerAt(testMovieUUID, services.EventTranscriptionTranslating, map[string]interface{}{"percentage": 37.5}))

		body := getPhaseStatus(t, h, "/api/v1/movies/"+testMovieUUID+"/transcribe/status")

		require.NotNil(t, body.Data.Phase)
		assert.Equal(t, "translating", *body.Data.Phase)
		require.NotNil(t, body.Data.Percentage)
		assert.Equal(t, 37.5, *body.Data.Percentage)
	})
}

// A stale record must never describe a run that is not going: the phase is
// only read when the single-flight table says in_progress.
func TestTranscriptionStatus_NoPhaseWhenIdle(t *testing.T) {
	h := soloEpisodeHandler(t, &fakeSoloGenerator{inProgress: false}, &mockTranscriptionService{})
	h.SetPhaseReader(trackerAt(testEpisodeUUID, services.EventTranscriptionProgress, nil))

	body := getPhaseStatus(t, h, "/api/v1/episodes/"+testEpisodeUUID+"/transcribe/status")

	assert.False(t, *body.Data.InProgress)
	assert.Nil(t, body.Data.Phase)
	assert.Nil(t, body.Data.Message)
}

// Running, but no progress reported yet (or no reader wired): the keys are
// absent and the dialog starts at extracting, as before.
func TestTranscriptionStatus_NoPhaseWithoutARecord(t *testing.T) {
	for name, reader := range map[string]PhaseReader{
		"no record": subtitle.NewPhaseTracker(),
		"no reader": nil,
	} {
		t.Run(name, func(t *testing.T) {
			h := soloEpisodeHandler(t, &fakeSoloGenerator{inProgress: true}, &mockTranscriptionService{})
			if reader != nil {
				h.SetPhaseReader(reader)
			}

			body := getPhaseStatus(t, h, "/api/v1/episodes/"+testEpisodeUUID+"/transcribe/status")

			assert.True(t, *body.Data.InProgress)
			assert.Nil(t, body.Data.Phase)
		})
	}
}
