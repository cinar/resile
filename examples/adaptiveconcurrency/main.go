// Copyright (c) 2026 Onur Cinar.
// The source code is provided under MIT License.
// https://github.com/cinar/resile

package main

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/cinar/resile"
)

func main() {
	// Create an adaptive concurrency limiter
	al := resile.NewAdaptiveLimiter()

	// Create a policy using the adaptive limiter
	p := resile.NewPolicy(
		resile.WithAdaptiveLimiterInstance(al),
	)

	fmt.Println("Starting simulation with adaptive concurrency...")
	fmt.Printf("Initial Max Concurrency: %d\n", al.GetMaxConcurrency())

	ctx := context.Background()
	var wg sync.WaitGroup

	// Simulate periodic bursts of traffic
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			// Each request has some random processing time
			processTime := time.Duration(10+rand.Intn(20)) * time.Millisecond

			// Periodically simulate a latency spike
			if i > 25 && i < 35 {
				processTime = time.Duration(100+rand.Intn(50)) * time.Millisecond
			}

			err := p.DoErr(ctx, func(innerCtx context.Context) error {
				time.Sleep(processTime)
				return nil
			})

			if err != nil {
				if errors.Is(err, resile.ErrShedLoad) {
					fmt.Printf("Request %d: Shed Load (Max: %d)\n", id, al.GetMaxConcurrency())
				} else {
					fmt.Printf("Request %d Error: %v\n", id, err)
				}
			} else if id%10 == 0 {
				fmt.Printf("Request %d Successful (Max: %d)\n", id, al.GetMaxConcurrency())
			}
		}(i)

		time.Sleep(10 * time.Millisecond)
	}

	wg.Wait()
	fmt.Printf("Final Max Concurrency: %d\n", al.GetMaxConcurrency())
}
