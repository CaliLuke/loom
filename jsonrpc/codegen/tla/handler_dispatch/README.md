# JSON-RPC handler dispatch ownership

Configuration can mount routes before, between or after installing middleware.
It must finish before requests begin. Each request must use the complete
configured chain in wrapper order; middleware rejection must prevent endpoint
invocation. No safe concurrent mutation contract is implied.

`HandlerDispatch.tla` explores a mount and two ordered `Use` operations followed
by an accepted or rejected request. `selected` is the chain chosen for dispatch,
not a trace of callbacks after a short circuit. Middleware order is the reverse
of installation order. A promoted embedded-handler method captures the chain
at mount time; an explicit server method retains server identity and resolves
its current chain at request time.

The legacy configuration violates `CurrentChain` when mounting precedes later
middleware installation. The checked configuration satisfies complete-chain
selection, wrapper order and rejection across 30 generated / 24 distinct
states. The model assumes immutable configuration during requests and abstracts
handler behavior to acceptance/rejection. It does not model Go memory races,
CORS, protocol framing or WebSocket/SSE lifecycle.

Implementation correspondence:

- `top_level_sections.go` emits explicit `Server.ServeHTTP`; its `Use` method
  updates the handler chain, while every mount branch binds the server method.
- `server.go` includes that dispatch method for all JSON-RPC transport shapes.
- `TestMiddlewareDispatchGenerated` compiles ordinary, SSE, mixed and WebSocket
  services. It tests each mounted route (including SSE GET), direct calls,
  middleware rejection, wrapper order and context propagation with Mount before,
  between and after two registrations. A terminal handler isolates dispatch
  from framing; an additional ordinary service test exercises the real generated
  constructor, decoder and endpoint to verify rejection before invocation.
- Existing CORS and fixture tests check the surrounding protocol behavior.

Use TLA+ 1.7.4 `tla2tools.jar` (SHA-256
`936a262061c914694dfd669a543be24573c45d5aa0ff20a8b96b23d01e050e88`).
From this directory:

```sh
java -cp "$TLA_JAR" tlc2.TLC -config legacy.cfg -metadir /tmp/loom-handler-legacy HandlerDispatch.tla
java -cp "$TLA_JAR" tlc2.TLC -config checked.cfg -metadir /tmp/loom-handler-checked HandlerDispatch.tla
```

The first run must fail `CurrentChain` (not parsing or tooling); the second must
pass. Keep checker output outside this directory and remove it after review.
