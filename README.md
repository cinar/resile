# Resile: Ergonomic Execution Resilience for Go

[![Go Reference](https://pkg.go.dev/badge/github.com/cinar/resile.svg)](https://pkg.go.dev/badge/github.com/cinar/resile)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](https://opensource.org/licenses/MIT)
[![Build Status](https://github.com/cinar/resile/actions/workflows/ci.yml/badge.svg)](https://github.com/cinar/resile/actions/workflows/ci.yml)
[![Codecov](https://codecov.io/gh/cinar/resile/branch/main/graph/badge.svg)](https://codecov.io/gh/cinar/resile)
[![Go Report Card](https://goreportcard.com/badge/github.com/cinar/resile)](https://goreportcard.com/report/github.com/cinar/resile)

**Resile** is a production-grade execution resilience and retry library for Go, inspired by Python's [stamina](https://github.com/hynek/stamina). It provides a type-safe, ergonomic, and highly observable way to handle transient failures in distributed systems.

---

## Table of Contents
- [Installation](#installation)
- [Why Resile?](#why-resile)
- [Articles & Tutorials](#articles--tutorials)
- [Examples](#examples)
- [Common Use Cases](#common-use-cases)
  - [Simple Retries](#1-simple-retries)
  - [Value-Yielding Retries (Generics)](#2-value-yielding-retries-generics)
  - [Request Hedging (Speculative Retries)](#3-request-hedging-speculative-retries)
  - [Stateful Retries & Endpoint Rotation](#4-stateful-retries--endpoint-rotation)
  - [Handling Rate Limits (Retry-After)](#5-handling-rate-limits-retry-after)
  - [Aborting Retries (Pushback Signal)](#6-aborting-retries-pushback-signal)
  - [Fallback Strategies](#7-fallback-strategies)
  - [Bulkhead Pattern](#8-bulkhead-pattern)
  - [Rate Limiting Pattern](#9-rate-limiting-pattern)
  - [Layered Defense with Circuit Breaker](#10-layered-defense-with-circuit-breaker)
  - [Macro-Level Protection (Adaptive Retries)](#11-macro-level-protection-adaptive-retries)
  - [Adaptive Concurrency (TCP-Vegas)](#12-adaptive-concurrency-tcp-vegas)
  - [Structured Logging & Telemetry](#13-structured-logging--telemetry)
  - [Panic Recovery ("Let It Crash")](#14-panic-recovery-let-it-crash)
  - [Fast Unit Testing](#15-fast-unit-testing)
  - [Reusable Clients & Dependency Injection](#16-reusable-clients--dependency-injection)
  - [Marking Errors as Fatal](#17-marking-errors-as-fatal)
  - [Custom Error Filtering](#18-custom-error-filtering)
  - [Policy Composition & Chaining](#19-policy-composition--chaining)
  - [Native Multi-Error Aggregation](#20-native-multi-error-aggregation)
  - [Native Chaos Engineering (Fault & Latency Injection)](#21-native-chaos-engineering-fault--latency-injection)
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
* [Respecting Boundaries: Precise Rate Limiting in Go](docs/articles/rate-limiting.md)
* [Beyond Static Limits: Adaptive Concurrency with TCP-Vegas in Go](docs/articles/adaptive-concurrency.md)
* [Debugging the Timeline: Native Multi-Error Aggregation in Go](docs/articles/native-multi-error-aggregation.md)
* [Native Chaos Engineering: Testing Resilience with Fault & Latency Injection](docs/articles/chaos-engineering.md)


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
- **[Pushback Signal](examples/pushback/main.go)**: Aborting retries immediately using `CancelAllRetries`.
- **[Panic Recovery](examples/panicrecovery/main.go)**: Implementing Erlang's "Let It Crash" philosophy.
- **[State Machine](examples/statemachine/main.go)**: Building resilient state machines inspired by Erlang's `gen_statem`.
- **[Chaos Injection](examples/chaos/main.go)**: Simulating faults and latency to test your policies.

---

## Common Use Cases

### 1. Simple Retries
Retry a simple operation that only returns an error. If all retries fail, Resile returns an aggregated error containing the failures from every attempt.

```go
err := resile.DoErr(ctx, func(ctx context.Context) error {
    return db.PingContext(ctx)
})
```

### 2. Value-Yielding Retries (Generics)
Fetch data with full type safety. The return type is inferred from your closure.

```go
// val is automatically inferred as *User
user, err := resile.Do(ctx, func(ctx context.Context) (*User, error) {
    return apiClient.GetUser(ctx, userID)
}, resile.WithMaxAttempts(3))
```

### 3. Request Hedging (Speculative Retries)
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

### 4. Stateful Retries & Endpoint Rotation
Use `DoState` (or `DoErrState`) to access the `RetryState`, allowing you to rotate endpoints or fallback logic based on the failure history.

```go
endpoints := []string{"api-v1.example.com", "api-v2.example.com"}

data, err := resile.DoState(ctx, func(ctx context.Context, state resile.RetryState) (string, error) {
    // Rotate endpoint based on attempt number
    url := endpoints[state.Attempt % uint(len(endpoints))]
    return client.Get(ctx, url)
})
```

### 5. Handling Rate Limits (Retry-After)
Resile automatically detects if an error implements `RetryAfterError`. It can override the jittered backoff with a server-dictated duration and can also signal immediate termination (pushback).

```go
type RateLimitError struct {
    WaitUntil time.Time
}

func (e *RateLimitError) Error() string { return "too many requests" }
func (e *RateLimitError) RetryAfter() time.Duration {
    return time.Until(e.WaitUntil)
}
func (e *RateLimitError) CancelAllRetries() bool {
    // Return true to abort the entire retry loop immediately.
    return false 
}

// Resile will sleep exactly until WaitUntil when this error is encountered.
```

### 6. Aborting Retries (Pushback Signal)
If a downstream service returns a terminal error (like "Quota Exceeded") that shouldn't be retried, implement `CancelAllRetries() bool` to abort the entire retry loop immediately.

```go
type QuotaExceededError struct{}
func (e *QuotaExceededError) Error() string { return "quota exhausted" }
func (e *QuotaExceededError) CancelAllRetries() bool { return true }

// Resile will stop immediately if this error is encountered,
// even if more attempts are remaining.
_, err := resile.Do(ctx, action, resile.WithMaxAttempts(10))
```

### 7. Fallback Strategies
Provide a fallback function to handle cases where all retries are exhausted or the circuit breaker is open. This is useful for returning stale data or default values.

```go
data, err := resile.Do(ctx, fetchData,
    resile.WithMaxAttempts(3),
    resile.WithFallback(func(ctx context.Context, err error) (string, error) {
        // Return stale data from cache if the primary fetch fails
        return cache.Get(ctx, key), nil 
    }),
)
```

### 8. Bulkhead Pattern
Isolate failures by limiting the number of concurrent executions to a specific resource.

```go
// Shared bulkhead with capacity of 10
bh := resile.NewBulkhead(10)

err := resile.DoErr(ctx, action, resile.WithBulkheadInstance(bh))
```

[Read more: Stop the Domino Effect: Bulkhead Isolation in Go](docs/articles/bulkhead-isolation.md)

### 9. Rate Limiting Pattern
Control the rate of executions using a time-based token bucket (e.g., requests per second).

```go
// Limit to 100 requests per second
rl := resile.NewRateLimiter(100, time.Second)

err := resile.DoErr(ctx, action, resile.WithRateLimiterInstance(rl))
```

[Read more: Respecting Boundaries: Precise Rate Limiting in Go](docs/articles/rate-limiting.md)

### 10. Layered Defense with Circuit Breaker
Combine retries (for transient blips) with a mathematically rigorous sliding window circuit breaker (for systemic outages). Resile supports both **Count-based** and **Time-based** sliding windows.

```go
import "github.com/cinar/resile/circuit"

// Create a circuit breaker that trips if >50% of the last 100 calls fail.
cb := circuit.New(circuit.Config{
    WindowType:           circuit.WindowCountBased,
    WindowSize:           100,
    FailureRateThreshold: 50.0,
    MinimumCalls:         10,
    ResetTimeout:         30 * time.Second,
})

// Use it within a Resile policy
err := resile.DoErr(ctx, action, resile.WithCircuitBreaker(cb))

// OR use it standalone with the context-aware Execute method
err = cb.Execute(ctx, func() error {
    return db.Ping()
})

// Manually reset the breaker if needed
cb.Reset()
```

[Read more: Resilience Beyond Counters: Sliding Window Circuit Breakers in Go](docs/articles/sliding-window-circuit-breakers.md)

### 11. Macro-Level Protection (Adaptive Retries)
Prevent "retry storms" by using a token bucket that is shared across your entire cluster of clients. If the downstream service is degraded, the bucket will quickly deplete, causing clients to fail fast locally instead of hammering the service.

```go
// Share this bucket across multiple executions/goroutines
bucket := resile.DefaultAdaptiveBucket()

err := resile.DoErr(ctx, action, resile.WithAdaptiveBucket(bucket))
```

### 12. Adaptive Concurrency (TCP-Vegas)
Automatically adjust concurrency limits based on Round-Trip Time (RTT). This pattern, inspired by TCP-Vegas, applies Little's Law to prevent cascading failures without manual rate-limit configuration. It increases concurrency when latency is stable and decreases it multiplicatively when queuing is detected.

```go
// Shared limiter across multiple calls
al := resile.NewAdaptiveLimiter()

err := resile.DoErr(ctx, action, resile.WithAdaptiveLimiterInstance(al))
```

### 13. Structured Logging & Telemetry
Integrate with `slog` or `OpenTelemetry` without bloating your core dependencies.

```go
import "github.com/cinar/resile/telemetry/resileslog"

logger := slog.Default()
resile.Do(ctx, action, 
    resile.WithName("get-inventory"), // Name your operation for metrics/logs
    resile.WithInstrumenter(resileslog.New(logger)),
)
```

### 14. Panic Recovery ("Let It Crash")
Convert unexpected Go panics into retryable errors, allowing your application to reset to a known good state without a hard crash.

```go
// val will succeed even if the first attempt panics
val, err := resile.Do(ctx, riskyAction, 
    resile.WithPanicRecovery(),
)
```

### 15. Fast Unit Testing
Never let retry timers slow down your CI. Use `WithTestingBypass` to make all retries execute instantly.

```go
func TestMyService(t *testing.T) {
    ctx := resile.WithTestingBypass(context.Background())
    
    // This will retry 10 times instantly without sleeping.
    err := service.Handle(ctx)
}
```

### 16. Reusable Clients & Dependency Injection
Use `resile.New()` to create a `Retryer` interface for cleaner code architecture and easier testing.

```go
// Create a reusable resilience strategy
retryer := resile.New(
    resile.WithMaxAttempts(3),
    resile.WithBaseDelay(200 * time.Millisecond),
)

// Use the interface to execute actions
err := retryer.DoErr(ctx, func(ctx context.Context) error {
    return service.Call(ctx)
})
```

### 17. Marking Errors as Fatal
Sometimes you know an error is terminal and shouldn't be retried (e.g., "Invalid API Key"). Use `resile.FatalError()` to abort the retry loop immediately.

```go
err := resile.DoErr(ctx, func(ctx context.Context) error {
    err := client.Do()
    if errors.Is(err, ErrAuthFailed) {
        return resile.FatalError(err) // Stops retries immediately
    }
    return err
})
```

### 18. Custom Error Filtering
Control which errors trigger a retry using `WithRetryIf` (for exact matches) or `WithRetryIfFunc` (for custom logic like checking status codes).

```go
err := resile.DoErr(ctx, action,
    // Only retry if the error is ErrConnReset
    resile.WithRetryIf(ErrConnReset),
    
    // OR use a custom function for complex logic
    resile.WithRetryIfFunc(func(err error) bool {
        return errors.Is(err, ErrTransient) || isTimeout(err)
    }),
)
```

### 19. Policy Composition & Chaining
While `Do` and `DoErr` provide a fixed execution order (Bulkhead -> Retry -> Timeout -> Circuit Breaker), you can use the `Policy` API to define a custom order of resilience layers. Policies are thread-safe and reusable.

The order of options in `NewPolicy` determines the execution hierarchy from **outermost to innermost**.

```go
// Define a reusable policy: Bulkhead(20) -> Circuit Breaker -> Retry(3) -> Timeout(1s)
standardPolicy := resile.NewPolicy(
    resile.WithBulkhead(20),
    resile.WithCircuitBreaker(cb),
    resile.WithRetry(3),
    resile.WithTimeout(1*time.Second),
)

// Reuse across multiple calls
val, err := standardPolicy.Do(ctx, action)
err := standardPolicy.DoErr(ctx, actionErr)
```

### 20. Native Multi-Error Aggregation
Resile uses Go 1.20's `errors.Join` to aggregate and return the complete timeline of failures in both standard and hedged retry loops.

```go
err := resile.DoErr(ctx, action, resile.WithMaxAttempts(3))

if err != nil {
    // Check if a specific error occurred in any of the attempts
    if errors.Is(err, context.DeadlineExceeded) {
        fmt.Println("At least one attempt timed out")
    }

    // Iterate over the chronological timeline of failures
    if multi, ok := err.(interface{ Unwrap() []error }); ok {
        for i, e := range multi.Unwrap() {
            fmt.Printf("Attempt %d failed: %v\n", i+1, e)
        }
    }
}
```

[Read more: Debugging the Timeline: Native Multi-Error Aggregation in Go](docs/articles/native-multi-error-aggregation.md)

### 21. Native Chaos Engineering (Fault & Latency Injection)
Safely test your resilience policies by synthetically inducing faults and latency into your application logic.

```go
import "github.com/cinar/resile/chaos"

cfg := chaos.Config{
    ErrorProbability:   0.1,                    // 10% chance of failure
    InjectedError:      errors.New("chaos!"),   // The error to return
    LatencyProbability: 0.2,                    // 20% chance of latency
    LatencyDuration:    100 * time.Millisecond, // Delay to inject
}

err := resile.DoErr(ctx, action, 
    resile.WithRetry(3),
    resile.WithChaos(cfg),
)
```

[Read more: Native Chaos Engineering: Testing Resilience with Fault & Latency Injection](docs/articles/chaos-engineering.md)

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
| `WithRateLimiter(limit, interval)` | Limits execution rate (token bucket). | `nil` |
| `WithRateLimiterInstance(rl)` | Attaches a shared rate limiter instance. | `nil` |
| `WithTimeout(duration)` | Sets an execution timeout for the operation. | `0` |
| `WithAdaptiveBucket(b)` | Attaches a token bucket for adaptive retries. | `nil` |
| `WithInstrumenter(inst)` | Attaches telemetry (slog/OTel/Prometheus). | `nil` |
| `WithFallback(f)` | Sets a generic fallback function. | `nil` |
| `WithFallbackErr(f)` | Sets a fallback function for error-only actions. | `nil` |
| `WithPanicRecovery()` | Enables "Let It Crash" panic handling. | `false` |
| `WithChaos(chaos.Config)` | Integrates a chaos injector for fault/latency injection. | `nil` |

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
