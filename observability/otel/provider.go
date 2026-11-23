package otel

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"reflect"
	"strings"

	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// validateSections rejects configurations that the OTLP exporters would
// silently reinterpret, such as an unparsable URL that falls back to
// localhost, before New constructs anything.
func validateSections(cfg Config, traces, metrics, logs otlpConfig) error {
	var errs []error
	if cfg.Traces.Enabled {
		errs = append(errs, traces.validate())
		if len(cfg.Traces.BatchOptions) > 0 && !traces.enabled() {
			errs = append(errs, errors.New("otel: traces BatchOptions require Endpoint or EndpointURL"))
		}
		errs = append(errs, validEntries("traces SpanProcessors", cfg.Traces.SpanProcessors))
	}
	if cfg.Metrics.Enabled {
		errs = append(errs, metrics.validate())
		if len(cfg.Metrics.ReaderOptions) > 0 && !metrics.enabled() {
			errs = append(errs, errors.New("otel: metrics ReaderOptions require Endpoint or EndpointURL"))
		}
		errs = append(errs, validEntries("metrics Readers", cfg.Metrics.Readers))
	}
	if cfg.Logs.Enabled {
		errs = append(errs, logs.validate())
		if len(cfg.Logs.BatchOptions) > 0 && !logs.enabled() {
			errs = append(errs, errors.New("otel: logs BatchOptions require Endpoint or EndpointURL"))
		}
		errs = append(errs, validEntries("logs Processors", cfg.Logs.Processors))
	}
	return errors.Join(errs...)
}

func newTraceExporter(ctx context.Context, cfg otlpConfig) (sdktrace.SpanExporter, error) {
	if cfg.protocol == OTLPProtocolGRPC {
		return otlptracegrpc.New(ctx, otlpOptions(cfg,
			otlptracegrpc.WithEndpoint,
			otlptracegrpc.WithEndpointURL,
			otlptracegrpc.WithInsecure,
			otlptracegrpc.WithHeaders,
		)...)
	}
	return otlptracehttp.New(ctx, otlpOptions(cfg,
		otlptracehttp.WithEndpoint,
		otlptracehttp.WithEndpointURL,
		otlptracehttp.WithInsecure,
		otlptracehttp.WithHeaders,
	)...)
}

func newMetricExporter(ctx context.Context, cfg otlpConfig) (sdkmetric.Exporter, error) {
	if cfg.protocol == OTLPProtocolGRPC {
		return otlpmetricgrpc.New(ctx, otlpOptions(cfg,
			otlpmetricgrpc.WithEndpoint,
			otlpmetricgrpc.WithEndpointURL,
			otlpmetricgrpc.WithInsecure,
			otlpmetricgrpc.WithHeaders,
		)...)
	}
	return otlpmetrichttp.New(ctx, otlpOptions(cfg,
		otlpmetrichttp.WithEndpoint,
		otlpmetrichttp.WithEndpointURL,
		otlpmetrichttp.WithInsecure,
		otlpmetrichttp.WithHeaders,
	)...)
}

func newLogExporter(ctx context.Context, cfg otlpConfig) (sdklog.Exporter, error) {
	if cfg.protocol == OTLPProtocolGRPC {
		return otlploggrpc.New(ctx, otlpOptions(cfg,
			otlploggrpc.WithEndpoint,
			otlploggrpc.WithEndpointURL,
			otlploggrpc.WithInsecure,
			otlploggrpc.WithHeaders,
		)...)
	}
	return otlploghttp.New(ctx, otlpOptions(cfg,
		otlploghttp.WithEndpoint,
		otlploghttp.WithEndpointURL,
		otlploghttp.WithInsecure,
		otlploghttp.WithHeaders,
	)...)
}

