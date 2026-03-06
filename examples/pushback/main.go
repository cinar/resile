// Copyright (c) 2026 Onur Cinar.
// The source code is provided under MIT License.
// https://github.com/cinar/resile

package main

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/cinar/resile"
	"github.com/cinar/resile/telemetry/resileslog"
)

// QuotaExceededError implements RetryAfterError to signal a pushback.
type QuotaExceededError struct{}

func (e *QuotaExceededError) Error() string { return "quota exhausted (HTTP 429)" }
func (e *QuotaExceededError) RetryAfter() time.Duration {
	return 0 // Irrelevant as we are cancelling all retries.
}
func (e *QuotaExceededError) CancelAllRetries() bool {
	return true // Pushback signal: abort all retries immediately.
}

func main() {
	ctx := context.Background()
	logger := slog.Default()

	fmt.Println("--- Pushback and Quota Signal Example ---")

	val, err := resile.Do(ctx, func(ctx context.Context) (string, error) {
		fmt.Println("Attempting to process mission-critical payload...")

		// Simulate a situation where the server returns a terminal quota error.
		// Even though we have 5 attempts configured, this should stop after the first.
		return "", &QuotaExceededError{}
	},
		resile.WithName("process-payload"),
		resile.WithInstrumenter(resileslog.New(logger)),
		resile.WithMaxAttempts(5),
	)

	if err != nil {
		fmt.Printf("Operation failed with terminal state: %v\n", err)
	} else {
		fmt.Printf("Operation success: %s\n", val)
	}

	fmt.Println("Done. Notice that the retry loop terminated after exactly 1 attempt.")
}
