# Sink group filter and membership ownership — #768

User decision: one enforced effective filter per named group on each stream.
Conflicting joins fail before delivery; independent subscriptions use distinct
groups. Filter rejection is final for its group, while stored entries and other
groups remain untouched. Coordinated migration is required. Root independently
approved revised readiness and exact final/integration reviews. Delivered and
confirmed closed as `7da1b00ee`; both medium review findings are resolved.

## Sequence

1. [x] Preserve the subscribed original and claimed-entry parent failures and the
   observed heterogeneous-filter handoff. Inspect all membership consumers.
2. [x] Model current heterogeneous admission and pending-state leak, reject split
   read/write admission, last-member identity deletion and independent map expiry.
3. [x] Reproduce focused failures on the delivered parent; add same-filter,
   conflicting-filter, empty-membership and TTL controls using existing helpers.
4. [x] Independent readiness review of this representation, model correspondence
   and bounded acceptance; resolve implementation gaps before production.
5. [x] Implement one typed membership owner, update every mutation/read consumer,
   then let shared message disposition acknowledge terminal rejections.
6. [x] Run focused lifecycle/filter/race/native Redis checks and scoped lint.
7. [x] Independent exact final review, atomic integration and delivery.

## Representation and authority

Replace each existing `stream:<stream>:sinks` map value (group key) with one
versioned record: effective filter identity plus its consumer set and registration
timestamps. Identity is
none, exact topic, or exact regex spelling; preserve current option precedence
(nonempty Topic wins over TopicPattern). Compile regex before publication. Empty
members do not erase the record. Sorting members makes serialization canonical;
old array records and malformed/unknown versions produce explicit coordinated
migration errors rather than being interpreted as unfiltered. Old binaries must
not run concurrently with migrated groups; this is not a compatibility mode.

Use one private membership owner for admission, removal and decoding. Read current
Redis value authoritatively, then use existing replicated-map SetIfNotExists or
TestAndSetEx to compare and publish the entire record. Retry CAS contention with
the returned/current value. Checking a local replica's filter and then appending
is insufficient: replicas may lag and concurrent first joins must have one winner.
If a new admission observes no identity record but the named Redis group exists,
reject unknown legacy ownership: absence after legacy TTL expiry is not evidence
of a fresh group. Only reconciliation of a previously admitted active registration
may restore its known identity after a reset; NewSink and first AddStream have no
such authority. Supported concurrent creators publish the record before creating
the group, so a losing first-join CAS retries and sees the winning identity.
Do not publish membership after checking filter in a separate mutation. Existing
revisioned map mutation/publication remains authoritative; no additional registry.

NewSink, AddStream, missing-group reconciliation, rotation preparation, retired
consumer cleanup, constructor rollback, stale-consumer cleanup and final removal
must all use this owner. Preserve membership-before-group creation, authored start
selectors and lifecycle mutex boundaries. Record each attempted ownership before
an uncertain write; errors may follow committed writes. Retry/removal must be
idempotent. Do not delete Redis consumers with pending entries as part of member
retirement. Preserve first-subscriber activation from #901.

The group-destruction script must inspect validated typed membership and check
member count atomically with XGROUP DESTROY. It retains the filter record whether
the Redis group exists or is destroyed. RemoveStream does not reset subscription
identity. Close retains memberships for the existing reclaim lifecycle.

Stream.Destroy resets the stream and membership map using the existing revisioned
map destroy protocol. Changing filter identity requires coordinated teardown (stop
all old users, Destroy, then join with the new filter), or a new group name.
Uncoordinated destruction/reset and active recovery retain #765 behavior for the
same identity; no claim is made that an external destructive reset preserves old
messages or permits safe concurrent identity migration.

## TTL ownership