// newExporters builds the OTLP exporter of every enabled section that has an
// endpoint. Exporters are the only fallible construction stage, so New builds
// them all before any provider: a failure leaves caller-supplied processors and
// readers untouched and shuts down only the exporters already built, in
// reverse order.
func newExporters(ctx context.Context, cfg Config, traces, metrics, logs otlpConfig) (sectionExporters, error) {
	var exp sectionExporters
	var acquired []func(context.Context) error
	if cfg.Traces.Enabled && traces.enabled() {
		span, err := newTraceExporter(ctx, traces)
		if err != nil {
			return sectionExporters{}, rollbackProviders(ctx, acquired, err)
		}
		exp.span = span
		acquired = append(acquired, span.Shutdown)
	}
	if cfg.Metrics.Enabled && metrics.enabled() {
		metric, err := newMetricExporter(ctx, metrics)
		if err != nil {
			return sectionExporters{}, rollbackProviders(ctx, acquired, err)
		}
		exp.metric = metric
		acquired = append(acquired, metric.Shutdown)
	}
	if cfg.Logs.Enabled && logs.enabled() {
		log, err := newLogExporter(ctx, logs)
		if err != nil {
			return sectionExporters{}, rollbackProviders(ctx, acquired, err)
		}
		exp.log = log
	}
	return exp, nil
}

// newRuntime builds the providers of every enabled section around the
// already constructed exporters. It cannot fail and publishes nothing.
func newRuntime(cfg Config, res *resource.Resource, propagators propagation.TextMapPropagator, exp sectionExporters) *Runtime {
	var shutdowns []func(context.Context) error
	rt := &Runtime{
		Propagators: propagators,
		Shutdown: func(ctx context.Context) error {
			return shutdownProviders(ctx, shutdowns)
		},
	}
	if cfg.Traces.Enabled {
		tp := newTraceProvider(cfg.Traces, res, exp.span)
		rt.TracerProvider = tp
		shutdowns = append(shutdowns, tp.Shutdown)
	}
	if cfg.Metrics.Enabled {
		mp := newMeterProvider(cfg.Metrics, res, exp.metric)
		rt.MeterProvider = mp
		shutdowns = append(shutdowns, mp.Shutdown)
	}
	if cfg.Logs.Enabled {
		lp := newLoggerProvider(cfg.Logs, res, exp.log)
		rt.LoggerProvider = lp
		shutdowns = append(shutdowns, lp.Shutdown)
	}
	return rt
}

// newTraceProvider cannot fail: New constructs the exporter beforehand.
func newTraceProvider(cfg TraceConfig, res *resource.Resource, exporter sdktrace.SpanExporter) *sdktrace.TracerProvider {
	opts := []sdktrace.TracerProviderOption{
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.ParentBased(sdktrace.TraceIDRatioBased(validSampleRatio(cfg.SampleRatio)))),
	}
	if exporter != nil {
		opts = append(opts, sdktrace.WithBatcher(exporter, cfg.BatchOptions...))
	}
	for _, processor := range cfg.SpanProcessors {
		opts = append(opts, sdktrace.WithSpanProcessor(processor))
	}
	return sdktrace.NewTracerProvider(opts...)
}

// newMeterProvider cannot fail: New constructs the exporter beforehand.
func newMeterProvider(cfg MetricConfig, res *resource.Resource, exporter sdkmetric.Exporter) *sdkmetric.MeterProvider {
	opts := []sdkmetric.Option{
		sdkmetric.WithResource(res),
	}
	if exporter != nil {
		opts = append(opts, sdkmetric.WithReader(sdkmetric.NewPeriodicReader(exporter, cfg.ReaderOptions...)))
	}
	for _, reader := range cfg.Readers {
		opts = append(opts, sdkmetric.WithReader(reader))
	}
	return sdkmetric.NewMeterProvider(opts...)
}

// newLoggerProvider cannot fail: New constructs the exporter beforehand.
func newLoggerProvider(cfg LogConfig, res *resource.Resource, exporter sdklog.Exporter) *sdklog.LoggerProvider {
	opts := []sdklog.LoggerProviderOption{
		sdklog.WithResource(res),
	}
	if exporter != nil {
		opts = append(opts, sdklog.WithProcessor(sdklog.NewBatchProcessor(exporter, cfg.BatchOptions...)))
	}
	for _, processor := range cfg.Processors {
		opts = append(opts, sdklog.WithProcessor(processor))
	}
	return sdklog.NewLoggerProvider(opts...)
}

