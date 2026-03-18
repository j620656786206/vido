package models

import "time"

// LogLevel represents the severity level of a log entry
type LogLevel string

const (
	LogLevelError LogLevel = "ERROR"
	LogLevelWarn  LogLevel = "WARN"
	LogLevelInfo  LogLevel = "INFO"
	LogLevelDebug LogLevel = "DEBUG"
)

// ValidLogLevels returns all valid log levels
func ValidLogLevels() []LogLevel {
	return []LogLevel{LogLevelError, LogLevelWarn, LogLevelInfo, LogLevelDebug}
}

// IsValidLogLevel checks if the given level is valid
func IsValidLogLevel(level LogLevel) bool {
	for _, valid := range ValidLogLevels() {
		if level == valid {
			return true
		}
	}
	return false
}

// SystemLog represents a log entry stored in the database
type SystemLog struct {
	ID          int64    `json:"id"`
	Level       LogLevel `json:"level"`
	Message     string   `json:"message"`
	Source      string   `json:"source,omitempty"`
	ContextJSON string   `json:"-"`
	Context     any      `json:"context,omitempty"`
	Hint        string   `json:"hint,omitempty"`
	CreatedAt   time.Time `json:"createdAt"`
}

// LogFilter contains filter parameters for querying logs
type LogFilter struct {
	Level   LogLevel
	Keyword string
	Page    int
	PerPage int
}
