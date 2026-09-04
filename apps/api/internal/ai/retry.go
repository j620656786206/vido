package ai

import (
	"context"
	"fmt"
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

// RetryNotice is what a RetryObserver sees each time an attempt fails
// transiently and another attempt WILL follow: the attempt that just failed
// (1-based), the attempt budget, the error, and the backoff about to be slept.
// Nothing is emitted for a permanent failure or for the final attempt — those
// surface as the returned error.
type RetryNotice struct {
	Op          string
	Attempt     int
	MaxAttempts int
	Err         error
	Delay       time.Duration
}

// RetryObserver is a per-call hook a caller attaches to the ctx (sub-6-2
// AC #4) so it can narrate a retry — "第 N 批逾時，重試 2/3" on the
// subtitle progress stream — without this package knowing what a chunk is.
// It runs synchronously on the retry path before the backoff sleep; keep it
// cheap and never block on the ctx it was attached to.
type RetryObserver func(RetryNotice)

type retryObserverKey struct{}

// WithRetryObserver returns a ctx that carries fn for every retryTransient
// call made under it. A nil fn leaves the ctx unchanged.
func WithRetryObserver(ctx context.Context, fn RetryObserver) context.Context {
	if fn == nil {
		return ctx
	}
	return context.WithValue(ctx, retryObserverKey{}, fn)
}

// RetryObserverFrom returns the observer attached by WithRetryObserver, or nil.
// Exported so a fake provider in another package's tests can narrate a retry
// exactly the way the real one does.
func RetryObserverFrom(ctx context.Context) RetryObserver {
	fn, _ := ctx.Value(retryObserverKey{}).(RetryObserver)
	return fn
}

// retryTransient runs fn up to retryMaxAttempts times with bounded exponential
// backoff. fn reports whether its error is retryable; permanent errors return
// immediately. Context cancellation aborts the wait between attempts.
//
// When every attempt fails transiently the last error is returned wrapped
// with ErrAIRetriesExhausted (sub-6-2): the caller can then tell a flaky
// provider (worth degrading around) from a permanent rejection (worth
// stopping for) with one errors.Is, and the leading error code the log and
// API envelopes key on is still the last attempt's own sentinel. A ctx
// cancelled mid-window returns the bare last error — the caller's ctx, not
// the provider, ended the window.
func retryTransient[T any](ctx context.Context, label string, fn func() (T, bool, error)) (T, error) {
	var zero T
	var lastErr error
	delay := retryBaseDelay
	observer := RetryObserverFrom(ctx)

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
		if observer != nil {
			observer(RetryNotice{Op: label, Attempt: attempt, MaxAttempts: retryMaxAttempts, Err: err, Delay: delay})
		}
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
	return zero, fmt.Errorf("%w: %w after %d attempts", lastErr, ErrAIRetriesExhausted, retryMaxAttempts)
}
