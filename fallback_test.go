// Copyright (c) 2026 Onur Cinar.
// The source code is provided under MIT License.
// https://github.com/cinar/resile

package resile

import (
	"context"
	"errors"
	"testing"

	"github.com/cinar/resile/circuit"
)

func TestDo_Fallback(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	errTest := errors.New("test error")
	fallbackValue := "stale data"

	t.Run("Fallback_ExhaustedRetries", func(t *testing.T) {
		var count int
		val, err := Do(ctx, func(ctx context.Context) (string, error) {
			count++
			return "", errTest
		},
			WithMaxAttempts(3),
			WithBaseDelay(0),
			WithFallback(func(ctx context.Context, err error) (string, error) {
				if !errors.Is(err, errTest) {
					t.Errorf("expected %v, got %v", errTest, err)
				}
				return fallbackValue, nil
			}),
		)

		if err != nil {
			t.Errorf("expected nil error after fallback, got %v", err)
		}
		if val != fallbackValue {
			t.Errorf("expected %q, got %q", fallbackValue, val)
		}
		if count != 3 {
			t.Errorf("expected 3 attempts, got %d", count)
		}
	})

	t.Run("Fallback_CircuitOpen", func(t *testing.T) {
		cb := circuit.New(circuit.Config{
			MinimumCalls:         1,
			FailureRateThreshold: 100,
		})

		// Trigger circuit open
		_ = DoErr(ctx, func(ctx context.Context) error {
			return errTest
		}, WithCircuitBreaker(cb), WithMaxAttempts(1))

		val, err := Do(ctx, func(ctx context.Context) (string, error) {
			return "fresh", nil
		},
			WithCircuitBreaker(cb),
			WithFallback(func(ctx context.Context, err error) (string, error) {
				if !errors.Is(err, circuit.ErrCircuitOpen) {
					t.Errorf("expected %v, got %v", circuit.ErrCircuitOpen, err)
				}
				return fallbackValue, nil
			}),
		)

		if err != nil {
			t.Errorf("expected nil error after fallback, got %v", err)
		}
		if val != fallbackValue {
			t.Errorf("expected %q, got %q", fallbackValue, val)
		}
	})

	t.Run("Fallback_TypeMismatch", func(t *testing.T) {
		// If the fallback type doesn't match, it should return the original error.
		_, err := Do(ctx, func(ctx context.Context) (string, error) {
			return "", errTest
		},
			WithMaxAttempts(1),
			WithFallbackErr(func(ctx context.Context, err error) error {
				return nil // Should not be called
			}),
		)

		if !errors.Is(err, errTest) {
			t.Errorf("expected %v, got %v", errTest, err)
		}
	})

	t.Run("Fallback_WithDoState", func(t *testing.T) {
		var capturedAttempt uint
		val, err := DoState(ctx, func(ctx context.Context, state RetryState) (string, error) {
			capturedAttempt = state.Attempt
			return "", errTest
		},
			WithMaxAttempts(2),
			WithBaseDelay(0),
			WithFallback(func(ctx context.Context, err error) (string, error) {
				return "stateful-fallback", nil
			}),
		)

		if err != nil {
			t.Errorf("expected nil error, got %v", err)
		}
		if val != "stateful-fallback" {
			t.Errorf("expected 'stateful-fallback', got %q", val)
		}
		if capturedAttempt != 1 {
			t.Errorf("expected last attempt to be 1, got %d", capturedAttempt)
		}
	})

	t.Run("Fallback_ReturningError", func(t *testing.T) {
		errFallback := errors.New("fallback failure")
		_, err := Do(ctx, func(ctx context.Context) (string, error) {
			return "", errTest
		},
			WithMaxAttempts(1),
			WithFallback(func(ctx context.Context, err error) (string, error) {
				return "", errFallback
			}),
		)

		if !errors.Is(err, errFallback) {
			t.Errorf("expected %v, got %v", errFallback, err)
		}
	})
}

func TestDoErr_Fallback(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	errTest := errors.New("test error")
	errFallback := errors.New("fallback error")

	t.Run("FallbackErr_Success", func(t *testing.T) {
		err := DoErr(ctx, func(ctx context.Context) error {
			return errTest
		},
			WithMaxAttempts(1),
			WithFallbackErr(func(ctx context.Context, err error) error {
				return nil
			}),
		)

		if err != nil {
			t.Errorf("expected nil error after fallback, got %v", err)
		}
	})

	t.Run("FallbackErr_Transformation", func(t *testing.T) {
		err := DoErr(ctx, func(ctx context.Context) error {
			return errTest
		},
			WithMaxAttempts(1),
			WithFallbackErr(func(ctx context.Context, err error) error {
				return errFallback
			}),
		)

		if !errors.Is(err, errFallback) {
			t.Errorf("expected %v, got %v", errFallback, err)
		}
	})
}

func TestRetryer_Fallback(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	errTest := errors.New("test error")
	fallbackValue := "fallback"

	t.Run("Retryer_Do_FallbackMatch", func(t *testing.T) {
		// If we configure with [any], it should match Retryer.Do which uses [any].
		retryer := New(
			WithMaxAttempts(1),
			WithFallback[any](func(ctx context.Context, err error) (any, error) {
				return fallbackValue, nil
			}),
		)

		val, err := retryer.Do(ctx, func(ctx context.Context) (any, error) {
			return nil, errTest
		})

		if err != nil {
			t.Errorf("expected nil error, got %v", err)
		}
		if val != fallbackValue {
			t.Errorf("expected %q, got %v", fallbackValue, val)
		}
	})

	t.Run("Retryer_Do_FallbackMismatch", func(t *testing.T) {
		// If we configure with [string], it won't match Retryer.Do which uses [any].
		retryer := New(
			WithMaxAttempts(1),
			WithFallback[string](func(ctx context.Context, err error) (string, error) {
				return fallbackValue, nil
			}),
		)

		_, err := retryer.Do(ctx, func(ctx context.Context) (any, error) {
			return nil, errTest
		})

		if !errors.Is(err, errTest) {
			t.Errorf("expected %v, got %v", errTest, err)
		}
	})
}
