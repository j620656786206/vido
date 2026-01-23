package metadata

import (
	"errors"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"
)

// CircuitState represents the current state of a circuit breaker
type CircuitState int

const (
	// CircuitStateClosed is the normal operating state
	CircuitStateClosed CircuitState = iota
	// CircuitStateOpen is when the circuit is tripped and rejecting requests
	CircuitStateOpen
	// CircuitStateHalfOpen is when the circuit is testing if it can close
	CircuitStateHalfOpen
)

// String returns the string representation of CircuitState
func (s CircuitState) String() string {
	switch s {
	case CircuitStateClosed:
		return "closed"
	case CircuitStateOpen:
		return "open"
	case CircuitStateHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

// ErrCircuitOpen is returned when the circuit breaker is open
var ErrCircuitOpen = errors.New("circuit breaker is open")

// CircuitBreakerConfig holds configuration for a circuit breaker
type CircuitBreakerConfig struct {
	// FailureThreshold is the number of consecutive failures before opening (default: 5)
	FailureThreshold int
	// SuccessThreshold is the number of consecutive successes in half-open to close (default: 2)
	SuccessThreshold int
	// Timeout is the duration to wait in open state before transitioning to half-open (default: 30s)
	Timeout time.Duration
	// HalfOpenMaxCalls is the max concurrent calls allowed in half-open state (default: 1)
	HalfOpenMaxCalls int
	// OnStateChange is called when the circuit state changes
	OnStateChange func(name string, from, to CircuitState)
}

// CircuitBreakerStats holds statistics for a circuit breaker
type CircuitBreakerStats struct {
	// TotalCalls is the total number of calls made
	TotalCalls int
	// SuccessCount is the number of successful calls
	SuccessCount int
	// FailureCount is the number of failed calls
	FailureCount int
	// ConsecutiveFailures is the current consecutive failure count
	ConsecutiveFailures int
	// ConsecutiveSuccesses is the current consecutive success count in half-open
	ConsecutiveSuccesses int
	// LastFailureTime is when the last failure occurred
	LastFailureTime time.Time
	// LastStateChange is when the state last changed
	LastStateChange time.Time
	// FailureThreshold from config
	FailureThreshold int
	// SuccessThreshold from config
	SuccessThreshold int
	// Timeout from config
	Timeout time.Duration
}

// CircuitBreaker implements the circuit breaker pattern
type CircuitBreaker struct {
	name   string
	config CircuitBreakerConfig

	mu                   sync.RWMutex
	state                CircuitState
	consecutiveFailures  int
	consecutiveSuccesses int
	totalCalls           int
	successCount         int
	failureCount         int
	lastFailureTime      time.Time
	lastStateChange      time.Time
	openedAt             time.Time

	// For half-open state limiting
	halfOpenCalls atomic.Int32
}

// NewCircuitBreaker creates a new circuit breaker with the given configuration
func NewCircuitBreaker(name string, config CircuitBreakerConfig) *CircuitBreaker {
	// Apply defaults
	if config.FailureThreshold <= 0 {
		config.FailureThreshold = 5
	}
	if config.SuccessThreshold <= 0 {
		config.SuccessThreshold = 2
	}
	if config.Timeout <= 0 {
		config.Timeout = 30 * time.Second
	}
	if config.HalfOpenMaxCalls <= 0 {
		config.HalfOpenMaxCalls = 1
	}

	return &CircuitBreaker{
		name:            name,
		config:          config,
		state:           CircuitStateClosed,
		lastStateChange: time.Now(),
	}
}

// Name returns the name of the circuit breaker
func (cb *CircuitBreaker) Name() string {
	return cb.name
}

// State returns the current state of the circuit breaker
func (cb *CircuitBreaker) State() CircuitState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.currentState()
}

// currentState returns the state, checking for timeout transition
// Must be called with at least read lock held
func (cb *CircuitBreaker) currentState() CircuitState {
	if cb.state == CircuitStateOpen {
		// Check if we should transition to half-open
		if time.Since(cb.openedAt) >= cb.config.Timeout {
			return CircuitStateHalfOpen
		}
	}
	return cb.state
}

// Execute runs the given function if the circuit is closed or half-open
func (cb *CircuitBreaker) Execute(fn func() error) error {
	// Check if we can proceed
	if err := cb.beforeExecute(); err != nil {
		return err
	}

	// Execute the function
	err := fn()

	// Record the result
	cb.afterExecute(err)

	return err
}

// beforeExecute checks if the request should proceed
func (cb *CircuitBreaker) beforeExecute() error {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	state := cb.currentState()

	switch state {
	case CircuitStateClosed:
		return nil

	case CircuitStateOpen:
		return ErrCircuitOpen

	case CircuitStateHalfOpen:
		// Check if we're at max concurrent calls for half-open
		if int(cb.halfOpenCalls.Load()) >= cb.config.HalfOpenMaxCalls {
			return ErrCircuitOpen
		}
		cb.halfOpenCalls.Add(1)

		// Transition from open to half-open if needed
		if cb.state == CircuitStateOpen {
			cb.transitionTo(CircuitStateHalfOpen)
		}
		return nil
	}

	return nil
}

// afterExecute records the result of the execution
func (cb *CircuitBreaker) afterExecute(err error) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.totalCalls++

	state := cb.currentState()

	// Decrement half-open calls if we were in half-open
	if state == CircuitStateHalfOpen {
		cb.halfOpenCalls.Add(-1)
	}

	if err != nil {
		cb.recordFailure()
	} else {
		cb.recordSuccess()
	}
}

