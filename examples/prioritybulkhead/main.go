package main

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/cinar/resile"
)

func main() {
	// 1. Define thresholds for different priorities.
	// Low priority traffic is shed when the bulkhead is 50% full.
	// Standard priority traffic is shed when the bulkhead is 80% full.
	// Critical priority traffic is allowed until the bulkhead is 100% full.
	thresholds := map[resile.Priority]float64{
		resile.PriorityLow:      0.5,
		resile.PriorityStandard: 0.8,
		resile.PriorityCritical: 1.0,
	}

	// 2. Create a PriorityBulkhead with a capacity of 10.
	bulkhead := resile.NewPriorityBulkhead(10, thresholds)

	var wg sync.WaitGroup

	// Helper to run a task with a specific priority
	runTask := func(id int, priority resile.Priority, name string) {
		defer wg.Done()

		// Attach priority to context
		ctx := resile.WithPriority(context.Background(), priority)

		err := bulkhead.Execute(ctx, func() error {
			fmt.Printf("[Task %d] Running %s priority task...\n", id, name)
			time.Sleep(100 * time.Millisecond) // Simulate work
			return nil
		})

		if err != nil {
			fmt.Printf("[Task %d] Failed (%s): %v\n", id, name, err)
		}
	}

	fmt.Println("Starting Priority-Aware Bulkhead demonstration...")

	// 3. Saturate the bulkhead to 60% capacity (6 tasks)
	// Low priority (threshold 0.5) should start failing now.
	for i := 1; i <= 6; i++ {
		wg.Add(1)
		go runTask(i, resile.PriorityStandard, "Standard")
	}

	// Give them a moment to start and occupy the semaphore
	time.Sleep(10 * time.Millisecond)

	// 4. Try running a Low priority task (should be shed because utilization > 0.5)
	wg.Add(1)
	go runTask(7, resile.PriorityLow, "Low")

	// 5. Try running a Critical priority task (should be allowed because utilization < 1.0)
	wg.Add(1)
	go runTask(8, resile.PriorityCritical, "Critical")

	wg.Wait()
	fmt.Println("Demonstration completed.")
}
