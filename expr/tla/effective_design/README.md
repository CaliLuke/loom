# Effective design composition: #755 / #757

This bounded lifecycle model informed the approved composition implementation
in `roadmap/effective-design-readiness.md`. The original red checkpoint used
`ce3f6a733f19de30b50986f86e9ce4cef06e48cb`; the corrected regressions now pass.
The model results remain design evidence within the bounds below, separate
from the implementation and generated-output acceptance checks.

## Run and expected results

The inspected shared JAR is `/Users/luca/.cache/loom-tools/tla2tools-1.7.4.jar`,
SHA-256 `936a262061c914694dfd669a543be24573c45d5aa0ff20a8b96b23d01e050e88`.
Despite its filename, it reports TLC 2.19, revision `5a47802` (8 August 2024).
Recorded runs used Corretto Java 25.0.4.1. From this directory:

```sh
export TLA_TOOLS_JAR=/Users/luca/.cache/loom-tools/tla2tools-1.7.4.jar
java -XX:+UseParallelGC -Xmx1g -cp "$TLA_TOOLS_JAR" tlc2.TLC -workers 1 -metadir /tmp/loom-effective-design-tlc/overwrite -config legacy-overwrite.cfg EffectiveDesign.tla
java -XX:+UseParallelGC -Xmx1g -cp "$TLA_TOOLS_JAR" tlc2.TLC -workers 1 -metadir /tmp/loom-effective-design-tlc/late -config legacy-late-validation.cfg EffectiveDesign.tla
java -XX:+UseParallelGC -Xmx1g -cp "$TLA_TOOLS_JAR" tlc2.TLC -workers 1 -metadir /tmp/loom-effective-design-tlc/shared -config shared-base-mutation.cfg EffectiveDesign.tla
java -XX:+UseParallelGC -Xmx1g -cp "$TLA_TOOLS_JAR" tlc2.TLC -workers 1 -metadir /tmp/loom-effective-design-tlc/overlay -config copied-overlay.cfg EffectiveDesign.tla
java -XX:+UseParallelGC -Xmx1g -cp "$TLA_TOOLS_JAR" tlc2.TLC -workers 1 -metadir /tmp/loom-effective-design-tlc/corrected -config corrected.cfg EffectiveDesign.tla
```

Run sequentially: this TLC build extracts standard modules into the shared Java
temporary directory, so parallel processes can race on those files. One parallel
attempt hit a missing `Sequences.tla` parse error; it was not counted as a
counterexample and the affected configuration passed its expected failure check
when rerun sequentially. Keep state directories outside the checkout.

Results on 2026-10-12 after the final model revision:

| Configuration | Required outcome | Generated / distinct states | Depth |
| --- | --- | ---: | ---: |
| legacy-overwrite | Exit 12, `AuthoredRefinementRetained` | 85 / 73 | 4 |
| legacy-late-validation | Exit 12, `AdmissionMatchesCompleteContract` | 109 / 89 | 4 |
| shared-base-mutation | Exit 12, `SourceUnchanged` | 157 / 121 | 5 |
| copied-overlay | Exit 12, `AdmissionMatchesCompleteContract` | 145 / 113 | 4 |
| corrected | Exit 0, all five invariants | 396 / 246 | 9 |

## Bounds, specification and correspondence

The initial state has three scenario classes, two DSL orders and the presence or
absence of an explicitly local default: 12 initial states. Two consumers advance
independently, including either order and arbitrary interleavings. Each can
mutate its emitted annotation once. Cell 0 represents the source, and cells 1/2
represent owned copies. Predicates are two opaque labels, defaults are the
finite values 1/2/5, and protobuf fields are `a`, `c` and `n` with tags 1/1/2.

Candidate admission checks concrete tag uniqueness in the visible field set and
whether inherited defaults have one value or an explicitly local selector.
`ShouldAdmit` states the expected scenario outcome independently. The old phase
checks only `{c}`; composition later adds `{a}`, and Finish exposes the accepted
collision. Correct composition publishes both fields before that same check.

