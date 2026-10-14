---
title: Pulse Lifecycle
weight: 9
description: "Lifecycle, shutdown, and recovery contracts for Loom Pulse Redis primitives."
llm_optimized: true
aliases:
---

Pulse provides Redis-backed replicated maps, event streams, distributed
workers, schedulers, and tickers under `github.com/CaliLuke/loom/pulse`. These
packages start background subscriptions or polling loops, so ownership and
shutdown are part of their public contract.

## Supported Redis

Pulse supports only the latest stable Redis release, currently **8.10.2**.
Older Redis releases are not supported. The real-Redis CI and local test runner
share the version pin in `scripts/test_pulse_redis.sh`; update that pin and this
guidance when adopting a newer stable release.

## Redis Client Module

Pulse currently uses `github.com/CaliLuke/go-redis/v9`, an exact pinned fork of
upstream v9.23.0 with the reviewed [WithTimeout hook-copy correction](https://github.com/redis/go-redis/pull/4095).
The fork makes the fix available to applications through ordinary Go module
resolution; applications do not need a `replace` directive.

Change client creation and custom hook imports to:

```go
import "github.com/CaliLuke/go-redis/v9"
```

The fork's `Client`, `Hook`, `Cmder`, and other Go types have a different package
identity from `github.com/redis/go-redis/v9`. Use the fork consistently wherever
clients or hook signatures cross Pulse boundaries. Instrumentation compiled
against the upstream package is not interchangeable; adapt its imports before
passing it fork clients. Upstream `extra/redisotel` and similar extension modules
have not been republished by this root-module fork.

Return to an upstream stable release once it contains the correction and the
shutdown/hook regressions pass against it; that transition also changes application
imports back to the upstream module.

## Shared Prerequisite and Ownership

Create and health-check a `*redis.Client` before constructing Pulse objects.
Pulse objects use the supplied client but do not own its process-level
lifecycle. Close readers, sinks, maps, and pool nodes first; close the Redis
client only after their background work has stopped.

Use bounded contexts for startup and mutations. A context passed to a constructor
or operation bounds that call; it is not a substitute for the explicit close
method on the returned object. Reader, Sink and Map runtime lifetimes are
independent of constructor-context cancellation. Explicitly close returned objects.

`Reader.Close`, `Sink.Close` and `Map.Close` seal local admission and cancel owned
work before waiting for it. Every concurrent caller waits for the same cleanup
to finish. These calls join local workers and release owned subscriptions; they
do not wait for application processing or outstanding external operations.
`Sink.Close(ctx)` joins cleanup even when `ctx` is canceled; that argument is not
a deadline for the join. Close Pulse objects before closing their shared client.

Owned Redis commands and subscription setup use finite operation and socket
budgets. Native blocking reads use finite polls with time for the reply; a zero
block duration continues listening through successive polls. Earlier caller
deadlines are not extended. Shorter positive client read/write limits constrain
operation budgets, but native blocking commands can have a longer driver socket
deadline. Cancellation alone does not interrupt every active socket operation.
There is no fixed five-second Close guarantee: admitted cleanup can contain
several bounded operations.

Pulse preserves the supplied client's options, connection pools and installed
hooks; it does not close that client. Hooks, custom dialers and logging callbacks
must return: Pulse cannot forcibly stop arbitrary application callbacks. Producer
operations such as `Stream.Add` and application calls to `Sink.Ack` retain caller
context/client ownership; Close does not cancel or join them. Supply suitable
contexts and client limits for these calls.

## Replicated Maps

`rmap.Join` loads the current Redis hash, subscribes to revisions, and starts a
local update loop. Always call `Map.Close` when that replica is no longer
needed. `Close` stops subscriptions and freezes the local snapshot; it does not
delete shared Redis state.

`Map.Subscribe` provides coalesced invalidations, not a journal of every update.
Read the map for current state. Each subscriber retains the strongest unread kind:
`EventReset` takes precedence over `EventDelete`, then `EventChange`. A later change
cannot hide an unread reset. Slow subscribers do not block replica updates.
Unsubscribe when finished; closing the map closes its subscription channels.

Choose shared-state mutations deliberately:

- `Reset` clears the map content while preserving the replicated-map identity.
- `Destroy` clears shared content through the revisioned destroy protocol so
  live replicas observe the change.
- Both operations affect every process joined to the same map name. They are
  not cleanup substitutes for `Close`.

## Streams, Readers, and Sinks

`streaming.NewStream` is a lightweight handle to a named Redis stream. The
objects that own background work are its readers and sinks:

```go
stream, err := streaming.NewStream("orders", rdb)
if err != nil {
    return err
}

reader, err := stream.NewReader(ctx)
if err != nil {
    return err
}
defer reader.Close()

sink, err := stream.NewSink(ctx, "workers")
if err != nil {
    return err
}
defer sink.Close(shutdownCtx)
```

Reader and Sink buffer sizes must be nonnegative; zero selects an unbuffered
channel. Sink acknowledgement grace periods must be positive. Constructors
reject invalid values before contacting Redis or registering a consumer, using
the final value when an option is supplied more than once. A zero block duration
still means continuous listening, implemented as finite repeated polls. Positive
grace periods retain the same strict stale threshold of more than two grace
periods, including large durations.

Readers give every reader its own copy of events. Named sinks form a consumer
group and share its cursor. Unless `WithSinkNoAck` is selected, acknowledge an
event with `Sink.Ack` only after its side effects are durable. Closing a sink
leaves consumer metadata available so another instance can claim pending
messages. A sink begins reading and claiming pending events only after its
first successful `Subscribe`. Construction still creates consumer membership
and starts keepalive. Delivery continues after all subscribers unsubscribe;
this does not promise replay of events consumed during that later interval.

Each named sink group has one persistent effective topic filter per stream:
unfiltered, an exact nonempty topic, or the exact spelling of a regular expression.
A nonempty `WithSinkTopic` takes precedence over `WithSinkTopicPattern`. Joining
that group with a different filter fails before delivery, including `AddStream`.
Use different group names for independent subscriptions. Filtered and malformed
events are acknowledged for that group, including entries reclaimed from its
pending list. This leaves stored entries and other groups untouched. Readers
skip rejected events without acknowledging them. Accepted sink events remain
application-owned; `WithSinkNoAck` applies to fresh reads, not historical pending
entries reclaimed by that instance.

Upgrading to enforced filters requires coordinated migration: stop all old
instances, preserve or drain any data that must survive, and either move to a
fresh stream or use `Stream.Destroy` for intentional shared teardown before
recreating it. Do not mix old and new binaries. Legacy membership records,
expiring membership hashes, and existing Redis groups without a recorded filter
are rejected rather than assigned a guessed identity. Group identity no longer
inherits the stream TTL and survives empty membership and `RemoveStream`.
`Stream.Destroy` removes it together with the stream; ordinary shutdown does not.
Use a new group name to change a subscription on an already migrated stream.
Stale-member cleanup also reconciles members left behind by stream expiry. A
new registration receives the existing grace period before its first heartbeat;
pending entries continue to protect their consumer from deletion.

A default reader captures the newest event ID once during `NewReader`. It skips
history present at that snapshot and retains events added afterward, including
before the first `Subscribe` or between polls. Adding a stream with a newest start
mode captures that stream's own newest ID once. Empty or missing streams start
at `0-0`. The lookup can return a Redis connectivity, permission or type error;
a failed addition leaves the reader's existing streams unchanged. Explicit oldest,
timestamp and event-ID positions require no new lookup. `AddStream` inherits the
reader's authored start mode unless its options override it.
Reader membership uses the stream key: a second `Stream` handle for the same
name is a duplicate on addition and removes the existing membership on removal.
Removal permits events from an already-admitted read to finish delivery.

`Reader.RemoveStream` stops future reads for that registration. An already-admitted
batch may still reach subscribers. Re-adding the stream installs its selected start
position independently: completion of an earlier read cannot move that new cursor.

Calling `Subscribe` after Reader or Sink shutdown has started returns an
already-closed channel. It does not restart polling.

`Reader.Close` and `Sink.Close` stop local polling and close subscriber
channels. `Stream.Destroy` is different: it deletes the shared stream and sink
membership state. Reserve it for intentional teardown, tests, or coordinated
retirement—not ordinary process shutdown.

An active sink repairs a missing consumer group and its membership using that
stream registration's original start selector. An existing group's shared cursor
is never reset. Oldest can replay retained entries after group deletion; newest
skips entries present when the group is recreated; explicit ID and timestamp
selectors remain unchanged. Redis cannot restore deleted events or a deleted
pending list. Recovery retries with backoff and stops creating groups after local
removal or shutdown. Stop participating sinks before intentional shared teardown.

`Sink.RemoveStream` releases its local map subscription even when another
instance still consumes that stream. On a membership-removal error, the local
stream is retained; retry removal because Redis may have applied a write whose
reply was lost. If membership removal succeeds but group cleanup fails, polling
stops and the sink retains the map for cleanup. Retry `RemoveStream` to finish,
or call `AddStream` to resume the stream with that same map. `Sink.Close` also
closes maps retained for pending cleanup.

A stale sink consumer is replaced across every active stream before the sink
switches to the new name. Rotation preserves pending messages on old Redis
consumers for idle-message recovery. Failed retirement remains owned and is
retried; removing a stream also removes any retained consumer names. With no
streams, the sink waits without creating a consumer or issuing empty reads;
adding a stream resumes polling.

## Pool Job Ownership and Upgrades

Pool protocol 3 records one owner and a monotonically increasing epoch for each
job key in Redis. Claims and releases are atomic; a stale worker cannot delete
a later owner's payload. Recovery rechecks the worker's keep-alive in Redis
and reads authoritative ownership instead of a local replica. Orphan recovery
checks ownership and the current payload atomically before requeueing, so stale
replicas cannot requeue an owned job or revive a stopped job. Handlers receive
the ownership token in `Job.Epoch`; the pool retains epoch counters even after
`Shutdown` so tokens never restart when the pool name is reused.

Each admitted start also has an internal incarnation, retained through routing,
rebalance and recovery. A stop records a cutoff: delayed earlier starts cannot
revive the stopped job, and a delayed stop cannot kill a later admitted start.
`StopJob` returns after queue publication, not after the handler finishes.

Upgrade from older pool protocols requires coordinated teardown with the old
binaries: end each recurring producer with `ErrScheduleStop` to clear its old
ledger, stop/drain its work, and complete pool `Shutdown` before starting
protocol 3 nodes. `Close` alone can leave pending operations whose identity the
new protocol cannot reconstruct. Recreate schedules and explicitly request the
jobs needed after upgrading. Persisted older protocol markers and live older
members are rejected; rolling upgrades and mixed-version rollbacks are unsupported.
If an older member appears later, protocol 3 nodes stop routing and claiming work
until it leaves or its heartbeat expires. `Node.Health(ctx)` reports this condition
and Redis errors. Older binaries cannot enforce this gate themselves.

A recurring producer retains responsibility for every requested Start, including
an admission timeout, cancellation, or `ErrJobExists`. `StopAll` selects that
producer's requested keys, excluding Start keys in the current plan; it does not
stop another producer's disjoint jobs. Explicit Stop still addresses its authored
global job key. Responsibility is removed only after confirmed stop, not enqueue.
Later owned ticks retry pending operations with their original identity and stop
cutoff. Handoff reads authoritative Redis intent, independently of local map lag.
An observed successful admission is not continuously restarted if it later stops.
A later authored Start is a new command, even with the same key and payload; an
empty plan only reconciles stored pending operations.

`ErrScheduleStop` ends planning and clears intent; it does not stop already
admitted jobs. A local node's departure permits another participating scheduler
to reconcile persisted intent. The supplied Schedule context still governs its
Redis operations; cancellation alone does not stop its planning loop. Local
Close or global `ErrScheduleStop` ends that loop. Pending retries require later
participating ticks, available Redis, and handlers that return. Stream retention
and `pendingEventTTL` bound recovery of individual deliveries; arbitrary outages
or permanent schedule shutdown have no delivery guarantee. Internal incarnation,
cutoff and generation counters survive resets/teardown to fence delayed copies.

Workers pause event handling and call `Stop` for running jobs if a heartbeat
has not succeeded within three quarters of the worker TTL. A separate monitor
checks every eighth of the TTL, independently of Redis heartbeat requests.
Successful renewal resumes a paused job only after Redis confirms the same
owner and epoch. Cleanup or transfer makes the old worker forget that job.
`Start` may therefore be called again with the same epoch after a pause. A
failed `Stop` or resume `Start` is logged and retried while the job stays fenced.

Handlers must finish callbacks promptly: the remaining eighth of the TTL must
cover scheduling delays and all `Stop` calls. A callback that blocks, a failed
`Stop`, or an unbounded process pause can outlast that margin. Ownership records
and local fencing do not make external side effects exactly once. Applications
that need to fence external writes must have the destination store check epochs
atomically and reject tokens older than the highest accepted token for that key.

Call `job.Worker.CheckOwnership(ctx, job.Key, job.Epoch)` from the job's work
loop to check that the local run and Redis owner still match. Abort the planned
operation on any error. `errors.Is(err, pool.ErrOwnershipLost)` identifies a
superseded, paused, or stopped run. A confirmed mismatch queues a stop of the matching
local run on a pool goroutine and returns immediately; this lets `Stop` safely
join the calling work loop. Failed stops are logged and retried. Redis failures
are returned without assuming an ownership transfer. The claimed job is
available to a work loop as soon as `Start` receives it; failed starts remove
that local claim. `Worker.Jobs` includes starting and paused jobs. Workers returned by
`AddWorker`, `Worker.Jobs`, and `Node.Workers` support this local check; remote
`Node.PoolWorkers` entries do not.

A successful check can become stale immediately. To protect a downstream
write, include the epoch in the write and atomically compare it with the
highest epoch recorded for that key in the destination store. Reject a lower
epoch; accept an equal epoch for further operations by the same owner, and
advance the stored epoch when accepting a higher one. This prevents an old
worker from writing after the store has accepted the new owner's epoch. It
does not prevent duplicate operations within an epoch: use an operation ID
and destination-side deduplication when that is required.

## Worker Pool Shutdown

A failed `AddNode` releases local map, sink and reader resources and attempts to
withdraw only its private stream and node registration. Cleanup continues after
individual failures and preserves them alongside the construction error. It uses
a context detached from caller cancellation, bounded by a positive worker shutdown
TTL (or the default when configured nonpositive). Shared pool data and healthy
nodes remain owned by the pool. Uncertain Redis results can leave private state;
a failed node never starts a runtime that renews its registration.

`AddWorker` prepares its private resources before publishing membership. Only a
successful admission starts handlers and lease renewal; handlers may start before
the successful call returns. `Close` excludes later admissions and waits for
in-flight attempts before stopping local workers.

On failure, `AddWorker` releases local resources and attempts to withdraw its
private stream, membership and initial heartbeat. This cleanup uses a separate
context bounded by the worker shutdown TTL, so cancellation of the caller does
not skip cleanup. The returned error includes any cleanup failures. During a Redis
outage, publication may remain until its initial lease expires. Queued pool events
remain eligible for existing recovery within stream retention and event expiry;
this does not guarantee delivery through an indefinite outage.

Every non-client pool node returned by `pool.AddNode` must end through exactly
one of these idempotent paths:

- `Node.Close(ctx)` stops this node, stops its local workers and schedulers,
  and requeues unfinished jobs for other nodes. It does not shut down the
  distributed pool.
- `Node.Shutdown(ctx)` broadcasts a pool-wide shutdown, prevents new work,
  waits for workers across participating nodes, and removes shared pool
  resources. It is not available on a `WithClientOnly` node.

Use `Close` for rolling deploys, instance termination, and local failures. Use
`Shutdown` only when the whole named pool is intentionally stopping. Bound
either operation with a context sized for the configured worker shutdown TTL;
surface a timeout instead of terminating the process while jobs are still
being recovered.

Pool tickers have the same local-versus-shared distinction. `Ticker.Close`
stops only the local participant and leaves the distributed ticker active on
other nodes. `Ticker.Stop` deletes the shared ticker entry and stops that named
ticker for all participants.

## Shutdown Order

For an application using multiple Pulse layers, shut down from the consumers
inward:

1. Stop accepting new application work.
2. Close or shut down pool nodes and stop local schedulers/tickers.
3. Close stream sinks, then readers.
4. Close replicated maps.
5. Close the Redis client.

Do not call shared `Destroy`, pool-wide `Shutdown`, or `Ticker.Stop` merely
because one process is exiting. Those operations change distributed state and
can interrupt healthy peers.

## Failure and Recovery Guidance

- Give worker jobs stable keys. Duplicate dispatch returns `pool.ErrJobExists`
  instead of silently starting a second owner.
- Tune worker TTL and acknowledgement grace periods to the deployment's
  failure-detection needs; values that are too short cause premature recovery
  during network pauses, while values that are too long delay requeue.
- Nodes reconcile job placement on membership changes and every half worker
  TTL. This catches joining workers whose heartbeat arrives after membership,
  and jobs delivered after the first rebalance pass. In-flight dispatch guards,
  handler callbacks, and Redis availability can delay completion further.
- After rebalance releases a job, a failed Redis requeue reply never restarts
  it on the old worker: Redis may already have queued it elsewhere. If the
  write did not happen, the orphan sweep recovers the retained payload after
  observing it without a worker for `max(2 * workerTTL, ackGracePeriod)`, on
  a subsequent sweep. Redis must be available for recovery.
- Let unacknowledged sink events remain pending when processing fails. Another
  consumer can claim them after recovery.
- Treat Redis connectivity errors as operational failures. Log the object
  name and operation, retry only idempotent work, and keep destructive cleanup
  out of automatic reconnect paths.
