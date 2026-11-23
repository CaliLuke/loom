package otel

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	"github.com/CaliLuke/loom/observability/otel/internal/testkit"
)

type (
	// shutdownCountingProcessor records Shutdown calls so tests can tell who
	// owns a caller-supplied span processor.
	shutdownCountingProcessor struct {
		sdktrace.SpanProcessor
		shutdowns int
	}

	// sliceSpanProcessor holds a slice, so hashing it panics.
	sliceSpanProcessor struct {
		sdktrace.SpanProcessor
		names []string
	}

	// valueSpanProcessor has a comparable type but can hold an unhashable
	// processor at run time.
	valueSpanProcessor struct {
		sdktrace.SpanProcessor
	}
)

func TestOTLPOptions(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		cfg  otlpConfig
		want []string
	}{
		{name: "empty config", cfg: otlpConfig{}},
		{
			name: "endpoint settings",
			cfg:  otlpConfig{endpoint: "localhost:4318", insecure: true, headers: map[string]string{"x-api-key": "secret"}},
			want: []string{"endpoint", "insecure", "headers"},
		},
		{
			name: "endpoint url",
			cfg:  otlpConfig{endpointURL: "https://collector.example/otlp/v1/traces"},
			want: []string{"url"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := otlpOptions(tc.cfg,
				func(string) string { return "endpoint" },
				func(string) string { return "url" },
				func() string { return "insecure" },
				func(map[string]string) string { return "headers" },
			)
			require.ElementsMatch(t, tc.want, got)
		})
	}
}

