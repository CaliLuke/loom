package otel

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/log"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	collogs "go.opentelemetry.io/proto/otlp/collector/logs/v1"
	colmetrics "go.opentelemetry.io/proto/otlp/collector/metrics/v1"
	coltrace "go.opentelemetry.io/proto/otlp/collector/trace/v1"
	"google.golang.org/grpc"
)

type (
	// otlpReceiver counts OTLP export requests per signal for both
	// transports.
	otlpReceiver struct {
		mu    sync.Mutex
		calls map[string]int
	}

	grpcTraceReceiver struct {
		coltrace.UnimplementedTraceServiceServer
		receiver *otlpReceiver
	}

	grpcMetricReceiver struct {
		colmetrics.UnimplementedMetricsServiceServer
		receiver *otlpReceiver
	}

	grpcLogReceiver struct {
		collogs.UnimplementedLogsServiceServer
		receiver *otlpReceiver
	}
)

func TestNewExportsEverySignalOverEachProtocol(t *testing.T) {
	cases := []struct {
		name string
		run  func(t *testing.T) (Config, *otlpReceiver, map[string]string)
	}{
		{
			name: "http endpoint uses default signal paths",
			run: func(t *testing.T) (Config, *otlpReceiver, map[string]string) {
				receiver, host := newHTTPReceiver(t)
				return Config{
					Traces:  TraceConfig{Enabled: true, Endpoint: host, Insecure: true},
					Metrics: MetricConfig{Enabled: true, Endpoint: host, Insecure: true},
					Logs:    LogConfig{Enabled: true, Endpoint: host, Insecure: true},
				}, receiver, map[string]string{
					"traces": "/v1/traces", "metrics": "/v1/metrics", "logs": "/v1/logs",
				}
			},
		},
		{
			name: "http endpoint url keeps the full path",
			run: func(t *testing.T) (Config, *otlpReceiver, map[string]string) {
				receiver, host := newHTTPReceiver(t)
				base := "http://" + host + "/collector"
				return Config{
					Traces:  TraceConfig{Enabled: true, Protocol: OTLPProtocolHTTPProtobuf, EndpointURL: base + "/traces"},
					Metrics: MetricConfig{Enabled: true, EndpointURL: base + "/metrics"},
					Logs:    LogConfig{Enabled: true, EndpointURL: base + "/logs"},
				}, receiver, map[string]string{
					"traces": "/collector/traces", "metrics": "/collector/metrics", "logs": "/collector/logs",
				}
			},
		},
		{
			name: "grpc endpoint",
			run: func(t *testing.T) (Config, *otlpReceiver, map[string]string) {
				receiver, host := newGRPCReceiver(t)
				return Config{
					Traces:  TraceConfig{Enabled: true, Protocol: OTLPProtocolGRPC, Endpoint: host, Insecure: true},
					Metrics: MetricConfig{Enabled: true, Protocol: OTLPProtocolGRPC, Endpoint: host, Insecure: true},
					Logs:    LogConfig{Enabled: true, Protocol: OTLPProtocolGRPC, Endpoint: host, Insecure: true},
				}, receiver, grpcSignals()
			},
		},
		{
			name: "grpc endpoint url",
			run: func(t *testing.T) (Config, *otlpReceiver, map[string]string) {
				receiver, host := newGRPCReceiver(t)
				url := "http://" + host
				return Config{
					Traces:  TraceConfig{Enabled: true, Protocol: OTLPProtocolGRPC, EndpointURL: url},
					Metrics: MetricConfig{Enabled: true, Protocol: OTLPProtocolGRPC, EndpointURL: url + "/"},
					Logs:    LogConfig{Enabled: true, Protocol: OTLPProtocolGRPC, EndpointURL: url},
				}, receiver, grpcSignals()
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			restoreOTelGlobals(t)
			cfg, receiver, want := tc.run(t)
			cfg.ServiceName = "export"
			rt, err := New(context.Background(), cfg)
			require.NoError(t, err)

			_, span := rt.TracerProvider.Tracer("export").Start(context.Background(), "work")
			span.End()
			counter, err := rt.MeterProvider.Meter("export").Int64Counter("work")
			require.NoError(t, err)
			counter.Add(context.Background(), 1)
			emitLogRecord(rt.LoggerProvider, "work")

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			require.NoError(t, rt.Shutdown(ctx), "shutdown flushes every exporter")
			for signal, key := range want {
				require.Positive(t, receiver.count(key), "%s export to %s", signal, key)
			}
		})
	}
}

