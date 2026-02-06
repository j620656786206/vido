package services

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vido/api/internal/parser"
)

func TestIsAIServiceError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{"quota exceeded", errors.New("quota exceeded"), true},
		{"rate limit", errors.New("rate limit reached"), true},
		{"service unavailable", errors.New("service unavailable"), true},
		{"timeout", errors.New("context deadline exceeded timeout"), true},
		{"connection refused", errors.New("connection refused"), true},
		{"503 error", errors.New("503 Service Unavailable"), true},
		{"429 error", errors.New("429 Too Many Requests"), true},
		{"context deadline", context.DeadlineExceeded, true},
		{"context canceled", context.Canceled, false},
		{"normal error", errors.New("invalid JSON response"), false},
		{"parse error", errors.New("failed to parse AI response"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsAIServiceError(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAIFallbackMessage(t *testing.T) {
	msg := AIFallbackMessage()
	assert.Contains(t, msg, "AI")
	assert.Contains(t, msg, "服務")
}

func TestParseWithFallback_NoFallbackNeeded(t *testing.T) {
	// Create parser service
	parserService := NewParserService()

	// Standard filename that regex can handle
	filename := "The.Matrix.1999.1080p.BluRay.x264-GROUP.mkv"

	result := parserService.ParseFilename(filename)

	require.NotNil(t, result)
	assert.Equal(t, parser.ParseStatusSuccess, result.Status)
	// Normal regex parsing doesn't set MetadataSource - only fallback does
	// This verifies no degradation message is set for normal parsing
	assert.Empty(t, result.DegradationMessage)
	assert.NotEqual(t, parser.MetadataSourceRegexFallback, result.MetadataSource)
}

func TestApplyRegexFallback(t *testing.T) {
	// Create a result
	result := &parser.ParseResult{
		OriginalFilename: "[字幕組] 測試影片.mkv",
		Status:           parser.ParseStatusSuccess,
		MediaType:        parser.MediaTypeMovie,
		Title:            "測試影片",
		MetadataSource:   parser.MetadataSourceRegex,
		Confidence:       60,
	}

	// Apply fallback marking
	ApplyRegexFallback(result)

	assert.Equal(t, parser.MetadataSourceRegexFallback, result.MetadataSource)
	assert.Equal(t, AIFallbackMessage(), result.DegradationMessage)
}
