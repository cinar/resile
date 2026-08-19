// Copyright (c) 2026 Onur Cinar.
// The source code is provided under MIT License.
// https://github.com/cinar/resile

package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"testing"
)

func TestShouldRetry(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"canceled", context.Canceled, false},
		{"deadline", context.DeadlineExceeded, false},
		{"transport", errors.New("connection reset by peer"), true},
		{"wrapped transport", fmt.Errorf("request failed: %w", errors.New("connection reset by peer")), true},
		{"500", &HTTPError{StatusCode: http.StatusInternalServerError}, true},
		{"502", &HTTPError{StatusCode: http.StatusBadGateway}, true},
		{"504", &HTTPError{StatusCode: http.StatusGatewayTimeout}, true},
		{"wrapped 500", fmt.Errorf("do: %w", &HTTPError{StatusCode: http.StatusInternalServerError}), true},
		{"400", &HTTPError{StatusCode: http.StatusBadRequest}, false},
		{"401", &HTTPError{StatusCode: http.StatusUnauthorized}, false},
		{"404", &HTTPError{StatusCode: http.StatusNotFound}, false},
		{"503", &HTTPError{StatusCode: http.StatusServiceUnavailable}, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := shouldRetry(tc.err); got != tc.want {
				t.Errorf("shouldRetry(%v) = %v, want %v", tc.err, got, tc.want)
			}
		})
	}
}
