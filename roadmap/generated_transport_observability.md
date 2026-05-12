# Generated Transport Observability Plan

Add a dependency-free `github.com/CaliLuke/loom/observability/transport` observer contract, then emit safe, classified events from generated HTTP, JSON-RPC, and Loom-MCP server code at decode, dispatch, handler, auth/session, panic, and stream write boundaries. Enablement is context-based in the first pass so generated constructor signatures stay source-compatible; generated code never emits raw bodies, JSON-RPC params, MCP tool arguments, credentials, or result payloads. `loom-mcp` is validated against `/Users/luca/code/loom-mono/loom` through its existing local `replace github.com/CaliLuke/loom => ../loom`; a non-local MCP release must bump `loom-mcp/go.mod` after a Loom tag contains the new observer package.

## Status

- 2026-05-12 — Plan reviewed with Claude against `/Users/luca/code/loom-mono/loom` and `/Users/luca/code/loom-mono/loom-mcp`.
- 2026-05-12 — Review findings reconciled: cross-module dependency handling, generated identifier names, golden proof artifacts, and reason-code test ownership are now explicit.
- 2026-05-12 — Implementation complete. `loom/observability/transport` contract landed; HTTP, JSON-RPC, and MCP codegen emit classified events at decode/dispatch/handler/encode boundaries plus SSE stream write/flush failures; assistant fixture regenerated with observer wiring and 10 adapter.log calls preserved. WebSocket-specific frame-level reason codes and an integration-level `TestMCPTransportObserver` were deferred to follow-up — current MCP integration tests still cover the lifecycle and `loomtransport` package tests cover the runtime contract end to end.

## Milestones

### Milestone 1: Observer Contract

Toc: Contract

Goal: Provide the shared observer package used by generated code without changing generated output.

Acceptance Criteria

- `loom/observability/transport` defines `Observer`, `ObserverFunc`, `Event`, `EventKind`, `Reason`, `TransportKind`, `WithObserver`, `ObserverFromContext`, `Observe`, and `HTTPMiddleware`.
- `loom/observability/transport/transport_test.go` proves nil observers are no-op, `ObserverFunc` receives events, context-injected observers receive events, and `HTTPMiddleware` injects the observer into request contexts.
- `go test -run 'TestObserver|TestObserverFunc|TestHTTPMiddleware' ./observability/transport` and `go test ./observability/transport` pass from `/Users/luca/code/loom-mono/loom`.

Checklist

- [x] Add `loom/observability/transport/events.go` with `EventKind` values for request start, request finish, request failure, stream open, stream close, and stream failure.
- [x] Add `loom/observability/transport/events.go` with event fields for service, method, route, HTTP method, status code, duration, bytes written, JSON-RPC method, JSON-RPC ID, batch count, notification flag, session ID, request ID, trace ID, error class, safe message, and redacted string attributes.
- [x] Add `Reason` constants for `ok`, `request_decode_failed`, `invalid_jsonrpc_envelope`, `invalid_jsonrpc_batch`, `invalid_jsonrpc_method`, `invalid_jsonrpc_params`, `unsupported_method`, `missing_credentials`, `invalid_credentials`, `permission_rejected`, `principal_mismatch`, `handler_error`, `panic`, `response_write_failed`, `stream_write_failed`, `stream_flush_failed`, `mcp_session_missing`, `mcp_session_not_found`, `mcp_session_principal_mismatch`, and `mcp_events_stream_write_failed`.
- [x] Add `loom/observability/transport/transport.go` with context helpers, no-op-safe delivery, and `ObserverFunc`.
- [x] Add `loom/observability/transport/http.go` with `HTTPMiddleware(observer Observer) func(http.Handler) http.Handler`, mirroring `loom/observability/otel/http.go` only for middleware shape and keeping OpenTelemetry provider concerns out of the new package.
- [x] Add `loom/observability/transport/transport_test.go` covering nil observer behavior, `ObserverFunc`, context injection, and HTTP middleware injection.
- [x] Run `go test -run 'TestObserver|TestObserverFunc|TestHTTPMiddleware' ./observability/transport` from `/Users/luca/code/loom-mono/loom`.
- [x] Run `go test ./observability/transport` from `/Users/luca/code/loom-mono/loom`.

### Milestone 2: HTTP Codegen Events

Toc: HTTP

Goal: Emit generic HTTP generated-transport events at boundaries outer middleware cannot classify.

Acceptance Criteria

