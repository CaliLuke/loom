# SSE handler failure admission

`SSEHandlerFailure.tla` is a bounded design model of one generated HTTP SSE
handler whose endpoint returns an error while a stream caller may still use the
shared `SSEStreamWriter`. It compares the current routing with the candidate
admission-only `SSEStreamWriter.ClaimFailure`.

- **Current.** The generated handler passes `stream.started()` (a read of
  `SSEStreamWriter.Started`) to `HandlerLifecycle.HandlerFailed`. That method then
  reads capture commitment and encodes the ordinary error. Nothing closes writer
  admission until the deferred `End`, so an admitted writer operation can append
  SSE bytes after the error response.
- **Candidate.** `ClaimFailure` takes the writer lock, snapshots `started`, sets
  `closed = true`, releases the lock and returns the snapshot. It checks no
  context, changes no `failErr` or `openErr`, records no observation, drains
  nothing and writes nothing. It replaces only the `Started` read: the
  lifecycle's capture check, error encoding, fallback and deferred `End` are
  unchanged. `End` stays cleanup only: it still unconditionally sets `closed`,
  calls `RecordClose` and drains, with no early return.

This model implements nothing and does not cover #976 status selection,
publishing `Close`, successful zero-event finalization or JSON-RPC protocol
suppression. TLC is a design aid here. It is not part of `make`, hooks, CI or
release eligibility.

## Actors and bounds

- One handler: service error, decision, capture read, error encoding with one
  fallback, deferred `End`.
- One stream caller with at most two operations from `Ops`: `open`, `write`
  (`Send`/`SendWithContext`/`SendComment`, with a pre-lock stage), `complete`
  (`Complete` with a message), `completeNil` and `close`.
- One writer, seeded from `Seeds` with a prior state and its wire effects.
- The wire: attempted final statuses in order (the first one wins) and body
  tokens `sse`/`err`.

### Seeded prior states

Each seed sets the writer and the wire together:

| Seed | started | closed | failErr | attempts | body | Go states represented |
| --- | --- | --- | --- | --- | --- | --- |
| `fresh` | no | no | none | – | – | new writer; a failed `Complete` preparation; a failed pre-lock encoding |
| `opened` | yes | no | none | 200 | – | `Open`; an untouched `WriteEvent` callback error (`initHeaders` runs first) |
| `streamed` | yes | no | none | 200 | sse | a written frame; post-frame cancellation (no flush, `fail` or close) |
| `flushFailed` | yes | yes | io | 200 | – | `Open` flush failure |
| `writeFailed` | yes | yes | io | 200 | sse | touched write or post-frame flush failure |
| `silentClosed` | no | yes | none | – | – | `Close`; `Complete(nil)`; `Complete` with a context cancelled before headers |
| `completed` | yes | yes | none | 200 | sse | successful `Complete` with a message |

## Correspondence

| Model action | Go owner |
| --- | --- |
| `StartedRead` | generated `stream.started()` → `SSEStreamWriter.Started` (`http/sse_writer.go`); removed by the change |
| `Claim` | `SSEStreamWriter.ClaimFailure`, called as the `committed` argument in both generated HTTP SSE failure branches (`http/codegen/template_sources.go`: the mixed-result SSE branch and the dedicated SSE branch) |
| `CaptureRead` | `HandlerLifecycle.HandlerFailed`: `committed \|\| l.responseCommitted()` → `RequestObserver.ResponseCommitted` → `ResponseCapture.StatusCode` (first final status, compare-and-swap) |
| `EncodeFirst`, `CheckCommitted`, `EncodeFallback` | `HandlerLifecycle.encodeError`: first encoding, commitment check, header restore and fallback encoding, `handleFailure` |
| `End` | deferred `SSEStreamWriter.End` → `close` |
| `Prepare` | generated `SendWithContext` (`EncodeSSEData` before `WriteEvent`); `SendComment` line-break validation |
| `Admit` | `Open` and `writeEventLocked` (`checkContext`, then `closedErr`); `Complete` (`closedErr`) |
| `PublishOpen` | `Open` after its guards: `initHeaders`, `flush`, `fail` |
| `Frame` / `PublishWrite` | `writeEventLocked` after its guards: `initHeaders`, `sseFrameWriter.touched`, `fail`, post-frame `checkContext`, `flush` |
| `PublishComplete` | `Complete`: `prepare`, `writeEventLocked`, then `closed = true` |
| `CloseOp` | `Close` / `close`; `Complete(nil)` |
| wire `attempts` | net/http and `httptest.ResponseRecorder` keep the first final `WriteHeader` and still accept later body writes |

`Admit` through `Publish*` is one Go critical section. `StartedRead`, `Claim`,
`SplitClose`, `SecondStartedRead`, `Reclaim` and `End` need the lock to be free,
so none of them runs in the middle of a held operation. `CaptureRead` and the
encoder steps take no writer lock.

