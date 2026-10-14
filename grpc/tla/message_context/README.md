# gRPC operation context ownership (#726)

These models compared cancellation designs before implementation and now record
the selected owner and its correspondence to runtime tests.
The generated regression in `../../codegen/stream_context_readiness_test.go`
first reproduces actual ignored contexts on parent `7baf82f92`; native ownership
and panic probes are in `../../native_stream_context_test.go` and
`../../native_stream_panic_test.go`. The latter preserves the original native-worker prototype; the actual runtime
owner is checked separately by `stream_context*_test.go`.

## Models and correspondence

- `MessageContext` reproduces ignored context, pre-canceled I/O and partial
  cancellation alternatives. Its `owned` configuration is the **initial RPC
  execution-worker candidate**, which passes lifetime checks but is rejected by
  `PanicAffinity`. Do not treat that configuration alone as the chosen server
  architecture. Client opening-context cancellation and CloseSend half-close
  distinctions remain relevant to the final design.
- `OperationLease` orders native completion, context callback and method return.
  Once completion publishes success, a late callback cannot cancel the RPC or a
  later operation. The unsafe configuration demonstrates why callback lifetime
  must be joined/arbitrated, not merely stopped without checking its result.
- `PanicAffinity` rejects moving service execution off the native handler stack,
  even with a recover-and-relay mechanism: after handler termination a service
  panic cannot reach its original interceptor. `same-stack` is the selected
  service path. `native-panic` models the narrower native-hook boundary: active
  panic relays to its caller, while a late native panic is contained and reported.
  It does not assert outer recovery for a late panic after that stack has ended.
- `BoundedOperations` is the selected server ownership abstraction: one send and
  one receive at most, caller cancellation returns before native I/O finishes,
  terminal retries cannot create unowned work, and cooperative handler return
  releases both tracked operations. Error observation occurs on the caller path
  before handler completion; native unwind does not mutate terminal observation.
  A native operation may outlive its canceled method; it remains owned and bounded.
- `MessageOwnership` rejects borrowing outgoing message data across a canceled
  return. A detached protobuf snapshot keeps native serialization independent of
  application buffer reuse. Private receive results are the symmetric runtime
  obligation; they are checked in implementation tests, not modeled separately.

- `AdmissionCleanup` separates rejected operations from admitted native work.
  Blanket receive finalization wrongly terminates a healthy RPC when cancellation
  arrives between preflight checks; retaining parent callbacks after send abort
  leaks registration until the parent ends. The selected owner carries explicit
  admission and detaches callbacks on every first terminal transition, including
  registrations that finish after termination.

## Assumptions and limits

One RPC, two native directions and one context lease represent the ownership
boundary; retries are bounded to three to expose accumulation. Weak fairness
models scheduled callbacks, native closure and service return after a canceled
operation. It does not prove real-time bounds or forcibly terminate arbitrary Go
code. The actual grpc-go API has no server-side cancel method; native handler
return is necessary. Native bufconn tests check this assumption for receive and
flow-controlled send. Client cancellation uses the context supplied at native
stream creation, not CloseSend or a premature ClientStream.Context call.

Service execution remains synchronous, retaining observer End/re-panic and outer
interceptor recovery. Native custom codecs/stream overrides must eventually
return; their late panic is contained and reported after cancellation, with no
claim that an already-returned outer interceptor can recover it. Framework
conversion/view code stays on the caller stack. Concurrent application mutation
while conversion runs, uncooperative observer callbacks and application-created
workers are outside the guarantees. Same-direction native concurrency remains
restricted by grpc-go; #758 is independent.

## Run and expected results

Use Corretto 25 and pinned TLA+ 1.7.4 (`tla2tools` reports TLC 2.19), SHA-256
`936a262061c914694dfd669a543be24573c45d5aa0ff20a8b96b23d01e050e88`.
From this directory, for example:

```sh
java -XX:+UseParallelGC -cp /path/to/tla2tools-1.7.4.jar tlc2.TLC \
  -workers 1 -metadir /tmp/loom-message-context-bounded \
  -config bounded.cfg BoundedOperations.tla
```

Use `MessageContext.tla` for current through owned; `OperationLease.tla` for
unjoined/lease; `PanicAffinity.tla` for naive-worker/relay/same-stack/native-panic;
`BoundedOperations.tla` for bounded/unbounded; and `MessageOwnership.tla` for
shared/detached. Use `AdmissionCleanup.tla` for blanket-receive, retained-callback
and admission-cleanup. Expected negative runs are invariant (exit 12) or temporal
(exit 13) counterexamples, never parser or configuration failures.

