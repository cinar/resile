// Copyright (c) 2026 Onur Cinar.
// The source code is provided under MIT License.
// https://github.com/cinar/resile

package main

import (
	"context"
	"fmt"
	"time"

	"github.com/cinar/resile"
	"github.com/redis/go-redis/v9"
)

func main() {
	ctx := context.Background()

	// 1. Setup Redis Client
	// For this example, we assume a local Redis server.
	rdb := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	fmt.Println("--- Redis Resilience Example ---")

	// 2. Wrap Redis Get command with Resile
	// We combine retries (for transient network issues) and a bulkhead
	// (to limit concurrent Redis operations and prevent cascading failures).
	val, err := resile.Do(ctx, func(ctx context.Context) (string, error) {
		fmt.Println("Attempting to GET 'my-key' from Redis...")
		return rdb.Get(ctx, "my-key").Result()
	},
		resile.WithMaxAttempts(3),
		resile.WithBaseDelay(100*time.Millisecond),
		resile.WithBulkhead(10), // Limit to 10 concurrent operations
	)

	if err != nil {
		fmt.Printf("Redis GET failed: %v\n", err)
	} else {
		fmt.Printf("Redis GET succeeded: %s\n", val)
	}

	// 3. Wrap Redis Set command with Resile (error only)
	err = resile.DoErr(ctx, func(ctx context.Context) error {
		fmt.Println("Attempting to SET 'my-key' in Redis...")
		return rdb.Set(ctx, "my-key", "resilient-value", 0).Err()
	},
		resile.WithMaxAttempts(2),
	)

	if err != nil {
		fmt.Printf("Redis SET failed: %v\n", err)
	} else {
		fmt.Println("Redis SET succeeded")
	}
}
