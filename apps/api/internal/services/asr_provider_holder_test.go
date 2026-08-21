package services

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vido/api/internal/ai"
)

// Mirrors holderWithKey in claude_provider_holder_test.go — the ASR half of the
// same M1.5 promise (sub-5-2 AC #1).
func asrHolderWithKey(t *testing.T, key string, opts ...ai.WhisperOption) *ASRProviderHolder {
	t.Helper()
	return NewASRProviderHolder(
		NewKeyResolver(&fakeSecrets{}, EnvKeys{OpenAI: key}, nil), "", "", nil, opts...)
}

// ─── AC #1: fingerprint-cached rebuild (Rule 14) ───────────────────────────

func TestASRProviderHolder_SameFingerprintReturnsSameInstance(t *testing.T) {
	h := asrHolderWithKey(t, "sk-one")

	first, err := h.Get(context.Background())
	require.NoError(t, err)
	second, err := h.Get(context.Background())
	require.NoError(t, err)

	assert.Same(t, first, second)
}

func TestASRProviderHolder_ChangedKeyRebuilds(t *testing.T) {
	secrets := &fakeSecrets{}
	h := NewASRProviderHolder(
		NewKeyResolver(secrets, EnvKeys{OpenAI: "sk-env"}, nil), "", "", nil)

	before, err := h.Get(context.Background())
	require.NoError(t, err)

	require.NoError(t, secrets.Store(context.Background(), SecretNameOpenAI, "sk-user-typed"))

	after, err := h.Get(context.Background())
	require.NoError(t, err)

	assert.NotSame(t, before, after, "a runtime key change must reach a NEW client")
}

func TestASRProviderHolder_UnconfiguredReturnsErrWhisperNotConfigured(t *testing.T) {
	h := asrHolderWithKey(t, "")

	provider, err := h.Get(context.Background())

	require.ErrorIs(t, err, ai.ErrWhisperNotConfigured,
		"reuse the shipped sentinel — consumers already classify on it")
	assert.Nil(t, provider)
}

// TestASRProviderHolder_RecoversWhenAKeyArrivesLater — the headline promise:
// a container booted with no ASR key must start transcribing once one is saved,
// with no restart.
func TestASRProviderHolder_RecoversWhenAKeyArrivesLater(t *testing.T) {
	secrets := &fakeSecrets{}
	h := NewASRProviderHolder(NewKeyResolver(secrets, EnvKeys{}, nil), "", "", nil)

	_, err := h.Get(context.Background())
	require.ErrorIs(t, err, ai.ErrWhisperNotConfigured)
	require.False(t, h.IsConfigured(context.Background()))

	require.NoError(t, secrets.Store(context.Background(), SecretNameOpenAI, "sk-late"))

	provider, err := h.Get(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, provider)
	assert.True(t, h.IsConfigured(context.Background()))
}

// TestASRProviderHolder_GovernorSurvivesRebuild — a new client must never mean a
// new budget pool. The 9R-11 Governor caps concurrency, QPS and spend across ASR
// *and* LLM; rebuilding with a fresh one would silently reset the run budget
// every time a key is edited.
func TestASRProviderHolder_GovernorSurvivesRebuild(t *testing.T) {
	governor := ai.NewGovernor(2, 5, 2)
	secrets := &fakeSecrets{}
	h := NewASRProviderHolder(
		NewKeyResolver(secrets, EnvKeys{OpenAI: "sk-env"}, nil), "", "",
		nil, ai.WithWhisperGovernor(governor))

	before, err := h.Get(context.Background())
	require.NoError(t, err)
	require.NoError(t, secrets.Store(context.Background(), SecretNameOpenAI, "sk-new"))
	after, err := h.Get(context.Background())
	require.NoError(t, err)

	require.NotSame(t, before, after, "precondition: it really did rebuild")
	beforeWhisper, ok := before.(*ai.WhisperClient)
	require.True(t, ok)
	afterWhisper, ok := after.(*ai.WhisperClient)
	require.True(t, ok)
	assert.Same(t, governor, afterWhisper.Governor(),
		"the SAME governor instance must be carried into the rebuilt client")
	assert.Same(t, beforeWhisper.Governor(), afterWhisper.Governor())
}

// NFR-S1 (sub-5-2 AC #7(b), added at CR): the rebuild log line must never carry
// the key or any prefix of it. Hand-inspection said so; this pins it — the
// gemini_test.go MissingUsageMetadata log-capture pattern.
func TestASRProviderHolder_RebuildLogNeverContainsTheKey(t *testing.T) {
	const secret = "sk-THISMUSTNEVERBELOGGED-0123456789"

	var logs bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logs, &slog.HandlerOptions{Level: slog.LevelDebug}))

	h := NewASRProviderHolder(
		NewKeyResolver(&fakeSecrets{}, EnvKeys{OpenAI: secret}, nil), "", "", logger)

	_, err := h.Get(context.Background())
	require.NoError(t, err)

	out := logs.String()
	assert.Contains(t, out, "asr provider (re)built", "precondition: the rebuild line was emitted")
	assert.Contains(t, out, "key_source", "the SOURCE is loggable — that is the NFR-S1-safe shape")
	assert.NotContains(t, out, secret, "the key itself must never be logged")
	assert.NotContains(t, out, secret[:6], "…nor any prefix of it (NFR-S1)")
}

