package resile

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestRetryLoop_AdaptiveBucket_FailFast(t *testing.T) {
	t.Parallel()

	// Given a bucket with very limited capacity (e.g. max 5, cost 5, refill 1)
	// Initial state: 5 tokens.
	bucket := NewAdaptiveBucket(5, 5, 1)

	attempts := 0
	action := func(ctx context.Context, state RetryState) error {
		attempts++
		return errors.New("simulated transient error")
	}

	// First execution (attempt 0 does not cost a token, but attempt 1 costs 5).
	// tokens: 5. First retry will consume 5 tokens -> 0.
	// Second retry will fail to acquire -> fail fast.
	err := DoErrState(context.Background(), action, WithMaxAttempts(5), WithBaseDelay(time.Millisecond), WithAdaptiveBucket(bucket))

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	// Expecting only 2 attempts:
	// Attempt 0 (fails) -> acquires token -> 0 tokens left.
	// Attempt 1 (fails) -> tries to acquire token -> fails -> terminates.
	if attempts != 2 {
		t.Fatalf("expected 2 attempts due to adaptive bucket fail-fast, got %d", attempts)
	}
}

func TestRetryLoop_AdaptiveBucket_SuccessRefill(t *testing.T) {
	t.Parallel()

	bucket := NewAdaptiveBucket(10, 5, 5)

	// First execution succeeds on the 2nd attempt.
	// Attempt 0: fails.
	// Attempt 1: acquire (10 -> 5). succeeds -> refill (5 -> 10).
	attempts1 := 0
	action1 := func(ctx context.Context, state RetryState) error {
		attempts1++
		if attempts1 < 2 {
			return errors.New("error")
		}
		return nil
	}

	err := DoErrState(context.Background(), action1, WithMaxAttempts(5), WithBaseDelay(time.Millisecond), WithAdaptiveBucket(bucket))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Bucket should have 10 tokens again.
	if bucket.tokens != 10 {
		t.Fatalf("expected 10 tokens, got %v", bucket.tokens)
	}
}
