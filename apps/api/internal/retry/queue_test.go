package retry

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRetryableError_Error(t *testing.T) {
	err := &RetryableError{
		Code:    "TEST_ERROR",
		Message: "This is a test error",
	}

	assert.Equal(t, "This is a test error", err.Error())
}

func TestRetryableError_IsRetryable(t *testing.T) {
	tests := []struct {
		name     string
		err      *RetryableError
		expected bool
	}{
		{
			name:     "retryable error - timeout",
			err:      ErrTimeout,
			expected: true,
		},
		{
			name:     "retryable error - rate limit",
			err:      ErrRateLimit,
			expected: true,
		},
		{
			name:     "retryable error - network error",
			err:      ErrNetworkError,
			expected: true,
		},
		{
			name:     "retryable error - service down",
			err:      ErrServiceDown,
			expected: true,
		},
		{
			name:     "retryable error - bad gateway",
			err:      ErrBadGateway,
			expected: true,
		},
		{
			name:     "retryable error - gateway timeout",
			err:      ErrGatewayTimeout,
			expected: true,
		},
		{
			name:     "non-retryable error - not found",
			err:      ErrNotFound,
			expected: false,
		},
		{
			name:     "non-retryable error - invalid input",
			err:      ErrInvalidInput,
			expected: false,
		},
		{
			name:     "non-retryable error - unauthorized",
			err:      ErrUnauthorized,
			expected: false,
		},
		{
			name:     "non-retryable error - forbidden",
			err:      ErrForbidden,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.err.IsRetryable())
		})
	}
}

func TestNewRetryableError(t *testing.T) {
	err := NewRetryableError("CUSTOM_ERROR", "Custom error message", true, 500)

	assert.Equal(t, "CUSTOM_ERROR", err.Code)
	assert.Equal(t, "Custom error message", err.Message)
	assert.True(t, err.Retryable)
	assert.Equal(t, 500, err.StatusCode)
}

func TestWrapRetryable(t *testing.T) {
	originalErr := assert.AnError
	wrapped := WrapRetryable(originalErr, "WRAPPED_ERROR")

	assert.Equal(t, "WRAPPED_ERROR", wrapped.Code)
	assert.Equal(t, originalErr.Error(), wrapped.Message)
	assert.True(t, wrapped.Retryable)
}

func TestWrapNonRetryable(t *testing.T) {
	originalErr := assert.AnError
	wrapped := WrapNonRetryable(originalErr, "WRAPPED_ERROR")

	assert.Equal(t, "WRAPPED_ERROR", wrapped.Code)
	assert.Equal(t, originalErr.Error(), wrapped.Message)
	assert.False(t, wrapped.Retryable)
}

func TestRetryItem_ToResponse(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name           string
		item           *RetryItem
		expectedPrefix string
	}{
		{
			name: "future retry time",
			item: &RetryItem{
				ID:            "retry-1",
				TaskID:        "task-1",
				TaskType:      TaskTypeParse,
				AttemptCount:  1,
				MaxAttempts:   4,
				LastError:     "TMDb timeout",
				NextAttemptAt: now.Add(5 * time.Second),
			},
			expectedPrefix: "5s",
		},
		{
			name: "past retry time",
			item: &RetryItem{
				ID:            "retry-2",
				TaskID:        "task-2",
				TaskType:      TaskTypeMetadataFetch,
				AttemptCount:  2,
				MaxAttempts:   4,
				LastError:     "Network error",
				NextAttemptAt: now.Add(-1 * time.Second),
			},
			expectedPrefix: "即將重試",
		},
		{
			name: "immediate retry",
			item: &RetryItem{
				ID:            "retry-3",
				TaskID:        "task-3",
				TaskType:      TaskTypeParse,
				AttemptCount:  0,
				MaxAttempts:   4,
				NextAttemptAt: now,
			},
			expectedPrefix: "即將重試",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := tt.item.ToResponse()

			assert.Equal(t, tt.item.ID, resp.ID)
			assert.Equal(t, tt.item.TaskID, resp.TaskID)
			assert.Equal(t, tt.item.TaskType, resp.TaskType)
			assert.Equal(t, tt.item.AttemptCount, resp.AttemptCount)
			assert.Equal(t, tt.item.MaxAttempts, resp.MaxAttempts)
			assert.Equal(t, tt.item.LastError, resp.LastError)
			assert.Equal(t, tt.item.NextAttemptAt, resp.NextAttemptAt)
			// TimeUntilRetry is dynamic, just check it's not empty
			assert.NotEmpty(t, resp.TimeUntilRetry)
		})
	}
}

