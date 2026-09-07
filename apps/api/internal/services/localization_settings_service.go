package services

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/vido/api/internal/ai/prompts"
	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/repository"
)

// SettingKeyLocalizationLevel is the settings-table key for the sub-7-4
// localization dial. Precedence: settings table > SUBTITLE_LOCALIZATION_LEVEL
// env > "standard" — the API-key precedence, so a value saved from the
// settings page wins and applies to the next run without a restart.
const SettingKeyLocalizationLevel = "subtitle.localization_level"

// LocalizationSource names where the effective level came from.
type LocalizationSource string

const (
	LocalizationSourceSettings LocalizationSource = "settings"
	LocalizationSourceEnv      LocalizationSource = "env"
	LocalizationSourceDefault  LocalizationSource = "default"
)

// LocalizationSettings is the resolved dial: the level in force and why.
type LocalizationSettings struct {
	Level  prompts.LocalizationLevel `json:"level"`
	Source LocalizationSource        `json:"source"`
}

// LocalizationSettingsService reads and writes the localization level. It is
// NOT cached: the level rides every run's PromptVersion, and a stale answer
// would key a cache entry under the wrong taste. One primary-key read per
// item is the price.
type LocalizationSettingsService struct {
	repo   repository.SettingsRepositoryInterface
	env    string // the boot-time env value ("" = unset)
	logger *slog.Logger
}

// NewLocalizationSettingsService wires the service. env is the validated
// SUBTITLE_LOCALIZATION_LEVEL value from config ("" when unset).
func NewLocalizationSettingsService(repo repository.SettingsRepositoryInterface, env string, logger *slog.Logger) *LocalizationSettingsService {
	if logger == nil {
		logger = slog.Default()
	}
	return &LocalizationSettingsService{repo: repo, env: strings.TrimSpace(env), logger: logger.With("service", "localization_settings")}
}

// Resolve returns the effective level and its source.
func (s *LocalizationSettingsService) Resolve(ctx context.Context) LocalizationSettings {
	if s.repo != nil {
		raw, err := s.repo.GetString(ctx, SettingKeyLocalizationLevel)
		switch {
		case err == nil && strings.TrimSpace(raw) != "":
			level, perr := prompts.ParseLocalizationLevel(raw)
			if perr == nil {
				return LocalizationSettings{Level: level, Source: LocalizationSourceSettings}
			}
			s.logger.Warn("stored localization level is invalid; falling back", "value", raw, "error", perr)
		case err != nil && !isSettingNotFound(err):
			s.logger.Warn("localization level read failed; falling back", "error", err)
		}
	}
	if s.env != "" {
		if level, err := prompts.ParseLocalizationLevel(s.env); err == nil {
			return LocalizationSettings{Level: level, Source: LocalizationSourceEnv}
		}
	}
	return LocalizationSettings{Level: prompts.DefaultLocalizationLevel, Source: LocalizationSourceDefault}
}

// Level is the pipeline-facing accessor (the func the translation legs take).
func (s *LocalizationSettingsService) Level(ctx context.Context) prompts.LocalizationLevel {
	return s.Resolve(ctx).Level
}

// Set validates and stores the level. An unknown value is rejected — the
// user asked for a taste and must not silently get another.
func (s *LocalizationSettingsService) Set(ctx context.Context, raw string) (LocalizationSettings, error) {
	level, err := prompts.ParseLocalizationLevel(raw)
	if err != nil {
		return LocalizationSettings{}, &models.ValidationError{Field: "level", Message: err.Error()}
	}
	if s.repo == nil {
		return LocalizationSettings{}, fmt.Errorf("settings repository not configured")
	}
	if err := s.repo.Set(ctx, &models.Setting{Key: SettingKeyLocalizationLevel, Value: string(level), Type: "string"}); err != nil {
		return LocalizationSettings{}, fmt.Errorf("save localization level: %w", err)
	}
	s.logger.Info("localization level saved", "level", level)
	return LocalizationSettings{Level: level, Source: LocalizationSourceSettings}, nil
}

// isSettingNotFound recognises the settings repository's not-found error: it
// is a plain fmt.Errorf("setting with key %s not found") without %w
// (settings_repository.go), so the wording is the only signal.
func isSettingNotFound(err error) bool {
	return err != nil && strings.Contains(strings.ToLower(err.Error()), "not found")
}
