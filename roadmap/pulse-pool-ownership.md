# Pulse Pool Job Ownership

Status: active. Design accepted (owner-record redesign). Tickets 1 to 4
are done. Tickets 5 to 8 are open.

Code references are to `main` at `967f4fbe`. The model lives in
[`pulse/pool/tla`](../pulse/pool/tla/README.md).

## Problem

`pulse/pool` promises that each job key runs on one worker at a time. The
code does not keep that promise. A TLA+ model of the pool, checked with TLC,
finds short interleavings that break it. A review confirmed that every trace
is possible in the Go code.

Nothing in Redis records which worker owns a key. Ownership is inferred from
three sources that can disagree:

- the consistent hash over each node's replica of the active workers,
- the `jobMap` replica (worker to keys),
- the payload map.

Every duplicate-start event that the pool creates (redelivery, dispatch retry,
orphan requeue, rebalance, cleanup) therefore reaches `handler.Start`.

### Findings

"Trace" is the number of states in the shortest TLC counterexample (single
worker, breadth-first). Configurations are under `pulse/pool/tla/cfg/`.

| Id | Defect | Code | Trace | Status |
| --- | --- | --- | ---: | --- |
| B1 | The sink redelivers an unacked start event (XAUTOCLAIM after `ackGracePeriod`). `startJob` never checks `w.jobs`, so the job starts twice on one worker, or on two workers if the second router's view differs. | `worker.go:248-273`, `streaming/sink_consumers.go:308-316` | 6 (`double_asis`), 7 (`runner_asis`) | same worker fixed by ticket 2; across workers open until ticket 5 |
| B2 | `dispatchJob` releases the pending guard on timeout while the start event is still queued. A retry is admitted and adds a second start. | `node_jobs.go:112-123`, `scripts.go:41-51` | 8 (`double_noredeliver`), 9 (`runner_dispatch_retry`) | fixed by ticket 4 |
| B3 | `Close` and `cleanupWorker` delete the worker's `jobMap` entry but keep the payload. The orphan sweep takes no lock and requeues a job whose requeue is still in flight. | `node_cleanup.go:59`, `node_recovery.go:80-134` | 12 (`double_orphan`) | open |
| B4 | `rebalance` requeues without checking whether cleanup already moved the key, and without removing `jobMap[w]`. Rebalance and cleanup both requeue the same job. | `worker.go:370-388`, `node_recovery.go:145,161` | 15 (`double_rebalance`) | open |
| B5 | `startJob` writes `jobMap` then the payload. These are separate `rmap`s with separate subscriptions. A node that sees the payload without the `jobMap` entry for longer than the grace period requeues a running job. | `worker.go:252,256`, `node_recovery.go:104-126` | 10 (`runner_orphan_lag`) | open, needs lag longer than `2*workerTTL` |
| B6 | Cleanup trusts a stale keep-alive snapshot and never re-checks it after taking the lock. Eviction calls `worker.stop`, which never calls `handler.Stop`, so a worker that only looked dead keeps running its jobs until the process exits. | `node_recovery.go:37-69`, `node_events.go:231-245`, `worker.go:230-245` | 12 (`runner_false_death`) | zombie after eviction fixed by ticket 3; false-death cleanup open until tickets 5 and 7; a requeue routed to the still-running worker waits for the orphan sweep until tickets 5 to 7 |
| L1 | `removeWorkerFromMaps` deletes the cleanup lock without checking the holder. An eviction on one goroutine deletes the lock another node just acquired, and both nodes hold it. | `node_cleanup.go:47` | 11 (`lock_asis`) | fixed by ticket 1 |
| L2 | `acquireCleanupLock` deleted a stale lock unconditionally, then called `SetIfNotExists`. Two nodes that saw the same stale value could both acquire. | `node_membership.go` before `ba2af97c` | 17 (`lock_pre_release_only`) | fixed on main by `ba2af97c` |

`ba2af97c` replaces the stale-lock delete with `TestAndSetEx` on the observed
value (`node_membership.go:62-93`). TLC confirms that this closes L2. Ticket 1
closes L1; `lock_fixed` checks the combination.

Each trace, translated into Go calls:

- **B1.** `DispatchJob(k1)` adds `s1`. `routeWorkerEvent(s1)` sends it to
  `w1`. The event stays unacked past `ackGracePeriod`, so XAUTOCLAIM
  redelivers it and `routeWorkerEvent(s1)` runs again. `w1` calls
  `startJob(s1)` twice. In `runner_asis`, the second router sees `w1` as
  stale and sends the copy to `w2`, which starts it too.
- **B2.** `DispatchJob(k1)` adds `s1`, then times out and releases the guard.
  `DispatchJob(k1)` is admitted again (no payload, no guard) and adds `s2`.
  Both events start the job.
- **B3.** `n1.Close()` requeues `k1` as `s2` and deletes `jobMap[w1]`. Before
  `w2` handles `s2`, the orphan sweep on `n2` runs twice and adds `s3`. `w2`
  starts `k1` twice.
