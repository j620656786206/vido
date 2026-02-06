// Package parser provides filename parsing functionality for media files.
// It extracts metadata such as title, year, quality, and episode information
// from standardized filename formats.
package parser

// ParseStatus represents the result status of a parsing operation.
type ParseStatus string

const (
	// ParseStatusSuccess indicates the filename was successfully parsed.
	ParseStatusSuccess ParseStatus = "success"
	// ParseStatusParsing indicates parsing is currently in progress (for async/UI updates).
	ParseStatusParsing ParseStatus = "parsing"
	// ParseStatusNeedsAI indicates the filename could not be parsed with
	// standard patterns and requires AI-powered parsing.
	ParseStatusNeedsAI ParseStatus = "needs_ai"
	// ParseStatusFailed indicates the parsing operation failed completely.
	ParseStatusFailed ParseStatus = "failed"
)

// MetadataSource indicates the method used to parse the filename.
type MetadataSource string

const (
	// MetadataSourceRegex indicates metadata was extracted using regex patterns.
	MetadataSourceRegex MetadataSource = "regex"
	// MetadataSourceRegexFallback indicates regex was used as fallback when AI was unavailable.
	MetadataSourceRegexFallback MetadataSource = "regex_fallback"
	// MetadataSourceAI indicates metadata was extracted using generic AI parsing.
	MetadataSourceAI MetadataSource = "ai"
	// MetadataSourceAIFansub indicates metadata was extracted using specialized AI fansub parser.
	MetadataSourceAIFansub MetadataSource = "ai_fansub"
	// MetadataSourceManual indicates metadata was manually entered.
	MetadataSourceManual MetadataSource = "manual"
	// MetadataSourceLearned indicates metadata was applied from a learned pattern.
	MetadataSourceLearned MetadataSource = "learned"
)

// MediaType represents the type of media identified from the filename.
type MediaType string

const (
	// MediaTypeMovie indicates the file is a movie.
	MediaTypeMovie MediaType = "movie"
	// MediaTypeTVShow indicates the file is a TV show episode.
	MediaTypeTVShow MediaType = "tv"
	// MediaTypeUnknown indicates the media type could not be determined.
	MediaTypeUnknown MediaType = "unknown"
)

// ParseResult contains all metadata extracted from a filename.
type ParseResult struct {
	// OriginalFilename is the input filename that was parsed.
	OriginalFilename string `json:"original_filename"`

	// Status indicates whether parsing succeeded, needs AI, or failed.
	Status ParseStatus `json:"status"`
	// MediaType indicates if this is a movie, TV show, or unknown.
	MediaType MediaType `json:"media_type"`

	// Title is the extracted media title (may contain separators).
	Title string `json:"title"`
	// CleanedTitle is the title prepared for search queries.
	CleanedTitle string `json:"cleaned_title"`
	// Year is the release year (typically 1900-2099).
	Year int `json:"year,omitempty"`

	// Season is the season number (TV shows only).
	Season int `json:"season,omitempty"`
	// Episode is the episode number (TV shows only).
	Episode int `json:"episode,omitempty"`
	// EpisodeEnd is the end episode for ranges (e.g., E01-E03).
	EpisodeEnd int `json:"episode_end,omitempty"`

	// Quality is the video resolution (e.g., "1080p", "720p", "2160p").
	Quality string `json:"quality,omitempty"`
	// Source is the release source (e.g., "BluRay", "WEB-DL", "HDTV").
	Source string `json:"source,omitempty"`
	// VideoCodec is the video encoding format (e.g., "x264", "x265", "AV1").
	VideoCodec string `json:"video_codec,omitempty"`
	// AudioCodec is the audio encoding format (e.g., "AAC", "DTS", "Atmos").
	AudioCodec string `json:"audio_codec,omitempty"`

	// ReleaseGroup is the group that released the file (e.g., "SPARKS", "YTS").
	ReleaseGroup string `json:"release_group,omitempty"`

	// Language indicates subtitle/audio language if detected.
	Language string `json:"language,omitempty"`

	// Confidence is a score from 0-100 indicating parsing reliability.
	Confidence int `json:"confidence"`

	// ErrorMessage contains details if parsing failed.
	ErrorMessage string `json:"error_message,omitempty"`

	// MetadataSource indicates which method was used to parse the filename.
	MetadataSource MetadataSource `json:"metadata_source,omitempty"`

	// ParseDurationMs is the time taken to parse the filename in milliseconds.
	ParseDurationMs int64 `json:"parse_duration_ms,omitempty"`

	// AIProvider is the AI provider used if metadata was extracted via AI.
	AIProvider string `json:"ai_provider,omitempty"`

	// LearnedPatternID is the ID of the learned pattern that matched (if MetadataSource is "learned").
	LearnedPatternID string `json:"learned_pattern_id,omitempty"`

	// LearnedTmdbID is the TMDb ID from the learned pattern.
	LearnedTmdbID int `json:"learned_tmdb_id,omitempty"`

	// LearnedMetadataID is the metadata ID (movie/series) from the learned pattern.
	LearnedMetadataID string `json:"learned_metadata_id,omitempty"`

	// DegradationMessage contains a user-friendly message when operating in degraded mode.
	// E.g., "AI 服務暫時無法使用，使用基本解析" (NFR-R11)
	DegradationMessage string `json:"degradation_message,omitempty"`
}

// Parser defines the interface for filename parsers.
// This allows for different parsing implementations (regex-based, AI-powered).
type Parser interface {
	// Parse attempts to extract metadata from a filename.
	Parse(filename string) *ParseResult

	// CanParse returns true if this parser can handle the given filename.
	CanParse(filename string) bool
}
