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

// HasOpenAIKey returns true if an OpenAI API key is configured
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

// GetEncryptionKey returns the encryption key or empty string if not set
func (c *Config) GetEncryptionKey() string {
	return c.EncryptionKey
}
