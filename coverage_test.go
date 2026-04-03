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

func TestCoverageOptions(t *testing.T) {
	t.Parallel()

	t.Run("WithBulkheadInstance", func(t *testing.T) {
		bh := NewBulkhead(10)
		p := NewPolicy(WithBulkheadInstance(bh))
		if p.config.Bulkhead != bh {
			t.Errorf("expected bulkhead %v, got %v", bh, p.config.Bulkhead)
		}
	})

	t.Run("WithRateLimiter", func(t *testing.T) {
		p := NewPolicy(WithRateLimiter(10, time.Second))
		if p.config.RateLimiter == nil {
			t.Fatal("expected rate limiter to be set")
		}
		if p.config.RateLimiter.limit != 10 {
			t.Errorf("expected limit 10, got %f", p.config.RateLimiter.limit)
		}
	})

	t.Run("WithRateLimiterInstance", func(t *testing.T) {
		rl := NewRateLimiter(5, time.Second)
		p := NewPolicy(WithRateLimiterInstance(rl))
		if p.config.RateLimiter != rl {
			t.Errorf("expected rate limiter %v, got %v", rl, p.config.RateLimiter)
		}
	})

	t.Run("WithAdaptiveLimiter", func(t *testing.T) {
		p := NewPolicy(WithAdaptiveLimiter())
		if p.config.AdaptiveLimiter == nil {
			t.Fatal("expected adaptive limiter to be set")
		}
	})

	t.Run("WithTimeout", func(t *testing.T) {
		timeout := 500 * time.Millisecond
		p := NewPolicy(WithTimeout(timeout))
		if p.config.Timeout != timeout {
			t.Errorf("expected timeout %v, got %v", timeout, p.config.Timeout)
		}
	})

	t.Run("OptionsWithoutPipeline", func(t *testing.T) {
		// Calling options directly on a config with nil pipeline
		c := DefaultConfig()
		WithMaxAttempts(3)(c)
		WithFallbackErr(func(ctx context.Context, err error) error { return nil })(c)
		WithInstrumenter(nil)(c)
		WithPanicRecovery()(c)

		if c.MaxAttempts != 3 {
			t.Errorf("expected max attempts 3, got %d", c.MaxAttempts)
		}
		WithFallback(func(ctx context.Context, err error) (string, error) { return "", nil })(c)
	})
}

func TestCoveragePanicError(t *testing.T) {
	t.Parallel()
	pe := &PanicError{StackTrace: "test stack"}
	if pe.Error() != "panic: test stack" {
		t.Errorf("expected 'panic: test stack', got %q", pe.Error())
	}
}

func TestCoveragePolicyDo(t *testing.T) {
	t.Parallel()

	p := NewPolicy(WithMaxAttempts(1))
	ctx := context.Background()

	res, err := p.Do(ctx, func(ctx context.Context) (any, error) {
		return "success", nil
	})

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if res != "success" {
		t.Errorf("expected result 'success', got %v", res)
	}
}

func TestCoverageRateLimiter(t *testing.T) {
	t.Parallel()

	t.Run("AcquireSuccess", func(t *testing.T) {
		rl := NewRateLimiter(1, time.Hour)
		if !rl.Acquire(context.Background()) {
			t.Error("expected to acquire token")
		}
	})

	t.Run("AcquireFailure", func(t *testing.T) {
		rl := NewRateLimiter(0, time.Hour)
		// Tokens are initialized to limit. If limit is 0, tokens are 0.
		if rl.Acquire(context.Background()) {
			t.Error("expected not to acquire token")
		}
	})

	t.Run("Refill", func(t *testing.T) {
		rl := NewRateLimiter(1, 10*time.Millisecond)
		if !rl.Acquire(context.Background()) {
			t.Fatal("expected to acquire first token")
		}
		if rl.Acquire(context.Background()) {
			t.Fatal("expected not to acquire second token immediately")
		}
		time.Sleep(20 * time.Millisecond)
		if !rl.Acquire(context.Background()) {
			t.Error("expected to acquire token after refill")
		}
	})
}

func TestCoverageHedged(t *testing.T) {
	t.Parallel()

	t.Run("DoHedged", func(t *testing.T) {
		res, err := DoHedged(context.Background(), func(ctx context.Context) (string, error) {
			return "hedged success", nil
		}, WithMaxAttempts(2), WithHedgingDelay(time.Millisecond))

		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if res != "hedged success" {
			t.Errorf("expected 'hedged success', got %v", res)
		}
	})

	t.Run("DoErrStateHedged", func(t *testing.T) {
		p := NewPolicy(WithMaxAttempts(2), WithHedgingDelay(time.Millisecond))
		err := p.config.DoErrHedged(context.Background(), func(ctx context.Context) error {
			return nil
		})
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
	})

	t.Run("DoStateHedged", func(t *testing.T) {
		p := NewPolicy(WithMaxAttempts(2), WithHedgingDelay(time.Millisecond))
		res, err := p.config.DoHedged(context.Background(), func(ctx context.Context) (any, error) {
			return "success", nil
		})
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if res != "success" {
			t.Errorf("expected 'success', got %v", res)
		}
	})
}

func TestCoverageMiddlewares(t *testing.T) {
	t.Parallel()

	t.Run("RateLimiterMiddlewareExceeded", func(t *testing.T) {
		rl := NewRateLimiter(0, time.Hour)
		p := NewPolicy(WithRateLimiterInstance(rl), WithMaxAttempts(1))
		err := p.DoErr(context.Background(), func(ctx context.Context) error {
			return nil
		})
		if !errors.Is(err, ErrRateLimitExceeded) {
			t.Errorf("expected ErrRateLimitExceeded, got %v", err)
		}
	})

	t.Run("TimeoutMiddleware", func(t *testing.T) {
		p := NewPolicy(WithTimeout(10*time.Millisecond), WithMaxAttempts(1))
		err := p.DoErr(context.Background(), func(ctx context.Context) error {
			time.Sleep(50 * time.Millisecond)
			return ctx.Err()
		})
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Errorf("expected DeadlineExceeded, got %v", err)
		}
	})

	t.Run("FallbackMiddlewareDoErr", func(t *testing.T) {
		fallbackErr := errors.New("fallback error")
		p := NewPolicy(
			WithFallbackErr(func(ctx context.Context, err error) error {
				return fallbackErr
			}),
			WithMaxAttempts(1),
		)
		err := p.DoErr(context.Background(), func(ctx context.Context) error {
			return errors.New("original error")
		})
		if !errors.Is(err, fallbackErr) {
			t.Errorf("expected %v, got %v", fallbackErr, err)
		}
	})

	t.Run("ChaosMiddlewareNil", func(t *testing.T) {
		c := &Config{}
		middleware := c.chaosMiddleware()
		action := middleware(func(ctx context.Context, state RetryState) error {
			return nil
		})
		err := action(context.Background(), RetryState{})
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
	})
}

func TestCoverageChaosMiddleware(t *testing.T) {
	t.Parallel()

	t.Run("ChaosInjectorEnabled", func(t *testing.T) {
		p := NewPolicy(
			WithChaos(chaos.Config{
				ErrorProbability: 1.0,
				InjectedError:    errors.New("chaos error"),
			}),
			WithMaxAttempts(1),
		)
		err := p.DoErr(context.Background(), func(ctx context.Context) error {
			return nil
		})
		if err == nil || err.Error() != "chaos error" {
			t.Errorf("expected chaos error, got %v", err)
		}
	})
}
