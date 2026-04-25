# Resile: Ergonomic Execution Resilience for Go

[![Go Reference](https://pkg.go.dev/badge/github.com/cinar/resile.svg)](https://pkg.go.dev/badge/github.com/cinar/resile)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](https://opensource.org/licenses/MIT)
[![Build Status](https://github.com/cinar/resile/actions/workflows/ci.yml/badge.svg)](https://github.com/cinar/resile/actions/workflows/ci.yml)
[![Codecov](https://codecov.io/gh/cinar/resile/branch/main/graph/badge.svg)](https://codecov.io/gh/cinar/resile)
[![Go Report Card](https://goreportcard.com/badge/github.com/cinar/resile)](https://goreportcard.com/report/github.com/cinar/resile)
[![YouTube](https://img.shields.io/badge/YouTube-@resile--go-red?logo=youtube)](https://www.youtube.com/@resile-go)
[![Dev.to](https://img.shields.io/badge/dev.to-onurcinar-0a0a0a?logo=dev.to&logoColor=white)](https://dev.to/onurcinar)

**Resile** is a production-grade execution resilience and retry library for Go, inspired by Python's [stamina](https://github.com/hynek/stamina). It provides a type-safe, ergonomic, and highly observable way to handle transient failures in distributed systems.

---

## See Resile in Action

[![Resile Demo Video](https://img.youtube.com/vi/OCNQ-8gq0k8/0.jpg)](https://www.youtube.com/watch?v=OCNQ-8gq0k8)

Subscribe to our [YouTube channel](https://www.youtube.com/@resile-go) for deep dives and tutorials.

---

## The "Aha!" Snippet

Resile allows you to compose complex resilience strategies with a single, type-safe call. No `interface{}` casting, no reflection—just clean Go.

```go
user, err := resile.Do(ctx, func(ctx context.Context) (*User, error) {
    return client.GetUser(ctx, id)
},
    resile.WithBulkhead(10),            // Max 10 concurrent requests
    resile.WithCircuitBreaker(cb),      // Stop if failure rate > 50%
    resile.WithRetry(3),               // 3 attempts with AWS Full Jitter
    resile.WithTimeout(1*time.Second),   // Each attempt max 1s
    resile.WithFallback(func(ctx context.Context, err error) (*User, error) {
        return cache.Get(id), nil       // Return stale data on failure
    }),
)
```

---

## Table of Contents
- [See Resile in Action](#see-resile-in-action)
- [Installation](#installation)
- [Why Resile?](#why-resile)
- [Articles & Tutorials](#articles--tutorials)
- [Examples](#examples)
- [Resilience Cookbook](#resilience-cookbook)
  - [Simple Retries](#1-simple-retries)
  - [Value-Yielding Retries (Generics)](#2-value-yielding-retries-generics)
  - [Request Hedging (Speculative Retries)](#3-request-hedging-speculative-retries)
  - [Stateful Retries & Endpoint Rotation](#4-stateful-retries--endpoint-rotation)
  - [Handling Rate Limits (Retry-After)](#5-handling-rate-limits-retry-after)
  - [Aborting Retries (Pushback Signal)](#6-aborting-retries-pushback-signal)
  - [Fallback Strategies](#7-fallback-strategies)
  - [Bulkhead Pattern](#8-bulkhead-pattern)
  - [Priority-Aware Bulkhead](#9-priority-aware-bulkhead)
  - [Rate Limiting Pattern](#10-rate-limiting-pattern)
  - [Layered Defense with Circuit Breaker](#11-layered-defense-with-circuit-breaker)
  - [Macro-Level Protection (Adaptive Retries)](#12-macro-level-protection-adaptive-retries)
  - [Adaptive Concurrency (TCP-Vegas)](#13-adaptive-concurrency-tcp-vegas)
  - [Structured Logging & Telemetry](#14-structured-logging--telemetry)
  - [Panic Recovery ("Let It Crash")](#15-panic-recovery-let-it-crash)
  - [Fast Unit Testing](#16-fast-unit-testing)
  - [Reusable Clients & Dependency Injection](#17-reusable-clients--dependency-injection)
  - [Marking Errors as Fatal](#18-marking-errors-as-fatal)
  - [Custom Error Filtering](#19-custom-error-filtering)
  - [Policy Composition & Chaining](#20-policy-composition--chaining)
  - [Native Multi-Error Aggregation](#21-native-multi-error-aggregation)
  - [Native Chaos Engineering (Fault & Latency Injection)](#22-native-chaos-engineering-fault--latency-injection)
  - [Distributed Deadline Propagation](#23-distributed-deadline-propagation)
  - [Reliable File Downloads (HTTP Resumption)](#24-reliable-file-downloads-http-resumption)
  - [SQL Resilience](#25-sql-resilience)
  - [Redis Resilience](#26-redis-resilience)
- [Built on Hyperscaler Research](#built-on-hyperscaler-research)
- [Configuration Reference](#configuration-reference)
- [Architecture & Design](#architecture--design)
- [License](#license)

---

## Installation

```bash
go get github.com/cinar/resile
```

---

## Why Resile?

In distributed systems, transient failures are a mathematical certainty. Resile simplifies the "Correct Way" to retry:
- **AWS Full Jitter**: Uses the industry-standard algorithm to prevent "thundering herd" synchronization.
- **Adaptive Retries**: Built-in token bucket rate limiting to prevent "retry storms" across a cluster.
- **Generic-First**: No `interface{}` or reflection. Full compile-time type safety.
- **Context-Aware**: Strictly respects `context.Context` cancellation and deadlines.
- **Zero-Dependency Core**: The core library only depends on the Go standard library.
- **Opinionated Defaults**: Sensible production-ready defaults (5 attempts, exponential backoff).
- **Chaos-Ready**: Built-in support for fault and latency injection to test your resilience policies.

---

## Articles & Tutorials

Want to learn more about the philosophy behind Resile and advanced resilience patterns in Go? Check out these deep dives:

* [Stop Writing Manual Retry Loops in Go: Why Your Current Logic is Probably Dangerous](docs/articles/stop-writing-manual-loops.md)
* [Python's Stamina for Go: Bringing Ergonomic Resilience to Gophers](docs/articles/python-stamina-for-go.md)
* [Beating Tail Latency: A Guide to Request Hedging in Go Microservices](docs/articles/beating-tail-latency.md)
* [Preventing Microservice Meltdowns: Adaptive Retries and Circuit Breakers in Go](docs/articles/preventing-meltdowns.md)
* [Self-Healing State Machines: Resilient State Transitions in Go](docs/articles/self-healing-state-machines.md)
* [Resilience Beyond Counters: Sliding Window Circuit Breakers in Go](docs/articles/sliding-window-circuit-breakers.md)
* [Stop the Domino Effect: Bulkhead Isolation in Go](docs/articles/bulkhead-isolation.md)
* [Reliable Redis: Combining Retries and Bulkheads for Rock-Solid Caching](docs/articles/redis-resilience-with-go.md)
* [Prioritize Your Traffic: Priority-Aware Bulkheads in Go](docs/articles/priority-aware-bulkheads.md)
* [Respecting Boundaries: Precise Rate Limiting in Go](docs/articles/rate-limiting.md)
* [Beyond Static Limits: Adaptive Concurrency with TCP-Vegas in Go](docs/articles/adaptive-concurrency.md)
* [Debugging the Timeline: Native Multi-Error Aggregation in Go](docs/articles/native-multi-error-aggregation.md)
* [Native Chaos Engineering: Testing Resilience with Fault & Latency Injection](docs/articles/chaos-engineering.md)
* [Stopping the Zombie Requests: Distributed Deadline Propagation in Go](docs/articles/distributed-deadline-propagation.md)
* [Reliable File Downloads with HTTP Range Resumption](docs/articles/streaming-http-resumption.md)
* [Building Bulletproof Database Clients in Go: SQL Resilience with Resile](docs/articles/sql-resilience.md)

Also, check out our [Dev.to space](https://dev.to/onurcinar) for more articles and discussions.

---
## Examples

The [examples/](examples/) directory contains standalone programs showing how to use Resile in various scenarios:

- **[Basic Retry](examples/basic/main.go)**: Simple `Do` and `DoErr` calls.
- **[Request Hedging](examples/hedging/main.go)**: Reducing tail latency with speculative retries.
- **[HTTP with Rate Limits](examples/http/main.go)**: Respecting `Retry-After` headers and using `slog`.
- **[Fallback Strategies](examples/fallback/main.go)**: Returning stale data when all attempts fail.
- **[Stateful Rotation](examples/stateful/main.go)**: Rotating API endpoints using `RetryState`.
- **[Circuit Breaker](examples/circuitbreaker/main.go)**: Layering defensive strategies.
- **[Adaptive Retries](examples/adaptiveretry/main.go)**: Preventing retry storms with a token bucket.
- **[Adaptive Concurrency](examples/adaptiveconcurrency/main.go)**: Dynamic concurrency limits based on latency (TCP-Vegas).
- **[Priority Bulkhead](examples/prioritybulkhead/main.go)**: Load shedding based on traffic priority.
- **[Pushback Signal](examples/pushback/main.go)**: Aborting retries immediately using `CancelAllRetries`.
- **[Panic Recovery](examples/panicrecovery/main.go)**: Implementing Erlang's "Let It Crash" philosophy.
- **[State Machine](examples/statemachine/main.go)**: Building resilient state machines inspired by Erlang's `gen_statem`.
- **[Chaos Injection](examples/chaos/main.go)**: Simulating faults and latency to test your policies.
- **[HTTP Resumption](examples/http_resume_stream/main.go)**: Resuming large file downloads using HTTP Range.
- **[SQL Resilience](examples/sql/main.go)**: Using Resile with standard `database/sql`.
- **[Redis Resilience](examples/redis/main.go)**: Adding resilience to Redis operations with shared bulkheads.

---

## Resilience Cookbook

### 1. Simple Retries
**The Problem**: A database connection or network request might fail intermittently due to transient blips.

**The Recipe**:
Retry a simple operation that only returns an error. If all retries fail, Resile returns an aggregated error containing the failures from every attempt.

```go
err := resile.DoErr(ctx, func(ctx context.Context) error {
    return db.PingContext(ctx)
})
```

### 2. Value-Yielding Retries (Generics)
**The Problem**: Fetching data from a microservice needs to be resilient and type-safe without boilerplate casting.

**The Recipe**:
Fetch data with full type safety. The return type is inferred from your closure.

```go
// val is automatically inferred as *User
user, err := resile.Do(ctx, func(ctx context.Context) (*User, error) {
    return apiClient.GetUser(ctx, userID)
}, resile.WithMaxAttempts(3))
```

### 3. Request Hedging (Speculative Retries)
**The Problem**: Long-tail latency (the 99th percentile) slows down your entire system even when most requests are fast.

**The Recipe**:
Speculative retries reduce tail latency by starting a second request if the first one doesn't finish within a configured `HedgingDelay`. The first successful result is used, and the other is cancelled.

```go
// For value-yielding operations
data, err := resile.DoHedged(ctx, action, 
    resile.WithMaxAttempts(3),
    resile.WithHedgingDelay(100 * time.Millisecond),
)

// For error-only operations
err := resile.DoErrHedged(ctx, action,
    resile.WithMaxAttempts(2),
    resile.WithHedgingDelay(50 * time.Millisecond),
)
```

[Read more: Beating Tail Latency: A Guide to Request Hedging in Go Microservices](docs/articles/beating-tail-latency.md)

### 4. Stateful Retries & Endpoint Rotation
**The Problem**: Retrying against the same failing endpoint is futile; you need to cycle through a list of healthy hosts.

**The Recipe**:
Use `DoState` (or `DoErrState`) to access the `RetryState`, allowing you to rotate endpoints or fallback logic based on the failure history.

```go
endpoints := []string{"api-v1.example.com", "api-v2.example.com"}

data, err := resile.DoState(ctx, func(ctx context.Context, state resile.RetryState) (string, error) {
    // Rotate endpoint based on attempt number
    url := endpoints[state.Attempt % uint(len(endpoints))]
    return client.Get(ctx, url)
})
```

[Read more: Self-Healing State Machines: Resilient State Transitions in Go](docs/articles/self-healing-state-machines.md)

### 5. Handling Rate Limits (Retry-After)
**The Problem**: Downstream services may return `429 Too Many Requests` with a specific time you must wait.

**The Recipe**:
Resile automatically detects if an error implements `RetryAfterError`. It can override the jittered backoff with a server-dictated duration.

```go
type RateLimitError struct {
    WaitUntil time.Time
}

func (e *RateLimitError) Error() string { return "too many requests" }
func (e *RateLimitError) RetryAfter() time.Duration {
    return time.Until(e.WaitUntil)
}
func (e *RateLimitError) CancelAllRetries() bool {
    return false 
}

// Resile will sleep exactly until WaitUntil when this error is encountered.
```

### 6. Aborting Retries (Pushback Signal)
**The Problem**: Some errors are terminal (e.g., "Quota Exceeded") and retrying them just wastes resources and adds latency.

**The Recipe**:
Implement `CancelAllRetries() bool` to abort the entire retry loop immediately.

```go
type QuotaExceededError struct{}
func (e *QuotaExceededError) Error() string { return "quota exhausted" }
func (e *QuotaExceededError) CancelAllRetries() bool { return true }

// Resile will stop immediately if this error is encountered.
_, err := resile.Do(ctx, action, resile.WithMaxAttempts(10))
```

### 7. Fallback Strategies
**The Problem**: When a service is down, your application should degrade gracefully instead of failing completely.

**The Recipe**:
Provide a fallback function to return stale data from a cache or a sensible default value when all attempts fail.

```go
data, err := resile.Do(ctx, fetchData,
    resile.WithMaxAttempts(3),
    resile.WithFallback(func(ctx context.Context, err error) (string, error) {
        return cache.Get(ctx, key), nil 
    }),
)
```

### 8. Bulkhead Pattern
**The Problem**: One slow or failing resource (like a specific database) consumes all available goroutines, causing a "cascading failure" across the whole app.

**The Recipe**:
Isolate failures by limiting the number of concurrent executions to a specific resource.

```go
// Shared bulkhead with capacity of 10
bh := resile.NewBulkhead(10)

err := resile.DoErr(ctx, action, resile.WithBulkheadInstance(bh))
```

[Read more: Stop the Domino Effect: Bulkhead Isolation in Go](docs/articles/bulkhead-isolation.md)

### 9. Priority-Aware Bulkhead
**The Problem**: During high load, you want to ensure critical user requests succeed even if background tasks are dropped.

**The Recipe**:
Implement load shedding based on traffic priority to protect critical paths during saturation.

```go
thresholds := map[resile.Priority]float64{
    resile.PriorityLow:      0.5, // Shed when >50% full
    resile.PriorityStandard: 0.8, // Shed when >80% full
    resile.PriorityCritical: 1.0, // Allow until 100% full
}

err := resile.DoErr(ctx, action, 
    resile.WithPriorityBulkhead(20, thresholds),
)
```

[Read more: Prioritize Your Traffic: Priority-Aware Bulkheads in Go](docs/articles/priority-aware-bulkheads.md)

### 10. Rate Limiting Pattern
**The Problem**: Your application must strictly adhere to an external API's rate limits (e.g., 100 requests/sec).

**The Recipe**:
Control the rate of executions using a time-based token bucket.

```go
// Limit to 100 requests per second
rl := resile.NewRateLimiter(100, time.Second)

err := resile.DoErr(ctx, action, resile.WithRateLimiterInstance(rl))
```

[Read more: Respecting Boundaries: Precise Rate Limiting in Go](docs/articles/rate-limiting.md)

### 11. Layered Defense with Circuit Breaker
**The Problem**: Retrying against a service experiencing a total outage just adds load and delays failure detection.

**The Recipe**:
Combine retries with a mathematically rigorous sliding window circuit breaker. Resile supports both **Count-based** and **Time-based** windows.

```go
import "github.com/cinar/resile/circuit"

// Trip if >50% of the last 100 calls fail.
cb := circuit.New(circuit.Config{
    WindowType:           circuit.WindowCountBased,
    WindowSize:           100,
    FailureRateThreshold: 50.0,
    MinimumCalls:         10,
    ResetTimeout:         30 * time.Second,
})

err := resile.DoErr(ctx, action, resile.WithCircuitBreaker(cb))
```

[Read more: Resilience Beyond Counters: Sliding Window Circuit Breakers in Go](docs/articles/sliding-window-circuit-breakers.md)

### 12. Macro-Level Protection (Adaptive Retries)
**The Problem**: A "retry storm" occurs when hundreds of clients all retry at once, preventing a struggling service from recovering.

**The Recipe**:
Use a token bucket shared across your entire client. If the service degrades, the bucket depletes, causing clients to fail fast locally.

```go
// Share this bucket across multiple executions
bucket := resile.DefaultAdaptiveBucket()

err := resile.DoErr(ctx, action, resile.WithAdaptiveBucket(bucket))
```

[Read more: Preventing Microservice Meltdowns: Adaptive Retries and Circuit Breakers in Go](docs/articles/preventing-meltdowns.md)

### 13. Adaptive Concurrency (TCP-Vegas)
**The Problem**: Setting static concurrency limits is hard; too high and you crash the server, too low and you waste throughput.

**The Recipe**:
Automatically adjust concurrency limits based on Round-Trip Time (RTT). Resile increases concurrency when latency is stable and decreases it when queuing is detected.

```go
// Shared limiter across multiple calls
al := resile.NewAdaptiveLimiter()

err := resile.DoErr(ctx, action, resile.WithAdaptiveLimiterInstance(al))
```

[Read more: Beyond Static Limits: Adaptive Concurrency with TCP-Vegas in Go](adaptive-concurrency.md)

### 14. Structured Logging & Telemetry
**The Problem**: You need to know when retries are happening and why, without cluttering your business logic.

**The Recipe**:
Integrate with `slog` or `OpenTelemetry` seamlessly.

```go
import "github.com/cinar/resile/telemetry/resileslog"

logger := slog.Default()
resile.Do(ctx, action, 
    resile.WithName("get-inventory"),
    resile.WithInstrumenter(resileslog.New(logger)),
)
```

### 15. Panic Recovery ("Let It Crash")
**The Problem**: A single unexpected panic in a request handler can take down an entire process.

**The Recipe**:
Convert Go panics into retryable errors, allowing your application to reset to a known good state.

```go
val, err := resile.Do(ctx, riskyAction, 
    resile.WithPanicRecovery(),
)
```

### 16. Fast Unit Testing
**The Problem**: Waiting for exponential backoff timers in CI/CD pipelines makes tests slow and flaky.

**The Recipe**:
Use `WithTestingBypass` to make all retries execute instantly during tests.

```go
func TestMyService(t *testing.T) {
    ctx := resile.WithTestingBypass(context.Background())
    err := service.Handle(ctx) // Retries 10 times instantly
}
```

### 17. Reusable Clients & Dependency Injection
**The Problem**: Passing individual retry parameters everywhere leads to inconsistent resilience policies.

**The Recipe**:
Use `resile.New()` to create a `Retryer` interface for consistent, reusable strategies.

```go
retryer := resile.New(
    resile.WithMaxAttempts(3),
    resile.WithBaseDelay(200 * time.Millisecond),
)

err := retryer.DoErr(ctx, service.Call)
```

### 18. Marking Errors as Fatal
**The Problem**: Some errors occur deep in your code that you know shouldn't be retried.

**The Recipe**:
Use `resile.FatalError()` to abort the retry loop immediately from within your action.

```go
err := resile.DoErr(ctx, func(ctx context.Context) error {
    if errors.Is(err, ErrAuthFailed) {
        return resile.FatalError(err)
    }
    return err
})
```

### 19. Custom Error Filtering
**The Problem**: You only want to retry on specific database errors (like `ErrConnReset`) but not on logic errors.

**The Recipe**:
Control which errors trigger a retry using `WithRetryIf` or `WithRetryIfFunc`.

```go
err := resile.DoErr(ctx, action,
    resile.WithRetryIf(ErrConnReset),
    // OR
    resile.WithRetryIfFunc(func(err error) bool {
        return isTransient(err)
    }),
)
```

### 20. Policy Composition & Chaining
**The Problem**: You need a specific execution order for your resilience layers (e.g., Timeout *inside* the Bulkhead).

**The Recipe**:
Use the `Policy` API. The order of options in `NewPolicy` determines the execution hierarchy from **outermost to innermost**.

```go
standardPolicy := resile.NewPolicy(
    resile.WithBulkhead(20),
    resile.WithCircuitBreaker(cb),
    resile.WithRetry(3),
    resile.WithTimeout(1*time.Second),
)

val, err := standardPolicy.Do(ctx, action)
```

### 21. Native Multi-Error Aggregation
**The Problem**: When a request fails after 5 retries, you usually only see the last error, losing the context of earlier failures.

**The Recipe**:
Resile aggregates the complete timeline of failures using Go 1.20's `errors.Join`.

```go
err := resile.DoErr(ctx, action, resile.WithMaxAttempts(3))

if multi, ok := err.(interface{ Unwrap() []error }); ok {
    for i, e := range multi.Unwrap() {
        fmt.Printf("Attempt %d failed: %v\n", i+1, e)
    }
}
```

[Read more: Debugging the Timeline: Native Multi-Error Aggregation in Go](docs/articles/native-multi-error-aggregation.md)

### 22. Native Chaos Engineering (Fault & Latency Injection)
**The Problem**: You don't know if your resilience policies actually work until a real production outage occurs.

**The Recipe**:
Safely test your policies by synthetically inducing faults and latency into your application logic.

```go
import "github.com/cinar/resile/chaos"

err := resile.DoErr(ctx, action, 
    resile.WithRetry(3),
    resile.WithChaos(chaos.Config{
        ErrorProbability: 0.1, 
        InjectedError:    errors.New("chaos!"),
    }),
)
```

[Read more: Native Chaos Engineering: Testing Resilience with Fault & Latency Injection](docs/articles/chaos-engineering.md)

### 23. Distributed Deadline Propagation
**The Problem**: A request times out at the gateway, but downstream microservices keep working on it, wasting CPU and memory.

**The Recipe**:
Stop "zombie requests" by propagating remaining time budget across service boundaries.

```go
// Early Abort if less than 10ms remains
_, err := resile.Do(ctx, action, 
    resile.WithMinDeadlineThreshold(10 * time.Millisecond),
)

// Inject into HTTP headers
resile.InjectDeadlineHeader(ctx, req.Header, "X-Request-Timeout")
```

[Read more: Stopping the Zombie Requests: Distributed Deadline Propagation in Go](docs/articles/distributed-deadline-propagation.md)

### 24. Reliable File Downloads (HTTP Resumption)
**The Problem**: Downloading a 1GB file fails at 900MB; starting over from zero is wasteful.

**The Recipe**:
Combine `DoErr` with HTTP `Range` headers to resume downloads from the last successful byte.

```go
var bytesReceived int64
err := resile.DoErr(ctx, func(ctx context.Context) error {
    if bytesReceived > 0 {
        req.Header.Set("Range", fmt.Sprintf("bytes=%d-", bytesReceived))
    }
    // ... do request and io.Copy ...
    bytesReceived += n
    return err
}, resile.WithMaxAttempts(10))
```

[Read more: Reliable File Downloads with HTTP Range Resumption](docs/articles/streaming-http-resumption.md)

### 25. SQL Resilience
**The Problem**: Databases are critical yet vulnerable to transient network errors and failovers.

**The Recipe**:
Wrap standard `database/sql` calls with retries and a circuit breaker to protect against both blips and systemic outages.

```go
_, err := resile.Do(ctx, func(ctx context.Context) (sql.Result, error) {
    return db.ExecContext(ctx, "UPDATE users SET active = ? WHERE id = ?", true, 42)
},
    resile.WithRetry(3),
    resile.WithCircuitBreaker(breaker),
)
```

[Read more: Building Bulletproof Database Clients in Go: SQL Resilience with Resile](docs/articles/sql-resilience.md)

### 26. Redis Resilience
**The Problem**: Database connection pools (SQL or NoSQL like Redis) can be exhausted when the database slows down, leading to cascading failures.

**The Recipe**:
Combine retries for transient blips with a shared bulkhead to strictly limit the number of concurrent operations hitting the connection pool.

```go
// 1. Create a shared bulkhead matching your pool size
redisBulkhead := resile.NewBulkhead(20)

// 2. Wrap your Redis or SQL calls
val, err := resile.Do(ctx, func(ctx context.Context) (string, error) {
    return rdb.Get(ctx, "key").Result()
},
    resile.WithMaxAttempts(3),
    resile.WithBulkheadInstance(redisBulkhead),
)
```

[Read more: Reliable Redis: Combining Retries and Bulkheads for Rock-Solid Caching](docs/articles/redis-resilience-with-go.md)

---

## Built on Hyperscaler Research

Resile isn't just a collection of wrappers; it implements proven resilience algorithms used by the world's largest engineering organizations:

*   **AWS Architecture**: [Exponential Backoff and Jitter](https://aws.amazon.com/blogs/architecture/exponential-backoff-and-jitter/) (Full Jitter) to solve the thundering herd problem.
*   **Google SRE**: [Adaptive Throttling](https://sre.google/sre-book/handling-overload/#eq2101) logic to implement client-side rejection and protect degraded downstream services.
*   **Uber Engineering**: [Adaptive Concurrency](https://www.uber.com/blog/microservice-architecture/) based on TCP-Vegas/Little's Law to dynamically adjust load without static limits.
*   **Netflix**: Bulkhead and Circuit Breaker patterns popularized by Hystrix, refined for modern Go concurrency.

---

## Configuration Reference

| Option | Description | Default |
| :--- | :--- | :--- |
| `WithName(string)` | Identifies the operation in logs/metrics. | `""` |
| `WithMaxAttempts(uint)` | Total number of attempts (initial + retries). | `5` |
| `WithRetry(uint)` | Alias for `WithMaxAttempts` that adds a retry policy. | - |
| `WithBaseDelay(duration)` | Initial backoff duration. | `100ms` |
| `WithMaxDelay(duration)` | Maximum possible backoff duration. | `30s` |
| `WithBackoff(Backoff)` | Custom backoff algorithm (e.g. constant). | `Full Jitter` |
| `WithHedgingDelay(duration)`| Delay before speculative retries. | `0` |
| `WithRetryIf(error)` | Only retry if `errors.Is(err, target)`. | All non-fatal |
| `WithRetryIfFunc(func)` | Custom logic to decide if an error is retriable. | `nil` |
| `WithCircuitBreaker(cb)` | Attaches a circuit breaker state machine. | `nil` |
| `WithBulkhead(uint)` | Limits concurrent executions. | `nil` |
| `WithBulkheadInstance(b)` | Attaches a shared bulkhead instance. | `nil` |
| `WithPriorityBulkhead(uint, map)` | Limits concurrency based on priority. | `nil` |
| `WithPriorityBulkheadInstance(b)` | Attaches a shared priority bulkhead. | `nil` |
| `WithRateLimiter(limit, interval)` | Limits execution rate (token bucket). | `nil` |
| `WithRateLimiterInstance(rl)` | Attaches a shared rate limiter instance. | `nil` |
| `WithTimeout(duration)` | Sets an execution timeout for the operation. | `0` |
| `WithAdaptiveBucket(b)` | Attaches a token bucket for adaptive retries. | `nil` |
| `WithInstrumenter(inst)` | Attaches telemetry (slog/OTel/Prometheus). | `nil` |
| `WithFallback(f)` | Sets a generic fallback function. | `nil` |
| `WithFallbackErr(f)` | Sets a fallback function for error-only actions. | `nil` |
| `WithPanicRecovery()` | Enables "Let It Crash" panic handling. | `false` |
| `WithChaos(chaos.Config)` | Integrates a chaos injector for fault/latency injection. | `nil` |
| `WithMinDeadlineThreshold(d)`| Min remaining time required to start an attempt. | `5ms` |

---

## Architecture & Design

Resile is built for high-performance, concurrent applications:
- **Memory Safety**: Uses `time.NewTimer` with proper cleanup to prevent memory leaks in long-running loops.
- **Context Integrity**: Every internal sleep is a `select` between the timer and `ctx.Done()`.
- **Zero Allocations**: Core execution loop is designed to be allocation-efficient.
- **Errors are Values**: Leverage standard `errors.Is` and `errors.As` for all policy decisions.

---

## Acknowledgements

- **AWS Architecture Blog**: For the definitive [Exponential Backoff and Jitter](https://aws.amazon.com/blogs/architecture/exponential-backoff-and-jitter/) algorithm (Full Jitter).
- **Stamina & Tenacity**: For pioneering ergonomic retry APIs in the Python ecosystem that inspired the design of Resile.

## License

Resile is released under the [MIT License](LICENSE).

```
Copyright (c) 2026 Onur Cinar.
The source code is provided under MIT License.

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
```
