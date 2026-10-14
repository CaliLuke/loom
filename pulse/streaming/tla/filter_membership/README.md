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

## Persistent member retirement (final-review correction)

Stream expiry deletes Redis consumer inventory but does not delete the persistent
filter record. `MembershipRetirement.tla` starts after this expiry and recreation:
a closed stale member is absent from native inventory, a fresh join is published
before native creation, and another member owns a pending entry. The native-only
alternative cannot retire the closed member (2 distinct states); deleting every
native-absent member loses the fresh join (4 distinct states). The owned candidate
preserves fresh and pending members and eventually retires the closed member
under weakly fair cleanup (5 generated / 4 distinct states).

```sh
for mode in native_only absent_only owned; do
  java -XX:+UseParallelGC -cp "$HOME/.cache/loom-tools/tla2tools-1.7.4.jar" \
    tlc2.TLC -workers 1 -config "retirement-$mode.cfg" MembershipRetirement.tla
done
```

`deleteStreamStaleConsumers` reconciles the union of typed members and native
consumers. Each CAS member carries its registration timestamp. An authoritative
keepalive, when present, determines freshness; registration age protects the
interval before the first keepalive and permits retirement if that keepalive is
absent after shutdown/expiry. The existing two-grace threshold remains unchanged.
The Redis XPENDING/DELCONSUMER script checks pending ownership atomically even if
an entry arrived after inventory was read. CAS removal retains filter identity;
conditional keepalive deletion preserves a newer observed timestamp.

The model abstracts timestamps to a freshness predicate and assumes cleanup
continues while the named group exists. It does not model clock skew, indefinite
constructor pauses beyond the existing grace lease, Redis failures or a heartbeat
refresh racing a stale snapshot; these do not establish a distributed lease lock.
`TestSinkFilterReviewExpiredStreamRetiresClosedMember` checks actual expiry and
final group removal; `TestSinkFilterCleanupPreservesJoiningMember` interleaves
cleanup after group creation but before consumer/heartbeat publication;
`TestSinkFilterRetiresMemberWithoutKeepAlive` checks old/fresh registration
fallback. Existing pending-consumer controls check the atomic PEL guard. Strict
record tests now distinguish absent/null/empty filter from explicit `none` and
reject missing/invalid member registration timestamps. JSON parsing remains a Go
test obligation, not a property of this bounded state model.
