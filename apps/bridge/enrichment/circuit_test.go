package enrichment

import (
	"testing"
	"time"
)

func TestCircuitBreaker_AllowsInitially(t *testing.T) {
	cb := NewCircuitBreaker(3, time.Minute, 5*time.Minute)
	if !cb.Allow() {
		t.Fatal("expected Allow() to be true initially")
	}
}

func TestCircuitBreaker_TripsAfterThreshold(t *testing.T) {
	cb := NewCircuitBreaker(3, time.Minute, 5*time.Minute)

	cb.RecordFailure()
	cb.RecordFailure()
	if cb.IsTripped() {
		t.Fatal("should not be tripped after 2 failures")
	}

	cb.RecordFailure() // 3rd failure — trips
	if !cb.IsTripped() {
		t.Fatal("should be tripped after 3 failures")
	}
	if cb.Allow() {
		t.Fatal("should not allow calls when tripped")
	}
}

func TestCircuitBreaker_ResetsOnSuccess(t *testing.T) {
	cb := NewCircuitBreaker(3, time.Minute, 5*time.Minute)

	cb.RecordFailure()
	cb.RecordFailure()
	cb.RecordSuccess()

	// After success, counter resets — 3rd failure shouldn't trip
	cb.RecordFailure()
	if cb.IsTripped() {
		t.Fatal("should not be tripped — success reset the counter")
	}
}

func TestCircuitBreaker_CooldownReset(t *testing.T) {
	cb := NewCircuitBreaker(2, time.Minute, 50*time.Millisecond)

	cb.RecordFailure()
	cb.RecordFailure() // trips
	if !cb.IsTripped() {
		t.Fatal("should be tripped")
	}

	// Wait for cooldown
	time.Sleep(60 * time.Millisecond)

	if cb.IsTripped() {
		t.Fatal("should have reset after cooldown")
	}
	if !cb.Allow() {
		t.Fatal("should allow calls after cooldown")
	}
}

func TestCircuitBreaker_WindowReset(t *testing.T) {
	cb := NewCircuitBreaker(3, 50*time.Millisecond, 5*time.Minute)

	cb.RecordFailure()
	cb.RecordFailure()

	// Wait for window to expire
	time.Sleep(60 * time.Millisecond)

	// This failure starts a new window — shouldn't trip (only 1 in new window)
	cb.RecordFailure()
	if cb.IsTripped() {
		t.Fatal("should not be tripped — window expired, only 1 failure in new window")
	}
}
