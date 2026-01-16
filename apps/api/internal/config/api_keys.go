package config

// HasTMDbKey returns true if a TMDb API key is configured
func (c *Config) HasTMDbKey() bool {
	return c.TMDbAPIKey != ""
}

// HasGeminiKey returns true if a Gemini API key is configured
func (c *Config) HasGeminiKey() bool {
	return c.GeminiAPIKey != ""
}

// HasEncryptionKey returns true if an encryption key is configured
func (c *Config) HasEncryptionKey() bool {
	return c.EncryptionKey != ""
}

// HasAIProvider returns true if any AI provider key is configured
func (c *Config) HasAIProvider() bool {
	return c.HasGeminiKey()
}

// GetTMDbAPIKey returns the TMDb API key or empty string if not set
func (c *Config) GetTMDbAPIKey() string {
	return c.TMDbAPIKey
}

// GetGeminiAPIKey returns the Gemini API key or empty string if not set
func (c *Config) GetGeminiAPIKey() string {
	return c.GeminiAPIKey
}

// GetEncryptionKey returns the encryption key or empty string if not set
func (c *Config) GetEncryptionKey() string {
	return c.EncryptionKey
}
