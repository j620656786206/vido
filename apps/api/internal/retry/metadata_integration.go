package retry

import (
	"errors"
	"strings"
)

// Metadata error codes from provider.go
const (
	ErrCodeTimeout     = "METADATA_TIMEOUT"
	ErrCodeRateLimited = "METADATA_RATE_LIMITED"
	ErrCodeUnavailable = "METADATA_UNAVAILABLE"
	ErrCodeNoResults   = "METADATA_NO_RESULTS"
	ErrCodeCircuitOpen = "METADATA_CIRCUIT_OPEN"
)

// IsRetryableMetadataError checks if an error from the metadata package is retryable
// This function recognizes ProviderError types from the metadata package
func IsRetryableMetadataError(err error) bool {
	if err == nil {
		return false
	}

	// Check error string for known retryable patterns
	errStr := strings.ToLower(err.Error())

	// Retryable error codes from metadata package
	retryableCodes := []string{
		ErrCodeTimeout,
		ErrCodeRateLimited,
		ErrCodeUnavailable,
		ErrCodeCircuitOpen,
	}

	for _, code := range retryableCodes {
		if strings.Contains(errStr, strings.ToLower(code)) {
			return true
		}
	}

	// Check for common network-related patterns
	retryablePatterns := []string{
		"timeout",
		"rate limit",
		"rate_limit",
		"unavailable",
		"connection refused",
		"connection reset",
		"network error",
		"circuit open",
		"circuit breaker",
		"503",
		"502",
		"504",
		"429",
		"too many requests",
		"service unavailable",
		"bad gateway",
		"gateway timeout",
		"i/o timeout",
		"context deadline exceeded",
	}

	for _, pattern := range retryablePatterns {
		if strings.Contains(errStr, pattern) {
			return true
		}
	}

	// Non-retryable error patterns
	nonRetryablePatterns := []string{
		"not found",
		"no results",
		"invalid",
		"unauthorized",
		"forbidden",
		"400",
		"401",
		"403",
		"404",
	}

	for _, pattern := range nonRetryablePatterns {
		if strings.Contains(errStr, pattern) {
			return false
		}
	}

	// If we can't determine, default to not retryable for safety
	return false
}

// ClassifyMetadataError classifies a metadata error and returns the appropriate RetryableError
func ClassifyMetadataError(err error) *RetryableError {
	if err == nil {
		return nil
	}

	errStr := strings.ToLower(err.Error())

	// Check for timeout errors
	if strings.Contains(errStr, "timeout") ||
		strings.Contains(errStr, "i/o timeout") ||
		strings.Contains(errStr, ErrCodeTimeout) ||
		strings.Contains(errStr, "context deadline exceeded") {
		return &RetryableError{
			Code:      "METADATA_TIMEOUT",
			Message:   err.Error(),
			Retryable: true,
		}
	}

	// Check for rate limit errors
	if strings.Contains(errStr, "rate limit") ||
		strings.Contains(errStr, "rate_limit") ||
		strings.Contains(errStr, ErrCodeRateLimited) ||
		strings.Contains(errStr, "429") ||
		strings.Contains(errStr, "too many requests") {
		return &RetryableError{
			Code:       "METADATA_RATE_LIMITED",
			Message:    err.Error(),
			Retryable:  true,
			StatusCode: 429,
		}
	}

	// Check for service unavailable errors
	if strings.Contains(errStr, "unavailable") ||
		strings.Contains(errStr, ErrCodeUnavailable) ||
		strings.Contains(errStr, "503") ||
		strings.Contains(errStr, "service unavailable") {
		return &RetryableError{
			Code:       "METADATA_UNAVAILABLE",
			Message:    err.Error(),
			Retryable:  true,
			StatusCode: 503,
		}
	}

	// Check for circuit breaker errors
	if strings.Contains(errStr, "circuit") ||
		strings.Contains(errStr, ErrCodeCircuitOpen) {
		return &RetryableError{
			Code:      "METADATA_CIRCUIT_OPEN",
			Message:   err.Error(),
			Retryable: true,
		}
	}

	// Check for bad gateway / gateway timeout
	if strings.Contains(errStr, "502") ||
		strings.Contains(errStr, "504") ||
		strings.Contains(errStr, "bad gateway") ||
		strings.Contains(errStr, "gateway timeout") {
		return &RetryableError{
			Code:      "METADATA_GATEWAY_ERROR",
			Message:   err.Error(),
			Retryable: true,
		}
	}

	// Check for network errors
	if strings.Contains(errStr, "connection") ||
		strings.Contains(errStr, "network") {
		return &RetryableError{
			Code:      "METADATA_NETWORK_ERROR",
			Message:   err.Error(),
			Retryable: true,
		}
	}

	// Check for no results (not retryable)
	if strings.Contains(errStr, "no results") ||
		strings.Contains(errStr, ErrCodeNoResults) {
		return &RetryableError{
			Code:      "METADATA_NO_RESULTS",
			Message:   err.Error(),
			Retryable: false,
		}
	}

	// Check for not found (not retryable)
	if strings.Contains(errStr, "not found") ||
		strings.Contains(errStr, "404") {
		return &RetryableError{
			Code:       "METADATA_NOT_FOUND",
			Message:    err.Error(),
			Retryable:  false,
			StatusCode: 404,
		}
	}

	// Default: unknown error, not retryable
	return &RetryableError{
		Code:      "METADATA_UNKNOWN_ERROR",
		Message:   err.Error(),
		Retryable: false,
	}
}

// ExtractRetryableErrors examines a list of source attempt errors and returns retryable ones
// This is useful for processing FallbackStatus from the orchestrator
func ExtractRetryableErrors(attemptErrors []error) []error {
	var retryable []error
	for _, err := range attemptErrors {
		if err != nil && IsRetryableMetadataError(err) {
			retryable = append(retryable, err)
		}
	}
	return retryable
}

// ShouldQueueRetry determines if a metadata search failure should be queued for retry
// It returns true only if there are retryable errors and not all errors are non-retryable
func ShouldQueueRetry(attemptErrors []error) bool {
	if len(attemptErrors) == 0 {
		return false
	}

	hasRetryable := false
	allNoResults := true

	for _, err := range attemptErrors {
		if err == nil {
			continue
		}

		if IsRetryableMetadataError(err) {
			hasRetryable = true
		}

		// Check if this is a "no results" error
		errStr := strings.ToLower(err.Error())
		if !strings.Contains(errStr, "no results") && !strings.Contains(errStr, ErrCodeNoResults) {
			allNoResults = false
		}
	}

	// Don't retry if all errors are just "no results" (nothing to retry)
	if allNoResults {
		return false
	}

	return hasRetryable
}

// WrapAsRetryable wraps an error as a RetryableError based on classification
func WrapAsRetryable(err error) error {
	if err == nil {
		return nil
	}

	classified := ClassifyMetadataError(err)
	if classified != nil {
		return classified
	}

	return err
}

// IsTemporaryError checks if an error is likely temporary (network issues, timeouts, etc.)
func IsTemporaryError(err error) bool {
	if err == nil {
		return false
	}

	// Check if it implements interface{ Temporary() bool }
	type temporary interface {
		Temporary() bool
	}

	if t, ok := err.(temporary); ok {
		return t.Temporary()
	}

	// Check if wrapped error has Temporary()
	var tempErr temporary
	if errors.As(err, &tempErr) {
		return tempErr.Temporary()
	}

	// Fall back to string matching for common temporary errors
	return IsRetryableMetadataError(err)
}
