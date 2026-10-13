# gRPC per-message context ownership — #726

Readiness parent: `7baf82f92f6b47c882ca46caef85b277271abf8d`. Work in the
managed `grpc-message-context` checkout. Creation succeeded; attachment and one
retry hit the desktop's 100-identity limit. Readiness was independently approved;
implementation is complete and awaiting exact-final independent review.

1. [x] Reproduce ignored contexts with complete generated client/server code mounted
   over bufconn: pre-canceled operations must perform no native I/O, and blocked
   client/server receives must return the explicit context error. Preserve normal
   duplex exchange and send-half-close/final-receive controls.
2. [x] Verify the pinned upstream gRPC API and demonstrate native lifetime boundaries
   in a direct mounted probe. A derived server context does not interrupt native
   RecvMsg; returning the gRPC handler closes its transport. Client opening
   context cancellation does close its native stream. Do not assume a context
   wrapper supplies a server cancellation API that does not exist.
3. [x] Model the current ignored-context behavior and rejected partial designs before
   selecting an owner. Check cancellation progress, no I/O after terminal state,
   no worker-per-message accumulation, natural completion, half-close preservation,
   callback/operation races and observer terminal ordering.
4. [x] Define one gRPC stream lifecycle owner. Generated wrappers derive contextual
   operations from it; runtime owns termination. Preserve conversion, view and
   status mapping. Preserve pre-frame authentication admission (#743), original
   request cancellation and client Close half-close semantics (#720). Keep #859
   mixed service accessor and #758 concurrent-send work independent.
5. [x] Freeze readiness tests, model, upstream/source correspondence and bounded
   implementation acceptance for independent review. Only then implement, compare
   representative generated modules/affected goldens, compile/vet/race, update
   public cancellation docs and obtain fresh exact-final review before root commit.

## Existing contract and API evidence

Generated `SendWithContext`, `RecvWithContext`, `SendAndCloseWithContext` and
`CloseAndRecvWithContext` simply call their context-free counterparts in
`grpc/codegen/jennifer_stream.go`. The service interfaces and interceptor hooks
already expose explicit operation contexts; ignoring them is a defect.

The pinned dependency is grpc-go v1.84.0. Its
[ServerStream contract](https://github.com/grpc/grpc-go/blob/v1.84.0/stream.go#L1670)
exposes Context/SendMsg/RecvMsg and metadata methods, with no cancel/close method.
The native methods wait for transport completion, independently of a derived
application context. Its
[server handler lifecycle](https://github.com/grpc/grpc-go/blob/v1.84.0/server.go#L1440)
writes final status after the handler returns. Client stream cancellation derives
from its opening context; `CloseSend` closes only the send direction and must
remain usable with later Recv/CloseAndRecv.

The existing HTTP WebSocket owner checks an already-done context before starting
I/O. During an active operation it aborts the transport when that context ends.
The gRPC contract should match those boundaries: a pre-canceled operation returns
its context error without I/O; canceling an active blocked operation terminates
the whole RPC because native gRPC offers no per-message abort. An operation that
finishes before cancellation remains successful. User service code must still
cooperate with cancellation; the framework cannot forcibly stop arbitrary Go
code or custom interceptors that refuse to return.

## Owner alternatives to evaluate

- A preflight context check alone prevents only already-canceled I/O.
- Canceling a derived server context cannot unblock native I/O.
- Returning from a select while leaving a new native operation goroutine behind
  on each call leaks work and admits conflicting later operations.
- Waiting for the service worker before returning the gRPC handler forms a cycle:
  native receive needs handler return, while service completion needs receive.
- Candidate: `ServeStream` owns one RPC-level service execution worker and a
  terminal signal. Per-operation callbacks signal that owner; native operations
  stay in their caller goroutine. Handler return releases native server I/O.
  A client owner instead cancels the child context supplied when opening its
  native RPC. Its lifetime must survive response construction and half-close.

This initial candidate was evaluated and rejected as recorded below. Native
probes and models establish the selected design and its explicit limits.

## Readiness findings and selected candidate

The mounted generated parent module builds and vets, then fails six contract
branches: pre-canceled server Send/SendAndClose and client Send perform native
writes; server Recv, client Recv and client CloseAndRecv ignore a 100 ms deadline
for at least 600 ms. The bidirectional half-close/reply control passes. Native
bufconn probes independently establish that derived server cancellation does not
release RecvMsg; native handler return releases both blocked receive and
flow-controlled send. Canceling the client opening context releases its receive.

The initial RPC execution-worker candidate passes the lifetime abstraction, but
is rejected by the recovery boundary. `RequestObserver.End` and configured gRPC
stream recovery interceptors rely on the original handler stack. Relaying a
worker panic works only before handler return; a later service panic cannot reach
that interceptor. Keep service execution synchronous.

The selected candidate is **one per-RPC native-operation owner**, with these
implementation obligations:

1. `ServeStream` admits authentication synchronously, restores method/service and
   observation context, then installs the stream owner before Decode/Handle.
   Both execute on the original handler stack. Preserve original request
   cancellation even when authentication supplies a replacement value context.
2. The server owner retains at most one active native send and one active native
   receive. All generated native operations use these same direction owners.
   On contextual cancellation, latch the first terminal cause, cancel the
   service-derived context, reject later operations and return the cause to
   existing operation callers. Do not dispatch repeated canceled operations.
   The native handler can then return normally or through outer recovery; native
   gRPC closes its transport and releases the tracked workers. Never join a
   blocked native worker before returning the handler.
3. Native workers own their data. Clone outgoing protobuf messages before native
   dispatch; receive destinations/results stay worker-private until successful
   rendezvous. Conversion/view projection stays on the operation caller stack.
   A canceled return must not expose a destination still being mutated or let
   later payload reuse change a pending send. The service and authored codec must
   not concurrently mutate input while conversion itself is running.
4. Use one operation lease for context registration and completion arbitration.
   A pre-canceled operation returns its own context error before native I/O and
   leaves a healthy RPC usable. Active cancellation terminates the RPC. Completion
   that wins arbitration disables/joins its callback; canceling that completed
   operation's context must not terminate a later operation. Pending operations
   see the owner's first terminal cause; do not turn a successful half-close into
   full cancellation.
5. The client owner creates the child context used to open the native RPC and
   survives response decoding. Give streaming invokers explicit ownership, e.g.
   a meaningful `NewStreamInvoker`, while retaining unary invoker behavior.
   Generated response adapters retain the owner-bearing invocation context.
   Never obtain the lifetime by prematurely calling native ClientStream.Context,
   which disables grpc-go retries. Release on opening/decoding failure, terminal
   client receive, or completed CloseAndRecv. Send EOF is not final status: retain
   the receive path. CloseSend is a half-close and does not release the owner.
6. Preserve existing error mapping and observation origin in the caller after
   the operation rendezvous. A canceled server write retains stream-write failure
   classification; canceled receive retains request-decode classification.
   Native workers never publish late observation or mutate terminal request state.
   ServeStream must not emit clean completion when its owner is terminal, even
   if service code returns nil after a canceled operation. Native errors, EOF,
   conversion failures, designed errors and successful completion keep their
   existing distinctions.
7. Keep ordinary service and conversion panics on their original stack. A native
   custom stream override/codec panic in a worker is relayed to the active caller
   and re-panicked there. If cancellation already won, contain and report that
   late native panic through an internal diagnostic; it cannot be delivered to
   an already-returned interceptor. Do not let it crash an unrelated goroutine,
   silently discard it, or change already-finalized transport observation. This
   is a narrow asynchronous-native-hook limit, not an exception for service code.

Native cancellation/reclamation assumes grpc-go transport progress and service
code returning after its operation reports cancellation. Arbitrary service loops,
custom native hooks that never return, blocking custom observer callbacks and
application-created goroutines cannot be forcibly stopped by Go. Bounded work
ownership is not a claim of unconditional termination for such code. Concurrent
calls of the same native direction retain grpc-go's documented restriction;
#758 does not become part of this repair.

## Bounded implementation acceptance

- Direct owner tests: pre-cancel no-I/O and healthy reuse; cancel blocked send and
  receive; first terminal cause; terminal retry rejection; at-most-one direction
  owner; detached bytes and private receive output; completion/callback race;
  worker drain after native termination; active panic relay and late containment
  with a captured diagnostic. Use gates instead of sleeps for ordering races.
- Mounted generated two-method witness: all original red assertions turn green;
  add successful operation followed by cancellation of its old context while a
  later operation succeeds, canceled-operation retry, nil-return-after-cancel
  failure observation, original request cancellation and service recovery before
  and after an operation deadline. Preserve client half-close followed by final
  receive. Add a real flow-controlled server send with a large bytes field.
- Reuse the five `TestServiceClientStreamArtifacts` designs: server stream,
  client stream, client stream without a result, duplex, and shared SSE/gRPC
  service. Add the current two-method witness and `StreamAdmissionDSL` for seven
  complete module comparisons. Each distinct changed adapter/close shape is
  represented; shared service/SSE output must remain unchanged. Preserve viewed
  conversion and designed streaming error structural/golden controls already in
  `TestStreaming`, `TestStreamingWithErrors`, `TestStreamingErrorsWithValidation`
  and `TestStreamingErrorComparison`; extend one existing specimen if its view
  branch is not covered by these seven designs.
- Run existing three mounted stream-admission cases and direct authentication
  context controls. They prove denial before initial receive and replacement
  authentication contexts retaining original cancellation and metadata. Run
  direct runtime observation/response/error contracts for origin and EOF.
- Compare generated artifacts against the pushed parent and an independent
  candidate repeat byte for byte; compile/vet each distinct complete module,
  race the changed mounted cases. Intended differences are stream-owner wiring,
  contextual native operations, and corresponding focused goldens. Protocol
  descriptors, service declarations, unary behavior and HTTP/SSE output stay
  byte-identical. Do not run the full exported-design corpus.
- Update `docs/grpc-guide.md`, consumer operation-context guidance, framework
  ownership guidance and model index. Render Markdown, run affected lint and
  diff checks, then freeze exact source/evidence/artifact hashes for independent
  final review. The original readiness packet contained no production changes.

Implementation authorized after independent readiness approval; safely advanced
to pushed `641c103c21c8f48558b9801323f7d62c8a07919f`. Original readiness inputs
are preserved in `/tmp/loom-726-readiness-sources`. #785 concerns JSON-RPC wrapper
shape; the distinct gRPC concurrent-send report is #758.

## Implementation checkpoint

- [x] Shared runtime owner and generated adapters implemented. Service execution,
  conversion and observation remain synchronous on the caller stack. Final
  CloseAndRecv reserves both native directions; a gated regression failed before
  that refinement and passes afterward. A decoded nil client result releases its
  opening context instead of transferring an unreachable lifetime.
- [x] Full runtime package passes race checks; the exact final timing is recorded in
  the evidence packet. The generated mounted
  context test, all three admission cases, streaming view/error controls,
  service-client stream controls, client endpoint initialization and response
  decoder goldens pass together under race (30.242 s).
- [x] Seven representative designs produce 85 generated artifacts per source.
  Exactly 21 files change: each design's gRPC server adapter, client adapter and
  client decoder. Parent and candidate modules all build and pass vet; isolated
  candidate repetition is byte-identical. The shared SSE/gRPC full CLI companion
  has 29 artifacts and only the same three gRPC files change. Its HTTP/SSE,
  service and protobuf files are byte-identical; both complete modules build/vet.
- [x] Focused snapshots regenerated from rendered sections: 28 streaming snippet
  values, seven streaming client-initializer goldens, four streaming decoder
  goldens. Unary expectations are unchanged. No generated tree was hand edited.
- [x] Model correspondence updated for atomic final receive reservation. The
  refined bounded configuration passes (118 generated / 68 distinct states);
  unbounded negative control still fails WorkersBounded (17 / 14). Other 17
  configurations retain their independently reviewed, unchanged model evidence.
- [x] Public guide, consumer guidance, framework owner guidance and formal index
  updated. Scoped runtime/codegen lint is green; fixture lint and final manifest
  checks are recorded in the final evidence packet.
- [ ] Fresh exact-final nonimplementer review, then root integration/commit/push.
- [ ] After delivery and review, preserve compact receipts and reproduction inputs
  before removing inactive generated modules, model state and raw logs.

Final evidence lives under `/tmp/loom-726-*` until review: runtime and generated
final logs, base/candidate/repeat artifact hash ledgers, all 14 module build/vet
results, shared CLI comparison/build results, model refinement results and the
final manifest. The frozen readiness manifest and independent verdict remain
separate historical evidence. No exhaustive exported-design audit was run.

### Pre-freeze review corrections

Root's owner inspection found two further lifetime boundaries, each reproduced
before repair. A context cancellation between receive preflight checks incorrectly
finalized a client despite no admitted I/O; active send cancellation left a
parent callback registered. The shared runner now reports admission explicitly,
and rejected operations cannot finalize the client. First terminal transition
detaches every parent watch; registering after termination stops the new watch
immediately under the same ownership lock. No adapter changes were needed.

The new AdmissionCleanup model reproduces both defects and passes the selected
owner (11 / 8 states). Gated direct tests cover both registration orders and
rejected final receive while either direction is occupied. Runtime race,
mounted context/admission and scoped lint gates were rerun after correction.
The earlier unchanged view/error/golden, fixture lint and byte comparisons remain
valid; no redundant generated corpus or repeat builds were added.
