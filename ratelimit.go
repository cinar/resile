// Copyright (c) 2026 Onur Cinar.
// The source code is provided under MIT License.
// https://github.com/cinar/resile

package resile

import (
	"context"
	"errors"
	"sync"
	"time"
)

// ErrRateLimitExceeded is returned when the rate limit is exceeded.
var ErrRateLimitExceeded = errors.New("rate limit exceeded")

// RateLimiter implements a time-based token bucket rate limiter.
type RateLimiter struct {
	mu           sync.Mutex
	limit        float64
	tokens       float64
	interval     time.Duration
	lastRefillAt time.Time
}

// NewRateLimiter creates a new RateLimiter with the specified limit and interval.
// Example: NewRateLimiter(100, time.Second) allows 100 requests per second.
func NewRateLimiter(limit float64, interval time.Duration) *RateLimiter {
	return &RateLimiter{
		limit:        limit,
		tokens:       limit,
		interval:     interval,
		lastRefillAt: time.Now(),
	}
}

// Acquire attempts to consume a token from the bucket.
// Returns true if a token was acquired, false otherwise.
func (rl *RateLimiter) Acquire(ctx context.Context) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(rl.lastRefillAt)
	
	// Refill tokens based on elapsed time.
	refill := float64(elapsed) / float64(rl.interval) * rl.limit
	if refill > 0 {
		rl.tokens += refill
		if rl.tokens > rl.limit {
			rl.tokens = rl.limit
		}
		rl.lastRefillAt = now
	}

	if rl.tokens >= 1 {
		rl.tokens -= 1
		return true
	}

	return false
}
