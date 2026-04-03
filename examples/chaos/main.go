// Copyright (c) 2026 Onur Cinar.
// The source code is provided under MIT License.
// https://github.com/cinar/resile

package main

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/cinar/resile"
	"github.com/cinar/resile/chaos"
)

func main() {
	ctx := context.Background()

	// Configure chaos injection.
	// In production, you might want to load this from environment variables.
	cfg := chaos.Config{
		ErrorProbability:   0.5,
		InjectedError:      errors.New("synthetic failure"),
		LatencyProbability: 0.2,
		LatencyDuration:    100 * time.Millisecond,
	}

	fmt.Println("Running with chaos injection (50% error, 20% latency)...")

	for i := 1; i <= 5; i++ {
		fmt.Printf("\n--- Request %d ---\n", i)

		start := time.Now()
		err := resile.DoErr(ctx, func(ctx context.Context) error {
			fmt.Println("Executing business logic...")
			return nil
		},
			resile.WithRetry(3),
			resile.WithChaos(cfg),
		)

		duration := time.Since(start)

		if err != nil {
			fmt.Printf("Request failed after retries: %v (took %v)\n", err, duration)
		} else {
			fmt.Printf("Request succeeded (took %v)\n", duration)
		}
	}
}
