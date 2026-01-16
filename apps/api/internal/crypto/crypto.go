// Package crypto provides AES-256-GCM encryption/decryption for secrets management.
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"io"
)

var (
	// ErrInvalidKeySize indicates the encryption key is not 32 bytes (required for AES-256).
	ErrInvalidKeySize = errors.New("invalid key size: must be 32 bytes for AES-256")

	// ErrCiphertextShort indicates the ciphertext is too short to contain nonce and data.
	ErrCiphertextShort = errors.New("ciphertext too short")

	// ErrDecryptionFailed indicates decryption failed due to authentication error.
	ErrDecryptionFailed = errors.New("decryption failed: authentication error")
)

const (
	// KeySize is the required key size for AES-256 (32 bytes).
	KeySize = 32
)

// Encrypt encrypts plaintext using AES-256-GCM (Galois/Counter Mode).
// Returns: nonce (12 bytes) + ciphertext + auth tag (16 bytes).
// The nonce is randomly generated for each encryption operation.
func Encrypt(plaintext []byte, key []byte) ([]byte, error) {
	if len(key) != KeySize {
		return nil, ErrInvalidKeySize
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	// Generate random nonce for each encryption
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	// Seal appends ciphertext + auth tag to nonce
	// Result format: [nonce (12 bytes)][ciphertext][auth tag (16 bytes)]
	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

// Decrypt decrypts ciphertext using AES-256-GCM.
// Expects input format: [nonce (12 bytes)][ciphertext][auth tag (16 bytes)].
// Returns ErrDecryptionFailed if the ciphertext has been tampered with or wrong key is used.
func Decrypt(ciphertext []byte, key []byte) ([]byte, error) {
	if len(key) != KeySize {
		return nil, ErrInvalidKeySize
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, ErrCiphertextShort
	}

	nonce, encryptedData := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, encryptedData, nil)
	if err != nil {
		return nil, ErrDecryptionFailed
	}

	return plaintext, nil
}