`EncodeFirst` atomically writes the error bytes and advances to `cleanup`.
That phase abstracts the interval after those bytes and before deferred `End`;
the actual Go regression invokes the retained comment inside the encoder
callback. The model reproduces the wire/admission ordering, not the callback
program counter or exact callback timing.

## Configurations and expected outcomes

The table lists **expected** outcomes. Record the actual TLC result for each
configuration after running it. A parse error, semantic error or deadlock is a
model error, not an expected negative. Terminal stuttering is a legal behaviour.

| Configuration | Design | Expected | Classification |
| --- | --- | --- | --- |
| `current-witness` | current | `ObservedWitnessAbsent` violated: attempts `<<400, 200>>`, body `<<err, sse>>`, comment invoked after the encoder wrote, writer neither closed nor failed | matches the executed Go seam defect |
| `current-inflight` | current | `InflightWitnessAbsent` violated: an operation invoked during the service publishes after the ordinary decision | model and source evidence only; not executed in Go |
| `current-silent` | current | pass | control: current silent `Close`/`Complete(nil)` are not body-mix witnesses |
| `claim` | candidate | pass: all listed invariants and properties | candidate |
| `claim-fallback` | candidate | `FallbackUnreachable` violated | reachability witness: the claim doesn't block the ordinary owner's precommit fallback |
| `second-started` | rejected | `NoMixedBody` violated | wrong candidate: a later `Started` read is still check-then-act |
| `split-claim` | rejected | `StableClaim` violated: `Open` runs between the snapshot and the close | wrong candidate, contract only |
| `split-claim-routing` | rejected | pass | shows split-claim routes correctly because capture sees the intervening `Open`; not a body mix |
| `cause-overwrite` | rejected | action property `CausePreserved` violated from a seeded I/O failure | wrong candidate |

### Invariant and property scope

| Name | Claim and scope |
| --- | --- |
| `WireConsistent` | Writer flags agree with their own wire effects, including seeded prior winners. |
| `NoMixedBody` | No response carries both ordinary-error bytes and SSE bytes. Bytes written before a claim are never rolled back; an opened prior state routes late instead. |
| `SingleFinalAttempt` | Local to this handler and writer. It is not a global protocol claim. |
| `OrdinaryNeverOpens`, `LateOnlyIfOpened` | `LateOnlyIfOpened` is valid only because no response writer other than the SSE writer acts before routing. `HandlerLifecycle` also accepts capture commitment from any writer. |
| `StableClaim`, `ReclaimStable` | The returned snapshot is the writer's `started` flag and remains so; a repeated claim returns the same value. |
| `NoHandlerCause`, `CausePreserved` | The claim never records the handler error and never replaces an earlier cause. |
| `ClosedMonotonic`, `StartedMonotonic` | These flags never revert. |
| `NoPublicationAfterClaim` | After the claim, the writer adds no 200 attempt and no SSE token. The returned error for a refused operation is not modelled: context and invalid-comment errors keep their precedence. |

## Reproduce (opt-in)

Requirements: Java (verified family: Amazon Corretto 25.0.5) and `tla2tools.jar`
1.7.4 (TLC 2.19, SHA-256
`936a262061c914694dfd669a543be24573c45d5aa0ff20a8b96b23d01e050e88`). Set `JAVA`
and `TLA_JAR`. From this directory:

```sh
for cfg in *.cfg; do
  "$JAVA" -XX:+UseParallelGC -cp "$TLA_JAR" tlc2.TLC -workers 1 \
    -config "$cfg" -metadir "/tmp/loom-sse-handler-failure-tlc/${cfg%.cfg}" \
    SSEHandlerFailure.tla
done
```

Passing configurations should exit 0. Each negative configuration should exit
with a safety violation that names its expected invariant or property. Record
the exit code and the reported name. Don't use `-deadlock`: the `Terminated`
action makes the final states stutter, so a reported deadlock is a model error.
Delete the metadirectory after review. Don't commit TLC output.

## Assumptions and limits

- **Response writes.** Each `ResponseWriter` call is atomic. Overlapping calls
  from the encoder and a writer operation would be a data race; that race is
  outside the model, and the executed witness doesn't need it.
- **Encoder.** The first encoding either commits 400 with an error body, or fails
  before any commit. The fallback either commits 500 or fails, which routes to
  `handleFailure`. Encoding failures after commitment route to `handleFailure`
  and are omitted.
- **Cancellation.**
  - It can refuse `Open`/`WriteEvent` admission at any time, with no state
    change. This over-approximates: later operations may still proceed with
    fresh per-call contexts.
  - Post-frame cancellation has the same writer state as `ok`.
  - Mid-operation deadline cancellation appears as the touched, untouched or
    flush-failure outcomes the Go code produces.
  - `Complete` with a cancelled context closes silently.
