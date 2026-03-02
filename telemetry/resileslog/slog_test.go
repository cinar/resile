// Copyright (c) 2026 Onur Cinar.
// The source code is provided under MIT License.
// https://github.com/cinar/resile

package resileslog

import (
	"bytes"
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/cinar/resile"
)

func TestSlogInstrumenter(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))
	instr := New(logger)

	ctx := context.Background()
	state := resile.RetryState{
		Attempt:       1,
		MaxAttempts:   3,
		LastError:     context.DeadlineExceeded,
		TotalDuration: 100 * time.Millisecond,
		NextDelay:     200 * time.Millisecond,
	}

	instr.AfterAttempt(ctx, state)

	output := buf.String()
	if output == "" {
		t.Fatal("expected log output, got nothing")
	}

	// Verify JSON content contains expected fields.
	expectedFields := []string{
		`"retry.attempt":1`,
		`"error.message":"context deadline exceeded"`,
		`"retry.next_delay":200000000`,
	}

	for _, field := range expectedFields {
		if !contains(output, field) {
			t.Errorf("expected field %s in log output: %s", field, output)
		}
	}
}

func TestSlogInstrumenter_WithName(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))
	instr := New(logger)

	ctx := context.Background()
	state := resile.RetryState{
		Name:      "test-op",
		LastError: context.DeadlineExceeded,
	}

	instr.AfterAttempt(ctx, state)
	output := buf.String()
	if !contains(output, `"retry.name":"test-op"`) {
		t.Errorf("expected retry.name in log: %s", output)
	}
}

func TestSlogInstrumenter_BeforeAttempt(t *testing.T) {
	instr := New(slog.Default())
	ctx := context.Background()
	newCtx := instr.BeforeAttempt(ctx, resile.RetryState{})
	if newCtx != ctx {
		t.Error("BeforeAttempt should return the original context")
	}
}

func TestSlogInstrumenter_AfterAttempt_Success(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))
	instr := New(logger)
	instr.AfterAttempt(context.Background(), resile.RetryState{LastError: nil})
	if buf.Len() > 0 {
		t.Error("AfterAttempt should not log on success")
	}
}

func contains(s, substr string) bool {
	return bytes.Contains([]byte(s), []byte(substr))
}
