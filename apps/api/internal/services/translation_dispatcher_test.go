package services

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vido/api/internal/ai"
	"github.com/vido/api/internal/models"
)

// gemini-translation-dispatch: which holder a model goes to, and what an
// omitted model means for each key combination.

func dispatcherWith(env EnvKeys, gap time.Duration) *TranslationDispatcher {
	resolver := NewKeyResolver(&fakeSecrets{}, env, nil)
	claude := NewClaudeProviderHolder(resolver, "", nil)
	gemini := NewGeminiProviderHolder(resolver, gap, nil, nil)
	return NewTranslationDispatcher(claude, gemini)
}

func TestDispatcher_ClaudeIDsGoToClaudeGeminiIDsGoToGemini(t *testing.T) {
	d := dispatcherWith(EnvKeys{Claude: "sk-ant", Gemini: "gm"}, 13*time.Second)
	ctx := context.Background()

	c, err := d.GetFor(ctx, "claude-haiku-4-5")
	require.NoError(t, err)
	_, isClaude := c.(*ai.ClaudeProvider)
	assert.True(t, isClaude)

	g, err := d.GetFor(ctx, "gemini-3.5-flash-lite")
	require.NoError(t, err)
	compat, isCompat := g.(*ai.OpenAICompatProvider)
	require.True(t, isCompat, "Gemini runs on the OpenAI-compatible client")
	assert.Equal(t, "gemini-3.5-flash-lite", compat.Model())
	assert.Equal(t, 13*time.Second, compat.MinInterval(), "the pacing the holder was built with travels with the client")
	assert.Equal(t, 13*time.Second, d.MinIntervalFor("gemini-3.5-flash-lite"))
	assert.Zero(t, d.MinIntervalFor("claude-sonnet-5"))
}

func TestDispatcher_DefaultIsClaudeWheneverAClaudeKeyResolves(t *testing.T) {
	d := dispatcherWith(EnvKeys{Claude: "sk-ant", Gemini: "gm"}, 0)
	assert.Equal(t, ai.DefaultClaudeModel, d.EffectiveModel(), "Alexyu 2026-10-14: the default stays Claude")
	p, err := d.Get(context.Background())
	require.NoError(t, err)
	_, isClaude := p.(*ai.ClaudeProvider)
	assert.True(t, isClaude)
}

func TestDispatcher_GeminiOnlyInstallDefaultsToGemini(t *testing.T) {
	d := dispatcherWith(EnvKeys{Gemini: "gm"}, 0)
	ctx := context.Background()
	assert.True(t, d.IsConfigured(ctx), "one key is enough to translate")
	assert.Equal(t, DefaultGeminiTranslationModel, d.EffectiveModel())
	p, err := d.Get(ctx)
	require.NoError(t, err)
	_, isCompat := p.(*ai.OpenAICompatProvider)
	assert.True(t, isCompat)
}

func TestDispatcher_GeminiModelWithoutAGeminiKeyIsRefusedBeforeAnyCall(t *testing.T) {
	d := dispatcherWith(EnvKeys{Claude: "sk-ant"}, 0)
	_, err := d.GetFor(context.Background(), "gemini-3.5-flash-lite")
	assert.ErrorIs(t, err, ai.ErrAINotConfigured)
}

func TestDispatcher_NothingConfiguredIsNotConfigured(t *testing.T) {
	d := dispatcherWith(EnvKeys{}, 0)
	assert.False(t, d.IsConfigured(context.Background()))
	assert.Equal(t, ai.DefaultClaudeModel, d.EffectiveModel())
}

func TestDispatcher_KeySavedLaterReachesTheNextCall(t *testing.T) {
	secrets := &fakeSecrets{}
	resolver := NewKeyResolver(secrets, EnvKeys{Claude: "sk-ant"}, nil)
	d := NewTranslationDispatcher(NewClaudeProviderHolder(resolver, "", nil), NewGeminiProviderHolder(resolver, 0, nil, nil))
	ctx := context.Background()
	_, err := d.GetFor(ctx, "gemini-3.5-flash-lite")
	require.ErrorIs(t, err, ai.ErrAINotConfigured)

	require.NoError(t, secrets.Store(ctx, SecretNameGemini, "gm-late"))
	_, err = d.GetFor(ctx, "gemini-3.5-flash-lite")
	assert.NoError(t, err)
}

