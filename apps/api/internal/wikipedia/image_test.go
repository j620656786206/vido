package wikipedia

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestImageResult(t *testing.T) {
	t.Run("stores image result data", func(t *testing.T) {
		result := &ImageResult{
			URL:      "https://upload.wikimedia.org/image.jpg",
			Filename: "image.jpg",
			HasImage: true,
		}

		assert.Equal(t, "https://upload.wikimedia.org/image.jpg", result.URL)
		assert.Equal(t, "image.jpg", result.Filename)
		assert.True(t, result.HasImage)
		assert.Empty(t, result.PlaceholderReason)
	})

	t.Run("stores placeholder result", func(t *testing.T) {
		result := ImageNotAvailable()

		assert.Empty(t, result.URL)
		assert.False(t, result.HasImage)
		assert.Equal(t, NoImagePlaceholder, result.PlaceholderReason)
	})
}

func TestCleanImageFilename(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple filename",
			input:    "Poster.jpg",
			expected: "Poster.jpg",
		},
		{
			name:     "with File prefix",
			input:    "File:Poster.jpg",
			expected: "Poster.jpg",
		},
		{
			name:     "with Image prefix",
			input:    "Image:Poster.jpg",
			expected: "Poster.jpg",
		},
		{
			name:     "with wiki link syntax",
			input:    "[[File:Poster.jpg]]",
			expected: "Poster.jpg",
		},
		{
			name:     "with parameters",
			input:    "Poster.jpg|300px|thumb",
			expected: "Poster.jpg",
		},
		{
			name:     "full wiki syntax",
			input:    "[[File:Poster.jpg|300px|thumb|Caption]]",
			expected: "Poster.jpg",
		},
		{
			name:     "lowercase prefix",
			input:    "file:poster.jpg",
			expected: "poster.jpg",
		},
		{
			name:     "with spaces",
			input:    "  Poster.jpg  ",
			expected: "Poster.jpg",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cleanImageFilename(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsImageFile(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		expected bool
	}{
		{
			name:     "jpg file",
			filename: "poster.jpg",
			expected: true,
		},
		{
			name:     "jpeg file",
			filename: "poster.jpeg",
			expected: true,
		},
		{
			name:     "png file",
			filename: "poster.png",
			expected: true,
		},
		{
			name:     "gif file",
			filename: "poster.gif",
			expected: true,
		},
		{
			name:     "svg file",
			filename: "logo.svg",
			expected: true,
		},
		{
			name:     "webp file",
			filename: "image.webp",
			expected: true,
		},
		{
			name:     "uppercase extension",
			filename: "POSTER.JPG",
			expected: true,
		},
		{
			name:     "text file",
			filename: "document.txt",
			expected: false,
		},
		{
			name:     "pdf file",
			filename: "document.pdf",
			expected: false,
		},
		{
			name:     "no extension",
			filename: "imagefile",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isImageFile(tt.filename)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractFilenameFromWikiLink(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple File link",
			input:    "[[File:Poster.jpg]]",
			expected: "Poster.jpg",
		},
		{
			name:     "Image link",
			input:    "[[Image:Poster.jpg]]",
			expected: "Poster.jpg",
		},
		{
			name:     "with parameters",
			input:    "[[File:Poster.jpg|300px]]",
			expected: "Poster.jpg",
		},
		{
			name:     "in context",
			input:    "Some text [[File:Movie poster.jpg|thumb]] more text",
			expected: "Movie poster.jpg",
		},
		{
			name:     "lowercase",
			input:    "[[file:poster.jpg]]",
			expected: "poster.jpg",
		},
		{
			name:     "no file link",
			input:    "Just some text without links",
			expected: "",
		},
		{
			name:     "regular wiki link",
			input:    "[[Some Article]]",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractFilenameFromWikiLink(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestImageExtractor_findImageInWikitext(t *testing.T) {
	extractor := &ImageExtractor{}

	tests := []struct {
		name     string
		wikitext string
		expected string
	}{
		{
			name: "finds File link",
			wikitext: `{{Infobox film
| name = Test Movie
}}
[[File:Test Movie Poster.jpg|thumb|Movie poster]]`,
			expected: "Test Movie Poster.jpg",
		},
		{
			name: "finds Image link",
			wikitext: `Some text
[[Image:Movie.png|300px]]`,
			expected: "Movie.png",
		},
		{
			name: "finds image in infobox field",
			wikitext: `{{Infobox film
| name = Test
| image = Poster.jpg
| director = Someone
}}`,
			expected: "Poster.jpg",
		},
		{
			name: "finds chinese image field",
			wikitext: `{{電影資訊框
| 片名 = 測試
| 圖片 = MoviePoster.png
}}`,
			expected: "MoviePoster.png",
		},
		{
			name: "no image",
			wikitext: `{{Infobox film
| name = Test
| director = Someone
}}
Just text without any images.`,
			expected: "",
		},
		{
			name: "ignores non-image files",
			wikitext: `[[File:Document.pdf]]
[[File:Movie.jpg]]`,
			expected: "Movie.jpg",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractor.findImageInWikitext(tt.wikitext)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetPlaceholderURL(t *testing.T) {
	t.Run("returns empty string for frontend placeholder", func(t *testing.T) {
		url := GetPlaceholderURL()
		assert.Empty(t, url)
	})
}

func TestNoImagePlaceholder(t *testing.T) {
	t.Run("placeholder message is correct", func(t *testing.T) {
		assert.Equal(t, "No poster available from Wikipedia", NoImagePlaceholder)
	})
}
