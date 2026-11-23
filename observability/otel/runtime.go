package otel

import (
	"context"
	"errors"
	"slices"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
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
		// New publishes it as the global propagator. When nil, New uses W3C
		// TraceContext and Baggage.
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
		// Protocol selects the OTLP transport. Empty selects http/protobuf.
		Protocol OTLPProtocol
		// Endpoint is the OTLP collector host:port, without a scheme or path. It
		// enables the OTLP exporter.
		// HTTP exports go to the default /v1/traces path unless
		// OTEL_EXPORTER_OTLP_* environment variables set a path. Set at most one
		// of Endpoint and EndpointURL.
		Endpoint string
		// EndpointURL is the complete OTLP endpoint URL with an http or https
		// scheme. It enables the OTLP exporter. For http/protobuf it is the full
		// signal URL, path included, as in OTEL_EXPORTER_OTLP_TRACES_ENDPOINT; a
		// URL without a path posts to the root path. For grpc it must not have a
		// path. The scheme selects TLS, so Insecure must not be set with it.
		EndpointURL string
		// Insecure disables TLS for an Endpoint export. When false, an
		// OTEL_EXPORTER_OTLP_* environment variable may still select TLS.
		Insecure bool
		// Headers are sent on OTLP export requests.
		Headers map[string]string
		// SampleRatio controls TraceID-based sampling of root spans.
		// Valid local and remote parents retain their sampling decision.
		// Values <= 0 or > 1 use the default ratio of 1.
		SampleRatio float64
		// BatchOptions tune the batch span processor of the OTLP exporter. They
		// require Endpoint or EndpointURL.
		BatchOptions []sdktrace.BatchSpanProcessorOption
		// SpanProcessors are registered after the OTLP exporter, or alone when
		// no endpoint is set. They are registered only when the
		// section is enabled. A successful New with the section enabled
		// transfers ownership: Runtime.Shutdown shuts them down. A failed New,
		// or a disabled section, leaves them owned by the caller. Each entry
		// must be non-nil and appear once.
		SpanProcessors []sdktrace.SpanProcessor
	}

	// MetricConfig configures metric provider bootstrap.
	MetricConfig struct {
		// Enabled enables metric provider bootstrap.
		Enabled bool
		// Protocol selects the OTLP transport. Empty selects http/protobuf.
		Protocol OTLPProtocol
		// Endpoint is the OTLP collector host:port, without a scheme or path. It
		// enables the OTLP exporter.
		// HTTP exports go to the default /v1/metrics path unless
		// OTEL_EXPORTER_OTLP_* environment variables set a path. Set at most one
		// of Endpoint and EndpointURL.
		Endpoint string
		// EndpointURL is the complete OTLP endpoint URL, with the same rules as
		// TraceConfig.EndpointURL.
		EndpointURL string
		// Insecure disables TLS for an Endpoint export. When false, an
		// OTEL_EXPORTER_OTLP_* environment variable may still select TLS.
		Insecure bool
		// Headers are sent on OTLP export requests.
		Headers map[string]string
		// ReaderOptions tune the periodic reader of the OTLP exporter, such as
		// sdkmetric.WithInterval. They require Endpoint or EndpointURL.
		ReaderOptions []sdkmetric.PeriodicReaderOption
		// Readers are registered after the OTLP reader, or alone when no
		// endpoint is set. They are registered only when the
		// section is enabled. A successful New with the section enabled
		// transfers ownership: Runtime.Shutdown shuts them down. A failed New,
		// or a disabled section, leaves them owned by the caller. Each entry
		// must be non-nil and appear once.
		Readers []sdkmetric.Reader
	}

	// LogConfig configures log provider bootstrap.
	LogConfig struct {
		// Enabled enables log provider bootstrap.
		Enabled bool
		// Protocol selects the OTLP transport. Empty selects http/protobuf.
		Protocol OTLPProtocol
		// Endpoint is the OTLP collector host:port, without a scheme or path. It
		// enables the OTLP exporter.
		// HTTP exports go to the default /v1/logs path unless
		// OTEL_EXPORTER_OTLP_* environment variables set a path. Set at most one
		// of Endpoint and EndpointURL.
		Endpoint string
		// EndpointURL is the complete OTLP endpoint URL, with the same rules as
		// TraceConfig.EndpointURL.
		EndpointURL string
		// Insecure disables TLS for an Endpoint export. When false, an
		// OTEL_EXPORTER_OTLP_* environment variable may still select TLS.
		Insecure bool
		// Headers are sent on OTLP export requests.
		Headers map[string]string
		// BatchOptions tune the batch processor of the OTLP exporter. They
		// require Endpoint or EndpointURL.
		BatchOptions []sdklog.BatchProcessorOption
		// Processors are registered after the OTLP exporter, or alone when no
		// endpoint is set. They are registered only when the
		// section is enabled. A successful New with the section enabled
		// transfers ownership: Runtime.Shutdown shuts them down. A failed New,
		// or a disabled section, leaves them owned by the caller. Each entry
		// must be non-nil and appear once.
		Processors []sdklog.Processor
	}

	// OTLPProtocol selects the OTLP exporter transport, using the values of
	// OTEL_EXPORTER_OTLP_PROTOCOL.
	OTLPProtocol string

	// sectionExporters holds the OTLP exporters New built before any provider.
	sectionExporters struct {
		span   sdktrace.SpanExporter
		metric sdkmetric.Exporter
		log    sdklog.Exporter
	}

	otlpConfig struct {
		signal      string
		protocol    OTLPProtocol
		endpoint    string
		endpointURL string
		insecure    bool
		headers     map[string]string
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

const (
	// OTLPProtocolHTTPProtobuf exports protobuf over HTTP, the default.
	OTLPProtocolHTTPProtobuf OTLPProtocol = "http/protobuf"
	// OTLPProtocolGRPC exports over gRPC.
	OTLPProtocolGRPC OTLPProtocol = "grpc"
)

const rollbackTimeout = 5 * time.Second

// New initializes the configured OpenTelemetry providers and returns the runtime.
// It publishes enabled providers globally only after all construction succeeds.
// Every successful call also publishes the runtime propagator globally, whether
// or not any provider is enabled, so instrumentation with nil Propagators
// extracts and injects request context. To keep a propagator the application
// registered earlier, pass it as Config.Propagators. On failure it attempts
// reverse-order cleanup with an uncanceled five-second context and returns the
// construction error joined with any cleanup errors.
// Failed construction leaves the existing global providers and propagator intact.
func New(ctx context.Context, cfg Config) (*Runtime, error) {
	res, err := newResource(cfg)
	if err != nil {
		return nil, err
	}

	traces, metrics, logs := cfg.Traces.otlp(), cfg.Metrics.otlp(), cfg.Logs.otlp()
	if err := validateSections(cfg, traces, metrics, logs); err != nil {
		return nil, err
	}

	propagators := cfg.Propagators
	if propagators == nil {
		propagators = propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{},
			propagation.Baggage{},
		)
	}

	exp, err := newExporters(ctx, cfg, traces, metrics, logs)
	if err != nil {
		return nil, err
	}
	rt := newRuntime(cfg, res, propagators, exp)

	otel.SetTextMapPropagator(propagators)
	if rt.TracerProvider != nil {
		otel.SetTracerProvider(rt.TracerProvider)
	}
	if rt.MeterProvider != nil {
		otel.SetMeterProvider(rt.MeterProvider)
	}
	if rt.LoggerProvider != nil {
		otel.SetLoggerProvider(rt.LoggerProvider)
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

func shutdownProviders(ctx context.Context, shutdowns []func(context.Context) error) error {
	var errs []error
	for _, shutdown := range slices.Backward(shutdowns) {
		if err := shutdown(ctx); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func rollbackProviders(ctx context.Context, shutdowns []func(context.Context) error, cause error) error {
	cleanup, cancel := context.WithTimeout(context.WithoutCancel(ctx), rollbackTimeout)
	defer cancel()
	return errors.Join(cause, shutdownProviders(cleanup, shutdowns))
}
