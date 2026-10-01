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

	"github.com/vido/api/internal/ai"
	"github.com/vido/api/internal/preview"
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

// ─── sub-7-8c: local preview ────────────────────────────────────────────────

type stubPreviewer struct {
	grades   map[string]preview.LocalGrade
	estimate map[string]float64
	err      error
	calls    []string
}

func (s *stubPreviewer) Preview(_ context.Context, model string) (preview.LocalGrade, error) {
	s.calls = append(s.calls, model)
	if s.err != nil {
		return preview.LocalGrade{}, s.err
	}
	return preview.LocalGrade{ModelID: model, Cues: 20, ZeroRate: 0.05, NaturalRate: 0.7, CostUSD: 0.0123, JudgeModel: "claude-sonnet-5"}, nil
}
func (s *stubPreviewer) LocalGrades(context.Context) map[string]preview.LocalGrade { return s.grades }
func (s *stubPreviewer) EstimateUSD(model string) float64                          { return s.estimate[model] }

func previewRouter(catalog ModelCatalogReader, p ModelPreviewer) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := NewModelSettingsHandler(catalog)
	if p != nil {
		h.WithPreview(p)
	}
	h.RegisterRoutes(r.Group("/api/v1"))
	return r
}

func TestGetModels_CarriesLocalGradeAndEstimateWhenPreviewIsOn(t *testing.T) {
	catalog := &stubModelCatalog{models: []ai.ModelInfo{
		{ID: "claude-opus-4-8", Provider: "claude", DisplayName: "Opus", Tier: "max", InputPer1M: 5, OutputPer1M: 25},
		{ID: "claude-sonnet-5", Provider: "claude", DisplayName: "Sonnet", Tier: "balanced", InputPer1M: 3, OutputPer1M: 15, IsDefault: true, QualityGrade: "A", QualityNote: "n"},
	}, defaultModel: "claude-sonnet-5"}
	p := &stubPreviewer{
		grades:   map[string]preview.LocalGrade{"claude-opus-4-8": {ModelID: "claude-opus-4-8", Cues: 20, ZeroRate: 0.1, NaturalRate: 0.6, CostUSD: 0.05}},
		estimate: map[string]float64{"claude-opus-4-8": 0.06, "claude-sonnet-5": 0.02},
	}
	r := previewRouter(catalog, p)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/settings/models", nil))
	require.Equal(t, http.StatusOK, w.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	models := body["data"].(map[string]any)["models"].([]any)
	opus := models[0].(map[string]any)
	sonnet := models[1].(map[string]any)
	// sub-6-8a keys untouched…
	assert.Equal(t, "A", sonnet["quality_grade"])
	assert.Equal(t, 15.0, sonnet["output_per_1m"])
	// …plus the additive ones.
	assert.Equal(t, 0.06, opus["preview_estimate_usd"])
	assert.Equal(t, 0.1, opus["local_grade"].(map[string]any)["zero_rate"])
	_, hasLocal := sonnet["local_grade"]
	assert.False(t, hasLocal, "no preview on this box → key absent, not null")
	assert.Equal(t, 0.02, sonnet["preview_estimate_usd"])

	// Preview off: neither key appears and POST is a 404.
	r = previewRouter(catalog, nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/settings/models", nil))
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	first := body["data"].(map[string]any)["models"].([]any)[0].(map[string]any)
	_, hasEst := first["preview_estimate_usd"]
	assert.False(t, hasEst)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/v1/settings/models/claude-opus-4-8/preview", nil))
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestPreviewModel_StatusMapping(t *testing.T) {
	catalog := &stubModelCatalog{models: []ai.ModelInfo{{ID: "claude-opus-4-8"}}}
	post := func(p *stubPreviewer) (*httptest.ResponseRecorder, map[string]any) {
		w := httptest.NewRecorder()
		previewRouter(catalog, p).ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/v1/settings/models/claude-opus-4-8/preview", nil))
		var body map[string]any
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
		return w, body
	}
	ok := &stubPreviewer{}
	w, body := post(ok)
	require.Equal(t, http.StatusOK, w.Code)
	data := body["data"].(map[string]any)
	assert.Equal(t, "claude-opus-4-8", data["model_id"])
	assert.Equal(t, 0.0123, data["cost_usd"])
	assert.Equal(t, []string{"claude-opus-4-8"}, ok.calls)

	cases := []struct {
		err    error
		status int
		code   string
	}{
		{preview.ErrPreviewUnsupportedModel, http.StatusBadRequest, "VALIDATION_INVALID_FORMAT"},
		{preview.ErrPreviewTooSoon, http.StatusTooManyRequests, "AI_PREVIEW_TOO_SOON"},
		{ai.ErrAINotConfigured, http.StatusConflict, "AI_NOT_CONFIGURED"},
		{ai.ErrAIUnauthorized, http.StatusConflict, "AI_UNAUTHORIZED"},
		{errors.New("boom"), http.StatusInternalServerError, ""},
	}
	for _, tc := range cases {
		w, body := post(&stubPreviewer{err: tc.err})
		assert.Equal(t, tc.status, w.Code, tc.err.Error())
		if tc.code != "" {
			assert.Equal(t, tc.code, body["error"].(map[string]any)["code"], tc.err.Error())
		}
	}
}
