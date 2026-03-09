# Python's Stamina for Go: Bringing Ergonomic Resilience to Gophers

If you've ever worked in the Python ecosystem, you've likely encountered [tenacity](https://github.com/jd/tenacity) or its opinionated wrapper, [stamina](https://github.com/hynek/stamina). They make retrying transient failures feel like magic: a single decorator, sensible production defaults (exponential backoff + jitter), and built-in observability.

When I moved back to Go, I felt the "Resilience Gap." 

The Go ecosystem has powerful tools, but they often require a lot of boilerplate, use reflection, or lack the "Correct by Default" philosophy that makes `stamina` so great.

That's why I built **[Resile](https://github.com/cinar/resile)**. It’s a love letter to Python's ergonomics, written in idiomatic, type-safe Go 1.18+.

---

## The Ergonomic Gap

In Python, retrying an API call with `stamina` looks like this:

```python
import stamina

@stamina.retry(on=httpx.HTTPError, attempts=3)
def get_data():
    return httpx.get("https://api.example.com").json()
```

Before Go 1.18, achieving this level of simplicity was nearly impossible. You either had to write verbose `for` loops or use libraries that relied on `interface{}` and reflection—which meant losing type safety and slowing down your code.

With the arrival of **Generics**, the game changed. We can now have the best of both worlds: Python-level ergonomics with Go’s compile-time safety.

---

## The Resile Way: One-Line Resilience

Resile uses Go 1.18+ Type Parameters to wrap your logic in a resilience envelope. Here is how you fetch data with Resile:

```go
import "github.com/cinar/resile"

// data is automatically inferred as *User
data, err := resile.Do(ctx, func(ctx context.Context) (*User, error) {
    return apiClient.GetUser(ctx, userID)
}, resile.WithMaxAttempts(3))
```

It looks and feels like a simple function call, but under the hood, Resile is doing the heavy lifting:
*   **AWS Full Jitter Backoff**: Spreading out retries to protect your database.
*   **Context-Awareness**: Cancelling retries immediately if the request times out.
*   **Memory Safety**: Using managed timers to prevent goroutine leaks.

---

## Why "Correct by Default" Matters

One of the best things about `stamina` is that it makes the *right* thing the *easy* thing. Resile follows this philosophy strictly:

1.  **Exponential Backoff is the Default**: You don't have to configure it; it's there from attempt one.
2.  **Jitter is Not Optional**: Resile forces randomization to prevent "thundering herd" outages.
3.  **Zero Dependencies**: The core of Resile depends only on the Go standard library. No bloated dependency graphs.
4.  **No Reflection**: Unlike many older Go retry libraries, Resile uses static type parameters. This means zero runtime overhead and zero chance of a "type mismatch" panic.

---

## Batteries Included (But Removable)

Just like `stamina` integrates with `structlog` and `Prometheus`, Resile provides optional sub-packages for modern observability:

*   **`telemetry/resileslog`**: High-performance structured logging with Go 1.21’s `slog`.
*   **`telemetry/resileotel`**: Full OpenTelemetry tracing. See every retry attempt as a sub-span in your Jaeger or Honeycomb dashboard.
*   **Adaptive Retries**: A client-side token bucket (inspired by Google's SRE book) to prevent your fleet from killing a degraded service.

---

## Fast CI/CD: The "Stamina" Secret

A hidden feature of `stamina` that I absolutely loved was the ability to globally disable wait times in unit tests. 

Resile brings this to Go through **Context Overrides**. You can make your 30-second retry loop execute in 1 millisecond during tests without changing a single line of business logic:

```go
func TestService(t *testing.T) {
    // This context tells Resile to skip all sleep timers
    ctx := resile.WithTestingBypass(context.Background())
    
    err := myService.PerformTask(ctx) // Retries instantly!
}
```

---

## Conclusion

We don't have to choose between Go's performance and Python's developer experience. By leveraging modern Go features like Generics and `slog`, we can build tools that are both powerful and a joy to use.

If you’ve been missing `stamina` in your Go projects, give **Resile** a try.

**Check it out on GitHub:** [github.com/cinar/resile](https://github.com/cinar/resile)

#golang #python #backend #programming #opensource
