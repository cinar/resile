# Resile: Ergonomic Execution Resilience for Go

[![Go Reference](https://pkg.go.dev/badge/github.com/cinar/resile.svg)](https://pkg.go.dev/github.com/cinar/resile)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](https://opensource.org/licenses/MIT)
[![Build Status](https://github.com/cinar/resile/actions/workflows/go.yml/badge.svg)](https://github.com/cinar/resile/actions/workflows/go.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/cinar/resile)](https://goreportcard.com/report/github.com/cinar/resile)

**Resile** is a production-grade execution resilience and retry library for Go, inspired by Python's [stamina](https://github.com/hynek/stamina). It provides a type-safe, ergonomic, and highly observable way to handle transient failures in distributed systems.

---

## Table of Contents
- [Installation](#installation)
- [Why Resile?](#why-resile)
- [Common Use Cases](#common-use-cases)
  - [Simple Retries](#1-simple-retries)
  - [Value-Yielding Retries (Generics)](#2-value-yielding-retries-generics)
  - [Stateful Retries & Endpoint Rotation](#3-stateful-retries--endpoint-rotation)
  - [Handling Rate Limits (Retry-After)](#4-handling-rate-limits-retry-after)
  - [Layered Defense with Circuit Breaker](#5-layered-defense-with-circuit-breaker)
  - [Structured Logging & Telemetry](#6-structured-logging--telemetry)
  - [Fast Unit Testing](#7-fast-unit-testing)
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
- **Generic-First**: No `interface{}` or reflection. Full compile-time type safety.
- **Context-Aware**: Strictly respects `context.Context` cancellation and deadlines.
- **Zero-Dependency Core**: The core library only depends on the Go standard library.
- **Opinionated Defaults**: Sensible production-ready defaults (5 attempts, exponential backoff).

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

### 3. Stateful Retries & Endpoint Rotation
Use `DoState` to access the `RetryState`, allowing you to rotate endpoints or fallback logic based on the failure history.

```go
endpoints := []string{"api-v1.example.com", "api-v2.example.com"}

data, err := resile.DoState(ctx, func(ctx context.Context, state resile.RetryState) (string, error) {
    // Rotate endpoint based on attempt number
    url := endpoints[state.Attempt % uint(len(endpoints))]
    return client.Get(ctx, url)
})
```

### 4. Handling Rate Limits (Retry-After)
Resile automatically detects if an error implements `RetryAfterError` and overrides the jittered backoff with the server-dictated duration.

```go
type RateLimitError struct {
    WaitUntil time.Time
}

func (e *RateLimitError) Error() string { return "too many requests" }
func (e *RateLimitError) RetryAfter() time.Duration {
    return time.Until(e.WaitUntil)
}

// Resile will sleep exactly until WaitUntil when this error is encountered.
```

### 5. Layered Defense with Circuit Breaker
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

### 6. Structured Logging & Telemetry
Integrate with `slog` or `OpenTelemetry` without bloating your core dependencies.

```go
import "github.com/cinar/resile/telemetry/resileslog"

logger := slog.Default()
resile.Do(ctx, action, 
    resile.WithName("get-inventory"), // Name your operation for metrics/logs
    resile.WithInstrumenter(resileslog.New(logger)),
)
```

### 7. Fast Unit Testing
Never let retry timers slow down your CI. Use `WithTestingBypass` to make all retries execute instantly.

```go
func TestMyService(t *testing.T) {
    ctx := resile.WithTestingBypass(context.Background())
    
    // This will retry 10 times instantly without sleeping.
    err := service.Handle(ctx)
}
```

---

## Configuration Reference

| Option | Description | Default |
| :--- | :--- | :--- |
| `WithName(string)` | Identifies the operation in logs/metrics. | `""` |
| `WithMaxAttempts(uint)` | Total number of attempts (initial + retries). | `5` |
| `WithBaseDelay(duration)` | Initial backoff duration. | `100ms` |
| `WithMaxDelay(duration)` | Maximum possible backoff duration. | `30s` |
| `WithRetryIf(error)` | Only retry if `errors.Is(err, target)`. | All non-fatal |
| `WithRetryIfFunc(func)` | Custom logic to decide if an error is retriable. | `nil` |
| `WithCircuitBreaker(cb)` | Attaches a circuit breaker state machine. | `nil` |
| `WithInstrumenter(inst)` | Attaches telemetry (slog/OTel/Prometheus). | `nil` |

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
