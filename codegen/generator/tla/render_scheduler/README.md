# Admitted artifact scheduling

`RenderScheduler` models four admitted artifacts, two workers and every subset
of deterministic per-file failures. It compares completion-order diagnostics
with ordered dispatch, stop after the first observed error, joining all started
jobs and returning the earliest failing plan index. It also tracks pending and
active render references: completed work releases its reference; failure drops
unstarted references without releasing active callbacks early.

With official TLA+ v1.7.4 (TLC 2.19), jar SHA-256
`936a262061c914694dfd669a543be24573c45d5aa0ff20a8b96b23d01e050e88`, and
Corretto Java 25.0.4.1:

- `Ordered.cfg` passes bounded activity, unique ordered dispatch, stopped
  dispatch, joining, error precedence, success completeness and owned-reference
  release, plus completion under weak fairness: 319 generated, 246 distinct
  states, depth 10.
- `CompletionOrder.cfg` violates `CorrectError`: 183 generated, 157 distinct,
  depth 6. Jobs 1 and 2 fail; reporting 2 merely because it completes first
  violates stable plan-order precedence.

From this directory:

```sh
java -Xmx512m -cp "$TLA_JAR" tlc2.TLC -workers 1 \
  -metadir "$TLC_STATE" -config Ordered.cfg RenderScheduler.tla
```

Repeat with CompletionOrder.cfg, requiring the named invariant failure. Keep
checker states, traces and raw logs outside the repository.

`codegen/generator/artifact_plan.go` implements the ownership/dispatch boundary.
Its pending slice is private, each dispatch clears its slot, the result channel
holds at most W metadata records, failure clears unstarted references, and return
joins active goroutines. `artifact_plan_test.go` checks admission conflicts,
ordered failure, joining, bounded start and caller-section preservation. Existing
merge, write-intent, cleanup and CLI staging tests cover integration contracts.

The model assumes independently determined file outcomes, stable filesystem
state, finite callbacks and successful goroutine scheduling. A callback's shared
state or existing filesystem aliases can violate independence. Admission rejects
lexical file/descendant overlaps, case-folded path collisions and framework
manifest conflicts before rendering, but it is not a complete filesystem alias
proof. CLI installation remains a separate transaction owner.

The model bounds active files, not bytes: a single artifact can be large,
analysis and pending closures remain live, and plugins can retain their own
references. Arbitrary Section.Write and finalizer callbacks cannot be interrupted
safely; cancellation here stops new dispatch and joins started work.
