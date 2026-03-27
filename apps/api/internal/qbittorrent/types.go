package qbittorrent

import (
	"fmt"
	"time"
)

// Config holds qBittorrent connection configuration.
type Config struct {
	Host     string        `json:"host"`
	Username string        `json:"username"`
	Password string        `json:"-"`
	BasePath string        `json:"base_path,omitempty"`
	Timeout  time.Duration `json:"-"`
}

// VersionInfo holds qBittorrent version information returned by a successful connection test.
type VersionInfo struct {
	AppVersion string `json:"app_version"`
	APIVersion string `json:"api_version"`
}

// ConnectionError represents a qBittorrent connection error with an error code.
type ConnectionError struct {
	Code    string
	Message string
	Cause   error
}

func (e *ConnectionError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s: %s", e.Code, e.Message, e.Cause.Error())
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (e *ConnectionError) Unwrap() error {
	return e.Cause
}

// Error code constants for qBittorrent operations.
const (
	ErrCodeConnectionFailed = "QB_CONNECTION_FAILED"
	ErrCodeAuthFailed       = "QB_AUTH_FAILED"
	ErrCodeTimeout          = "QB_TIMEOUT"
	ErrCodeNotConfigured    = "QB_NOT_CONFIGURED"
)
