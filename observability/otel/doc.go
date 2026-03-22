/*
Package otel provides framework-owned OpenTelemetry bootstrap and transport
instrumentation helpers.

The package builds on the lower-level HTTP and gRPC OpenTelemetry wrappers in
`github.com/CaliLuke/loom/http/middleware/otel` and
`github.com/CaliLuke/loom/grpc/middleware/otel`. Use this package when a service wants
Loom-owned provider bootstrap, HTTP metric-mode selection, and transport-level
attribute hooks in addition to normal transport instrumentation.
*/
package otel
