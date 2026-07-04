package plugins

import "fmt"

// Error code constants for DVR plugin operations (Rule 7 {SOURCE}_{ERROR_TYPE};
// local-package convention like qbittorrent/types.go). DVR_* is a new prefix
// registered in project-context.md; PLUGIN_* codes were reserved there since
// §7 and get their first live use in the plugin manager.
const (
	ErrCodeNotConfigured    = "DVR_NOT_CONFIGURED"
	ErrCodeConnectionFailed = "DVR_CONNECTION_FAILED"
	ErrCodeAuthFailed       = "DVR_AUTH_FAILED"
	ErrCodeTimeout          = "DVR_TIMEOUT"
	ErrCodeAddFailed        = "DVR_ADD_FAILED"
	ErrCodeTestFailed       = "DVR_TEST_FAILED"
	ErrCodeNotSupported     = "DVR_NOT_SUPPORTED"
	// ErrCodeTVDBNotFound — the requested series has no TVDB entry, which
	// Sonarr fundamentally cannot search (Story 13-4b AC #1). The ONE
	// fulfilment error that is terminal: the request row goes 'failed',
	// never stranded 'pending' (retrying cannot fix TVDB absence).
	ErrCodeTVDBNotFound = "DVR_TVDB_NOT_FOUND"

	ErrCodePluginInitFailed        = "PLUGIN_INIT_FAILED"
	ErrCodePluginHealthCheckFailed = "PLUGIN_HEALTH_CHECK_FAILED"
)

// PluginError represents a DVR plugin operation error with a Rule 7 error
// code. Handlers lift Code into APIError.Code via errors.As (the
// qbittorrent_handler pattern).
type PluginError struct {
	Code    string
	Message string
	Cause   error
}

func (e *PluginError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s: %s", e.Code, e.Message, e.Cause.Error())
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (e *PluginError) Unwrap() error {
	return e.Cause
}
