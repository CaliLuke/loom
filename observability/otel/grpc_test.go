package otel

import (
	"testing"

	"google.golang.org/grpc"
	health "google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"

	"github.com/CaliLuke/loom/observability/otel/internal/testkit"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"
)

func TestGRPCOptionsEmitClientAndServerSpans(t *testing.T) {
	traceHarness := testkit.NewTraceHarness(t)
	metricHarness := testkit.NewMetricHarness(t)
	healthServer := health.NewServer()
	healthServer.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)
	fixture := testkit.NewGRPCFixture(t,
		[]grpc.ServerOption{
			GRPCServerOption(GRPCConfig{
				TracerProvider: traceHarness.Provider,
				MeterProvider:  metricHarness.Provider,
			}),
		},
		[]grpc.DialOption{
			GRPCClientOption(GRPCConfig{
				TracerProvider: traceHarness.Provider,
				MeterProvider:  metricHarness.Provider,
			}),
		},
		func(server *grpc.Server) {
			healthpb.RegisterHealthServer(server, healthServer)
		},
	)

	client := healthpb.NewHealthClient(fixture.Conn)
	resp, err := client.Check(t.Context(), &healthpb.HealthCheckRequest{})
	require.NoError(t, err)
	require.Equal(t, healthpb.HealthCheckResponse_SERVING, resp.Status)

	// Receiving the response does not wait for the server's deferred stats
	// handler. Drain the server before inspecting its completed spans.
	fixture.Server.GracefulStop()
	ended := traceHarness.Recorder.Ended()
	require.Len(t, ended, 2)
	require.ElementsMatch(t, []trace.SpanKind{trace.SpanKindClient, trace.SpanKindServer},
		[]trace.SpanKind{ended[0].SpanKind(), ended[1].SpanKind()})
}
