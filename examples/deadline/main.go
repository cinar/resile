// Copyright (c) 2026 Onur Cinar.
// The source code is provided under MIT License.
// https://github.com/cinar/resile

package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/cinar/resile"
)

func main() {
	// 1. Define a global overall deadline for the entire request chain.
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// 2. Configure a Resile policy with an early-abort threshold.
	// If the context has less than 50ms remaining, don't even try the network call.
	policy := resile.New(
		resile.WithMaxAttempts(10),
		resile.WithBackoff(&fixedBackoff{delay: 25 * time.Millisecond}),
		resile.WithMinDeadlineThreshold(50*time.Millisecond),
	)

	// 3. Simulate an outgoing HTTP call.
	err := policy.DoErr(ctx, func(innerCtx context.Context) error {
		// Create a dummy request.
		req, _ := http.NewRequestWithContext(innerCtx, "GET", "http://api.internal/v1/resource", nil)

		// Automatically calculate remaining time and inject it into headers.
		// For standard HTTP services:
		resile.InjectDeadlineHeader(innerCtx, req.Header, "X-Resile-Timeout")

		// For gRPC-compatible services (uses gRPC unit formatting, e.g., "150m"):
		resile.InjectDeadlineHeader(innerCtx, req.Header, "Grpc-Timeout")

		fmt.Printf("[Attempt] Remaining: %v | X-Resile-Timeout: %s | Grpc-Timeout: %s\n",
			getRemaining(innerCtx),
			req.Header.Get("X-Resile-Timeout"),
			req.Header.Get("Grpc-Timeout"),
		)

		// Simulate a retryable failure.
		return errors.New("network blip")
	})

	if err != nil {
		fmt.Printf("\nFinal Result: %v\n", err)
	}
}

func getRemaining(ctx context.Context) string {
	if d, ok := ctx.Deadline(); ok {
		return time.Until(d).String()
	}
	return "none"
}

type fixedBackoff struct {
	delay time.Duration
}

func (f *fixedBackoff) Next(_ uint) time.Duration {
	return f.delay
}
