package config

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
)

// ConfigSource indicates where a configuration value came from
type ConfigSource int

const (
	// SourceDefault indicates the value is the default
	SourceDefault ConfigSource = iota
	// SourceEnvVar indicates the value came from an environment variable
	SourceEnvVar
	// SourceConfigFile indicates the value came from a config file
	SourceConfigFile
)

// String returns a human-readable representation of the config source
func (s ConfigSource) String() string {
	switch s {
	case SourceEnvVar:
		return "env"
	case SourceConfigFile:
		return "file"
	default:
		return "default"
	}
}

// Config holds all application configuration
type Config struct {
	// Server configuration
	Port     string
	Env      string
	LogLevel string

	// CORS configuration
	CORSOrigins []string

	// Paths
	DataDir   string
	MediaDirs []string

	// API Keys (optional)
	TMDbAPIKey    string
	GeminiAPIKey  string
	ClaudeAPIKey  string
	EncryptionKey string

	// AI Provider configuration (Story 3.1)
	AIProvider string // "gemini" or "claude"

	// TMDb configuration
	TMDbDefaultLanguage   string
	TMDbFallbackLanguages []string
	TMDbCacheTTLHours     int

	// Metadata fallback chain configuration (Story 3.3)
	EnableDouban                   bool
	EnableWikipedia                bool
	EnableCircuitBreaker           bool
	FallbackDelayMs                int
	CircuitBreakerFailureThreshold int
	CircuitBreakerTimeoutSeconds   int

	// Database configuration
	Database *DatabaseConfig

	// Source tracking - maps config key to its source
	Sources map[string]ConfigSource
}

// Load reads configuration from environment variables with defaults
func Load() (*Config, error) {
	cfg := &Config{
		Sources: make(map[string]ConfigSource),
	}

	// Port - VIDO_PORT takes precedence over PORT for backward compatibility
	cfg.Port = cfg.loadWithFallback("VIDO_PORT", "PORT", "8080")

	// Environment
	cfg.Env = cfg.loadString("ENV", "development")

	// Log level
	cfg.LogLevel = cfg.loadString("VIDO_LOG_LEVEL", "info")

	// CORS origins
	cfg.CORSOrigins = cfg.loadStringSlice("VIDO_CORS_ORIGINS", "*")

	// Data directory
	cfg.DataDir = cfg.loadString("VIDO_DATA_DIR", "/vido-data")

	// Media directories (comma-separated)
	cfg.MediaDirs = cfg.loadStringSlice("VIDO_MEDIA_DIRS", "/media")

	// API Keys (optional - empty string is valid default)
	cfg.TMDbAPIKey = cfg.loadString("TMDB_API_KEY", "")
	cfg.GeminiAPIKey = cfg.loadString("GEMINI_API_KEY", "")
	cfg.ClaudeAPIKey = cfg.loadString("CLAUDE_API_KEY", "")
	cfg.EncryptionKey = cfg.loadString("ENCRYPTION_KEY", "")

	// AI Provider configuration (Story 3.1) - defaults to "gemini" if not set
	cfg.AIProvider = cfg.loadString("AI_PROVIDER", "gemini")

	// TMDb configuration (Story 2.1)
	cfg.TMDbDefaultLanguage = cfg.loadString("TMDB_DEFAULT_LANGUAGE", "zh-TW")
	cfg.TMDbFallbackLanguages = cfg.loadStringSlice("TMDB_FALLBACK_LANGUAGES", "zh-TW,zh-CN,en")
	cfg.TMDbCacheTTLHours = cfg.loadInt("TMDB_CACHE_TTL_HOURS", 24)

	// Metadata fallback chain configuration (Story 3.3)
	// Providers enabled by default for future implementation
	cfg.EnableDouban = cfg.loadBool("ENABLE_DOUBAN", false)
	cfg.EnableWikipedia = cfg.loadBool("ENABLE_WIKIPEDIA", false)
	cfg.EnableCircuitBreaker = cfg.loadBool("ENABLE_CIRCUIT_BREAKER", true)
	cfg.FallbackDelayMs = cfg.loadInt("FALLBACK_DELAY_MS", 100)
	cfg.CircuitBreakerFailureThreshold = cfg.loadInt("CIRCUIT_BREAKER_FAILURE_THRESHOLD", 5)
	cfg.CircuitBreakerTimeoutSeconds = cfg.loadInt("CIRCUIT_BREAKER_TIMEOUT_SECONDS", 30)

	// Load database configuration
	dbCfg, err := LoadDatabaseConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load database config: %w", err)
	}
	cfg.Database = dbCfg

	return cfg, nil
}

