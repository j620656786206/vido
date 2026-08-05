package services

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func settingsWith(t *testing.T, secrets *fakeSecrets, env EnvKeys, writable bool) *KeySettingsService {
	t.Helper()
	return NewKeySettingsService(NewKeyResolver(secrets, env, nil), secrets, writable)
}

func stateOf(s KeySettings, name KeyName) KeyState {
	for _, k := range s.Keys {
		if k.Name == name {
			return k
		}
	}
	return KeyState{}
}

// ─── AC #3: GET shape and masking rules ────────────────────────────────────

func TestKeySettings_MasksOnlySecretSourcedKeys(t *testing.T) {
	secrets := &fakeSecrets{values: map[string]string{SecretNameClaude: "sk-ant-api03-abcdefghijk7f3a"}}
	svc := settingsWith(t, secrets, EnvKeys{TMDb: "tmdb-env-key-value"}, true)

	got := svc.List(context.Background())

	claude := stateOf(got, KeyClaude)
	assert.True(t, claude.Configured)
	assert.Equal(t, KeySourceSecret, claude.Source)
	assert.Equal(t, "sk-ant…7f3a", claude.Masked)

	tmdb := stateOf(got, KeyTMDb)
	assert.True(t, tmdb.Configured)
	assert.Equal(t, KeySourceEnv, tmdb.Source)
	// An env value is a DEPLOY secret. Re-exposing even a masked slice of it
	// through the API would leak what the operator put in the environment
	// specifically to keep out of the app.
	assert.Empty(t, tmdb.Masked, "env-sourced keys are never masked-echoed")
}

func TestKeySettings_NeverLeaksTheValue(t *testing.T) {
	const secret = "sk-ant-api03-SUPERSECRETVALUE"
	secrets := &fakeSecrets{values: map[string]string{SecretNameClaude: secret}}
	svc := settingsWith(t, secrets, EnvKeys{}, true)

	for _, k := range svc.List(context.Background()).Keys {
		assert.NotEqual(t, secret, k.Masked)
		assert.NotContains(t, secret, k.Masked+"|impossible") // guard against accidental full echo
		assert.NotContains(t, k.Masked, "SUPERSECRET")
	}
}

func TestKeySettings_UnconfiguredKeyReportsNone(t *testing.T) {
	svc := settingsWith(t, &fakeSecrets{}, EnvKeys{}, true)

	openai := stateOf(svc.List(context.Background()), KeyOpenAI)

	assert.False(t, openai.Configured)
	assert.Equal(t, KeySourceNone, openai.Source)
	assert.Empty(t, openai.Masked)
}

func TestKeySettings_ShortValueIsFullyMasked(t *testing.T) {
	secrets := &fakeSecrets{values: map[string]string{SecretNameClaude: "abc"}}
	svc := settingsWith(t, secrets, EnvKeys{}, true)

	assert.Equal(t, "•••", stateOf(svc.List(context.Background()), KeyClaude).Masked,
		"a short value must not be mostly revealed by head+tail masking")
}

func TestKeySettings_ListOrderIsStable(t *testing.T) {
	svc := settingsWith(t, &fakeSecrets{}, EnvKeys{}, true)

	first := svc.List(context.Background())
	second := svc.List(context.Background())

	require.Len(t, first.Keys, 3)
	assert.Equal(t, first.Keys, second.Keys, "the page must not reshuffle between loads")
	assert.Equal(t, KeyClaude, first.Keys[0].Name)
}

// ─── AC #4: encryption-key pre-flight ──────────────────────────────────────

func TestKeySettings_ReadOnlyWithoutEncryptionKey(t *testing.T) {
	svc := settingsWith(t, &fakeSecrets{}, EnvKeys{Claude: "sk-env"}, false)

	got := svc.List(context.Background())

	assert.False(t, got.Writable)
	assert.Equal(t, ReasonEncryptionKeyMissing, got.Reason)
	// Read-only still SHOWS state — an env-configured deployment must not look
	// broken just because it cannot accept new keys.
	assert.True(t, stateOf(got, KeyClaude).Configured)
	assert.Equal(t, KeySourceEnv, stateOf(got, KeyClaude).Source)
}

func TestKeySettings_SaveRefusedWithoutEncryptionKey(t *testing.T) {
	svc := settingsWith(t, &fakeSecrets{}, EnvKeys{}, false)

	err := svc.Save(context.Background(), map[KeyName]string{KeyClaude: "sk-new"})

	// Without this the page would accept input and fail opaquely at the storage
	// layer — the same silent-no-op family this story exists to remove.
	require.ErrorIs(t, err, ErrKeysNotWritable)
}

