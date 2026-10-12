# Recursive imported schema references readiness

Status: implementation and bounded validation complete; awaiting independent
final-diff review. Nothing committed or pushed from this worktree.
Owner #792. Implementation parent `214bab44d0b17eb0d020f6fa9d4292b4b42e2e1a`.
Historical readiness evidence below uses parent `56bc7eae`.
Worktree `/Users/luca/.codex/worktrees/recursive-import-references/loom`.

## Implementation checklist

Readiness independently approved; implementation starts at pushed `214bab44`.
The twelve readiness files retained their exact reviewed bytes during the
fast-forward. Root owns delivery and a different agent owns final review.

- [x] Safely advance the isolated worktree and verify readiness hashes.
- [x] Add direct failing reference-planning and mandatory-cycle regressions.
- [x] Build one immutable occurrence plan after declaration/error indexing;
  consume it in component and child rendering without changing acyclic output.
- [x] Promote constructor probes into identity/metadata and diagnostics tests.
- [x] Run bounded import/generation/spec/compilation and deterministic-output
  comparisons, then affected importer and lint gates.
- [x] Freeze the exact diff, evidence and guidance handoff for independent review.

## Contract and ownership

Imported recursive component schemas must compile, evaluate and regenerate the
same named reference graph. A Go initializer dependency is not the schema graph.
`internal/openapiimport` owns lowering the latter into valid DSL declarations;
no generated service patch or consumer workaround is acceptable.

Observed: `renderer.schemaReferenceExpression` always emits `Imported` plus
`NamedSchema.GoName`. `namedSchema` places references inside package-variable
initializers, including delayed DSL closures, which Go still counts as initializer
dependencies. Self objects and mutually recursive fields consequently fail compilation.

Inventory: `render.go` namedSchema, `render_schema.go` schemaExpression,
arraySchemaExpression, objectSchemaExpression and schemaBlock; `render_index.go`
component/error index; `plan.go` planDocument and schemaAdmission. The planner
currently has no common reference initialization policy. Nested array/map child
renderers copy only document and schema index, so any selected policy must reach
these instances rather than being inferred independently from path strings.

The existing DSL can resolve names in delayed Attribute/ArrayOf/MapOf calls.
However `Type` bases and `Extend` require concrete DataType values, and top-level
array constructors execute immediately. OneOf is different: its string variants
can create symbolic forward references (`dsl/attribute_oneof.go`), which the
evaluator later resolves. Classification must follow these actual APIs. A blanket conversion of references
to strings is not valid. Establish eager versus deferred dependencies before
choosing the lowering strategy. Pure alias or eager constructor cycles need
explicit evidence about DSL representability; do not silently drop them or assume
all schema cycles are inheritance cycles. Preserve valid mixed inheritance/field
recursion delivered by #755.

## Candidate designs to compare before implementation

1. Classify references once in a render plan, preserving concrete constructor
   dependencies while lowering deferred references that participate in initializer
   cycles through existing named lookup. Retain acyclic output where possible.
2. A two-phase declaration mechanism that registers names before materializing
   shapes could express more graphs, but may require a public DSL/evaluator change.
   Do not add one without showing a supported graph that requires it and checking
   existing evaluation ownership.

Use a bounded reference-graph model with current Go dependency failure controls,
corrected acyclicity and name-resolution invariants. Include self, mutual, diamond,
array/map indirection, mixed Extend/field edges and pure eager cycles. Model proofs
must distinguish compile-time graph properties from actual evaluator behavior.
The initial exploration is recorded below; the concrete proposal and bounded model results follow. No production implementation is authorized yet.

## Bounded acceptance

- Table-driven red compile/evaluation cases: self object array, mutual array/map,
  mixed inherited fields; a normal acyclic control. Retain diagnostics and exact
  parent/source/toolchain evidence.
- Add graph-specific direct tests before selecting/rendering references; prove
  component identity and required/nullability/validation metadata survive.
