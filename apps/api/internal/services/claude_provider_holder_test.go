package services

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vido/api/internal/ai"
)

func holderWithKey(t *testing.T, key string, opts ...ai.ClaudeProviderOption) *ClaudeProviderHolder {
	t.Helper()
	return NewClaudeProviderHolder(
		NewKeyResolver(&fakeSecrets{}, EnvKeys{Claude: key}, nil), "", nil, opts...)
}

// ─── AC #2: fingerprint-cached rebuild (Rule 14) ───────────────────────────

func TestClaudeProviderHolder_SameFingerprintReturnsSameInstance(t *testing.T) {
	h := holderWithKey(t, "sk-one")

	first, err := h.Get(context.Background())
	require.NoError(t, err)
	second, err := h.Get(context.Background())
	require.NoError(t, err)

	// Pointer equality, not merely equal values: Rule 14 forbids building an
	// expensive client per request.
	assert.Same(t, first, second)
}

func TestClaudeProviderHolder_ChangedKeyRebuilds(t *testing.T) {
	secrets := &fakeSecrets{}
	h := NewClaudeProviderHolder(
		NewKeyResolver(secrets, EnvKeys{Claude: "sk-env"}, nil), "", nil)

	before, err := h.Get(context.Background())
	require.NoError(t, err)

	// Simulate the settings page storing a key at runtime — Break 2's whole point.
	require.NoError(t, secrets.Store(context.Background(), SecretNameClaude, "sk-user-typed"))

	after, err := h.Get(context.Background())
	require.NoError(t, err)

	assert.NotSame(t, before, after, "a runtime key change must reach a NEW client")
}

func TestClaudeProviderHolder_UnconfiguredReturnsErrAINotConfigured(t *testing.T) {
	h := holderWithKey(t, "")

	provider, err := h.Get(context.Background())

	require.Error(t, err)
	assert.True(t, errors.Is(err, ai.ErrAINotConfigured),
		"consumers switch on this sentinel to degrade fail-soft (NFR-R1)")
	assert.Nil(t, provider)
}

// TestClaudeProviderHolder_RecoversWhenAKeyArrivesLater — the exact M1.5 promise:
// a container booted with no key must start translating once one is saved, with
// no restart.
func TestClaudeProviderHolder_RecoversWhenAKeyArrivesLater(t *testing.T) {
	secrets := &fakeSecrets{}
	h := NewClaudeProviderHolder(NewKeyResolver(secrets, EnvKeys{}, nil), "", nil)

	_, err := h.Get(context.Background())
	require.ErrorIs(t, err, ai.ErrAINotConfigured)

	require.NoError(t, secrets.Store(context.Background(), SecretNameClaude, "sk-late"))

	provider, err := h.Get(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, provider)
	assert.True(t, h.IsConfigured(context.Background()))
}

// TestClaudeProviderHolder_GovernorSurvivesRebuild — a new client must never mean
// a new budget pool. The Governor (9R-11) caps concurrency, QPS and spend across
// ASR *and* LLM; rebuilding with a fresh one would silently reset the run budget
// every time a key is edited.
func TestClaudeProviderHolder_GovernorSurvivesRebuild(t *testing.T) {
	governor := ai.NewGovernor(2, 5, 2)
	secrets := &fakeSecrets{}
	h := NewClaudeProviderHolder(
		NewKeyResolver(secrets, EnvKeys{Claude: "sk-env"}, nil), "",
		nil, ai.WithClaudeGovernor(governor))

	before, err := h.Get(context.Background())
	require.NoError(t, err)
	require.NoError(t, secrets.Store(context.Background(), SecretNameClaude, "sk-new"))
	after, err := h.Get(context.Background())
	require.NoError(t, err)

	require.NotSame(t, before, after, "precondition: it really did rebuild")
	// Get honours the stamped contract and returns ai.TextCompleter; the concrete
	// type is asserted here only to read the governor back out.
	beforeClaude, ok := before.(*ai.ClaudeProvider)
	require.True(t, ok)
	afterClaude, ok := after.(*ai.ClaudeProvider)
	require.True(t, ok)
	assert.Same(t, governor, afterClaude.Governor(),
		"the SAME governor instance must be carried into the rebuilt client")
	assert.Same(t, beforeClaude.Governor(), afterClaude.Governor())
}

// ─── AC #2: the holder IS the completer (services construct unconditionally) ──

// TestClaudeProviderHolder_ImplementsCachingCompleter is the guard against a
// silent capability loss. TranslationService type-asserts ai.CachingCompleter
// (translation_service.go:323) and, when the assertion fails, logs one line and
// carries on WITHOUT prompt caching or usage reporting — which would quietly
// void sub-1-5b's entire caching design. A holder that only implemented
// TextCompleter would compile, pass every other test, and cost real money.
func TestClaudeProviderHolder_ImplementsCachingCompleter(t *testing.T) {
	var completer ai.TextCompleter = holderWithKey(t, "sk-one")

	_, ok := completer.(ai.CachingCompleter)

	assert.True(t, ok, "the holder must delegate CachingCompleter, not just TextCompleter")
}

