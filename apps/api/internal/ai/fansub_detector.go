// Package ai provides AI provider abstraction for filename parsing.
package ai

import (
	"regexp"
	"strings"
)

// FansubBracketType represents different bracket styles used by fansub groups.
type FansubBracketType string

const (
	// BracketSquare represents standard ASCII square brackets [].
	BracketSquare FansubBracketType = "square"
	// BracketFullwidth represents fullwidth CJK brackets【】.
	BracketFullwidth FansubBracketType = "fullwidth"
	// BracketCorner represents corner brackets「」(less common).
	BracketCorner FansubBracketType = "corner"
	// BracketNone indicates no bracket pattern detected.
	BracketNone FansubBracketType = "none"
)

// Fansub detection patterns.
var (
	// squareBracketPattern matches ASCII square brackets at start: [GroupName].
	squareBracketPattern = regexp.MustCompile(`^\s*\[[^\]]+\]`)

	// fullwidthBracketPattern matches fullwidth brackets:【字幕組】.
	fullwidthBracketPattern = regexp.MustCompile(`^\s*【[^】]+】`)

	// cornerBracketPattern matches corner brackets:「字幕組」.
	cornerBracketPattern = regexp.MustCompile(`^\s*「[^」]+」`)

	// anyBracketStartPattern matches any bracket type at the start.
	anyBracketStartPattern = regexp.MustCompile(`^\s*[\[【「]`)

	// chineseEpisodePattern matches Chinese episode notation: 第XX話, 第XX集, 第XX话.
	chineseEpisodePattern = regexp.MustCompile(`第\s*\d+\s*[話集话]`)

	// koreanEpisodePattern matches Korean episode notation: 제XX화.
	koreanEpisodePattern = regexp.MustCompile(`제\s*\d+\s*화`)

	// episodeDashPattern matches "- XX" episode pattern (pre-compiled for performance).
	episodeDashPattern = regexp.MustCompile(`\s+-\s*\d{1,3}(\s|\.|\[|\(|$)`)

	// knownFansubGroups contains common fansub group names for quick matching.
	knownFansubGroups = []string{
		// Japanese raws
		"Leopard-Raws", "SubsPlease", "Erai-raws", "Commie", "HorribleSubs",
		"ANK-Raws", "VCB-Studio", "DHD", "Moozzi2", "U3-Web",
		// Chinese groups
		"幻櫻字幕組", "极影字幕社", "動漫國字幕組", "华盟字幕社", "天使动漫论坛",
		"喵萌奶茶屋", "悠哈璃羽字幕社", "诸神字幕组", "风车字幕组",
	}
)

// FansubDetectionResult contains details about detected fansub patterns.
type FansubDetectionResult struct {
	// IsFansub indicates if the filename appears to be a fansub release.
	IsFansub bool `json:"is_fansub"`
	// BracketType indicates the bracket style detected.
	BracketType FansubBracketType `json:"bracket_type"`
	// GroupName is the extracted fansub group name (if detected).
	GroupName string `json:"group_name,omitempty"`
	// HasChineseEpisode indicates Chinese episode notation detected.
	HasChineseEpisode bool `json:"has_chinese_episode"`
	// HasKnownGroup indicates a known fansub group was detected.
	HasKnownGroup bool `json:"has_known_group"`
	// Confidence is a score from 0.0 to 1.0 indicating detection confidence.
	Confidence float64 `json:"confidence"`
}

// IsFansubFilename checks if a filename appears to be a fansub release.
// It detects common fansub naming patterns including bracket styles and episode notation.
func IsFansubFilename(filename string) bool {
	result := DetectFansub(filename)
	return result.IsFansub
}

// DetectFansub performs detailed fansub pattern detection on a filename.
// Returns detailed information about detected fansub patterns.
func DetectFansub(filename string) *FansubDetectionResult {
	result := &FansubDetectionResult{
		BracketType: BracketNone,
		Confidence:  0.0,
	}

	// Check for bracket patterns at start
	bracketType, groupName := detectBrackets(filename)
	result.BracketType = bracketType
	result.GroupName = groupName

	// Check for Chinese/Korean episode notation
	result.HasChineseEpisode = hasChineseEpisodeNotation(filename)

	// Check for known fansub groups
	result.HasKnownGroup = hasKnownFansubGroup(filename)

	// Calculate confidence and determine if it's a fansub
	result.Confidence = calculateFansubConfidence(result, filename)
	result.IsFansub = result.Confidence >= 0.5

	return result
}

