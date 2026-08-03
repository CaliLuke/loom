package otel

import (
	"context"

	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func newTraceProvider(ctx context.Context, cfg TraceConfig, res *resource.Resource) (*sdktrace.TracerProvider, error) {
	opts := []sdktrace.TracerProviderOption{
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.TraceIDRatioBased(validSampleRatio(cfg.SampleRatio))),
	}
	if cfg.Endpoint != "" {
		exporter, err := otlptracehttp.New(ctx, traceExporterOptions(cfg)...)
		if err != nil {
			return nil, err
		}
		opts = append(opts, sdktrace.WithBatcher(exporter))
	}
	return sdktrace.NewTracerProvider(opts...), nil
}

func newMeterProvider(ctx context.Context, cfg MetricConfig, res *resource.Resource) (*sdkmetric.MeterProvider, error) {
	opts := []sdkmetric.Option{
		sdkmetric.WithResource(res),
	}
	if cfg.Endpoint != "" {
		exporter, err := otlpmetrichttp.New(ctx, metricExporterOptions(cfg)...)
		if err != nil {
			return nil, err
		}
		opts = append(opts, sdkmetric.WithReader(sdkmetric.NewPeriodicReader(exporter)))
	}
	return sdkmetric.NewMeterProvider(opts...), nil
}

func newLoggerProvider(ctx context.Context, cfg LogConfig, res *resource.Resource) (*sdklog.LoggerProvider, error) {
	opts := []sdklog.LoggerProviderOption{
		sdklog.WithResource(res),
	}
	if cfg.Endpoint != "" {
		exporter, err := otlploghttp.New(ctx, logExporterOptions(cfg)...)
		if err != nil {
			return nil, err
		}
		opts = append(opts, sdklog.WithProcessor(sdklog.NewBatchProcessor(exporter)))
	}
	return sdklog.NewLoggerProvider(opts...), nil
}

func traceExporterOptions(cfg TraceConfig) []otlptracehttp.Option {
	return otlpHTTPConfig{cfg.Endpoint, cfg.Insecure, cfg.Headers}.options(
		otlptracehttp.WithEndpoint,
		otlptracehttp.WithInsecure,
		otlptracehttp.WithHeaders,
	)
}

func metricExporterOptions(cfg MetricConfig) []otlpmetrichttp.Option {
	return otlpHTTPConfig{cfg.Endpoint, cfg.Insecure, cfg.Headers}.options(
		otlpmetrichttp.WithEndpoint,
		otlpmetrichttp.WithInsecure,
		otlpmetrichttp.WithHeaders,
	)
}

func logExporterOptions(cfg LogConfig) []otlploghttp.Option {
	return otlpHTTPConfig{cfg.Endpoint, cfg.Insecure, cfg.Headers}.options(
		otlploghttp.WithEndpoint,
		otlploghttp.WithInsecure,
		otlploghttp.WithHeaders,
	)
}

func (cfg otlpHTTPConfig) options[T any](
	withEndpoint func(string) T,
	withInsecure func() T,
	withHeaders func(map[string]string) T,
) []T {
	opts := make([]T, 0, 3)
	if cfg.endpoint != "" {
		opts = append(opts, withEndpoint(cfg.endpoint))
	}
	if cfg.insecure {
		opts = append(opts, withInsecure())
	}
	if len(cfg.headers) > 0 {
		opts = append(opts, withHeaders(cfg.headers))
	}
	return opts
}

func validSampleRatio(ratio float64) float64 {
	switch {
	case ratio <= 0:
		return 1
	case ratio > 1:
		return 1
	default:
		return ratio
	}
}
