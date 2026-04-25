# Stop the Domino Effect: Bulkhead Isolation in Go

In a distributed system, failure is inevitable. But a failure in one part of your system shouldn't bring down everything else. 

Imagine your Go service depends on three different downstream APIs: Payments, Inventory, and Recommendations. Suddenly, the Recommendations API starts taking 30 seconds to respond. If your service doesn't have isolation, your goroutines will start piling up waiting for Recommendations. Eventually, you'll hit your process limit, and even the critical Payments API calls will start failing because there are no resources left to handle them.

This is the **Domino Effect**, and the **Bulkhead Pattern** is how you stop it.

---

## The Problem: Resource Exhaustion

When one dependency slows down, it consumes resources:
- **Goroutines**: Blocked waiting for a response.
- **Memory**: Each blocked goroutine carries a stack.
- **File Descriptors/Sockets**: Open connections to the slow service.

Without a bulkhead, a single slow dependency can "starve" the rest of your application, leading to a total system collapse.

---

## The Solution: The Bulkhead Pattern

Named after the partitioned sections of a ship's hull, a **Bulkhead** isolates failures. If one section of the ship is flooded, the others remain buoyant. In software, we achieve this by limiting the number of concurrent executions allowed for a specific resource or dependency.

### Implementing with Resile:

[Resile](https://github.com/cinar/resile) makes it trivial to add bulkhead isolation to any operation.

```go
// Allow only 10 concurrent calls to this specific operation.
// If an 11th call comes in, it fails fast with resile.ErrBulkheadFull.
err := resile.DoErr(ctx, action, 
    resile.WithBulkhead(10),
)
```

### Using a Shared Bulkhead

Often, you want to limit concurrency across multiple different call sites that hit the same downstream service. You can create a shared `Bulkhead` instance for this:

```go
// Create a shared bulkhead for the "Inventory Service"
inventoryBulkhead := resile.NewBulkhead(20)

// Call Site A
resile.DoErr(ctx, fetchItem, resile.WithBulkheadInstance(inventoryBulkhead))

// Call Site B
resile.DoErr(ctx, updateStock, resile.WithBulkheadInstance(inventoryBulkhead))
```

By sharing the instance, you ensure that the *total* concurrency hitting the Inventory Service never exceeds 20, regardless of which part of your code is making the call.

---

## Static vs. Adaptive Bulkheads

Traditional bulkheads are **static**. You pick a number (like 20) and hope it stays the right choice as your traffic and infrastructure change.

But what if your downstream service is sometimes fast and sometimes slow? Or what if you move your database to a faster region? In these cases, a static limit might be too conservative (wasting capacity) or too aggressive (causing queuing).

### The Dynamic Alternative: Adaptive Concurrency

Resile also provides an **Adaptive Concurrency Limiter** (TCP-Vegas style). It automatically discovers the optimal concurrency limit by monitoring Round-Trip Time (RTT). It increases the limit when latency is stable and decreases it multiplicatively when queuing is detected.

```go
// Shared limiter that dynamically adjusts its capacity
al := resile.NewAdaptiveLimiter()

err := resile.DoErr(ctx, action, 
    resile.WithAdaptiveLimiterInstance(al),
)
```

If your infrastructure is highly dynamic, consider using the `AdaptiveLimiter` as a "smart bulkhead" that tunes itself in real-time.

[Read more: Beyond Static Limits: Adaptive Concurrency with TCP-Vegas in Go](adaptive-concurrency.md)

---

## Practical Application: Database Connection Pools

A common use case for shared bulkheads is protecting database connection pools (SQL or NoSQL like Redis). By using a bulkhead that matches your pool size, you ensure that your application never blocks indefinitely on the pool itself.

[Read more: Reliable Redis: Combining Retries and Bulkheads for Rock-Solid Caching](redis-resilience-with-go.md)

---

## Why "Fail-Fast" Matters

When a bulkhead is full, Resile immediately returns `resile.ErrBulkheadFull`. 

This is much better than waiting for a timeout. By failing fast, you:
1. **Preserve Resources**: You don't spawn another goroutine or open another connection.
2. **Provide Immediate Feedback**: Your upstream callers get an error instantly and can decide how to handle it (e.g., show a cached result or a "service busy" message).

---

## Observability: Monitoring the Walls

You need to know when your bulkheads are working. If a bulkhead is frequently full, it might mean your downstream service is struggling, or you need to re-evaluate your capacity limits.

If you use Resile's telemetry integrations (like `slog` or `OpenTelemetry`), you'll get automatic alerts when a bulkhead saturates. The `OnBulkheadFull` event is triggered every time a request is rejected due to capacity limits.

---

## Conclusion

Bulkheads are a fundamental building block of resilient systems. By isolating your dependencies, you ensure that a local fire doesn't become a global conflagration.

Resile provides a clean, "Go-native" way to implement bulkheads without complex boilerplate, allowing you to focus on your business logic while keeping your system stable.

**Explore Resile on GitHub:** [github.com/cinar/resile](https://github.com/cinar/resile)

#golang #microservices #resilience #bulkhead #concurrency #backend #distributed-systems
