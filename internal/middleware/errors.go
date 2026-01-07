package middleware

import (
	"fmt"
	"net/http"
	"runtime/debug"

	"github.com/alexyu/vido/internal/config"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

// ErrorResponse represents the standard error response format
type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

// ErrorDetail contains the error code and message
type ErrorDetail struct {
	Code    string `json:"code" example:"VALIDATION_ERROR"`
	Message string `json:"message" example:"Invalid email format"`
}

// AppError represents a custom application error
type AppError struct {
	Code       string
	Message    string
	StatusCode int
	Err        error
}

// Error implements the error interface
func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

// Unwrap returns the wrapped error
func (e *AppError) Unwrap() error {
	return e.Err
}

// Common error constructors
func NewValidationError(message string) *AppError {
	return &AppError{
		Code:       "VALIDATION_ERROR",
		Message:    message,
		StatusCode: http.StatusBadRequest,
	}
}

func NewNotFoundError(message string) *AppError {
	return &AppError{
		Code:       "NOT_FOUND",
		Message:    message,
		StatusCode: http.StatusNotFound,
	}
}

func NewInternalError(message string, err error) *AppError {
	return &AppError{
		Code:       "INTERNAL_ERROR",
		Message:    message,
		StatusCode: http.StatusInternalServerError,
		Err:        err,
	}
}

func NewUnauthorizedError(message string) *AppError {
	return &AppError{
		Code:       "UNAUTHORIZED",
		Message:    message,
		StatusCode: http.StatusUnauthorized,
	}
}

func NewForbiddenError(message string) *AppError {
	return &AppError{
		Code:       "FORBIDDEN",
		Message:    message,
		StatusCode: http.StatusForbidden,
	}
}

// ErrorHandler returns a middleware that handles errors and formats them consistently
func ErrorHandler(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Process request
		c.Next()

		// Check if there are any errors
		if len(c.Errors) > 0 {
			// Get the last error
			err := c.Errors.Last()

			// Check if it's an AppError
			if appErr, ok := err.Err.(*AppError); ok {
				handleAppError(c, appErr, cfg)
			} else {
				// Handle generic error
				handleGenericError(c, err.Err, cfg)
			}
		}
	}
}

// handleAppError handles custom application errors
func handleAppError(c *gin.Context, appErr *AppError, cfg *config.Config) {
	// Log the error
	logEvent := log.Error().
		Str("request_id", getRequestID(c)).
		Str("error_code", appErr.Code).
		Str("path", c.Request.URL.Path).
		Str("method", c.Request.Method)

	if appErr.Err != nil {
		logEvent = logEvent.Err(appErr.Err)
	}

	// Include stack trace in development mode
	if cfg.IsDevelopment() {
		logEvent = logEvent.Str("stack_trace", string(debug.Stack()))
	}

	logEvent.Msg(appErr.Message)

	// Send error response
	c.JSON(appErr.StatusCode, ErrorResponse{
		Error: ErrorDetail{
			Code:    appErr.Code,
			Message: appErr.Message,
		},
	})
}

// handleGenericError handles generic errors
func handleGenericError(c *gin.Context, err error, cfg *config.Config) {
	// Log the error
	logEvent := log.Error().
		Str("request_id", getRequestID(c)).
		Str("path", c.Request.URL.Path).
		Str("method", c.Request.Method).
		Err(err)

	// Include stack trace in development mode
	if cfg.IsDevelopment() {
		logEvent = logEvent.Str("stack_trace", string(debug.Stack()))
	}

	logEvent.Msg("Internal server error")

	// Send generic error response
	c.JSON(http.StatusInternalServerError, ErrorResponse{
		Error: ErrorDetail{
			Code:    "INTERNAL_ERROR",
			Message: "An internal server error occurred",
		},
	})
}

// Recovery returns a middleware that recovers from panics and logs them
func Recovery(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				// Log the panic
				logEvent := log.Error().
					Str("request_id", getRequestID(c)).
					Str("path", c.Request.URL.Path).
					Str("method", c.Request.Method).
					Interface("panic", r).
					Str("stack_trace", string(debug.Stack()))

				logEvent.Msg("Panic recovered")

				// Send error response
				c.AbortWithStatusJSON(http.StatusInternalServerError, ErrorResponse{
					Error: ErrorDetail{
						Code:    "INTERNAL_ERROR",
						Message: "An internal server error occurred",
					},
				})
			}
		}()

		c.Next()
	}
}

// getRequestID retrieves the request ID from the context
func getRequestID(c *gin.Context) string {
	if requestID, exists := c.Get("request_id"); exists {
		if id, ok := requestID.(string); ok {
			return id
		}
	}
	return ""
}
