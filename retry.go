// Copyright (c) 2026 Onur Cinar.
// The source code is provided under MIT License.
// https://github.com/cinar/resile

package resile

import (
	"context"
	"errors"
	"runtime/debug"
	"sync"
	"time"

	"github.com/cinar/resile/circuit"
)

// Retryer defines the interface for executing actions with resilience.
type Retryer interface {
	// Do executes a function that returns a value and an error.
	Do(ctx context.Context, action func(context.Context) (any, error)) (any, error)
	// DoHedged executes a function using speculative retries (hedging).
	DoHedged(ctx context.Context, action func(context.Context) (any, error)) (any, error)
	// DoErr executes a function that returns only an error.
	DoErr(ctx context.Context, action func(context.Context) error) error
	// DoErrHedged executes a function using speculative retries (hedging).
	DoErrHedged(ctx context.Context, action func(context.Context) error) error
}

// PanicError represents a recovered panic during execution.
type PanicError struct {
	Value      any
	StackTrace string
}

// Error implements the error interface.
func (p *PanicError) Error() string {
	return "panic: " + p.StackTrace
}

// Config represents the configuration for the retry execution.
type Config struct {
	Name            string
	MaxAttempts     uint
	BaseDelay       time.Duration
	MaxDelay        time.Duration
	HedgingDelay    time.Duration
	Backoff         Backoff
	Policy          *retryPolicy
	Instrumenter    Instrumenter
	CircuitBreaker  *circuit.Breaker
	Fallback        any
	AdaptiveBucket  *AdaptiveBucket
	RecoverPanics   bool
	Bulkhead        *Bulkhead
	Timeout         time.Duration
	RateLimiter     *RateLimiter
	AdaptiveLimiter *AdaptiveLimiter
	pipeline        []middleware
}

func (c *Config) adaptiveLimiterMiddleware() middleware {
	return func(next doAction) doAction {
		return func(ctx context.Context, state RetryState) error {
			if c.AdaptiveLimiter == nil {
				return next(ctx, state)
			}

			return c.AdaptiveLimiter.Execute(ctx, func() error {
				return next(ctx, state)
			})
		}
	}
}

// Do executes an action with retry logic using the provided options.
// This generic function handles functions returning (T, error).
func Do[T any](ctx context.Context, action func(context.Context) (T, error), opts ...Option) (T, error) {
	return DoState(ctx, func(innerCtx context.Context, _ RetryState) (T, error) {
		return action(innerCtx)
	}, opts...)
}

