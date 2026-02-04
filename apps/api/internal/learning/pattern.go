package learning

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

// FilenameMapping represents a learned filename pattern mapping
type FilenameMapping struct {
	ID           string    `json:"id"`
	Pattern      string    `json:"pattern"`
	PatternType  string    `json:"pattern_type"` // "exact", "regex", "fuzzy", "fansub", "standard"
	PatternRegex string    `json:"pattern_regex,omitempty"`
	FansubGroup  string    `json:"fansub_group,omitempty"`
	TitlePattern string    `json:"title_pattern,omitempty"`
	MetadataType string    `json:"metadata_type"` // "movie" or "series"
	MetadataID   string    `json:"metadata_id"`
	TmdbID       int       `json:"tmdb_id,omitempty"`
	Confidence   float64   `json:"confidence"`
	UseCount     int       `json:"use_count"`
	CreatedAt    time.Time `json:"created_at"`
	LastUsedAt   *time.Time `json:"last_used_at,omitempty"`
}

// ExtractedPattern represents the result of extracting a pattern from a filename
type ExtractedPattern struct {
	OriginalFilename string
	FansubGroup      string
	TitlePattern     string
	Regex            string
	PatternType      string // "fansub", "standard", "exact"
}

// PatternExtractor extracts reusable patterns from filenames
type PatternExtractor struct {
	// Regex patterns for extraction
	fansubSquarePattern  *regexp.Regexp
	fansubChinesePattern *regexp.Regexp
	qualityPattern       *regexp.Regexp
	episodePatterns      []*regexp.Regexp
	yearPattern          *regexp.Regexp
	extensionPattern     *regexp.Regexp
	separatorPattern     *regexp.Regexp
	multiSpacePattern    *regexp.Regexp
}

