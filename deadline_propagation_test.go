// Copyright (c) 2026 Onur Cinar.
// The source code is provided under MIT License.
// https://github.com/cinar/resile

package resile

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

func TestDeadlinePropagation_EarlyAbort(t *testing.T) {
	t.Parallel()

	// Set a deadline that is very soon.
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Millisecond)
	defer cancel()

	// Set a threshold higher than the deadline.
	config := DefaultConfig()
	config.MinDeadlineThreshold = 10 * time.Millisecond
	config.MaxAttempts = 3
	config.BaseDelay = 0

	var attempts int32
	err := config.execute(ctx, func(ctx context.Context, _ RetryState) error {
		atomic.AddInt32(&attempts, 1)
		return nil
	})

	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("expected context.DeadlineExceeded, got %v", err)
	}

	if atomic.LoadInt32(&attempts) != 0 {
		t.Errorf("expected 0 attempts due to early abort, got %d", atomic.LoadInt32(&attempts))
	}
}

type constantBackoff struct {
	delay time.Duration
}

func (c *constantBackoff) Next(_ uint) time.Duration {
	return c.delay
}

func TestDeadlinePropagation_DynamicRecalculation(t *testing.T) {
	t.Parallel()

	// Set a deadline.
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	config := DefaultConfig()
	config.MinDeadlineThreshold = 50 * time.Millisecond
	config.MaxAttempts = 5
	config.Backoff = &constantBackoff{delay: 20 * time.Millisecond}

	var attempts int32
	err := config.execute(ctx, func(ctx context.Context, _ RetryState) error {
		atomic.AddInt32(&attempts, 1)
		return errors.New("retryable error")
	})

	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("expected context.DeadlineExceeded, got %v", err)
	}

	// First attempt happens (remaining ~100ms > 50ms).
	// After 1st attempt, it sleeps 20ms. Remaining ~80ms.
	// Second attempt happens (remaining ~80ms > 50ms).
	// After 2nd attempt, it sleeps 20ms. Remaining ~60ms.
	// Third attempt happens (remaining ~60ms > 50ms).
	// After 3rd attempt, it sleeps 20ms. Remaining ~40ms.
	// Fourth attempt should be aborted early because remaining ~40ms < 50ms.

	if atomic.LoadInt32(&attempts) != 3 {
		t.Errorf("expected 3 attempts before early abort, got %d", atomic.LoadInt32(&attempts))
	}
}

type mockHeader struct {
	values map[string]string
}

func (m *mockHeader) Set(key, value string) {
	m.values[key] = value
}

func TestInjectDeadlineHeader(t *testing.T) {
	t.Parallel()

	t.Run("CustomHeader", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()

		header := &mockHeader{values: make(map[string]string)}
		InjectDeadlineHeader(ctx, header, "X-Timeout")

		val, ok := header.values["X-Timeout"]
		if !ok {
			t.Fatal("X-Timeout header missing")
		}

		// Milliseconds should be around 1000.
		if val == "" || val == "0" {
			t.Errorf("unexpected header value: %s", val)
		}
	})

	t.Run("GrpcHeader", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()

		header := &mockHeader{values: make(map[string]string)}
		InjectDeadlineHeader(ctx, header, "Grpc-Timeout")

		val, ok := header.values["Grpc-Timeout"]
		if !ok {
			t.Fatal("Grpc-Timeout header missing")
		}

		// gRPC format for ~1s should be "1S" or "1000m".
		// Based on our implementation: d < time.Second is "dm", d < time.Minute is "dS".
		// 1s is exactly time.Second, so it should be "1S" if it's 1.0s or "1000m" if it's 0.999s.
		if val == "" {
			t.Errorf("unexpected header value: %s", val)
		}
	})
}

func TestFormatGRPCTimeout(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		duration time.Duration
		want     string
	}{
		{"Nanoseconds", 500 * time.Nanosecond, "500n"},
		{"Microseconds", 500 * time.Microsecond, "500u"},
		{"Milliseconds", 500 * time.Millisecond, "500m"},
		{"Seconds", 5 * time.Second, "5S"},
		{"Minutes", 5 * time.Minute, "5M"},
		{"Hours", 5 * time.Hour, "5H"},
		{"Zero", 0, "0n"},
		{"Negative", -1 * time.Second, "0n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := formatGRPCTimeout(tt.duration); got != tt.want {
				t.Errorf("formatGRPCTimeout() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDeadlinePropagation_HedgedEarlyAbort(t *testing.T) {
	t.Parallel()

	// Set a deadline that is very soon.
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Millisecond)
	defer cancel()

	// Set a threshold higher than the deadline.
	config := DefaultConfig()
	config.MinDeadlineThreshold = 10 * time.Millisecond
	config.MaxAttempts = 3
	config.HedgingDelay = 50 * time.Millisecond

	var attempts int32
	err := config.executeHedged(ctx, func(ctx context.Context, _ RetryState) error {
		atomic.AddInt32(&attempts, 1)
		return nil
	})

	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("expected context.DeadlineExceeded, got %v", err)
	}

	// In executeHedged, it starts the first attempt immediately.
	// The deadlineMiddleware should catch it if it's within the attempt.
	if atomic.LoadInt32(&attempts) != 0 {
		t.Errorf("expected 0 attempts due to early abort in hedged execution, got %d", atomic.LoadInt32(&attempts))
	}
}

func TestWithMinDeadlineThreshold(t *testing.T) {
	t.Parallel()

	threshold := 50 * time.Millisecond
	c := DefaultConfig()
	WithMinDeadlineThreshold(threshold)(c)

	if c.MinDeadlineThreshold != threshold {
		t.Errorf("expected MinDeadlineThreshold %v, got %v", threshold, c.MinDeadlineThreshold)
	}
}
