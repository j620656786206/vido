package handlers

// Story dsr-6a AC #2 [@contract-v1] — GET …/transcribe/estimate, the price the
// 管理字幕 dialog shows on its paid buttons.

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/repository"
	"github.com/vido/api/internal/services"
)

type recordingEstimator struct {
	targets []services.TranscriptionEstimateTarget
	result  services.TranscriptionEstimate
}

func (e *recordingEstimator) Estimate(_ context.Context, target services.TranscriptionEstimateTarget) services.TranscriptionEstimate {
	e.targets = append(e.targets, target)
	out := e.result
	out.MediaID = target.MediaID
	out.MediaType = target.MediaType
	return out
}

func sampleEstimate() services.TranscriptionEstimate {
	return services.TranscriptionEstimate{
		Plan:                  services.TranscriptionPlanFull,
		ASRAvailable:          true,
		TranslationConfigured: true,
		ModelID:               "claude-sonnet-5",
		RuntimeMinutes:        48.5,
		RuntimeKnown:          true,
		RuntimeSource:         services.RuntimeSourceFFprobe,
		EstimatedUSD:          0.68,
	}
}

func getEstimate(t *testing.T, h *TranscriptionHandler, path string) (*httptest.ResponseRecorder, APIResponse) {
	t.Helper()
	r := setupTranscriptionRouter(h)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, path, nil)
	r.ServeHTTP(w, req)
	var resp APIResponse
	if w.Code != http.StatusNotFound || w.Body.Len() > 0 {
		_ = json.Unmarshal(w.Body.Bytes(), &resp)
	}
	return w, resp
}

func movieWithFile(t *testing.T) *models.Movie {
	t.Helper()
	movie := &models.Movie{FilePath: models.NewNullString(createTempMediaFile(t))}
	movie.ID = testMovieUUID
	movie.DurationSeconds = models.NewNullInt64(2910)
	movie.Runtime = models.NewNullInt64(50)
	return movie
}

