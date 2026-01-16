package media

import (
	"bytes"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadMediaConfig_ValidDirectories(t *testing.T) {
	// Create temp directories
	dir1 := t.TempDir()
	dir2 := t.TempDir()

	// Create some files in dir1 to test file count
	f1, err := os.Create(filepath.Join(dir1, "movie1.mkv"))
	require.NoError(t, err)
	f1.Close()
	f2, err := os.Create(filepath.Join(dir1, "movie2.mkv"))
	require.NoError(t, err)
	f2.Close()

	config := LoadMediaConfig([]string{dir1, dir2})

	assert.Equal(t, 2, config.TotalCount)
	assert.Equal(t, 2, config.ValidCount)
	assert.False(t, config.SearchOnlyMode)
	assert.Len(t, config.Directories, 2)

	// First directory should have 2 files
	assert.Equal(t, StatusAccessible, config.Directories[0].Status)
	assert.Equal(t, 2, config.Directories[0].FileCount)
}

func TestLoadMediaConfig_MixedValidity(t *testing.T) {
	validDir := t.TempDir()
	invalidDir := "/nonexistent/path/for/testing"

	config := LoadMediaConfig([]string{validDir, invalidDir})

	assert.Equal(t, 2, config.TotalCount)
	assert.Equal(t, 1, config.ValidCount)
	assert.False(t, config.SearchOnlyMode, "Should not be search-only mode with 1 valid dir")

	// Check statuses
	assert.Equal(t, StatusAccessible, config.Directories[0].Status)
	assert.Equal(t, StatusNotFound, config.Directories[1].Status)
	assert.NotEmpty(t, config.Directories[1].Error)
}

func TestLoadMediaConfig_NoDirectories(t *testing.T) {
	config := LoadMediaConfig([]string{})

	assert.Equal(t, 0, config.TotalCount)
	assert.Equal(t, 0, config.ValidCount)
	assert.True(t, config.SearchOnlyMode)
	assert.Len(t, config.Directories, 0)
}

func TestLoadMediaConfig_NilDirectories(t *testing.T) {
	config := LoadMediaConfig(nil)

	assert.Equal(t, 0, config.TotalCount)
	assert.Equal(t, 0, config.ValidCount)
	assert.True(t, config.SearchOnlyMode)
	assert.Len(t, config.Directories, 0)
}

func TestLoadMediaConfig_AllInvalidDirectories(t *testing.T) {
	config := LoadMediaConfig([]string{"/nonexistent/path1", "/nonexistent/path2"})

	assert.Equal(t, 2, config.TotalCount)
	assert.Equal(t, 0, config.ValidCount)
	assert.True(t, config.SearchOnlyMode, "Should be search-only mode with no valid dirs")
}

func TestLoadMediaConfig_EmptyStringsFiltered(t *testing.T) {
	dir1 := t.TempDir()
	dir2 := t.TempDir()

	// Test with empty strings in the slice
	config := LoadMediaConfig([]string{dir1, "", dir2, ""})

	// Empty entries should be filtered out
	assert.Equal(t, 2, config.TotalCount)
	assert.Equal(t, 2, config.ValidCount)
}

func TestLoadMediaConfig_PathSanitization(t *testing.T) {
	// Create a temp directory
	dir := t.TempDir()

	// Test that paths are sanitized with filepath.Clean
	// Pass path with extra slashes and dot segments
	dirtyPath := dir + "/./subdir/../"
	config := LoadMediaConfig([]string{dirtyPath})

	assert.Equal(t, 1, config.TotalCount)
	assert.Equal(t, 1, config.ValidCount)
	// Path should be cleaned
	assert.Equal(t, filepath.Clean(dirtyPath), config.Directories[0].Path)
}

func TestMediaConfig_GetAccessibleDirectories(t *testing.T) {
	validDir := t.TempDir()
	invalidDir := "/nonexistent/path/for/testing"

	config := LoadMediaConfig([]string{validDir, invalidDir})
	accessible := config.GetAccessibleDirectories()

	assert.Len(t, accessible, 1)
	assert.Equal(t, validDir, accessible[0].Path)
	assert.Equal(t, StatusAccessible, accessible[0].Status)
}

func TestInferMediaType(t *testing.T) {
	tests := []struct {
		path     string
		expected string
	}{
		{"/media/movies", "movies"},
		{"/data/Movies", "movies"},
		{"/mnt/movie-collection", "movies"},
		{"/tv-shows", "tv"},
		{"/TV", "tv"},
		{"/series", "tv"},
		{"/my-series", "tv"},
		{"/tv-series", "tv"},
		{"/shows", "tv"},
		{"/anime", "anime"},
		{"/Anime", "anime"},
		{"/downloads", "mixed"},
		{"/media", "mixed"},
		{"/data", "mixed"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := InferMediaType(tt.path)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestValidateDirectory_ValidDirectory(t *testing.T) {
	dir := t.TempDir()

	// Create a file to test file count
	f, err := os.Create(filepath.Join(dir, "test.txt"))
	require.NoError(t, err)
	f.Close()

	result := ValidateDirectory(dir)

	assert.Equal(t, dir, result.Path)
	assert.Equal(t, StatusAccessible, result.Status)
	assert.Equal(t, 1, result.FileCount)
	assert.Empty(t, result.Error)
}

func TestValidateDirectory_NotFound(t *testing.T) {
	result := ValidateDirectory("/nonexistent/path/for/testing")

	assert.Equal(t, StatusNotFound, result.Status)
	assert.Equal(t, "directory does not exist", result.Error)
}

func TestValidateDirectory_NotDirectory(t *testing.T) {
	// Create a temp file (not a directory)
	f, err := os.CreateTemp("", "test-file")
	require.NoError(t, err)
	defer os.Remove(f.Name())
	f.Close()

	result := ValidateDirectory(f.Name())

	assert.Equal(t, StatusNotDir, result.Status)
	assert.Equal(t, "path is not a directory", result.Error)
}

func TestValidateDirectory_EmptyDirectory(t *testing.T) {
	dir := t.TempDir()

	result := ValidateDirectory(dir)

	assert.Equal(t, StatusAccessible, result.Status)
	assert.Equal(t, 0, result.FileCount)
}

func TestMediaConfig_GracefulDegradation(t *testing.T) {
	// Test that the application continues with partial valid paths
	validDir := t.TempDir()

	// Mix of valid and invalid paths
	config := LoadMediaConfig([]string{validDir, "/invalid1", "/invalid2"})

	// Should have 1 valid directory out of 3 total
	assert.Equal(t, 3, config.TotalCount)
	assert.Equal(t, 1, config.ValidCount)
	assert.False(t, config.SearchOnlyMode, "Should continue with valid paths")

	// Accessible directories should only contain the valid one
	accessible := config.GetAccessibleDirectories()
	assert.Len(t, accessible, 1)
	assert.Equal(t, validDir, accessible[0].Path)
}

// Issue #3 fix: Test LogMediaConfigStatus
func TestLogMediaConfigStatus_SearchOnlyMode(t *testing.T) {
	// Capture log output
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))
	slog.SetDefault(logger)

	config := &MediaConfig{
		Directories:    []MediaDirectory{},
		ValidCount:     0,
		TotalCount:     0,
		SearchOnlyMode: true,
	}

	LogMediaConfigStatus(config)

	logOutput := buf.String()
	assert.Contains(t, logOutput, "search-only mode")
	assert.Contains(t, logOutput, "no accessible media directories configured")
}

func TestLogMediaConfigStatus_WithAccessibleDirectories(t *testing.T) {
	// Capture log output
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))
	slog.SetDefault(logger)

	config := &MediaConfig{
		Directories: []MediaDirectory{
			{
				Path:      "/media/movies",
				Type:      "movies",
				Status:    StatusAccessible,
				FileCount: 100,
			},
		},
		ValidCount:     1,
		TotalCount:     1,
		SearchOnlyMode: false,
	}

	LogMediaConfigStatus(config)

	logOutput := buf.String()
	assert.Contains(t, logOutput, "Media directories loaded")
	assert.Contains(t, logOutput, "Media directory accessible")
	assert.Contains(t, logOutput, "/media/movies")
}

func TestLogMediaConfigStatus_WithMixedDirectories(t *testing.T) {
	// Capture log output
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))
	slog.SetDefault(logger)

	config := &MediaConfig{
		Directories: []MediaDirectory{
			{
				Path:      "/media/movies",
				Type:      "movies",
				Status:    StatusAccessible,
				FileCount: 100,
			},
			{
				Path:   "/media/tv",
				Type:   "tv",
				Status: StatusNotFound,
				Error:  "directory does not exist",
			},
		},
		ValidCount:     1,
		TotalCount:     2,
		SearchOnlyMode: false,
	}

	LogMediaConfigStatus(config)

	logOutput := buf.String()
	assert.Contains(t, logOutput, "Media directories loaded")
	assert.Contains(t, logOutput, "Media directory accessible")
	assert.Contains(t, logOutput, "Media directory unavailable")
	assert.Contains(t, logOutput, "/media/tv")
}