func TestRetryItem_JSONSerialization(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	payload := json.RawMessage(`{"filename":"test.mkv"}`)

	item := &RetryItem{
		ID:            "retry-uuid-123",
		TaskID:        "parse-task-456",
		TaskType:      TaskTypeParse,
		Payload:       payload,
		AttemptCount:  2,
		MaxAttempts:   4,
		LastError:     "Service temporarily unavailable",
		NextAttemptAt: now.Add(4 * time.Second),
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	// Serialize
	data, err := json.Marshal(item)
	require.NoError(t, err)

	// Deserialize
	var decoded RetryItem
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, item.ID, decoded.ID)
	assert.Equal(t, item.TaskID, decoded.TaskID)
	assert.Equal(t, item.TaskType, decoded.TaskType)
	assert.Equal(t, item.AttemptCount, decoded.AttemptCount)
	assert.Equal(t, item.MaxAttempts, decoded.MaxAttempts)
	assert.Equal(t, item.LastError, decoded.LastError)
	assert.Equal(t, string(item.Payload), string(decoded.Payload))
}

func TestTaskTypeConstants(t *testing.T) {
	assert.Equal(t, "parse", TaskTypeParse)
	assert.Equal(t, "metadata_fetch", TaskTypeMetadataFetch)
}

func TestMaxRetryAttempts(t *testing.T) {
	assert.Equal(t, 4, MaxRetryAttempts)
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		expected string
	}{
		{
			name:     "less than a second",
			duration: 500 * time.Millisecond,
			expected: "即將重試",
		},
		{
			name:     "exactly 1 second",
			duration: 1 * time.Second,
			expected: "1s",
		},
		{
			name:     "5 seconds",
			duration: 5 * time.Second,
			expected: "5s",
		},
		{
			name:     "59 seconds",
			duration: 59 * time.Second,
			expected: "59s",
		},
		{
			name:     "1 minute exactly",
			duration: 60 * time.Second,
			expected: "1m",
		},
		{
			name:     "1 minute 30 seconds",
			duration: 90 * time.Second,
			expected: "1m30s",
		},
		{
			name:     "5 minutes",
			duration: 5 * time.Minute,
			expected: "5m",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatDuration(tt.duration)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRetryStats(t *testing.T) {
	stats := &RetryStats{
		TotalPending:   5,
		TotalSucceeded: 10,
		TotalFailed:    2,
	}

	data, err := json.Marshal(stats)
	require.NoError(t, err)

	var decoded RetryStats
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, 5, decoded.TotalPending)
	assert.Equal(t, 10, decoded.TotalSucceeded)
	assert.Equal(t, 2, decoded.TotalFailed)
}

func TestRetryResponse(t *testing.T) {
	now := time.Now()
	resp := &RetryResponse{
		Items: []*RetryItemResponse{
			{
				ID:             "retry-1",
				TaskID:         "task-1",
				TaskType:       TaskTypeParse,
				AttemptCount:   1,
				MaxAttempts:    4,
				LastError:      "timeout",
				NextAttemptAt:  now,
				TimeUntilRetry: "2s",
			},
		},
		Stats: &RetryStats{
			TotalPending:   1,
			TotalSucceeded: 5,
			TotalFailed:    0,
		},
	}

	data, err := json.Marshal(resp)
	require.NoError(t, err)

	var decoded RetryResponse
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	require.Len(t, decoded.Items, 1)
	assert.Equal(t, "retry-1", decoded.Items[0].ID)
	assert.Equal(t, 1, decoded.Stats.TotalPending)
}

func TestPredefinedErrors(t *testing.T) {
	// Test that predefined errors have correct codes and status codes
	tests := []struct {
		name       string
		err        *RetryableError
		code       string
		retryable  bool
		statusCode int
	}{
		{"timeout", ErrTimeout, "TIMEOUT", true, 0},
		{"rate limit", ErrRateLimit, "RATE_LIMIT", true, 429},
		{"network error", ErrNetworkError, "NETWORK_ERROR", true, 0},
		{"service down", ErrServiceDown, "SERVICE_DOWN", true, 503},
		{"bad gateway", ErrBadGateway, "BAD_GATEWAY", true, 502},
		{"gateway timeout", ErrGatewayTimeout, "GATEWAY_TIMEOUT", true, 504},
		{"not found", ErrNotFound, "NOT_FOUND", false, 404},
		{"invalid input", ErrInvalidInput, "INVALID_INPUT", false, 0},
		{"unauthorized", ErrUnauthorized, "UNAUTHORIZED", false, 401},
		{"forbidden", ErrForbidden, "FORBIDDEN", false, 403},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.code, tt.err.Code)
			assert.Equal(t, tt.retryable, tt.err.Retryable)
			assert.Equal(t, tt.statusCode, tt.err.StatusCode)
		})
	}
}
