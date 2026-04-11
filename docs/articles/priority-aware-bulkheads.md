# Prioritize Your Traffic: Priority-Aware Bulkheads in Go

Not all traffic is created equal. When your system is under heavy load, should a background cleanup task compete for the same resources as a user's checkout request? 

In a standard bulkhead, the answer is often "yes"—the first 10 requests get in, and the 11th is rejected, regardless of its importance. This is where **Priority-Aware Bulkheads** come in.

---

## The Problem: The "Fairness" Trap

Standard bulkheads are fair. They treat every request the same. But in a real-world system, fairness can be a liability:
- **Critical Traffic**: User-facing requests (e.g., "Complete Purchase", "Login") that directly impact revenue or user experience.
- **Standard Traffic**: Regular API calls (e.g., "View Profile", "Search") that are important but not immediately critical.
- **Low-Priority Traffic**: Background tasks (e.g., "Generate Report", "Sync Analytics", "Cache Warming") that can be delayed or retried later.

When your system is at 90% capacity, you want to stop accepting "Generate Report" requests to ensure there's enough room for "Complete Purchase" calls. A standard bulkhead can't do this; it will fill up with whatever arrives first.

---

## The Solution: Priority-Aware Bulkheads

A **Priority-Aware Bulkhead** uses **Load Shedding** based on priority levels. It defines utilization thresholds for different types of traffic. 

For example:
- **Low Priority**: Allowed only if the bulkhead is less than 50% full.
- **Standard Priority**: Allowed only if the bulkhead is less than 80% full.
- **Critical Priority**: Allowed until the bulkhead is 100% full.

This ensures that your most important traffic always has a "buffer" of capacity reserved for it, even when the system is under significant pressure.

---

## Implementing with Resile

[Resile](https://github.com/cinar/resile) provides a built-in `PriorityBulkhead` that makes this pattern easy to implement.

### 1. Define Your Priorities

Resile uses a simple `Priority` type with three levels: `PriorityLow`, `PriorityStandard`, and `PriorityCritical`.

```go
thresholds := map[resile.Priority]float64{
    resile.PriorityLow:      0.5, // Shed at 50% utilization
    resile.PriorityStandard: 0.8, // Shed at 80% utilization
    resile.PriorityCritical: 1.0, // Shed only when 100% full
}

// Create a bulkhead with a capacity of 20
pb := resile.NewPriorityBulkhead(20, thresholds)
```

### 2. Attach Priority to Context

You communicate the importance of a request by attaching a priority to its `context.Context`.

```go
// Create a context with Critical priority
ctx := resile.WithPriority(context.Background(), resile.PriorityCritical)

// Execute the action within the priority bulkhead
err := pb.Execute(ctx, func() error {
    return processOrder()
})
```

### 3. Handle Shedded Load

When a request is rejected because its priority threshold is exceeded, Resile returns `resile.ErrShedLoad`. If the bulkhead is physically full (100% capacity), it returns `resile.ErrBulkheadFull`.

```go
if errors.Is(err, resile.ErrShedLoad) {
    // This low/standard priority request was shedded to save capacity 
    // for higher-priority traffic.
}
```

---

## Why Use Priority-Aware Bulkheads?

1. **Protect the Critical Path**: Ensure that your most important business processes remain available even during traffic spikes.
2. **Graceful Degradation**: Instead of a total system failure, your service gracefully degrades by dropping non-essential background work first.
3. **Better User Experience**: Users performing critical actions see no slowdown, while background "noise" is managed behind the scenes.
4. **Cost Efficiency**: You don't need to over-provision your infrastructure to handle peak "background" load if you can simply shed it when necessary.

---

## Comparison: Static vs. Priority vs. Adaptive

| Feature | Static Bulkhead | Priority-Aware Bulkhead | Adaptive Concurrency |
| :--- | :--- | :--- | :--- |
| **Limit Type** | Fixed (e.g., 20) | Fixed + Thresholds | Dynamic (Auto-tuned) |
| **Traffic Awareness** | None (All equal) | High (Priority-based) | None (All equal) |
| **Best For** | Simple isolation | Multi-tenant or Tiered apps | Volatile environments |

[Read more about Static Bulkheads](bulkhead-isolation.md) or [Explore Adaptive Concurrency](adaptive-concurrency.md).

---

## Conclusion

Resilience isn't just about keeping the lights on; it's about keeping the *right* lights on. Priority-Aware Bulkheads give you the surgical precision needed to manage your system's resources effectively during times of stress.

**Check out the full example:** [Priority Bulkhead Example](https://github.com/cinar/resile/tree/main/examples/prioritybulkhead)

**Learn more about Resile:** [github.com/cinar/resile](https://github.com/cinar/resile)

#golang #microservices #resilience #bulkhead #priority #loadshedding #backend
