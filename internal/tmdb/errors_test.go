package tmdb

import (
	"errors"
	"net/http"
	"testing"
)

func TestNewRateLimitError(t *testing.T) {
	err := NewRateLimitError()

	if err.Code != ErrCodeRateLimitExceeded {
		t.Errorf("Code = %v, want %v", err.Code, ErrCodeRateLimitExceeded)
	}

	if err.StatusCode != http.StatusTooManyRequests {
		t.Errorf("StatusCode = %v, want %v", err.StatusCode, http.StatusTooManyRequests)
	}

	if err.Message == "" {
		t.Error("Message should not be empty")
	}
}

func TestNewNotFoundError(t *testing.T) {
	tests := []struct {
		name         string
		resource     string
		wantMessage  string
	}{
		{
			name:         "with resource",
			resource:     "movie",
			wantMessage:  "TMDb resource not found: movie",
		},
		{
			name:         "empty resource",
			resource:     "",
			wantMessage:  "TMDb resource not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewNotFoundError(tt.resource)

			if err.Code != ErrCodeNotFound {
				t.Errorf("Code = %v, want %v", err.Code, ErrCodeNotFound)
			}

			if err.StatusCode != http.StatusNotFound {
				t.Errorf("StatusCode = %v, want %v", err.StatusCode, http.StatusNotFound)
			}

			if err.Message != tt.wantMessage {
				t.Errorf("Message = %v, want %v", err.Message, tt.wantMessage)
			}
		})
	}
}

func TestNewUnauthorizedError(t *testing.T) {
	tests := []struct {
		name        string
		message     string
		wantMessage string
	}{
		{
			name:        "with custom message",
			message:     "Invalid API key",
			wantMessage: "Invalid API key",
		},
		{
			name:        "empty message",
			message:     "",
			wantMessage: "TMDb API authentication failed. Please check your API key.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewUnauthorizedError(tt.message)

			if err.Code != ErrCodeUnauthorized {
				t.Errorf("Code = %v, want %v", err.Code, ErrCodeUnauthorized)
			}

			if err.StatusCode != http.StatusUnauthorized {
				t.Errorf("StatusCode = %v, want %v", err.StatusCode, http.StatusUnauthorized)
			}

			if err.Message != tt.wantMessage {
				t.Errorf("Message = %v, want %v", err.Message, tt.wantMessage)
			}
		})
	}
}

func TestNewServerError(t *testing.T) {
	originalErr := errors.New("connection failed")
	err := NewServerError(originalErr)

	if err.Code != ErrCodeServerError {
		t.Errorf("Code = %v, want %v", err.Code, ErrCodeServerError)
	}

	if err.StatusCode != http.StatusBadGateway {
		t.Errorf("StatusCode = %v, want %v", err.StatusCode, http.StatusBadGateway)
	}

	if err.Message == "" {
		t.Error("Message should not be empty")
	}

	if err.Err != originalErr {
		t.Errorf("Err = %v, want %v", err.Err, originalErr)
	}
}

func TestNewBadRequestError(t *testing.T) {
	tests := []struct {
		name        string
		message     string
		wantMessage string
	}{
		{
			name:        "with custom message",
			message:     "Invalid query parameter",
			wantMessage: "Invalid query parameter",
		},
		{
			name:        "empty message",
			message:     "",
			wantMessage: "Invalid request parameters for TMDb API",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewBadRequestError(tt.message)

			if err.Code != ErrCodeBadRequest {
				t.Errorf("Code = %v, want %v", err.Code, ErrCodeBadRequest)
			}

			if err.StatusCode != http.StatusBadRequest {
				t.Errorf("StatusCode = %v, want %v", err.StatusCode, http.StatusBadRequest)
			}

			if err.Message != tt.wantMessage {
				t.Errorf("Message = %v, want %v", err.Message, tt.wantMessage)
			}
		})
	}
}

