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
)

func main() {
	ctx := context.Background()
	endpoints := []string{
		"primary-api.example.com",
		"secondary-api.example.com",
		"tertiary-api.example.com",
	}

	fmt.Println("--- Stateful Retry with Endpoint Rotation ---")

	// 3. Use DoState to access RetryState for endpoint rotation fallback.
	val, err := resile.DoState(ctx, func(ctx context.Context, state resile.RetryState) (string, error) {
		// Use Attempt count to choose an endpoint.
		endpoint := endpoints[state.Attempt%uint(len(endpoints))]
		fmt.Printf("Attempt %d: Connecting to %s...\n", state.Attempt+1, endpoint)

		// Simulate connection refusal on primary and secondary.
		if state.Attempt < 2 {
			return "", errors.New("connection refused")
		}

		return "Successful connection to " + endpoint, nil
	}, resile.WithMaxAttempts(3), resile.WithBaseDelay(100*time.Millisecond))

	if err != nil {
		fmt.Printf("Endpoint rotation failed: %v\n", err)
	} else {
		fmt.Printf("Final result: %s\n", val)
	}
}
