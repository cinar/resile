// Copyright (c) 2026 Onur Cinar.
// The source code is provided under MIT License.
// https://github.com/cinar/resile

package resile

import (
	"errors"
	"testing"
)

var errTest = errors.New("test error")
var errOther = errors.New("other error")

func TestFatalError(t *testing.T) {
	t.Parallel()

	t.Run("WrapsError", func(t *testing.T) {
		err := FatalError(errTest)
		if err.Error() != errTest.Error() {
			t.Errorf("expected %q, got %q", errTest.Error(), err.Error())
		}
	})

	t.Run("UnwrapsError", func(t *testing.T) {
		err := FatalError(errTest)
		if !errors.Is(err, errTest) {
			t.Errorf("expected errors.Is(err, errTest) to be true")
		}
	})

	t.Run("IdentifiesFatal", func(t *testing.T) {
		err := FatalError(errTest)
		if !isFatal(err) {
			t.Error("expected isFatal(err) to be true")
		}
	})

	t.Run("NestedFatal", func(t *testing.T) {
		err := FatalError(errTest)
		wrapped := errors.Join(err, errOther)
		if !isFatal(wrapped) {
			t.Error("expected isFatal(wrapped) to be true for nested fatal error")
		}
	})

	t.Run("NilFatal", func(t *testing.T) {
		if FatalError(nil) != nil {
			t.Error("FatalError(nil) should be nil")
		}
	})
}

func TestRetryPolicy_ShouldRetry(t *testing.T) {
	t.Parallel()

	t.Run("DefaultPolicy", func(t *testing.T) {
		policy := &retryPolicy{}
		if !policy.shouldRetry(errTest) {
			t.Error("expected default policy to retry errors")
		}
		if policy.shouldRetry(FatalError(errTest)) {
			t.Error("expected default policy to NOT retry fatal errors")
		}
	})

	t.Run("RetryIfTarget", func(t *testing.T) {
		policy := &retryPolicy{retryIf: errTest}
		if !policy.shouldRetry(errTest) {
			t.Error("expected policy to retry errTest")
		}
		if policy.shouldRetry(errOther) {
			t.Error("expected policy to NOT retry errOther")
		}
	})

	t.Run("RetryIfFunc", func(t *testing.T) {
		policy := &retryPolicy{
			retryIfFunc: func(err error) bool {
				return errors.Is(err, errTest)
			},
		}
		if !policy.shouldRetry(errTest) {
			t.Error("expected policy to retry errTest via func")
		}
		if policy.shouldRetry(errOther) {
			t.Error("expected policy to NOT retry errOther via func")
		}
	})

	t.Run("FatalOverridesPolicy", func(t *testing.T) {
		policy := &retryPolicy{retryIf: errTest}
		if policy.shouldRetry(FatalError(errTest)) {
			t.Error("expected fatal error to override retryIf policy")
		}
	})
}