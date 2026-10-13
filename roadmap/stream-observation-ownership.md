# Stream observation ownership — #749

Status: implementation and bounded validation complete on `f9ee29706`; awaiting
independent final review. The original
three direct failures reproduce under race on this parent (HTTP 0.570s,
JSON-RPC 0.293s). Unopened and ordinary-response controls still pass.

## Sequence and acceptance

1. [x] Reproduce missing SSE lifecycle events and lost SSE/JSON-RPC event identity
   at existing runtime seams, retaining unopened and ordinary-response controls.
2. [x] Reconcile successful negotiation, repeated close, recoverable callback
   failures and physical connection shutdown against the current public event
   contract. Trace multiplexed JSON-RPC operation identity separately from its
   connection context. Reuse existing transport fixtures and lifecycle models.
3. [x] Model only the remaining observation ordering/identity obligation; confirm
   the current counterexample and a candidate that cannot invent, duplicate or
   misattribute stream lifecycle events. Keep callback execution outside runtime
   ownership locks and retain request failure classification as a separate rule.
4. [x] Repair shared request/stream observation ownership. Generated adapters
   provide static identity and call the owner; remove bare-event reconstruction.
5. [x] Verify direct events and actual generated HTTP SSE/WebSocket and JSON-RPC
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

## Selected owner proposal (readiness)

The transport observation layer owns immutable event identity and finite stream
lifecycle delivery. Existing SSE/WebSocket owners remain the only authorities on
commitment and physical release. A stream observation binding snapshots request
identity when the stream/operation is created, independently of RequestObserver's
finished flag. It retains the existing observer and origin time, not user payloads.
This permits a method wrapper retained after synchronous dispatch to report its
own operation without reopening a request span or mutating the connection identity.

| Boundary | Authoritative transition and consumer |
| --- | --- |
| HTTP handler | Attach its existing RequestObserver to lifecycle.Context; SSE construction uses that context. |
| SSE | Commit of successful response headers records open, before flush. Close/terminal failure records close only if committed. Repeated close/unopened close emit nothing. |
| SSE callback error | A prewrite error emits a failure and leaves the stream usable; touched/flush errors retain existing terminal behavior. No observation changes failure routing. |
| HTTP WebSocket | Explicit server-only binding before lazy upgrade. Record opening after successful Upgrade, including a subsequently rejected configuration. Record close after actual native release, including late attachment cleanup. Never infer an inbound observer from client SetConn contexts. |
| JSON-RPC socket | Service/route/verb connection identity has no method or request ID. The same physical connection owns one open/close pair. |
| JSON-RPC envelope | Receive/dispatch binds decoded method/ID/notification identity independently. Generated method wrappers retain this immutable binding for sends after dispatch returns or with a different per-call context. No mutable last-method field on the socket. |
| Suppressed final value | Existing StreamClose + ReasonStreamFinalResponseSuppressed is a logical marker on the operation binding. It never transitions/deduplicates the physical connection's close; the reason distinguishes it from physical ReasonOK closure. |
| gRPC | Preserve existing RequestObserver stream event controls and failure-plus-natural-close behavior; no change to admission, native I/O ownership or error classification. |

### Bounded delivery and callback affinity

Do not introduce a general event queue. Each stream binding stores at most one
pending open and one pending physical close. One drainer emits that finite pair
outside transport/observation mutexes; concurrent/reentrant Close records pending
close and returns, and the active drainer emits it after open. The drainer releases
its claim with defer even when an observer panics; pending terminal metadata remains
available to handler cleanup. A callback panic propagates on the invoking stack.

Repeated write/flush failures and suppression markers are synchronous calls using
immutable event snapshots, not queued telemetry. Observers already permit concurrent
calls; there is no new total callback order across racing write failures or separate
request spans. A slow observer can delay its caller, but cannot cause an unbounded
failure queue while other writes continue. Lifecycle events are recorded at real
transitions; duration is captured there, not when a delayed callback runs.

The WebSocket background pump/cancellation callback records finite terminal metadata
only. It does not invoke a new arbitrary observer callback on a background stack.
Foreground SetConn/write/Close and RunHandler cleanup drain it outside locks. A
handler that ignores cancellation can delay delivery of a recorded close; this does
not imply the physical connection is still live. Preserve that narrow timing limit
explicitly rather than silently swallowing observer panics. RunHandler cleanup
must retain its existing original-panic precedence when observation also panics.

A per-envelope request span may end when synchronous dispatch returns. Its retained
immutable stream identity remains usable for later method-wrapper operations; do
not prolong the request span or filter those events using RequestObserver.finished.
Bindings do not own, replace or cancel per-call contexts.

### Rejected alternatives

- Emit from SSE/WS state locks: reentrant Started/Close deadlocks.
- Unlock then independently emit open and close: a later close can overtake open.
- Put failures into a serialized queue: an arbitrarily slow observer permits an
  unbounded queue under recoverable failures. Queue only the finite lifecycle pair.
- Reuse the connection RequestObserver as the current JSON-RPC method: retained
  operations acquire another method's identity; ending dispatch loses later sends.
