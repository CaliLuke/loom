# Projected acquisition protocol

Opt-in finite design models for the shared acquisition and held-completion owner. Models inform the design; implementation and generated contracts require separate tests. They are never CI, hook, build or release gates.

## Fixture and independently supplied facts

The cyclic Pet→Cat→Pet fixture has full/removed occurrence contexts, five root orders including completed-root reuse, an optional OtherPet claimant, structurally equal unnamed aliases, and preferred/canonical requests eligible for current same-hint reuse. `FP` abstracts the existing structural acquisition fingerprint including complete occurrence wire policy. `Hint` is the registration selector; `Claim` and canonical request status are separate. Neither declaration ID nor canonical boolean universally partitions component identity. This model has one eligible registration per F/hint; the real ordered bucket, noRef/async/alias branches and conditional canonical claims need source tests.

`ContributorKey` is an independent declaration/context preparation key. Contributor routing maps to a component handle selected through the component request; direct roots bypass that routing memo. The finite fixture supplies equal root/backedge fingerprints. It does not prove that equality in the actual recursive design: implementation traces must measure F, hint, recorded claim, C and returned handle and distinguish direct reuse from cross-F canonical comparison.

`Equivalent` is an independent compatibility oracle supplied by the scenario. Conflict is derived from recorded distinct declarations sharing a nonempty claim for which this oracle is false. EqualClaims instead gives OtherPet the same child closure and true oracle. The model does not implement schema equality or authorize changing canonical product semantics. Structural alias identity and preferred/canonical role reuse have their own positive/negative controls.

`Authority` supplies already-selected binding tokens, never a Go algorithm deriving ownership from logical IDs. `annotationBindings` preserves these tokens. `annotations` counts actual annotation-memo acquisitions with increment-on-miss; `preparations` separately counts target preparation per contributor key. Counts are not callback/sample counts: authored callback multiplicity, ExampleSet position extraction and allocated random sampling contexts need the existing Go correspondence tests. AnnotationOrigins checks binding preservation only. It does not authorize canonical JSON suffixing, forced merges or conflict rejection caused solely by samples. Ordinary Body(namedType) canonical behavior is governed by the documented DSL and the separately reviewed canonical-binding companion.

## Candidate phases and safety

Each outer root is one held completion batch. Reentry reserves/selects stable handles, records every dependency edge and may return references while bodies are visiting. Visiting baseline state is external to `a.schemas` in the implementation. Local completion does not imply closure completion.

The held outer scope then performs:

1. Plan projected references from the complete captured logical graph.
2. Install all reserved output-reference slots, in either order.
3. Materialize projections/read body caches only after all reachable bodies and slots are complete.
4. Read the updated live comparison state, compare canonical claims, then release the scope.

`planned`, `installed`, `resolved`, and `snapshots` distinguish these boundaries. In Go, use a live body cell instead of an early shallow comparison snapshot; `snapshots` is specification history for whether comparison sees installed/materialized state. `cacheReads` abstracts construction/publication dependency/context cache population. Planning reads the sealed logical graph; it must not run today's rendered-body cache readers on absent mapping slots. The protocol requires no acquisition or example preparation during drain. Missing captured handles are errors, not an unmodeled restart.

Reachability is the transitive closure of recorded edges, bounded by record count. `NoIncompleteProjection` checks the entire reachable closure. `ProjectOnFinish` specifically fails when Cat is locally complete but Pet is visiting. `CachesAfterSlots` also requires reachable slot installation. `materialFields` stores independently materialized fields; `OwnClosure` compares them to expected root-context fields, and WrongFields is a non-tautological negative.

On error/conflict, successful publication is forbidden. Rollback removes current-batch registration entries while retaining earlier completed-batch entries internally. FailureConflict creates both body-error and conflict facts; OverwriteCause proves first-error protection is exercised. This does not prove each concrete Go index cleanup or permit reuse of a failed Analyzer.

## Checks and controls

The final review-input runs use TLC 2.19 (`tla2tools-1.7.4.jar`, SHA-256 `936a262061c914694dfd669a543be24573c45d5aa0ff20a8b96b23d01e050e88`) and Amazon Corretto 25.0.5, one worker and 256 MiB. Owned explores 188 distinct states, queue empty, 21 invariants and weak-fair termination. Ten further positive configurations pass, including equal/conflicting claims, combined failure, aliases, role reuse, different hints, distinct annotation bindings, completed reuse and the late-contributor-only partial repair.

Twenty-one negatives fail on their intended invariant: duplicate reservation, repeated preparation, public-name-filter duplicate allocation, broad canonical reuse, logical-ID overpartition, canonical-role overpartition, context collapse, repeated annotation acquisition, wrong binding, local-finish/early projection, early cache population, comparison-before-drain, stale comparison snapshot, dropped backedge, wrong child fields, ignored conflict, retained failed indices, overwritten first cause, and failed/early publication. The exact manifest/receipts live in the review packet; raw checker output is not a proposed repository addition.

Current combines two actual mechanisms: late contributor memo insertion and strict public-ref spelling rejecting a suffixed canonical registration. CurrentPreparation exposes repeated preparation separately from annotation memoization. PublicFilter uses a nonrecursive completed-root case to isolate duplicate allocation. LateContributor alone passes this finite fixture because correct component registration reuse terminates it; that positive is not evidence that a contributor preparation memo is generally unnecessary. Fixing exact-bucket suffix reuse is an intended allocation difference and requires rebinding the actual declared Body/callback families, not merely fixing recursive termination.

Run explicitly from this directory after installing the pinned tools:

