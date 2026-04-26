// Copyright (c) 2026 Onur Cinar.
// The source code is provided under MIT License.
// https://github.com/cinar/resile

package circuit

import (
	"time"
)

// State represents the health state of a resilience component.
type HealthState int

const (
	// StateHealthy represents a healthy state where requests are allowed.
	HealthStateHealthy HealthState = iota
	// StateDegraded represents a degraded but functional state.
	HealthStateDegraded
	// StateUnhealthy represents an unhealthy state (e.g., circuit open).
	HealthStateUnhealthy
)

// StateEvent represents a state change event emitted by resilience components.
type StateEvent struct {
	Component string
	State     HealthState
	Timestamp time.Time
	Message   string
}
