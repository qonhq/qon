package core

import (
	"sync"
	"time"

	"github.com/qonhq/qon/internal/config"
)

type breakerState int

const (
	stateClosed breakerState = iota
	stateOpen
	stateHalfOpen
)

type circuitBreaker struct {
	mu                 sync.Mutex
	state              breakerState
	failures           int
	halfOpenInFlight   int
	openedAt           time.Time
	failureThreshold   int
	openDuration       time.Duration
	halfOpenMaxRequest int
}

func newCircuitBreaker(cfg config.CircuitBreakerConfig) *circuitBreaker {
	if cfg.FailureThreshold <= 0 {
		cfg.FailureThreshold = 10
	}
	if cfg.OpenDuration <= 0 {
		cfg.OpenDuration = 5 * time.Second
	}
	if cfg.HalfOpenRequests <= 0 {
		cfg.HalfOpenRequests = 1
	}
	return &circuitBreaker{
		state:              stateClosed,
		failureThreshold:   cfg.FailureThreshold,
		openDuration:       cfg.OpenDuration,
		halfOpenMaxRequest: cfg.HalfOpenRequests,
	}
}

func (cb *circuitBreaker) Allow() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	now := time.Now()
	switch cb.state {
	case stateOpen:
		if now.Sub(cb.openedAt) >= cb.openDuration {
			cb.state = stateHalfOpen
			cb.halfOpenInFlight = 0
		} else {
			return false
		}
	}

	if cb.state == stateHalfOpen {
		if cb.halfOpenInFlight >= cb.halfOpenMaxRequest {
			return false
		}
		cb.halfOpenInFlight++
	}

	return true
}

func (cb *circuitBreaker) OnSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.failures = 0
	cb.state = stateClosed
	cb.halfOpenInFlight = 0
}

func (cb *circuitBreaker) OnFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	if cb.state == stateHalfOpen {
		cb.state = stateOpen
		cb.openedAt = time.Now()
		cb.halfOpenInFlight = 0
		return
	}
	cb.failures++
	if cb.failures >= cb.failureThreshold {
		cb.state = stateOpen
		cb.openedAt = time.Now()
	}
}
