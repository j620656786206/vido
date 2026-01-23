package metadata

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCircuitState_String(t *testing.T) {
	tests := []struct {
		state    CircuitState
		expected string
	}{
		{CircuitStateClosed, "closed"},
		{CircuitStateOpen, "open"},
		{CircuitStateHalfOpen, "half-open"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.state.String())
		})
	}
}

func TestNewCircuitBreaker_DefaultConfig(t *testing.T) {
	cb := NewCircuitBreaker("test", CircuitBreakerConfig{})

	assert.Equal(t, "test", cb.Name())
	assert.Equal(t, CircuitStateClosed, cb.State())

	// Verify default thresholds
	stats := cb.Stats()
	assert.Equal(t, 5, stats.FailureThreshold)
	assert.Equal(t, 2, stats.SuccessThreshold)
	assert.Equal(t, 30*time.Second, stats.Timeout)
}

func TestNewCircuitBreaker_CustomConfig(t *testing.T) {
	cfg := CircuitBreakerConfig{
		FailureThreshold:  10,
		SuccessThreshold:  3,
		Timeout:           time.Minute,
		HalfOpenMaxCalls:  2,
		OnStateChange:     func(name string, from, to CircuitState) {},
	}

	cb := NewCircuitBreaker("custom", cfg)

	stats := cb.Stats()
	assert.Equal(t, 10, stats.FailureThreshold)
	assert.Equal(t, 3, stats.SuccessThreshold)
	assert.Equal(t, time.Minute, stats.Timeout)
}

func TestCircuitBreaker_ClosedState_Success(t *testing.T) {
	cb := NewCircuitBreaker("test", CircuitBreakerConfig{})

	// Successful execution should keep circuit closed
	err := cb.Execute(func() error {
		return nil
	})

	assert.NoError(t, err)
	assert.Equal(t, CircuitStateClosed, cb.State())

	stats := cb.Stats()
	assert.Equal(t, 1, stats.TotalCalls)
	assert.Equal(t, 1, stats.SuccessCount)
	assert.Equal(t, 0, stats.FailureCount)
}

func TestCircuitBreaker_ClosedState_FailuresBelowThreshold(t *testing.T) {
	cfg := CircuitBreakerConfig{
		FailureThreshold: 5,
	}
	cb := NewCircuitBreaker("test", cfg)

	testErr := errors.New("test error")

	// Execute 4 failures (below threshold of 5)
	for i := 0; i < 4; i++ {
		err := cb.Execute(func() error {
			return testErr
		})
		assert.Error(t, err)
		assert.Equal(t, CircuitStateClosed, cb.State(), "circuit should remain closed")
	}

	stats := cb.Stats()
	assert.Equal(t, 4, stats.FailureCount)
	assert.Equal(t, 4, stats.ConsecutiveFailures)
}

func TestCircuitBreaker_OpensAfterThreshold(t *testing.T) {
	stateChanged := make(chan CircuitState, 10)
	cfg := CircuitBreakerConfig{
		FailureThreshold: 5,
		OnStateChange: func(name string, from, to CircuitState) {
			stateChanged <- to
		},
	}
	cb := NewCircuitBreaker("test", cfg)

	testErr := errors.New("test error")

	// Execute 5 failures to reach threshold
	for i := 0; i < 5; i++ {
		cb.Execute(func() error {
			return testErr
		})
	}

	assert.Equal(t, CircuitStateOpen, cb.State())

	// Wait for state change callback
	select {
	case state := <-stateChanged:
		assert.Equal(t, CircuitStateOpen, state)
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for state change")
	}

	stats := cb.Stats()
	assert.Equal(t, 5, stats.ConsecutiveFailures)
}

func TestCircuitBreaker_OpenState_RejectsRequests(t *testing.T) {
	cfg := CircuitBreakerConfig{
		FailureThreshold: 1,
		Timeout:          time.Hour, // Long timeout to stay open
	}
	cb := NewCircuitBreaker("test", cfg)

	// Trip the circuit
	cb.Execute(func() error {
		return errors.New("error")
	})

	assert.Equal(t, CircuitStateOpen, cb.State())

	// Next request should be rejected without executing
	executed := false
	err := cb.Execute(func() error {
		executed = true
		return nil
	})

	assert.False(t, executed, "function should not be executed when circuit is open")
	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrCircuitOpen))
}

func TestCircuitBreaker_TransitionsToHalfOpen(t *testing.T) {
	cfg := CircuitBreakerConfig{
		FailureThreshold: 1,
		Timeout:          10 * time.Millisecond, // Very short timeout for testing
	}
	cb := NewCircuitBreaker("test", cfg)

	// Trip the circuit
	cb.Execute(func() error {
		return errors.New("error")
	})

	assert.Equal(t, CircuitStateOpen, cb.State())

	// Wait for timeout
	time.Sleep(20 * time.Millisecond)

	// Next call should transition to half-open and execute
	executed := false
	err := cb.Execute(func() error {
		executed = true
		return nil
	})

	assert.True(t, executed, "function should execute in half-open state")
	assert.NoError(t, err)
}

