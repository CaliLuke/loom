# SSE terminal ownership and redirect exploration

Companion to SSEOpen. Its "current" mode and current-source notes record the
pre-#976 baseline that the model checked; the adopted design is implemented.
[Runtime owner](RUNTIME-OWNER.md) gives the exact operation results and their
regressions. No model execution enters default gates.

SSETerminal models two actors, each making at most one operation, with one shared
writer owner. It separates live/terminal intent, physical opened status, retained
I/O cause, metadata snapshot, publication and observer-delivery steps. Seeds are
fresh, opened, I/O-failed, failure-claimed, suppressed and completed. Context is a
fixed cancellation Boolean; handshake metadata is absent or selected 201, with
200 in already-open/current states. It models a failed message preparation and
an abstract touched I/O failure, not arbitrary encoder or socket execution.

Current mode includes actual ClaimFailure. Close and nil Complete close without
publication. Its finalize operation is an abstract successful handler return
followed by existing silent End, not a claim that the current Go API already has
FinalizeSuccess. The terminal labels identify the first closing operation's
intent; they are not existing Go fields. All three current publication negatives
finish with opened=false, no wire status and a successful operation. This is the
accepted #976 publication obligation missing from current behavior, not a
regression introduced by the committed failure-claim correction.

Candidate mode shares one terminal winner across successful completion,
suppression, failure claim and cleanup. It validates/stores metadata under the
same owner as first open. Complete checks terminal admission before preparation;
failed message preparation leaves the stream usable. Open/event cancellation
still precedes closed admission. Close/Finalize preserve any existing terminal
outcome as nil; first admitted suppression succeeds, repeats fail. Selection
failure on Open remains live, while selection/context failure of a newly
admitted terminal operation closes without publishing success. ClaimFailure
performs no I/O, cancellation check or observation and retains prior cause.

The original 11 candidate state invariants cover types, one publication, successful
publication, claim preservation, prior cause, consistency with the structurally
assumed observation outside the calling actor's lock, no claim callback, metadata admission, selected publication,
context precedence and selection-failure effects. Five step properties cover
cause stability, terminal stability, silent cleanup/suppression, usable failed
preparation and no claim-time observation. Observations may run while a
*different* actor holds the writer; globally forbidding that was an invalid
model assertion and has been corrected. Release-before-Observe and the one-operation
actor bound structurally guarantee outside-calling-owner delivery. Thus
CallbacksOutsideOwner and the heldBy conjunct of SuppressionMarkersAuthorized
cannot independently detect a callback executed under that actor's lock. Concrete
Go/race correspondence tests must establish this obligation. Callbacks are not
arbitrary code here.

## Opt-in reproduction

Use the pinned Corretto 25.0.5 and tla2tools 1.7.4 recorded in README. For each
configuration run the existing TLC pattern, substituting SSETerminal.tla:

```sh
"$JAVA" -Xmx1536m -XX:+UseParallelGC -cp "$TLA_JAR" tlc2.TLC -workers 1 \
  -config terminal-checked.cfg -metadir /tmp/loom-976-terminal SSETerminal.tla
```

Observed finite SSETerminal results. After adding the missing-marker policy,
the checked candidate was rerun with the same 78,430 generated / 65,027 distinct
states and an empty queue. The other policies retain their earlier results:
the added guard is true for those policies and does not change their transitions.

| Configuration | Exit | Generated / distinct | Result |
| --- | ---: | ---: | --- |
| terminal-current-close | 12 | 8 / 8 | SuccessfulPublication counterexample: silent Close |
| terminal-current-nil | 12 | 8 / 8 | SuccessfulPublication counterexample: nil completion |
| terminal-current-return | 12 | 8 / 8 | SuccessfulPublication counterexample: successful return + cleanup |
| terminal-current-preservation | 0 | 20,764 / 17,447 | Current owner invariants pass; publication obligation deliberately omitted |
| terminal-checked | 0 | 78,430 / 65,027 | Candidate invariants/step properties pass, queue empty |
| terminal-late-metadata | 12 | 36 / 36 | MetadataAdmission rejects a setter bypassing first-open/terminal admission |
| terminal-overwrite-winner | 13 | 220 / 220 | TerminalStable rejects replacement of an earlier outcome |
| terminal-preparation-recovery | 12 | 31 / 31 | Intended reachability: failed preparation followed by successful completion |
| terminal-suppression-winner | 12 | 31 / 31 | Intended reachability: suppression wins, later Close is nil/no publication |
| terminal-handshake-failure | 12 | 8 / 8 | Intended reachability: fresh successful-terminal intent fails handshake with no publication |
| terminal-wrong-suppression-wrapper | 12 | 29 / 26 | SuppressionMarkersAuthorized rejects observation after refused suppression |
| terminal-missing-suppression-marker | 12 | 8 / 8 | SuppressionCompletedObservation: successful suppression completes observation with no marker |
| terminal-duplicate-suppression-marker | 12 | 34 / 30 | SuppressionMarkersUnique rejects two markers from unconditional wrapper observation |
| terminal-suppression-repeat | 12 | 34 / 30 | First suppression succeeds, repeat is closed, one marker and no wire status |
| terminal-suppression-close | 12 | 115 / 99 | Suppression wins; Close is nil and cannot open |
| terminal-suppression-finalize | 12 | 115 / 99 | Suppression wins; finalization is nil and cannot open |
| terminal-suppression-complete | 12 | 115 / 99 | Suppression wins; message completion is closed |
| terminal-suppression-canceled | 12 | 8 / 8 | Suppression ignores cancellation and emits its marker after admission |
| terminal-suppression-pending | 12 | 6 / 6 | Admitted suppression can await observation with no marker yet |
| terminal-suppression-loses | 12 | 120 / 104 | Message completion wins; rejected suppression produces no marker |

