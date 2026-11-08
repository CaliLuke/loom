# Discriminator import representation choice (#1005 design)

`DiscriminatorImport.tla` checks the classifier for an imported discriminated
`oneOf`. It inspects each branch, then decides one of three outcomes: the
tagged envelope, the full-object untagged union with a hint, or a named block.

Bounds:

- Branch 1 ranges over every modeled fact:
  - envelope or flat shape;
  - discriminating enum equal to its mapping key, another value, or free;
  - closed (`additionalProperties: false`);
  - an extra tag constraint;
  - whether it requires the property;
  - whether it is referenced elsewhere.
- Branch 2 ranges over four controls.
- Documents vary:
  - the importer source version: 3.0, 3.1 or 3.2. Export targets are 3.2.0
    and 3.1.1 and are not modeled here;
  - whether `defaultMapping` is present;
  - a JSON or form occurrence;
  - union-level closure.

`DomainPreserved` compares the source `oneOf` with the representation's
regenerated contract over 64 JSON values. The values vary:

- the discriminating value;
- an undeclared member;
- an over-long tag;
- payload validity;
- payload nesting.

```sh
java -cp "$TLA2TOOLS_JAR" tlc2.TLC -workers 1 -config checked.cfg -metadir "$TMPDIR/discriminator-checked" DiscriminatorImport.tla
```

Observed with Java 25.0.5 and TLC 1.7.4 (`tla2tools` sha256
`936a262061c914694dfd669a543be24573c45d5aa0ff20a8b96b23d01e050e88`). All
configurations check `Classified`, `DomainPreserved`, `TagAgreement`,
`NeverTaggedOptional`, `VersionRules` and `Admits31Optional`.

| Configuration | Expected and observed |
|---|---|
| `checked` | pass, 36 864 distinct states (9 216 initial documents × 4 steps) |
| `lenient-closed` | exit 12, `DomainPreserved`: a closed envelope tagged; regenerated envelope accepts an undeclared member |
| `lenient-tag` | exit 12, `DomainPreserved`: tag `maxLength` dropped |
| `lenient-enum` | exit 12, `TagAgreement`: wire tag `o` is not the branch's mapping key |
| `tagged-everywhere` | exit 12, `DomainPreserved`: tagged in a form body, whose Loom codec flattens object branches |
| `hint-routing` | exit 12, `DomainPreserved`: validating only the mapped branch changed the `oneOf` outcome |
| `prior` | exit 12, `Admits31Optional`: a 3.1 SHOULD treated as a MUST |
| `same-30-31` | exit 12, `VersionRules`: a 3.0 optional property admitted despite OAS 3.0.3's MUST |

Assumptions and limits:

- **Closure.** The undeclared member `extra` is undeclared by every branch, so
  closing first and counting first coincide here. The distinguishing case is
  checked in
  [`UntaggedClosure.tla`](../../../../pkg/tla/untagged_json/README.md)
  (`ClosureCloseFirstDiscriminated`).
- **Mappings.** Mapping resolution and regeneration are proved separately in
  [`DiscriminatorMapping.lean`](../../../../http/codegen/openapi/internal/ir/lean/discriminator_metadata/README.md).
  Here each branch's key is fixed.
- **Not modeled.** Annotations, envelope names and descriptions, and payload
  sibling preservation. These are direct-test obligations.
- **Covered in Lean instead.** The `allOf` parent-hint form, alias keys,
  explicit keys that shadow implicit ones, and duplicate alternatives are
  mapping or requiredness questions with no union-domain state. They are
  proved in `DiscriminatorMapping.lean`. The exact-one runtime result for
  duplicate alternatives is a #1014 matcher test obligation.
- **Not proved.** The model does not prove Go import, rendering, generated
  decoders or OpenAPI serialization.

Implementation seams:

- import admission and representation selection in `internal/openapiimport`;
- the tagged decoder in `codegen/uniongen/codecs.go`;
- the exporter in `http/codegen/openapi/internal/ir/analyzer.go` and
  `discriminator_metadata.go`.

The [import validation record](../../discriminator_import_validation.md) maps
the exactness, version, mapping and generated-roundtrip obligations to durable
regression tests. Native `CatalogEvent` symmetry and wire checks remain owned by
the HTTP exporter and transport tests.

