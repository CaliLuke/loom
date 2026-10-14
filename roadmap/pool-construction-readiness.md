# Node construction ownership — #687

Current production evidence starts at `c80c5b3595c772018ad9ba3607e36dd985bdc398`.
Readiness packet parent: `877d4135f23cc5177bf65a92b75bd39eefa52332` (#910 only changes
an unrelated integration module and plan; all pool/streaming owners are identical).
This plan replaces the historical mixed node/worker plan in the dirty
`pool-construction-ownership` worktree. That checkout is reference material only.
#704 worker admission is delivered; its code, tests and WorkerPublication model
remain separate and unchanged.

## Sequence

- [x] Trace current AddNode, protocol 3 registration, subordinate lifetimes and
  existing construction fault hook; preserve current four failing regressions.
- [x] Model current leaks and rejected partial rollback alternatives, then check
  one acquisition owner under explicit finite subordinate-close assumptions.
- [x] Independent review of this node-only proposal and readiness evidence.
- [x] Implement the accepted owner; extend focused fault and success controls.
- [x] Resolve the dependency gate through #800's exact published module fork and
  compose the accepted finite-operation owner with this constructor stack.
- [ ] Run published-dependency acceptance and obtain final integration review.
- [ ] Commit, push, close #687, and preserve compact evidence before cleanup.

The original readiness packet contained no production changes. Its independently
reviewed implementation is now being integrated on #800 with the published Redis
module pin. #800 must be delivered first: a constructor cleanup context cannot
itself make a subordinate Close finite.

## Current failure and owning boundary

