package retry

import (
	"math"
	"math/rand"
	"time"
)

// BackoffCalculator calculates retry delays with exponential backoff
type BackoffCalculator struct {
	BaseDelay  time.Duration // Base delay for first retry (1 second)
	MaxDelay   time.Duration // Maximum delay cap (8 seconds)
	Multiplier float64       // Delay multiplier between retries (2.0)
	JitterMax  float64       // Maximum jitter percentage (0.1 = 10%)
}

// NewBackoffCalculator creates a new BackoffCalculator with default values
// following NFR-R5: 1s → 2s → 4s → 8s pattern
func NewBackoffCalculator() *BackoffCalculator {
	return &BackoffCalculator{
		BaseDelay:  1 * time.Second,
		MaxDelay:   8 * time.Second,
		Multiplier: 2.0,
		JitterMax:  0.1, // 10% jitter to prevent thundering herd
	}
}

// Calculate returns the delay for the given attempt number
// Attempt 0: 1s (base)
// Attempt 1: 2s (1s * 2^1)
// Attempt 2: 4s (1s * 2^2)
// Attempt 3: 8s (1s * 2^3) - max
func (b *BackoffCalculator) Calculate(attempt int) time.Duration {
	if attempt < 0 {
		attempt = 0
	}

	// Calculate exponential delay: base * multiplier^attempt
	delay := float64(b.BaseDelay) * math.Pow(b.Multiplier, float64(attempt))

	// Cap at max delay
	if delay > float64(b.MaxDelay) {
		delay = float64(b.MaxDelay)
	}

	// Add jitter: ±JitterMax%
	// Jitter helps prevent thundering herd when many retries are scheduled
	if b.JitterMax > 0 {
		jitter := delay * b.JitterMax * (2*rand.Float64() - 1)
		delay += jitter
	}

	// Ensure we never return a negative delay
	if delay < 0 {
		delay = float64(b.BaseDelay)
	}

	return time.Duration(delay)
}

// CalculateWithoutJitter returns the delay without jitter (for testing)
func (b *BackoffCalculator) CalculateWithoutJitter(attempt int) time.Duration {
	if attempt < 0 {
		attempt = 0
	}

	delay := float64(b.BaseDelay) * math.Pow(b.Multiplier, float64(attempt))

	if delay > float64(b.MaxDelay) {
		delay = float64(b.MaxDelay)
	}

	return time.Duration(delay)
}

// NextAttemptTime calculates the time for the next retry attempt
func (b *BackoffCalculator) NextAttemptTime(attempt int) time.Time {
	return time.Now().Add(b.Calculate(attempt))
}

// NextAttemptTimeWithoutJitter calculates the time without jitter (for testing)
func (b *BackoffCalculator) NextAttemptTimeWithoutJitter(attempt int) time.Time {
	return time.Now().Add(b.CalculateWithoutJitter(attempt))
}

// WithBaseDelay sets the base delay and returns the calculator for chaining
func (b *BackoffCalculator) WithBaseDelay(d time.Duration) *BackoffCalculator {
	b.BaseDelay = d
	return b
}

// WithMaxDelay sets the max delay and returns the calculator for chaining
func (b *BackoffCalculator) WithMaxDelay(d time.Duration) *BackoffCalculator {
	b.MaxDelay = d
	return b
}

// WithMultiplier sets the multiplier and returns the calculator for chaining
func (b *BackoffCalculator) WithMultiplier(m float64) *BackoffCalculator {
	b.Multiplier = m
	return b
}

// WithJitter sets the jitter percentage and returns the calculator for chaining
func (b *BackoffCalculator) WithJitter(j float64) *BackoffCalculator {
	b.JitterMax = j
	return b
}

// NoJitter disables jitter and returns the calculator for chaining
func (b *BackoffCalculator) NoJitter() *BackoffCalculator {
	b.JitterMax = 0
	return b
}
