package metadata

import (
	"errors"
	"strings"

	"github.com/vido/api/internal/retry"
)

// Pre-lowered ErrCode values used by the string-matching classifiers below.
// Hoisted to package scope so the classifiers do not re-allocate on every call
// (hot path: retry decisions during metadata fallback).
var (
	errCodeTimeoutLower     = strings.ToLower(ErrCodeTimeout)
	errCodeRateLimitedLower = strings.ToLower(ErrCodeRateLimited)
	errCodeUnavailableLower = strings.ToLower(ErrCodeUnavailable)
	errCodeCircuitOpenLower = strings.ToLower(ErrCodeCircuitOpen)
	errCodeNoResultsLower   = strings.ToLower(ErrCodeNoResults)

	retryableLoweredCodes = []string{
		errCodeTimeoutLower,
		errCodeRateLimitedLower,
		errCodeUnavailableLower,
		errCodeCircuitOpenLower,
	}
)

// IsRetryableMetadataError checks if an error from the metadata package is retryable.
// This function recognizes ProviderError types from the metadata package.
//
// Located in the metadata package (not retry) because the classification rules are
// defined by the metadata wire contract (ErrCode* constants owned here). retry
// remains a zero-internal-deps leaf per project-context.md Rule 19.
// (followup-metadata-prefix-dedup 2026-04-24 party-mode decision; supersedes
// Winston 2026-04-20 draft AC #5 — retry → metadata cycle blocked by
// metadata → tmdb → repository → retry existing path.)
func IsRetryableMetadataError(err error) bool {
	if err == nil {
		return false
	}

	// Check error string for known retryable patterns
	errStr := strings.ToLower(err.Error())

	for _, code := range retryableLoweredCodes {
		if strings.Contains(errStr, code) {
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

	// Default to not retryable for safety. A prior non-retryable-pattern pre-check
	// was removed as observationally dead code (it returned the same value as this
	// default); keeping the retryable checks above is sufficient.
	return false
}

// ClassifyMetadataError classifies a metadata error and returns the appropriate RetryableError
func ClassifyMetadataError(err error) *retry.RetryableError {
	if err == nil {
		return nil
	}

	errStr := strings.ToLower(err.Error())

	// Check for timeout errors
	if strings.Contains(errStr, "timeout") ||
		strings.Contains(errStr, "i/o timeout") ||
		strings.Contains(errStr, errCodeTimeoutLower) ||
		strings.Contains(errStr, "context deadline exceeded") {
		return &retry.RetryableError{
			Code:      ErrCodeTimeout,
			Message:   err.Error(),
			Retryable: true,
		}
	}

	// Check for rate limit errors
	if strings.Contains(errStr, "rate limit") ||
		strings.Contains(errStr, "rate_limit") ||
		strings.Contains(errStr, errCodeRateLimitedLower) ||
		strings.Contains(errStr, "429") ||
		strings.Contains(errStr, "too many requests") {
		return &retry.RetryableError{
			Code:       ErrCodeRateLimited,
			Message:    err.Error(),
			Retryable:  true,
			StatusCode: 429,
		}
	}

	// Check for service unavailable errors
	if strings.Contains(errStr, "unavailable") ||
		strings.Contains(errStr, errCodeUnavailableLower) ||
		strings.Contains(errStr, "503") ||
		strings.Contains(errStr, "service unavailable") {
		return &retry.RetryableError{
			Code:       ErrCodeUnavailable,
			Message:    err.Error(),
			Retryable:  true,
			StatusCode: 503,
		}
	}

	// Check for circuit breaker errors
	if strings.Contains(errStr, "circuit") ||
		strings.Contains(errStr, errCodeCircuitOpenLower) {
		return &retry.RetryableError{
			Code:      ErrCodeCircuitOpen,
			Message:   err.Error(),
			Retryable: true,
		}
	}

	// Check for bad gateway / gateway timeout
	if strings.Contains(errStr, "502") ||
		strings.Contains(errStr, "504") ||
		strings.Contains(errStr, "bad gateway") ||
		strings.Contains(errStr, "gateway timeout") {
		return &retry.RetryableError{
			Code:      ErrCodeGatewayError,
			Message:   err.Error(),
			Retryable: true,
		}
	}

	// Check for network errors
	if strings.Contains(errStr, "connection") ||
		strings.Contains(errStr, "network") {
		return &retry.RetryableError{
			Code:      ErrCodeNetworkError,
			Message:   err.Error(),
			Retryable: true,
		}
	}

	// Check for no results (not retryable)
	if strings.Contains(errStr, "no results") ||
		strings.Contains(errStr, errCodeNoResultsLower) {
		return &retry.RetryableError{
			Code:      ErrCodeNoResults,
			Message:   err.Error(),
			Retryable: false,
		}
	}

	// Check for not found (not retryable)
	if strings.Contains(errStr, "not found") ||
		strings.Contains(errStr, "404") {
		return &retry.RetryableError{
			Code:       ErrCodeNotFound,
			Message:    err.Error(),
			Retryable:  false,
			StatusCode: 404,
		}
	}

	// Default: unknown error, not retryable
	return &retry.RetryableError{
		Code:      ErrCodeUnknownError,
		Message:   err.Error(),
		Retryable: false,
	}
}

// ExtractRetryableErrors examines a list of source attempt errors and returns retryable ones.
// This is useful for processing FallbackStatus from the orchestrator.
func ExtractRetryableErrors(attemptErrors []error) []error {
	var retryable []error
	for _, err := range attemptErrors {
		if err != nil && IsRetryableMetadataError(err) {
			retryable = append(retryable, err)
		}
	}
	return retryable
}

// ShouldQueueRetry determines if a metadata search failure should be queued for retry.
// It returns true only if there are retryable errors and not all errors are non-retryable.
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
		if !strings.Contains(errStr, "no results") && !strings.Contains(errStr, errCodeNoResultsLower) {
			allNoResults = false
		}
	}

	// Don't retry if all errors are just "no results" (nothing to retry)
	if allNoResults {
		return false
	}

	return hasRetryable
}

// WrapAsRetryable wraps an error as a RetryableError based on classification.
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

// IsTemporaryError checks if an error is likely temporary (network issues, timeouts, etc.).
func IsTemporaryError(err error) bool {
	if err == nil {
		return false
	}

	// errors.As checks the direct error first before unwrapping, so it subsumes
	// a direct type assertion for this interface.
	type temporary interface {
		Temporary() bool
	}

	var tempErr temporary
	if errors.As(err, &tempErr) {
		return tempErr.Temporary()
	}

	// Fall back to string matching for common temporary errors
	return IsRetryableMetadataError(err)
}
