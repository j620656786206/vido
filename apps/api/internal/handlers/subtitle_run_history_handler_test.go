package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/repository"
	"github.com/vido/api/internal/services"
)

// disc-2026-11-spend-history-b-api. The handler runs on the REAL service (a
// fake ledger underneath), so the JSON asserted here is the shape the page
// reads — not a hand-built struct.

type stubRunHistoryLedger struct {
	rows []repository.RecentSubtitleRun
	err  error
}

func (s *stubRunHistoryLedger) FinishedRunsStartedBetween(context.Context, time.Time, time.Time) ([]repository.RecentSubtitleRun, error) {
	return s.rows, s.err
}

var runHistoryNow = time.Date(2026, 11, 3, 12, 0, 0, 0, time.FixedZone("Asia/Taipei", 8*3600))

func runHistoryRow(id, route string, spent *float64, started time.Time) repository.RecentSubtitleRun {
	return repository.RecentSubtitleRun{
		Run: models.SubtitleRun{
			ID: id, MediaID: "m-" + id, MediaType: models.SubtitleRunMediaMovie, Status: models.SubtitleRunCompleted,
			Route: route, ModelID: "claude-sonnet-5", StartedAt: started.UTC(), SpentUSD: spent,
		},
		MediaTitle: "片 " + id,
	}
}

func getRunHistory(t *testing.T, ledger *stubRunHistoryLedger, query string) (*httptest.ResponseRecorder, map[string]any) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	svc := services.NewSubtitleRunHistoryService(ledger, nil)
	svc.SetClock(func() time.Time { return runHistoryNow })
	r := gin.New()
	NewSubtitleRunHistoryHandler(svc, nil).RegisterRoutes(r.Group("/api/v1"))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/subtitles/runs"+query, nil))
	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	return w, body
}

func spentPtr(v float64) *float64 { return &v }

func TestListRuns_DefaultsToThisMonthPageOneOfFifty(t *testing.T) {
	day := runHistoryNow.Add(-24 * time.Hour)
	ledger := &stubRunHistoryLedger{rows: []repository.RecentSubtitleRun{
		runHistoryRow("legacy", "", nil, day),
	}}
	w, body := getRunHistory(t, ledger, "")

	require.Equal(t, http.StatusOK, w.Code)
	data := body["data"].(map[string]any)
	assert.Equal(t, "2026-11", data["month"])
	assert.Equal(t, 1.0, data["page"])
	assert.Equal(t, 50.0, data["page_size"])
	assert.Equal(t, 1.0, data["total"])

	summary := data["summary"].(map[string]any)
	assert.Equal(t, 1.0, summary["unpriced_runs"])
	assert.Contains(t, summary, "comparable_spent_usd")
	assert.Nil(t, summary["comparable_spent_usd"], "null on the wire, never 0")
	assert.Nil(t, summary["comparable_estimate_usd"])

	series := data["series"].([]any)
	require.Len(t, series, 3)
	first := series[0].(map[string]any)
	assert.Equal(t, "2026-11-01", first["date"])
	assert.Equal(t, 0.0, first["spent_usd"])
	byCat := first["by_category"].(map[string]any)
	assert.Len(t, byCat, 4)
	for _, k := range []string{"chinese_track", "translate_english", "asr", "legacy"} {
		assert.Contains(t, byCat, k)
	}

	items := data["items"].([]any)
	require.Len(t, items, 1)
	item := items[0].(map[string]any)
	for _, k := range []string{"completed_at", "route", "source", "spent_usd", "estimate_usd",
		"translation_model", "asr_provider", "asr_model", "batch_id", "triggered_by"} {
		assert.Contains(t, item, k, "unrecorded %s is sent as null, not dropped", k)
		assert.Nil(t, item[k], k)
	}
	assert.Equal(t, "", item["failure_reason"])
	assert.Equal(t, "legacy", item["category"])
	media := item["media"].(map[string]any)
	assert.Equal(t, "movie", media["type"])
	assert.Equal(t, "片 legacy", media["title"])
	assert.Contains(t, media, "season")
	assert.Nil(t, media["season"])
}

func TestListRuns_CategoryFiltersItemsButNotSummaryOrSeries(t *testing.T) {
	day := runHistoryNow.Add(-24 * time.Hour)
	ledger := &stubRunHistoryLedger{rows: []repository.RecentSubtitleRun{
		runHistoryRow("asr", models.SubtitleRunRouteASR, spentPtr(3.10), day),
		runHistoryRow("tr", models.SubtitleRunRouteTranslate, spentPtr(0.81), day.Add(-time.Hour)),
	}}
	_, all := getRunHistory(t, ledger, "?month=2026-11")
	w, filtered := getRunHistory(t, ledger, "?month=2026-11&category=asr&page=1&page_size=10")

	require.Equal(t, http.StatusOK, w.Code)
	allData, data := all["data"].(map[string]any), filtered["data"].(map[string]any)
	assert.Equal(t, allData["summary"], data["summary"])
	assert.Equal(t, allData["series"], data["series"])
	assert.Equal(t, 1.0, data["total"])
	assert.Equal(t, 10.0, data["page_size"])
	items := data["items"].([]any)
	require.Len(t, items, 1)
	assert.Equal(t, "asr", items[0].(map[string]any)["id"])
	assert.Equal(t, 3.91, data["summary"].(map[string]any)["spent_usd"])
}

func TestListRuns_MalformedQueryIs400InvalidFormat(t *testing.T) {
	for _, q := range []string{
		"?month=2026-1", "?month=november", "?month=2026-13",
		"?category=translate",
		"?page=0", "?page=abc", "?page_size=0", "?page_size=201", "?page_size=x",
	} {
		w, body := getRunHistory(t, &stubRunHistoryLedger{}, q)
		assert.Equal(t, http.StatusBadRequest, w.Code, q)
		errBody := body["error"].(map[string]any)
		assert.Equal(t, "VALIDATION_INVALID_FORMAT", errBody["code"], q)
		assert.NotContains(t, errBody["message"], "invalid run history query", "the sentinel's prefix is not shown")
	}
}

func TestListRuns_LedgerFailureIs500(t *testing.T) {
	w, body := getRunHistory(t, &stubRunHistoryLedger{err: errors.New("db locked")}, "")
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Equal(t, false, body["success"])
}
