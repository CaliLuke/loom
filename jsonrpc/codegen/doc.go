/*
Package codegen generates JSON-RPC 2.0 servers and clients from an evaluated
Loom design. It builds on top of http/codegen because JSON-RPC over HTTP
reuses the HTTP wire machinery: framing, headers, status codes, routing, and
request/response body types.

# Pipeline position

	service.ServicesData              (codegen/service)
	        |
	        v
	httpcodegen.NewServicesData(svc, JSONRPC.HTTPExpr)  -- reused
	        |
	        v
	jsonrpc/codegen adaptation: add JSON-RPC envelope + streaming semantics
	to each endpoint (adaptation.go, stream_sections.go).
	        |
	        v
	Section renderers (server.go, client.go, sse_server.go, websocket_*.go,
	handler_*.go, client_stream_sections*.go)
	        |
	        v
	[]*codegen.File consumed by codegen/generator

The JSON-RPC generator does NOT define its own ServicesData type; it uses
http/codegen's *ServicesData constructed from r.API.JSONRPC.HTTPExpr (which is
derived from the JSON-RPC design at eval time). Adaptation is applied per
endpoint through service builder helpers in this package.

# What this package owns

  - JSON-RPC envelope emission: method name, id, params, result, error
    framing (handler_sections.go, top_level_sections.go, decoder_sections.go).
  - Batch request handling (handler_batch_writer_sections.go).
  - Header/path conventions specific to the JSON-RPC over HTTP binding
    (header.go, paths.go).
  - Server-Sent Events transport for streaming JSON-RPC responses
    (sse.go, sse_server.go, sse_integration_test.go).
  - WebSocket transport for bidirectional JSON-RPC streaming
    (websocket_server.go, websocket_client.go,
    client_stream_sections_websocket.go).
  - Stream section emission: buffered reads, event/frame boundary detection,
    context-aware cancellation (stream_sections.go,
    client_stream_sections.go).
  - JSON-RPC example server + client CLI (example_*.go, client_cli.go).

# Adaptation layer

adaptation.go is the entry point that bridges the HTTP ServicesData produced
by http/codegen into JSON-RPC expectations (e.g., every endpoint has a JSON-RPC
method name, result framing is the JSON-RPC Response object). When adding a
new JSON-RPC feature, start by deciding whether the adaptation belongs here or
in http/codegen's base types — if the feature is transport-neutral (e.g.,
changes to error shape), prefer http/codegen.

# File layout

  - adaptation.go                         — per-endpoint JSON-RPC adaptation.
  - server.go / client.go                 — server/client section renderers.
  - server_types.go / client_types.go     — type-file renderers.
  - handler_*_sections.go                 — request handler section templates.
  - decoder_sections.go                   — JSON-RPC request/response decoders.
  - top_level_sections.go                 — package-level helpers + batch.
  - client_cli.go / example_*.go          — client CLI + example server.
  - paths.go / header.go                  — HTTP binding specifics.
  - sse.go / sse_server.go                — Server-Sent Events server pieces.
  - websocket_server.go / websocket_client.go / client_stream_sections_websocket.go
    — WebSocket server/client pieces.
  - stream_sections.go                    — shared streaming frame helpers.
  - client_stream_sections.go             — client-side stream receiver helpers.
  - testdata                              — fixture specs used by tests.

# Extension points

  - Add a new JSON-RPC method-level feature (e.g., notification-only): extend
    adaptation.go so the flag propagates through MethodData, then teach the
    handler section renderers to special-case it.
  - Add a new streaming transport: create a sibling file next to sse_server.go
    / websocket_server.go following the existing structure; reuse
    stream_sections.go for framing helpers.
  - Add a new error shape: extend decoder_sections.go and the error templates
    in handler_sections.go. Keep JSON-RPC error codes in one place; do NOT
    scatter them across handler files.

# Invariants

  - No direct edits to http/codegen ServicesData at JSON-RPC analysis time —
    pass a JSON-RPC-shaped HTTPExpr in instead (see codegen/generator
    transport.go).
  - Streaming codegen files must not duplicate frame-boundary logic — it lives
    in stream_sections.go and is shared by SSE and WebSocket renderers.
  - SSE regression surface: the checked-in fixture at
    jsonrpc/integration_tests/fixtures/ticktock covers POST-initiated SSE
    only; the raw `events/stream` GET listener is NOT exercised there.
*/
package codegen
