package ai

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ─── sub-6-8a AC #1/#2: the catalog ────────────────────────────────────────

// TestCatalog_EveryPricedModelIsDescribed is the maintenance guard: the two
// tables are keyed by the same ids and a model in one but not the other is a
// bug with money attached — an undescribed model cannot be offered, and an
// unpriced one would be quoted at the fallback tier.
func TestCatalog_EveryPricedModelIsDescribed(t *testing.T) {
	for id := range defaultLLMPricing {
		meta, ok := modelMetadata[id]
		require.True(t, ok, "priced model %q has no catalog metadata", id)
		assert.NotEmpty(t, meta.provider, "%q needs a provider", id)
		assert.NotEmpty(t, meta.displayName, "%q needs a display name", id)
		assert.Contains(t, []string{TierFast, TierBalanced, TierMax}, meta.tier, "%q has an unknown tier", id)
	}
	for id, meta := range modelMetadata {
		_, priced := defaultLLMPricing[id]
		assert.True(t, priced, "described model %q has no price — it would be metered at the fallback rate", id)
		if meta.qualityGrade != "" {
			assert.NotEmpty(t, meta.qualityNote,
				"%q carries a grade with no provenance — an ungrounded grade is a marketing claim", id)
		}
	}
}

func TestCatalog_DefaultIsSonnetAndIsTheCallersToStamp(t *testing.T) {
	assert.Equal(t, "claude-sonnet-5", DefaultClaudeModel,
		"eval-1 ruled the default on measured quality (1.3% unusable vs Haiku's 3.6%)")

	// CR H1: this package must NOT decide which entry is "the default" — a
	// deployment's CLAUDE_MODEL override is the truth, and stamping the
	// constant here priced every quote at Sonnet for an operator whose runs
	// billed Haiku.
	for _, m := range Catalog() {
		assert.False(t, m.IsDefault,
			"%s: the catalog knows prices, not configuration — the caller stamps IsDefault", m.ID)
	}
}

func TestCatalog_ExcludesRetiredModelsButKeepsPricingThem(t *testing.T) {
	for _, m := range Catalog() {
		assert.NotEqual(t, "gemini-2.0-flash", m.ID,
			"a model Google shut down must never be offered — it 404s on the first call")
	}
	assert.False(t, IsSelectableModel("gemini-2.0-flash"))

	// …but a run recorded against it must still meter at the rate it was billed.
	assert.Equal(t, ModelPricing{InputPer1M: 0.10, OutputPer1M: 0.40}, PricingFor("gemini-2.0-flash"))
}

func TestCatalog_CarriesPricesAndMeasuredGrades(t *testing.T) {
	byID := map[string]ModelInfo{}
	for _, m := range Catalog() {
		byID[m.ID] = m
	}

	sonnet := byID["claude-sonnet-5"]
	require.NotEmpty(t, sonnet.ID)
	assert.Equal(t, ProviderNameClaude, sonnet.Provider)
	assert.Equal(t, 3.0, sonnet.InputPer1M)
	assert.Equal(t, 15.0, sonnet.OutputPer1M)
	assert.Equal(t, "A", sonnet.QualityGrade)
	assert.Contains(t, sonnet.QualityNote, "eval-1")

	haiku := byID["claude-haiku-4-5"]
	assert.Equal(t, "B", haiku.QualityGrade)

	// A model nobody has blind-scored carries NO grade — silence, not a guess.
	assert.Empty(t, byID["claude-opus-4-8"].QualityGrade,
		"an unevaluated model must not imply parity with an evaluated one")
	assert.Empty(t, byID["claude-opus-4-8"].QualityNote)
}

func TestCatalog_IsOrderedStablyCheapestFirstWithinProvider(t *testing.T) {
	first := Catalog()
	assert.Equal(t, first, Catalog(), "the list must not reshuffle between requests")

	var lastProvider string
	var lastOutput float64
	for _, m := range first {
		if m.Provider != lastProvider {
			lastProvider, lastOutput = m.Provider, m.OutputPer1M
			continue
		}
		assert.GreaterOrEqual(t, m.OutputPer1M, lastOutput,
			"within a provider the cheap options come first: %s", m.ID)
		lastOutput = m.OutputPer1M
	}
	assert.Equal(t, ProviderNameClaude, first[0].Provider, "the translation default's provider leads")
}

func TestIsSelectableModel(t *testing.T) {
	assert.True(t, IsSelectableModel("claude-sonnet-5"))
	assert.True(t, IsSelectableModel("claude-haiku-4-5"))
	assert.False(t, IsSelectableModel(""), "empty is 'the default', which callers handle before asking")
	assert.False(t, IsSelectableModel("claude-sonnet-5 "), "no fuzzy matching — an id is an id")
	assert.False(t, IsSelectableModel("gpt-4o"))
	assert.Equal(t, ProviderNameGemini, ProviderOf("gemini-2.5-flash"))
	assert.Empty(t, ProviderOf("nope"))
}

// ─── AC #5: the per-run model rides the ctx ────────────────────────────────

func TestModelIDContext(t *testing.T) {
	ctx := context.Background()
	assert.Empty(t, ModelIDFromContext(ctx), "no choice means the deployment default, never a guess")

	assert.Empty(t, ModelIDFromContext(WithModelID(ctx, "")),
		"an unset choice passes through without special-casing at the call site")

	pinned := WithModelID(ctx, "claude-haiku-4-5")
	assert.Equal(t, "claude-haiku-4-5", ModelIDFromContext(pinned))
	assert.Empty(t, ModelIDFromContext(ctx), "the parent ctx is untouched")

	// A nested choice wins, which is what lets a per-item option override a
	// batch-wide one.
	assert.Equal(t, "claude-opus-4-8", ModelIDFromContext(WithModelID(pinned, "claude-opus-4-8")))
}
