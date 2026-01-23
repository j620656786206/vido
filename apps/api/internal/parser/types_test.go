package parser

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseStatus_Constants(t *testing.T) {
	// Verify all expected status constants exist
	assert.Equal(t, ParseStatus("success"), ParseStatusSuccess)
	assert.Equal(t, ParseStatus("parsing"), ParseStatusParsing)
	assert.Equal(t, ParseStatus("needs_ai"), ParseStatusNeedsAI)
	assert.Equal(t, ParseStatus("failed"), ParseStatusFailed)
}

func TestMediaType_Constants(t *testing.T) {
	// Verify all expected media type constants exist
	assert.Equal(t, MediaType("movie"), MediaTypeMovie)
	assert.Equal(t, MediaType("tv"), MediaTypeTVShow)
	assert.Equal(t, MediaType("unknown"), MediaTypeUnknown)
}

func TestParseResult_JSONSerialization(t *testing.T) {
	result := &ParseResult{
		OriginalFilename: "The.Matrix.1999.1080p.BluRay.mkv",
		Status:           ParseStatusSuccess,
		MediaType:        MediaTypeMovie,
		Title:            "The Matrix",
		CleanedTitle:     "The Matrix",
		Year:             1999,
		Quality:          "1080p",
		Source:           "BluRay",
		VideoCodec:       "x264",
		ReleaseGroup:     "GROUP",
		Confidence:       85,
	}

	// Test serialization
	jsonBytes, err := json.Marshal(result)
	require.NoError(t, err)

	// Test deserialization
	var decoded ParseResult
	err = json.Unmarshal(jsonBytes, &decoded)
	require.NoError(t, err)

	assert.Equal(t, result.OriginalFilename, decoded.OriginalFilename)
	assert.Equal(t, result.Status, decoded.Status)
	assert.Equal(t, result.MediaType, decoded.MediaType)
	assert.Equal(t, result.Title, decoded.Title)
	assert.Equal(t, result.Year, decoded.Year)
	assert.Equal(t, result.Quality, decoded.Quality)
	assert.Equal(t, result.Source, decoded.Source)
	assert.Equal(t, result.Confidence, decoded.Confidence)
}

func TestParseResult_JSONOmitEmpty(t *testing.T) {
	// Test that empty/zero values are omitted
	result := &ParseResult{
		OriginalFilename: "test.mkv",
		Status:           ParseStatusNeedsAI,
		MediaType:        MediaTypeUnknown,
		Confidence:       0,
	}

	jsonBytes, err := json.Marshal(result)
	require.NoError(t, err)

	jsonStr := string(jsonBytes)

	// These fields should be omitted when empty/zero
	assert.NotContains(t, jsonStr, "year")
	assert.NotContains(t, jsonStr, "season")
	assert.NotContains(t, jsonStr, "episode")
	assert.NotContains(t, jsonStr, "quality")
	assert.NotContains(t, jsonStr, "source")
	assert.NotContains(t, jsonStr, "video_codec")
	assert.NotContains(t, jsonStr, "audio_codec")
	assert.NotContains(t, jsonStr, "release_group")
	assert.NotContains(t, jsonStr, "error_message")
}

func TestParseResult_TVShowFields(t *testing.T) {
	result := &ParseResult{
		OriginalFilename: "Breaking.Bad.S01E05.720p.mkv",
		Status:           ParseStatusSuccess,
		MediaType:        MediaTypeTVShow,
		Title:            "Breaking Bad",
		CleanedTitle:     "Breaking Bad",
		Season:           1,
		Episode:          5,
		EpisodeEnd:       0,
		Quality:          "720p",
		Confidence:       85,
	}

	assert.Equal(t, 1, result.Season)
	assert.Equal(t, 5, result.Episode)
	assert.Equal(t, 0, result.EpisodeEnd)
	assert.Equal(t, MediaTypeTVShow, result.MediaType)
}

func TestParseResult_EpisodeRange(t *testing.T) {
	result := &ParseResult{
		OriginalFilename: "Friends.S01E01-E02.DVDRip.mkv",
		Status:           ParseStatusSuccess,
		MediaType:        MediaTypeTVShow,
		Title:            "Friends",
		Season:           1,
		Episode:          1,
		EpisodeEnd:       2,
		Confidence:       80,
	}

	assert.Equal(t, 1, result.Episode)
	assert.Equal(t, 2, result.EpisodeEnd)
}