// baseURL and model reach the built client via the constructor params (option
// plumbing). NOTE deliberately NOT named a fingerprint test: baseURL/model are
// constructor-fixed per holder, so their fingerprint components can never
// change within one holder's lifetime — they are future-proofing for when the
// endpoint becomes settings-configurable, and only the KEY dimension of the
// fingerprint is falsifiable today (covered by the rebuild tests above).
func TestASRProviderHolder_BaseURLAndModelReachTheClient(t *testing.T) {
	resolver := NewKeyResolver(&fakeSecrets{}, EnvKeys{OpenAI: "sk"}, nil)

	hosted := NewASRProviderHolder(resolver, "", "", nil)
	selfHosted := NewASRProviderHolder(resolver, "http://nas:8000/v1/audio/transcriptions", "Systran/faster-whisper-small", nil)

	a, err := hosted.Get(context.Background())
	require.NoError(t, err)
	b, err := selfHosted.Get(context.Background())
	require.NoError(t, err)

	assert.NotSame(t, a, b)
	assert.True(t, a.(*ai.WhisperClient).BaseURL() == ai.WhisperAPIURL)
	assert.Equal(t, "http://nas:8000/v1/audio/transcriptions", b.(*ai.WhisperClient).BaseURL())
}

// ─── AC #3: self-hosted ASR needs no key ───────────────────────────────────

// A self-hosted OpenAI-compatible engine (Speaches, WhisperLive, Subgen) needs
// no API key. Before sub-5-2 the boot gate was cfg.HasOpenAIKey(), so such a
// deployment had to invent a dummy OPENAI_API_KEY or get no ASR at all — while
// sub-5-1 was already metering it at $0 and the docs advertised it as supported.
func TestASRProviderHolder_SelfHostedIsConfiguredWithoutAKey(t *testing.T) {
	h := NewASRProviderHolder(
		NewKeyResolver(&fakeSecrets{}, EnvKeys{}, nil),
		"http://nas:8000/v1/audio/transcriptions", "", nil)

	assert.True(t, h.IsConfigured(context.Background()),
		"a self-hosted endpoint IS a configured ASR — no key required")

	provider, err := h.Get(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, provider)
}

// The hosted API still costs money and still needs a key — the AC #3 relaxation
// must not leak into the paid path.
func TestASRProviderHolder_HostedStillRequiresAKey(t *testing.T) {
	h := NewASRProviderHolder(NewKeyResolver(&fakeSecrets{}, EnvKeys{}, nil), "", "", nil)

	assert.False(t, h.IsConfigured(context.Background()))
	_, err := h.Get(context.Background())
	require.ErrorIs(t, err, ai.ErrWhisperNotConfigured)
}

// The trap sub-5-1 CR M1 pinned for the metering side, re-pinned here for the
// availability side: ASR_BASE_URL explicitly set to the OFFICIAL endpoint is
// hosted, not self-hosted — a second detector that answered "non-empty ⇒
// self-hosted" would hand out a keyless client that 401s on every call.
func TestASRProviderHolder_ExplicitOfficialURLIsNotSelfHosted(t *testing.T) {
	h := NewASRProviderHolder(
		NewKeyResolver(&fakeSecrets{}, EnvKeys{}, nil), ai.WhisperAPIURL, "", nil)

	assert.False(t, h.IsConfigured(context.Background()))
	_, err := h.Get(context.Background())
	require.ErrorIs(t, err, ai.ErrWhisperNotConfigured)
}

// ─── AC #1: the holder IS the ASRProvider (per-call delegation) ────────────

func TestASRProviderHolder_TranscribeDeclinesWhenUnconfigured(t *testing.T) {
	h := asrHolderWithKey(t, "")

	_, err := h.Transcribe(context.Background(), "/tmp/audio.wav")
	require.ErrorIs(t, err, ai.ErrWhisperNotConfigured)

	_, err = h.TranscribeWithLanguage(context.Background(), "/tmp/audio.wav", "en")
	require.ErrorIs(t, err, ai.ErrWhisperNotConfigured)
}

func TestASRProviderHolder_IsConfiguredMirrorsResolver(t *testing.T) {
	assert.True(t, asrHolderWithKey(t, "sk").IsConfigured(context.Background()))
	assert.False(t, asrHolderWithKey(t, "").IsConfigured(context.Background()))
}

// ─── AC #2: TranscriptionService availability is now a per-call question ───