This is opt-in design evidence. It is not a build, CI or release gate.

## Implementation correspondence

The implementation reuses these checked bounds; it does not change the model's
logical assumptions. The model's `json` and `form` values describe codecs, not
MIME types: JSON-content fields in form or multipart bodies take the `json`
state, while flattened form roots take `form`. JSON query parameters retain
FullObject because the existing native parameter grammar rejects tagged unions;
that admission boundary is checked directly, outside this JSON/form model. Explicit style serialization
continues to use the shared resolver's admitted shape, rather than pretending a
JSON envelope has a bracketed codec.

- `discriminator.go` defers hint validation until the normalized declaration
  graph is complete. It joins resolver outcomes by authored occurrence and
  compares alternative identities independently of their value-domain shape.
- `discriminator_envelope.go` checks exactness before any representation change.
  `discriminator_occurrence.go` computes a decreasing fixed point: a non-JSON
  occurrence or enclosing FullObject branch can remove tagged eligibility;
  no visit can restore it. Shared field codecs supply JSON eligibility.
- `render_discriminator.go` retains aliases and envelope annotations. Synthesized
  envelope declarations remain in the admission graph but their tagged owner
  emits the representation, avoiding a second declaration reserving the name.
- `analyze_reference_siblings.go` retains explicit union closure at a reference
  occurrence, using the same count-before-close contract as adjacent `oneOf`.

`TestDiscriminatorEnvelopeSelection`, `TestDiscriminatorExactEnvelopeFacts`,
`TestDiscriminatorOccurrenceSelection` and
`TestDiscriminatorSharedOccurrenceAndNestedBranch` cover the exactness and codec
conditions. `TestDiscriminatorUnionVersionMatrix` and
`TestDiscriminatorParentVersionMatrix` cover the three source-version rules.
`TestDiscriminatorImportedRoundtrip` compares source and generated membership
and runs the existing closure oracle over explicit accepted/rejected values.
`TestOpenAPIImportDiscriminatorRoundTrip` generates, compiles and runs tagged,
FullObject and duplicate-alternative decoder controls before and after CLI
reimport, with independent-process byte comparisons.

The fixed-point traversal, parser annotation presence, source identity join,
field/parameter-codec capture and generated code are implementation obligations, not new
claims proved by the finite TLA model. The recorded checker results above remain
historical design evidence. No proof execution is added to ordinary gates.

## Deferred normalization ownership

`DeferredNormalization.tla` models two normalization copies followed by deferred
hint and sibling checks. The last copy is the emitted schema. It ranges over
valid/invalid hints and redundant/nonredundant siblings; it does not model JSON
values or parser details.

| Configuration | Observed with the pinned TLC above |
|---|---|
| `deferred-value-copy` | exit 12, `HintRetained`: a late hint update affects only the discarded source |
| `deferred-shared-hint` | exit 12, `AdmissionPreserved`: hint retention alone leaves a sibling rejection on the discarded source |
| `deferred-shared-and-retargeted` | pass, 16 distinct states: shared hint carrier plus checks retargeted to the surviving schema |

Run each explicitly, for example:

```sh
java -XX:+UseParallelGC -cp "$TLA2TOOLS_JAR" tlc2.TLC -workers 1 \
  -config deferred-shared-and-retargeted.cfg \
  -metadir "$TMPDIR/discriminator-deferred" DeferredNormalization.tla
```

`analyzer.schemaSource` allocates the hint carrier before normalization can copy
it. `copyNormalizedSchema` transfers pending sibling checks when nullable or
singleton normalization replaces a value. The source diagnostic path remains
unchanged. `TestDiscriminatorSurvivesNullableNormalization` and
`TestDiscriminatorNullableNormalizationKeepsSiblingAdmission` reproduce the two
implementation defects and exercise their corrections. The original shared
oneOf normalization tests remain ordinary implementation gates.


## ParentHint declaration projection

`ParentHintProjection.tla` separates source admission, validation dependencies,
weak declaration retention and final export. The source has one named and one
inline direct child; a third target represents a non-child fallback. Every
subset of representable targets, strong selected uses and partial/strict mode
is checked. Source admission uses the complete canonical graph, before operation
selection; a child found only in an unselected operation keeps the source hint
valid but cannot establish coherence of the selected projection.

