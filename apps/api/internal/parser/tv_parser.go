package parser

import (
	"regexp"
	"strconv"
)

// Additional TV patterns not in patterns.go
var (
	// Anime with Episode/Ep prefix: Naruto.Episode.01.720p.mkv or One.Piece.Ep.100.mkv
	tvPatternAnimeEp = regexp.MustCompile(
		`(?i)^(?P<title>.+?)` + // Title (non-greedy)
			`[.\s_-]+` + // Separator
			`(?:Episode|Ep)\.?` + // Episode/Ep prefix
			`\s*(?P<episode>\d{1,3})` + // Episode number
			`(?:[.\s_-]+.+)?` + // Optional rest
			`(?:\.[a-z0-9]{2,4})?$`, // Extension
	)

	// Anime with dash before episode number: Attack.on.Titan.-.01.720p.mkv
	tvPatternAnimeDash = regexp.MustCompile(
		`(?i)^(?P<title>.+?)` + // Title (non-greedy)
			`[.\s_]*-[.\s_]*` + // Dash with optional separators (dots, spaces, underscores)
			`(?P<episode>\d{1,3})` + // Episode number
			`(?:[.\s_-]+.+)?` + // Optional rest
			`(?:\.[a-z0-9]{2,4})?$`, // Extension
	)
)

// TVParser parses TV show filenames using regex patterns.
type TVParser struct{}

// NewTVParser creates a new TVParser instance.
func NewTVParser() *TVParser {
	return &TVParser{}
}

// Parse attempts to extract TV show metadata from a filename.
func (p *TVParser) Parse(filename string) *ParseResult {
	result := &ParseResult{
		OriginalFilename: filename,
		Status:           ParseStatusNeedsAI,
		MediaType:        MediaTypeUnknown,
	}

	// Skip anime fansub bracket format - these need AI
	if animePattern.MatchString(filename) {
		return result
	}

	// Try standard S01E05 format first (most common)
	if p.parseStandardFormat(filename, result) {
		return result
	}

	// Try alternative 1x05 format
	if p.parseAltFormat(filename, result) {
		return result
	}

	// Try daily show format (2024.01.15)
	if p.parseDailyFormat(filename, result) {
		return result
	}

	// Try anime Episode/Ep format
	if p.parseAnimeEpFormat(filename, result) {
		return result
	}

	// Try anime dash format (Title - 01)
	if p.parseAnimeDashFormat(filename, result) {
		return result
	}

	return result
}

// CanParse returns true if this parser can handle the given filename.
func (p *TVParser) CanParse(filename string) bool {
	// Skip fansub bracket format
	if animePattern.MatchString(filename) {
		return false
	}

	return tvPatternStandard.MatchString(filename) ||
		tvPatternAlt.MatchString(filename) ||
		tvPatternDaily.MatchString(filename) ||
		tvPatternAnimeEp.MatchString(filename) ||
		tvPatternAnimeDash.MatchString(filename)
}

// parseStandardFormat handles Show.Name.S01E05.720p.WEB-DL.mkv format
func (p *TVParser) parseStandardFormat(filename string, result *ParseResult) bool {
	match := tvPatternStandard.FindStringSubmatch(filename)
	if match == nil {
		return false
	}

	groups := getNamedGroups(tvPatternStandard, match)

	titleRaw := groups["title"]
	seasonStr := groups["season"]
	episodeStr := groups["episode"]
	episodeEndStr := groups["episode_end"]

	if titleRaw == "" || seasonStr == "" || episodeStr == "" {
		return false
	}

	season, err := strconv.Atoi(seasonStr)
	if err != nil {
		return false
	}

	episode, err := strconv.Atoi(episodeStr)
	if err != nil {
		return false
	}

	title := cleanTitleSeparators(titleRaw)

	result.Status = ParseStatusSuccess
	result.MediaType = MediaTypeTVShow
	result.Title = title
	result.CleanedTitle = title
	result.Season = season
	result.Episode = episode

	// Handle episode range
	if episodeEndStr != "" {
		episodeEnd, err := strconv.Atoi(episodeEndStr)
		if err == nil && episodeEnd > episode {
			result.EpisodeEnd = episodeEnd
		}
	}

	// Extract additional metadata
	p.extractQualityInfo(filename, result)

	return true
}

// parseAltFormat handles Show.Name.1x05.mkv format
func (p *TVParser) parseAltFormat(filename string, result *ParseResult) bool {
	match := tvPatternAlt.FindStringSubmatch(filename)
	if match == nil {
		return false
	}

	groups := getNamedGroups(tvPatternAlt, match)

	titleRaw := groups["title"]
	seasonStr := groups["season"]
	episodeStr := groups["episode"]

	if titleRaw == "" || seasonStr == "" || episodeStr == "" {
		return false
	}

	season, err := strconv.Atoi(seasonStr)
	if err != nil {
		return false
	}

	episode, err := strconv.Atoi(episodeStr)
	if err != nil {
		return false
	}

	title := cleanTitleSeparators(titleRaw)

	result.Status = ParseStatusSuccess
	result.MediaType = MediaTypeTVShow
	result.Title = title
	result.CleanedTitle = title
	result.Season = season
	result.Episode = episode

	// Extract additional metadata
	p.extractQualityInfo(filename, result)

	return true
}

