// Copyright (c) 2026 Onur Cinar.
// The source code is provided under MIT License.
// https://github.com/cinar/resile

package main

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/cinar/resile"
)

func main() {
	ctx := context.Background()

	// 1. Create a shared bulkhead with capacity of 2.
	bh := resile.NewBulkhead(2)

	var wg sync.WaitGroup

	// 2. Start 5 concurrent tasks. Only 2 should be allowed at a time.
	for i := 1; i <= 5; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			err := resile.DoErr(ctx, func(ctx context.Context) error {
				fmt.Printf("Task %d: Entered Bulkhead\n", id)
				time.Sleep(100 * time.Millisecond) // Simulate work
				return nil
			}, resile.WithBulkheadInstance(bh))

			if err != nil {
				fmt.Printf("Task %d: Failed - %v\n", id, err)
			} else {
				fmt.Printf("Task %d: Success\n", id)
			}
		}(i)
	}

	wg.Wait()
}
