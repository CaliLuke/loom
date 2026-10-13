# JSON-RPC stream expectation adoption (#898)

The delivered shared stream lifecycle/observation owner binds operation identity,
completes SSE through the owned writer and releases terminal writers. Four older
JSON-RPC tests retain the previous generated helper or omit those owner calls.
This task adopts the implemented contract without changing production behavior.

## Sequence and acceptance

- [x] Read the owning templates and affected assertions before edits. Reproduce
  the four current-source failures in `/tmp/loom-ci-current-jsonrpc-selected.log`;
  the older `2a98e4123` CI job reports the same failures.
- [x] Update the SSE helper assertion to require `CompleteSSEStream` with its
  writer and original request-ID presence. Regenerate only the three affected
  snapshots via their existing tests. Retain safe-error checks and WebSocket
  request-ID/client runtime assertions.
- [x] Run the four existing comparison/structural tests under race without update
  mode. Run `TestJSONRPCViewedStreamResultGeneratedModule` for compiled SSE and
  WebSocket observation/terminal paths and the existing stream observation and
  completion runtime tests. Every delta must be a writer End or retained observer
  binding already delivered by the shared owner.
- [ ] Freeze the exact diff for independent review, commit/push one adoption
  ticket, close #898, and preserve compact evidence before cleanup.

Workflow: red reproduction precedes expectation edits; focused comparisons and
existing generated runtime controls precede independent review and atomic
publication. No new protocol policy, production source changes, new harness or
exhaustive corpus run. Existing stream models and proof evidence remain valid.

Verification on `41d2d578b`: exact comparisons and the existing generated SSE/WS
module pass under race (7.338s), including generated compilation/vet and retained
observer/terminal cases. Focused runtime races pass (JSON-RPC 1.306s, observation
1.287s); configured lint reports zero new issues. The three snapshots add only
three writer releases and the existing WebSocket observer import, field, three
context bindings and constructor binding. No other generated output changes.
