# Self-Healing State Machines: Resilient State Transitions in Go

Distributed systems are inherently stateful. Whether you're managing a database connection pool, a multi-step payment workflow, or a complex IoT device lifecycle, you need to transition between states reliably.

Standard state machines (FSMs) are great for logic, but they are often brittle. What happens if a transition involves a network call that fails? Most developers end up wrapping their `machine.Transition()` calls in manual retry loops, cluttering their business logic and losing visibility into *why* a transition failed.

Inspired by Erlang's `gen_statem` behavior, **Resile** introduces `resile.StateMachine`: a standardized, resilient state machine where every transition is inherently protected by resilience policies.

---

## The Resile Way: One-Line Resilience for Transitions

With `resile.StateMachine`, you don't just define how to move from State A to State B. You define a **Resilient Transition**.

Here is how you implement a self-healing connection manager:

```go
import "github.com/cinar/resile"

// 1. Define your State, Data, and Events
type State string
const (
    Disconnected State = "Disconnected"
    Connected    State = "Connected"
)

type Event string
const (
    Connect Event = "Connect"
)

// 2. Define the Transition Logic
transition := func(ctx context.Context, state State, data Data, event Event, rs resile.RetryState) (State, Data, error) {
    if state == Disconnected && event == Connect {
        // This transition involves a network call.
        // If it fails, Resile will automatically retry it 
        // using the configured backoff and jitter.
        err := apiClient.Connect(ctx, data.Endpoint)
        if err != nil {
            return "", data, err
        }
        return Connected, data, nil
    }
    return state, data, nil
}

// 3. Initialize the Resilient State Machine
sm := resile.NewStateMachine(
    Disconnected, 
    Data{Endpoint: "api.example.com"}, 
    transition,
    resile.WithMaxAttempts(3),
    resile.WithBaseDelay(100 * time.Millisecond),
)

// 4. Handle events safely
err := sm.Handle(ctx, Connect)
```

### What happens under the hood?
1. When you call `sm.Handle(ctx, event)`, Resile enters a execution envelope.
2. It executes your `transition` function.
3. If the `transition` returns an error, Resile applies your retry policy (e.g., Exponential Backoff with Jitter).
4. Only when the `transition` succeeds does the `StateMachine` update its internal state and data.
5. If the retries are exhausted, the `StateMachine` remains in its previous state, ensuring consistency.

---

## Why "Self-Healing"?

Most state machine implementations are "fire and forget" or "fail and stop." A **Self-Healing** state machine assumes that transitions are risky and provides the infrastructure to recover from those risks automatically.

*   **Automatic Retries**: No more manual loops inside your state logic.
*   **Circuit Breakers**: If a specific transition (e.g., to a "Maintenance" state) is failing repeatedly, the circuit breaker can trip to prevent overwhelming the system.
*   **Context Awareness**: If the transition is part of a timed-out request, the state machine cancels the transition attempt immediately, preventing goroutine leaks.
*   **Observability**: Every transition attempt—including retries—is tracked by Resile's telemetry hooks. You can see exactly how many times your machine "struggled" to reach the `Connected` state.

---

## Observability: Tracking State Success

By using Resile's OpenTelemetry or `slog` integrations, you get deep insights into your state machine's health:

*   **Attempts per Transition**: See which events are causing the most retries.
*   **Transition Latency**: Measure how long it takes to move from one state to another, including backoff time.
*   **Failure Patterns**: Identify if a specific state is a "dead end" due to persistent errors.

---

## Conclusion

Resilience isn't just for simple API calls. By bringing resilience to the core of your stateful logic, you build systems that are not only more robust but also significantly easier to debug and monitor.

Stop writing manual retry loops around your state changes. Let `resile.StateMachine` handle the complexity of the "unreliable world" while you focus on the logic of your application.

**Give Resile a star on GitHub:** [github.com/cinar/resile](https://github.com/cinar/resile)

How are you managing state transitions in your Go microservices? Let's discuss!

#golang #microservices #programming #distributedsystems #resilience
