package wikipedia

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestContentExtractor_CleanWikitext(t *testing.T) {
	extractor := NewContentExtractor()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "removes wiki links with display text",
			input:    "Directed by [[奉俊昊|Bong Joon-ho]]",
			expected: "Directed by Bong Joon-ho",
		},
		{
			name:     "removes wiki links without display text",
			input:    "Directed by [[奉俊昊]]",
			expected: "Directed by 奉俊昊",
		},
		{
			name:     "removes templates",
			input:    "Released {{Film date|2019|5|21}}",
			expected: "Released",
		},
		{
			name:     "removes HTML tags",
			input:    "<small>small text</small>",
			expected: "small text",
		},
		{
			name:     "removes bold and italic",
			input:    "'''Bold''' and ''italic'' text",
			expected: "Bold and italic text",
		},
		{
			name:     "removes external links with text",
			input:    "See [http://example.com Example Site] for more",
			expected: "See Example Site for more",
		},
		{
			name:     "removes external links without text",
			input:    "Visit [http://example.com]",
			expected: "Visit",
		},
		{
			name:     "cleans HTML entities",
			input:    "Rock&nbsp;&amp;&nbsp;Roll",
			expected: "Rock & Roll",
		},
		{
			name:     "normalizes whitespace",
			input:    "Multiple   spaces   here",
			expected: "Multiple spaces here",
		},
		{
			name:     "complex wikitext",
			input:    "'''《寄生上流》'''（{{lang-ko|기생충}}）是[[奉俊昊]]執導的{{link-en|韓國電影|Korean cinema}}。",
			expected: "《寄生上流》（기생충）是奉俊昊執導的韓國電影。",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractor.CleanWikitext(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestContentExtractor_ExtractSummary(t *testing.T) {
	extractor := NewContentExtractor()

	t.Run("uses extract if available", func(t *testing.T) {
		content := &PageContent{
			Extract:  "This is the plain text extract.",
			Wikitext: "{{Infobox}}\nThis is wikitext.",
		}

		summary := extractor.ExtractSummary(content)

		assert.Equal(t, "This is the plain text extract.", summary)
	})

	t.Run("extracts from wikitext if no extract", func(t *testing.T) {
		content := &PageContent{
			Wikitext: `{{Infobox film
| name = Test Movie
}}
'''Test Movie''' is a 2020 film directed by [[John Doe]]. It tells the story of a young hero who saves the world from evil.

== Plot ==
The movie begins with...`,
		}

		summary := extractor.ExtractSummary(content)

		assert.Contains(t, summary, "Test Movie")
		assert.Contains(t, summary, "John Doe")
		assert.NotContains(t, summary, "[[")
		assert.NotContains(t, summary, "{{")
	})

	t.Run("returns empty for empty content", func(t *testing.T) {
		content := &PageContent{}

		summary := extractor.ExtractSummary(content)

		assert.Empty(t, summary)
	})
}

func TestContentExtractor_cleanWikiLinks(t *testing.T) {
	extractor := NewContentExtractor()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple link",
			input:    "[[Link]]",
			expected: "Link",
		},
		{
			name:     "link with display text",
			input:    "[[Link|Display]]",
			expected: "Display",
		},
		{
			name:     "multiple links",
			input:    "[[Link1]] and [[Link2|Text2]]",
			expected: "Link1 and Text2",
		},
		{
			name:     "chinese link",
			input:    "[[奉俊昊]]導演的電影",
			expected: "奉俊昊導演的電影",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractor.cleanWikiLinks(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestContentExtractor_removeTemplates(t *testing.T) {
	extractor := NewContentExtractor()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple template",
			input:    "Before {{template}} after",
			expected: "Before  after",
		},
		{
			name:     "template with parameters",
			input:    "{{Film date|2019|5|21}}",
			expected: "",
		},
		{
			name:     "nested templates",
			input:    "{{outer|{{inner}}}}",
			expected: "",
		},
		{
			name:     "multiple templates",
			input:    "{{A}} text {{B}}",
			expected: " text ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractor.removeTemplates(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestContentExtractor_cleanHTML(t *testing.T) {
	extractor := NewContentExtractor()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple tag",
			input:    "<b>bold</b>",
			expected: "bold",
		},
		{
			name:     "self-closing tag",
			input:    "line<br/>break",
			expected: "linebreak",
		},
		{
			name:     "HTML comment",
			input:    "visible<!-- hidden -->more",
			expected: "visiblemore",
		},
		{
			name:     "HTML entities",
			input:    "&nbsp;&amp;&lt;&gt;",
			expected: " &<>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractor.cleanHTML(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestContentExtractor_ExtractCategories(t *testing.T) {
	extractor := NewContentExtractor()

	t.Run("extracts english categories", func(t *testing.T) {
		wikitext := `[[Category:2019 films]]
[[Category:South Korean films]]
[[Category:Korean-language films]]`

		categories := extractor.ExtractCategories(wikitext)

		assert.Contains(t, categories, "2019 films")
		assert.Contains(t, categories, "South Korean films")
		assert.Len(t, categories, 3)
	})

	t.Run("extracts chinese categories", func(t *testing.T) {
		wikitext := `[[分類:2019年電影]]
[[分類:韓國電影]]`

		categories := extractor.ExtractCategories(wikitext)

		assert.Contains(t, categories, "2019年電影")
		assert.Contains(t, categories, "韓國電影")
	})

	t.Run("handles categories with sort keys", func(t *testing.T) {
		wikitext := `[[Category:Films|Parasite]]`

		categories := extractor.ExtractCategories(wikitext)

		assert.Contains(t, categories, "Films")
	})

	t.Run("returns empty for no categories", func(t *testing.T) {
		wikitext := "Just regular text"

		categories := extractor.ExtractCategories(wikitext)

		assert.Empty(t, categories)
	})
}

func TestContentExtractor_ExtractInterwikiLinks(t *testing.T) {
	extractor := NewContentExtractor()

	t.Run("extracts interwiki links", func(t *testing.T) {
		wikitext := `[[en:Parasite (2019 film)]]
[[ja:パラサイト 半地下の家族]]
[[ko:기생충 (영화)]]`

		links := extractor.ExtractInterwikiLinks(wikitext)

		assert.Equal(t, "Parasite (2019 film)", links["en"])
		assert.Equal(t, "パラサイト 半地下の家族", links["ja"])
		assert.Equal(t, "기생충 (영화)", links["ko"])
	})

	t.Run("returns empty for no interwiki", func(t *testing.T) {
		wikitext := "Just regular text"

		links := extractor.ExtractInterwikiLinks(wikitext)

		assert.Empty(t, links)
	})
}

func TestContentExtractor_TruncateSummary(t *testing.T) {
	extractor := NewContentExtractor()

	t.Run("does not truncate short text", func(t *testing.T) {
		text := "Short text"

		result := extractor.TruncateSummary(text, 100)

		assert.Equal(t, "Short text", result)
	})

	t.Run("truncates at sentence end", func(t *testing.T) {
		text := "First sentence. Second sentence. Third sentence."

		result := extractor.TruncateSummary(text, 30)

		assert.Equal(t, "First sentence.", result)
	})

	t.Run("truncates chinese at sentence end", func(t *testing.T) {
		text := "這是第一句話。這是第二句話。這是第三句話。"

		result := extractor.TruncateSummary(text, 10) // 10 runes

		// Should truncate at first sentence
		assert.Contains(t, result, "。")
		// Check rune length is reasonable
		assert.True(t, len([]rune(result)) <= 15, "Result should be reasonably short")
	})

	t.Run("adds ellipsis for word break", func(t *testing.T) {
		text := "This is a long sentence without any periods"

		result := extractor.TruncateSummary(text, 20)

		assert.Contains(t, result, "...")
	})

	// Additional edge case tests
	t.Run("empty string returns empty", func(t *testing.T) {
		result := extractor.TruncateSummary("", 100)
		assert.Equal(t, "", result)
	})

	t.Run("exact length returns unchanged", func(t *testing.T) {
		text := "Exactly10!" // 10 characters
		result := extractor.TruncateSummary(text, 10)
		assert.Equal(t, "Exactly10!", result)
	})

	t.Run("single character under limit", func(t *testing.T) {
		result := extractor.TruncateSummary("A", 100)
		assert.Equal(t, "A", result)
	})

	t.Run("handles chinese exclamation mark", func(t *testing.T) {
		text := "這是一個測試！這是另一個測試。"
		result := extractor.TruncateSummary(text, 8)
		// Should truncate at ! (index 7)
		assert.Equal(t, "這是一個測試！", result)
	})

	t.Run("handles chinese question mark", func(t *testing.T) {
		text := "這是問題嗎？這是答案。"
		result := extractor.TruncateSummary(text, 7)
		// Should truncate at fullwidth ？ (index 5, included in result)
		assert.Equal(t, "這是問題嗎？", result)
	})

	t.Run("multi-byte truncation preserves valid UTF-8", func(t *testing.T) {
		text := "日本語テスト文字列です。"
		result := extractor.TruncateSummary(text, 5)
		// Result should be valid UTF-8
		assert.NotEmpty(t, result)
		// Should not have broken characters
		for _, r := range result {
			assert.True(t, r != 0xFFFD, "Should not contain replacement character")
		}
	})

	t.Run("very short max length", func(t *testing.T) {
		text := "Hello World"
		result := extractor.TruncateSummary(text, 3)
		// Should return something with ellipsis or truncated
		assert.NotEmpty(t, result)
		assert.True(t, len([]rune(result)) <= 6, "Result should be reasonably short")
	})
}

func TestContentExtractor_cleanFormatting(t *testing.T) {
	extractor := NewContentExtractor()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "removes bold",
			input:    "'''Bold text'''",
			expected: "Bold text",
		},
		{
			name:     "removes italic",
			input:    "''Italic text''",
			expected: "Italic text",
		},
		{
			name:     "removes mixed bold and italic",
			input:    "'''''Both'''''",
			expected: "Both",
		},
		{
			name:     "removes references",
			input:    "Text<ref>Reference</ref> more",
			expected: "Text more",
		},
		{
			name:     "removes self-closing ref",
			input:    "Text<ref name=\"test\"/> more",
			expected: "Text more",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractor.cleanFormatting(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
