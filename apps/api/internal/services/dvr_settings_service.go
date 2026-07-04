// Package services — DVRSettingsService (Story 13-4a, AC #3/#4).
//
// Per-plugin *arr config over the settings table + secretsService (the
// qBittorrent precedent — RULING: no new table, no migration). Parameterized
// by plugin name so 13-4b's sonarr reuses it with zero duplication. The
// settings endpoints' shape carries [@contract-v1] (13-4a AC #4, consumer 13-6).
package services

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/vido/api/internal/plugins"
	"github.com/vido/api/internal/repository"
	"github.com/vido/api/internal/secrets"
)

// DVRConfigInput is the PUT /settings/{plugin} body (snake_case per Rule 6).
// An empty APIKey means "keep the stored key" — the GET response never
// returns the key, so the UI cannot echo it back.
type DVRConfigInput struct {
	URL              string `json:"url"`
	APIKey           string `json:"api_key"`
	Enabled          bool   `json:"enabled"`
	QualityProfileID int64  `json:"quality_profile_id"`
	RootFolderPath   string `json:"root_folder_path"`
}

// DVRConfigStatus is the GET /settings/{plugin} response — config sans key
// (has_api_key only) + the live health block (AC #4).
type DVRConfigStatus struct {
	URL              string               `json:"url"`
	Enabled          bool                 `json:"enabled"`
	QualityProfileID int64                `json:"quality_profile_id"`
	RootFolderPath   string               `json:"root_folder_path"`
	HasAPIKey        bool                 `json:"has_api_key"`
	Health           plugins.PluginHealth `json:"health"`
}

// DVRSettingsServiceInterface defines the contract for DVR plugin settings
// operations (Rule 11 — handlers consume this from the services package).
type DVRSettingsServiceInterface interface {
	GetConfig(ctx context.Context, plugin string) (*DVRConfigStatus, error)
	SaveConfig(ctx context.Context, plugin string, input DVRConfigInput) error
	TestConnection(ctx context.Context, plugin string, input *DVRConfigInput) error
	GetQualityProfiles(ctx context.Context, plugin string) ([]plugins.QualityProfile, error)
	GetRootFolders(ctx context.Context, plugin string) ([]plugins.RootFolder, error)
}

// DVRSettingsService implements DVRSettingsServiceInterface over the plugin
// manager (test/health/client) + settings repo + secrets service (storage).
type DVRSettingsService struct {
	manager      *plugins.Manager
	settingsRepo repository.SettingsRepositoryInterface
	secrets      secrets.SecretsServiceInterface
}

// Compile-time interface verification.
var _ DVRSettingsServiceInterface = (*DVRSettingsService)(nil)

// NewDVRSettingsService creates a new DVRSettingsService.
func NewDVRSettingsService(
	manager *plugins.Manager,
	settingsRepo repository.SettingsRepositoryInterface,
	secretsService secrets.SecretsServiceInterface,
) *DVRSettingsService {
	return &DVRSettingsService{manager: manager, settingsRepo: settingsRepo, secrets: secretsService}
}

// GetConfig returns the stored config sans key + the live health block.
func (s *DVRSettingsService) GetConfig(ctx context.Context, plugin string) (*DVRConfigStatus, error) {
	url, _ := s.settingsRepo.GetString(ctx, plugins.SettingKeyURL(plugin))
	enabled, _ := s.settingsRepo.GetBool(ctx, plugins.SettingKeyEnabled(plugin))
	profileID, _ := s.settingsRepo.GetInt(ctx, plugins.SettingKeyQualityProfileID(plugin))
	rootFolder, _ := s.settingsRepo.GetString(ctx, plugins.SettingKeyRootFolderPath(plugin))
	hasKey, _ := s.secrets.Exists(ctx, plugins.SettingKeyAPIKey(plugin))

	return &DVRConfigStatus{
		URL:              url,
		Enabled:          enabled,
		QualityProfileID: int64(profileID),
		RootFolderPath:   rootFolder,
		HasAPIKey:        hasKey,
		Health:           s.manager.Health(plugin),
	}, nil
}