func TestNewRejectsInvalidOTLPConfig(t *testing.T) {
	sharedReader := sdkmetric.NewManualReader()
	cases := []struct {
		name string
		cfg  Config
		want string
	}{
		{
			name: "unknown protocol",
			cfg:  Config{Traces: TraceConfig{Enabled: true, Protocol: "http/json", Endpoint: "localhost:4318"}},
			want: `traces Protocol "http/json"`,
		},
		{
			name: "endpoint and url",
			cfg:  Config{Metrics: MetricConfig{Enabled: true, Endpoint: "localhost:4318", EndpointURL: "http://localhost:4318/v1/metrics"}},
			want: "metrics sets both Endpoint and EndpointURL",
		},
		{
			name: "url with insecure",
			cfg:  Config{Logs: LogConfig{Enabled: true, EndpointURL: "https://collector/v1/logs", Insecure: true}},
			want: "logs EndpointURL selects TLS",
		},
		{
			name: "unparsable url",
			cfg:  Config{Traces: TraceConfig{Enabled: true, EndpointURL: "http://[::1"}},
			want: "traces EndpointURL",
		},
		{
			name: "url without scheme",
			cfg:  Config{Traces: TraceConfig{Enabled: true, EndpointURL: "collector:4318/v1/traces"}},
			want: "scheme must be http or https",
		},
		{
			name: "url without host",
			cfg:  Config{Traces: TraceConfig{Enabled: true, EndpointURL: "https:///v1/traces"}},
			want: "requires a host",
		},
		{
			name: "url with query",
			cfg:  Config{Traces: TraceConfig{Enabled: true, EndpointURL: "https://collector/v1/traces?tenant=a"}},
			want: "must not contain user info, a query or a fragment",
		},
		{
			name: "url with user info",
			cfg:  Config{Traces: TraceConfig{Enabled: true, EndpointURL: "https://user:pass@collector/v1/traces"}},
			want: "must not contain user info, a query or a fragment",
		},
		{
			name: "endpoint with scheme",
			cfg:  Config{Traces: TraceConfig{Enabled: true, Endpoint: "https://collector:4318"}},
			want: "traces Endpoint must be host:port; use EndpointURL",
		},
		{
			name: "endpoint with path",
			cfg:  Config{Logs: LogConfig{Enabled: true, Endpoint: "collector:4318/otel"}},
			want: "logs Endpoint must be host:port",
		},
		{
			name: "non-http scheme",
			cfg:  Config{Traces: TraceConfig{Enabled: true, EndpointURL: "ftp://collector/v1/traces"}},
			want: "scheme must be http or https",
		},
		{
			name: "url with fragment",
			cfg:  Config{Traces: TraceConfig{Enabled: true, EndpointURL: "https://collector/v1/traces#x"}},
			want: "must not contain user info, a query or a fragment",
		},
		{
			name: "url with empty query",
			cfg:  Config{Traces: TraceConfig{Enabled: true, EndpointURL: "https://collector/v1/traces?"}},
			want: "must not contain user info, a query or a fragment",
		},
		{
			name: "repeated reader",
			cfg:  Config{Metrics: MetricConfig{Enabled: true, Readers: []sdkmetric.Reader{sharedReader, sharedReader}}},
			want: "metrics Readers[1] repeats metrics Readers[0]",
		},
		{
			name: "grpc url with path",
			cfg:  Config{Metrics: MetricConfig{Enabled: true, Protocol: OTLPProtocolGRPC, EndpointURL: "https://collector:4317/otlp"}},
			want: "metrics gRPC EndpointURL must not have a path",
		},
		{
			name: "batch options without endpoint",
			cfg:  Config{Traces: TraceConfig{Enabled: true, BatchOptions: []sdktrace.BatchSpanProcessorOption{sdktrace.WithMaxExportBatchSize(1)}}},
			want: "traces BatchOptions require",
		},
		{
			name: "reader options without endpoint",
			cfg:  Config{Metrics: MetricConfig{Enabled: true, ReaderOptions: []sdkmetric.PeriodicReaderOption{sdkmetric.WithInterval(1)}}},
			want: "metrics ReaderOptions require",
		},
		{
			name: "log batch options without endpoint",
			cfg:  Config{Logs: LogConfig{Enabled: true, BatchOptions: []sdklog.BatchProcessorOption{sdklog.WithExportMaxBatchSize(1)}}},
			want: "logs BatchOptions require",
		},
		{
			name: "typed nil span processor",
			cfg:  Config{Traces: TraceConfig{Enabled: true, SpanProcessors: []sdktrace.SpanProcessor{(*shutdownCountingProcessor)(nil)}}},
			want: "traces SpanProcessors[0] is nil",
		},
		{
			name: "nil span processor",
			cfg:  Config{Traces: TraceConfig{Enabled: true, SpanProcessors: []sdktrace.SpanProcessor{nil}}},
			want: "traces SpanProcessors[0] is nil",
		},
		{
			name: "nil reader",
			cfg:  Config{Metrics: MetricConfig{Enabled: true, Readers: []sdkmetric.Reader{nil}}},
			want: "metrics Readers[0] is nil",
		},
		{
			name: "nil log processor",
			cfg:  Config{Logs: LogConfig{Enabled: true, Processors: []sdklog.Processor{nil}}},
			want: "logs Processors[0] is nil",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			restoreOTelGlobals(t)
			traces, metrics, logs := otel.GetTracerProvider(), otel.GetMeterProvider(), otel.GetLoggerProvider()
			propagator := otel.GetTextMapPropagator()

			rt, err := New(context.Background(), tc.cfg)
			require.ErrorContains(t, err, tc.want)
			require.Nil(t, rt)
			require.Same(t, traces, otel.GetTracerProvider())
			require.Same(t, metrics, otel.GetMeterProvider())
			require.Same(t, logs, otel.GetLoggerProvider())
			require.Equal(t, propagator, otel.GetTextMapPropagator())
		})
	}
}

