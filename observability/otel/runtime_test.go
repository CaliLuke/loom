package otel

import (
	"context"
	"testing"

	"go.opentelemetry.io/otel"
	otelglobal "go.opentelemetry.io/otel/log/global"
	"go.opentelemetry.io/otel/trace/noop"

	"github.com/stretchr/testify/require"
)

func TestNewInitializesEnabledProvidersAndGlobals(t *testing.T) {
	tpBefore := otel.GetTracerProvider()
	mpBefore := otel.GetMeterProvider()
	lpBefore := otelglobal.GetLoggerProvider()
	propBefore := otel.GetTextMapPropagator()

	rt, err := New(context.Background(), Config{
		ServiceName:    "autok",
		ServiceVersion: "v1.2.3",
		Environment:    "test",
		Traces: TraceConfig{
			Enabled:     true,
			SampleRatio: 0.5,
		},
		Metrics: MetricConfig{
			Enabled: true,
		},
		Logs: LogConfig{
			Enabled: true,
		},
	})
	require.NoError(t, err)
	require.NotNil(t, rt.TracerProvider)
	require.NotNil(t, rt.MeterProvider)
	require.NotNil(t, rt.LoggerProvider)
	require.NotNil(t, rt.Propagators)
	require.NotEqual(t, tpBefore, otel.GetTracerProvider())
	require.NotEqual(t, mpBefore, otel.GetMeterProvider())
	require.NotEqual(t, lpBefore, otelglobal.GetLoggerProvider())
	require.NotEqual(t, propBefore, otel.GetTextMapPropagator())
	require.NoError(t, rt.Shutdown(context.Background()))
}

func TestNewDisabledSectionsDoNotReplaceUnrelatedGlobals(t *testing.T) {
	sentinel := noop.NewTracerProvider()
	otel.SetTracerProvider(sentinel)
	mpBefore := otel.GetMeterProvider()
	lpBefore := otelglobal.GetLoggerProvider()

	rt, err := New(context.Background(), Config{
		ServiceName: "metrics-only",
		Metrics: MetricConfig{
			Enabled: true,
		},
	})
	require.NoError(t, err)
	require.NotNil(t, rt.MeterProvider)
	require.Equal(t, sentinel, otel.GetTracerProvider())
	require.NotEqual(t, mpBefore, otel.GetMeterProvider())
	require.Same(t, lpBefore, otelglobal.GetLoggerProvider())
	require.NoError(t, rt.Shutdown(context.Background()))
}

func TestNewSupportsLocalOnlyOperation(t *testing.T) {
	rt, err := New(context.Background(), Config{
		ServiceName: "local-only",
		Traces: TraceConfig{
			Enabled: true,
		},
		Metrics: MetricConfig{
			Enabled: true,
		},
		Logs: LogConfig{
			Enabled: true,
		},
	})
	require.NoError(t, err)
	require.NotNil(t, rt.TracerProvider)
	require.NotNil(t, rt.MeterProvider)
	require.NotNil(t, rt.LoggerProvider)
	require.NoError(t, rt.Shutdown(context.Background()))
}