func TestClaudeProviderHolder_CompleteTextDeclinesWhenUnconfigured(t *testing.T) {
	h := holderWithKey(t, "")

	_, err := h.CompleteText(context.Background(), "sys", "user", 10)

	// The services are now constructed unconditionally, so "no key" has to be a
	// per-call decline rather than a nil service (Break 2's second half).
	require.ErrorIs(t, err, ai.ErrAINotConfigured)
}

func TestClaudeProviderHolder_CompleteTextWithUsageDeclinesWhenUnconfigured(t *testing.T) {
	h := holderWithKey(t, "")

	_, err := h.CompleteTextWithUsage(context.Background(), ai.CompletionRequest{UserPrompt: "hi"})

	require.ErrorIs(t, err, ai.ErrAINotConfigured)
}

func TestClaudeProviderHolder_IsConfiguredMirrorsResolver(t *testing.T) {
	assert.True(t, holderWithKey(t, "sk").IsConfigured(context.Background()))
	assert.False(t, holderWithKey(t, "").IsConfigured(context.Background()))
}

// ─── The regression the pre-ship self-review caught ────────────────────────

// Removing main.go's `if cfg.HasClaudeKey()` guard (AC #2) means these services
// now ALWAYS exist. Both of their consumers gate on `!= nil && IsConfigured()`
// (subtitle/engine.go:191, transcription_service.go:416), and IsConfigured used
// to be `provider != nil` — which is now always true. Without a key-aware probe
// a keyless install would report configured, attempt AI terminology correction
// and ASR translation, and fail with ErrAINotConfigured where it previously
// skipped cleanly. These pin the probe.
func TestServices_ReportUnconfiguredWhenTheHolderHasNoKey(t *testing.T) {
	holder := holderWithKey(t, "")

	assert.False(t, NewTranslationService(holder, nil).IsConfigured(),
		"a keyless holder must not report the translation service as configured")
	assert.False(t, NewTerminologyCorrectionService(holder).IsConfigured(),
		"…nor the terminology service")
}

func TestServices_ReportConfiguredOnceAKeyResolves(t *testing.T) {
	holder := holderWithKey(t, "sk-present")

	assert.True(t, NewTranslationService(holder, nil).IsConfigured())
	assert.True(t, NewTerminologyCorrectionService(holder).IsConfigured())
}

// A provider WITHOUT the probe (Gemini, or a directly-constructed Claude client)
// must keep the old behaviour: present means configured.
func TestServices_PlainProviderWithoutProbeStaysConfigured(t *testing.T) {
	plain := ai.NewClaudeProvider("sk-direct")

	assert.True(t, NewTranslationService(plain, nil).IsConfigured())
	assert.True(t, NewTerminologyCorrectionService(plain).IsConfigured())
}

// ─── sub-6-5: the model in force is a holder property, never "" ────────────

func TestClaudeProviderHolder_EffectiveModel_DefaultWhenNoOverride(t *testing.T) {
	h := holderWithKey(t, "sk-one")
	assert.Equal(t, ai.DefaultClaudeModel, h.EffectiveModel())
	assert.NotEmpty(t, h.EffectiveModel(), "a default-model run must never record an empty model id")
}

func TestClaudeProviderHolder_EffectiveModel_ReportsOverride(t *testing.T) {
	h := NewClaudeProviderHolder(
		NewKeyResolver(&fakeSecrets{}, EnvKeys{Claude: "sk-one"}, nil), "claude-sonnet-5", nil)
	assert.Equal(t, "claude-sonnet-5", h.EffectiveModel())

	// And the client it builds sends that same id — one truth, two readers.
	p, err := h.Get(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "claude-sonnet-5", p.(*ai.ClaudeProvider).Model())
}

// ─── sub-6-6: TestKey judges transport, not reply text ──────────────────────

