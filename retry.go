// Copyright (c) 2026 Onur Cinar.
// The source code is provided under MIT License.
// https://github.com/cinar/resile

package resile

import (
	"context"
	"errors"
	"time"

	"github.com/cinar/resile/circuit"
)

// Retryer defines the interface for executing actions with resilience.
type Retryer interface {
	// Do executes a function that returns a value and an error.
	Do(ctx context.Context, action func(context.Context) (any, error)) (any, error)
	// DoErr executes a function that returns only an error.
	DoErr(ctx context.Context, action func(context.Context) error) error
}

// Config represents the configuration for the retry execution.
type Config struct {
	Name           string
	MaxAttempts    uint
	BaseDelay      time.Duration
	MaxDelay       time.Duration
	Backoff        Backoff
	Policy         *retryPolicy
	Instrumenter   Instrumenter
	CircuitBreaker *circuit.Breaker
}

// Do executes an action with retry logic using the provided options.
// This generic function handles functions returning (T, error).
func Do[T any](ctx context.Context, action func(context.Context) (T, error), opts ...Option) (T, error) {
	return DoState(ctx, func(innerCtx context.Context, _ RetryState) (T, error) {
		return action(innerCtx)
	}, opts...)
}

// DoState executes a stateful action with retry logic using the provided options.
// The RetryState is passed to the closure, allowing it to adapt to failure history.
func DoState[T any](ctx context.Context, action func(context.Context, RetryState) (T, error), opts ...Option) (T, error) {
	c := DefaultConfig()
	for _, opt := range opts {
		opt(c)
	}

	var result T
	err := c.execute(ctx, func(innerCtx context.Context, state RetryState) error {
		var innerErr error
		result, innerErr = action(innerCtx, state)
		return innerErr
	})

	return result, err
}

// DoErr executes an action with retry logic using the provided options.
// This function handles functions returning only error.
func DoErr(ctx context.Context, action func(context.Context) error, opts ...Option) error {
	return DoErrState(ctx, func(innerCtx context.Context, _ RetryState) error {
		return action(innerCtx)
	}, opts...)
}

// DoErrState executes a stateful action with retry logic using the provided options.
func DoErrState(ctx context.Context, action func(context.Context, RetryState) error, opts ...Option) error {
	c := DefaultConfig()
	for _, opt := range opts {
		opt(c)
	}
	return c.execute(ctx, action)
}

// New returns a new Retryer pre-configured with the provided options.
// This is useful for dependency injection and reusable resilience clients.
func New(opts ...Option) Retryer {
	c := DefaultConfig()
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// Do satisfies the Retryer interface. Note: returns any for interface compliance.
func (c *Config) Do(ctx context.Context, action func(context.Context) (any, error)) (any, error) {
	return Do(ctx, action, func(config *Config) {
		*config = *c // Copy existing config
	})
}

// DoErr satisfies the Retryer interface.
func (c *Config) DoErr(ctx context.Context, action func(context.Context) error) error {
	return DoErr(ctx, action, func(config *Config) {
		*config = *c // Copy existing config
	})
}

// DefaultConfig returns a reasonable production-grade configuration.
func DefaultConfig() *Config {
	return &Config{
		MaxAttempts: 5,
		BaseDelay:   100 * time.Millisecond,
		MaxDelay:    30 * time.Second,
		Backoff:     NewFullJitter(100*time.Millisecond, 30*time.Second),
		Policy:      &retryPolicy{},
	}
}

// doAction is an internal type to support both stateless and stateful execution.
type doAction func(context.Context, RetryState) error

// execute executes the action with the provided configuration and context.
func (c *Config) execute(ctx context.Context, action doAction) error {
	var lastErr error
	start := time.Now()

	for attempt := uint(0); attempt < c.MaxAttempts; attempt++ {
		state := RetryState{
			Name:          c.Name,
			Attempt:       attempt,
			MaxAttempts:   c.MaxAttempts,
			LastError:     lastErr,
			TotalDuration: time.Since(start),
		}

		// If a circuit breaker is configured, wrap the attempt execution.
		if c.CircuitBreaker != nil {
			err := c.CircuitBreaker.Execute(func() error {
				return c.executeOnce(ctx, action, state)
			})
			lastErr = err
		} else {
			lastErr = c.executeOnce(ctx, action, state)
		}

		// Success terminates the loop.
		if lastErr == nil {
			return nil
		}

		// Check for circuit open to avoid retries.
		if errors.Is(lastErr, circuit.ErrCircuitOpen) {
			return lastErr
		}

		// Check if we should retry based on the error policy.
		if !c.Policy.shouldRetry(lastErr) {
			return lastErr
		}

		// If this was the last attempt, don't sleep.
		if attempt+1 >= c.MaxAttempts {
			break
		}

		// Calculate the next delay.
		delay := c.Backoff.Next(attempt)

		// Check for Retry-After override.
		var retryAfter RetryAfterError
		if errors.As(lastErr, &retryAfter) {
			delay = retryAfter.RetryAfter()
		}

		// Sleep safely with context awareness.
		if err := sleep(ctx, delay); err != nil {
			// Context canceled during sleep.
			return errors.Join(lastErr, err)
		}
	}

	return lastErr
}

// executeOnce performs a single attempt and triggers instrumentation.
func (c *Config) executeOnce(ctx context.Context, action doAction, state RetryState) error {
	// Invoke instrumentation before attempt.
	if c.Instrumenter != nil {
		ctx = c.Instrumenter.BeforeAttempt(ctx, state)
	}

	// Execute the user closure.
	err := action(ctx, state)

	// Update state for AfterAttempt.
	state.LastError = err
	if c.Instrumenter != nil {
		c.Instrumenter.AfterAttempt(ctx, state)
	}

	return err
}

type contextKey string

const bypassDelayKey contextKey = "resile_bypass_delay"

// WithTestingBypass returns a new context that signals the retry loop to skip all sleep delays.
// This is intended for use in unit tests to prevent CI pipelines from being slowed down by backoff.
func WithTestingBypass(ctx context.Context) context.Context {
	return context.WithValue(ctx, bypassDelayKey, true)
}

// sleep provides a memory-safe, context-aware delay using time.NewTimer.
func sleep(ctx context.Context, delay time.Duration) error {
	if delay <= 0 {
		return nil
	}

	// Check for testing bypass.
	if bypass, ok := ctx.Value(bypassDelayKey).(bool); ok && bypass {
		return nil
	}

	timer := time.NewTimer(delay)
	defer func() {
		if !timer.Stop() {
			select {
			case <-timer.C:
			default:
			}
		}
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