// loadString loads a string value from env or uses default, tracking source
func (c *Config) loadString(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		c.Sources[key] = SourceEnvVar
		return value
	}
	c.Sources[key] = SourceDefault
	return defaultValue
}

// loadWithFallback loads from primary env var, falls back to secondary, then default
func (c *Config) loadWithFallback(primary, fallback, defaultValue string) string {
	// Check primary first
	if value := os.Getenv(primary); value != "" {
		c.Sources[primary] = SourceEnvVar
		return value
	}
	// Check fallback
	if fallback != "" {
		if value := os.Getenv(fallback); value != "" {
			c.Sources[primary] = SourceEnvVar // Track under primary key
			return value
		}
	}
	// Use default
	c.Sources[primary] = SourceDefault
	return defaultValue
}

// loadStringSlice loads a comma-separated string slice from env or uses default
func (c *Config) loadStringSlice(key, defaultValue string) []string {
	value := os.Getenv(key)
	if value != "" {
		c.Sources[key] = SourceEnvVar
	} else {
		c.Sources[key] = SourceDefault
		value = defaultValue
	}
	return parseStringSlice(value)
}

// loadBool loads a boolean value from env or uses default, tracking source
// Accepts "true", "1", "yes" as true; "false", "0", "no" as false (case-insensitive)
func (c *Config) loadBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		c.Sources[key] = SourceEnvVar
		lower := strings.ToLower(value)
		return lower == "true" || lower == "1" || lower == "yes"
	}
	c.Sources[key] = SourceDefault
	return defaultValue
}

// loadInt loads an integer value from env or uses default, tracking source
// If the env var is set but cannot be parsed, it logs a warning and uses the default
func (c *Config) loadInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		intVal, err := strconv.Atoi(value)
		if err != nil {
			slog.Warn("Invalid integer value for environment variable, using default",
				"key", key,
				"value", value,
				"default", defaultValue,
				"error", err.Error(),
			)
			c.Sources[key] = SourceDefault
			return defaultValue
		}
		c.Sources[key] = SourceEnvVar
		return intVal
	}
	c.Sources[key] = SourceDefault
	return defaultValue
}

