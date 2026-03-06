// Copyright (c) 2026 Onur Cinar.
// The source code is provided under MIT License.
// https://github.com/cinar/resile

package resile

import (
	"context"
	"time"

	"github.com/cinar/resile/circuit"
)

// Option defines a functional option for configuring a retry execution.
type Option func(*Config)

// WithFallback sets a function to be called if all retries are exhausted or if the circuit breaker is open.
// T must match the return type of the retry action.
func WithFallback[T any](f func(context.Context, error) (T, error)) Option {
	return func(c *Config) {
		c.Fallback = f
	}
}

// WithFallbackErr sets a fallback function for operations that only return an error.
func WithFallbackErr(f func(context.Context, error) error) Option {
	return func(c *Config) {
		c.Fallback = f
	}
}

// WithName sets the name for the operation. This is used in telemetry labels.
func WithName(name string) Option {
	return func(c *Config) {
		c.Name = name
	}
}

// WithMaxAttempts sets the maximum number of execution attempts.
func WithMaxAttempts(attempts uint) Option {
	return func(c *Config) {
		c.MaxAttempts = attempts
	}
}

// WithBaseDelay sets the initial delay for the backoff algorithm.
func WithBaseDelay(delay time.Duration) Option {
	return func(c *Config) {
		c.BaseDelay = delay
		// If using the default backoff, we update its parameters.
		if fj, ok := c.Backoff.(*fullJitter); ok {
			fj.base = delay
		}
	}
}

// WithMaxDelay sets the maximum delay for the backoff algorithm.
func WithMaxDelay(delay time.Duration) Option {
	return func(c *Config) {
		c.MaxDelay = delay
		if fj, ok := c.Backoff.(*fullJitter); ok {
			fj.cap = delay
		}
	}
}

// WithBackoff sets a custom backoff algorithm.
func WithBackoff(backoff Backoff) Option {
	return func(c *Config) {
		c.Backoff = backoff
	}
}

// WithRetryIf sets a specific error to trigger a retry.
func WithRetryIf(target error) Option {
	return func(c *Config) {
		c.Policy.retryIf = target
	}
}

// WithRetryIfFunc sets a custom function to determine if an error should be retried.
func WithRetryIfFunc(f func(error) bool) Option {
	return func(c *Config) {
		c.Policy.retryIfFunc = f
	}
}

// WithInstrumenter sets a telemetry instrumenter.
func WithInstrumenter(instr Instrumenter) Option {
	return func(c *Config) {
		c.Instrumenter = instr
	}
}

// WithCircuitBreaker integrates a circuit breaker into the retry execution.
func WithCircuitBreaker(cb *circuit.Breaker) Option {
	return func(c *Config) {
		c.CircuitBreaker = cb
	}
}

// WithHedgingDelay sets the delay for speculative retries (hedging).
// If a response doesn't arrive within this delay, another attempt is started concurrently.
func WithHedgingDelay(delay time.Duration) Option {
	return func(c *Config) {
		c.HedgingDelay = delay
	}
}

// WithAdaptiveBucket sets a token bucket for adaptive retries.
// The bucket should be shared across multiple executions to protect downstream services globally.
func WithAdaptiveBucket(bucket *AdaptiveBucket) Option {
	return func(c *Config) {
		c.AdaptiveBucket = bucket
	}
}