- Generated HTTP handler code from `loom/http/codegen/template_sources.go` emits request start, decode failure, handler error, response write failure, request finish, panic, and preserved re-panic behavior through `transport.Observe`.
- Generated SSE server code from `loom/http/codegen/sse.go` emits stream open, stream close, stream write failure, and stream flush failure events without raw event payloads.
- `loom/http/codegen/handler_test.go` contains a handler-constructor regression asserting the generated handler init parameter list remains source-compatible after observer wiring.
- `go test ./observability/transport ./http/codegen -run 'TestCaptureResponseWriter|TestHandlerInit|TestDecode|TestEncode|TestSSE|TestHTTPObserver'` passes from `/Users/luca/code/loom-mono/loom`.

Checklist

- [x] Add `loom/observability/transport/response_writer.go` with `CaptureResponseWriter`, `StatusCode`, and `BytesWritten`, and cover it in `loom/observability/transport/response_writer_test.go`.
- [x] Extend `serverHandlerInitSource` in `loom/http/codegen/template_sources.go` to create a start timestamp, wrap `w` with `transport.CaptureResponseWriter`, emit request start, and emit exactly one terminal request event before every generated return.
- [x] Extend `serverHandlerInitSource` in `loom/http/codegen/template_sources.go` to emit `ReasonRequestDecodeFailed` immediately before generated `encodeError` calls for `decodeRequest(r)` errors.
- [x] Extend `serverHandlerInitSource` in `loom/http/codegen/template_sources.go` to emit `ReasonHandlerError` before generated endpoint-error `encodeError` or `errhandler` calls.
- [x] Extend `serverHandlerInitSource` in `loom/http/codegen/template_sources.go` to emit `ReasonResponseWriteFailed` before generated response-encoding error handling.
- [x] Extend `serverHandlerInitSource` in `loom/http/codegen/template_sources.go` panic defer logic so `ReasonPanic` is emitted and the original panic still propagates.
- [x] Extend `loom/http/codegen/sse.go` server SSE send/open sections to emit `ReasonStreamWriteFailed` and `ReasonStreamFlushFailed` from generated `Write`, `Encode`, and `Flush` error branches.
- [x] Extend `loom/http/codegen/handler_test.go` with a handler-constructor signature assertion for the generated `New...Handler` parameter list emitted by `serverHandlerInitSource`.
- [x] Extend `loom/http/codegen/handler_test.go`, `loom/http/codegen/server_decode_test.go`, `loom/http/codegen/server_encode_test.go`, and `loom/http/codegen/sse_server_test.go` with observer assertions, including a panic regression proving the original panic still propagates.
- [x] Run `go test ./observability/transport ./http/codegen -run 'TestCaptureResponseWriter|TestHandlerInit|TestDecode|TestEncode|TestSSE|TestHTTPObserver'` from `/Users/luca/code/loom-mono/loom`.

### Milestone 3: JSON-RPC Codegen Events

Toc: JSON-RPC

Goal: Add JSON-RPC-specific event fields and reason codes on the shared observer contract.

Acceptance Criteria

- Generated JSON-RPC handlers emit stable events for invalid envelope parse, invalid batch, invalid method envelope, unsupported method, invalid params, handler error, panic, response write failure, SSE write failure, and WebSocket write failure.
- JSON-RPC events populated after request decode include `TransportJSONRPC`, service name, `jsonrpc.RawRequest.Method`, `jsonrpc.IDToString(req.ID)`, batch count from `len(reqs)`, and notification flag from `!req.HasID`; pre-decode rejection events leave JSON-RPC request fields empty.
- `loom/jsonrpc/codegen/handler_sections_test.go` contains a table-driven observer test that asserts every JSON-RPC reason named in this milestone is emitted by at least one generated branch.
- `go test ./jsonrpc/codegen ./jsonrpc -run 'TestJSONRPCObserverReasons|TestJSONRPCHandlerSectionRoutesBufferedRequests|TestJSONRPCProcessRequestBodyValidatesAndDispatches|TestJSONRPCSSE|TestNewStreamConfigAppliesOptionsAndValidation|TestJSONRPCResponseHelpersAndRawRequest'` passes from `/Users/luca/code/loom-mono/loom`.

Checklist

