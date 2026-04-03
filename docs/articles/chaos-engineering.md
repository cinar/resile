# Native Chaos Engineering: Testing Resilience with Fault & Latency Injection

You’ve implemented retries, circuit breakers, and timeouts. Your application is now "resilient." But how do you know these policies actually work? Waiting for a production meltdown to verify your configuration is a high-stakes gamble. 

**Native Chaos Engineering** in Resile allows you to synthetically induce failure and latency directly into your application's execution path, ensuring your resilience policies are battle-tested before they're ever needed in production.

---

## The Problem: "Dark Code" in Resilience Policies

Resilience policies—like retries and circuit breakers—are often "dark code." These are execution paths that are rarely traversed under normal operating conditions. Because they only trigger during failure, they are notoriously difficult to test and prone to:

1.  **Buggy Configurations**: A retry limit that is too high, or a circuit breaker threshold that never trips.
2.  **Unintended Side Effects**: A retry loop that accidentally consumes all available database connections.
3.  **Silent Failures**: A fallback strategy that actually panics because it hasn't been executed in months.

Traditional chaos engineering tools often operate at the infrastructure layer (e.g., killing pods or dropping network packets). While powerful, these tools can be difficult to set up in local development or staging environments and often lack the granularity to test specific application-level logic.

---

## The Solution: Fault & Latency Injection

Resile provides a **Chaos Injector** middleware that can be integrated directly into any execution policy. By injecting synthetic faults (errors) and latency (delays) with configurable probabilities, you can simulate various failure scenarios without touching your infrastructure.

### Key Features:
-   **Deterministic Randomness**: Uses Go 1.22's `math/rand/v2` for efficient and predictable random number generation.
-   **Context-Aware**: Latency injection strictly respects `context.Context` cancellation. If your request times out while Resile is injecting chaos latency, it exits immediately.
-   **Zero Dependencies**: Just like the rest of the Resile core, the chaos package depends only on the Go standard library.
-   **Granular Control**: Configure error and latency probabilities independently for fine-tuned simulation.

---

## Practical Usage

Integrating chaos into your existing Resile policies is as simple as adding the `WithChaos` option.

### 1. Basic Chaos Configuration

You can define a chaos configuration that injects a 10% error rate and adds 100ms of latency to 20% of requests.

```go
import (
    "github.com/cinar/resile"
    "github.com/cinar/resile/chaos"
)

// Configure chaos injection
cfg := chaos.Config{
    ErrorProbability:   0.1,                    // 10% chance of failure
    InjectedError:      errors.New("chaos!"),   // The error to return
    LatencyProbability: 0.2,                    // 20% chance of latency
    LatencyDuration:    100 * time.Millisecond, // Delay to inject
}

// Apply it to an execution
err := resile.DoErr(ctx, action, 
    resile.WithRetry(3),
    resile.WithChaos(cfg),
)
```

### 2. Testing Your Circuit Breaker

Chaos injection is exceptionally useful for verifying that your circuit breaker trips under pressure. By setting a high `ErrorProbability`, you can force the breaker to transition from `Closed` to `Open` in a controlled environment.

```go
cb := circuit.New(circuit.Config{
    WindowSize:           10,
    FailureRateThreshold: 50.0,
})

// Force 80% error rate to trip the breaker quickly
cfg := chaos.Config{
    ErrorProbability: 0.8,
    InjectedError:    errors.New("synthetic failure"),
}

for i := 0; i < 20; i++ {
    resile.DoErr(ctx, action, 
        resile.WithCircuitBreaker(cb),
        resile.WithChaos(cfg),
    )
}

fmt.Printf("Circuit Breaker State: %v\n", cb.State()) // Should be Open
```

---

## Configuration Reference

The `chaos.Config` struct provides the following options:

| Field | Type | Description |
| :--- | :--- | :--- |
| `ErrorProbability` | `float64` | The probability of injecting an error (0.0 to 1.0). |
| `InjectedError` | `error` | The error to be returned when an error is injected. |
| `LatencyProbability` | `float64` | The probability of injecting latency (0.0 to 1.0). |
| `LatencyDuration` | `time.Duration` | The duration of the latency to be injected. |

---

## Best Practices

1.  **Environment Gating**: Never enable chaos injection in production unless you are performing a planned game day. Use environment variables to gate the configuration:
    ```go
    if os.Getenv("ENABLE_CHAOS") == "true" {
        opts = append(opts, resile.WithChaos(loadChaosCfg()))
    }
    ```
2.  **Observability**: Ensure your `Instrumenter` (like `slog` or `OTel`) is active. This allows you to see the injected errors and latencies in your logs and traces, making it easier to verify how your application responds.
3.  **Start Small**: Begin with low probabilities (e.g., 1-2%) to identify subtle race conditions or timeout issues before increasing the "blast radius."

---

## Conclusion

Resilience is not a "set it and forget it" feature. It requires continuous verification. By bringing chaos engineering directly into your application's execution policies, Resile empowers you to build systems that aren't just theoretically resilient, but practically battle-hardened.

For more information and advanced usage, visit the [github.com/cinar/resile](https://github.com/cinar/resile) project.
