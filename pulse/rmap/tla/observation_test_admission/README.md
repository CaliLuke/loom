# Observation test admission (#935)

The cancellation fixture must establish a pending observation before it waits for
its snapshot-read gate. Unsubscribing does not establish that condition: the
replica owner can publish the committed revision through periodic reconciliation
before the caller receives the write reply. `Map.observe` then correctly returns
without another snapshot.

`Admission.tla` models that ordering. `Current` allows background publication
between commit and observation but always requires a read: `ValidFixture` fails
with 11 generated / 11 distinct states. `Controlled` separates deliberately stale
and already-published fixtures: the invariant and completion property pass with
6 generated / 6 distinct states. Checked with TLC 1.7.4 and Java 25.

Run from this directory, with `TLA2TOOLS` naming the installed TLC jar:

```sh
java -cp "$TLA2TOOLS" tlc2.TLC -config Current.cfg Admission.tla
java -cp "$TLA2TOOLS" tlc2.TLC -config Controlled.cfg Admission.tla
```

The first command is expected to fail `ValidFixture`; the second must pass.
Use an external `-metadir` for disposable checker state.

The corrected `TestReplicaObservationCancellationAndClose` uses a real committed
write and a deliberately stale observer with no competing background publisher.
It invokes the shared `Map.observe` seam and joins that operation through the
same wait group used by `Close`. It checks caller cancellation, map closure,
remote-write preservation, no partial publication, later observation, and the
borrowed Redis client's continued availability.

`TestReplicaAlreadyObservedWriteSkipsSnapshot` retains the full public
`SetAndWait` path and delays the mutation reply until the actual replica owner
publishes the committed value. Requiring a tagged snapshot in that ordering
reproduced the old assertion failure in 1.04 seconds; the corrected control
expects success without another read. `TestReplicaConcurrentWriteObservations`
continues to cover public admission and concurrent requests.

This small model concerns fixture admission, not Redis transport cancellation or
full replica correctness. The tests exercise those implementation boundaries;
the adjacent replica-authority and lifecycle models cover their own assumptions.
