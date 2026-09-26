# TLA+ model of job ownership in `pulse/pool`

`PoolOwnership.tla` models how a job key is admitted, routed, started,
rebalanced, requeued and recovered across pool nodes and workers. It covers
the code on `main` at commit `967f4fbe`. Behind `FIX_*` toggles, it also
covers the owner-record redesign.

The bugs found, the redesign and the implementation plan are in
[`roadmap/pulse-pool-ownership.md`](../../../roadmap/pulse-pool-ownership.md).
This file describes the model and how to run it.

## Files

| File | Purpose |
| --- | --- |
| `PoolOwnership.tla` | The model: state, actions, invariants, liveness, toggles. |
| `PoolOwnership.cfg` | Main as written, all safety invariants, full scope. |
| `cfg/*.cfg` | One configuration per question. See [Results](#results). |
| `cfg/guard_*.cfg` | Pool stream trimming and the dispatch guard (issue #385). See [Pool stream trimming](#pool-stream-trimming-and-the-dispatch-guard-issue-385). |
| `cfg/requeue_*.cfg` | Lost requeued start events (issue #416). See [Lost requeued starts](#lost-requeued-starts-issue-416). |
| `WorkerEventAck.tla` | How a node matches a worker ack to the pool event it routed (issue #381). See [Worker event acks](#worker-event-acks-workereventacktla). |
| `cfg/ack_*.cfg` | One `WorkerEventAck` configuration per routing design. |

## Running TLC

TLC ships in `tla2tools.jar` (tested with TLC 2.19 on Java 25).

```sh
curl -L -o /tmp/tla2tools.jar \
  https://github.com/tlaplus/tlaplus/releases/latest/download/tla2tools.jar
cd pulse/pool/tla
java -XX:+UseParallelGC -cp /tmp/tla2tools.jar tlc2.TLC \
  -workers 1 -deadlock -metadir /tmp/tlc-states \
  -config cfg/lock_asis.cfg PoolOwnership.tla
```

- `-deadlock` is required. A state with no enabled action is a normal end
  state here (for example, all event ids in use), not a bug.
- `-metadir` keeps TLC's `states/` directory out of the source tree.
- `-workers 1` gives a strict breadth-first search. The first counterexample
  is then a shortest one, and state counts are reproducible. With
  `-workers auto`, the violating trace, the distinct-state count at the stop,
  and the reported depth can differ from run to run.
- Violating configurations finish in seconds to a few minutes.
- Exhaustive configurations run much longer:
  - `lock_fixed` and `redesign_crash_stop` take up to about an hour with one
    worker.
  - The `redesign_*_min` fencing configurations need 60 to 80 million states.
    Run them with `-workers auto`. On 10 cores they take 20 to 30 minutes.

## Abstraction

### Topology

There are two nodes (`n1`, `n2`), two workers (`w1` on `n1`, `w2` on `n2`) and
one job key `k1`. One key is enough for every bug found. A second key only
multiplies the state space.

### Redis state (authoritative variables)

| Variable | Go structure |
| --- | --- |
| `wmap` | `workerMap` (`"live"`, `"dash"` for the `"-"` marker, `"absent"`) |
| `ka`, `kaStale` | `workerKeepAliveMap` entry, and whether it is older than `workerTTL` |
| `cl`, `staleTok` | `workerCleanupMap` lock token, and which tokens are older than `workerTTL` |
| `jm` | `jobMap` (worker to set of keys) |
| `pl` | `jobPayloadMap` (presence only) |
| `pend` | `jobPendingMap` guard (holds the dispatch event id) |
| `pool` | Unacked events in the pool stream (`start` or `stop`). With `TRACK_STREAM`, each record also has its Redis state `st` and a `disp` flag for dispatch starts. |
| `routed` | `GUARD_DESIGN` `marker` and `marker_check` only: the routed marker |
| `wstream`, `wsExists` | Worker streams and whether each exists |
| `owner`, `ownerEp` | Redesign only: the owner record and the per-key epoch counter |

### Local state (per node or per worker)

| Variable | Go structure |
| --- | --- |
| `rep[n][m]` | Node `n`'s `rmap` replica of `wmap`, `cl`, `jm` or `pl` |
| `jobs[w]` | `Worker.jobs` |
| `running[w][k]` | Number of `handler.Start` calls minus `handler.Stop` calls (capped at 2) |
| `runEp[w][k]` | Redesign: the epoch the worker holds for its run |
| `stopped[w]`, `fenced[w]` | `Worker.stop` was called; redesign: the worker has fenced itself |
| `cstate[n]` | Progress through `acquireCleanupLock` and `cleanupWorker`, including the observed lock value `seen` |
| `inbox[n]` | `TRACK_STREAM` only: the pool event node `n`'s router read from the sink and has not yet added to a worker stream |
| `orphanSeen[n]` | `node.orphanedPayloads` first-seen marks |
| `nodeUp[n]` | Node process is running |
| `gen`, `runGen`, `staleWrite` | History variables for `NoStaleWrite` |

### Replication

Each `rmap` has its own pub/sub subscription (`rmap/map_runtime.go:70`), so
maps replicate independently. The model gives each node a replica of each map.
`Replicate(n, m)` copies the current Redis value. A replica can therefore be
any earlier version of its map. There is no ordering between maps. The
keep-alive map is not replicated separately. Its lag only matters as "a live
worker looks dead", which `FALSE_DEATH` already covers.

### Time

Time has no clock. Actions make "older than the TTL" true at any moment:

- `KeepAliveExpire` makes a keep-alive stale. By default this can happen only
  to a crashed or stopped worker.
  - With `FALSE_DEATH = TRUE` it can also happen to a live worker (GC pause,
    clock skew, Redis latency).
  - With `FIX_SELF_FENCE` and `LEASE_BOUND`, it can happen to a live worker
    only after that worker has fenced itself. This models a bounded pause and
    skew.
- `LockExpire` makes a cleanup lock stale. By default this happens only when no
  node is working under that lock. With `LEASE_EXPIRE_HELD = TRUE` it can also
  happen while the holder is still working.
- `DispatchRelease` can clear a pending guard at any moment. This covers the
  dispatcher's timeout and the stale-guard paths.
- The orphan grace period is two sweeps by the same node.

### Hashing

`Owner(k, S)` picks the first worker of `WOrder` that is in `S`. The Go code
uses jump hashing over a list sorted by creation time. Every result depends
only on one property: nodes with different views of the active workers can
pick different workers.

### Bounds

`MaxEv` bounds how many event ids are in use at the same time. `MaxTok` does
the same for lock tokens. Ids and tokens are recycled once nothing refers to
them, so these bounds limit concurrency, not the length of a history.
`MaxStream` bounds a worker stream. `MaxEpoch` bounds epochs and handler
starts per key when epochs are tracked.

### Simplifications

- The worker's ack and the router's `XACK` happen in the same step as
  `startJob`, unless `SPLIT_ACK` is set. Redelivery can still happen before
  the worker handles the event, which is the window every redelivery trace
  uses.
- `NodeClose` is atomic, and `handler.Start` never fails.
- `pendingEventTTL` is not modeled. `routeWorkerEvent` drops events older than
  2 minutes (`node_events.go:61`). Without that filter, the model allows
  routing that Go refuses, so the model has more interleavings than Go. No
  reported trace depends on that. In every counterexample, each event is
  routed well within 2 minutes of its creation. The longest wait before a
  routing is the dispatch timeout in B2 (`2*ackGracePeriod`, 40s by default).
  Since roadmap ticket 4, Go also releases a pending guard once its event
  is older than `pendingEventTTL`, even if the event is still unacked.
  `FIX_DISPATCH` releases only after the ack, so it covers Go up to that
  age. The `guard_*` configurations add pool stream trimming, XAUTOCLAIM
  and the TTL (`TRACK_STREAM`); see
  [Pool stream trimming](#pool-stream-trimming-and-the-dispatch-guard-issue-385).
- Only one node can crash. With two nodes and crash-only deaths, only one node
  survives, so `CleanupLockMutualExclusion` is checked meaningfully only in
  configurations where live workers can look dead.
- When all event ids or tokens are in use, an action that needs a new one is
  disabled. This removes behaviors. It never adds false ones.

## Toggles

### Environment

| Toggle | Meaning |
| --- | --- |
| `FALSE_DEATH` | A live worker's keep-alive can look stale. |
| `LEASE_BOUND` | With `FIX_SELF_FENCE`: others see a live worker as stale only after it fenced itself. |
| `LEASE_EXPIRE_HELD` | A cleanup lock can go stale while its holder still works. |
| `REDELIVER_INFLIGHT` | The sink can redeliver an unacked event while an earlier copy is still queued (XAUTOCLAIM after `ackGracePeriod`). |
| `ORPHAN_GRACE_COVERS_LAG` | Replicas catch up within the orphan grace period (`2*workerTTL`). |
| `ALLOW_PARTIAL` | `poolStream.Add` can fail inside `cleanupWorker`. |
| `ENABLE_*` | Turn stop, close, crash, rebalance, orphan requeue or handler writes on or off to narrow a search. |
| `PRE_BA2AF97C` | Cleanup lock as before `ba2af97c`: unconditional `Delete`, then `SetIfNotExists`. |
| `TRACK_STREAM` | Pool stream detail: delivery state per entry, `MAXLEN` trimming, two-step routing. `FALSE` gives the model as before issue #385. |
| `REDIS7` | XAUTOCLAIM purges the pending entry of a trimmed id (Redis 7 and later). Also stands for the sink acking deleted ids on every version (issue #411), which is the same state change. |
| `ENABLE_TTL` | `pendingEventTTL` can pass for an event that no router or worker stream holds. |
| `GUARD_DESIGN` | How `in_flight` tells a start event that can still start from one that cannot. See [Pool stream trimming](#pool-stream-trimming-and-the-dispatch-guard-issue-385). |
| `REQUEUE_DESIGN` | How a job whose requeued start event is lost gets back to a worker. See [Lost requeued starts](#lost-requeued-starts-issue-416). |
| `SPLIT_ACK` | The router's `XACK` of a start (`luaAckStart`, which deletes the guard) is a later step (`AckStart`) than the worker's `startJob`. `FALSE` keeps them in one step. |

The cleanup-lock configurations also use the state constraint `LockScope`:
only `w1` can look dead.

### Fixes and redesign steps (`FALSE` means main as written)

| Toggle | Change |
| --- | --- |
| `FIX_LOCK_RELEASE` | `removeWorkerFromMaps` deletes the cleanup lock only when the caller holds its token. |
| `FIX_DEDUP` | `startJob` returns early when `w.jobs` holds the key. |
| `FIX_DISPATCH` | The pending guard stays while its dispatch event is unacked. |
| `FIX_EVICT_STOP` | Eviction calls `handler.Stop` for every local job. |
| `FIX_OWNER` | Owner record: a claim script in `startJob`, compare-and-delete on stop, rebalance and close, and one token-checked cleanup script that reads owners from Redis. Dispatch and orphan requeue skip owned keys. |
| `FIX_RECHECK` | The cleanup script aborts when the worker's keep-alive is fresh in Redis. |
| `FIX_SELF_FENCE` | A worker that cannot refresh its keep-alive stops its handlers. It resumes a job only if owner and epoch still match. |
| `FIX_EPOCH` | Handler writes carry the epoch, and a stale-epoch write is rejected. |

## Action map (model to Go, `main` at `967f4fbe`)

| Action | Go |
| --- | --- |
| `Dispatch` | `dispatchJob` (`node_jobs.go:89`): `luaClaimDispatch` (`scripts.go:21`), then `poolStream.Add` (`node_jobs.go:99`). Since roadmap ticket 4 (`FIX_DISPATCH`), one `luaClaimDispatch` script adds the event and writes the guard. |
| `DispatchRelease` | `releaseDispatchPending` on ack or timeout (`node_jobs.go:112-123`). Stale guard replaced in `luaClaimDispatch` (`scripts.go:41-51`) or swept by `cleanupStalePendingJobs` (`node_membership.go:29`). Since ticket 4, every path clears a guard only when its event is not in flight, and `ackWorkerEvent` XACKs before the dispatch return. |
| `StopJob` | `node_jobs.go:207` |
| `Route` | `routeWorkerEvent` (`node_events.go:59-93`): `activeWorkers` on replicas (`:72`), hash (`:76`), `Add` with `WithOnlyIfStreamExists` (`:83`). Redelivery comes from XAUTOCLAIM with `MinIdle = ackGracePeriod` (`streaming/sink_consumers.go:308-316`). |
| `WorkerHandle` | `handleEvents` (`worker.go:186`), then `startJob` (`worker.go:248-273`) or `stopJob` (`worker.go:277-293`). Includes the ack path: `ackPoolEvent` (`worker.go:313`), then `ackWorkerEvent` (`node_events.go:134-163`). |
| `KeepAlive` | `Worker.keepAlive` (`worker.go:331-351`). `Set` recreates a deleted entry. |
| `SelfFence`, `JobWrite` | Redesign only (roadmap). |
| `Evict` | First loop of `handleWorkerMapUpdate` (`node_events.go:231-245`): `deleteWorker`, then `worker.stop` (`worker.go:230-245`). |
| `Rebalance` | Second loop of `handleWorkerMapUpdate` (`node_events.go:248-256`), then `Worker.rebalance` (`worker.go:354-390`). |
| `CleanupBegin` | Candidate selection on replicas in `cleanupInactiveWorkers` (`node_recovery.go:37-65`). Then `acquireCleanupLock` reads the lock from the replica (`node_membership.go:53`) and rejects a fresh one (`:78`). |
| `CleanupAcquire` | `claimCleanupLock` (`node_membership.go:62-93`): `SetIfNotExists` when nothing was seen (`:65`), otherwise `TestAndSetEx(seen, now)` (`:82`). |
| `CleanupFinish` | As coded: `cleanupWorker` (`node_recovery.go:145-195`) reads `GetValues` (`:145`) and `JobPayload` (`:161`) from replicas, calls `poolStream.Add` (`:179`), returns early after a partial failure (`:186-188`), then `deleteWorker` (`:193`). `deleteWorker` calls `removeWorkerFromMaps` (`node_cleanup.go:40-62`), which deletes the cleanup lock without a check (`:47`). With `FIX_OWNER`: the cleanup script from the roadmap. |
| `Orphan`, `OrphanClear` | `requeueOrphanedPayloads` (`node_recovery.go:80-134`). |
| `NodeClose` | `close(shutdown=false)` (`node_jobs.go:311-380`): `requeueAllJobs` (`:360`, `worker.go:394`, `:500`), then `removeWorker` (`:368`). |
| `NodeCrash` | Process death. The node's handlers stop, and Redis state stays as it was. |
| `Deliver`, `RouteIn`, `RouteDrop` | `TRACK_STREAM` only. The sink delivers an entry that is in the stream (`XREADGROUP`, or XAUTOCLAIM of a pending entry), then `routeWorkerEvent` adds it to a worker stream (`node_events.go:91`) or fails and leaves it pending. |
| `Trim` | `MAXLEN ~ maxQueuedJobs` on any pool stream `XADD` (`scripts.go` `luaClaimDispatch`, `Stream.Add`). |
| `AutoClaim` | The sink idle check reaches a trimmed pending entry (`streaming/sink_claim.go`). |
| `EventExpire` | `pendingEventTTL` passes: `in_flight` is false by age, and routing acks the event as stale (`node_events.go:61`). |
| `AckStart` | `SPLIT_ACK` only. The worker's ack reaches the router node, which runs `luaAckStart` (`ackRoutedEvent`). |

## Properties

1. `AtMostOneRunner`: for each key, at most one worker on a live process has
   `running > 0`. A worker that is stopped but whose process is alive counts,
   because its handler keeps running.
2. `NoDoubleStartSameWorker`: `running[w][k] <= 1`.
3. `CleanupLockMutualExclusion`: two live nodes are never both in the
   `holding` phase for the same worker.
4. `EventuallyRuns` (liveness, under `FairSpec`): a queued start event for `k`
   eventually results in `k` running on a live worker. Fairness covers the
   system's own progress (replication, routing, worker loops, dispatch
   return, cleanup, orphan sweeps, time passing for dead workers and abandoned
   locks). It does not cover faults or client calls.
5. `NoStaleWrite`: no handler write from a superseded run is accepted. A run
   is superseded when a later `handler.Start` for the same key has happened.
6. `GuardHeld` (`TRACK_STREAM`): the dispatch guard names its start event
   while that event can still start the job: while a live router holds it, a
   live worker's stream queues it, or the sink can still deliver it.
7. `GuardReleased` (liveness, under `FairGuardSpec`): every dispatch guard is
   eventually released, so no guard leaks. The fairness adds delivery,
   routing, XAUTOCLAIM and, per event id, the TTL.
8. `JobRecovered` (liveness, under `FairJobSpec`): once the faults stop, a
   job that has a payload (it started and was not stopped) runs again on a
   live worker, even if one of its start events was lost. See
   [Lost requeued starts](#lost-requeued-starts-issue-416).
9. `OwnerReached` (liveness, under `FairJobSpec`): once the faults stop, a
   job that runs on a worker other than its owner among the workers that
   are really active stops running there. `StopJob` and `NotifyWorker` route
   to the owner, so a job left elsewhere cannot be stopped or notified.

## Results

- **Trace** is the number of states in the counterexample printed by a
  `-workers 1` run, including the initial state.
- **States** is the number of distinct states when TLC stopped. For an
  exhaustive run, that is the whole bounded state space.

### Main as written (`967f4fbe`)

| Config | Property | Scope | Result | States | Trace |
| --- | --- | --- | --- | ---: | ---: |
| `PoolOwnership.cfg` | 1, 2, 3 | full | **violated** (2), B1 | 1,703 | 6 |
| `cfg/double_asis` | 2 | full | **violated**, B1 | 1,703 | 6 |
| `cfg/double_noredeliver` | 2 | no in-flight redelivery | **violated**, B2 | 14,030 | 8 |
| `cfg/double_orphan` | 2 | also `FIX_DISPATCH`, grace covers lag | **violated**, B3 | 639,949 | 12 |
| `cfg/double_rebalance` | 2 | same, without orphan, close, stop | **violated**, B4 | 836,010 | 15 |
| `cfg/runner_asis` | 1 | full | **violated**, B1 across workers | 5,655 | 7 |
| `cfg/runner_dispatch_retry` | 1 | `FIX_DEDUP`, no in-flight redelivery | **violated**, B2 across workers | 42,309 | 9 |
| `cfg/runner_orphan_lag` | 1 | also `FIX_DISPATCH` | **violated**, B5 | 93,909 | 10 |
| `cfg/runner_false_death` | 1 | also grace covers lag | **violated**, B6 | 612,039 | 12 |
| `cfg/runner_crash_only` | 1, 2 | as `runner_false_death`, crash-only deaths, `MaxEv=3`, `MaxTok=2`, no stop | holds, exhaustive | 603,857 | — |
| `cfg/lock_asis` | 3 | cleanup only (`LockScope`) | **violated**, L1 | 4,252 | 11 |
| `cfg/lock_pre_ba2af97c` | 3 | same, `PRE_BA2AF97C` | **violated**, L1 | 4,252 | 11 |
| `cfg/lock_pre_release_only` | 3 | `PRE_BA2AF97C` + `FIX_LOCK_RELEASE` | **violated**, L2 | 90,638 | 17 |
| `cfg/lock_fixed` | 3 | main CAS + `FIX_LOCK_RELEASE` | holds, exhaustive | 12,902,828 | — |
| `cfg/lock_fixed_expire` | 3 | same, lease may expire while held | **violated**, inherent to leases | 678 | 8 |
| `cfg/asis_live` | 4 | crash-only deaths, no partial cleanup | holds, exhaustive | 515,413 | — |

Two readings of these results:

- `lock_pre_release_only` fails and `lock_fixed` holds. The only difference
  is the `ba2af97c` CAS, so that commit closes L2.
- `lock_asis` fails on `967f4fbe`, because L1 is open there. Ticket 1 of the
  roadmap implements `FIX_LOCK_RELEASE` in Go, which `lock_fixed` checks.

### Owner-record redesign

All redesign configurations enable `FIX_LOCK_RELEASE`, `FIX_DEDUP`,
`FIX_DISPATCH`, `FIX_EVICT_STOP`, `FIX_OWNER` and `FIX_RECHECK`, plus the
toggles listed in the scope column.

| Config | Property | Scope | Result | States | Trace |
| --- | --- | --- | --- | ---: | ---: |
| `cfg/redesign_crash` | 1, 2, 3 | crash-only deaths, `MaxEv=3`, `MaxTok=2`, no stop | holds, exhaustive | 82,100 | — |
| `cfg/redesign_crash_stop` | 1, 2, 3 | crash-only deaths, with stop, no close | holds, exhaustive | 14,473,376 | — |
| `cfg/redesign_live` | 4 | crash-only deaths, `MaxStream=1`, no stop | holds, exhaustive | 31,349 | — |
| `cfg/redesign_false_death` | 1 | live workers may look dead, no self-fencing | **violated** | 126,785 | 10 |
| `cfg/redesign_fence_min` | 1, 2, 3 | self-fencing, `LEASE_BOUND`, fencing scope | holds, exhaustive | 60,530,855 | — |
| `cfg/redesign_fence_norecheck` | 1 | self-fencing, `LEASE_BOUND`, without `FIX_RECHECK` | **violated** | 66,016 | 12 |
| `cfg/redesign_pause` | 1 | self-fencing and epochs, unbounded pause | **violated**, inherent to leases | 55,513 | 10 |
| `cfg/redesign_pause_writes_min` | 5, 2 | same, fencing scope without rebalance | holds, exhaustive | 79,236,048 | — |
| `cfg/redesign_pause_noepoch` | 5 | self-fencing without `FIX_EPOCH`, unbounded pause | **violated** | 140,541 | 11 |
| `cfg/redesign_lease_expire_min` | 1, 2 | fencing scope, lease may expire while held | no violation, **not exhaustive** (stopped at depth 27) | 91,603,209 | — |
| `cfg/redesign_lease_expire_lock` | 3 | self-fencing, lease may expire while held | **violated**, inherent to leases | 3,433 | 9 |

The fencing scope is `MaxEv=2`, `MaxTok=3`, `MaxStream=1`, with no stop,
close, crash or orphan sweep. Only `workerMap` and cleanup-lock replicas
update. Under `FIX_OWNER` with the orphan sweep off, the `jobMap` and payload
replicas only select cleanup candidates, so freezing them removes behaviors
and never adds any.

The three violating fencing traces (`redesign_fence_norecheck`,
`redesign_pause`, `redesign_pause_noepoch`) use none of the disabled
features. They are also counterexamples in the fencing scope.

`redesign_pause_writes_min` also turns off rebalance. A rebalancing worker
stops its handler before it releases ownership, so rebalance creates no
superseded run that could write. A run of the same configuration with
rebalance enabled found no violation in 64,015,268 distinct states. It
reached depth 28 and was stopped before it finished.

The `_min` exhaustive runs used `-workers auto`. For an exhaustive run, the
distinct-state count is the whole reachable space, so it does not depend on
the number of workers.

The roadmap document translates each trace into Go calls.

### Pool stream trimming and the dispatch guard (issue #385)

The `guard_*` configurations set `TRACK_STREAM`. Each pool stream record
then has a Redis state: `new` (in the stream, undelivered), `pend`
(delivered, pending), `trim` (trimmed by `MAXLEN` while pending) and
`purged` (its pending entry purged by XAUTOCLAIM, `REDIS7`). An acked
record, or one trimmed before delivery, is dropped. Routing is two steps,
`Deliver` and `RouteIn`, so a router can stall with a delivered event for
longer than `ackGracePeriod`. `Trim` can remove any entry at any time, which
over-approximates `MAXLEN ~` (it trims oldest first). `EventExpire` models
`pendingEventTTL`. It fires only for an event that no router or worker
stream holds, the same assumption as the existing residual gap in the
roadmap: the TTL exceeds the time an event waits on a worker stream.

`GUARD_DESIGN` selects how `in_flight` decides:

| Design | `in_flight(id)` | Other change |
| --- | --- | --- |
| `asis` | pending, or in the stream and undelivered (main before #385) | — |
| `marker` | `asis`, or a routed marker exists | `RouteIn` sets the marker with the worker `XADD`; the worker ack clears it. |
| `marker_check` | same as `marker` | `RouteIn` also refuses an event whose pending entry is gone. |
| `minid` | `asis` | A trim never passes the oldest pending id (`XTRIM MINID`). |
| `ackclear` | `asis`, or delivered and in neither the stream nor the pending list | The worker ack deletes the guard that names the event, in the `XACK` script. |

`REDIS7 = FALSE` is Redis 6.2 before issue #411: XAUTOCLAIM keeps a
trimmed pending entry, so the model has no step for it. `REDIS7 = TRUE`
also stands for the sink acking deleted ids, which it does on every Redis
version since #411: both leave a delivered event in neither the stream nor
the pending list, once the entry is idle for `ackGracePeriod`.

Scopes. All configurations use `MaxEv=3`, `MaxTok=2`, `MaxStream=1`,
`REDELIVER_INFLIGHT = FALSE` and the fixes of tickets 1 to 4
(`FIX_LOCK_RELEASE`, `FIX_DEDUP`, `FIX_DISPATCH`, `FIX_EVICT_STOP`). With
in-flight redelivery, the first worker ack of one copy ends the guard while
another copy is queued. That is B1 across workers (ticket 5), not a guard
defect.

- Guard scope (`guard_<design>_r6|r7`): crash-only deaths, with stop, close
  and crash, no orphan sweep, rebalance or partial cleanup. The omitted
  features add requeue starts, which carry no guard, and the cleanup of a
  live worker, which the full scope covers up to its bound.
- End-to-end scope (`guard_double_*`): `FIX_DEDUP = FALSE`, as in
  `double_noredeliver`, and no orphan, rebalance, close, crash or false
  death. With close and crash enabled, TLC first finds the known
  close-plus-cleanup double requeue (17 states, both Redis versions),
  which is not a guard defect.
- Liveness scope (`guard_live_*`): as `asis_live`, crash-only, no partial
  cleanup, no stop, under `FairGuardSpec`.
- Full scope (`guard_ackclear_r7_full`): the guard scope plus false death,
  orphan sweep, rebalance and partial cleanup. The run used `-workers 3` and
  was stopped with 12 million states still queued.

The exhaustive runs used `-workers 1` to `-workers 3`. The largest,
`guard_marker_check_r7`, took about 90 minutes on 10 cores shared with the
other runs.

| Config | Property | Design | Redis | Result | States | Trace |
| --- | --- | --- | --- | --- | ---: | ---: |
| `cfg/guard_asis_r7` | 6 | `asis` | 7+ | **violated**, #385 | 657 | 6 |
| `cfg/guard_asis_r6` | 6 | `asis` | 6.2 | holds, exhaustive | 10,410,672 | — |
| `cfg/guard_marker_r7` | 6 | `marker` | 7+ | **violated** | 664 | 6 |
| `cfg/guard_marker_r6` | 6 | `marker` | 6.2 | holds, exhaustive | 12,717,552 | — |
| `cfg/guard_marker_check_r7` | 6 | `marker_check` | 7+ | holds, exhaustive | 24,555,168 | — |
| `cfg/guard_marker_check_r6` | 6 | `marker_check` | 6.2 | holds, exhaustive | 12,717,552 | — |
| `cfg/guard_minid_r7` | 6 | `minid` | 7+ | holds, exhaustive | 2,746,992 | — |
| `cfg/guard_minid_r6` | 6 | `minid` | 6.2 | holds, exhaustive | 2,746,992 | — |
| `cfg/guard_ackclear_r7` | 6 | `ackclear` | 7+ | holds, exhaustive | 17,816,256 | — |
| `cfg/guard_ackclear_r6` | 6 | `ackclear` | 6.2 | holds, exhaustive | 9,983,040 | — |
| `cfg/guard_ackclear_r7_full` | 6 | `ackclear` | 7+ | no violation, **not exhaustive** (stopped at depth 17) | 22,331,060 | — |
| `cfg/guard_double_asis_r7` | 2 | `asis` | 7+ | **violated**, B2 | 8,839 | 12 |
| `cfg/guard_double_asis_r6` | 2 | `asis` | 6.2 | holds, exhaustive | 59,264 | — |
| `cfg/guard_double_ackclear_r7` | 2 | `ackclear` | 7+ | holds, exhaustive | 111,296 | — |
| `cfg/guard_double_ackclear_r6` | 2 | `ackclear` | 6.2 | holds, exhaustive | 55,040 | — |
| `cfg/guard_live_ackclear_r7` | 7 | `ackclear` | 7+ | holds, exhaustive | 2,350,707 | — |
| `cfg/guard_live_ackclear_r6` | 7 | `ackclear` | 6.2 | holds, exhaustive | 1,429,785 | — |

The traces:

- `guard_double_asis_r7` is issue #385 as reported. `DispatchJob(k1)` adds
  `s1`. The sink delivers it and the router adds it to `w1`'s stream. `MAXLEN`
  trims `s1`, and XAUTOCLAIM purges its pending entry. `in_flight` finds `s1`
  neither pending nor in the stream, so the guard is released. A retry adds
  `s2`, and `w1` starts `k1` for `s1` and again for `s2`.
- `guard_asis_r7` is shorter and needs no worker stream. The router holds `s1`
  between delivery and its worker `XADD` while `s1` is trimmed and purged, and
  the guard is released.
- `guard_marker_r7` is the same trace. The marker is written only with the
  worker `XADD`, which has not happened yet. `marker_check` closes it by
  refusing to route an event whose pending entry is gone.

What the results mean:

- Redis 6.2 as written before #411 is safe (`guard_asis_r6`,
  `guard_double_asis_r6`). Only the Redis 7 purge, or acking deleted ids
  (#411), opens B2.
- `marker_check`, `minid` and `ackclear` all hold on both versions. Go
  implements `ackclear`:
  - It changes one branch of `in_flight` and adds one script on the ack
    path. It needs no new key and no change to routing or trimming.
  - `marker_check` needs a routing script that duplicates `Stream.Add`
    for the worker stream, and one more write per routed event.
  - `minid` needs every pool stream `XADD` to run a trim script. It also
    lets one stuck pending entry grow the pool stream past
    `maxQueuedJobs` until `pendingEventTTL`.
  - Ack-aware trimming (`XADD`/`XTRIM ... ACKED`) exists only from
    Redis 8.2. Redis 6.2.24 and 7.4.11 reject it as a syntax error.
- Under `ackclear`, a guard whose start event was purged and whose worker
  stream copy was destroyed (worker cleanup, close) is held until
  `pendingEventTTL`. `GuardReleased` shows that the TTL releases every such
  guard. Nothing else in Redis can tell such an event from one that may
  still start.
- Go also holds until `pendingEventTTL` the guard of a start event that
  was trimmed before delivery, once the group delivered a later entry: the
  event's id is then below `last-delivered-id`, so it looks purged. The
  model drops such a record and releases the guard. That divergence is
  sound for `GuardHeld`: the model releases in a superset of Go's cases,
  and a record trimmed before delivery has no copy anywhere. For
  `GuardReleased`, the TTL releases the Go guard too.
- `GuardReleased` cannot tell a release by the ack from a release by the
  TTL. That a normal start's guard is released at its ack, with no added
  wait, rests on the Go tests `TestAckStartDeletesGuard` and
  `TestDispatchGuardClearedByAckOfTrimmedStart`.
- `REDIS7 = TRUE` also covers the sink acking deleted ids, so with
  `ackclear` in place the sink acks them on every version (#411). For #411
  `guard_double_ackclear_r7` and `guard_live_ackclear_r7` were run again
  with no change to the model and gave the results above.

### Lost requeued starts (issue #416)

Four paths requeue a job that already has a payload: `cleanupWorker`,
`close` (`requeueJobs`), the orphan sweep and `rebalance`. Before the fix,
each added a plain start event with no dispatch guard. Such an event can be
lost before any worker handles it:

- `MAXLEN ~ maxQueuedJobs` trims it before delivery, when the pool sink lags
  by `maxQueuedJobs` events.
- Routing fails, and `MAXLEN` then trims the pending entry. The sink never
  redelivers a trimmed entry.
- Routing acks it as stale after `pendingEventTTL`.

After `cleanupWorker`, `close` or an orphan requeue, no `jobMap` entry
lists the key and the payload stays, so the orphan sweep requeues the job
after its grace period. `rebalance` left the key in the old worker's
`jobMap` entry. The orphan sweep skips a key that `jobMap` lists, and
cleanup skips a live worker. The job then ran nowhere until the old worker
left the pool. Issue #416 expected only the first kind of loss.

`JobRecovered` (property 8) checks this. `EventuallyRuns` follows a queued
start event, so it cannot see a start that is lost. `JobRecovered` follows
the payload: once the faults stop, a job with a payload runs again on a live
worker. The faults are trims, failed routings (`RouteDrop`), TTL expiries and
false deaths. The property needs them to stop, because a pool that loses
every event can run nothing. `OwnerReached` (property 9) checks that a job
also reaches its owner, which `JobRecovered` does not ask.

Both run under `FairJobSpec`, which differs from `FairGuardSpec` in four
ways:

- It is fair to each replica and to the orphan sweep of each node and key.
  `WF_vars(DoReplicate)` lets one map replicate forever while another never
  does. Under it, TLC reported a false violation in which the payload
  replica never caught up.
- It is fair only to the keep-alive expiry of a dead or stopped worker.
  With `FALSE_DEATH`, `WF_vars(DoKeepAliveExp)` forces false deaths forever,
  so the faults never stop and `JobRecovered` holds for no reason.
- The keep-alive loop of a live worker is fair.
- Rebalance and `AckStart` are fair.

`REQUEUE_DESIGN` selects the candidate design:

| Design | Change |
| --- | --- |
| `asis` | Main before #416. |
| `acktrim` | Trimming never removes an unacked entry. This is `XADD`/`XTRIM ... ACKED`, which exists only from Redis 8.2. |
| `release` | `rebalance` removes the key from `jobMap[w]` before it adds the start event. |
| `guard` | `release`. Also, `rebalance` and the orphan sweep add the start event and write the dispatch guard that names it in one script (`GuardedStart`, Go `luaClaimRequeue`). The script refuses while a guard of the key is in flight. The worker ack deletes the guard (`luaAckStart`). A rebalance that meets an in-flight guard keeps the job running and does not retry. |
| `guard_retry` | `guard`, and a rebalance that meets an in-flight guard retries (Go `retryRebalance`, after `ackGracePeriod`). |

`GUARD_DESIGN = "minid"` (a trim never passes the oldest pending entry) is
the sixth candidate. It is not a `REQUEUE_DESIGN` value.

In the guard designs, rebalance is triggered, as in Go. `rebal[w]` says a
pass of `w` is due. A worker map change on a node's replica, or a worker
that returns after a false death (which stands for a join), sets it. A
pass that moves the job clears it, and so does a refusal in `guard`. In
the other designs `rebal` stays `TRUE`, and rebalance can fire whenever the
node's view calls for it. A pass with nothing to move keeps the trigger.
This leaves out a race that exists on main too: a pass can run before a
start lands on the worker, and nothing rebalances the job afterwards. A
variant of the model in which such a pass ends the trigger fails
`OwnerReached` in 12 states for `guard`. That trace has no refusal, so the
retry does not help (#433). In Go, `handleWorkerMapUpdate` also runs only on
worker map updates, and a joining worker's map entry can arrive before its
keep-alive, so the join's rebalance can see the old worker set.

Scopes. Every configuration has the fixes of tickets 1 to 4, `ackclear`
where the stream is tracked, `MaxEv=3`, `MaxTok=2` and no stop. The model
has a fixed set of workers, so a worker that comes back after a false
death stands for a worker that joins. The `guard_retry` configurations
also set `SPLIT_ACK`.

- Rebalance scope (`requeue_live_*`): false deaths, rebalance, orphan
  sweep and cleanup, no close or crash, `MaxStream=1`.
- Crash scope (`requeue_live_*crash*`): crash-only deaths with close and
  crash, as `guard_live_*`. Rebalance is enabled but can hardly fire,
  because a crashed worker never comes back.
- Owner scope (`requeue_owner_*`): the rebalance scope with atomic routing,
  no trimming or TTL, and `SPLIT_ACK`.
- Runner scope (`requeue_runner_*`): safety with atomic routing, false
  deaths, close and crash, `MaxStream=2`, grace covers lag. The state
  constraint `NoLiveCleanup` keeps cleanup from taking a running worker,
  which leaves out the known B6 double runner.
- Stream scope (`requeue_stream_*`): the runner scope with two-step
  routing, trimming, the TTL and `MaxStream=1`.

| Config | Property | Design | Scope | Result | States | Trace |
| --- | --- | --- | --- | --- | ---: | ---: |
| `cfg/requeue_live_notrim` | 8 | `asis` | rebalance, no trimming or TTL | no violation, **not exhaustive** (depth 17) | 1,696,500 | — |
| `cfg/requeue_live_asis_r7` | 8 | `asis` | rebalance, trimming, Redis 7+ | **violated**, #416 | 227,507 | 13 |
| `cfg/requeue_live_asis_r6` | 8 | `asis` | rebalance, trimming, Redis 6.2 | **violated**, #416 | 249,516 | 13 |
| `cfg/requeue_live_ttl_r7` | 8 | `acktrim` | rebalance, TTL, Redis 7+ | **violated** | 242,140 | 13 |
| `cfg/requeue_live_minid_r7` | 8 | `minid` | rebalance, no TTL, Redis 7+ | **violated** | 472,834 | 14 |
| `cfg/requeue_live_crash_r7` | 8 | `asis` | crash, trimming and TTL, Redis 7+ | no violation, **not exhaustive** (depth 29) | 2,014,043 | — |
| `cfg/requeue_live_retry_r7` | 8 | `guard_retry` | rebalance, trimming and TTL, Redis 7+ | no violation, **not exhaustive** (depth 18) | 2,495,398 | — |
| `cfg/requeue_live_retry_crash_r7` | 8 | `guard_retry` | crash, trimming and TTL, Redis 7+ | no violation, **not exhaustive** (depth 26) | 1,628,785 | — |
| `cfg/requeue_owner_asis` | 9 | `asis` | owner | no violation, **not exhaustive** (depth 17) | 1,932,138 | — |
| `cfg/requeue_owner_guard` | 9 | `guard` | owner | **violated** | 92,487 | 12 |
| `cfg/requeue_owner_retry` | 9 | `guard_retry` | owner | no violation, **not exhaustive** (depth 18) | 3,198,780 | — |
| `cfg/requeue_runner_asis` | 1, 2 | `asis` | runner | holds, exhaustive | 846,024 | — |
| `cfg/requeue_runner_release` | 1, 2 | `release` | runner | **violated** (1) | 46,875 | 14 |
| `cfg/requeue_runner_retry` | 1, 2 | `guard_retry` | runner | holds, exhaustive | 3,660,345 | — |
| `cfg/requeue_stream_asis_r7` | 1, 2, 6 | `asis` | stream, Redis 7+ | no violation, **not exhaustive** (depth 30) | 4,139,906 | — |
| `cfg/requeue_stream_retry_r7` | 1, 2, 6 | `guard_retry` | stream, Redis 7+ | no violation, **not exhaustive** (depth 36) | 61,608,283 | — |

For a liveness run that was stopped, States is the graph size at the last
periodic liveness check that finished without a violation. TLC checks
liveness on the graph found so far, so a violation in that part would have
been reported. The violations were reported at the first check after 90,000
to 470,000 states. The larger liveness runs grow by millions of states, and
each check then takes 5 to 30 minutes, so they were stopped after one to two
hours with 1 to 3 workers each. The `requeue_stream_*` safety runs were
stopped too: `requeue_stream_asis_r7` is a baseline, and
`requeue_stream_retry_r7` checked 61 million states in 90 minutes with 3
workers.

The traces:

- `requeue_live_asis_r7` is issue #416 on the rebalance path. `w1` looks
  dead, so `DispatchJob(k1)` is routed to `w2`, which starts `k1`. `w1`
  refreshes its keep-alive. `n2` sees `w1` active again, and
  `w2.rebalance` stops `k1` and adds a start event for it. `MAXLEN` trims
  that event before delivery. `jobMap[w2]` still lists `k1`, so the orphan
  sweep skips it, and cleanup skips `w2`, which is alive. `k1` runs
  nowhere for the rest of the behavior. `requeue_live_asis_r6` is the same
  trace.
- `requeue_live_ttl_r7` is the same trace with the TTL in place of the
  trim: routing drops the rebalance start as stale. So `acktrim` does not
  close the loss.
- `requeue_runner_release` is B3 on the rebalance path. `w1` starts `k1`,
  then looks dead. `w1.rebalance` on `n1` stops `k1`, removes it from
  `jobMap[w1]` and adds `s2`, which is routed to `w2`'s stream. `w1`
  refreshes its keep-alive. Before `w2` handles `s2`, the orphan sweep on
  `n1` finds `k1` in no `jobMap` entry and adds `s3`, which is routed to
  `w1`. `w1` and `w2` both start `k1`. So `release` turns the loss into a
  double runner.
- `requeue_owner_guard` is the refusal without a retry. `DispatchJob(k1)`
  adds `s1` and its guard. `w1` looks dead, so `s1` is routed to `w2`,
  which starts `k1`. The ack of `s1` has not reached `luaAckStart`, so the
  guard is still in flight. `w1` returns, `w2.rebalance` meets the guard,
  keeps `k1` and ends the pass. Then the ack deletes the guard. No trigger
  follows, so `k1` stays on `w2` while `w1` owns it. On main the pass
  moved the job at once.

What the results mean:

- The loss is real on both Redis versions. On the cleanup, close and
  orphan paths, the orphan sweep recovers the job, as the issue expected.
  On the rebalance path, nothing recovers it while the old worker is alive.
- `guard_retry` is the only candidate that holds for all three properties
  checked. Go implements it:
  - `acktrim` needs Redis 8.2 and still loses a start to the TTL.
  - `minid` still trims an undelivered entry while nothing is pending
    (`requeue_live_minid_r7`, the `asis` trace).
  - `release` recovers the job, but adds a B3 double runner.
  - `guard` leaves a job on a worker that does not own it when the pass
    meets the guard of the start that placed the job there.
- The guard is in flight while its start event may still start the job,
  so the orphan sweep and rebalance wait for it. Once the event is acked,
  the job map lists the key. Once the event is lost, the guard no longer
  blocks, and the sweep requeues the job. A requeue guard is written with a
  TTL that has already passed, so only `in_flight` decides.
- Rebalance checks the guard before it stops the job
  (`luaRequeueInFlight`), so a refused pass does not stop and restart the
  handler. If the guard appears between that check and `luaClaimRequeue`,
  the job is restarted and the retry follows.
- Recovery takes the orphan grace period plus one sweep period. In Go, a
  start trimmed before delivery looks purged once the group delivers a
  later entry (see the dispatch guard results above), so the sweep can
  wait up to `pendingEventTTL` (2 minutes). A rebalance that meets such a
  guard retries every `ackGracePeriod` for as long.
- `in_flight` is false once an event is older than `pendingEventTTL`, so a
  requeued start that waits longer on a worker stream can be requeued a
  second time. This is the residual gap of the dispatch guard.
- A client-side error after `luaClaimRequeue` ran on the server, such as a
  timeout, makes rebalance restart the job locally while the start it
  queued starts the job on the owner too: two runners. `poolStream.Add`
  had the same risk before this change. The model has no client errors.
- `close` and `cleanupWorker` still requeue without a guard, so B3 stays
  open on those paths until ticket 6 of the roadmap.
- Tests, on miniredis and on Redis 6.2 and 7.4:
  - `TestLostRebalanceStartIsRecovered` replays the `requeue_live_asis_r7`
    trace. It fails on main in two places (the `jobMap` entry and the
    recovery). With `release` alone it fails on the orphan sweep's second
    start.
  - `TestRebalanceWaitsForUnackedStart` replays the `requeue_owner_guard`
    trace. Without the check and the retry it fails twice: the pass stops
    and restarts the job, and the job never reaches `w2`.
  - `TestClaimRequeue`, `TestRebalanceReleaseAndRestart` and the
    `claimRequeue` and `requeueInFlight` paths of
    `TestDispatchGuardInFlight` cover the scripts.

## Worker event acks (`WorkerEventAck.tla`)

`WorkerEventAck.tla` is a separate, smaller model. It covers how
`routeWorkerEvent` registers a routed pool event and how `ackWorkerEvent`
matches the worker's ack to it (issue #381). `PoolOwnership.tla` puts the
worker ack and the router's `XACK` in one step, so it cannot show this race.

### Abstraction

One node routes one pool event to one worker. `MaxRoutes` bounds the
routings: XAUTOCLAIM redelivers the event while it is unacked, and each
routing gets a new worker event id. The router, the worker and the node's
ack loop are separate processes. `Add` is two steps: Redis applies the
`XADD`, then the call returns. The worker can ack in between.

`DESIGN` selects how the router registers the pending event:

| Design | Router | Ack loop |
| --- | --- | --- |
| `asis` | Registers after `Add` returns (main before #381). | Drops an unknown ack. |
| `prereg` | Registers before `Add` and unregisters when it fails. Go cannot do this, because `XADD` assigns the id. | Drops an unknown ack. |
| `lock` | Holds one lock across `Add` and the registration. | Looks the id up under the lock. |
| `park` | After `Add`, consumes a parked ack or registers. | Parks an unknown ack. Parked acks are pruned by age. |

Faults and environment:

| Toggle | Meaning |
| --- | --- |
| `AMBIGUOUS_FAIL` | `Add` returns an error after Redis applied the `XADD` (for example a client timeout). `AddFail` covers a failure without effect. |
| `DUP_ACK` | A second ack for the same id reaches the node stream. |
| `SHUTDOWN` | The node stops. Its in-memory maps are gone. |
| `PRUNE_ANYTIME` | `park`: a parked ack can be pruned while the router is still adding that id. The default assumes the prune age exceeds any `Add` call. |

### Action map (Go)

| Action | Go |
| --- | --- |
| `RouteBegin`, `AddApply`, `AddFail` | `routeWorkerEvent` (`node_events.go:59`): `stream.Add` (`:91`). A missing worker stream makes `Add` return no id and no error; `routeWorkerEvent` then returns an error and registers nothing (`:95`). |
| `RouteRegister`, `RouteFailed` | `registerPendingEvent` (`node_events.go:112`). |
| `WorkerAck` | `handleEvents` (`worker.go:204`), then `ackPoolEvent` (`worker.go:432`). |
| `ProcessAck` | `ackWorkerEvent` (`node_events.go:167`): `LoadAndDelete` under `pendingEventsLock`, otherwise `parkWorkerAck`. A match calls `completePendingEvent`: `XACK`, then the dispatch return. |
| `Prune` | `parkWorkerAck` (`node_events.go:228`) drops parked acks older than `ackGracePeriod`. |
| `Redeliver` | XAUTOCLAIM with `MinIdle = ackGracePeriod`. |

### Properties

- `NoUnknownAckForRoutedEvent`: no ack of an event whose `Add` returned ok
  is dropped or pruned before an ack of that id matched.
- `NoRegistrationWithoutAdd`: a registration exists only for an event whose
  `Add` returned ok.
- `AckedOnlyAfterWorker`: the pool event is acked only after the worker
  acked a routed copy.
- `MatchedAtMostOnce`: one registration consumes at most one ack, so it
  sends one `XACK` and one dispatch return.
- `AckedWhenWorkerAcked`, `NoLeakWhenSettled`: once nothing is in flight,
  the pool event is acked if the worker acked a copy whose `Add` returned ok,
  and no registration remains for an acked id.
- `ParkedBounded`: parked acks are acks the worker sent.
- `ParkedDrains` (liveness): every parked ack is consumed or pruned.
- `PromptAck` (liveness): after the worker acks a copy whose `Add` returned
  ok, the pool event is acked without a redelivery, unless the node stops.

Every configuration checks all properties with `FairSpec`, `MaxRoutes = 3`
and every fault toggle on.

### Results

| Config | Design | Result | States | Trace |
| --- | --- | --- | ---: | ---: |
| `cfg/ack_asis` | `asis` | **violated**, `NoUnknownAckForRoutedEvent` | 24 | 5 |
| `cfg/ack_prereg` | `prereg` | holds, exhaustive | 8,486 | — |
| `cfg/ack_lock` | `lock` | holds, exhaustive | 5,054 | — |
| `cfg/ack_park` | `park` | holds, exhaustive | 17,638 | — |
| `cfg/ack_park_prune_anytime` | `park`, `PRUNE_ANYTIME` | **violated**, `NoUnknownAckForRoutedEvent` | 44 | 6 |

The `asis` trace is issue #381: the router starts `Add`, Redis applies the
`XADD`, the worker acks, and the ack loop drops the ack as unknown. Only then
does `Add` return and the router registers the event. With the other
invariants alone, `asis` also violates `AckedWhenWorkerAcked` and
`NoLeakWhenSettled` (6-state traces), and `PromptAck` fails: every
redelivery can race the same way.

Go implements `park`, the same pattern as the dispatch-return fix.
`prereg` needs the id before the `XADD`. `lock` holds a mutex across a Redis
call and stalls the node's ack and dispatch-return loop behind every routing.
`park_prune_anytime` shows that the prune age must exceed the `Add`
duration. If an `Add` is slower than `ackGracePeriod`, the ack is lost and
the pool event is redelivered, as on main before the fix.

## Ambiguous requeue replies (#436)

`RequeueReply.tla` models a successful release followed by a guarded requeue
on two workers. Redis can fail before executing the script or lose the reply
after executing it; another requeue can also win before the call. Delivery
can precede the reply. The former local restart violates `AtMostOneRunner`
in six states with a lost reply (`requeue_reply_asis.cfg`). Keeping the old
worker stopped and allowing orphan recovery passes safety and `Recovered`
with 10 distinct states (`requeue_reply_sweep.cfg`).

Run from this directory with a local `tla2tools.jar`:

```sh
java -XX:+UseParallelGC -cp /tmp/tla2tools.jar tlc2.TLC -workers 1 -deadlock \
  -metadir /tmp/requeue-reply-asis -config cfg/requeue_reply_asis.cfg RequeueReply.tla
java -XX:+UseParallelGC -cp /tmp/tla2tools.jar tlc2.TLC -workers 1 -deadlock \
  -metadir /tmp/requeue-reply-sweep -config cfg/requeue_reply_sweep.cfg RequeueReply.tla
```

The first command must fail its invariant; the second must pass. Liveness
assumes Redis recovers, the orphan grace passes, and delivery and the sweep
continue. It does not model crashes, stale job-map replicas, or ownership
races outside this handoff; those remain in `PoolOwnership.tla` and the
owner-record roadmap. `TestRebalanceRequeueReply` exercises the corresponding
Redis faults, refusal, orphan recovery, and target-worker start in Go.
