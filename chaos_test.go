// Copyright (c) 2026 Onur Cinar.
// The source code is provided under MIT License.
// https://github.com/cinar/resile

package resile

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/cinar/resile/chaos"
)

type mockRNG struct {
	val float64
}

func (m *mockRNG) Float64() float64 {
	return m.val
}

func TestChaosErrorInjection(t *testing.T) {
	t.Parallel()

	injectedErr := errors.New("chaos error")
	cfg := chaos.Config{
		ErrorProbability: 1.0,
		InjectedError:    injectedErr,
	}
	// Use mock RNG to guarantee injection
	injector := chaos.NewInjectorWithRNG(cfg, &mockRNG{val: 0.0})

	err := DoErr(context.Background(), func(ctx context.Context) error {
		return nil
	}, func(c *Config) {
		c.Chaos = injector
	})

	if !errors.Is(err, injectedErr) {
		t.Errorf("expected error %v, got %v", injectedErr, err)
	}

	// Test no injection with 0 probability
	cfg.ErrorProbability = 0.0
	injector = chaos.NewInjectorWithRNG(cfg, &mockRNG{val: 0.5})
	err = DoErr(context.Background(), func(ctx context.Context) error {
		return nil
	}, func(c *Config) {
		c.Chaos = injector
	})
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestChaosLatencyInjection(t *testing.T) {
	t.Parallel()

	latency := 50 * time.Millisecond
	cfg := chaos.Config{
		LatencyProbability: 1.0,
		LatencyDuration:    latency,
	}
	injector := chaos.NewInjectorWithRNG(cfg, &mockRNG{val: 0.0})

	start := time.Now()
	err := DoErr(context.Background(), func(ctx context.Context) error {
		return nil
	}, func(c *Config) {
		c.Chaos = injector
	})
	duration := time.Since(start)

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if duration < latency {
		t.Errorf("expected latency at least %v, got %v", latency, duration)
	}

	// Test no latency with 0 probability
	cfg.LatencyProbability = 0.0
	injector = chaos.NewInjectorWithRNG(cfg, &mockRNG{val: 0.5})
	start = time.Now()
	err = DoErr(context.Background(), func(ctx context.Context) error {
		return nil
	}, func(c *Config) {
		c.Chaos = injector
	})
	duration = time.Since(start)
	if duration >= latency {
		t.Errorf("expected no latency, got %v", duration)
	}
}

func TestChaosContextCancellation(t *testing.T) {
	t.Parallel()

	latency := 1 * time.Second
	cfg := chaos.Config{
		LatencyProbability: 1.0,
		LatencyDuration:    latency,
	}
	injector := chaos.NewInjectorWithRNG(cfg, &mockRNG{val: 0.0})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	err := DoErr(ctx, func(ctx context.Context) error {
		return nil
	}, func(c *Config) {
		c.Chaos = injector
	})

	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("expected context deadline exceeded, got %v", err)
	}
}

func TestChaosIntegrationWithRetry(t *testing.T) {
	t.Parallel()

	injectedErr := errors.New("chaos error")
	cfg := chaos.Config{
		ErrorProbability: 0.5,
		InjectedError:    injectedErr,
	}

	// Use a mock RNG that returns 0.0 then 1.0 to simulate one failure then success
	type sequentialRNG struct {
		vals []float64
		idx  int
	}
	srng := &sequentialRNG{vals: []float64{0.0, 0.9}}
	rngFunc := func() float64 {
		v := srng.vals[srng.idx]
		srng.idx = (srng.idx + 1) % len(srng.vals)
		return v
	}

	mock := &customRNG{f: rngFunc}
	injector := chaos.NewInjectorWithRNG(cfg, mock)

	attempts := 0
	err := DoErr(WithTestingBypass(context.Background()), func(ctx context.Context) error {
		attempts++
		return nil
	}, WithRetry(3), func(c *Config) {
		c.Chaos = injector
	})

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if attempts != 1 {
		// Note: The action is called only after successful chaos check.
		// Attempt 1: Chaos injects error. Action not called.
		// Attempt 2: Chaos passes. Action called.
		t.Errorf("expected 1 attempt, got %d", attempts)
	}
	// Wait, if chaos injects error, the retry middleware should catch it and retry.
	// In the first attempt, chaos.Execute returns injectedErr.
	// retryMiddleware catches it, sleeps, and retries.
	// In the second attempt, chaos.Execute calls action, which returns nil.
}

type customRNG struct {
	f func() float64
}

func (c *customRNG) Float64() float64 {
	return c.f()
}

func TestWithChaosOption(t *testing.T) {
	t.Parallel()

	injectedErr := errors.New("chaos error")
	err := DoErr(context.Background(), func(ctx context.Context) error {
		return nil
	}, WithChaos(chaos.Config{
		ErrorProbability: 1.0,
		InjectedError:    injectedErr,
	}))

	// Note: WithChaos uses default RNG which is random, but with 1.0 prob it should always fail.
	if !errors.Is(err, injectedErr) {
		t.Errorf("expected error %v, got %v", injectedErr, err)
	}
}

func TestChaosHedged(t *testing.T) {
	t.Parallel()

	injectedErr := errors.New("chaos error")
	cfg := chaos.Config{
		ErrorProbability: 1.0,
		InjectedError:    injectedErr,
	}

	err := DoErrHedged(context.Background(), func(ctx context.Context) error {
		return nil
	}, WithChaos(cfg), WithMaxAttempts(2))

	if !errors.Is(err, injectedErr) {
		t.Errorf("expected error %v, got %v", injectedErr, err)
	}
}