// SaveConfig persists a plugin config — with the server-side test-before-save
// guard (AC #4, §7 "must pass before save"; NEW vs the UI-driven qBT
// precedent): when the plugin is enabled, TestConnection runs against the
// INCOMING config and a failure refuses persistence with DVR_TEST_FAILED.
// Disabling skips the probe — turning a plugin off must not require a
// reachable server.
func (s *DVRSettingsService) SaveConfig(ctx context.Context, plugin string, input DVRConfigInput) error {
	if input.URL == "" {
		return fmt.Errorf("url is required")
	}

	effectiveKey, err := s.effectiveAPIKey(ctx, plugin, input.APIKey)
	if err != nil {
		return err
	}

	if input.Enabled {
		candidate := plugins.PluginConfig{URL: input.URL, APIKey: effectiveKey}
		if err := s.manager.TestConfig(ctx, plugin, candidate); err != nil {
			return &plugins.PluginError{
				Code:    plugins.ErrCodeTestFailed,
				Message: fmt.Sprintf("%s connection test failed — config not saved", plugin),
				Cause:   err,
			}
		}
	}

	if err := s.settingsRepo.SetString(ctx, plugins.SettingKeyURL(plugin), input.URL); err != nil {
		return fmt.Errorf("save url: %w", err)
	}
	if err := s.settingsRepo.SetBool(ctx, plugins.SettingKeyEnabled(plugin), input.Enabled); err != nil {
		return fmt.Errorf("save enabled: %w", err)
	}
	if err := s.settingsRepo.SetInt(ctx, plugins.SettingKeyQualityProfileID(plugin), int(input.QualityProfileID)); err != nil {
		return fmt.Errorf("save quality profile id: %w", err)
	}
	if err := s.settingsRepo.SetString(ctx, plugins.SettingKeyRootFolderPath(plugin), input.RootFolderPath); err != nil {
		return fmt.Errorf("save root folder path: %w", err)
	}
	if input.APIKey != "" {
		if err := s.secrets.Store(ctx, plugins.SettingKeyAPIKey(plugin), input.APIKey); err != nil {
			return fmt.Errorf("encrypt api key: %w", err)
		}
	}

	// Refresh health immediately so the settings GET reflects the new config
	// without waiting for the next 60s sweep (also swaps the fingerprint cache).
	s.manager.CheckHealth(ctx, plugin)

	slog.Info("DVR plugin configuration saved", "plugin", plugin, "url", input.URL, "enabled", input.Enabled)
	return nil
}

// TestConnection probes the body config when provided (test-then-save UX),
// else the saved config (qBT TestConnectionWithConfig pattern).
func (s *DVRSettingsService) TestConnection(ctx context.Context, plugin string, input *DVRConfigInput) error {
	if input != nil && input.URL != "" {
		effectiveKey, err := s.effectiveAPIKey(ctx, plugin, input.APIKey)
		if err != nil {
			return err
		}
		return s.manager.TestConfig(ctx, plugin, plugins.PluginConfig{URL: input.URL, APIKey: effectiveKey})
	}

	config, _, err := s.manager.LoadConfig(ctx, plugin)
	if err != nil {
		return err
	}
	if config.URL == "" || config.APIKey == "" {
		return &plugins.PluginError{
			Code:    plugins.ErrCodeNotConfigured,
			Message: fmt.Sprintf("%s is not configured", plugin),
		}
	}
	return s.manager.TestConfig(ctx, plugin, config)
}

// GetQualityProfiles passes through the plugin's quality profiles (AC #4 —
// needed to choose a valid quality_profile_id; consumed by 13-6).
func (s *DVRSettingsService) GetQualityProfiles(ctx context.Context, plugin string) ([]plugins.QualityProfile, error) {
	lister, err := s.profileLister(ctx, plugin)
	if err != nil {
		return nil, err
	}
	return lister.GetQualityProfiles(ctx)
}

// GetRootFolders passes through the plugin's root folders — see GetQualityProfiles.
func (s *DVRSettingsService) GetRootFolders(ctx context.Context, plugin string) ([]plugins.RootFolder, error) {
	lister, err := s.profileLister(ctx, plugin)
	if err != nil {
		return nil, err
	}
	return lister.GetRootFolders(ctx)
}

// profileLister returns the configured plugin client as a ProfileLister.
func (s *DVRSettingsService) profileLister(ctx context.Context, plugin string) (plugins.ProfileLister, error) {
	client, err := s.manager.GetClient(ctx, plugin)
	if err != nil {
		return nil, err
	}
	lister, ok := client.(plugins.ProfileLister)
	if !ok {
		return nil, &plugins.PluginError{
			Code:    plugins.ErrCodeNotSupported,
			Message: fmt.Sprintf("%s does not expose quality profiles / root folders", plugin),
		}
	}
	return lister, nil
}

// effectiveAPIKey resolves the key to use: the provided one, or the stored
// one when the input omits it (the GET response never echoes the key back).
func (s *DVRSettingsService) effectiveAPIKey(ctx context.Context, plugin, inputKey string) (string, error) {
	if inputKey != "" {
		return inputKey, nil
	}
	exists, _ := s.secrets.Exists(ctx, plugins.SettingKeyAPIKey(plugin))
	if !exists {
		return "", nil
	}
	stored, err := s.secrets.Retrieve(ctx, plugins.SettingKeyAPIKey(plugin))
	if err != nil {
		return "", fmt.Errorf("decrypt stored api key: %w", err)
	}
	return stored, nil
}
