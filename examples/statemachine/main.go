// Copyright (c) 2026 Onur Cinar.
// The source code is provided under MIT License.
// https://github.com/cinar/resile

package main

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/cinar/resile"
)

type State string

const (
	Disconnected State = "Disconnected"
	Connecting   State = "Connecting"
	Connected    State = "Connected"
)

type Event string

const (
	Connect    Event = "Connect"
	Disconnect Event = "Disconnect"
)

type Data struct {
	Endpoint string
	Metadata map[string]string
}

func main() {
	ctx := context.Background()

	// 1. Define the transition logic.
	transition := func(ctx context.Context, state State, data Data, event Event, rs resile.RetryState) (State, Data, error) {
		fmt.Printf("[Transition] State: %s, Event: %s, Attempt: %d\n", state, event, rs.Attempt+1)

		switch state {
		case Disconnected:
			if event == Connect {
				// Simulate a connection attempt that fails twice then succeeds.
				if rs.Attempt < 2 {
					return "", data, errors.New("network timeout")
				}
				fmt.Println("Successfully connected!")
				return Connected, data, nil
			}
		case Connected:
			if event == Disconnect {
				return Disconnected, data, nil
			}
		}

		return state, data, nil
	}

	// 2. Initialize the resilient state machine.
	sm := resile.NewStateMachine(
		Disconnected,
		Data{Endpoint: "api.example.com"},
		transition,
		resile.WithMaxAttempts(3),
		resile.WithBaseDelay(100*time.Millisecond),
	)

	fmt.Printf("Initial State: %s\n", sm.GetState())

	// 3. Process events.
	fmt.Println("\n--- Sending Connect Event ---")
	err := sm.Handle(ctx, Connect)
	if err != nil {
		fmt.Printf("Transition failed: %v\n", err)
	} else {
		fmt.Printf("Current State: %s\n", sm.GetState())
	}

	fmt.Println("\n--- Sending Disconnect Event ---")
	err = sm.Handle(ctx, Disconnect)
	if err != nil {
		fmt.Printf("Transition failed: %v\n", err)
	} else {
		fmt.Printf("Current State: %s\n", sm.GetState())
	}
}
