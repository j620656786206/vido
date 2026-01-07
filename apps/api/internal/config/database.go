package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

// DatabaseConfig holds database-specific configuration
type DatabaseConfig struct {
	// Path to the SQLite database file
	Path string

	// WAL mode settings
	WALEnabled     bool
	WALSyncMode    string // OFF, NORMAL, FULL
	WALCheckpoint  int    // Number of frames before auto-checkpoint

	// Connection pool settings
	MaxOpenConns    int           // Maximum number of open connections
	MaxIdleConns    int           // Maximum number of idle connections
	ConnMaxLifetime time.Duration // Maximum lifetime of a connection
	ConnMaxIdleTime time.Duration // Maximum idle time of a connection

	// Additional settings
	BusyTimeout time.Duration // How long to wait when database is locked
	CacheSize   int           // Cache size in pages (negative = KB)
}

// LoadDatabaseConfig reads database configuration from environment variables
func LoadDatabaseConfig() (*DatabaseConfig, error) {
	cfg := &DatabaseConfig{
		// Default values
		Path:            getEnvOrDefault("DB_PATH", "./data/vido.db"),
		WALEnabled:      getEnvBoolOrDefault("DB_WAL_ENABLED", true),
		WALSyncMode:     getEnvOrDefault("DB_WAL_SYNC_MODE", "NORMAL"),
		WALCheckpoint:   getEnvIntOrDefault("DB_WAL_CHECKPOINT", 1000),
		MaxOpenConns:    getEnvIntOrDefault("DB_MAX_OPEN_CONNS", 25),
		MaxIdleConns:    getEnvIntOrDefault("DB_MAX_IDLE_CONNS", 5),
		ConnMaxLifetime: getEnvDurationOrDefault("DB_CONN_MAX_LIFETIME", 5*time.Minute),
		ConnMaxIdleTime: getEnvDurationOrDefault("DB_CONN_MAX_IDLE_TIME", 1*time.Minute),
		BusyTimeout:     getEnvDurationOrDefault("DB_BUSY_TIMEOUT", 5*time.Second),
		CacheSize:       getEnvIntOrDefault("DB_CACHE_SIZE", -64000), // 64MB
	}

	// Validate WAL sync mode
	validSyncModes := map[string]bool{
		"OFF":    true,
		"NORMAL": true,
		"FULL":   true,
	}
	if !validSyncModes[cfg.WALSyncMode] {
		return nil, fmt.Errorf("invalid WAL sync mode: %s (valid: OFF, NORMAL, FULL)", cfg.WALSyncMode)
	}

	// Validate connection pool settings
	if cfg.MaxOpenConns < 1 {
		return nil, fmt.Errorf("max open connections must be at least 1, got: %d", cfg.MaxOpenConns)
	}
	if cfg.MaxIdleConns < 0 {
		return nil, fmt.Errorf("max idle connections must be non-negative, got: %d", cfg.MaxIdleConns)
	}
	if cfg.MaxIdleConns > cfg.MaxOpenConns {
		return nil, fmt.Errorf("max idle connections (%d) cannot exceed max open connections (%d)", cfg.MaxIdleConns, cfg.MaxOpenConns)
	}

	return cfg, nil
}

// GetDatabaseDir returns the directory containing the database file
func (c *DatabaseConfig) GetDatabaseDir() string {
	return filepath.Dir(c.Path)
}

// GetConnectionString returns the SQLite connection string with parameters
func (c *DatabaseConfig) GetConnectionString() string {
	// SQLite connection string format: file:path?param=value&param=value
	params := ""

	if c.WALEnabled {
		// WAL mode and related settings are applied via PRAGMA after connection
		// The connection string itself is just the file path
		params = "?cache=shared&mode=rwc"
	} else {
		params = "?cache=shared&mode=rwc"
	}

	return fmt.Sprintf("file:%s%s", c.Path, params)
}

// Helper functions for environment variable parsing

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvIntOrDefault(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvBoolOrDefault(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

func getEnvDurationOrDefault(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		// Try parsing as duration string (e.g., "5m", "30s")
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
		// Try parsing as seconds
		if seconds, err := strconv.Atoi(value); err == nil {
			return time.Duration(seconds) * time.Second
		}
	}
	return defaultValue
}
