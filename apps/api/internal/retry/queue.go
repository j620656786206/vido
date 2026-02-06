package retry

import (
	"encoding/json"
	"fmt"
	"time"
)

// RetryItem represents a task queued for retry
type RetryItem struct {
	ID            string          `json:"id" db:"id"`
	TaskID        string          `json:"taskId" db:"task_id"`
	TaskType      string          `json:"taskType" db:"task_type"` // "parse", "metadata_fetch"
	Payload       json.RawMessage `json:"payload" db:"payload"`    // Task-specific data
	AttemptCount  int             `json:"attemptCount" db:"attempt_count"`
	MaxAttempts   int             `json:"maxAttempts" db:"max_attempts"`
	LastError     string          `json:"lastError,omitempty" db:"last_error"`
	NextAttemptAt time.Time       `json:"nextAttemptAt" db:"next_attempt_at"`
	CreatedAt     time.Time       `json:"createdAt" db:"created_at"`
	UpdatedAt     time.Time       `json:"updatedAt" db:"updated_at"`
}

// RetryableError represents an error that can be retried
type RetryableError struct {
	Code       string `json:"code"`
	Message    string `json:"message"`
	Retryable  bool   `json:"retryable"`
	StatusCode int    `json:"statusCode,omitempty"`
}

// Error implements the error interface
func (e *RetryableError) Error() string {
	return e.Message
}

// IsRetryable returns whether this error can be retried
func (e *RetryableError) IsRetryable() bool {
	return e.Retryable
}

// Common retryable error types
var (
	ErrTimeout = &RetryableError{
		Code:      "TIMEOUT",
		Message:   "Request timed out",
		Retryable: true,
	}
	ErrRateLimit = &RetryableError{
		Code:       "RATE_LIMIT",
		Message:    "Rate limit exceeded",
		Retryable:  true,
		StatusCode: 429,
	}
	ErrNetworkError = &RetryableError{
		Code:      "NETWORK_ERROR",
		Message:   "Network connection failed",
		Retryable: true,
	}
	ErrServiceDown = &RetryableError{
		Code:       "SERVICE_DOWN",
		Message:    "Service temporarily unavailable",
		Retryable:  true,
		StatusCode: 503,
	}
	ErrBadGateway = &RetryableError{
		Code:       "BAD_GATEWAY",
		Message:    "Bad gateway",
		Retryable:  true,
		StatusCode: 502,
	}
	ErrGatewayTimeout = &RetryableError{
		Code:       "GATEWAY_TIMEOUT",
		Message:    "Gateway timeout",
		Retryable:  true,
		StatusCode: 504,
	}

	// Non-retryable errors
	ErrNotFound = &RetryableError{
		Code:       "NOT_FOUND",
		Message:    "Resource not found",
		Retryable:  false,
		StatusCode: 404,
	}
	ErrInvalidInput = &RetryableError{
		Code:      "INVALID_INPUT",
		Message:   "Invalid input provided",
		Retryable: false,
	}
	ErrUnauthorized = &RetryableError{
		Code:       "UNAUTHORIZED",
		Message:    "Authentication required",
		Retryable:  false,
		StatusCode: 401,
	}
	ErrForbidden = &RetryableError{
		Code:       "FORBIDDEN",
		Message:    "Access denied",
		Retryable:  false,
		StatusCode: 403,
	}
)

// NewRetryableError creates a new retryable error with the given parameters
func NewRetryableError(code, message string, retryable bool, statusCode int) *RetryableError {
	return &RetryableError{
		Code:       code,
		Message:    message,
		Retryable:  retryable,
		StatusCode: statusCode,
	}
}

// WrapRetryable wraps an error as a retryable error
func WrapRetryable(err error, code string) *RetryableError {
	return &RetryableError{
		Code:      code,
		Message:   err.Error(),
		Retryable: true,
	}
}

// WrapNonRetryable wraps an error as a non-retryable error
func WrapNonRetryable(err error, code string) *RetryableError {
	return &RetryableError{
		Code:      code,
		Message:   err.Error(),
		Retryable: false,
	}
}

// TaskTypes for retry queue
const (
	TaskTypeParse         = "parse"
	TaskTypeMetadataFetch = "metadata_fetch"
)

// MaxRetryAttempts is the default maximum number of retry attempts
const MaxRetryAttempts = 4

// RetryStats contains statistics about the retry queue
type RetryStats struct {
	TotalPending   int `json:"totalPending"`
	TotalSucceeded int `json:"totalSucceeded"`
	TotalFailed    int `json:"totalFailed"`
}

// RetryResponse contains pending retries and stats for API response
type RetryResponse struct {
	Items []*RetryItemResponse `json:"items"`
	Stats *RetryStats          `json:"stats"`
}

// RetryItemResponse is the API response format for a retry item
type RetryItemResponse struct {
	ID             string    `json:"id"`
	TaskID         string    `json:"taskId"`
	TaskType       string    `json:"taskType"`
	AttemptCount   int       `json:"attemptCount"`
	MaxAttempts    int       `json:"maxAttempts"`
	LastError      string    `json:"lastError,omitempty"`
	NextAttemptAt  time.Time `json:"nextAttemptAt"`
	TimeUntilRetry string    `json:"timeUntilRetry"`
}

// ToResponse converts a RetryItem to RetryItemResponse
func (r *RetryItem) ToResponse() *RetryItemResponse {
	timeUntil := time.Until(r.NextAttemptAt)
	timeUntilStr := "即將重試"
	if timeUntil > 0 {
		seconds := int(timeUntil.Seconds())
		if seconds > 0 {
			timeUntilStr = formatDuration(timeUntil)
		}
	}

	return &RetryItemResponse{
		ID:             r.ID,
		TaskID:         r.TaskID,
		TaskType:       r.TaskType,
		AttemptCount:   r.AttemptCount,
		MaxAttempts:    r.MaxAttempts,
		LastError:      r.LastError,
		NextAttemptAt:  r.NextAttemptAt,
		TimeUntilRetry: timeUntilStr,
	}
}

// formatDuration formats a duration as a human-readable string
func formatDuration(d time.Duration) string {
	if d < time.Second {
		return "即將重試"
	}

	seconds := int(d.Seconds())
	if seconds < 60 {
		return fmt.Sprintf("%ds", seconds)
	}

	minutes := seconds / 60
	remainingSeconds := seconds % 60
	if remainingSeconds > 0 {
		return fmt.Sprintf("%dm%ds", minutes, remainingSeconds)
	}
	return fmt.Sprintf("%dm", minutes)
}
