package images

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// createTestJPEG creates a test JPEG image for testing
func createTestJPEG(t *testing.T, width, height int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	// Fill with a solid color
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.RGBA{R: 255, G: 128, B: 64, A: 255})
		}
	}

	var buf bytes.Buffer
	err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 90})
	require.NoError(t, err)
	return buf.Bytes()
}

// createTestPNG creates a test PNG image for testing
func createTestPNG(t *testing.T, width, height int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.RGBA{R: 64, G: 128, B: 255, A: 255})
		}
	}

	var buf bytes.Buffer
	err := png.Encode(&buf, img)
	require.NoError(t, err)
	return buf.Bytes()
}

func TestNewImageProcessor(t *testing.T) {
	tmpDir := t.TempDir()
	processor, err := NewImageProcessor(tmpDir)

	require.NoError(t, err)
	assert.NotNil(t, processor)
	assert.Equal(t, tmpDir, processor.cacheDir)
}

func TestNewImageProcessor_CreatesDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	cacheDir := filepath.Join(tmpDir, "posters")

	processor, err := NewImageProcessor(cacheDir)

	require.NoError(t, err)
	assert.NotNil(t, processor)

	// Directory should be created
	info, err := os.Stat(cacheDir)
	require.NoError(t, err)
	assert.True(t, info.IsDir())
}

func TestImageProcessor_ProcessPoster_JPEG(t *testing.T) {
	tmpDir := t.TempDir()
	processor, err := NewImageProcessor(tmpDir)
	require.NoError(t, err)

	// Create a test JPEG image (400x600)
	jpegData := createTestJPEG(t, 400, 600)

	result, err := processor.ProcessPoster(bytes.NewReader(jpegData), "test-movie-1")

	require.NoError(t, err)
	assert.NotNil(t, result)

	// Check paths are set
	assert.Contains(t, result.PosterPath, "test-movie-1")
	assert.Contains(t, result.ThumbnailPath, "test-movie-1-thumb")

	// Verify files exist
	assert.FileExists(t, result.PosterPath)
	assert.FileExists(t, result.ThumbnailPath)

	// Verify poster dimensions (300x450)
	posterFile, err := os.Open(result.PosterPath)
	require.NoError(t, err)
	defer posterFile.Close()

	posterImg, _, err := image.Decode(posterFile)
	require.NoError(t, err)
	bounds := posterImg.Bounds()
	assert.Equal(t, 300, bounds.Dx())
	assert.Equal(t, 450, bounds.Dy())

	// Verify thumbnail dimensions (100x150)
	thumbFile, err := os.Open(result.ThumbnailPath)
	require.NoError(t, err)
	defer thumbFile.Close()

	thumbImg, _, err := image.Decode(thumbFile)
	require.NoError(t, err)
	thumbBounds := thumbImg.Bounds()
	assert.Equal(t, 100, thumbBounds.Dx())
	assert.Equal(t, 150, thumbBounds.Dy())
}

func TestImageProcessor_ProcessPoster_PNG(t *testing.T) {
	tmpDir := t.TempDir()
	processor, err := NewImageProcessor(tmpDir)
	require.NoError(t, err)

	// Create a test PNG image
	pngData := createTestPNG(t, 500, 750)

	result, err := processor.ProcessPoster(bytes.NewReader(pngData), "test-movie-2")

	require.NoError(t, err)
	assert.NotNil(t, result)

	// Check that JPEG files are created (WebP requires CGO)
	assert.Contains(t, result.PosterPath, ".jpg")
	assert.Contains(t, result.ThumbnailPath, ".jpg")

	// Verify files exist
	assert.FileExists(t, result.PosterPath)
	assert.FileExists(t, result.ThumbnailPath)
}

func TestImageProcessor_ProcessPoster_SmallImage(t *testing.T) {
	tmpDir := t.TempDir()
	processor, err := NewImageProcessor(tmpDir)
	require.NoError(t, err)

	// Create a small test image (100x150) - smaller than target
	jpegData := createTestJPEG(t, 100, 150)

	result, err := processor.ProcessPoster(bytes.NewReader(jpegData), "test-movie-3")

	require.NoError(t, err)
	assert.NotNil(t, result)

	// Should still process successfully
	assert.FileExists(t, result.PosterPath)
	assert.FileExists(t, result.ThumbnailPath)
}

