package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Config holds all application configuration
type Config struct {
	// Server configuration
	Port string `envconfig:"PORT" default:"8080"`
	Env  string `envconfig:"ENV" default:"development"`

	// CORS configuration
	CORSOrigins []string `envconfig:"CORS_ORIGINS" default:"http://localhost:3000"`

	// Logging configuration
	LogLevel string `envconfig:"LOG_LEVEL" default:"info"`

	// API configuration
	APIVersion string `envconfig:"API_VERSION" default:"v1"`

	// TMDb configuration
	TMDbAPIKey         string `envconfig:"TMDB_API_KEY"`
	TMDbDefaultLanguage string `envconfig:"TMDB_DEFAULT_LANGUAGE" default:"zh-TW"`
}

// Load reads configuration from environment variables with defaults
func Load() (*Config, error) {
	cfg := &Config{}

	// Port
	if port := os.Getenv("PORT"); port != "" {
		cfg.Port = port
	} else {
		cfg.Port = "8080"
	}

	// Environment
	if env := os.Getenv("ENV"); env != "" {
		cfg.Env = env
	} else {
		cfg.Env = "development"
	}

	// CORS Origins
	if corsOrigins := os.Getenv("CORS_ORIGINS"); corsOrigins != "" {
		cfg.CORSOrigins = strings.Split(corsOrigins, ",")
		// Trim whitespace from each origin
		for i, origin := range cfg.CORSOrigins {
			cfg.CORSOrigins[i] = strings.TrimSpace(origin)
		}
	} else {
		cfg.CORSOrigins = []string{"http://localhost:3000"}
	}

	// Log Level
	if logLevel := os.Getenv("LOG_LEVEL"); logLevel != "" {
		cfg.LogLevel = strings.ToLower(logLevel)
		// Validate log level
		validLevels := map[string]bool{
			"debug": true,
			"info":  true,
			"warn":  true,
			"error": true,
		}
		if !validLevels[cfg.LogLevel] {
			return nil, fmt.Errorf("invalid log level: %s (valid: debug, info, warn, error)", cfg.LogLevel)
		}
	} else {
		cfg.LogLevel = "info"
	}

	// API Version
	if apiVersion := os.Getenv("API_VERSION"); apiVersion != "" {
		cfg.APIVersion = apiVersion
	} else {
		cfg.APIVersion = "v1"
	}

	// TMDb API Key
	cfg.TMDbAPIKey = os.Getenv("TMDB_API_KEY")

	// TMDb Default Language
	if tmdbLang := os.Getenv("TMDB_DEFAULT_LANGUAGE"); tmdbLang != "" {
		cfg.TMDbDefaultLanguage = tmdbLang
	} else {
		cfg.TMDbDefaultLanguage = "zh-TW"
	}

	return cfg, nil
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

// GetAddress returns the full server address (e.g., ":8080")
func (c *Config) GetAddress() string {
	return ":" + c.Port
}
