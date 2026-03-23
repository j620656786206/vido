package subtitle

import (
	"fmt"
	"log/slog"
	"math/rand"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Supported subtitle output formats.
var supportedFormats = map[string]bool{
	"srt": true,
	"ass": true,
}

// PlacerConfig controls subtitle file placement behavior.
type PlacerConfig struct {
	// BackupExisting controls whether existing subtitles are backed up (.bak)
	// before overwriting. Default: true.
	BackupExisting bool
}

// DefaultPlacerConfig returns the default placer configuration.
func DefaultPlacerConfig() PlacerConfig {
	return PlacerConfig{BackupExisting: true}
}

// Placer handles subtitle file placement alongside media files.
// It performs pure file operations with no database dependency.
type Placer struct {
	config PlacerConfig
}

// NewPlacer creates a subtitle file placer with the given config.
func NewPlacer(config PlacerConfig) *Placer {
	return &Placer{config: config}
}

// PlaceRequest contains the parameters for placing a subtitle file.
type PlaceRequest struct {
	// MediaFilePath is the absolute path to the media file.
	MediaFilePath string

	// SubtitleData is the raw subtitle content bytes.
	SubtitleData []byte

	// Language is the detected language tag (e.g., "zh-Hant", "zh-Hans").
	Language string

	// Format is a hint for the subtitle format (e.g., "srt", "ass").
	// If empty, format is detected from content.
	Format string

	// Score is the subtitle scoring result (stored in DB).
	Score float64
}

// PlaceResult contains the outcome of a subtitle placement operation.
type PlaceResult struct {
	// SubtitlePath is the absolute path where the subtitle was saved.
	SubtitlePath string

	// Language is the normalized BCP 47 language tag used in the filename.
	Language string

	// BackupPath is the path of the backup file, if one was created. Empty otherwise.
	BackupPath string
}

// Place writes a subtitle file next to its media file with a standardized name.
//
// The naming convention follows IETF BCP 47:
//
//	Movie.2024.1080p.mkv → Movie.2024.1080p.zh-Hant.srt
func (p *Placer) Place(req PlaceRequest) (*PlaceResult, error) {
	// Clean the media file path to prevent path traversal (e.g., /../../../etc/Movie.mkv)
	cleanPath := filepath.Clean(req.MediaFilePath)
	if !filepath.IsAbs(cleanPath) {
		return nil, fmt.Errorf("placer: media file path must be absolute: %s", req.MediaFilePath)
	}
	req.MediaFilePath = cleanPath

	// Validate media file directory exists
	mediaDir := filepath.Dir(req.MediaFilePath)
	if _, err := os.Stat(mediaDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("placer: media directory does not exist: %s", mediaDir)
	}

	// Detect or validate subtitle format
	format, err := detectFormat(req.SubtitleData, req.Format)
	if err != nil {
		return nil, fmt.Errorf("placer: %w", err)
	}

	// Normalize language tag
	langTag := normalizeLanguageTag(req.Language)

	// Build target filename
	targetPath := buildSubtitleFilename(req.MediaFilePath, langTag, format)

	// Backup existing subtitle if present
	var backupPath string
	if p.config.BackupExisting {
		bp, err := backupExistingFile(targetPath)
		if err != nil {
			return nil, fmt.Errorf("placer: backup failed: %w", err)
		}
		backupPath = bp
	}

	// Write subtitle file atomically
	if err := writeFileAtomic(targetPath, req.SubtitleData, 0644); err != nil {
		return nil, fmt.Errorf("placer: write failed: %w", err)
	}

	slog.Info("Subtitle placed successfully",
		"path", targetPath,
		"language", langTag,
		"format", format,
		"size", len(req.SubtitleData),
	)

	return &PlaceResult{
		SubtitlePath: targetPath,
		Language:     langTag,
		BackupPath:   backupPath,
	}, nil
}

// normalizeLanguageTag maps various language tag formats to IETF BCP 47.
func normalizeLanguageTag(lang string) string {
	lower := strings.ToLower(lang)
	switch lower {
	case "zh-hant", "zh-tw", "cht", "繁體", "繁体":
		return "zh-Hant"
	case "zh-hans", "zh-cn", "chs", "簡體", "简体":
		return "zh-Hans"
	case "zh":
		return "zh"
	case "en":
		return "en"
	default:
		if lang != "" {
			// Sanitize: only allow alphanumeric, hyphens, and underscores
			// to prevent path traversal via crafted language tags
			if !safeTagPattern.MatchString(lang) {
				return "und"
			}
			return lang
		}
		return "und"
	}
}

// safeTagPattern matches valid BCP 47-like language tags (alphanumeric + hyphens).
var safeTagPattern = regexp.MustCompile(`^[a-zA-Z0-9\-]+$`)

// buildSubtitleFilename creates the subtitle path from media path, language, and format.
// Example: /media/Movie.2024.1080p.mkv → /media/Movie.2024.1080p.zh-Hant.srt
func buildSubtitleFilename(mediaPath, langTag, subtitleExt string) string {
	dir := filepath.Dir(mediaPath)
	base := filepath.Base(mediaPath)

	// Strip media extension
	ext := filepath.Ext(base)
	nameWithoutExt := strings.TrimSuffix(base, ext)

	// Build: {name}.{langTag}.{ext}
	subtitleName := fmt.Sprintf("%s.%s.%s", nameWithoutExt, langTag, subtitleExt)
	return filepath.Join(dir, subtitleName)
}

// detectFormat determines the subtitle format from content or hint.
func detectFormat(data []byte, hintFormat string) (string, error) {
	// Use hint if provided and supported
	if hintFormat != "" {
		hint := strings.ToLower(strings.TrimPrefix(hintFormat, "."))
		if supportedFormats[hint] {
			return hint, nil
		}
		return "", fmt.Errorf("unsupported subtitle format: %q", hintFormat)
	}

	// Content-based detection
	content := string(data[:min(len(data), 500)])

	// ASS format: starts with [Script Info]
	if strings.Contains(content, "[Script Info]") || strings.Contains(content, "[V4+ Styles]") {
		return "ass", nil
	}

	// SRT format: digit followed by timestamp pattern (00:00:00,000 --> 00:00:00,000)
	if strings.Contains(content, " --> ") {
		return "srt", nil
	}

	return "", fmt.Errorf("unable to detect subtitle format from content")
}

// writeFileAtomic writes data to a file atomically using temp+rename pattern.
func writeFileAtomic(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	tmpPath := filepath.Join(dir, fmt.Sprintf(".%s.tmp.%d", filepath.Base(path), rand.Int63()))

	if err := os.WriteFile(tmpPath, data, perm); err != nil {
		// Clean up on failure
		os.Remove(tmpPath)
		return fmt.Errorf("write temp file: %w", err)
	}

	if err := os.Rename(tmpPath, path); err != nil {
		// Clean up on failure
		os.Remove(tmpPath)
		return fmt.Errorf("rename temp to target: %w", err)
	}

	return nil
}

// backupExistingFile renames an existing file to .bak if it exists.
// Returns the backup path if a backup was created, empty string otherwise.
func backupExistingFile(path string) (string, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return "", nil // No existing file — nothing to backup
	}

	backupPath := path + ".bak"

	// Guard: if .bak target exists as a directory, refuse to overwrite
	if info, err := os.Stat(backupPath); err == nil && info.IsDir() {
		return "", fmt.Errorf("backup target is a directory: %s", backupPath)
	}

	if err := os.Rename(path, backupPath); err != nil {
		return "", fmt.Errorf("rename to backup: %w", err)
	}

	slog.Info("Existing subtitle backed up", "backup", backupPath)
	return backupPath, nil
}

// Cleanup removes a subtitle file and its backup from disk.
// Ignores missing files (idempotent).
func Cleanup(subtitlePath string) error {
	if subtitlePath == "" {
		return nil
	}

	// Remove subtitle file
	if err := os.Remove(subtitlePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove subtitle: %w", err)
	}

	// Remove backup if exists
	bakPath := subtitlePath + ".bak"
	if err := os.Remove(bakPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove backup: %w", err)
	}

	slog.Info("Subtitle files cleaned up", "path", subtitlePath)
	return nil
}
