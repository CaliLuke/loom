# Generated Transport Observability Plan

Add a dependency-free `github.com/CaliLuke/loom/observability/transport` observer contract, then emit safe, classified events from generated HTTP, JSON-RPC, and Loom-MCP server code at decode, dispatch, handler, auth/session, panic, and stream write boundaries. Enablement is context-based in the first pass so generated constructor signatures stay source-compatible; generated code never emits raw bodies, JSON-RPC params, tool arguments, credentials, or result payloads.

## Milestones

### Milestone 1: Observer Contract

Goal: Provide the shared observer package used by generated code without changing generated output.

Acceptance Criteria

- `loom/observability/transport` defines `Observer`, `ObserverFunc`, `Event`, `EventKind`, `Reason`, `TransportKind`, `WithObserver`, `ObserverFromContext`, `Observe`, and an HTTP middleware that injects an observer into request contexts.
- `loom/observability/transport/transport_test.go` proves nil observers are no-op and context-injected observers receive events.
- `go test ./observability/transport` passes from `/Users/luca/code/loom-mono/loom`.

Checklist

- [ ] Add `loom/observability/transport/transport.go` with context helpers, no-op-safe delivery, and `ObserverFunc`.
- [ ] Add `loom/observability/transport/events.go` with event fields for service, method, route, HTTP method, status code, duration, bytes written, JSON-RPC method, JSON-RPC ID, batch count, notification flag, session ID, request ID, trace ID, error class, safe message, and redacted string attributes.
- [ ] Add reason constants for `ok`, `request_decode_failed`, `invalid_jsonrpc_envelope`, `invalid_jsonrpc_batch`, `invalid_jsonrpc_method`, `invalid_jsonrpc_params`, `unsupported_method`, `missing_credentials`, `invalid_credentials`, `permission_rejected`, `principal_mismatch`, `handler_error`, `panic`, `response_write_failed`, `stream_write_failed`, `stream_flush_failed`, `mcp_session_missing`, `mcp_session_not_found`, `mcp_session_principal_mismatch`, and `mcp_events_stream_write_failed`.
- [ ] Add `loom/observability/transport/http.go` with `HTTPMiddleware(observer Observer) func(http.Handler) http.Handler`.
- [ ] Add `loom/observability/transport/transport_test.go` covering nil observer behavior, `ObserverFunc`, context injection, and HTTP middleware injection.
- [ ] Run `go test ./observability/transport` from `/Users/luca/code/loom-mono/loom`.

### Milestone 2: HTTP Codegen Events

Goal: Emit generic HTTP generated-transport events at boundaries outer middleware cannot classify.

Acceptance Criteria

- Generated HTTP server code from `loom/http/codegen/server_sections.go` and SSE server code from `loom/http/codegen/sse.go` emits request start, decode failure, handler error, response write failure, request finish, panic, stream write failure, and stream flush failure events through `transport.Observe`.
- Existing generated HTTP constructor signatures in `loom/http/codegen/server_sections.go` remain source-compatible; this is checked by unchanged `New...Server` function parameters in `loom/http/codegen/server_init_test.go` expectations.
- `go test ./observability/transport ./http/codegen -run 'TestCaptureResponseWriter|TestServer|TestHandler|TestSSE'` passes from `/Users/luca/code/loom-mono/loom`.

Checklist

- [ ] Extend `loom/http/codegen/server_sections.go` generated handlers to create a start timestamp, capture response status and bytes, emit `EventRequestStart`, and emit one terminal event before return.
- [ ] Add `loom/observability/transport/response_writer.go` with `CaptureResponseWriter`, `StatusCode`, and `BytesWritten`, and cover it in `loom/observability/transport/response_writer_test.go`.
- [ ] Emit `ReasonRequestDecodeFailed` in generated request decoder error branches owned by `loom/http/codegen/template_sources_request_decoder.go` and multipart decoder branches owned by `loom/http/codegen/service_data_multipart.go`.
- [ ] Emit `ReasonHandlerError` when endpoint invocation returns an error before the generated error encoder writes a response.
- [ ] Emit `ReasonResponseWriteFailed` when generated response encoding returns an error.
- [ ] Emit `ReasonPanic` in generated handler defer logic and re-panic after observation.
- [ ] Emit `ReasonStreamWriteFailed` and `ReasonStreamFlushFailed` from generated SSE send/open paths in `loom/http/codegen/sse.go`.
- [ ] Extend `loom/http/codegen/server_handler_test.go`, `loom/http/codegen/server_decode_test.go`, `loom/http/codegen/server_encode_test.go`, and `loom/http/codegen/sse_server_test.go` with observer assertions.
- [ ] Run `go test ./observability/transport ./http/codegen -run 'TestCaptureResponseWriter|TestServer|TestHandler|TestSSE'` from `/Users/luca/code/loom-mono/loom`.

### Milestone 3: JSON-RPC Codegen Events

Goal: Add JSON-RPC-specific event fields and reason codes on the shared observer contract.

Acceptance Criteria

