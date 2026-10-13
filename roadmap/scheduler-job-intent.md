# Scheduler job intent (#737)

Status: implemented against `f1d1368fb11c9b0e0f494e771af1a1065ccbc4b0`;
final independent review and atomic delivery remain pending.

## Contract and owners

A producer owns responsibility for jobs it requests, including starts whose reply
is lost or returns `ErrJobExists`. `StopAll` selects only that producer's set,
excluding the current plan's Start keys. A stop enqueue is not completion.

Two existing boundaries need correction together. The scheduler's authoritative
producer ledger owns requested operations and conditional completion. The pool's
shared admission/ownership protocol owns job-incarnation ordering across routed
copies, stop requests and recovery. Producer membership is not worker eligibility:
ending a schedule must not invalidate already admitted jobs.

## Sequence

- [x] Reproduce timeout/ErrJobExists, committed cancellation, stale replica handoff
  and enqueue-success deletion; retain healthy isolation/Start-exclusion controls.
- [x] Reproduce old routed Start after successful stop using actual routing,
  worker reading and acknowledgment; retain valid fresh Start after stop.
- [x] Compare current and rejected partial fixes in the bounded pool intent model.
- [x] Independently review this concrete owner and acceptance before production.
- [x] Implement both boundaries; delete post-success bookkeeping and unfenced
  completion paths. Run only the affected tests/models and native Redis controls.
- [ ] Freeze exact final diff for independent review and atomic delivery.

## Selected proposal

1. Publish the producer's requested operations before dispatch. A revisioned
   authoritative update reads and updates the producer set atomically, so handoff
   and StopAll do not use a lagging `jobMap.Keys()` replica. Failed or unknown
   publication must not dispatch. A committed record survives a lost reply.
2. One Redis script compares the current producer revision/operation and records
   issuance, pool token allocation and event publication in the same atomic
   action. A separate successful check is never authorization to issue later.
   Superseded or reset operations emit no event and allocate no token. Completion
   also compares the exact revision. Later plans supersede unissued older work;
   already issued work retains pool ordering. A late completion never deletes a
   newer Start. Producer revisions/generations must not be reused after Reset,
   restart or producer-name reuse; retain a monotonic generation beyond Reset.
   Different nodes may finish long Plan callbacks in either order: Redis
   publication orders their effects, without a new callback lock or lease.
3. Extend the existing per-key `owner-epochs` allocation for distinct incarnation
   and stop-cutoff tokens. This durable counter already survives Shutdown.
   Worker ownership claims continue allocating their own `Job.Epoch`; gaps are
   harmless. Incarnation is separate from worker ownership, not an alias for the
   currently held Epoch. Read tokens as exact strings, as existing scripts do.
4. External start admission atomically allocates an incarnation with its queued
   event. Retain it through routing, worker jobs, authoritative incarnation
   storage and every requeue. Dead-worker recovery currently rebuilds a job from
   raw user payload and a fresh timestamp; it must carry the retained incarnation.
   Keep raw user payload unchanged and do not introduce a new generic allocator.
5. Stop publication atomically installs a cutoff and queues the stop. Worker
   claim rejects incarnations at/below that cutoff or no longer current. Stop
   handling acts only on an eligible old incarnation. Thus an old Start cannot
   revive a stopped job and an old Stop cannot kill a later external Start.
   Local `jobLock` orders claim/callback/stop/release. Authoritative ownership
   prevents claiming absence while an old worker still owns the job.
6. Reuse routed acknowledgments for stop completion. Confirm only after the
   owning worker's callback/release succeeds, or authoritative absence plus the
   installed cutoff makes old copies ineligible. Redis/handler/release errors
   and uncertain replies retain the pending producer operation. Completion
   conditionally removes only its exact revision. Public `StopJob` still returns
   after accepted queue publication; no synchronous public API is introduced.