The parent `consumersMapOptions` copied stream TTL. rmap applies sliding expiry only
on map mutation; stream Add refreshes the stream separately, while ordinary
keepalive updates another map. The membership hash can therefore expire while
its stream/group remains live. Filter identity cannot use that independent TTL.
Implemented correction: the existing membership map is persistent until explicit
Stream.Destroy; stream data TTL remains unchanged. No second metadata key or
periodic best-effort refresh. Admission reads authoritative PTTL before any publication and explicitly rejects
an existing expiring membership hash, even when its own group field is absent.
Every cooperating writer uses no-TTL options, so a persistent/absent hash cannot
acquire expiry between that check and CAS. Concurrent legacy binaries or manual
EXPIRE are outside coordinated migration, not supported concurrent writers.
Existing old TTL state must be removed during coordinated teardown; admission
does not silently persist/migrate it. A zero-TTL option does not cancel existing
Redis expiry (`keyttl.apply_ttl` does not). This lifetime change is documented as
part of enforced persistent group identity; source TTL tests must still pass.

## Shared disposition

`streamEvents` is the single fresh/claimed/Reader delivery owner. Its malformed
and filtered branches share terminal rejection completion. A Reader skips without
XACK. A Sink attempts XACK, logs failures and leaves precommit failure pending for
ordinary reclaim. A lost successful reply has already removed pending ownership; XAUTOCLAIM cannot
return that entry again. Completion is physical PEL removal, not observing a local
success reply. Only a still-pending precommit failure is reclaimed for retry. NoAck fresh reads
have no pending entry, but claimed historical entries can: do not globally disable
terminal-rejection acknowledgment using NoAck. Accepted events remain application
owned, including interrupted fanout and no-subscriber cases. This is group PEL
removal, never XDEL. No new routing, scheduling or shutdown protocol.

## Bounded acceptance

- Existing original five-filtered-entry barrier and claimed-entry red; exact,
  pattern, no-filter and malformed parity; Reader and NoAck controls.
- Concurrent conflicting first joins admit one identity; same identity admits both
  without losing a member. Equivalent-looking but differently spelled regexes
  conflict; exact-topic precedence is retained.
- AddStream conflict leaves its prior registration intact. Rotation, missing-group
  recovery and retirement retain the same identity and existing start positions.
- Empty membership / final RemoveStream may destroy group but cannot change
  identity; coordinated Destroy permits a new filter. Legacy record rejects.
- Stream activity outliving membership's former TTL cannot admit conflicting
  identity. Existing fixed/sliding stream TTL remains correct.
- Existing remove/close/activation/rotation/recovery tests plus narrowly injected
  CAS contention and uncertain reply; no independent duplicated lifecycle model.
- Terminal ack failure/lost reply/reclaim; accepted unconsumed events stay pending,
  original stored entry and a second group's pending entry survive rejection.
- Selected existing tests run against native Redis and miniredis; no codegen suite.

## Model evidence and limits

`pulse/streaming/tla/filter_membership` initially records two small models. Membership covers
same/different first joins, stale snapshots, removal, atomic group cleanup and
coordinated identity reset. Disposition covers accepted/rejected/malformed entries,
initial pending or no-pending state, one failed/lost acknowledgment and reclaim.
Both compare broken alternatives before the proposed owner. Existing rotation,
recovery, map ownership and subscriber models continue to own their detailed
lifecycle obligations. JSON serialization, Redis errors, migration decoding and
network availability are checked by direct tests, not established by these models.

Parent b1a142df2 focused race run (1.606s) records nine expected failures: two
disposition, four conflicting admission, empty membership and two TTL cases.
Four same-filter/precedence cases and independent groups pass in the same run.
This was the preserved pre-implementation checkpoint.

Readiness re-review corrections: shared admission rejects an expiring existing
map even for a new group, legacy array records, and an existing Redis group with
missing identity. Three additional parent tests reproduce those failures. The
disposition model now allows reclaim only while pending and establishes eventual
PEL removal rather than a fictitious local success stage after lost replies.


## Implementation correspondence and validation

