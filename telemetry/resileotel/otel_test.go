// Copyright (c) 2026 Onur Cinar.
// The source code is provided under MIT License.
// https://github.com/cinar/resile

package resileotel

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/cinar/resile"
	"go.opentelemetry.io/otel/metric/noop"
	tracenoop "go.opentelemetry.io/otel/trace/noop"
)

func TestOtelInstrumenter(t *testing.T) {
	// Using no-op implementations for simplicity in unit test.
	tp := tracenoop.NewTracerProvider()
	mp := noop.NewMeterProvider()

	instr, err := New(tp, mp)
	if err != nil {
		t.Fatalf("failed to create instrumenter: %v", err)
	}

	ctx := context.Background()
	state := resile.RetryState{
		Attempt:       1,
		MaxAttempts:   3,
		LastError:     errors.New("otel error"),
		TotalDuration: 100 * time.Millisecond,
		NextDelay:     200 * time.Millisecond,
	}

	// Test lifecycle: Should not panic.
	ctx = instr.BeforeAttempt(ctx, state)
	instr.AfterAttempt(ctx, state)
}

func TestOtelInstrumenter_Detailed(t *testing.T) {
	// Test defaults in New
	instr, err := New(nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()

	// Test attempt > 0 (for counter) and success state
	state := resile.RetryState{
		Attempt: 1,
		Name:    "test",
	}
	ctx = instr.BeforeAttempt(ctx, state)
	instr.AfterAttempt(ctx, state)

	// Test with NextDelay
	state.NextDelay = 1 * time.Second
	instr.AfterAttempt(ctx, state)
}
