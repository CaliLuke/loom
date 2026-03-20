package testkit

import (
	"context"
	"net"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

type (
	// GRPCFixture wraps a bufconn-backed gRPC server and client connection.
	GRPCFixture struct {
		Listener *bufconn.Listener
		Server   *grpc.Server
		Conn     *grpc.ClientConn
	}
)

// NewGRPCFixture creates a bufconn-backed gRPC server and client connection.
func NewGRPCFixture(tb testing.TB, serverOpts []grpc.ServerOption, dialOpts []grpc.DialOption, register func(*grpc.Server)) *GRPCFixture {
	tb.Helper()
	listener := bufconn.Listen(1024 * 1024)
	server := grpc.NewServer(serverOpts...)
	register(server)
	go func() {
		_ = server.Serve(listener)
	}()
	conn, err := grpc.NewClient("passthrough:///bufconn",
		append([]grpc.DialOption{
			grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
				return listener.DialContext(ctx)
			}),
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		}, dialOpts...)...,
	)
	if err != nil {
		tb.Fatalf("new grpc client: %v", err)
	}
	tb.Cleanup(func() {
		_ = conn.Close()
		server.Stop()
		_ = listener.Close()
	})
	return &GRPCFixture{Listener: listener, Server: server, Conn: conn}
}