`newSinkFilter` owns effective identity and matching; `changeSinkMembership`
compares the full versioned value through revisioned rmap CAS for every writer.
`sinkConsumerMap` keeps the original Stream alongside its replica, including
cleanup-only registrations, so retirement uses the same Redis source and key.
`readSinkMembership` owns authoritative value/TTL admission; only recovery and
rotation of admitted active registrations pass restore authority. The group
cleanup script compares the exact decoded empty record before destruction,
retaining identity and avoiding another JSON decoder in Lua. `ackRejectedEvent`
is shared by both terminal rejection branches of `streamEvents`.

The #772 constructor checks and overflow-safe stale comparisons remain unchanged
on parent 6f82fb52f. Existing lifecycle tests now decode the typed record instead
of the old array. The activation pending-claim fixture first creates and closes
an admitted sink before adding a foreign pending consumer, preserving its
first-subscriber assertion without inventing a legacy group with no identity.
Wrong-type failures now occur at authoritative group-identity preflight before
creation; their type-error/no-leak assertions remain.

Focused controls cover simultaneous missing-record reads, both matching and
conflicting filters, final-removal identity, coordinated reset, AddStream rollback,
uncertain committed admission replies, malformed records, Reader/NoAck behavior,
other-group/stored-message preservation and failed/lost ACK replies followed by
actual reclaim. The gated concurrent tests and fault hooks check implementation
correspondence; models retain the original bounded assumptions and state counts.
No generated code or OpenAPI surface changes, so no generator corpus is needed.


Initial final-candidate validation (before review corrections): full affected streaming/options race tests pass (8.344s/1.342s),
scoped golangci-lint reports zero issues, and native Redis 8.10.2 checks pass:
filter/activation/remove/recovery/rotation selection 4.228s plus final lifecycle
controls 2.057s. The original 12 parent failures and five positive controls remain
in the readiness evidence. One intermediate new test compared a registration to a
Stream while holding the mutex; its assertion failure blocked test cleanup. That
test-only mistake was corrected by snapshotting registrations under the lock and
asserting afterward. It is not counted as a production counterexample.

The original membership/disposition models remain unchanged from the independently
approved readiness packet. Their recorded results remain valid; the review
correction below adds the bounded persistent-member retirement obligation.
No broader CI success is claimed.


## Final-review corrections R1 and R2

Independent review reproduced two defects in the first candidate: closed members
survived actual stream expiry because cleanup visited only native consumers, and
missing/null filter objects were admitted as unfiltered. Both are corrected at
the shared membership owner, without changing group creation/publication order.

The record now requires an explicit filter kind (`none`, `exact`, or `pattern`)
and each member has a positive registration timestamp. Missing/null/empty filter
objects and malformed member records reject with the existing migration error.
`deleteStreamStaleConsumers` reconciles persistent and native membership together;
authoritative keepalive determines freshness when present, otherwise registration
age protects a published join until its first heartbeat and lets abandoned old
members retire. Native absence alone never authorizes retirement. The existing
atomic pending check prevents deletion of consumers owning pending entries.
Empty membership still retains filter identity, and the existing exact-record
check owns final group destruction.

The additional bounded model reproduces the native-only ghost and the unsafe
absent-only alternative, then checks freshness/pending safety and eventual closed
member retirement. Its freshness abstraction assumes cooperating clock/lease
behavior, as the existing heartbeat protocol does; it is not a distributed lease
lock. Actual tests cover configured stream expiry, no-heartbeat age fallback,
cleanup interleaved before native consumer creation, explicit-none admission and
four absent/null/empty-filter rejection cases. Existing lifecycle assertions now
inspect member names without discarding stored registration age. One manually
assembled rotation fixture now uses the canonical explicit-none filter identity.

Correction validation: full streaming/options race checks pass (10.369s/1.325s),
bounded native Redis 8.10.2 retirement/rotation/recovery/removal checks pass
(4.349s), and scoped golangci-lint reports zero issues. Original unaffected
admission/disposition evidence remains valid. The corrected exact packet passed
focused re-review and integration review, and was delivered as `7da1b00ee`.
No aggregate CI success is claimed.
