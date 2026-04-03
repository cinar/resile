// Copyright (c) 2026 Onur Cinar.
// The source code is provided under MIT License.
// https://github.com/cinar/resile

package chaos

import (
	"context"
	"math/rand/v2"
	"time"
)

// Config represents the configuration for the chaos injector.
type Config struct {
	// ErrorProbability is the probability of injecting an error (0.0 - 1.0).
	ErrorProbability float64
	// InjectedError is the error to be returned when an error is injected.
	InjectedError error
	// LatencyProbability is the probability of injecting latency (0.0 - 1.0).
	LatencyProbability float64
	// LatencyDuration is the duration of the latency to be injected.
	LatencyDuration time.Duration
}

// RNG defines the interface for randomization, allowing for mock implementations in tests.
type RNG interface {
	// Float64 returns a random float64 in the range [0.0, 1.0).
	Float64() float64
}

// defaultRNG is the default implementation of RNG using math/rand/v2.
type defaultRNG struct{}

func (r *defaultRNG) Float64() float64 {
	return rand.Float64()
}

// Injector is the chaos injector that can inject synthetic faults and latency.
type Injector struct {
	config Config
	rng    RNG
}

// NewInjector creates a new chaos injector with the provided configuration.
func NewInjector(cfg Config) *Injector {
	return &Injector{
		config: cfg,
		rng:    &defaultRNG{},
	}
}

// NewInjectorWithRNG creates a new chaos injector with the provided configuration and RNG.
func NewInjectorWithRNG(cfg Config, rng RNG) *Injector {
	return &Injector{
		config: cfg,
		rng:    rng,
	}
}

// Execute wraps an action and injects synthetic faults and latency based on the configuration.
func (i *Injector) Execute(ctx context.Context, action func() error) error {
	if i == nil {
		return action()
	}

	// Roll for latency.
	if i.config.LatencyDuration > 0 && i.rng.Float64() < i.config.LatencyProbability {
		timer := time.NewTimer(i.config.LatencyDuration)
		defer func() {
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
		}()

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timer.C:
			// Latency injected successfully.
		}
	}

	// Roll for error.
	if i.config.InjectedError != nil && i.rng.Float64() < i.config.ErrorProbability {
		return i.config.InjectedError
	}

	return action()
}