7. Reconcile uncompleted producer operations on later owned ticks. A Start
   ordered after a pending Stop stays pending until it can be admitted; that
   older incarnation's `ErrJobExists` is not completion. Mark successful/admitted
   or observed unfenced existing jobs complete while retaining responsibility.
   Do not continuously restart jobs whose admission was already observed.
   Pending stops retry while the schedule participates; no new permanent job
   supervision is added. Recorded start/stop completion must survive reply loss.
   The first issuance persists the operation identity, incarnation or original
   stop cutoff and delivery receipt in that same atomic script. After an unknown
   reply, reconciliation reads that authoritative issuance record. Reissue may
   create a new delivery ID but must reuse the original operation and pool token:
   a retry of cutoff 5 cannot allocate cutoff 8 and kill a fresh Start 7.
   This applies to Start incarnations too. Every authored Start gets a new producer
   revision, even with identical
   key/payload or a still-pending predecessor. A repeated pending Stop retains its
   original identity/cutoff. Reconciliation retries the stored operation without
   publishing a new command. Removed/superseded records reject old retries rather
   than treating their absence as unissued work.

Explicit Stop retains its authored key semantics. Disjoint producers stay
isolated; overlapping keys retain the pool's existing global key identity rather
than inventing per-producer copies or exclusive ownership.

## Consumers and deleted paths

Replace scheduler post-success Set, replica StopAll enumeration and deletion on
StopJob's nil return. Replace unfenced worker admission/stop and recovery's
reconstruction of an unrelated incarnation. Extend the existing Lua helpers and
route/ack owner; do not add a parallel scheduler dispatcher. Consumers are
scheduler publication/reconciliation, claimDispatch, worker claim/release,
route/ack completion, rebalance/orphan/dead-worker recovery, job wire encoding,
protocol registration/health and shutdown cleanup. A protocol revision rejects
mixed old nodes; no old wire fallback or compatibility mode.

## Preserved lifetime and limits

Schedule uses its supplied context for Redis operations but has no ctx.Done loop
branch. Local Close and global ErrScheduleStop own planning termination. This
repair does not change that lifetime policy. ErrScheduleStop resets the producer
ledger and ends recurring planning, not running jobs; it abandons unissued
retries without invalidating admitted incarnations. Another active node can
reconcile persisted intent after local departure.

Delivery remains bounded by event retention and pendingEventTTL. Pending producer
operations can be reissued on later owned ticks; arbitrary outages, permanent
schedule shutdown, Redis data loss and callbacks that never return have no
liveness guarantee. Incarnation/cutoff records must outlive delayed copies, not
expire with event TTL. Retain the counter's monotonicity across pool-name reuse.

## Bounded acceptance

- Five parent reds and healthy controls in scheduler_intent_test.go. The canceled
  admission test preloads its script and verifies ctx.Err plus a committed guard;
  this distinguishes actual cancellation from a timeout after NOSCRIPT fallback.
- Pause an old issuer after its read, publish a newer plan or Reset/recreate,
  then resume: the atomic issuing script emits no event/token for the stale
  operation. Verify revision/generation non-reuse across Reset.
- Lose the reply after a stop cutoff commits, complete that stop, admit a fresh
  Start, then retry the old operation through real Redis/route/ack: its operation
  identity and cutoff are unchanged and the fresh job survives. Likewise a lost
  Start reply cannot allocate a new incarnation on retry.
- Publication failure before commit emits no event; lost committed reply retains
  intent. A successor sees it without replica progress. Lost stop reply and
  callback/release failure retain intent and retry without erasing a newer start.
- Fresh Start after confirmed stop succeeds; replay old Start and Stop through
  actual routing/ack and preserve the new incarnation. A newer Start while stop
  completes is eventually admitted and survives its older completion.
- Rebalance, orphan and dead-worker recovery preserve incarnation. Native Redis
  exercises atomic admission/cutoff, dispatch guards and ownership release.
- Same-producer handoff uses authoritative intent; disjoint producer jobs never
  enter StopAll. Current Start keys remain excluded.
- Existing Schedule/ErrScheduleStop/Close controls remain valid. An admitted
  pending start remains eligible when recurring planning ends. Mixed old pool
  nodes are rejected by the version gate.

The bounded SchedulerIntent model compares six rejected variants and a candidate
with 95 distinct states. Separate bounded issue/retry traces reject split checks,
reset-generation reuse and cutoff reallocation; the corrected variant passes
13 distinct states. They reuse pool replica and surviving-copy assumptions;
its completed-admission/stop abstraction is not a proof of callback, Redis or Lua
wiring. Tests check those seams. See pulse/pool/tla/SchedulerIntent.md for exact
commands, counterexamples and limitations. No full Redis suite or new general
coordination framework is required.

