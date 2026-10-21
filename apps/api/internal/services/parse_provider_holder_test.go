package services

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vido/api/internal/ai"
)

// staticKeys is a KeyResolver over a fixed map, for the holder's choice rules.
type staticKeys map[KeyName]string

func (k staticKeys) Get(_ context.Context, name KeyName) (string, KeySource, error) {
	if v := k[name]; v != "" {
		return v, KeySourceSecret, nil
	}
	return "", KeySourceNone, nil
}

func (k staticKeys) Has(ctx context.Context, name KeyName) bool {
	v, _, _ := k.Get(ctx, name)
	return v != ""
}

func TestParseProviderHolder_ChoiceRules(t *testing.T) {
	ctx := context.Background()
	cases := []struct {
		name      string
		preferred string
		keys      staticKeys
		want      string
	}{
		{"preferred has a key", "claude", staticKeys{KeyClaude: "c", KeyGemini: "g"}, "claude"},
		{"default prefers gemini when both exist", "", staticKeys{KeyClaude: "c", KeyGemini: "g"}, "gemini"},
		{"preferred missing → the other one", "gemini", staticKeys{KeyClaude: "c"}, "claude"},
		{"preferred missing → the other one (claude preferred)", "claude", staticKeys{KeyGemini: "g"}, "gemini"},
		{"neither → not configured", "gemini", staticKeys{}, ""},
		{"unknown AI_PROVIDER is treated as gemini, never a boot failure", "openai", staticKeys{KeyClaude: "c", KeyGemini: "g"}, "gemini"},
		{"case-insensitive", "CLAUDE", staticKeys{KeyClaude: "c", KeyGemini: "g"}, "claude"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := NewParseProviderHolder(tc.keys, tc.preferred, "", "", nil, (&fakeParseFactory{}).build, nil)
			assert.Equal(t, tc.want, h.ProviderName(ctx))
			assert.Equal(t, tc.want != "", h.IsConfigured(ctx))
		})
	}
}

func TestParseProviderHolder_NoKeyIsTheSentinel(t *testing.T) {
	h := NewParseProviderHolder(staticKeys{}, "gemini", "", "", nil, nil, nil)
	_, err := h.Resolve(context.Background())
	assert.ErrorIs(t, err, ai.ErrAINotConfigured)
}

func TestParseProviderHolder_DeletedKeysDropTheCachedClients(t *testing.T) {
	ctx := context.Background()
	keys := staticKeys{KeyGemini: "g"}
	h := NewParseProviderHolder(keys, "gemini", "", "", nil, (&fakeParseFactory{}).build, nil)
	_, err := h.Resolve(ctx)
	require.NoError(t, err)
	require.Len(t, h.clients, 1)

	delete(keys, KeyGemini)
	_, err = h.Resolve(ctx)
	require.ErrorIs(t, err, ai.ErrAINotConfigured)
	assert.Empty(t, h.clients, "a deleted key must not stay live in memory")
}

func TestParseProviderHolder_DefaultFactoryBuildsNativeProviders(t *testing.T) {
	gov := ai.NewGovernor(1, 1, 1)
	_, isGemini := defaultParseProviderFactory(ai.ProviderGemini, "g", ai.DefaultGeminiModel, gov).(*ai.GeminiProvider)
	assert.True(t, isGemini, "Gemini parsing stays on the native provider, not the translation client")
	_, isClaude := defaultParseProviderFactory(ai.ProviderClaude, "c", ai.DefaultClaudeModel, gov).(*ai.ClaudeProvider)
	assert.True(t, isClaude)
}
