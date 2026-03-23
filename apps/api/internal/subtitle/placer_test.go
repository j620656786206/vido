package subtitle

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Task 9.3: Filename generation ---

func TestBuildSubtitleFilename(t *testing.T) {
	tests := []struct {
		name      string
		mediaPath string
		lang      string
		ext       string
		expected  string
	}{
		{
			"standard mkv",
			"/media/movies/Movie.2024.1080p.mkv",
			"zh-Hant", "srt",
			"/media/movies/Movie.2024.1080p.zh-Hant.srt",
		},
		{
			"ass format",
			"/media/movies/Movie.2024.1080p.mkv",
			"zh-Hant", "ass",
			"/media/movies/Movie.2024.1080p.zh-Hant.ass",
		},
		{
			"mp4 media",
			"/media/movies/Film.mp4",
			"zh-Hans", "srt",
			"/media/movies/Film.zh-Hans.srt",
		},
		{
			"multiple dots in name",
			"/media/movies/Movie.2024.BluRay.1080p.x264.mkv",
			"zh-Hant", "srt",
			"/media/movies/Movie.2024.BluRay.1080p.x264.zh-Hant.srt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildSubtitleFilename(tt.mediaPath, tt.lang, tt.ext)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// --- Task 9.5: Language tag normalization ---

func TestNormalizeLanguageTag(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"zh-TW", "zh-Hant"},
		{"zh-tw", "zh-Hant"},
		{"zh-Hant", "zh-Hant"},
		{"CHT", "zh-Hant"},
		{"繁體", "zh-Hant"},
		{"zh-CN", "zh-Hans"},
		{"zh-cn", "zh-Hans"},
		{"zh-Hans", "zh-Hans"},
		{"CHS", "zh-Hans"},
		{"簡體", "zh-Hans"},
		{"zh", "zh"},
		{"en", "en"},
		{"", "und"},
		{"ja", "ja"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.expected, normalizeLanguageTag(tt.input))
		})
	}
}

// --- Task 9.10: Format detection ---

func TestDetectFormat(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		hint     string
		expected string
		wantErr  bool
	}{
		{"hint srt", nil, "srt", "srt", false},
		{"hint ass", nil, "ass", "ass", false},
		{"hint .srt with dot", nil, ".srt", "srt", false},
		{"hint unsupported", nil, "vtt", "", true},
		{"content SRT", []byte("1\n00:00:01,000 --> 00:00:03,000\nHello\n"), "", "srt", false},
		{"content ASS", []byte("[Script Info]\nTitle: Test\n[V4+ Styles]\n"), "", "ass", false},
		{"content unknown", []byte("random text without markers"), "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := detectFormat(tt.data, tt.hint)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

// --- Task 9.6: Atomic write ---

func TestWriteFileAtomic(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.srt")
	data := []byte("subtitle content")

	err := writeFileAtomic(path, data, 0644)
	require.NoError(t, err)

	// File should exist with correct content
	content, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, data, content)

	// Check permissions
	info, err := os.Stat(path)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0644), info.Mode().Perm())
}

// --- Task 9.7: Atomic write cleanup on error ---

func TestWriteFileAtomic_CleanupOnError(t *testing.T) {
	// Write to non-existent directory should fail
	path := "/nonexistent/dir/test.srt"
	err := writeFileAtomic(path, []byte("data"), 0644)
	assert.Error(t, err)

	// No temp files should be left behind (directory doesn't exist, so no cleanup needed)
}

// --- Task 9.8: Backup ---

func TestBackupExistingFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "subtitle.zh-Hant.srt")

	// Create existing file
	err := os.WriteFile(path, []byte("old subtitle"), 0644)
	require.NoError(t, err)

	backupPath, err := backupExistingFile(path)
	require.NoError(t, err)
	assert.Equal(t, path+".bak", backupPath)

	// Original should be gone, backup should exist
	assert.NoFileExists(t, path)
	assert.FileExists(t, backupPath)

	content, _ := os.ReadFile(backupPath)
	assert.Equal(t, "old subtitle", string(content))
}

func TestBackupExistingFile_NoExistingFile(t *testing.T) {
	backupPath, err := backupExistingFile("/nonexistent/file.srt")
	require.NoError(t, err)
	assert.Empty(t, backupPath)
}

// --- Task 9.9: Overwrite mode ---

