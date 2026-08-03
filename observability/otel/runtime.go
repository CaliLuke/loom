package otel

import (
	"context"
	"errors"
	"slices"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/log"
	otelglobal "go.opentelemetry.io/otel/log/global"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/trace"
)

type (
	// Config configures framework-owned OpenTelemetry bootstrap.
	Config struct {
		// ServiceName is the logical service identity exported on resources.
		ServiceName string
		// ServiceVersion is the service version exported on resources.
		ServiceVersion string
		// Environment is the deployment environment exported on resources.
		Environment string
		// ResourceAttributes are merged into the provider resource.
		ResourceAttributes []attribute.KeyValue
		// Propagators configures request propagation for traces and logs.
		Propagators propagation.TextMapPropagator
		// Traces configures trace provider bootstrap.
		Traces TraceConfig
		// Metrics configures metric provider bootstrap.
		Metrics MetricConfig
		// Logs configures log provider bootstrap.
		Logs LogConfig
	}

	// TraceConfig configures trace provider bootstrap.
	TraceConfig struct {
		// Enabled enables trace provider bootstrap.
		Enabled bool
		// Endpoint is the OTLP HTTP endpoint host:port.
		Endpoint string
		// Insecure disables TLS for OTLP HTTP export.
		Insecure bool
		// Headers are sent on OTLP HTTP export requests.
		Headers map[string]string
		// SampleRatio controls TraceID-based head sampling.
		SampleRatio float64
	}

	// MetricConfig configures metric provider bootstrap.
	MetricConfig struct {
		// Enabled enables metric provider bootstrap.
		Enabled bool
		// Endpoint is the OTLP HTTP endpoint host:port.
		Endpoint string
		// Insecure disables TLS for OTLP HTTP export.
		Insecure bool
		// Headers are sent on OTLP HTTP export requests.
		Headers map[string]string
	}

	// LogConfig configures log provider bootstrap.
	LogConfig struct {
		// Enabled enables log provider bootstrap.
		Enabled bool
		// Endpoint is the OTLP HTTP endpoint host:port.
		Endpoint string
		// Insecure disables TLS for OTLP HTTP export.
		Insecure bool
		// Headers are sent on OTLP HTTP export requests.
		Headers map[string]string
	}

	otlpHTTPConfig struct {
		endpoint string
		insecure bool
		headers  map[string]string
	}

	// Runtime contains the initialized OpenTelemetry providers.
	Runtime struct {
		// TracerProvider is the initialized trace provider, if enabled.
		TracerProvider trace.TracerProvider
		// MeterProvider is the initialized metric provider, if enabled.
		MeterProvider metric.MeterProvider
		// LoggerProvider is the initialized log provider, if enabled.
		LoggerProvider log.LoggerProvider
		// Propagators is the propagator set used by this runtime.
		Propagators propagation.TextMapPropagator
		// Shutdown closes initialized providers in reverse initialization order.
		Shutdown func(context.Context) error
	}
)

// New initializes the configured OpenTelemetry providers and returns the
// resulting runtime.
func New(ctx context.Context, cfg Config) (*Runtime, error) {
	res, err := newResource(cfg)
	if err != nil {
		return nil, err
	}

	propagators := cfg.Propagators
	if propagators == nil {
		propagators = propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{},
			propagation.Baggage{},
		)
	}

	rt := &Runtime{
		Propagators: propagators,
		Shutdown: func(context.Context) error {
			return nil
		},
	}

	var shutdowns []func(context.Context) error

	if cfg.Traces.Enabled {
		tp, err := newTraceProvider(ctx, cfg.Traces, res)
		if err != nil {
			return nil, err
		}
		rt.TracerProvider = tp
		otel.SetTracerProvider(tp)
		otel.SetTextMapPropagator(propagators)
		shutdowns = append(shutdowns, tp.Shutdown)
	}
	if cfg.Metrics.Enabled {
		mp, err := newMeterProvider(ctx, cfg.Metrics, res)
		if err != nil {
			return nil, err
		}
		rt.MeterProvider = mp
		otel.SetMeterProvider(mp)
		shutdowns = append(shutdowns, mp.Shutdown)
	}
	if cfg.Logs.Enabled {
		lp, err := newLoggerProvider(ctx, cfg.Logs, res)
		if err != nil {
			return nil, err
		}
		rt.LoggerProvider = lp
		otelglobal.SetLoggerProvider(lp)
		shutdowns = append(shutdowns, lp.Shutdown)
	}

	rt.Shutdown = func(ctx context.Context) error {
		var errs []error
		for _, shutdown := range slices.Backward(shutdowns) {
			if err := shutdown(ctx); err != nil {
				errs = append(errs, err)
			}
		}
		return errors.Join(errs...)
	}

	return rt, nil
}

func newResource(cfg Config) (*resource.Resource, error) {
	attrs := append([]attribute.KeyValue{}, cfg.ResourceAttributes...)
	if cfg.ServiceName != "" {
		attrs = append(attrs, attribute.String("service.name", cfg.ServiceName))
	}
	if cfg.ServiceVersion != "" {
		attrs = append(attrs, attribute.String("service.version", cfg.ServiceVersion))
	}
	if cfg.Environment != "" {
		attrs = append(attrs, attribute.String("deployment.environment", cfg.Environment))
	}
	return resource.Merge(resource.Default(), resource.NewSchemaless(attrs...))
}