- Treat suppression as physical closure or every failure as terminal: contradicts
  multiplexing and recoverable prewrite errors. Retain existing reason semantics.

## Implementation sequence and bounded acceptance

1. [x] Refresh clean managed worktree after #737 delivery and reproduce original
   runtime failures using its unchanged test inputs on current source.
2. [x] Add a bounded observation model for missing events, independent publication,
   finite lifecycle delivery and immutable retained identity. Reuse the existing
   WebSocketShutdown model for release ordering; do not remodel transport I/O.
3. [x] Independent readiness review of owner, callback affinity and acceptance.
4. [x] Implement immutable stream observation binding and finite lifecycle owner;
   integrate runtime transitions, then generated context/binding construction.
   Delete bare failure/suppression event reconstruction and obsolete emission paths.
5. [x] Direct race checks cover normal/open-flush-failure/unopened/repeated-close,
   recoverable prewrite failure then successful send, observer reentry and panic,
   blocked observer plus repeated failures (no retained queue), physical shutdown
   before handler return and delayed foreground draining, two retained method
   bindings after dispatch completion, client SetConn isolation, and suppression
   followed by successful same-socket dispatch. Preserve gRPC existing controls.
6. [x] Reuse one HTTP mixed SSE/WebSocket fixture and one JSON-RPC fixture with SSE,
   WS methods and events/stream: emitted binding/source assertions plus actual
   mounted normal/rejected/lazy/GET/notification/failure flows. Compare affected
   parent/candidate/fresh-process output and compile the bounded modules. Regenerate
   affected checked-in fixtures through their owning generator only.
7. [ ] Freeze exact packet, independent final review, atomic delivery and cleanup.

No new public logging policy, payload capture, background telemetry service or
scheduler changes. Observation must not change wire bytes, native shutdown cause,
request failure classification, service invocation or response suppression.


## Readiness evidence

`observability/transport/tla/stream_observation` holds the bounded observation
model. Current missing-open fails in 5 states; rejected shared identity, split
publication, callback-under-lock and general queue designs fail in 16, 11, 12 and
36 states respectively. The finite binding candidate passes all invariants over
528 states. `identity-shared` is a rejected proposal, not a claimed existing
connection mutation. Existing physical WebSocket models are reused unchanged.

Durable runtime probes are `http/stream_observation_readiness_test.go` and
`jsonrpc/stream_observation_readiness_test.go`. Besides the original three
failures, the added no-blocking TryLock probe proves failure callbacks currently
run under the SSE write mutex. The same call then successfully sends a comment,
confirming that a prewrite callback error is recoverable before the observation
assertion fails. Logs and exact owner hashes are frozen in the readiness manifest.

The proposed shared API is a stream observation binding constructed from a
RequestObserver snapshot (or an explicitly unknown fallback identity). It owns
finite lifecycle state and direct immutable failure/logical-event delivery;
request spans remain separate. Server WebSocket creation explicitly installs the
binding before lazy attachment; shared SetConn remains neutral for clients.
Generated wrappers retain the binding, not an entire mutable connection observer.
Exact API spelling is an implementation choice; no per-template emission policy
or per-consumer state machine is authorized.

## Implementation refinement: generated SSE terminal ownership

The first generated checks found two additional consumers of the same lifecycle
contract. The existing HTTP authorization-stream specimen emits opening but no
closing for a successful SSE handler because its adapter never closes the writer
on return. The existing JSON-RPC viewed-stream specimen holds `s.mu` across
`writer.Open/WriteEvent/Close`; moving callbacks outside the writer mutex alone
still calls an observer under this enclosing generated terminal/view mutex.
A generated private-package TryLock probe reproduces that conflict without a
hanging goroutine. This is the same callback ownership requirement, not a new
streaming policy or an unrelated issue.

The refinement moves terminal admission into the already authoritative
`SSEStreamWriter`, deleting JSON-RPC's generated `closed` flag and write mutex.
Keep a view-only mutex for mutable view selection; sample it before I/O.

- Add one terminal writer operation accepting a preparation callback that returns
  an `SSEMessage` or an error. Preparation runs after closed admission while the
  writer owns serialization. An encoding error leaves the writer open. Once a
  frame is prepared, write/flush failure still closes it, matching existing
  JSON-RPC `complete` semantics. A nil preparation closes without committing a
  frame (notification suppression). No competing send/final can pass admission
  after this operation. Context failure after admission preserves the existing
  final-operation terminal behavior; generic Open/WriteEvent cancellation rules
  remain unchanged.
- A shared JSON-RPC runtime completion function owns response encoding
  and suppression for this terminal writer operation. Generated methods pass
  typed response construction, request-ID presence and writer. The existing
  wire framing and response/view/error conversion stay unchanged. Suppression
  is emitted only after successful terminal admission, outside the writer lock;
  a repeated completion cannot emit another suppression marker. Its existing
  StreamClose/suppression reason remains logically distinct from physical close.
- Add a writer handler-finalization method for deferred cleanup. HTTP normal and
  mixed SSE adapters, JSON-RPC typed handlers and protocol-error adapters defer
  it. It closes the writer, emits closure only if committed, and preserves an original panic if the
  observer also panics. It does not make streams usable after handler return.
