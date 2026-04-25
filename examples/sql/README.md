# SQL Example

This example demonstrates how to use Resile with Go's standard `database/sql` package.

It wraps `db.ExecContext` call with retries and a circuit breaker. The example uses a mock SQL driver instead of a real database so it can run anywhere without extra setup.

The mock driver fails the first two attempts with a temporary error and succeeds on the third attempt. This shows Resile retrying the operation.

## Run
```bash
go run ./examples/sql
```

Expected output:
```
query succeeded after 3 attempts; rows affected: 1
```

## How It Works
```go
result, err := resile.Do(ctx, func(ctx context.Context) (sql.Result, error) {
    return db.ExecContext(ctx, "UPDATE users SET active = ? WHERE id = ?",
        true,
        42,
    )
},
    resile.WithRetry(3),
    resile.WithBaseDelay(100*time.Millisecond),
    resile.WithCircuitBreaker(breaker),
)
```

The SQL call is wrapped with `resile.Do`.
The circuit breaker is included to show how you can stop calling a database when repeated failures suggest it is unhealthy.

## Note about SQL Writes
Be careful when retrying write queries like UPDATE, INSERT, or DELETE.
A retry in this case will run the same write more than once if the first attempt has reached the database but client received no response. In production applications, transactions, idempotency keys, unique constraints, or other safeguards are employed when retrying writes.
