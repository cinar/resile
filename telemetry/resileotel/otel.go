// Copyright (c) 2026 Onur Cinar.
// The source code is provided under MIT License.
// https://github.com/cinar/resile

package resileotel

import (
	"context"
	"fmt"

	"github.com/cinar/resile"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

const (
	instrumentationName = "github.com/cinar/resile"
)

type otelInstrumenter struct {
	tracer  trace.Tracer
	counter metric.Int64Counter
}

// New returns a new resile.Instrumenter that exports spans and metrics via OpenTelemetry.
func New(tp trace.TracerProvider, mp metric.MeterProvider) (resile.Instrumenter, error) {
	if tp == nil {
		tp = otel.GetTracerProvider()
	}
	if mp == nil {
		mp = otel.GetMeterProvider()
	}

	tracer := tp.Tracer(instrumentationName)
	meter := mp.Meter(instrumentationName)

	counter, err := meter.Int64Counter("resile_retries_total",
		metric.WithDescription("Total number of retry attempts"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create counter: %w", err)
	}

	return &otelInstrumenter{
		tracer:  tracer,
		counter: counter,
	}, nil
}

func (o *otelInstrumenter) BeforeAttempt(ctx context.Context, state resile.RetryState) context.Context {
	// Start a new span for each retry attempt.
	spanName := "retry.attempt"
	if state.Name != "" {
		spanName = fmt.Sprintf("%s: %s", spanName, state.Name)
	}

	ctx, _ = o.tracer.Start(ctx, spanName,
		trace.WithAttributes(
			attribute.Int64("retry.num", int64(state.Attempt)),
			attribute.String("retry.name", state.Name),
		),
	)

	// Increment the retry counter.
	// We use Attempt > 0 to count actual retries (excluding the first attempt).
	if state.Attempt > 0 {
		attrs := []attribute.KeyValue{
			attribute.Int64("retry_num", int64(state.Attempt)),
			attribute.String("error_type", fmt.Sprintf("%T", state.LastError)),
		}
		if state.Name != "" {
			attrs = append(attrs, attribute.String("retry_name", state.Name))
		}
		o.counter.Add(ctx, 1, metric.WithAttributes(attrs...))
	}

	return ctx
}

func (o *otelInstrumenter) AfterAttempt(ctx context.Context, state resile.RetryState) {
	span := trace.SpanFromContext(ctx)
	defer span.End()

	if state.LastError != nil {
		span.RecordError(state.LastError)
		span.SetStatus(codes.Error, state.LastError.Error())
		span.SetAttributes(
			attribute.String("error.type", fmt.Sprintf("%T", state.LastError)),
		)
	} else {
		span.SetStatus(codes.Ok, "success")
	}

	if state.NextDelay > 0 {
		span.SetAttributes(
			attribute.String("retry.backoff_duration", state.NextDelay.String()),
		)
	}
}