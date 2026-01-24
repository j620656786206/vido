package wikipedia

import (
	"regexp"
	"strings"
	"unicode"
)

// ContentExtractor extracts and cleans content from Wikipedia pages
type ContentExtractor struct{}

// NewContentExtractor creates a new content extractor
func NewContentExtractor() *ContentExtractor {
	return &ContentExtractor{}
}

// ExtractSummary extracts a clean summary from page content
func (e *ContentExtractor) ExtractSummary(content *PageContent) string {
	// Prefer the pre-extracted plain text extract
	if content.Extract != "" {
		return e.cleanText(content.Extract)
	}

	// Fallback to extracting from wikitext
	if content.Wikitext != "" {
		return e.extractSummaryFromWikitext(content.Wikitext)
	}

	return ""
}

// extractSummaryFromWikitext extracts the first paragraph from wikitext
func (e *ContentExtractor) extractSummaryFromWikitext(wikitext string) string {
	// Remove templates (including nested ones)
	text := e.removeTemplates(wikitext)

	// Split into lines and find the first non-empty paragraph
	lines := strings.Split(text, "\n")
	var paragraphs []string
	var currentParagraph strings.Builder

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Skip empty lines, headings, and special lines
		if line == "" {
			if currentParagraph.Len() > 0 {
				paragraphs = append(paragraphs, currentParagraph.String())
				currentParagraph.Reset()
			}
			continue
		}

		// Skip headings (== Heading ==)
		if strings.HasPrefix(line, "=") && strings.HasSuffix(line, "=") {
			continue
		}

		// Skip lists and other special formatting
		if strings.HasPrefix(line, "*") || strings.HasPrefix(line, "#") ||
			strings.HasPrefix(line, ":") || strings.HasPrefix(line, ";") {
			continue
		}

		// Skip categories and files
		if strings.HasPrefix(strings.ToLower(line), "[[category:") ||
			strings.HasPrefix(strings.ToLower(line), "[[file:") ||
			strings.HasPrefix(strings.ToLower(line), "[[image:") {
			continue
		}

		if currentParagraph.Len() > 0 {
			currentParagraph.WriteString(" ")
		}
		currentParagraph.WriteString(line)
	}

	// Add the last paragraph if any
	if currentParagraph.Len() > 0 {
		paragraphs = append(paragraphs, currentParagraph.String())
	}

	// Return the first substantial paragraph
	for _, p := range paragraphs {
		cleaned := e.CleanWikitext(p)
		if len(cleaned) > 50 { // Minimum length for a useful summary
			return cleaned
		}
	}

	// If no substantial paragraph found, return the first one
	if len(paragraphs) > 0 {
		return e.CleanWikitext(paragraphs[0])
	}

	return ""
}

// CleanWikitext removes all wiki markup from text
func (e *ContentExtractor) CleanWikitext(text string) string {
	// Remove wiki links: [[Link|Text]] → Text or [[Link]] → Link
	text = e.cleanWikiLinks(text)

	// Remove external links: [http://... text] → text
	text = e.cleanExternalLinks(text)

	// Remove templates: {{...}} → ""
	text = e.removeTemplates(text)

	// Remove HTML tags and comments
	text = e.cleanHTML(text)

	// Clean up formatting
	text = e.cleanFormatting(text)

	// Clean up whitespace
	text = e.cleanWhitespace(text)

	return text
}

// cleanWikiLinks removes wiki link syntax
func (e *ContentExtractor) cleanWikiLinks(text string) string {
	// [[Link|Display text]] → Display text
	// [[Link]] → Link
	linkRe := regexp.MustCompile(`\[\[(?:[^|\]]*\|)?([^\]]+)\]\]`)
	return linkRe.ReplaceAllString(text, "$1")
}

// cleanExternalLinks removes external link syntax
func (e *ContentExtractor) cleanExternalLinks(text string) string {
	// [http://url text] → text
	// [http://url] → ""
	linkWithTextRe := regexp.MustCompile(`\[https?://[^\s\]]+\s+([^\]]+)\]`)
	text = linkWithTextRe.ReplaceAllString(text, "$1")

	linkOnlyRe := regexp.MustCompile(`\[https?://[^\]]+\]`)
	text = linkOnlyRe.ReplaceAllString(text, "")

	return text
}

// removeTemplates removes template syntax (including nested)
func (e *ContentExtractor) removeTemplates(text string) string {
	// First, handle special link templates that should preserve display text
	// {{link-en|display text|article name}} → display text
	// {{lang-ko|text}} → text
	linkEnRe := regexp.MustCompile(`\{\{link-[a-z]+\|([^|{}]+)\|[^{}]+\}\}`)
	text = linkEnRe.ReplaceAllString(text, "$1")

	langRe := regexp.MustCompile(`\{\{lang(?:-[a-z]+)?\|([^|{}]+)\}\}`)
	text = langRe.ReplaceAllString(text, "$1")

	// Keep removing templates until none remain
	// This handles nested templates
	for {
		// Find innermost templates (no nested {{ }})
		templateRe := regexp.MustCompile(`\{\{[^{}]*\}\}`)
		newText := templateRe.ReplaceAllString(text, "")

		if newText == text {
			break
		}
		text = newText
	}

	return text
}

