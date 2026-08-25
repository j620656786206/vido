package qbittorrent

import (
	"errors"
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
	ErrCodeConnectionFailed = "QBITTORRENT_CONNECTION_FAILED"
	ErrCodeAuthFailed       = "QBITTORRENT_AUTH_FAILED"
	ErrCodeTimeout          = "QBITTORRENT_TIMEOUT"
	ErrCodeNotConfigured    = "QBITTORRENT_NOT_CONFIGURED"
	// ErrCodeConfigDecryptFailed is returned when the stored password exists but
	// cannot be decrypted — almost always ENCRYPTION_KEY changing under a saved
	// secret. It is split out from INTERNAL_ERROR because the fix is specific and
	// the user can perform it; a generic 500 sends them to "try again later",
	// which will never work.
	ErrCodeConfigDecryptFailed = "QBITTORRENT_CONFIG_DECRYPT_FAILED"
)

// ErrConfigDecryptFailed is the sentinel behind ErrCodeConfigDecryptFailed.
// It lives in this package rather than in services/ because the handler layer
// deliberately does not import services (it declares its own interface), and
// both layers already import this one.
var ErrConfigDecryptFailed = errors.New("qbittorrent: stored password could not be decrypted")
