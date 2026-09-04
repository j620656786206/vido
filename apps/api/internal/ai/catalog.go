package ai

import "sort"

// ModelInfo is one selectable translation model as the settings/consent screens
// need to see it: what to call it, what it costs, and what Vido actually
// measured it doing.
//
// [@contract-v1] — served verbatim by GET /api/v1/settings/models (sub-6-8a
// AC #2) and consumed by sub-6-8b. Adding a field is additive; renaming or
// removing one is a Rule 20 bump.
type ModelInfo struct {
	ID          string `json:"id"`
	Provider    string `json:"provider"`
	DisplayName string `json:"display_name"`
	// Tier is the coarse speed/quality band a UI can group by:
	// fast | balanced | max.
	Tier        string  `json:"tier"`
	InputPer1M  float64 `json:"input_per_1m"`
	OutputPer1M float64 `json:"output_per_1m"`
	IsDefault   bool    `json:"is_default"`
	// QualityGrade is a MEASURED grade, never a vendor claim: it is filled only
	// for models Vido has actually blind-scored on its own library. Empty means
	// "not evaluated yet" and the UI must say so rather than implying parity.
	QualityGrade string `json:"quality_grade,omitempty"`
	QualityNote  string `json:"quality_note,omitempty"`
}

// Provider names as they appear on ModelInfo.
const (
	ProviderNameClaude = "claude"
	ProviderNameGemini = "gemini"
)

// Model tiers.
const (
	TierFast     = "fast"
	TierBalanced = "balanced"
	TierMax      = "max"
)

// evalNote is the provenance stamp on every grade below. A grade with no
// provenance is a marketing claim; this one points at a specific run.
const evalNote = "Vido 實測 2026-09（eval-1 盲測，10,304 句）"

// modelMetadata is everything about a model that is NOT its price. It is keyed
// by the same ids as defaultLLMPricing and maintained WITH it — a model priced
// but not described here would be unsellable, and one described but not priced
// would be quoted at the fallback tier. TestCatalog_EveryPricedModelIsDescribed
// is the guard.
//
// `retired` keeps a model out of the catalog while leaving its pricing row in
// place: a shut-down model must never be offered as a choice (it 404s), but a
// deployment that still has runs recorded against it must keep metering them at
// the rate they were billed.
var modelMetadata = map[string]struct {
	provider     string
	displayName  string
	tier         string
	qualityGrade string
	qualityNote  string
	retired      bool
}{
	"claude-sonnet-5": {
		provider: ProviderNameClaude, displayName: "Claude Sonnet 5", tier: TierBalanced,
		// eval-1: 0 分率 1.3%、2 分率 89.6% over the full 10,304-cue corpus.
		qualityGrade: "A", qualityNote: evalNote,
	},
	"claude-haiku-4-5": {
		provider: ProviderNameClaude, displayName: "Claude Haiku 4.5", tier: TierFast,
		// eval-1: 0 分率 3.6%、2 分率 71.8% — usable, and 2.7× cheaper.
		qualityGrade: "B", qualityNote: evalNote,
	},
	"claude-sonnet-4-6": {provider: ProviderNameClaude, displayName: "Claude Sonnet 4.6", tier: TierBalanced},
	"claude-opus-4-8":   {provider: ProviderNameClaude, displayName: "Claude Opus 4.8", tier: TierMax},

	"gemini-3.7-flash":      {provider: ProviderNameGemini, displayName: "Gemini 3.7 Flash", tier: TierBalanced},
	"gemini-3.6-flash":      {provider: ProviderNameGemini, displayName: "Gemini 3.6 Flash", tier: TierBalanced},
	"gemini-3.5-flash-lite": {provider: ProviderNameGemini, displayName: "Gemini 3.5 Flash Lite", tier: TierFast},
	"gemini-2.5-flash":      {provider: ProviderNameGemini, displayName: "Gemini 2.5 Flash", tier: TierBalanced},
	"gemini-2.5-flash-lite": {provider: ProviderNameGemini, displayName: "Gemini 2.5 Flash Lite", tier: TierFast},
	// Shut down by Google 2026-06-01. Priced (so old runs still meter
	// correctly), never offered.
	"gemini-2.0-flash": {provider: ProviderNameGemini, displayName: "Gemini 2.0 Flash (retired)", tier: TierFast, retired: true},
}

// Catalog returns every SELECTABLE model, cheapest first within a provider and
// providers in a stable order, with IsDefault set on DefaultClaudeModel.
// Retired models are excluded (see modelMetadata).
//
// The caller decides which providers the user can actually reach — this
// package knows prices and grades, not which keys are configured.
func Catalog() []ModelInfo {
	out := make([]ModelInfo, 0, len(modelMetadata))
	for id, meta := range modelMetadata {
		if meta.retired {
			continue
		}
		pricing, priced := defaultLLMPricing[id]
		if !priced {
			// Unpriced means the metering fallback would invent a number for
			// it. Never offer a model whose cost we would misreport.
			continue
		}
		out = append(out, ModelInfo{
			ID:           id,
			Provider:     meta.provider,
			DisplayName:  meta.displayName,
			Tier:         meta.tier,
			InputPer1M:   pricing.InputPer1M,
			OutputPer1M:  pricing.OutputPer1M,
			IsDefault:    id == DefaultClaudeModel,
			QualityGrade: meta.qualityGrade,
			QualityNote:  meta.qualityNote,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Provider != out[j].Provider {
			// Claude before Gemini: the translation default lives there, and a
			// stable order keeps the settings list from reshuffling per request.
			return out[i].Provider < out[j].Provider
		}
		if out[i].OutputPer1M != out[j].OutputPer1M {
			return out[i].OutputPer1M < out[j].OutputPer1M
		}
		return out[i].ID < out[j].ID
	})
	return out
}

// ProviderOf returns the provider that serves a model id, or "" when the id is
// unknown to the catalog.
func ProviderOf(model string) string {
	if meta, ok := modelMetadata[model]; ok {
		return meta.provider
	}
	return ""
}

// IsSelectableModel reports whether a model id may be requested for a run: it
// must be described, priced and not retired. This is the validation the API
// applies before a model id reaches a provider (sub-6-8a AC #4) — an unknown
// id must fail at the boundary with a 400, not at the provider with a 404
// after the user has already consented to a charge.
func IsSelectableModel(model string) bool {
	meta, ok := modelMetadata[model]
	if !ok || meta.retired {
		return false
	}
	_, priced := defaultLLMPricing[model]
	return priced
}
