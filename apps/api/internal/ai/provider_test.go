package ai

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestProviderName_String(t *testing.T) {
	tests := []struct {
		name     string
		provider ProviderName
		want     string
	}{
		{"gemini", ProviderGemini, "gemini"},
		{"claude", ProviderClaude, "claude"},
		{"custom", ProviderName("custom"), "custom"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.provider.String())
		})
	}
}

func TestProviderName_IsValid(t *testing.T) {
	tests := []struct {
		name     string
		provider ProviderName
		want     bool
	}{
		{"gemini is valid", ProviderGemini, true},
		{"claude is valid", ProviderClaude, true},
		{"unknown is invalid", ProviderName("unknown"), false},
		{"empty is invalid", ProviderName(""), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.provider.IsValid())
		})
	}
}

func TestDefaultPrompt(t *testing.T) {
	// Verify default prompt contains essential parsing instructions
	assert.Contains(t, DefaultPrompt, "title")
	assert.Contains(t, DefaultPrompt, "media_type")
	assert.Contains(t, DefaultPrompt, "movie")
	assert.Contains(t, DefaultPrompt, "tv")
	assert.Contains(t, DefaultPrompt, "confidence")
	assert.Contains(t, DefaultPrompt, "JSON")
	assert.Contains(t, DefaultPrompt, "%s") // Placeholder for filename
}
