# Preventing Microservice Meltdowns: Adaptive Retries and Circuit Breakers in Go

We’ve all been there. A downstream database has a momentary blip. Your service instances, being "resilient," immediately start retrying their failed requests. 

Suddenly, the database isn't just "having a blip" anymore—it’s being hammered by a self-inflicted DDoS attack from its own clients. This is the **Retry Storm** (or Thundering Herd), and it’s one of the most common ways distributed systems experience total meltdowns.

Standard exponential backoff protects individual services, but it doesn't protect the *cluster*. To do that, you need a layered defense-in-depth approach.

Here is how to prevent microservice meltdowns in Go using [Resile](https://github.com/cinar/resile).

---

## The Problem: Aggregate Load

Imagine you have 100 instances of your API. Each instance is configured to retry 3 times. If the database slows down, you suddenly have **300 extra requests** hitting it exactly when it's struggling to recover.

Even with jitter, the aggregate load can be enough to keep the database in a "failed" state indefinitely. To solve this, we need three patterns working together: **Adaptive Retries**, **Circuit Breakers**, and **Adaptive Concurrency Control**.

---

## 1. Adaptive Retries (The Token Bucket)

Inspired by Google's SRE book and AWS SDKs, **Adaptive Retries** use a client-side token bucket to "fail fast" locally.

The logic is simple:
- Every **success** adds a small amount of "credit" to your bucket.
- Every **retry** consumes a significant amount of credit.
- If the bucket is empty, Resile **stops retrying immediately** and fails fast locally.

This ensures that if a downstream service is fundamentally degraded, your fleet of clients will automatically throttle their retry pressure at the source, giving the service breathing room to recover.

### Implementing with Resile:

```go
// Share this bucket across multiple executions or even your entire service
bucket := resile.DefaultAdaptiveBucket()

err := resile.DoErr(ctx, action, 
    resile.WithAdaptiveBucket(bucket),
)
```

---

## 2. Circuit Breakers (The Kill Switch)

While retries assume "eventual success," a **Circuit Breaker** assumes "statistical failure." 

Resile implements a **Sliding Window** circuit breaker. If the failure rate of the last 100 calls exceeds 50%, the breaker "trips" (opens). For the next 30 seconds, every call to that service will fail **instantly** without even trying to hit the network. This protects your downstream infrastructure from useless traffic and saves your local resources (threads, memory, sockets).

### Standalone Usage:

```go
import "github.com/cinar/resile/circuit"

// Create a breaker with a sliding window
cb := circuit.New(circuit.Config{
    WindowType:           circuit.WindowCountBased,
    WindowSize:           100,
    FailureRateThreshold: 50.0,
    ResetTimeout:         30 * time.Second,
})

// Use the context-aware Execute method
err := cb.Execute(ctx, func() error {
    return callRemoteService()
})
```

### Layering in Resile Policies:

```go
err := resile.DoErr(ctx, action, 
    resile.WithCircuitBreaker(cb),
)
```

---

## 3. Adaptive Concurrency (The Buffer)

Inspired by TCP-Vegas, **Adaptive Concurrency Control** dynamically adjusts your concurrency limits based on Round-Trip Time (RTT). 

While a Circuit Breaker is an "all-or-nothing" kill switch, Adaptive Concurrency is a "sliding scale." It detects when your downstream dependencies are beginning to queue (latency rises) and slashes your concurrency limit to match the actual available capacity.

```go
// Create a shared limiter: discovery optimal limit in real-time
al := resile.NewAdaptiveLimiter()

err := resile.DoErr(ctx, action, 
    resile.WithAdaptiveLimiterInstance(al),
)
```

---

## The Ultimate Defense: Layered Resilience

The real power of Resile comes from combining these patterns. You can layer Retries, Circuit Breakers, Adaptive Buckets, and Adaptive Concurrency into a single execution strategy.

```go
err := resile.DoErr(ctx, action,
    resile.WithMaxAttempts(3),           // Layer 1: Handle random blips
    resile.WithCircuitBreaker(cb),      // Layer 2: Stop hitting a dead service
    resile.WithAdaptiveBucket(bucket),  // Layer 3: Prevent cluster-wide retry storms
    resile.WithAdaptiveLimiterInstance(al), // Layer 4: Match concurrency to downstream capacity
)
```

In this setup:
1. **Retries** handle the "one-off" network glitches.
2. **The Circuit Breaker** stops you from wasting time on a service that is clearly down.
3. **The Adaptive Bucket** ensures that even if the breaker hasn't tripped yet, you won't overwhelm the system with aggregate retry load.
4. **Adaptive Concurrency** prevents your own service from becoming a bottleneck when latency rises, intelligently shedding load before failures occur.

---

## Manual Recovery

If you know a service has recovered before the circuit breaker's timeout, you can manually reset it:

```go
// Force the circuit to Closed state
cb.Reset()
```

---

## Observability: Seeing the Shield in Action

Protecting your system is great, but *knowing* you’re being protected is better. 

If you use Resile's `slog` or `OpenTelemetry` integrations, you'll see exactly when these shields activate. Your logs will show `retry.throlled=true` when the adaptive bucket kicks in, or your traces will show a `circuit.open` error when the breaker prevents a call.

---

## Conclusion

Building resilient microservices isn't just about making individual calls "smarter." It's about ensuring that your entire architecture can survive a storm without collapsing under its own weight.

By combining opinionated retries, circuit breakers, and adaptive throttling, Resile gives you a production-grade resilience engine that scales with your infrastructure.

**Try Resile today:** [github.com/cinar/resile](https://github.com/cinar/resile)

#golang #microservices #sre #devops #backend #distributedsystems
