// Copyright (c) 2026 Onur Cinar.
// The source code is provided under MIT License.
// https://github.com/cinar/resile

package circuit

import (
	"context"
	"errors"
	"sync"
	"time"
)

// State represents the current state of the circuit breaker.
type State int

const (
	// StateClosed represents a healthy state where requests are allowed.
	StateClosed State = iota
	// StateOpen represents a failed state where requests are fast-failed.
	StateOpen
	// StateHalfOpen represents a state where a limited number of probes are allowed.
	StateHalfOpen
)

var (
	// ErrCircuitOpen is returned when the circuit breaker is in the Open state.
	ErrCircuitOpen = errors.New("circuit breaker is open")
)

// WindowType specifies whether the window is count-based or time-based.
type WindowType int

const (
	// WindowCountBased uses the last N calls to calculate failure rate.
	WindowCountBased WindowType = iota
	// WindowTimeBased uses calls in the last N duration to calculate failure rate.
	WindowTimeBased
)

const (
	// NumBuckets is the number of buckets used for time-based sliding window.
	NumBuckets = 10
)

// Config defines the configuration for the circuit breaker.
type Config struct {
	// WindowType specifies the type of sliding window.
	WindowType WindowType

	// WindowSize is the size of the count-based sliding window.
	// Default is 100.
	WindowSize uint32

	// WindowDuration is the duration of the time-based sliding window.
	// Default is 60 seconds.
	WindowDuration time.Duration

	// FailureRateThreshold is the failure rate percentage (0.0 to 100.0).
	// If failures/total >= FailureRateThreshold/100, the circuit opens.
	// Default is 50.0.
	FailureRateThreshold float64

	// MinimumCalls is the minimum number of calls in the current window before the failure rate is calculated.
	// Default is 10.
	MinimumCalls uint64

	// ResetTimeout is the duration to wait in Open state before moving to Half-Open.
	// Default is 60 seconds.
	ResetTimeout time.Duration

	// HalfOpenMaxCalls is the number of calls allowed in Half-Open state.
	// Default is 10.
	HalfOpenMaxCalls uint64
}

type bucket struct {
	success uint64
	failure uint64
}

// Breaker implements a circuit breaker state machine with sliding windows.
type Breaker struct {
	mu sync.RWMutex

	config Config
	state  State

	openedAt time.Time

	// Count-based Window fields
	ring     []bool
	ringIdx  uint32
	ringFull bool
	total    uint64
	failures uint64

	// Time-based Window fields
	buckets        []bucket
	bucketDuration time.Duration
	lastUpdateTime time.Time

	// Half-Open probe counters
	halfOpenCalls     uint64
	halfOpenFailures  uint64
	halfOpenCompleted uint64
}

// New returns a new Breaker with the provided configuration.
func New(cfg Config) *Breaker {
	// Apply defaults and validate
	if cfg.WindowSize == 0 {
		cfg.WindowSize = 100
	}
	if cfg.WindowDuration <= 0 {
		cfg.WindowDuration = 60 * time.Second
	}
	if cfg.FailureRateThreshold <= 0 || cfg.FailureRateThreshold > 100 {
		cfg.FailureRateThreshold = 50.0
	}
	if cfg.MinimumCalls == 0 {
		cfg.MinimumCalls = 10
	}
	if cfg.ResetTimeout <= 0 {
		cfg.ResetTimeout = 60 * time.Second
	}
	if cfg.HalfOpenMaxCalls == 0 {
		cfg.HalfOpenMaxCalls = 10
	}

	b := &Breaker{
		config: cfg,
		state:  StateClosed,
	}

	if cfg.WindowType == WindowCountBased {
		b.ring = make([]bool, cfg.WindowSize)
	} else {
		b.buckets = make([]bucket, NumBuckets)
		b.bucketDuration = cfg.WindowDuration / time.Duration(NumBuckets)
		if b.bucketDuration == 0 {
			b.bucketDuration = time.Second
		}
	}

	return b
}

// State returns the current state of the breaker.
func (b *Breaker) State() State {
	b.mu.RLock()
	state := b.state
	openedAt := b.openedAt
	b.mu.RUnlock()

	if state == StateOpen && time.Since(openedAt) > b.config.ResetTimeout {
		b.mu.Lock()
		defer b.mu.Unlock()
		// Double check after acquiring write lock.
		if b.state == StateOpen && time.Since(b.openedAt) > b.config.ResetTimeout {
			b.transitionToHalfOpen()
		}
		return b.state
	}

	return state
}

