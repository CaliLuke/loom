package otel

import (
	"context"
	"net"
	"testing"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	health "google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/test/bufconn"

	"github.com/stretchr/testify/require"
)

func TestServerAndClientOptionsCreateTracingSpans(t *testing.T) {
	t.Parallel()

	recorder := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider()
	tp.RegisterSpanProcessor(recorder)
	t.Cleanup(func() {
		require.NoError(t, tp.Shutdown(context.Background()))
	})

	listener := bufconn.Listen(1024 * 1024)
	server := grpc.NewServer(ServerOption(otelgrpc.WithTracerProvider(tp)))
	healthpb.RegisterHealthServer(server, health.NewServer())
	go func() {
		_ = server.Serve(listener)
	}()
	t.Cleanup(server.Stop)

	conn, err := grpc.NewClient("passthrough:///bufconn",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		ClientOption(otelgrpc.WithTracerProvider(tp)),
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return listener.Dial()
		}),
	)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, conn.Close())
	})

	client := healthpb.NewHealthClient(conn)
	_, err = client.Check(context.Background(), &healthpb.HealthCheckRequest{})
	require.NoError(t, err)

	spans := recorder.Ended()
	require.Len(t, spans, 2)
	require.ElementsMatch(t,
		[]trace.SpanKind{trace.SpanKindClient, trace.SpanKindServer},
		[]trace.SpanKind{spans[0].SpanKind(), spans[1].SpanKind()},
	)
	require.ElementsMatch(t,
		[]string{"grpc.health.v1.Health/Check", "grpc.health.v1.Health/Check"},
		[]string{spans[0].Name(), spans[1].Name()},
	)
}
