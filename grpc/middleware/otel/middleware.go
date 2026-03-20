package otel

import (
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/stats"
)

type (
	// Option configures gRPC OpenTelemetry instrumentation.
	Option = otelgrpc.Option
)

// NewServerHandler creates an OpenTelemetry gRPC server stats handler.
func NewServerHandler(opts ...Option) stats.Handler {
	return otelgrpc.NewServerHandler(opts...)
}

// NewClientHandler creates an OpenTelemetry gRPC client stats handler.
func NewClientHandler(opts ...Option) stats.Handler {
	return otelgrpc.NewClientHandler(opts...)
}

// ServerOption returns a grpc.ServerOption that installs OpenTelemetry stats
// handling on the server.
func ServerOption(opts ...Option) grpc.ServerOption {
	return grpc.StatsHandler(NewServerHandler(opts...))
}

// ClientOption returns a gRPC dial option that installs OpenTelemetry stats
// handling on the client.
func ClientOption(opts ...Option) grpc.DialOption {
	return grpc.WithStatsHandler(NewClientHandler(opts...))
}
