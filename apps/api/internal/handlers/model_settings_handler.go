package handlers

import (
	"context"

	"github.com/gin-gonic/gin"

	"github.com/vido/api/internal/ai"
)

// ModelCatalogReader is the narrow surface the handler drives (Rule 11).
// *services.ModelCatalogService satisfies it.
type ModelCatalogReader interface {
	Available(ctx context.Context) []ai.ModelInfo
	DefaultModel(ctx context.Context) string
}

// ModelSettingsHandler serves the selectable-model list (story sub-6-8a AC #2).
//
// It is a settings endpoint rather than part of the consent payload because
// the list is a property of the DEPLOYMENT (which keys are saved), not of a
// particular sweep: the settings page and the consent dialog ask the same
// question and must get the same answer.
type ModelSettingsHandler struct {
	catalog ModelCatalogReader
}

// NewModelSettingsHandler creates the handler.
func NewModelSettingsHandler(catalog ModelCatalogReader) *ModelSettingsHandler {
	return &ModelSettingsHandler{catalog: catalog}
}

// RegisterRoutes registers the model settings route (Rule 10 — the group is
// already /api/v1).
//
// ⚠️ Must be registered BEFORE SettingsHandler, whose `/settings/:key` param
// route would otherwise swallow `/settings/models` (the same ordering note the
// log / cache / status / backup / export handlers carry in main.go).
func (h *ModelSettingsHandler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET("/settings/models", h.GetModels)
}

// ModelListResponse is the GET body.
//
// [@contract-v1] — consumed by sub-6-8b (the per-run model picker). `models`
// is ordered for display and may be EMPTY: a deployment with no AI key
// configured can reach this endpoint, and an empty list is the honest answer,
// never an error. `default_model_id` is what the picker should pre-select.
type ModelListResponse struct {
	Models         []ai.ModelInfo `json:"models"`
	DefaultModelID string         `json:"default_model_id"`
}

// GetModels handles GET /api/v1/settings/models.
// @Summary List the translation models this deployment can run
// @Description Returns the priced, currently-reachable translation models — the built-in catalog narrowed to providers whose API key resolves (a Claude-only install sees no Gemini models). Each entry carries per-1M-token input/output prices so a client can quote a run before it starts, and `quality_grade` (with `quality_note`) ONLY for models Vido has blind-scored on real subtitles; an absent grade means "not evaluated", never "equivalent". `default_model_id` is the id a picker should pre-select. An install with no AI key returns 200 with an empty list.
// @Tags settings
// @Produce json
// @Success 200 {object} APIResponse "{models:[{id,provider,display_name,tier,input_per_1m,output_per_1m,is_default,quality_grade?,quality_note?}], default_model_id}"
// @Router /api/v1/settings/models [get]
func (h *ModelSettingsHandler) GetModels(c *gin.Context) {
	ctx := c.Request.Context()
	models := h.catalog.Available(ctx)
	if models == nil {
		// A nil slice marshals to `null`; the client contract says list.
		models = []ai.ModelInfo{}
	}
	SuccessResponse(c, ModelListResponse{
		Models:         models,
		DefaultModelID: h.catalog.DefaultModel(ctx),
	})
}
