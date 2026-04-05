package testkit

import (
	"context"
	"testing"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
)

type (
	// TraceHarness captures ended spans for tests.
	TraceHarness struct {
		Provider *sdktrace.TracerProvider
		Recorder *tracetest.SpanRecorder
	}
)

// NewTraceHarness creates a trace provider backed by a span recorder.
func NewTraceHarness(tb testing.TB) *TraceHarness {
	tb.Helper()
	recorder := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
	tb.Cleanup(func() {
		if err := provider.Shutdown(context.Background()); err != nil {
			tb.Errorf("shutdown trace provider: %v", err)
		}
	})
	return &TraceHarness{Provider: provider, Recorder: recorder}
}

// TracerProvider returns the configured trace provider.
func (h *TraceHarness) TracerProvider() trace.TracerProvider {
	return h.Provider
}
