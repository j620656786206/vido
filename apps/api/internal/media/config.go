package media

import (
	"log/slog"
	"os"
	"strings"
)

// MediaDirectoryStatus indicates the accessibility status of a media directory
type MediaDirectoryStatus string

const (
	// StatusAccessible indicates the directory is accessible and readable
	StatusAccessible MediaDirectoryStatus = "accessible"
	// StatusNotFound indicates the directory does not exist
	StatusNotFound MediaDirectoryStatus = "not_found"
	// StatusNotDir indicates the path exists but is not a directory
	StatusNotDir MediaDirectoryStatus = "not_directory"
	// StatusNotReadable indicates the directory exists but cannot be read
	StatusNotReadable MediaDirectoryStatus = "not_readable"
)

// MediaDirectory represents a single configured media directory with its status
type MediaDirectory struct {
	Path      string               `json:"path"`
	Type      string               `json:"type,omitempty"` // movies, tv, anime, mixed
	Status    MediaDirectoryStatus `json:"status"`
	FileCount int                  `json:"file_count,omitempty"`
	Error     string               `json:"error,omitempty"`
}

// MediaConfig holds the configuration and status of all media directories
type MediaConfig struct {
	Directories    []MediaDirectory `json:"directories"`
	ValidCount     int              `json:"valid_count"`
	TotalCount     int              `json:"total_count"`
	SearchOnlyMode bool             `json:"search_only_mode"`
}

// LoadMediaConfig loads and validates media directories from VIDO_MEDIA_DIRS environment variable.
// It returns a MediaConfig with all configured directories and their validation status.
// If no directories are configured, SearchOnlyMode is set to true.
func LoadMediaConfig() *MediaConfig {
	rawDirs := os.Getenv("VIDO_MEDIA_DIRS")
	if rawDirs == "" {
		slog.Info("No media directories configured, running in search-only mode",
			"recommendation", "Set VIDO_MEDIA_DIRS to enable library features")
		return &MediaConfig{
			Directories:    []MediaDirectory{},
			SearchOnlyMode: true,
		}
	}

	paths := strings.Split(rawDirs, ",")
	config := &MediaConfig{
		Directories: make([]MediaDirectory, 0, len(paths)),
		TotalCount:  0,
	}

	for _, p := range paths {
		path := strings.TrimSpace(p)
		if path == "" {
			continue
		}

		config.TotalCount++
		dir := ValidateDirectory(path)
		config.Directories = append(config.Directories, dir)

		if dir.Status == StatusAccessible {
			config.ValidCount++
		}
	}

	config.SearchOnlyMode = config.ValidCount == 0
	return config
}

// LogMediaConfigStatus logs the status of all configured media directories.
// It logs warnings for inaccessible directories and an info message for the overall status.
func LogMediaConfigStatus(config *MediaConfig) {
	if config.SearchOnlyMode {
		slog.Info("Running in search-only mode",
			"reason", "no accessible media directories configured",
			"recommendation", "Set VIDO_MEDIA_DIRS to enable library features")
		return
	}

	slog.Info("Media directories loaded",
		"total", config.TotalCount,
		"valid", config.ValidCount,
		"search_only_mode", config.SearchOnlyMode)

	for _, dir := range config.Directories {
		if dir.Status == StatusAccessible {
			slog.Info("Media directory accessible",
				"path", dir.Path,
				"type", dir.Type,
				"file_count", dir.FileCount)
		} else {
			slog.Warn("Media directory unavailable",
				"path", dir.Path,
				"status", dir.Status,
				"error", dir.Error)
		}
	}
}

// GetAccessibleDirectories returns only the directories that are accessible
func (c *MediaConfig) GetAccessibleDirectories() []MediaDirectory {
	accessible := make([]MediaDirectory, 0, c.ValidCount)
	for _, dir := range c.Directories {
		if dir.Status == StatusAccessible {
			accessible = append(accessible, dir)
		}
	}
	return accessible
}
