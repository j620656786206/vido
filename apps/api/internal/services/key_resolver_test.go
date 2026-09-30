package services

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeSecrets is a minimal SecretsServiceInterface stand-in. `err` makes every
// read fail, which is how the fail-soft fallthrough is exercised.
type fakeSecrets struct {
	values map[string]string
	err    error
	reads  []string
}

func (f *fakeSecrets) Store(_ context.Context, name, value string) error {
	if f.err != nil {
		return f.err
	}
	if f.values == nil {
		f.values = map[string]string{}
	}
	f.values[name] = value
	return nil
}

func (f *fakeSecrets) Retrieve(_ context.Context, name string) (string, error) {
	f.reads = append(f.reads, name)
	if f.err != nil {
		return "", f.err
	}
	v, ok := f.values[name]
	if !ok {
		return "", errors.New("not found")
	}
	return v, nil
}

func (f *fakeSecrets) Delete(_ context.Context, name string) error {
	if f.err != nil {
		return f.err
	}
	delete(f.values, name)
	return nil
}

func (f *fakeSecrets) Exists(_ context.Context, name string) (bool, error) {
	if f.err != nil {
		return false, f.err
	}
	_, ok := f.values[name]
	return ok, nil
}

func (f *fakeSecrets) List(_ context.Context) ([]string, error) {
	if f.err != nil {
		return nil, f.err
	}
	out := make([]string, 0, len(f.values))
	for k := range f.values {
		out = append(out, k)
	}
	return out, nil
}

// ─── AC #1: secret > env, one source of truth ──────────────────────────────

func TestKeyResolver_SecretWinsOverEnv(t *testing.T) {
	secrets := &fakeSecrets{values: map[string]string{SecretNameClaude: "sk-from-secret"}}
	r := NewKeyResolver(secrets, EnvKeys{Claude: "sk-from-env"}, nil)

	value, source, err := r.Get(context.Background(), KeyClaude)

	require.NoError(t, err)
	// The UI is the newer, more specific intent. If env won, an in-app edit
	// would silently do nothing on every env-configured deployment — the exact
	// silent-no-op this story exists to remove.
	assert.Equal(t, "sk-from-secret", value)
	assert.Equal(t, KeySourceSecret, source)
}

func TestKeyResolver_FallsBackToEnv(t *testing.T) {
	r := NewKeyResolver(&fakeSecrets{}, EnvKeys{Claude: "sk-from-env"}, nil)

	value, source, err := r.Get(context.Background(), KeyClaude)

	require.NoError(t, err)
	assert.Equal(t, "sk-from-env", value)
	assert.Equal(t, KeySourceEnv, source)
}

func TestKeyResolver_NeitherSourceIsNotAnError(t *testing.T) {
	r := NewKeyResolver(&fakeSecrets{}, EnvKeys{}, nil)

	value, source, err := r.Get(context.Background(), KeyClaude)

	require.NoError(t, err, "an unconfigured key is a STATE, not a failure")
	assert.Empty(t, value)
	assert.Equal(t, KeySourceNone, source)
}

// TestKeyResolver_SecretsFailureFallsThroughToEnv — the fail-soft guarantee.
// A decryption failure (the qBittorrent password error seen live on the NAS)
// must not take down an otherwise-working env-configured deployment.
func TestKeyResolver_SecretsFailureFallsThroughToEnv(t *testing.T) {
	secrets := &fakeSecrets{err: errors.New("decryption failed: authentication error")}
	r := NewKeyResolver(secrets, EnvKeys{Claude: "sk-from-env"}, nil)

	value, source, err := r.Get(context.Background(), KeyClaude)

	require.NoError(t, err, "a secrets-service failure degrades, it does not propagate")
	assert.Equal(t, "sk-from-env", value)
	assert.Equal(t, KeySourceEnv, source)
}

func TestKeyResolver_SecretsFailureWithNoEnvIsStillNotAnError(t *testing.T) {
	secrets := &fakeSecrets{err: errors.New("boom")}
	r := NewKeyResolver(secrets, EnvKeys{}, nil)

	_, source, err := r.Get(context.Background(), KeyClaude)

	require.NoError(t, err)
	assert.Equal(t, KeySourceNone, source)
}

func TestKeyResolver_HasMirrorsGet(t *testing.T) {
	configured := NewKeyResolver(&fakeSecrets{}, EnvKeys{Claude: "sk"}, nil)
	unconfigured := NewKeyResolver(&fakeSecrets{}, EnvKeys{}, nil)

	assert.True(t, configured.Has(context.Background(), KeyClaude))
	assert.False(t, unconfigured.Has(context.Background(), KeyClaude))
}

func TestKeyResolver_ResolvesEachKeyIndependently(t *testing.T) {
	secrets := &fakeSecrets{values: map[string]string{SecretNameTMDb: "tmdb-secret"}}
	r := NewKeyResolver(secrets, EnvKeys{Claude: "claude-env", OpenAI: ""}, nil)

	claude, claudeSrc, _ := r.Get(context.Background(), KeyClaude)
	tmdb, tmdbSrc, _ := r.Get(context.Background(), KeyTMDb)
	openai, openaiSrc, _ := r.Get(context.Background(), KeyOpenAI)

	assert.Equal(t, "claude-env", claude)
	assert.Equal(t, KeySourceEnv, claudeSrc)
	assert.Equal(t, "tmdb-secret", tmdb)
	assert.Equal(t, KeySourceSecret, tmdbSrc)
	assert.Empty(t, openai)
	assert.Equal(t, KeySourceNone, openaiSrc)
}

