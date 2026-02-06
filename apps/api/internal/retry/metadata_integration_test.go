package retry

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsRetryableMetadataError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "timeout error",
			err:      errors.New("METADATA_TIMEOUT: request timed out"),
			expected: true,
		},
		{
			name:     "rate limited error",
			err:      errors.New("TMDb: METADATA_RATE_LIMITED: too many requests"),
			expected: true,
		},
		{
			name:     "service unavailable",
			err:      errors.New("METADATA_UNAVAILABLE: service temporarily down"),
			expected: true,
		},
		{
			name:     "circuit breaker open",
			err:      errors.New("METADATA_CIRCUIT_OPEN: circuit breaker is open"),
			expected: true,
		},
		{
			name:     "generic timeout",
			err:      errors.New("request timeout after 30s"),
			expected: true,
		},
		{
			name:     "connection refused",
			err:      errors.New("connection refused: dial tcp"),
			expected: true,
		},
		{
			name:     "503 error",
			err:      errors.New("HTTP 503 Service Unavailable"),
			expected: true,
		},
		{
			name:     "429 error",
			err:      errors.New("HTTP 429 Too Many Requests"),
			expected: true,
		},
		{
			name:     "context deadline exceeded",
			err:      errors.New("context deadline exceeded"),
			expected: true,
		},
		{
			name:     "no results - not retryable",
			err:      errors.New("METADATA_NO_RESULTS: no results found"),
			expected: false,
		},
		{
			name:     "not found - not retryable",
			err:      errors.New("movie not found"),
			expected: false,
		},
		{
			name:     "invalid request - not retryable",
			err:      errors.New("invalid search query"),
			expected: false,
		},
		{
			name:     "unauthorized - not retryable",
			err:      errors.New("unauthorized: invalid API key"),
			expected: false,
		},
		{
			name:     "forbidden - not retryable",
			err:      errors.New("forbidden: access denied"),
			expected: false,
		},
		{
			name:     "bad gateway",
			err:      errors.New("502 Bad Gateway"),
			expected: true,
		},
		{
			name:     "gateway timeout",
			err:      errors.New("504 Gateway Timeout"),
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsRetryableMetadataError(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestClassifyMetadataError(t *testing.T) {
	tests := []struct {
		name         string
		err          error
		expectedCode string
		retryable    bool
	}{
		{
			name:         "nil error",
			err:          nil,
			expectedCode: "",
			retryable:    false,
		},
		{
			name:         "timeout error",
			err:          errors.New("METADATA_TIMEOUT: request timed out"),
			expectedCode: "METADATA_TIMEOUT",
			retryable:    true,
		},
		{
			name:         "context deadline exceeded",
			err:          errors.New("context deadline exceeded"),
			expectedCode: "METADATA_TIMEOUT",
			retryable:    true,
		},
		{
			name:         "rate limited",
			err:          errors.New("rate limit exceeded"),
			expectedCode: "METADATA_RATE_LIMITED",
			retryable:    true,
		},
		{
			name:         "429 error",
			err:          errors.New("HTTP 429 Too Many Requests"),
			expectedCode: "METADATA_RATE_LIMITED",
			retryable:    true,
		},
		{
			name:         "service unavailable",
			err:          errors.New("503 Service Unavailable"),
			expectedCode: "METADATA_UNAVAILABLE",
			retryable:    true,
		},
		{
			name:         "circuit open",
			err:          errors.New("circuit breaker is open"),
			expectedCode: "METADATA_CIRCUIT_OPEN",
			retryable:    true,
		},
		{
			name:         "bad gateway",
			err:          errors.New("502 Bad Gateway"),
			expectedCode: "METADATA_GATEWAY_ERROR",
			retryable:    true,
		},
		{
			name:         "gateway timeout",
			err:          errors.New("504 Gateway Timeout"),
			expectedCode: "METADATA_TIMEOUT", // "Timeout" is matched first
			retryable:    true,
		},
		{
			name:         "connection error",
			err:          errors.New("connection refused"),
			expectedCode: "METADATA_NETWORK_ERROR",
			retryable:    true,
		},
		{
			name:         "no results",
			err:          errors.New("no results found"),
			expectedCode: "METADATA_NO_RESULTS",
			retryable:    false,
		},
		{
			name:         "not found",
			err:          errors.New("404 Not Found"),
			expectedCode: "METADATA_NOT_FOUND",
			retryable:    false,
		},
		{
			name:         "unknown error",
			err:          errors.New("something went wrong"),
			expectedCode: "METADATA_UNKNOWN_ERROR",
			retryable:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ClassifyMetadataError(tt.err)

			if tt.err == nil {
				assert.Nil(t, result)
				return
			}

			assert.NotNil(t, result)
			assert.Equal(t, tt.expectedCode, result.Code)
			assert.Equal(t, tt.retryable, result.Retryable)
		})
	}
}

func TestExtractRetryableErrors(t *testing.T) {
	tests := []struct {
		name     string
		errs     []error
		expected int
	}{
		{
			name:     "empty list",
			errs:     []error{},
			expected: 0,
		},
		{
			name:     "all nil",
			errs:     []error{nil, nil, nil},
			expected: 0,
		},
		{
			name: "all retryable",
			errs: []error{
				errors.New("timeout"),
				errors.New("rate limit"),
				errors.New("503 unavailable"),
			},
			expected: 3,
		},
		{
			name: "mixed",
			errs: []error{
				errors.New("timeout"),
				errors.New("not found"),
				errors.New("rate limit"),
			},
			expected: 2,
		},
		{
			name: "all non-retryable",
			errs: []error{
				errors.New("not found"),
				errors.New("invalid input"),
				errors.New("unauthorized"),
			},
			expected: 0,
		},
		{
			name: "with nil values",
			errs: []error{
				nil,
				errors.New("timeout"),
				nil,
				errors.New("rate limit"),
			},
			expected: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractRetryableErrors(tt.errs)
			assert.Len(t, result, tt.expected)
		})
	}
}

