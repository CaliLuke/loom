# Sink registration and missing-group recovery — #765

Implementation complete; awaiting independent final review. Parent is delivered
#901 `4db060dfc`.
The active registration owns its Stream identity and selected group-creation
start position. On the parent, AddStream discarded its override after initial creation and
shared read-error handling mistook every NOGROUP for completed removal.

## Contract and owner

Retain each active registration as one private record containing its Stream and
selected start ID. Derive XREADGROUP arguments from these records. Retained map
handles awaiting failed removal cleanup are not active registrations. Replace
the separate active Stream/cursor arrays, rather than adding another start-ID map.
Constructor/AddStream publish the record after admission; RemoveStream removes it
after confirmed deregistration. Existing lifecycle mutex orders all these changes.

One locked registration reconciler restores current membership, creates an absent
group with that record's selector, and ensures the current consumer. Membership
publication precedes group creation, preserving SinkAddStream's peer-removal
ordering. BUSYGROUP never resets a live shared cursor. Only current records are
visited; never reconstruct intent from an earlier XREADGROUP argument snapshot.

Read NOGROUP and replacement-consumer NOGROUP enter this same owner. Rotation runs
before XREADGROUP, so repairing only read failures cannot reach that case. Failed
repair retains a pending obligation until all current registrations confirm;
uncertain replies do not discard it. Healthy rotation keeps its existing candidate
and retired-consumer cleanup protocol. Recovery is serialized with Remove/Close;
there is no detached completion that can recreate a removed stream.

Stream.Destroy deletes the stream and resets the replicated membership map in
separate commands. The latter can arrive after recovery recreated the group.
The direct reset witness establishes why an active registration remains responsible.
The existing ensureConsumer and periodic keepalive paths consult local replica
membership and invoke the same reconciler only for missing membership or retained
repair. Keepalive is necessary when a healthy BLOCK 0 read has no subsequent event.
No new polling worker, unconditional remote writes per read, or cleanup-only-map
healing is introduced. Replica lag may cause harmless idempotent republishing;
external destruction is not an atomic distributed transaction with recovery.

After a missing-group attempt or failure, reuse the interruptible retry owner with
a positive floor (the existing 100ms retry interval) and bounded jitter. Periodic
keepalive already has a bounded cadence. No hot loop, deadline guarantee for Redis
I/O, or change to #800 shutdown is claimed. Stop retrying creation once closing or
no active registrations remain. Generic Reader error handling need not gain sink
policy: the Sink consumes NOGROUP before its generic error handler.

## Accepted start semantics

Only absent-group creation reuses the registration's authored selector, as initial
creation does. Oldest may replay retained entries after XGROUP DESTROY; newest
skips entries already present at recreation; explicit ID and timestamp remain
unchanged. A surviving group's cursor is never reset. Deleted events and deleted
pending entries cannot be recovered; no exactly-once guarantee is added. This
interpretation was accepted by the orchestrator from the existing public contract;
no new policy question remains.

## Sequence and bounded acceptance

1. [x] Preserve actual parent read-spin, rotation and membership-reset negatives.
   Two controls retain removed-registration nonrevival and initial per-stream
   override behavior. The five-test race run fails only the three expected cases.
2. [x] Compare current hot-loop, stale-snapshot and main-start alternatives in the
   bounded model; corrected owner checks safety and finite-recovery progress.
3. [x] Transfer these inputs/model beside the owner after independent readiness
   review. Implement one record/reconciler and delete superseded cursor rebuilding.
4. [x] Direct correspondence: read and rotation repair, delayed map reset,
   membership-before-group, failed/uncertain recovery then retry, bounded backoff,
   Remove/Close before repair and repair before removal, re-add with new selector,
   and surviving peer-group cursor preservation. Do not edit model-derived plan
   records to simulate behavior. Hook barriers control command ordering.
5. [x] Native Redis specimen covers Destroy/re-add delivery, group-only deletion
   with oldest/newest/explicit selector and per-stream override, live peer cursor,
   and delayed membership reset. Reuse existing activation/rotation/removal tests.
   Affected streaming race tests and scoped lint; no generator or Redis corpus.
6. [ ] Fresh exact final review, atomic #765 delivery by root, then cleanup.

Exit condition: current active registrations regain usable group/consumer and
membership after finite remote disruption under retry/replica progress; local
removal/shutdown prevents later recovery creation, selectors remain intentional,
and failed recovery does not spin. Models abstract Redis protocol and command
latency; implementation tests check actual commands and outside-model behavior.
#768 filtered pending entries, #800 shutdown, initial constructor rollback,
all-unsubscribed replay and unbounded destructive peers remain separate contracts.

## Implementation correspondence and results

`sinkRegistration` replaces the separate stream/cursor arrays and owns its
`recoveryPending` flag. `reconcileRegistrations` retains a failed flag, publishes
membership before group creation, and never rolls back an existing consumer's
pending list after a keepalive failure. `armRead` derives its keys and `>` markers
from current registrations. Both rotation and read NOGROUP enter the reconciler;
rotation returns its original failure after repair so the read owner still backs
off and joined retirement errors remain visible. Existing keepalive checks local
replica membership without starting delivery. Constructor rollback and generic
Reader error policy remain unchanged.

The original rotation probe was strengthened to assert the initial NOGROUP error
remains visible and the next preflight succeeds, rather than requiring the repair
to hide that failure. The original frozen readiness negative is retained externally.
Direct controls cover four selectors, re-add identity, peer cursor preservation,
eight before/after command failures, PEL retention, lifecycle serialization, retry
floor/interruption and keepalive-only reset repair. Miniredis reports short `0`
where native Redis reports `0-0`; assertions normalize equivalent zero IDs, and
peer progress uses real XREADGROUP rather than unsupported miniredis XGROUP SETID.

Full streaming race checks passed (11.651s); the same bounded recovery family on
Redis8.10.2 passed (4.172s). The unchanged accepted model passed safety and finite
recovery (1212 generated /651 distinct). Original parent tests and all model
negative traces remain evidence, not passing gates. No generated output changes
or full Redis audit were needed. Final lint and exact source hashes are in the
external final manifest; temporary logs/checker output are excluded from Git.
The final bounded lifecycle joins replace unbounded test waits; the native run
checks that exact final harness. The full package run used identical production
source and assertions before this timeout-only harness adjustment.
