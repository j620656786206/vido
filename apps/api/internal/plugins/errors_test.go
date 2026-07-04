package plugins

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPluginError_Error(t *testing.T) {
	t.Run("without cause", func(t *testing.T) {
		err := &PluginError{Code: ErrCodeNotConfigured, Message: "radarr not configured"}
		assert.Equal(t, "DVR_NOT_CONFIGURED: radarr not configured", err.Error())
	})

	t.Run("with cause", func(t *testing.T) {
		cause := errors.New("dial tcp: connection refused")
		err := &PluginError{Code: ErrCodeConnectionFailed, Message: "request failed", Cause: cause}
		assert.Equal(t, "DVR_CONNECTION_FAILED: request failed: dial tcp: connection refused", err.Error())
	})
}

func TestPluginError_Unwrap(t *testing.T) {
	cause := errors.New("underlying")
	err := &PluginError{Code: ErrCodeTimeout, Message: "timed out", Cause: cause}

	assert.ErrorIs(t, err, cause)

	var pluginErr *PluginError
	require.ErrorAs(t, fmt.Errorf("add movie: %w", err), &pluginErr)
	assert.Equal(t, ErrCodeTimeout, pluginErr.Code)
}

func TestErrorCodes_Values(t *testing.T) {
	// Rule 7 {SOURCE}_{ERROR_TYPE} — DVR_* new prefix + first live PLUGIN_* use (AC #7).
	assert.Equal(t, "DVR_NOT_CONFIGURED", ErrCodeNotConfigured)
	assert.Equal(t, "DVR_CONNECTION_FAILED", ErrCodeConnectionFailed)
	assert.Equal(t, "DVR_AUTH_FAILED", ErrCodeAuthFailed)
	assert.Equal(t, "DVR_TIMEOUT", ErrCodeTimeout)
	assert.Equal(t, "DVR_ADD_FAILED", ErrCodeAddFailed)
	assert.Equal(t, "DVR_TEST_FAILED", ErrCodeTestFailed)
	assert.Equal(t, "DVR_NOT_SUPPORTED", ErrCodeNotSupported)
	assert.Equal(t, "PLUGIN_INIT_FAILED", ErrCodePluginInitFailed)
	assert.Equal(t, "PLUGIN_HEALTH_CHECK_FAILED", ErrCodePluginHealthCheckFailed)
}

func TestPluginConfig_APIKeyNeverSerialized(t *testing.T) {
	// AC #1 — json:"-" is the guard: the masking slog handler is NOT wired,
	// so the key must never survive marshalling.
	cfg := PluginConfig{URL: "http://radarr:7878", APIKey: "super-secret"}
	data, err := json.Marshal(cfg)
	require.NoError(t, err)
	assert.Contains(t, string(data), "http://radarr:7878")
	assert.NotContains(t, string(data), "super-secret")
}

// compile-time proof that a DVRPlugin implementation satisfies the AC #1 shape.
type fakePlugin struct{}

func (f *fakePlugin) Name() string                                                  { return "fake" }
func (f *fakePlugin) TestConnection(ctx context.Context, config PluginConfig) error { return nil }
func (f *fakePlugin) AddMovie(ctx context.Context, tmdbID int64, opts AddOptions) (int64, error) {
	return 1, nil
}
func (f *fakePlugin) AddSeries(ctx context.Context, tmdbID int64, opts AddOptions) (int64, error) {
	return 0, &PluginError{Code: ErrCodeNotSupported, Message: "fake is movie-only"}
}
func (f *fakePlugin) GetQueue(ctx context.Context) ([]QueueItem, error) {
	return []QueueItem{}, nil
}

var _ DVRPlugin = (*fakePlugin)(nil)
