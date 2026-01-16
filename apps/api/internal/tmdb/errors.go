package tmdb

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// TMDb-specific error codes following project convention: SOURCE_ERROR_TYPE
const (
	// ErrCodeTimeout indicates the API request timed out
	ErrCodeTimeout = "TMDB_TIMEOUT"
	// ErrCodeRateLimitExceeded indicates the API rate limit has been exceeded
	ErrCodeRateLimitExceeded = "TMDB_RATE_LIMIT"
	// ErrCodeNotFound indicates the requested resource was not found
	ErrCodeNotFound = "TMDB_NOT_FOUND"
	// ErrCodeUnauthorized indicates authentication failed or API key is invalid
	ErrCodeUnauthorized = "TMDB_UNAUTHORIZED"
	// ErrCodeServerError indicates a TMDb server error
	ErrCodeServerError = "TMDB_SERVER_ERROR"
	// ErrCodeBadRequest indicates invalid request parameters
	ErrCodeBadRequest = "TMDB_BAD_REQUEST"
)

// TMDb API status codes from their documentation
const (
	TMDbStatusInvalidAPIKey        = 7
	TMDbStatusResourceNotFound     = 34
	TMDbStatusRateLimitExceeded    = 25
	TMDbStatusInvalidParameters    = 22
	TMDbStatusAuthenticationFailed = 30
)

// TMDbError represents a TMDb API error with standardized fields
// This follows the project's AppError pattern for consistent error handling
type TMDbError struct {
	Code       string `json:"code"`
	Message    string `json:"message"`
	Suggestion string `json:"suggestion,omitempty"`
	StatusCode int    `json:"-"` // HTTP status code for response
	Cause      error  `json:"-"` // Underlying error
}

// Error implements the error interface
func (e *TMDbError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s (cause: %v)", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Unwrap returns the underlying error for errors.Is/As support
func (e *TMDbError) Unwrap() error {
	return e.Cause
}

// NewTimeoutError creates an error for request timeouts
func NewTimeoutError(err error) *TMDbError {
	return &TMDbError{
		Code:       ErrCodeTimeout,
		Message:    "TMDb API request timed out",
		Suggestion: "Please try again in a few moments",
		StatusCode: http.StatusGatewayTimeout,
		Cause:      err,
	}
}

// NewRateLimitError creates an error for rate limit exceeded
func NewRateLimitError() *TMDbError {
	return &TMDbError{
		Code:       ErrCodeRateLimitExceeded,
		Message:    "TMDb API rate limit exceeded",
		Suggestion: "Please wait a moment before retrying",
		StatusCode: http.StatusTooManyRequests,
	}
}

// NewNotFoundError creates an error for resource not found
func NewNotFoundError(id int) *TMDbError {
	return &TMDbError{
		Code:       ErrCodeNotFound,
		Message:    fmt.Sprintf("Media with ID %d not found in TMDb", id),
		Suggestion: "Verify the TMDb ID is correct",
		StatusCode: http.StatusNotFound,
	}
}

// NewNotFoundErrorWithResource creates an error for resource not found with custom message
func NewNotFoundErrorWithResource(resource string) *TMDbError {
	message := "TMDb resource not found"
	if resource != "" {
		message = fmt.Sprintf("TMDb resource not found: %s", resource)
	}
	return &TMDbError{
		Code:       ErrCodeNotFound,
		Message:    message,
		Suggestion: "Verify the resource identifier is correct",
		StatusCode: http.StatusNotFound,
	}
}

// NewUnauthorizedError creates an error for authentication failures
func NewUnauthorizedError(message string) *TMDbError {
	if message == "" {
		message = "TMDb API authentication failed. Please check your API key."
	}
	return &TMDbError{
		Code:       ErrCodeUnauthorized,
		Message:    message,
		Suggestion: "Verify your TMDb API key is valid",
		StatusCode: http.StatusUnauthorized,
	}
}

// NewServerError creates an error for TMDb server errors
func NewServerError(err error) *TMDbError {
	return &TMDbError{
		Code:       ErrCodeServerError,
		Message:    "TMDb API server error occurred",
		Suggestion: "TMDb may be experiencing issues, please try again later",
		StatusCode: http.StatusBadGateway,
		Cause:      err,
	}
}

// NewBadRequestError creates an error for invalid request parameters
func NewBadRequestError(message string) *TMDbError {
	if message == "" {
		message = "Invalid request parameters for TMDb API"
	}
	return &TMDbError{
		Code:       ErrCodeBadRequest,
		Message:    message,
		Suggestion: "Please check your request parameters",
		StatusCode: http.StatusBadRequest,
	}
}

// ParseAPIError parses a TMDb API error response and returns an appropriate TMDbError
// This function converts TMDb error responses into our application's error types
func ParseAPIError(statusCode int, body []byte) *TMDbError {
	// Try to parse TMDb error response
	var errorResp ErrorResponse
	if err := json.Unmarshal(body, &errorResp); err == nil && errorResp.StatusMessage != "" {
		// Map TMDb status codes to our error types
		switch errorResp.StatusCode {
		case TMDbStatusInvalidAPIKey, TMDbStatusAuthenticationFailed:
			return NewUnauthorizedError(errorResp.StatusMessage)
		case TMDbStatusResourceNotFound:
			return NewNotFoundErrorWithResource("")
		case TMDbStatusRateLimitExceeded:
			return NewRateLimitError()
		case TMDbStatusInvalidParameters:
			return NewBadRequestError(errorResp.StatusMessage)
		default:
			// For other TMDb-specific errors, create a server error with the message
			return &TMDbError{
				Code:       ErrCodeServerError,
				Message:    fmt.Sprintf("TMDb API error: %s", errorResp.StatusMessage),
				Suggestion: "Please try again later",
				StatusCode: mapHTTPStatus(statusCode),
				Cause:      fmt.Errorf("TMDb status code %d: %s", errorResp.StatusCode, errorResp.StatusMessage),
			}
		}
	}

	// If we can't parse the error response, map based on HTTP status code
	return mapHTTPStatusToError(statusCode, body)
}

// mapHTTPStatus maps TMDb HTTP status codes to appropriate HTTP status codes
func mapHTTPStatus(statusCode int) int {
	switch {
	case statusCode >= 500:
		return http.StatusBadGateway
	case statusCode == http.StatusTooManyRequests:
		return http.StatusTooManyRequests
	case statusCode == http.StatusUnauthorized:
		return http.StatusUnauthorized
	case statusCode == http.StatusNotFound:
		return http.StatusNotFound
	case statusCode >= 400:
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}

// mapHTTPStatusToError creates an appropriate error based on HTTP status code
func mapHTTPStatusToError(statusCode int, body []byte) *TMDbError {
	switch {
	case statusCode == http.StatusTooManyRequests:
		return NewRateLimitError()
	case statusCode == http.StatusUnauthorized:
		return NewUnauthorizedError("")
	case statusCode == http.StatusNotFound:
		return NewNotFoundErrorWithResource("")
	case statusCode >= 500:
		return NewServerError(fmt.Errorf("TMDb server error: HTTP %d", statusCode))
	case statusCode >= 400:
		// Try to extract message from body
		message := fmt.Sprintf("TMDb API error: HTTP %d", statusCode)
		if len(body) > 0 {
			message = fmt.Sprintf("TMDb API error: HTTP %d - %s", statusCode, string(body))
		}
		return NewBadRequestError(message)
	default:
		return NewServerError(fmt.Errorf("unexpected TMDb API response: HTTP %d", statusCode))
	}
}
