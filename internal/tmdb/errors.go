package tmdb

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/alexyu/vido/internal/middleware"
)

// TMDb-specific error codes
const (
	// ErrCodeRateLimitExceeded indicates the API rate limit has been exceeded
	ErrCodeRateLimitExceeded = "TMDB_RATE_LIMIT_EXCEEDED"
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
	TMDbStatusInvalidAPIKey      = 7
	TMDbStatusResourceNotFound   = 34
	TMDbStatusRateLimitExceeded  = 25
	TMDbStatusInvalidParameters  = 22
	TMDbStatusAuthenticationFailed = 30
)

// NewRateLimitError creates an error for rate limit exceeded
func NewRateLimitError() *middleware.AppError {
	return &middleware.AppError{
		Code:       ErrCodeRateLimitExceeded,
		Message:    "TMDb API rate limit exceeded. Please try again later.",
		StatusCode: http.StatusTooManyRequests,
	}
}

// NewNotFoundError creates an error for resource not found
func NewNotFoundError(resource string) *middleware.AppError {
	message := fmt.Sprintf("TMDb resource not found: %s", resource)
	if resource == "" {
		message = "TMDb resource not found"
	}
	return &middleware.AppError{
		Code:       ErrCodeNotFound,
		Message:    message,
		StatusCode: http.StatusNotFound,
	}
}

// NewUnauthorizedError creates an error for authentication failures
func NewUnauthorizedError(message string) *middleware.AppError {
	if message == "" {
		message = "TMDb API authentication failed. Please check your API key."
	}
	return &middleware.AppError{
		Code:       ErrCodeUnauthorized,
		Message:    message,
		StatusCode: http.StatusUnauthorized,
	}
}

// NewServerError creates an error for TMDb server errors
func NewServerError(err error) *middleware.AppError {
	return &middleware.AppError{
		Code:       ErrCodeServerError,
		Message:    "TMDb API server error occurred",
		StatusCode: http.StatusBadGateway,
		Err:        err,
	}
}

// NewBadRequestError creates an error for invalid request parameters
func NewBadRequestError(message string) *middleware.AppError {
	if message == "" {
		message = "Invalid request parameters for TMDb API"
	}
	return &middleware.AppError{
		Code:       ErrCodeBadRequest,
		Message:    message,
		StatusCode: http.StatusBadRequest,
	}
}

// ParseAPIError parses a TMDb API error response and returns an appropriate AppError
// This function converts TMDb error responses into our application's error types
func ParseAPIError(statusCode int, body []byte) *middleware.AppError {
	// Try to parse TMDb error response
	var errorResp ErrorResponse
	if err := json.Unmarshal(body, &errorResp); err == nil && errorResp.StatusMessage != "" {
		// Map TMDb status codes to our error types
		switch errorResp.StatusCode {
		case TMDbStatusInvalidAPIKey, TMDbStatusAuthenticationFailed:
			return NewUnauthorizedError(errorResp.StatusMessage)
		case TMDbStatusResourceNotFound:
			return NewNotFoundError("")
		case TMDbStatusRateLimitExceeded:
			return NewRateLimitError()
		case TMDbStatusInvalidParameters:
			return NewBadRequestError(errorResp.StatusMessage)
		default:
			// For other TMDb-specific errors, create a server error with the message
			return &middleware.AppError{
				Code:       ErrCodeServerError,
				Message:    fmt.Sprintf("TMDb API error: %s", errorResp.StatusMessage),
				StatusCode: mapHTTPStatus(statusCode),
				Err:        fmt.Errorf("TMDb status code %d: %s", errorResp.StatusCode, errorResp.StatusMessage),
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
func mapHTTPStatusToError(statusCode int, body []byte) *middleware.AppError {
	switch {
	case statusCode == http.StatusTooManyRequests:
		return NewRateLimitError()
	case statusCode == http.StatusUnauthorized:
		return NewUnauthorizedError("")
	case statusCode == http.StatusNotFound:
		return NewNotFoundError("")
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
