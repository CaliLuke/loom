# Imported schema reference dependencies

Design exploration and implementation correspondence for #792. The finite models
do not prove the production implementation.
The model separates semantic reference edges from Go initializer edges. A deferred
edge represents a reference whose DSL location can safely look up an already
registered component by name during later DSL evaluation. Classification itself
is an implementation obligation; not every closure edge is eligible (`Extend`
needs a concrete DataType).

The legacy candidate emits every edge as a Go variable dependency. The corrected
candidate lowers eligible edges within strongly connected regions to name lookups,
then admits exactly graphs with an acyclic remaining mandatory dependency graph.
The three-node exhaustive domain contains all absent, mandatory and eligible
edge combinations, including self edges. Parallel reference occurrences are
collapsed: a pair is eligible only when every occurrence is eligible; one
mandatory occurrence keeps that pair mandatory. The production planner must
retain occurrence identity for rendering and test this dominance rule. It checks emitted acyclicity, admission
of every graph representable under this classification, and that only eligible
edges are removed from the initializer graph. Semantic identities stay unchanged.
This last fact is an assumption of name lookup and needs generated evaluation tests.

This does not establish that rejected eager cycles lack a valid alternative DSL
representation, prove arbitrary-size graphs, implement SCC discovery, or cover
metadata/nullability/error cloning. Do not infer a new public rejection policy
from this model. The bounded diagnostic contract was separately accepted in
`roadmap/recursive-import-readiness.md` after readiness review.

## Run

Use the shared pinned TLC JAR at
`/Users/luca/.cache/loom-tools/tla2tools-1.7.4.jar`, SHA-256
`936a262061c914694dfd669a543be24573c45d5aa0ff20a8b96b23d01e050e88`
(TLC 2.19). Run sequentially with Java 25; keep state and trace output outside
this directory:

```sh
java -XX:+UseParallelGC -Xmx1g -cp /Users/luca/.cache/loom-tools/tla2tools-1.7.4.jar tlc2.TLC -workers 1 -metadir /tmp/loom-import-792-model/legacy -config legacy.cfg SchemaReferences.tla
java -XX:+UseParallelGC -Xmx1g -cp /Users/luca/.cache/loom-tools/tla2tools-1.7.4.jar tlc2.TLC -workers 1 -metadir /tmp/loom-import-792-model/corrected -config corrected.cfg SchemaReferences.tla
```

Expected: legacy fails GeneratedInitializersAcyclic; corrected passes all three
invariants. Record actual results before relying on them. Direct source witness:
`internal/openapiimport/render_recursive_test.go`.

## Recorded readiness results

On Go 1.27.2 / Java 25, parent `56bc7eae`:

- Current rendered self-array, mutual map/array, and mixed inheritance/field
  schemas all fail with Go initialization cycles. The acyclic control passes.
  Mixed inheritance retains the existing, explicitly accepted
  `schema-allof-flattened` warning; it is not mistaken for this compile defect.
- `named_lookup_probe_test.go.txt` makes only three exact substitutions in these
  rendered specimens to test existing DSL name lookup. All four evaluate
  successfully (11.435s package result). This is a candidate-mechanism control,
  not an implementation or a general reference rewriting algorithm. To rerun,
  copy it temporarily to `internal/openapiimport/recursive_lookup_readiness_test.go`
  and run `go test ./internal/openapiimport -run
  '^TestRecursiveNamedLookupReadinessControl$' -count=1 -v -timeout=2m` from root.
  Remove that temporary test copy after the run.
- Legacy model limits eligibility to all references (512 graphs), reproducing
  the observed eligible self-reference counterexample: exit12 at
  GeneratedInitializersAcyclic, 514 generated/distinct states, depth2.
- Corrected model enumerates all 19,683 three-node classified graphs; all three
  invariants pass, 39,366 generated/distinct states, depth2.

The recorded candidate classifications were subsequently checked with the actual
constructor probes below and accepted in readiness review. The historical `.txt`
controls must be run against the recorded parent: corrected production output
already contains the selected names and no longer has the expected legacy reds.

