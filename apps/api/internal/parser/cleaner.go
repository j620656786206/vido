package parser

import (
	"regexp"
	"strings"
)

var (
	// Patterns for cleaning titles
	yearCleanPattern     = regexp.MustCompile(`(?:^|[\s(])((19|20)\d{2})(?:[\s)]|$)`)
	qualityCleanPattern  = regexp.MustCompile(`(?i)\b(2160p|4k|uhd|1080p|1080i|720p|576p|480p|sd)\b`)
	sourceCleanPattern   = regexp.MustCompile(`(?i)\b(blu-?ray|bdrip|brrip|web-?dl|webrip|hdtv|pdtv|dsr|dvdrip|dvd|hdcam|cam|telesync|ts|screener|scr|r5)\b`)
	codecCleanPattern    = regexp.MustCompile(`(?i)\b(x264|h\.?264|avc|x265|h\.?265|hevc|av1|xvid|divx)\b`)
	bracketCleanPattern  = regexp.MustCompile(`\[[^\]]*\]|\([^)]*\)`)
	groupCleanPattern    = regexp.MustCompile(`-[A-Za-z0-9]+(?:\.[a-z0-9]{2,4})?$`)
	separatorPattern     = regexp.MustCompile(`[._]+`)
	multiSpacePattern    = regexp.MustCompile(`\s+`)
	whitespacePattern    = regexp.MustCompile(`[\s\t\n\r]+`)
)

// CleanTitle replaces common separators (dots, underscores, dashes) with spaces
// and normalizes whitespace.
func CleanTitle(title string) string {
	if title == "" {
		return ""
	}

	cleaned := title

	// Replace separators with spaces
	cleaned = separatorPattern.ReplaceAllString(cleaned, " ")

	// Replace standalone dashes with spaces (but preserve dashes within words)
	cleaned = strings.ReplaceAll(cleaned, " - ", " ")

	// Normalize whitespace
	cleaned = multiSpacePattern.ReplaceAllString(cleaned, " ")
	cleaned = strings.TrimSpace(cleaned)

	return cleaned
}

// CleanTitleForSearch prepares a title for search queries by removing
// years, quality indicators, sources, codecs, brackets, and release groups.
func CleanTitleForSearch(title string) string {
	if title == "" {
		return ""
	}

	cleaned := title

	// Remove bracketed content first
	cleaned = RemoveBrackets(cleaned)

	// Remove quality indicators
	cleaned = RemoveQualityIndicators(cleaned)

	// Remove years
	cleaned = yearCleanPattern.ReplaceAllString(cleaned, " ")

	// Remove release group
	cleaned = RemoveReleaseGroup(cleaned)

	// Replace separators with spaces
	cleaned = separatorPattern.ReplaceAllString(cleaned, " ")

	// Normalize whitespace
	cleaned = NormalizeWhitespace(cleaned)

	return cleaned
}

// RemoveReleaseGroup removes release group tags from the end of a filename.
// Examples: "-SPARKS", "-YTS.MX"
func RemoveReleaseGroup(title string) string {
	if title == "" {
		return ""
	}

	// Check if the title ends with a pattern like -GROUP or -GROUP.ext
	match := groupCleanPattern.FindStringIndex(title)
	if match == nil {
		return title
	}

	// Check if there's an extension to preserve
	extMatch := regexp.MustCompile(`\.[a-z0-9]{2,4}$`).FindString(title)

	cleaned := title[:match[0]]
	if extMatch != "" {
		cleaned += extMatch
	}

	return strings.TrimSpace(cleaned)
}

// RemoveBrackets removes content within square brackets and parentheses.
func RemoveBrackets(title string) string {
	if title == "" {
		return ""
	}

	cleaned := bracketCleanPattern.ReplaceAllString(title, " ")
	cleaned = NormalizeWhitespace(cleaned)

	return cleaned
}

// RemoveQualityIndicators removes quality, source, and codec indicators.
func RemoveQualityIndicators(title string) string {
	if title == "" {
		return ""
	}

	cleaned := title

	// Remove quality (1080p, 720p, etc.)
	cleaned = qualityCleanPattern.ReplaceAllString(cleaned, " ")

	// Remove source (BluRay, WEB-DL, etc.)
	cleaned = sourceCleanPattern.ReplaceAllString(cleaned, " ")

	// Remove codec (x264, HEVC, etc.)
	cleaned = codecCleanPattern.ReplaceAllString(cleaned, " ")

	// Normalize whitespace
	cleaned = NormalizeWhitespace(cleaned)

	return cleaned
}

// NormalizeWhitespace replaces multiple whitespace characters with a single space
// and trims leading/trailing whitespace.
func NormalizeWhitespace(s string) string {
	if s == "" {
		return ""
	}

	cleaned := whitespacePattern.ReplaceAllString(s, " ")
	return strings.TrimSpace(cleaned)
}
