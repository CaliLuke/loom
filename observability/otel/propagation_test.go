package otel

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"google.golang.org/grpc"
	health "google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"

	"github.com/CaliLuke/loom/observability/otel/internal/testkit"
)

type (
	// recordingPropagator writes its name into one carrier field and records
	// every Inject and Extract call so tests can tell which propagator an
	// instrumentation used.
	recordingPropagator struct {
		name      string
		mu        sync.Mutex
		injected  int
		extracted int
	}
)

const recordingPropagatorField = "x-loom-propagator"

func TestNilPropagatorsUseGlobalPropagator(t *testing.T) {
	transports := []struct {
		name string
		run  func(t *testing.T, configured propagation.TextMapPropagator)
	}{
		{name: "http server", run: runHTTPServerPropagation},
		{name: "http client", run: runHTTPClientPropagation},
		{name: "grpc", run: runGRPCPropagation},
	}
	for _, transport := range transports {
		t.Run(transport.name, func(t *testing.T) {
			cases := []struct {
				name     string
				explicit bool
			}{
				{name: "nil uses global"},
				{name: "explicit overrides global", explicit: true},
			}
			for _, tc := range cases {
				t.Run(tc.name, func(t *testing.T) {
					restoreOTelGlobals(t)
					global := &recordingPropagator{name: "global"}
					otel.SetTextMapPropagator(global)
					var configured propagation.TextMapPropagator
					want := global
					if tc.explicit {
						explicit := &recordingPropagator{name: "explicit"}
						configured = explicit
						want = explicit
					}

					transport.run(t, configured)

					require.Positive(t, want.calls(), "selected propagator was not used")
					if tc.explicit {
						require.Zero(t, global.calls(), "explicit propagator must replace the global one")
					}
				})
			}
		})
	}
}

func TestWrapHTTPClientServiceNameNamesSpans(t *testing.T) {
	cases := []struct {
		name        string
		serviceName string
		want        string
	}{
		{name: "service name", serviceName: "github", want: "GET github"},
		{name: "empty keeps otelhttp default", want: "HTTP GET"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			traceHarness := testkit.NewTraceHarness(t)
			ts := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusNoContent)
			}))
			client := WrapHTTPClient(ts.Client(), HTTPClientConfig{
				ServiceName:    tc.serviceName,
				MetricMode:     HTTPMetricModeNone,
				TracerProvider: traceHarness.Provider,
			})

			req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, ts.URL, nil)
			require.NoError(t, err)
			resp, err := client.Do(req)
			require.NoError(t, err)
			require.NoError(t, resp.Body.Close())

			ended := traceHarness.Recorder.Ended()
			require.Len(t, ended, 1)
			require.Equal(t, tc.want, ended[0].Name())
		})
	}
}

func runHTTPServerPropagation(t *testing.T, configured propagation.TextMapPropagator) {
	t.Helper()
	fixture := testkit.NewHTTPFixture(t)
	fixture.Mux.Use(HTTPMiddleware(HTTPConfig{
		ServiceName: "propagation",
		MetricMode:  HTTPMetricModeNone,
		Propagators: configured,
	}))
	fixture.Mux.Handle(http.MethodGet, "/probe", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	resp, err := fixture.Request(t.Context(), http.MethodGet, "/probe", nil)
	require.NoError(t, err)
	require.Equal(t, http.StatusNoContent, resp.StatusCode)
	require.NoError(t, resp.Body.Close())
}

func runHTTPClientPropagation(t *testing.T, configured propagation.TextMapPropagator) {
	t.Helper()
	ts := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	client := WrapHTTPClient(ts.Client(), HTTPClientConfig{
		MetricMode:  HTTPMetricModeNone,
		Propagators: configured,
	})
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, ts.URL, nil)
	require.NoError(t, err)
	resp, err := client.Do(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusNoContent, resp.StatusCode)
	require.NoError(t, resp.Body.Close())
}

func runGRPCPropagation(t *testing.T, configured propagation.TextMapPropagator) {
	t.Helper()
	healthServer := health.NewServer()
	healthServer.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)
	cfg := GRPCConfig{Propagators: configured}
	fixture := testkit.NewGRPCFixture(t,
		[]grpc.ServerOption{GRPCServerOption(cfg)},
		[]grpc.DialOption{GRPCClientOption(cfg)},
		func(server *grpc.Server) {
			healthpb.RegisterHealthServer(server, healthServer)
		},
	)
	resp, err := healthpb.NewHealthClient(fixture.Conn).Check(t.Context(), &healthpb.HealthCheckRequest{})
	require.NoError(t, err)
	require.Equal(t, healthpb.HealthCheckResponse_SERVING, resp.Status)
	fixture.Server.GracefulStop()
}

func (p *recordingPropagator) Inject(_ context.Context, carrier propagation.TextMapCarrier) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.injected++
	carrier.Set(recordingPropagatorField, p.name)
}

func (p *recordingPropagator) Extract(ctx context.Context, _ propagation.TextMapCarrier) context.Context {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.extracted++
	return ctx
}

func (p *recordingPropagator) Fields() []string {
	return []string{recordingPropagatorField}
}

func (p *recordingPropagator) calls() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.injected + p.extracted
}