The initial pruner policy failed `ProjectedHintDefined`. Making every child a
strong dependency failed `ValidationIndependent`. The checked policy retains
representable named child closures, counts retained inline identities as child
evidence, and reports a hint-local strict refusal when no child can be emitted.
Partial import may then omit the hint with an explicit report, preserving the
parent's validation contract. The model was extended when implementation exposed
named-only coherence and selected-source timing failures. These are separate
from original source validity; the model does not authorize rejecting the
parent's value schema because a child is unsupported.

| Configuration | Observed with the pinned TLC above |
|---|---|
| `parent-checked` | pass, 384 distinct states |
| `parent-pruned` | exit 12, `ProjectedHintDefined` |
| `parent-strong-all` | exit 12, `ValidationIndependent` |
| `parent-unreported` | exit 12, `OmissionReported` |
| `parent-shared` | exit 12, `SourceIsolated` |
| `parent-drop-unused` | exit 12, `ProjectedHintDefined` |
| `parent-named-only` | exit 12, `ValidationIndependent` |
| `parent-source-selected` | exit 12, `SourceAdmissionIndependent` |

```sh
java -XX:+UseParallelGC -cp "$TLA2TOOLS_JAR" tlc2.TLC -workers 1 \
  -config parent-checked.cfg -metadir "$TMPDIR/parent-hint" ParentHintProjection.tla
```

Implementation correspondence:

- `directAllOfChildren` consumes #957's canonical typed allOf reference edges
  and verifies the containing Schema grammar role. It neither normalizes
  unselected operations nor invents implicit component names.
- `prepareDiscriminatorChildren` uses the existing planner and source diagnostics
  to retain immutable base-qualified child plans. Each component pruner owns
  weak-edge eligibility for its operation projection; the transitive fixed point
  below discharges nested hint obligations against actual retained occurrences.
- `componentPruner.visitSchema` inventories retained schema occurrences through
  the shared walk, with pointer identity guarding cycles. Projection compares
  their canonical source identities with the source child inventory.
- `omitUnprojectableParentHints` detaches affected declarations; it does not
  mutate another operation or the original document. Final combined projection
  reconciles reports when a second retained operation supplies an inline child.
- Export must preserve retained children and allOf edges even without explicit
  mappings or body reachability. `parent-drop-unused` is the negative control;
  the generated implementation regression remains required independently.

`TestDiscriminatorParentSelection`, `TestDiscriminatorParentInlineSelection`,
`TestDiscriminatorParentProjectionIsolation`,
`TestDiscriminatorParentCanonicalChildIdentity` and
`TestDocumentSchemaOccurrencesCycles` cover the implementation joins, transitive
unsupported dependencies, shared occurrence isolation and external aliases.
`TestDiscriminatorImplicitParentRoundtrip` and
`TestDiscriminatorPartialParentRoundtrip` check publication and unchanged parent
membership. The bounded model abstracts the planner's representability result;
it does not prove the parser, arbitrary graph traversal, Go generation or export.


## Transitive weak-child qualification

`ParentHintQualification.tla` refines the representability assumption in the
projection model. The independent review reproduced a selected parent P with a
valid child C2 and a second implicit child C1. C1 strongly referenced parent Q,
whose only child had unsupported composition. The previous one-pass qualifier
checked C1 before nested hint coherence was enabled, retained C1 and Q, and then
rejected P's selection. C2 alone was a faithful projection. The maintained
`TestDiscriminatorTransitiveChildQualification` reproduces that failure.

The model abstracts the shared planner's strong-closure result into base
validity plus typed parent-hint obligations. Three named child identities belong
to two parents; every base-valid subset, obligation graph and strong-target
subset is explored. An inline witness is absent or supplied by child 1 or child 2
for Q. An operation with an inline Q child is selected or unselected. The model
computes actual reachability through strong and candidate weak edges, rather
than treating all source occurrences as witnesses. Removing any currently
unqualified child is allowed, so terminal maximality also checks removal order.

Starting from base-valid candidates and repeatedly removing unsatisfied
obligations reaches the greatest fixed point. A mutually supporting finite
cycle can qualify; an unsupported descendant propagates removal. Explicit and
fallback targets remain strong even when they are not viable weak children.

