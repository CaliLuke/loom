# Buffer lineage (#745)

One scope has a parent, child and sibling. The parent logs one entry before
With derivation. Each child logs once, and every context can flush once, in any
interleaving. Current buffers copy the parent's entry and retain separate flush
state. Shared mode retains one buffer owner. Counted output must contain every
admitted entry after a flush, exactly once; later logs must be immediately
emitted after the first scope flush.

The model assumes serialized buffer transitions. It proves no Go lock behavior,
callback/writer safety, process crash durability, memory bound or request lifetime.
Entry fields and destination/format selection require separate runtime controls;
the accepted policy keeps each entry's original settings. OutputSelection models
queued-entry snapshots separately from mutable logger settings; runtime tests
exercise both destination and format changes on a child or the same context. #744 configuration and #746 fresh request models
remain independent prerequisites. Production must always acquire logger before
buffer locks and never acquire a logger from the buffer owner.

Use pinned TLC1.7.4 (JAR SHA256
`936a262061c914694dfd669a543be24573c45d5aa0ff20a8b96b23d01e050e88`)
and Corretto Java25.0.4.1. Run from this directory with separate paths:

```sh
mkdir -p /tmp/loom-745-java-shared
java -Djava.io.tmpdir=/tmp/loom-745-java-shared -cp "$HOME/.cache/loom-tools/tla2tools-1.7.4.jar" tlc2.TLC -workers 1 -metadir /tmp/loom-745-state-shared -config shared.cfg BufferLineage.tla
```

Current must report a semantic invariant failure; shared must pass. Concrete
correspondence is the one buffer shared by With descendants and created anew by
Context initialization/requestContext. Direct lineage tests check both flush
directions, siblings, duplicates, field snapshots, deterministic ordering and
concurrent post-flush logs. HTTP/unary/streaming Endpoint tests prove the request
context flush reaches entries from derived service/method contexts.

Recorded TLC result: current fails CompleteAfterFlush (8 generated/8 distinct);
shared passes (84 generated/33 distinct). Run OutputSelection with current-output.cfg
and snapshot-output.cfg in separate state/JVM directories: current must fail
OriginalOutput; snapshot must pass. This abstracts writer and formatter together
as one output identity and does not claim immutable application-owned objects.

Implementation seam: logBuffer holds queue/disabled state and serializes both
immediate and buffered output; With shares it and requestContext creates it.
bufferedEntry retains the immutable outputs slice published by Context. Options
stay immutable because Context/WithOutputs detach configuration storage (#744).
Sibling output regression uses an atomic overlap detector without a writer lock.
The derived-buffering regression distinguishes shared flush state from isolated
configuration by creating a fresh request scope after a child disables buffering.
