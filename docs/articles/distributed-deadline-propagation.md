# Stopping the Zombie Requests: Distributed Deadline Propagation in Go

Imagine a common scenario in a microservice architecture: A user clicks a "Buy" button, triggering a request to **Service A**. Service A calls **Service B**, which in turn calls **Service C**.

Suddenly, Service A times out. The user sees an error message and refreshes the page. But **Service B and Service C are still working** on the original request, consuming CPU, memory, and database connections for a result that will never be seen.

These are **Zombie Requests**. In a high-traffic system, they can lead to cascading failures and resource exhaustion, even if the underlying services are technically "healthy."

To stop the zombies, you need **Distributed Deadline Propagation**. Here is how to implement it effortlessly using [Resile](https://github.com/cinar/resile).

---

## What is Distributed Deadline Propagation?

Deadlines are not just local timeouts. A deadline represents the **absolute point in time** after which the entire request chain should be abandoned.

Distributed Deadline Propagation is the process of:
1.  **Tracking** the remaining time (the "budget") as a request moves through the system.
2.  **Communicating** that budget to downstream services via metadata (like HTTP headers).
3.  **Aborting early** if the remaining budget is too small to realistically complete the work.

---

## The Resile Way: Smart Deadlines

Resile provides two powerful mechanisms to handle distributed deadlines: **Early Abort** and **Header Injection**.

### 1. Early Abort: `WithMinDeadlineThreshold`

Why start a request if you only have 2 milliseconds left? The network latency alone will likely exceed that, and you'll just be wasting resources.

Resile's `WithMinDeadlineThreshold` allows you to define a "safety buffer." If the remaining time in the `context.Context` is less than this threshold, Resile will **abort the execution immediately** with a `context.DeadlineExceeded` error, before even attempting the work.

```go
import (
    "context"
    "time"
    "github.com/cinar/resile"
)

// Define a policy with a 10ms "Early Abort" threshold.
policy := resile.NewPolicy(
    resile.WithRetry(3),
    resile.WithMinDeadlineThreshold(10 * time.Millisecond),
)

// If ctx has only 5ms left, this returns context.DeadlineExceeded instantly.
result, err := policy.Do(ctx, func(ctx context.Context) (string, error) {
    return apiClient.FetchData(ctx)
})
```

### 2. Header Injection: `InjectDeadlineHeader`

To propagate the deadline to downstream services, you need to "inject" the remaining time into your outgoing requests. Resile provides a transport-agnostic `InjectDeadlineHeader` function that supports both standard HTTP and gRPC.

#### For REST/HTTP:
You can inject the remaining milliseconds into a custom header (e.g., `X-Request-Timeout`).

```go
func (c *Client) FetchData(ctx context.Context) (string, error) {
    req, _ := http.NewRequestWithContext(ctx, "GET", "http://service-b/data", nil)

    // Inject the remaining milliseconds into the header.
    resile.InjectDeadlineHeader(ctx, req.Header, "X-Request-Timeout")

    resp, err := c.httpClient.Do(req)
    // ...
}
```

#### For gRPC:
Resile natively supports the standard `Grpc-Timeout` header format, ensuring compatibility with the gRPC ecosystem.

```go
func (c *Client) FetchData(ctx context.Context) (string, error) {
    md := metadata.New(map[string]string{})
    
    // Inject using the gRPC-specific format (e.g., "100m" for 100ms).
    resile.InjectDeadlineHeader(ctx, md, "Grpc-Timeout")
    
    ctx = metadata.NewOutgoingContext(ctx, md)
    return c.grpcClient.GetData(ctx, &pb.Request{})
}
```

---

## Why This Matters for Resilience

Without distributed deadlines, your system is vulnerable to **Resource Exhaustion Attacks**—not from malicious actors, but from your own retries and slow dependencies.

By implementing propagation:
*   **You save money:** You're not paying for cloud compute that produces "zombie" results.
*   **You prevent meltdowns:** Downstream services are protected from "retry storms" that they can't possibly satisfy in time.
*   **You improve UX:** Failures happen faster (Fail-Fast), allowing the UI to react or switch to a fallback immediately.

[Read more: Preventing Meltdowns: How Adaptive Retries Protect Your Downstream](preventing-meltdowns.md)

---

## Comparison: Static vs. Distributed Deadlines

| Feature | Static Timeouts | Distributed Deadlines (Resile) |
| :--- | :--- | :--- |
| **Scope** | Single Service | Entire Request Chain |
| **Awareness** | Blind to upstream delays | Aware of the total "time budget" |
| **Efficiency** | High waste (Zombie requests) | Zero waste (Early Abort) |
| **Protocol** | Internal only | HTTP/gRPC compatible |

---

## Conclusion

Resilience isn't just about making things "work"; it's about knowing when to **stop working**. 

Distributed Deadline Propagation is the "social contract" of a microservice architecture. It ensures that every service in the chain is working towards a common goal—and respects the reality that sometimes, time simply runs out.

With Resile, implementing this complex pattern becomes a matter of a few lines of configuration.

**Explore Resile on GitHub:** [github.com/cinar/resile](https://github.com/cinar/resile)

How are you handling request budgets in your distributed systems? Let's discuss!

#golang #microservices #distributedsystems #resilience #performance