func TestTranscriptionEstimate_MovieReturnsTheContractShape(t *testing.T) {
	movie := movieWithFile(t)
	est := &recordingEstimator{result: sampleEstimate()}
	h := NewTranscriptionHandler(&mockTranscriptionMovieGetter{movie: movie}, nil, &mockTranscriptionService{available: true})
	h.SetEstimator(est)

	w, resp := getEstimate(t, h, "/api/v1/movies/"+testMovieUUID+"/transcribe/estimate")

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.True(t, resp.Success)
	data, ok := resp.Data.(map[string]any)
	require.True(t, ok)

	keys := make([]string, 0, len(data))
	for k := range data {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	assert.Equal(t, []string{
		"asr_available", "estimated_usd", "media_id", "media_type", "model_id", "plan",
		"runtime_known", "runtime_minutes", "runtime_source", "self_hosted_asr", "translation_configured",
	}, keys, "the wire keys are a stamped contract — spelling and case matter")

	usd, isNumber := data["estimated_usd"].(float64)
	require.True(t, isNumber, "estimated_usd must be a JSON number, not a decimal string")
	assert.Equal(t, 0.68, usd)
	assert.Equal(t, "movie", data["media_type"])
	assert.Equal(t, testMovieUUID, data["media_id"])

	require.Len(t, est.targets, 1)
	assert.Equal(t, services.TranscriptionEstimateTarget{
		MediaID:         testMovieUUID,
		MediaType:       models.SubtitleRunMediaMovie,
		FilePath:        movie.FilePath.String,
		DurationSeconds: movie.DurationSeconds,
		Runtime:         movie.Runtime,
	}, est.targets[0])
}

func TestTranscriptionEstimate_PricesEvenWhenSpeechRecognitionIsUnavailable(t *testing.T) {
	est := &recordingEstimator{result: sampleEstimate()}
	h := NewTranscriptionHandler(&mockTranscriptionMovieGetter{movie: movieWithFile(t)}, nil, &mockTranscriptionService{available: false})
	h.SetEstimator(est)

	w, _ := getEstimate(t, h, "/api/v1/movies/"+testMovieUUID+"/transcribe/estimate")
	assert.Equal(t, http.StatusOK, w.Code, "the trigger's 503 gate must not guard the price — the dialog disables the button itself")
}

func TestTranscriptionEstimate_MovieErrors(t *testing.T) {
	t.Run("unknown movie → 404", func(t *testing.T) {
		h := NewTranscriptionHandler(&mockTranscriptionMovieGetter{err: errors.New("sql: no rows")}, nil, &mockTranscriptionService{})
		h.SetEstimator(&recordingEstimator{})
		w, resp := getEstimate(t, h, "/api/v1/movies/"+testMovieUUID+"/transcribe/estimate")
		assert.Equal(t, http.StatusNotFound, w.Code)
		require.NotNil(t, resp.Error)
		assert.Equal(t, "DB_NOT_FOUND", resp.Error.Code)
	})

	t.Run("no file path → 400", func(t *testing.T) {
		movie := &models.Movie{}
		movie.ID = testMovieUUID
		est := &recordingEstimator{}
		h := NewTranscriptionHandler(&mockTranscriptionMovieGetter{movie: movie}, nil, &mockTranscriptionService{})
		h.SetEstimator(est)
		w, resp := getEstimate(t, h, "/api/v1/movies/"+testMovieUUID+"/transcribe/estimate")
		assert.Equal(t, http.StatusBadRequest, w.Code)
		require.NotNil(t, resp.Error)
		assert.Equal(t, "VALIDATION_REQUIRED_FIELD", resp.Error.Code)
		assert.Empty(t, est.targets)
	})

	t.Run("file gone from disk → 400", func(t *testing.T) {
		movie := &models.Movie{FilePath: models.NewNullString("/nonexistent/dsr-6a/movie.mkv")}
		movie.ID = testMovieUUID
		h := NewTranscriptionHandler(&mockTranscriptionMovieGetter{movie: movie}, nil, &mockTranscriptionService{})
		h.SetEstimator(&recordingEstimator{})
		w, resp := getEstimate(t, h, "/api/v1/movies/"+testMovieUUID+"/transcribe/estimate")
		assert.Equal(t, http.StatusBadRequest, w.Code)
		require.NotNil(t, resp.Error)
		assert.Equal(t, "VALIDATION_REQUIRED_FIELD", resp.Error.Code)
	})
}

func TestTranscriptionEstimate_EpisodeReturnsTheEpisodeTarget(t *testing.T) {
	ep := episodeWithFile(t)
	ep.DurationSeconds = models.NewNullInt64(2700)
	est := &recordingEstimator{result: sampleEstimate()}
	h := newEpisodeHandler(ep, nil, &mockTranscriptionService{available: true})
	h.SetEstimator(est)

	w, resp := getEstimate(t, h, "/api/v1/episodes/"+testEpisodeUUID+"/transcribe/estimate")

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	data := resp.Data.(map[string]any)
	assert.Equal(t, "episode", data["media_type"])
	require.Len(t, est.targets, 1)
	assert.Equal(t, models.SubtitleRunMediaEpisode, est.targets[0].MediaType)
	assert.Equal(t, ep.FilePath.String, est.targets[0].FilePath)
	assert.Equal(t, ep.DurationSeconds, est.targets[0].DurationSeconds)
}

func TestTranscriptionEstimate_EpisodeErrors(t *testing.T) {
	t.Run("not found → 404", func(t *testing.T) {
		h := newEpisodeHandler(nil, repository.ErrEpisodeNotFound, &mockTranscriptionService{})
		h.SetEstimator(&recordingEstimator{})
		w, _ := getEstimate(t, h, "/api/v1/episodes/"+testEpisodeUUID+"/transcribe/estimate")
		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("lookup failure → 500, not a misleading 404", func(t *testing.T) {
		h := newEpisodeHandler(nil, errors.New("database is locked"), &mockTranscriptionService{})
		h.SetEstimator(&recordingEstimator{})
		w, resp := getEstimate(t, h, "/api/v1/episodes/"+testEpisodeUUID+"/transcribe/estimate")
		assert.Equal(t, http.StatusInternalServerError, w.Code)
		require.NotNil(t, resp.Error)
		assert.Equal(t, "INTERNAL_ERROR", resp.Error.Code)
	})

	t.Run("no file path → 400", func(t *testing.T) {
		h := newEpisodeHandler(&models.Episode{ID: testEpisodeUUID}, nil, &mockTranscriptionService{})
		h.SetEstimator(&recordingEstimator{})
		w, resp := getEstimate(t, h, "/api/v1/episodes/"+testEpisodeUUID+"/transcribe/estimate")
		assert.Equal(t, http.StatusBadRequest, w.Code)
		require.NotNil(t, resp.Error)
		assert.Equal(t, "VALIDATION_REQUIRED_FIELD", resp.Error.Code)
	})
}

func TestTranscriptionEstimate_RoutesAreMountedOnlyWhenWired(t *testing.T) {
	t.Run("no estimator → both GETs 404", func(t *testing.T) {
		h := newEpisodeHandler(episodeWithFile(t), nil, &mockTranscriptionService{available: true})
		for _, path := range []string{
			"/api/v1/movies/" + testMovieUUID + "/transcribe/estimate",
			"/api/v1/episodes/" + testEpisodeUUID + "/transcribe/estimate",
		} {
			w, _ := getEstimate(t, h, path)
			assert.Equal(t, http.StatusNotFound, w.Code, path)
		}
	})

	t.Run("no episode getter → only the movie GET exists", func(t *testing.T) {
		h := NewTranscriptionHandler(&mockTranscriptionMovieGetter{movie: movieWithFile(t)}, nil, &mockTranscriptionService{})
		h.SetEstimator(&recordingEstimator{result: sampleEstimate()})

		movie, _ := getEstimate(t, h, "/api/v1/movies/"+testMovieUUID+"/transcribe/estimate")
		assert.Equal(t, http.StatusOK, movie.Code)
		episode, _ := getEstimate(t, h, "/api/v1/episodes/"+testEpisodeUUID+"/transcribe/estimate")
		assert.Equal(t, http.StatusNotFound, episode.Code)
	})
}