| Configuration | Expected exit | Generated / distinct states |
| --- | ---: | ---: |
| `current` | 13 | 32 / 32 |
| `preflight` | 12 | 10 / 10 |
| `derived` | 13 | 36 / 36 |
| `join-worker` | 13 | 46 / 44 |
| `per-call` | 12 | 26 / 26 |
| `close-cancels` | 12 | 19 / 19 |
| `late-observer` | 12 | 33 / 33 |
| `unguarded` | 12 | 44 / 44 |
| `owned` | 0 | 61 / 52 |
| `unjoined` | 12 | 12 / 9 |
| `lease` | 0 | 13 / 10 |
| `naive-worker` | 12 | 6 / 5 |
| `relay` | 12 | 7 / 6 |
| `same-stack` | 0 | 7 / 6 |
| `bounded` | 0 | 118 / 68 |
| `unbounded` | 12 | 17 / 14 |
| `native-panic` | 0 | 7 / 6 |
| `shared` | 12 | 7 / 6 |
| `detached` | 0 | 8 / 6 |
| `blanket-receive` | 12 | 3 / 3 |
| `retained-callback` | 12 | 10 / 8 |
| `admission-cleanup` | 0 | 11 / 8 |

Rerun affected models if implementation changes arbitration, worker ownership or lifetime
assumptions. Keep raw checker states and logs outside the repository.

## Implemented correspondence

`grpc/stream_context.go` owns `begin`, `cancelOperation`, `complete` and
`abortLocked` under one mutex. `runStreamOperation` installs one callback and
stops or joins it before returning. A native completion publishes an immutable
result; cancellation retains occupied slots until native completion and rejects
new work. `CloseAndRecv` reserves both slots atomically: `StartFinal` conservatively
represents its single native worker as two occupied direction slots. Ordinary
duplex work still uses one worker per direction.

`SendStreamMessage` clones protobuf storage on the caller stack; receive workers
hold private results. Generated conversion, error mapping and observation remain
on the caller stack. `ServeStream` keeps Decode/Handle synchronous and watches
both admitted and original RPC contexts. It suppresses clean success after a
terminal cancellation. `NewStreamInvoker` transfers its opening child context
only with a nonnil successful decoded response; failures and empty responses
release it. Client final receive releases ownership; CloseSend does not.

`grpc/stream_context*_test.go` checks direction bounds, final receive reservation,
data detachment, completed-context isolation, first cause, original authentication
cancellation, observation finality and active/late panic behavior. The generated
`TestGRPCMessageContextGenerated` mounts the actual wrappers and covers pre-cancel,
blocked receive/final receive, real flow-controlled send, half-close, completed
operation isolation and original-stack service recovery. Native bufconn probes
check the grpc-go handler-return assumptions independently. These tests establish
implementation correspondence; the finite models do not prove arbitrary Go code.

`runStreamOperation` returns an admission flag in addition to its value/error;
`ReceiveStreamMessage` finalizes clients only for admitted operations. Rejections
before admission, including direction conflicts, leave a healthy owner intact.
`abortLocked` detaches parent callbacks on every terminal transition. `watch`
arbitrates callback registration against that same transition, stopping a newly
registered callback immediately if termination already won. The gated tests in
`stream_context_admission_test.go` reproduce the original pre-admission and
retained-callback defects, then verify both registration/termination orders.

## Terminal half-close admission (#872)

`HalfClose.tla` isolates cleanup admission from ordinary message operations.
Current admission returns an error on a released opening context, so terminal
receive followed by generated `Close()` exposes internal cancellation (or the
owner's terminal status when release races active Close). The rejected `retain`
alternative avoids cancellation by retaining a finished RPC's resources.
`classified` instead treats half-close as already satisfied after terminal
release, without native I/O. Live half-close still uses the send lease and leaves
receive ownership open. Native completion published before release retains its
result; late completion after release cannot replace the terminal outcome.

Run the same pinned TLC tool as above against `HalfClose.tla`, with each of
`HalfCloseCurrent.cfg`, `HalfCloseRetain.cfg`, `HalfCloseClassified.cfg`, and put
`-metadir` outside this checkout. Readiness results:

| Configuration | Exit | Generated / distinct states | Result |
| --- | --- | --- | --- |
| HalfCloseCurrent | 12 | 4 / 4 | Internal terminal error escapes cleanup |
| HalfCloseRetain | 12 | 2 / 2 | Terminal resources remain retained |
| HalfCloseClassified | 0 | 42 / 30 | All invariants hold |

`streamOperationKind` separates half-close policy from the send-direction lease.
`begin` observes operation cancellation and terminal state under the owner lock;
a terminal half-close yields no operation, callback or worker. For an admitted
half-close, the existing locked completion arbitration preserves published native
results; terminal release otherwise satisfies cleanup while the native worker
keeps its direction slot until unwinding. Original context cancellation is also
recognized before the parent's scheduled callback publishes the terminal cause.

The model bounds cleanup to two calls and abstracts received status details,
parent-callback mechanics and the existing send/receive direction lease. Those
remain obligations of the original models and implementation tests; this model
does not prove that a custom native implementation returns. The new direct
`TestStreamHalfCloseAfterTerminalReceive` and gated
`TestStreamHalfCloseRacesTerminalRelease` fail on the original owner and pass with
classified admission. Live
pre-canceled/no-I/O and custom native-result controls preserve existing admission
behavior; send-overlap and original-parent cancellation controls test the other
runtime boundaries. `TestRecursiveValidationGenerated` is the unchanged mounted
regression.
