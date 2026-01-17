package parser

import "regexp"

// Movie patterns for standard naming conventions
var (
	// Standard: Movie.Name.2024.1080p.BluRay.x264-GROUP.mkv
	// Matches: title + year + optional quality/source/codec/group
	moviePatternStandard = regexp.MustCompile(
		`(?i)^(?P<title>.+?)` + // Title (non-greedy)
			`[.\s_-]+` + // Separator
			`(?P<year>(?:19|20)\d{2})` + // Year (1900-2099)
			`(?:[.\s_-]+.+)?` + // Optional rest
			`(?:\.[a-z0-9]{2,4})?$`, // Extension
	)

	// Year in parentheses: Movie Name (2024) 1080p.mkv
	moviePatternParens = regexp.MustCompile(
		`(?i)^(?P<title>.+?)` + // Title (non-greedy)
			`\s*\((?P<year>(?:19|20)\d{2})\)` + // Year in parentheses
			`(?:[.\s_-]+.+)?` + // Optional rest
			`(?:\.[a-z0-9]{2,4})?$`, // Extension
	)

	// TV Show detection patterns (to exclude from movie parsing)
	tvShowPattern = regexp.MustCompile(`(?i)[Ss]\d{1,2}[Ee]\d{1,3}|\d{1,2}x\d{1,3}`)

	// Anime/fansub detection (to exclude from movie parsing)
	animePattern = regexp.MustCompile(`^\[.+?\]`)
)

// TV Show patterns for standard naming conventions
var (
	// Standard: Show.Name.S01E05.720p.WEB-DL.mkv
	tvPatternStandard = regexp.MustCompile(
		`(?i)^(?P<title>.+?)` + // Title (non-greedy)
			`[.\s_-]+` + // Separator
			`[Ss](?P<season>\d{1,2})[Ee](?P<episode>\d{1,3})` + // S01E05
			`(?:-?[Ee]?(?P<episode_end>\d{1,3}))?` + // Optional episode range
			`(?:[.\s_-]+.+)?` + // Optional rest
			`(?:\.[a-z0-9]{2,4})?$`, // Extension
	)

	// Alternative: Show.Name.1x05.mkv
	tvPatternAlt = regexp.MustCompile(
		`(?i)^(?P<title>.+?)` + // Title (non-greedy)
			`[.\s_-]+` + // Separator
			`(?P<season>\d{1,2})x(?P<episode>\d{1,3})` + // 1x05
			`(?:[.\s_-]+.+)?` + // Optional rest
			`(?:\.[a-z0-9]{2,4})?$`, // Extension
	)

	// Daily show: Show.Name.2024.01.15.mkv
	tvPatternDaily = regexp.MustCompile(
		`(?i)^(?P<title>.+?)` + // Title (non-greedy)
			`[.\s_-]+` + // Separator
			`(?P<year>\d{4})[.\s_-](?P<month>\d{2})[.\s_-](?P<day>\d{2})` + // Date
			`(?:[.\s_-]+.+)?` + // Optional rest
			`(?:\.[a-z0-9]{2,4})?$`, // Extension
	)

	// Anime: [Group] Show Name - 01 [1080p].mkv
	tvPatternAnime = regexp.MustCompile(
		`(?i)^\[(?P<group>[^\]]+)\]\s*` + // [Group]
			`(?P<title>.+?)` + // Title
			`\s*-\s*` + // Separator
			`(?P<episode>\d{1,3})` + // Episode
			`(?:[.\s_\[\]-]+.+)?` + // Optional rest
			`(?:\.[a-z0-9]{2,4})?$`, // Extension
	)
)

// Quality patterns
var qualityPattern = regexp.MustCompile(`(?i)(2160p|4k|uhd|1080p|1080i|720p|576p|480p|sd)`)

// Source patterns - Use word boundaries to avoid matching partial words (e.g., "Scrubs" matching "scr")
var sourcePattern = regexp.MustCompile(`(?i)(?:^|[.\s_-])(blu-?ray|bdrip|brrip|web-?dl|webrip|hdtv|pdtv|dsr|dvdrip|dvd|hdcam|cam|telesync|ts|screener|scr|r5)(?:[.\s_-]|$)`)

// Video codec patterns
var videoCodecPattern = regexp.MustCompile(`(?i)(x264|h\.?264|avc|x265|h\.?265|hevc|av1|xvid|divx)`)

// Audio codec patterns
var audioCodecPattern = regexp.MustCompile(`(?i)(aac|ac3|dts|dts-?hd|truehd|atmos|flac|mp3|eac3|dd5\.?1|dd\+?)`)

// Release group pattern - matches -GROUP at the end before extension
var releaseGroupPattern = regexp.MustCompile(`(?i)-([a-z0-9]+)(?:\.[a-z0-9]{2,4})?$`)

// Year pattern for extracting years
// Uses word boundaries to avoid consuming separators between consecutive years
var yearPattern = regexp.MustCompile(`(?:^|[.\s_-])((19|20)\d{2})(?:[.\s_-]|$)`)

// yearPatternSimple finds all 4-digit years (19xx or 20xx) regardless of separators
var yearPatternSimple = regexp.MustCompile(`((?:19|20)\d{2})`)
