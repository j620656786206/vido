package services

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vido/api/internal/ai"
)

// ─── sub-6-8a AC #2: the list is what this deployment can actually run ─────

func catalogWith(t *testing.T, claudeKey, effectiveDefault string) *ModelCatalogService {
	t.Helper()
	return NewModelCatalogService(
		NewKeyResolver(&fakeSecrets{}, EnvKeys{Claude: claudeKey}, nil),
		func() string { return effectiveDefault },
	)
}

func idsOf(models []ai.ModelInfo) []string {
	out := make([]string, 0, len(models))
	for _, m := range models {
		out = append(out, m.ID)
	}
	return out
}

func TestModelCatalog_ListsClaudeOnlyBecauseTranslationDispatchIsClaudeOnly(t *testing.T) {
	ctx := context.Background()

	listed := catalogWith(t, "sk-one", "").Available(ctx)
	require.NotEmpty(t, listed)
	for _, m := range listed {
		assert.Equal(t, ai.ProviderNameClaude, m.Provider,
			// CR H2: TranslationService is built with the Claude holder and
			// nothing else. A Gemini id would be posted to api.anthropic.com
			// and answered with a 404 — after the user consented to a price.
			"%s: offering a model the pipeline cannot dispatch to sells a guaranteed paid failure", m.ID)
	}

	assert.Empty(t, catalogWith(t, "", "").Available(ctx),
		"no Claude key is an empty list, not an error — the settings page must still render")
}

func TestModelCatalog_DefaultFollowsTheDeploymentOverrideNotThePackageConstant(t *testing.T) {
	ctx := context.Background()

	// The common case: no CLAUDE_MODEL, so the package default applies.
	assert.Equal(t, ai.DefaultClaudeModel, catalogWith(t, "sk-one", "").DefaultModel(ctx))

	// CR H1: an operator who set CLAUDE_MODEL=haiku runs Haiku when model_id
	// is omitted. Reporting Sonnet here would quote every sweep 2.7× high and,
	// once the picker pre-selects what this says, silently upgrade their runs.
	haikuInstall := catalogWith(t, "sk-one", "claude-haiku-4-5")
	assert.Equal(t, "claude-haiku-4-5", haikuInstall.DefaultModel(ctx))

	var marked []string
	for _, m := range haikuInstall.Available(ctx) {
		if m.IsDefault {
			marked = append(marked, m.ID)
		}
	}
	assert.Equal(t, []string{"claude-haiku-4-5"}, marked, "exactly one entry may claim to be the default")

	assert.Empty(t, catalogWith(t, "", "").DefaultModel(ctx))
}

func TestModelCatalog_AnUnlistableOverrideStillPreSelectsSomethingReal(t *testing.T) {
	// CLAUDE_MODEL may legally name an alias or preview id the catalog cannot
	// render. The picker must still land on a real entry rather than an id it
	// has no row for.
	c := catalogWith(t, "sk-one", "claude-some-preview-alias")

	def := c.DefaultModel(context.Background())
	assert.Contains(t, idsOf(c.Available(context.Background())), def)
}

func TestModelCatalog_SupportsIsTheRequestBoundaryCheck(t *testing.T) {
	ctx := context.Background()
	claude := catalogWith(t, "sk-one", "")

	assert.True(t, claude.Supports(ctx, ""), "absent means 'the default' and is always legal")
	assert.True(t, claude.Supports(ctx, "claude-haiku-4-5"))
	assert.False(t, claude.Supports(ctx, "gemini-2.5-flash"),
		"a model the pipeline cannot dispatch to must be refused before a batch is started")
	assert.False(t, claude.Supports(ctx, "gpt-4o"))
	assert.False(t, claude.Supports(ctx, "gemini-2.0-flash"), "a retired model is never selectable")

	assert.False(t, catalogWith(t, "", "").Supports(ctx, "claude-sonnet-5"),
		"a keyless install supports no model at all")
}

func TestModelCatalog_NilDefaultSourceFallsBackToThePackageDefault(t *testing.T) {
	c := NewModelCatalogService(NewKeyResolver(&fakeSecrets{}, EnvKeys{Claude: "sk-one"}, nil), nil)

	assert.Equal(t, ai.DefaultClaudeModel, c.DefaultModel(context.Background()))
}
