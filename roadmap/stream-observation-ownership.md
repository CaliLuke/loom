# Stream observation ownership — #749

Status: root readiness investigation on `60da15241`; no production repair yet.

## Sequence and acceptance

1. [x] Reproduce missing SSE lifecycle events and lost SSE/JSON-RPC event identity
   at existing runtime seams, retaining unopened and ordinary-response controls.
2. [ ] Reconcile successful negotiation, repeated close, recoverable callback
   failures and physical connection shutdown against the current public event
   contract. Trace multiplexed JSON-RPC operation identity separately from its
   connection context. Reuse existing transport fixtures and lifecycle models.
3. [ ] Model only the remaining observation ordering/identity obligation; confirm
   the current counterexample and a candidate that cannot invent, duplicate or
   misattribute stream lifecycle events. Keep callback execution outside runtime
   ownership locks and retain request failure classification as a separate rule.
4. [ ] Repair shared request/stream observation ownership. Generated adapters
   provide static identity and call the owner; remove bare-event reconstruction.
5. [ ] Verify direct events and actual generated HTTP SSE/WebSocket and JSON-RPC
   SSE/WebSocket behavior, including rejected requests, lazy/unopened streams,
   GET listeners, notifications, failure and natural completion. Compare bounded
   generated outputs and compile them; retain existing gRPC observation controls.
6. [ ] Independent final review, atomic delivery, issue closure and cleanup.

## Established evidence

Two additive test-only overlays on the exact source above exercise existing
runtime APIs without changing production. HTTP race checks reproduce absent open
and close after a successful Open/Close, plus a flush failure event with empty
service/method/route/verb. JSON-RPC reproduces lost operation identity on a
suppressed notification response. Never-opened SSE and ordinary JSON-RPC response
controls pass. The opening-flush probe currently also expects an opening event;
its exact negotiation boundary must be settled from the existing contract before
using that assertion as implementation acceptance.

Inputs: `/tmp/loom-749-http-observation_test.go`,
`/tmp/loom-749-rpc-observation_test.go`,
`/tmp/loom-749-readiness-overlay.json`; results:
`/tmp/loom-749-readiness.log` (three failing cases, two passing controls).
These are direct runtime witnesses, not generated-module acceptance.

## Shared boundary and current gaps

- `observability/transport.RequestObserver` owns operation identity, timing,
  capture metadata and request terminal classification. Its existing stream
  methods enrich events, but do not own stream state or deduplicate transitions.
  Public Event fields are optional where identity is not yet known.
- `http.HandlerLifecycle` creates this observer but does not attach it to its
  returned context. HTTP SSE templates additionally construct the writer with
  the original request context instead of the enriched lifecycle context.
- `http.SSEStreamWriter` owns commitment/write/close state; it constructs bare
  failures and emits no open/close. A callback failure before touching the writer
  can be recoverable, so StreamFailure cannot be treated as unconditional stream
  termination. Request failure classification is explicitly separate today.
- `http.WebSocketStream` owns attachment and terminal connection cleanup. Its
  configuration and read pump can end the connection before the handler returns;
  observed opening/closing must follow those actual transitions without reviving
  streams or invoking arbitrary observer code under its ownership locks.
- `jsonrpc.ServeSSE` already attaches a request observer with decoded identity.
  `CompleteStream`/`CompleteStreamError` reconstruct bare suppression events,
  discarding it. A suppressed logical final response is not proof that the shared
  physical WebSocket connection closed.
- `jsonrpc.ServeWebSocket` currently has no request observation owner or Service
  field in its spec. One socket can carry several method invocations. Mutating
  one shared observer with the most recent method would misattribute later
  failures and completion. Connection-level fields and per-operation identity
  need explicit correspondence before implementing this branch.
- gRPC `ServeStream` already emits open/natural close through RequestObserver;
  its write-failure helper enriches identity. Keep this control and do not infer
  that StreamFailure replaces StreamClose: current tests allow both events.

The fix must make existing runtime transitions feed the shared identity owner.
Adding independent emission rules to each generated template or treating every
failure as terminal would leave the underlying disagreement in place. No new
user-selected logging policy, payload capture or observer backend is needed.