// cleanHTML removes HTML tags and comments
func (e *ContentExtractor) cleanHTML(text string) string {
	// Remove HTML comments: <!-- ... -->
	commentRe := regexp.MustCompile(`<!--[\s\S]*?-->`)
	text = commentRe.ReplaceAllString(text, "")

	// Remove HTML tags: <tag> or </tag> or <tag />
	tagRe := regexp.MustCompile(`<[^>]+>`)
	text = tagRe.ReplaceAllString(text, "")

	// Decode common HTML entities
	text = strings.ReplaceAll(text, "&nbsp;", " ")
	text = strings.ReplaceAll(text, "&ndash;", "-")
	text = strings.ReplaceAll(text, "&mdash;", "—")
	text = strings.ReplaceAll(text, "&amp;", "&")
	text = strings.ReplaceAll(text, "&lt;", "<")
	text = strings.ReplaceAll(text, "&gt;", ">")
	text = strings.ReplaceAll(text, "&quot;", "\"")
	text = strings.ReplaceAll(text, "&#39;", "'")

	return text
}

// cleanFormatting removes wiki formatting markup
func (e *ContentExtractor) cleanFormatting(text string) string {
	// Remove bold/italic: '''text''' or ''text''
	text = strings.ReplaceAll(text, "'''", "")
	text = strings.ReplaceAll(text, "''", "")

	// Remove references: <ref>...</ref> or <ref name="..."/>
	refRe := regexp.MustCompile(`<ref[^>]*>.*?</ref>|<ref[^/]*/>`)
	text = refRe.ReplaceAllString(text, "")

	return text
}

// cleanWhitespace normalizes whitespace
func (e *ContentExtractor) cleanWhitespace(text string) string {
	// Replace multiple spaces with single space
	spaceRe := regexp.MustCompile(`\s+`)
	text = spaceRe.ReplaceAllString(text, " ")

	return strings.TrimSpace(text)
}

// cleanText applies basic text cleanup
func (e *ContentExtractor) cleanText(text string) string {
	// Remove any stray template or link markers
	text = e.cleanHTML(text)
	text = e.cleanWhitespace(text)
	return text
}

// ExtractCategories extracts category names from wikitext
func (e *ContentExtractor) ExtractCategories(wikitext string) []string {
	var categories []string

	// Match [[Category:CategoryName]] or [[分類:CategoryName]]
	categoryRe := regexp.MustCompile(`\[\[(?:Category|分類|分类):([^\]|]+)(?:\|[^\]]*)?\]\]`)
	matches := categoryRe.FindAllStringSubmatch(wikitext, -1)

	for _, match := range matches {
		if len(match) > 1 {
			category := strings.TrimSpace(match[1])
			if category != "" {
				categories = append(categories, category)
			}
		}
	}

	return categories
}

// ExtractInterwikiLinks extracts interwiki links (links to other language Wikipedias)
func (e *ContentExtractor) ExtractInterwikiLinks(wikitext string) map[string]string {
	links := make(map[string]string)

	// Match [[en:English Title]] or [[ja:Japanese Title]]
	interwikiRe := regexp.MustCompile(`\[\[([a-z]{2,3}):([^\]]+)\]\]`)
	matches := interwikiRe.FindAllStringSubmatch(wikitext, -1)

	for _, match := range matches {
		if len(match) > 2 {
			lang := match[1]
			title := strings.TrimSpace(match[2])
			if title != "" {
				links[lang] = title
			}
		}
	}

	return links
}

// TruncateSummary truncates a summary to the specified length
func (e *ContentExtractor) TruncateSummary(summary string, maxLength int) string {
	if len(summary) <= maxLength {
		return summary
	}

	// Find a good break point (end of sentence or word)
	runes := []rune(summary)
	if len(runes) <= maxLength {
		return summary
	}

	truncated := runes[:maxLength]

	// Try to find a sentence end (handles both ASCII and fullwidth punctuation)
	for i := len(truncated) - 1; i >= len(truncated)-50 && i >= 0; i-- {
		r := truncated[i]
		if r == '。' || r == '.' || // Period
			r == '！' || r == '!' || // Exclamation (fullwidth and ASCII)
			r == '？' || r == '?' { // Question (fullwidth and ASCII)
			return string(truncated[:i+1])
		}
	}

	// Find a word break
	for i := len(truncated) - 1; i >= len(truncated)-20 && i >= 0; i-- {
		if unicode.IsSpace(truncated[i]) {
			return string(truncated[:i]) + "..."
		}
	}

	return string(truncated) + "..."
}