func TestParseResult_NeedsAIStatus(t *testing.T) {
	result := &ParseResult{
		OriginalFilename: "[Leopard-Raws] Kimetsu no Yaiba - 26 (BD 1920x1080).mkv",
		Status:           ParseStatusNeedsAI,
		MediaType:        MediaTypeUnknown,
		Confidence:       0,
		ErrorMessage:     "Could not parse with standard patterns",
	}

	assert.Equal(t, ParseStatusNeedsAI, result.Status)
	assert.Equal(t, MediaTypeUnknown, result.MediaType)
	assert.Equal(t, 0, result.Confidence)
	assert.NotEmpty(t, result.ErrorMessage)
}

func TestParserInterface_Exists(t *testing.T) {
	// Test that Parser interface can be used as a type
	var _ Parser = (*mockParser)(nil)
}

// mockParser implements Parser interface for testing
type mockParser struct{}

func (m *mockParser) Parse(filename string) *ParseResult {
	return &ParseResult{
		OriginalFilename: filename,
		Status:           ParseStatusSuccess,
	}
}

func (m *mockParser) CanParse(filename string) bool {
	return true
}

func TestMetadataSource_Constants(t *testing.T) {
	// Verify all expected metadata source constants exist
	assert.Equal(t, MetadataSource("regex"), MetadataSourceRegex)
	assert.Equal(t, MetadataSource("ai"), MetadataSourceAI)
	assert.Equal(t, MetadataSource("ai_fansub"), MetadataSourceAIFansub)
	assert.Equal(t, MetadataSource("manual"), MetadataSourceManual)
}

func TestParseResult_NewFansubFields(t *testing.T) {
	// Test new fields added for fansub parsing (Story 3.2)
	result := &ParseResult{
		OriginalFilename: "【幻櫻字幕組】我的英雄學院 第01話 1080P.mp4",
		Status:           ParseStatusSuccess,
		MediaType:        MediaTypeTVShow,
		Title:            "我的英雄學院",
		Season:           1,
		Episode:          1,
		Quality:          "1080p",
		ReleaseGroup:     "幻櫻字幕組",
		Language:         "Traditional Chinese",
		MetadataSource:   MetadataSourceAIFansub,
		ParseDurationMs:  250,
		AIProvider:       "gemini",
		Confidence:       92,
	}

	// Verify all new fields
	assert.Equal(t, "Traditional Chinese", result.Language)
	assert.Equal(t, MetadataSourceAIFansub, result.MetadataSource)
	assert.Equal(t, int64(250), result.ParseDurationMs)
	assert.Equal(t, "gemini", result.AIProvider)
}

func TestParseResult_ParsingStatus(t *testing.T) {
	// Test the new parsing status for in-progress operations
	result := &ParseResult{
		OriginalFilename: "test.mkv",
		Status:           ParseStatusParsing,
		MediaType:        MediaTypeUnknown,
		Confidence:       0,
	}

	assert.Equal(t, ParseStatusParsing, result.Status)
}

func TestParseResult_JSONSerialization_NewFields(t *testing.T) {
	result := &ParseResult{
		OriginalFilename: "[SubsPlease] Anime - 01.mkv",
		Status:           ParseStatusSuccess,
		MediaType:        MediaTypeTVShow,
		Title:            "Anime",
		Season:           1,
		Episode:          1,
		Quality:          "1080p",
		ReleaseGroup:     "SubsPlease",
		Language:         "Japanese",
		MetadataSource:   MetadataSourceAIFansub,
		ParseDurationMs:  150,
		AIProvider:       "claude",
		Confidence:       88,
	}

	// Test serialization includes new fields
	jsonBytes, err := json.Marshal(result)
	require.NoError(t, err)

	jsonStr := string(jsonBytes)

	// New fields should be present
	assert.Contains(t, jsonStr, `"language":"Japanese"`)
	assert.Contains(t, jsonStr, `"metadata_source":"ai_fansub"`)
	assert.Contains(t, jsonStr, `"parse_duration_ms":150`)
	assert.Contains(t, jsonStr, `"ai_provider":"claude"`)

	// Test deserialization
	var decoded ParseResult
	err = json.Unmarshal(jsonBytes, &decoded)
	require.NoError(t, err)

	assert.Equal(t, result.Language, decoded.Language)
	assert.Equal(t, result.MetadataSource, decoded.MetadataSource)
	assert.Equal(t, result.ParseDurationMs, decoded.ParseDurationMs)
	assert.Equal(t, result.AIProvider, decoded.AIProvider)
}
