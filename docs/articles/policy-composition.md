# Power of Policy Composition: Building Layered Resilience in Go

In the world of distributed systems, a single resilience strategy is rarely enough. Transient network glitches might require **Retries**, while systemic service failures demand **Circuit Breakers**. Concurrent load spikes might necessitate **Bulkheads**, and slow responses require **Timeouts**.

Until now, `resile` provided these features with a sensible, but fixed, execution order. With the introduction of the **Policy Composition & Chaining API**, you now have full control over how your resilience layers are stacked.

## The Onion Model

Resile adopts the "Onion Model" for policy composition. When you compose multiple policies, they form layers around your action. The order in which you define them determines the execution hierarchy from the **outermost** layer to the **innermost** layer.

```go
standardPolicy := resile.NewPolicy(
    resile.WithBulkhead(20),
    resile.WithCircuitBreaker(cb),
    resile.WithRetry(3),
    resile.WithTimeout(1*time.Second),
)
```

In this example, the execution flow looks like this:

1.  **Bulkhead (Outermost)**: The first line of defense. It limits the total number of concurrent executions allowed to enter the entire resilience stack. If the bulkhead is full, the call fails fast before even attempting a retry or checking the circuit breaker.
2.  **Circuit Breaker**: Wraps the retry loop. If the whole retry process (all 3 attempts) fails consistently, the circuit breaker trips. This protects your system from wasting resources on a downstream service that is clearly down.
3.  **Retry**: The driver of the loop. It handles transient errors by re-executing the inner layers.
4.  **Timeout (Innermost)**: Applied to each individual attempt. If a single attempt takes longer than 1 second, it is cancelled, and the retry policy decides whether to try again.

## Why Order Matters

Reversing the order can fundamentally change the behavior of your resilience strategy.

### Example A: Retry wraps Circuit Breaker (The Default)
```go
resile.NewPolicy(
    resile.WithRetry(3),
    resile.WithCircuitBreaker(cb),
)
```
Each retry attempt is individually tracked by the circuit breaker. If the circuit is open, the retry loop terminates early. This is great for handling blips while still providing fast-fail protection for each attempt.

### Example B: Circuit Breaker wraps Retry
```go
resile.NewPolicy(
    resile.WithCircuitBreaker(cb),
    resile.WithRetry(3),
)
```
The circuit breaker only sees a failure if the *entire* retry loop fails. This is useful when you expect transient failures that retries can fix, and you only want to open the circuit if the service is truly unreachable even after multiple attempts.

## Reusable and Thread-Safe

Policies created with `NewPolicy` are immutable, thread-safe, and designed to be reused across your entire application. You can define your "Standard Production Policy" once and inject it into all your service clients.

```go
// Reusable policy
var MySQLPolicy = resile.NewPolicy(
    resile.WithRetry(3),
    resile.WithTimeout(500 * time.Millisecond),
)

// Use it everywhere
val, err := MySQLPolicy.Do(ctx, func(ctx context.Context) (any, error) {
    return db.QueryRow(...)
})
```

## Conclusion

The new Policy Composition API brings a new level of flexibility to `resile`, allowing Go developers to build sophisticated, layered defense strategies with the same ergonomic, type-safe API they've come to love.

Check out the [README.md](../../README.md) for a full list of available policy options!
