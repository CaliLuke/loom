# OpenTelemetry V2

## Goal

Add a second-phase OpenTelemetry capability that can replace the repeated
observability glue currently carried in downstream Goa services.

This phase expands `goa-light` from transport-only wrappers into a framework
observability surface that covers:

- provider bootstrap
- HTTP and gRPC transport policy
- request-scoped transport enrichment hooks
- transport metric-mode selection
- optional OTLP log bootstrap and logrus bridging

The framework continues to rely on official OpenTelemetry libraries instead of
custom tracing, metrics, or logging implementations.

## Replacement Target

This capability is considered successful when it can replace the generic
observability seams that a consumer like Auto-K currently owns itself:

- trace provider bootstrap
- metric provider bootstrap
- OTLP log bridge bootstrap
- direct `otelhttp.NewMiddleware(...)` wiring
- transport-level span enrichment middleware
- transport-level RED metrics middleware

This phase does not attempt to replace:

- application-specific slog sinks
- domain-specific app metrics
- environment-variable parsing

## Staging

1. Write this plan doc and mark it as the first current OTel step.
2. Build the observability harness first.
3. Encode the replacement contract as failing/target tests.
4. Implement the new package and adapters against that harness.
5. Update user-facing Goa guidance once the harness is green.

## Public API Direction

Add:

- `goa.design/goa/v3/observability/otel`
- `goa.design/goa/v3/observability/otel/logrusbridge`

Keep:

- `goa.design/goa/v3/http/middleware/otel`
- `goa.design/goa/v3/grpc/middleware/otel`

The existing transport wrappers remain the low-level escape hatch. The new root
package becomes the preferred path when users want framework-owned bootstrap and
transport observability.

## Root Runtime Contract

The new root package should expose:

- a `Config` describing service identity plus trace, metric, and log provider
  configuration
- a `Runtime` holding the initialized providers and propagators
- `New(ctx, cfg)` to initialize enabled providers independently and return a
  single shutdown function

The framework should:

- support local-only operation when no endpoint is configured
- set globals only for enabled providers
- close initialized providers in reverse order

## Transport Contract

The new package should expose:

- HTTP middleware and HTTP client wrapping
- gRPC server and client options
- request-scoped attribute hooks
- transport metric-mode selection

HTTP metric mode must support:

- `otel_only`
- `custom_only`
- `both`
- `none`

`custom_only` is the critical mode because it must let a consumer suppress
`otelhttp` metrics while still using framework-owned tracing and custom RED
metrics hooks.

## Harness-First Testing

Build the harness before the feature code in:

- `observability/otel/internal/testkit`

The harness must provide:

- in-memory trace capture
- in-memory metric capture
- in-memory log capture
- Goa HTTP mux fixtures
- gRPC bufconn fixtures
- synthetic auth/project enrichment helpers
- optional custom HTTP RED metrics recorder

## Required Harness Scenarios

1. Bootstrap lifecycle
2. HTTP route-aware tracing
3. HTTP enrichment replacement
4. HTTP metrics mode matrix
5. HTTP client propagation
6. gRPC stats-handler wiring
7. OTLP log bootstrap plus logrus bridge
8. Auto-K-style replacement integration

## Finish Standard

This phase is done only when:

- the harness is green
- the framework package can replace the generic Auto-K observability setup
- the Goa skill reflects the new preferred observability workflow

## Status

Completed on 2026-03-19:

- the plan doc landed first and drove the implementation sequence
- the harness was built before the feature package
- `goa.design/goa/v3/observability/otel` now owns bootstrap plus transport
  policy
- `goa.design/goa/v3/observability/otel/logrusbridge` now provides the
  optional logrus bridge
