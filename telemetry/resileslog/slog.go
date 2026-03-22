// Copyright (c) 2026 Onur Cinar.
// The source code is provided under MIT License.
// https://github.com/cinar/resile

package resileslog

import (
	"context"
	"log/slog"

	"github.com/cinar/resile"
)

type slogInstrumenter struct {
	logger *slog.Logger
}

// New returns a new resile.Instrumenter that logs retry attempts using slog.
func New(logger *slog.Logger) resile.Instrumenter {
	return &slogInstrumenter{
		logger: logger,
	}
}

func (s *slogInstrumenter) BeforeAttempt(ctx context.Context, state resile.RetryState) context.Context {
	// We don't mutate the context in the slog implementation.
	return ctx
}

func (s *slogInstrumenter) AfterAttempt(ctx context.Context, state resile.RetryState) {
	// We only log if there was an error and it's not the first attempt,
	// or if it was the last attempt.
	if state.LastError == nil {
		return
	}

	level := slog.LevelWarn
	msg := "retry attempt failed"

	attrs := []slog.Attr{
		slog.Uint64("retry.attempt", uint64(state.Attempt)),
		slog.Duration("retry.total_duration", state.TotalDuration),
		slog.String("error.message", state.LastError.Error()),
	}

	if state.Name != "" {
		attrs = append(attrs, slog.String("retry.name", state.Name))
	}

	if state.NextDelay > 0 {
		attrs = append(attrs, slog.Duration("retry.next_delay", state.NextDelay))
	}

	s.logger.LogAttrs(ctx, level, msg, attrs...)
}

func (s *slogInstrumenter) OnBulkheadFull(ctx context.Context, state resile.RetryState) {
	s.logger.WarnContext(ctx, "resile bulkhead capacity reached",
		slog.String("resile.name", state.Name),
	)
}

func (s *slogInstrumenter) OnRateLimitExceeded(ctx context.Context, state resile.RetryState) {
	s.logger.WarnContext(ctx, "resile rate limit exceeded",
		slog.String("resile.name", state.Name),
	)
}
