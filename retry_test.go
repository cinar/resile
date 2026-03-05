// Copyright (c) 2026 Onur Cinar.
// The source code is provided under MIT License.
// https://github.com/cinar/resile

package resile

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

func TestRetryLoop_ContextCancellation(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	config := DefaultConfig()
	config.BaseDelay = 200 * time.Millisecond // Longer than the context timeout.

	err := config.execute(ctx, func(ctx context.Context, _ RetryState) error {
		return errTest
	})

	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("expected context.DeadlineExceeded, got %v", err)
	}

	if !errors.Is(err, errTest) {
		t.Errorf("expected wrapped lastErr %v, got %v", errTest, err)
	}
}

func TestRetryLoop_MaxAttempts(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	config := DefaultConfig()
	config.MaxAttempts = 3
	config.BaseDelay = 0 // Skip sleep.
	config.Backoff = NewFullJitter(0, 0)

	var count uint
	err := config.execute(ctx, func(ctx context.Context, _ RetryState) error {
		count++
		return errTest
	})

	if count != 3 {
		t.Errorf("expected 3 attempts, got %d", count)
	}

	if !errors.Is(err, errTest) {
		t.Errorf("expected %v, got %v", errTest, err)
	}
}

func TestRetryLoop_FatalError(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	config := DefaultConfig()
	config.MaxAttempts = 10
	config.BaseDelay = 0

	var count uint
	err := config.execute(ctx, func(ctx context.Context, _ RetryState) error {
		count++
		if count == 2 {
			return FatalError(errTest)
		}
		return errOther
	})

	if count != 2 {
		t.Errorf("expected execution to stop at attempt 2 due to fatal error, got %d", count)
	}

	if !errors.Is(err, errTest) {
		t.Errorf("expected %v, got %v", errTest, err)
	}
}

func TestRetryLoop_Success(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	config := DefaultConfig()
	config.MaxAttempts = 5
	config.BaseDelay = 0

	var count uint
	err := config.execute(ctx, func(ctx context.Context, _ RetryState) error {
		count++
		if count == 3 {
			return nil
		}
		return errTest
	})

	if count != 3 {
		t.Errorf("expected execution to stop at attempt 3 due to success, got %d", count)
	}

	if err != nil {
		t.Errorf("expected nil error on success, got %v", err)
	}
}

type mockRetryAfter struct {
	error
	delay time.Duration
}

func (m *mockRetryAfter) RetryAfter() time.Duration {
	return m.delay
}

func (m *mockRetryAfter) Unwrap() error {
	return m.error
}

func TestRetryLoop_RetryAfter(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	config := DefaultConfig()
	config.MaxAttempts = 2
	// Default jitter would be random up to baseDelay (100ms)
	// We override with a very short duration to ensure it's used.
	expectedDelay := 10 * time.Millisecond

	start := time.Now()
	err := config.execute(ctx, func(ctx context.Context, _ RetryState) error {
		return &mockRetryAfter{errTest, expectedDelay}
	})
	duration := time.Since(start)

	if !errors.Is(err, errTest) {
		t.Errorf("expected %v, got %v", errTest, err)
	}

	// We expect roughly 10ms sleep.
	if duration < expectedDelay {
		t.Errorf("expected sleep of at least %v, got %v", expectedDelay, duration)
	}
}

type mockInstrumenter struct {
	beforeCount int
	afterCount  int
}

func (m *mockInstrumenter) BeforeAttempt(ctx context.Context, state RetryState) context.Context {
	m.beforeCount++
	return ctx
}

func (m *mockInstrumenter) AfterAttempt(ctx context.Context, state RetryState) {
	m.afterCount++
}

func TestRetryLoop_Instrumentation(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	config := DefaultConfig()
	config.MaxAttempts = 3
	config.BaseDelay = 0
	instr := &mockInstrumenter{}
	config.Instrumenter = instr

	_ = config.execute(ctx, func(ctx context.Context, _ RetryState) error {
		return errTest
	})

	if instr.beforeCount != 3 {
		t.Errorf("expected 3 BeforeAttempt calls, got %d", instr.beforeCount)
	}
	if instr.afterCount != 3 {
		t.Errorf("expected 3 AfterAttempt calls, got %d", instr.afterCount)
	}
}