// DoHedged executes an action using speculative retries (hedging).
// It starts multiple attempts concurrently if previous ones take too long.
func DoHedged[T any](ctx context.Context, action func(context.Context) (T, error), opts ...Option) (T, error) {
	return DoStateHedged(ctx, func(innerCtx context.Context, _ RetryState) (T, error) {
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

	if err != nil && c.Fallback != nil {
		if f, ok := c.Fallback.(func(context.Context, error) (T, error)); ok {
			return f(ctx, err)
		}
	}

	return result, err
}

// DoStateHedged executes a stateful action with speculative retries (hedging).
func DoStateHedged[T any](ctx context.Context, action func(context.Context, RetryState) (T, error), opts ...Option) (T, error) {
	c := DefaultConfig()
	for _, opt := range opts {
		opt(c)
	}

	var result T
	var once sync.Once
	err := c.executeHedged(ctx, func(innerCtx context.Context, state RetryState) error {
		val, innerErr := action(innerCtx, state)
		if innerErr == nil {
			once.Do(func() {
				result = val
			})
		}
		return innerErr
	})

	if err != nil && c.Fallback != nil {
		if f, ok := c.Fallback.(func(context.Context, error) (T, error)); ok {
			return f(ctx, err)
		}
	}

	return result, err
}

// DoErr executes an action with retry logic using the provided options.
// This function handles functions returning only error.
func DoErr(ctx context.Context, action func(context.Context) error, opts ...Option) error {
	return DoErrState(ctx, func(innerCtx context.Context, _ RetryState) error {
		return action(innerCtx)
	}, opts...)
}

// DoErrHedged executes an action using speculative retries (hedging).
func DoErrHedged(ctx context.Context, action func(context.Context) error, opts ...Option) error {
	return DoErrStateHedged(ctx, func(innerCtx context.Context, _ RetryState) error {
		return action(innerCtx)
	}, opts...)
}

// DoErrState executes a stateful action with retry logic using the provided options.
func DoErrState(ctx context.Context, action func(context.Context, RetryState) error, opts ...Option) error {
	c := DefaultConfig()
	for _, opt := range opts {
		opt(c)
	}
	err := c.execute(ctx, action)

	if err != nil && c.Fallback != nil {
		if f, ok := c.Fallback.(func(context.Context, error) error); ok {
			return f(ctx, err)
		}
	}

	return err
}

// DoErrStateHedged executes a stateful action with speculative retries (hedging).
func DoErrStateHedged(ctx context.Context, action func(context.Context, RetryState) error, opts ...Option) error {
	c := DefaultConfig()
	for _, opt := range opts {
		opt(c)
	}
	err := c.executeHedged(ctx, action)

	if err != nil && c.Fallback != nil {
		if f, ok := c.Fallback.(func(context.Context, error) error); ok {
			return f(ctx, err)
		}
	}

	return err
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

// DoHedged satisfies the Retryer interface.
func (c *Config) DoHedged(ctx context.Context, action func(context.Context) (any, error)) (any, error) {
	return DoHedged(ctx, action, func(config *Config) {
		*config = *c // Copy existing config
	})
}

// DoErr satisfies the Retryer interface.
func (c *Config) DoErr(ctx context.Context, action func(context.Context) error) error {
	return DoErr(ctx, action, func(config *Config) {
		*config = *c // Copy existing config
	})
}

// DoErrHedged satisfies the Retryer interface.
func (c *Config) DoErrHedged(ctx context.Context, action func(context.Context) error) error {
	return DoErrHedged(ctx, action, func(config *Config) {
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
	if len(c.pipeline) == 0 {
		c.buildDefaultPipeline()
	}

	h := action
	for i := len(c.pipeline) - 1; i >= 0; i-- {
		h = c.pipeline[i](h)
	}

	return h(ctx, RetryState{
		Name:        c.Name,
		MaxAttempts: c.MaxAttempts,
	})
}

// buildDefaultPipeline builds the legacy hardcoded pipeline order.
// Order: Bulkhead -> Retry ( CircuitBreaker -> Instrumenter -> PanicRecovery )
func (c *Config) buildDefaultPipeline() {
	if c.RateLimiter != nil {
		c.pipeline = append(c.pipeline, c.rateLimiterMiddleware())
	}

	if c.AdaptiveLimiter != nil {
		c.pipeline = append(c.pipeline, c.adaptiveLimiterMiddleware())
	}

	if c.Bulkhead != nil {
		c.pipeline = append(c.pipeline, c.bulkheadMiddleware())
	}

	// Retry is the primary driver in the legacy model.
	c.pipeline = append(c.pipeline, c.retryMiddleware())

	if c.Timeout > 0 {
		c.pipeline = append(c.pipeline, c.timeoutMiddleware(c.Timeout))
	}

	if c.CircuitBreaker != nil {
		c.pipeline = append(c.pipeline, c.circuitBreakerMiddleware())
	}

	c.pipeline = append(c.pipeline, c.instrumenterMiddleware())

	if c.RecoverPanics {
		c.pipeline = append(c.pipeline, c.panicRecoveryMiddleware())
	}
}

func (c *Config) rateLimiterMiddleware() middleware {
	return func(next doAction) doAction {
		return func(ctx context.Context, state RetryState) error {
			if c.RateLimiter == nil {
				return next(ctx, state)
			}
			if !c.RateLimiter.Acquire(ctx) {
				if c.Instrumenter != nil {
					c.Instrumenter.OnRateLimitExceeded(ctx, state)
				}
				return ErrRateLimitExceeded
			}
			return next(ctx, state)
		}
	}
}

func (c *Config) timeoutMiddleware(timeout time.Duration) middleware {
	return func(next doAction) doAction {
		return func(ctx context.Context, state RetryState) error {
			ctx, cancel := context.WithTimeout(ctx, timeout)
			defer cancel()
			return next(ctx, state)
		}
	}
}

func (c *Config) retryMiddleware() middleware {
	return func(next doAction) doAction {
		return func(ctx context.Context, state RetryState) error {
			var lastErr error
			start := time.Now()

			for attempt := uint(0); attempt < c.MaxAttempts; attempt++ {
				state.Attempt = attempt
				state.LastError = lastErr
				state.TotalDuration = time.Since(start)

				lastErr = next(ctx, state)

				// Success terminates the loop.
				if lastErr == nil {
					if c.AdaptiveBucket != nil {
						c.AdaptiveBucket.AddSuccessToken()
					}
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

				// Check adaptive bucket before committing to next attempt.
				if c.AdaptiveBucket != nil && !c.AdaptiveBucket.AcquireRetryToken() {
					break
				}

				// Calculate the next delay.
				delay := c.Backoff.Next(attempt)

				// Check for Retry-After override.
				var retryAfter RetryAfterError
				if errors.As(lastErr, &retryAfter) {
					if retryAfter.CancelAllRetries() {
						return lastErr
					}
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
	}
}

func (c *Config) circuitBreakerMiddleware() middleware {
	return func(next doAction) doAction {
		return func(ctx context.Context, state RetryState) error {
			if c.CircuitBreaker == nil {
				return next(ctx, state)
			}
			return c.CircuitBreaker.Execute(func() error {
				return next(ctx, state)
			})
		}
	}
}

func (c *Config) bulkheadMiddleware() middleware {
	return func(next doAction) doAction {
		return func(ctx context.Context, state RetryState) error {
			if c.Bulkhead == nil {
				return next(ctx, state)
			}
			err := c.Bulkhead.Execute(ctx, func() error {
				return next(ctx, state)
			})
			if errors.Is(err, ErrBulkheadFull) && c.Instrumenter != nil {
				c.Instrumenter.OnBulkheadFull(ctx, state)
			}
			return err
		}
	}
}

func (c *Config) instrumenterMiddleware() middleware {
	return func(next doAction) doAction {
		return func(ctx context.Context, state RetryState) error {
			if c.Instrumenter == nil {
				return next(ctx, state)
			}
			ctx = c.Instrumenter.BeforeAttempt(ctx, state)
			err := next(ctx, state)
			state.LastError = err
			c.Instrumenter.AfterAttempt(ctx, state)
			return err
		}
	}
}

func (c *Config) panicRecoveryMiddleware() middleware {
	return func(next doAction) doAction {
		return func(ctx context.Context, state RetryState) (err error) {
			defer func() {
				if r := recover(); r != nil {
					err = &PanicError{
						Value:      r,
						StackTrace: string(debug.Stack()),
					}
				}
			}()
			return next(ctx, state)
		}
	}
}

func (c *Config) fallbackMiddleware() middleware {
	return func(next doAction) doAction {
		return func(ctx context.Context, state RetryState) error {
			err := next(ctx, state)
			if err != nil && c.Fallback != nil {
				// Fallback for DoErr
				if f, ok := c.Fallback.(func(context.Context, error) error); ok {
					return f(ctx, err)
				}
				// Note: Fallback for value-returning Do is handled at the top-level
				// in DoState to preserve type safety without reflection.
			}
			return err
		}
	}
}

// executeHedged executes the action using speculative retries (hedging).
func (c *Config) executeHedged(ctx context.Context, action doAction) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	results := make(chan error, c.MaxAttempts)
	var lastErr error
	var inFlight int
	var attemptsStarted uint

	start := time.Now()

	// Build attempt pipeline (everything except the outer retry loop).
	// For hedging, the retry loop is handled by executeHedged itself.
	h := action
	if c.RateLimiter != nil {
		h = c.rateLimiterMiddleware()(h)
	}
	if c.Bulkhead != nil {
		h = c.bulkheadMiddleware()(h)
	}
	if c.Timeout > 0 {
		h = c.timeoutMiddleware(c.Timeout)(h)
	}
	if c.CircuitBreaker != nil {
		h = c.circuitBreakerMiddleware()(h)
	}
	h = c.instrumenterMiddleware()(h)
	if c.RecoverPanics {
		h = c.panicRecoveryMiddleware()(h)
	}

	for {
		if attemptsStarted < c.MaxAttempts {
			canStart := true
			if attemptsStarted > 0 && c.AdaptiveBucket != nil {
				canStart = c.AdaptiveBucket.AcquireRetryToken()
			}

			if canStart {
				attempt := attemptsStarted
				attemptsStarted++
				inFlight++
				go func(a uint) {
					state := RetryState{
						Name:          c.Name,
						Attempt:       a,
						MaxAttempts:   c.MaxAttempts,
						TotalDuration: time.Since(start),
						NextDelay:     c.HedgingDelay,
					}

					err := h(ctx, state)

					select {
					case results <- err:
					case <-ctx.Done():
					}
				}(attempt)
			} else {
				attemptsStarted = c.MaxAttempts
			}
		}

		if attemptsStarted >= c.MaxAttempts && inFlight == 0 {
			break
		}

		var timerCh <-chan time.Time
		var timer *time.Timer
		if attemptsStarted < c.MaxAttempts {
			timer = time.NewTimer(c.HedgingDelay)
			timerCh = timer.C
		}

		select {
		case <-ctx.Done():
			if timer != nil {
				timer.Stop()
			}
			return ctx.Err()
		case err := <-results:
			if timer != nil {
				timer.Stop()
			}
			inFlight--
			if err == nil {
				cancel()
				if c.AdaptiveBucket != nil {
					c.AdaptiveBucket.AddSuccessToken()
				}
				return nil
			}
			lastErr = err

			// Check for circuit open to avoid further retries.
			if errors.Is(lastErr, circuit.ErrCircuitOpen) {
				cancel()
				return lastErr
			}

			// Check for pushback signal to cancel all retries.
			var retryAfter RetryAfterError
			if errors.As(lastErr, &retryAfter) && retryAfter.CancelAllRetries() {
				cancel()
				return lastErr
			}

			// If error is not retryable, cancel all and return.
			if !c.Policy.shouldRetry(lastErr) {
				cancel()
				return lastErr
			}

			// If no more attempts are in-flight, start next one immediately.
			if inFlight == 0 && attemptsStarted < c.MaxAttempts {
				continue
			}
		case <-timerCh:
			// Hedging delay reached, start next attempt if available.
		}
	}

	return lastErr
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