- **B4.** `w1` runs `k1`, and its keep-alive looks stale. `cleanupWorker(w1)`
  requeues `k1` and deletes `w1`, and `w2` starts `k1`. `n1`'s `workerMap`
  replica still lists `w1` when `handleWorkerMapUpdate` runs, so
  `w1.rebalance` stops `k1` and requeues it again. `w2` starts it a second
  time. An earlier run of the model found the mirror order: rebalance first,
  then a cleanup that reads the stale `jobMap[w1]` entry.
- **B6.** `w1` runs `k1`, but its keep-alive looks stale (GC pause, skew or
  Redis latency). `cleanupWorker(w1)` requeues `k1` and deletes `w1`, and
  `w2` starts `k1`. The cleanup never re-checks the keep-alive. When `n1`
  evicts `w1`, `worker.stop` leaves the handler running, so both run until
  `n1` exits.
- **L1.** `n1` cleans up `w1`, which deletes the lock. Stale replicas on both
  nodes show `w1` again. `n1` acquires the lock with `SetIfNotExists`. `n1`'s
  eviction path runs `deleteWorker(w1)`, which deletes `n1`'s own lock. `n2`
  then acquires the lock with `SetIfNotExists`. Both nodes hold it.

## Decision

Add an authoritative owner record per job key in Redis. Every transition of
ownership is one Lua script. A start event is only a request: a start whose
key belongs to another worker, or is already running, does nothing. Duplicate
start events are therefore harmless, whatever created them.

The design has eight parts:

1. **Owner record.** `startJob` claims the key with a compare-and-set.
   Unowned means claim with a new epoch. Owned by this worker means re-enter.
   Owned by another worker means drop the start and ack it.
2. **Release with compare-and-delete.** `stopJob`, `rebalance`, `requeueJob`
   (close) and a failed `handler.Start` release only an owner record that
   names this worker. The epoch counter is never reset.
3. **Atomic cleanup.** `cleanupWorker` runs one script. The script:
   - checks the cleanup-lock token,
   - re-checks the worker's keep-alive against Redis `TIME`,
   - reads the dead worker's keys from the owner record, not from a replica,
   - requeues and releases those keys,
   - deletes the worker.

   A variant that read the `jobMap` replica lost jobs for good in TLC: a stale
   replica showed no keys, and the owner record kept naming the dead worker.
4. **Idempotent start.** `startJob` returns early when `w.jobs` holds the key.
5. **Stop on eviction.** `handleWorkerMapUpdate` calls `handler.Stop` for
   every job of an evicted local worker.
6. **Self-fencing.** A worker whose keep-alive `Set` has not succeeded for
   `workerTTL - fenceMargin` stops its handlers and stops reading its stream.
   It keeps `w.jobs`. After the next successful keep-alive, it resumes only
   the keys whose owner record still names it with the same epoch.
7. **Holder-only lock release.** Only the node that holds a cleanup lock can
   delete it. This fixes L1.
