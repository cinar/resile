package main

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"github.com/cinar/resile"
	"github.com/cinar/resile/circuit"
	"sync/atomic"
	"time"
)

var attempts atomic.Int32

// transientFailureDriver is a mock SQL driver that fails the first two calls.
type transientFailureDriver struct{}
type transientFailureConnection struct{}

func (transientFailureDriver) Open(name string) (driver.Conn, error) {
	return transientFailureConnection{}, nil
}

func (transientFailureConnection) Prepare(query string) (driver.Stmt, error) {
	return nil, errors.New("Prepare is not implemented")
}

func (transientFailureConnection) Begin() (driver.Tx, error) {
	return nil, errors.New("transactions are not implemented")
}

func (transientFailureConnection) Close() error {
	return nil
}

func (transientFailureConnection) ExecContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	current := attempts.Add(1)
	if current < 3 {
		return nil, errors.New("temporary database error")
	}
	return driver.RowsAffected(1), nil
}

func main() {
	sql.Register("transient-sql", transientFailureDriver{})

	db, err := sql.Open("transient-sql", "")
	if err != nil {
		panic(err)
	}
	defer db.Close()

	ctx := context.Background()

	breaker := circuit.New(circuit.Config{
		WindowType:           circuit.WindowCountBased,
		WindowSize:           10,
		MinimumCalls:         3,
		FailureRateThreshold: 50,
		ResetTimeout:         time.Second,
	})

	// SQL call wrapped with Resile so transient db errors can be retried
	result, err := resile.Do(ctx, func(ctx context.Context) (sql.Result, error) {
		return db.ExecContext(ctx, "UPDATE users SET active = ? WHERE id = ?", true, 42)
	},
		resile.WithRetry(3),
		resile.WithBaseDelay(100*time.Millisecond),
		resile.WithCircuitBreaker(breaker),
	)

	if err != nil {
		fmt.Printf("query failed: %v\n", err)
		return
	}

	rowsAffected, _ := result.RowsAffected()
	fmt.Printf("query succeeded after %d attempts; rows affected: %d\n",
		attempts.Load(), rowsAffected)
}
