// Package transport defines the dependency-free observer contract used by
// Loom's generated HTTP, JSON-RPC, and Loom-MCP transports.
//
// # Enablement
//
// Enablement is context-based. Applications attach an [Observer] to the
// request context via [HTTPMiddleware] (or [WithObserver] for non-HTTP
// entry points), and generated code resolves it transparently. Generated
// constructor signatures remain source-compatible with prior Loom
// releases — turning observability on or off is purely a wiring choice at
// the application boundary.
//
// A minimal observer that prints reasons to stderr:
//
//	mw := transport.HTTPMiddleware(transport.ObserverFunc(func(_ context.Context, e transport.Event) {
//	    log.Printf("%s/%s %s reason=%s status=%d", e.Transport, e.Kind, e.Method, e.Reason, e.StatusCode)
//	}))
//	http.ListenAndServe(":8080", mw(generatedServer))
//
// For non-HTTP entry points (e.g. a JSON-RPC consumer that reads frames
// from a Kafka topic), use [WithObserver] directly to inject the observer
// into the request-scoped context before invoking the generated handler.
//
// # Redaction rules
//
// Generated code never emits raw bodies, JSON-RPC params, MCP tool
// arguments, credentials, or result payloads. [Event] only carries
// classified, low-cardinality fields safe for metric labeling and
// structured-log enrichment:
//
//   - Transport (http/jsonrpc/mcp), Kind (start/finish/failure/stream*),
//     and Reason are stable enumerations and may be used as metric labels.
//   - Service, Method, Route, HTTPMethod identify the operation.
//   - StatusCode, BytesWritten, Duration measure the response.
//   - JSONRPCMethod, JSONRPCID, BatchCount, Notification are populated only
//     after the JSON-RPC envelope has been decoded; pre-decode rejection
//     events leave them empty by design.
//   - SafeMessage carries operator-redacted text, never raw user input.
//
// Observer implementations that need to extract more detail (request id,
// trace id, principal) should read from the request context inside
// [Observer.ObserveEvent] instead of expecting it on [Event].
//
// # Boundary with observability/otel
//
// This package handles request-level classification only. OpenTelemetry
// span attributes, propagators, metric mode, and provider bootstrap remain
// in [github.com/CaliLuke/loom/observability/otel]. The two are composable:
// `otel.HTTPMiddleware` traces the request, `transport.HTTPMiddleware`
// classifies it. Stack them in any order; neither package depends on the
// other.
//
// # MCP release dependency
//
// `loom-mcp` consumes this package through its `go.mod` `replace
// github.com/CaliLuke/loom => ../loom` directive during local
// development. A non-local `loom-mcp` release that drops the replace must
// bump `github.com/CaliLuke/loom` in `loom-mcp/go.mod` to a Loom tag that
// contains this package — otherwise generated MCP server code will not
// compile against the public Loom module.
package transport
