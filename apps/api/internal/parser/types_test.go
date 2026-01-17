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