func otlpOptions[T any](
	cfg otlpConfig,
	withEndpoint func(string) T,
	withEndpointURL func(string) T,
	withInsecure func() T,
	withHeaders func(map[string]string) T,
) []T {
	opts := make([]T, 0, 3)
	if cfg.endpoint != "" {
		opts = append(opts, withEndpoint(cfg.endpoint))
	}
	if cfg.endpointURL != "" {
		opts = append(opts, withEndpointURL(cfg.endpointURL))
	}
	if cfg.insecure {
		opts = append(opts, withInsecure())
	}
	if len(cfg.headers) > 0 {
		opts = append(opts, withHeaders(cfg.headers))
	}
	return opts
}

// validEntries rejects nil entries and a pointer entry repeated in the slice,
// which the SDK would register once and shut down twice. Only pointers can
// share one component's state, and comparing them cannot panic, so value
// entries are not checked for repetition.
func validEntries[T any](field string, entries []T) error {
	seen := make(map[any]int, len(entries))
	for i, entry := range entries {
		value := any(entry)
		if value == nil {
			return fmt.Errorf("otel: %s[%d] is nil", field, i)
		}
		rv := reflect.ValueOf(value)
		if rv.Kind() != reflect.Pointer {
			continue
		}
		if rv.IsNil() {
			return fmt.Errorf("otel: %s[%d] is nil", field, i)
		}
		if first, ok := seen[value]; ok {
			return fmt.Errorf("otel: %s[%d] repeats %s[%d]", field, i, field, first)
		}
		seen[value] = i
	}
	return nil
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

func (cfg TraceConfig) otlp() otlpConfig {
	return otlpConfig{
		signal:      "traces",
		protocol:    cfg.Protocol,
		endpoint:    cfg.Endpoint,
		endpointURL: cfg.EndpointURL,
		insecure:    cfg.Insecure,
		headers:     cfg.Headers,
	}
}

func (cfg MetricConfig) otlp() otlpConfig {
	return otlpConfig{
		signal:      "metrics",
		protocol:    cfg.Protocol,
		endpoint:    cfg.Endpoint,
		endpointURL: cfg.EndpointURL,
		insecure:    cfg.Insecure,
		headers:     cfg.Headers,
	}
}

func (cfg LogConfig) otlp() otlpConfig {
	return otlpConfig{
		signal:      "logs",
		protocol:    cfg.Protocol,
		endpoint:    cfg.Endpoint,
		endpointURL: cfg.EndpointURL,
		insecure:    cfg.Insecure,
		headers:     cfg.Headers,
	}
}

// enabled reports whether the section exports through OTLP.
func (cfg otlpConfig) enabled() bool {
	return cfg.endpoint != "" || cfg.endpointURL != ""
}

func (cfg otlpConfig) validate() error {
	switch cfg.protocol {
	case "", OTLPProtocolHTTPProtobuf, OTLPProtocolGRPC:
	default:
		return fmt.Errorf("otel: %s Protocol %q is not %q or %q", cfg.signal, cfg.protocol, OTLPProtocolHTTPProtobuf, OTLPProtocolGRPC)
	}
	if strings.Contains(cfg.endpoint, "/") {
		return fmt.Errorf("otel: %s Endpoint must be host:port; use EndpointURL for a URL or path", cfg.signal)
	}
	if cfg.endpointURL == "" {
		return nil
	}
	if cfg.endpoint != "" {
		return fmt.Errorf("otel: %s sets both Endpoint and EndpointURL", cfg.signal)
	}
	if cfg.insecure {
		return fmt.Errorf("otel: %s EndpointURL selects TLS by scheme; do not set Insecure", cfg.signal)
	}
	u, err := url.Parse(cfg.endpointURL)
	if err != nil {
		return fmt.Errorf("otel: %s EndpointURL: %w", cfg.signal, err)
	}
	switch {
	case u.Scheme != "http" && u.Scheme != "https":
		return fmt.Errorf("otel: %s EndpointURL scheme must be http or https", cfg.signal)
	case u.Host == "":
		return fmt.Errorf("otel: %s EndpointURL requires a host", cfg.signal)
	case u.User != nil || u.RawQuery != "" || u.Fragment != "" || u.ForceQuery:
		return fmt.Errorf("otel: %s EndpointURL must not contain user info, a query or a fragment", cfg.signal)
	case cfg.protocol == OTLPProtocolGRPC && u.Path != "" && u.Path != "/":
		return fmt.Errorf("otel: %s gRPC EndpointURL must not have a path", cfg.signal)
	}
	return nil
}
