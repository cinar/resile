# Reliable Redis: Combining Retries and Bulkheads for Rock-Solid Caching

Redis is the bedrock of many high-performance Go applications. It's incredibly fast, but like any distributed component, it's not invincible. Network blips, Redis server restarts, or connection pool exhaustion can turn your lightning-fast cache into a source of application errors.

In this article, we'll explore how to use [Resile](https://github.com/cinar/resile) to build a resilient Redis integration that handles transient failures gracefully and prevents connection pool saturation.

---

## The Problem: The Invisible Bottleneck

Most Go developers use `go-redis` or `redigo`. While these clients are excellent, they often hide a critical bottleneck: the **Connection Pool**.

When Redis slows down (e.g., during a BGSAVE or a complex `KEYS *` command), your Go application continues to spawn goroutines that attempt to acquire a connection from the pool. If the pool is exhausted, your goroutines block, waiting for a connection. This leads to:
1.  **Increased Latency**: Every call starts waiting for the pool.
2.  **Resource Leaks**: Goroutines pile up, consuming memory.
3.  **Cascading Failure**: Your application process eventually hits its limits, failing even non-Redis related tasks.

---

## The Solution: Layered Resilience

To build a truly resilient Redis client, we need two layers of protection:
1.  **Retries**: To handle transient network blips.
2.  **Shared Bulkheads**: To strictly limit the number of concurrent operations hitting the connection pool.

### Implementation with Resile

Resile allows you to wrap Redis calls in a type-safe, declarative way. Here’s how you can implement a resilient GET operation:

```go
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
    rdb := redis.NewClient(&redis.Options{Addr: "localhost:6379"})

    // 1. Create a Shared Bulkhead. 
    // This ensures that across our ENTIRE application, we never 
    // have more than 20 concurrent Redis operations.
    redisBulkhead := resile.NewBulkhead(20)

    // 2. Define our Resilience Policy.
    // We combine retries with the shared bulkhead.
    opts := []resile.Option{
        resile.WithMaxAttempts(3),
        resile.WithBaseDelay(100 * time.Millisecond),
        resile.WithBulkheadInstance(redisBulkhead),
    }

    // 3. Execute with Type Safety.
    // resile.Do automatically infers the return type (string).
    val, err := resile.Do(ctx, func(ctx context.Context) (string, error) {
        return rdb.Get(ctx, "user:123").Result()
    }, opts...)

    if err != nil {
        fmt.Printf("Redis operation failed: %v\n", err)
        return
    }
    fmt.Printf("User: %s\n", val)
}
```

---

## Why Shared Bulkheads are Critical

In the example above, `redisBulkhead` is created once and passed to `WithBulkheadInstance`. 

If you have 5 different services (User, Order, Catalog, etc.) all hitting the same Redis instance, you should use the **same bulkhead instance** for all of them. This creates a "global" limit for your process. If one service starts misbehaving and hammers Redis, the bulkhead will fill up and start shedding load *before* the `go-redis` connection pool is completely exhausted, keeping the rest of your application responsive.

[Read more about Bulkhead Isolation](bulkhead-isolation.md)

---

## Type Safety with `resile.Do`

One of the pain points of using generic resilience libraries in Go is losing type safety. Resile uses Go Generics (v1.18+) to ensure that `resile.Do` returns the exact type your Redis command returns.

Whether you are fetching a `string`, a `struct` (via JSON), or a `map`, `resile.Do` preserves the types:

```go
// Returns (User, error) - no manual type casting required!
user, err := resile.Do(ctx, func(ctx context.Context) (User, error) {
    var u User
    err := rdb.Get(ctx, "user:456").Scan(&u)
    return u, err
}, opts...)
```

---

## Advanced: Adding a Circuit Breaker

For even more protection, you can add a **Circuit Breaker**. If Redis goes down completely, the breaker will "trip" and stop all attempts for a cooldown period, preventing your application from wasting resources on retries that are guaranteed to fail.

```go
cb := resile.NewCircuitBreaker()

resile.Do(ctx, action,
    resile.WithCircuitBreakerInstance(cb),
    resile.WithBulkheadInstance(redisBulkhead),
    resile.WithMaxAttempts(3),
)
```

[Learn about Sliding Window Circuit Breakers](sliding-window-circuit-breakers.md)

---

## Conclusion

Redis is fast, but your application's resilience shouldn't rely on "hope." By combining shared bulkheads to protect your connection pool and retries to handle transient blips, you can build a Go application that remains stable even when your infrastructure isn't.

**Explore Resile on GitHub:** [github.com/cinar/resile](https://github.com/cinar/resile)

#golang #redis #resilience #microservices #backend #caching
