package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHasOpenAIKey(t *testing.T) {
	cfg := &Config{OpenAIAPIKey: "sk-test-key"}
	assert.True(t, cfg.HasOpenAIKey())

	cfg = &Config{OpenAIAPIKey: ""}
	assert.False(t, cfg.HasOpenAIKey())
}

func TestGetOpenAIAPIKey(t *testing.T) {
	cfg := &Config{OpenAIAPIKey: "sk-test-key-123"}
	assert.Equal(t, "sk-test-key-123", cfg.GetOpenAIAPIKey())

	cfg = &Config{OpenAIAPIKey: ""}
	assert.Equal(t, "", cfg.GetOpenAIAPIKey())
}
