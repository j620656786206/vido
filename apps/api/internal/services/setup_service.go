package services

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/repository"
	"github.com/vido/api/internal/secrets"
)

// ErrSetupAlreadyCompleted is returned when setup has already been completed.
var ErrSetupAlreadyCompleted = errors.New("setup already completed")

// SetupService provides business logic for the setup wizard.
// Implements handlers.SetupServiceInterface.
type SetupService struct {
	settingsRepo   repository.SettingsRepositoryInterface
	secretsService secrets.SecretsServiceInterface
}

// NewSetupService creates a new SetupService.
func NewSetupService(settingsRepo repository.SettingsRepositoryInterface, secretsSvc secrets.SecretsServiceInterface) *SetupService {
	return &SetupService{
		settingsRepo:   settingsRepo,
		secretsService: secretsSvc,
	}
}

// IsFirstRun checks if the setup wizard has been completed.
// Returns true if setup_completed flag is not set or is false.
// Returns an error only for real failures (DB errors), not for missing keys.
func (s *SetupService) IsFirstRun(ctx context.Context) (bool, error) {
	completed, err := s.settingsRepo.GetBool(ctx, "setup_completed")
	if err != nil {
		// Settings repo returns "not found" for missing keys — treat as first run.
		// Other errors (DB connection failure, etc.) are propagated.
		if strings.Contains(err.Error(), "not found") {
			slog.Debug("Setup completed flag not found, treating as first run")
			return true, nil
		}
		slog.Error("Failed to check setup status", "error", err)
		return false, fmt.Errorf("check setup status: %w", err)
	}
	return !completed, nil
}

// CompleteSetup saves all wizard settings and marks setup as completed.
func (s *SetupService) CompleteSetup(ctx context.Context, config models.SetupConfig) error {
	// Check if setup is already completed
	isFirst, err := s.IsFirstRun(ctx)
	if err != nil {
		return fmt.Errorf("check first run: %w", err)
	}
	if !isFirst {
		return ErrSetupAlreadyCompleted
	}

	// Save language
	if config.Language != "" {
		if err := s.settingsRepo.SetString(ctx, "language", config.Language); err != nil {
			return fmt.Errorf("save language: %w", err)
		}
	}

	// Save qBittorrent settings (optional)
	// Store using both legacy (qbt_*) and canonical (qbittorrent.*) keys
	// so both setup wizard and connection settings page can read them.
	if config.QBTUrl != "" {
		if err := s.settingsRepo.SetString(ctx, "qbt_url", config.QBTUrl); err != nil {
			return fmt.Errorf("save qbt_url: %w", err)
		}
		if err := s.settingsRepo.SetString(ctx, SettingQBHost, config.QBTUrl); err != nil {
			return fmt.Errorf("save qbittorrent.host: %w", err)
		}
		if config.QBTUsername != "" {
			if err := s.settingsRepo.SetString(ctx, "qbt_username", config.QBTUsername); err != nil {
				return fmt.Errorf("save qbt_username: %w", err)
			}
			if err := s.settingsRepo.SetString(ctx, SettingQBUsername, config.QBTUsername); err != nil {
				return fmt.Errorf("save qbittorrent.username: %w", err)
			}
		}
		if config.QBTPassword != "" && s.secretsService != nil {
			if err := s.secretsService.Store(ctx, "qbt_password", config.QBTPassword); err != nil {
				return fmt.Errorf("save qbt_password: %w", err)
			}
			if err := s.secretsService.Store(ctx, SettingQBPassword, config.QBTPassword); err != nil {
				return fmt.Errorf("save qbittorrent.password: %w", err)
			}
		}
	}

	// Save media folder path
	if config.MediaFolderPath != "" {
		if err := s.settingsRepo.SetString(ctx, "media_folder_path", config.MediaFolderPath); err != nil {
			return fmt.Errorf("save media_folder_path: %w", err)
		}
	}

	// Save API keys (optional, encrypted)
	if config.TMDbApiKey != "" && s.secretsService != nil {
		if err := s.secretsService.Store(ctx, "tmdb_api_key", config.TMDbApiKey); err != nil {
			return fmt.Errorf("save tmdb_api_key: %w", err)
		}
	}

	if config.AIProvider != "" {
		if err := s.settingsRepo.SetString(ctx, "ai_provider", config.AIProvider); err != nil {
			return fmt.Errorf("save ai_provider: %w", err)
		}
	}

	if config.AIApiKey != "" && s.secretsService != nil {
		if err := s.secretsService.Store(ctx, "ai_api_key", config.AIApiKey); err != nil {
			return fmt.Errorf("save ai_api_key: %w", err)
		}
	}

	// Mark setup as completed
	if err := s.settingsRepo.SetBool(ctx, "setup_completed", true); err != nil {
		return fmt.Errorf("mark setup completed: %w", err)
	}

	slog.Info("Setup wizard completed successfully",
		"language", config.Language,
		"has_qbt", config.QBTUrl != "",
		"has_tmdb_key", config.TMDbApiKey != "",
		"has_ai_key", config.AIApiKey != "",
	)

	return nil
}

// ValidateStep validates individual step data.
func (s *SetupService) ValidateStep(ctx context.Context, step string, data map[string]interface{}) error {
	switch step {
	case "welcome":
		return s.validateWelcomeStep(data)
	case "qbittorrent":
		return s.validateQBittorrentStep(ctx, data)
	case "media-folder":
		return s.validateMediaFolderStep(data)
	case "api-keys":
		return s.validateApiKeysStep(data)
	case "complete":
		return nil
	default:
		return fmt.Errorf("unknown step: %s", step)
	}
}

func (s *SetupService) validateWelcomeStep(data map[string]interface{}) error {
	lang, ok := data["language"].(string)
	if !ok || lang == "" {
		return fmt.Errorf("language is required")
	}
	return nil
}

func (s *SetupService) validateQBittorrentStep(ctx context.Context, data map[string]interface{}) error {
	url, _ := data["qbtUrl"].(string)
	if url == "" {
		// Skip is allowed
		return nil
	}
	// Basic URL validation
	if len(url) < 7 {
		return fmt.Errorf("invalid qBittorrent URL")
	}
	return nil
}

func (s *SetupService) validateMediaFolderStep(data map[string]interface{}) error {
	path, ok := data["mediaFolderPath"].(string)
	if !ok || path == "" {
		return fmt.Errorf("media folder path is required")
	}
	// Check if the path exists
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("media folder path does not exist: %s", path)
	}
	if !info.IsDir() {
		return fmt.Errorf("media folder path is not a directory: %s", path)
	}
	return nil
}

func (s *SetupService) validateApiKeysStep(data map[string]interface{}) error {
	tmdbKey, _ := data["tmdbApiKey"].(string)
	if tmdbKey != "" {
		// TMDb API keys are 32 character hex strings
		if len(tmdbKey) < 16 {
			return fmt.Errorf("invalid TMDb API key format")
		}
	}
	// API keys are optional - skip is allowed
	return nil
}