- Reuse importer round-trip harnesses and meaningful HTTP OpenAPI specimens.
  Import JSON and YAML, render/evaluate design, generate/build/vet an HTTP app,
  and compare re-exported named reference graphs using libopenapi and pinned
  Vacuum rules. Include a recursive failure response and aliased component.
- Compare a bounded parent/current specimen selection byte for byte, list every
  intended declaration/reference/schema difference, repeat candidate generation
  in isolated processes. Do not run the whole exported-design corpus.
- Run affected importer tests and scoped lint; refresh owning importer docs and
  framework guidance. Independent readiness approval precedes production, and
  exact final diff review precedes an atomic #792 commit/push/closure.

#706/#719/#793 parameter/form wire serialization is a separate owner. Initial
inspection confirms export/import disagreement, but their reports' suggested
`deepObject` behavior does not cover arbitrary nested collections: OpenAPI 3.2
only defines scalar properties and treats explode as ineffective for deepObject.
Do not copy those proposed fixes without a shape/version-aware contract.
Source: https://spec.openapis.org/oas/v3.2.0.html#style-values

## Initial evidence checkpoint

The four-case importer regression now reproduces three compile failures (self
array, mutual map/array, mixed inheritance/field); the acyclic case passes.
The mixed case explicitly retains the existing allOf-flattening warning.
A narrow name-lookup probe makes those same specimens compile and evaluate,
without changing production rendering. The graph model reproduces the current
eligible self-cycle and checks the candidate over all19,683 three-node classified
graphs. Commands, limits and results are beside
`internal/openapiimport/tla/schema_references/README.md`.

The existing requireRenderedDesignEvaluates helper now uses verified pushed pin
`v1.10.0-alpha.9.0.20261012181439-651372ce6ff0` with its existing explicit local
replace. This is the sole existing test-helper edit; production remains untouched.
This initial checkpoint preceded the constructor and error/union probes recorded
below; it did not authorize rejecting any existing supported contract.

## Reference-context acceptance detail

The shared plan must distinguish individual occurrences before collapsing its
reachability graph. An eligible field reference and a concrete Extend reference
can point to the same component: the concrete occurrence still contributes a Go
initializer edge. A graph pair is removable only when no mandatory occurrence
remains. Test both declaration orders, repeated references and diamond sharing.
Keep resolved component identity separate from allocated Go names and use the
same admitted plan in Analyze and Render. Error clones and nested child renderers
must consume that plan instead of recomputing eligibility from printed strings.

Current public import documentation (`docs/openapi-import-coverage.md`) promises
an explicit strict subset and diagnostics for unrenderable constructs. It does
not define arbitrary recursive eager aliases. Establish actual DSL support for
those cases before selecting either a supported representation or a precise
import diagnostic. A mere Go compile failure after successful import is never
acceptable. This is a representation boundary investigation, not permission to
reject the ordinary recursive object graphs covered by #792.

The existing `TestOneOfTypeConstructorSupportsRecursiveNamedVariants` and
forward-declaration controls in `dsl/oneof_default_and_forward_test.go` are
relevant preservation evidence. Top-level OneOf symbolic variants may make a
cycle representable even without an object-field lookup edge. Add an imported
union/array control before concluding that all remaining immediate constructor
cycles need rejection. No blanket eager/deferred heuristic is authorized.

## Concrete reference plan proposed for approval

The correction stays inside `internal/openapiimport`; no DSL constructor or
public registration mechanism changes. Add a private immutable reference plan to
`documentPlan`, built from normalized components **after inline promotion** and
from the final error-clone declarations. `Analyze` and `Render` both obtain it
through `planDocument`. Move clone allocation/indexing into that planning result
so rendering does not independently rebuild a second declaration graph.

The plan records each occurrence's declaration identity, resolved target identity,
logical schema path, constructor use, and execution phase. Names in the graph are
component identities, not collision-resolved Go variable identifiers. Lookup
literals use the exact registered DSL type name; concrete references use the
existing allocated Go name. JSON-pointer decoding belongs to the existing
reference resolver; no reference identity should be recovered by inspecting
printed Go text.

