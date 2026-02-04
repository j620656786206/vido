// Package images provides image processing functionality for poster optimization.
package images

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"log/slog"
	"os"
	"path/filepath"

	"golang.org/x/image/draw"
	// Register WebP decoder
	_ "golang.org/x/image/webp"
)

// Constants for image processing
const (
	// PosterWidth is the standard poster width in pixels
	PosterWidth = 300
	// PosterHeight is the standard poster height in pixels
	PosterHeight = 450
	// ThumbnailWidth is the thumbnail width in pixels
	ThumbnailWidth = 100
	// ThumbnailHeight is the thumbnail height in pixels
	ThumbnailHeight = 150
	// JpegQuality is the JPEG encoding quality (0-100)
	JpegQuality = 80
	// ThumbnailQuality is the thumbnail JPEG quality
	ThumbnailQuality = 75
)

// ProcessedImage contains information about a processed poster image
type ProcessedImage struct {
	PosterPath    string
	ThumbnailPath string
	OriginalSize  int64
	ProcessedSize int64
}

// ImageProcessor handles poster image optimization
type ImageProcessor struct {
	cacheDir string
	logger   *slog.Logger
}

// NewImageProcessor creates a new ImageProcessor with the given cache directory
func NewImageProcessor(cacheDir string) (*ImageProcessor, error) {
	// Ensure cache directory exists
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create cache directory: %w", err)
	}

	return &ImageProcessor{
		cacheDir: cacheDir,
		logger:   slog.Default(),
	}, nil
}

// ProcessPoster processes an uploaded poster image:
// 1. Decodes the image
// 2. Resizes to poster dimensions (300x450)
// 3. Creates thumbnail (100x150)
// 4. Saves as optimized JPEG (WebP not supported in standard library without CGO)
func (p *ImageProcessor) ProcessPoster(input io.Reader, mediaID string) (*ProcessedImage, error) {
	// Read all data to calculate original size
	data, err := io.ReadAll(input)
	if err != nil {
		return nil, fmt.Errorf("failed to read image data: %w", err)
	}
	originalSize := int64(len(data))

	// Decode the image
	img, format, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %w", err)
	}

	p.logger.Debug("Decoded image",
		"format", format,
		"width", img.Bounds().Dx(),
		"height", img.Bounds().Dy(),
		"original_size", originalSize,
	)

	// Resize to poster dimensions
	poster := p.resizeAndCrop(img, PosterWidth, PosterHeight)

	// Create thumbnail
	thumbnail := p.resizeAndCrop(img, ThumbnailWidth, ThumbnailHeight)

	// Save poster
	posterPath := filepath.Join(p.cacheDir, fmt.Sprintf("%s.webp", mediaID))
	posterSize, err := p.saveAsJPEG(poster, posterPath, JpegQuality)
	if err != nil {
		return nil, fmt.Errorf("failed to save poster: %w", err)
	}
	// Rename to actual format used
	actualPosterPath := filepath.Join(p.cacheDir, fmt.Sprintf("%s.jpg", mediaID))
	if err := os.Rename(posterPath, actualPosterPath); err != nil {
		// If rename fails, keep the webp extension but it's actually JPEG
		actualPosterPath = posterPath
	}

	// Save thumbnail
	thumbPath := filepath.Join(p.cacheDir, fmt.Sprintf("%s-thumb.webp", mediaID))
	_, err = p.saveAsJPEG(thumbnail, thumbPath, ThumbnailQuality)
	if err != nil {
		return nil, fmt.Errorf("failed to save thumbnail: %w", err)
	}
	actualThumbPath := filepath.Join(p.cacheDir, fmt.Sprintf("%s-thumb.jpg", mediaID))
	if err := os.Rename(thumbPath, actualThumbPath); err != nil {
		actualThumbPath = thumbPath
	}

	p.logger.Info("Processed poster image",
		"media_id", mediaID,
		"original_size", originalSize,
		"processed_size", posterSize,
		"poster_path", actualPosterPath,
		"thumbnail_path", actualThumbPath,
	)

	return &ProcessedImage{
		PosterPath:    actualPosterPath,
		ThumbnailPath: actualThumbPath,
		OriginalSize:  originalSize,
		ProcessedSize: posterSize,
	}, nil
}

