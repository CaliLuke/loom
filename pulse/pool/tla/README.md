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
  `startJob`. Redelivery can still happen before the worker handles the event,
  which is the window every redelivery trace uses.
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
| `REDIS7` | XAUTOCLAIM purges the pending entry of a trimmed id (Redis 7 and later). Also stands for the sink acking deleted ids on 6.2 (issue #411), which is the same state change. |
| `ENABLE_TTL` | `pendingEventTTL` can pass for an event that no router or worker stream holds. |
| `GUARD_DESIGN` | How `in_flight` tells a start event that can still start from one that cannot. See [Pool stream trimming](#pool-stream-trimming-and-the-dispatch-guard-issue-385). |

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
| `AutoClaim` | XAUTOCLAIM reaches a trimmed pending entry (`streaming/sink_autoclaim.go`). |
| `EventExpire` | `pendingEventTTL` passes: `in_flight` is false by age, and routing acks the event as stale (`node_events.go:61`). |

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

`REDIS7 = FALSE` is Redis 6.2: XAUTOCLAIM keeps a trimmed pending entry,
so the model has no step for it. `REDIS7 = TRUE` also stands for the sink
acking deleted ids on 6.2 (issue #411): both leave a delivered event in
neither the stream nor the pending list.

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

- Redis 6.2 as written is safe (`guard_asis_r6`, `guard_double_asis_r6`).
  Only the Redis 7 purge, or acking deleted ids on 6.2 (#411), opens B2.
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
- `REDIS7 = TRUE` also covers the sink acking deleted ids on 6.2, so #411
  can ack them once this design is in place.

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

