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

	"github.com/vido/api/internal/services"
)

type fakeSpendReader struct {
	summary *services.SubtitleSpendSummary
	err     error
	gotID   string
}

func (f *fakeSpendReader) MonthSummary(_ context.Context, batchID string) (*services.SubtitleSpendSummary, error) {
	f.gotID = batchID
	return f.summary, f.err
}

func getSpend(t *testing.T, reader *fakeSpendReader, query string) (*httptest.ResponseRecorder, map[string]any) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	NewSubtitleSpendHandler(reader, nil).RegisterRoutes(r.Group("/api/v1"))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/subtitles/spend"+query, nil))
	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	return w, body
}

func TestGetSpend_ServesTheSummaryWithTheHonestNullsIntact(t *testing.T) {
	reader := &fakeSpendReader{summary: &services.SubtitleSpendSummary{
		Period: "month", TranslatedRuns: 3, TranslatedUSD: 0.75, ASRRuns: 1, ASRUSD: 1.2,
		SkippedDeliverCount: 4, SkippedSavedUSDEstimate: 2.1, SkippedSavedRuntimeAssumed: true,
		ByModel: []services.SpendByModel{{ModelID: "claude-sonnet-5", Runs: 4, USD: 1.95}},
	}}
	w, body := getSpend(t, reader, "?period=month&batch_id=b1")

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "b1", reader.gotID)
	data := body["data"].(map[string]any)
	assert.Equal(t, "month", data["period"])
	assert.Equal(t, 0.75, data["translated_usd"])
	assert.Equal(t, true, data["skipped_saved_runtime_assumed"])
	assert.Contains(t, data, "cache_saved_usd_estimate")
	assert.Nil(t, data["cache_saved_usd_estimate"], "unmeasured cache saving is null on the wire, never 0")
	assert.NotContains(t, data, "by_batch", "omitted when the service attached none")
	byModel := data["by_model"].([]any)
	require.Len(t, byModel, 1)
}

func TestGetSpend_RejectsAnyPeriodButMonth(t *testing.T) {
	for _, q := range []string{"", "?period=week", "?period=Month"} {
		w, body := getSpend(t, &fakeSpendReader{}, q)
		assert.Equal(t, http.StatusBadRequest, w.Code, q)
		errBody := body["error"].(map[string]any)
		assert.Equal(t, "VALIDATION_INVALID_FORMAT", errBody["code"], q)
	}
}

func TestGetSpend_LedgerFailureIs500NotAnEmptySummary(t *testing.T) {
	w, body := getSpend(t, &fakeSpendReader{err: errors.New("db locked")}, "?period=month")
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Equal(t, false, body["success"])
}
