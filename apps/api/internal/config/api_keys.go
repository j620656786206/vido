package config

// HasTMDbKey returns true if a TMDb API key is configured
func (c *Config) HasTMDbKey() bool {
	return c.TMDbAPIKey != ""
}

// HasGeminiKey returns true if a Gemini API key is configured
func (c *Config) HasGeminiKey() bool {
	return c.GeminiAPIKey != ""
}

// HasClaudeKey returns true if a Claude API key is configured
func (c *Config) HasClaudeKey() bool {
	return c.ClaudeAPIKey != ""
}

// HasOpenAIKey returns true if an OpenAI API key is configured in the
// ENVIRONMENT.
//
// ⚠️ Since sub-5-2 this is NOT the answer to "is speech recognition
// configured": a key stored at runtime through /settings/keys resolves through
// services.KeyResolver and never appears here, and a self-hosted ASR endpoint is
// configured with no key at all. Ask services.ASRProviderHolder.IsConfigured (or
// TranscriptionService.IsAvailable) instead — gating anything on this predicate
// re-introduces the boot-frozen bug sub-5-2 removed. Kept as a plain env
// accessor (the HasClaudeKey precedent after sub-2-1a).
func (c *Config) HasOpenAIKey() bool {
	return c.OpenAIAPIKey != ""
}

// GetOpenAIAPIKey returns the OpenAI API key or empty string if not set
func (c *Config) GetOpenAIAPIKey() string {
	return c.OpenAIAPIKey
}

// HasEncryptionKey returns true if an encryption key is configured
func (c *Config) HasEncryptionKey() bool {
	return c.EncryptionKey != ""
}

// HasAIProvider returns true if any text AI provider key is configured (Gemini or Claude).
// Note: OpenAI key (for Whisper audio transcription) is checked separately via HasOpenAIKey().
func (c *Config) HasAIProvider() bool {
	return c.HasGeminiKey() || c.HasClaudeKey()
}

// GetAIProvider returns the configured AI provider name ("gemini" or "claude")
func (c *Config) GetAIProvider() string {
	return c.AIProvider
}

// GetTMDbAPIKey returns the TMDb API key or empty string if not set
func (c *Config) GetTMDbAPIKey() string {
	return c.TMDbAPIKey
}

// GetGeminiAPIKey returns the Gemini API key or empty string if not set
func (c *Config) GetGeminiAPIKey() string {
	return c.GeminiAPIKey
}

// GetClaudeAPIKey returns the Claude API key or empty string if not set
func (c *Config) GetClaudeAPIKey() string {
	return c.ClaudeAPIKey
}

// GetClaudeModel returns the CLAUDE_MODEL override or empty string if not set
// (empty means the ai package's DefaultClaudeModel is used).
func (c *Config) GetClaudeModel() string {
	return c.ClaudeModel
}

// GetEncryptionKey returns the encryption key or empty string if not set
func (c *Config) GetEncryptionKey() string {
	return c.EncryptionKey
}
