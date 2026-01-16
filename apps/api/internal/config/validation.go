package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// ValidationError represents a configuration validation error
type ValidationError struct {
	Field   string
	Message string
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// Validate validates all configuration values and returns an error if any are invalid
func (c *Config) Validate() error {
	var errs []string

	// Port validation (1-65535)
	if err := c.validatePort(); err != nil {
		errs = append(errs, err.Error())
	}

	// Log level validation
	if err := c.validateLogLevel(); err != nil {
		errs = append(errs, err.Error())
	}

	// DataDir validation - create if not exists
	if err := c.validateDataDir(); err != nil {
		errs = append(errs, err.Error())
	}

	// MediaDirs validation is optional - paths don't need to exist at startup
	// They will be validated when actually used

	if len(errs) > 0 {
		return fmt.Errorf("configuration validation failed:\n  - %s", strings.Join(errs, "\n  - "))
	}

	return nil
}

// validatePort validates that the port is a valid number between 1 and 65535
func (c *Config) validatePort() error {
	port, err := strconv.Atoi(c.Port)
	if err != nil {
		return ValidationError{
			Field:   "VIDO_PORT",
			Message: fmt.Sprintf("invalid port '%s' (must be a number)", c.Port),
		}
	}
	if port < 1 || port > 65535 {
		return ValidationError{
			Field:   "VIDO_PORT",
			Message: fmt.Sprintf("invalid port '%d' (must be 1-65535)", port),
		}
	}
	return nil
}

// validateLogLevel validates that the log level is one of: debug, info, warn, error
func (c *Config) validateLogLevel() error {
	validLevels := map[string]bool{
		"debug": true,
		"info":  true,
		"warn":  true,
		"error": true,
	}
	level := strings.ToLower(c.LogLevel)
	if !validLevels[level] {
		return ValidationError{
			Field:   "VIDO_LOG_LEVEL",
			Message: fmt.Sprintf("invalid level '%s' (must be debug/info/warn/error)", c.LogLevel),
		}
	}
	return nil
}

// validateDataDir validates that the data directory can be created/accessed
func (c *Config) validateDataDir() error {
	if c.DataDir == "" {
		return ValidationError{
			Field:   "VIDO_DATA_DIR",
			Message: "data directory cannot be empty",
		}
	}

	// Try to create the directory if it doesn't exist
	if err := os.MkdirAll(c.DataDir, 0755); err != nil {
		return ValidationError{
			Field:   "VIDO_DATA_DIR",
			Message: fmt.Sprintf("cannot create directory '%s': %v", c.DataDir, err),
		}
	}

	// Verify it's writable by creating a temp file
	testFile := c.DataDir + "/.vido_write_test"
	f, err := os.Create(testFile)
	if err != nil {
		return ValidationError{
			Field:   "VIDO_DATA_DIR",
			Message: fmt.Sprintf("directory '%s' is not writable: %v", c.DataDir, err),
		}
	}
	f.Close()
	os.Remove(testFile)

	return nil
}

// ValidateMediaDirs validates that media directories exist (optional validation)
func (c *Config) ValidateMediaDirs() error {
	var errs []string

	for _, dir := range c.MediaDirs {
		info, err := os.Stat(dir)
		if os.IsNotExist(err) {
			errs = append(errs, fmt.Sprintf("media directory '%s' does not exist", dir))
			continue
		}
		if err != nil {
			errs = append(errs, fmt.Sprintf("cannot access media directory '%s': %v", dir, err))
			continue
		}
		if !info.IsDir() {
			errs = append(errs, fmt.Sprintf("media path '%s' is not a directory", dir))
		}
	}

	if len(errs) > 0 {
		return ValidationError{
			Field:   "VIDO_MEDIA_DIRS",
			Message: strings.Join(errs, "; "),
		}
	}

	return nil
}
