package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vido/api/internal/subtitle/miner"
)

type stubMiner struct {
	mu      sync.Mutex
	running bool
	single  []string
	sweeps  int
	err     error
}

func (s *stubMiner) MineSeries(_ context.Context, id string) (miner.MineResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.single = append(s.single, id)
	if s.err != nil {
		return miner.MineResult{SeriesID: id, Error: s.err.Error()}, s.err
	}
	return miner.MineResult{SeriesID: id, Title: "Show", TermsFound: 3, TermsInserted: 2}, nil
}
func (s *stubMiner) StartPartial(context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.running {
		return miner.ErrMinerBusy
	}
	s.sweeps++
	return nil
}
func (s *stubMiner) Status() miner.MineStatus {
	s.mu.Lock()
	defer s.mu.Unlock()
	return miner.MineStatus{Running: s.running, Results: []miner.MineResult{}}
}

func mineRouter(m OfficialSubtitleMiner) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	NewGlossaryMineHandler(m).RegisterRoutes(r.Group("/api/v1"))
	return r
}

func TestGlossaryMine_SingleShowIsSynchronous(t *testing.T) {
	m := &stubMiner{}
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/subtitles/glossary/mine", strings.NewReader(`{"series_id":"s1"}`))
	req.Header.Set("Content-Type", "application/json")
	mineRouter(m).ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	data := body["data"].(map[string]any)
	assert.Equal(t, "s1", data["series_id"])
	assert.Equal(t, 2.0, data["terms_inserted"])
	assert.Equal(t, []string{"s1"}, m.single)

	// Query form works too.
	w = httptest.NewRecorder()
	mineRouter(m).ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/v1/subtitles/glossary/mine?series_id=s2", nil))
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, []string{"s1", "s2"}, m.single)
}

func TestGlossaryMine_SweepIsAcceptedAndRefusedWhileRunning(t *testing.T) {
	m := &stubMiner{}
	w := httptest.NewRecorder()
	mineRouter(m).ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/v1/subtitles/glossary/mine", nil))
	require.Equal(t, http.StatusAccepted, w.Code)
	assert.Equal(t, 1, m.sweeps, "claimed before the 202 goes out")

	m.running = true
	w = httptest.NewRecorder()
	mineRouter(m).ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/v1/subtitles/glossary/mine", nil))
	assert.Equal(t, http.StatusConflict, w.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "GLOSSARY_MINE_RUNNING", body["error"].(map[string]any)["code"])

	w = httptest.NewRecorder()
	mineRouter(m).ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/subtitles/glossary/mine", nil))
	require.Equal(t, http.StatusOK, w.Code)
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, true, body["data"].(map[string]any)["running"])
}

func TestGlossaryMine_ErrorsMap(t *testing.T) {
	busy := &stubMiner{err: miner.ErrMinerBusy}
	w := httptest.NewRecorder()
	mineRouter(busy).ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/v1/subtitles/glossary/mine?series_id=s1", nil))
	assert.Equal(t, http.StatusConflict, w.Code)

	failed := &stubMiner{err: errors.New("series s1: not found")}
	w = httptest.NewRecorder()
	mineRouter(failed).ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/v1/subtitles/glossary/mine?series_id=s1", nil))
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "GLOSSARY_MINE_FAILED", body["error"].(map[string]any)["code"])

	w = httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/subtitles/glossary/mine", strings.NewReader(`{bad json`))
	req.Header.Set("Content-Type", "application/json")
	mineRouter(&stubMiner{}).ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}
