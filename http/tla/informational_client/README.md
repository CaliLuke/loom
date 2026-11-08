# Informational client per-Do ownership (#976, M6)

Opt-in finite exploration of the implemented per-Do owner,
`http.ObserveInformational` in `http/informational_client.go`. No public policy
change or CI requirement follows from these checks.
[Informational Responses](../../../docs/http-guide.md#informational-responses)
owns the public semantics. Existing selector/ResponsePhases models and
ClaimFailure are separate owners.

## Contract and pinned source

One ObserveInformational owner belongs to one Do invocation, never a reusable
context. GotFirstResponseByte resets exchange counters only, before the first
interim in each native response exchange. WroteHeaders never resets them. Every non-100 interim
counts, even undeclared codes; 100 has its own budget and a declared 100 can
invoke an observer. After the first Loom error is retained, later Loom hook
calls return that error without new admission, counting, decoding or observer invocation. The
existing monotonic stored error is the guard, not a new state enum. Store the
first Loom error before canceling. Go composes the caller hook afterward; Loom must not manually invoke it again. Finish
reads the synchronized first error after Do, then returns errors in order:
Loom error, available Do error, available returned-response Close error.
A successful response can race cancellation: nil Do error does not waive
closing that response when the owner already has an observer error.
Ordinary success retains cancellation until EOF or body Close.

Pinned Go1.27.3 sources establish:

- `net/http/httptrace/trace.go:235–245`: compose2to1 invokes new then old,
  returning the old result when present. This is the existing
  [Go #76669](https://github.com/golang/go/issues/76669) workaround contract;
  no dependency development or removal is authorized by this model.
- `net/http/transport.go:3109–3139`: Do may return on cancellation while the
  composed caller hook is unfinished, or return a response racing cancellation.
- `net/http/transport.go:2716–2728`: a nil composed hook result permits reading
  another buffered interim, including after Do/finish returns.
- `net/http/transport.go:2690–2728`: GotFirstResponseByte runs before the
  HTTP/1 readResponse internal 1xx loop, once per readResponse invocation.
- `net/http/internal/http2/transport.go:2151–2169,2252–2288`: firstByte guards
  GotFirstResponseByte before handleResponse/1xx delivery. Informational frames
  reset pastHeaders only, not firstByte. Unlike WroteHeaders, this callback is
  ordered on the response read path before every admitted interim.
- HTTP/2 `transport.go:1128–1148` creates a fresh stream, including the direct
  ClientConn path (`net/http/clientconn.go:254–273`). Retry cloning at
  `internal/http2/api.go:64–81` preserves context and omits stream state;
  `transport.go:506–522` uses that clone. Redirects preserve request context
  at `net/http/client.go:676–685`; each new response gets the same composed
  trace at its own first-byte boundary.
- HTTP/2 `transport.go:1433–1434,1613` invokes WroteHeaders after flushing;
  its separate read loop can deliver interims first. Request-write completion
  is therefore not an admissible counter reset boundary. GotConn is not used;
  direct ClientConn bypasses connection acquisition callbacks.

Exact source copies and the [accepted design](../../codegen/internal/transportir/response_selection_design.md) are
pinned in the isolated evidence packet. The original
`/tmp/loom-976-trace-proof` raw directory is missing;
only its retained design summary was available. No new Go transport observation
is claimed. The old model's fifteen results remain historical evidence of its
old assumptions, not verification of this corrected transition relation.

## Concurrent progress and synchronization abstraction

The model uses two independent program counters: hook progress (`pc`) and
Do/finish progress (`doPC`). Hook captures admission and first error atomically,
then unlocks; CancelHook follows. Admission is guarded by the stored error;
history records delivery separately from admission/observer invocation as
proof observations. After cancellation, Do can return during the
caller hook or before a defective delayed store. Finish independently reads
the owner. CallerHook and bounded buffered Hook steps may complete afterward.
They cannot mutate the completed snapshot, returned error sequence or body
ownership. The abort invariant checks failures observed **at finish**, not
future callbacks; checking every future failure against an already returned
result would invent a different contract.

The `cancel-before-store` mutant moves storage after cancellation. A real
interleaving reaches Finish before that store and loses the observer error in
the result. `AbortPreserved` catches this, rather than a redundant field-order
predicate. `StoreBeforeCancel` was removed. Named reach controls cover caller
completion after Do, a hook after finish with the first result intact, and
successful Do racing an observer abort followed by response closure.

Response presence is chosen at DoReturns, not freely at initialization.
Nil Do error requires a returned response/body; a nonnil Do error may have one
or none. A bodyless successful HTTP response still has a closeable body, such
as http.NoBody. Finish closes a returned response whenever its synchronized
observer-error snapshot is nonempty, independent of Do error. A no-observer
Do error follows the ordinary error branch of the
[accepted design](../../codegen/internal/transportir/response_selection_design.md).

Hooks are serialized and observer execution is abstracted by a returned value.
Atomic critical sections assume a correctly implemented lock. EOF and CloseEnd
atomically test/set the shared once bit: this is an explicit abstraction of a
correct once primitive, **not** a proof of an unsynchronized check-then-set.
`double-cancel` tests omission of that guard, not a low-level memory race.
Go `-race` and controlled concurrent EOF/Close tests remain mandatory
implementation correspondence. CloseBegin/CloseEnd permit EOF to interleave
with the inner close. Repeated underlying Close, panics, actual deadlocks,
HTTP/2 frames, external cancellation, transport timeouts (including
ResponseHeaderTimeout), body bytes and scheduling/liveness are
outside these finite safety checks. No callback-termination or unread-body
release guarantee is proved. Calling cancel on abort/finish is idempotent;
body cancel-once is not a count of all raw CancelFunc calls.

## Finite bounds and partitions

100 limit is1; non-100 Limit is2 (runtime8). Codes100/103 are declared,
102 is undeclared but counted. Caller hook is absent, nil-returning or failing;
stream/bodyless responses and nil/failing Close are included. Observer result
is nil or symbolic observerA/observerB. The positive partitions are:

- `checked`: one exchange, up to three deliveries, both observer errors.
  Distinct budgets admit two non-100s and reject the third; a second100 rejects.
- `checked-reset`: two exchanges, up to two deliveries per exchange, observerA
  or nil. Explores resets and completion races without the full two-by-three
  delivery product. A separate cross-reset witness offers both error identities
  and confirms that the second observer is not invoked.

These are bounded partitions, not a proof of arbitrary counts or the omitted
full Cartesian product. Limits and bounds are explicit cfg constants.
`swapped-budget` and `shared-budget` fail the independent history oracle.
The separate `hundred-charges-other` mutant retains the correct100 limit but
charges100 against the other budget: the actual100/102/102 trace rejects the
second non100 prematurely. `reach-independent-full-budgets` reaches one100
plus two non100s successfully.
Named reaches show two accepted non-100s, third non-100 overflow and a failing
100 observer. BudgetReset checks counters against history for the current
exchange, independently of their implementation assignments.

A new first-byte boundary after an abort is deliberately a conservative stress
interleaving, not a claim Go retries after Loom cancellation. The R2 cross-reset
observerA/observerB witness remains historical: invoking
observerB after retaining observerA is now rejected. Its successor
`reach-retained-error-across-reset` offers a later delivery selected for B, but
short-circuits it without invoking B and returns the retained A. This remains
a conservative reset-after-abort stress case, not an actual retry claim. A separate
`reach-clean-reset-failure` requires an actual completed clean first exchange
before reset, followed by observerA in the second exchange; it does not rely
on retry-after-abort. Healthy reset additionally reaches a successful read
with separate100/non100 counts in both exchanges.

## Checks and mutation sensitivity

Candidate checks: TypeOK, FirstErrorRetained, BudgetReset,
BudgetsIndependent, CallerExactlyOnce, AbortPreserved, ErrorJoin,
CallerOnlyError, StreamingAlive, ReadSurvives, BodyCancelOnce, BodyRelease,
CloseIdentity, ResponsePresence, NoObserverAfterAbort and
NoAdmissionAfterAbort. ResponsePresence is explicitly a modeled transport
assumption check, not independent proof of net/http. The two admission checks
observe history prefixes and are sensitive to bypassing the stored-error guard.
CallerExactlyOnce is
bookkeeping over atomic CallerHook steps; `double-hook` shows sensitivity to
replaying the caller, not proof of real callback mechanics.

The `late-observer` mutant bypasses the stored-error guard and fails
NoObserverAfterAbort on an actual new selected observer invocation. Positive
post-finish-hook and post-finish-caller witnesses retain the finished result,
skip the Loom observer and still invoke the caller through composition.

The response/error properties now have targeted mutants: reversed join order,
dropped Close error, changed inner-Close identity and skipping Close on nil Do
error. These must fail their named invariants; matching source expressions
alone are not offered as independent evidence. Other negatives retain the
observed plain-return error loss/eager-cancel lifetime cases and synthetic
no-reset/reset-error/shared-budget/double-hook/double-cancel cases. Synthetic
mutants are not additional observed production bugs.

Reach configurations intentionally assert the absence of a desired state and
must fail that predicate. They are non-vacuity witnesses, not product failures.
`PerDoReach` only defines predicates over PerDo, not a different behavior.
The packet's qualified-results.json records exact inputs, expected invariant,
terminal outcomes, bounds, state counts and source/tool hashes. Parser errors,
timeouts or zero-match checks are not accepted as negative evidence.

## Opt-in reproduction

Pinned Java: Amazon Corretto25.0.5; TLC1.7.4 JAR SHA256
`936a262061c914694dfd669a543be24573c45d5aa0ff20a8b96b23d01e050e88`.
Set JAVA to that installed java executable and TLA_JAR to that exact JAR.
Choose a configuration explicitly from this directory:

```sh
"$JAVA" -Xmx1536m -XX:+UseParallelGC -cp "$TLA_JAR" tlc2.TLC -workers 1 \
  -config checked.cfg -metadir /tmp/loom-per-do-checked PerDo.tla
```

Both checked configs expect exit0 and queue0. `current-late-write-other` uses
PerDoOther.tla; its constraint only excludes100 deliveries. Other mutant/reach configs expect
exit12 and the exact named invariant; all `reach-*` configs use PerDoReach.tla.
The evidence runner uses a120s process-group supervisor, one worker and1.5GiB.
It runs configurations sequentially. Preserve durable inputs/configurations,
concise results and review-needed traces; delete temporary checker state only
after independent review confirms no active process needs it. Keep shared
installed tools. No proof invocation belongs in ordinary build/test/CI/hooks.

## Go correspondence

The maintained controls are TestHTTPTraceComposeDiscardsNewHookError and the
TestObserveInformational* direct/race tests in `http/informational_client_test.go`
and `http/informational_client_wire_test.go`, including
TestObserveInformationalLateCallerAndBufferedHook,
TestObserveInformationalSuccessfulResponseRacesAbort,
TestObserveInformationalDeclared100Abort and
TestObserveInformationalResetAndFirstError. Generated callers are checked by
TestGeneratedInformationalDoBoundary and TestInformationalRegistrationOnly.
These textual correspondences do not assert a new model run. Preserve absent/nil/failing
caller hooks, original caller once, ordered errors.Is/Join identities,
first-error retention across first-byte boundaries, distinct100/non100 limits and
undeclared counting, streaming/raw reads after Do, bodyless cleanup and
concurrent EOF/Close returning the unchanged Close error.

Race controls: caller hook completes after Do/finish; buffered
Loom hook after finish invokes no new observer and cannot rewrite the returned
outcome, while the pre-existing caller hook still runs exactly once; cancellation
races a successful response, whose body must close on stored observer error
with nil Do error. Cover declared100 observer errors and a clean first exchange
before second-exchange failure. The model does not supply runtime proof of
those tests. Generated callers must compose once per Do only when observers
exist, consume finish before normal response decode, and retain existing body
transfer/cleanup. No-op observers and HTTP/2 remain implementation obligations.

## Abort admission decision (R3)

Two alternatives were considered: continue invoking selected Loom observers
for buffered deliveries after the first error, or stop Loom admission using
the existing first-error record. The chosen contract stops admission as soon
as that error is retained. This avoids new application side effects after
Loom has aborted the operation. It applies before and after Do/finish return,
not just to a timer or public-name class. No additional runtime flag or enum
is needed. An already-running callback is not forcibly interrupted; the
serialized owner retains its returned error before later admission.

Accepted: observerA fails, a buffered103 arrives, Loom returns the retained A
without invoking its selected observerB, and Go still invokes the existing
caller hook once. Accepted: the caller is still running when Do/finish returns;
it may complete later without rewriting the finished result. Accepted: the
first observer is still executing; its return is awaited under the existing
synchronization abstraction, not interrupted. Rejected: invoke B after A has
been retained, reset the retained error at GotFirstResponseByte, or manually invoke
the caller to compensate for Go composition.

This is an explicit tightening of the new feature's abort contract, authorized
by the task owner before implementation, not a claim that prior docs already
settled late observer side effects. The detached owning design amendment in
the packet states the same rule. No runtime adoption follows from these model
checks; actual direct/race tests must establish short-circuiting and normal
Go-owned caller composition.


## Response-boundary correction (R4)

Physical NextExchange, FirstResponseByte and WroteHeaders are now separate
transport actions. Hooks require firstByte. FirstResponseByte occurs once for
each modeled exchange and resets only its two counters in checked mode. A
header-write notification can occur before or after interims, or after Do
settles; checked mode ignores it. The firstByte/wroteHeaders/lateWrite fields
are transport progress/history observations in the model, not new runtime
state. Runtime needs only a different composed trace callback, no generation
counter or lifecycle flag. First error and body/finish ownership are unchanged.

`current-late-write` uses the reviewed implementation's header-write reset and
fails the independent history budget oracle: first byte, 100, late header-write
notification, second100 accepted in the same exchange. The additional
`current-late-write-other` configuration uses PerDoOther and a non100-only
state constraint with the identical transitions and oracle; it accepts a third
non100 after the same late reset at model Limit2. Counter equality is
intentionally not asserted in that config so it reaches the externally relevant
excess admission rather than stopping at the counter reset alone. In checked
mode new reaches show a late notification preserves a full non100 budget, the
next non100 rejects, and a second100 rejects. Existing healthy two-exchange
reset, retained error across reset and independent full-budget reaches remain
selected checks. Exact renewed counts and outcomes belong to the successor's
qualified-results; the R3 33-run receipts remain historical, not a claim all
configurations were rerun after this boundary correction.

BudgetReset is checked once the current exchange has received its first byte;
before that callback the counters can legitimately still describe the prior
exchange. No interim can be admitted in that interval. The history oracle still
checks every actual admitted delivery against its physical exchange identity.
The native ordering assumption is supported by pinned Go1.27.3 source, not a
proof of arbitrary custom Doer/RoundTripper trace behavior. A custom transport
that delivers 1xx before its first-byte callback or repeats first-byte midstream
violates that assumed callback order. No generic transport patch or relaxed
budget is proposed. Native retries after no response need no intermediate reset:
there were no interims to count; the next first response resets before delivery.

Go regressions must drive first-byte → eight hints → late WroteHeaders → ninth
hint rejection, plus the100 variant; a new first-byte callback refreshes counters
but never firstErr. Preserve old caller hooks through native composition and
explicitly assert HTTP/1 versus HTTP/2 in redirect/Expect-Continue controls.
Real transport retry policy, frame scheduling and concurrent callback internals
remain source/race-test obligations, not broader claims of this bounded model.

## Declared Continue admission

`Declarations.tla` makes the frontend assumption behind PerDo's declared 100
explicit. Exact 100 is an observation/documentation declaration; its emission
domain is empty. StatusField constraints intersect the observation domain for
informational declarations, so an unshadowed 1XX constrained to 100 is also
valid. Exact declarations shadow a class regardless of callback registration.
The finite selector projection uses 0 for 1XX and 999 for no selection.

Run these opt-in checks from this directory with the same pinned TLC tool:

```
java -cp "$TLA_JAR" tlc2.TLC -workers 1 -config declarations.cfg Declarations
java -cp "$TLA_JAR" tlc2.TLC -workers 1 -config current-declaration-rejection.cfg Declarations
java -cp "$TLA_JAR" tlc2.TLC -workers 1 -config current-emission-constraints.cfg Declarations
```

The corrected boundary checked 5,488 generated / 3,456 distinct states.
Both current-defect configurations fail `DeclaredHundredAvailable` with actual
exit 151: the invariant is a constant false declaration obligation, detected
before state exploration, not a temporal counterexample. The first models the
blanket exact-100 rejection; the second models admission using the narrower
emission domain. No PerDo transition changed or previous run was relabeled.
The positive configuration also checks the existing first-error, independent
budget, abort and error-join invariants with one exchange and one event.

This does not model parsing, arbitrary schema predicates, generated Go, native
Expect handling or network ordering. `TestInformationalContinueAdmission` and
`TestImportContinueDeclaration` check actual declaration admission and retained
empty emission domains; `TestInformationalContinueArtifacts` checks exported
JSON/YAML and re-import. The maintained generated module's
`TestDeclaredContinueNativeWire` checks typed observation of net/http's automatic
100, exact nil shadowing and callback abort; `TestDeclaredContinueCannotEmit`
checks both literal and native-field Send rejection before publication. The
shared `buildOutgoingStatusPlan` validates literal membership in the captured
emission domain, preserving the first error just like its native-field arm.
The installed runtime and existing broader PerDo evidence remain unchanged.
