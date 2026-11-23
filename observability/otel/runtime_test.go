package otel

import (
	"context"
	"testing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace/noop"

	"github.com/stretchr/testify/require"
)

// restoreOTelGlobals snapshots the process-global OTel providers and
// propagator mutated by New and restores them when the test finishes. OTel's
// default delegates bind once to the first value ever set, so restoring the
// default does not unbind it; tests must not depend on the default delegate.
func restoreOTelGlobals(t *testing.T) {
	t.Helper()
	tp := otel.GetTracerProvider()
	mp := otel.GetMeterProvider()
	lp := otel.GetLoggerProvider()
	prop := otel.GetTextMapPropagator()
	t.Cleanup(func() {
		otel.SetTracerProvider(tp)
		otel.SetMeterProvider(mp)
		otel.SetLoggerProvider(lp)
		otel.SetTextMapPropagator(prop)
	})
}

func TestNewInitializesEnabledProvidersAndGlobals(t *testing.T) {
	restoreOTelGlobals(t)
	tpBefore := otel.GetTracerProvider()
	mpBefore := otel.GetMeterProvider()
	lpBefore := otel.GetLoggerProvider()
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
	require.NotEqual(t, lpBefore, otel.GetLoggerProvider())
	require.NotEqual(t, propBefore, otel.GetTextMapPropagator())
	require.NoError(t, rt.Shutdown(context.Background()))
}

func TestNewDisabledSectionsDoNotReplaceUnrelatedGlobals(t *testing.T) {
	restoreOTelGlobals(t)
	sentinel := noop.NewTracerProvider()
	otel.SetTracerProvider(sentinel)
	mpBefore := otel.GetMeterProvider()
	lpBefore := otel.GetLoggerProvider()

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
	require.Same(t, lpBefore, otel.GetLoggerProvider())
	require.NoError(t, rt.Shutdown(context.Background()))
}

func TestNewPublishesRuntimePropagators(t *testing.T) {
	explicit := &recordingPropagator{name: "explicit"}
	cases := []struct {
		name string
		cfg  Config
	}{
		{name: "nothing enabled", cfg: Config{}},
		{name: "traces only", cfg: Config{Traces: TraceConfig{Enabled: true}}},
		{name: "metrics only", cfg: Config{Metrics: MetricConfig{Enabled: true}}},
		{name: "logs only", cfg: Config{Logs: LogConfig{Enabled: true}}},
		{name: "explicit propagators", cfg: Config{Propagators: explicit, Logs: LogConfig{Enabled: true}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			restoreOTelGlobals(t)
			otel.SetTextMapPropagator(&recordingPropagator{name: "previous"})
			tc.cfg.ServiceName = "propagation"

			rt, err := New(context.Background(), tc.cfg)
			require.NoError(t, err)
			t.Cleanup(func() {
				require.NoError(t, rt.Shutdown(context.Background()))
			})

			require.Equal(t, rt.Propagators, otel.GetTextMapPropagator())
			if tc.cfg.Propagators != nil {
				require.Same(t, explicit, otel.GetTextMapPropagator())
			}
		})
	}
}

func TestNewSupportsLocalOnlyOperation(t *testing.T) {
	restoreOTelGlobals(t)
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
