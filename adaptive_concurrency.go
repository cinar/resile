// Copyright (c) 2026 Onur Cinar.
// The source code is provided under MIT License.
// https://github.com/cinar/resile

package resile

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// AdaptiveLimiter implements a dynamic concurrency limiter using Additive Increase,
// Multiplicative Decrease (AIMD) based on Round Trip Time (RTT), applying Little's Law.
type AdaptiveLimiter struct {
	mu sync.RWMutex

	// Atomic state for high-frequency checks
	currentConcurrency int32
	maxConcurrency     int32

	// Little's Law & AIMD state
	minBaselineRTT time.Duration
	rttEWMA        time.Duration

	// Configuration
	minConcurrency int32
	limitScaling   float64
	threshold      float64
	alpha          float64 // EWMA weight

	// Decay mechanism
	lastDecayUpdate time.Time
	decayInterval   time.Duration

	// Health channel for backpressure signaling
	healthCh     chan StateEvent
	healthChInit bool

	// Track previous max for change detection
	prevMaxConcurrency int32
}

// NewAdaptiveLimiter creates a new AdaptiveLimiter with default configurations.
func NewAdaptiveLimiter() *AdaptiveLimiter {
	return &AdaptiveLimiter{
		maxConcurrency:     10, // Initial guess
		prevMaxConcurrency: 10,
		minConcurrency:     1,   // Floor
		limitScaling:       0.8, // Multiplicative Decrease factor
		threshold:          1.5, // Threshold for increase/decrease
		alpha:              0.1, // EWMA smoothing factor
		decayInterval:      time.Minute,
		lastDecayUpdate:    time.Now(),
	}
}

// Execute wraps an action with adaptive concurrency control.
func (al *AdaptiveLimiter) Execute(ctx context.Context, action func() error) error {
	// 1. Admission Control
	current := atomic.AddInt32(&al.currentConcurrency, 1)
	defer atomic.AddInt32(&al.currentConcurrency, -1)

	max := atomic.LoadInt32(&al.maxConcurrency)
	if current > max {
		return ErrShedLoad
	}

	// 2. Measure RTT
	start := time.Now()
	err := action()
	rtt := time.Since(start)

	// 3. Update state
	al.update(rtt)

	return err
}

func (al *AdaptiveLimiter) update(rtt time.Duration) {
	al.mu.Lock()
	defer al.mu.Unlock()

	// Handle initial state
	if al.minBaselineRTT == 0 {
		al.minBaselineRTT = rtt
		al.rttEWMA = rtt
		return
	}

	// Decay minBaselineRTT periodically to allow recalibration
	now := time.Now()
	if now.Sub(al.lastDecayUpdate) > al.decayInterval {
		al.minBaselineRTT = time.Duration(float64(al.minBaselineRTT) * 1.05)
		al.lastDecayUpdate = now
	}

	// Update minBaselineRTT if we found a new faster path
	if rtt < al.minBaselineRTT {
		al.minBaselineRTT = rtt
	}

	// Update EWMA
	al.rttEWMA = time.Duration((1-al.alpha)*float64(al.rttEWMA) + al.alpha*float64(rtt))

	// AIMD Logic
	// Calculate queue delay threshold
	if float64(al.rttEWMA) < float64(al.minBaselineRTT)*al.threshold {
		// Additive Increase
		atomic.AddInt32(&al.maxConcurrency, 1)
		al.checkAndEmitEvent(false)
	} else {
		// Multiplicative Decrease
		newMax := float64(atomic.LoadInt32(&al.maxConcurrency)) * al.limitScaling
		if newMax < float64(al.minConcurrency) {
			newMax = float64(al.minConcurrency)
		}
		atomic.StoreInt32(&al.maxConcurrency, int32(newMax))
		al.checkAndEmitEvent(true)
	}
}

func (al *AdaptiveLimiter) checkAndEmitEvent(decreased bool) {
	currentMax := atomic.LoadInt32(&al.maxConcurrency)
	prev := al.prevMaxConcurrency

	if decreased {
		threshold := float64(prev) * 0.5
		if float64(currentMax) <= threshold {
			al.emitStateEvent(HealthStateDegraded, fmt.Sprintf("capacity decreased from %d to %d", prev, currentMax))
		}
	} else {
		if currentMax > prev {
			al.emitStateEvent(HealthStateHealthy, fmt.Sprintf("capacity increased from %d to %d", prev, currentMax))
		}
	}
	al.prevMaxConcurrency = currentMax
}

// GetMaxConcurrency returns the current maximum concurrency limit.
func (al *AdaptiveLimiter) GetMaxConcurrency() int32 {
	return atomic.LoadInt32(&al.maxConcurrency)
}

// Health returns a channel that emits state change events.
// The channel is created lazily on first access.
// The caller should use a select with default to handle non-blocking receive.
func (al *AdaptiveLimiter) Health() <-chan StateEvent {
	al.mu.Lock()
	defer al.mu.Unlock()

	if !al.healthChInit {
		al.healthCh = make(chan StateEvent, 10)
		al.healthChInit = true
	}
	return al.healthCh
}

func (al *AdaptiveLimiter) emitStateEvent(state HealthState, message string) {
	if al.healthChInit {
		select {
		case al.healthCh <- StateEvent{
			Component: "adaptive-limiter",
			State:     state,
			Timestamp: time.Now(),
			Message:   message,
		}:
		default:
		}
	}
}
