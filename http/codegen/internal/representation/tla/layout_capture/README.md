# Shared HTTP declaration layouts

Issue [#1071](https://github.com/CaliLuke/loom/issues/1071) showed that each
response target retained three complete service layout maps. Registry size
and target count both grew with methods times errors.

`LayoutCapture.tla` compares that dense capture with an immutable service
registry plus occurrence-local overrides. Borrowing mutable maps violates
snapshot stability. Keeping only structurally reachable keys loses the layout
of a sibling that composition/discriminator publication can later request.
The selected design shares the complete immutable registry and detaches every
changed fact, including overrides outside the current target graph.

## Explicit reproduction

Use TLA+ 1.7.4 `tla2tools.jar`, SHA-256
`936a262061c914694dfd669a543be24573c45d5aa0ff20a8b96b23d01e050e88`.
Checked with Homebrew OpenJDK 26.0.1. From this directory:

```sh
java -Xmx128m -XX:+UseParallelGC -cp "$TLA_TOOLS_JAR" tlc2.TLC \
  -workers 1 -config shared.cfg -metadir /tmp/<fresh> LayoutCapture.tla
```

Repeat with each configuration. Results:

| Configuration | Expected and observed result | Distinct states |
| --- | --- | ---: |
| `dense.cfg` | Exit 12, `LinearStorage` violation | 2 |
| `borrow.cfg` | Exit 12, `SnapshotStable` violation | 9 |
| `reachable.cfg` | Exit 12, `PublicationComplete` violation | 2 |
| `shared.cfg` | Exit 0, all invariants pass | 9 |

Parser/evaluation errors are not counterexamples. Earlier exploratory runs with
a Boolean/string comparison produced evaluation errors and are excluded.
These commands stay opt-in, outside ordinary tests and CI.

## Implementation and limits

- `codegen.TypeLayouts` stores detached physical facts, not semantic attributes.
  `AttributeContext.TypeLayout` resolves independently recorded false/true
  overrides before immutable facts. `Dup` shares only the immutable owner.
- `PrepareService` captures a schema registry once. `PrepareStreamSchema`
  reuses it, retaining the existing mixed-stream layout exception. An unprepared
  service gets a call-local snapshot, so later mutable layout changes remain
  visible rather than installing a stale service-wide cache.
- `recordServiceTypeLayouts` captures server/client registries after allocation.
  Later view-specific facts stay in the mutable emitter maps. HTTP projection
  capture retains every difference from the immutable registry.
- `captureProjectionContext` snapshots scalar context, shares the registry and
  detaches overrides. `wireContextIdentity` uses the registry's once-computed
  deterministic SHA-256 identity. Overrides recapture effective facts for their
  own identity; registry contents are not serialized into every contributor key.
  Internal key bytes change; generated contract bytes must remain unchanged.
- `TestErrorCatalogProjectionLayoutScaling` first failed with 51, 99 and 195
  map entries per response for 2, 4 and 8 methods and 8 errors. It now checks
  shared registry identity and bounded response-local storage.
- `TestProjectionContextCaptureAllocation` bounds target-capture allocations
  with 16 and 1024 declarations. Snapshot/override tests cover recorded false,
  missing facts, unrelated sibling overrides, recapture and detached exports.
  The identity-size regression first failed at 1,303 and 72,691 bytes for those
  registry sizes; fingerprints keep the layout portion constant in size.

The finite model has four layout keys, three targets and one root override per
target. Source mutation happens after capture. Stored facts are an abstract
resource count, not byte allocations. It assumes registry/override construction
is correct; direct Go tests cover mutations between captures and independent
presence/pointer/default facts. It does not prove arbitrary graph semantics,
overall generator memory bounds, GC/RSS behavior, concurrency or wire equivalence.
The related target-capture and body-projection models retain their separate
semantic and publication obligations. Complete-registry scans of mutable emitter
maps remain; this change removes repeated retained registry storage.
