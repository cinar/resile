# Beyond Static Limits: Adaptive Concurrency with TCP-Vegas in Go

Traditional concurrency limits (like bulkheads) are static. You pick a number—say, 10 concurrent requests— and hope for the best. But in the dynamic world of cloud infrastructure, "10" might be too conservative when the network is fast, or dangerously high when a downstream service starts to queue.

Static limits require manual tuning, which is often done *after* an outage has already happened. To build truly resilient systems, we need **Adaptive Concurrency Control**.

Here is how to implement dynamic concurrency limits in Go using [Resile](https://github.com/cinar/resile), inspired by the TCP-Vegas congestion control algorithm.

---

## The Problem: The "Fixed-Limit" Trap

Imagine your service talks to a database. You've set a bulkhead limit of 50 concurrent connections. 
- **Scenario A (Normal):** Database latency is 10ms. 50 concurrent requests mean you're handling 5,000 RPS. Everything is fine.
- **Scenario B (Degraded):** Database latency spikes to 500ms due to a background maintenance task. Your 50 "slots" are now filled with slow requests. Your throughput drops to 100 RPS, and new incoming requests start to pile up in your own service's memory, eventually leading to a cascade of failures.

In Scenario B, 50 is **too many**. You're holding onto resources that are essentially waiting on a bottleneck. You should have reduced your concurrency limit to prevent your own service from becoming part of the problem.

---

## The Solution: Little's Law & TCP-Vegas

Adaptive Concurrency uses two core principles:
1. **Little's Law ($L = \lambda W$):** The number of items in a system ($L$) is equal to the arrival rate ($\lambda$) multiplied by the average time an item spends in the system ($W$).
2. **TCP-Vegas AIMD:** An Additive Increase, Multiplicative Decrease (AIMD) logic based on Round-Trip Time (RTT).

### How it works:
- **Baseline:** The algorithm tracks the minimum RTT (the fastest the system can possibly go).
- **Additive Increase:** If current latency is close to the baseline (no queuing detected), it cautiously increases the concurrency limit by 1.
- **Multiplicative Decrease:** If latency spikes above a threshold (e.g., $1.5 \times$ baseline), it assumes queuing is happening downstream and immediately slashes the concurrency limit by 20%.

This allows your service to automatically "breathe" with the network. It expands to use available capacity when things are fast and contracts instantly to protect itself when things slow down.

---

## Implementing with Resile

Resile makes it trivial to add adaptive concurrency to your Go services.

```go
// 1. Create a shared AdaptiveLimiter.
// This should be shared across multiple calls to the same resource.
al := resile.NewAdaptiveLimiter()

// 2. Use it in your policy.
p := resile.NewPolicy(
    resile.WithAdaptiveLimiterInstance(al),
)

// 3. Execute your action.
err := p.DoErr(ctx, func(ctx context.Context) error {
    return callDownstreamService()
})

if errors.Is(err, resile.ErrShedLoad) {
    // The limiter has dynamically reduced the limit and shed this request
    // to protect the system.
}
```

### Why "TCP-Vegas"?

Unlike other congestion control algorithms (like TCP-Reno) that wait for packet loss to react, TCP-Vegas reacts to **latency changes**. This is perfect for microservices where "packet loss" usually means a timed-out request or a 503 error—both of which we want to avoid *before* they happen.

---

## Zero-Configuration Resilience

One of the biggest benefits of Adaptive Concurrency is that it requires **zero manual configuration**. You don't need to know if your database can handle 50 or 500 connections. The `AdaptiveLimiter` will discover the optimal limit in real-time.

It even handles "Network Drift." Over time, the minimum baseline RTT is gradually decayed, allowing the system to recalibrate if you migrate your database to a faster region or if the network topology changes.

---

## Conclusion

Resilience isn't just about surviving failures; it's about **adapting** to them. By moving from static bulkheads to adaptive concurrency, you're building a system that can intelligently protect itself from cascading failures while maximizing throughput during "peace time."

Check out the [Adaptive Concurrency Example](https://github.com/cinar/resile/tree/main/examples/adaptiveconcurrency) in the Resile repository to see it in action.