func TestKeySettings_NilSecretsServiceIsNeverWritable(t *testing.T) {
	svc := NewKeySettingsService(NewKeyResolver(nil, EnvKeys{}, nil), nil, true)

	assert.False(t, svc.List(context.Background()).Writable,
		"writable must also require a secrets service, not just the config flag")
}

// ─── AC #3: PUT semantics ──────────────────────────────────────────────────

func TestKeySettings_SaveStoresAndResolutionFlipsToSecret(t *testing.T) {
	secrets := &fakeSecrets{}
	svc := settingsWith(t, secrets, EnvKeys{Claude: "sk-env"}, true)

	require.NoError(t, svc.Save(context.Background(), map[KeyName]string{KeyClaude: "sk-user"}))

	claude := stateOf(svc.List(context.Background()), KeyClaude)
	assert.Equal(t, KeySourceSecret, claude.Source, "the stored key must now win over env")
}

// TestKeySettings_EmptyStringDeletesAndRevertsToEnv — the "I mistyped my key"
// recovery path. Without it a bad key saved from the UI is unfixable from the UI.
func TestKeySettings_EmptyStringDeletesAndRevertsToEnv(t *testing.T) {
	secrets := &fakeSecrets{values: map[string]string{SecretNameClaude: "sk-bad"}}
	svc := settingsWith(t, secrets, EnvKeys{Claude: "sk-env"}, true)
	require.Equal(t, KeySourceSecret, stateOf(svc.List(context.Background()), KeyClaude).Source)

	require.NoError(t, svc.Save(context.Background(), map[KeyName]string{KeyClaude: ""}))

	claude := stateOf(svc.List(context.Background()), KeyClaude)
	assert.Equal(t, KeySourceEnv, claude.Source)
	assert.True(t, claude.Configured, "deleting the secret reverts to env, it does not disable the key")
}

func TestKeySettings_OmittedKeysAreUntouched(t *testing.T) {
	secrets := &fakeSecrets{values: map[string]string{
		SecretNameClaude: "sk-claude", SecretNameTMDb: "tmdb-secret",
	}}
	svc := settingsWith(t, secrets, EnvKeys{}, true)

	require.NoError(t, svc.Save(context.Background(), map[KeyName]string{KeyClaude: "sk-new"}))

	assert.Equal(t, "tmdb-secret", secrets.values[SecretNameTMDb], "a partial update must not clear siblings")
	assert.Equal(t, "sk-new", secrets.values[SecretNameClaude])
}

func TestKeySettings_ClearingAnAbsentKeyIsANoOp(t *testing.T) {
	svc := settingsWith(t, &fakeSecrets{}, EnvKeys{}, true)

	// A page that submits its whole form will clear keys that were never set.
	require.NoError(t, svc.Save(context.Background(), map[KeyName]string{KeyOpenAI: ""}))
}

func TestKeySettings_UnknownKeyNameIsRejected(t *testing.T) {
	svc := settingsWith(t, &fakeSecrets{}, EnvKeys{}, true)

	err := svc.Save(context.Background(), map[KeyName]string{KeyName("gemini"): "x"})

	require.Error(t, err)
}

// TestKeySettings_UnknownNameRejectsBeforeAnyWrite pins the CR sub-2-1a M3 fix:
// names are validated BEFORE storage is touched, so a request mixing a valid
// key with an unknown one must not half-apply (map iteration order is random —
// without the pre-pass the valid key would be committed on some runs only).
func TestKeySettings_UnknownNameRejectsBeforeAnyWrite(t *testing.T) {
	secrets := &fakeSecrets{}
	svc := settingsWith(t, secrets, EnvKeys{}, true)

	err := svc.Save(context.Background(), map[KeyName]string{
		KeyClaude:         "sk-valid",
		KeyName("gemini"): "x",
	})

	require.Error(t, err)
	assert.Empty(t, secrets.values, "a rejected request must commit NOTHING")
}

// TestKeySettings_WhitespaceOnlyValueClearsRatherThanStores — a form submitting
// "   " means the user cleared the field, not that they want a blank key stored
// (which would shadow env — see KeyResolver's blank-secret test).
func TestKeySettings_WhitespaceOnlyValueClearsRatherThanStores(t *testing.T) {
	secrets := &fakeSecrets{values: map[string]string{SecretNameClaude: "sk-old"}}
	svc := settingsWith(t, secrets, EnvKeys{Claude: "sk-env"}, true)

	require.NoError(t, svc.Save(context.Background(), map[KeyName]string{KeyClaude: "   "}))

	assert.Equal(t, KeySourceEnv, stateOf(svc.List(context.Background()), KeyClaude).Source)
}
