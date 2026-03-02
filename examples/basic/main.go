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

	fmt.Println("--- Basic Retry Example ---")

	// 1. Simple retry for an operation that returns only an error.
	err := resile.DoErr(ctx, func(ctx context.Context) error {
		fmt.Println("Pinging database...")
		// Simulate a transient failure.
		return errors.New("connection reset by peer")
	}, resile.WithMaxAttempts(3), resile.WithBaseDelay(100*time.Millisecond))

	if err != nil {
		fmt.Printf("Database ping failed after retries: %v\n", err)
	}

	fmt.Println("\n--- Value-Yielding Retry Example ---")

	// 2. Retry for an operation that returns a value and an error.
	// Type safety is guaranteed by generics.
	val, err := resile.Do(ctx, func(ctx context.Context) (string, error) {
		fmt.Println("Fetching configuration...")
		// Use a neutral configuration string.
		return "DATABASE_URL=postgres://localhost:5432/mydb", nil
	})

	if err == nil {
		fmt.Printf("Configuration fetched: %s\n", val)
	}
}
