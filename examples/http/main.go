// Copyright (c) 2026 Onur Cinar.
// The source code is provided under MIT License.
// https://github.com/cinar/resile

package main

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"time"

	"github.com/cinar/resile"
	"github.com/cinar/resile/telemetry/resileslog"
)

// MyAPIError implements the RetryAfterError interface to support server-dictated rate limits.
type MyAPIError struct {
	WaitUntil time.Time
}

func (e *MyAPIError) Error() string { return "too many requests (HTTP 429)" }
func (e *MyAPIError) RetryAfter() time.Duration {
	return time.Until(e.WaitUntil)
}

func main() {
	ctx := context.Background()
	logger := slog.Default()

	fmt.Println("--- HTTP and Rate Limit Example ---")

	val, err := resile.Do(ctx, func(ctx context.Context) (string, error) {
		fmt.Println("Requesting API endpoint...")

		// Simulate a rare rate limit event.
		if rand.Float32() < 0.3 {
			retryAt := time.Now().Add(500 * time.Millisecond)
			fmt.Printf("Rate limit encountered, server requested retry at %v\n", retryAt)
			return "", &MyAPIError{WaitUntil: retryAt}
		}

		return "Response payload", nil
	},
		resile.WithName("fetch-inventory"),
		resile.WithInstrumenter(resileslog.New(logger)), // Log all attempts
		resile.WithMaxAttempts(3),
	)

	if err != nil {
		fmt.Printf("API request failed: %v\n", err)
	} else {
		fmt.Printf("API response received: %s\n", val)
	}
}
