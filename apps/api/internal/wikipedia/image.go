package wikipedia

import (
	"context"
	"log/slog"
	"strings"
)

// ImageResult represents the result of an image extraction
type ImageResult struct {
	// URL is the direct URL to the image, empty if not found
	URL string
	// Filename is the original Wikipedia filename
	Filename string
	// HasImage indicates whether an image was found
	HasImage bool
	// PlaceholderReason explains why a placeholder should be shown
	PlaceholderReason string
}

// NoImagePlaceholder is the placeholder reason for Wikipedia sources
const NoImagePlaceholder = "No poster available from Wikipedia"

// ImageExtractor extracts images from Wikipedia pages
type ImageExtractor struct {
	client *Client
	logger *slog.Logger
}

// NewImageExtractor creates a new image extractor
func NewImageExtractor(client *Client, logger *slog.Logger) *ImageExtractor {
	if logger == nil {
		logger = slog.Default()
	}
	return &ImageExtractor{
		client: client,
		logger: logger,
	}
}

// ExtractFromInfobox extracts the image URL from Infobox data
func (e *ImageExtractor) ExtractFromInfobox(ctx context.Context, infobox *InfoboxData) *ImageResult {
	if infobox == nil || infobox.Image == "" {
		return &ImageResult{
			HasImage:          false,
			PlaceholderReason: NoImagePlaceholder,
		}
	}

	// Clean the image filename
	filename := cleanImageFilename(infobox.Image)
	if filename == "" {
		return &ImageResult{
			HasImage:          false,
			PlaceholderReason: NoImagePlaceholder,
		}
	}

	// Get the image URL from Wikipedia
	imageInfo, err := e.client.GetImageInfo(ctx, filename)
	if err != nil {
		e.logger.Debug("Failed to get image URL from Wikipedia",
			"filename", filename,
			"error", err,
		)
		return &ImageResult{
			Filename:          filename,
			HasImage:          false,
			PlaceholderReason: NoImagePlaceholder,
		}
	}

	return &ImageResult{
		URL:      imageInfo.URL,
		Filename: filename,
		HasImage: true,
	}
}

// ExtractFromPage extracts images from a Wikipedia page
// This attempts to find an image even if Infobox parsing failed
func (e *ImageExtractor) ExtractFromPage(ctx context.Context, content *PageContent) *ImageResult {
	if content == nil || content.Wikitext == "" {
		return &ImageResult{
			HasImage:          false,
			PlaceholderReason: NoImagePlaceholder,
		}
	}

	// Try to find an image in the wikitext
	filename := e.findImageInWikitext(content.Wikitext)
	if filename == "" {
		return &ImageResult{
			HasImage:          false,
			PlaceholderReason: NoImagePlaceholder,
		}
	}

	// Get the image URL from Wikipedia
	imageInfo, err := e.client.GetImageInfo(ctx, filename)
	if err != nil {
		e.logger.Debug("Failed to get image URL from Wikipedia",
			"filename", filename,
			"error", err,
		)
		return &ImageResult{
			Filename:          filename,
			HasImage:          false,
			PlaceholderReason: NoImagePlaceholder,
		}
	}

	return &ImageResult{
		URL:      imageInfo.URL,
		Filename: filename,
		HasImage: true,
	}
}

// findImageInWikitext looks for image references in wikitext
func (e *ImageExtractor) findImageInWikitext(wikitext string) string {
	// First, look for image field in Infobox
	lines := strings.Split(wikitext, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Check for image field in Infobox (without full parsing)
		lineLower := strings.ToLower(line)
		if strings.HasPrefix(lineLower, "| image") || strings.HasPrefix(lineLower, "|image") ||
			strings.HasPrefix(lineLower, "| 圖片") || strings.HasPrefix(lineLower, "|圖片") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				filename := cleanImageFilename(strings.TrimSpace(parts[1]))
				if filename != "" && isImageFile(filename) {
					return filename
				}
			}
		}
	}

	// Then look for all File: or Image: links in the wikitext
	// Keep searching until we find an actual image file
	for _, filename := range extractAllFilenamesFromWikitext(wikitext) {
		if isImageFile(filename) {
			return filename
		}
	}

	return ""
}

