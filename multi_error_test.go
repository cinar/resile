// Copyright (c) 2026 Onur Cinar.
// The source code is provided under MIT License.
// https://github.com/cinar/resile

package resile

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"
)

func TestMultiErrorAggregation_Retry(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	err1 := errors.New("error 1")
	err2 := errors.New("error 2")
	err3 := errors.New("error 3")

	var count int
	err := DoErr(ctx, func(ctx context.Context) error {
		count++
		switch count {
		case 1:
			return err1
		case 2:
			return err2
		case 3:
			return err3
		default:
			return nil
		}
	}, WithMaxAttempts(3), WithBaseDelay(0))

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	// Verify it contains all errors
	if !errors.Is(err, err1) {
		t.Errorf("expected error to wrap %v", err1)
	}
	if !errors.Is(err, err2) {
		t.Errorf("expected error to wrap %v", err2)
	}
	if !errors.Is(err, err3) {
		t.Errorf("expected error to wrap %v", err3)
	}

	// Verify Unwrap() []error support
	type multiError interface {
		Unwrap() []error
	}

	if me, ok := err.(multiError); ok {
		unwrapped := me.Unwrap()
		if len(unwrapped) != 3 {
			t.Errorf("expected 3 unwrapped errors, got %d", len(unwrapped))
		}
	} else {
		t.Error("error does not implement Unwrap() []error")
	}

	expectedStr := fmt.Sprintf("%v\n%v\n%v", err1, err2, err3)
	if err.Error() != expectedStr {
		t.Errorf("expected error string %q, got %q", expectedStr, err.Error())
	}
}

func TestMultiErrorAggregation_Hedged(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	err1 := errors.New("error 1")
	err2 := errors.New("error 2")

	err := DoErrStateHedged(ctx, func(ctx context.Context, state RetryState) error {
		// In hedging, multiple attempts run concurrently.
		if state.Attempt == 0 {
			return err1
		}
		return err2
	}, WithMaxAttempts(2), WithHedgingDelay(0))

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !errors.Is(err, err1) {
		t.Errorf("expected error to wrap %v", err1)
	}
	if !errors.Is(err, err2) {
		t.Errorf("expected error to wrap %v", err2)
	}
}

func TestMultiErrorAggregation_Mixed(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	err1 := errors.New("error 1")

	err := DoErr(ctx, func(innerCtx context.Context) error {
		cancel() // Cancel context during the first attempt
		return err1
	}, WithMaxAttempts(3), WithBaseDelay(10*time.Millisecond))

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !errors.Is(err, err1) {
		t.Errorf("expected error to wrap %v", err1)
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected error to wrap context.Canceled, got %v", err)
	}
}

func TestMultiErrorAggregation_Hedged_Stateful(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	err1 := errors.New("error 1")
	err2 := errors.New("error 2")

	err := DoErrStateHedged(ctx, func(ctx context.Context, state RetryState) error {
		if state.Attempt == 0 {
			return err1
		}
		return err2
	}, WithMaxAttempts(2), WithHedgingDelay(0))

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !errors.Is(err, err1) {
		t.Errorf("expected error to wrap %v", err1)
	}
	if !errors.Is(err, err2) {
		t.Errorf("expected error to wrap %v", err2)
	}
}
