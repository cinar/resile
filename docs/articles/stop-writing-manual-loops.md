# Stop Writing Manual Retry Loops in Go: Why Your Current Logic is Probably Dangerous

If you've been writing Go for more than a week, you've likely written a retry loop. It usually starts like this:

```go
for i := 0; i < 3; i++ {
    err := doSomething()
    if err == nil {
        break
    }
    time.Sleep(1 * time.Second)
}
```

It's simple, idiomatic, and... **a ticking time bomb in production.**

In a distributed system, transient failures—network blips, database locks, rate limits—are mathematical certainties. While a simple `for` loop feels like enough, it often fails exactly when your system is under the most stress.

Here is why your manual retry logic is probably dangerous, and how to fix it using [Resile](https://github.com/cinar/resile).

---

## The 4 Silent Killers of Manual Retries

### 1. The Thundering Herd (Missing Jitter)
If your service has 1,000 instances and the database goes down for a second, all 1,000 instances will fail at once. With a fixed `time.Sleep(1 * time.Second)`, all 1,000 instances will then wake up at the exact same millisecond and hammer the database again. 

This is a self-inflicted DDoS attack. Without **Jitter** (randomized delay), your retries are just synchronized waves of traffic that prevent your dependencies from ever recovering.

### 2. The Context Blindness
Does your retry loop respect `context.Context`? Most don't. If a user cancels their request or a global timeout is reached, a `time.Sleep` will block that goroutine until the timer expires. 

In a high-concurrency environment, these "hanging" goroutines pile up, leading to memory exhaustion and silent failures.

### 3. The `time.After` Memory Leak
Even "advanced" developers trying to be context-aware often use this:

```go
select {
case <-ctx.Done():
    return ctx.Err()
case <-time.After(delay): // DANGER!
    // proceed
}
```

According to the Go standard library, the timer created by `time.After` **is not garbage collected until it fires**, even if the `ctx.Done()` case is chosen. In a busy service with long retries, this creates a slow-motion memory leak that is incredibly hard to debug.

### 4. The Missing History (Last-Error Bias)
When a loop retries 3 times and finally gives up, what error does it return? Usually just the last one. If your first attempt failed with an "Authentication Error" but the third one failed with "Context Canceled" because the user hung up, you've lost the most important piece of information for debugging.

---

## Introducing Resile: Ergonomic Resilience for Go

I built **Resile** because I missed the ergonomics of Python’s `stamina` and `tenacity` libraries, but I wanted the uncompromising type safety of Go 1.18+ Generics.

Resile is a zero-dependency, execution resilience library that makes the "Correct Way" to retry as easy as a single function call.

### The "Hello, World" of Resile

Instead of a manual loop, you use `DoErr` for actions that only return an error:

```go
err := resile.DoErr(ctx, func(ctx context.Context) error {
    return db.PingContext(ctx)
})
```

Or `Do` for value-yielding operations with full type safety:

```go
// user is automatically inferred as *User
user, err := resile.Do(ctx, func(ctx context.Context) (*User, error) {
    return apiClient.GetUser(ctx, userID)
}, resile.WithMaxAttempts(3))
```

---

## Why Resile?

### 1. AWS Full Jitter by Default
Resile implements the industry-standard **Full Jitter** algorithm. Instead of sleeping for a fixed time, it calculates an exponential backoff and then picks a random value between 0 and that maximum. This perfectly spreads out the load across your cluster.

### 2. Memory-Safe Timer Management
Resile doesn't use `time.After`. It uses a managed `time.Timer` with explicit cleanup. Whether your retry succeeds, fails, or the context is cancelled, Resile ensures all resources are returned to the runtime immediately.

### 3. Native Multi-Error Aggregation
Resile uses Go 1.20's `errors.Join` to return every error encountered during the retry timeline. You don't just see the last failure; you see everything that happened.

```go
// Printing the error shows all attempts
fmt.Println(err)

// You can still use standard primitives
if errors.Is(err, ErrSpecificFailure) { ... }
```

[Read more: Debugging the Timeline: Native Multi-Error Aggregation in Go](native-multi-error-aggregation.md)

### 4. Generic-First API
No `interface{}`, no reflection, and no type casting. Because it uses Go Generics, the compiler checks your types at build time. If your function returns a `*User`, Resile returns a `*User`.

### 5. Fast Unit Testing
One of the biggest pain points of retries is that they slow down your CI/CD pipelines. Who wants to wait 10 seconds for a test to finish because of backoff?

With Resile, you can use `WithTestingBypass` to make all retries execute instantly in your tests:

```go
func TestMyService(t *testing.T) {
    ctx := resile.WithTestingBypass(context.Background())
    
    // This will retry 5 times INSTANTLY without sleeping.
    err := service.Handle(ctx)
}
```

---

## Beyond Simple Retries

Resile isn't just a retry loop; it's a resilience toolkit. Out of the box, you get:
- **Request Hedging**: Start a second request if the first one is taking too long (beating tail latency).
- **Adaptive Retries**: A client-side token bucket to prevent "retry storms" across a cluster.
- **Circuit Breaker Integration**: Stop retrying when a service is fundamentally down.
- **Panic Recovery**: Convert unexpected panics into retryable errors (the Erlang "Let It Crash" way).

---

## Conclusion

Retrying is a distributed systems problem, not just a loop problem. By moving away from manual loops to a dedicated resilience engine like Resile, you protect your downstream services, eliminate memory leaks, and keep your code clean and type-safe.

**Check out Resile on GitHub:** [github.com/cinar/resile](https://github.com/cinar/resile)

How are you handling transient failures in your Go services? Let's discuss in the comments!

#golang #backend #distributedsystems #microservices