func TestShouldQueueRetry(t *testing.T) {
	tests := []struct {
		name     string
		errs     []error
		expected bool
	}{
		{
			name:     "empty list",
			errs:     []error{},
			expected: false,
		},
		{
			name:     "all nil",
			errs:     []error{nil, nil},
			expected: false,
		},
		{
			name: "has retryable errors",
			errs: []error{
				errors.New("timeout"),
				errors.New("no results"),
			},
			expected: true,
		},
		{
			name: "all no results",
			errs: []error{
				errors.New("no results"),
				errors.New("METADATA_NO_RESULTS"),
			},
			expected: false,
		},
		{
			name: "retryable and non-retryable",
			errs: []error{
				errors.New("timeout"),
				errors.New("not found"),
			},
			expected: true,
		},
		{
			name: "only non-retryable (not no results)",
			errs: []error{
				errors.New("not found"),
				errors.New("invalid input"),
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ShouldQueueRetry(tt.errs)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestWrapAsRetryable(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		isRetryErr bool
	}{
		{
			name:       "nil error",
			err:        nil,
			isRetryErr: false,
		},
		{
			name:       "timeout error",
			err:        errors.New("request timeout"),
			isRetryErr: true,
		},
		{
			name:       "unknown error",
			err:        errors.New("something random"),
			isRetryErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := WrapAsRetryable(tt.err)

			if tt.err == nil {
				assert.Nil(t, result)
				return
			}

			if tt.isRetryErr {
				_, ok := result.(*RetryableError)
				assert.True(t, ok, "expected *RetryableError")
			}
		})
	}
}

func TestIsTemporaryError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "timeout error",
			err:      errors.New("timeout"),
			expected: true,
		},
		{
			name:     "connection error",
			err:      errors.New("connection refused"),
			expected: true,
		},
		{
			name:     "not found error",
			err:      errors.New("not found"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsTemporaryError(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}
