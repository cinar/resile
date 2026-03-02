// Copyright (c) 2026 Onur Cinar.
// The source code is provided under MIT License.
// https://github.com/cinar/resile

package resile

import (
	"math"
	"math/rand/v2"
	"time"
)

// Backoff defines the interface for temporal distribution of retries.
type Backoff interface {
	// Next calculates the duration to wait before the specified attempt.
	// The attempt parameter is 0-indexed.
	Next(attempt uint) time.Duration
}

// fullJitter implements the AWS "Full Jitter" algorithm.
// See: https://aws.amazon.com/blogs/architecture/exponential-backoff-and-jitter/
type fullJitter struct {
	base time.Duration
	cap  time.Duration
}

// NewFullJitter returns a Backoff implementation using the AWS Full Jitter algorithm.
// The base duration dictates the initial delay, and the cap defines the absolute maximum.
func NewFullJitter(base, cap time.Duration) Backoff {
	return &fullJitter{
		base: base,
		cap:  cap,
	}
}

// Next calculates the next sleep duration.
// Formula: delay = random(0, min(cap, base * 2^attempt))
func (f *fullJitter) Next(attempt uint) time.Duration {
	// Calculate the exponential maximum for this attempt.
	// We use float64 to prevent overflow during the power operation.
	exp := math.Pow(2, float64(attempt))
	maxDelay := float64(f.base) * exp

	// Ensure the calculated maximum does not exceed the cap.
	if maxDelay > float64(f.cap) {
		maxDelay = float64(f.cap)
	}

	if maxDelay <= 0 {
		return 0
	}

	// Generate a random duration between 0 and the calculated maximum.
	// math/rand/v2 Int64N is used for better distribution.
	randomDelay := rand.Int64N(int64(maxDelay))

	return time.Duration(randomDelay)
}