| Configuration | Observed with pinned TLC 1.7.4 / Java 25.0.5 |
|---|---|
| `qualification-checked` | pass, 99 296 distinct states after root-owned/shared-walk refinement |
| `qualification-one-pass` | exit 12, `ClosureRepresentable`: a transitive hint obligation was ignored |
| `qualification-empty-seed` | exit 12, `MaximalAndOrderIndependent`: supported declarations, including productive cycles, are lost |
| `qualification-weaken-strong` | exit 12, `StrongRetained`: explicit/default dependency was pruned |
| `qualification-roots-only` | exit 12, `MaximalAndOrderIndependent`: a root-owned or retained operation witness is missed |
| `qualification-roots-only-sibling` | exit 12, `MaximalAndOrderIndependent`: a reachable sibling's inline witness is missed |
| `qualification-unselected-witness` | exit 12, `ClosureRepresentable`: an unselected operation supplies an invalid witness |
| `qualification-root-unaware` | exit 12, `MaximalAndOrderIndependent`: a child is dropped for a root-owned obligation |
| `qualification-weak-owned` | exit 12, `ClosureRepresentable`: a candidate's own reach exempts the obligation it introduces |

```sh
java -XX:+UseParallelGC -cp "$TLA2TOOLS_JAR" tlc2.TLC -workers 1 \
  -config qualification-checked.cfg \
  -metadir "$TMPDIR/parent-hint-qualification" ParentHintQualification.tla
```

`planDocumentGraph` owns base planning and returns unresolved projection as typed
`[]NamedSchema` obligations. Ordinary `planDocument` turns those obligations into
projection diagnostics. `schemaDeclarationQualification` retains them for immutable base child plans;
it does not filter diagnostic text or bypass projection through a preparation
flag. `projectWeakChildren` computes the fixed point for each
`pruneComponents` call. Candidate weak edges are supplied to the existing
`componentPruner` through `walkSelectedRoots`, which visits selected operation
roots once per round. `parentHintsWithoutWitness` compares typed introduced
obligations with that walk's `retainedIdentities`. The same two helpers check
final projection coherence. There is no second schema walker or global
eligible-child field.

Direct tests cover the reviewed failure, a no-dependency control, supported
nested hints, an unsupported seed spanning two iterations, occurrence-owned
inline evidence, productive cycles with and without an independent sibling,
explicit/default retention and reversed declaration order. Existing parent
selection, partial omission and source-copy isolation tests remain green.
`TestDiscriminatorProjectionKeepsPlannerDiagnostics` checks that separating typed
obligations does not suppress projection errors alongside base planning errors.
When P explicitly maps C1, C2's strong closure also includes that authored edge,
so Q remains one of C2's obligations. P's explicit edge already places Q in the
selected strong graph, however; see the root-owned correction below. C1 and Q
remain strong targets, Q's projection obligation is reported, and C2 is retained.

The model does not prove the planner's schema grammar, canonical-source joins,
rendering or Go implementation. It establishes the fixed-point properties under
the stated finite abstraction. This remains opt-in design evidence, not a gate.


### Shared closure witness correction

Independent review of the first fixed-point implementation found a stronger
witness case. C1 required Q; sibling C2 extended P and contained an inline child
of Q. C1's own strong closure lacked that child, and the old candidate set
recorded only named roots. Selection silently dropped C1 although the emitted
graph including C2 made C1 coherent. `TestDiscriminatorSharedInlineWitness`
reproduced the failure, with an unrelated inline declaration as a negative
control. The initial 18 560-state model abstracted a witness to its own candidate
and did not cover shared closure; that result remains historical evidence only.

The refined 50 240-state model evaluates the reachable closure under candidate
weak edges. The reviewed sibling counterexample and a selected-operation
counterexample are both explicit negative configurations. The checked policy
allows either witness only when reachable in the projection. Immutable base
plans are shared on `Document`; eligibility is computed by pruning and never
written into a shared `Discriminator`. A narrow projection cannot alter a later
broad projection or full analysis.

`TestDiscriminatorSelectedOperationInlineWitness` checks selected/unselected
operation scope and the final combined partial projection.
`TestDiscriminatorProjectionPlansAreLocal` alternates projections of the same
normalized document, compares full Render bytes after each, and verifies that a
witness operation rejected by partial import cannot support the final graph.
The finite model abstracts schema planning and operation representability; the
actual skipped-operation and copy-isolation behavior remains a direct-test
obligation. Formal execution remains opt-in and outside ordinary gates.


