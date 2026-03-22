# OpenTelemetry Instrumentation

## Completed

- Add first-class OpenTelemetry transport wrappers in `loom`:
  - `github.com/CaliLuke/loom/http/middleware/otel`
  - `github.com/CaliLuke/loom/grpc/middleware/otel`
- Use the official contrib libraries instead of custom tracing code:
  - `go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp`
  - `go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc`
- Keep provider/exporter/resource bootstrap app-owned while moving transport instrumentation into the framework.
- Make the HTTP wrapper route-aware by default so spans use Loom's matched `METHOD /pattern` name when `loomhttp.NewMuxer()` has populated `r.Pattern`.
- Add the higher-level framework-owned observability package:
  - `github.com/CaliLuke/loom/observability/otel`
  - `github.com/CaliLuke/loom/observability/otel/logrusbridge`
- Support framework-owned trace, metric, and OTLP log bootstrap while keeping environment parsing app-owned.
- Add HTTP transport metric modes:
  - `otel_only`
  - `custom_only`
  - `both`
  - `none`
- Add request-scoped HTTP transport attribute collection via `AddHTTPAttributes(...)` so downstream middlewares can enrich the active request span without mutating it directly.
- Add a reusable observability harness in `observability/otel/internal/testkit` covering in-memory traces, metrics, logs, Loom HTTP mux usage, gRPC bufconn transport, and an Auto-K-style compatibility contract.

## Contract

- Prefer `github.com/CaliLuke/loom/observability/otel` when the service wants framework-owned provider bootstrap and transport observability policy.
- Keep environment parsing and domain-specific metrics app-owned.
- Use the lower-level `http/middleware/otel` and `grpc/middleware/otel` packages only when transport-only instrumentation is sufficient.
- Prefer `loomhttp.NewMuxer()` plus `otel.HTTPMiddleware(...)` for HTTP so span names and route attributes stay stable across path parameters.
- Use `otel.AddHTTPAttributes(...)` from downstream HTTP middleware when request-scoped transport attributes need to be attached after the span has started.
- Prefer `otel.GRPCServerOption(...)` and `otel.GRPCClientOption(...)` for gRPC instead of reviving the legacy trace/X-Ray middleware.