// parseDailyFormat handles Show.Name.2024.01.15.mkv format (daily shows)
func (p *TVParser) parseDailyFormat(filename string, result *ParseResult) bool {
	match := tvPatternDaily.FindStringSubmatch(filename)
	if match == nil {
		return false
	}

	groups := getNamedGroups(tvPatternDaily, match)

	titleRaw := groups["title"]
	yearStr := groups["year"]
	monthStr := groups["month"]
	dayStr := groups["day"]

	if titleRaw == "" || yearStr == "" || monthStr == "" || dayStr == "" {
		return false
	}

	year, err := strconv.Atoi(yearStr)
	if err != nil || year < 1900 || year > 2099 {
		return false
	}

	// Validate month and day
	month, err := strconv.Atoi(monthStr)
	if err != nil || month < 1 || month > 12 {
		return false
	}

	day, err := strconv.Atoi(dayStr)
	if err != nil || day < 1 || day > 31 {
		return false
	}

	title := cleanTitleSeparators(titleRaw)

	result.Status = ParseStatusSuccess
	result.MediaType = MediaTypeTVShow
	result.Title = title
	result.CleanedTitle = title
	result.Year = year
	// Note: For daily shows, we store the year but not season/episode

	// Extract additional metadata
	p.extractQualityInfo(filename, result)

	return true
}

// parseAnimeEpFormat handles Naruto.Episode.01.720p.mkv or One.Piece.Ep.100.mkv
func (p *TVParser) parseAnimeEpFormat(filename string, result *ParseResult) bool {
	match := tvPatternAnimeEp.FindStringSubmatch(filename)
	if match == nil {
		return false
	}

	groups := getNamedGroups(tvPatternAnimeEp, match)

	titleRaw := groups["title"]
	episodeStr := groups["episode"]

	if titleRaw == "" || episodeStr == "" {
		return false
	}

	episode, err := strconv.Atoi(episodeStr)
	if err != nil {
		return false
	}

	title := cleanTitleSeparators(titleRaw)

	result.Status = ParseStatusSuccess
	result.MediaType = MediaTypeTVShow
	result.Title = title
	result.CleanedTitle = title
	result.Episode = episode
	// Season defaults to 0 for anime without explicit season

	// Extract additional metadata
	p.extractQualityInfo(filename, result)

	return true
}

// parseAnimeDashFormat handles Attack.on.Titan.-.01.720p.mkv
func (p *TVParser) parseAnimeDashFormat(filename string, result *ParseResult) bool {
	// Check that this doesn't look like S01E01 format first
	if tvShowPattern.MatchString(filename) {
		return false
	}

	match := tvPatternAnimeDash.FindStringSubmatch(filename)
	if match == nil {
		return false
	}

	groups := getNamedGroups(tvPatternAnimeDash, match)

	titleRaw := groups["title"]
	episodeStr := groups["episode"]

	if titleRaw == "" || episodeStr == "" {
		return false
	}

	episode, err := strconv.Atoi(episodeStr)
	if err != nil {
		return false
	}

	title := cleanTitleSeparators(titleRaw)

	result.Status = ParseStatusSuccess
	result.MediaType = MediaTypeTVShow
	result.Title = title
	result.CleanedTitle = title
	result.Episode = episode

	// Extract additional metadata
	p.extractQualityInfo(filename, result)

	return true
}

// extractQualityInfo extracts quality, source, codec, and release group from filename
func (p *TVParser) extractQualityInfo(filename string, result *ParseResult) {
	// Extract quality
	if match := qualityPattern.FindStringSubmatch(filename); match != nil {
		result.Quality = normalizeQuality(match[1])
	}

	// Extract source
	if match := sourcePattern.FindStringSubmatch(filename); match != nil {
		result.Source = normalizeSource(match[1])
	}

	// Extract video codec
	if match := videoCodecPattern.FindStringSubmatch(filename); match != nil {
		result.VideoCodec = normalizeVideoCodec(match[1])
	}

	// Extract audio codec
	if match := audioCodecPattern.FindStringSubmatch(filename); match != nil {
		result.AudioCodec = normalizeAudioCodec(match[1])
	}

	// Extract release group
	if match := releaseGroupPattern.FindStringSubmatch(filename); match != nil {
		result.ReleaseGroup = match[1]
	}
}
