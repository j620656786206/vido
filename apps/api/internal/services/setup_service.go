package services

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
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
	libraryService MediaLibraryServiceInterface
	keyWriter      SetupKeyWriter
	// mediaRoots are the folders mounted into the container (VIDO_MEDIA_DIRS).
	mediaRoots []string
	// usageReport applies the wizard's opt-in answer (a2). Optional.
	usageReport UsageReportServiceInterface
}

// SetUsageReport wires the opt-in anonymous usage report, so the wizard's
// answer is stored through the same path as the settings page.
func (s *SetupService) SetUsageReport(svc UsageReportServiceInterface) {
	s.usageReport = svc
}

// SetupKeyWriter is the slice of KeySettingsService the wizard stores API keys
// through, so the wizard and the settings page share ONE storage path: the same
// secret names KeyResolver reads, and the same ENCRYPTION_KEY gate.
type SetupKeyWriter interface {
	Writable() bool
	Save(ctx context.Context, updates map[KeyName]string) error
}

// NewSetupService creates a new SetupService.
func NewSetupService(settingsRepo repository.SettingsRepositoryInterface, secretsSvc secrets.SecretsServiceInterface) *SetupService {
	return &SetupService{
		settingsRepo:   settingsRepo,
		secretsService: secretsSvc,
	}
}

// SetMediaRoots records the folders mounted into the container
// (VIDO_MEDIA_DIRS) so a wrong wizard path can be answered with the folders
// Vido CAN see (disc-setup-wizard-container-path-hint).
func (s *SetupService) SetMediaRoots(roots []string) {
	s.mediaRoots = roots
}

// SetLibraryService sets the media library service for creating libraries during setup.
func (s *SetupService) SetLibraryService(libraryService MediaLibraryServiceInterface) {
	s.libraryService = libraryService
}

// SetKeyWriter wires API-key storage (dsr-13). Without one the wizard refuses
// keys instead of dropping them.
func (s *SetupService) SetKeyWriter(keyWriter SetupKeyWriter) {
	s.keyWriter = keyWriter
}

func (s *SetupService) keysWritable() bool {
	return s.keyWriter != nil && s.keyWriter.Writable()
}

