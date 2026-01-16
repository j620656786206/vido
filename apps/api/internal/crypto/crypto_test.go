package crypto

import (
	"bytes"
	"crypto/rand"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEncrypt_ValidInput(t *testing.T) {
	key := make([]byte, 32)
	_, err := rand.Read(key)
	require.NoError(t, err)

	plaintext := []byte("my-secret-api-key")

	ciphertext, err := Encrypt(plaintext, key)
	require.NoError(t, err)

	// Ciphertext should be different from plaintext
	assert.NotEqual(t, plaintext, ciphertext)

	// Ciphertext should be longer than plaintext (nonce + auth tag)
	// Nonce: 12 bytes, Auth tag: 16 bytes
	assert.Greater(t, len(ciphertext), len(plaintext))
}

func TestEncrypt_InvalidKeySize(t *testing.T) {
	tests := []struct {
		name    string
		keySize int
	}{
		{"too short - 16 bytes", 16},
		{"too short - 24 bytes", 24},
		{"too long - 48 bytes", 48},
		{"empty key", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := make([]byte, tt.keySize)
			plaintext := []byte("test")

			_, err := Encrypt(plaintext, key)
			assert.ErrorIs(t, err, ErrInvalidKeySize)
		})
	}
}

func TestDecrypt_ValidInput(t *testing.T) {
	key := make([]byte, 32)
	_, err := rand.Read(key)
	require.NoError(t, err)

	originalPlaintext := []byte("my-secret-api-key")

	ciphertext, err := Encrypt(originalPlaintext, key)
	require.NoError(t, err)

	decrypted, err := Decrypt(ciphertext, key)
	require.NoError(t, err)

	assert.Equal(t, originalPlaintext, decrypted)
}

func TestDecrypt_InvalidKeySize(t *testing.T) {
	key := make([]byte, 16) // Wrong size
	ciphertext := make([]byte, 50)

	_, err := Decrypt(ciphertext, key)
	assert.ErrorIs(t, err, ErrInvalidKeySize)
}

func TestDecrypt_CiphertextTooShort(t *testing.T) {
	key := make([]byte, 32)
	_, err := rand.Read(key)
	require.NoError(t, err)

	// Ciphertext shorter than nonce size (12 bytes)
	ciphertext := []byte("short")

	_, err = Decrypt(ciphertext, key)
	assert.ErrorIs(t, err, ErrCiphertextShort)
}

func TestDecrypt_WrongKey(t *testing.T) {
	key1 := make([]byte, 32)
	key2 := make([]byte, 32)
	_, err := rand.Read(key1)
	require.NoError(t, err)
	_, err = rand.Read(key2)
	require.NoError(t, err)

	// Ensure keys are different
	for bytes.Equal(key1, key2) {
		_, _ = rand.Read(key2)
	}

	plaintext := []byte("secret-data")

	ciphertext, err := Encrypt(plaintext, key1)
	require.NoError(t, err)

	// Decrypt with wrong key should fail
	_, err = Decrypt(ciphertext, key2)
	assert.ErrorIs(t, err, ErrDecryptionFailed)
}

func TestDecrypt_TamperedCiphertext(t *testing.T) {
	key := make([]byte, 32)
	_, err := rand.Read(key)
	require.NoError(t, err)

	plaintext := []byte("secret-data")

	ciphertext, err := Encrypt(plaintext, key)
	require.NoError(t, err)

	// Tamper with ciphertext
	ciphertext[len(ciphertext)-1] ^= 0xFF

	// Decryption should fail due to authentication error
	_, err = Decrypt(ciphertext, key)
	assert.ErrorIs(t, err, ErrDecryptionFailed)
}

func TestEncryptDecrypt_Roundtrip(t *testing.T) {
	key := make([]byte, 32)
	_, err := rand.Read(key)
	require.NoError(t, err)

	testCases := []struct {
		name      string
		plaintext string
	}{
		{"short string", "abc"},
		{"typical API key", "sk-1234567890abcdef"},
		{"long string", "this-is-a-very-long-secret-key-that-might-be-used-for-something"},
		{"unicode characters", "密碼測試123"},
		{"empty string", ""}, // Note: empty []byte becomes nil after decryption in Go
		{"special characters", "!@#$%^&*()_+-=[]{}|;':\",./<>?"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			plaintext := []byte(tc.plaintext)

			ciphertext, err := Encrypt(plaintext, key)
			require.NoError(t, err)

			decrypted, err := Decrypt(ciphertext, key)
			require.NoError(t, err)

			// Use bytes.Equal for comparison to handle nil vs empty slice
			assert.True(t, bytes.Equal(plaintext, decrypted), "decrypted content should match original")
		})
	}
}

func TestEncrypt_UniqueNonce(t *testing.T) {
	key := make([]byte, 32)
	_, err := rand.Read(key)
	require.NoError(t, err)

	plaintext := []byte("same-plaintext")

	// Encrypt same plaintext multiple times
	ciphertext1, err := Encrypt(plaintext, key)
	require.NoError(t, err)

	ciphertext2, err := Encrypt(plaintext, key)
	require.NoError(t, err)

	// Ciphertexts should be different due to random nonce
	assert.NotEqual(t, ciphertext1, ciphertext2)

	// Both should decrypt to same plaintext
	decrypted1, err := Decrypt(ciphertext1, key)
	require.NoError(t, err)

	decrypted2, err := Decrypt(ciphertext2, key)
	require.NoError(t, err)

	assert.Equal(t, plaintext, decrypted1)
	assert.Equal(t, plaintext, decrypted2)
}
