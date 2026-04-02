// Copyright (c) 2026 Onur Cinar.
// The source code is provided under MIT License.
// https://github.com/cinar/resile

package circuit

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

func TestBreaker_Defaults(t *testing.T) {
	t.Parallel()
	cb := New(Config{})
	if cb.config.WindowSize != 100 {
		t.Errorf("expected 100, got %d", cb.config.WindowSize)
	}
	if cb.config.ResetTimeout != 60*time.Second {
		t.Errorf("expected 60s, got %v", cb.config.ResetTimeout)
	}
	if cb.config.FailureRateThreshold != 50.0 {
		t.Errorf("expected 50.0, got %f", cb.config.FailureRateThreshold)
	}
}

func TestBreaker_CountBased(t *testing.T) {
	t.Parallel()
	config := Config{
		WindowType:           WindowCountBased,
		WindowSize:           10,
		FailureRateThreshold: 50.0,
		MinimumCalls:         5,
		ResetTimeout:         200 * time.Millisecond,
		HalfOpenMaxCalls:     2,
	}
	cb := New(config)
	errTest := errors.New("test error")
	ctx := context.Background()

	t.Run("InitiallyClosed", func(t *testing.T) {
		if cb.State() != StateClosed {
			t.Error("expected circuit to be closed initially")
		}
	})

	t.Run("DoesNotOpenBeforeMinCalls", func(t *testing.T) {
		// 4 failures, still below MinimumCalls=5
		for i := 0; i < 4; i++ {
			_ = cb.Execute(ctx, func() error { return errTest })
		}
		if cb.State() != StateClosed {
			t.Error("expected circuit to stay closed before min calls")
		}
	})

	t.Run("OpensAfterThresholdReached", func(t *testing.T) {
		// 5th call is a failure. 5 failures out of 5 calls = 100% >= 50%
		_ = cb.Execute(ctx, func() error { return errTest })
		if cb.State() != StateOpen {
			t.Error("expected circuit to open after 5 failures")
		}
	})

	t.Run("HalfOpenAfterTimeout", func(t *testing.T) {
		time.Sleep(300 * time.Millisecond)
		if cb.State() != StateHalfOpen {
			t.Errorf("expected half-open, got %v", cb.State())
		}
	})

	t.Run("ClosesAfterSuccessfulProbes", func(t *testing.T) {
		// 2 probes allowed. Both success = 0% failure < 50%
		_ = cb.Execute(ctx, func() error { return nil })
		_ = cb.Execute(ctx, func() error { return nil })

		if cb.State() != StateClosed {
			t.Error("expected circuit to close after successful probes")
		}
	})

	t.Run("ReopensAfterFailedProbes", func(t *testing.T) {
		// Trip it again
		for i := 0; i < 5; i++ {
			_ = cb.Execute(ctx, func() error { return errTest })
		}
		time.Sleep(300 * time.Millisecond)

		// 1st probe fails. Should trip immediately.
		_ = cb.Execute(ctx, func() error { return errTest })

		if cb.State() != StateOpen {
			t.Error("expected circuit to reopen after failed probes")
		}
	})
}