func TestParseAPIError(t *testing.T) {
	tests := []struct {
		name           string
		statusCode     int
		body           []byte
		wantCode       string
		wantStatusCode int
	}{
		{
			name:           "invalid API key",
			statusCode:     401,
			body:           []byte(`{"status_code":7,"status_message":"Invalid API key: You must be granted a valid key.","success":false}`),
			wantCode:       ErrCodeUnauthorized,
			wantStatusCode: http.StatusUnauthorized,
		},
		{
			name:           "resource not found",
			statusCode:     404,
			body:           []byte(`{"status_code":34,"status_message":"The resource you requested could not be found.","success":false}`),
			wantCode:       ErrCodeNotFound,
			wantStatusCode: http.StatusNotFound,
		},
		{
			name:           "rate limit exceeded",
			statusCode:     429,
			body:           []byte(`{"status_code":25,"status_message":"Your request count (30) is over the allowed limit of 20.","success":false}`),
			wantCode:       ErrCodeRateLimitExceeded,
			wantStatusCode: http.StatusTooManyRequests,
		},
		{
			name:           "invalid parameters",
			statusCode:     400,
			body:           []byte(`{"status_code":22,"status_message":"Invalid parameter: query","success":false}`),
			wantCode:       ErrCodeBadRequest,
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name:           "authentication failed",
			statusCode:     401,
			body:           []byte(`{"status_code":30,"status_message":"Authentication failed.","success":false}`),
			wantCode:       ErrCodeUnauthorized,
			wantStatusCode: http.StatusUnauthorized,
		},
		{
			name:           "unknown TMDb error",
			statusCode:     500,
			body:           []byte(`{"status_code":999,"status_message":"Unknown error occurred.","success":false}`),
			wantCode:       ErrCodeServerError,
			wantStatusCode: http.StatusBadGateway,
		},
		{
			name:           "unparseable error - 429",
			statusCode:     429,
			body:           []byte(`not json`),
			wantCode:       ErrCodeRateLimitExceeded,
			wantStatusCode: http.StatusTooManyRequests,
		},
		{
			name:           "unparseable error - 401",
			statusCode:     401,
			body:           []byte(`not json`),
			wantCode:       ErrCodeUnauthorized,
			wantStatusCode: http.StatusUnauthorized,
		},
		{
			name:           "unparseable error - 404",
			statusCode:     404,
			body:           []byte(`not json`),
			wantCode:       ErrCodeNotFound,
			wantStatusCode: http.StatusNotFound,
		},
		{
			name:           "unparseable error - 500",
			statusCode:     500,
			body:           []byte(`server error`),
			wantCode:       ErrCodeServerError,
			wantStatusCode: http.StatusBadGateway,
		},
		{
			name:           "unparseable error - 503",
			statusCode:     503,
			body:           []byte(`service unavailable`),
			wantCode:       ErrCodeServerError,
			wantStatusCode: http.StatusBadGateway,
		},
		{
			name:           "unparseable error - 400",
			statusCode:     400,
			body:           []byte(`bad request`),
			wantCode:       ErrCodeBadRequest,
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name:           "unparseable error - unknown",
			statusCode:     418,
			body:           []byte(`I'm a teapot`),
			wantCode:       ErrCodeBadRequest,
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name:           "unparseable error - unexpected",
			statusCode:     200,
			body:           []byte(`unexpected`),
			wantCode:       ErrCodeServerError,
			wantStatusCode: http.StatusBadGateway,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ParseAPIError(tt.statusCode, tt.body)

			if err.Code != tt.wantCode {
				t.Errorf("Code = %v, want %v", err.Code, tt.wantCode)
			}

			if err.StatusCode != tt.wantStatusCode {
				t.Errorf("StatusCode = %v, want %v", err.StatusCode, tt.wantStatusCode)
			}

			if err.Message == "" {
				t.Error("Message should not be empty")
			}
		})
	}
}

func TestMapHTTPStatus(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		want       int
	}{
		{
			name:       "500 error",
			statusCode: 500,
			want:       http.StatusBadGateway,
		},
		{
			name:       "503 error",
			statusCode: 503,
			want:       http.StatusBadGateway,
		},
		{
			name:       "429 rate limit",
			statusCode: 429,
			want:       http.StatusTooManyRequests,
		},
		{
			name:       "401 unauthorized",
			statusCode: 401,
			want:       http.StatusUnauthorized,
		},
		{
			name:       "404 not found",
			statusCode: 404,
			want:       http.StatusNotFound,
		},
		{
			name:       "400 bad request",
			statusCode: 400,
			want:       http.StatusBadRequest,
		},
		{
			name:       "418 other 4xx",
			statusCode: 418,
			want:       http.StatusBadRequest,
		},
		{
			name:       "200 unexpected success",
			statusCode: 200,
			want:       http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mapHTTPStatus(tt.statusCode)
			if got != tt.want {
				t.Errorf("mapHTTPStatus() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMapHTTPStatusToError(t *testing.T) {
	tests := []struct {
		name           string
		statusCode     int
		body           []byte
		wantCode       string
		wantStatusCode int
	}{
		{
			name:           "429 rate limit",
			statusCode:     429,
			body:           nil,
			wantCode:       ErrCodeRateLimitExceeded,
			wantStatusCode: http.StatusTooManyRequests,
		},
		{
			name:           "401 unauthorized",
			statusCode:     401,
			body:           nil,
			wantCode:       ErrCodeUnauthorized,
			wantStatusCode: http.StatusUnauthorized,
		},
		{
			name:           "404 not found",
			statusCode:     404,
			body:           nil,
			wantCode:       ErrCodeNotFound,
			wantStatusCode: http.StatusNotFound,
		},
		{
			name:           "500 server error",
			statusCode:     500,
			body:           nil,
			wantCode:       ErrCodeServerError,
			wantStatusCode: http.StatusBadGateway,
		},
		{
			name:           "503 server error",
			statusCode:     503,
			body:           nil,
			wantCode:       ErrCodeServerError,
			wantStatusCode: http.StatusBadGateway,
		},
		{
			name:           "400 bad request with body",
			statusCode:     400,
			body:           []byte("invalid parameter"),
			wantCode:       ErrCodeBadRequest,
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name:           "400 bad request without body",
			statusCode:     400,
			body:           []byte{},
			wantCode:       ErrCodeBadRequest,
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name:           "unexpected status code",
			statusCode:     200,
			body:           nil,
			wantCode:       ErrCodeServerError,
			wantStatusCode: http.StatusBadGateway,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := mapHTTPStatusToError(tt.statusCode, tt.body)

			if err.Code != tt.wantCode {
				t.Errorf("Code = %v, want %v", err.Code, tt.wantCode)
			}

			if err.StatusCode != tt.wantStatusCode {
				t.Errorf("StatusCode = %v, want %v", err.StatusCode, tt.wantStatusCode)
			}

			if err.Message == "" {
				t.Error("Message should not be empty")
			}
		})
	}
}
