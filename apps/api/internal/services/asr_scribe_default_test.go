package services

import (
	"context"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vido/api/internal/ai"
	"github.com/vido/api/internal/models"
)

// asr-scribe-b: which engine the holder hands out, given the keys and the
// operator's choice. Alexyu 2026-10-14 ruling after the ASR exam: an
// ElevenLabs key makes Scribe the default; Whisper stays one choice away.

func holderWith(env EnvKeys, preference string) *ASRProviderHolder {
	h := NewASRProviderHolder(NewKeyResolver(&fakeSecrets{}, env, nil), "", "", nil)
	if preference != "" {
		h.SetPreferenceSource(func(context.Context) string { return preference })
	}
	return h
}

func TestASRHolder_ElevenLabsKeyMakesScribeTheDefault(t *testing.T) {
	h := holderWith(EnvKeys{OpenAI: "sk-openai", ElevenLabs: "el-key"}, "")
	ctx := context.Background()

	assert.Equal(t, ASRProviderElevenLabs, h.Provider(ctx))
	p, err := h.Get(ctx)
	require.NoError(t, err)
	_, isScribe := p.(*ai.ScribeClient)
	assert.True(t, isScribe, "auto + ElevenLabs key → Scribe")
	assert.Equal(t, "elevenlabs|scribe_v2", h.EndpointFingerprint())

	rate, provider := h.ASRRate(ctx)
	assert.Equal(t, ASRProviderElevenLabs, provider)
	assert.True(t, rate.Equal(ai.ScribePerMinute()), "quoted at the Scribe rate, got %s", rate)
}

func TestASRHolder_OpenAIKeyAloneStillMeansWhisper(t *testing.T) {
	h := holderWith(EnvKeys{OpenAI: "sk-openai"}, "")
	ctx := context.Background()

	assert.Equal(t, ASRProviderOpenAI, h.Provider(ctx))
	p, err := h.Get(ctx)
	require.NoError(t, err)
	_, isWhisper := p.(*ai.WhisperClient)
	assert.True(t, isWhisper)
	assert.Equal(t, "|"+ai.WhisperModel, h.EndpointFingerprint(),
		"the pre-asr-scribe-b fingerprint, so every cached Whisper chunk is still found")

	rate, _ := h.ASRRate(ctx)
	assert.True(t, rate.Equal(ai.HostedASRPerMinute()))
}

func TestASRHolder_PinnedWhisperIgnoresTheElevenLabsKey(t *testing.T) {
	h := holderWith(EnvKeys{OpenAI: "sk-openai", ElevenLabs: "el-key"}, ASRProviderOpenAI)
	p, err := h.Get(context.Background())
	require.NoError(t, err)
	_, isWhisper := p.(*ai.WhisperClient)
	assert.True(t, isWhisper, "the operator said Whisper")
}

func TestASRHolder_PinnedEngineWithoutItsKeyIsUnconfigured(t *testing.T) {
	// Pinned to Scribe, only an OpenAI key: no silent fallback to the other
	// company's bill.
	h := holderWith(EnvKeys{OpenAI: "sk-openai"}, ASRProviderElevenLabs)
	ctx := context.Background()
	assert.False(t, h.IsConfigured(ctx))
	assert.Equal(t, "", h.Provider(ctx))
	_, err := h.Get(ctx)
	assert.ErrorIs(t, err, ai.ErrWhisperNotConfigured)
}

func TestASRHolder_UnknownPreferenceReadsAsAuto(t *testing.T) {
	h := holderWith(EnvKeys{ElevenLabs: "el-key"}, "something-else")
	assert.Equal(t, ASRProviderElevenLabs, h.Provider(context.Background()))
}

