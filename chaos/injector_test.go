// Copyright (c) 2026 Onur Cinar.
// The source code is provided under MIT License.
// https://github.com/cinar/resile

package chaos

import (
	"context"
	"errors"
	"testing"
	"time"
)

type mockRNG struct {
	val float64
}

func (m *mockRNG) Float64() float64 {
	return m.val
}

func TestInjector_Execute_Error(t *testing.T) {
	t.Parallel()

	injectedErr := errors.New("chaos error")
	cfg := Config{
		ErrorProbability: 1.0,
		InjectedError:    injectedErr,
	}
	injector := NewInjectorWithRNG(cfg, &mockRNG{val: 0.0})

	err := injector.Execute(context.Background(), func() error {
		return nil
	})

	if !errors.Is(err, injectedErr) {
		t.Errorf("expected error %v, got %v", injectedErr, err)
	}
}

func TestInjector_Execute_Latency(t *testing.T) {
	t.Parallel()

	latency := 10 * time.Millisecond
	cfg := Config{
		LatencyProbability: 1.0,
		LatencyDuration:    latency,
	}
	injector := NewInjectorWithRNG(cfg, &mockRNG{val: 0.0})

	start := time.Now()
	err := injector.Execute(context.Background(), func() error {
		return nil
	})
	duration := time.Since(start)

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if duration < latency {
		t.Errorf("expected latency at least %v, got %v", latency, duration)
	}
}

func TestInjector_Execute_ContextCancelled(t *testing.T) {
	t.Parallel()

	latency := 100 * time.Millisecond
	cfg := Config{
		LatencyProbability: 1.0,
		LatencyDuration:    latency,
	}
	injector := NewInjectorWithRNG(cfg, &mockRNG{val: 0.0})

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err := injector.Execute(ctx, func() error {
		return nil
	})

	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context canceled, got %v", err)
	}
}

func TestInjector_Nil(t *testing.T) {
	t.Parallel()

	var injector *Injector
	err := injector.Execute(context.Background(), func() error {
		return nil
	})
	if err != nil {
		t.Errorf("expected no error from nil injector, got %v", err)
	}
}

func TestDefaultRNG(t *testing.T) {
	t.Parallel()

	rng := &defaultRNG{}
	for i := 0; i < 100; i++ {
		val := rng.Float64()
		if val < 0.0 || val >= 1.0 {
			t.Errorf("expected value in [0, 1), got %f", val)
		}
	}
}

func TestNewInjector(t *testing.T) {
	t.Parallel()

	cfg := Config{
		ErrorProbability: 0.5,
	}
	injector := NewInjector(cfg)
	if injector == nil {
		t.Fatal("expected injector to be non-nil")
	}
	if injector.config.ErrorProbability != 0.5 {
		t.Errorf("expected probability 0.5, got %f", injector.config.ErrorProbability)
	}
	if injector.rng == nil {
		t.Error("expected default RNG to be set")
	}
}
