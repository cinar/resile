# Respecting Boundaries: Precise Rate Limiting in Go

Traffic spikes are a double-edged sword. On one hand, you’re busy! On the other, those spikes can overwhelm your services or exceed your downstream quotas. 

Whether you're protecting your own database from an unexpected burst or respecting a third-party API’s strict 100 requests-per-second (RPS) limit, you need a precise way to shape your traffic.

Enter the **Token Bucket Rate Limiter** in [Resile](https://github.com/cinar/resile).

---

## The Problem: Unbounded Traffic

In a distributed environment, your clients don't know about each other. If 50 different microservice instances all decide to call a downstream API at the same time, the aggregate traffic can easily exceed the capacity of the target system. 

When you exceed these limits, you'll often see:
- **HTTP 429 (Too Many Requests)**: Downstream services start rejecting you.
- **Cascading Latency**: The target system slows down for *everyone* because it's processing too many requests at once.
- **Cost Overruns**: Many cloud providers and SaaS APIs charge significant premiums for exceeding agreed-upon quotas.

---

## The Solution: The Token Bucket Algorithm

The **Token Bucket** is a classic algorithm used for traffic shaping. 

Imagine a bucket that refills with "tokens" at a constant rate (e.g., 100 tokens per second). Every request must consume a token from the bucket. If the bucket is empty, the request is rejected immediately. This allows for small "bursts" (filling the bucket) while maintaining a precise long-term average rate.

### Implementing with Resile:

Resile makes adding rate limiting to your executions simple.

```go
// Allow 100 requests per second.
// If the limit is exceeded, it fails fast with resile.ErrRateLimitExceeded.
err := resile.DoErr(ctx, action, 
    resile.WithRateLimiter(100, time.Second),
)
```

### Rate Limiting vs. Adaptive Retries

Wait, doesn't Resile already have `AdaptiveBucket`? What's the difference?

- **AdaptiveBucket** is *success-based*. It tracks how many requests are succeeding vs. failing and throttles *retries* accordingly. It's designed specifically to prevent "retry storms" when a service is failing.
- **RateLimiter** is *time-based*. It enforces a strict, constant quota of requests over a time interval. It’s designed for general traffic shaping and quota management.

For maximum protection, you can even use them together!

---

## Shared Rate Limiters

Often, you want to enforce a global rate limit across your entire service instance. You can create a shared `RateLimiter` and pass it to multiple executions:

```go
// Shared rate limiter for a specific API key or downstream service
limiter := resile.NewRateLimiter(50, time.Second)

// Each call will consume tokens from the same shared bucket.
err := resile.DoErr(ctx, myAction, 
    resile.WithRateLimiterInstance(limiter),
)
```

---

## Observability: Seeing the Shaping

Knowing *when* and *why* your traffic is being throttled is essential for operational visibility. 

If you use Resile's telemetry integrations (like `slog` or `OpenTelemetry`), you'll get automatic visibility into these events. The `OnRateLimitExceeded` event is triggered whenever a request is rejected by the rate limiter, allowing you to monitor your quota utilization in real-time.

---

## Conclusion

Rate limiting is not just about saying "no"; it's about being a good citizen in a distributed ecosystem. By respecting boundaries and shaping your traffic at the source, you protect both your own service and the systems you depend on.

Resile provides a production-grade rate limiter that integrates seamlessly into your resilience policies, giving you fine-grained control over your traffic flow.

**Learn more about Resile:** [github.com/cinar/resile](https://github.com/cinar/resile)

#golang #microservices #ratelimiting #trafficshaping #sre #devops #backend
