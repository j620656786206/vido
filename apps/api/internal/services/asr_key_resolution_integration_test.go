package services

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vido/api/internal/ai"
)

// This is sub-5-2 AC #7(a) — the end-to-end proof that the ASR boot-gate is
// gone, run against a REAL migrated secrets table and the REAL AES-256-GCM
// secrets service rather than a fake (Rule 15 / the bugfix-20-1 precedent: a
// mocked repo hid a read path that silently returned the zero value for months).
// `newRealSecretsService` is shared with key_resolution_integration_test.go.
//
// The scenario is the one the product copy had to apologise for: a NAS boots
// with no ASR key, the owner saves one in /settings/keys, and — before this
// story — transcription stayed 503 until the container was restarted.

// asrStack builds the production wiring shape: one resolver, one holder, one
// transcription service. Nothing in it is rebuilt between assertions, which is
// the whole point.
func asrStack(t *testing.T, baseURL string) (*KeySettingsService, *TranscriptionService, *ASRProviderHolder) {
	t.Helper()
	secretsSvc, cleanup := newRealSecretsService(t)
	t.Cleanup(cleanup)

	resolver := NewKeyResolver(secretsSvc, EnvKeys{}, nil)
	holder := NewASRProviderHolder(resolver, baseURL, "", nil)
	extractor := &AudioExtractorService{available: true, semaphore: make(chan struct{}, 1)}
	svc := NewTranscriptionService(extractor, holder, nil, nil)

	return NewKeySettingsService(resolver, secretsSvc, true), svc, holder
}

func TestASRKeyResolution_SavedKeyUnGatesTranscriptionWithoutRestart(t *testing.T) {
	ctx := context.Background()
	keySettings, svc, holder := asrStack(t, "")

	require.False(t, svc.IsAvailable(), "precondition: a keyless boot has no ASR")
	require.False(t, holder.IsConfigured(ctx))
	_, err := holder.Get(ctx)
	require.ErrorIs(t, err, ai.ErrWhisperNotConfigured)

	// Exactly what PUT /api/v1/settings/keys does.
	require.NoError(t, keySettings.Save(ctx, map[KeyName]string{KeyOpenAI: "sk-saved-on-the-nas"}))

	// No object is reconstructed here. That is the acceptance criterion.
	assert.True(t, svc.IsAvailable(),
		"saving the cloud ASR key must un-gate transcription with no restart")
	assert.True(t, holder.IsConfigured(ctx))
	client, err := holder.Get(ctx)
	require.NoError(t, err)
	assert.NotNil(t, client)
}

// The settings page reports where a key came from; the encrypted store must win
// over the environment, or an in-app edit would be silently inert (the Break 1
// trap sub-2-1a documented, re-checked here for the ASR row).
func TestASRKeyResolution_StoredKeyOutranksTheEnvironment(t *testing.T) {
	ctx := context.Background()
	secretsSvc, cleanup := newRealSecretsService(t)
	defer cleanup()

	resolver := NewKeyResolver(secretsSvc, EnvKeys{OpenAI: "sk-from-env"}, nil)
	holder := NewASRProviderHolder(resolver, "", "", nil)

	fromEnv, err := holder.Get(ctx)
	require.NoError(t, err)

	require.NoError(t, secretsSvc.Store(ctx, SecretNameOpenAI, "sk-from-settings-page"))

	fromSecret, err := holder.Get(ctx)
	require.NoError(t, err)
	assert.NotSame(t, fromEnv, fromSecret,
		"the stored key must reach a NEW client, not the boot-time env one")

	state := NewKeySettingsService(resolver, secretsSvc, true).List(ctx)
	var openai KeyState
	for _, k := range state.Keys {
		if k.Name == KeyOpenAI {
			openai = k
		}
	}
	assert.Equal(t, KeySourceSecret, openai.Source)
	assert.True(t, openai.Configured)
}

// Clearing the row (an explicit "") deletes the secret and reverts resolution to
// the environment — the "I mistyped my key" recovery path. With no env value
// behind it, availability goes back to false in the same process.
func TestASRKeyResolution_ClearingTheKeyRevokesAvailability(t *testing.T) {
	ctx := context.Background()
	keySettings, svc, _ := asrStack(t, "")

	require.NoError(t, keySettings.Save(ctx, map[KeyName]string{KeyOpenAI: "sk-typo"}))
	require.True(t, svc.IsAvailable())

	require.NoError(t, keySettings.Save(ctx, map[KeyName]string{KeyOpenAI: ""}))

	assert.False(t, svc.IsAvailable(),
		"clearing the only configured key must re-gate transcription immediately")
}

// AC #3 end-to-end: a self-hosted engine is available on a keyless install, and
// the settings page still honestly reports the KEY as unset — availability and
// key-configured are different questions.
func TestASRKeyResolution_SelfHostedIsAvailableWithNoKeyAtAll(t *testing.T) {
	ctx := context.Background()
	keySettings, svc, holder := asrStack(t, "http://nas.local:8000/v1/audio/transcriptions")

	assert.True(t, svc.IsAvailable(),
		"a self-hosted ASR endpoint needs no key — requiring one forced deployments to invent a dummy OPENAI_API_KEY")
	assert.True(t, holder.IsConfigured(ctx))

	for _, k := range keySettings.List(ctx).Keys {
		if k.Name == KeyOpenAI {
			assert.False(t, k.Configured, "the KEY is genuinely unset; only the ENDPOINT is configured")
			assert.Equal(t, KeySourceNone, k.Source)
		}
	}
}
