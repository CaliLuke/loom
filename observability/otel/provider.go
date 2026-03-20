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
	opts := make([]otlptracehttp.Option, 0, 3)
	if cfg.Endpoint != "" {
		opts = append(opts, otlptracehttp.WithEndpoint(cfg.Endpoint))
	}
	if cfg.Insecure {
		opts = append(opts, otlptracehttp.WithInsecure())
	}
	if len(cfg.Headers) > 0 {
		opts = append(opts, otlptracehttp.WithHeaders(cfg.Headers))
	}
	return opts
}

func metricExporterOptions(cfg MetricConfig) []otlpmetrichttp.Option {
	opts := make([]otlpmetrichttp.Option, 0, 3)
	if cfg.Endpoint != "" {
		opts = append(opts, otlpmetrichttp.WithEndpoint(cfg.Endpoint))
	}
	if cfg.Insecure {
		opts = append(opts, otlpmetrichttp.WithInsecure())
	}
	if len(cfg.Headers) > 0 {
		opts = append(opts, otlpmetrichttp.WithHeaders(cfg.Headers))
	}
	return opts
}

func logExporterOptions(cfg LogConfig) []otlploghttp.Option {
	opts := make([]otlploghttp.Option, 0, 3)
	if cfg.Endpoint != "" {
		opts = append(opts, otlploghttp.WithEndpoint(cfg.Endpoint))
	}
	if cfg.Insecure {
		opts = append(opts, otlploghttp.WithInsecure())
	}
	if len(cfg.Headers) > 0 {
		opts = append(opts, otlploghttp.WithHeaders(cfg.Headers))
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
