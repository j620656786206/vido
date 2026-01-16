// Package secrets provides encrypted secrets management functionality.
package secrets

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"log/slog"

	"github.com/vido/api/internal/crypto"
)

var (
	// ErrSecretNotFound indicates the requested secret does not exist.
	ErrSecretNotFound = errors.New("secret not found")

	// ErrInvalidEncryptedData indicates the encrypted data is malformed.
	ErrInvalidEncryptedData = errors.New("invalid encrypted data")
)

// SecretsRepositoryInterface defines the contract for secrets persistence.
// This interface is implemented by SecretsRepository in the repository package.
type SecretsRepositoryInterface interface {
	// Set creates or updates an encrypted secret
	Set(ctx context.Context, name string, encryptedValue string) error

	// Get retrieves an encrypted secret by name
	Get(ctx context.Context, name string) (string, error)

	// Delete removes a secret by name
	Delete(ctx context.Context, name string) error

	// Exists checks if a secret exists
	Exists(ctx context.Context, name string) (bool, error)

	// List returns all secret names (not values)
	List(ctx context.Context) ([]string, error)
}

// SecretsService provides encrypted secrets management.
// It encrypts secrets before storage and decrypts them on retrieval.
type SecretsService struct {
	repo SecretsRepositoryInterface
	key  []byte
}

// SecretsServiceInterface defines the public interface for secrets management.
type SecretsServiceInterface interface {
	// Store encrypts and saves a secret
	Store(ctx context.Context, name string, value string) error

	// Retrieve decrypts and returns a secret
	Retrieve(ctx context.Context, name string) (string, error)

	// Delete removes a secret
	Delete(ctx context.Context, name string) error

	// Exists checks if a secret exists
	Exists(ctx context.Context, name string) (bool, error)

	// List returns all secret names
	List(ctx context.Context) ([]string, error)
}

// NewSecretsService creates a new secrets service with the given repository and encryption key.
// The key must be exactly 32 bytes for AES-256 encryption.
func NewSecretsService(repo SecretsRepositoryInterface, key []byte) (*SecretsService, error) {
	if len(key) != crypto.KeySize {
		return nil, fmt.Errorf("invalid key size: must be %d bytes for AES-256", crypto.KeySize)
	}

	return &SecretsService{
		repo: repo,
		key:  key,
	}, nil
}

// NewSecretsServiceWithKeyDerivation creates a new secrets service that derives the encryption key.
// It uses ENCRYPTION_KEY environment variable or falls back to machine ID.
func NewSecretsServiceWithKeyDerivation(repo SecretsRepositoryInterface) (*SecretsService, error) {
	key, source, err := crypto.DeriveKey()
	if err != nil {
		return nil, fmt.Errorf("failed to derive encryption key: %w", err)
	}

	slog.Info("Secrets service initialized",
		"key_source", source,
	)

	return &SecretsService{
		repo: repo,
		key:  key,
	}, nil
}

// Store encrypts and saves a secret with the given name.
// The secret is encrypted using AES-256-GCM before storage.
func (s *SecretsService) Store(ctx context.Context, name string, value string) error {
	slog.Info("Storing secret",
		"name", name,
		"value", MaskSecret(value),
	)

	// Encrypt the value
	encrypted, err := crypto.Encrypt([]byte(value), s.key)
	if err != nil {
		return fmt.Errorf("failed to encrypt secret: %w", err)
	}

	// Base64 encode for safe storage
	encoded := base64.StdEncoding.EncodeToString(encrypted)

	if err := s.repo.Set(ctx, name, encoded); err != nil {
		return fmt.Errorf("failed to store secret: %w", err)
	}

	return nil
}

// Retrieve decrypts and returns the secret with the given name.
// Returns ErrSecretNotFound if the secret does not exist.
func (s *SecretsService) Retrieve(ctx context.Context, name string) (string, error) {
	slog.Debug("Retrieving secret", "name", name)

	// Get encrypted value from repository
	encoded, err := s.repo.Get(ctx, name)
	if err != nil {
		if errors.Is(err, ErrSecretNotFound) {
			return "", ErrSecretNotFound
		}
		return "", fmt.Errorf("failed to retrieve secret: %w", err)
	}

	// Base64 decode
	encrypted, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", fmt.Errorf("%w: base64 decode failed: %v", ErrInvalidEncryptedData, err)
	}

	// Decrypt
	decrypted, err := crypto.Decrypt(encrypted, s.key)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt secret: %w", err)
	}

	return string(decrypted), nil
}

// Delete removes the secret with the given name.
func (s *SecretsService) Delete(ctx context.Context, name string) error {
	slog.Info("Deleting secret", "name", name)

	if err := s.repo.Delete(ctx, name); err != nil {
		return fmt.Errorf("failed to delete secret: %w", err)
	}

	return nil
}

// Exists checks if a secret with the given name exists.
func (s *SecretsService) Exists(ctx context.Context, name string) (bool, error) {
	return s.repo.Exists(ctx, name)
}

// List returns all secret names (not values).
func (s *SecretsService) List(ctx context.Context) ([]string, error) {
	return s.repo.List(ctx)
}

// Compile-time interface verification
var _ SecretsServiceInterface = (*SecretsService)(nil)