- **Generated HTTP surface.** Generated HTTP SSE streams expose `Open`,
  `SendComment`, `Send`/`SendWithContext` and `Close`. `complete` and
  `completeNil` are included as operations on the shared writer to check earlier
  winners and preparation usability. JSON-RPC routing through
  `CompleteSSEStream` is not modelled and is unchanged.
- **Out of scope.** `StreamObserver` recording and delivery, `End`'s panic
  precedence, observer reentrancy and callback timing are not in this model.
  `ClaimFailure` has no observation operation, so the transition proof doesn't
  need them. Go tests carry them, as listed below.
- **Bounds.** The model doesn't cover service-method return lifetime or any
  right to retain the stream. It checks safety for every interleaving of the
  bounded actors.

## Implementation regression coverage

- `http`, `TestSSEHandlerFailureClaimsBeforeRetainedWrite`: the deterministic
  lifecycle/error-prefix regression, with `stream.Started()` replaced by
  `stream.ClaimFailure()`. Assert:
  - status 400, the problem `Content-Type` and the exact prefix body;
  - the retained comment is refused;
  - no SSE headers and no flush from the writer;
  - encoded 1, handled 0.
- `http`, `TestSSEHandlerFailureEncoderFallback`: the first encoder attempt
  fails before commitment and the fallback commits. Exact body and status, no
  writer bytes, and the routing counts for both ordinary encoding and the error
  handler.
- `http`, `TestSSEStreamWriterClaimFailure`, table-driven over: fresh; opened;
  previous I/O or flush failure (the cause is kept); silent `Close`; prior
  `Complete`; repeated claim. Assert a stable result and the original cause
  using uncancelled calls. The preparation-failure and untouched-callback rows
  check their distinct started expectations.
- `TestSSEStreamWriterClaimFailurePreservesContextPrecedence` covers cancelled
  request context and invalid-comment precedence without recording a cause.
- `TestSSEFailureClaimAfterPostFrameCancellation` checks the started, buffered
  frame with no terminal I/O cause, using the existing deadline-capable writer.
  A cancellation hook on a writer without deadline support can instead fail the
  touched operation; that is a different prior state.
- `http/sse_failure_observation_test.go` and existing
  `http/stream_observation_readiness_test.go`:
  - the claim emits no StreamClose;
  - the handler failure and error handler run before the deferred close in the
    single-handler trace;
  - close-observer reentrancy doesn't deadlock;
  - `TestSSEHandlerEndPanicPrecedence` still holds.
- `TestSSEHandlerFailureWaitsForAdmittedWrite` is channel-ordered: an in-flight
  operation holds the writer,
  the claim waits and returns true, and the lifecycle routes late. Run the
  relevant runtime tests under `-race`.
- `http/codegen/sse_server_test.go`: both generated branches call
  `stream.writer.ClaimFailure()`. Remove the now-unused generated private
  `started()` pass-through in the same fix. The public runtime `Started` method
  remains unchanged.
- `http/integration_tests/fixtures/ticktock`: a retained generated `SendComment`
  and a typed `Send` are refused after an endpoint error by
  `TestGeneratedSSEHandlerFailureClaimsAdmission`. Compile after
  generation through `TestHTTPSSEFixtureRegeneratesAndBuilds`.

## Recorded qualification

TLC 2.19 / tla2tools 1.7.4 with Corretto 25.0.5 checked these configurations
with one worker. All expected outcomes above were observed; no parse error,
semantic error, deadlock or timeout counted as a result. Negative searches stop
at their named witness and are not exhaustive proofs.

| Configuration | Exit | Distinct states | Observed result |
| --- | ---: | ---: | --- |
| `current-witness` | 12 | 253 | Invariant ObservedWitnessAbsent is violated. |
| `claim` | 0 | 4411 | pass |
| `current-inflight` | 12 | 388 | Invariant InflightWitnessAbsent is violated. |
| `current-silent` | 0 | 373 | pass |
| `claim-fallback` | 12 | 154 | Invariant FallbackUnreachable is violated. |
| `second-started` | 12 | 404 | Invariant NoMixedBody is violated. |
| `split-claim` | 12 | 73 | Invariant StableClaim is violated. |
| `split-claim-routing` | 0 | 1944 | pass |
| `cause-overwrite` | 13 | 4 | Action property CausePreserved is violated. |

The candidate completed with an empty queue and checked nine state invariants
and four action properties across all seven seeded prior states and five
operation kinds, at most two stream calls. The fallback negative demonstrates
reachability of the legitimate 500 fallback; it is not a defect. These results
do not prove observer callback, panic, reentrancy or cleanup-delivery behavior;
the Go obligations above remain required.
