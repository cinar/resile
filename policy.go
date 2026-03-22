// Copyright (c) 2026 Onur Cinar.
// The source code is provided under MIT License.
// https://github.com/cinar/resile

package resile

import (
	"context"
	"errors"
	"time"
)

// fatalError is a private wrapper to indicate an error should not be retried.
type fatalError struct {
	err error
}

func (f *fatalError) Error() string {
	return f.err.Error()
}

func (f *fatalError) Unwrap() error {
	return f.err
}

// FatalError wraps an error to indicate that the retry loop should terminate immediately.
func FatalError(err error) error {
	if err == nil {
		return nil
	}
	return &fatalError{err: err}
}

// isFatal checks if an error (or any error in its chain) is a fatal error.
func isFatal(err error) bool {
	var f *fatalError
	return errors.As(err, &f)
}

// RetryAfterError is implemented by errors that specify how long to wait before retrying.
// This is commonly used with HTTP 429 (Too Many Requests) or 503 (Service Unavailable)
// to respect Retry-After headers. It also supports pushback signals to cancel all retries.
type RetryAfterError interface {
	error
	RetryAfter() time.Duration
	CancelAllRetries() bool
}

// retryPolicy defines the logic to determine if an error is retriable.
type retryPolicy struct {
	retryIf     error
	retryIfFunc func(error) bool
}

// shouldRetry evaluates the error against the configured policy.
func (p *retryPolicy) shouldRetry(err error) bool {
	if err == nil {
		return false
	}

	// Fatal errors always terminate the loop.
	if isFatal(err) {
		return false
	}

	// If no specific policy is set, we default to retrying all non-fatal errors.
	if p.retryIf == nil && p.retryIfFunc == nil {
		return true
	}

	// Check if the error matches a specific target.
	if p.retryIf != nil && errors.Is(err, p.retryIf) {
		return true
	}

	// Check if the error matches a custom function.
	if p.retryIfFunc != nil && p.retryIfFunc(err) {
		return true
	}

	return false
}

// Policy represents a composed resilience strategy.
// It is thread-safe and reusable across multiple calls.
type Policy struct {
	config *Config
}

// NewPolicy creates a new Policy with the given options.
// The order of options determines the execution order from outermost to innermost.
// Example: NewPolicy(WithFallback(f), WithRetry(3)) results in Fallback(Retry(Action)).
func NewPolicy(opts ...Option) *Policy {
	c := DefaultConfig()
	c.pipeline = make([]middleware, 0)
	for _, opt := range opts {
		opt(c)
	}
	return &Policy{config: c}
}

// Do executes an action within the resilience policy.
func (p *Policy) Do(ctx context.Context, action func(context.Context) (any, error)) (any, error) {
	return p.config.Do(ctx, action)
}

// DoErr executes an action within the resilience policy.
func (p *Policy) DoErr(ctx context.Context, action func(context.Context) error) error {
	return p.config.DoErr(ctx, action)
}

// middleware defines a function that wraps a doAction with additional resilience logic.
type middleware func(doAction) doAction
