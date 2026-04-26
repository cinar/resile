// Copyright (c) 2026 Onur Cinar.
// The source code is provided under MIT License.
// https://github.com/cinar/resile

package resile

import (
	"time"
)

// HealthState represents the health state of a resilience component.
type HealthState int

const (
	// HealthStateHealthy represents a healthy state where requests are allowed.
	HealthStateHealthy HealthState = iota
	// HealthStateDegraded represents a degraded but functional state.
	HealthStateDegraded
	// HealthStateUnhealthy represents an unhealthy state (e.g., circuit open).
	HealthStateUnhealthy
)

// StateEvent represents a state change event emitted by resilience components.
type StateEvent struct {
	Component string
	State     HealthState
	Timestamp time.Time
	Message   string
}
