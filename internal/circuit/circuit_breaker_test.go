package circuit

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCircuitBreakerBasic(t *testing.T) {
	cb := NewCircuitBreaker(2, 100*time.Millisecond)

	assert.Equal(t, StateClosed, cb.GetState())

	err := cb.Execute(func() error {
		return errors.New("test error")
	})
	assert.Error(t, err)
	assert.Equal(t, StateClosed, cb.GetState())

	err = cb.Execute(func() error {
		return errors.New("test error")
	})
	assert.Error(t, err)
	assert.Equal(t, StateOpen, cb.GetState())
}

func TestCircuitBreakerSuccess(t *testing.T) {
	cb := NewCircuitBreaker(2, 100*time.Millisecond)

	cb.Execute(func() error {
		return errors.New("test error")
	})

	err := cb.Execute(func() error {
		return nil
	})
	assert.NoError(t, err)
	assert.Equal(t, StateClosed, cb.GetState())
}

func TestCircuitBreakerHalfOpen(t *testing.T) {
	cb := NewCircuitBreaker(2, 50*time.Millisecond)

	// Trigger circuit breaker to open
	cb.Execute(func() error { return errors.New("error 1") })
	cb.Execute(func() error { return errors.New("error 2") })
	assert.Equal(t, StateOpen, cb.GetState())

	// Wait for reset timeout
	time.Sleep(60 * time.Millisecond)

	// First call should transition to Half-Open and succeed
	err := cb.Execute(func() error {
		return nil
	})
	assert.NoError(t, err)
	assert.Equal(t, StateClosed, cb.GetState())
}
