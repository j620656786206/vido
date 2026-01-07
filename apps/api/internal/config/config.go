package config

import (
	"fmt"
	"os"
	"strconv"
)

// Config holds all application configuration
type Config struct {
	// Server configuration
	Port string
	Env  string

	// Database configuration
	Database *DatabaseConfig
}

// Load reads configuration from environment variables with defaults
func Load() (*Config, error) {
	cfg := &Config{}

	// Port
	if port := os.Getenv("PORT"); port != "" {
		cfg.Port = port
	} else {
		cfg.Port = "3000"
	}

	// Environment
	if env := os.Getenv("ENV"); env != "" {
		cfg.Env = env
	} else {
		cfg.Env = "development"
	}

	// Load database configuration
	dbCfg, err := LoadDatabaseConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load database config: %w", err)
	}
	cfg.Database = dbCfg

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

// GetAddress returns the full server address (e.g., ":3000")
func (c *Config) GetAddress() string {
	return ":" + c.Port
}