func TestClaudeProviderHolder_TestKey_EmptyReplyIsValid(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"m","type":"message","role":"assistant","model":"claude-sonnet-5","content":[],"stop_reason":"max_tokens","usage":{"input_tokens":1,"output_tokens":0}}`))
	}))
	defer srv.Close()

	h := holderWithKey(t, "sk-one", ai.WithClaudeBaseURL(srv.URL))
	require.NoError(t, h.TestKey(context.Background(), ""), "resolved key: an empty 2xx reply is a pass")
	require.NoError(t, h.TestKey(context.Background(), "sk-candidate"), "candidate key: same rule")
}

func TestClaudeProviderHolder_TestKey_UnauthorizedStillFails(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(401)
		_, _ = w.Write([]byte(`{"type":"error","error":{"type":"authentication_error","message":"bad"}}`))
	}))
	defer srv.Close()
	h := holderWithKey(t, "sk-one", ai.WithClaudeBaseURL(srv.URL))
	assert.ErrorIs(t, h.TestKey(context.Background(), "sk-bad"), ai.ErrAIUnauthorized)
}

// ─── sub-6-8a AC #5: per-run model selection ───────────────────────────────

func TestClaudeProviderHolder_GetForBuildsOneClientPerModelAndSharesTheGovernor(t *testing.T) {
	governor := ai.NewGovernor(2, 2, 2)
	h := holderWithKey(t, "sk-one", ai.WithClaudeGovernor(governor))
	ctx := context.Background()

	sonnet, err := h.GetFor(ctx, "claude-sonnet-5")
	require.NoError(t, err)
	haiku, err := h.GetFor(ctx, "claude-haiku-4-5")
	require.NoError(t, err)

	assert.NotSame(t, sonnet, haiku, "two models are two clients — one would send the wrong model id")
	assert.Equal(t, "claude-sonnet-5", sonnet.(*ai.ClaudeProvider).Model())
	assert.Equal(t, "claude-haiku-4-5", haiku.(*ai.ClaudeProvider).Model())

	// The load-bearing half: a new client must NEVER mean a new budget pool,
	// or switching model mid-batch would silently reset the run ceiling.
	assert.Same(t, governor, sonnet.(*ai.ClaudeProvider).Governor())
	assert.Same(t, governor, haiku.(*ai.ClaudeProvider).Governor())

	again, err := h.GetFor(ctx, "claude-sonnet-5")
	require.NoError(t, err)
	assert.Same(t, sonnet, again, "a model already built is cached (Rule 14)")
}

func TestClaudeProviderHolder_GetReadsTheModelOffTheContext(t *testing.T) {
	h := holderWithKey(t, "sk-one")

	byDefault, err := h.Get(context.Background())
	require.NoError(t, err)
	assert.Equal(t, ai.DefaultClaudeModel, byDefault.(*ai.ClaudeProvider).Model())

	pinned, err := h.Get(ai.WithModelID(context.Background(), "claude-haiku-4-5"))
	require.NoError(t, err)
	assert.Equal(t, "claude-haiku-4-5", pinned.(*ai.ClaudeProvider).Model(),
		"the per-run choice must reach the client without any caller in between passing a parameter")
}

func TestClaudeProviderHolder_GetForRejectsAnUnsupportedModel(t *testing.T) {
	h := holderWithKey(t, "sk-one")

	_, err := h.GetFor(context.Background(), "gpt-4o")

	require.Error(t, err)
	assert.ErrorIs(t, err, ai.ErrAIModelNotFound,
		"a typo must fail here, not at the API after the user consented to a charge")
}

func TestClaudeProviderHolder_ClientCacheIsBounded(t *testing.T) {
	h := holderWithKey(t, "sk-one")
	ctx := context.Background()

	// Five distinct models through a four-slot cache: the least-recently-used
	// one is evicted. Without the bound, a caller looping over model ids would
	// grow this map forever (Rule 14).
	models := []string{"claude-sonnet-5", "claude-haiku-4-5", "claude-opus-4-8", "claude-sonnet-4-6", "gemini-2.5-flash"}
	built := map[string]ai.TextCompleter{}
	for _, m := range models {
		p, err := h.GetFor(ctx, m)
		require.NoError(t, err, m)
		built[m] = p
	}

	h.mu.Lock()
	cached := len(h.clients)
	h.mu.Unlock()
	assert.Equal(t, maxCachedClients, cached, "the cache must not grow past its bound")

	// The oldest is gone (rebuilt = a different pointer); the newest is kept.
	evicted, err := h.GetFor(ctx, models[0])
	require.NoError(t, err)
	assert.NotSame(t, built[models[0]], evicted, "the least-recently-used client is the one evicted")

	kept, err := h.GetFor(ctx, models[len(models)-1])
	require.NoError(t, err)
	assert.Same(t, built[models[len(models)-1]], kept)
}

func TestClaudeProviderHolder_RecentUseKeepsAClientAlive(t *testing.T) {
	h := holderWithKey(t, "sk-one")
	ctx := context.Background()

	first, err := h.GetFor(ctx, "claude-sonnet-5")
	require.NoError(t, err)
	for _, m := range []string{"claude-haiku-4-5", "claude-opus-4-8"} {
		_, err := h.GetFor(ctx, m)
		require.NoError(t, err)
	}
	// Touch the first again, then push two more through: LRU order must have
	// moved it out of the eviction seat.
	touched, err := h.GetFor(ctx, "claude-sonnet-5")
	require.NoError(t, err)
	assert.Same(t, first, touched)

	for _, m := range []string{"claude-sonnet-4-6", "gemini-2.5-flash"} {
		_, err := h.GetFor(ctx, m)
		require.NoError(t, err)
	}
	still, err := h.GetFor(ctx, "claude-sonnet-5")
	require.NoError(t, err)
	assert.Same(t, first, still, "the model actually in use must survive an eviction round")
}
