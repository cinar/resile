// Copyright (c) 2026 Onur Cinar.
// The source code is provided under MIT License.
// https://github.com/cinar/resile

package resile

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
)

func TestStateMachine(t *testing.T) {
	t.Parallel()

	type state string
	const (
		stateA state = "A"
		stateB state = "B"
	)

	type event string
	const (
		eventGoToB event = "GoToB"
	)

	t.Run("SuccessfulTransition", func(t *testing.T) {
		t.Parallel()
		sm := NewStateMachine(stateA, 0, func(ctx context.Context, s state, d int, e event, rs RetryState) (state, int, error) {
			if s == stateA && e == eventGoToB {
				return stateB, d + 1, nil
			}
			return s, d, nil
		})

		err := sm.Handle(context.Background(), eventGoToB)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if sm.GetState() != stateB {
			t.Errorf("expected state %v, got %v", stateB, sm.GetState())
		}

		if sm.GetData() != 1 {
			t.Errorf("expected data 1, got %v", sm.GetData())
		}
	})

	t.Run("RetriedTransition", func(t *testing.T) {
		t.Parallel()
		var attempts int32
		sm := NewStateMachine(stateA, 0, func(ctx context.Context, s state, d int, e event, rs RetryState) (state, int, error) {
			val := atomic.AddInt32(&attempts, 1)
			if val < 3 {
				return "", 0, errors.New("temporary error")
			}
			return stateB, d + 1, nil
		}, WithMaxAttempts(5))

		ctx := WithTestingBypass(context.Background())
		err := sm.Handle(ctx, eventGoToB)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if sm.GetState() != stateB {
			t.Errorf("expected state %v, got %v", stateB, sm.GetState())
		}

		if sm.GetData() != 1 {
			t.Errorf("expected data 1, got %v", sm.GetData())
		}

		if atomic.LoadInt32(&attempts) != 3 {
			t.Errorf("expected 3 attempts, got %d", attempts)
		}
	})

	t.Run("FailedTransition", func(t *testing.T) {
		t.Parallel()
		sm := NewStateMachine(stateA, 0, func(ctx context.Context, s state, d int, e event, rs RetryState) (state, int, error) {
			return "", 0, errors.New("permanent error")
		}, WithMaxAttempts(2))

		ctx := WithTestingBypass(context.Background())
		err := sm.Handle(ctx, eventGoToB)
		if err == nil {
			t.Fatal("expected error, got nil")
		}

		// State and Data should not change on failure.
		if sm.GetState() != stateA {
			t.Errorf("expected state %v, got %v", stateA, sm.GetState())
		}

		if sm.GetData() != 0 {
			t.Errorf("expected data 0, got %v", sm.GetData())
		}
	})
}

func TestStateMachineConcurrency(t *testing.T) {
	t.Parallel()
	type state string
	type event string
	sm := NewStateMachine("initial", 0, func(ctx context.Context, s state, d int, e event, rs RetryState) (state, int, error) {
		return s, d + 1, nil
	})

	const numGoroutines = 100
	done := make(chan bool)
	for i := 0; i < numGoroutines; i++ {
		go func() {
			_ = sm.Handle(context.Background(), "increment")
			done <- true
		}()
	}

	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	if sm.GetData() != numGoroutines {
		t.Errorf("expected data %d, got %d", numGoroutines, sm.GetData())
	}
}
