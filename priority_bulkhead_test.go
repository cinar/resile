package resile

import (
	"context"
	"errors"
	"sync"
	"testing"
)

func TestPriorityBulkhead(t *testing.T) {
	t.Parallel()

	capacity := uint(10)
	thresholds := map[Priority]float64{
		PriorityLow:      0.5, // 50% utilization
		PriorityStandard: 0.8, // 80% utilization
		PriorityCritical: 1.0, // 100% utilization
	}

	bh := NewPriorityBulkhead(capacity, thresholds)

	// Channels to coordinate goroutines
	active := make(chan struct{})
	release := make(chan struct{})
	defer close(release)

	fill := func(count int) {
		for i := 0; i < count; i++ {
			go func() {
				_ = bh.Execute(WithPriority(context.Background(), PriorityCritical), func() error {
					active <- struct{}{}
					<-release
					return nil
				})
			}()
			<-active // Wait for it to be active inside the bulkhead
		}
	}

	// Fill up to 50%
	fill(5)

	// Low priority should be shed now (utilization is 5/10 = 0.5)
	err := bh.Execute(WithPriority(context.Background(), PriorityLow), func() error {
		return nil
	})
	if !errors.Is(err, ErrShedLoad) {
		t.Errorf("expected ErrShedLoad for low priority, got %v", err)
	}

	// Standard priority should still pass
	err = bh.Execute(WithPriority(context.Background(), PriorityStandard), func() error {
		return nil
	})
	if err != nil {
		t.Errorf("expected no error for standard priority, got %v", err)
	}

	// Fill up to 80% (3 more)
	fill(3)

	// Standard priority should be shed now (utilization is 8/10 = 0.8)
	err = bh.Execute(WithPriority(context.Background(), PriorityStandard), func() error {
		return nil
	})
	if !errors.Is(err, ErrShedLoad) {
		t.Errorf("expected ErrShedLoad for standard priority, got %v", err)
	}

	// Critical priority should still pass
	err = bh.Execute(WithPriority(context.Background(), PriorityCritical), func() error {
		return nil
	})
	if err != nil {
		t.Errorf("expected no error for critical priority, got %v", err)
	}

	// Fill up to 100% (2 more)
	fill(2)

	// Critical priority should be full now
	err = bh.Execute(WithPriority(context.Background(), PriorityCritical), func() error {
		return nil
	})
	if !errors.Is(err, ErrBulkheadFull) {
		t.Errorf("expected ErrBulkheadFull for critical priority, got %v", err)
	}
}

func TestPriorityBulkheadDefaultPriority(t *testing.T) {
	t.Parallel()

	capacity := uint(10)
	thresholds := map[Priority]float64{
		PriorityStandard: 0.5,
	}

	bh := NewPriorityBulkhead(capacity, thresholds)

	// Channels to coordinate goroutines
	active := make(chan struct{})
	release := make(chan struct{})
	defer close(release)

	// Fill up to 50%
	for i := 0; i < 5; i++ {
		go func() {
			_ = bh.Execute(context.Background(), func() error {
				active <- struct{}{}
				<-release
				return nil
			})
		}()
		<-active
	}

	// Standard priority (default) should be shed
	err := bh.Execute(context.Background(), func() error {
		return nil
	})
	if !errors.Is(err, ErrShedLoad) {
		t.Errorf("expected ErrShedLoad for default priority, got %v", err)
	}
}

func TestPriorityBulkheadIntegration(t *testing.T) {
	t.Parallel()

	bh := NewPriorityBulkhead(10, map[Priority]float64{
		PriorityLow: 0.1,
	})

	// Occupy 1 slot
	active := make(chan struct{})
	release := make(chan struct{})

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		_ = DoErr(context.Background(), func(ctx context.Context) error {
			active <- struct{}{}
			<-release
			return nil
		}, WithPriorityBulkheadInstance(bh))
	}()

	<-active

	// Low priority should fail. Use WithMaxAttempts(1) to avoid retries.
	err := DoErr(WithPriority(context.Background(), PriorityLow), func(ctx context.Context) error {
		return nil
	}, WithPriorityBulkheadInstance(bh), WithMaxAttempts(1))

	if !errors.Is(err, ErrShedLoad) {
		t.Errorf("expected ErrShedLoad, got %v", err)
	}

	// Trigger release before waiting
	close(release)
	wg.Wait()
}