### Root-owned obligations and shared walk correction

Re-review found that a weak child was rejected for any unwitnessed obligation,
including a hint the selected strong graph already emits. With `mapping: {one:
C1}` and Q's only child unsupported, C2's closure reaches Q through P's explicit
edge; with a selected operation strongly returning Q, C1 reaches Q directly.
Strict import already fails on Q, but partial import silently dropped C2 or C1
although re-adding it was planner- and render-valid. That obligation is root
owned: the hint is emitted without any weak child, and a child can only add
witnesses. Each round also walked every selected root once per candidate, and
rescanned the whole occurrence inventory per candidate.

The model adds `rootQ`, a selected operation that strongly references Q without
a child. `RootOwned` is the set of hints visited by the strong graph alone
(selected roots, operation references, explicit/default targets). The contract
judges a child only on `Introduced(c) == needs[c] \ RootOwned`. The rejected
alternative, `weak-owned`, exempts hints visited through current candidates; a
child that alone reaches Q then exempts itself and publishes an unwitnessed Q.
`root-unaware` is the previous policy. Its 3-state counterexample is the
selected-root case above; pinning the explicit case (strong C1 and candidate C2
both needing Q) also fails in 3 states, while `fixed-point` keeps C2.

The checked policy evaluates every reached candidate against one shared walk,
`Reach(strong, eligible, 3)`, and drops unreached candidates. They cannot be
emitted, and their weak edges are never traversed. Since eligibility of an
unreachable child is unobservable, `MaximalAndOrderIndependent` now compares the
result with `Emittable`, the viable children reachable in the maximal graph.
`blocked` is final coherence: a retained child without base validity, or a
visited hint without a retained witness. `StrongFailuresRemain` requires blocking
for a base-invalid strong target or one whose obligations stay unwitnessed in
the maximal graph. `OptionalCannotPoison` forbids blocking when that maximal graph
is coherent. The new `WeakIntroducesNoOmission` states that every partial
omission is root owned. Only `qualification-checked.cfg` changed among the
existing configurations (it adds that invariant). The other six are unchanged
and fail with their recorded invariants; `roots-only-sibling` keeps its 3-state
trace. `roots-only` keeps its historical predicate without the exemption, so its
first counterexample is now the root-owned case. A selected operation's inline
Q child also places Q in the strong graph, so that witness is subsumed.

`walkSelectedRoots` is the single selected-root walk: operations, source-only
grammar dependencies and effective security under the supplied weak edges. It
computes the strong-only `forced` names once per projection, then one walk per
round. `retainedIdentities` builds that round's witness inventory once, and
`introducedObligations` filters a copy of each immutable base plan.
`pruneComponents` retains the converged walk directly rather than walking again.
A projection with no candidate children walks once: the strong walk is the
projection walk.

`TestDiscriminatorRootOwnedObligation` covers the explicit, default and
selected-root cases in strict and partial imports, an unselected-root control
and mutually introduced obligations. It checks retained names, the single
projection diagnostic or omission and its operation, the detached omitted hint,
and Render. `TestDiscriminatorProjectionScaling` logs 20/60/120 durations without
asserting them. The expected values for explicit/default retention in
`TestDiscriminatorTransitiveChildQualification` now include C2.

The model still abstracts obligations, witnesses and the strong graph to two
parents and three children. It does not prove the Go walk's monotonicity,
source-only grammar retention or complexity. Direct tests and measurements check
those.

Weak ParentHint edges are normalized-only: the raw child→parent `allOf`
reference never leads from a parent to its children. A parent retained first
by the source walk therefore relies on the shared `retainComponent` owner,
which walks its normalized body (the generic #957 correction in
`../blocked_selection/README.md`). That README and `SourceRetention.tla`
describe the owner as #957 delivers it: normalized bodies only, body-less
example and security-scheme kinds, and source prefixes reset per operation.
#1005 extends the same owner as described in the next section; the #957
description and its checked results remain accurate for #957 alone.
`TestDiscriminatorSourceRetainedWeakChild`
is the public regression, built on the generic source-retention fixture;
`TestComponentPrunerSourceRetainedWeakChild` is its synthetic seam.


