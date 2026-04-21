# ticktock fixture — HTTP integration tests

## What this fixture proves

This fixture is the HTTP transport regression surface for server-streaming
endpoints. Codegen must produce a compiling app, and the app must deliver
events correctly over the HTTP stream.

Covered:

- HTTP POST → Server-Sent Events response stream.
- SSE event framing: `event:` / `data:` / `id:` emission, flush semantics.
- Streaming endpoint wiring: generated server struct, handler registration,
  upgrade path, stream interface (`Send`, `SendWithContext`, `Close`).
- Client-side streaming receiver: event parsing, Recv loop, graceful close.
- Authorization logic running before the SSE upgrade (once-semantics).
- Generated app compiles against the repo-local or pinned-remote Loom
  framework, depending on `make loom-local` / `make loom-remote` mode.

## What this fixture does NOT prove

- The raw `events/stream` GET listener contract (non-POST-initiated SSE).
  That is a separate branch with its own framing; test it with a dedicated
  fixture if touched.
- Pre-stream endpoint failures (e.g., authorization failing before any SSE
  frame is written). The happy path here does not exercise the error-before-
  first-event shape — adversarial coverage must be added for changes that
  affect that branch.
- Event-type compatibility for protocol-level errors. Frame structure is
  exercised, but the error-event negotiation surface is not.
- Compile-after-generation of the emitted fixture app in CI with a pinned
  remote pin on unpushed changes — verify with
  `make loom-local` for unpushed work.
- WebSocket transport. WebSocket lives behind a separate fixture.
- JSON-RPC streaming. See `jsonrpc/integration_tests/fixtures/ticktock`.
- Connection timing / timeout / retry semantics beyond what the fixture
  happens to exercise in its golden path.

## When to update

Update this fixture (and this SCOPE.md) when:

- Adding or changing server-streaming HTTP codegen output.
- Introducing a new SSE emission concern (new header, new framing rule).
- Changing the upgrade-once semantics for HTTP streams.
- Changing the stream interface shape emitted by codegen.

Do NOT update this fixture for:

- Non-HTTP transport changes.
- Unary (non-streaming) endpoint changes — use a different fixture.
- Framework-level changes that don't flow through codegen output.

## Regenerating

```bash
# Regenerate with the repo-local Loom framework (use during iteration):
make loom-local
cd http/integration_tests/fixtures/ticktock
loom gen github.com/CaliLuke/loom/http/integration_tests/fixtures/ticktock/design

# Regenerate with a pinned remote commit (parity with CI):
make loom-remote
# ...then re-run loom gen.
```

`server-*.log` files are byproducts of local test runs and should be cleaned
up periodically; they are not part of the fixture contract.