// detectBrackets identifies the bracket type and extracts the group name.
func detectBrackets(filename string) (FansubBracketType, string) {
	// Check fullwidth brackets first (most specific to CJK fansubs)
	if matches := fullwidthBracketPattern.FindString(filename); matches != "" {
		groupName := extractGroupName(matches, '【', '】')
		return BracketFullwidth, groupName
	}

	// Check corner brackets
	if matches := cornerBracketPattern.FindString(filename); matches != "" {
		groupName := extractGroupName(matches, '「', '」')
		return BracketCorner, groupName
	}

	// Check square brackets
	if matches := squareBracketPattern.FindString(filename); matches != "" {
		groupName := extractGroupName(matches, '[', ']')
		return BracketSquare, groupName
	}

	return BracketNone, ""
}

// extractGroupName extracts the text between brackets.
func extractGroupName(s string, open, close rune) string {
	start := strings.IndexRune(s, open)
	end := strings.IndexRune(s, close)
	if start != -1 && end != -1 && end > start {
		return strings.TrimSpace(s[start+len(string(open)) : end])
	}
	return ""
}

// hasChineseEpisodeNotation checks for Chinese or Korean episode notation.
func hasChineseEpisodeNotation(filename string) bool {
	return chineseEpisodePattern.MatchString(filename) ||
		koreanEpisodePattern.MatchString(filename)
}

// hasKnownFansubGroup checks if the filename contains a known fansub group name.
func hasKnownFansubGroup(filename string) bool {
	lowerFilename := strings.ToLower(filename)
	for _, group := range knownFansubGroups {
		if strings.Contains(lowerFilename, strings.ToLower(group)) {
			return true
		}
	}
	return false
}

// calculateFansubConfidence determines confidence that the filename is a fansub.
func calculateFansubConfidence(result *FansubDetectionResult, filename string) float64 {
	confidence := 0.0

	// Bracket type is a strong indicator
	switch result.BracketType {
	case BracketFullwidth:
		confidence += 0.5 // Fullwidth is very specific to CJK fansubs
	case BracketCorner:
		confidence += 0.4
	case BracketSquare:
		confidence += 0.3 // Square brackets could be scene releases too
	}

	// Chinese episode notation is a strong indicator
	if result.HasChineseEpisode {
		confidence += 0.3
	}

	// Known fansub group is definitive
	if result.HasKnownGroup {
		confidence += 0.4
	}

	// Additional heuristics
	if containsCJKCharacters(filename) && result.BracketType != BracketNone {
		confidence += 0.1
	}

	// Episode dash pattern with brackets suggests fansub: [Group] Title - 01
	if result.BracketType != BracketNone && containsEpisodeDashPattern(filename) {
		confidence += 0.1
	}

	// Cap at 1.0
	if confidence > 1.0 {
		confidence = 1.0
	}

	return confidence
}

// containsCJKCharacters checks if the string contains CJK characters.
func containsCJKCharacters(s string) bool {
	for _, r := range s {
		// CJK Unified Ideographs and common extensions
		if (r >= 0x4E00 && r <= 0x9FFF) || // CJK Unified Ideographs
			(r >= 0x3400 && r <= 0x4DBF) || // CJK Extension A
			(r >= 0x3040 && r <= 0x309F) || // Hiragana
			(r >= 0x30A0 && r <= 0x30FF) || // Katakana
			(r >= 0xAC00 && r <= 0xD7AF) { // Hangul Syllables
			return true
		}
	}
	return false
}

// containsEpisodeDashPattern checks for "- XX" episode pattern.
func containsEpisodeDashPattern(s string) bool {
	return episodeDashPattern.MatchString(s)
}

// GetKnownFansubGroups returns the list of known fansub group names.
// Useful for testing and documentation.
func GetKnownFansubGroups() []string {
	groups := make([]string, len(knownFansubGroups))
	copy(groups, knownFansubGroups)
	return groups
}