| Model fault / property | Production seam and Go test |
| --- | --- |
| Base overwrites local predicates and default | `expr/attribute.go:Finalize` / `Merge`; `TestExtendCompositionRefinement` loses maximum 100, local description and default 5 in both orders |
| Validate local tags, then merge inherited fields | `eval/eval.go:RunDSL`, `expr/grpc_field_tags.go:rpcMessageAttribute`; `TestExtendCompositionRejectsInheritedRPCNumberCollision` covers named and inline payloads |
| Completed field points to source cell | `Merge` inserts base `AttributeExpr` pointers; `TestExtendCompositionOwnsInheritedFields` mutates description, metadata slice and validation bound, observing both base and second consumer change |
| Copied inherited default treated as local selector | **Rejected repair**, not a claim that current Merge implements this algorithm. `dsl/attribute.go:Attribute` copies the inherited field before local DSL. The control demonstrates why a repair cannot infer explicitness from a non-nil `DefaultValue`; `TestExtendCompositionPeerDefaults` tests actual evaluator admission and local choice |
| Corrected sequence | `RootExpr.Prepare` calls `composeDesign` unconditionally before dependent preparation/validation; `designComposer` owns fields and retains detached constraint inputs, and the existing effective-constraint owner conjoins them |

The model does not execute Go cloning, named-type extraction, enum equality,
nullable semantics, metadata interpretation or recursive inheritance traversal.
Diamonds, recursive fields, inheritance-cycle diagnostics, idempotent repeated
preparation, source locations, generated code and transports remain explicit
Go/test obligations in readiness. It does not prove arbitrary conjunction
satisfiability or the existing Lean alias theorems for an inheritance DAG.

Deadlock checking is disabled because both consumers reaching terminal state
and completing their optional mutation is an intentional terminal state. No
liveness property, fairness assumption, state constraint or termination proof
is claimed. Corrected safety is checked over every reachable bounded state.

## Historical red evidence

```sh
go test ./expr -run '^TestExtendComposition' -count=1 -v
```

With Go 1.27.2 and unchanged production source, this returns exit 1: ten leaf
cases fail for the intended behavior (two overwritten refinements, incompatible
type, contradictory bounds authored before Extend, two shared-mutation targets,
two default-peer cases, two gRPC collisions). Three preservation controls pass:
an invalid default already rejected when the inherited minimum has been copied,
and implicit nullable inheritance in both DSL orders. The failing checks are
not evidence that all implementations or transport paths are covered.

After independent review, preserve this summary and model/test sources, then
remove this task's `/tmp/loom-effective-design-tlc` state output once no checker
process uses it. Keep the shared JAR and installed Java toolchain.

## Implementation correspondence discovered during composition

An Extend-only inline payload initially carries the shared `expr.Empty`
sentinel. `Dup(Empty)` deliberately returns that sentinel. Treating it like an
ordinary named object during composition mutated every unrelated empty method.
The composer now replaces it with an owned object; the direct
`TestExtendCompositionPreservesEmptySingleton` regression checks both method
preparation orders. The bounded model's source-ownership invariant covers the
abstract obligation, but does not model this concrete singleton representation.

The production constraint graph retains local authored layers and Extend
parents inside the existing value-occurrence owner. Nested collection predicates,
enum member identity, named ancestry, nullable representation and metadata
interpretation remain implementation-test obligations outside the bounded TLA
state space. `TestExtendCompositionGraphs`, `TestExtendCompositionMetadata`,
`TestExtendCompositionNestedConstraints` and the HTTP/gRPC/OpenAPI generated
specimen cover these additional boundaries.

`TestExtendCompositionOwnsConstraintSources` additionally mutates an original
base bound after composition and confirms a subsequent derived constraint query
retains its owned input. `TestExtendCompositionCoalescesEquivalentDefaults`
checks admitted `int(1)` and `int64(1)` in an Int64 field. The latter failed with
raw reflected equality; inherited default comparison now resolves both values
in the same effective occurrence using `effective_default.go`. Neither concrete
snapshot behavior nor numeric representation equality is encoded by this model.

## Recursive graph review counterexample

Independent review exposed a boundary absent from `EffectiveDesign.tla`:
`Base.next` may reference `Node` while `Node` extends `Base`. This member cycle
is valid; an inheritance-only cycle is not. Base-first traversal incorrectly
rejected the mixed cycle; Node-first traversal snapshotted an unfinished named
reference. Merely suppressing the first diagnostic is therefore insufficient.

