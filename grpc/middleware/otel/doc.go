/*
Package otel provides thin OpenTelemetry helpers for Loom gRPC servers and
clients.

The package intentionally wraps the official OpenTelemetry contrib gRPC
instrumentation rather than implementing tracing itself. Applications remain
responsible for configuring the tracer provider, exporter, and resource
attributes; Loom owns the transport seam so generated and hand-written gRPC
services can share one instrumentation path.

Use ServerOption and ClientOption with grpc.NewServer and grpc.NewClient (or
grpc.Dial):

	server := grpc.NewServer(otel.ServerOption())
	conn, err := grpc.NewClient(target, otel.ClientOption())
*/
package otel
