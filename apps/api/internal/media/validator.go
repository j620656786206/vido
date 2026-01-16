package media

import (
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

// ValidateDirectory checks if a path is a valid, accessible directory.
// It returns a MediaDirectory struct with the validation status and any error message.
func ValidateDirectory(path string) MediaDirectory {
	dir := MediaDirectory{
		Path: path,
		Type: InferMediaType(path),
	}

	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			dir.Status = StatusNotFound
			dir.Error = "directory does not exist"
			slog.Warn("Media directory not found",
				"path", path,
				"recommendation", "Check if the path is correctly mounted in Docker")
		} else if os.IsPermission(err) {
			dir.Status = StatusNotReadable
			dir.Error = "permission denied"
			slog.Warn("Media directory permission denied",
				"path", path,
				"error", err)
		} else {
			dir.Status = StatusNotReadable
			dir.Error = err.Error()
			slog.Warn("Media directory not accessible",
				"path", path,
				"error", err)
		}
		return dir
	}

	if !info.IsDir() {
		dir.Status = StatusNotDir
		dir.Error = "path is not a directory"
		slog.Warn("Media path is not a directory", "path", path)
		return dir
	}

	// Check readability by attempting to list contents
	entries, err := os.ReadDir(path)
	if err != nil {
		dir.Status = StatusNotReadable
		dir.Error = "cannot read directory contents"
		slog.Warn("Cannot read media directory",
			"path", path,
			"error", err)
		return dir
	}

	dir.Status = StatusAccessible
	dir.FileCount = len(entries)
	slog.Info("Media directory validated",
		"path", path,
		"type", dir.Type,
		"file_count", dir.FileCount)

	return dir
}

// InferMediaType guesses the media type from the directory path name.
// Returns "movies", "tv", "anime", or "mixed" based on path patterns.
func InferMediaType(path string) string {
	base := strings.ToLower(filepath.Base(path))

	switch {
	case strings.Contains(base, "movie"):
		return "movies"
	case strings.Contains(base, "tv") || strings.Contains(base, "series") || strings.Contains(base, "show"):
		return "tv"
	case strings.Contains(base, "anime"):
		return "anime"
	default:
		return "mixed"
	}
}