| Occurrence | Name lookup eligibility | Reason |
| --- | --- | --- |
| Attribute type inside a component's Type DSL closure | Eligible | Attribute resolves an already registered user type during DSL execution. |
| ArrayOf element / MapOf value within that delayed closure, including nested constructors and validation callbacks | Eligible | The surrounding expression executes after component registration. |
| Top-level ArrayOf element / MapOf value in a Type initializer | Mandatory concrete dependency | Their string form resolves immediately; a forward name is an error. |
| Type base / named alias | Mandatory concrete dependency | Type accepts a DataType or DSL function, not a type-name string. |
| Extend base, even inside a closure | Mandatory concrete dependency | Extend has a DataType parameter; a closure alone does not make a name valid. |
| Direct OneOf variant | Eligible symbolic dependency in either phase | Its existing string form retains a forward reference for evaluator resolution. This does not widen importer union-shape admission. |
| Error clone Extend of its source component | Mandatory concrete dependency | Error clones reuse the source object with an independent name; they are additional declaration vertices. |
| Service/API operation references | Keep concrete | They do not create a return dependency from a component initializer. |

The component-root phase depends on the actual emitted constructor: a plain
object emits Type(name, func()), while an object rendered as MapOf remains an
immediate Type-base expression. Crossing a field declaration starts delayed
execution; arrays/maps preserve their surrounding phase. A OneOf variant's
symbolic capability is specific to that argument, not inherited by unrelated
constructor arguments. Normalization may promote an apparent inline map inside
an object array to a separate top-level map declaration; that declaration's
children are then immediate. This distinction was demonstrated by a probe.

Compute strongly connected components over the complete initializer dependency
graph. Lower eligible occurrences whose endpoints are in a cyclic component to
existing named lookup; preserve every mandatory occurrence and every acyclic
reference. Rebuild initializer edges from remaining **occurrences**, then require
that graph to be acyclic. An eligible occurrence and an Extend occurrence may
share a source/target pair; lowering the eligible one must not erase the Extend
edge. Repeated references, diamonds, both declaration orders and self edges are
direct planner-test obligations. A deterministic cycle witness identifies each
remaining constructor/path, rather than reporting an undifferentiated recursive
schema error.

Rendering consumes the selected per-occurrence lookup mode; it does not discover
cycles or classify eligibility again. Existing logical schema paths can be keys,
but their text is never used to infer execution phase. Both temporary child
renderers in `arraySchemaExpression` / `objectSchemaExpression` must receive the
same plan. Component declarations, aliases, error clones and response adapters
share its resolved names. Keep schema overlays, Nullable, validations, Required,
canonical names and error field overrides on their existing owners. Changing a
reference expression must not reconstruct or flatten its target schema.

## Remaining-cycle admission recommendation

Ordinary object recursion, object/array and object/map cycles, aliases anchored
in those graphs, and mixed Extend/field recursion remain supported. They have
eligible occurrences that break only Go initialization cycles while retaining
all semantic references. Recursive errors also remain supported.

Pure self/mutual alias documents already fail in libopenapi with an infinite
circular-reference error. Preserve that diagnostic. Pure array/map cycles are
currently admitted and then emitted as illegal Go initializers. Existing public
DSL constructors cannot express their forward/self lookup: direct probes fail
with `invalid ArrayOf argument` / `invalid MapOf value argument` before a target
is registered. A blanket string rewrite does not repair them. Recommend a
specific blocking `schema-reference-initialization` diagnostic in Analyze and
Render for a remaining mandatory cycle, with component paths and constructor
kinds. This converts a false successful import into the documented strict-subset
failure, rather than claiming arbitrary JSON Schema recursion is invalid.

The considered alternative is broader two-phase DataType registration or general
symbolic Type/ArrayOf/MapOf support. That is a new DSL/evaluator contract, not
needed to express the ordinary recursive object graphs behind #792. Moving all
rendered declarations into Go init functions alone does not solve a pure eager
constructor cycle: the first constructor still lacks its target. This proposal
does not adopt expression-object mutation or placeholder types as an importer
workaround. Independent readiness review must accept the bounded diagnostic
before production proceeds; no public DSL change is authorized here.