func TestNewRegistersCustomPipelines(t *testing.T) {
	restoreOTelGlobals(t)
	spans := tracetest.NewSpanRecorder()
	reader := sdkmetric.NewManualReader()
	logs := &testkit.LogExporter{}

	rt, err := New(context.Background(), Config{
		ServiceName: "custom",
		Traces:      TraceConfig{Enabled: true, SpanProcessors: []sdktrace.SpanProcessor{spans}},
		Metrics:     MetricConfig{Enabled: true, Readers: []sdkmetric.Reader{reader}},
		Logs:        LogConfig{Enabled: true, Processors: []sdklog.Processor{sdklog.NewSimpleProcessor(logs)}},
	})
	require.NoError(t, err)

	_, span := rt.TracerProvider.Tracer("custom").Start(context.Background(), "work")
	span.End()
	counter, err := rt.MeterProvider.Meter("custom").Int64Counter("work")
	require.NoError(t, err)
	counter.Add(context.Background(), 1)
	emitLogRecord(rt.LoggerProvider, "work")

	require.Len(t, spans.Ended(), 1)
	require.True(t, collectedMetric(t, reader, "work"))
	require.Len(t, logs.Records(), 1)

	require.NoError(t, rt.Shutdown(context.Background()))
	require.ErrorIs(t, reader.Collect(context.Background(), &metricdata.ResourceMetrics{}), sdkmetric.ErrReaderShutdown,
		"Runtime.Shutdown shuts down supplied readers")
}

func TestFailedNewLeavesSuppliedProcessorsUntouched(t *testing.T) {
	restoreOTelGlobals(t)
	processor := &shutdownCountingProcessor{SpanProcessor: tracetest.NewSpanRecorder()}

	rt, err := New(context.Background(), Config{
		Traces:  TraceConfig{Enabled: true, SpanProcessors: []sdktrace.SpanProcessor{processor}},
		Metrics: MetricConfig{Enabled: true, Endpoint: "a b"},
	})
	require.Error(t, err)
	require.Nil(t, rt)
	require.Zero(t, processor.shutdowns, "a failed New must not shut down caller-owned processors")

	rt, err = New(context.Background(), Config{
		Traces: TraceConfig{Enabled: true, SpanProcessors: []sdktrace.SpanProcessor{processor}},
	})
	require.NoError(t, err)
	require.NoError(t, rt.Shutdown(context.Background()))
	require.Equal(t, 1, processor.shutdowns, "a successful New transfers ownership")
}

func (p *shutdownCountingProcessor) Shutdown(ctx context.Context) error {
	p.shutdowns++
	return p.SpanProcessor.Shutdown(ctx)
}

func TestNewAcceptsValidOTLPEndpoints(t *testing.T) {
	cases := []struct {
		name string
		cfg  Config
	}{
		{name: "ipv6 endpoint", cfg: Config{Traces: TraceConfig{Enabled: true, Endpoint: "[::1]:4318", Insecure: true}}},
		{name: "ipv6 url", cfg: Config{Traces: TraceConfig{Enabled: true, EndpointURL: "http://[::1]:4318/v1/traces"}}},
		{name: "grpc url with root path", cfg: Config{Logs: LogConfig{Enabled: true, Protocol: OTLPProtocolGRPC, EndpointURL: "https://collector:4317/"}}},
		{name: "disabled section is not validated", cfg: Config{Traces: TraceConfig{Endpoint: "https://ignored"}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			restoreOTelGlobals(t)
			rt, err := New(context.Background(), tc.cfg)
			require.NoError(t, err)
			require.NoError(t, rt.Shutdown(context.Background()))
		})
	}
}

func TestNewAcceptsUppercaseURLScheme(t *testing.T) {
	restoreOTelGlobals(t)
	receiver, host := newHTTPReceiver(t)
	rt, err := New(context.Background(), Config{
		Metrics: MetricConfig{Enabled: true, EndpointURL: "HTTP://" + host + "/v1/metrics"},
	})
	require.NoError(t, err)
	require.NoError(t, rt.Shutdown(context.Background()))
	require.Positive(t, receiver.count("/v1/metrics"))
}

func TestNewAcceptsValueTypedComponents(t *testing.T) {
	restoreOTelGlobals(t)
	inner := sliceSpanProcessor{SpanProcessor: tracetest.NewSpanRecorder(), names: []string{"a"}}
	processor := valueSpanProcessor{SpanProcessor: inner}
	rt, err := New(context.Background(), Config{
		Traces: TraceConfig{Enabled: true, SpanProcessors: []sdktrace.SpanProcessor{processor, processor}},
	})
	require.NoError(t, err, "value entries are not checked for repetition and must not panic")
	require.NoError(t, rt.Shutdown(context.Background()))
}
