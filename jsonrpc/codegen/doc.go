/*
Package codegen generates JSON-RPC 2.0 servers and clients from an evaluated
Loom design. It shares HTTP service data and typed generator sections because
JSON-RPC over HTTP uses the same headers, routing, and body types.

# Pipeline

	service.ServicesData (codegen/service)
	        |
	        v
	httpcodegen.NewServicesData(service, JSONRPC.HTTPExpr)
	        |
	        v
	typed shared sections selected by Method.IsJSONRPC plus JSON-RPC-owned
	server, client, SSE, WebSocket, and example sections
	        |
	        v
	[]*codegen.File consumed by codegen/generator

The package does not define a parallel ServicesData model. JSON-RPC behavior
must be selected through evaluated method metadata or explicit generator
configuration; generated source is never parsed or rewritten.

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

# File layout

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

  - Add a new method-level feature by propagating evaluated metadata through
    MethodData and selecting behavior in typed templates or section builders.
  - Add a new streaming transport: create a sibling file next to sse_server.go
    / websocket_server.go following the existing structure; reuse
    stream_sections.go for framing helpers.
  - Add a new error shape: extend decoder_sections.go and the error templates
    in handler_sections.go. Keep JSON-RPC error codes in one place; do NOT
    scatter them across handler files.

# Invariants

  - Do not parse or rewrite rendered source. Extend typed method metadata,
    section builders, or explicit transport configuration instead.
  - Do not edit HTTP ServicesData during JSON-RPC analysis. Pass a
    JSON-RPC-shaped HTTPExpr to the shared data builder.
  - Streaming codegen files must not duplicate frame-boundary logic — it lives
    in stream_sections.go and is shared by SSE and WebSocket renderers.
  - SSE regression surface: the checked-in fixture at
    jsonrpc/integration_tests/fixtures/ticktock covers POST-initiated SSE
    only; the raw `events/stream` GET listener is NOT exercised there.
*/
package codegen
