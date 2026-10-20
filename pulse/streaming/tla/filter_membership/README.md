# Group filter membership and rejection completion

Design models developed before #768 implementation. The accepted
contract is one effective filter per stream/group, checked atomically with member
publication, and group-final acknowledgment of rejected events.

Run with the existing pinned TLC jar:

```sh
java -XX:+UseParallelGC -cp "$HOME/.cache/loom-tools/tla2tools-1.7.4.jar" \
  tlc2.TLC -workers 1 -config owned.cfg FilterMembership.tla
java -XX:+UseParallelGC -cp "$HOME/.cache/loom-tools/tla2tools-1.7.4.jar" \
  tlc2.TLC -workers 1 -config disposition-owned.cfg FilterDisposition.tla
```

| Case | Result | Generated / distinct |
| --- | --- | --- |
| current | heterogeneous admission violates Homogeneous | 37 / 34 |
| split | separate filter check/publication loses a member | 40 / 37 |
| forget | final member removal loses identity | 16 / 14 |
| ttl | independent metadata expiration loses ownership | 15 / 13 |
| owned | safety passes for distinct filters | 164 / 104 |
| same-owned | safety passes for shared filter | 275 / 156 |
| expiring-owned | expiring membership hash rejects every admission | 29 / 18 |
| unknown-group-owned | existing group without identity rejects admission | 13 / 9 |
| disposition-current | filtered pending entry has no completion owner | 10 / 10 |
| disposition-owned | safety and physical PEL removal pass | 28 / 24 |

Current membership models the unguarded heterogeneous join behavior with a ghost
filter identity (the parent implementation stored only names). Split/forget/ttl
are rejected alternatives. Candidate CAS compares the entire snapshot; filter and
members change together. RemoveGroup checks membership at destruction time and
retains identity. CoordinatedDestroy abstracts an explicit reset with no admitted
members, pending join or group; it is not automatic empty-member cleanup. Ordinary
Close/recovery and concurrent external destruction retain the existing lifecycle
models' scope; this model does not imply a global distributed lifecycle lock.

Membership safety: admitted names are preserved; all admitted names share the
record's filter; identity remains until coordinated reset; activation follows
membership. Two actors and two identities bound the model. Identity-reset models
assume coordinated users; arbitrary manual metadata deletion is excluded.

Disposition assumes eventual reclaim plus an eventual successful acknowledgment
under weak fairness and at most one failed/lost reply. Accepted entries are never
automatically acknowledged. Rejection does not delete stored messages or change
another group's pending state. Initial pending=false represents NoAck fresh reads;
pending=true also covers historical claims by NoAck instances. The model does not
claim exactly-once application delivery, bounded retry latency or Redis availability.

Implementation correspondence: `changeSinkMembership` uses full-record rmap CAS
for NewSink, AddStream, reconciliation, rotation/retirement/rollback and stale
removal. `readSinkMembership` checks authoritative PTTL and strict record decoding;
only previously admitted registrations have restore authority. Go validates empty
membership, then `destroyGroupScript` compares that exact record atomically before
XGROUP DESTROY. `streamEvents` calls `ackRejectedEvent` for malformed/filtered
entries; accepted fanout remains unchanged.
Source tests must cover actual Redis CAS/revision behavior, strict legacy decoding,
option precedence, lifecycle fault handling, TTL and Reader/NoAck paths. Models
establish properties only under their explicit assumptions.

Readiness correction: Reclaim is enabled only while pending. LostReply removes
the PEL entry and leaves a locally uncertain stage that need not become done.
RejectionLeavesPEL expresses physical completion. An actual precommit failure
stays pending and reaches AckSuccess under the stated finite-failure/fairness
assumptions. No model path reclaims an acknowledged entry.

