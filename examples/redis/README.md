# Redis Resilience Example

This example demonstrates how to use **Resile** to add resilience to a Redis client using the popular `github.com/redis/go-redis` package.

## Features Covered

1. **Retries:** Automatically retry failed Redis commands (e.g., due to transient connection issues).
2. **Bulkhead:** Limit the number of concurrent operations to Redis to prevent overloading the database or the application.
3. **Type Safety:** Using `resile.Do` to maintain type information for the Redis response.

## Prerequisites

- [Redis](https://redis.io/) server running on `localhost:6379` (optional, the example shows the pattern even if it fails to connect).
- [Go](https://go.dev/) 1.18+

## How to Run

1. Initialize dependencies:
   ```bash
   go mod tidy
   ```

2. Run the example:
   ```bash
   go run main.go
   ```

## Key Pattern

```go
val, err := resile.Do(ctx, func(ctx context.Context) (string, error) {
    return rdb.Get(ctx, "key").Result()
}, 
    resile.WithMaxAttempts(3),
    resile.WithBulkhead(10),
)
```
