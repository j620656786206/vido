package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vido/api/internal/ai"
)

type stubBatchUsage struct {
	snap ai.BudgetSnapshot
	ok   bool
}

func (s stubBatchUsage) ActiveSnapshot() (ai.BudgetSnapshot, bool) { return s.snap, s.ok }

type stubManualUsage struct {
	spent, budget float64
	ok            bool
}

func (s stubManualUsage) ActiveManualUsage() (float64, float64, bool) {
	return s.spent, s.budget, s.ok
}

func getUsage(t *testing.T, h *AIUsageHandler) map[string]any {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h.RegisterRoutes(r.Group("/api/v1"))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/ai/usage", nil))
	require.Equal(t, http.StatusOK, w.Code, "idle is not a 404")
	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, true, body["success"])
	return body["data"].(map[string]any)
}

func TestAIUsage_Idle(t *testing.T) {
	data := getUsage(t, NewAIUsageHandler(stubBatchUsage{}, stubManualUsage{}))
	assert.Contains(t, data, "active_run")
	assert.Nil(t, data["active_run"])
}

func TestAIUsage_BatchWinsOverManual(t *testing.T) {
	data := getUsage(t, NewAIUsageHandler(
		stubBatchUsage{snap: ai.BudgetSnapshot{SpentUSD: 1.25, BudgetUSD: 5}, ok: true},
		stubManualUsage{spent: 0.1, budget: 2, ok: true},
	))
	run := data["active_run"].(map[string]any)
	assert.Equal(t, "batch", run["kind"])
	assert.Equal(t, 1.25, run["spent_usd"])
	assert.Equal(t, 5.0, run["budget_usd"])
	assert.Equal(t, 3.75, run["remaining_usd"])
	assert.NotContains(t, run, "started_at")
}

func TestAIUsage_Manual(t *testing.T) {
	data := getUsage(t, NewAIUsageHandler(stubBatchUsage{}, stubManualUsage{spent: 0.42, budget: 0.4, ok: true}))
	run := data["active_run"].(map[string]any)
	assert.Equal(t, "manual", run["kind"])
	assert.Equal(t, 0.0, run["remaining_usd"], "over-spent floors at 0")
}
