package otel

import (
	"context"
	"net"
	"sync"
	"testing"
	"time"

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

type spanCompletionRecorder struct {
	*tracetest.SpanRecorder
	published     chan struct{}
	serverRelease <-chan struct{}
}

func TestServerAndClientOptionsCreateTracingSpans(t *testing.T) {
	t.Parallel()
	for _, gated := range []bool{false, true} {
		t.Run(map[bool]string{false: "ordinary", true: "server publication after RPC return"}[gated], func(t *testing.T) {
			testTracingSpans(t, gated)
		})
	}
}

func TestSpanCompletionDeadline(t *testing.T) {
	recorder := &spanCompletionRecorder{SpanRecorder: tracetest.NewSpanRecorder(), published: make(chan struct{}, 1)}
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancel()
	spans, err := recorder.wait(ctx, 2)
	require.ErrorIs(t, err, context.DeadlineExceeded)
	require.Empty(t, spans)
}

func TestSpanCompletionKeepsUnexpectedSpans(t *testing.T) {
	recorder := &spanCompletionRecorder{SpanRecorder: tracetest.NewSpanRecorder(), published: make(chan struct{}, 1)}
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
	t.Cleanup(func() {
		require.NoError(t, tp.Shutdown(context.Background()))
	})
	for range 3 {
		_, span := tp.Tracer("completion-test").Start(context.Background(), "extra")
		span.End()
	}
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	spans, err := recorder.wait(ctx, 2)
	require.NoError(t, err)
	require.Len(t, spans, 3, "completion must not filter extra spans to satisfy the expected count")
}

// OnEnd publishes the span before notifying waiters; the optional server gate
// forces late publication without changing production instrumentation.
func (r *spanCompletionRecorder) OnEnd(span sdktrace.ReadOnlySpan) {
	if r.serverRelease != nil && span.SpanKind() == trace.SpanKindServer {
		<-r.serverRelease
	}
	r.SpanRecorder.OnEnd(span)
	select {
	case r.published <- struct{}{}:
	default:
	}
}

func testTracingSpans(t *testing.T, gated bool) {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	release := make(chan struct{})
	var releaseOnce sync.Once
	releaseServer := func() {
		releaseOnce.Do(func() {
			close(release)
		})
	}
	defer releaseServer()
	recorder := &spanCompletionRecorder{SpanRecorder: tracetest.NewSpanRecorder(), published: make(chan struct{}, 1)}
	if gated {
		recorder.serverRelease = release
	}
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
	t.Cleanup(func() {
		require.NoError(t, tp.Shutdown(context.Background()))
	})

	listener := bufconn.Listen(1024 * 1024)
	server := grpc.NewServer(ServerOption(otelgrpc.WithTracerProvider(tp)))
	healthpb.RegisterHealthServer(server, health.NewServer())
	serveDone := make(chan error, 1)
	go func() {
		serveDone <- server.Serve(listener)
	}()
	t.Cleanup(func() {
		server.Stop()
		select {
		case err := <-serveDone:
			require.NoError(t, err)
		case <-time.After(5 * time.Second):
			t.Error("gRPC Serve did not complete after Stop")
		}
	})

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
	_, err = client.Check(ctx, &healthpb.HealthCheckRequest{})
	require.NoError(t, err)
	if gated {
		require.Len(t, recorder.Ended(), 1, "RPC return does not publish the gated server span")
	}
	releaseServer()
	spans, err := recorder.wait(ctx, 2)
	require.NoError(t, err)
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

func (r *spanCompletionRecorder) wait(ctx context.Context, expected int) ([]sdktrace.ReadOnlySpan, error) {
	for {
		spans := r.Ended()
		if len(spans) >= expected {
			return spans, nil
		}
		select {
		case <-ctx.Done():
			return spans, ctx.Err()
		case <-r.published:
		}
	}
}