// resizeAndCrop resizes the image to the target dimensions, cropping if necessary
// to maintain the target aspect ratio
func (p *ImageProcessor) resizeAndCrop(src image.Image, targetWidth, targetHeight int) image.Image {
	srcBounds := src.Bounds()
	srcWidth := srcBounds.Dx()
	srcHeight := srcBounds.Dy()

	// Calculate source aspect ratio and target aspect ratio
	srcRatio := float64(srcWidth) / float64(srcHeight)
	targetRatio := float64(targetWidth) / float64(targetHeight)

	var cropRect image.Rectangle

	if srcRatio > targetRatio {
		// Source is wider - crop horizontally
		newWidth := int(float64(srcHeight) * targetRatio)
		xOffset := (srcWidth - newWidth) / 2
		cropRect = image.Rect(xOffset, 0, xOffset+newWidth, srcHeight)
	} else {
		// Source is taller - crop vertically
		newHeight := int(float64(srcWidth) / targetRatio)
		yOffset := (srcHeight - newHeight) / 2
		cropRect = image.Rect(0, yOffset, srcWidth, yOffset+newHeight)
	}

	// Create the cropped image
	croppedWidth := cropRect.Dx()
	croppedHeight := cropRect.Dy()

	// Create destination image
	dst := image.NewRGBA(image.Rect(0, 0, targetWidth, targetHeight))

	// Use high-quality resampling
	draw.CatmullRom.Scale(dst, dst.Bounds(), src, cropRect, draw.Over, nil)

	p.logger.Debug("Resized image",
		"src_width", srcWidth,
		"src_height", srcHeight,
		"cropped_width", croppedWidth,
		"cropped_height", croppedHeight,
		"target_width", targetWidth,
		"target_height", targetHeight,
	)

	return dst
}

// saveAsJPEG saves the image as a JPEG file and returns the file size
func (p *ImageProcessor) saveAsJPEG(img image.Image, path string, quality int) (int64, error) {
	file, err := os.Create(path)
	if err != nil {
		return 0, fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	// Encode as JPEG
	opts := &jpeg.Options{Quality: quality}
	if err := jpeg.Encode(file, img, opts); err != nil {
		return 0, fmt.Errorf("failed to encode JPEG: %w", err)
	}

	// Get file size
	info, err := file.Stat()
	if err != nil {
		return 0, fmt.Errorf("failed to get file info: %w", err)
	}

	return info.Size(), nil
}

// saveAsPNG saves the image as a PNG file (for lossless output if needed)
func (p *ImageProcessor) saveAsPNG(img image.Image, path string) (int64, error) {
	file, err := os.Create(path)
	if err != nil {
		return 0, fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	if err := png.Encode(file, img); err != nil {
		return 0, fmt.Errorf("failed to encode PNG: %w", err)
	}

	info, err := file.Stat()
	if err != nil {
		return 0, fmt.Errorf("failed to get file info: %w", err)
	}

	return info.Size(), nil
}

// GetPosterURL returns the URL path for a poster
func (p *ImageProcessor) GetPosterURL(mediaID string) string {
	return fmt.Sprintf("/posters/%s.jpg", mediaID)
}

// GetThumbnailURL returns the URL path for a thumbnail
func (p *ImageProcessor) GetThumbnailURL(mediaID string) string {
	return fmt.Sprintf("/posters/%s-thumb.jpg", mediaID)
}

// GetPosterPath returns the full file path for a poster
func (p *ImageProcessor) GetPosterPath(mediaID string) string {
	return filepath.Join(p.cacheDir, fmt.Sprintf("%s.jpg", mediaID))
}

// GetThumbnailPath returns the full file path for a thumbnail
func (p *ImageProcessor) GetThumbnailPath(mediaID string) string {
	return filepath.Join(p.cacheDir, fmt.Sprintf("%s-thumb.jpg", mediaID))
}

// DeletePoster removes the poster and thumbnail files for a media item
func (p *ImageProcessor) DeletePoster(mediaID string) error {
	posterPath := p.GetPosterPath(mediaID)
	thumbPath := p.GetThumbnailPath(mediaID)

	var errs []error

	if err := os.Remove(posterPath); err != nil && !os.IsNotExist(err) {
		errs = append(errs, fmt.Errorf("failed to delete poster: %w", err))
	}

	if err := os.Remove(thumbPath); err != nil && !os.IsNotExist(err) {
		errs = append(errs, fmt.Errorf("failed to delete thumbnail: %w", err))
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors deleting poster files: %v", errs)
	}

	return nil
}
