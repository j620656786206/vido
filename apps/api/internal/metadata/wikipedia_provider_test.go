package metadata

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/wikipedia"
)

func TestNewWikipediaProvider(t *testing.T) {
	provider := NewWikipediaProvider(WikipediaProviderConfig{
		Enabled: true,
	})

	assert.NotNil(t, provider)
	assert.Equal(t, "Wikipedia", provider.Name())
	assert.Equal(t, models.MetadataSourceWikipedia, provider.Source())
	assert.True(t, provider.IsAvailable())
	assert.Equal(t, ProviderStatusAvailable, provider.Status())
}

func TestWikipediaProvider_Disabled(t *testing.T) {
	provider := NewWikipediaProvider(WikipediaProviderConfig{
		Enabled: false,
	})

	assert.False(t, provider.IsAvailable())
	assert.Equal(t, ProviderStatusUnavailable, provider.Status())
}

func TestWikipediaProvider_Search_Disabled(t *testing.T) {
	provider := NewWikipediaProvider(WikipediaProviderConfig{
		Enabled: false,
	})

	result, err := provider.Search(context.Background(), &SearchRequest{
		Query:     "Test Movie",
		MediaType: MediaTypeMovie,
	})

	// Should return error indicating provider is disabled
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "disabled")
}

func TestWikipediaProvider_SetEnabled(t *testing.T) {
	provider := NewWikipediaProvider(WikipediaProviderConfig{
		Enabled: true,
	})

	assert.True(t, provider.IsAvailable())

	provider.SetEnabled(false)
	assert.False(t, provider.IsAvailable())
	assert.Equal(t, ProviderStatusUnavailable, provider.Status())

	provider.SetEnabled(true)
	assert.True(t, provider.IsAvailable())
	assert.Equal(t, ProviderStatusAvailable, provider.Status())
}

func TestWikipediaProvider_ImplementsInterface(t *testing.T) {
	provider := NewWikipediaProvider(WikipediaProviderConfig{})

	// Compile-time interface verification
	var _ MetadataProvider = provider
}

func TestDefaultWikipediaProviderConfig(t *testing.T) {
	config := DefaultWikipediaProviderConfig()

	assert.True(t, config.Enabled)
	assert.True(t, config.CacheConfig.Enabled)
	assert.Equal(t, 5, config.CircuitBreakerConfig.FailureThreshold)
	assert.Equal(t, 2, config.CircuitBreakerConfig.SuccessThreshold)
}

func TestWikipediaProvider_CircuitBreaker(t *testing.T) {
	config := DefaultWikipediaProviderConfig()
	provider := NewWikipediaProvider(config)

	// Check initial circuit breaker stats
	stats := provider.GetCircuitBreakerStats()
	assert.Equal(t, 0, stats.ConsecutiveFailures)

	// Reset should work without crashing
	provider.ResetCircuitBreaker()
	stats = provider.GetCircuitBreakerStats()
	assert.Equal(t, 0, stats.ConsecutiveFailures)
}

