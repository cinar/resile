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

	fmt.Println("--- Erlang-style Panic Recovery Example ---")

	// 1. Using WithPanicRecovery to catch unexpected panics.
	// This implements the "Let It Crash" philosophy where the operation
	// is reset to a known good state without crashing the whole application.
	attempts := 0
	val, err := resile.Do(ctx, func(ctx context.Context) (string, error) {
		attempts++
		fmt.Printf("Attempt %d: Executing risky operation...\n", attempts)

		if attempts < 3 {
			// Simulate an unexpected bug that triggers a panic.
			panic("something went catastrophically wrong (transient)")
		}

		return "SUCCESS: result after 2 transient panics", nil
	},
		resile.WithPanicRecovery(), // Enable panic recovery
		resile.WithMaxAttempts(5),
		resile.WithBaseDelay(500*time.Millisecond),
	)

	if err != nil {
		fmt.Printf("\nOperation failed after retries: %v\n", err)
	} else {
		fmt.Printf("\nOperation succeeded: %s\n", val)
	}

	fmt.Println("\n--- Handling the Panic Error ---")

	// 2. You can inspect the panic error to log the stack trace.
	err = resile.DoErr(ctx, func(ctx context.Context) error {
		panic("irrecoverable failure")
	}, resile.WithPanicRecovery(), resile.WithMaxAttempts(1))

	var panicErr *resile.PanicError
	if errors.As(err, &panicErr) {
		fmt.Printf("Caught Panic Value: %v\n", panicErr.Value)
		fmt.Println("First line of stack trace captured:")
		// Printing the whole stack trace might be too verbose for an example.
		// panicErr.StackTrace contains the full debug.Stack().
		fmt.Println("Check the logs or instrumenter for the full stack trace.")
	}
}