// TestNewAppliesExporterTuning sets a 10ms export cadence on each signal and
// requires an export within a window shorter than the SDK default cadence
// (5s for spans, 60s for metrics, 1s for logs), so dropped options fail.
func TestNewAppliesExporterTuning(t *testing.T) {
	const window = 500 * time.Millisecond
	cases := []struct {
		name string
		path string
		cfg  func(host string) Config
		emit func(t *testing.T, rt *Runtime)
	}{
		{
			name: "trace batch options",
			path: "/v1/traces",
			cfg: func(host string) Config {
				return Config{Traces: TraceConfig{
					Enabled:      true,
					Endpoint:     host,
					Insecure:     true,
					BatchOptions: []sdktrace.BatchSpanProcessorOption{sdktrace.WithBatchTimeout(10 * time.Millisecond)},
				}}
			},
			emit: func(_ *testing.T, rt *Runtime) {
				_, span := rt.TracerProvider.Tracer("tuning").Start(context.Background(), "work")
				span.End()
			},
		},
		{
			name: "metric reader options",
			path: "/v1/metrics",
			cfg: func(host string) Config {
				return Config{Metrics: MetricConfig{
					Enabled:       true,
					Endpoint:      host,
					Insecure:      true,
					ReaderOptions: []sdkmetric.PeriodicReaderOption{sdkmetric.WithInterval(10 * time.Millisecond)},
				}}
			},
			emit: func(t *testing.T, rt *Runtime) {
				counter, err := rt.MeterProvider.Meter("tuning").Int64Counter("work")
				require.NoError(t, err)
				counter.Add(context.Background(), 1)
			},
		},
		{
			name: "log batch options",
			path: "/v1/logs",
			cfg: func(host string) Config {
				return Config{Logs: LogConfig{
					Enabled:      true,
					Endpoint:     host,
					Insecure:     true,
					BatchOptions: []sdklog.BatchProcessorOption{sdklog.WithExportInterval(10 * time.Millisecond)},
				}}
			},
			emit: func(_ *testing.T, rt *Runtime) {
				emitLogRecord(rt.LoggerProvider, "work")
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			restoreOTelGlobals(t)
			receiver, host := newHTTPReceiver(t)
			rt, err := New(context.Background(), tc.cfg(host))
			require.NoError(t, err)
			t.Cleanup(func() {
				require.NoError(t, rt.Shutdown(context.Background()))
			})
			tc.emit(t, rt)
			require.Eventually(t, func() bool {
				return receiver.count(tc.path) > 0
			}, window, 5*time.Millisecond, "the tuned cadence exports before shutdown")
		})
	}
}

func newHTTPReceiver(t *testing.T) (*otlpReceiver, string) {
	t.Helper()
	receiver := &otlpReceiver{calls: make(map[string]int)}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receiver.record(r.URL.Path)
		w.Header().Set("Content-Type", "application/x-protobuf")
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(server.Close)
	return receiver, strings.TrimPrefix(server.URL, "http://")
}

func newGRPCReceiver(t *testing.T) (*otlpReceiver, string) {
	t.Helper()
	receiver := &otlpReceiver{calls: make(map[string]int)}
	listener, err := (&net.ListenConfig{}).Listen(t.Context(), "tcp", "127.0.0.1:0")
	require.NoError(t, err)
	server := grpc.NewServer()
	coltrace.RegisterTraceServiceServer(server, grpcTraceReceiver{receiver: receiver})
	colmetrics.RegisterMetricsServiceServer(server, grpcMetricReceiver{receiver: receiver})
	collogs.RegisterLogsServiceServer(server, grpcLogReceiver{receiver: receiver})
	done := make(chan error, 1)
	go func() {
		done <- server.Serve(listener)
	}()
	t.Cleanup(func() {
		server.Stop()
		if err := <-done; err != nil && !errors.Is(err, grpc.ErrServerStopped) {
			t.Errorf("serve otlp receiver: %v", err)
		}
	})
	return receiver, listener.Addr().String()
}

func grpcSignals() map[string]string {
	return map[string]string{"traces": "grpc:traces", "metrics": "grpc:metrics", "logs": "grpc:logs"}
}

func emitLogRecord(provider log.LoggerProvider, body string) {
	var record log.Record
	record.SetBody(attribute.StringValue(body))
	provider.Logger("otel-test").Emit(context.Background(), record)
}

func collectedMetric(t *testing.T, reader sdkmetric.Reader, name string) bool {
	t.Helper()
	var rm metricdata.ResourceMetrics
	require.NoError(t, reader.Collect(context.Background(), &rm))
	for _, scope := range rm.ScopeMetrics {
		for _, metric := range scope.Metrics {
			if metric.Name == name {
				return true
			}
		}
	}
	return false
}

func (r *otlpReceiver) record(key string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.calls[key]++
}

func (r *otlpReceiver) count(key string) int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.calls[key]
}

func (s grpcTraceReceiver) Export(context.Context, *coltrace.ExportTraceServiceRequest) (*coltrace.ExportTraceServiceResponse, error) {
	s.receiver.record("grpc:traces")
	return &coltrace.ExportTraceServiceResponse{}, nil
}

func (s grpcMetricReceiver) Export(context.Context, *colmetrics.ExportMetricsServiceRequest) (*colmetrics.ExportMetricsServiceResponse, error) {
	s.receiver.record("grpc:metrics")
	return &colmetrics.ExportMetricsServiceResponse{}, nil
}

func (s grpcLogReceiver) Export(context.Context, *collogs.ExportLogsServiceRequest) (*collogs.ExportLogsServiceResponse, error) {
	s.receiver.record("grpc:logs")
	return &collogs.ExportLogsServiceResponse{}, nil
}
