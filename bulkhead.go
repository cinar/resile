// Copyright (c) 2026 Onur Cinar.
// The source code is provided under MIT License.
// https://github.com/cinar/resile

package resile

import (
	"context"
	"errors"
)

// ErrBulkheadFull is returned when the bulkhead capacity is reached.
var ErrBulkheadFull = errors.New("bulkhead capacity reached")

// Bulkhead limits the number of concurrent executions to prevent overloading a resource.
type Bulkhead struct {
	semaphore chan struct{}
}

// NewBulkhead creates a new Bulkhead with the specified capacity.
func NewBulkhead(capacity uint) *Bulkhead {
	return &Bulkhead{
		semaphore: make(chan struct{}, capacity),
	}
}

// Execute wraps an action with bulkhead concurrency control.
// It returns ErrBulkheadFull if the capacity is reached.
func (b *Bulkhead) Execute(ctx context.Context, action func() error) error {
	select {
	case b.semaphore <- struct{}{}:
		defer func() { <-b.semaphore }()
		return action()
	case <-ctx.Done():
		return ctx.Err()
	default:
		return ErrBulkheadFull
	}
}
