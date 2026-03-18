# Production: Observability

Use this for Clue, tracing, metrics, logs, and health checks.

## Core Areas

1. Distributed tracing
2. Metrics
3. Logs

## Recommended Stack

Goa recommends Clue on top of OpenTelemetry for observability.

Typical setup includes:

- `clue.NewConfig(...)`
- `clue.ConfigureOpenTelemetry(...)`
- OTLP trace exporter
- OTLP metric exporter
- `log.Context(...)`

## Runtime Patterns

- Start spans around important service operations
- Record structured attributes on spans
- Record counters and latency histograms
- Use structured request-scoped logs
- Expose health endpoints for critical dependencies
