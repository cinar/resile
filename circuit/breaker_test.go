// Copyright (c) 2026 Onur Cinar.
// The source code is provided under MIT License.
// https://github.com/cinar/resile

package circuit

import (
	"errors"
	"testing"
	"time"
)

func TestBreaker_Transitions(t *testing.T) {
	config := Config{
		FailureThreshold: 2,
		ResetTimeout:     100 * time.Millisecond,
	}
	cb := New(config)

	errTest := errors.New("test error")

	t.Run("InitiallyClosed", func(t *testing.T) {
		if cb.State() != StateClosed {
			t.Error("expected circuit to be closed initially")
		}
	})

	t.Run("OpensAfterThreshold", func(t *testing.T) {
		_ = cb.Execute(func() error { return errTest })
		_ = cb.Execute(func() error { return errTest })

		if cb.State() != StateOpen {
			t.Error("expected circuit to open after 2 failures")
		}

		err := cb.Execute(func() error { return nil })
		if !errors.Is(err, ErrCircuitOpen) {
			t.Errorf("expected ErrCircuitOpen, got %v", err)
		}
	})

	t.Run("HalfOpenAfterTimeout", func(t *testing.T) {
		time.Sleep(150 * time.Millisecond) // Wait for ResetTimeout to elapse.

		if cb.State() != StateHalfOpen {
			t.Errorf("expected circuit to be half-open after timeout, got %v", cb.State())
		}
	})

	t.Run("ClosesAfterSuccessInHalfOpen", func(t *testing.T) {
		err := cb.Execute(func() error { return nil })
		if err != nil {
			t.Errorf("expected nil error on success in half-open, got %v", err)
		}
		if cb.State() != StateClosed {
			t.Error("expected circuit to close after success in half-open")
		}
	})

	t.Run("ReopensAfterFailureInHalfOpen", func(t *testing.T) {
		// Open it again.
		_ = cb.Execute(func() error { return errTest })
		_ = cb.Execute(func() error { return errTest })

		time.Sleep(150 * time.Millisecond) // Wait for ResetTimeout.

		// In Half-Open, a single failure should reopen.
		_ = cb.Execute(func() error { return errTest })
		if cb.State() != StateOpen {
			t.Error("expected circuit to reopen after failure in half-open")
		}
	})
}