func TestRetryLoop_TestingBypass(t *testing.T) {
	t.Parallel()

	// Use a long delay that would normally time out the test.
	config := DefaultConfig()
	config.MaxAttempts = 3
	config.BaseDelay = 10 * time.Second

	ctx := WithTestingBypass(context.Background())

	start := time.Now()
	_ = config.execute(ctx, func(ctx context.Context, _ RetryState) error {
		return errTest
	})
	duration := time.Since(start)

	if duration > 1*time.Second {
		t.Errorf("expected execution to be fast due to bypass, took %v", duration)
	}
}

func TestDo_PublicAPI(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	t.Run("Do_Success", func(t *testing.T) {
		val, err := Do(ctx, func(ctx context.Context) (string, error) {
			return "ok", nil
		}, WithMaxAttempts(3))

		if err != nil {
			t.Errorf("expected nil error, got %v", err)
		}
		if val != "ok" {
			t.Errorf("expected 'ok', got %q", val)
		}
	})

	t.Run("DoErr_Success", func(t *testing.T) {
		err := DoErr(ctx, func(ctx context.Context) error {
			return nil
		}, WithMaxAttempts(3))

		if err != nil {
			t.Errorf("expected nil error, got %v", err)
		}
	})

	t.Run("Do_Retry", func(t *testing.T) {
		var count int
		val, err := Do(ctx, func(ctx context.Context) (int, error) {
			count++
			if count < 3 {
				return 0, errTest
			}
			return 42, nil
		}, WithMaxAttempts(5), WithBaseDelay(0))

		if err != nil {
			t.Errorf("expected nil error, got %v", err)
		}
		if val != 42 {
			t.Errorf("expected 42, got %d", val)
		}
		if count != 3 {
			t.Errorf("expected 3 attempts, got %d", count)
		}
	})
}

func TestDo_Stateful(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	endpoints := []string{"primary", "secondary", "tertiary"}

	var usedEndpoints []string
	_, _ = DoState(ctx, func(ctx context.Context, state RetryState) (string, error) {
		usedEndpoints = append(usedEndpoints, endpoints[state.Attempt])
		return "", errTest
	}, WithMaxAttempts(3), WithBaseDelay(0))

	if len(usedEndpoints) != 3 {
		t.Errorf("expected 3 attempts, got %d", len(usedEndpoints))
	}
	for i, ep := range endpoints {
		if usedEndpoints[i] != ep {
			t.Errorf("expected endpoint %s, got %s", ep, usedEndpoints[i])
		}
	}
}

func TestDo_WithName(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	name := "fetch-data"

	var capturedName string

	// We'll reuse the mock but we need to check the name.
	// Let's just use a simple inline instrumenter for this check.
	_, _ = Do(ctx, func(ctx context.Context) (string, error) {
		return "", errTest
	}, WithMaxAttempts(1), WithName(name), WithInstrumenter(&nameCaptureInstrumenter{name: &capturedName}))

	if capturedName != name {
		t.Errorf("expected name %q, got %q", name, capturedName)
	}
}

func TestRetryerInterface(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	retryer := New(WithMaxAttempts(2), WithBaseDelay(0))

	t.Run("Retryer_Do", func(t *testing.T) {
		var count int
		_, _ = retryer.Do(ctx, func(ctx context.Context) (any, error) {
			count++
			return nil, errTest
		})
		if count != 2 {
			t.Errorf("expected 2 attempts, got %d", count)
		}
	})

	t.Run("Retryer_DoErr", func(t *testing.T) {
		var count int
		_ = retryer.DoErr(ctx, func(ctx context.Context) error {
			count++
			return errTest
		})
		if count != 2 {
			t.Errorf("expected 2 attempts, got %d", count)
		}
	})
}

type nameCaptureInstrumenter struct {
	name *string
}