// recordFailure records a failure
// Must be called with lock held
func (cb *CircuitBreaker) recordFailure() {
	cb.failureCount++
	cb.consecutiveFailures++
	cb.consecutiveSuccesses = 0
	cb.lastFailureTime = time.Now()

	state := cb.currentState()

	switch state {
	case CircuitStateClosed:
		if cb.consecutiveFailures >= cb.config.FailureThreshold {
			cb.transitionTo(CircuitStateOpen)
		}

	case CircuitStateHalfOpen:
		// Any failure in half-open reopens the circuit
		cb.transitionTo(CircuitStateOpen)
	}
}

// recordSuccess records a success
// Must be called with lock held
func (cb *CircuitBreaker) recordSuccess() {
	cb.successCount++
	cb.consecutiveFailures = 0

	state := cb.currentState()

	if state == CircuitStateHalfOpen {
		cb.consecutiveSuccesses++
		if cb.consecutiveSuccesses >= cb.config.SuccessThreshold {
			cb.transitionTo(CircuitStateClosed)
		}
	}
}

// transitionTo changes the circuit state
// Must be called with lock held
func (cb *CircuitBreaker) transitionTo(newState CircuitState) {
	if cb.state == newState {
		return
	}

	oldState := cb.state
	cb.state = newState
	cb.lastStateChange = time.Now()

	if newState == CircuitStateOpen {
		cb.openedAt = time.Now()
		cb.consecutiveSuccesses = 0
	}

	if newState == CircuitStateClosed {
		cb.consecutiveFailures = 0
		cb.consecutiveSuccesses = 0
	}

	slog.Info("Circuit breaker state changed",
		"name", cb.name,
		"from", oldState.String(),
		"to", newState.String(),
	)

	if cb.config.OnStateChange != nil {
		// Call in goroutine to avoid deadlock if callback tries to access circuit
		go cb.config.OnStateChange(cb.name, oldState, newState)
	}
}

// Reset resets the circuit breaker to closed state
func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	oldState := cb.state
	cb.state = CircuitStateClosed
	cb.consecutiveFailures = 0
	cb.consecutiveSuccesses = 0
	cb.totalCalls = 0
	cb.successCount = 0
	cb.failureCount = 0
	cb.lastFailureTime = time.Time{}
	cb.lastStateChange = time.Now()
	cb.halfOpenCalls.Store(0)

	if oldState != CircuitStateClosed {
		slog.Info("Circuit breaker reset",
			"name", cb.name,
			"previous_state", oldState.String(),
		)

		if cb.config.OnStateChange != nil {
			go cb.config.OnStateChange(cb.name, oldState, CircuitStateClosed)
		}
	}
}

// Stats returns the current statistics
func (cb *CircuitBreaker) Stats() CircuitBreakerStats {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	return CircuitBreakerStats{
		TotalCalls:           cb.totalCalls,
		SuccessCount:         cb.successCount,
		FailureCount:         cb.failureCount,
		ConsecutiveFailures:  cb.consecutiveFailures,
		ConsecutiveSuccesses: cb.consecutiveSuccesses,
		LastFailureTime:      cb.lastFailureTime,
		LastStateChange:      cb.lastStateChange,
		FailureThreshold:     cb.config.FailureThreshold,
		SuccessThreshold:     cb.config.SuccessThreshold,
		Timeout:              cb.config.Timeout,
	}
}