- Generated JSON-RPC handlers emit stable events for invalid envelope parse, invalid batch, unsupported method, invalid params, handler error, panic, response write failure, SSE write failure, and WebSocket write failure.
- JSON-RPC events populated after request decode include `TransportJSONRPC`, service name, JSON-RPC method, safe JSON-RPC ID string, batch count, and notification flag from `jsonrpc.RawRequest`; pre-decode rejection events leave JSON-RPC request fields empty.
- `go test ./jsonrpc/codegen ./jsonrpc -run 'Test.*JSONRPC|Test.*SSE|TestNewStreamConfig|TestJSONRPCResponseHelpersAndRawRequest'` passes from `/Users/luca/code/loom-mono/loom`.

Checklist

- [ ] Add observer emission to JSON-RPC request parsing and batch dispatch in `loom/jsonrpc/codegen/handler_request_sections.go`.
- [ ] Add observer emission to method dispatch, params decode, endpoint invocation, response capture, and response write branches in `loom/jsonrpc/codegen/handler_sections.go`.
- [ ] Add observer emission to SSE routing and SSE write branches in `loom/jsonrpc/codegen/handler_sections_sse.go` and `loom/jsonrpc/codegen/sse.go`.
- [ ] Add observer emission to WebSocket upgrade, request read, dispatch, and write branches in `loom/jsonrpc/codegen/stream_sections_websocket.go` and `loom/jsonrpc/codegen/stream_sections_websocket_request.go`.
- [ ] Preserve `jsonrpc.StreamConfig.ErrorHandler` in `loom/jsonrpc/websocket_config.go` as the client/WebSocket stream error callback and do not replace it with the transport observer.
- [ ] Extend `loom/jsonrpc/codegen/handler_sections_test.go`, `loom/jsonrpc/codegen/sse_test.go`, `loom/jsonrpc/codegen/sse_integration_test.go`, and `loom/jsonrpc/types_config_test.go` with observer assertions and the unchanged stream config contract.
- [ ] Run `go test ./jsonrpc/codegen ./jsonrpc -run 'Test.*JSONRPC|Test.*SSE|TestNewStreamConfig|TestJSONRPCResponseHelpersAndRawRequest'` from `/Users/luca/code/loom-mono/loom`.

### Milestone 4: Loom-MCP Bridge

Goal: Forward generated Loom-MCP streamable-HTTP and events-stream boundaries through the same Loom observer contract.

Acceptance Criteria

- Generated SDK server code from `loom-mcp/codegen/mcp/sdk_server_file.go` emits `TransportMCP` events for streamable HTTP request lifecycle, missing session, session not found, principal mismatch, tool handler error, events-stream open, events-stream write failure, and events-stream close.
- Current `adapter.log` calls in generated MCP SDK server output remain present, so existing adapter-local logging behavior is unchanged.
- `go test ./codegen/mcp ./integration_tests/framework -run 'Test.*MCP|Test.*SDK|Test.*Transport'` passes from `/Users/luca/code/loom-mono/loom-mcp`.

Checklist

- [ ] Import `github.com/CaliLuke/loom/observability/transport` in generated MCP SDK server files emitted by `loom-mcp/codegen/mcp/sdk_server_file.go`.
- [ ] Emit `ReasonMCPSessionMissing`, `ReasonMCPSessionNotFound`, and `ReasonMCPSessionPrincipalMismatch` beside the existing generated rejection branches in `serveSDKEventsStream`.
- [ ] Emit `ReasonMCPEventsStreamWriteFailed` beside the existing `writeSDKNotificationEvent` error branch.
- [ ] Emit MCP request lifecycle and handler error events around the generated `base.ServeHTTP(responseObserver, r)` call in `newSDKHandler`.
- [ ] Extend `loom-mcp/codegen/mcp/contract_test.go`, `loom-mcp/codegen/mcp/golden_multi_service_test.go`, and `loom-mcp/integration_tests/framework/runner_fixture_prep.go` backed tests to assert generated observer imports and event paths.
- [ ] Run `go test ./codegen/mcp ./integration_tests/framework -run 'Test.*MCP|Test.*SDK|Test.*Transport'` from `/Users/luca/code/loom-mono/loom-mcp`.

### Milestone 5: Documentation And Gates

Goal: Document the opt-in observer contract and close the work with repo-specific quality gates.

Acceptance Criteria

- `loom/observability/transport/doc.go` documents context-based enablement, redaction rules, stable reason codes, and the boundary with `loom/observability/otel`.
- `loom/roadmap/ROADMAP.md` links this plan and states that generated transport observability is implemented without raw payload emission.
- `./check.sh` passes from `/Users/luca/code/loom-mono/loom`, and `make test` plus `make verify-mcp-local` pass from `/Users/luca/code/loom-mono/loom-mcp`.

Checklist

- [ ] Add package documentation in `loom/observability/transport/doc.go` with a minimal observer example and an HTTP middleware injection example.
- [ ] Add a roadmap link to `loom/roadmap/ROADMAP.md`.
- [ ] Run `./check.sh` from `/Users/luca/code/loom-mono/loom`.
- [ ] Run `make test` from `/Users/luca/code/loom-mono/loom-mcp`.
- [ ] Run `make verify-mcp-local` from `/Users/luca/code/loom-mono/loom-mcp`.
