# Building Bulletproof Database Clients in Go: SQL Resilience with Resile

You’ve seen it before: a brief network blip causes a database connection timeout, and suddenly your logs are flooded with errors. Even worse, if the database is struggling under heavy load, your application's aggressive retry loops might just be the "final straw" that causes a complete database meltdown.

Database operations are the most critical—yet often the most fragile—part of a backend system. In this article, we'll explore how to build a resilient SQL client in Go using [Resile](https://github.com/cinar/resile) to handle transient failures gracefully while protecting your database from overload.

---

## The "Retry-Breaker" Pattern: The Golden Standard

When dealing with databases, you typically want two layers of protection:

1.  **Retries**: To handle "transient" errors. These are short-lived blips like a temporary network hiccup or a row lock that clears in milliseconds.
2.  **Circuit Breaker**: To handle "systemic" failures. If the database is down or undergoing a failover, retrying immediately and indefinitely will only waste resources and potentially prevent the database from recovering.

By combining them, you get the best of both worlds: you retry the small stuff, but you "stop the bleeding" when things go seriously wrong.

---

## The Implementation

Resile makes it incredibly easy to wrap standard `database/sql` calls. Because Resile is designed with Go generics and the `context` package in mind, it integrates seamlessly with existing database drivers.

Here is how you can wrap a standard `ExecContext` call:

```go
import (
    "context"
    "database/sql"
    "time"

    "github.com/cinar/resile"
    "github.com/cinar/resile/circuit"
)

func UpdateUserStatus(ctx context.Context, db *sql.DB, userID int, active bool) error {
    // 1. Define a circuit breaker (usually defined once at the service level)
    breaker := circuit.New(circuit.Config{
        WindowType:           circuit.WindowCountBased,
        WindowSize:           10,
        MinimumCalls:         3,
        FailureRateThreshold: 50.0,
        ResetTimeout:         time.Second,
    })

    // 2. Wrap the SQL call with Resile
    _, err := resile.Do(ctx, func(ctx context.Context) (sql.Result, error) {
        return db.ExecContext(ctx, "UPDATE users SET active = ? WHERE id = ?", active, userID)
    },
        resile.WithRetry(3),                          // Retry up to 3 times
        resile.WithBaseDelay(100*time.Millisecond),    // Wait between retries
        resile.WithCircuitBreaker(breaker),           // Trip the circuit if failures persist
    )

    return err
}
```

### Why This Works
*   **Context Awareness**: If the user cancels the request or the deadline is reached, Resile stops retrying immediately and honors the context.
*   **Exponential Backoff**: By default, Resile uses a smart backoff strategy, preventing "retry storms."
*   **Shared Intelligence**: If multiple SQL calls share the same `breaker` instance, a failure in one query can help protect the entire database connection pool.

---

## The Critical Caveat: Idempotency and SQL Writes

While retries are powerful, they come with a major risk for **Write** operations (`INSERT`, `UPDATE`, `DELETE`).

Imagine this scenario:
1.  Your app sends an `UPDATE` command to the database.
2.  The database processes it successfully.
3.  The network fails *after* the update but *before* the database can send the "OK" back to your app.
4.  Resile sees a network error and **retries** the operation.

If your operation isn't **idempotent**, you might end up with duplicate data or corrupted state.

### How to Stay Safe:
*   **Use Idempotency Keys**: For `INSERT` operations, use a unique request ID or a `UUID` to prevent duplicates.
*   **Atomic Updates**: Use `WHERE` clauses that check for the previous state (e.g., `UPDATE orders SET status='shipped' WHERE id=123 AND status='pending'`).
*   **Transactions**: Wrap complex multi-step operations in a single SQL transaction to ensure "all or nothing" execution.

---

## Beyond Simple Retries

Database resilience isn't just about trying again. In complex systems, you might want to combine SQL resilience with other Resile features:

*   **[Sliding Window Circuit Breakers](sliding-window-circuit-breakers.md)**: For more accurate failure detection over time.
*   **[Bulkhead Isolation](bulkhead-isolation.md)**: To ensure that a slow "Reports" query doesn't consume all database connections, starving your "User Login" flow.
*   **[Chaos Engineering](chaos-engineering.md)**: To test how your application reacts when the database suddenly starts returning `500` errors or high latency.

---

## Conclusion

The standard library `database/sql` package is excellent, but it leaves resilience as an "exercise for the reader." By using Resile, you can transform a basic database client into a production-grade, self-healing system with just a few lines of declarative code.

**Ready to make your Go services more resilient?**
Check out the full project and more examples on GitHub: [github.com/cinar/resile](https://github.com/cinar/resile)

#golang #sql #database #resilience #sre #microservices #distributed-systems
