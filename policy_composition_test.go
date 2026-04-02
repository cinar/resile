// Copyright (c) 2026 Onur Cinar.
// The source code is provided under MIT License.
// https://github.com/cinar/resile

package resile

import (
	"context"
	"errors"
	"github.com/cinar/resile/circuit"
	"testing"
)

func TestPolicyComposition_Order(t *testing.T) {
	t.Run("Retry wraps CircuitBreaker", func(t *testing.T) {
		cb := circuit.New(circuit.Config{
			MinimumCalls:         10,
			FailureRateThreshold: 100,
		})

		// Retry(3) wraps CircuitBreaker.
		// Each retry attempt goes through CB.
		policy := NewPolicy(
			WithRetry(3),
			WithCircuitBreaker(cb),
		)

		ctx := context.Background()
		var calls int
		err := policy.DoErr(ctx, func(ctx context.Context) error {
			calls++
			return errors.New("fail")
		})

		if err == nil {
			t.Fatal("expected error")
		}
		if calls != 3 {
			t.Errorf("expected 3 calls, got %d", calls)
		}

		// CB should have seen 3 failures.
		// We can't easily check internal failure count, but we can check if it's still closed.
		if cb.State() != circuit.StateClosed {
			t.Errorf("expected CB to be Closed, got %v", cb.State())
		}
	})

	t.Run("CircuitBreaker wraps Retry", func(t *testing.T) {
		cb := circuit.New(circuit.Config{
			MinimumCalls:         2,
			FailureRateThreshold: 100,
		})

		// CircuitBreaker wraps Retry(3).
		// The whole retry loop is one CB execution.
		policy := NewPolicy(
			WithCircuitBreaker(cb),
			WithRetry(3),
		)

		ctx := context.Background()
		var calls int
		err := policy.DoErr(ctx, func(ctx context.Context) error {
			calls++
			return errors.New("fail")
		})

		if err == nil {
			t.Fatal("expected error")
		}
		if calls != 3 {
			t.Errorf("expected 3 calls, got %d", calls)
		}

		// CB should have seen only 1 failure (the whole retry loop failing).
		// Since threshold is 2, it should still be Closed.
		if cb.State() != circuit.StateClosed {
			t.Errorf("expected CB to be Closed, got %v", cb.State())
		}

		// Run it again to trip it.
		_ = policy.DoErr(ctx, func(ctx context.Context) error {
			return errors.New("fail")
		})

		if cb.State() != circuit.StateOpen {
			t.Errorf("expected CB to be Open after 2nd failed retry loop, got %v", cb.State())
		}
	})
}

func TestPolicyComposition_Bulkhead(t *testing.T) {
	policy := NewPolicy(
		WithBulkhead(1),
	)

	ctx := context.Background()
	start := make(chan struct{})
	done := make(chan struct{})

	go func() {
		_ = policy.DoErr(ctx, func(ctx context.Context) error {
			close(start)
			<-done
			return nil
		})
	}()

	<-start

	// Second call should fail due to bulkhead
	err := policy.DoErr(ctx, func(ctx context.Context) error {
		return nil
	})

	if !errors.Is(err, ErrBulkheadFull) {
		t.Errorf("expected ErrBulkheadFull, got %v", err)
	}

	close(done)
}

func TestLegacyNew_DefaultOrder(t *testing.T) {
	cb := circuit.New(circuit.Config{
		MinimumCalls:         1,
		FailureRateThreshold: 100,
	})

	// Legacy New should use default order: Retry wraps CB.
	retryer := New(
		WithMaxAttempts(3),
		WithCircuitBreaker(cb),
	)

	ctx := context.Background()
	_ = retryer.DoErr(ctx, func(ctx context.Context) error {
		return errors.New("fail")
	})

	// If it was default order, CB should be Open after 3 attempts.
	if cb.State() != circuit.StateOpen {
		t.Errorf("expected CB to be Open with legacy New, got %v", cb.State())
	}
}