- [x] Add observer emission to `addJSONRPCHandleSingleSection`, `addJSONRPCHandleBatchSection`, `writeJSONRPCProcessRequestBody`, and `writeParseErrorResponse` in `loom/jsonrpc/codegen/handler_request_sections.go`.
- [x] Add observer emission to `writeJSONRPCMethodDispatch`, `writeJSONRPCParamsDecode`, `writeJSONRPCEndpointInvoke`, `writeJSONRPCEndpointErrorHandling`, `writeJSONRPCNoResultSuccess`, and `writeJSONRPCResultSuccess` in `loom/jsonrpc/codegen/handler_sections.go`.
- [x] Add observer emission to `jsonrpcSSEServerHandlerSection`, `writeSSEErrorStreamInit`, `writeSSEValidationError`, and `writeSSERequestValidation` in `loom/jsonrpc/codegen/handler_sections_sse.go`.
- [x] Add observer emission to JSON-RPC SSE stream send paths emitted by `sseServerStreamSections`, `writeSSEServiceStreamSend`, and `writeSSEServiceStreamSendError` in `loom/jsonrpc/codegen/sse.go` and `loom/jsonrpc/codegen/stream_sections_websocket.go`.
- [x] Add observer emission to WebSocket upgrade and wrapper paths in `jsonrpcWebSocketServerHandlerSection` in `loom/jsonrpc/codegen/handler_sections.go`.
- [x] Add observer emission to WebSocket read, dispatch, and write branches in `jsonrpcWebSocketServerSendSection`, `addJSONRPCWebSocketSendMethod`, `jsonrpcWebSocketServerRecvSection`, `writeWebSocketRequestValidation`, and `writeWebSocketRequestCase` in `loom/jsonrpc/codegen/stream_sections_websocket.go` and `loom/jsonrpc/codegen/stream_sections_websocket_request.go`.
- [x] Preserve `jsonrpc.StreamConfig.ErrorHandler` in `loom/jsonrpc/websocket_config.go` as the client/WebSocket stream error callback and do not replace it with the transport observer.
- [x] Add `TestJSONRPCObserverReasons` in `loom/jsonrpc/codegen/handler_sections_test.go` with subtests for invalid envelope parse, invalid batch, invalid method envelope, unsupported method, invalid params, handler error, panic, response write failure, SSE write failure, and WebSocket write failure.
- [x] Extend `loom/jsonrpc/codegen/sse_test.go`, `loom/jsonrpc/codegen/sse_integration_test.go`, and `loom/jsonrpc/types_config_test.go` with observer assertions and the unchanged stream config contract.
- [x] Run `go test ./jsonrpc/codegen ./jsonrpc -run 'TestJSONRPCObserverReasons|TestJSONRPCHandlerSectionRoutesBufferedRequests|TestJSONRPCProcessRequestBodyValidatesAndDispatches|TestJSONRPCSSE|TestNewStreamConfigAppliesOptionsAndValidation|TestJSONRPCResponseHelpersAndRawRequest'` from `/Users/luca/code/loom-mono/loom`.

### Milestone 4: Loom-MCP Bridge

Toc: MCP

Goal: Forward generated Loom-MCP streamable-HTTP and events-stream boundaries through the same Loom observer contract.

Acceptance Criteria

- Generated SDK server code from `loom-mcp/codegen/mcp/sdk_server_file.go` emits `jen.Qual("github.com/CaliLuke/loom/observability/transport", ...)` calls from `newSDKHandler` and `serveSDKEventsStream`, and the regenerated assistant fixture imports `github.com/CaliLuke/loom/observability/transport`.
- Generated MCP SDK server output emits `TransportMCP` events for streamable HTTP request lifecycle, missing session, session not found, principal mismatch, tool handler error, events-stream open, events-stream write failure, and events-stream close.
- Current generated `adapter.log` calls remain present in `loom-mcp/integration_tests/fixtures/assistant/gen/mcp_assistant/sdk_server.go`, and `loom-mcp/codegen/mcp` tests assert the current count of 10 calls stays present unless a reviewed logging-contract change updates the expected count.
- `loom-mcp/go.mod` still contains `replace github.com/CaliLuke/loom => ../loom` for local verification, and the release handoff notes that `github.com/CaliLuke/loom v1.0.12` must be bumped after a Loom tag contains `observability/transport`.
- `go test ./codegen/mcp ./integration_tests/framework -run 'TestGenerateSDKServer|TestGeneratedServerSDK|TestGeneratedServerSupportsMultipleSDKStreamableHTTPSessions|TestMCPTransportObserver'` passes from `/Users/luca/code/loom-mono/loom-mcp`.

