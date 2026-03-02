// Copyright (c) 2026 Onur Cinar.
// The source code is provided under MIT License.
// https://github.com/cinar/resile

package circuit

import (
	"errors"
	"sync"
	"time"
)

// State represents the current state of the circuit breaker.
type State int

const (
	StateClosed State = iota
	StateOpen
	StateHalfOpen
)

var (
	// ErrCircuitOpen is returned when the circuit breaker is in the Open state.
	ErrCircuitOpen = errors.New("circuit breaker is open")
)

// Config defines the configuration for the circuit breaker.
type Config struct {
	// FailureThreshold is the number of failures before the circuit opens.
	FailureThreshold uint
	// ResetTimeout is the duration to wait before transitioning from Open to Half-Open.
	ResetTimeout time.Duration
}

// Breaker implements a circuit breaker state machine.
type Breaker struct {
	mu sync.RWMutex

	config Config
	state  State

	failureCount uint
	lastFailure  time.Time
}

// New returns a new Breaker with the provided configuration.
func New(cfg Config) *Breaker {
	if cfg.FailureThreshold == 0 {
		cfg.FailureThreshold = 5
	}
	if cfg.ResetTimeout == 0 {
		cfg.ResetTimeout = 60 * time.Second
	}

	return &Breaker{
		config: cfg,
		state:  StateClosed,
	}
}

// State returns the current state of the breaker.
func (b *Breaker) State() State {
	b.mu.RLock()
	defer b.mu.RUnlock()

	// If we're open, check if the reset timeout has elapsed.
	if b.state == StateOpen && time.Since(b.lastFailure) > b.config.ResetTimeout {
		return StateHalfOpen
	}

	return b.state
}

// Execute wraps an action with the circuit breaker logic.
func (b *Breaker) Execute(action func() error) error {
	state := b.State()

	if state == StateOpen {
		return ErrCircuitOpen
	}

	// For Closed or Half-Open, we execute the action.
	err := action()

	b.mu.Lock()
	defer b.mu.Unlock()

	if err != nil {
		b.onFailure()
		return err
	}

	b.onSuccess()
	return nil
}

func (b *Breaker) onFailure() {
	b.failureCount++
	b.lastFailure = time.Now()

	if b.state == StateClosed && b.failureCount >= b.config.FailureThreshold {
		b.state = StateOpen
	} else if b.state == StateHalfOpen {
		// Any failure in Half-Open immediately opens the circuit again.
		b.state = StateOpen
	}
}

func (b *Breaker) onSuccess() {
	b.failureCount = 0
	b.state = StateClosed
}