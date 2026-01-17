package parser

import (
	"regexp"
	"strconv"
	"strings"
)

// MovieParser parses movie filenames using regex patterns.
type MovieParser struct{}

// NewMovieParser creates a new MovieParser instance.
func NewMovieParser() *MovieParser {
	return &MovieParser{}
}

// Parse attempts to extract movie metadata from a filename.
func (p *MovieParser) Parse(filename string) *ParseResult {
	result := &ParseResult{
		OriginalFilename: filename,
		Status:           ParseStatusNeedsAI,
		MediaType:        MediaTypeUnknown,
	}

	// Skip if it looks like a TV show
	if tvShowPattern.MatchString(filename) {
		return result
	}

	// Skip if it looks like anime/fansub
	if animePattern.MatchString(filename) {
		return result
	}

	// Try standard movie pattern first
	if p.parseStandardFormat(filename, result) {
		return result
	}

	// Try parentheses year format
	if p.parseParensFormat(filename, result) {
		return result
	}

	return result
}

// CanParse returns true if this parser can handle the given filename.
func (p *MovieParser) CanParse(filename string) bool {
	// Exclude TV shows
	if tvShowPattern.MatchString(filename) {
		return false
	}

	// Exclude anime/fansub
	if animePattern.MatchString(filename) {
		return false
	}

	// Must have a year pattern
	return moviePatternStandard.MatchString(filename) || moviePatternParens.MatchString(filename)
}

// parseStandardFormat handles Movie.Name.2024.1080p.BluRay.mkv format
func (p *MovieParser) parseStandardFormat(filename string, result *ParseResult) bool {
	// Find all years in the filename using simple pattern that doesn't consume separators
	allYearIndices := yearPatternSimple.FindAllStringSubmatchIndex(filename, -1)
	if len(allYearIndices) == 0 {
		return false
	}

	// Strategy: Find the release year by checking from the end.
	// The release year is typically followed by quality/source info or end of filename.
	// Years embedded in titles (like "2001 A Space Odyssey" or "Blade Runner 2049")
	// are followed by another year.

	var releaseYear int
	var releaseYearStartIdx int

	for i := len(allYearIndices) - 1; i >= 0; i-- {
		indices := allYearIndices[i]
		yearStartIdx := indices[2] // Start of group 1 (the year digits)
		yearEndIdx := indices[3]   // End of group 1

		yearStr := filename[yearStartIdx:yearEndIdx]
		year, err := strconv.Atoi(yearStr)
		if err != nil || year < 1900 || year > 2099 {
			continue
		}

		// Check what follows this year
		afterYear := ""
		if yearEndIdx < len(filename) {
			afterYear = filename[yearEndIdx:]
		}

		// This is the release year if:
		// 1. It's at the end (only extension follows), OR
		// 2. Quality/source info follows (e.g., .1080p.BluRay), OR
		// 3. It's the last valid year and nothing else looks like a year after it
		isReleaseYear := false

		if afterYear == "" {
			isReleaseYear = true
		} else {
			// Check if what follows looks like technical info, not more title
			afterLower := strings.ToLower(afterYear)
			// If it starts with separator and then quality/source/extension
			if len(afterLower) > 0 && (afterLower[0] == '.' || afterLower[0] == ' ' || afterLower[0] == '_' || afterLower[0] == '-') {
				rest := strings.TrimLeft(afterLower, ".-_ ")
				// Check if rest starts with quality, source, codec, or is just extension
				if rest == "" || qualityPattern.MatchString(rest) || sourcePattern.MatchString(rest) ||
					videoCodecPattern.MatchString(rest) || isJustExtension(rest) {
					isReleaseYear = true
				}
				// Check if there are no more years after this one
				remainingYears := yearPattern.FindAllString(afterYear, -1)
				if len(remainingYears) == 0 {
					isReleaseYear = true
				}
			}
		}

		if isReleaseYear {
			releaseYear = year
			releaseYearStartIdx = yearStartIdx
			break
		}
	}

	if releaseYear == 0 {
		return false
	}

	// Extract title: everything before the release year
	// Account for the separator before the year
	titleEndIdx := releaseYearStartIdx
	if titleEndIdx > 0 && (filename[titleEndIdx-1] == '.' || filename[titleEndIdx-1] == ' ' ||
		filename[titleEndIdx-1] == '_' || filename[titleEndIdx-1] == '-') {
		titleEndIdx--
	}

	titlePart := filename[:titleEndIdx]
	titlePart = strings.TrimRight(titlePart, ".-_ ")

	if titlePart == "" {
		return false
	}

	title := cleanTitleSeparators(titlePart)

	result.Status = ParseStatusSuccess
	result.MediaType = MediaTypeMovie
	result.Title = title
	result.CleanedTitle = title
	result.Year = releaseYear

	// Extract additional metadata
	p.extractQualityInfo(filename, result)

	return true
}

// isJustExtension checks if the string is just a file extension
func isJustExtension(s string) bool {
	extensions := []string{"mkv", "mp4", "avi", "mov", "wmv", "flv", "webm", "m4v"}
	lower := strings.ToLower(s)
	for _, ext := range extensions {
		if lower == ext {
			return true
		}
	}
	return false
}

