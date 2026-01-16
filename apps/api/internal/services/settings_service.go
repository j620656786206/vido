package services

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/repository"
	"github.com/vido/api/internal/secrets"
)

// SettingsService provides business logic for application settings.
// It uses SettingsRepositoryInterface for data access, enabling
// testing with mock implementations and future database migrations.
// API keys are stored encrypted using the SecretsService.
type SettingsService struct {
	repo           repository.SettingsRepositoryInterface
	secretsService secrets.SecretsServiceInterface
}

// NewSettingsService creates a new SettingsService with the given repository.
// If secretsService is nil, API key functionality will not be available.
func NewSettingsService(repo repository.SettingsRepositoryInterface) *SettingsService {
	return &SettingsService{
		repo:           repo,
		secretsService: nil,
	}
}

// NewSettingsServiceWithSecrets creates a new SettingsService with secrets support.
// This enables encrypted storage for API keys.
func NewSettingsServiceWithSecrets(repo repository.SettingsRepositoryInterface, secretsSvc secrets.SecretsServiceInterface) *SettingsService {
	return &SettingsService{
		repo:           repo,
		secretsService: secretsSvc,
	}
}

// Set creates or updates a setting with validation.
func (s *SettingsService) Set(ctx context.Context, setting *models.Setting) error {
	if setting == nil {
		return fmt.Errorf("setting cannot be nil")
	}

	if setting.Key == "" {
		return fmt.Errorf("setting key cannot be empty")
	}

	slog.Info("Setting value", "key", setting.Key, "type", setting.Type)

	if err := s.repo.Set(ctx, setting); err != nil {
		slog.Error("Failed to set setting", "error", err, "key", setting.Key)
		return fmt.Errorf("failed to set setting: %w", err)
	}

	return nil
}

// Get retrieves a setting by its key.
func (s *SettingsService) Get(ctx context.Context, key string) (*models.Setting, error) {
	if key == "" {
		return nil, fmt.Errorf("setting key cannot be empty")
	}

	setting, err := s.repo.Get(ctx, key)
	if err != nil {
		slog.Error("Failed to get setting", "error", err, "key", key)
		return nil, err
	}

	return setting, nil
}

// GetAll retrieves all settings.
func (s *SettingsService) GetAll(ctx context.Context) ([]models.Setting, error) {
	settings, err := s.repo.GetAll(ctx)
	if err != nil {
		slog.Error("Failed to get all settings", "error", err)
		return nil, fmt.Errorf("failed to get all settings: %w", err)
	}

	return settings, nil
}

// Delete removes a setting by its key.
func (s *SettingsService) Delete(ctx context.Context, key string) error {
	if key == "" {
		return fmt.Errorf("setting key cannot be empty")
	}

	slog.Info("Deleting setting", "key", key)

	if err := s.repo.Delete(ctx, key); err != nil {
		slog.Error("Failed to delete setting", "error", err, "key", key)
		return fmt.Errorf("failed to delete setting: %w", err)
	}

	return nil
}

// GetString retrieves a setting as a string value.
func (s *SettingsService) GetString(ctx context.Context, key string) (string, error) {
	if key == "" {
		return "", fmt.Errorf("setting key cannot be empty")
	}

	value, err := s.repo.GetString(ctx, key)
	if err != nil {
		return "", err
	}

	return value, nil
}

// GetInt retrieves a setting as an integer value.
func (s *SettingsService) GetInt(ctx context.Context, key string) (int, error) {
	if key == "" {
		return 0, fmt.Errorf("setting key cannot be empty")
	}

	value, err := s.repo.GetInt(ctx, key)
	if err != nil {
		return 0, err
	}

	return value, nil
}

// GetBool retrieves a setting as a boolean value.
func (s *SettingsService) GetBool(ctx context.Context, key string) (bool, error) {
	if key == "" {
		return false, fmt.Errorf("setting key cannot be empty")
	}

	value, err := s.repo.GetBool(ctx, key)
	if err != nil {
		return false, err
	}

	return value, nil
}

// SetString is a convenience method to set a string value.
func (s *SettingsService) SetString(ctx context.Context, key, value string) error {
	if key == "" {
		return fmt.Errorf("setting key cannot be empty")
	}

	slog.Info("Setting string value", "key", key)

	if err := s.repo.SetString(ctx, key, value); err != nil {
		slog.Error("Failed to set string setting", "error", err, "key", key)
		return fmt.Errorf("failed to set string setting: %w", err)
	}

	return nil
}

