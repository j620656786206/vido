package prompts

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildFansubPrompt(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		wantContains []string
	}{
		{
			name:     "includes filename",
			filename: "[Test-Group] Anime Title - 01.mkv",
			wantContains: []string{
				"[Test-Group] Anime Title - 01.mkv",
				"Filename:",
			},
		},
		{
			name:     "includes Chinese examples",
			filename: "【字幕組】標題.mp4",
			wantContains: []string{
				"【幻櫻字幕組】",
				"我的英雄學院",
				"繁體",
			},
		},
		{
			name:     "includes Japanese examples",
			filename: "[Leopard-Raws] Test.mkv",
			wantContains: []string{
				"Leopard-Raws",
				"Kimetsu no Yaiba",
				"SubsPlease",
			},
		},
		{
			name:     "includes JSON schema",
			filename: "test.mkv",
			wantContains: []string{
				`"title"`,
				`"episode"`,
				`"media_type"`,
				`"confidence"`,
				`"fansub_group"`,
			},
		},
		{
			name:     "includes extraction rules",
			filename: "test.mkv",
			wantContains: []string{
				"Extraction Rules",
				"Confidence Guidelines",
				"Pattern Components",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prompt := BuildFansubPrompt(tt.filename)

			for _, want := range tt.wantContains {
				assert.Contains(t, prompt, want, "prompt should contain: %s", want)
			}
		})
	}
}

func TestFansubParseResponse_JSONStructure(t *testing.T) {
	// Test that FansubParseResponse can be properly marshaled/unmarshaled
	response := FansubParseResponse{
		Title:          "我的英雄學院",
		TitleRomanized: "Boku no Hero Academia",
		Episode:        intPtr(1),
		Season:         intPtr(1),
		Quality:        "1080p",
		Source:         "BD",
		Codec:          "x264",
		FansubGroup:    "幻櫻字幕組",
		Language:       "Traditional Chinese",
		MediaType:      "tv",
		Confidence:     0.9,
	}

	// Marshal to JSON
	jsonBytes, err := json.Marshal(response)
	require.NoError(t, err)

	// Unmarshal back
	var parsed FansubParseResponse
	err = json.Unmarshal(jsonBytes, &parsed)
	require.NoError(t, err)

	// Verify all fields
	assert.Equal(t, response.Title, parsed.Title)
	assert.Equal(t, response.TitleRomanized, parsed.TitleRomanized)
	assert.Equal(t, *response.Episode, *parsed.Episode)
	assert.Equal(t, *response.Season, *parsed.Season)
	assert.Equal(t, response.Quality, parsed.Quality)
	assert.Equal(t, response.Source, parsed.Source)
	assert.Equal(t, response.Codec, parsed.Codec)
	assert.Equal(t, response.FansubGroup, parsed.FansubGroup)
	assert.Equal(t, response.Language, parsed.Language)
	assert.Equal(t, response.MediaType, parsed.MediaType)
	assert.Equal(t, response.Confidence, parsed.Confidence)
}

func TestFansubParseResponse_OptionalFields(t *testing.T) {
	// Test with minimal required fields only
	response := FansubParseResponse{
		Title:      "Test Title",
		MediaType:  "movie",
		Confidence: 0.5,
	}

	jsonBytes, err := json.Marshal(response)
	require.NoError(t, err)

	// Should not contain optional fields when nil/empty
	jsonStr := string(jsonBytes)
	assert.NotContains(t, jsonStr, "episode")
	assert.NotContains(t, jsonStr, "season")
	assert.NotContains(t, jsonStr, "year")
	assert.Contains(t, jsonStr, "title")
	assert.Contains(t, jsonStr, "media_type")
	assert.Contains(t, jsonStr, "confidence")
}

func TestFansubExamples_ValidTestData(t *testing.T) {
	// Verify all examples have valid test data
	for _, example := range FansubExamples {
		t.Run(example.Filename, func(t *testing.T) {
			// Filename should not be empty
			assert.NotEmpty(t, example.Filename, "filename should not be empty")

			// Expected should have required fields
			assert.NotEmpty(t, example.Expected.Title, "expected title should not be empty")
			assert.NotEmpty(t, example.Expected.MediaType, "expected media_type should not be empty")
			assert.True(t, example.Expected.Confidence > 0, "expected confidence should be > 0")

			// MediaType should be valid
			assert.True(t,
				example.Expected.MediaType == "tv" || example.Expected.MediaType == "movie",
				"media_type should be 'tv' or 'movie'",
			)
		})
	}
}

func TestFansubPrompt_ContainsBracketPatterns(t *testing.T) {
	prompt := BuildFansubPrompt("test.mkv")

	// Should document both bracket styles
	assert.Contains(t, prompt, "[]", "prompt should document square brackets")
	assert.Contains(t, prompt, "【】", "prompt should document fullwidth brackets")
}

func TestFansubPrompt_ContainsEpisodePatterns(t *testing.T) {
	prompt := BuildFansubPrompt("test.mkv")

	// Should document various episode notations
	// Check for similar patterns in prompt
	hasChineseEpisode := strings.Contains(prompt, "第") && strings.Contains(prompt, "話")
	hasEnglishEpisode := strings.Contains(prompt, "EP") || strings.Contains(prompt, "S01E01")
	hasDashNotation := strings.Contains(prompt, "- XX")

	assert.True(t, hasChineseEpisode, "prompt should document Chinese episode patterns (第XX話)")
	assert.True(t, hasEnglishEpisode, "prompt should document English episode patterns (EP, S01E01)")
	assert.True(t, hasDashNotation, "prompt should document dash notation (- XX)")
}

func TestFansubPrompt_JSONOnly(t *testing.T) {
	prompt := BuildFansubPrompt("test.mkv")

	// Should instruct for JSON-only response
	assert.Contains(t, prompt, "valid JSON")
	assert.Contains(t, prompt, "No markdown", "should instruct to not use markdown")
}

func TestFansubPrompt_ConfidenceGuidelines(t *testing.T) {
	prompt := BuildFansubPrompt("test.mkv")

	// Should contain confidence scoring guidelines
	assert.Contains(t, prompt, "Confidence Guidelines")
	assert.Contains(t, prompt, "1.0")
	assert.Contains(t, prompt, "0.0")
}

func TestIntPtr(t *testing.T) {
	val := 42
	ptr := intPtr(val)

	require.NotNil(t, ptr)
	assert.Equal(t, val, *ptr)
}