// parseParensFormat handles Movie Name (2024) 1080p.mkv format
func (p *MovieParser) parseParensFormat(filename string, result *ParseResult) bool {
	match := moviePatternParens.FindStringSubmatch(filename)
	if match == nil {
		return false
	}

	groups := getNamedGroups(moviePatternParens, match)

	titleRaw := groups["title"]
	yearStr := groups["year"]

	if titleRaw == "" || yearStr == "" {
		return false
	}

	year, err := strconv.Atoi(yearStr)
	if err != nil || year < 1900 || year > 2099 {
		return false
	}

	title := cleanTitleSeparators(titleRaw)

	result.Status = ParseStatusSuccess
	result.MediaType = MediaTypeMovie
	result.Title = title
	result.CleanedTitle = title
	result.Year = year

	// Extract additional metadata
	p.extractQualityInfo(filename, result)

	return true
}

// extractTitleWithEmbeddedYear handles movies with years in their title.
// For example, "2001.A.Space.Odyssey.1968" or "Blade.Runner.2049.2017"
func (p *MovieParser) extractTitleWithEmbeddedYear(filename string, releaseYear int) string {
	// Find all year occurrences
	matches := yearPattern.FindAllStringSubmatchIndex(filename, -1)
	if len(matches) < 2 {
		return ""
	}

	// If there are multiple years, the last one is likely the release year
	// Everything before that (including embedded years) is the title
	lastMatch := matches[len(matches)-1]
	yearStartIdx := lastMatch[2] // Start of the year capture group

	// Extract title portion (everything before the release year)
	titlePart := filename[:yearStartIdx]
	titlePart = strings.TrimRight(titlePart, ".-_ ")

	return cleanTitleSeparators(titlePart)
}

// extractQualityInfo extracts quality, source, codec, and release group from filename
func (p *MovieParser) extractQualityInfo(filename string, result *ParseResult) {
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

// getNamedGroups extracts named capture groups from a regex match
func getNamedGroups(re *regexp.Regexp, match []string) map[string]string {
	groups := make(map[string]string)
	for i, name := range re.SubexpNames() {
		if i > 0 && i < len(match) && name != "" {
			groups[name] = match[i]
		}
	}
	return groups
}

// cleanTitleSeparators replaces dots, underscores with spaces and normalizes
func cleanTitleSeparators(title string) string {
	// Replace separators with spaces
	cleaned := strings.NewReplacer(
		".", " ",
		"_", " ",
		"-", " ",
	).Replace(title)

	// Normalize multiple spaces to single space
	spacePattern := regexp.MustCompile(`\s+`)
	cleaned = spacePattern.ReplaceAllString(cleaned, " ")

	return strings.TrimSpace(cleaned)
}

// normalizeQuality converts various quality indicators to standard format
func normalizeQuality(quality string) string {
	lower := strings.ToLower(quality)
	switch {
	case lower == "4k" || lower == "uhd":
		return "2160p"
	case lower == "1080i":
		return "1080p"
	case lower == "sd":
		return "480p"
	default:
		return strings.ToLower(quality)
	}
}

// normalizeSource converts various source indicators to standard format
func normalizeSource(source string) string {
	lower := strings.ToLower(source)
	switch {
	case lower == "bluray" || lower == "blu-ray" || lower == "bdrip" || lower == "brrip":
		return "BluRay"
	case lower == "web-dl" || lower == "webdl":
		return "WEB-DL"
	case lower == "webrip":
		return "WEBRip"
	case lower == "hdtv":
		return "HDTV"
	case lower == "pdtv":
		return "PDTV"
	case lower == "dsr":
		return "DSR"
	case lower == "dvdrip" || lower == "dvd":
		return "DVDRip"
	case lower == "hdcam":
		return "HDCAM"
	case lower == "cam":
		return "CAM"
	case lower == "telesync" || lower == "ts":
		return "TS"
	case lower == "screener" || lower == "scr":
		return "SCR"
	case lower == "r5":
		return "R5"
	default:
		return source
	}
}

// normalizeVideoCodec converts various codec indicators to standard format
func normalizeVideoCodec(codec string) string {
	lower := strings.ToLower(codec)
	switch {
	case lower == "x264" || lower == "h264" || lower == "h.264" || lower == "avc":
		return "x264"
	case lower == "x265" || lower == "h265" || lower == "h.265" || lower == "hevc":
		return "x265"
	case lower == "av1":
		return "AV1"
	case lower == "xvid":
		return "XviD"
	case lower == "divx":
		return "DivX"
	default:
		return codec
	}
}

// normalizeAudioCodec converts various audio codec indicators to standard format
func normalizeAudioCodec(codec string) string {
	lower := strings.ToLower(codec)
	switch {
	case lower == "aac":
		return "AAC"
	case lower == "ac3":
		return "AC3"
	case lower == "dts" || lower == "dts-hd" || lower == "dtshd":
		return "DTS"
	case lower == "truehd":
		return "TrueHD"
	case lower == "atmos":
		return "Atmos"
	case lower == "flac":
		return "FLAC"
	case lower == "mp3":
		return "MP3"
	case lower == "eac3" || lower == "dd+" || lower == "dd5.1" || lower == "dd51":
		return "EAC3"
	default:
		return codec
	}
}