// extractAllFilenamesFromWikitext extracts all file references from wikitext
func extractAllFilenamesFromWikitext(wikitext string) []string {
	var filenames []string
	prefixes := []string{"[[File:", "[[Image:", "[[file:", "[[image:"}

	remaining := wikitext
	for len(remaining) > 0 {
		found := false
		for _, prefix := range prefixes {
			idx := strings.Index(remaining, prefix)
			if idx >= 0 {
				start := idx + len(prefix)
				rest := remaining[start:]

				// Find the end of the filename
				endPipe := strings.Index(rest, "|")
				endBracket := strings.Index(rest, "]]")

				end := -1
				if endPipe >= 0 && endBracket >= 0 {
					if endPipe < endBracket {
						end = endPipe
					} else {
						end = endBracket
					}
				} else if endPipe >= 0 {
					end = endPipe
				} else if endBracket >= 0 {
					end = endBracket
				}

				if end > 0 {
					filename := strings.TrimSpace(rest[:end])
					filenames = append(filenames, filename)
					remaining = rest[end:]
					found = true
					break
				}
			}
		}
		if !found {
			break
		}
	}

	return filenames
}

// extractFilenameFromWikiLink extracts the filename from a wiki link
func extractFilenameFromWikiLink(text string) string {
	// Find [[File:...]] or [[Image:...]]
	prefixes := []string{"[[File:", "[[Image:", "[[file:", "[[image:"}

	for _, prefix := range prefixes {
		idx := strings.Index(text, prefix)
		if idx >= 0 {
			start := idx + len(prefix)
			remaining := text[start:]

			// Find the end of the filename (first | or ]])
			endPipe := strings.Index(remaining, "|")
			endBracket := strings.Index(remaining, "]]")

			// Determine the end position
			end := -1
			if endPipe >= 0 && endBracket >= 0 {
				if endPipe < endBracket {
					end = endPipe
				} else {
					end = endBracket
				}
			} else if endPipe >= 0 {
				end = endPipe
			} else if endBracket >= 0 {
				end = endBracket
			}

			if end > 0 {
				filename := remaining[:end]
				return strings.TrimSpace(filename)
			}
		}
	}

	return ""
}

// cleanImageFilename cleans up an image filename
func cleanImageFilename(filename string) string {
	// Remove wiki link syntax first
	filename = strings.TrimPrefix(filename, "[[")
	if idx := strings.Index(filename, "]]"); idx > 0 {
		filename = filename[:idx]
	}

	// Remove File: or Image: prefix (case insensitive)
	for _, prefix := range []string{"File:", "Image:", "file:", "image:"} {
		if strings.HasPrefix(filename, prefix) {
			filename = strings.TrimPrefix(filename, prefix)
			break
		}
	}

	// Remove parameters after |
	if idx := strings.Index(filename, "|"); idx > 0 {
		filename = filename[:idx]
	}

	return strings.TrimSpace(filename)
}

// isImageFile checks if a filename looks like an image
func isImageFile(filename string) bool {
	lower := strings.ToLower(filename)
	extensions := []string{".jpg", ".jpeg", ".png", ".gif", ".svg", ".webp"}

	for _, ext := range extensions {
		if strings.HasSuffix(lower, ext) {
			return true
		}
	}

	return false
}

// GetPlaceholderURL returns a placeholder image URL for Wikipedia sources
// This can be used when no poster is available
func GetPlaceholderURL() string {
	// Return empty string to indicate frontend should show default placeholder
	return ""
}

// ImageNotAvailable returns an ImageResult indicating no image
func ImageNotAvailable() *ImageResult {
	return &ImageResult{
		HasImage:          false,
		PlaceholderReason: NoImagePlaceholder,
	}
}
