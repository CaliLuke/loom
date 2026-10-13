# Stream client fixture alignment — #866

The typed stream contract exposes `Recv()` and `RecvWithContext(ctx)`. Existing
embedded consumers and two snapshots still use the previous generated shape.
This is a test-only repair of demonstrated main failures.

1. Reproduce the five named failures on pushed `7baf82f92` before editing.
2. Inventory client calls in `jsonrpc/codegen/success_result_member_test.go`,
   `union_body_names_test.go`, `viewed_stream_result_test.go` and `jsonrpc/codegen/viewed_stream_markers_test.go`. Retain server connection-stream `Recv(ctx)` calls;
   convert only typed client calls to `RecvWithContext(ctx)`.
3. Regenerate only `render_client-sse-object.golden` and
   `jsonrpc-websocket-client-file.golden` through their owning test update flag.
4. Run the five affected tests, including their generated vet/race checks. Run
   scoped lint and inspect the exact diff. Production/DSL inputs are unchanged,
   so the repair needs no new ownership model or generated-corpus audit.
5. Freeze the atomic patch for independent review, resolve findings, commit,
   push and close its issue. Retain compact evidence before temporary cleanup.

Acceptance: all five previously failing tests pass without weakening assertions
or changing production output. Context-bearing client calls retain the same
context; snapshots reflect only the already-delivered typed stream contract.

Reproduction log: `/tmp/loom-stale-stream-clients-before.log`. HTTP fails its
snapshot; JSON-RPC fails three generated module compilations and one snapshot.

## Validation

The four harness sources change exactly twelve typed client calls from
`Recv(ctx)` to `RecvWithContext(ctx)`. Normalizing that method spelling makes
each file byte-identical to the parent, proving all assertions and supplied
contexts are retained. Server connection-stream calls remain unchanged.
The HTTP snapshot records service-interface embedding, saved opening context,
parameterless Recv and explicit RecvWithContext. The WebSocket snapshot adds
only the service-interface assertion. Both were regenerated through their tests.

All five original failures now pass: HTTP 0.379s and JSON-RPC 11.117s, including
three complete generated module vet/race suites. Scoped pinned lint reports zero
issues. Evidence: `/tmp/loom-866-after.log`, `/tmp/loom-866-lint.log`, and
`/tmp/loom-866-golden-update.log`. Production and DSL fixture inputs are unchanged.
Independent exact-final review and delivery remain pending.