// Removing main.go's `if cfg.HasOpenAIKey()` guard means the service ALWAYS gets
// a non-nil asr. Without a key-aware probe, `s.asr != nil` would report every
// keyless install as available — the FR12 503 gate, the worker-pool sweep gate
// and the legacy batch gate would all lie.
func TestTranscriptionService_UnavailableWhenTheHolderHasNoKey(t *testing.T) {
	extractor := &AudioExtractorService{available: true, semaphore: make(chan struct{}, 1)}
	svc := NewTranscriptionService(extractor, asrHolderWithKey(t, ""), nil, nil)

	assert.False(t, svc.IsAvailable(),
		"a keyless holder must not report the transcription service as available")
}

func TestTranscriptionService_AvailableOnceAKeyResolves(t *testing.T) {
	extractor := &AudioExtractorService{available: true, semaphore: make(chan struct{}, 1)}
	svc := NewTranscriptionService(extractor, asrHolderWithKey(t, "sk-present"), nil, nil)

	assert.True(t, svc.IsAvailable())
}

// The hot-reload itself, at the service seam: nothing is reconstructed between
// the two assertions.
func TestTranscriptionService_AvailabilityFlipsWithoutRebuilding(t *testing.T) {
	secrets := &fakeSecrets{}
	holder := NewASRProviderHolder(NewKeyResolver(secrets, EnvKeys{}, nil), "", "", nil)
	extractor := &AudioExtractorService{available: true, semaphore: make(chan struct{}, 1)}
	svc := NewTranscriptionService(extractor, holder, nil, nil)

	require.False(t, svc.IsAvailable(), "precondition: keyless boot")

	require.NoError(t, secrets.Store(context.Background(), SecretNameOpenAI, "sk-saved-at-runtime"))

	assert.True(t, svc.IsAvailable(),
		"saving a key must un-gate transcription with no restart and no re-wiring")
}

// A provider WITHOUT the probe (a directly-constructed WhisperClient, or any
// test fake) must keep the old behaviour: present means available. This is the
// compatibility clause that leaves every pre-sub-5-2 test untouched.
func TestTranscriptionService_PlainProviderWithoutProbeStaysAvailable(t *testing.T) {
	extractor := &AudioExtractorService{available: true, semaphore: make(chan struct{}, 1)}
	svc := NewTranscriptionService(extractor, ai.NewWhisperClient("sk-direct"), nil, nil)

	assert.True(t, svc.IsAvailable())
}

// ─── 9R-5: the detailed (hallucination-filter) seam must survive the holder ──

// CR M2: the pipeline holds the HOLDER, not the client. If the holder does not
// carry ai.DetailedTranscriber through, transcribeAudio's type assertion fails
// and the whole-file empty-transcript guard silently never engages — no error,
// no log, just a feature that quietly does nothing.
func TestASRProviderHolder_SatisfiesDetailedTranscriber(t *testing.T) {
	var _ ai.DetailedTranscriber = (*ASRProviderHolder)(nil)
}

func TestASRProviderHolder_TranscribeDetailedForwardsToTheClient(t *testing.T) {
	body := `{"language":"english","duration":109.0,"segments":[
	  {"id":0,"start":10.0,"end":12.0,"text":"Real dialogue","avg_logprob":-0.25,"compression_ratio":1.4,"no_speech_prob":0.01},
	  {"id":1,"start":13.0,"end":15.0,"text":"More dialogue","avg_logprob":-0.3,"compression_ratio":1.3,"no_speech_prob":0.02},
	  {"id":2,"start":100.0,"end":103.0,"text":"Thanks for watching!","avg_logprob":-0.9,"compression_ratio":1.5,"no_speech_prob":0.55},
	  {"id":3,"start":103.0,"end":106.0,"text":"Like and subscribe.","avg_logprob":-1.2,"compression_ratio":1.6,"no_speech_prob":0.72},
	  {"id":4,"start":106.0,"end":109.0,"text":"See you next time.","avg_logprob":-0.95,"compression_ratio":1.4,"no_speech_prob":0.61}
	]}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(body))
	}))
	defer server.Close()

	h := NewASRProviderHolder(
		NewKeyResolver(&fakeSecrets{}, EnvKeys{OpenAI: "sk-test"}, nil),
		server.URL, "", nil)

	audio := filepath.Join(t.TempDir(), "audio.wav")
	require.NoError(t, os.WriteFile(audio, []byte("RIFFfakewavdata"), 0644))

	detail, err := h.TranscribeDetailed(context.Background(), audio, "en")
	require.NoError(t, err)

	assert.True(t, detail.Filtered, "the filter verdict must reach the caller through the holder")
	assert.Equal(t, 5, detail.SegmentsIn)
	assert.Equal(t, 2, detail.SegmentsKept)
	assert.NotContains(t, detail.SRT, "subscribe")
	assert.Contains(t, detail.Unfiltered, "subscribe",
		"the recovery text the whole-file guard depends on must survive the hop")
}

func TestASRProviderHolder_TranscribeDetailedPropagatesUnconfigured(t *testing.T) {
	h := NewASRProviderHolder(NewKeyResolver(&fakeSecrets{}, EnvKeys{}, nil), "", "", nil)
	_, err := h.TranscribeDetailed(context.Background(), "irrelevant.wav", "en")
	require.ErrorIs(t, err, ai.ErrWhisperNotConfigured)
}