ExpiringHash models an admission guard before any publication, not automatic
WithTTL(0) persistence. Under cooperating new writers the guard remains stable;
legacy concurrent writers and manual EXPIRE are prohibited migration violations.
InitialUnknownGroup is the expired-legacy-map case: a new join cannot claim a
group whose identity is absent. Previously admitted active recovery is handled
by the existing group-recovery model and explicit local registration authority;
new admission cannot use that authority. Strict version decoding remains a Go
test obligation rather than a modeled JSON parser.

## Persistent member retirement and heartbeat races

Stream expiry removes Redis consumer inventory without removing the persistent
filter record. Cleanup therefore considers both typed membership records and
native consumers. Registration age protects the interval before a member's first
heartbeat; afterward, `LastKeepAlive` in the member record is the freshness
authority. `updateConsumerKeepAlive` changes that same record through
`changeSinkMembership`, and stale retirement compares the exact member value
through the same rmap CAS protocol. There is no separate keepalive hash whose
snapshot can disagree with membership.

`MembershipRetirement.tla` models the existing native-absence, registration
freshness and pending-member obligations. Its `pending` set is fixed at
initialization: it checks that cleanup does not discard an entry already pending
when its native inventory is taken. It does not model new deliveries or PEL
changes during cleanup. The model keeps the `native_only`, `absent_only` and
`owned` controls for the older cleanup alternatives; this model is not evidence
that pending preservation is dynamic.

`HeartbeatRetirement.tla` models the heartbeat and pending-entry races across
native deletion, membership retirement and final group destruction. The
snapshot-only alternative violates `FreshPreserved` when a heartbeat arrives
between snapshot and native deletion (24 generated / 22 distinct states, depth
5). A separate freshness recheck still fails when the heartbeat arrives after
that recheck but before membership CAS (48 / 34, depth 6). Versioned CAS passes
fresh-member preservation (136 / 80, depth 12), but the current unguarded
final-destroy path violates `PendingPreserved` when a new read becomes pending
after native deletion (123 / 76, depth 11). Adding a pending check to the
atomic final group-destroy decision preserves both invariants (127 / 76, depth
11).

```sh
for cfg in heartbeat-snapshot.cfg heartbeat-recheck.cfg heartbeat-versioned-cas.cfg \
  heartbeat-pending-current.cfg heartbeat-pending-guard.cfg; do
  java -XX:+UseParallelGC -cp "$HOME/.cache/loom-tools/tla2tools-1.7.4.jar" \
    tlc2.TLC -workers 1 -config "$cfg" HeartbeatRetirement.tla
done
```

The heartbeat model has one stream, one refreshable member and one other
member. It abstracts heartbeat values to a membership revision and pending
entries to whether the group has a PEL. It does not model clock skew, Redis
failures, multiple simultaneous entries, stream contents or application
processing. `DeliverPending` represents an outstanding blocked read that can
complete after `DELCONSUMER`. Membership retirement can still remove that member;
the final atomic `destroyGroupScript` decision preserves an empty group's PEL.
An unacknowledged group remains claimable by a later same-name sink. The model
assumes cooperating writers and a successful atomic Redis command; Go tests
cover actual Lua behavior and rmap CAS/revision publication.

`TestSinkStaleSnapshotCannotRetireRefreshedMember` blocks both before native
delete and after successful native deletion but before membership CAS. It
commits a heartbeat and verifies membership, the shared group and subsequent
active delivery. `TestSinkStaleDeletePreservesPendingReadBeforeMembershipRetirement`
starts an outstanding read, creates pending work in that same gap, removes the
other member, and verifies the group retains an acknowledgeable PEL entry.
`TestSinkFilterReviewExpiredStreamRetiresClosedMember` checks stale closed-member
retirement after expiry. `TestSinkFilterCleanupPreservesJoiningMember` checks the
pre-heartbeat registration window, `TestSinkFilterRetiresMemberWithoutKeepAlive`
checks registration-age fallback, and
`TestSinkKeepsStaleConsumerWithPendingEventsUntilClaimed` checks the native
pending-entry guard. Strict record tests cover v1 rejection and JSON field
validation; JSON parsing remains a Go test obligation.
