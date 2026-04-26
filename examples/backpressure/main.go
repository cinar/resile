// Copyright (c) 2026 Onur Cinar.
// The source code is provided under MIT License.
// https://github.com/cinar/resile

package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/cinar/resile"
	"github.com/cinar/resile/circuit"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		fmt.Println("\nShutting down...")
		cancel()
	}()

	fmt.Println("=== Go-Native Backpressure Signaling Example ===")
	fmt.Println()

	demonstrateCircuitBreakerBackpressure(ctx)
	fmt.Println()
	demonstrateAdaptiveLimiterBackpressure(ctx)
}

func demonstrateCircuitBreakerBackpressure(ctx context.Context) {
	fmt.Println("--- Circuit Breaker Backpressure ---")

	cb := circuit.New(circuit.Config{
		WindowType:           circuit.WindowCountBased,
		WindowSize:          5,
		FailureRateThreshold: 50.0,
		MinimumCalls:        3,
		ResetTimeout:        2 * time.Second,
		HalfOpenMaxCalls:    2,
	})

	p := resile.NewPolicy(
		resile.WithCircuitBreaker(cb),
		resile.WithMaxAttempts(1),
	)

	healthCh := cb.Health()

	action := func(ctx context.Context) error {
		return errors.New("service unavailable")
	}

	circuitBreakerWorker(ctx, p, healthCh, action, 10)
}

func demonstrateAdaptiveLimiterBackpressure(ctx context.Context) {
	fmt.Println("--- Adaptive Limiter Backpressure ---")

	al := resile.NewAdaptiveLimiter()

	p := resile.NewPolicy(
		resile.WithAdaptiveLimiterInstance(al),
	)

	healthCh := al.Health()

	action := func(ctx context.Context) error {
		time.Sleep(50 * time.Millisecond)
		return nil
	}

	limiterWorker(ctx, p, healthCh, action, 20)
}

func circuitBreakerWorker(ctx context.Context, p *resile.Policy, healthCh <-chan circuit.StateEvent, action func(context.Context) error, iterations int) {
	var wg sync.WaitGroup
	paused := false

	for i := 0; i < iterations; i++ {
		select {
		case <-ctx.Done():
			return
		case event := <-healthCh:
			switch event.State {
			case circuit.HealthStateUnhealthy:
				fmt.Printf("  [!] Backpressure: Circuit OPEN - pausing workers\n")
				paused = true
			case circuit.HealthStateDegraded:
				fmt.Printf("  [!] Backpressure: Circuit HALF-OPEN - limiting probes\n")
				paused = false
			case circuit.HealthStateHealthy:
				fmt.Printf("  [!] Backpressure: Circuit CLOSED - resuming workers\n")
				paused = false
			}
			fmt.Printf("      Event: %s: %s\n", event.Component, event.Message)
		default:
		}

		if paused {
			fmt.Println("  [.] Worker paused, waiting for recovery...")
			time.Sleep(100 * time.Millisecond)
			continue
		}

		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			err := p.DoErr(ctx, action)
			if err != nil {
				log.Printf("Worker %d: error: %v", id, err)
			} else if id%5 == 0 {
				fmt.Printf("Worker %d: processed\n", id)
			}
		}(i)

		time.Sleep(50 * time.Millisecond)
	}

	wg.Wait()
}

func limiterWorker(ctx context.Context, p *resile.Policy, healthCh <-chan resile.StateEvent, action func(context.Context) error, iterations int) {
	var wg sync.WaitGroup

	for i := 0; i < iterations; i++ {
		select {
		case <-ctx.Done():
			return
		case event := <-healthCh:
			switch event.State {
			case resile.HealthStateDegraded:
				fmt.Printf("  [!] Backpressure: Capacity DEGRADED - slowing down\n")
			case resile.HealthStateHealthy:
				fmt.Printf("  [!] Backpressure: Capacity HEALTHY - normal processing\n")
			}
			fmt.Printf("      Event: %s: %s\n", event.Component, event.Message)
		default:
		}

		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			err := p.DoErr(ctx, action)
			if err != nil {
				log.Printf("Worker %d: error: %v", id, err)
			} else if id%5 == 0 {
				fmt.Printf("Worker %d: processed\n", id)
			}
		}(i)

		time.Sleep(30 * time.Millisecond)
	}

	wg.Wait()
}