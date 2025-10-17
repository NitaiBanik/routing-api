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

	cb.Execute(func() error { return errors.New("error 1") })
	cb.Execute(func() error { return errors.New("error 2") })
	assert.Equal(t, StateOpen, cb.GetState())

	time.Sleep(60 * time.Millisecond)
	err := cb.Execute(func() error {
		return nil
	})
	assert.NoError(t, err)
	assert.Equal(t, StateClosed, cb.GetState())
}

func TestCircuitBreaker_SlowResponse(t *testing.T) {
	cb := NewCircuitBreakerWithSlowThreshold(5, 100*time.Millisecond, 50*time.Millisecond, 2)

	err := cb.Execute(func() error {
		time.Sleep(10 * time.Millisecond)
		return nil
	})
	assert.NoError(t, err)
	assert.Equal(t, StateClosed, cb.GetState())
	assert.Equal(t, 0, cb.GetSlowCount())

	err = cb.Execute(func() error {
		time.Sleep(100 * time.Millisecond)
		return nil
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "response too slow")
	assert.Equal(t, StateClosed, cb.GetState())
	assert.Equal(t, 1, cb.GetSlowCount())

	err = cb.Execute(func() error {
		time.Sleep(100 * time.Millisecond)
		return nil
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "response too slow")
	assert.Equal(t, StateOpen, cb.GetState())
	assert.Equal(t, 2, cb.GetSlowCount())
}

func TestCircuitBreaker_SlowResponseWithError(t *testing.T) {
	cb := NewCircuitBreakerWithSlowThreshold(2, 100*time.Millisecond, 50*time.Millisecond, 2)

	err := cb.Execute(func() error {
		time.Sleep(100 * time.Millisecond)
		return errors.New("network error")
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "network error")
	assert.Equal(t, StateClosed, cb.GetState())
	assert.Equal(t, 1, cb.GetFailureCount())
	assert.Equal(t, 1, cb.GetSlowCount())
}
