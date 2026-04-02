# Debugging the Timeline: Native Multi-Error Aggregation in Go

When a service call fails after three retries, knowing that it failed is only half the story. The real question is: *How* did it fail? 

Did it timeout twice and then hit a 503? Or did it fail with a DNS error first, followed by a connection refused? In a complex distributed system, the **timeline of failures** is the most valuable piece of evidence you have for debugging transient issues.

Conventional retry loops often throw away this history, returning only the last error encountered. With [Resile](https://github.com/cinar/resile), you get the complete picture using Go 1.20's native multi-error aggregation.

---

## The Problem: The "Last Error Wins" Anti-Pattern

Most manual retry loops look like this:

```go
var lastErr error
for i := 0; i < 3; i++ {
    if err := doWork(); err != nil {
        lastErr = err
        continue
    }
    return nil
}
return lastErr
```

In this pattern, if your first attempt failed because of a critical configuration error (like an invalid API key) and subsequent attempts failed because the context was cancelled, you would only see `context.Canceled`. The root cause—the configuration error—is lost forever.

## The Solution: Native `errors.Join` Support

Resile leverages `errors.Join`, introduced in Go 1.20, to aggregate every single error encountered during an execution lifecycle. 

Whether you are using standard retries or speculative hedging, Resile preserves the entire sequence of failures. This means you don't just get an error; you get a **chronological audit log** of what went wrong.

### How it Works in Practice

When you execute a function with Resile, and it exhausts all retry attempts, the returned error wraps every intermediate failure.

```go
err := resile.DoErr(ctx, func(ctx context.Context) error {
    return fmt.Errorf("attempt failed")
}, resile.WithMaxAttempts(3))

if err != nil {
    // Printing the error shows all attempts separated by newlines
    fmt.Println(err)
    /*
       Output:
       attempt failed
       attempt failed
       attempt failed
    */
}
```

### Full Compatibility with `errors.Is` and `errors.As`

Because Resile uses the standard library's multi-error implementation, it remains fully compatible with Go's error inspection primitives. You can check if *any* of the attempts failed for a specific reason:

```go
if errors.Is(err, context.DeadlineExceeded) {
    // At least one attempt (likely the last one) timed out
}

var dnsErr *net.DNSError
if errors.As(err, &dnsErr) {
    // One of the attempts encountered a DNS issue
}
```

---

## Aggregation in Hedged Retries

Multi-error aggregation is even more critical when using **Request Hedging** (`DoHedged`). In hedging, multiple attempts run concurrently. If they all fail, Resile collects the errors from every parallel execution path and joins them.

This ensures that if your primary request failed with a "Connection Refused" but your speculative "hedged" request failed with a "Read Timeout," you see both in the final error report.

---

## Inspecting the Error Slice

If you need to programmatically iterate over the individual errors (for example, to log them to different systems or count failure types), you can use the `Unwrap() []error` pattern supported by Go 1.20+:

```go
if err != nil {
    if multi, ok := err.(interface{ Unwrap() []error }); ok {
        errs := multi.Unwrap()
        fmt.Printf("Encountered %d failures across all attempts\n", len(errs))
        
        for i, e := range errs {
            fmt.Printf("Attempt %d: %v\n", i+1, e)
        }
    }
}
```

---

## Conclusion

Resilience isn't just about keeping the system running; it's about making the system **observable** when things go wrong. By aggregating errors natively, Resile gives you a high-fidelity view of failure modes without the need for custom error types or complex boilerplate.

Stop guessing why your retries are failing. Start seeing the full timeline.

**Explore the Resile Project:** [github.com/cinar/resile](https://github.com/cinar/resile)

How do you handle multi-error reporting in your Go applications? Let's connect on GitHub!

#golang #backend #microservices #observability #distributedsystems
