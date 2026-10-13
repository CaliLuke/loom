# Logger configuration ownership (#744)

`With` derives two logger contexts from a root. All initially share identical
configuration. Current `Context` option application mutates the shared options;
shallow copy separates debug flags but still aliases output slice storage.
Detached snapshots separate both. Each child configures once in either order.
The invariant requires every logger to retain its own intended configuration.

The model abstracts an output writer by a token, not an actual writer or format
callback. It does not model output I/O, buffering/flush lifetime, child field
snapshots, retained application mutation of values, or Go synchronization. Reuse
of LogOption closures and repeated updates require direct tests. Slots stand for
fresh snapshot allocations; concrete code must allocate fresh snapshots on every
configuration update, not reuse a previously published slot.

Run with pinned TLC 1.7.4 (JAR SHA
`936a262061c914694dfd669a543be24573c45d5aa0ff20a8b96b23d01e050e88`)
and Corretto Java25.0.4.1, from this directory. Use separate JVM/state directories:

```sh
mkdir -p /tmp/loom-744-jvm-deep
java -Djava.io.tmpdir=/tmp/loom-744-jvm-deep -cp "$HOME/.cache/loom-tools/tla2tools-1.7.4.jar" tlc2.TLC -workers 1 -metadir /tmp/loom-744-state-deep -config deep.cfg OptionsOwnership.tla
```

Current and shallow must fail ConfigurationIsolated; deep must pass. This is
configuration ownership evidence only. #745 buffer lineage and #746 request
scope need their own models before implementation.

Implementation correspondence: `clue/log.Context` publishes a detached options
snapshot while holding that logger's existing lock; `With` can share an immutable
snapshot. Clone output, static-field and callback slices; callbacks/writers and
values themselves remain application-owned. `WithOutputs` must not mutate its
input slice or expose a reusable option's private slice to later option updates.
`options_ownership_test.go` checks all configuration fields through observable
output, parent/sibling isolation, reusable output options and concurrent updates.

Observed on parent `27255d025` with Go 1.27.2: all five option-isolation
cases, reusable/caller-slice cases, and all three transport cases fail as
expected; the concurrent test reports actual races. Existing With, no-op,
debug toggle and disabled-buffer controls pass. TLC current fails at 2/2
states, shallow fails at 4/4, and deep completes at 13/9 (generated/distinct).
The Context buffer-retention control passes on the parent and must stay green.