Reachability controls deliberately assert absence of a valid state and expect a
counterexample. They are not product failures. Negative searches stop at their
named witness; no exhaustive negative-state proof is claimed. Syntax/state-
specification failures during model construction are excluded from evidence.

## Scope and limits

SSEOpen transitions remain unchanged and own the fuller literal/status-mode/required-header
matrix; this companion assumes a prepared valid status or absence. It does not
implement another selector. Source-to-model correspondence must still be checked
by the plan's Go/race/generated tests: exact error identities, request versus
per-call cancellation, context changes during I/O, flush/partial-write results,
failed metadata/header conversion, touched versus untouched callbacks, observer
reentrancy and original-panic precedence. Neither fairness nor eventual callback
delivery is proved; at most two operations do not prove arbitrary executions.
The mutex is one modeled owner, not a proof of Go lock acquisition or absence of
inversions between concrete SSE, exchange and view locks.

ResponsePhases adds redirect as status302/Location, alongside the four existing
final-kind abstractions. Its deliberately unlocked redirect is a defective
candidate design, not the adopted `ExchangeWriter`, which closes admission for
redirect like every other final start.
Raw/file kinds retain correspondence to final start and need their own runtime
placement checks. Automatic100/client budget and other existing model bounds
remain unchanged. No model/tool belongs in normal test/build/commit/push/CI gates.

## Suppression observation and source correspondence

The added histories record admitted suppression actors and observed protocol
markers. They are specification-only evidence, not a new runtime enum, flag or
retained collection. Four additional invariants require at most one admission,
markers authorized by a successful admission, at most one marker, and exactly
one marker for a successful suppression **when that actor reaches completed
observation** (zero for a rejected actor). The wrong-wrapper mutant leaves all
admission transitions intact and observes even a refused Suppress. This exposes
both an unauthorized marker and a duplicate after two callers. It is a defective
prospective wrapper, not a claim that the current JSON-RPC wrapper does this.
SuppressionAdmissionUnique and SuppressionMarkersUnique are derived consistency
checks, not independently mutation-proven obligations: monotone terminal
admission gives one winner, and authorized delivery plus one Observe per actor
gives one marker. The duplicate mutant violates authorization first.

The missing-suppression-marker policy instead retains successful Suppress and
completed Observe but omits the marker. Its dedicated config checks
SuppressionCompletedObservation, exposing a wrapper that drops Suppressed after
successful suppression. It changes only marker recording; all prior policy
values reduce to the same prior transition formulas. Both the checked candidate
and this negative were executed against the changed transition source; their
expected outcomes are in the table above.

There is deliberately no fairness or eventual-observation assumption. Release
and Observe are distinct steps; the pending-observation reach has an admitted
suppression but no marker. Observe atomically records a completed callback; it
does not prove callback termination, panic behavior, reentrancy, or exactly-once
external side effects if a callback itself fails. A callback may run while the
other actor owns the lock. Excluding observation under its own actor's write
ownership is a structural model assumption, not an independently tested lock
property; concrete Go/race tests must establish it.

Pre-#976 source correspondence was `http/sse_writer.go`: Complete checks closed
admission before preparation, failed preparation leaves the stream usable, nil
preparation closes silently; Close is silent and always nil; End is cleanup;
ClaimFailure closes admission without publication or observation and preserves
physical started/cause. `jsonrpc/stream_server.go` then called Complete(nil)
for notifications and calls Suppressed only after success. The adopted
wrapper replaces only that terminal call with explicit Suppress and retains
this success-only observation guard; `TestSSESuppressionAdmission` checks the
writer's single admission. `transport.StreamObserver.Suppressed`
does not deduplicate events for the caller. The current-mode logical suppress
operation would abstract that existing guarded notification completion; it does
not claim a current Go Suppress method exists. All four current-mode configs
exclude the suppress operation and suppressed seed. Their four suppression
invariants therefore hold vacuously, and no current-mode run establishes the
notification guard: that correspondence is source-only. Those unchanged
current configurations were not rerun for the missing-marker policy.

SSEOpen's logical `end` trigger means successful FinalizeSuccess before deferred
cleanup, never Go End. Explicit Close shares its logical complete/end opening
transition; SSETerminal owns Close-specific admission and return behavior. Only
correspondence comments changed in SSEOpen. Existing
ResponsePhases redirect evidence (797,800 checked distinct states and the
13,546-state unlocked-redirect counterexample) is reused, not rerun here.
Selector and fuller SSEOpen header/status semantics likewise remain under their
existing owners. The accepted runtime API/result table is unchanged; a fresh
Close/finalization handshake failure is terminal without success publication,
whereas failed Complete preparation remains usable. Exact Go behavior still
requires the bounded runtime/race and generated-caller correspondence tests.