```sh
java -Xmx256m -XX:+UseParallelGC -cp /path/to/tla2tools-1.7.4.jar tlc2.TLC -workers 1 -config Owned.cfg ProjectedAcquisition.tla
java -Xmx256m -XX:+UseParallelGC -cp /path/to/tla2tools-1.7.4.jar tlc2.TLC -workers 1 -config ProjectOnFinish.cfg ProjectedAcquisition.tla
```

Owned and the positive configs must exit 0. Negative configs must report the invariant named in that config, normally TLC exit 12. A parse/tool error or an exhausted supervisor is not a semantic counterexample. Keep all model execution opt-in, outside hooks/CI/build/test defaults.

## Bounds and remaining correspondence

This is a closed, finite graph, not an arbitrary-graph theorem. It assumes deterministic complete context fingerprints, selected binding facts, the supplied compatibility oracle and weak fairness. Slot planning/materialization do not acquire new baselines. The model explores installation/materialization orders within this fixed batch, not arbitrary dynamic dependency growth, concrete hash collisions, Go pointer copies, reflect.DeepEqual, sampler values, final name allocation, all caches or all rollback indices. No full public-name proof follows from its abstract published handle set. The source API proposal and focused implementation tests remain required, including multiple mappings, prepared target provenance, actual canonical controls and generated OpenAPI validation.

## Contributor-slot and reverse-role refinement

See [ACQUISITION-SLOTS.md](ACQUISITION-SLOTS.md) for the explicit ReserveC/BindC/final-result/install protocol, nullable mapping-record ownership and post-scope projection owner. The old atomic Acquire is an abstraction over those intermediate steps, not proof of them. Its single eligible registration per F/hint fixture does not cover preferred-only-suffixed-before-canonical order; the refinement adds that order and makes selected claim association explicit. Real uniqueness is per selected registration, not a universal prohibition on multiple registrations in an F/hint bucket. The core TLA/config bytes and their 32 accepted runs remain unchanged; the concrete owners are recorded below.

## Implementation correspondence

`component_acquisition.go` retains visiting reservations outside `Analyzer.schemas`. Ordered structural fingerprint buckets still select components; contributor keys route to selected handles without partitioning structural sharing by declaration ID. `body_projection.go` reserves a contributor before preparation/traversal and fills a stable final-result cell after reference-local examples and overlays are available. A one-shot binding is consumed at that contributor root's first named selection, including projected-result and overlay-base routing, before nested properties. Canonical comparison binds the selected owner before candidate traversal.

`schemaAnalysisScope` holds the outer depth during `completeSchemaScope`. `projection_completion.go` plans from the sealed logical graph, reserves references, installs captured discriminator mapping records, materializes stable projection result cells, then permits live canonical comparison. Nullable wrapping retains the captured discriminator pointer; composition edges retain contributor result cells. `publicationBaseline` reads a detached logical mapping view so dependency/identity caches never observe missing output slots. This immutable logical-cache read belongs to planning; physical copies/publication follow installation. Post-scope `projectBaseline` uses the same owner. Completion neither prepares examples nor acquires new baseline handles.

`acquisition_scope.go` removes provisional indexes on failure while preserving prior completed entries and the first error. It does not undo user callback side effects or promise reuse of an errored Analyzer. Comparison reads live completed body pointers rather than pre-drain shallow snapshots. Actual canonical Body selection, fresh allocation contexts and suffixed prepared bindings remain distinct; see [canonical claims](../canonical_body_binding/README.md) and [prepared claim binding](CLAIM-BINDING.md).

Correspondence tests: `TestAcquisitionProtocol*` covers sharing, preferred/canonical orders and recursive local annotations; `TestAcquisitionCompletionBindings` records actual F/C/claim/handle relationships, nested-property exclusion, root/drain reuse and no new completion owners; `TestAcquisitionCompletionMappingBatch` covers two mappings and different byte constraints in both orders; rollback and comparison timing have direct tests; `TestNullableParentExplicitMapping` covers resolved child/backedge in both orders,3.1 and supported3.2, plus missing-fallback rejection. These bounded controls do not prove arbitrary graphs, schema equality, serializer correctness or sampler values.


## Nested example-position memo

`AnnotationPositions.tla` refines the acquired annotation binding at nested
component positions. A declaration and its occurrence overlay can have the same
context and structural fingerprint but different prepared example-plan nodes.
Three captured positions contain `declaration`, `local`, `local`; each is read
twice in every bounded traversal order. `AnnotationPositionCurrent.cfg` models
the omitted node and fails `OwnValue` after declaration then occurrence (6 states,
exit12). `AnnotationPositionOwned.cfg` includes the opaque node and passes all
five invariants across271 states. `AnnotationPositionReach.cfg` deliberately
fails `NoComplete` at six reads (182 states), proving the completed deduplication
check is reached. These runs use the pinned TLC/Java above, one worker,256MiB.

Run the same explicit command above with module `AnnotationPositions.tla` and
the selected `AnnotationPosition*.cfg`. Current and Reach are expected negative
controls, not CI gates. Prepared values are inputs; this companion abstracts
codecs, sampling, actual graph extraction, annotation composition and naming.
Memo identity is not semantic identity: completed outcomes with equal values
still share a semantic class, and no pointer enters a public name.

Correspondence: `componentAnnotationPosition.examplePlan` distinguishes the
nested acquired binding in `applySchemaExample`; the outer acquired component
scope and final complete-graph deduplication remain unchanged.
`TestDerivedMessagePreservesAuthoredOccurrenceExample` preserves the authored
local value in both event responses and the derived message. The generated
producer first/second rounds cover Service's native wrapping, which exposes
this cache even when raw inline analysis does not. Nullable occurrence examples
remain on the whole occurrence during `effectiveFacts` comparison, separate
from non-null component projection. The model does not prove this extraction;
`TestEventCorrespondenceComparesFinalizedTargetGraph` and the producer regression
check it directly.