func TestBreaker_TimeBased(t *testing.T) {
	t.Parallel()
	config := Config{
		WindowType:           WindowTimeBased,
		WindowDuration:       400 * time.Millisecond,
		FailureRateThreshold: 50.0,
		MinimumCalls:         2,
		ResetTimeout:         200 * time.Millisecond,
		HalfOpenMaxCalls:     1,
	}
	cb := New(config)
	errTest := errors.New("test error")
	ctx := context.Background()

	t.Run("OpensBasedOnTimeWindow", func(t *testing.T) {
		_ = cb.Execute(ctx, func() error { return errTest })
		_ = cb.Execute(ctx, func() error { return errTest })

		if cb.State() != StateOpen {
			t.Error("expected circuit to open")
		}
	})

	t.Run("WindowSlides", func(t *testing.T) {
		cb = New(config)
		// Record 2 successes
		_ = cb.Execute(ctx, func() error { return nil })
		_ = cb.Execute(ctx, func() error { return nil })

		// Wait for window to slide past those successes
		time.Sleep(500 * time.Millisecond)

		// Now record 2 failures. Since successes are gone, failure rate is 100%
		_ = cb.Execute(ctx, func() error { return errTest })
		_ = cb.Execute(ctx, func() error { return errTest })

		if cb.State() != StateOpen {
			t.Error("expected circuit to open after window slid")
		}
	})

	t.Run("AdvanceLongTime", func(t *testing.T) {
		cb = New(config)
		_ = cb.Execute(ctx, func() error { return nil })
		// Sleep much longer than window duration
		time.Sleep(800 * time.Millisecond)
		_ = cb.Execute(ctx, func() error { return errTest })

		cb.mu.Lock()
		total, _ := cb.getStatsLocked()
		cb.mu.Unlock()

		if total != 1 {
			t.Errorf("expected 1 call in window, got %d", total)
		}
	})

	t.Run("AdvancePartial", func(t *testing.T) {
		configPartial := Config{
			WindowType:     WindowTimeBased,
			WindowDuration: 400 * time.Millisecond, // bucket = 40ms
			MinimumCalls:   1,
		}
		cb = New(configPartial)
		_ = cb.Execute(ctx, func() error { return nil })
		time.Sleep(120 * time.Millisecond) // Advance ~3 buckets
		_ = cb.Execute(ctx, func() error { return nil })

		cb.mu.Lock()
		total, _ := cb.getStatsLocked()
		cb.mu.Unlock()

		if total != 2 {
			t.Errorf("expected 2 calls, got %d", total)
		}
	})
}

func TestBreaker_CountBased_RingFull(t *testing.T) {
	t.Parallel()
	config := Config{
		WindowType: WindowCountBased,
		WindowSize: 3,
	}
	cb := New(config)
	ctx := context.Background()

	// Fill it: S, F, S
	_ = cb.Execute(ctx, func() error { return nil })
	_ = cb.Execute(ctx, func() error { return errors.New("fail") })
	_ = cb.Execute(ctx, func() error { return nil })

	// Stats should be 3 total, 1 failure
	cb.mu.Lock()
	total, failures := cb.getStatsLocked()
	cb.mu.Unlock()

	if total != 3 || failures != 1 {
		t.Errorf("expected 3/1, got %d/%d", total, failures)
	}

	// Overwrite first S with F: F, F, S
	_ = cb.Execute(ctx, func() error { return errors.New("fail") })
	cb.mu.Lock()
	total, failures = cb.getStatsLocked()
	cb.mu.Unlock()

	if total != 3 || failures != 2 {
		t.Errorf("expected 3/2, got %d/%d", total, failures)
	}

	// Overwrite second F with S: F, S, S
	_ = cb.Execute(ctx, func() error { return nil })
	cb.mu.Lock()
	total, failures = cb.getStatsLocked()
	cb.mu.Unlock()

	if total != 3 || failures != 1 {
		t.Errorf("expected 3/1, got %d/%d", total, failures)
	}
}

func TestBreaker_Concurrency(t *testing.T) {
	t.Parallel()
	cb := New(Config{
		WindowType:           WindowCountBased,
		WindowSize:           100,
		FailureRateThreshold: 50.0,
		MinimumCalls:         10,
	})

	const goroutines = 10
	const iterations = 100
	var wg sync.WaitGroup
	wg.Add(goroutines)
	ctx := context.Background()

	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				_ = cb.Execute(ctx, func() error {
					if j%2 == 0 {
						return errors.New("error")
					}
					return nil
				})
			}
		}()
	}

	wg.Wait()
}

func TestBreaker_Reset(t *testing.T) {
	t.Parallel()
	cb := New(Config{
		WindowSize:   10,
		MinimumCalls: 1,
	})
	ctx := context.Background()

	_ = cb.Execute(ctx, func() error { return errors.New("fail") })
	if cb.State() != StateOpen {
		t.Error("expected open")
	}

	cb.Reset()
	if cb.State() != StateClosed {
		t.Error("expected closed after reset")
	}

	cb.mu.Lock()
	total, _ := cb.getStatsLocked()
	cb.mu.Unlock()
	if total != 0 {
		t.Errorf("expected empty window, got %d", total)
	}
}

func TestBreaker_Context(t *testing.T) {
	t.Parallel()
	cb := New(Config{})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := cb.Execute(ctx, func() error { return nil })
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context canceled error, got %v", err)
	}
}
