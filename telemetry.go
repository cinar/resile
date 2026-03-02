// Copyright (c) 2026 Onur Cinar.
// The source code is provided under MIT License.
// https://github.com/cinar/resile

package resile

import (
	"context"
	"time"
)

// RetryState encapsulates the current state of a retry execution.
type RetryState struct {
	// Name is the optional identifier for the operation being retried.
	Name string
	// Attempt is the current 0-indexed retry iteration.
	Attempt uint
	// MaxAttempts is the maximum number of attempts allowed.
	MaxAttempts uint
	// LastError is the error encountered in the previous attempt.
	LastError error
	// TotalDuration is the cumulative time spent across all attempts and sleeps.
	TotalDuration time.Duration
	// NextDelay is the duration to be slept before the next attempt.
	NextDelay time.Duration
}

// Instrumenter defines the lifecycle hooks for monitoring retry executions.
// It is a zero-dependency interface to allow custom implementations for
// logging, metrics, and tracing.
type Instrumenter interface {
	// BeforeAttempt is called before each execution attempt.
	// It can return a new context (e.g., to inject trace spans).
	BeforeAttempt(ctx context.Context, state RetryState) context.Context
	// AfterAttempt is called after each execution attempt.
	AfterAttempt(ctx context.Context, state RetryState)
}