func TestGeminiHolder_RefusesNonGeminiModelsAndCachesByKeyAndModel(t *testing.T) {
	resolver := NewKeyResolver(&fakeSecrets{}, EnvKeys{Gemini: "gm"}, nil)
	h := NewGeminiProviderHolder(resolver, 0, nil, nil)
	ctx := context.Background()
	_, err := h.GetFor(ctx, "claude-sonnet-5")
	assert.ErrorIs(t, err, ai.ErrAIModelNotFound)
	_, err = h.GetFor(ctx, "gemini-2.0-flash")
	assert.ErrorIs(t, err, ai.ErrAIModelNotFound, "a retired model is not selectable")

	a, err := h.GetFor(ctx, "gemini-3.5-flash-lite")
	require.NoError(t, err)
	b, err := h.GetFor(ctx, "gemini-3.5-flash-lite")
	require.NoError(t, err)
	assert.Same(t, a, b)
	c, err := h.GetFor(ctx, "gemini-3.6-flash")
	require.NoError(t, err)
	assert.NotSame(t, a, c)
}

func TestModelCatalog_ListsEachProviderOnlyWhenItsKeyResolves(t *testing.T) {
	ctx := context.Background()
	both := NewModelCatalogService(NewKeyResolver(&fakeSecrets{}, EnvKeys{Claude: "sk", Gemini: "gm"}, nil), nil).Available(ctx)
	providers := map[string]bool{}
	for _, m := range both {
		providers[m.Provider] = true
	}
	assert.True(t, providers[ai.ProviderNameClaude])
	assert.True(t, providers[ai.ProviderNameGemini])

	geminiOnly := NewModelCatalogService(NewKeyResolver(&fakeSecrets{}, EnvKeys{Gemini: "gm"}, nil), nil).Available(ctx)
	require.NotEmpty(t, geminiOnly)
	for _, m := range geminiOnly {
		assert.Equal(t, ai.ProviderNameGemini, m.Provider, "%s offered without a Claude key", m.ID)
	}
	assert.Empty(t, NewModelCatalogService(NewKeyResolver(&fakeSecrets{}, EnvKeys{}, nil), nil).Available(ctx))
}

func TestGenerationCandidates_PacedModelQuotesMoreMinutes(t *testing.T) {
	// 60 runtime minutes ≈ 835 cues ≈ 42 batches of 20 (Flash-Lite's batch,
	// disc-2026-10-translation-batch-size-default); at 13 s a batch that is
	// ~9 extra minutes.
	svc := NewGenerationCandidateService(&stubMovieFinder{movies: []models.Movie{movieRow("m1", "Dune", "/m/dune.mkv", 60, "")}}, nil, nil, &stubPredictor{probeRoute: RouteExtract}, false, 0, nil)
	svc.SetModelCatalog(&stubCandidateCatalog{
		models: []ai.ModelInfo{
			{ID: "claude-sonnet-5", Provider: ai.ProviderNameClaude},
			{ID: "gemini-3.5-flash-lite", Provider: ai.ProviderNameGemini},
		},
		byDeflt: "claude-sonnet-5",
	})
	svc.SetTranslationPacing(func(model string) time.Duration {
		if ai.ProviderOf(model) == ai.ProviderNameGemini {
			return 13 * time.Second
		}
		return 0
	})
	res, err := svc.Analyze(context.Background(), nil)
	require.NoError(t, err)
	claude := res.EstimatedMinutesByModel["claude-sonnet-5"]
	gemini := res.EstimatedMinutesByModel["gemini-3.5-flash-lite"]
	// Plus the base processing share: Flash-Lite is unmeasured, so it quotes
	// the 10-cue anchor share while Sonnet quotes its own 50-cue share.
	wantGap := 9 + 60*(translationTimeShareAnchor-translationTimeShareByModel["claude-sonnet-5"])
	assert.InDelta(t, wantGap, gemini-claude, 1, "claude %v gemini %v", claude, gemini)
}
