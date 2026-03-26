package enrichment

import (
	"sync"
	"time"
)

// CircuitBreaker tracks failures per source and trips after a threshold.
// When tripped, calls are rejected for a cooldown period.
type CircuitBreaker struct {
	mu        sync.Mutex
	failures  int
	threshold int
	cooldown  time.Duration
	window    time.Duration
	firstFail time.Time
	trippedAt time.Time
}

// NewCircuitBreaker creates a breaker that trips after `threshold` failures
// within `window`, and stays open for `cooldown`.
func NewCircuitBreaker(threshold int, window, cooldown time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		threshold: threshold,
		window:    window,
		cooldown:  cooldown,
	}
}

// Allow returns true if the circuit is closed (calls allowed).
func (cb *CircuitBreaker) Allow() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if !cb.trippedAt.IsZero() {
		if time.Since(cb.trippedAt) > cb.cooldown {
			// Reset — half-open, allow one call
			cb.trippedAt = time.Time{}
			cb.failures = 0
			return true
		}
		return false // still tripped
	}
	return true
}

// RecordSuccess resets the failure counter.
func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.failures = 0
	cb.firstFail = time.Time{}
}

// RecordFailure increments the failure counter and trips if threshold reached.
func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	now := time.Now()

	// Reset window if first failure is too old
	if !cb.firstFail.IsZero() && now.Sub(cb.firstFail) > cb.window {
		cb.failures = 0
		cb.firstFail = time.Time{}
	}

	if cb.firstFail.IsZero() {
		cb.firstFail = now
	}

	cb.failures++
	if cb.failures >= cb.threshold {
		cb.trippedAt = now
	}
}

// IsTripped returns whether the breaker is currently open.
func (cb *CircuitBreaker) IsTripped() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	if cb.trippedAt.IsZero() {
		return false
	}
	if time.Since(cb.trippedAt) > cb.cooldown {
		return false
	}
	return true
}