### Raw grammar dependencies of normalized-only retention

Independent review found the inverse direction of the generic correction.
#1005 adds three normalized-only edge classes to authored components: weak
children, explicit mapping targets and `defaultMapping`. Mapping occurrences
are excluded from bundled references, and a weak child has no raw
parent-to-child edge. A component retained only through such an edge had its
normalized body walked, but its raw prefix never was. A blocker on a reference
that normalization dropped, such as a redundant closed-union sibling, therefore
vanished. Selected and partial import succeeded without it, and qualification
disagreed with the same blocker placed where it survives normalization.

`RetentionClosure.tla` keeps raw edges, normalized edges, weak edges and strong
mapping edges separate. Its oracle is the least fixed point of norm ∪ raw from
every retained node. Qualification uses the same closure without weak edges,
and projection follows weak edges only to qualified children. Five fixed
scenarios replace an edge-set product:

- the reviewed weak child;
- a strong explicit or default target;
- a raw operation-root control;
- normalized blocker placement;
- an invalid weak child beside a valid cyclic sibling.

Both operation orders and every worklist order are explored.

Root ran these with pinned TLC 1.7.4 / Java 25.0.5, before production changed.
The model is `0bf8794e…` and the logs are in `projection_r5/model-runs`:

| Configuration | Observed |
|---|---|
| `retention-checked` (`closure`) | pass: all seven invariants and `Termination`, 226 distinct states |
| `retention-body-once-weak` (R4 owner) | exit 12, `QualificationExact`: C qualifies, X is lost |
| `retention-body-once-strong` (R4 owner) | exit 12, `ExactClosure`: the strong target keeps {P, M, U} without X |
| `retention-body-once-controls` | pass: raw-path and placement outcomes; `QualificationExact` deliberately omitted |
| `retention-projection-only` | exit 12, `QualificationExact`: qualification must inherit the owner |
| `retention-per-edge` | exit 12, `QualificationExact`: raw traversal at discriminator call sites is not transitive |

The controls configuration matches the public tests. Raw-path and unselected
selection outcomes can be correct while the base qualification of C is still
wrong, so qualification is asserted separately.

Run each configuration explicitly, for example:

```sh
java -XX:+UseParallelGC -cp "$TLA2TOOLS_JAR" tlc2.TLC -workers 1 \
  -config retention-checked.cfg -metadir "$TMPDIR/retention-closure" RetentionClosure.tla
```

`componentPruner.retainComponent` remains the single retention owner. On a
component's first retention it walks the normalized body and, when the pruner
has a bundle, the raw component prefix. One per-pruner `bodies` key guards
both. `newDocumentPruner` supplies the bundle to every production walk:
selection, `schemaDeclarationQualification` and `documentSchemaOccurrences`.
Operation, path-parameter and root-security source walks share the pruner's
`sources` set. Examples and security schemes retain through the same owner.
Component prefixes are answered from `bundledDescription.componentReferences`.
`bundleResolution` builds that index once, as its last step after every
reference is recorded, and nothing writes it afterwards. Every pruner,
qualification candidate and projection copy of an analysis therefore shares
one immutable index without synchronization. Weak, mapping and default call
sites are unchanged. Existing owners decide the
outcomes:

- a disqualified optional child is omitted, and valid siblings survive;
- a strong target retains its real blocker, which fails strict import and skips
  the operation in partial import;
- otherwise the existing ParentHint diagnostic or omission applies.

`TestDiscriminatorReferenceRetention` asserts base qualification, the strict
selected outcome and the partial outcome as independent subtests. It covers
the weak, explicit and default links with raw-only placement, a valid sibling,
the normalized-placement, raw-path and unselected controls, the exact blocker
Path and Origin, retained names, and Render. `TestComponentPrunerRetainedSource`
is the synthetic seam: raw retention is transitive across a cycle, terminates,
and repeating a retention leaves the closure and visited sets unchanged. A
deliberate write after finalization is ignored, which shows walks read the
finalized index rather than rescanning the references.
`TestBundleComponentReferencesShared` checks a resolver-built bundle's index
against a prefix scan, and checks that projection walks share it, by identity,
without writing it.

The model abstracts component kinds, source-pointer identity, diagnostics and
Render. It does not prove the Go walk or the analyzer's edge inventory. The
direct tests check those.
