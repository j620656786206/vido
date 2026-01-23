package ai

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		request *ParseRequest
		wantErr bool
	}{
		{
			name: "valid request with filename",
			request: &ParseRequest{
				Filename: "[Fansub] Title - 01 [1080p].mkv",
			},
			wantErr: false,
		},
		{
			name: "valid request with filename and prompt",
			request: &ParseRequest{
				Filename: "some.file.mkv",
				Prompt:   "Custom prompt",
			},
			wantErr: false,
		},
		{
			name: "invalid request empty filename",
			request: &ParseRequest{
				Filename: "",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.request.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestParseResponse_IsMovie(t *testing.T) {
	tests := []struct {
		name      string
		mediaType string
		want      bool
	}{
		{"movie type", "movie", true},
		{"tv type", "tv", false},
		{"empty type", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &ParseResponse{MediaType: tt.mediaType}
			assert.Equal(t, tt.want, r.IsMovie())
		})
	}
}

func TestParseResponse_IsTVShow(t *testing.T) {
	tests := []struct {
		name      string
		mediaType string
		want      bool
	}{
		{"tv type", "tv", true},
		{"movie type", "movie", false},
		{"empty type", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &ParseResponse{MediaType: tt.mediaType}
			assert.Equal(t, tt.want, r.IsTVShow())
		})
	}
}

func TestErrorConstants(t *testing.T) {
	// Verify error codes follow project-context.md Rule 7 format: {SOURCE}_{ERROR_TYPE}
	assert.Contains(t, ErrAITimeout.Error(), "AI_TIMEOUT")
	assert.Contains(t, ErrAIQuotaExceeded.Error(), "AI_QUOTA_EXCEEDED")
	assert.Contains(t, ErrAIInvalidResponse.Error(), "AI_INVALID_RESPONSE")
	assert.Contains(t, ErrAIProviderError.Error(), "AI_PROVIDER_ERROR")
	assert.Contains(t, ErrAINotConfigured.Error(), "AI_NOT_CONFIGURED")
}

func TestCleanJSONResponse(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "plain JSON - no change",
			input:    `{"title": "Test", "media_type": "movie"}`,
			expected: `{"title": "Test", "media_type": "movie"}`,
		},
		{
			name:     "JSON with json code block",
			input:    "```json\n{\"title\": \"Test\"}\n```",
			expected: `{"title": "Test"}`,
		},
		{
			name:     "JSON with plain code block",
			input:    "```\n{\"title\": \"Test\"}\n```",
			expected: `{"title": "Test"}`,
		},
		{
			name:     "JSON with code block and whitespace",
			input:    "  ```json\n  {\"title\": \"Test\"}\n  ```  ",
			expected: `{"title": "Test"}`,
		},
		{
			name: "multiline JSON in code block",
			input: "```json\n{\n  \"title\": \"Test\",\n  \"year\": 2023\n}\n```",
			expected: `{
  "title": "Test",
  "year": 2023
}`,
		},
		{
			name:     "already clean - with leading/trailing whitespace",
			input:    "  {\"title\": \"Test\"}  ",
			expected: `{"title": "Test"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CleanJSONResponse(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
