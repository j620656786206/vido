package crypto

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeriveKeyFromEnv_Valid(t *testing.T) {
	// Set environment variable
	testKey := "my-test-encryption-key-for-testing"
	os.Setenv("ENCRYPTION_KEY", testKey)
	defer os.Unsetenv("ENCRYPTION_KEY")

	key, err := DeriveKeyFromEnv()
	require.NoError(t, err)

	// Should return 32-byte key for AES-256
	assert.Len(t, key, KeySize)
}

func TestDeriveKeyFromEnv_NotSet(t *testing.T) {
	// Ensure env var is not set
	os.Unsetenv("ENCRYPTION_KEY")

	key, err := DeriveKeyFromEnv()

	// Should return ErrEncryptionKeyNotSet
	assert.ErrorIs(t, err, ErrEncryptionKeyNotSet)
	assert.Nil(t, key)
}

func TestDeriveKeyFromEnv_Deterministic(t *testing.T) {
	testKey := "deterministic-test-key"
	os.Setenv("ENCRYPTION_KEY", testKey)
	defer os.Unsetenv("ENCRYPTION_KEY")

	key1, err := DeriveKeyFromEnv()
	require.NoError(t, err)

	key2, err := DeriveKeyFromEnv()
	require.NoError(t, err)

	// Same input should produce same key
	assert.Equal(t, key1, key2)
}

func TestDeriveKeyFromMachineID(t *testing.T) {
	key, err := DeriveKeyFromMachineID()
	require.NoError(t, err)

	// Should return 32-byte key for AES-256
	assert.Len(t, key, KeySize)
}

func TestDeriveKeyFromMachineID_Deterministic(t *testing.T) {
	key1, err := DeriveKeyFromMachineID()
	require.NoError(t, err)

	key2, err := DeriveKeyFromMachineID()
	require.NoError(t, err)

	// Same machine should produce same key
	assert.Equal(t, key1, key2)
}

func TestDeriveKey_PrefersEnvVar(t *testing.T) {
	testKey := "env-var-key"
	os.Setenv("ENCRYPTION_KEY", testKey)
	defer os.Unsetenv("ENCRYPTION_KEY")

	keyFromEnv, err := DeriveKeyFromEnv()
	require.NoError(t, err)

	keyFromDeriveKey, _, err := DeriveKey()
	require.NoError(t, err)

	// DeriveKey should return same key as DeriveKeyFromEnv when env var is set
	assert.Equal(t, keyFromEnv, keyFromDeriveKey)
}

func TestDeriveKey_FallsBackToMachineID(t *testing.T) {
	// Ensure env var is not set
	os.Unsetenv("ENCRYPTION_KEY")

	keyFromMachine, err := DeriveKeyFromMachineID()
	require.NoError(t, err)

	keyFromDeriveKey, source, err := DeriveKey()
	require.NoError(t, err)

	// DeriveKey should return same key as DeriveKeyFromMachineID when env var is not set
	assert.Equal(t, keyFromMachine, keyFromDeriveKey)
	assert.Equal(t, KeySourceMachineID, source)
}

func TestDeriveKey_ReturnsKeySource(t *testing.T) {
	// Test with env var set
	os.Setenv("ENCRYPTION_KEY", "test-key")
	defer os.Unsetenv("ENCRYPTION_KEY")

	_, source, err := DeriveKey()
	require.NoError(t, err)
	assert.Equal(t, KeySourceEnvVar, source)

	// Test with env var not set
	os.Unsetenv("ENCRYPTION_KEY")

	_, source, err = DeriveKey()
	require.NoError(t, err)
	assert.Equal(t, KeySourceMachineID, source)
}

func TestDeriveKeyFromString(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"short string", "abc"},
		{"long string", "this-is-a-very-long-encryption-key-that-should-still-work"},
		{"unicode", "密碼測試"},
		{"special chars", "!@#$%^&*()"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := deriveKeyFromString(tt.input)

			// Should always produce 32-byte key
			assert.Len(t, key, KeySize)
		})
	}
}

func TestDeriveKeyFromString_Deterministic(t *testing.T) {
	input := "test-input-string"

	key1 := deriveKeyFromString(input)
	key2 := deriveKeyFromString(input)

	assert.Equal(t, key1, key2)
}

func TestDeriveKeyFromString_DifferentInputs(t *testing.T) {
	key1 := deriveKeyFromString("input-1")
	key2 := deriveKeyFromString("input-2")

	// Different inputs should produce different keys
	assert.NotEqual(t, key1, key2)
}
