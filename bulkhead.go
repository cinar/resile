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

// ErrShedLoad is returned when the adaptive concurrency or priority bulkhead limit is reached.
var ErrShedLoad = errors.New("load shed: limit reached")

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

// PriorityBulkhead limits concurrency based on priority levels.
// It preserves capacity for higher-priority traffic by shedding lower-priority traffic
// when the system approaches saturation.
type PriorityBulkhead struct {
	capacity   uint
	thresholds map[Priority]float64
	semaphore  chan struct{}
}

// NewPriorityBulkhead creates a new PriorityBulkhead with the specified capacity and thresholds.
// The thresholds map defines the maximum utilization (0.0 to 1.0) allowed for each priority.
// If a priority is not in the map, it defaults to 1.0 (no shedding until full capacity).
func NewPriorityBulkhead(capacity uint, thresholds map[Priority]float64) *PriorityBulkhead {
	// Create a deep copy of the thresholds map to prevent external mutation
	copiedThresholds := make(map[Priority]float64, len(thresholds))
	for p, t := range thresholds {
		copiedThresholds[p] = t
	}

	return &PriorityBulkhead{
		capacity:   capacity,
		thresholds: copiedThresholds,
		semaphore:  make(chan struct{}, capacity),
	}
}

// Execute wraps an action with priority-aware bulkhead concurrency control.
// It returns ErrShedLoad if the utilization threshold for the priority is exceeded.
// It returns ErrBulkheadFull if the absolute capacity is reached.
//
// Performance Note: This implementation uses a "soft limit" design where the utilization
// check is decoupled from the actual semaphore acquisition. This provides high throughput
// and avoids global locks, which is ideal for load shedding. While this might allow
// slight over-utilization under extremely high contention, it maintains high performance
// for the majority of cases.
func (b *PriorityBulkhead) Execute(ctx context.Context, action func() error) error {
	priority := GetPriority(ctx)
	threshold, ok := b.thresholds[priority]
	if !ok {
		threshold = 1.0
	}

	// Utilization check
	utilization := float64(len(b.semaphore)) / float64(b.capacity)

	// If physically full, always return ErrBulkheadFull for consistency
	if utilization >= 1.0 {
		return ErrBulkheadFull
	}

	// If threshold exceeded, return ErrShedLoad
	if utilization >= threshold {
		return ErrShedLoad
	}

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