// setupKeyUpdates collects the keys the wizard was given. Blank values are left
// out rather than sent as "": to KeySettingsService an empty string means DELETE.
func setupKeyUpdates(tmdbKey, claudeKey string) map[KeyName]string {
	updates := map[KeyName]string{}
	if strings.TrimSpace(tmdbKey) != "" {
		updates[KeyTMDb] = tmdbKey
	}
	if strings.TrimSpace(claudeKey) != "" {
		updates[KeyClaude] = claudeKey
	}
	return updates
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

	// Refuse keys BEFORE writing anything: failing after the libraries were
	// created would make the user's retry create them a second time.
	keyUpdates := setupKeyUpdates(config.TMDbApiKey, config.ClaudeApiKey)
	if len(keyUpdates) > 0 && !s.keysWritable() {
		return ErrKeysNotWritable
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

	// Save media libraries (Story 7b-3: multi-library support)
	if len(config.Libraries) > 0 && s.libraryService != nil {
		for _, entry := range config.Libraries {
			name := entry.Path
			// Derive name from last path component
			parts := strings.Split(strings.TrimRight(entry.Path, "/"), "/")
			if len(parts) > 0 {
				name = parts[len(parts)-1]
			}
			contentType := entry.ContentType
			if contentType == "" {
				contentType = "movie"
			}
			_, err := s.libraryService.CreateLibrary(ctx, CreateLibraryRequest{
				Name:        name,
				ContentType: contentType,
				Paths:       []string{entry.Path},
			})
			if err != nil {
				return fmt.Errorf("create library %q: %w", entry.Path, err)
			}
		}
	} else if config.MediaFolderPath != "" {
		// Backward compatibility: single path (deprecated)
		if err := s.settingsRepo.SetString(ctx, "media_folder_path", config.MediaFolderPath); err != nil {
			return fmt.Errorf("save media_folder_path: %w", err)
		}
	}

	// API keys go through KeySettingsService — the settings page's own path — so
	// they land under the names KeyResolver reads and obey the same
	// ENCRYPTION_KEY gate. Before dsr-13 the wizard wrote "tmdb_api_key" /
	// "ai_api_key" / "ai_provider", names nothing ever read, so a key typed here
	// was silently discarded while the summary said 已設定. Claude is the only AI
	// key collected: the only text-AI key the resolver reads back from the secret
	// store (Gemini is env-only).
	if len(keyUpdates) > 0 {
		if err := s.keyWriter.Save(ctx, keyUpdates); err != nil {
			return fmt.Errorf("save api keys: %w", err)
		}
	}

	// Opt-in anonymous usage report (P1-040-2). Only a YES is written: "no"
	// is the default and needs no row. A failure here leaves the report OFF —
	// the privacy-safe direction — rather than failing a setup whose libraries
	// are already created (a retry would create them twice).
	if config.UsageReportEnabled && s.usageReport != nil {
		if _, err := s.usageReport.SetEnabled(ctx, true); err != nil {
			slog.Warn("Setup wizard: usage report opt-in not saved — left off", "error", err)
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
		"has_claude_key", config.ClaudeApiKey != "",
		"usage_report_opt_in", config.UsageReportEnabled,
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
	case "usage-report":
		// A yes/no question with a safe default — any answer is valid.
		return nil
	case "complete":
		return nil
	default:
		return fmt.Errorf("unknown step: %s", step)
	}
}

func (s *SetupService) validateWelcomeStep(data map[string]interface{}) error {
	lang, ok := data["language"].(string)
	if !ok || lang == "" {
		return errors.New("請選擇語言。")
	}
	return nil
}

func (s *SetupService) validateQBittorrentStep(ctx context.Context, data map[string]interface{}) error {
	url, _ := data["qbt_url"].(string)
	if url == "" {
		// Skip is allowed
		return nil
	}
	// Basic URL validation
	if len(url) < 7 {
		return errors.New("qBittorrent 網址看起來不對，請填完整網址，例如 http://192.168.1.10:8080。")
	}
	return nil
}

func (s *SetupService) validateMediaFolderStep(data map[string]interface{}) error {
	// Try new multi-library format first
	if libraries, ok := data["libraries"].([]interface{}); ok && len(libraries) > 0 {
		for i, lib := range libraries {
			libMap, ok := lib.(map[string]interface{})
			if !ok {
				return fmt.Errorf("第 %d 個資料夾的資料不完整，請重新填寫。", i+1)
			}
			path, _ := libMap["path"].(string)
			if err := s.checkWizardFolder(path); err != nil {
				return err
			}
			contentType, _ := libMap["content_type"].(string)
			if contentType != "" && contentType != "movie" && contentType != "series" {
				return errors.New("資料夾的類型只能是電影或影集。")
			}
		}
		return nil
	}

	// Backward compatibility: single path
	path, _ := data["media_folder_path"].(string)
	return s.checkWizardFolder(path)
}

// maxFolderSuggestions caps the example paths in a not-found message.
const maxFolderSuggestions = 4

// checkWizardFolder validates one wizard folder path and answers in zh-TW.
// A path the container cannot see is the common first-run mistake: the user
// types the NAS's own path (/video/Movies) while Vido runs in Docker and sees
// only what is mounted (VIDO_MEDIA_DIRS, /media by default) — so the message
// names folders that DO exist in the container (disc-setup-wizard-container-path-hint).
func (s *SetupService) checkWizardFolder(path string) error {
	if strings.TrimSpace(path) == "" {
		return errors.New("還有資料夾沒填路徑。")
	}
	info, err := os.Stat(path)
	if err != nil {
		// Present but unreadable is a PUID/permission problem, not a mount
		// problem — don't send the user off to edit volumes.
		if errors.Is(err, fs.ErrPermission) {
			return fmt.Errorf("沒有權限讀取「%s」：Vido 所在容器的使用者（PUID／PGID）讀不到這個資料夾。", path)
		}
		return s.folderNotFoundError(path)
	}
	if !info.IsDir() {
		return fmt.Errorf("「%s」是檔案，不是資料夾。", path)
	}
	return nil
}

func (s *SetupService) folderNotFoundError(path string) error {
	const lead = "找不到「%s」。Vido 在 Docker 裡，只看得到掛進容器的資料夾"
	if examples := s.containerFolderExamples(); len(examples) > 0 {
		return fmt.Errorf(lead+"——請填容器裡的路徑，例如 %s。", path, strings.Join(examples, "、"))
	}
	root := "/media"
	if len(s.mediaRoots) > 0 {
		root = s.mediaRoots[0]
	}
	return fmt.Errorf(lead+"，但目前容器裡沒有 %s——請在 docker-compose 的 volumes 把媒體資料夾掛到 %s。", path, root, root)
}

// containerFolderExamples lists folders that exist in the container: the
// media roots first (in configured order — with VIDO_MEDIA_DIRS=/media/movies,
// /media/tv the roots ARE the library folders), then their direct subfolders
// (sorted; hidden and Synology system folders like @eaDir skipped; no
// recursion — a NAS share can be huge), capped at maxFolderSuggestions.
func (s *SetupService) containerFolderExamples() []string {
	var subfolders, roots []string
	seen := map[string]bool{}
	for _, root := range s.mediaRoots {
		root = filepath.Clean(root)
		if seen[root] {
			continue
		}
		info, err := os.Stat(root)
		if err != nil || !info.IsDir() {
			continue
		}
		seen[root] = true
		roots = append(roots, root)
		entries, err := os.ReadDir(root)
		if err != nil {
			continue
		}
		for _, e := range entries {
			name := e.Name()
			if !e.IsDir() || strings.HasPrefix(name, ".") || strings.HasPrefix(name, "@") || strings.HasPrefix(name, "#") {
				continue
			}
			subfolders = append(subfolders, filepath.Join(root, name))
		}
	}
	sort.Strings(subfolders)
	examples := roots
	for _, sub := range subfolders {
		// A root nested in another root (VIDO_MEDIA_DIRS=/media,/media/movies)
		// is already listed.
		if !seen[sub] {
			seen[sub] = true
			examples = append(examples, sub)
		}
	}
	if len(examples) > maxFolderSuggestions {
		examples = examples[:maxFolderSuggestions]
	}
	return examples
}

func (s *SetupService) validateApiKeysStep(data map[string]interface{}) error {
	tmdbKey, _ := data["tmdb_api_key"].(string)
	claudeKey, _ := data["claude_api_key"].(string)
	if tmdbKey != "" {
		// TMDb API keys are 32 character hex strings
		if len(tmdbKey) < 16 {
			return errors.New("TMDb 金鑰格式不對，請確認有完整貼上。")
		}
	}
	// Say it on the step where the key was typed, not after 完成設定.
	if len(setupKeyUpdates(tmdbKey, claudeKey)) > 0 && !s.keysWritable() {
		return ErrKeysNotWritable
	}
	// API keys are optional - skip is allowed
	return nil
}
