// Copyright (c) 2026 Onur Cinar.
// The source code is provided under MIT License.
// https://github.com/cinar/resile

package resile

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"
)

// ErrShedLoad is returned when the adaptive concurrency limit is reached.
var ErrShedLoad = errors.New("load shed: adaptive concurrency limit reached")

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
}

// NewAdaptiveLimiter creates a new AdaptiveLimiter with default configurations.
func NewAdaptiveLimiter() *AdaptiveLimiter {
	return &AdaptiveLimiter{
		maxConcurrency:  10,  // Initial guess
		minConcurrency:  1,   // Floor
		limitScaling:    0.8, // Multiplicative Decrease factor
		threshold:       1.5, // Threshold for increase/decrease
		alpha:           0.1, // EWMA smoothing factor
		decayInterval:   time.Minute,
		lastDecayUpdate: time.Now(),
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
	} else {
		// Multiplicative Decrease
		newMax := float64(atomic.LoadInt32(&al.maxConcurrency)) * al.limitScaling
		if newMax < float64(al.minConcurrency) {
			newMax = float64(al.minConcurrency)
		}
		atomic.StoreInt32(&al.maxConcurrency, int32(newMax))
	}
}

// GetMaxConcurrency returns the current maximum concurrency limit.
func (al *AdaptiveLimiter) GetMaxConcurrency() int32 {
	return atomic.LoadInt32(&al.maxConcurrency)
}
