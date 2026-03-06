// Copyright (c) 2026 Onur Cinar.
// The source code is provided under MIT License.
// https://github.com/cinar/resile

package resile

import (
	"testing"
)

func TestAdaptiveBucket_AcquireAndRefill(t *testing.T) {
	t.Parallel()

	// Given a bucket with max 10, cost 5, refill 2
	b := NewAdaptiveBucket(10, 5, 2)

	// Attempt 1: acquire succeeds, tokens 10 -> 5
	if !b.AcquireRetryToken() {
		t.Fatal("expected to acquire retry token")
	}

	// Attempt 2: acquire succeeds, tokens 5 -> 0
	if !b.AcquireRetryToken() {
		t.Fatal("expected to acquire second retry token")
	}

	// Attempt 3: acquire fails, tokens 0 < 5
	if b.AcquireRetryToken() {
		t.Fatal("expected to fail acquiring third retry token")
	}

	// Add success token, tokens 0 -> 2
	b.AddSuccessToken()

	// Still can't acquire, tokens 2 < 5
	if b.AcquireRetryToken() {
		t.Fatal("expected to fail acquiring after insufficient refill")
	}

	// Add two more success tokens, tokens 2 -> 6
	b.AddSuccessToken()
	b.AddSuccessToken()

	// Now can acquire, tokens 6 -> 1
	if !b.AcquireRetryToken() {
		t.Fatal("expected to acquire after sufficient refill")
	}
}

func TestAdaptiveBucket_MaxCapacity(t *testing.T) {
	t.Parallel()

	b := DefaultAdaptiveBucket() // 500 max, 5 cost, 1 refill

	// Attempt to add token when full
	b.AddSuccessToken()

	if b.tokens > 500 {
		t.Fatalf("expected tokens not to exceed max capacity, got %v", b.tokens)
	}

	// Drain slightly
	b.AcquireRetryToken()

	// Refill multiple times past max
	for i := 0; i < 10; i++ {
		b.AddSuccessToken()
	}

	if b.tokens > 500 {
		t.Fatalf("expected tokens not to exceed max capacity after multiple refills, got %v", b.tokens)
	}
}
