package services

import (
	"context"
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"

	"github.com/vido/api/internal/repository"
	"github.com/vido/api/internal/secrets"
)

// This is sub-2-1a AC #6.4 — the end-to-end proof that Break 1 is closed, run
// against a REAL migrated database and the REAL AES-256-GCM secrets service
// rather than a fake (Rule 15 / the bugfix-20-1 precedent: a mocked repo hid a
// read path that silently returned the zero value for months).
//
// Break 1 was: the key the pipeline reads is env-only, nothing consults the
// secrets store. A settings page could save successfully and change nothing —
// the user sets a key, sees "saved", and translation stays gated. These tests
// exercise store → resolve → gate-flips as one chain.
//
// It lives in `services`, not `repository`: Rule 19 forbids repository →
// services, so a repository-package test importing services is an import cycle
// (`go vet` rejects it). services → repository is the legal direction.

func newRealSecretsService(t *testing.T) (secrets.SecretsServiceInterface, func()) {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	// The migration-005 schema verbatim — a real table, not a stub, so the
	// encrypted round trip exercises the same columns production uses.
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS secrets (
			id TEXT PRIMARY KEY,
			name TEXT UNIQUE NOT NULL,
			encrypted_value TEXT NOT NULL,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
		CREATE INDEX IF NOT EXISTS idx_secrets_name ON secrets(name);`)
	require.NoError(t, err)

	svc, err := secrets.NewSecretsService(repository.NewSecretsRepository(db), []byte("0123456789abcdef0123456789abcdef"))
	require.NoError(t, err)
	return svc, func() { _ = db.Close() }
}

func TestKeyResolution_StoredKeyReachesTheResolver(t *testing.T) {
	ctx := context.Background()
	secretsSvc, cleanup := newRealSecretsService(t)
	defer cleanup()

	resolver := NewKeyResolver(secretsSvc, EnvKeys{}, nil)
	require.False(t, resolver.Has(ctx, KeyClaude), "precondition: nothing configured")

	// Exactly what PUT /api/v1/settings/keys does.
	settings := NewKeySettingsService(resolver, secretsSvc, true)
	require.NoError(t, settings.Save(ctx, map[KeyName]string{KeyClaude: "sk-ant-real-key"}))

	value, source, err := resolver.Get(ctx, KeyClaude)
	require.NoError(t, err)
	assert.Equal(t, "sk-ant-real-key", value, "round-tripped through real AES-256-GCM encryption")
	assert.Equal(t, KeySourceSecret, source)
	assert.True(t, resolver.Has(ctx, KeyClaude),
		"THE assertion: the capability gate flips true from a UI save alone — no restart, no env-var")
}

func TestKeyResolution_StoredKeyOverridesEnv(t *testing.T) {
	ctx := context.Background()
	secretsSvc, cleanup := newRealSecretsService(t)
	defer cleanup()

	resolver := NewKeyResolver(secretsSvc, EnvKeys{Claude: "sk-from-env"}, nil)
	settings := NewKeySettingsService(resolver, secretsSvc, true)

	require.NoError(t, settings.Save(ctx, map[KeyName]string{KeyClaude: "sk-from-ui"}))

	value, source, err := resolver.Get(ctx, KeyClaude)
	require.NoError(t, err)
	assert.Equal(t, "sk-from-ui", value)
	assert.Equal(t, KeySourceSecret, source)
}

// TestKeyResolution_HolderRebuildsAfterAStore is Break 2's proof: the provider
// used to be built once at boot, so a key saved at runtime reached a client that
// no longer existed to receive it.
func TestKeyResolution_HolderRebuildsAfterAStore(t *testing.T) {
	ctx := context.Background()
	secretsSvc, cleanup := newRealSecretsService(t)
	defer cleanup()

	resolver := NewKeyResolver(secretsSvc, EnvKeys{}, nil)
	holder := NewClaudeProviderHolder(resolver, "", nil)
	settings := NewKeySettingsService(resolver, secretsSvc, true)

	_, err := holder.Get(ctx)
	require.Error(t, err, "precondition: a keyless boot has no provider")
	require.False(t, holder.IsConfigured(ctx))

	require.NoError(t, settings.Save(ctx, map[KeyName]string{KeyClaude: "sk-late-arrival"}))

	provider, err := holder.Get(ctx)
	require.NoError(t, err, "the holder must pick the key up with no restart")
	assert.NotNil(t, provider)
	assert.True(t, holder.IsConfigured(ctx))
}

// TestKeyResolution_ClearRevertsToEnvOnRealStorage — the mistyped-key recovery
// path, verified against real encryption + a real delete.
func TestKeyResolution_ClearRevertsToEnvOnRealStorage(t *testing.T) {
	ctx := context.Background()
	secretsSvc, cleanup := newRealSecretsService(t)
	defer cleanup()

	resolver := NewKeyResolver(secretsSvc, EnvKeys{Claude: "sk-from-env"}, nil)
	settings := NewKeySettingsService(resolver, secretsSvc, true)
	require.NoError(t, settings.Save(ctx, map[KeyName]string{KeyClaude: "sk-typo"}))
	require.Equal(t, KeySourceSecret, sourceOf(t, resolver, KeyClaude))

	require.NoError(t, settings.Save(ctx, map[KeyName]string{KeyClaude: ""}))

	value, source, err := resolver.Get(ctx, KeyClaude)
	require.NoError(t, err)
	assert.Equal(t, "sk-from-env", value)
	assert.Equal(t, KeySourceEnv, source)
}

// TestKeyResolution_SettingsListNeverExposesTheStoredValue guards the masking
// rule against the real encrypted round trip, not just a fake.
func TestKeyResolution_SettingsListNeverExposesTheStoredValue(t *testing.T) {
	ctx := context.Background()
	secretsSvc, cleanup := newRealSecretsService(t)
	defer cleanup()

	const key = "sk-ant-api03-DO-NOT-LEAK-ME-1234"
	resolver := NewKeyResolver(secretsSvc, EnvKeys{}, nil)
	settings := NewKeySettingsService(resolver, secretsSvc, true)
	require.NoError(t, settings.Save(ctx, map[KeyName]string{KeyClaude: key}))

	for _, state := range settings.List(ctx).Keys {
		assert.NotEqual(t, key, state.Masked)
		assert.NotContains(t, state.Masked, "DO-NOT-LEAK-ME")
	}
}

func sourceOf(t *testing.T, r KeyResolver, name KeyName) KeySource {
	t.Helper()
	_, source, err := r.Get(context.Background(), name)
	require.NoError(t, err)
	return source
}
