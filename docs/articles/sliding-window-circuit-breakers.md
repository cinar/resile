# Resilience Beyond Counters: Sliding Window Circuit Breakers in Go

Imagine your service is recovering from a major outage. The downstream database was down for 10 minutes, but it's finally back up. Your circuit breaker, which has been "Open" and failing fast, detects the timeout has expired and switches to "Half-Open". 

Suddenly, **thousands of queued requests** slam into the database at the exact same moment. The database, still fragile and warming up its caches, collapses again. This is the **Thundering Herd**, and it's the nightmare scenario for any distributed systems engineer.

Standard circuit breakers often fail because they are too "jittery" or too "all-or-nothing." In this article, we'll explore why **Sliding Window Circuit Breakers** are the mathematically superior choice for production Go services and how [Resile](https://github.com/cinar/resile) implements them to provide a smooth recovery path.

---

## The "Memory" Problem: Why Simple Counters Fail

A basic circuit breaker often uses a simple counter: "If 5 requests fail in a row, trip the circuit." While better than nothing, this approach has two fatal flaws:

1.  **The "Success-Reset" Trap**: If you have 4 failures followed by 1 lucky success, a simple counter resets to zero. However, your service still has an **80% failure rate**, which is clearly a systemic issue.
2.  **No Context**: 5 failures out of 5 calls (100% failure) is a crisis. 5 failures out of 10,000 calls (0.05% failure) is background noise. A simple counter can't tell the difference.

## The Solution: Sliding Windows

A sliding window maintains a "rolling history" of recent calls. Instead of looking at consecutive failures, it looks at the **failure percentage** over a specific window of time or count.

### 1. Count-based Sliding Window (The Precision Tool)
This window tracks the last `N` calls (e.g., the last 100 requests). It uses a **Ring Buffer** (circular array) to store results. As the 101st request comes in, it overwrites the 1st request.

*   **Pros**: Highly predictable. It reacts based on actual traffic volume.
*   **Cons**: In low-traffic periods, the window might contain "stale" data from an hour ago.

### 2. Time-based Sliding Window (The Real-time Tool)
This window tracks calls in the last `T` seconds (e.g., the last 60 seconds). Resile divides this duration into small **Buckets**. As time moves forward, the oldest buckets are dropped.

*   **Pros**: Excellent for low-traffic services. It ensures the circuit breaker only reacts to *current* conditions.
*   **Cons**: Requires more complex "bucket" management to remain efficient.

---

## Taming the Thundering Herd: The Half-Open State

The most dangerous moment for a service is transitioning from **Open** (failing fast) back to **Closed** (allowing all traffic).

Resile solves the thundering herd problem using a controlled **Half-Open** state. When the `ResetTimeout` expires:
1.  The circuit doesn't just open the floodgates.
2.  It enters **Half-Open** and allows only a strictly limited number of "probes" (e.g., `HalfOpenMaxCalls = 10`).
3.  All other concurrent requests are still failed fast with `circuit.ErrCircuitOpen`.
4.  Only if those 10 probes succeed (or meet the success threshold) does the circuit transition back to **Closed**.

This "trial period" ensures that the downstream service can handle a small amount of traffic before being hit with the full load.

---

## Standalone Usage with Resile

Resile's circuit breaker is a standalone component in the `circuit` package. You can use it directly in your own logic without the full `resile` policy engine.

### The New `Execute` Signature

The `Execute` method provides a clean, context-aware way to wrap any action:

```go
import (
    "context"
    "github.com/cinar/resile/circuit"
)

func main() {
    cb := circuit.New(circuit.Config{
        WindowType:           circuit.WindowCountBased,
        WindowSize:           100,
        FailureRateThreshold: 50.0,
        ResetTimeout:         30 * time.Second,
    })

    ctx := context.Background()

    // The Execute method handles the state machine logic automatically
    err := cb.Execute(ctx, func() error {
        return callRemoteService()
    })

    if err == circuit.ErrCircuitOpen {
        // Handle fast-fail
    }
}
```

### Manual Intervention with `Reset()`

Sometimes, an operator knows the system is healthy before the timeout expires (e.g., after a manual database failover). You can manually reset the breaker to its healthy state:

```go
// Force the circuit to Closed state and clear all window history
cb.Reset()
```

---

## Testing Your Breaker: Chaos Engineering

How do you know your failure thresholds are tuned correctly? Instead of waiting for a real outage, you can use Resile's native **Chaos Engineering** features to synthetically trip your breaker.

By injecting a high `ErrorProbability` using the `WithChaos` option, you can verify exactly how your application behaves when the circuit opens and how gracefully it recovers during the `Half-Open` state.

[Read more: Native Chaos Engineering: Testing Resilience with Fault & Latency Injection](chaos-engineering.md)

---

## Efficiency Matters: Performance in Go

Implementing a sliding window can be expensive if not done carefully. Resile’s implementation is designed for high-performance Go services:

*   **Zero Allocations**: Once the breaker is initialized, executing an action performs **zero heap allocations**. The ring buffers and buckets are pre-allocated and reused.
*   **Constant Time Complexity**: Adding a result to the count-based window is $O(1)$. Calculating the failure rate is also $O(1)$ because we maintain a running sum.
*   **RWMutex Optimization**: We use a `sync.RWMutex` to allow concurrent requests to check the state while only locking during state transitions or window updates.

---

## Conclusion

A circuit breaker is more than a simple "if error, stop" check. It is a statistical shield that protects your infrastructure from cascading failures and thundering herds. By using Resile's sliding window implementation, you gain a mathematically rigorous way to handle outages with the performance your Go services require.

**Ready to build more resilient systems?**
Explore the full Resile project on GitHub: [github.com/cinar/resile](https://github.com/cinar/resile)

#golang #resilience #circuitbreaker #microservices #sre #distributedsystems
