# Replicated-map snapshot authority (#672, witness #673)

Status: implementation and focused acceptance complete; final independent review pending.

1. [x] Preserve current gap and fixed/sliding TTL SetAndWait failures on the delivered finite-operation owner.
2. [x] Model loss, expiry/recreation, delayed snapshots and waiter observation. Compare numeric revision-only, gap-only, generation-only and one serialized generation-aware authority.
3. [x] Independently review common cause, concrete contract and bounded acceptance before implementation.
4. [x] Implement the sole replica owner and shared rmap/Pool writer protocol; preserve borrowed Redis hooks/pools and synchronous close.
5. [x] Verify actual loss/reconnect/expiry, retained notification strength, SetAndWait identity, cancellation and Close; retain #673 as a second witness.
6. [ ] Independently review the exact atomic #672 diff, integrate, commit, push and reconcile both reports.

## Shared defect

The replica treats an expiring numeric counter as evidence that its whole snapshot
is complete. A received revision can skip a lost mutation, and expiry reuses
revision numbers for unrelated contents. Both the incremental apply path and
SetAndWait trust that same invalid comparison. Gap detection alone misses a lost
last event and does not distinguish new incarnations. Generation alone does not
repair missed updates. A persistent revision counter avoids reuse but retains
unbounded metadata for every ephemeral map name.

## Candidate to test

- One serialized replica authority publishes snapshots and applies deltas; no
  delayed pre-expiry read may overwrite a newer incarnation. Local Get/Map reads
  continue to read an immutable complete snapshot without waiting for network I/O.
- A hash incarnation plus within-incarnation sequence identifies each mutation.
  TTL expiry naturally removes the incarnation; a later successful mutation
  assigns a fresh candidate ID atomically. No separate persistent keys.
- The shared Lua mutation owner preserves atomic hash/value/version/publication
  and TTL semantics. Failed conditional writes do not create metadata or extend
  TTL beyond the existing authored rule. Reset/Destroy remain same-incarnation
  operations; their existing monotonic semantics are retained.
- Contiguous same-incarnation deltas can apply. Gaps or different incarnations
  trigger an authoritative snapshot before claiming a newer complete state.
  Periodic reconciliation covers missed final messages, silent resubscription,
  and expiry with no later publication. Timing interval is an internal policy,
  not a new promise of instant consistency or hard network deadline.
- SetAndWait succeeds only after its exact incarnation/revision is covered by
  an authoritative observation. Reused numeric revisions cannot satisfy it.
  A write whose incarnation expires before observation returns an error instead
  of fabricated success. Its own observation must not depend on Pub/Sub delivery.
- Notification kind is part of the authority: pool scheduler consumes EventReset
  specially. A snapshot cannot blindly report Reset. Preserve last-reset and
  last-delete sequence markers within the incarnation so gap repair retains the
  strongest unseen kind; incarnation retirement is a real reset.

## Concrete owner and migration

Use the existing run goroutine as the sole replica transition owner. Its select
loop handles deltas, reconciliation ticks, reconnect and SetAndWait observation
requests. Each request is enqueued after the write returns its incarnation and
revision; the worker either recognizes an already complete matching cursor or
performs a new snapshot read. This gives the observation a causal post-write
barrier without a second snapshot publisher. Remove numeric waiter maps and their
separate lock/notification path. Requests have buffered result delivery, and
callers select their cancellation and the map lifetime while enqueueing/waiting.
The finite Redis operation owner and lifetime cancellation still bound worker I/O;
local Get/Map locks only protect publication, never network reads. Close seals
admission, cancels work and joins the same worker before freezing the local view.
Reset/Destroy's existing ability to mutate remote state after Close stays intact.

A periodic internal reconciliation tick repairs otherwise invisible loss and
expiry; it does not create a user-facing freshness deadline. On read failures the
replica keeps its previous complete state, logs and retries on later work. A
SetAndWait observation reports its own failure rather than waiting indefinitely
for unrelated traffic. Matching generation and adequate revision mean the write
was observed as part of a complete state, not that later writes cannot replace it.
A different authoritative generation means the targeted incarnation retired and
the call returns a descriptive error; cancellation remains observable.

The shared Lua envelope owns the generation, sequence and recovery-kind markers
and emits the same envelope for every successful mutator. Do not allocate metadata
for no-op conditional writes. All metadata lives in the hash and shares its TTL.
Use an existing collision-resistant ID dependency for candidate generations;
Redis chooses the first candidate atomically. Reset/Destroy preserve the generation
and sequence while clearing application fields, retaining necessary kind markers.

This is a coordinated wire/metadata upgrade: stop old writers before deploying
new participants. On first adoption, an atomic metadata initialization preserves
application fields and the existing hash TTL, assigns a generation and resets the
new protocol's counters. Do not create a hash merely to join an absent map.
Unknown future protocol metadata is rejected. Mixed old writers are unsupported; do not add an old-message decoder or compatibility mode.
Document that existing participants must restart together. Initial adoption is a
snapshot baseline, so historical pre-upgrade notification obligations are outside
the supported coordinated migration. All Pool writers derive the protocol from the shared owner below; Streaming keeps its ordinary map consumer API.

## Exploration results

Current numeric authority fails completeness (44 generated / 36 distinct states).
Gap-only repair falsely acknowledges after expiry (176 / 114); generation-only
still loses data (44 / 36). Gap plus generation checks without periodic repair
fail convergence after a lost final event (557 / 270). The combined authority
passes safety and eventual convergence (1498 / 492). Parallel snapshot publication
regresses (18 / 13); serialization passes (11 / 10). Recovery-kind latest-only
storage also loses an unseen reset (35 / 35); retained per-kind markers pass
(205 / 142). These finite models establish ownership and invalid counter
comparisons, not Redis implementation correctness, wall-clock freshness,
uniqueness of random identifiers or infinite reconnect
fairness. Concrete failure/cancellation tests remain required.

