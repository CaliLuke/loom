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

## Shared Prerequisite and Ownership

Create and health-check a `*redis.Client` before constructing Pulse objects.
Pulse objects use the supplied client but do not own its process-level
lifecycle. Close readers, sinks, maps, and pool nodes first; close the Redis
client only after their background work has stopped.

Use bounded contexts for startup, mutations, and shutdown. A context passed to
a constructor or operation bounds that call; it is not a substitute for the
explicit close method on the returned object.

## Replicated Maps

`rmap.Join` loads the current Redis hash, subscribes to revisions, and starts a
local update loop. Always call `Map.Close` when that replica is no longer
needed. `Close` stops subscriptions and freezes the local snapshot; it does not
delete shared Redis state.

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

Readers give every reader its own copy of events. Named sinks form a consumer
group and share its cursor. Unless `WithSinkNoAck` is selected, acknowledge an
event with `Sink.Ack` only after its side effects are durable. Closing a sink
leaves consumer metadata available so another instance can claim pending
messages.

`Reader.Close` and `Sink.Close` stop local polling and close subscriber
channels. `Stream.Destroy` is different: it deletes the shared stream and sink
membership state. Reserve it for intentional teardown, tests, or coordinated
retirement—not ordinary process shutdown.

## Worker Pool Shutdown

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
- Let unacknowledged sink events remain pending when processing fails. Another
  consumer can claim them after recovery.
- Treat Redis connectivity errors as operational failures. Log the object
  name and operation, retry only idempotent work, and keep destructive cleanup
  out of automatic reconnect paths.
