package handlers

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/vido/api/internal/ai"
	"github.com/vido/api/internal/preview"
)

// ModelCatalogReader is the narrow surface the handler drives (Rule 11).
// *services.ModelCatalogService satisfies it.
type ModelCatalogReader interface {
	Available(ctx context.Context) []ai.ModelInfo
	DefaultModel(ctx context.Context) string
}

// ModelPreviewer is the narrow surface of *preview.Service the handler drives
// (sub-7-8c). nil = previews disabled: the list is served without
// `local_grade` / `preview_estimate_usd` and POST …/preview answers 404.
type ModelPreviewer interface {
	Preview(ctx context.Context, model string) (preview.LocalGrade, error)
	LocalGrades(ctx context.Context) map[string]preview.LocalGrade
	EstimateUSD(model string) float64
}

// ModelSettingsHandler serves the selectable-model list (story sub-6-8a AC #2).
//
// It is a settings endpoint rather than part of the consent payload because
// the list is a property of the DEPLOYMENT (which keys are saved), not of a
// particular sweep: the settings page and the consent dialog ask the same
// question and must get the same answer.
type ModelSettingsHandler struct {
	catalog ModelCatalogReader
	preview ModelPreviewer
}

// NewModelSettingsHandler creates the handler.
func NewModelSettingsHandler(catalog ModelCatalogReader) *ModelSettingsHandler {
	return &ModelSettingsHandler{catalog: catalog}
}

// WithPreview enables 「試跑 20 句」 (sub-7-8c).
func (h *ModelSettingsHandler) WithPreview(p ModelPreviewer) *ModelSettingsHandler {
	h.preview = p
	return h
}

// RegisterRoutes registers the model settings route (Rule 10 — the group is
// already /api/v1).
//
// ⚠️ Must be registered BEFORE SettingsHandler, whose `/settings/:key` param
// route would otherwise swallow `/settings/models` (the same ordering note the
// log / cache / status / backup / export handlers carry in main.go).
func (h *ModelSettingsHandler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET("/settings/models", h.GetModels)
	rg.POST("/settings/models/:id/preview", h.PreviewModel)
}

// ModelEntry is one row of GET /settings/models: the sub-6-8a ModelInfo plus
// the sub-7-8c additions. Embedding keeps the wire shape of ModelInfo intact
// (0 bump); the two new keys are additive and absent when previews are off.
//
// [@contract-v1] (sub-7-8c additive on sub-6-8a [@contract-v1]).
type ModelEntry struct {
	ai.ModelInfo
	// LocalGrade is THIS box's own preview result, if any. It is shown as
	// 「你的實測」 and never replaces QualityGrade.
	LocalGrade *preview.LocalGrade `json:"local_grade,omitempty"`
	// PreviewEstimateUSD prices the 「試跑 20 句」 button for this model.
	PreviewEstimateUSD float64 `json:"preview_estimate_usd,omitempty"`
}

// ModelListResponse is the GET body.
//
// [@contract-v1] — consumed by sub-6-8b (the per-run model picker). `models`
// is ordered for display and may be EMPTY: a deployment with no AI key
// configured can reach this endpoint, and an empty list is the honest answer,
// never an error. `default_model_id` is what the picker should pre-select.
type ModelListResponse struct {
	Models         []ModelEntry `json:"models"`
	DefaultModelID string       `json:"default_model_id"`
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
	available := h.catalog.Available(ctx)
	// A nil slice marshals to `null`; the client contract says list.
	entries := make([]ModelEntry, 0, len(available))
	var local map[string]preview.LocalGrade
	if h.preview != nil {
		local = h.preview.LocalGrades(ctx)
	}
	for _, m := range available {
		e := ModelEntry{ModelInfo: m}
		if h.preview != nil {
			if g, ok := local[m.ID]; ok {
				g := g
				e.LocalGrade = &g
			}
			e.PreviewEstimateUSD = h.preview.EstimateUSD(m.ID)
		}
		entries = append(entries, e)
	}
	SuccessResponse(c, ModelListResponse{
		Models:         entries,
		DefaultModelID: h.catalog.DefaultModel(ctx),
	})
}

// PreviewModel handles POST /api/v1/settings/models/:id/preview (sub-7-8c AC #1).
//
// @Summary Try an ungraded model on 20 golden cues with this box's own key
// @Description Translates the first 20 cues of Vido's golden sample with the model, scores them the way cmd/grade does, stores the result on THIS box only (`local_grade`), and returns it. Costs real money (the box's key; ceiling ≈ estimate ×3, at most $0.20). Not a media run: it does not appear in subtitle_runs or the monthly spend summary. 400 for a model this deployment cannot run, 409 AI_NOT_CONFIGURED without a key, 429 AI_PREVIEW_TOO_SOON within 60s of the previous preview of the same model.
// @Tags settings
// @Produce json
// @Param id path string true "model id from GET /settings/models"
// @Success 200 {object} APIResponse "preview.LocalGrade [@contract-v1]"
// @Router /settings/models/{id}/preview [post]
func (h *ModelSettingsHandler) PreviewModel(c *gin.Context) {
	if h.preview == nil {
		NotFoundError(c, "model preview")
		return
	}
	grade, err := h.preview.Preview(c.Request.Context(), c.Param("id"))
	switch {
	case err == nil:
		SuccessResponse(c, grade)
	case errors.Is(err, preview.ErrPreviewUnsupportedModel):
		BadRequestError(c, "VALIDATION_INVALID_FORMAT",
			"不支援的模型：請從 GET /api/v1/settings/models 提供的清單中選擇")
	case errors.Is(err, preview.ErrPreviewTooSoon):
		ErrorResponse(c, http.StatusTooManyRequests, "AI_PREVIEW_TOO_SOON",
			"這個模型剛剛才試跑過", "同一個模型 60 秒內只會試跑一次，免得重複扣款。")
	case errors.Is(err, ai.ErrAINotConfigured):
		ErrorResponse(c, http.StatusConflict, "AI_NOT_CONFIGURED",
			"尚未設定翻譯服務金鑰", "先到 設定 → API 金鑰 存一組 Claude 金鑰。")
	case errors.Is(err, ai.ErrAIUnauthorized):
		ErrorResponse(c, http.StatusConflict, "AI_UNAUTHORIZED", "翻譯服務金鑰無效", "請到 設定 → API 金鑰 檢查金鑰。")
	default:
		InternalServerError(c, "試跑失敗："+err.Error())
	}
}
