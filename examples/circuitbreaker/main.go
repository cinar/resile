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
	"github.com/cinar/resile/circuit"
)

func main() {
	ctx := context.Background()

	fmt.Println("--- Layered Defense with Sliding Window Circuit Breaker ---")

	// Create a circuit breaker with a Count-based sliding window.
	// It will trip if 50% of the last 10 calls fail, provided at least 4 calls were made.
	cb := circuit.New(circuit.Config{
		WindowType:           circuit.WindowCountBased,
		WindowSize:           10,
		FailureRateThreshold: 50.0,
		MinimumCalls:         4,
		ResetTimeout:         5 * time.Second,
		HalfOpenMaxCalls:     2,
	})

	action := func(ctx context.Context) error {
		fmt.Println("Attempting operation...")
		return errors.New("service temporarily unavailable")
	}

	// In this example, each iteration performs 2 retry attempts.
	// Since 2 attempts will fail in the first iteration, and 2 in the second,
	// the total failures will reach 4 (MinimumCalls) and the failure rate will be 100% (>50%).
	for i := 0; i < 5; i++ {
		fmt.Printf("\nIteration %d:\n", i+1)
		err := resile.DoErr(ctx, action,
			resile.WithCircuitBreaker(cb),
			resile.WithMaxAttempts(2),
			resile.WithBaseDelay(0), // Skip delay for simulation.
		)

		if errors.Is(err, circuit.ErrCircuitOpen) {
			fmt.Printf("FAIL: Operation fast-failed because the circuit breaker is OPEN!\n")
		} else if err != nil {
			fmt.Printf("FAIL: Operation failed: %v\n", err)
		} else {
			fmt.Println("SUCCESS: Operation succeeded!")
		}
	}
}
