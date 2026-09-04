package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vido/api/internal/ai"
)

// ─── sub-6-8a AC #2: GET /settings/models ──────────────────────────────────

type stubModelCatalog struct {
	models       []ai.ModelInfo
	defaultModel string
}

func (s *stubModelCatalog) Available(context.Context) []ai.ModelInfo { return s.models }
func (s *stubModelCatalog) DefaultModel(context.Context) string      { return s.defaultModel }

func getModels(t *testing.T, catalog ModelCatalogReader) (*httptest.ResponseRecorder, map[string]any) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	NewModelSettingsHandler(catalog).RegisterRoutes(r.Group("/api/v1"))

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/settings/models", nil))

	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	return w, body
}

func TestGetModels_ServesTheCatalogWithPricesAndGrades(t *testing.T) {
	w, body := getModels(t, &stubModelCatalog{
		models: []ai.ModelInfo{
			{ID: "claude-haiku-4-5", Provider: "claude", DisplayName: "Claude Haiku 4.5", Tier: "fast",
				InputPer1M: 1, OutputPer1M: 5, QualityGrade: "B", QualityNote: "Vido 實測"},
			{ID: "claude-sonnet-5", Provider: "claude", DisplayName: "Claude Sonnet 5", Tier: "balanced",
				InputPer1M: 3, OutputPer1M: 15, IsDefault: true, QualityGrade: "A", QualityNote: "Vido 實測"},
		},
		defaultModel: "claude-sonnet-5",
	})

	require.Equal(t, http.StatusOK, w.Code)
	assert.True(t, body["success"].(bool))
	data := body["data"].(map[string]any)
	assert.Equal(t, "claude-sonnet-5", data["default_model_id"])

	models := data["models"].([]any)
	require.Len(t, models, 2)
	first := models[0].(map[string]any)
	assert.Equal(t, "claude-haiku-4-5", first["id"])
	assert.Equal(t, "fast", first["tier"])
	assert.EqualValues(t, 5, first["output_per_1m"], "a client cannot quote a run without the price")
	assert.Equal(t, "B", first["quality_grade"])

	second := models[1].(map[string]any)
	assert.Equal(t, true, second["is_default"])
}

func TestGetModels_UngradedModelOmitsTheGradeRatherThanFakingOne(t *testing.T) {
	_, body := getModels(t, &stubModelCatalog{
		models:       []ai.ModelInfo{{ID: "claude-opus-4-8", Provider: "claude", DisplayName: "Claude Opus 4.8", Tier: "max", InputPer1M: 5, OutputPer1M: 25}},
		defaultModel: "claude-opus-4-8",
	})

	model := body["data"].(map[string]any)["models"].([]any)[0].(map[string]any)
	_, hasGrade := model["quality_grade"]
	assert.False(t, hasGrade, "an unevaluated model must be silent, not imply parity with a measured one")
	_, hasNote := model["quality_note"]
	assert.False(t, hasNote)
}

func TestGetModels_KeylessInstallGetsAnEmptyListNotAnError(t *testing.T) {
	w, body := getModels(t, &stubModelCatalog{})

	require.Equal(t, http.StatusOK, w.Code, "an unconfigured install must still be able to render the settings page")
	data := body["data"].(map[string]any)
	assert.Equal(t, []any{}, data["models"], "an empty LIST, never null — the client contract says array")
	assert.Equal(t, "", data["default_model_id"])
}
