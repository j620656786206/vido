package ai

import (
	"context"
	"log/slog"

	"golang.org/x/time/rate"
)

// Governor throttles outbound AI calls (ASR + LLM) with a shared concurrency
// cap and a token-bucket rate limiter (Story 9R-11). Batch translate/transcribe
// over a whole library would otherwise fan out unbounded requests and trip the
// upstream rate limit. One Governor is created at startup and shared by the
// Claude and Whisper clients so ASR and LLM draw from the SAME budget of
// in-flight slots and requests-per-second.
type Governor struct {
	sem     chan struct{}
	limiter *rate.Limiter
}

// NewGovernor builds a Governor. maxConcurrent caps simultaneous in-flight
// calls (<=0 disables the concurrency cap); ratePerSec + burst configure the
// token-bucket QPS (ratePerSec <=0 disables rate limiting). Reuse the returned
// value — do not construct per request (Rule 14 / Rule 27 ①).
func NewGovernor(maxConcurrent int, ratePerSec float64, burst int) *Governor {
	g := &Governor{}
	if maxConcurrent > 0 {
		g.sem = make(chan struct{}, maxConcurrent)
	}
	if ratePerSec > 0 {
		if burst < 1 {
			burst = 1
		}
		g.limiter = rate.NewLimiter(rate.Limit(ratePerSec), burst)
	}
	return g
}

// Acquire blocks until a rate token AND a concurrency slot are available, or the
// context is cancelled. The returned release func MUST be called (defer) to free
// the slot. A nil Governor is a no-op (release is a no-op) so callers need no
// nil checks.
func (g *Governor) Acquire(ctx context.Context) (release func(), err error) {
	if g == nil {
		return func() {}, nil
	}
	// Rate token first (Rule 27 ①: Wait before the slot so a throttled call
	// doesn't hold a concurrency slot while it waits).
	if g.limiter != nil {
		if err := g.limiter.Wait(ctx); err != nil {
			return func() {}, err
		}
	}
	if g.sem != nil {
		select {
		case g.sem <- struct{}{}:
			return func() { <-g.sem }, nil
		case <-ctx.Done():
			return func() {}, ctx.Err()
		}
	}
	return func() {}, nil
}

// governed runs fn under an acquired slot, releasing it afterward. A budget
// pre-check (ctx) short-circuits before acquiring so an exhausted run stops
// spending immediately.
func governed[T any](ctx context.Context, g *Governor, op string, fn func() (T, error)) (T, error) {
	var zero T
	if b := BudgetFromContext(ctx); b != nil && b.Exceeded() {
		slog.Warn("AI budget exhausted — skipping call", "op", op, "spent_usd", b.SpentUSD())
		return zero, ErrBudgetExceeded
	}
	release, err := g.Acquire(ctx)
	if err != nil {
		return zero, err
	}
	defer release()
	return fn()
}
