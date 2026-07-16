# ticktock fixture — JSON-RPC integration tests

## What this fixture proves

This fixture is the JSON-RPC transport regression surface for
**POST-initiated** Server-Sent Events streaming. It is not a comprehensive
JSON-RPC streaming fixture — it covers one specific branch of the
JSON-RPC-over-HTTP contract.

Covered:

- JSON-RPC request framing: method name, id, params for the call that
  initiates the stream.
- JSON-RPC-over-SSE response framing: each streamed event is a JSON-RPC
  `Response` (or notification, depending on the method) wrapped in an SSE
  `data:` frame.
- Server codegen wiring: handler registration, request decoding, stream
  initialization, event emission loop.
- Client codegen wiring: request marshaling, SSE event parsing, Recv loop,
  graceful close.
- `SendAndCloseWithContext` + `SendAndClose` interface on the server stream
  (see http/codegen's WebSocket/SSE stream emit — the wrapper is a thin
  forwarder to the with-context variant).
- Temp-copy regeneration rewrites the fixture's Loom replacement to the current
  repository root and proves the regenerated app compiles against the change
  under test.

## What this fixture does NOT prove

- **The raw `events/stream` GET listener contract**. That branch has its own
  framing and initialization and is NOT exercised here; it is covered by
  `tests/sse_get_listener_test.go`, which regenerates a temp events/stream
  variant of this fixture and drives a real GET listener. Extend that test
  when touching the GET-listener code path.
- Pre-stream endpoint failures (authorization failing before any event, body
  parse failure on the initiating POST). The happy path does not cover the
  "error-before-first-event" shape; add adversarial coverage when changing
  that branch.
- Event-type compatibility for protocol-level errors (e.g., JSON-RPC error
  response shapes delivered inside the SSE stream). The fixture exercises
  successful event emission but not the error-as-event negotiation.
- Branch-specific connection timing / retry / backoff semantics beyond what
  the happy path exercises.
- WebSocket transport. WebSocket JSON-RPC is covered separately by
  `websocket_*` codegen and its own fixtures.

## When to update

Update this fixture (and this SCOPE.md) when:

- Changing JSON-RPC SSE event framing.
- Changing JSON-RPC request decoder for POST-initiated streams.
- Changing the `SendAndClose[WithContext]` emit shape.
- Changing stream initialization (pre-emission setup, once-semantics).

Do NOT update this fixture for:

- Raw GET `events/stream` listener changes — extend `tests/sse_get_listener_test.go`.
- WebSocket JSON-RPC changes — use a WebSocket-specific fixture.
- Unary JSON-RPC changes — use a unary fixture.

## Regenerating

```bash
cd jsonrpc/integration_tests/fixtures/ticktock
go run -mod=mod github.com/CaliLuke/loom/cmd/loom gen example.com/ticktock/design -o .
```

This checked-in fixture has an explicit local `replace` directive. The shared
`make loom-local` / `make loom-remote` mode controls dynamically generated
HTTP and JSON-RPC temp modules, not direct regeneration inside this fixture.

`server-*.log`, `loom*` temp directories, and similar byproducts in this
directory are NOT part of the fixture contract; they can be cleaned up at
any time.