`AddNode` returns from several acquisition stages without releasing preceding
resources. Root's current-source race run reproduces four subscription leaks:
shutdown rejection (one map), later map join (three maps), initialization after
sink acquisition (eleven maps including the sink's two), and client-only
initialization (two maps). The durable regression uses the existing
`constructionFailureHook`; it does not introduce another Redis harness.

`joinOwnershipProtocol` atomically publishes protocol 3 and the initial heartbeat.
It does not backfill job ownership. Its reply can fail after Redis commits. The
private node stream's initialization has the same uncertainty. Recording only
successful returned acquisitions therefore cannot own all rollback obligations.
A Sink starts its keepalive after construction, but starts polling and claiming
only at Subscribe. AddNode does not subscribe either sink or reader until the
complete Node has been assembled.

Use one private constructor acquisition owner, local to the pool package. Enroll
local release immediately after acquisition; enroll a remote withdrawal before
attempting a write. One deferred abort unwinds the recorded sequence in reverse,
collecting failures without skipping later releases. Successful assembly transfers
ownership exactly once before subscribing or starting node loops. The existing
special protocol-error branch must be subsumed by this owner, not retained as a
second cleanup policy. Never call complete-node `close`, `cleanupNode` or
`cleanupPool` on a partial value.

| Acquisition | Constructor obligation |
| --- | --- |
| shutdown and node-heartbeat map replicas | Close local handles, never destroy shared keys |
| attempted protocol admission | Remove only this ULID's heartbeat and protocol fields; retain shared version marker |
| pool stream wrapper | No local goroutine or private shared-stream deletion |
| worker/job/payload/heartbeat/ticker/cleanup/pending replicas | Close each acquired handle |
| pool sink | Close local runtime and replicas; preserve existing pending-event/membership recovery contract |
| attempted private node-stream initialization | Withdraw this unique private stream even after a lost write reply |
| private reader | Close unsubscribed reader |
| runtime cancellation function | Cancel only while constructor still owns it |

The sink's retained consumer membership is intentional existing streaming state,
not a subscription leak; Close stops its renewals and future stale cleanup retains
pending-event recovery. Do not destroy its shared group, filter identity, stream
or shared replicated maps to make a rollback assertion pass.

Remote cleanup uses a bounded context detached from caller cancellation, preserving
logging values. Use the configured worker shutdown bound when positive; for a nonpositive
value use the existing default worker shutdown bound. Current option parsing
does not reject nonpositive values; this internal cleanup bound adds no option
or unrelated admission policy. A canceled caller must not suppress
withdrawal. Report original and cleanup errors together using errors.Join, with
operation context and errors.Is identity preserved. Heartbeat and protocol-field
withdrawals must both be attempted even when one fails. Local Close has no error
return; it must still run after any remote cleanup failure.

This promises local release and attempted private withdrawal, not atomic rollback
across Redis operations. A failed/uncertain cleanup can leave private stream or
registration state; no failed node runtime may renew it. ULIDs are attempt-unique.
The model permits arbitrary surviving subsets and does not promise absence after
an outage, delayed commands, or finite caller latency without subordinate liveness.

## Bounded acceptance

1. Preserve all four parent-red subscription assertions and error identity.
2. At protocol registration and private XADD, inject both rejected writes and
   committed writes with lost replies, including caller cancellation. Check
   authoritative node-specific fields/private stream and healthy peer state.
3. Inject withdrawal failure: all remaining cleanup attempts and local releases
   still occur; both original and cleanup causes remain discoverable. Do not
   require successful remote rollback on that branch.
4. Directly exercise owner ordering, empty rollback and single transfer; verify
   reader/runtime cancellation are not released after transfer. There is currently
   no fallible AddNode operation after reader acquisition; use a direct owner test
   for that suffix instead of inventing an unreachable Redis hook.
5. Keep successful ordinary and client-only nodes usable and closeable; retain
   protocol rejection and healthy peer job routing. Reuse existing pool tests and
   miniredis controls; one native finite-close check belongs to the #800 gate.
6. Preserve shared map keys, version marker, healthy peer fields and pending-event
   contract; no worker admission or ownership-protocol rewrite.

No generator or full exported-design corpus gate applies. Models are bounded
constructor-order safety checks; implementation tests establish correspondence.
The current model assumptions and run commands live beside NodeConstruction.tla.

## Readiness evidence

The checked-in node test reproduces all eight expected parent failures under
`go test -race ./pulse/pool -run '^TestConstructionNode' -count=1` (5.430s):
four original subscription leaks and registration/private-stream lost replies,
each with and without caller cancellation. Every lost-reply case reached its
committed-write hook; the shared protocol marker assertions pass. This recorded readiness run predates
production changes. Existing dispatch, client-only dispatch and protocol controls
pass in the separate focused race run recorded in the packet.

The refined model represents heartbeat and protocol withdrawal separately;
registration writes both atomically, while cleanup can leave either independently.
Owned safety checks pass with 542 generated / 530 distinct states. Current leaks,
successful-reply-only enrollment and stop-on-error alternatives fail the intended
invariants. Results and exact commands: `/tmp/loom-687-model-results.json`.
The root's original four-failure log remains independently preserved alongside
the new test log; no repeated full pool suite or generator run was used.

## Historical conditional implementation evidence

Isolated worktree: `/Users/luca/.codex/worktrees/numeric-comparison/loom`.
Clean parent after exact archived #768 inverse/fast-forward: `ce26ecbe9`.
The exact reviewed #800 source is a conditional base tree
`15844df052331cdbc8fa13ec2703fbd5cc71d27b`; the #687 patch is computed against
that tree, excluding all #800 source. `/tmp/loom-687-conditional-base-receipt.json`
records source hashes and exact dependency revision. Neither repository go.mod nor
go.sum changes: tests use `/tmp/loom-687-conditional.mod` with the reviewed local
go-redis correction. These are conditional tests, not original-module fetchability
or #800 delivery evidence.

`nodeConstruction` enrolls each map at Join success, protocol/heartbeat before
registration and private stream before XADD. Reverse abort closes the unsubscribed
reader/sink, cancels owned runtime state and attempts both registration deletions;
all errors are joined. Private-stream DEL and protocol HDel borrow the shared redisop client for finite
I/O budgets. Only the private stream key was initialized; the node reader creates
no sink group or membership map to withdraw. The runtime Stream keeps the original
client so its Reader can derive its own blocking-read budget without inheriting
an ordinary-command deadline hook. Caller clients are never closed or reconfigured. Successful assembly transfers before Subscribe.

Parent-on-#800 retains all eight expected reds (5.487s). Initial owner fixes all
of them; final acceptance additionally checks fifteen individual reverse cleanup
steps, joined distinct errors, cancellation/value/deadline context behavior,
idempotent abort, transferred runtime preservation, failed remote withdrawal with
local subscription release, and a healthy peer's registration across lost replies.
Existing ordinary/client dispatch, protocol and worker admission controls remain.
The unchanged model results are reused: implementation did not alter its ownership
or uncertainty assumptions. No generator output exists in this runtime-only scope.

Final bounded checks on this conditional source: focused race selection passes
5.586s; native Redis selection passes 3.781s; scoped pool lint reports zero issues.
The native selection covers all node construction failure/uncertainty cases plus
ordinary and client-only dispatch. Lint uses a recorded Go shim only for x/tools'
standard-library release-tag probe with GO111MODULE=off; actual package loads
retain the disposable modfile. The initial unshimmed lint invocation could not
load modules and is not positive evidence. A final empty test-key struct formatting
correction changes no declarations' semantics; gofmt is exact. All source remains
uncommitted, with exact conditional base/source hashes in the final packet.

## Published-dependency integration acceptance

The delivery candidate composes the exact reviewed node owner with #800's
shared finite-operation/lifecycle owner and actual published Redis fork pin,
plus the delivered #820 module graph. No local replacements or test modfiles
are required. Active Redis imports are translated consistently; the narrow
constructor guidance preserves current main and #800 lifecycle guidance.

Focused construction failure/lost-reply, reverse-order/error aggregation,
transfer, client-only admission and worker admission race controls pass
(12.715s). Actual Redis constructor and failed-withdrawal/local-release controls
plus client-only dispatch pass (3.073s), with no skipped tests. Scoped pool lint
reports zero issues. The unchanged owned model passes again with 542 generated
and 530 distinct states. Earlier negative-model results remain applicable: no
state transition or configuration changed.

The implementation differs from the previously reviewed conditional owner only
in Go import identity and formatting; the dependency and surrounding-owner
composition are the new integration review boundary. #800's existing complete
pool race run remains valid as baseline evidence; no full pool rerun is needed.
