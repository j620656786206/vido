package tmdb

import (
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTMDbError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *TMDbError
		wantStr  string
	}{
		{
			name: "error without cause",
			err: &TMDbError{
				Code:    ErrCodeNotFound,
				Message: "Not found",
			},
			wantStr: "TMDB_NOT_FOUND: Not found",
		},
		{
			name: "error with cause",
			err: &TMDbError{
				Code:    ErrCodeTimeout,
				Message: "Timeout",
				Cause:   errors.New("connection reset"),
			},
			wantStr: "TMDB_TIMEOUT: Timeout (cause: connection reset)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.wantStr, tt.err.Error())
		})
	}
}

func TestTMDbError_Unwrap(t *testing.T) {
	cause := errors.New("underlying error")
	err := &TMDbError{
		Code:  ErrCodeServerError,
		Cause: cause,
	}

	assert.Equal(t, cause, err.Unwrap())
	assert.True(t, errors.Is(err, cause))
}

func TestNewTimeoutError(t *testing.T) {
	cause := errors.New("connection timeout")
	err := NewTimeoutError(cause)

	assert.Equal(t, ErrCodeTimeout, err.Code)
	assert.Equal(t, http.StatusGatewayTimeout, err.StatusCode)
	assert.Contains(t, err.Message, "timed out")
	assert.Equal(t, cause, err.Cause)
}

func TestNewRateLimitError(t *testing.T) {
	err := NewRateLimitError()

	assert.Equal(t, ErrCodeRateLimitExceeded, err.Code)
	assert.Equal(t, http.StatusTooManyRequests, err.StatusCode)
	assert.Contains(t, err.Message, "rate limit")
}

func TestNewNotFoundError(t *testing.T) {
	err := NewNotFoundError(550)

	assert.Equal(t, ErrCodeNotFound, err.Code)
	assert.Equal(t, http.StatusNotFound, err.StatusCode)
	assert.Contains(t, err.Message, "550")
}

func TestNewNotFoundErrorWithResource(t *testing.T) {
	tests := []struct {
		name     string
		resource string
		wantMsg  string
	}{
		{
			name:     "with resource",
			resource: "movie",
			wantMsg:  "movie",
		},
		{
			name:     "without resource",
			resource: "",
			wantMsg:  "TMDb resource not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewNotFoundErrorWithResource(tt.resource)
			assert.Contains(t, err.Message, tt.wantMsg)
		})
	}
}

func TestNewUnauthorizedError(t *testing.T) {
	tests := []struct {
		name    string
		message string
	}{
		{
			name:    "with custom message",
			message: "Invalid API key",
		},
		{
			name:    "with empty message",
			message: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewUnauthorizedError(tt.message)
			assert.Equal(t, ErrCodeUnauthorized, err.Code)
			assert.Equal(t, http.StatusUnauthorized, err.StatusCode)
		})
	}
}

func TestNewServerError(t *testing.T) {
	cause := errors.New("server error")
	err := NewServerError(cause)

	assert.Equal(t, ErrCodeServerError, err.Code)
	assert.Equal(t, http.StatusBadGateway, err.StatusCode)
	assert.Equal(t, cause, err.Cause)
}

func TestNewBadRequestError(t *testing.T) {
	tests := []struct {
		name    string
		message string
	}{
		{
			name:    "with custom message",
			message: "Invalid parameter",
		},
		{
			name:    "with empty message",
			message: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewBadRequestError(tt.message)
			assert.Equal(t, ErrCodeBadRequest, err.Code)
			assert.Equal(t, http.StatusBadRequest, err.StatusCode)
		})
	}
}

func TestParseAPIError(t *testing.T) {
	tests := []struct {
		name          string
		statusCode    int
		body          []byte
		wantErrorCode string
	}{
		{
			name:          "invalid API key",
			statusCode:    http.StatusUnauthorized,
			body:          []byte(`{"status_code": 7, "status_message": "Invalid API key"}`),
			wantErrorCode: ErrCodeUnauthorized,
		},
		{
			name:          "authentication failed",
			statusCode:    http.StatusUnauthorized,
			body:          []byte(`{"status_code": 30, "status_message": "Auth failed"}`),
			wantErrorCode: ErrCodeUnauthorized,
		},
		{
			name:          "resource not found",
			statusCode:    http.StatusNotFound,
			body:          []byte(`{"status_code": 34, "status_message": "Not found"}`),
			wantErrorCode: ErrCodeNotFound,
		},
		{
			name:          "rate limit exceeded",
			statusCode:    http.StatusTooManyRequests,
			body:          []byte(`{"status_code": 25, "status_message": "Rate limit"}`),
			wantErrorCode: ErrCodeRateLimitExceeded,
		},
		{
			name:          "invalid parameters",
			statusCode:    http.StatusBadRequest,
			body:          []byte(`{"status_code": 22, "status_message": "Invalid params"}`),
			wantErrorCode: ErrCodeBadRequest,
		},
		{
			name:          "unknown TMDb error",
			statusCode:    http.StatusBadRequest,
			body:          []byte(`{"status_code": 99, "status_message": "Unknown error"}`),
			wantErrorCode: ErrCodeServerError,
		},
		{
			name:          "unparseable response - rate limit",
			statusCode:    http.StatusTooManyRequests,
			body:          []byte(`not json`),
			wantErrorCode: ErrCodeRateLimitExceeded,
		},
		{
			name:          "unparseable response - unauthorized",
			statusCode:    http.StatusUnauthorized,
			body:          []byte(`not json`),
			wantErrorCode: ErrCodeUnauthorized,
		},
		{
			name:          "unparseable response - not found",
			statusCode:    http.StatusNotFound,
			body:          []byte(`not json`),
			wantErrorCode: ErrCodeNotFound,
		},
		{
			name:          "unparseable response - server error",
			statusCode:    http.StatusInternalServerError,
			body:          []byte(`not json`),
			wantErrorCode: ErrCodeServerError,
		},
		{
			name:          "unparseable response - bad request",
			statusCode:    http.StatusBadRequest,
			body:          []byte(`not json`),
			wantErrorCode: ErrCodeBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ParseAPIError(tt.statusCode, tt.body)
			assert.Equal(t, tt.wantErrorCode, err.Code)
		})
	}
}

func TestMapHTTPStatus(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		want       int
	}{
		{"server error", 500, http.StatusBadGateway},
		{"server error 503", 503, http.StatusBadGateway},
		{"rate limit", http.StatusTooManyRequests, http.StatusTooManyRequests},
		{"unauthorized", http.StatusUnauthorized, http.StatusUnauthorized},
		{"not found", http.StatusNotFound, http.StatusNotFound},
		{"bad request", http.StatusBadRequest, http.StatusBadRequest},
		{"other 4xx", 403, http.StatusBadRequest},
		{"default", 200, http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mapHTTPStatus(tt.statusCode)
			assert.Equal(t, tt.want, got)
		})
	}
}