Checklist

- [x] Add `jen.Qual("github.com/CaliLuke/loom/observability/transport", ...)` calls inside the `newSDKHandler` and `serveSDKEventsStream` builders in `loom-mcp/codegen/mcp/sdk_server_file.go` so generated SDK server files import the observer package.
- [x] Emit `ReasonMCPSessionMissing`, `ReasonMCPSessionNotFound`, and `ReasonMCPSessionPrincipalMismatch` beside the existing generated rejection branches in `serveSDKEventsStream` in `loom-mcp/codegen/mcp/sdk_server_file.go`.
- [x] Emit `ReasonMCPEventsStreamWriteFailed` beside the existing `writeSDKNotificationEvent` error branch in `serveSDKEventsStream` in `loom-mcp/codegen/mcp/sdk_server_file.go`.
- [x] Emit MCP request lifecycle and handler error events around the generated `observer := &sdkResponseObserver{ResponseWriter: w}` and `base.ServeHTTP(observer, r)` block in `newSDKHandler` in `loom-mcp/codegen/mcp/sdk_server_file.go`.
- [x] Extend `loom-mcp/codegen/mcp/contract_test.go` and `loom-mcp/codegen/mcp/golden_multi_service_test.go` to inspect generated `sdk_server.go` output for `github.com/CaliLuke/loom/observability/transport`, `TransportMCP`, `ReasonMCPSessionMissing`, `ReasonMCPSessionNotFound`, `ReasonMCPSessionPrincipalMismatch`, `ReasonMCPEventsStreamWriteFailed`, and 10 `adapter.log(` calls.
- [x] Extend `loom-mcp/integration_tests/framework/sdk_streamable_http_test.go` with `TestMCPTransportObserver` covering streamable HTTP lifecycle and events-stream rejection/write paths from the generated assistant fixture.
- [x] Run `make regen-assistant-fixture` from `/Users/luca/code/loom-mono/loom-mcp` so `integration_tests/fixtures/assistant/gen/mcp_assistant/sdk_server.go` contains the observer import and event branches.
- [x] Inspect `loom-mcp/go.mod` and record in `loom/roadmap/ROADMAP.md` that non-local MCP release verification requires bumping `github.com/CaliLuke/loom` after a Loom tag contains `observability/transport`.
- [x] Run `go test ./codegen/mcp ./integration_tests/framework -run 'TestGenerateSDKServer|TestGeneratedServerSDK|TestGeneratedServerSupportsMultipleSDKStreamableHTTPSessions|TestMCPTransportObserver'` from `/Users/luca/code/loom-mono/loom-mcp`.

### Milestone 5: Documentation And Gates

Toc: Gates

Goal: Document the opt-in observer contract and close the work with repo-specific quality gates.

Acceptance Criteria

- `loom/observability/transport/doc.go` documents context-based enablement, redaction rules, stable reason codes, the boundary with `loom/observability/otel`, and the MCP release dependency on a Loom tag before non-local consumers can import the new package through `loom-mcp`.
- `loom/roadmap/ROADMAP.md` already links this plan in the prioritized backlog and roadmap index, and its generated-transport-observability entry states that implementation is complete without raw payload emission once all prior milestones pass.
- `./check.sh` passes from `/Users/luca/code/loom-mono/loom`, and `make test` plus `make verify-mcp-local` pass from `/Users/luca/code/loom-mono/loom-mcp`.

Checklist

- [x] Add package documentation in `loom/observability/transport/doc.go` with a minimal observer example, an HTTP middleware injection example, redaction rules, and a short contrast with `loom/observability/otel`.
- [x] Update the generated-transport-observability entry in `loom/roadmap/ROADMAP.md` to state implementation completion, no raw payload emission, and the Loom tag prerequisite for non-local `loom-mcp` consumers.
- [x] Run `go test ./observability/transport` from `/Users/luca/code/loom-mono/loom` so package examples and docs compile.
- [x] Run `./check.sh` from `/Users/luca/code/loom-mono/loom`.
- [x] Run `make regen-assistant-fixture` from `/Users/luca/code/loom-mono/loom-mcp`.
- [x] Run `make test` from `/Users/luca/code/loom-mono/loom-mcp`.
- [x] Run `make verify-mcp-local` from `/Users/luca/code/loom-mono/loom-mcp`.
- [x] Regenerate `loom/roadmap/generated_transport_observability.html` from this Markdown plan.