`RecursiveComposition.tla` isolates this two-declaration shape dependency. Both
inputs contain the recursive member edge; one also contains a genuine Extend
cycle. Composition can advance only when inheritance dependencies are complete.
The legacy mixed-cycle configuration represents the observed Base-first
rejection; the early-copy configuration represents the unfinished Node snapshot
and the rejected repair that only suppresses the cycle diagnostic. The corrected
configuration defers snapshots until both shapes are complete. The additional
complete-unbound control shows that shape completeness alone does not preserve
structural identity: a copied self-reference may point to a separate same-ID
clone instead of the owning declaration. The Boolean `ownerBound` abstracts the
canonical-binding obligation at publication; it does not simulate Go pointers.

Using the same pinned JAR and Java toolchain, run from this directory:

```sh
java -XX:+UseParallelGC -Xmx1g -cp "$TLA_TOOLS_JAR" tlc2.TLC -workers 1 -metadir /tmp/loom-effective-recursive-tlc/mixed -config recursive-mixed-cycle.cfg RecursiveComposition.tla
java -XX:+UseParallelGC -Xmx1g -cp "$TLA_TOOLS_JAR" tlc2.TLC -workers 1 -metadir /tmp/loom-effective-recursive-tlc/early -config recursive-early-copy.cfg RecursiveComposition.tla
java -XX:+UseParallelGC -Xmx1g -cp "$TLA_TOOLS_JAR" tlc2.TLC -workers 1 -metadir /tmp/loom-effective-recursive-tlc/unbound -config recursive-unbound.cfg RecursiveComposition.tla
java -XX:+UseParallelGC -Xmx1g -cp "$TLA_TOOLS_JAR" tlc2.TLC -workers 1 -metadir /tmp/loom-effective-recursive-tlc/complete -config recursive-complete.cfg RecursiveComposition.tla
```

Checked results: mixed-cycle exits 12 on `OnlyInheritanceCyclesRejected` (3
states); early-copy exits 12 on `CopiesSeeCompleteShape` (7 states); unbound
exits 12 on `StructuralRecursionUsesOwner` (7 states); complete exits 0 with all
three invariants (7 states). Generated and distinct counts coincide.
An initial model expression lacked grouping around a disjunction and produced a
TLC evaluation exception; it is excluded from evidence. The corrected run is
recorded separately. No liveness or arbitrary-graph proof is claimed; the model
abstracts named-reference shape, not Go cloning or metadata/constraint storage.

The implementation boundary is `compositionGraph`: discovery resolves named
references, inheritance validation follows only alias and Extend edges, and
member traversal composes owned shapes while copies retain named references.
`designComposer.detachNamedCopies` computes every replacement before publishing
any, after composition completes. Local constraint and inherited constraint
snapshots use the same deferred-copy phase as structural fields, but have no
canonical owner binding: they remain fully detached. Structural copies seed the
copy registry with their enclosing declaration, so self and mutual back edges
resolve to that owner without losing occurrence-local metadata. This prevents
publication order from changing subsequent snapshots and avoids an extra
same-ID node that changes OpenAPI schema fingerprints. `TestExtendCompositionRecursiveBase`
and `TestExtendCompositionMutualRecursion` check both declaration orders,
complete nested shapes and canonical back references.
`TestCompositionRecursiveConstraintSnapshots` mutates canonical field validation
and metadata after composition and confirms both local and inherited snapshots
remain unchanged; it also checks that derived occurrence metadata is owned.
These concrete pointer and mutation properties remain Go obligations beyond the
model's abstract binding flag. `RecursiveEffectiveCompositionDSL` compiles both orders
through HTTP and gRPC; existing genuine-cycle rejection remains covered.

Peer JSON metadata is a separate direct-test obligation: the resolved full tag
and options must agree even when one peer uses the name-only key. The shared
`internal/encodingmeta.JSONTag` owner supplies normalization to both composition
and rendering, using shared `ExplicitPresence` representation facts and the
existing effective containing-requiredness owner. It is not encoded by this
shape-order model. After re-review, remove task-owned checker output under
`/tmp/loom-effective-recursive-tlc` once no checker process uses it.
