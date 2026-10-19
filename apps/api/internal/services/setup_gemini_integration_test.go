package services

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/vido/api/internal/ai"
)

// setup-wizard-byok AC #1: a Gemini key typed into the wizard, and nothing
// else, must land where translation reads it — run against the REAL
// AES-256-GCM secrets service (Rule 28), not a mocked resolver. Before this
// story the wizard had no Gemini field and the comment said Gemini was
// env-only; a Gemini-only newcomer could not translate without editing env.
func TestSetupWizard_GeminiOnlyKeyReachesTranslation(t *testing.T) {
	ctx := context.Background()
	secretsSvc, cleanup := newRealSecretsService(t)
	defer cleanup()

	resolver := NewKeyResolver(secretsSvc, EnvKeys{}, nil)
	require.False(t, resolver.Has(ctx, KeyGemini), "precondition: nothing configured")

	repo := new(MockSettingsRepo)
	repo.On("GetBool", mock.Anything, "setup_completed").
		Return(false, errors.New("setting with key setup_completed not found"))
	repo.On("SetString", mock.Anything, "language", "zh-TW").Return(nil)
	repo.On("SetBool", mock.Anything, "setup_completed", true).Return(nil)

	svc := NewSetupService(repo, secretsSvc)
	svc.SetKeyWriter(NewKeySettingsService(resolver, secretsSvc, true))

	require.NoError(t, svc.CompleteSetup(ctx, SetupConfig{Language: "zh-TW", GeminiApiKey: "AIza-from-wizard"}))

	value, source, err := resolver.Get(ctx, KeyGemini)
	require.NoError(t, err)
	assert.Equal(t, "AIza-from-wizard", value, "round-tripped through real encryption")
	assert.Equal(t, KeySourceSecret, source)
	assert.False(t, resolver.Has(ctx, KeyClaude), "no Claude key was given")

	// THE assertion: with only the wizard's Gemini key, the default translation
	// provider is Gemini — no settings page, no env var, no restart.
	dispatcher := NewTranslationDispatcher(
		NewClaudeProviderHolder(resolver, "", nil),
		NewGeminiProviderHolder(resolver, 0, nil, nil),
	)
	p, err := dispatcher.Get(ctx)
	require.NoError(t, err)
	_, isGemini := p.(*ai.OpenAICompatProvider)
	assert.True(t, isGemini, "Gemini-only install translates on Gemini, got %T", p)
	repo.AssertExpectations(t)
}