func TestImageProcessor_ProcessPoster_InvalidImage(t *testing.T) {
	tmpDir := t.TempDir()
	processor, err := NewImageProcessor(tmpDir)
	require.NoError(t, err)

	// Invalid image data
	invalidData := []byte("not an image")

	result, err := processor.ProcessPoster(bytes.NewReader(invalidData), "test-invalid")

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "decode")
}

func TestImageProcessor_ProcessPoster_OutputFormat(t *testing.T) {
	tmpDir := t.TempDir()
	processor, err := NewImageProcessor(tmpDir)
	require.NoError(t, err)

	jpegData := createTestJPEG(t, 400, 600)

	result, err := processor.ProcessPoster(bytes.NewReader(jpegData), "test-format")

	require.NoError(t, err)

	// Output should be WebP or JPEG
	ext := filepath.Ext(result.PosterPath)
	assert.True(t, ext == ".webp" || ext == ".jpg", "Output should be webp or jpg, got: %s", ext)
}

func TestImageProcessor_ProcessPoster_SizeReduction(t *testing.T) {
	tmpDir := t.TempDir()
	processor, err := NewImageProcessor(tmpDir)
	require.NoError(t, err)

	// Create a large test image
	jpegData := createTestJPEG(t, 1200, 1800)

	result, err := processor.ProcessPoster(bytes.NewReader(jpegData), "test-size")

	require.NoError(t, err)

	// Processed file should be smaller than original
	info, err := os.Stat(result.PosterPath)
	require.NoError(t, err)
	assert.Less(t, info.Size(), int64(len(jpegData)))
}

func TestImageProcessor_ProcessPoster_OriginalSize(t *testing.T) {
	tmpDir := t.TempDir()
	processor, err := NewImageProcessor(tmpDir)
	require.NoError(t, err)

	jpegData := createTestJPEG(t, 400, 600)

	result, err := processor.ProcessPoster(bytes.NewReader(jpegData), "test-original-size")

	require.NoError(t, err)
	assert.Equal(t, int64(len(jpegData)), result.OriginalSize)
	assert.Greater(t, result.ProcessedSize, int64(0))
}

func TestImageProcessor_GetPosterURL(t *testing.T) {
	tmpDir := t.TempDir()
	processor, err := NewImageProcessor(tmpDir)
	require.NoError(t, err)

	url := processor.GetPosterURL("test-movie-id")

	assert.Equal(t, "/posters/test-movie-id.webp", url)
}

func TestImageProcessor_GetThumbnailURL(t *testing.T) {
	tmpDir := t.TempDir()
	processor, err := NewImageProcessor(tmpDir)
	require.NoError(t, err)

	url := processor.GetThumbnailURL("test-movie-id")

	assert.Equal(t, "/posters/test-movie-id-thumb.webp", url)
}

func TestImageProcessor_MaintainsAspectRatio(t *testing.T) {
	tmpDir := t.TempDir()
	processor, err := NewImageProcessor(tmpDir)
	require.NoError(t, err)

	// Create a wide image (not 2:3 aspect ratio)
	wideImage := createTestJPEG(t, 800, 400)

	result, err := processor.ProcessPoster(bytes.NewReader(wideImage), "test-wide")

	require.NoError(t, err)

	// File should exist
	assert.FileExists(t, result.PosterPath)

	// Open and check dimensions - should maintain 2:3 target ratio
	posterFile, err := os.Open(result.PosterPath)
	require.NoError(t, err)
	defer posterFile.Close()

	posterImg, _, err := image.Decode(posterFile)
	require.NoError(t, err)
	bounds := posterImg.Bounds()

	// Should be 300x450 (cropped/fitted to poster ratio)
	assert.Equal(t, 300, bounds.Dx())
	assert.Equal(t, 450, bounds.Dy())
}
