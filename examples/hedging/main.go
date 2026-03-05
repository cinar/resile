// Copyright (c) 2026 Onur Cinar.
// The source code is provided under MIT License.
// https://github.com/cinar/resile

package main

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/cinar/resile"
)

func main() {
	fmt.Println("Starting Hedging (Speculative Retries) Example...")

	// Context with an overall timeout.
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Hedging is useful when you have a service with long-tail latency.
	// We send an initial request, and if it doesn't respond within 100ms,
	// we start another one concurrently. We use the first successful response.
	result, err := resile.DoHedged(ctx, func(ctx context.Context) (string, error) {
		// Simulate a backend that is sometimes slow.
		latency := time.Duration(rand.Intn(500)) * time.Millisecond
		fmt.Printf("Attempt started... simulating %v latency\n", latency)

		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(latency):
			return fmt.Sprintf("Success after %v", latency), nil
		}
	},
		resile.WithMaxAttempts(3),
		resile.WithHedgingDelay(100*time.Millisecond),
		resile.WithName("hedged-api-call"),
	)

	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		fmt.Printf("Final Result: %v\n", result)
	}
}
