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

func TestAdaptiveLimiter_AdmissionControl(t *testing.T) {
	al := NewAdaptiveLimiter()
	atomic.StoreInt32(&al.maxConcurrency, 2)

	// Occupy both slots
	ctx := context.Background()
	done := make(chan struct{})

	go func() {
		_ = al.Execute(ctx, func() error {
			<-done
			return nil
		})
	}()

	go func() {
		_ = al.Execute(ctx, func() error {
			<-done
			return nil
		})
	}()

	// Wait a bit for goroutines to start
	time.Sleep(10 * time.Millisecond)

	// Third one should fail
	err := al.Execute(ctx, func() error {
		return nil
	})

	if !errors.Is(err, ErrShedLoad) {
		t.Errorf("expected ErrShedLoad, got %v", err)
	}

	close(done)
}

func TestAdaptiveLimiter_AIMD_Increase(t *testing.T) {
	al := NewAdaptiveLimiter()
	atomic.StoreInt32(&al.maxConcurrency, 5)

	// Establish baseline
	_ = al.Execute(context.Background(), func() error {
		time.Sleep(10 * time.Millisecond)
		return nil
	})

	initialMax := al.GetMaxConcurrency()

	// Perform successful fast requests
	for i := 0; i < 5; i++ {
		_ = al.Execute(context.Background(), func() error {
			time.Sleep(10 * time.Millisecond)
			return nil
		})
	}

	if al.GetMaxConcurrency() <= initialMax {
		t.Errorf("expected maxConcurrency to increase, got %d", al.GetMaxConcurrency())
	}
}

func TestAdaptiveLimiter_AIMD_Decrease(t *testing.T) {
	al := NewAdaptiveLimiter()
	atomic.StoreInt32(&al.maxConcurrency, 10)

	// Establish baseline
	_ = al.Execute(context.Background(), func() error {
		time.Sleep(10 * time.Millisecond)
		return nil
	})

	// Perform slow requests to trigger multiplicative decrease
	// RTT EWMA will eventually exceed 1.5 * 10ms = 15ms
	for i := 0; i < 20; i++ {
		_ = al.Execute(context.Background(), func() error {
			time.Sleep(50 * time.Millisecond)
			return nil
		})
	}

	if al.GetMaxConcurrency() >= 10 {
		t.Errorf("expected maxConcurrency to decrease, got %d", al.GetMaxConcurrency())
	}
}

func TestAdaptiveLimiter_Decay(t *testing.T) {
	al := NewAdaptiveLimiter()
	al.decayInterval = 10 * time.Millisecond

	_ = al.Execute(context.Background(), func() error {
		time.Sleep(10 * time.Millisecond)
		return nil
	})

	baseline1 := al.minBaselineRTT

	time.Sleep(20 * time.Millisecond)

	_ = al.Execute(context.Background(), func() error {
		time.Sleep(baseline1 + 5*time.Millisecond)
		return nil
	})

	baseline2 := al.minBaselineRTT

	if baseline2 <= baseline1 {
		t.Errorf("expected minBaselineRTT to decay (increase), got %v -> %v", baseline1, baseline2)
	}
}

func TestAdaptiveLimiter_ZeroAllocation(t *testing.T) {
	al := NewAdaptiveLimiter()
	ctx := context.Background()
	action := func() error { return nil }

	allocs := testing.AllocsPerRun(100, func() {
		_ = al.Execute(ctx, action)
	})

	if allocs > 0 {
		t.Errorf("expected zero allocations, got %f", allocs)
	}
}

func TestAdaptiveLimiter_PolicyIntegration(t *testing.T) {
	al := NewAdaptiveLimiter()
	atomic.StoreInt32(&al.maxConcurrency, 1)

	p := NewPolicy(WithAdaptiveLimiterInstance(al))

	// Occupy the only slot
	done := make(chan struct{})
	go func() {
		_ = p.DoErr(context.Background(), func(ctx context.Context) error {
			<-done
			return nil
		})
	}()

	time.Sleep(10 * time.Millisecond)

	// Second one should fail
	err := p.DoErr(context.Background(), func(ctx context.Context) error {
		return nil
	})

	if !errors.Is(err, ErrShedLoad) {
		t.Errorf("expected ErrShedLoad, got %v", err)
	}

	close(done)
}
