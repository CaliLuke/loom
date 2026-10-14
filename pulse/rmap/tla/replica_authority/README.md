# Replicated-map authority exploration

The shared defect behind #672 and #673 is treating an expiring numeric revision
as proof that a local snapshot is complete. These bounded models compare repairs
before implementation. The current Go regressions are
`TestReplicaAuthorityRejectsIncompleteGap` and
`TestReplicaAuthorityTTLObservation` (fixed and sliding TTL); all three fail on
parent `1d03a9e9bd2e0f7d1801490321cf0f331dda0582` with Go 1.27.2, jsonv2 and race.

## Models and correspondence

`ReplicaAuthority` bounds three writes and one expiry, with Pub/Sub loss and
arbitrary delivery interleavings. Snapshot fields represent an atomic HGETALL;
queued message snapshot fields are specification ghosts used to detect incomplete
claims. Incarnation IDs are unique symbols, not numerically compared. Current
`Map.applyMessageLocked` corresponds to numeric-only acceptance;
`registerWaiter`/`notifyWaitersUpTo` correspond to numeric-only observation.
Owned acceptance requires contiguous same-generation deltas; gaps and generation
changes request a snapshot. Periodic snapshots establish convergence after traffic
quiesces under fair successful read execution. Drop fairness represents eventual
drain or loss of finite queued traffic. This is an eventual consistency argument,
not a latency bound, proof of random-ID uniqueness, Redis atomicity, arbitrary
unbounded failures, or network liveness. The waiter invariant proves only absence
of false success; Go tests must cover causal post-write reads, explicit retired
incarnation errors, cancellation and Close.

`SnapshotOrdering` isolates stale concurrent reads. Its source integers are ghost
chronology, not a new persistent Redis counter. A sole run-goroutine publisher
serializes capture through publication, so a delayed old read cannot overwrite
newer state. Snapshot and delta execution must use that same owner. Local Get/Map
only takes the publication lock and does not wait for Redis.

`RecoveryKinds` bounds three arbitrary change/delete/reset operations. Snapshot
repair must retain the strongest kind since the previous complete cursor. Keeping
only the latest kind loses a reset followed by a set; last-reset and last-delete
revision markers recover that obligation. It does not model expiry: retiring a
known incarnation emits Reset, and an initial Join has no prior subscription
obligation. The existing `../notification_precedence` model owns retention after
publication; do not replace it with latest-event notifications. The actual pool
scheduler checks EventReset, so gratuitous Reset on every repair is invalid.

## Run and expected results

Use pinned TLC 1.7.4, with checker state outside the repository:

```sh
java -cp /Users/luca/.local/share/tlaplus/tla2tools-1.7.4.jar tlc2.TLC \
  -metadir /tmp/loom-rmap-authority-owned -config owned.cfg ReplicaAuthority.tla
```

| Configuration | Expected result | Generated / distinct |
| --- | --- | --- |
| current | NoIncompleteClaims fails | 44 / 36 |
| gaponly | WaiterSound fails after expiry | 176 / 114 |
| generationonly | NoIncompleteClaims fails | 44 / 36 |
| triggered | Converges fails without periodic repair | 557 / 270 |
| owned | Safety and convergence pass | 1498 / 492 |
| snapshot-parallel | NoSnapshotRegression fails | 18 / 13 |
| snapshot-serialized | NoSnapshotRegression passes | 11 / 10 |
| kinds-latest | RetainsUnseenKind fails | 35 / 35 |
| kinds-markers | RetainsUnseenKind passes | 205 / 142 |

Use `SnapshotOrdering.tla` for snapshot configs and `RecoveryKinds.tla` for kinds
configs. Other configs use `ReplicaAuthority.tla`. The roadmap records implementation, migration and concrete acceptance. Preserve borrowed Redis client identity/hooks and the delivered finite
operation lifecycle; the model does not supersede those separate obligations.

## Atomic writer correspondence

`ReplicaAuthority.Write` denotes both public rmap mutators and Pool's atomic
ownership, dispatch and scheduler scripts. An initial rmap-only prototype failed
Pool admission because two copied Pool writers still published the old protocol.
All now derive their preflight and publication from `pulse/internal/rmapstate`.
A repository-wide production Go search finds revision increments and publication
metadata writes only there. Pool retains its own lease/fencing/stream decisions.

`WriterAdmission` models an atomic script writing a stream/owner plus two maps.
Checking each map only when its helper runs leaves prior writes behind on an
invalid second map. Shared preflight over every declared map key rejects before
any body effect: local admission fails after 13/13 states; shared admission passes
11/11. Run `admission-local.cfg` and `admission-shared.cfg` with that module.
This checks protocol/revision admission errors known before the body. It does not
claim Redis rolls back arbitrary later runtime errors, outages or exhausted
resources. `TestSharedWriterPreflightPrecedesAllEffects` checks the exact Redis
behavior, including unsupported protocol, reserved-field collision and exhausted
revision; `TestSharedWriterExactRevisionAndIncarnation` covers decimal precision.

The delivered owner seams are `receiveMessage`, `reconcile`, `observe`,
`readSnapshot`, and shared Lua `publish_map` with whole-script `validate_map`.
The old revision waiter map and reconnect replay publisher are removed. The run
loop serializes snapshots through publication; background and post-write reads
cannot overtake each other. Snapshot failures retain the old complete state.
Recovery tests cover loss without later traffic, both TTL modes on native Redis,
retired incarnation errors, strongest kinds, stale generation hints, coordinated
metadata adoption, no-op TTL and reserved-key admission. Existing shutdown and
notification tests remain separate obligations.

## Pool upgrade admission

`ProtocolAdmission` models the atomic old/new joins in both orders. Reusing marker
3 allows an old join after new admission (12 generated / 9 distinct states,
`ExclusiveAdmission` fails). Advancing to marker 4 prevents simultaneous mixed
membership (11 / 8, invariants pass). Run `protocol_current.cfg` and
`protocol_owned.cfg` with `ProtocolAdmission.tla`. The model starts with no live
members and covers absent, 3, 4, older, malformed and future markers. It assumes
Redis serializes each script and callers cannot bypass admission. It does not
prove a heartbeat lease means an old process has stopped: deployment must stop all
old writers before migration, as documented. `TestProtocolAdmissionConcurrentOldJoin`
runs historical admission against the current script; migration controls preserve
application fields/TTLs and reject unsupported markers without mutation.

`MigrationOrdering` covers the constructor boundary before atomic Pool admission.
Acquiring/adopting maps first violates `RejectPreservesState` (6/6 states);
admission before acquisition passes (5/5). Run `migration_before.cfg` and
`migration_owned.cfg` with that module. `TestProtocolRejectedConstructorPreservesOldMaps`
exercises public AddNode for worker and client nodes: live old members and
unsupported markers leave old maps, expiries and membership unchanged. Protocol
admission owns heartbeat withdrawal before any local handle is available; the
shared writer deletes only that node field if present. The constructor ownership
model was replayed with registration first (see its correspondence record).
