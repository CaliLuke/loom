# Node constructor acquisition owner (#687)

This node-only model is separate from delivered WorkerPublication/#704. It uses
current protocol 4 (no job backfill) and deferred Sink/Reader Subscribe. The
historical dirty-worktree node model is not current implementation evidence.

`AcquireLocal` enrolls each successful local acquisition. `AttemptRemote` enrolls
node registration and private-stream initialization before Redis executes; a
failed reply nondeterministically means no write or committed write. Registration
publishes heartbeat and protocol fields atomically but records separate withdrawal
obligations; cleanup explores independently surviving fields. `Fail`
represents early shutdown rejection or any later fallible acquisition.
`Transfer` gives all local resources to the assembled runtime exactly once.
`ReleaseLocal` and `WithdrawRemote` unwind one ordered stack. Remote deletion may
succeed, fail before commit, or lose its reply after committing. All reported
cleanup failures are retained with the original error. Failed construction never
starts a node runtime; no constructor heartbeat renewal is possible.

## Assumptions and correspondence

- Local Close returns. This is an explicit **#800 delivery dependency**, not a
  theorem about blocking Redis calls. NewSink only starts keepalive until
  Subscribe; the private Reader is unsubscribed. The model abstracts calls into
  terminating transitions and checks safety, not unbounded network liveness.
- Grouping seven later map handles as `otherMaps` bounds the state space. A Go
  owner-order test must check each individual registration/reverse release.
- Unique node IDs make node fields and private stream attempt-owned. Redis
  executions already completed may have uncertain replies. Arbitrarily delayed
  server execution after cleanup is outside scope; no distributed rollback or
  confirmed remote absence is promised after cleanup failure.
- `shared` is an explicit frame condition: map keys, protocol-version marker,
  pool stream/group/filter identity and peer state are never cleanup targets.
  Implementation tests must verify this noninterference, not infer it from the
  abstract immutable variable. Sink Close's retained recoverable membership is
  not a node-private deletion target.
- Current mode reproduces ordinary early-return leaks, notably shutdown rejection
  after one acquired map. It does not reproduce the existing special case that
  closes two maps on protocol rejection; that branch cannot prevent the modeled
  earlier and later leaks.
- There is no current AddNode failure after Reader acquisition; suffix ownership
  is a direct-owner test obligation rather than an invented failing Redis command.

`node_current` must fail LocalRelease. `node_successful_only` rejects enrollment
only after successful write replies (AttemptedWithdrawal). `node_stop_on_error`
rejects abandoning cleanup after its first error (LocalRelease).
`node_owned` must pass all seven invariants. No invariant requires that an
uncertain remote cleanup succeeds; it requires that all attempted writes receive
a withdrawal attempt and all local resources are released.

## Running

Use pinned TLA+ 1.7.4 and Java 25 as documented in README.md:

```sh
java -cp /Users/luca/.cache/loom-tools/tla2tools-1.7.4.jar tlc2.TLC -workers 1 \
  -metadir /tmp/loom-687-model-owned -config node_owned.cfg NodeConstruction.tla
```

Run from this directory. Repeat with each negative configuration and a distinct
outside-repository metadir; expected exit is 12 for those and 0 for owned. Preserve
summarized state counts and counterexample identities before deleting checker
output after independent review. Tests map acquisition/withdrawal transitions to
AddNode, joinOwnershipProtocol, Sink.Close, private-key DEL and Reader.Close.

The four parent subscription witnesses are
`TestConstructionNodeFailureReleasesSubscriptions`.
`TestConstructionNodeLostWriteReply` and
`TestConstructionNodeCleanupFailuresReleaseLocalResources` cover committed writes
with lost replies, cancellation, cleanup errors and healthy-peer preservation.

## Readiness results

| Configuration | Outcome | Generated / distinct states |
| --- | --- | --- |
| node_current | LocalRelease fails | 5 / 5 |
| node_successful_only | AttemptedWithdrawal fails | 26 / 26 |
| node_stop_on_error | LocalRelease fails | 12 / 12 |
| node_owned | All seven safety invariants pass | 542 / 530 |

Each negative run returned TLC exit 12; owned returned 0. The actual lost-reply
regression `TestConstructionNodeLostWriteReply` checks both registration fields
and private stream, with and without cancellation, while preserving shared marker.
It fails all four parent cases. The original four subscription cases also fail;
existing dispatch, client-only and protocol controls pass independently.

## Implementation correspondence

`nodeConstruction.joinMap` enrolls successful handles; `own` records separate
heartbeat/protocol withdrawals before `joinOwnershipProtocol` and private stream
withdrawal before XADD. `abort` consumes the release stack in reverse, joining
errors under a detached bounded context. `transfer` clears constructor ownership
immediately before subscription/runtime starts. `TestNodeConstructionAbortOrderAndErrors`
checks all fifteen actual cleanup steps and distinct error identities;
`TestNodeConstructionTransfer` checks the transferred runtime survives later abort.
`TestConstructionNodeCleanupFailuresReleaseLocalResources` checks surviving remote
state and closed local subscriptions when all three withdrawals fail. The expanded
lost-reply regression preserves a healthy peer's registration. Integration checks
use #800's shared operation/lifecycle owner with the published hook-preserving
module pin. The node change follows #800 in delivery; no local dependency
replacement is required. The module rename leaves this model's transitions and
finite subordinate-close assumptions unchanged.


## Replica protocol admission ordering (#672)

Registration now precedes every map acquisition because joining an old map adopts
its metadata. Atomic protocol admission rejects incompatible members/markers before
that migration. `rmapstate.ValidName` gives Pool the same pre-write input validation
as rmap. Heartbeat withdrawal uses the shared Lua owner directly and therefore does
not depend on a Map handle. After later construction failure, local handles close
before the attempted registration is withdrawn; shared migrated state remains.

Replaying this acquisition order gives owned 388/376 states with all invariants
passing. The negative current/successful-only/stop-on-error modes fail
`AttemptedWithdrawal` at 2/2, 6/6 and 6/6 respectively. Original historical counts
above describe the earlier acquisition order. The owner-order test follows the new
fifteen-step order. Lost-write reply controls identify the exact node argument
rather than a trailing metadata candidate; worker fault hooks likewise retain their
original authored argument positions while accounting for that appended candidate.


Local resource release and remote server observation are separate transitions.
`ReleaseLocal` means the local lifecycle is joined; it does not require the server
socket reader to have removed its subscriber registry entry. A deterministic
miniredis disconnect barrier in `TestConstructionLocalClosePrecedesServerUnsubscribe`
holds server removal after local Close: abort returns and its local subscriber
channel is closed while NUMSUB still reports one. Releasing the gate permits
bounded observation of zero. Constructor tests share that bounded server-observation
helper; all error aggregation, attempted withdrawal and retained-state assertions
remain unchanged. The historical immediate NUMSUB assertion fails intermittently
on main and in CI; it must not imply an unjoined local owner.
