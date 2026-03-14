# Beating Tail Latency: A Guide to Request Hedging in Go Microservices

In distributed systems, we often talk about "The Long Tail." 

You might have a service where 95% of requests finish in under 100ms. But that last 1% (the P99 latency)? Those requests might take 2 seconds or more. In a microservice architecture where one user action triggers 10 different service calls, that one slow dependency will bottleneck the entire user experience.

Standard retries don't help here. Why? Because a "Tail Latency" request hasn't failed yet—it’s just *slow*. 

Waiting for a 2-second timeout to trigger a retry is a waste of time. To beat the long tail, you need **Request Hedging** (also known as Speculative Retries).

Here is how to implement it safely in Go using [Resile](https://github.com/cinar/resile).

---

## What is Request Hedging?

The concept is simple but powerful: If a request is taking longer than usual (say, longer than the P95 latency), don't kill it. Instead, **start a second, identical request in parallel.**

Whichever request finishes first, you take its result and cancel the other one.

This "speculative" approach drastically reduces P99 latency because the mathematical probability of *two* identical requests hitting the "long tail" simultaneously is extremely low.

---

## The Complexity of Manual Hedging

Implementing hedging manually in Go is a nightmare of goroutine management:
1. You need a `select` block with a timer.
2. You need to coordinate between two (or more) goroutines.
3. You must ensure that once one succeeds, the others are cancelled immediately to save resources.
4. You have to handle race conditions where both might succeed at the exact same millisecond.

Most developers end up with hundreds of lines of brittle boilerplate code to handle just one hedged call.

---

## The Resile Way: `DoHedged`

**Resile** makes request hedging as simple as a single function call. It handles the goroutine lifecycle, context cancellation, and race conditions for you.

Here is how you fetch data with a 100ms hedging delay:

```go
import "github.com/cinar/resile"

// data is automatically inferred as *User
data, err := resile.DoHedged(ctx, func(ctx context.Context) (*User, error) {
    return apiClient.GetUser(ctx, userID)
}, 
    resile.WithMaxAttempts(3),
    resile.WithHedgingDelay(100 * time.Millisecond),
)
```

### What happens under the hood?
1. Resile starts the first request.
2. It waits for 100ms (`HedgingDelay`).
3. If the first request hasn't finished, it starts a **second** request.
4. As soon as one returns a successful result, Resile **cancels the context** of the other request and returns the data to you.

---

## Picking the Right Hedging Delay

The "magic" of hedging lies in the delay. 
* **Too short:** You double your traffic unnecessarily, putting extra load on your downstream services.
* **Too long:** You don't gain much latency benefit.

**Pro-Tip:** A good rule of thumb is to set your `HedgingDelay` to your **P95 or P99 latency**. This ensures you only "hedge" the slowest 1-5% of requests, providing a massive latency win with minimal extra load.

---

## Observability: Tracking the "Speculative" Wins

If you're using Resile's OpenTelemetry integration (`telemetry/resileotel`), you can actually see these wins in your distributed traces. 

Each hedged attempt is recorded as a sub-span. When a hedged request wins, you'll see the first span get cancelled and the second one succeed—providing clear proof that hedging saved your user from a 2-second wait.

---

## Conclusion

Request hedging used to be a technique reserved for companies with massive infrastructure teams. With Resile, it’s a tool that every Go developer can use to build snappier, more resilient microservices.

By moving from "Wait and Retry" to "Hedge and Win," you can turn your long-tail latency into a competitive advantage.

**Give Resile a star on GitHub:** [github.com/cinar/resile](https://github.com/cinar/resile)

How are you handling tail latency in your Go services? Let's discuss in the comments!

#golang #microservices #performance #backend #distributedsystems