func TestCircuitBreaker_HalfOpen_SuccessCloses(t *testing.T) {
	stateChanged := make(chan CircuitState, 10)
	cfg := CircuitBreakerConfig{
		FailureThreshold: 1,
		SuccessThreshold: 2,
		Timeout:          10 * time.Millisecond,
		OnStateChange: func(name string, from, to CircuitState) {
			stateChanged <- to
		},
	}
	cb := NewCircuitBreaker("test", cfg)

	// Trip the circuit
	cb.Execute(func() error {
		return errors.New("error")
	})

	// Wait for open state
	select {
	case <-stateChanged:
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for open state")
	}

	// Wait for timeout
	time.Sleep(20 * time.Millisecond)

	// Execute successful calls to reach success threshold
	for i := 0; i < 2; i++ {
		cb.Execute(func() error {
			return nil
		})
	}

	assert.Equal(t, CircuitStateClosed, cb.State())

	// Drain the channel and check for closed state
	foundClosed := false
	for {
		select {
		case state := <-stateChanged:
			if state == CircuitStateClosed {
				foundClosed = true
			}
		case <-time.After(100 * time.Millisecond):
			goto done
		}
	}
done:
	assert.True(t, foundClosed, "should have transitioned to closed state")
}

func TestCircuitBreaker_HalfOpen_FailureReopens(t *testing.T) {
	stateChanged := make(chan CircuitState, 10)
	cfg := CircuitBreakerConfig{
		FailureThreshold: 1,
		Timeout:          10 * time.Millisecond,
		OnStateChange: func(name string, from, to CircuitState) {
			stateChanged <- to
		},
	}
	cb := NewCircuitBreaker("test", cfg)

	// Trip the circuit
	cb.Execute(func() error {
		return errors.New("error")
	})

	// Wait for open state
	select {
	case state := <-stateChanged:
		assert.Equal(t, CircuitStateOpen, state)
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for open state")
	}

	// Wait for timeout
	time.Sleep(20 * time.Millisecond)

	// Fail in half-open state
	cb.Execute(func() error {
		return errors.New("error")
	})

	assert.Equal(t, CircuitStateOpen, cb.State())

	// Collect remaining state changes
	stateChanges := []CircuitState{}
	for {
		select {
		case state := <-stateChanged:
			stateChanges = append(stateChanges, state)
		case <-time.After(100 * time.Millisecond):
			goto done
		}
	}
done:
	// Verify state transitions: half-open -> open
	assert.Contains(t, stateChanges, CircuitStateHalfOpen)
	assert.Contains(t, stateChanges, CircuitStateOpen)
}

func TestCircuitBreaker_Reset(t *testing.T) {
	cfg := CircuitBreakerConfig{
		FailureThreshold: 1,
	}
	cb := NewCircuitBreaker("test", cfg)

	// Trip the circuit
	cb.Execute(func() error {
		return errors.New("error")
	})

	assert.Equal(t, CircuitStateOpen, cb.State())

	// Reset
	cb.Reset()

	assert.Equal(t, CircuitStateClosed, cb.State())
	stats := cb.Stats()
	assert.Equal(t, 0, stats.ConsecutiveFailures)
	assert.Equal(t, 0, stats.SuccessCount)
	assert.Equal(t, 0, stats.FailureCount)
}

func TestCircuitBreaker_Concurrent(t *testing.T) {
	cfg := CircuitBreakerConfig{
		FailureThreshold: 100,
	}
	cb := NewCircuitBreaker("test", cfg)

	var wg sync.WaitGroup
	numGoroutines := 50
	numCalls := 100

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < numCalls; j++ {
				cb.Execute(func() error {
					if j%2 == 0 {
						return nil
					}
					return errors.New("error")
				})
			}
		}()
	}

	wg.Wait()

	stats := cb.Stats()
	assert.Equal(t, numGoroutines*numCalls, stats.TotalCalls)
}

func TestCircuitBreaker_HalfOpenMaxCalls(t *testing.T) {
	cfg := CircuitBreakerConfig{
		FailureThreshold: 1,
		SuccessThreshold: 3,
		Timeout:          10 * time.Millisecond,
		HalfOpenMaxCalls: 1, // Only allow 1 call in half-open
	}
	cb := NewCircuitBreaker("test", cfg)

	// Trip the circuit
	cb.Execute(func() error {
		return errors.New("error")
	})

	// Wait for timeout
	time.Sleep(20 * time.Millisecond)

	// First call in half-open should execute
	executed1 := false
	done1 := make(chan bool)
	go func() {
		cb.Execute(func() error {
			executed1 = true
			time.Sleep(50 * time.Millisecond) // Hold the slot
			return nil
		})
		done1 <- true
	}()

	// Give goroutine time to start
	time.Sleep(5 * time.Millisecond)

	// Second call should be rejected while first is running
	executed2 := false
	err := cb.Execute(func() error {
		executed2 = true
		return nil
	})

	<-done1

	assert.True(t, executed1, "first call should execute")
	assert.False(t, executed2, "second call should be rejected due to max calls limit")
	assert.Error(t, err)
}

func TestCircuitBreakerStats_LastFailure(t *testing.T) {
	cb := NewCircuitBreaker("test", CircuitBreakerConfig{
		FailureThreshold: 5,
	})

	// No failures yet
	stats := cb.Stats()
	assert.True(t, stats.LastFailureTime.IsZero())

	// Record a failure
	testErr := errors.New("test error")
	cb.Execute(func() error {
		return testErr
	})

	stats = cb.Stats()
	assert.False(t, stats.LastFailureTime.IsZero())
	assert.WithinDuration(t, time.Now(), stats.LastFailureTime, time.Second)
}
