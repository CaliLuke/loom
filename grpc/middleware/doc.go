// Package middleware contains gRPC server and client interceptors that wraps
// unary and streaming RPCs to provide additional functionality.
//
// This package contains the following middleware:
//
//   - Stream Canceler server middleware for canceling streaming requests.
//   - OpenTelemetry server and client middleware in the otel subpackage.
//
// Example to use the server middleware:
//
//	srv := grpc.NewServer(middleware.StreamCanceler())
//
// Example to use the client middleware:
//
//	conn, err := grpc.Dial(host,
//	    otel.GRPCClientOption("service"),
//	)
package middleware
