package otel

import (
	"testing"

	"github.com/CaliLuke/loom/v3/observability/otel/internal/testkit"
	"google.golang.org/grpc"
	health "google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"

	"github.com/stretchr/testify/require"
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

	ended := traceHarness.Recorder.Ended()
	require.GreaterOrEqual(t, len(ended), 2)
}