// Execute wraps an action with the circuit breaker logic.
func (b *Breaker) Execute(ctx context.Context, action func() error) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	b.mu.Lock()
	// Check if Open -> Half-Open transition is needed.
	if b.state == StateOpen && time.Since(b.openedAt) > b.config.ResetTimeout {
		b.transitionToHalfOpen()
	}

	if b.state == StateOpen {
		b.mu.Unlock()
		return ErrCircuitOpen
	}

	// For Half-Open, we limit the number of probe calls.
	if b.state == StateHalfOpen {
		if b.halfOpenCalls >= b.config.HalfOpenMaxCalls {
			b.mu.Unlock()
			return ErrCircuitOpen
		}
		b.halfOpenCalls++
	}
	state := b.state
	b.mu.Unlock()

	err := action()

	b.mu.Lock()
	defer b.mu.Unlock()

	if err != nil {
		b.onFailure(state)
		return err
	}

	b.onSuccess(state)
	return nil
}

// Reset manually resets the circuit breaker to Closed state and clears the window.
func (b *Breaker) Reset() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.transitionToClosed()
}

func (b *Breaker) onFailure(state State) {
	if state == StateHalfOpen {
		b.halfOpenFailures++
		b.halfOpenCompleted++
		// Any failure in Half-Open trips it back to Open.
		b.transitionToOpen()
		return
	}

	b.record(false)
	b.checkFailureRate()
}

func (b *Breaker) onSuccess(state State) {
	if state == StateHalfOpen {
		b.halfOpenCompleted++
		if b.halfOpenCompleted >= b.config.HalfOpenMaxCalls {
			// Calculate failure rate for all Half-Open probes.
			rate := (float64(b.halfOpenFailures) / float64(b.halfOpenCompleted)) * 100.0
			if rate < b.config.FailureRateThreshold {
				b.transitionToClosed()
			} else {
				b.transitionToOpen()
			}
		}
		return
	}

	b.record(true)
	b.checkFailureRate()
}

func (b *Breaker) transitionToOpen() {
	b.state = StateOpen
	b.openedAt = time.Now()
	// Clear window stats when opening.
	b.clearWindow()
}

func (b *Breaker) transitionToHalfOpen() {
	b.state = StateHalfOpen
	b.halfOpenCalls = 0
	b.halfOpenFailures = 0
	b.halfOpenCompleted = 0
}

func (b *Breaker) transitionToClosed() {
	b.state = StateClosed
	b.clearWindow()
}

func (b *Breaker) clearWindow() {
	if b.config.WindowType == WindowCountBased {
		clear(b.ring)
		b.ringIdx = 0
		b.ringFull = false
		b.total = 0
		b.failures = 0
	} else {
		clear(b.buckets)
		b.lastUpdateTime = time.Time{}
	}
}

func (b *Breaker) record(success bool) {
	if b.config.WindowType == WindowCountBased {
		b.recordCountBased(success)
	} else {
		b.recordTimeBased(success, time.Now())
	}
}

func (b *Breaker) recordCountBased(success bool) {
	if b.ringFull {
		oldSuccess := b.ring[b.ringIdx]
		if !oldSuccess {
			b.failures--
		}
	} else {
		b.total++
	}

	b.ring[b.ringIdx] = success
	if !success {
		b.failures++
	}

	b.ringIdx = (b.ringIdx + 1) % uint32(len(b.ring))
	if b.ringIdx == 0 {
		b.ringFull = true
	}
}

func (b *Breaker) recordTimeBased(success bool, now time.Time) {
	b.advanceTimeBasedWindow(now)
	idx := b.getBucketIdx(now)
	if success {
		b.buckets[idx].success++
	} else {
		b.buckets[idx].failure++
	}
	b.lastUpdateTime = now
}

func (b *Breaker) advanceTimeBasedWindow(now time.Time) {
	if b.lastUpdateTime.IsZero() {
		b.lastUpdateTime = now
		return
	}

	diff := now.Sub(b.lastUpdateTime)
	if diff > b.config.WindowDuration {
		clear(b.buckets)
	} else if diff >= b.bucketDuration {
		lastIdx := b.getBucketIdx(b.lastUpdateTime)
		currIdx := b.getBucketIdx(now)
		for i := (lastIdx + 1) % NumBuckets; ; i = (i + 1) % NumBuckets {
			b.buckets[i] = bucket{}
			if i == currIdx {
				break
			}
		}
	}
}

func (b *Breaker) getBucketIdx(t time.Time) int {
	nano := t.UnixNano()
	if nano < 0 {
		nano = 0
	}
	return int((nano / int64(b.bucketDuration)) % int64(NumBuckets))
}

func (b *Breaker) checkFailureRate() {
	if b.state != StateClosed {
		return
	}

	total, failures := b.getStatsLocked()
	if total < b.config.MinimumCalls {
		return
	}

	rate := (float64(failures) / float64(total)) * 100.0
	if rate >= b.config.FailureRateThreshold {
		b.transitionToOpen()
	}
}

func (b *Breaker) getStatsLocked() (total uint64, failures uint64) {
	if b.config.WindowType == WindowCountBased {
		return b.total, b.failures
	}

	// Time-based
	b.advanceTimeBasedWindow(time.Now())
	var success uint64
	for _, bu := range b.buckets {
		success += bu.success
		failures += bu.failure
	}
	total = success + failures
	return
}
