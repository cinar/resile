// Copyright (c) 2026 Onur Cinar.
// The source code is provided under MIT License.
// https://github.com/cinar/resile

package resile

import (
	"context"
	"sync"
)

// TransitionFunc defines the signature for a state transition function.
// It takes the current state, data, and event, and returns the next state and data.
// It also receives the current RetryState to allow transitions to adapt to failure history.
type TransitionFunc[S any, D any, E any] func(ctx context.Context, state S, data D, event E, rs RetryState) (S, D, error)

// StateMachine represents a resilient state machine inspired by Erlang's gen_statem.
// Every state transition is protected by the configured resilience policies.
type StateMachine[S any, D any, E any] struct {
	mu         sync.RWMutex
	state      S
	data       D
	transition TransitionFunc[S, D, E]
	options    []Option
}

// NewStateMachine creates a new resilient state machine with the provided initial state, data, and transition function.
func NewStateMachine[S any, D any, E any](initialState S, initialData D, transition TransitionFunc[S, D, E], opts ...Option) *StateMachine[S, D, E] {
	return &StateMachine[S, D, E]{
		state:      initialState,
		data:       initialData,
		transition: transition,
		options:    opts,
	}
}

// Handle processes an event and performs a state transition.
// The transition is executed within a resilience envelope (Retries, Circuit Breakers, etc.).
// If the transition succeeds, the state machine updates its internal state and data.
func (sm *StateMachine[S, D, E]) Handle(ctx context.Context, event E) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	type update struct {
		state S
		data  D
	}

	result, err := DoState(ctx, func(ctx context.Context, rs RetryState) (update, error) {
		nextState, nextData, err := sm.transition(ctx, sm.state, sm.data, event, rs)
		if err != nil {
			return update{}, err
		}
		return update{state: nextState, data: nextData}, nil
	}, sm.options...)

	if err == nil {
		sm.state = result.state
		sm.data = result.data
	}

	return err
}

// GetState returns the current state of the state machine.
func (sm *StateMachine[S, D, E]) GetState() S {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.state
}

// GetData returns the current data of the state machine.
func (sm *StateMachine[S, D, E]) GetData() D {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.data
}
