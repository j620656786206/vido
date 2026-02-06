package metadata

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vido/api/internal/models"
)

func TestPartialResultHandler_MergePartialResults_AllComplete(t *testing.T) {
	handler := NewPartialResultHandler()

	results := []*MetadataResult{
		{
			Source:    "tmdb",
			Title:     "The Matrix",
			Year:      1999,
			Overview:  "A computer hacker learns about the true nature of reality.",
			PosterURL: "https://image.tmdb.org/t/p/w500/matrix.jpg",
		},
	}

	merged := handler.MergePartialResults(results)

	require.NotNil(t, merged)
	assert.Equal(t, models.DegradationNormal, merged.DegradationLevel)
	assert.Empty(t, merged.MissingFields)
	assert.Equal(t, "tmdb", merged.FallbackUsed[0])
}

func TestPartialResultHandler_MergePartialResults_MissingPoster(t *testing.T) {
	handler := NewPartialResultHandler()

	results := []*MetadataResult{
		{
			Source:   "tmdb",
			Title:    "The Matrix",
			Year:     1999,
			Overview: "A computer hacker learns about the true nature of reality.",
		},
	}

	merged := handler.MergePartialResults(results)

	require.NotNil(t, merged)
	assert.Equal(t, models.DegradationPartial, merged.DegradationLevel)
	assert.Contains(t, merged.MissingFields, "posterUrl")
}

func TestPartialResultHandler_MergePartialResults_MultiSource(t *testing.T) {
	handler := NewPartialResultHandler()

	results := []*MetadataResult{
		{
			Source: "tmdb",
			Title:  "The Matrix",
			Year:   1999,
		},
		{
			Source:    "douban",
			Overview:  "黑客尼奧發現了現實的真相。",
			PosterURL: "https://img.douban.com/matrix.jpg",
		},
	}

	merged := handler.MergePartialResults(results)

	require.NotNil(t, merged)
	assert.Equal(t, models.DegradationNormal, merged.DegradationLevel)
	assert.Contains(t, merged.FallbackUsed, "tmdb")
	assert.Contains(t, merged.FallbackUsed, "douban")

	// Check merged data
	data, ok := merged.Data.(*MergedMetadata)
	require.True(t, ok)
	assert.Equal(t, "The Matrix", data.Title)
	assert.Equal(t, 1999, data.Year)
	assert.Equal(t, "黑客尼奧發現了現實的真相。", data.Overview)
	assert.Equal(t, "https://img.douban.com/matrix.jpg", data.PosterURL)
}

func TestPartialResultHandler_MergePartialResults_AllMissing(t *testing.T) {
	handler := NewPartialResultHandler()

	results := []*MetadataResult{}

	merged := handler.MergePartialResults(results)

	require.NotNil(t, merged)
	assert.Equal(t, models.DegradationMinimal, merged.DegradationLevel)
	assert.Contains(t, merged.MissingFields, "title")
	assert.Contains(t, merged.MissingFields, "year")
	assert.Contains(t, merged.MissingFields, "overview")
	assert.Contains(t, merged.MissingFields, "posterUrl")
}

func TestPartialResultHandler_GenerateMessage(t *testing.T) {
	handler := NewPartialResultHandler()

	tests := []struct {
		name          string
		missing       []string
		expectContain string
	}{
		{"no missing", []string{}, ""},
		{"missing title", []string{"title"}, "標題"},
		{"missing year", []string{"year"}, "年份"},
		{"missing overview", []string{"overview"}, "簡介"},
		{"missing poster", []string{"posterUrl"}, "海報"},
		{"multiple missing", []string{"title", "year"}, "、"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := handler.GenerateMessage(tt.missing)
			if tt.expectContain == "" {
				assert.Empty(t, msg)
			} else {
				assert.Contains(t, msg, tt.expectContain)
			}
		})
	}
}

func TestPartialResultHandler_SetPlaceholder(t *testing.T) {
	handler := NewPartialResultHandler()
	item := &MergedMetadata{}

	handler.SetPlaceholder(item, "title")
	assert.Equal(t, "未知標題", item.Title)

	handler.SetPlaceholder(item, "overview")
	assert.Equal(t, "暫無簡介", item.Overview)

	handler.SetPlaceholder(item, "posterUrl")
	assert.Equal(t, "/images/placeholder-poster.webp", item.PosterURL)
}

func TestFieldAvailability(t *testing.T) {
	field := models.FieldAvailability{
		Field:     "title",
		Available: true,
		Source:    "tmdb",
	}

	assert.Equal(t, "title", field.Field)
	assert.True(t, field.Available)
	assert.Equal(t, "tmdb", field.Source)
}