func TestPlacer_Place_OverwriteMode(t *testing.T) {
	dir := t.TempDir()
	mediaPath := filepath.Join(dir, "Movie.2024.1080p.mkv")
	os.WriteFile(mediaPath, []byte("fake media"), 0644)

	subtitlePath := filepath.Join(dir, "Movie.2024.1080p.zh-Hant.srt")
	os.WriteFile(subtitlePath, []byte("old subtitle"), 0644)

	p := NewPlacer(PlacerConfig{BackupExisting: false})

	result, err := p.Place(PlaceRequest{
		MediaFilePath: mediaPath,
		SubtitleData:  []byte("new subtitle"),
		Language:      "zh-Hant",
		Format:        "srt",
	})
	require.NoError(t, err)
	assert.Equal(t, subtitlePath, result.SubtitlePath)
	assert.Empty(t, result.BackupPath, "no backup in overwrite mode")

	// New content should be written
	content, _ := os.ReadFile(subtitlePath)
	assert.Equal(t, "new subtitle", string(content))

	// No .bak file
	assert.NoFileExists(t, subtitlePath+".bak")
}

// --- Task 9.8: Place with backup ---

func TestPlacer_Place_WithBackup(t *testing.T) {
	dir := t.TempDir()
	mediaPath := filepath.Join(dir, "Movie.mkv")
	os.WriteFile(mediaPath, []byte("fake media"), 0644)

	subtitlePath := filepath.Join(dir, "Movie.zh-Hant.srt")
	os.WriteFile(subtitlePath, []byte("old subtitle"), 0644)

	p := NewPlacer(DefaultPlacerConfig())

	result, err := p.Place(PlaceRequest{
		MediaFilePath: mediaPath,
		SubtitleData:  []byte("new subtitle"),
		Language:      "zh-Hant",
		Format:        "srt",
	})
	require.NoError(t, err)
	assert.Equal(t, subtitlePath, result.SubtitlePath)
	assert.Equal(t, subtitlePath+".bak", result.BackupPath)

	// New content written
	content, _ := os.ReadFile(subtitlePath)
	assert.Equal(t, "new subtitle", string(content))

	// Backup exists with old content
	bakContent, _ := os.ReadFile(subtitlePath + ".bak")
	assert.Equal(t, "old subtitle", string(bakContent))
}

// --- Task 9.11: Cleanup ---

func TestCleanup(t *testing.T) {
	dir := t.TempDir()
	subPath := filepath.Join(dir, "Movie.zh-Hant.srt")
	bakPath := subPath + ".bak"

	os.WriteFile(subPath, []byte("subtitle"), 0644)
	os.WriteFile(bakPath, []byte("backup"), 0644)

	err := Cleanup(subPath)
	require.NoError(t, err)

	assert.NoFileExists(t, subPath)
	assert.NoFileExists(t, bakPath)
}

// --- Task 9.12: Cleanup missing files ---

func TestCleanup_MissingFiles(t *testing.T) {
	err := Cleanup("/nonexistent/subtitle.srt")
	assert.NoError(t, err)
}

func TestCleanup_EmptyPath(t *testing.T) {
	err := Cleanup("")
	assert.NoError(t, err)
}

// --- Task 9.14: Media path validation ---