// parseStringSlice parses a comma-separated string into a slice
func parseStringSlice(value string) []string {
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

// getEnvStringSliceOrDefault returns a string slice from env var or default
// This is a standalone helper function for use outside Config struct
func getEnvStringSliceOrDefault(key string, defaultValue string) []string {
	value := os.Getenv(key)
	if value == "" {
		value = defaultValue
	}
	return parseStringSlice(value)
}

// LogConfigSources logs which source each configuration value came from
func (c *Config) LogConfigSources() {
	slog.Info("Configuration loaded",
		"VIDO_PORT", c.Port,
		"VIDO_PORT_source", c.Sources["VIDO_PORT"].String(),
		"ENV", c.Env,
		"ENV_source", c.Sources["ENV"].String(),
		"VIDO_LOG_LEVEL", c.LogLevel,
		"VIDO_LOG_LEVEL_source", c.Sources["VIDO_LOG_LEVEL"].String(),
		"VIDO_DATA_DIR", c.DataDir,
		"VIDO_DATA_DIR_source", c.Sources["VIDO_DATA_DIR"].String(),
		"VIDO_MEDIA_DIRS", strings.Join(c.MediaDirs, ","),
		"VIDO_MEDIA_DIRS_source", c.Sources["VIDO_MEDIA_DIRS"].String(),
		"VIDO_CORS_ORIGINS", strings.Join(c.CORSOrigins, ","),
		"VIDO_CORS_ORIGINS_source", c.Sources["VIDO_CORS_ORIGINS"].String(),
		"TMDB_API_KEY", maskSecret(c.TMDbAPIKey),
		"TMDB_API_KEY_source", c.Sources["TMDB_API_KEY"].String(),
		"GEMINI_API_KEY", maskSecret(c.GeminiAPIKey),
		"GEMINI_API_KEY_source", c.Sources["GEMINI_API_KEY"].String(),
		"CLAUDE_API_KEY", maskSecret(c.ClaudeAPIKey),
		"CLAUDE_API_KEY_source", c.Sources["CLAUDE_API_KEY"].String(),
		"AI_PROVIDER", c.AIProvider,
		"AI_PROVIDER_source", c.Sources["AI_PROVIDER"].String(),
		"ENCRYPTION_KEY", maskSecret(c.EncryptionKey),
		"ENCRYPTION_KEY_source", c.Sources["ENCRYPTION_KEY"].String(),
		"TMDB_DEFAULT_LANGUAGE", c.TMDbDefaultLanguage,
		"TMDB_DEFAULT_LANGUAGE_source", c.Sources["TMDB_DEFAULT_LANGUAGE"].String(),
		"TMDB_FALLBACK_LANGUAGES", strings.Join(c.TMDbFallbackLanguages, ","),
		"TMDB_FALLBACK_LANGUAGES_source", c.Sources["TMDB_FALLBACK_LANGUAGES"].String(),
		"TMDB_CACHE_TTL_HOURS", c.TMDbCacheTTLHours,
		"TMDB_CACHE_TTL_HOURS_source", c.Sources["TMDB_CACHE_TTL_HOURS"].String(),
		// Metadata fallback chain configuration (Story 3.3)
		"ENABLE_DOUBAN", c.EnableDouban,
		"ENABLE_DOUBAN_source", c.Sources["ENABLE_DOUBAN"].String(),
		"ENABLE_WIKIPEDIA", c.EnableWikipedia,
		"ENABLE_WIKIPEDIA_source", c.Sources["ENABLE_WIKIPEDIA"].String(),
		"ENABLE_CIRCUIT_BREAKER", c.EnableCircuitBreaker,
		"ENABLE_CIRCUIT_BREAKER_source", c.Sources["ENABLE_CIRCUIT_BREAKER"].String(),
		"FALLBACK_DELAY_MS", c.FallbackDelayMs,
		"FALLBACK_DELAY_MS_source", c.Sources["FALLBACK_DELAY_MS"].String(),
		"CIRCUIT_BREAKER_FAILURE_THRESHOLD", c.CircuitBreakerFailureThreshold,
		"CIRCUIT_BREAKER_FAILURE_THRESHOLD_source", c.Sources["CIRCUIT_BREAKER_FAILURE_THRESHOLD"].String(),
		"CIRCUIT_BREAKER_TIMEOUT_SECONDS", c.CircuitBreakerTimeoutSeconds,
		"CIRCUIT_BREAKER_TIMEOUT_SECONDS_source", c.Sources["CIRCUIT_BREAKER_TIMEOUT_SECONDS"].String(),
	)
}

// maskSecret masks sensitive values for safe logging
func maskSecret(s string) string {
	if s == "" {
		return "(not set)"
	}
	if len(s) <= 8 {
		return "****"
	}
	return s[:4] + "****" + s[len(s)-4:]
}

// IsDevelopment returns true if running in development mode
func (c *Config) IsDevelopment() bool {
	return c.Env == "development" || c.Env == "dev"
}

// IsProduction returns true if running in production mode
func (c *Config) IsProduction() bool {
	return c.Env == "production" || c.Env == "prod"
}

// GetPort returns the port as an integer
func (c *Config) GetPort() (int, error) {
	return strconv.Atoi(c.Port)
}

// GetAddress returns the full server address (e.g., ":3000")
func (c *Config) GetAddress() string {
	return ":" + c.Port
}
