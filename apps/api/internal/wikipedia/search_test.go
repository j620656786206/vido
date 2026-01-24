package wikipedia

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefaultSearchOptions(t *testing.T) {
	t.Run("returns sensible defaults", func(t *testing.T) {
		opts := DefaultSearchOptions()

		assert.Equal(t, 5, opts.Limit)
		assert.True(t, opts.PreferTraditionalChinese)
		assert.Equal(t, MediaType(""), opts.MediaType)
		assert.Equal(t, 0, opts.Year)
	})
}

func TestNormalizeText(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple text",
			input:    "Hello World",
			expected: "hello world",
		},
		{
			name:     "chinese text",
			input:    "寄生上流",
			expected: "寄生上流",
		},
		{
			name:     "mixed text",
			input:    "寄生上流 (2019電影)",
			expected: "寄生上流 2019電影",
		},
		{
			name:     "punctuation removal",
			input:    "Hello, World! How are you?",
			expected: "hello world how are you",
		},
		{
			name:     "extra spaces",
			input:    "Hello   World",
			expected: "hello world",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizeText(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCalculateCharacterOverlap(t *testing.T) {
	tests := []struct {
		name     string
		s1       string
		s2       string
		minScore float64
		maxScore float64
	}{
		{
			name:     "identical strings",
			s1:       "hello",
			s2:       "hello",
			minScore: 0.99,
			maxScore: 1.01,
		},
		{
			name:     "completely different",
			s1:       "abc",
			s2:       "xyz",
			minScore: 0,
			maxScore: 0.01,
		},
		{
			name:     "partial overlap",
			s1:       "hello",
			s2:       "help",
			minScore: 0.5,
			maxScore: 0.9,
		},
		{
			name:     "empty strings",
			s1:       "",
			s2:       "",
			minScore: 0,
			maxScore: 0.01,
		},
		{
			name:     "chinese characters",
			s1:       "寄生上流",
			s2:       "寄生蟲",
			minScore: 0.4,
			maxScore: 0.8,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculateCharacterOverlap(tt.s1, tt.s2)
			assert.GreaterOrEqual(t, result, tt.minScore)
			assert.LessOrEqual(t, result, tt.maxScore)
		})
	}
}

func TestContainsMediaTypeIndicator(t *testing.T) {
	tests := []struct {
		name      string
		title     string
		mediaType MediaType
		expected  bool
	}{
		{
			name:      "movie indicator in chinese",
			title:     "寄生上流 (電影)",
			mediaType: MediaTypeMovie,
			expected:  true,
		},
		{
			name:      "movie indicator in english",
			title:     "Parasite (film)",
			mediaType: MediaTypeMovie,
			expected:  true,
		},
		{
			name:      "tv indicator in chinese",
			title:     "魷魚遊戲 (電視劇)",
			mediaType: MediaTypeTV,
			expected:  true,
		},
		{
			name:      "anime indicator",
			title:     "鬼滅之刃 (動畫)",
			mediaType: MediaTypeAnime,
			expected:  true,
		},
		{
			name:      "no indicator",
			title:     "寄生上流",
			mediaType: MediaTypeMovie,
			expected:  false,
		},
		{
			name:      "wrong media type",
			title:     "魷魚遊戲 (電視劇)",
			mediaType: MediaTypeMovie,
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := containsMediaTypeIndicator(tt.title, tt.mediaType)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSearcher_rankResults(t *testing.T) {
	searcher := &Searcher{}

	t.Run("ranks exact matches highest", func(t *testing.T) {
		results := []SearchResult{
			{PageID: 1, Title: "寄生上流 (電影)", Snippet: "2019年韓國電影"},
			{PageID: 2, Title: "寄生上流", Snippet: "韓國電影"},
			{PageID: 3, Title: "寄生蟲", Snippet: "一種生物"},
		}

		opts := DefaultSearchOptions()
		ranked := searcher.rankResults("寄生上流", results, opts)

		// Exact match should be first
		assert.Equal(t, int64(2), ranked[0].SearchResult.PageID)
		assert.Equal(t, "exact", ranked[0].MatchType)
		assert.Greater(t, ranked[0].Confidence, 0.9)
	})

	t.Run("title contains query ranks second", func(t *testing.T) {
		results := []SearchResult{
			{PageID: 1, Title: "關於寄生上流的一切", Snippet: "電影介紹"},
			{PageID: 2, Title: "完全不相關", Snippet: "寄生上流很好看"},
		}

		opts := DefaultSearchOptions()
		ranked := searcher.rankResults("寄生上流", results, opts)

		// Title contains should rank higher than snippet match
		assert.Equal(t, int64(1), ranked[0].SearchResult.PageID)
		assert.Equal(t, "title_contains", ranked[0].MatchType)
	})

	t.Run("media type boost works", func(t *testing.T) {
		results := []SearchResult{
			{PageID: 1, Title: "寄生上流 相關", Snippet: "相關頁面"},
			{PageID: 2, Title: "寄生上流 (電影)", Snippet: "2019年韓國電影"},
		}

		opts := SearchOptions{
			Limit:     5,
			MediaType: MediaTypeMovie,
		}
		ranked := searcher.rankResults("寄生上流", results, opts)

		// The one with (電影) should get a boost when both have similar base scores
		// PageID 2 has "電影" indicator so it should rank higher than PageID 1
		assert.Equal(t, int64(2), ranked[0].SearchResult.PageID)
	})
}

func TestSearcher_determineMatchType(t *testing.T) {
	searcher := &Searcher{}

	tests := []struct {
		name           string
		query          string
		result         SearchResult
		expectedType   string
	}{
		{
			name:  "exact match",
			query: "寄生上流",
			result: SearchResult{
				Title:   "寄生上流",
				Snippet: "韓國電影",
			},
			expectedType: "exact",
		},
		{
			name:  "title contains",
			query: "寄生上流",
			result: SearchResult{
				Title:   "寄生上流 (電影)",
				Snippet: "韓國電影",
			},
			expectedType: "title_contains",
		},
		{
			name:  "query contains title",
			query: "寄生上流 電影 2019",
			result: SearchResult{
				Title:   "寄生上流",
				Snippet: "韓國電影",
			},
			expectedType: "query_contains",
		},
		{
			name:  "snippet match",
			query: "奉俊昊導演",
			result: SearchResult{
				Title:   "寄生上流",
				Snippet: "由奉俊昊導演執導的電影",
			},
			expectedType: "snippet",
		},
		{
			name:  "fuzzy match",
			query: "完全不同",
			result: SearchResult{
				Title:   "寄生上流",
				Snippet: "韓國電影",
			},
			expectedType: "fuzzy",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			queryLower := normalizeText(tt.query)
			queryNormalized := normalizeText(tt.query)
			result := searcher.determineMatchType(queryLower, queryNormalized, tt.result)
			assert.Equal(t, tt.expectedType, result)
		})
	}
}

func TestRankedResult(t *testing.T) {
	t.Run("correctly stores ranked result data", func(t *testing.T) {
		result := RankedResult{
			SearchResult: SearchResult{
				PageID:    12345,
				Title:     "寄生上流",
				Snippet:   "韓國電影",
				WordCount: 5000,
			},
			Confidence: 0.95,
			MatchType:  "exact",
		}

		assert.Equal(t, int64(12345), result.PageID)
		assert.Equal(t, "寄生上流", result.Title)
		assert.Equal(t, 0.95, result.Confidence)
		assert.Equal(t, "exact", result.MatchType)
	})
}

func TestSearchOptions(t *testing.T) {
	t.Run("custom options work correctly", func(t *testing.T) {
		opts := SearchOptions{
			Limit:                    10,
			MediaType:                MediaTypeTV,
			Year:                     2021,
			PreferTraditionalChinese: false,
		}

		assert.Equal(t, 10, opts.Limit)
		assert.Equal(t, MediaTypeTV, opts.MediaType)
		assert.Equal(t, 2021, opts.Year)
		assert.False(t, opts.PreferTraditionalChinese)
	})
}
