// Copyright (c) 2026 Onur Cinar.
// The source code is provided under MIT License.
// https://github.com/cinar/resile

package resile

import (
	"testing"
	"time"
)

func TestFullJitter_Next(t *testing.T) {
	t.Parallel()

	base := 100 * time.Millisecond
	cap := 10 * time.Second
	bj := NewFullJitter(base, cap)

	t.Run("DelaysAreWithinRange", func(t *testing.T) {
		for attempt := uint(0); attempt < 20; attempt++ {
			delay := bj.Next(attempt)
			if delay < 0 {
				t.Errorf("attempt %d: negative delay %v", attempt, delay)
			}
			if delay > cap {
				t.Errorf("attempt %d: delay %v exceeds cap %v", attempt, delay, cap)
			}
		}
	})

	t.Run("CappingWorks", func(t *testing.T) {
		// A high attempt count should definitely hit the cap.
		// base * 2^100 would otherwise overflow any duration.
		for i := 0; i < 100; i++ {
			delay := bj.Next(100)
			if delay > cap {
				t.Fatalf("delay %v exceeded cap %v at attempt 100", delay, cap)
			}
		}
	})

	t.Run("Distribution", func(t *testing.T) {
		// Verify that different attempts yield different results.
		// Note: Since it's random, we check that it's not always the same value.
		lastDelay := bj.Next(0)
		diffCount := 0
		for i := 0; i < 10; i++ {
			delay := bj.Next(0)
			if delay != lastDelay {
				diffCount++
			}
			lastDelay = delay
		}
		if diffCount == 0 {
			t.Error("Next(0) returned the same value 10 times, distribution likely broken")
		}
	})
}