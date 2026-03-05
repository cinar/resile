// Copyright (c) 2026 Onur Cinar.
// The source code is provided under MIT License.
// https://github.com/cinar/resile

package resile

import (
	"errors"
	"testing"
	"time"

	"github.com/cinar/resile/circuit"
)

func TestOptions(t *testing.T) {
	t.Parallel()

	c := DefaultConfig()

	t.Run("WithName", func(t *testing.T) {
		WithName("test-op")(c)
		if c.Name != "test-op" {
			t.Errorf("expected test-op, got %s", c.Name)
		}
	})

	t.Run("WithMaxAttempts", func(t *testing.T) {
		WithMaxAttempts(10)(c)
		if c.MaxAttempts != 10 {
			t.Errorf("expected 10, got %d", c.MaxAttempts)
		}
	})

	t.Run("WithBaseDelay", func(t *testing.T) {
		WithBaseDelay(500 * time.Millisecond)(c)
		if c.BaseDelay != 500*time.Millisecond {
			t.Errorf("expected 500ms, got %v", c.BaseDelay)
		}
		if fj, ok := c.Backoff.(*fullJitter); ok {
			if fj.base != 500*time.Millisecond {
				t.Errorf("expected fj.base 500ms, got %v", fj.base)
			}
		}
	})

	t.Run("WithMaxDelay", func(t *testing.T) {
		WithMaxDelay(60 * time.Second)(c)
		if c.MaxDelay != 60*time.Second {
			t.Errorf("expected 60s, got %v", c.MaxDelay)
		}
		if fj, ok := c.Backoff.(*fullJitter); ok {
			if fj.cap != 60*time.Second {
				t.Errorf("expected fj.cap 60s, got %v", fj.cap)
			}
		}
	})

	t.Run("WithBackoff", func(t *testing.T) {
		custom := NewFullJitter(1, 1)
		WithBackoff(custom)(c)
		if c.Backoff != custom {
			t.Error("expected custom backoff")
		}
	})

	t.Run("WithRetryIf", func(t *testing.T) {
		err := errors.New("custom")
		WithRetryIf(err)(c)
		if c.Policy.retryIf != err {
			t.Error("expected custom retryIf")
		}
	})

	t.Run("WithRetryIfFunc", func(t *testing.T) {
		f := func(err error) bool { return true }
		WithRetryIfFunc(f)(c)
		if c.Policy.retryIfFunc == nil {
			t.Error("expected custom retryIfFunc")
		}
	})

	t.Run("WithInstrumenter", func(t *testing.T) {
		instr := &mockInstrumenter{}
		WithInstrumenter(instr)(c)
		if c.Instrumenter != instr {
			t.Error("expected custom instrumenter")
		}
	})

	t.Run("WithCircuitBreaker", func(t *testing.T) {
		cb := circuit.New(circuit.Config{})
		WithCircuitBreaker(cb)(c)
		if c.CircuitBreaker != cb {
			t.Error("expected custom circuit breaker")
		}
	})

	t.Run("WithHedgingDelay", func(t *testing.T) {
		WithHedgingDelay(50 * time.Millisecond)(c)
		if c.HedgingDelay != 50*time.Millisecond {
			t.Errorf("expected 50ms, got %v", c.HedgingDelay)
		}
	})
}
