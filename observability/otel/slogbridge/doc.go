/*
Package slogbridge provides a standard-library [log/slog.Handler] that emits
log records directly to the OpenTelemetry Logs API
(`go.opentelemetry.io/otel/log`).

The bridge preserves trace and span correlation from `context.Context`, maps
`slog.Level` values onto OpenTelemetry severities, preserves structured groups
and attributes, and supports narrow, stdlib-aligned attribute normalization
([Config.ReplaceAttr]) and fail-closed error formatting ([Config.FormatError]).

Application-owned logging policies such as canonical event naming, domain
redaction rules, and multi-sink fan-out remain outside this package. Compose
[NewHandler] with [log/slog.NewMultiHandler] when mirroring logs to both
console output and OpenTelemetry.
*/
package slogbridge
