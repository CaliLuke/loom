# OpenTelemetry Instrumentation

## Completed

- Add first-class OpenTelemetry transport wrappers in `goa-light`:
  - `goa.design/goa/v3/http/middleware/otel`
  - `goa.design/goa/v3/grpc/middleware/otel`
- Use the official contrib libraries instead of custom tracing code:
  - `go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp`
  - `go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc`
- Keep provider/exporter/resource bootstrap app-owned while moving transport instrumentation into the framework.
- Make the HTTP wrapper route-aware by default so spans use Goa's matched `METHOD /pattern` name when `goahttp.NewMuxer()` has populated `r.Pattern`.

## Contract

- Use the `otel` packages for framework-level transport instrumentation.
- Keep tracer provider, exporter, resource attributes, and sampling setup in application bootstrap.
- Prefer `goahttp.NewMuxer()` plus `goahttpotel.Middleware(...)` for HTTP so span names and route attributes stay stable across path parameters.
- Prefer `goagrpcotel.ServerOption(...)` and `goagrpcotel.ClientOption(...)` for gRPC instead of reviving the legacy trace/X-Ray middleware.
