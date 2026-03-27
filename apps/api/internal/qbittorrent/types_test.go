package qbittorrent

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestConfig_JSONSerialization(t *testing.T) {
	cfg := &Config{
		Host:     "http://192.168.1.100:8080",
		Username: "admin",
		Password: "secret",
		BasePath: "/qbt",
		Timeout:  10 * time.Second,
	}

	data, err := json.Marshal(cfg)
	assert.NoError(t, err)

	// Password and Timeout should not be serialized (json:"-")
	var result map[string]interface{}
	err = json.Unmarshal(data, &result)
	assert.NoError(t, err)

	assert.Equal(t, "http://192.168.1.100:8080", result["host"])
	assert.Equal(t, "admin", result["username"])
	assert.Equal(t, "/qbt", result["base_path"])
	assert.NotContains(t, result, "password", "password should not be serialized")
	assert.NotContains(t, result, "timeout", "timeout should not be serialized")
}

func TestVersionInfo_JSONSerialization(t *testing.T) {
	info := &VersionInfo{
		AppVersion: "v4.5.2",
		APIVersion: "2.9.3",
	}

	data, err := json.Marshal(info)
	assert.NoError(t, err)

	var result map[string]interface{}
	err = json.Unmarshal(data, &result)
	assert.NoError(t, err)

	assert.Equal(t, "v4.5.2", result["app_version"])
	assert.Equal(t, "2.9.3", result["api_version"])
}

func TestConnectionError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *ConnectionError
		expected string
	}{
		{
			name: "error with cause",
			err: &ConnectionError{
				Code:    ErrCodeConnectionFailed,
				Message: "login request failed",
				Cause:   errors.New("connection refused"),
			},
			expected: "QB_CONNECTION_FAILED: login request failed: connection refused",
		},
		{
			name: "error without cause",
			err: &ConnectionError{
				Code:    ErrCodeAuthFailed,
				Message: "authentication failed: invalid credentials",
			},
			expected: "QB_AUTH_FAILED: authentication failed: invalid credentials",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.err.Error())
		})
	}
}

func TestConnectionError_Unwrap(t *testing.T) {
	cause := errors.New("underlying error")
	err := &ConnectionError{
		Code:    ErrCodeConnectionFailed,
		Message: "test",
		Cause:   cause,
	}

	assert.ErrorIs(t, err, cause)
}

func TestConnectionError_Unwrap_NilCause(t *testing.T) {
	err := &ConnectionError{
		Code:    ErrCodeAuthFailed,
		Message: "test",
	}

	assert.Nil(t, err.Unwrap())
}

func TestErrorCodeConstants(t *testing.T) {
	assert.Equal(t, "QB_CONNECTION_FAILED", ErrCodeConnectionFailed)
	assert.Equal(t, "QB_AUTH_FAILED", ErrCodeAuthFailed)
	assert.Equal(t, "QB_TIMEOUT", ErrCodeTimeout)
	assert.Equal(t, "QB_NOT_CONFIGURED", ErrCodeNotConfigured)
}
