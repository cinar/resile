// Copyright (c) 2026 Onur Cinar.
// The source code is provided under MIT License.
// https://github.com/cinar/resile

package resile

import "sync"

// AdaptiveBucket implements a client-side token bucket rate limiter for retries.
// It protects downstream services from thundering herds by depleting tokens
// when retries occur and refilling them on successful responses.
// This allows a fleet of clients to quickly cut off traffic to a degraded
// system, providing macro-level protection.
type AdaptiveBucket struct {
	mu            sync.Mutex
	tokens        float64
	maxCapacity   float64
	retryCost     float64
	successRefill float64
}

// NewAdaptiveBucket creates a new AdaptiveBucket with the specified configuration.
// A common AWS-style configuration is maxCapacity=500, retryCost=5, successRefill=1.
func NewAdaptiveBucket(maxCapacity, retryCost, successRefill float64) *AdaptiveBucket {
	return &AdaptiveBucket{
		tokens:        maxCapacity,
		maxCapacity:   maxCapacity,
		retryCost:     retryCost,
		successRefill: successRefill,
	}
}

// DefaultAdaptiveBucket creates a new AdaptiveBucket with standard defaults
// (max capacity: 500, retry cost: 5, success refill: 1).
func DefaultAdaptiveBucket() *AdaptiveBucket {
	return NewAdaptiveBucket(500, 5, 1)
}

// AcquireRetryToken attempts to consume a token for a retry.
// Returns true if a token was successfully consumed, false if the bucket is empty.
func (b *AdaptiveBucket) AcquireRetryToken() bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.tokens >= b.retryCost {
		b.tokens -= b.retryCost
		return true
	}
	return false
}

// AddSuccessToken adds a fraction of a token back to the bucket upon a successful response.
func (b *AdaptiveBucket) AddSuccessToken() {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.tokens += b.successRefill
	if b.tokens > b.maxCapacity {
		b.tokens = b.maxCapacity
	}
}