8. **Pending guard until ack.** The dispatch guard stays while its start event
   is unacked. This fixes B2 admission. The claim script adds the start event
   itself, so the guard and its event id are written in one step. Today six
   paths can clear a guard while its event may still be unacked, and all of
   them must check first:
   - the dispatcher after a failed `poolStream.Add` (`node_jobs.go:102`),
   - the dispatcher after its timeout (`node_jobs.go:114`, release at `:123`),
   - the dispatcher after caller cancellation (`ctx.Done`, `node_jobs.go:116`,
     release at `:123`),
   - the dispatcher after a dispatch return (`node_jobs.go:113`, release at
     `:123`). `ackWorkerEvent` sends `evDispatchReturn` before it XACKs the
     pool event (`node_events.go:146-160`), so on a start error or a success
     this release runs while the event is still unacked,
   - the stale-guard replacement in `luaClaimDispatch` (`scripts.go:41-51`),
   - `cleanupStalePendingJobs` (`node_membership.go:29`).

   `ackWorkerEvent` changes to XACK the pool event first and send
   `evDispatchReturn` after. Every dispatcher release then follows the ack,
   and the one conditional release is correct for successes and start errors
   alike.

   See [claimDispatch](#claimdispatch-dispatchjob) and
   [Other changes](#other-changes) for the check.

Epochs are the fencing token for work that outlives a lease. `Job.Epoch`
exposes the epoch to the handler. Pulse writes job state only through
`claimJob` and `releaseJob`, and both check owner and epoch. Application
writes are protected only when the target store checks the epoch. The
application passes `workerID:epoch` to the store, and the store rejects a
write whose epoch is lower than the last one it accepted for the key. Pulse
cannot fence a store that ignores the epoch.

### Verification

The TLC model implements every part above behind the `FIX_*` toggles. The
redesign configurations enable all of them.

| Config | Property | Scope | Result | States (distinct) | Trace |
| --- | --- | --- | --- | ---: | ---: |
| `redesign_crash` | 1, 2, 3 | Crash-only deaths, `MaxEv=3`, `MaxTok=2`, no stop | holds, exhaustive | 82,100 | — |
| `redesign_crash_stop` | 1, 2, 3 | Crash-only deaths, with stop, no close | holds, exhaustive | 14,473,376 | — |
| `redesign_live` | 4 | Crash-only deaths, `MaxStream=1`, no stop | holds, exhaustive | 31,349 | — |
| `redesign_false_death` | 1 | Live workers may look dead, no fencing | **violated** | 126,785 | 10 |
| `redesign_fence_min` | 1, 2, 3 | Self-fencing, bounded pause and skew | holds, exhaustive | 60,530,855 | — |
| `redesign_fence_norecheck` | 1 | Self-fencing, no keep-alive re-check | **violated** | 66,016 | 12 |
| `redesign_pause` | 1 | Fencing plus epochs, unbounded pause | **violated**, inherent | 55,513 | 10 |
| `redesign_pause_writes_min` | 5, 2 | Fencing plus epochs, unbounded pause, no rebalance | holds, exhaustive | 79,236,048 | — |
| `redesign_pause_noepoch` | 5 | Fencing without epoch checks, unbounded pause | **violated** | 140,541 | 11 |
| `redesign_lease_expire_min` | 1, 2 | Fencing, cleanup lease may expire while held | no violation, not exhaustive (depth 27) | 91,603,209 | — |
| `redesign_lease_expire_lock` | 3 | Fencing, cleanup lease may expire while held | **violated**, inherent | 3,433 | 9 |

Properties: (1) `AtMostOneRunner`, (2) `NoDoubleStartSameWorker`,
(3) `CleanupLockMutualExclusion`, (4) `EventuallyRuns`, (5) `NoStaleWrite`.
The `_min` configurations use the reduced fencing scope described in the
README.

With two nodes and crash-only deaths, property 3 holds trivially: only one
node survives to clean up. The fencing configurations check it with both nodes
alive.

What the results mean:

- **Crash-only failures.** The redesign keeps the job on one handler, never
  starts it twice on a worker, and eventually runs every queued job.
- **Live workers that look dead.** An owner record alone cannot stop the old
  handler (`redesign_false_death`).
- **Self-fencing.** With a bounded pause and skew, others can see a worker as
  stale only after its own shorter local lease has expired. Under that
  assumption, self-fencing plus the keep-alive re-check restores property 1.
  Without the re-check, the worker can refresh its keep-alive and start a job
  between the stale snapshot and the cleanup (`redesign_fence_norecheck`).
- **Unbounded pause.** If a process stalls longer than the fence margin (GC,
  VM freeze, suspended container), two handlers run for a while
  (`redesign_pause`). This is unavoidable with leases. Epochs make it safe for
  state kept in stores that check the epoch: no write from a superseded run
  is accepted (`redesign_pause_writes_min`). Without epoch checks, one is
  (`redesign_pause_noepoch`). The model's `JobWrite` stands for a write to a
  store that checks `workerID:epoch`, such as pulse's own `claimJob` and
  `releaseJob` or an application store that opts in. Writes to stores that
  ignore the epoch are not protected, and the model does not claim they are.
- **Lease expiry while held.** Two nodes can both believe they hold the
  cleanup lock when a holder is slower than `workerTTL`
  (`redesign_lease_expire_lock`). This is inherent to TTL locks. The cleanup
  script checks the lock token, so only the current holder's cleanup has any
  effect. No violation of properties 1 or 2 was found in 91.6M states
  (`redesign_lease_expire_min`), but the run was stopped at depth 27 before
  it finished. Finishing it needs a larger machine or a tighter scope, which
  is open question 5.

## Redis Layout

New keys, for pool `P`:

| Key | Type | Content |
| --- | --- | --- |
| `P:owners` | hash | field `jobKey`, value `workerID:epoch` |
| `P:owner-epochs` | hash | field `jobKey`, value last issued epoch (integer, never decremented) |
| `P:protocol` | hash | field `nodeID`, value protocol version (`2`) |
| `P:protocol-version` | string | pool protocol marker (`2`), set by `backfillOwners` |

The pending-guard value in the existing `P:pending-jobs` map changes from
`untilNanos` to `untilNanos:eventID`. `claimDispatch` and the stale sweep
already reject malformed values, so they need the matching parser change in
ticket 4.

The owner record does not need to be replicated: only scripts read it. It is
a plain hash, not an `rmap`. Scripts that change `rmap` content (`jobMap`,
payloads, `workerMap`, keep-alive, cleanup lock) must bump `=rev` and publish
the `rmap` update message, the same way `luaClaimDispatch` does
(`scripts.go:51-55`).

## Scripts

The pseudo-code below uses `KEYS` and `ARGV` informally. `pub(map, op, ...)`
means "update the rmap revision and publish", as in `scripts.go`.

### claimDispatch (DispatchJob)

This replaces `luaClaimDispatch` followed by `poolStream.Add`. The script adds
the start event itself, so the guard is never written without its event id,
and the event is never queued without its guard. The model's `Dispatch` works
the same way.

```lua
-- KEYS: payload content, pending content, pending channel, owners, pool stream
-- ARGV: key, nowNanos, untilNanos, maxLen, jobBytes (marshalJob, built in Go)
if HGET payloads key or HEXISTS owners key then return {3, ""} end
local g = HGET pending key
if g then
  local until, id = parse_guard(g)                 -- "untilNanos:ms-seq"
  if not until then return {4, g} end              -- malformed
  if until >= nowNanos or in_flight(id) then return {2, g} end
end
local id = XADD pool MAXLEN ~ maxLen * n "j" p jobBytes
HSET pending key untilNanos .. ":" .. id; pub(pending, "set", ...)
return {1, id}
```

Details:

- `in_flight(id)` is true only if the event is younger than
  `pendingEventTTL` and may still start:
  - It is listed by `XPENDING pool events id id 1`. Routing does not ack
    an event, and a trim (`WithStreamMaxLen`, `node.go:144`) or XDEL
    leaves the pending entry, so this holds even when the stream no longer
    has the entry. A routed event can still start.
  - Or it is still in the stream (`XRANGE pool id id`) and not yet
    delivered: `id` is greater than the group's `last-delivered-id`
    (`XINFO GROUPS pool`), or the group does not exist.
  - Or it was delivered (`id` is at most `last-delivered-id`) and is in
    neither the stream nor the pending list. Redis 7 and later XAUTOCLAIM
    purges the pending entry of a trimmed id, although a router may already
    have queued the event on a worker stream (issue #385). Redis cannot tell
    this from an event that was acked and then trimmed, so the worker ack
    deletes the guard itself (`luaAckStart`, below).
- **Ack deletes the guard.** `completePendingEvent` acks a start event with
  `luaAckStart`: `XACK`, and in the same script the deletion of the guard
  of the job when the guard names that event. A guard can then name an
  acked event only after an ack that does not start the job (a stale or
  malformed event), and `in_flight` may keep such a guard until
  `pendingEventTTL`. A guard also lasts until `pendingEventTTL` when its
  start event is trimmed before delivery and the group then delivers a
  later entry: the lost event then looks delivered and purged. Without the
  later delivery it is released as before. Both cases hold a guard longer
  than needed, never shorter.
- The age bound keeps a guard from outliving every recovery path. Redis 7
  and later purge deleted entries from the pending list during XAUTOCLAIM,
  but Redis 6.2 keeps them, so a trimmed pending entry may never be acked.
  An event that no sink reads is never delivered. `routeWorkerEvent` acks
  events older than `pendingEventTTL` as stale without starting them, so
  after that age only an event already queued on a worker stream can
  start. The model has no time and does not model this bound.
- Closed gap on Redis 7 and later (issue #385). Before the fix, the guard
  could be released before `pendingEventTTL` when all of these held:
  - Redis 7 or later,
  - `maxQueuedJobs` (default 1000) or more events were added while the
    start event was unacked, so `MAXLEN ~` trimmed it,
  - XAUTOCLAIM ran after `ackGracePeriod` and purged its pending entry.

  A retry was then admitted while the first start was still queued on a
  worker stream or held by a stalled router, which is the B2 double start.
  TLC finds it in the `guard_*_r7` configurations. Three other designs were
  checked. A routed marker set with the worker `XADD` and cleared on the
  ack fails when a router stalls between the sink delivery and the `XADD`.
  The same marker with a pending-entry check before routing holds, but
  needs a routing script that duplicates `Stream.Add` and one more write
  per event. Trimming only below the oldest pending id (`XTRIM MINID`)
  holds, but lets one stuck pending entry grow the pool stream past
  `maxQueuedJobs`, and every pool stream `XADD` would need it. Ack-aware
  trimming (`XADD`/`XTRIM ... ACKED`) needs Redis 8.2. See the
  [model README](../pulse/pool/tla/README.md#pool-stream-trimming-and-the-dispatch-guard-issue-385).
- Stream ids are compared as numbers, never as strings: split `ms-seq`, then
  compare `ms`, then `seq`. Both fit in a Lua double.
- The script needs no Redis version beyond the 6.2 that the sink already
  requires for XAUTOCLAIM. On Redis 6.2, `XAUTOCLAIM` replies with a null
  entry for a deleted or trimmed pending id, which go-redis `XAutoClaim`
  cannot parse, so the sink parses the raw reply and skips the null entry
  (#408). It does not ack the id, so 6.2 pending lists keep such entries,
  and nothing acks them (issue #411). Since #385, `in_flight` no longer
  depends on that entry: acking it leaves the state that a Redis 7 purge
  leaves, which the `guard_*_r7` configurations check. #411 can therefore
  ack deleted ids.
- The script must trim and set expiry exactly as `Stream.Add` does
  (`MaxLen`, `Approx`, and the TTL in the same script through
  `pulse/internal/keyttl`, which emulates `EXPIRE ... NX` for 6.2). A golden
  test compares the entry fields with a `Stream.Add` entry.
- If the client sees an error but the server applied the script, the guard
  and event both exist, and the in-flight check below protects them. If the
  script failed, nothing was written.

### claimJob (startJob, before `handler.Start`)

```lua
-- KEYS: owners, epochs, jobMap content, payload content
-- ARGV: key, workerID, payload
local cur = HGET owners key
if cur then
  local w, ep = split(cur)
  if w ~= workerID then return {0, ""} end        -- owned elsewhere: drop start
  return {2, ep}                                  -- re-enter (w.jobs lost it)
end
local ep = HINCRBY epochs key 1
HSET owners key workerID .. ":" .. ep
append-unique jobMap[workerID] key; pub(jobMap, "set", ...)
HSET payloads key payload;          pub(payloads, "set", ...)
return {1, ep}
```

`startJob` checks `w.jobs` first (part 4). It calls `handler.Start` only on
status 1 or 2, and acks a status 0 start as a success with no effect. If
`handler.Start` fails, it runs `releaseJob` and deletes the payload, as today.

### releaseJob (stop, rebalance, requeue on close, failed start)

```lua
-- ARGV: key, workerID, epoch, deletePayload
if HGET owners key ~= workerID .. ":" .. epoch then return 0 end
HDEL owners key
remove key from jobMap[workerID]; pub(jobMap, ...)
if deletePayload == "1" then HDEL payloads key; pub(payloads, "del", ...) end
return 1
```

`rebalance` and `requeueJob` release with `deletePayload = 0`, then add the
start event. `stopJob` releases with `deletePayload = 1`.

### cleanupWorker

```lua
-- KEYS: owners, cleanup content, keepalive content, workers content,
--       jobMap content, payload content, pool stream, worker stream
-- ARGV: workerID, lockToken, workerTTLms, fenceSlackMs, nodeID
if HGET cleanup workerID ~= lockToken then return {0} end      -- lost the lock
local ka = HGET keepalive workerID
if ka and (redisNowMs() - ka/1e6) <= workerTTLms + fenceSlackMs then
  HDEL cleanup workerID; pub(cleanup, "del", ...)             -- worker is alive
  return {1}
end
local requeued = {}
for key, v in HSCAN owners do
  if owner(v) == workerID then
    HDEL owners key
    XADD poolStream * n "j" p pack_job(key, HGET payloads key, nodeID)
    push(requeued, key)
  end
end
delete workers[workerID], keepalive[workerID], cleanup[workerID] (token held),
       jobMap[workerID]; publish each
DEL workerStream
return {2, requeued}
```

- `pack_job` must produce the bytes of `marshalJob` (little-endian `int32`
  length prefixes). Lua `struct.pack("<i4c0<i4c0<i4c0<i8", ...)` can do this.
  A golden test must pin the encoding.
- `HSCAN` over all owners costs O(jobs). If that is too slow, keep a
  per-worker reverse index `P:owners-by-worker:<id>` (a set), updated in the
  same scripts.

### Other changes

- **Dispatch admission** is the `claimDispatch` script above. Ticket 4 adds
  the script, and ticket 5 adds the `HEXISTS owners` check that refuses a key
  with an owner record.
- **Pending guard.** The guard value becomes `untilNanos:eventID`, written
  only by `claimDispatch`.
  - Every path clears a guard only when `in_flight(id)` is false, through a
    release script that runs the same check:
    - the dispatcher after a client-side error from `claimDispatch` (today's
      add-failure release at `node_jobs.go:102`),
    - the dispatcher after its timeout (`node_jobs.go:114`),
    - the dispatcher after caller cancellation (`node_jobs.go:116`),
    - the dispatcher after a dispatch return, for a success or a start error
      (`node_jobs.go:113`),
    - the stale-guard replacement inside `claimDispatch` (today
      `scripts.go:41-51`),
    - `cleanupStalePendingJobs` (`node_membership.go:29`).
  - **Ack before return.** `ackWorkerEvent` (`node_events.go:146-160`) XACKs
    the pool event before it sends `evDispatchReturn`. Today it sends the
    return first.
    - With the new order, the dispatch-return release always finds the event
      acked and clears the guard. A failed start can then be retried
      immediately.
    - If the node crashes between the XACK and the return, the dispatcher
      times out. Its release finds the event acked and clears the guard, and
      the payload or owner record still refuses a job that started.
  - `FIX_DISPATCH` in the model matches this rule: `Dispatch` writes the guard
    and the event in one step, and no release happens while the event is
    unacked. The model's `WorkerHandle` handles, acks and returns in one step,
    which is the new Go order (XACK, then return, then release). It is not
    today's order.
  - The main configurations never trim the pool stream and have no time.
    The `guard_*` configurations (`TRACK_STREAM`) add trimming,
    XAUTOCLAIM, two-step routing and the `pendingEventTTL` bound. The
    model never fails `handler.Start`, so the start-error retry is covered
    only by the Go tests.
- **The orphan sweep** requeues only keys with no owner record.
- **Lock release** (`removeWorkerFromMaps`) runs a test-and-delete with the
  caller's token. Eviction, `close`, and `RemoveWorker` pass no token and leave
  the lock alone.
- **Resume after self-fencing** calls `claimJob`-style validation. Owner and
  epoch must match. It is read-only and restarts the handler with the same
  epoch.
- **Epoch-checked writes.**
  - Pulse's own per-job writes are the owner record, the `jobMap` entry and
    the payload. They happen only inside `claimJob` and `releaseJob`, which
    already check `workerID:epoch`, so no other pulse write needs a check.
  - For applications, pulse adds `Job.Epoch` and one helper,
    `Worker.CheckOwnership(ctx, key, epoch) error`. The helper is a read-only
    script. On a mismatch, it stops the local handler for that key, as the
    model's rejected `JobWrite` does.
  - The pulse docs describe how to fence a downstream store: keep the
    highest accepted epoch per key and reject lower ones.

## Migration and Compatibility

- `Job` gains an exported `Epoch uint64` field. This is additive.
- The wire format of pool stream events does not change.
- Mixed pools are not supported, because a protocol 1 node ignores owner
  records.
- **Joining.** Protocol 2 nodes register in `P:protocol` when they join. A
  joining protocol 2 node that sees a live entry in the node keep-alive map
  with no protocol entry treats it as a protocol 1 node. It refuses to start
  and names that node in the error.
- **Limitation: the gate works in one direction only.** Released protocol 1
  code never reads `P:protocol`. A protocol 1 node that joins after protocol
  2 nodes, for example during a rollback, is not refused. To detect it:
  - Protocol 2 nodes watch the node keep-alive map (an `rmap`, so updates are
    pushed).
  - When a live node without a protocol entry appears, each protocol 2 node
    stops routing and starting jobs, reports the node through the logger and
    the node's health error, and resumes only after that node's keep-alive
    expires.

  This limits the damage but cannot prevent it: the protocol 1 node can
  already have started duplicates. Rollback follows the same rule as upgrade:
  close every node of one version before starting the other.
- **Upgrade path.** Close all protocol 1 nodes with `Close`, not `Shutdown`,
  so payloads and requeued events survive, then start protocol 2 nodes.
- **Backfill.** A one-time `backfillOwners` script is keyed on a version
  marker, `P:protocol-version`, not on whether `P:owners` exists. When the
  marker is absent, the script:
  - gives each key in `jobMap` of a worker with a live keep-alive ownership
    at epoch 1,
  - sets `P:protocol-version = 2`,
  - does both atomically.

  Keys without a live worker stay unowned, and the orphan sweep requeues
  them. Because the marker and the owner logic ship in the same commit
  (ticket 5), owner records are never written by code that does not also
  maintain them.
- **Shutdown.** `Shutdown` does not delete the new hashes today:
  `cleanupPool` (`node_cleanup.go:126`) only destroys `node.maps()`, and the
  new keys are plain hashes. Ticket 5 extends `cleanupPool`:
  - It deletes `P:owners`, `P:protocol` and `P:protocol-version`.
  - It keeps `P:owner-epochs` on purpose. A downstream store may have
    recorded epochs, and restarting them at 1 would make a new pool's writes
    look stale.

## Implementation Tickets

Each ticket is one atomic commit. Each starts with a failing test that
reproduces the cited TLC trace. Tickets 1 to 4 are independent and are
useful even before the owner record lands.

1. **Holder-only cleanup-lock release (L1).** Done.
   `claimCleanupLock` returns the lock token. `cleanupWorker` passes it to
   `deleteWorker`, and `removeWorkerFromMaps` releases the lock with
   `TestAndDelete` on that token. Eviction, `close`, and `RemoveWorker` pass no
   token and leave the lock alone, as `FIX_LOCK_RELEASE` models. Test:
   `TestEvictionKeepsCleanupLockOfAnotherHolder` replays the 11-step
   `lock_asis` interleaving and ends with exactly one holder.
   - Known limit: a lock whose holder crashes before it finishes, on a worker
     that eviction, `close`, or `RemoveWorker` then removes, is never deleted.
     The worker is gone, so the entry has no effect beyond its size. The model
     keeps the lock in the same case.
2. **Idempotent `startJob` (B1, and B2 to B4 on one worker).** Done.
   `startJob` returns nil without calling the handler when `w.jobs` holds the
   key, so the duplicate is acked as a success with no effect, as
   `FIX_DEDUP` models. Test: `TestRedeliveredStartEventStartsJobOnce` blocks
   the first `handler.Start` past `ackGracePeriod`, so the sink redelivers
   the event to the same worker (the `double_asis` trace). `handler.Start`
   runs once, and both deliveries are acked.
3. **Stop handlers on eviction (B6 zombie).** Done. After `worker.stop`,
   `handleWorkerMapUpdate` calls `Worker.stopHandlers`, which runs
   `handler.Stop` for every job in `w.jobs` and forgets it locally, as
   `FIX_EVICT_STOP` models. It leaves `jobMap` and payloads alone, because
   cleanup has already requeued the jobs. `Worker.jobLock` serializes every
   change to `w.jobs` (start, stop, rebalance, requeue, eviction), so no
   stop or start of a key overlaps another. Test:
   `TestEvictionStopsJobHandlers` replays `runner_false_death`: cleanup on
   another node requeues `w1`'s jobs, and the test checks that the pool
   stream holds a start for each key. It then evicts `w1`, asserts
   `handler.Stop` for each job, and waits until `w2` runs every key. Two
   handlers still run between the cleanup and the eviction; tickets 5 and 7
   close that window.
   - Known limit: a router whose replica still lists `w1` can send a
     requeued start to `w1` while it runs the key. The ticket 2 duplicate
     check acks it without adding `jobMap[w1]` again. After
     `stopHandlers`, only the payload remains, and the key runs nowhere
     until the orphan sweep requeues it (its grace period plus one sweep
     period). Tickets 5 to 7 close this window.
4. **Keep the pending guard until ack (B2).** Done. Replace
   `luaClaimDispatch` and the separate `poolStream.Add` with the
   `claimDispatch` script, which adds the event and writes
   `untilNanos:eventID` in one step. Make every release
   path check `in_flight`, and make `ackWorkerEvent` XACK before it sends
   `evDispatchReturn` (see [Other changes](#other-changes)). The owner-record
   check in `claimDispatch` (`HEXISTS owners key`) is added in ticket 5.
   Tests, each with the first event unacked unless noted:
   - `handler.Start` returns an error, and the caller retries at once. The
     retry is admitted: the event was acked before the return, so the guard
     was cleared.
   - A successful start is acked before its dispatch return arrives, and the
     guard is cleared.
   - The dispatcher times out and the caller retries. The retry returns
     `ErrJobExists`.
   - The caller cancels its context and retries. The retry returns
     `ErrJobExists`.
   - The client sees an error after the script ran on the server. The script
     is called directly and its reply discarded. A retry returns
     `ErrJobExists`.
   - The guard TTL passes and `claimDispatch` sees a stale guard. It refuses
     admission.
   - `cleanupStalePendingJobs` runs after the guard TTL. The guard stays.
   - The event is trimmed from the pool stream while still pending. Every
     path treats the guard as in flight, until the event is older than
     `pendingEventTTL`.
   - Stream ids with a multi-digit sequence (`5-10` against `5-9`) compare as
     numbers.
   - After the event is acked, each path clears the guard.
   - A golden test shows the script's stream entry matches a `Stream.Add`
     entry.

   Compatibility: an older node rejects the new guard format as malformed
   (`dispatchMalformedPending`), so its `DispatchJob` calls fail while mixed
   versions run. To avoid this, either accept both formats for one release
   before writing the new one, or document that a rolling upgrade is not
   supported.

   Delivered:
   - `luaClaimDispatch` adds the start event (`MAXLEN ~` as `Stream.Add`;
     the pool stream has no TTL) and writes `untilNanos:eventID` in one
     step. `luaReleaseDispatch` clears a guard only when its value is
     unchanged and `in_flight` is false. The dispatcher (timeout,
     cancellation, return), the stale replacement in `luaClaimDispatch`
     and `cleanupStalePendingJobs` all use it. A client-side error from
     `claimDispatch` releases nothing.
   - `in_flight` checks the pending list first, so a trimmed event that
     is still pending stays in flight. A missing sink group means not yet
     delivered, so in flight. Every in-flight state ends once the event is
     older than `pendingEventTTL`; see
     [claimDispatch](#claimdispatch-dispatchjob).
   - Residual gap: after `pendingEventTTL`, an event already on a worker
     stream can still start. The Redis 7 early release of a trimmed
     pending event (issue #385) is closed: a delivered event that left the
     stream and the pending list stays in flight, and the worker ack
     deletes the guard (`luaAckStart`). Tests:
     `TestDispatchGuardInFlight` (purged and trimmed states),
     `TestAckStartDeletesGuard`,
     `TestDispatchGuardClearedByAckOfTrimmedStart`,
     `TestIdleClaimAfterTrimmedStart` and
     `TestRedisDispatchGuardAfterAutoClaimOfTrimmedStart`, on miniredis
     and on Redis 6.2 and 7.4.
   - `ackWorkerEvent` XACKs the pool event, then sends `evDispatchReturn`.
   - The commit before this ticket stopped losing a dispatch return that
     arrives before `dispatchJob` registers for it. The dispatch tests
     here depend on it under load.
   - Tests: `dispatch_guard_test.go` (dispatch paths, the
     `double_noredeliver` trace) and `dispatch_guard_script_test.go`
     (every delivery state against every release path, id order, formats,
     the entry golden).
   - Compatibility decision (conservative option): new nodes read both
     guard formats. A guard without an event id keeps its TTL-only
     meaning. Older nodes still reject the new format, so a rolling upgrade
     across this ticket is not supported: close every older node before
     starting new ones, as for ticket 5.
5. **Owner record, atomic cleanup, protocol gate and backfill.**
   - Add `claimJob` and `releaseJob`, and use them in stop, rebalance,
     requeue and a failed start.
   - Replace the body of `cleanupWorker` with the cleanup script: token
     check, keep-alive re-check, owners read from Redis.
   - Make `claimDispatch` refuse a key that has an owner record
     (`HEXISTS owners key`). Owner records exist from this ticket on.
   - Add `Job.Epoch`, `P:protocol` registration, the join-time protocol 1
     check, the protocol 1 watch, `backfillOwners` keyed on
     `P:protocol-version`, and the `cleanupPool` deletions.

   This must be one commit, because two partial orders both break:
   - Owner claims with the old cleanup: a dead worker keeps its keys
     forever, and every restart of those keys is dropped.
   - A backfill without owner maintenance: it leaves stale records that later
     drop valid starts.

   Tests:
   - B1 across workers: two routers, two workers, one `handler.Start`.
   - Both B4 orders, each with one start on `w2`.
   - The `redesign_fence_norecheck` trace: keep-alive refreshed after the
     snapshot, so cleanup aborts.
   - The lost-job trace from the replica variant.
   - A pinned encoding golden for `pack_job`.
   - A protocol 1 node keep-alive refuses a joining protocol 2 node. A
     protocol 1 keep-alive that appears later pauses the protocol 2 nodes.
   - Backfill runs once per marker. `Shutdown` removes owners and protocol
     keys and keeps epochs.
6. **Owner-aware orphan sweep (B3, B5).** The orphan sweep requeues only keys
   with no owner record. Tests: the B3 close and orphan trace, and the B5 lag
   trace (payload replica ahead of `jobMap`), each with a single start.
7. **Self-fencing and resume.** Add the keep-alive failure timer,
   `handler.Stop` on fence, and owner-and-epoch validation on resume. Tests:
   fence, cleanup, then resume drops the job; fence with no cleanup, then
   resume restarts it with the same epoch.
8. **Application fencing API and documentation.** Pulse's own writes are
   already checked by ticket 5. This ticket adds
   `Worker.CheckOwnership(ctx, key, epoch)`, which stops the local handler on
   a mismatch, and documents how to fence a downstream store with
   `Job.Epoch`. Tests: a check with a superseded epoch fails and stops the
   handler; the current epoch passes.

## Test Strategy

- Use `miniredis` for all script tests. It runs Lua through gopher-lua. The
  same tests also run against real Redis 6.2 and 7.4 in the opt-in tier
  (`make test-pulse-redis` and the `pulse-redis` CI job, issue #383). Pin
  behaviour that differs between versions, such as `XAUTOCLAIM` on deleted
  ids, in `pulse/pool/redis_semantics_test.go`.
- Make interleavings deterministic with seams instead of sleeps:
  - Pass observed replica values in explicitly (`claimCleanupLock` already
    does this since `ba2af97c`).
  - Add a clock interface for `isWithinTTL` and the keep-alive timer.
  - Drive `routeWorkerEvent`, `startJob`, `cleanupWorker` and
    `handleWorkerMapUpdate` directly, in the order of the TLC trace.
- Each regression test names its TLC configuration and cites the trace steps
  in a comment. The TLA+ model stays the reference, so a design change updates
  the model and its configurations in the same commit.
- Keep one end-to-end miniredis test per ticket that runs two nodes with real
  rmap replication, to catch integration breakage that the seams hide.

## Open Questions

1. **Fence margin.** How large should `fenceMargin` be? It must cover clock
   skew between nodes and Redis, plus expected pauses. A proposed default is
   `workerTTL/4`. Self-fencing on every Redis blip longer than `3/4*workerTTL`
   restarts handlers. Is that acceptable, or should fencing need several
   consecutive failures?
2. **Keep-alive re-check clock.** Keep-alive values are node-clock
   nanoseconds. The re-check compares them to Redis `TIME`. Either store
   keep-alive as Redis `TIME` (set by script), or add skew slack.
3. **Cleanup script size.** Requeue inside Lua needs `pack_job`. The
   alternative is a two-phase cleanup: the script releases and returns the
   keys, then Go adds the events, and the orphan sweep recovers a failed add.
   It is simpler, but liveness then depends on the sweep. It is not modeled
   yet.
4. **Epoch API naming.** Ticket 8 proposes `Worker.CheckOwnership`. Should it
   instead live on `Job` or `Node`, and should it return the current owner
   when the check fails?
5. **Model coverage.**
   - The lease-expiry check of the redesign (`redesign_lease_expire_min`) is
     not exhaustive.
   - The self-fencing scope freezes the `jobMap` and payload replicas and has
     no crash, close, stop or orphan sweep.
   - Liveness under self-fencing is not checked. Unbounded fence and unfence
     cycles break liveness by definition, so the check needs a fairness
     assumption on keep-alive recovery.
   - The two-phase cleanup variant (question 3) is not modeled.

   Before ticket 7, finish the lease-expiry run and add a liveness
   configuration with fencing.

The pending-guard question is decided: keep the guard until the event is
acked (part 8, ticket 4). The owner record would make duplicate admissions
harmless for ownership, but `DispatchJob` promises `ErrJobExists` for a key
that is already dispatched. Without the guard, a second caller's start would
be dropped by `claimJob` (status 0) and acked as a success. The caller would
get `nil` for a start that was dropped, instead of `ErrJobExists`.
