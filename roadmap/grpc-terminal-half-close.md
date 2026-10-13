# gRPC half-close admission after terminal release (#872)

Readiness at pushed `555d13306e126da0df4d0cefb156aa027e5ccfa5`; implementation
parent `d212ddc51a121549a18881b1ef4c603835b94986`. This is a
separate #726 follow-up; #699 readiness files in this checkout are paused and
excluded. Independent readiness approved before production by
`/root/omission_readiness_finish`; verdict
`/tmp/loom-872-independent-readiness-verdict.json`.

The original CI witnesses are in `/tmp/loom-ci-9508-failure.log`. Focused current
reproduction: `/tmp/loom-grpc-ci-repro.log`. Recursive-validation and designed
error assertions pass; the subsequent cleanup `Close()` fails because a terminal
receive released its opening context. The remote status was not corrupted.

## Contract and owner

Generated client `Close` for bidirectional/server-streaming wrappers means native
`CloseSend`, not final status reception. The pinned grpc-go implementation makes
this operation idempotent and returns status through receive. An already-terminal
RPC has no remaining send direction to close. Ordinary message operations still
reject terminal or pre-canceled admission; `CloseAndRecv` still receives status.

The shared stream lifetime owner must classify half-close independently from a
message send, while both use the same direction lease. Under one lock, terminal
half-close admission succeeds without I/O; active half-close keeps the opening
context alive. A terminal receive racing admission must not expose the owner's
internal cancellation as a new cleanup error. Completion that wins before terminal
retains its own native result, including custom native CloseSend errors. Do not
retain a canceled RPC, swallow message errors, or add template-local suppression.

## Sequence

- [x] Reproduce the three CI failures separately; distinguish two stale test
  assumptions from this runtime lifetime/admission defect.
- [x] Read pinned grpc-go CloseSend behavior and existing Loom half-close contract.
- [x] Add direct failing terminal receive -> repeated half-close cases for EOF,
  remote status, and successful final receive. Preserve the received result/cause.
- [x] Model current conflated admission, rejected lifetime-retention alternative,
  and classified half-close admission across both terminal orderings.
- [x] Obtain independent readiness review of exact owner design and report.
- [x] Implement shared classified admission, retaining first-terminal outcome,
  bounded direction leases, active cancellation and late native completion rules.
- [x] Cover terminal-before-close, close-before-terminal, active native close
  result, competing sends, live half-close then receive, repeated cleanup and
  original context cancellation. Rerun invalidated owner/model checks only.
- [x] Run unchanged mounted recursive-validation regression and existing stream
  lifecycle/admission tests under race. Generator output must remain unchanged;
  compare one existing bidirectional generated specimen parent/candidate/repeat
  and compile it, without a full generator sweep.
- [x] Freeze exact atomic packet for independent final review and root delivery.

## Separate CI test owners

`TestNativeContextStatus` expects injected remote errors even when pre-canceled
receive performs no I/O. Keep its remote matrix and add explicit no-receive local
context assertions in a separate test-only ticket.

`TestStreamSendRejectsBeforeWrite` supplies nil invocation context by constructing
zero wrappers. Use explicit nonnil operation context for its conversion-before-I/O
seam, retaining conversion-specific error and no-write checks in its own ticket.

Neither fixture issue warrants a production nil-context fallback or a change to
the accepted pre-canceled operation contract.

## Proposed implementation seam

Represent half-close as an operation kind distinct from ordinary send, receive,
and final receive; its direction reservation remains the send slot. Resolve the
invocation owner once, then let locked admission decide both terminal disposition
and operation-context rejection. A terminal half-close returns an explicit no-I/O
success outcome. A live owner still rejects an already-canceled operation context,
without ending that owner, and rejects an occupied send direction. This ordering
must be atomic with `abortLocked`; checking `ctx.Err` in the wrapper and suppressing
its error afterward is insufficient.

If terminal release wins while a native half-close is active, the shared operation
result arbitration returns successful cleanup without changing the recorded
terminal cause. If native completion wins first, preserve its exact result or
panic using existing completion arbitration. Keep its occupied slot until native
completion even when terminal cleanup has returned. No new goroutine, callback,
or native operation is permitted for terminal no-op admission. Existing sends,
receives, CloseAndRecv, and callback stop/join behavior retain their semantics.

Readiness evidence: `/tmp/loom-872-direct-parent.log` reproduces all three terminal
receive cases and the gated active-close/release race. The live admission control
passes. `/tmp/loom-872-HalfClose{Current,Retain,Classified}.log` records current
counterexample (4/4 states), rejected retention (2/2) and classified candidate
(42/30). The unchanged generated recursive regression fails only cleanup after
correct error decoding; `/tmp/loom-grpc-ci-repro.log` preserves that witness.

## Final validation

- Complete runtime package race tests pass in 1.928s, including original operation
  cancellation, admission cleanup, callback, first-cause and native panic controls.
- Unchanged `TestRecursiveValidationGenerated` plus `TestGRPCMessageContextGenerated`
  pass in 11.614s, including their module builds, vet and mounted race harnesses.
- Existing bidirectional specimen produces 13 identical generated files at pushed
  parent, candidate and fresh-process repeat. Candidate `go mod tidy`, build and
  vet pass. No generator/DSL/checked-in fixture output changes are intended.
- Existing bounded-worker, completed-operation lease and admission-cleanup models
  pass again (118/68, 13/10 and 11/8 generated/distinct states). The reviewed
  HalfClose model sources remain unchanged; prior negative/positive checks apply.
- Scoped golangci reports zero issues; `git diff --check` passes.
- Artifact ledger: `/tmp/loom-872-artifact-comparison.json`. Source/evidence hashes
  are frozen separately for final review. #699 untracked readiness and already
  delivered #873/#874 are excluded from this ticket's exact patch.

## Delivery

Delivered and pushed as `5c6535b6d`; GitHub confirms closure. Independent
integration verdict: `/tmp/loom-872-independent-integration-verdict.json`.