// NewPatternExtractor creates a new pattern extractor
func NewPatternExtractor() *PatternExtractor {
	return &PatternExtractor{
		// Fansub group patterns
		fansubSquarePattern:  regexp.MustCompile(`^\[([^\]]+)\]`),
		fansubChinesePattern: regexp.MustCompile(`^【([^】]+)】`),

		// Quality indicators that should not be treated as fansub groups
		qualityPattern: regexp.MustCompile(`(?i)^(1080p|720p|480p|2160p|4k|uhd|x264|x265|hevc|aac|flac|bd|web|hdtv)$`),

		// Episode number patterns to remove
		episodePatterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)[Ss]\d+[Ee]\d+(?:-?[Ee]?\d+)?`),              // S01E05, S01E05-E06
			regexp.MustCompile(`(?i)\s+-\s+\d{1,3}(?:\s|$|\[)`),                   // - 01, - 100
			regexp.MustCompile(`(?i)(?:Episode|Ep)\.?\s*\d+`),                     // Episode 01, Ep.01
			regexp.MustCompile(`(?i)\d+x\d+`),                                     // 1x05
			regexp.MustCompile(`(?i)第\d+[話话集]?`),                                // 第01話, 第1集
		},

		// Year pattern
		yearPattern: regexp.MustCompile(`(?:^|[.\s_-])((19|20)\d{2})(?:[.\s_-]|$)`),

		// File extension
		extensionPattern: regexp.MustCompile(`\.[a-zA-Z0-9]{2,4}$`),

		// Separators and cleanup
		separatorPattern:  regexp.MustCompile(`[._]+`),
		multiSpacePattern: regexp.MustCompile(`\s+`),
	}
}

// Extract extracts a reusable pattern from a filename
func (e *PatternExtractor) Extract(filename string) (*ExtractedPattern, error) {
	pattern := &ExtractedPattern{
		OriginalFilename: filename,
	}

	// Remove extension first
	nameWithoutExt := e.extensionPattern.ReplaceAllString(filename, "")

	// 1. Try to extract fansub group
	fansubGroup, restAfterFansub := e.extractFansubGroup(nameWithoutExt)
	pattern.FansubGroup = fansubGroup

	// 2. Extract title pattern
	if fansubGroup != "" {
		pattern.TitlePattern = e.extractTitleFromFansub(restAfterFansub)
		pattern.PatternType = "fansub"
	} else {
		pattern.TitlePattern = e.extractTitleFromStandard(nameWithoutExt)
		if pattern.TitlePattern != "" {
			pattern.PatternType = "standard"
		} else {
			pattern.PatternType = "exact"
			pattern.TitlePattern = e.cleanTitle(nameWithoutExt)
		}
	}

	// 3. Generate regex pattern
	if pattern.PatternType != "exact" {
		pattern.Regex = e.generateRegex(pattern)
	}

	return pattern, nil
}

// extractFansubGroup extracts the fansub group from the beginning of a filename
func (e *PatternExtractor) extractFansubGroup(filename string) (group string, rest string) {
	// Try square brackets first
	if matches := e.fansubSquarePattern.FindStringSubmatch(filename); len(matches) > 1 {
		group = matches[1]
		// Check if it's actually a quality indicator, not a fansub
		if e.qualityPattern.MatchString(group) {
			return "", filename
		}
		rest = strings.TrimPrefix(filename, matches[0])
		rest = strings.TrimSpace(rest)
		return group, rest
	}

	// Try Chinese brackets
	if matches := e.fansubChinesePattern.FindStringSubmatch(filename); len(matches) > 1 {
		group = matches[1]
		rest = strings.TrimPrefix(filename, matches[0])
		rest = strings.TrimSpace(rest)
		return group, rest
	}

	return "", filename
}

// extractTitleFromFansub extracts the title from a fansub-formatted filename
// Input: "Kimetsu no Yaiba - 26 (BD 1920x1080 x264 FLAC)" -> "Kimetsu no Yaiba"
func (e *PatternExtractor) extractTitleFromFansub(rest string) string {
	title := rest

	// Remove any trailing bracket content first
	title = regexp.MustCompile(`\s*[\[\(【].*$`).ReplaceAllString(title, "")

	// Handle "Title - Episode" format
	// Find the LAST " - " followed by a number (episode indicator)
	parts := strings.Split(title, " - ")
	if len(parts) >= 2 {
		// Check if the last part starts with a number (episode)
		lastPart := strings.TrimSpace(parts[len(parts)-1])
		if regexp.MustCompile(`^\d+`).MatchString(lastPart) {
			// Rejoin all parts except the last (episode)
			title = strings.Join(parts[:len(parts)-1], " - ")
		}
	}

	// Clean up
	title = strings.TrimSpace(title)
	return title
}

// extractTitleFromStandard extracts title from standard TV/movie format
// Input: "Breaking.Bad.S01E01.720p.BluRay.x264-DEMAND" -> "Breaking Bad"
func (e *PatternExtractor) extractTitleFromStandard(filename string) string {
	title := filename

	// Check if this is actually a standard format (has episode or year markers)
	hasEpisode := false
	for _, ep := range e.episodePatterns {
		if ep.MatchString(filename) {
			hasEpisode = true
			break
		}
	}
	hasYear := e.yearPattern.MatchString(filename)

	// If no episode or year markers, this is not a standard format
	if !hasEpisode && !hasYear {
		return ""
	}

	// Remove quality indicators (1080p, 720p, etc.)
	title = regexp.MustCompile(`(?i)\b\d{3,4}[ip]\b`).ReplaceAllString(title, " ")
	title = regexp.MustCompile(`(?i)\b(4k|uhd|sd)\b`).ReplaceAllString(title, " ")

	// Remove source indicators
	title = regexp.MustCompile(`(?i)\b(blu-?ray|bdrip|brrip|web-?dl|webrip|hdtv|dvdrip|dvd|cam|ts)\b`).ReplaceAllString(title, " ")

	// Remove codec indicators
	title = regexp.MustCompile(`(?i)\b(x264|h\.?264|x265|h\.?265|hevc|av1|xvid|aac|ac3|dts|flac)\b`).ReplaceAllString(title, " ")

	// Remove release group at end
	title = regexp.MustCompile(`(?i)-[a-z0-9]+$`).ReplaceAllString(title, "")

	// Remove episode patterns (S01E01, etc.)
	for _, ep := range e.episodePatterns {
		title = ep.ReplaceAllString(title, " ")
	}

	// Remove year
	title = e.yearPattern.ReplaceAllString(title, " ")

	// Replace separators with spaces
	title = e.separatorPattern.ReplaceAllString(title, " ")

	// Clean up whitespace
	title = e.multiSpacePattern.ReplaceAllString(title, " ")
	title = strings.TrimSpace(title)

	return title
}

// cleanTitle cleans a title for simple exact matching
func (e *PatternExtractor) cleanTitle(filename string) string {
	title := filename

	// Replace separators with spaces
	title = e.separatorPattern.ReplaceAllString(title, " ")

	// Clean up whitespace
	title = e.multiSpacePattern.ReplaceAllString(title, " ")
	title = strings.TrimSpace(title)

	return title
}

// generateRegex generates a regex pattern for matching similar filenames
func (e *PatternExtractor) generateRegex(p *ExtractedPattern) string {
	var parts []string

	if p.FansubGroup != "" {
		// Match fansub group with flexible brackets
		// Escape special regex characters in the group name
		escapedGroup := regexp.QuoteMeta(p.FansubGroup)
		parts = append(parts, fmt.Sprintf(`[\[【]%s[\]】]`, escapedGroup))
	}

	if p.TitlePattern != "" {
		// For standard patterns, convert spaces to flexible separator matching
		// "Breaking Bad" should match "Breaking.Bad", "Breaking_Bad", "Breaking Bad"
		escapedTitle := regexp.QuoteMeta(p.TitlePattern)
		// Replace spaces with flexible separator pattern
		// Note: QuoteMeta doesn't escape spaces, so we replace literal spaces
		flexibleTitle := strings.ReplaceAll(escapedTitle, " ", `[.\s_-]+`)
		parts = append(parts, flexibleTitle)
	}

	// Allow any episode number, quality info, extension
	parts = append(parts, `.*`)

	separator := `\s*`
	if p.PatternType == "standard" {
		separator = `[.\s_-]+`
	}

	return `(?i)` + strings.Join(parts, separator)
}

// MatchesFilename checks if a filename matches this pattern's regex
func (p *ExtractedPattern) MatchesFilename(filename string) bool {
	if p.Regex == "" {
		return false
	}

	re, err := regexp.Compile(p.Regex)
	if err != nil {
		return false
	}

	return re.MatchString(filename)
}

// ToFilenameMapping converts an extracted pattern to a FilenameMapping for storage
func (p *ExtractedPattern) ToFilenameMapping(metadataID, metadataType string, tmdbID int) *FilenameMapping {
	// Build the pattern string for display
	var patternStr string
	if p.FansubGroup != "" {
		patternStr = fmt.Sprintf("[%s] %s", p.FansubGroup, p.TitlePattern)
	} else {
		patternStr = p.TitlePattern
	}

	return &FilenameMapping{
		ID:           uuid.New().String(),
		Pattern:      patternStr,
		PatternType:  p.PatternType,
		PatternRegex: p.Regex,
		FansubGroup:  p.FansubGroup,
		TitlePattern: p.TitlePattern,
		MetadataType: metadataType,
		MetadataID:   metadataID,
		TmdbID:       tmdbID,
		Confidence:   1.0,
		UseCount:     0,
		CreatedAt:    time.Now(),
	}
}
