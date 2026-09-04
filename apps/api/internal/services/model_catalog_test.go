package services

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vido/api/internal/ai"
)

// ─── sub-6-8a AC #2: the list is narrowed to reachable providers ───────────

func catalogWith(t *testing.T, claudeKey string, gemini bool) *ModelCatalogService {
	t.Helper()
	return NewModelCatalogService(
		NewKeyResolver(&fakeSecrets{}, EnvKeys{Claude: claudeKey}, nil),
		func() bool { return gemini },
	)
}

func idsOf(models []ai.ModelInfo) []string {
	out := make([]string, 0, len(models))
	for _, m := range models {
		out = append(out, m.ID)
	}
	return out
}

func TestModelCatalog_OnlyListsProvidersWhoseKeyResolves(t *testing.T) {
	ctx := context.Background()

	claudeOnly := idsOf(catalogWith(t, "sk-one", false).Available(ctx))
	require.NotEmpty(t, claudeOnly)
	for _, id := range claudeOnly {
		assert.Equal(t, ai.ProviderNameClaude, ai.ProviderOf(id),
			"offering a Gemini model to a Claude-only install would quote a price for a run that fails on its first call")
	}

	both := idsOf(catalogWith(t, "sk-one", true).Available(ctx))
	assert.Greater(t, len(both), len(claudeOnly), "a Gemini key adds Gemini models")
	assert.Subset(t, both, claudeOnly)

	geminiOnly := idsOf(catalogWith(t, "", true).Available(ctx))
	require.NotEmpty(t, geminiOnly)
	for _, id := range geminiOnly {
		assert.Equal(t, ai.ProviderNameGemini, ai.ProviderOf(id))
	}

	assert.Empty(t, catalogWith(t, "", false).Available(ctx),
		"no keys is an empty list, not an error — the settings page must still render")
}

func TestModelCatalog_DefaultModelFollowsWhatIsReachable(t *testing.T) {
	ctx := context.Background()

	assert.Equal(t, ai.DefaultClaudeModel, catalogWith(t, "sk-one", false).DefaultModel(ctx))

	// A Gemini-only install must not pre-select a Claude model it cannot run.
	geminiDefault := catalogWith(t, "", true).DefaultModel(ctx)
	require.NotEmpty(t, geminiDefault)
	assert.Equal(t, ai.ProviderNameGemini, ai.ProviderOf(geminiDefault))

	assert.Empty(t, catalogWith(t, "", false).DefaultModel(ctx))
}

func TestModelCatalog_SupportsIsTheRequestBoundaryCheck(t *testing.T) {
	ctx := context.Background()
	claude := catalogWith(t, "sk-one", false)

	assert.True(t, claude.Supports(ctx, ""), "absent means 'the default' and is always legal")
	assert.True(t, claude.Supports(ctx, "claude-haiku-4-5"))
	assert.False(t, claude.Supports(ctx, "gemini-2.5-flash"),
		"a model whose provider has no key must be refused before a batch is started")
	assert.False(t, claude.Supports(ctx, "gpt-4o"))
	assert.False(t, claude.Supports(ctx, "gemini-2.0-flash"), "a retired model is never selectable")

	assert.False(t, catalogWith(t, "", false).Supports(ctx, "claude-sonnet-5"),
		"a keyless install supports no model at all")
}

func TestModelCatalog_NilGeminiProbeMeansNoGemini(t *testing.T) {
	c := NewModelCatalogService(NewKeyResolver(&fakeSecrets{}, EnvKeys{Claude: "sk-one"}, nil), nil)

	for _, id := range idsOf(c.Available(context.Background())) {
		assert.NotEqual(t, ai.ProviderNameGemini, ai.ProviderOf(id),
			"an unknown key state must hide the model, not offer it: a hidden model costs nothing, an offered one costs a failed run")
	}
}