// SetInt is a convenience method to set an integer value.
func (s *SettingsService) SetInt(ctx context.Context, key string, value int) error {
	if key == "" {
		return fmt.Errorf("setting key cannot be empty")
	}

	slog.Info("Setting int value", "key", key, "value", value)

	if err := s.repo.SetInt(ctx, key, value); err != nil {
		slog.Error("Failed to set int setting", "error", err, "key", key)
		return fmt.Errorf("failed to set int setting: %w", err)
	}

	return nil
}

// SetBool is a convenience method to set a boolean value.
func (s *SettingsService) SetBool(ctx context.Context, key string, value bool) error {
	if key == "" {
		return fmt.Errorf("setting key cannot be empty")
	}

	slog.Info("Setting bool value", "key", key, "value", value)

	if err := s.repo.SetBool(ctx, key, value); err != nil {
		slog.Error("Failed to set bool setting", "error", err, "key", key)
		return fmt.Errorf("failed to set bool setting: %w", err)
	}

	return nil
}

// GetStringWithDefault retrieves a string setting or returns the default if not found.
func (s *SettingsService) GetStringWithDefault(ctx context.Context, key, defaultValue string) string {
	value, err := s.GetString(ctx, key)
	if err != nil {
		return defaultValue
	}
	return value
}

// GetIntWithDefault retrieves an int setting or returns the default if not found.
func (s *SettingsService) GetIntWithDefault(ctx context.Context, key string, defaultValue int) int {
	value, err := s.GetInt(ctx, key)
	if err != nil {
		return defaultValue
	}
	return value
}

// GetBoolWithDefault retrieves a bool setting or returns the default if not found.
func (s *SettingsService) GetBoolWithDefault(ctx context.Context, key string, defaultValue bool) bool {
	value, err := s.GetBool(ctx, key)
	if err != nil {
		return defaultValue
	}
	return value
}

// SetAPIKey stores an API key with encryption.
// The key is encrypted using AES-256-GCM before storage.
// Requires secrets service to be configured.
func (s *SettingsService) SetAPIKey(ctx context.Context, name, value string) error {
	if s.secretsService == nil {
		return fmt.Errorf("secrets service not configured: cannot store API keys securely")
	}

	if name == "" {
		return fmt.Errorf("API key name cannot be empty")
	}

	slog.Info("Storing API key",
		"name", name,
		"value", secrets.MaskSecret(value),
	)

	if err := s.secretsService.Store(ctx, name, value); err != nil {
		slog.Error("Failed to store API key",
			"error", err,
			"name", name,
		)
		return fmt.Errorf("failed to store API key: %w", err)
	}

	return nil
}

// GetAPIKey retrieves and decrypts an API key.
// Returns the plaintext API key value.
// Requires secrets service to be configured.
func (s *SettingsService) GetAPIKey(ctx context.Context, name string) (string, error) {
	if s.secretsService == nil {
		return "", fmt.Errorf("secrets service not configured: cannot retrieve API keys")
	}

	if name == "" {
		return "", fmt.Errorf("API key name cannot be empty")
	}

	value, err := s.secretsService.Retrieve(ctx, name)
	if err != nil {
		slog.Error("Failed to retrieve API key",
			"error", err,
			"name", name,
		)
		return "", fmt.Errorf("failed to retrieve API key: %w", err)
	}

	return value, nil
}

// DeleteAPIKey removes an API key from encrypted storage.
// Requires secrets service to be configured.
func (s *SettingsService) DeleteAPIKey(ctx context.Context, name string) error {
	if s.secretsService == nil {
		return fmt.Errorf("secrets service not configured: cannot delete API keys")
	}

	if name == "" {
		return fmt.Errorf("API key name cannot be empty")
	}

	slog.Info("Deleting API key", "name", name)

	if err := s.secretsService.Delete(ctx, name); err != nil {
		slog.Error("Failed to delete API key",
			"error", err,
			"name", name,
		)
		return fmt.Errorf("failed to delete API key: %w", err)
	}

	return nil
}

// HasAPIKey checks if an API key exists in encrypted storage.
// Requires secrets service to be configured.
func (s *SettingsService) HasAPIKey(ctx context.Context, name string) (bool, error) {
	if s.secretsService == nil {
		return false, fmt.Errorf("secrets service not configured: cannot check API keys")
	}

	if name == "" {
		return false, fmt.Errorf("API key name cannot be empty")
	}

	return s.secretsService.Exists(ctx, name)
}

// ListAPIKeys returns the names of all stored API keys (not the values).
// Requires secrets service to be configured.
func (s *SettingsService) ListAPIKeys(ctx context.Context) ([]string, error) {
	if s.secretsService == nil {
		return nil, fmt.Errorf("secrets service not configured: cannot list API keys")
	}

	return s.secretsService.List(ctx)
}
