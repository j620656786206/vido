package services

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/vido/api/internal/qbittorrent"
	"github.com/vido/api/internal/repository"
	"github.com/vido/api/internal/secrets"
)

// qBittorrent setting keys stored in the settings repository.
const (
	SettingQBHost     = "qbittorrent.host"
	SettingQBUsername = "qbittorrent.username"
	SettingQBPassword = "qbittorrent.password"
	SettingQBBasePath = "qbittorrent.base_path"
)

// QBittorrentServiceInterface defines the contract for qBittorrent business operations.
type QBittorrentServiceInterface interface {
	GetConfig(ctx context.Context) (*qbittorrent.Config, error)
	SaveConfig(ctx context.Context, config *qbittorrent.Config) error
	TestConnection(ctx context.Context) (*qbittorrent.VersionInfo, error)
	IsConfigured(ctx context.Context) bool
}

// QBittorrentService provides business logic for qBittorrent connection management.
type QBittorrentService struct {
	settingsRepo   repository.SettingsRepositoryInterface
	secretsService secrets.SecretsServiceInterface
}

// NewQBittorrentService creates a new QBittorrentService.
func NewQBittorrentService(
	settingsRepo repository.SettingsRepositoryInterface,
	secretsService secrets.SecretsServiceInterface,
) *QBittorrentService {
	return &QBittorrentService{
		settingsRepo:   settingsRepo,
		secretsService: secretsService,
	}
}

// GetConfig retrieves the qBittorrent configuration.
// The password is decrypted from the secrets store.
func (s *QBittorrentService) GetConfig(ctx context.Context) (*qbittorrent.Config, error) {
	host, _ := s.settingsRepo.GetString(ctx, SettingQBHost)
	username, _ := s.settingsRepo.GetString(ctx, SettingQBUsername)
	basePath, _ := s.settingsRepo.GetString(ctx, SettingQBBasePath)

	var password string
	exists, _ := s.secretsService.Exists(ctx, SettingQBPassword)
	if exists {
		var err error
		password, err = s.secretsService.Retrieve(ctx, SettingQBPassword)
		if err != nil {
			slog.Error("Failed to decrypt qBittorrent password", "error", err)
			return nil, fmt.Errorf("failed to decrypt password: %w", err)
		}
	}

	return &qbittorrent.Config{
		Host:     host,
		Username: username,
		Password: password,
		BasePath: basePath,
	}, nil
}

// SaveConfig persists the qBittorrent configuration.
// The password is encrypted before storage using the secrets service (AC2: FR51).
func (s *QBittorrentService) SaveConfig(ctx context.Context, config *qbittorrent.Config) error {
	if config.Host == "" {
		return fmt.Errorf("host is required")
	}

	// Store non-sensitive settings
	if err := s.settingsRepo.SetString(ctx, SettingQBHost, config.Host); err != nil {
		return fmt.Errorf("save host: %w", err)
	}
	if err := s.settingsRepo.SetString(ctx, SettingQBUsername, config.Username); err != nil {
		return fmt.Errorf("save username: %w", err)
	}
	if err := s.settingsRepo.SetString(ctx, SettingQBBasePath, config.BasePath); err != nil {
		return fmt.Errorf("save base path: %w", err)
	}

	// Encrypt and store password (NFR-I4)
	if config.Password != "" {
		if err := s.secretsService.Store(ctx, SettingQBPassword, config.Password); err != nil {
			return fmt.Errorf("encrypt password: %w", err)
		}
	}

	slog.Info("qBittorrent configuration saved", "host", config.Host)
	return nil
}

// TestConnection tests connectivity to the configured qBittorrent instance.
func (s *QBittorrentService) TestConnection(ctx context.Context) (*qbittorrent.VersionInfo, error) {
	config, err := s.GetConfig(ctx)
	if err != nil {
		return nil, err
	}

	if config.Host == "" {
		return nil, &qbittorrent.ConnectionError{
			Code:    qbittorrent.ErrCodeNotConfigured,
			Message: "qBittorrent not configured",
		}
	}

	client := qbittorrent.NewClient(config)
	return client.TestConnection(ctx)
}

// IsConfigured returns true if qBittorrent host is set.
func (s *QBittorrentService) IsConfigured(ctx context.Context) bool {
	host, err := s.settingsRepo.GetString(ctx, SettingQBHost)
	return err == nil && host != ""
}

// Compile-time interface verification
var _ QBittorrentServiceInterface = (*QBittorrentService)(nil)