func (n *nameCaptureInstrumenter) BeforeAttempt(ctx context.Context, state RetryState) context.Context {
	*n.name = state.Name
	return ctx
}
func (n *nameCaptureInstrumenter) AfterAttempt(ctx context.Context, state RetryState) {}

func TestDoHedged_SuccessFirstAttempt(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	var count int32
	val, err := DoHedged(ctx, func(ctx context.Context) (string, error) {
		atomic.AddInt32(&count, 1)
		return "ok", nil
	}, WithMaxAttempts(3), WithHedgingDelay(50*time.Millisecond))

	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
	if val != "ok" {
		t.Errorf("expected 'ok', got %q", val)
	}
	if atomic.LoadInt32(&count) != 1 {
		t.Errorf("expected 1 attempt, got %d", count)
	}
}

func TestDoHedged_SpeculativeSuccess(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	var count int32
	start := time.Now()
	val, err := DoHedged(ctx, func(ctx context.Context) (string, error) {
		attempt := atomic.AddInt32(&count, 1)
		if attempt == 1 {
			// First attempt is slow.
			select {
			case <-ctx.Done():
				return "", ctx.Err()
			case <-time.After(200 * time.Millisecond):
				return "too-slow", nil
			}
		}
		// Second attempt is fast.
		return "fast-ok", nil
	}, WithMaxAttempts(3), WithHedgingDelay(50*time.Millisecond))

	duration := time.Since(start)

	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
	if val != "fast-ok" {
		t.Errorf("expected 'fast-ok', got %q", val)
	}
	// It should take roughly HedgingDelay (50ms) + small execution time.
	if duration > 150*time.Millisecond {
		t.Errorf("expected fast speculative success, took %v", duration)
	}
	if atomic.LoadInt32(&count) < 2 {
		t.Errorf("expected at least 2 attempts, got %d", count)
	}
}

func TestDoHedged_AllFail(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	var count int32
	_, err := DoHedged(ctx, func(ctx context.Context) (string, error) {
		atomic.AddInt32(&count, 1)
		return "", errTest
	}, WithMaxAttempts(3), WithHedgingDelay(10*time.Millisecond))

	if !errors.Is(err, errTest) {
		t.Errorf("expected %v, got %v", errTest, err)
	}
	if atomic.LoadInt32(&count) != 3 {
		t.Errorf("expected 3 attempts, got %d", count)
	}
}

func TestDoHedged_FatalError(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	var count int32
	_, err := DoHedged(ctx, func(ctx context.Context) (string, error) {
		attempt := atomic.AddInt32(&count, 1)
		if attempt == 1 {
			return "", FatalError(errTest)
		}
		return "", errOther
	}, WithMaxAttempts(3), WithHedgingDelay(10*time.Millisecond))

	if !errors.Is(err, errTest) {
		t.Errorf("expected %v, got %v", errTest, err)
	}
	// Note: since it's concurrent, count might be 1 or slightly more if others started before cancellation.
}

func TestDoErrHedged_Success(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	err := DoErrHedged(ctx, func(ctx context.Context) error {
		return nil
	}, WithMaxAttempts(3), WithHedgingDelay(50*time.Millisecond))

	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
}

func TestRetryerInterface_Hedged(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	retryer := New(WithMaxAttempts(2), WithHedgingDelay(0))

	t.Run("Retryer_DoHedged", func(t *testing.T) {
		var count int32
		_, _ = retryer.DoHedged(ctx, func(ctx context.Context) (any, error) {
			atomic.AddInt32(&count, 1)
			return nil, errTest
		})
		if atomic.LoadInt32(&count) != 2 {
			t.Errorf("expected 2 attempts, got %d", atomic.LoadInt32(&count))
		}
	})

	t.Run("Retryer_DoErrHedged", func(t *testing.T) {
		var count int32
		_ = retryer.DoErrHedged(ctx, func(ctx context.Context) error {
			atomic.AddInt32(&count, 1)
			return errTest
		})
		if atomic.LoadInt32(&count) != 2 {
			t.Errorf("expected 2 attempts, got %d", atomic.LoadInt32(&count))
		}
	})
}
