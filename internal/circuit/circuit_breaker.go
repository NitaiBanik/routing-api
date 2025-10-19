package circuit

import (
	"sync"
	"time"
)

type CircuitBreakerState int

const (
	StateClosed CircuitBreakerState = iota
	StateOpen
	StateHalfOpen
)

type CircuitBreaker struct {
	state         CircuitBreakerState
	failureCount  int
	maxFailures   int
	resetTimeout  time.Duration
	lastIssueTime time.Time
	slowThreshold time.Duration
	slowCount     int
	maxSlowCount  int
	mutex         sync.RWMutex
}

func NewCircuitBreaker(maxFailures int, resetTimeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		state:         StateClosed,
		failureCount:  0,
		maxFailures:   maxFailures,
		resetTimeout:  resetTimeout,
		slowThreshold: 5 * time.Second,
		slowCount:     0,
		maxSlowCount:  3,
	}
}

func NewCircuitBreakerWithSlowThreshold(maxFailures int, resetTimeout time.Duration, slowThreshold time.Duration, maxSlowCount int) *CircuitBreaker {
	return &CircuitBreaker{
		state:         StateClosed,
		failureCount:  0,
		maxFailures:   maxFailures,
		resetTimeout:  resetTimeout,
		slowThreshold: slowThreshold,
		slowCount:     0,
		maxSlowCount:  maxSlowCount,
	}
}

func (cb *CircuitBreaker) Execute(operation func() error) error {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	switch cb.state {
	case StateClosed:
	case StateOpen:
		if time.Since(cb.lastIssueTime) >= cb.resetTimeout {
			cb.state = StateHalfOpen
		} else {
			return &CircuitBreakerError{Message: "circuit breaker is open"}
		}
	case StateHalfOpen:
	}

	startTime := time.Now()
	err := operation()
	responseTime := time.Since(startTime)

	isSlow := responseTime > cb.slowThreshold

	if err != nil || isSlow {
		cb.lastIssueTime = time.Now()

		if err != nil {
			cb.failureCount++
		}
		if isSlow {
			cb.slowCount++
		}

		if cb.state == StateHalfOpen || cb.failureCount >= cb.maxFailures || cb.slowCount >= cb.maxSlowCount {
			cb.state = StateOpen
		}

		if isSlow && err == nil {
			return &CircuitBreakerError{Message: "response too slow"}
		}
		return err
	}

	cb.failureCount = 0
	cb.slowCount = 0
	cb.state = StateClosed
	return nil
}

func (cb *CircuitBreaker) IsOpen() bool {
	cb.mutex.RLock()
	defer cb.mutex.RUnlock()

	return cb.state == StateOpen
}

func (cb *CircuitBreaker) GetState() CircuitBreakerState {
	cb.mutex.RLock()
	defer cb.mutex.RUnlock()
	return cb.state
}

func (cb *CircuitBreaker) GetSlowCount() int {
	cb.mutex.RLock()
	defer cb.mutex.RUnlock()
	return cb.slowCount
}

func (cb *CircuitBreaker) GetFailureCount() int {
	cb.mutex.RLock()
	defer cb.mutex.RUnlock()
	return cb.failureCount
}

type CircuitBreakerError struct {
	Message string
}

func (e *CircuitBreakerError) Error() string {
	return e.Message
}