func TestWikipediaProvider_DetermineMediaType(t *testing.T) {
	provider := NewWikipediaProvider(WikipediaProviderConfig{Enabled: true})

	tests := []struct {
		name        string
		infoboxType string
		expected    MediaType
	}{
		{
			name:        "film infobox",
			infoboxType: "Infobox film",
			expected:    MediaTypeMovie,
		},
		{
			name:        "movie infobox",
			infoboxType: "Infobox movie",
			expected:    MediaTypeMovie,
		},
		{
			name:        "chinese film infobox",
			infoboxType: "電影資訊框",
			expected:    MediaTypeMovie,
		},
		{
			name:        "television infobox",
			infoboxType: "Infobox television",
			expected:    MediaTypeTV,
		},
		{
			name:        "tv series infobox",
			infoboxType: "Infobox TV series",
			expected:    MediaTypeTV,
		},
		{
			name:        "chinese tv infobox",
			infoboxType: "電視節目資訊框",
			expected:    MediaTypeTV,
		},
		{
			name:        "animanga infobox",
			infoboxType: "Infobox animanga/Header",
			expected:    MediaTypeTV,
		},
		{
			name:        "anime infobox",
			infoboxType: "Infobox anime",
			expected:    MediaTypeTV,
		},
		{
			name:        "chinese anime infobox",
			infoboxType: "動畫資訊框",
			expected:    MediaTypeTV,
		},
		{
			name:        "unknown defaults to movie",
			infoboxType: "Unknown Infobox",
			expected:    MediaTypeMovie,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := provider.determineMediaType(tt.infoboxType)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestWikipediaProvider_BuildMetadataItem(t *testing.T) {
	provider := NewWikipediaProvider(WikipediaProviderConfig{Enabled: true})

	t.Run("builds item from infobox data", func(t *testing.T) {
		rankedResult := wikipedia.RankedResult{
			SearchResult: wikipedia.SearchResult{
				PageID: 12345,
				Title:  "寄生上流",
			},
			Confidence: 0.95,
		}
		content := &wikipedia.PageContent{
			PageID: 12345,
			Title:  "寄生上流",
		}
		infobox := &wikipedia.InfoboxData{
			Type:         "Infobox film",
			Name:         "寄生上流",
			OriginalName: "기생충",
			Year:         2019,
			Genre:        []string{"驚悚", "黑色喜劇"},
		}
		summary := "《寄生上流》是2019年韓國驚悚黑色喜劇電影。"
		imageResult := &wikipedia.ImageResult{
			HasImage: true,
			URL:      "https://upload.wikimedia.org/poster.jpg",
		}

		item := provider.buildMetadataItem(rankedResult, content, infobox, summary, imageResult)

		assert.Equal(t, "12345", item.ID)
		assert.Equal(t, "寄生上流", item.Title)
		assert.Equal(t, "寄生上流", item.TitleZhTW)
		assert.Equal(t, "기생충", item.OriginalTitle)
		assert.Equal(t, 2019, item.Year)
		assert.Equal(t, summary, item.Overview)
		assert.Equal(t, summary, item.OverviewZhTW)
		assert.Equal(t, []string{"驚悚", "黑色喜劇"}, item.Genres)
		assert.Equal(t, "https://upload.wikimedia.org/poster.jpg", item.PosterURL)
		assert.Equal(t, MediaTypeMovie, item.MediaType)
		assert.Equal(t, 0.95, item.Confidence)
	})

	t.Run("falls back to page title when no infobox", func(t *testing.T) {
		rankedResult := wikipedia.RankedResult{
			SearchResult: wikipedia.SearchResult{
				PageID: 12345,
				Title:  "Test Page",
			},
			Confidence: 0.8,
		}
		content := &wikipedia.PageContent{
			PageID: 12345,
			Title:  "Test Page",
		}
		summary := "This is a test page summary."

		item := provider.buildMetadataItem(rankedResult, content, nil, summary, nil)

		assert.Equal(t, "12345", item.ID)
		assert.Equal(t, "Test Page", item.Title)
		assert.Equal(t, "Test Page", item.TitleZhTW)
		assert.Empty(t, item.PosterURL)
		assert.Equal(t, MediaTypeMovie, item.MediaType) // Default
	})

	t.Run("handles empty image result", func(t *testing.T) {
		rankedResult := wikipedia.RankedResult{
			SearchResult: wikipedia.SearchResult{
				PageID: 12345,
				Title:  "Test",
			},
			Confidence: 0.9,
		}
		content := &wikipedia.PageContent{
			PageID: 12345,
			Title:  "Test",
		}
		imageResult := &wikipedia.ImageResult{
			HasImage: false,
		}

		item := provider.buildMetadataItem(rankedResult, content, nil, "Summary", imageResult)

		assert.Empty(t, item.PosterURL)
	})
}

func TestWikipediaProvider_HandleSearchError(t *testing.T) {
	provider := NewWikipediaProvider(WikipediaProviderConfig{Enabled: true})
	req := &SearchRequest{Query: "Test", Page: 1}

	t.Run("not found error returns empty result", func(t *testing.T) {
		err := &wikipedia.NotFoundError{Query: "Test"}
		result, searchErr := provider.handleSearchError(err, req)

		assert.NoError(t, searchErr)
		assert.NotNil(t, result)
		assert.Empty(t, result.Items)
		assert.Equal(t, 0, result.TotalCount)
	})

	t.Run("parse error returns provider error", func(t *testing.T) {
		err := &wikipedia.ParseError{Field: "test", Reason: "failed"}
		result, searchErr := provider.handleSearchError(err, req)

		assert.Error(t, searchErr)
		assert.Nil(t, result)
		assert.Contains(t, searchErr.Error(), "parse error")
	})

	t.Run("api error returns provider error", func(t *testing.T) {
		err := &wikipedia.APIError{Code: "test", Info: "error"}
		result, searchErr := provider.handleSearchError(err, req)

		assert.Error(t, searchErr)
		assert.Nil(t, result)
		assert.Contains(t, searchErr.Error(), "API error")
	})

	t.Run("context deadline returns timeout error", func(t *testing.T) {
		err := context.DeadlineExceeded
		result, searchErr := provider.handleSearchError(err, req)

		assert.Error(t, searchErr)
		assert.Nil(t, result)
		assert.Contains(t, searchErr.Error(), "timeout")
	})
}