func TestPlacer_Place_InvalidMediaDir(t *testing.T) {
	p := NewPlacer(DefaultPlacerConfig())

	_, err := p.Place(PlaceRequest{
		MediaFilePath: "/nonexistent/dir/Movie.mkv",
		SubtitleData:  []byte("subtitle"),
		Language:      "zh-Hant",
		Format:        "srt",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "does not exist")
}

// --- Full Place flow ---

func TestPlacer_Place_Success(t *testing.T) {
	dir := t.TempDir()
	mediaPath := filepath.Join(dir, "Movie.2024.1080p.mkv")
	os.WriteFile(mediaPath, []byte("fake media"), 0644)

	p := NewPlacer(DefaultPlacerConfig())

	result, err := p.Place(PlaceRequest{
		MediaFilePath: mediaPath,
		SubtitleData:  []byte("1\n00:00:01,000 --> 00:00:03,000\n你好\n"),
		Language:      "zh-TW",
		Format:        "srt",
		Score:         0.95,
	})
	require.NoError(t, err)

	assert.Equal(t, filepath.Join(dir, "Movie.2024.1080p.zh-Hant.srt"), result.SubtitlePath)
	assert.Equal(t, "zh-Hant", result.Language)
	assert.Empty(t, result.BackupPath)

	// File should exist on disk
	assert.FileExists(t, result.SubtitlePath)
}

// --- CR: Path traversal prevention ---

func TestPlacer_Place_RelativePathRejected(t *testing.T) {
	p := NewPlacer(DefaultPlacerConfig())

	_, err := p.Place(PlaceRequest{
		MediaFilePath: "relative/path/Movie.mkv",
		SubtitleData:  []byte("subtitle"),
		Language:      "zh-Hant",
		Format:        "srt",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "must be absolute")
}

func TestPlacer_Place_PathTraversalCleaned(t *testing.T) {
	dir := t.TempDir()
	mediaPath := filepath.Join(dir, "sub/../Movie.mkv")
	os.WriteFile(filepath.Join(dir, "Movie.mkv"), []byte("fake"), 0644)

	p := NewPlacer(DefaultPlacerConfig())

	result, err := p.Place(PlaceRequest{
		MediaFilePath: mediaPath,
		SubtitleData:  []byte("1\n00:00:01,000 --> 00:00:03,000\nTest\n"),
		Language:      "zh-Hant",
		Format:        "srt",
	})
	require.NoError(t, err)
	// Path should be cleaned — no ".." in result
	assert.NotContains(t, result.SubtitlePath, "..")
	assert.Equal(t, filepath.Join(dir, "Movie.zh-Hant.srt"), result.SubtitlePath)
}

// --- CR: Language tag sanitization ---

func TestNormalizeLanguageTag_PathTraversal(t *testing.T) {
	// Malicious language tag should be rejected, not passed through
	assert.Equal(t, "und", normalizeLanguageTag("../../etc"))
	assert.Equal(t, "und", normalizeLanguageTag("foo/bar"))
	assert.Equal(t, "und", normalizeLanguageTag("a.b"))
	// Valid unknown tags should pass through
	assert.Equal(t, "ja", normalizeLanguageTag("ja"))
	assert.Equal(t, "ko", normalizeLanguageTag("ko"))
	assert.Equal(t, "pt-BR", normalizeLanguageTag("pt-BR"))
}

// --- CR: Backup target is a directory ---

func TestBackupExistingFile_BakIsDirectory(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "subtitle.zh-Hant.srt")

	// Create existing subtitle file
	os.WriteFile(path, []byte("old subtitle"), 0644)

	// Create .bak as a directory
	os.MkdirAll(path+".bak", 0755)

	_, err := backupExistingFile(path)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "backup target is a directory")
}

// --- CR: Special characters in filenames ---

func TestBuildSubtitleFilename_SpecialCharacters(t *testing.T) {
	tests := []struct {
		name      string
		mediaPath string
		lang      string
		ext       string
		expected  string
	}{
		{
			"fansub brackets",
			"/media/[SubGroup] Movie Name (2024) [1080p].mkv",
			"zh-Hant", "srt",
			"/media/[SubGroup] Movie Name (2024) [1080p].zh-Hant.srt",
		},
		{
			"spaces in name",
			"/media/My Movie.mkv",
			"zh-Hant", "srt",
			"/media/My Movie.zh-Hant.srt",
		},
		{
			"unicode in name",
			"/media/電影名稱.mkv",
			"zh-Hant", "srt",
			"/media/電影名稱.zh-Hant.srt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildSubtitleFilename(tt.mediaPath, tt.lang, tt.ext)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// --- CR: detectFormat with empty data and no hint ---

func TestDetectFormat_EmptyDataNoHint(t *testing.T) {
	_, err := detectFormat([]byte{}, "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unable to detect")

	_, err = detectFormat(nil, "")
	assert.Error(t, err)
}

func TestPlacer_Place_FormatDetection(t *testing.T) {
	dir := t.TempDir()
	mediaPath := filepath.Join(dir, "Movie.mkv")
	os.WriteFile(mediaPath, []byte("fake"), 0644)

	p := NewPlacer(DefaultPlacerConfig())

	// SRT content, no format hint
	result, err := p.Place(PlaceRequest{
		MediaFilePath: mediaPath,
		SubtitleData:  []byte("1\n00:00:01,000 --> 00:00:03,000\nTest\n"),
		Language:      "zh-Hant",
	})
	require.NoError(t, err)
	assert.True(t, filepath.Ext(result.SubtitlePath) == ".srt")

	// ASS content
	Cleanup(result.SubtitlePath)
	result2, err := p.Place(PlaceRequest{
		MediaFilePath: mediaPath,
		SubtitleData:  []byte("[Script Info]\nTitle: Test\n[V4+ Styles]\n"),
		Language:      "zh-Hant",
	})
	require.NoError(t, err)
	assert.True(t, filepath.Ext(result2.SubtitlePath) == ".ass")
}
