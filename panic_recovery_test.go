// Copyright (c) 2026 Onur Cinar.
// The source code is provided under MIT License.
// https://github.com/cinar/resile

package resile

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestPanicRecovery(t *testing.T) {
	t.Run("Default Behavior Panics", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("expected panic, but did not get one")
			}
		}()

		_, _ = Do(context.Background(), func(ctx context.Context) (string, error) {
			panic("test panic")
		})
	})

	t.Run("WithPanicRecovery Catches Panic", func(t *testing.T) {
		val, err := Do(context.Background(), func(ctx context.Context) (string, error) {
			panic("test panic")
		}, WithPanicRecovery(), WithMaxAttempts(1))

		if val != "" {
			t.Errorf("expected empty value, got %v", val)
		}

		var panicErr *PanicError
		if !errors.As(err, &panicErr) {
			t.Errorf("expected *PanicError, got %T: %v", err, err)
		} else if panicErr.Value != "test panic" {
			t.Errorf("expected panic value 'test panic', got %v", panicErr.Value)
		}
	})

	t.Run("Panics are Retried", func(t *testing.T) {
		attempts := 0
		_, err := Do(context.Background(), func(ctx context.Context) (string, error) {
			attempts++
			if attempts < 3 {
				panic("transient panic")
			}
			return "success", nil
		}, WithPanicRecovery(), WithMaxAttempts(5), WithBaseDelay(1*time.Millisecond))

		if err != nil {
			t.Errorf("expected success, got error: %v", err)
		}

		if attempts != 3 {
			t.Errorf("expected 3 attempts, got %d", attempts)
		}
	})

	t.Run("Hedged Panics are Recovered", func(t *testing.T) {
		val, err := DoHedged(context.Background(), func(ctx context.Context) (string, error) {
			panic("hedged panic")
		}, WithPanicRecovery(), WithMaxAttempts(2), WithHedgingDelay(10*time.Millisecond))

		if val != "" {
			t.Errorf("expected empty value, got %v", val)
		}

		var panicErr *PanicError
		if !errors.As(err, &panicErr) {
			t.Errorf("expected *PanicError, got %T: %v", err, err)
		}
	})
}
