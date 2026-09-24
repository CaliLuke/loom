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
| `pool` | Unacked events in the pool stream (`start` or `stop`) |
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
  age, with one more exception: on Redis 7 and later, XAUTOCLAIM purges the
  pending entry of a start event that `MAXLEN` trimmed, so Go can release
  the guard of an unacked event early (roadmap, issue #385).
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
