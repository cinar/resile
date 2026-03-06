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

func main() {
	// Create an AdaptiveBucket with standard defaults:
	// - max capacity: 500
	// - retry cost: 5
	// - success refill: 1
	// This bucket should be shared across all operations targeting a specific service
	// to provide macro-level protection across your cluster.
	bucket := resile.DefaultAdaptiveBucket()

	// Simulated downstream service that is currently overwhelmed.
	overwhelmedAction := func(ctx context.Context) error {
		fmt.Println("Executing action...")
		return errors.New("service unavailable")
	}

	// We'll run a loop to simulate multiple incoming requests hitting our service.
	for i := 1; i <= 3; i++ {
		fmt.Printf("\n--- Incoming Request #%d ---\n", i)
		err := resile.DoErr(context.Background(), overwhelmedAction,
			resile.WithName(fmt.Sprintf("request-%d", i)),
			resile.WithMaxAttempts(10), // We have a high retry budget...
			resile.WithBaseDelay(10*time.Millisecond),
			resile.WithAdaptiveBucket(bucket), // ...but the bucket will limit us.
		)

		if err != nil {
			fmt.Printf("Request %d failed: %v\n", i, err)
		}
	}

	fmt.Printf("\nBucket still has tokens: %v\n", bucket.AcquireRetryToken())
	fmt.Println("Notice how subsequent requests fail faster as the token bucket depletes.")
}

func init() {
	log.SetFlags(0)
}
