package ai

import (
	"context"
	"log/slog"
	"net/http"
	"time"
)

// 9R-4: bounded exponential backoff for transient ASR/LLM failures. One
// transient timeout previously killed a full transcription run (POC evidence);
// permanent errors (4xx other than 429) must NOT retry.
// retryMaxAttempts is the total number of attempts (1 initial + retries).
const retryMaxAttempts = 3

// retryBaseDelay is the first backoff delay; doubles per attempt
// (project-context §5: 1s → 2s → 4s → 8s), capped at retryMaxDelay.
// Vars (not consts) so the package test suite can shrink them.
var (
	retryBaseDelay = 1 * time.Second
	retryMaxDelay  = 8 * time.Second
)

// isTransientStatus reports whether an HTTP status is worth retrying:
// 429 (rate limit) and any 5xx. Other 4xx are permanent.
func isTransientStatus(code int) bool {
	return code == http.StatusTooManyRequests || code >= 500
}

// retryTransient runs fn up to retryMaxAttempts times with bounded exponential
// backoff. fn reports whether its error is retryable; permanent errors return
// immediately. Context cancellation aborts the wait between attempts.
func retryTransient[T any](ctx context.Context, label string, fn func() (T, bool, error)) (T, error) {
	var zero T
	var lastErr error
	delay := retryBaseDelay

	for attempt := 1; attempt <= retryMaxAttempts; attempt++ {
		result, retryable, err := fn()
		if err == nil {
			return result, nil
		}
		lastErr = err
		if !retryable || ctx.Err() != nil {
			return zero, err
		}
		if attempt == retryMaxAttempts {
			break
		}

		slog.Warn("transient AI failure — retrying with backoff",
			"op", label,
			"attempt", attempt,
			"max_attempts", retryMaxAttempts,
			"delay", delay.String(),
			"error", err,
		)
		select {
		case <-ctx.Done():
			return zero, lastErr
		case <-time.After(delay):
		}
		delay *= 2
		if delay > retryMaxDelay {
			delay = retryMaxDelay
		}
	}

	slog.Error("AI call failed after all retries",
		"op", label,
		"attempts", retryMaxAttempts,
		"error", lastErr,
	)
	return zero, lastErr
}
