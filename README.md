# Resile: Ergonomic Execution Resilience for Go

[![Go Reference](https://pkg.go.dev/badge/github.com/cinar/resile.svg)](https://pkg.go.dev/github.com/cinar/resile)
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
  - [Layered Defense with Circuit Breaker](#8-layered-defense-with-circuit-breaker)
  - [Macro-Level Protection (Adaptive Retries)](#9-macro-level-protection-adaptive-retries)
  - [Structured Logging & Telemetry](#10-structured-logging--telemetry)
  - [Panic Recovery ("Let It Crash")](#11-panic-recovery-let-it-crash)
  - [Fast Unit Testing](#12-fast-unit-testing)
  - [Reusable Clients & Dependency Injection](#13-reusable-clients--dependency-injection)
  - [Marking Errors as Fatal](#14-marking-errors-as-fatal)
  - [Custom Error Filtering](#15-custom-error-filtering)
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

---

## Articles & Tutorials

Want to learn more about the philosophy behind Resile and advanced resilience patterns in Go? Check out these deep dives:

* [Stop Writing Manual Retry Loops in Go: Why Your Current Logic is Probably Dangerous](https://dev.to/onurcinar/stop-writing-manual-retry-loops-in-go-why-your-current-logic-is-probably-dangerous-5bj5)
* [Python's Stamina for Go: Bringing Ergonomic Resilience to Gophers](https://dev.to/onurcinar/pythons-stamina-for-go-bringing-ergonomic-resilience-to-gophers-1lf2)
* [Beating Tail Latency: A Guide to Request Hedging in Go Microservices](https://dev.to/onurcinar/beating-tail-latency-a-guide-to-request-hedging-in-go-microservices-p81)
* [Preventing Microservice Meltdowns: Adaptive Retries and Circuit Breakers in Go](https://dev.to/onurcinar/preventing-microservice-meltdowns-adaptive-retries-and-circuit-breakers-in-go-30ho)


## Examples

The [examples/](examples/) directory contains standalone programs showing how to use Resile in various scenarios:

- **[Basic Retry](examples/basic/main.go)**: Simple `Do` and `DoErr` calls.
- **[Request Hedging](examples/hedging/main.go)**: Reducing tail latency with speculative retries.
- **[HTTP with Rate Limits](examples/http/main.go)**: Respecting `Retry-After` headers and using `slog`.
- **[Fallback Strategies](examples/fallback/main.go)**: Returning stale data when all attempts fail.
- **[Stateful Rotation](examples/stateful/main.go)**: Rotating API endpoints using `RetryState`.
- **[Circuit Breaker](examples/circuitbreaker/main.go)**: Layering defensive strategies.
- **[Adaptive Retries](examples/adaptiveretry/main.go)**: Preventing retry storms with a token bucket.
- **[Pushback Signal](examples/pushback/main.go)**: Aborting retries immediately using `CancelAllRetries`.
- **[Panic Recovery](examples/panicrecovery/main.go)**: Implementing Erlang's "Let It Crash" philosophy.

---

## Common Use Cases

### 1. Simple Retries
Retry a simple operation that only returns an error.

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

### 8. Layered Defense with Circuit Breaker
Combine retries (for transient blips) with a circuit breaker (for systemic outages).

```go
import "github.com/cinar/resile/circuit"

cb := circuit.New(circuit.Config{
    FailureThreshold: 5,
    ResetTimeout:     30 * time.Second,
})

// Returns circuit.ErrCircuitOpen immediately if the downstream is failing consistently.
err := resile.DoErr(ctx, action, resile.WithCircuitBreaker(cb))
```

### 9. Macro-Level Protection (Adaptive Retries)
Prevent "retry storms" by using a token bucket that is shared across your entire cluster of clients. If the downstream service is degraded, the bucket will quickly deplete, causing clients to fail fast locally instead of hammering the service.

```go
// Share this bucket across multiple executions/goroutines
bucket := resile.DefaultAdaptiveBucket()

err := resile.DoErr(ctx, action, resile.WithAdaptiveBucket(bucket))
```

### 10. Structured Logging & Telemetry
Integrate with `slog` or `OpenTelemetry` without bloating your core dependencies.

```go
import "github.com/cinar/resile/telemetry/resileslog"

logger := slog.Default()
resile.Do(ctx, action, 
    resile.WithName("get-inventory"), // Name your operation for metrics/logs
    resile.WithInstrumenter(resileslog.New(logger)),
)
```

### 11. Panic Recovery ("Let It Crash")
Convert unexpected Go panics into retryable errors, allowing your application to reset to a known good state without a hard crash.

```go
// val will succeed even if the first attempt panics
val, err := resile.Do(ctx, riskyAction, 
    resile.WithPanicRecovery(),
)
```

### 12. Fast Unit Testing
Never let retry timers slow down your CI. Use `WithTestingBypass` to make all retries execute instantly.

```go
func TestMyService(t *testing.T) {
    ctx := resile.WithTestingBypass(context.Background())
    
    // This will retry 10 times instantly without sleeping.
    err := service.Handle(ctx)
}
```

### 13. Reusable Clients & Dependency Injection
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

### 14. Marking Errors as Fatal
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

### 15. Custom Error Filtering
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

---

## Configuration Reference

| Option | Description | Default |
| :--- | :--- | :--- |
| `WithName(string)` | Identifies the operation in logs/metrics. | `""` |
| `WithMaxAttempts(uint)` | Total number of attempts (initial + retries). | `5` |
| `WithBaseDelay(duration)` | Initial backoff duration. | `100ms` |
| `WithMaxDelay(duration)` | Maximum possible backoff duration. | `30s` |
| `WithBackoff(Backoff)` | Custom backoff algorithm (e.g. constant). | `Full Jitter` |
| `WithHedgingDelay(duration)`| Delay before speculative retries. | `0` |
| `WithRetryIf(error)` | Only retry if `errors.Is(err, target)`. | All non-fatal |
| `WithRetryIfFunc(func)` | Custom logic to decide if an error is retriable. | `nil` |
| `WithCircuitBreaker(cb)` | Attaches a circuit breaker state machine. | `nil` |
| `WithAdaptiveBucket(b)` | Attaches a token bucket for adaptive retries. | `nil` |
| `WithInstrumenter(inst)` | Attaches telemetry (slog/OTel/Prometheus). | `nil` |
| `WithFallback(f)` | Sets a generic fallback function. | `nil` |
| `WithFallbackErr(f)` | Sets a fallback function for error-only actions. | `nil` |
| `WithPanicRecovery()` | Enables "Let It Crash" panic handling. | `false` |

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