## Constructor and occurrence refinement

`constructor_probe_test.go.txt` records actual DSL/API controls and seven imported
compile/evaluation mechanisms. Temporarily copy it to
`internal/openapiimport/constructor_readiness_test.go`, then run from repo root:

```sh
go test ./internal/openapiimport -run 'TestReferenceConstructorReadiness|TestImportedReferenceContextReadiness|TestEagerCycleAdmissionReadiness' -count=1 -v -timeout=2m
```

Remove the temporary runnable copy afterward. Tests use the existing importer
helper, its verified pushed module version and explicit local replace. The
probes establish the eager ArrayOf/MapOf and Type-base restrictions, symbolic
OneOf forward support, supported recursive error/alias graphs, and normalized
inline-promotion ownership. They do not implement reference planning; their
few exact replacements are deliberately specimen-specific. See the readiness
plan for expected diagnostics and the proposed remaining-cycle treatment.

`ReferenceOccurrences.tla` represents each dependency as source, target and
constructor class (`mandatory`, delayed `lookup`, or forward `symbolic`). A pair
can contain multiple classes. Its bad candidate erases the entire pair if one
eligible occurrence is lowered, reproducing loss of an Extend/constructor edge.
The corrected candidate lowers only selected occurrences before rebuilding edges.
Run the same pinned TLC command with `ReferenceOccurrences.tla` and:

- `occurrences_wrong_pair.cfg`: expected `PreservesMandatoryOccurrences` failure,
  4,110 generated/distinct states, depth 2.
- `occurrences.cfg`: all three invariants pass, 8,192 generated/distinct states,
  depth 2, exhausting 4,096 two-node occurrence graphs.

Results obtained with the same pinned toolchain on 2026-10-12. Mandatory includes
Type aliases, Extend, and immediate ArrayOf/MapOf arguments. Delayed lookup
requires component registration to have finished; symbolic OneOf can resolve
later. These classifications come from actual DSL probes, not from this model.
The model does not prove the renderer carries context correctly, preserves
metadata, or that a wider future DSL could not represent currently rejected
mandatory cycles. Temporary states/logs are `/tmp/loom-import-792-occurrences`.

## Production correspondence

- `plan_references.go` retains owner/path occurrences, resolves component identity
  before deciding mode, and computes SCCs with Tarjan's algorithm. Delayed
  Attribute/ArrayOf/MapOf and symbolic OneOf occurrences in an SCC become lookups;
  Type/Extend and immediate array/map occurrences remain dependencies. A second
  SCC pass diagnoses every remaining cyclic mandatory occurrence.
- `planDocument` and `documentPlanner.renderPlan` admit schema shapes and allocate
  final operation/error declarations before freezing that plan. Rendering uses
  the same plan, including through child expression buffers; error-clone Extend
  occurrences have distinct owners and stay concrete.
- `TestReferenceComponentsMatchReachability` compares the SCC implementation to
  independent transitive closure for all 512 three-node edge graphs, including
  parallel occurrences. Other direct tests retain mixed mandatory/eligible
  edges, reversed declaration order, shared targets and exact DSL names.
- `render_recursive_test.go` and `render_reference_context_test.go` replace the
  three original compile failures and specimen-specific substitutions with actual
  importer output. Evaluated declarations retain requiredness, nullability,
  constraints, source names and error identities. Owned expression snapshots
  retain named identity without requiring global pointer sharing.
- Nine generated specimens check exact exported component graphs, compile/vet,
  libopenapi JSON/YAML parsing and all-artifact independent-process determinism.
  Existing alias export and accepted Extend flattening remain explicit controls.
  Public strict-subset diagnostics live in `docs/openapi-import-coverage.md`.

The algorithm tests establish finite correspondence; they do not turn the TLA+
model into a proof of arbitrary-size SCC implementation or all DSL constructors.
Keep the constructor classification and occurrence paths aligned with renderer
changes. Bounded generation and external lint results are recorded in the owning
readiness document. Temporary outputs are removed after final independent review.
