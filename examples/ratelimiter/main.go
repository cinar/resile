// Copyright (c) 2026 Onur Cinar.
// The source code is provided under MIT License.
// https://github.com/cinar/resile

package main

import (
	"context"
	"fmt"
	"time"

	"github.com/cinar/resile"
)

func main() {
	ctx := context.Background()

	// 1. Create a RateLimiter that allows 5 requests per second.
	rl := resile.NewRateLimiter(5, time.Second)

	// 2. Try to make 10 requests rapidly.
	for i := 1; i <= 10; i++ {
		err := resile.DoErr(ctx, func(ctx context.Context) error {
			fmt.Printf("Request %d: Success\n", i)
			return nil
		}, resile.WithRateLimiterInstance(rl))

		if err != nil {
			fmt.Printf("Request %d: Rate limited - %v\n", i, err)
		}

		// Wait a bit to see the refill in action
		time.Sleep(50 * time.Millisecond)
	}
}