A union → array → same union specimen is already rejected by the existing
`schema-oneof-branch` rule because untagged branches cannot recursively contain
another union. Preserve that rejection. Separately, a top-level imported OneOf
with named array/object branches accepts symbolic forward variants, and its
recursive object branch evaluates successfully; the plan must preserve this
existing constructor capability rather than labeling all top-level references
mandatory.

## Additional readiness evidence

`constructor_probe_test.go.txt` beside the model contains reusable, non-production
probes. Copy it temporarily to `internal/openapiimport/constructor_readiness_test.go`
and run the three `*Readiness` tests listed in its README; remove the temporary
copy afterward. On parent `56bc7eae` and Go 1.27.2:

- Top-level ArrayOf/MapOf forward and self names fail with the expected immediate
  lookup diagnostic; Type string bases fail argument validation. Top-level OneOf
  forward variants evaluate and resolve to the registered target object identity.
- Seven imported specimens compile/evaluate after exact, explicitly chosen
  reference substitutions: top-level array/object, map/object, alias/object,
  nested arrays with annotations, inline-map promotion, recursive shared-error
  clones, and top-level union with named array/object variants. The probe is not
  a generic text-rewriting algorithm and is not the production implementation.
- The first exploratory nested specimen exposed synthetic component promotion;
  lowering its immediate nested map reference was inappropriate. The corrected
  probe lowers the enclosing object's deferred reference to the promoted
  component and retains the promoted component's concrete reference back.
- Raw alias cycles fail at the parser boundary, array/map self-cycles currently
  return successful unbuildable renderings, and union/array cycles receive the
  existing union-branch diagnostic. These outcomes are asserted in the probes.

`ReferenceOccurrences.tla` refines the original graph proof to retain parallel
mandatory, delayed-lookup and symbolic-OneOf occurrences for two components.
The incorrect pair-deletion candidate loses a mandatory occurrence at 4,110
states; the corrected occurrence-level candidate checks all 4,096 occurrence
graphs and passes at 8,192 states. It demonstrates why occurrence eligibility
must be preserved until initializer edges are rebuilt. Constructor classification,
name identity, arbitrary-size SCC implementation and metadata retention remain
Go-test obligations, not claims established by that finite model.

## Production sequence after independent readiness approval

1. Add direct plan/context tests (including mixed parallel edges and reversed
   declaration order), plus explicit remaining-cycle diagnostics to the red
   importer regressions. Assert Analyze/Render share those diagnostics.
2. Implement the private reference plan and integrate declaration/error-clone
   indexing into the admitted document plan. Keep renderer output unchanged for
   acyclic inputs. Feed the plan through nested renderer construction.
3. Promote the ordinary recursive probes to production regression tests that
   assert evaluated identity and required/nullable/validation metadata, not only
   successful DSL execution. Cover JSON and YAML input and alias/error clones.
4. Run bounded recursive round trips through generated HTTP build/vet and
   libopenapi/Vacuum, plus acyclic unchanged-output and independent-process
   determinism controls. Enumerate reference-only changes and any re-exported
   schema differences. Reuse existing specimen/harness machinery.
5. Run affected importer checks and scoped lint; update public import coverage
   and provide shared-skill/model-index deltas to root. Freeze final source and
   evidence for independent exact-diff review, then root delivers #792 atomically.

No production source changed during the readiness investigation. The earlier
red regression cases now pass against the implementation described below.

## Implementation correspondence

`planDocument` completes component and error-clone allocation before calling
`planSchemaReferences`. The latter records each reference by declaration owner
and escaped logical path, resolves its target through the existing local
JSON-pointer resolver, and classifies the actual constructor context. Tarjan
SCC discovery selects eligible occurrences within cycles; rebuilding the graph
without those occurrences detects remaining mandatory cycles. Diagnostics name
each cyclic mandatory occurrence's path and constructor. Parallel mandatory and
eligible occurrences are never collapsed before selection.

