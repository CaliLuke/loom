# #976 runtime owner

Adopted owner contract for `http/exchange_writer.go`, `http/sse_writer.go`, `http/response_metadata.go` and `jsonrpc/stream_server.go`. It was first proposed against prospective parent `e94b47128d3d936c97f70f7753f419c9d7b25ff4`; selector, DSL and importer work have their own owners.

## Shared exchange owner (M5)

The accepted `ExchangeWriter`, `FinalKind`, `Send`, `BeginFinal` and `HandlerLifecycle.Exchange` API is implemented. Include an explicit `FinalRedirect` kind alongside result, error, SSE open, upgrade, file and raw. `BeginFinal` closes informational admission under the exchange mutex before any final header work; it does not imply physical commitment and does not dispatch observations. `Send` owns snapshot → write → restore under that same mutex. It never calls back into the SSE owner. Use existing CaptureResponse for physical status/bytes and retain existing header rollback/error fallback before physical commit.

Placement follows [accepted design section 7](../../response_selection_design.md#7-exchange-writer-ownership-and-informational-phase): lifecycle EncodeResponse before header clone; HandlerFailed before routing; encodeError before headers; raw before first-byte peek; file before serving; lazy upgrade inside sync.Once before header preparation; redirect before http.Redirect; handler End closes informational admission only. No separate protocol writer/capture or duplicated request observer. The phase model's lock spans final preparation/commit, whereas actual BeginFinal may unlock after permanently closing admission: subsequent refusal commutes with final header work. It does not model two concurrent final encoders as valid application behavior.

## Single SSE owner (M7)

One writer mutex owns admission, physical `started`, first transport cause and handshake snapshot. Keep the minimum runtime state required by the operation table: the existing `closed`, `started` and `failErr` ownership plus the handshake snapshot can express these distinctions through centralized transitions. The model's winner/history labels (`live`, successful completion, suppression, claimed failure, I/O/terminal-operation failure, cleanup) are specification observations, not a required runtime enum or additional stored categories. Physical started remains distinct from closed admission. Do not add wrapper-level terminal state. Existing `ClaimFailure` is retained, not reimplemented in a generated wrapper. Its admission bool must still reflect the physical started state and preserve prior closure/cause.

Public runtime declarations:

- `SSEHandshake` holds resolved `Status int` and `Header http.Header`.
- `SSEStreamOption`, `WithSSEHandshake(func(context.Context) (SSEHandshake, error))`, and `WithSSEExchange(*ExchangeWriter)` extend NewSSEStreamWriter. With no handshake option the existing literal-200 behavior remains for JSON-RPC construction. The resolver is pure generated contract validation, invoked under writer ownership before the first handshake; it must not reenter the writer.
- `SetHandshake(prepare func() (SSEHandshake, error)) error` performs admission check, validation/conversion and snapshot replacement under the same mutex. Reject open or terminal state with `ErrSSEResponseCommitted`; failed preparation leaves the previous snapshot unchanged. Clone headers before storing. This API does not invent a new runtime cancellation check for metadata authoring; the generated `SetResponse(ctx, meta)` can use its context in its contract conversion callback. Mutable caller metadata is copied/validated within this admission scope.
- `FinalizeSuccess(ctx context.Context) error` deliberately publishes a successful zero-event handshake and closes, or preserves an existing terminal outcome as a nil no-op.
- `Suppress() error` is explicit protocol suppression: first admitted call closes silently and returns nil; repeat/closed admission returns existing `closedErr`. It does not inspect cancellation, matching the old nil Complete operation being replaced in JSON-RPC.
- Keep `Complete(ctx, prepare)`, `Close()`, `End()`, `ClaimFailure()` and existing Open/write methods. They derive behavior from the same private terminal owner, not separate state machines. Close uses the request context for a new successful terminal operation. End remains cleanup-only and preserves the original panic.

The generated SetResponse method converts the complete metadata into a validated handshake through SetHandshake. No Started()-then-store sequence, wrapper metadata lock, lock inversion, or post-open mutable closure is permitted. The initial resolver is immutable configuration; a successful setter installs a detached replacement snapshot. Generated event-view state is sampled under its existing view mutex before writer I/O, then that mutex is released. Lock order is SSE → exchange only; exchange never acquires SSE/view locks. Observations and failure callbacks run after both write locks have been released. Complete's message preparation remains under the SSE lock and may not reenter it. The bounded models do not prove callback purity or these concrete lock-order obligations.

## Exact operation/result table

`closedErr` means the existing ErrSSEStreamClosed error, joined with the retained transport cause when one exists. A terminal outcome is never overwritten. The model's `ok-noop` tag means nil, not an unspecified later choice.

| Operation/state | Result | State / I/O |
| --- | --- | --- |
| ClaimFailure, any context/state | prior started bool | Close live admission; retain terminal winner, cause and started; no headers/flush/context inspection/observer dispatch. |
| Open/event, cancelled context (even already closed) | context error (per-call before request context) | No new publication/cause. Existing invalid-comment validation remains before writer admission. |
| Open/event, live with invalid handshake | validation error | No publication; stays usable. Context checks precede handshake resolution. |
| Open/event, uncancelled terminal | closedErr | No work; existing cause remains accessible. |
| Complete with message, already terminal | closedErr | Preparation never runs, regardless of cancellation. |
| Complete message preparation fails while live | exact preparation error | Remains live, unchanged physical state/cause/snapshot; preparation precedes context checks as today. |
| Complete with nil, or successfully prepared message, while live | operation's context/handshake/write/flush error, or nil | Shared first-open validation/publication; then terminal even if this newly admitted operation fails. No guessed status or success headers on context/handshake failure. Stored transport cause is set only by the existing I/O failure owner. |
| Close / FinalizeSuccess, live | same newly admitted terminal-operation error or nil | No message preparation; context precedes handshake validation. Close uses request context. On successful publication close once; on context/handshake failure terminal failure with no success headers. |
| Close / FinalizeSuccess, any existing terminal outcome including failure/suppression/cleanup | nil | Idempotent no-op: no reopening, replacement winner, new observation, or spurious second HTTP error. Retain prior transport cause for ordinary rejected writes/Complete. This preserves current Close's nil-after-close behavior. |
| Suppress, live (including cancelled context) | nil | Silent first terminal admission; no handshake or context check. Caller may emit Suppressed once after nil. |
| Suppress, terminal | closedErr | No new suppression observation, even for a previously suppressed stream. |
| SetHandshake, opened or terminal | ErrSSEResponseCommitted | No callback/store; never changes selected handshake. |
| SetHandshake, live/unopened, invalid callback result | exact validation error | No snapshot replacement or terminal mutation. |
| End, any outcome / panic unwinding | existing cleanup/panic behavior | Never opens; closes live admission, retains previous outcome/cause, records and drains observations under existing rules. Original handler panic wins. |

These distinctions follow actual `sse_writer.go:130–160,181–193,227–295,355–361` and `jsonrpc/stream_server.go:248–260`, plus the successful publication/idempotence rules in [SSE Handshake Status and Completion](../../../../../../docs/http-guide.md#sse-handshake-status-and-completion). In the pre-#976 baseline, Close always returned nil and nil Complete skipped context. Complete still checks closedErr before preparation. A retained cause need not be re-returned by an idempotent Close to remain observable. JSON-RPC must emit Suppressed only after the first successful Suppress, never after rejected repeat admission.

Generated HTTP nil-error return invokes FinalizeSuccess before deferred End; its error follows ordinary lifecycle failure routing with current ClaimFailure. Both dedicated and mixed-result SSE callers change. JSON-RPC notification completion calls Suppress, then observes suppression only after nil. Message completion still uses Complete. No runtime behavior depends on whether a consumer installed a handshake option.

## Implementation correspondence tests

`TestSSEHandshakeOpenTriggers`, `TestSSEHandshakeSnapshot`, `TestSSEHandshakeAdmissionRace`, `TestSSEHandshakeFailureAdmission`, `TestSSETerminalSuccessAdmission`, `TestSSETerminalCancellation`, `TestSSESuppressionAdmission`, `TestSSETerminalObserverOutsideOwner`, `TestSSEEventResponse` and `TestExchangeWriterConcurrentSendDuringFinalStart` cover exact 201/status-selected handshakes and open triggers, required/missing headers, Close/Finalize idempotence and races, explicit suppression, cleanup/panic, event-response independence and redirect/interim ordering. They distinguish:

1. Failed Complete message preparation leaves the stream usable and a later completion can succeed.
2. Fresh Close/Finalize/nil Complete with missing handshake returns the validation error, publishes no success headers and closes; metadata validation failure in SetHandshake instead preserves the previous live snapshot. Open's pre-commit handshake failure remains retryable.
3. Cancellation after successful Complete preparation closes but does not fabricate an I/O cause; pre-existing terminal Complete skips its callback; Open still reports context first.
4. Repeated suppression returns closedErr and emits no second observation; repeated Close/Finalize returns nil and cannot overwrite suppression/claim/I/O failure.
5. SetResponse racing first Open/ClaimFailure/Close produces one atomic admitted snapshot or ErrSSEResponseCommitted. Include validation failure preserving the previous value; no wrapper check-then-act.
6. Actual observer reentrancy, original-panic precedence, per-call versus request cancellation, touched/untouched callback failure, partial write/flush failure, and no lost/double close/failure observations remain runtime/race obligations, not consequences of TLC success.

The SSE event response adapter (`SSEEventResponse`) and the per-Do informational owner (`ObserveInformational`) are separate owners and are not modeled here.
