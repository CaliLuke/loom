# Scheduler intent design probe

Run from the repository root with the pinned TLA+ 1.7.4 jar:

```sh
java -cp "$TLA2TOOLS_JAR" tlc2.TLC -workers 1 \
  -metadir /tmp/loom-scheduler-owned \
  -config pulse/pool/tla/cfg/scheduler_owned.cfg \
  pulse/pool/tla/SchedulerIntent.tla
```

| Configuration suffix | Expected result | Distinct states |
| --- | --- | --- |
| current | ResponsibilityBeforeDispatch failure | 4 |
| prewrite | AuthoritativeStopSet failure | 10 |
| enqueue | NoForgetOnEnqueue failure | 10 |
| ack-only | NoOldCopyAfterStop failure | 23 |
| no-cas | NoNewerIntentRemoval failure | 21 |
| start-fence-only | FreshStartSurvivesOldStop failure | 36 |
| owned | All safety invariants pass | 95 |

The parent lost-reply trace corresponds to
`TestSchedulerRetainsUncertainIntent`; enqueue deletion corresponds to
`TestSchedulerStopEnqueueRetainsIntent`.
`TestSchedulerLateRoutedStartCannotReviveStoppedJob` follows the real router,
worker reader and ack loop; it failed on the parent after the late copy was
acknowledged. The candidate rejects that old incarnation.
`TestSchedulerStopAllProducerIsolation` retains a disjoint producer control.

The model borrows independent map observation and surviving routed copies from
`PoolOwnership` and `WorkerEventAck`; it does not recheck their protocols. It has
one owned key, one fixed unrelated running job, one superseding intent and a
fresh admitted incarnation after stop. Lost ledger and stop replies are separate
actions. Incarnation fencing is independent of the producer revision.
The owned variant assumes authoritative intent publication, stop confirmation,
conditional revision removal and an admission fence. It does not establish these
atomic operations in the implementation. `running` abstracts completed ownership
and handler admission. The proposal relies on existing worker jobLock spanning
claim/callback/release and on authoritative ownership preventing absence while
another worker owns the old incarnation. Regression checks must retain those
ordering seams; the model does not re-prove lease/callback fencing. The unchanged unrelated key is only a
scope invariant, not a proof of key encoding or Redis key isolation.

No liveness claim, arbitrary network-outage guarantee, schedule-context change,
or global serialization of long producer callbacks is made. Record Redis wiring,
worker release and finite event-retention constraints in the owning plan before
implementation; do not use a passing candidate as an authorization to skip them.

## Review corrections: issuance and retry identity

The additional scenarios in the same module isolate issue/retry ordering without
multiplying the original replica/copy state space. Run the same command with
`scheduler_<suffix>.cfg`:

| Suffix | Expected result | Distinct states |
| --- | --- | --- |
| split-check | NoSupersededIssue failure | 7 |
| reset-reuse | NoSupersededIssue failure | 9 |
| cutoff-reallocate | RetryPreservesFreshStart failure | 7 |
| ordering-owned | All three ordering invariants pass | 13 |

`CheckRevision` is only a caller observation. `Issue` must atomically compare the
current revision and publish the operation/token/event; after `Supersede`, `Reset`
or non-reusing `Recreate`, the owned variant rejects the old issuer. The
reset-reuse variant demonstrates why equality alone is insufficient if a reset
recycles an operation revision.

`ChooseStop` records the original cutoff with issuance. The reply is lost; the
stop completes and a fresh incarnation 7 starts. `RetryStop` either reuses cutoff
5 or incorrectly allocates 8; `ApplyRetry` exposes the latter killing fresh work.
The owned mode checks stable cutoff as well as the surviving fresh job. A new
public stop invocation may be a new operation; these transitions model retry of
the same recorded scheduler operation only. Start issuance uses the same stable
operation/token rule, with exact Redis regression obligations in the plan.

The original 95-state owned model was rerun unchanged in meaning. The earlier six
counterexample receipts remain valid: `IntentNext` and its invariant definitions
are unchanged; the added record is constant in those configurations. No runtime
source or test changed for this review correction. Final implementation tests
must check Redis linearization and exact token reuse rather than infer them from
a model that assumes atomic issuance.

## Implementation correspondence

`luaPublishIntent` owns authoritative producer revision and responsibility;
`luaIssueIntent` is the atomic Issue boundary and persists the first receipt.
`luaCompleteIntent` removes only the matching revision/token. The generation
counter is separate from rmap Reset. `TestIntentIssuanceRejectsSupersededRevision`
checks stale issuance and Reset reuse; `TestIntentRetryPreservesIncarnationAndCutoff`
and `TestRetriedStopCannotStopFreshStart` check lost replies and fixed retries.

`luaClaimDispatch` allocates a new incarnation; `luaClaimRequeue` and
`luaCleanupOwner` retain it. The five-field operation envelope carries key,
token, producer, revision and raw job data (raw user payload stays unchanged).
`luaClaimJob` requires the authoritative unfenced incarnation. `stopOperation`
serializes callback/release and `luaConfirmStop` checks absence behind the cutoff.
Tests cover callback/release faults, later plans, actual routing/ack, and
`TestIntentResetPreservesAdmittedStart`. Ownership/recovery tests retain their
original claims and exact-token assertions with protocol-3 admission fixtures.

Redis script execution is atomic with respect to other clients, not a rollback
transaction on arbitrary command/type errors. Internal key types and supported
Redis commands are assumed; network replies may be lost before/after commits.
Models retain their stated bounded safety assumptions; no unlimited-delivery,
callback deadline, disk persistence or external side-effect proof is claimed.

## Final review R1: authored Start is not reconciliation

| Suffix | Expected result | Distinct states |
| --- | --- | --- |
| command-dedup | AuthoredCommandIsNew failure | 6 |
| completed-only | AuthoredCommandIsNew failure | 10 |
| commands-owned | Four command/reconciliation invariants pass | 11 |

These additional scenarios are bounded branches of the same module. They model
completed admission followed by external stop, an observed ownership claim whose
Handler.Start later fails, and pending admission fenced by an external stop.
`AuthoredStart` requires a new revision, while `ObserveWithoutCommand` must not
restart work and `RetryPending` must retain operation identity. The rejected
completed-only fix still suppresses a command in the pending/fenced case.

The implementation boundary is `luaPublishIntent`: every authored Start gets a
new revision/receipt, while a repeated pending Stop keeps its cutoff. Stored
reconciliation continues through `luaIssueIntent` without a new revision. Direct
`TestAuthoredStart*` regressions cover all three scenarios; the first two include
empty-plan controls showing that reconciliation alone does not restart work.
The model assumes available admission and completed callbacks in this small
branch; actual claim, dispatch and failure ordering remains covered by tests.
