package retry

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewBackoffCalculator(t *testing.T) {
	calc := NewBackoffCalculator()

	assert.Equal(t, 1*time.Second, calc.BaseDelay)
	assert.Equal(t, 8*time.Second, calc.MaxDelay)
	assert.Equal(t, 2.0, calc.Multiplier)
	assert.Equal(t, 0.1, calc.JitterMax)
}

func TestBackoffCalculator_CalculateWithoutJitter(t *testing.T) {
	calc := NewBackoffCalculator().NoJitter()

	tests := []struct {
		name     string
		attempt  int
		expected time.Duration
	}{
		{
			name:     "attempt 0 - base delay",
			attempt:  0,
			expected: 1 * time.Second,
		},
		{
			name:     "attempt 1 - 2x base",
			attempt:  1,
			expected: 2 * time.Second,
		},
		{
			name:     "attempt 2 - 4x base",
			attempt:  2,
			expected: 4 * time.Second,
		},
		{
			name:     "attempt 3 - 8x base (max)",
			attempt:  3,
			expected: 8 * time.Second,
		},
		{
			name:     "attempt 4 - capped at max",
			attempt:  4,
			expected: 8 * time.Second,
		},
		{
			name:     "attempt 10 - still capped at max",
			attempt:  10,
			expected: 8 * time.Second,
		},
		{
			name:     "negative attempt - treated as 0",
			attempt:  -1,
			expected: 1 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calc.CalculateWithoutJitter(tt.attempt)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBackoffCalculator_CalculateWithJitter(t *testing.T) {
	calc := NewBackoffCalculator()

	// Run multiple times to verify jitter is within bounds
	for i := 0; i < 100; i++ {
		for attempt := 0; attempt <= 3; attempt++ {
			delay := calc.Calculate(attempt)
			baseDelay := calc.CalculateWithoutJitter(attempt)

			// Delay should be within ±10% of base delay
			minDelay := time.Duration(float64(baseDelay) * 0.9)
			maxDelay := time.Duration(float64(baseDelay) * 1.1)

			assert.GreaterOrEqual(t, delay, minDelay,
				"attempt %d: delay %v should be >= %v", attempt, delay, minDelay)
			assert.LessOrEqual(t, delay, maxDelay,
				"attempt %d: delay %v should be <= %v", attempt, delay, maxDelay)
		}
	}
}

func TestBackoffCalculator_ExponentialPattern(t *testing.T) {
	calc := NewBackoffCalculator().NoJitter()

	// Verify the exact 1s → 2s → 4s → 8s pattern from NFR-R5
	expectedPattern := []time.Duration{
		1 * time.Second,
		2 * time.Second,
		4 * time.Second,
		8 * time.Second,
	}

	for attempt, expected := range expectedPattern {
		result := calc.Calculate(attempt)
		assert.Equal(t, expected, result,
			"attempt %d: expected %v, got %v", attempt, expected, result)
	}
}

func TestBackoffCalculator_CustomConfig(t *testing.T) {
	// Test with custom configuration
	calc := NewBackoffCalculator().
		WithBaseDelay(500 * time.Millisecond).
		WithMaxDelay(4 * time.Second).
		WithMultiplier(3.0).
		NoJitter()

	tests := []struct {
		attempt  int
		expected time.Duration
	}{
		{0, 500 * time.Millisecond}, // 500ms * 3^0 = 500ms
		{1, 1500 * time.Millisecond}, // 500ms * 3^1 = 1.5s
		{2, 4 * time.Second},         // 500ms * 3^2 = 4.5s, capped at 4s
		{3, 4 * time.Second},         // Capped at max
	}

	for _, tt := range tests {
		result := calc.CalculateWithoutJitter(tt.attempt)
		assert.Equal(t, tt.expected, result)
	}
}

func TestBackoffCalculator_NextAttemptTime(t *testing.T) {
	calc := NewBackoffCalculator().NoJitter()

	before := time.Now()
	nextTime := calc.NextAttemptTimeWithoutJitter(0)
	after := time.Now()

	// Next attempt time should be approximately 1 second from now
	expectedMin := before.Add(1 * time.Second)
	expectedMax := after.Add(1 * time.Second)

	assert.True(t, nextTime.After(expectedMin) || nextTime.Equal(expectedMin),
		"next time should be at or after expected min")
	assert.True(t, nextTime.Before(expectedMax) || nextTime.Equal(expectedMax),
		"next time should be at or before expected max")
}

func TestBackoffCalculator_ChainedConfig(t *testing.T) {
	calc := NewBackoffCalculator().
		WithBaseDelay(2 * time.Second).
		WithMaxDelay(16 * time.Second).
		WithMultiplier(2.5).
		WithJitter(0.2)

	assert.Equal(t, 2*time.Second, calc.BaseDelay)
	assert.Equal(t, 16*time.Second, calc.MaxDelay)
	assert.Equal(t, 2.5, calc.Multiplier)
	assert.Equal(t, 0.2, calc.JitterMax)
}

func TestBackoffCalculator_ZeroJitter(t *testing.T) {
	calc := NewBackoffCalculator()

	// Enable jitter first
	calc.WithJitter(0.1)
	assert.Equal(t, 0.1, calc.JitterMax)

	// Then disable it
	calc.NoJitter()
	assert.Equal(t, 0.0, calc.JitterMax)

	// Verify no jitter is applied
	for i := 0; i < 10; i++ {
		result := calc.Calculate(0)
		assert.Equal(t, 1*time.Second, result)
	}
}

func TestBackoffCalculator_JitterDistribution(t *testing.T) {
	calc := NewBackoffCalculator()

	// Collect samples to verify jitter creates variance
	samples := make(map[time.Duration]int)
	for i := 0; i < 1000; i++ {
		delay := calc.Calculate(0)
		samples[delay]++
	}

	// With jitter, we should have multiple different values
	assert.Greater(t, len(samples), 1,
		"jitter should create variance in delays")
}

func TestBackoffCalculator_MaxDelayEnforced(t *testing.T) {
	calc := NewBackoffCalculator().NoJitter()

	// Very high attempt numbers should still be capped
	for attempt := 100; attempt <= 105; attempt++ {
		result := calc.Calculate(attempt)
		assert.Equal(t, 8*time.Second, result,
			"high attempt %d should be capped at max delay", attempt)
	}
}

func TestBackoffCalculator_Calculate_NeverNegative(t *testing.T) {
	calc := NewBackoffCalculator()

	// Even with extreme jitter, delay should never be negative
	calc.JitterMax = 1.0 // 100% jitter (extreme case)

	for i := 0; i < 1000; i++ {
		delay := calc.Calculate(0)
		assert.GreaterOrEqual(t, delay, time.Duration(0),
			"delay should never be negative")
	}
}