func TestASRHolder_SelfHostedWinsOverEveryKey(t *testing.T) {
	h := NewASRProviderHolder(
		NewKeyResolver(&fakeSecrets{}, EnvKeys{ElevenLabs: "el-key", OpenAI: "sk"}, nil),
		"http://nas:8000/v1/audio/transcriptions", "", nil)
	ctx := context.Background()
	assert.Equal(t, ASRProviderSelfHosted, h.Provider(ctx))
	rate, _ := h.ASRRate(ctx)
	assert.True(t, rate.IsZero())
	p, err := h.Get(ctx)
	require.NoError(t, err)
	_, isWhisper := p.(*ai.WhisperClient)
	assert.True(t, isWhisper, "a self-hosted engine speaks Whisper's format")
}

func TestASRHolder_KeyArrivingAtRuntimeSwitchesEngineWithoutRestart(t *testing.T) {
	secrets := &fakeSecrets{}
	h := NewASRProviderHolder(NewKeyResolver(secrets, EnvKeys{OpenAI: "sk-openai"}, nil), "", "", nil)
	ctx := context.Background()

	before, err := h.Get(ctx)
	require.NoError(t, err)
	_, isWhisper := before.(*ai.WhisperClient)
	require.True(t, isWhisper)

	require.NoError(t, secrets.Store(ctx, SecretNameElevenLabs, "el-saved-later"))

	after, err := h.Get(ctx)
	require.NoError(t, err)
	_, isScribe := after.(*ai.ScribeClient)
	assert.True(t, isScribe, "the next transcribe after saving the key uses Scribe")
	assert.Equal(t, ASRProviderElevenLabs, h.Provider(ctx))
}

func TestASRHolder_PreferenceChangeRebuildsTheClient(t *testing.T) {
	pref := ASRPreferenceAuto
	h := NewASRProviderHolder(NewKeyResolver(&fakeSecrets{}, EnvKeys{OpenAI: "sk", ElevenLabs: "el"}, nil), "", "", nil)
	h.SetPreferenceSource(func(context.Context) string { return pref })
	ctx := context.Background()

	first, err := h.Get(ctx)
	require.NoError(t, err)
	pref = ASRProviderOpenAI
	second, err := h.Get(ctx)
	require.NoError(t, err)
	assert.NotSame(t, first, second)
	_, isWhisper := second.(*ai.WhisperClient)
	assert.True(t, isWhisper)
}

// ─── the estimators read the holder's rate ─────────────────────────────────

func TestGenerationCandidates_QuoteAtTheScribeRateWhenTheHolderSaysSo(t *testing.T) {
	movies := &stubMovieFinder{movies: []models.Movie{movieRow("m1", "Dune", "/m/dune.mkv", 60, "")}}
	svc := NewGenerationCandidateService(movies, nil, nil, &stubPredictor{probeRoute: RouteASR}, false, 0, nil)
	svc.SetASRRateSource(func(context.Context) (decimal.Decimal, string) {
		return ai.ScribePerMinute(), ASRProviderElevenLabs
	})

	res, err := svc.Analyze(context.Background(), nil)
	require.NoError(t, err)
	require.Len(t, res.Candidates, 1)
	assert.Equal(t, ASRProviderElevenLabs, res.Summary.ASRProvider)
	assert.False(t, res.Summary.SelfHostedASR)
	// 60 minutes: Scribe $0.22 + the ASR-leg translation; Whisper would be $0.36 + the same.
	whisper := estimateUSD(RouteASR, 60, ai.HostedASRPerMinute(), ai.DefaultClaudeModel)
	scribe := estimateUSD(RouteASR, 60, ai.ScribePerMinute(), ai.DefaultClaudeModel)
	assert.InDelta(t, scribe.InexactFloat64(), res.Candidates[0].EstimatedUSD, 1e-9)
	assert.Less(t, res.Candidates[0].EstimatedUSD, whisper.InexactFloat64())
}

