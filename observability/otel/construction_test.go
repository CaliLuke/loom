package otel

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
)

func TestFailedConstructionPreservesGlobals(t *testing.T) {
	for _, failed := range []string{"metric", "log"} {
		t.Run(failed, func(t *testing.T) {
			restoreOTelGlobals(t)
			traces := otel.GetTracerProvider()
			metrics := otel.GetMeterProvider()
			logs := otel.GetLoggerProvider()
			propagation := otel.GetTextMapPropagator()
			// Parent behavior publishes resources before failure. Close any
			// escaped providers so the red regression does not leave workers.
			t.Cleanup(func() {
				for _, provider := range []any{otel.GetLoggerProvider(), otel.GetMeterProvider(), otel.GetTracerProvider()} {
					if provider == logs || provider == metrics || provider == traces {
						continue
					}
					if owned, ok := provider.(interface{ Shutdown(context.Context) error }); ok {
						require.NoError(t, owned.Shutdown(context.Background()))
					}
				}
			})
			cfg := Config{ServiceName: "construction", Traces: TraceConfig{Enabled: true}, Metrics: MetricConfig{Enabled: true}, Logs: LogConfig{Enabled: true}}
			switch failed {
			case "metric":
				cfg.Metrics.Endpoint = "a b"
			case "log":
				cfg.Logs.Endpoint = "a b"
			}
			runtime, err := New(context.Background(), cfg)
			require.Error(t, err)
			require.Nil(t, runtime)
			for _, check := range []struct {
				name string
				want any
				got  any
			}{
				{"trace", traces, otel.GetTracerProvider()},
				{"metric", metrics, otel.GetMeterProvider()},
				{"log", logs, otel.GetLoggerProvider()},
				{"propagator", propagation, otel.GetTextMapPropagator()},
			} {
				t.Run(check.name, func(t *testing.T) {
					if check.name == "propagator" {
						require.Equal(t, check.want, check.got)
					} else {
						require.Same(t, check.want, check.got)
					}
				})
			}
		})
	}
}

func TestProviderShutdownOrderAndErrors(t *testing.T) {
	for _, count := range []int{0, 1, 3} {
		t.Run(fmt.Sprint(count), func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			var calls []int
			shutdowns := make([]func(context.Context) error, 0, count)
			causes := make([]error, 0, count)
			for i := range count {
				cause := fmt.Errorf("shutdown %d", i)
				causes = append(causes, cause)
				shutdowns = append(shutdowns, func(received context.Context) error {
					require.Same(t, ctx, received)
					require.ErrorIs(t, received.Err(), context.Canceled)
					calls = append(calls, i)
					return cause
				})
			}
			err := shutdownProviders(ctx, shutdowns)
			if count == 0 {
				require.NoError(t, err)
			}
			require.Len(t, calls, count)
			for i, cause := range causes {
				require.ErrorIs(t, err, cause)
				require.Equal(t, count-i-1, calls[i])
			}
		})
	}
}

func TestProviderRollbackContextAndErrors(t *testing.T) {
	type contextKey struct{}
	cause := errors.New("construction")
	cleanupError := errors.New("cleanup")
	for _, canceled := range []bool{false, true} {
		t.Run(fmt.Sprint(canceled), func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.WithValue(context.Background(), contextKey{}, "retained"), time.Millisecond)
			defer cancel()
			if canceled {
				cancel()
			}
			before := time.Now()
			var used context.Context
			calls := 0
			err := rollbackProviders(ctx, []func(context.Context) error{
				func(received context.Context) error {
					calls++
					used = received
					require.NoError(t, received.Err())
					require.Equal(t, "retained", received.Value(contextKey{}))
					deadline, ok := received.Deadline()
					require.True(t, ok)
					require.False(t, deadline.Before(before.Add(rollbackTimeout)))
					require.False(t, deadline.After(time.Now().Add(rollbackTimeout)))
					return cleanupError
				},
			}, cause)
			require.Equal(t, 1, calls)
			require.ErrorIs(t, err, cause)
			require.ErrorIs(t, err, cleanupError)
			require.ErrorIs(t, used.Err(), context.Canceled)
			require.ErrorIs(t, rollbackProviders(ctx, nil, cause), cause)
		})
	}
}