## Implementation and bounded validation

The ledger is now revisioned requested intent, not successful-dispatch timestamps.
`luaPublishIntent` replaces post-success Set and local replica enumeration;
`luaIssueIntent` compares revision, persists the first receipt and issues atomically;
`luaCompleteIntent` conditionally completes acknowledgment. Scheduler map access
is restricted to lifetime/reset operations. The existing dispatch waiter is reused.

The existing per-key epoch counter supplies exact incarnation/cutoff tokens.
`operation.go` owns the shared five-field envelope and stop confirmation;
`claimDispatch`, `claimRequeue`, `claimJob`, worker stop and dead-worker cleanup
consume that identity. Raw job and application payload encoding remains unchanged.
Obsolete protocol-2 backfill is removed. Protocol 3 rejects persisted older markers
and live older members. Upgrade requires ending old schedules (clearing their old
ledgers), draining/stopping work and completing old-binary Shutdown; Close alone
can leave unidentified pending operations. See the public Pulse guide.

Validation uses the existing miniredis/native Redis tier with the pinned Go 1.27.2,
Redis 8.10.2 and JSON v2 configuration. No generator output is changed, so no
code-generation specimen matrix applies. Native contract/ordering/recovery tests
passed under race (26.103s); actual failed/successful callback acknowledgments and
unidentified-wire rejection passed (3.109s). The initial callback test used an
hour-long worker TTL and timed out in Reader.Close after its assertions; the
corrected test uses the existing default bounded TTL. That pre-existing reader
shutdown limitation is not hidden or changed by this repair.

The selected existing consumer run (77.131s) covered dispatch/return, ownership,
lease fencing, recovery, worker admission/close, schedule and shutdown. It exposed
old exact-epoch, stream-length, validation-message and absent-stop expectations.
Those affected controls were corrected without weakening their assertions and
rerun under race (5.251s and 2.930s). Earlier adopted guard/claim controls passed
apart from three corresponding fixture assumptions; their focused repair passed
(5.589s). Exact tokens beyond Lua's integer range and raw payload equality remain
asserted. Private ownership fixtures now establish admitted incarnation state;
production does not accept missing identity to accommodate tests.

The original negative model evidence remains valid and the exact unchanged
candidate model was checked again: owned 95 distinct states, ordering-owned 13,
all safety invariants pass. The independent readiness corrections for atomic
revision issue and immutable retry cutoff are implemented and directly tested.
Final ignored-StopAll, rejected-plan, callback retry and release controls passed
on native Redis (5.599s). Final source/evidence hashes and commands are preserved
in the review packet.

## Final review R1: authored commands versus reconciliation

The first implementation incorrectly deduplicated every same-key/payload Start.
Root review and exact-source probes showed that the parent accepted a later Start
after external StopJob or after a claim-visible Handler.Start failure, while the
candidate permanently suppressed it. Merely renewing completed records also
misses a pending Start already fenced by an external stop.

Publication now gives every authored Start a fresh revision and unissued receipt.
Reconciliation without a new Start retains the old revision/token and never
reopens completed admission. Repeated pending stops keep their original cutoff.
This removes Start payload deduplication from the shared owner; no worker or
consumer exception is added. Existing guard/owner admission still prevents
competing starts, and delayed acknowledgments compare the exact revision/token.

`TestAuthoredStartAfterExternalStop`, `TestAuthoredStartAfterClaimFailure`, and
`TestAuthoredStartAfterFencedPending` all failed before this correction. The first
two also retain explicit empty-plan controls against automatic restart. The
bounded model adds command-dedup (6-state counterexample), completed-only
(10-state counterexample) and commands-owned (11-state pass), isolating this
semantic distinction without multiplying earlier ordering/replica state spaces.

After correction, the authored-command regressions and affected scheduler/intent
controls pass on native Redis under race (21.047s), and scoped lint reports zero
issues. The original owned/ordering configurations still pass at 95/13 distinct
states. Earlier non-publication runtime/consumer checks remain valid and were
not rerun. Original frozen source/evidence and the parent/candidate comparison
remain preserved for the narrow re-review.
