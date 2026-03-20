/*
Package otel provides framework-owned OpenTelemetry bootstrap and transport
instrumentation helpers.

The package builds on the lower-level HTTP and gRPC OpenTelemetry wrappers in
`goa.design/goa/v3/http/middleware/otel` and
`goa.design/goa/v3/grpc/middleware/otel`. Use this package when a service wants
Goa-owned provider bootstrap, HTTP metric-mode selection, and transport-level
attribute hooks in addition to normal transport instrumentation.
*/
package otel