func TestKeyResolver_UnknownKeyNameIsAnError(t *testing.T) {
	r := NewKeyResolver(&fakeSecrets{}, EnvKeys{}, nil)

	_, _, err := r.Get(context.Background(), KeyName("gemini"))

	// Distinct from "not configured": a typo'd key name is a programming error
	// and must not masquerade as an unconfigured key.
	require.Error(t, err)
	assert.False(t, r.Has(context.Background(), KeyName("gemini")))
}

// TestKeyResolver_BlankSecretIsTreatedAsAbsent — a stored empty string must not
// shadow a working env-var, or "clear my key" would break the deployment
// instead of reverting it (AC #3's explicit-"" delete path relies on this).
func TestKeyResolver_BlankSecretIsTreatedAsAbsent(t *testing.T) {
	secrets := &fakeSecrets{values: map[string]string{SecretNameClaude: "   "}}
	r := NewKeyResolver(secrets, EnvKeys{Claude: "sk-from-env"}, nil)

	value, source, err := r.Get(context.Background(), KeyClaude)

	require.NoError(t, err)
	assert.Equal(t, "sk-from-env", value)
	assert.Equal(t, KeySourceEnv, source)
}

func TestKeyResolver_NilSecretsServiceIsEnvOnly(t *testing.T) {
	// main.go can construct the resolver before/without a secrets service
	// (no ENCRYPTION_KEY). It must degrade to env, not panic.
	r := NewKeyResolver(nil, EnvKeys{Claude: "sk-from-env"}, nil)

	value, source, err := r.Get(context.Background(), KeyClaude)

	require.NoError(t, err)
	assert.Equal(t, "sk-from-env", value)
	assert.Equal(t, KeySourceEnv, source)
}

// ─── sub-7-7a: the bundled tier ─────────────────────────────────────────────
//
// A key compiled into the release binary is the LAST resort: it exists so a
// fresh install works with no TMDb account, and it must never shadow anything
// the user set — in the UI or in the environment.

func TestKeyResolver_BundledIsUsedWhenNothingElseIsSet(t *testing.T) {
	r := NewKeyResolver(&fakeSecrets{}, EnvKeys{}, nil,
		WithBundledKeys(BundledKeys{TMDb: "bundled-tmdb"}))

	value, source, err := r.Get(context.Background(), KeyTMDb)

	require.NoError(t, err)
	assert.Equal(t, "bundled-tmdb", value)
	assert.Equal(t, KeySourceBundled, source)
	assert.True(t, r.Has(context.Background(), KeyTMDb))
}

func TestKeyResolver_EnvBeatsBundled(t *testing.T) {
	r := NewKeyResolver(&fakeSecrets{}, EnvKeys{TMDb: "tmdb-from-env"}, nil,
		WithBundledKeys(BundledKeys{TMDb: "bundled-tmdb"}))

	value, source, err := r.Get(context.Background(), KeyTMDb)

	require.NoError(t, err)
	assert.Equal(t, "tmdb-from-env", value)
	assert.Equal(t, KeySourceEnv, source)
}

func TestKeyResolver_SecretBeatsBundled(t *testing.T) {
	secrets := &fakeSecrets{values: map[string]string{SecretNameTMDb: "tmdb-from-ui"}}
	r := NewKeyResolver(secrets, EnvKeys{}, nil,
		WithBundledKeys(BundledKeys{TMDb: "bundled-tmdb"}))

	value, source, err := r.Get(context.Background(), KeyTMDb)

	require.NoError(t, err)
	assert.Equal(t, "tmdb-from-ui", value)
	assert.Equal(t, KeySourceSecret, source)
}

// A source build has no bundled key: the answer must stay "none", exactly as
// before sub-7-7a, so local development keeps asking for a key.
func TestKeyResolver_NoBundledKeyIsStillNone(t *testing.T) {
	r := NewKeyResolver(&fakeSecrets{}, EnvKeys{}, nil, WithBundledKeys(BundledKeys{}))

	value, source, err := r.Get(context.Background(), KeyTMDb)

	require.NoError(t, err)
	assert.Equal(t, "", value)
	assert.Equal(t, KeySourceNone, source)
	assert.False(t, r.Has(context.Background(), KeyTMDb))
}

// Only TMDb is ever bundled. The paid keys (Claude, OpenAI) must not pick up a
// fallback by accident even if someone widens BundledKeys later.
func TestKeyResolver_BundledTierIsTMDbOnly(t *testing.T) {
	r := NewKeyResolver(&fakeSecrets{}, EnvKeys{}, nil,
		WithBundledKeys(BundledKeys{TMDb: "bundled-tmdb"}))

	for _, name := range []KeyName{KeyClaude, KeyOpenAI} {
		value, source, err := r.Get(context.Background(), name)
		require.NoError(t, err)
		assert.Equal(t, "", value, string(name))
		assert.Equal(t, KeySourceNone, source, string(name))
	}
}

// A whitespace-only bundled value (a mis-set CI secret) must not be handed to
// TMDb as a blank key — it counts as absent, the same rule blank secrets follow.
func TestKeyResolver_BlankBundledIsAbsent(t *testing.T) {
	r := NewKeyResolver(&fakeSecrets{}, EnvKeys{}, nil, WithBundledKeys(BundledKeys{TMDb: "  \n"}))

	_, source, err := r.Get(context.Background(), KeyTMDb)

	require.NoError(t, err)
	assert.Equal(t, KeySourceNone, source)
}
