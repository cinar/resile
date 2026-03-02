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

	fmt.Println("--- Layered Defense with Circuit Breaker ---")

	cb := circuit.New(circuit.Config{
		FailureThreshold: 2,                // Open the circuit after 2 failures.
		ResetTimeout:     10 * time.Second, // Wait 10 seconds before trying again.
	})

	action := func(ctx context.Context) error {
		fmt.Println("Attempting operation...")
		return errors.New("service temporarily unavailable")
	}

	// 4. Combine retry loop and circuit breaker.
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
		}
	}
}