`schemaReferenceExpression` consumes the selected exact DSL name; it does not
recompute eligibility. `childRenderer` carries the same frozen plan and owner
through annotated array/map expressions. Error clone `Extend` edges stay concrete.
No target schema or annotation is reconstructed by the reference plan.

Direct tests cover parallel Extend/field edges, both declaration orders, eager
and symbolic constructors, shared acyclic targets, distinct Go/DSL names, and
blocking Analyze/Render diagnostics. Four original compile specimens and fourteen
JSON/YAML constructor-context specimens assert evaluated graph relationships,
requiredness, nullability, validation and canonical-name metadata. Expression
composition may own detached named snapshots in promoted containers or inherited
error fields; those tests assert retained named identity there, not global pointer
sharing. Ordinary declaration and direct back-reference identity is checked by
pointer where the expression contract retains the canonical declaration.

## Final validation evidence

Go 1.27.2, implementation parent `214bab44`. All temporary modules require the
verified pushed `v1.10.0-alpha.9.0.20261012181439-651372ce6ff0` and explicitly replace
it with the source under test. No dependency or source-mode policy changed.

- The direct planner regressions were added before implementation. Parallel
  Extend/field recursion lacked selected names; three mandatory-cycle cases
  lacked the promised diagnostic. These failed against the old owner, alongside
  the original three generated-design compile failures retained from readiness.
- Final direct planner/admission run passes (0.599s), including all 512 small
  graph comparisons. Self/mutual pure aliases retain the parser's infinite-cycle
  error; a union-through-array retains `schema-oneof-branch` rejection.
- Full importer and CLI packages pass: 334.803s and 63.952s. After extracting the
  admitted render-plan phase to satisfy maintainability lint, direct admission
  checks and all affected generation/evaluation cases were rerun. No unrelated
  generator or runtime source changed.
- Final eighteen evaluation cases and nine generated graph/build/vet/complete
  artifact-repeat cases pass (102.638s). Exact exported component assertions cover
  the named graph and occurrence metadata, not only successful parsing.
- Independent CLI comparison captures twelve inputs: the four original cases,
  seven constructor-context cases and existing `testdata/supported.yaml`. The
  acyclic and supported inputs retain identical imported DSL and all 32 generated
  artifacts at the parent and candidate. The other ten imported DSL differences
  contain only selected `ImportedType` expressions becoming exact quoted DSL
  names. Concrete Type/ArrayOf/MapOf/Extend edges and all metadata stay unchanged.
- Twelve candidate modules compile/test and vet (192 artifacts). Fresh final-source
  generations match all 192 prior bytes. The first comparison invocation used
  different executable basenames, which appear in generated command comments;
  subsequent runs use identical `loom` basenames and the final parent comparisons
  are exact without output normalization. The nine durable generation tests also
  compare all artifacts across separate processes using the same invocation.
- Existing export normalization is explicit: the alias specimen exports the
  underlying `Branch` identity; accepted allOf/Extend flattens `Base` into `Node`;
  recursive error responses export source `Node`. Feeding the candidate DSL for
  alias, mixed inheritance, promoted container and error-clone specimens through
  the parent generator yields the same 64 artifact bytes. These effects are not
  new importer reference policy.
- All 24 captured candidate JSON/YAML specs pass Vacuum v0.33.0 using the existing
  pinned libopenapi replacement and repository contract rules. The durable nine
  specimen checks also parse both formats with libopenapi.
- Scoped pinned golangci-lint reports zero issues. File-size and gofmt gates pass
  (existing large-file warnings only); `git diff --check` passes.

Temporary evidence: `/tmp/loom-792-contract-final.log`,
`/tmp/loom-792-direct-final.log`, `/tmp/loom-792-packages.log`,
`/tmp/loom-792-lint-final.log`, and `/tmp/loom-792-comparison/` (render differences,
generated hashes, final repeat, parent mechanism controls and Vacuum results).
Root retains these until independent final review, then removes them along with
the parent archive and temporary Vacuum build. Durable model sources and the
recorded evidence above remain. Shared framework/consumer/model-index guidance
is handed to root for the separately reviewed integration; this worktree does
not edit shared skills or the master trackers.