- Delete generated JSON-RPC `complete` locking/state and redundant write guards;
  Open, comment, ordinary send and final send consume the shared writer's state.
  Do not unlock around old bookkeeping or add an asynchronous observation queue.

The existing `StreamObservation` model now has a bounded terminal mode with two
final callers and one ordinary sender. `terminal-wrapper` reproduces callback
execution under the enclosing owner (14 distinct states before the violation).
`terminal-split` rejects releasing ownership after admission but before write
(35 distinct states, late write). `terminal-owned` preserves single
final, no write after terminal, and callback outside ownership over 65 distinct
states. Preparation failure occurs while the admitted caller owns serialization, then
releases ownership without publishing terminal state; another caller can complete. Physical I/O and codec
behavior remain implementation-test obligations; existing lifecycle/identity
configurations are unaffected by the separate terminal mode.

Focused implementation acceptance adds the generated TryLock control, concurrent
ordinary/final writes, duplicate final and duplicate suppression, preparation
failure then successful error completion, unopened/natural/panic handler end.
Reuse the existing viewed-stream module and authorization-stream module; no new
fixture family, protocol model or broad suite is needed. Model run command:

```sh
java -cp /path/to/tla2tools-1.7.4.jar tlc2.TLC -deadlock \
  -metadir /tmp/stream-terminal-owned -config terminal-owned.cfg StreamObservation.tla
```

The initial model draft omitted parentheses around boolean next-state assignments;
that checker setup error was fixed before the reported invariant checks. The
root independently accepted this refinement, including serialized preparation
failure, before its production changes.

## Final acceptance evidence

The shared implementation deletes bare SSE failure and JSON-RPC suppression event
construction, plus generated SSE terminal flags/serialization. The shared writer
owns terminal admission and handler end. View selection alone remains locked in
generated SSE wrappers. WebSocket observation is installed only by server
construction; per-envelope bindings remain immutable after dispatch.

- Whole affected runtime packages pass race: HTTP 6.468s, JSON-RPC 2.015s,
  transport observation 1.264s. After moving eager JSON-RPC attachment inside
  `RunHandler` cleanup, JSON-RPC passes again in 2.173s. Its opening-observer-panic
  witness previously returned abnormal EOF/1006 through deferred cancellation;
  now it follows the existing 1011 panic cleanup path and preserves the panic.
  This is cleanup-cause correspondence, not a claim that the parent leaked a
  connection indefinitely. Existing gRPC admission/observation controls pass.
- The authorization-stream HTTP specimen and viewed-result JSON-RPC specimen
  build, vet and pass mounted races (3.561s/4.328s outer checks). They cover lazy
  authorization rejection, successful SSE/WS, retained dynamic/fixed method
  identity, terminal callback reentry, invalid-view error completion and ID-less
  notification suppression. The same generated RPC module passes races after
  the final eager-attachment cleanup correction.
- Parent source is a physical `git archive f9ee29706`, with only the two existing
  test output directories redirected for capture; no CLI overlay is relied on.
  All 48 generated specimen artifacts compare against the candidate: exactly
  five server files change, with static context/binding construction and shared
  completion replacing generated state. All other files are byte-identical;
  fresh-process candidate generation is exactly identical. Parent modules also
  compile/run their original tests; both viewed-module gates include vet.
- The existing HTTP ticktock, JSON-RPC ticktock and mixedtick fixture trees are
  regenerated through the CLI. All 55 parent generated files match checked-in
  source. Parent/candidate modules build/vet; candidate/fresh-process output is
  exact. Exactly five server files change (one HTTP handler, two RPC handlers,
  two RPC stream implementations). Candidate fixture runtime races pass; the
  mixed fixture has no tests. Source fixture trees match candidate hashes.
- The existing regenerated GET listener test now also checks mounted observation
  identity, one physical pair and one logical suppression marker. Its focused
  generated test uses the variant's `clock.go` and its observation test because
  the copied fixture's other tests target Tick/Tock. The network wire assertions
  remain unchanged. The combined check passes in 7.587s.
- Owning SSE golden tests and seven actual HTTP handler snapshots are regenerated
  from codegen, with all differences inspected. Scoped sections pass 2.776s/3.666s.
  Runtime/codegen and integration-test scoped lint both report zero issues.

The bounded observation model retains 528 passing states; corrected serialized
terminal preparation has 65 passing states, with enclosing-lock and split-write
alternatives failing their intended invariants at 14 and 35 states. Codec/I/O,
observer panic precedence, reentry and finite callback storage are checked by
implementation tests, not claimed as unbounded model proofs. No asynchronous
queue, changed timeout, request failure policy or wire encoding is introduced.

The initial candidate's generated import spelling/header, missing required title
in two added test values, and archive-root Git discovery were corrected before the
reported passing gates. Snapshot capture also initially assumed only WS changes;
the mixed HTTP/SSE handler's intended context/End difference was inspected and
adopted from rendered output. Regeneration-only module sum drift is preserved in
temporary evidence and excluded from this transport repair. No dependency changes.
