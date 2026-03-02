// Copyright (c) 2026 Onur Cinar.
// The source code is provided under MIT License.
// https://github.com/cinar/resile

package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/cinar/resile"
)

var errUnreachable = errors.New("remote service unreachable")

func main() {
	ctx := context.Background()

	// Example 1: Fallback for value-yielding retries (e.g., returning stale data)
	fmt.Println("--- Example 1: Fallback for Value-Yielding Retry ---")
	data, err := resile.Do(ctx, func(ctx context.Context) (string, error) {
		fmt.Println("Attempting to fetch fresh data...")
		return "", errUnreachable
	},
		resile.WithMaxAttempts(3),
		resile.WithBaseDelay(100*time.Millisecond),
		resile.WithFallback(func(ctx context.Context, err error) (string, error) {
			fmt.Printf("Primary fetch failed after all attempts: %v\n", err)
			fmt.Println("Returning stale data from cache as fallback...")
			return "stale data from cache (2026-03-01)", nil
		}),
	)

	if err != nil {
		log.Fatalf("Unexpected error: %v", err)
	}
	fmt.Printf("Result: %s\n\n", data)

	// Example 2: Fallback for error-only retries (e.g., logging or transforming error)
	fmt.Println("--- Example 2: Fallback for Error-Only Retry ---")
	err = resile.DoErr(ctx, func(ctx context.Context) error {
		fmt.Println("Attempting to perform a critical operation...")
		return errUnreachable
	},
		resile.WithMaxAttempts(2),
		resile.WithBaseDelay(100*time.Millisecond),
		resile.WithFallbackErr(func(ctx context.Context, err error) error {
			fmt.Printf("Critical operation failed: %v. Triggering alert system...\n", err)
			// Transform to a user-friendly error or return nil if the failure is non-critical
			return errors.New("the system is currently under maintenance, please try again later")
		}),
	)

	if err != nil {
		fmt.Printf("Final User Error: %v\n", err)
	}
}
