package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vido/api/internal/services"
)

type fakeBurnedInChecker struct {
	res       services.BurnedInCheckResult
	err       error
	calls     int
	gotSeries string
	gotSeason int
}

func (f *fakeBurnedInChecker) CheckSeason(_ context.Context, seriesID string, season int) (services.BurnedInCheckResult, error) {
	f.calls++
	f.gotSeries, f.gotSeason = seriesID, season
	return f.res, f.err
}

func postBurnedInCheck(t *testing.T, checker *fakeBurnedInChecker, body string) (*httptest.ResponseRecorder, map[string]any) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	NewBurnedInSubtitleHandler(checker, nil).RegisterRoutes(r.Group("/api/v1"))
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/subtitles/burned-in/check", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	var out map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &out))
	return w, out
}

func TestBurnedInCheck_Success(t *testing.T) {
	result := "zh-Hans"
	checker := &fakeBurnedInChecker{res: services.BurnedInCheckResult{
		Result: &result, EpisodesUpdated: 12, CostUSD: 0.0046, CheckedEpisodeIDs: []string{"e1", "e6"},
	}}
	w, body := postBurnedInCheck(t, checker, `{"series_id": "s1", "season": 2}`)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "s1", checker.gotSeries)
	assert.Equal(t, 2, checker.gotSeason)
	data := body["data"].(map[string]any)
	assert.Equal(t, "zh-Hans", data["result"])
	assert.EqualValues(t, 12, data["episodes_updated"])
	assert.InDelta(t, 0.0046, data["cost_usd"], 1e-12)
	assert.Equal(t, []any{"e1", "e6"}, data["checked_episode_ids"])
	assert.NotContains(t, data, "reason")
}

func TestBurnedInCheck_UndecidedIsNullWithReason(t *testing.T) {
	checker := &fakeBurnedInChecker{res: services.BurnedInCheckResult{Reason: services.BurnedInReasonFramesUnreadable, CheckedEpisodeIDs: []string{"e1"}}}
	w, body := postBurnedInCheck(t, checker, `{"series_id": "s1", "season": 0}`)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, 0, checker.gotSeason, "season 0 (specials) is a real season")
	data := body["data"].(map[string]any)
	assert.Contains(t, data, "result")
	assert.Nil(t, data["result"], "undecided is null on the wire, never \"none\"")
	assert.Equal(t, "frames_unavailable", data["reason"])
	assert.EqualValues(t, 0, data["episodes_updated"])
}

func TestBurnedInCheck_NoClaudeKeyIs400(t *testing.T) {
	checker := &fakeBurnedInChecker{err: services.ErrBurnedInNotConfigured}
	w, body := postBurnedInCheck(t, checker, `{"series_id": "s1", "season": 1}`)

	require.Equal(t, http.StatusBadRequest, w.Code)
	errBody := body["error"].(map[string]any)
	assert.Equal(t, "AI_NOT_CONFIGURED", errBody["code"])
	assert.Contains(t, errBody["message"], "Claude 金鑰")
}

func TestBurnedInCheck_BadParamsAre400(t *testing.T) {
	for name, body := range map[string]string{
		"not JSON":         `series_id=s1`,
		"missing season":   `{"series_id": "s1"}`,
		"negative season":  `{"series_id": "s1", "season": -1}`,
		"missing series":   `{"season": 1}`,
		"blank series":     `{"series_id": "  ", "season": 1}`,
		"season as string": `{"series_id": "s1", "season": "1"}`,
	} {
		t.Run(name, func(t *testing.T) {
			checker := &fakeBurnedInChecker{}
			w, out := postBurnedInCheck(t, checker, body)
			require.Equal(t, http.StatusBadRequest, w.Code)
			assert.Equal(t, "VALIDATION_ERROR", out["error"].(map[string]any)["code"])
			assert.Zero(t, checker.calls, "nothing is checked on a bad request")
		})
	}
}

func TestBurnedInCheck_FailuresMapToStatus(t *testing.T) {
	w, body := postBurnedInCheck(t, &fakeBurnedInChecker{err: errors.Join(services.ErrBurnedInModelCall, errors.New("529"))}, `{"series_id": "s1", "season": 1}`)
	require.Equal(t, http.StatusBadGateway, w.Code)
	assert.Equal(t, "AI_PROVIDER_ERROR", body["error"].(map[string]any)["code"])

	w, _ = postBurnedInCheck(t, &fakeBurnedInChecker{err: errors.New("database is locked")}, `{"series_id": "s1", "season": 1}`)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