## Bounded acceptance proposal

Cover contiguous/duplicate/out-of-order/gap messages, lost-final-event convergence,
fixed/sliding expiry with and without recreation, same numeric revision in two
incarnations, overlapping writes and SetAndWait, stale snapshot completion,
reset/delete followed by later set, concurrent Close and wait cancellation.
Use miniredis for deterministic clock/fault seams and native Redis for automatic
resubscription/socket behavior. Run affected rmap races and representative pool
scheduler/streaming membership consumers; do not rerun unrelated generators or
the entire framework corpus. Existing notification-precedence and finite-I/O
models remain obligations, with new correspondence recorded beside rmap.

## Consumer writer boundary discovered during implementation

Pool ownership and dispatch guard scripts had independent copies of rmap revision
publication. The first consumer run failed worker admission with the new reader,
so an rmap-only implementation is insufficient. `pulse/internal/rmapstate` now
owns the Lua protocol and generation-candidate injection for both packages.
Every atomic Pool script declares its mutated map KEYS positions; protocol and
revision admission runs before any body command because Redis Lua errors do not
roll prior writes back. Pool still owns its guards, leases, fencing and stream
operations. Remove all copied revision/publication blocks; keep representative
Pool ownership/scheduler/dispatch controls and native execution in acceptance.


Pool admission uses marker 4 to exclude old writers, not merely an advisory version.
Stopped protocol 3 state may migrate once, atomically with new registration; job
operations retain their format and identity. No protocol 3 runtime writer remains.
Live incompatible members reject admission; older/malformed/future markers reject
without mutation. Both old/new join orders are modeled and exercised directly.
Migration retains application fields and TTLs and requires stopping all old writers.


## Acceptance evidence

Toolchain: Go 1.27.2, GOEXPERIMENT=jsonv2, pinned native Redis 8.10.2 at
loopback port 32768, TLC 1.7.4. Source parent is
`1d03a9e9bd2e0f7d1801490321cf0f331dda0582`; final packet hashes bind the source
and retained evidence. No dependencies or generated artifacts changed.

- Original gap and both fixed/sliding TTL observation regressions fail on parent.
- Whole shared writer and rmap race suites pass (1.779s / 8.567s). Focused final
  owner, observation cancellation/concurrency, protocol and construction races pass
  (1.275s / 6.656s / 6.120s). Canceled observation leaves committed Redis state
  intact, releases the sole publisher and preserves the borrowed client.
- Native shared writer/rmap/Pool/Streaming controls pass (1.725s / 3.986s /
  39.898s / 2.728s); native TTL and protocol4/construction controls pass
  (7.177s / 7.533s). Native Pool ownership, scheduler and dispatch controls are
  repeated after the marker change (rmap observation controls 1.599s, Pool 68.309s).
  Test packages sharing disposable databases run
  serially (`-p 1`). No full framework or generated-design sweep is needed.
- Focused golangci-lint reports zero issues. All bounded model families
  retain current counterexamples and pass the owned candidates. The existing
  notification precedence owner is unchanged; its owned model passes 170/98 states.

Recovery inputs and tests live beside the implementation. Raw logs and TLC state
remain outside the repository; the final review manifest records their hashes.
After acceptance, archive concise receipts and remove inactive temporary builds
and model states, preserving installed toolchains, shared caches and proof sources.


## Independent review: admission must precede migration

The first candidate joined/adopted shutdown and heartbeat maps before Pool protocol
admission. A live protocol 3 rejection could therefore reset existing map revision
metadata. Move atomic protocol admission before any map acquisition; enroll both
private registration withdrawals before the attempt and make heartbeat withdrawal
use the shared Lua writer directly, independent of an acquired Map handle. Share
map-name validation with the internal protocol owner so invalid inputs still fail
before Redis writes. Test public AddNode rejection for live old members and unknown
markers with old hash fields/TTLs/version unchanged, and preserve committed-write
lost-reply rollback. Extend bounded ordering exploration before implementation.

The public rejection witness fails on the first candidate and passes after moving
registration first. The ordering model fails at 6/6 and passes at 5/5 states. The
constructor ownership model was replayed under the new acquisition order (owned
388/376, all invariants). Fault-injection correspondence was also refreshed: shared
scripts append a generation candidate, so exact-length hooks must include that
argument and committed-node capture must still point at the authored node ID.

Re-review acceptance: focused Pool protocol/construction/worker-admission races
pass (29.641s); native protocol, construction, worker admission and fencing controls
pass (27.788s); shared owner and rmap name checks pass (2.338s / 2.387s).
The independent public probe now leaves revision 82 and every old field unchanged.
Focused lint reports zero issues. Earlier replica and native consumer evidence is
reused for unchanged paths. Original review packet is retained as superseded.


Integration exposed the pre-existing immediate server NUMSUB assertion in the
constructor cleanup-error test. A bounded current-main repeat fails 6 of 20 runs;
a deterministic miniredis disconnect gate establishes that local abort/channel
closure completes before server subscriber removal. The two constructor tests now
share bounded server observation, retaining synchronous local closure and all
cleanup/error/state assertions. This repairs the affected construction test owner,
not runtime cleanup or its contract.

Observation repair acceptance: focused constructor race controls pass 3.871s; the
20-run formerly failing cleanup case passes under GOMAXPROCS=2; focused lint is
clean. No production source changed in this observation-only delta.