func TestGenerationCandidates_WithoutARateSourceTheBootBoolStillDecides(t *testing.T) {
	svc := NewGenerationCandidateService(&stubMovieFinder{movies: []models.Movie{movieRow("m1", "Dune", "/m/dune.mkv", 60, "")}}, nil, nil, &stubPredictor{probeRoute: RouteASR}, true, 0, nil)
	res, err := svc.Analyze(context.Background(), nil)
	require.NoError(t, err)
	assert.True(t, res.Summary.SelfHostedASR)
	assert.Equal(t, ASRProviderSelfHosted, res.Summary.ASRProvider)
}

func TestTranscriptionEstimate_PricesTheHolderEngine(t *testing.T) {
	plan := &stubPlanSource{available: true, translate: true}
	est := newTestEstimator(plan, false)
	est.SetASRRateSource(func(context.Context) (decimal.Decimal, string) {
		return ai.ScribePerMinute(), ASRProviderElevenLabs
	})
	e := est.Estimate(context.Background(), sixtyMinuteMovie())
	assert.Equal(t, ASRProviderElevenLabs, e.ASRProvider)
	assert.False(t, e.SelfHostedASR)
	want := estimateUSD(RouteASR, 60, ai.ScribePerMinute(), "claude-sonnet-5")
	assert.True(t, usdOf(t, e).Equal(want), "got %s want %s", usdOf(t, e), want)
}

// ─── the settings page: preference + live engine ───────────────────────────

type fakePreferenceStore struct{ values map[string]string }

func (f *fakePreferenceStore) GetStringWithDefault(_ context.Context, key, def string) string {
	if v, ok := f.values[key]; ok {
		return v
	}
	return def
}
func (f *fakePreferenceStore) SetString(_ context.Context, key, value string) error {
	if f.values == nil {
		f.values = map[string]string{}
	}
	f.values[key] = value
	return nil
}

func TestKeySettings_ListsElevenLabsAndTheEngineChoice(t *testing.T) {
	secrets := &fakeSecrets{}
	resolver := NewKeyResolver(secrets, EnvKeys{ElevenLabs: "el-env"}, nil)
	holder := NewASRProviderHolder(resolver, "", "", nil)
	svc := NewKeySettingsService(resolver, secrets, true)
	store := &fakePreferenceStore{}
	svc.SetASRPreference(store, "auto", holder.Provider)
	holder.SetPreferenceSource(svc.ASRPreference)
	ctx := context.Background()

	out := svc.List(ctx)
	assert.Equal(t, KeyElevenLabs, stateOf(out, KeyElevenLabs).Name)
	assert.Equal(t, KeySourceEnv, stateOf(out, KeyElevenLabs).Source)
	assert.Equal(t, ASRPreferenceAuto, out.ASRProvider)
	assert.Equal(t, ASRProviderElevenLabs, out.ASRActive)

	require.NoError(t, svc.SaveASRPreference(ctx, "OpenAI "))
	out = svc.List(ctx)
	assert.Equal(t, ASRProviderOpenAI, out.ASRProvider)
	assert.Equal(t, "", out.ASRActive, "pinned to Whisper with no OpenAI key: nothing configured")

	assert.ErrorIs(t, svc.SaveASRPreference(ctx, "whisperx"), ErrInvalidASRPreference)
}

func TestKeySettings_EnvDefaultWhenTheTableIsEmpty(t *testing.T) {
	secrets := &fakeSecrets{}
	svc := NewKeySettingsService(NewKeyResolver(secrets, EnvKeys{}, nil), secrets, true)
	svc.SetASRPreference(&fakePreferenceStore{}, "openai", nil)
	assert.Equal(t, ASRProviderOpenAI, svc.ASRPreference(context.Background()))

	bare := NewKeySettingsService(NewKeyResolver(secrets, EnvKeys{}, nil), secrets, true)
	assert.Equal(t, ASRPreferenceAuto, bare.ASRPreference(context.Background()))
	assert.ErrorIs(t, bare.SaveASRPreference(context.Background(), "openai"), ErrASRPreferenceNotWritable)
}
