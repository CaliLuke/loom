# Alias pairing and schema annotation authority

These bounded models belong to `expr/value_plan_build.go`,
`expr/value_copy.go`, `expr/value_plan_schema.go` and OpenAPI IR component allocation. They describe
candidate rules with independently reviewed TLC results. The Go repair passes
focused correspondence tests; final generated comparisons and review remain open.

## Counterexamples that changed the design

- `TypeIdentityDSL`: a generated target wrapper incorrectly advances the semantic
  source alias cursor. A query then loses captured declaration identity and picks
  a transport name and a different example context.
- `DeclarationAuthorityDSL`: sharing context by declaration and baseline collapses
  distinct authored alias examples in ordinary generation. The corrected
  planned-first comparison also loses one sampled Base component and redirects
  AliasOne to the canonical Base, demonstrating that a plan-valid guard is
  insufficient. Both effective renderer overrides were confirmed; the earlier
  parent run had an unapplied `/tmp` versus `/private/tmp` overlay and is invalid.
- The complete 66-probe comparison also finds the same context collapse in
  `codegen-user-type-package-transports`: the moved object's independently sampled
  `Item` component disappears. No Go/protobuf artifacts change.
- Explicit `Body(Extend(...))` preserves its own declaration while mapping to a
  payload. Reusing one origin field for both roles mislabels the body; requiring
  its structural definition to occur in the payload alias chain rejects valid DSL.

Actual paired-origin traces show that an authored alias child matches the next
source alias's ancestry, while an inserted wrapper child matches the current
source node's ancestry. The previous builder advanced in both cases. Declaration
identity, semantic source pairing and annotation ownership are separate decisions.
Do not hide these failures by relaxing schema equivalence or approving examples
as an intended difference.

## Models and limits

`AliasPairing.tla` represents four distinct source origins and target chains of
up to six nodes. Repeated target origins insert wrappers; source aliases may be
skipped. A final structural target may retain an outer alias's origin while
replacing its Type, matching `UnwrapInlineHTTPBody`. Checked inputs also include
backwards and foreign origins.
The candidate selects the nearest matching source ancestry at or below the
current cursor, rejecting no-match. Zero advancement denotes a representation-only
edge; positive advancement denotes an authored transition. After that decision,
structural targets separately descend to the exact underlying semantic source.
An explicitly bound body may have an independent structural origin: its parent
binding plus kind and exact member/branch checks authorize that descent. Named
alias edges still require ancestry. `structuralChecks` abstracts those checks,
including rejection, rather than proving their Go implementation. The terminal-
ancestry control wrongly rejects a valid independently defined structural body.
Invariants compare source identity and edge role with the input ancestry and
target structural position. The legacy control uses valid inputs only and
reproduces `<<0, 0, 0>>`: the inserted wrapper incorrectly moves source 0 to 1.
The ancestry-only repair fails on a flattened `<<0, 0>>` target because it omits
structural descent. Neither old nor new valid inputs are excluded from checking.
The checked configuration includes all valid and invalid inputs at the bound.

This abstraction assumes distinct, correctly captured source origins and an
acyclic alias chain. It does not establish Go pointer ancestry, graph capture,
cycle termination, repeated source-origin handling or resolved member/branch
identity. Those require direct extraction and projection tests, including visited
node guards, recursion, selections, enums, numeric constraints and codecs.

`SchemaDeclaration.tla` independently varies declaration and semantic-source
identities, baseline labels, source cursor, incoming edge role, allocation
contexts and byte projection. The allocator's actual decision is independent:
a fresh component owns its allocated context; actual reuse owns the selected
existing binding. Captured target identity survives renaming, and projection
copies that exact acquired owner. Matching declaration, baseline or edge role
cannot choose annotation ownership.

`ResultBodyUserRequiredDSL` exposed the previous model's false premise. Its
baseline allocator selected `Body_99e66af2c29e5c06`, but the representation-edge
heuristic replaced the context with `Body` before sampling. The old
`ExistingContextReused` obligation required this inference instead of checking
actual allocation. It is replaced by `AcquiredOwnerPreserved`. `checked` now
checks allocation ownership; `edge-checked`, plan-only and cached-role controls
reject inferred reuse. `target-only` is a positive allocation control, no longer
a false negative. Byte-canonical and fingerprint-cached first acquire the correct
owner, then fail only projection preservation. Identity/mapping controls remain
independent failures.

The Go naming function is reached only for a newly allocated component; actual
memo reuse returns earlier. Explicit canonical naming remains a separate contract.
`selected_body_authority_test.go` checks parent names, references and both sample
levels. The parent TypeIdentity IR already contains an independently sampled
`Other` suffix component; its retained async cut references canonical `Other`,
and public pruning removes the unused ordinary component. Tests preserve this
actual allocation rather than forbidding every raw suffix. External public
pipeline comparisons for selected body, TypeIdentity and DeclarationAuthority
are byte-identical to the parent after removing the inference.

The model assumes correctly captured declaration identities, the independent
allocator outcome and its selected binding. It does not prove Go extraction,
actual sampling or recursive graph traversal. Declaration/source provenance and
pairing remain separate from allocation. Byte variants must preserve the exact
baseline binding and complete annotation path; equal fingerprints cannot recover
an independently allocated `noRef` position. Callback evidence distinguishes the
two restored ordinary parent samples from three preexisting async adapter calls;
no total parent callback-parity claim follows from this model.

## Complete annotation paths

`AnnotationPath.tla` covers a boundary absent from the scalar context model.
The repaired Bytes supplement preserved terminal Base samples but introduced
reachable alias examples. The parent IR retains pure alias references without
examples; its renderer collapses them. Projected analysis resampled those
previously unannotated references, preventing collapse and competing for the
canonical Base name. The terminal-only test missed both effects.

The model traverses baseline paths of one to three annotation positions, including
absence, and inserts zero to two schema layers at any position. Original owners
and annotation payloads must remain ordered; inserted layers cannot acquire
annotations. These are newly materialized schema layers, even when their DSL
aliases already existed. Existing authored annotations must not be erased.
`path-terminal-only` deliberately checks only the old terminal-owner obligation
and passes the defective reconstruction. `path-reconstruct` checks the full path
and reproduces the addition; `path-drop-authored` rejects removing intermediate
annotations. `path-resample-absent` reproduces sampling an existing unannotated
IR layer. `path-checked` preserves annotations and absence while allowing
unannotated representation references.

This is an abstract single-path transformation, not a proof of Go traversal,
complete schema equivalence, branching/recursive graph extraction or public-name
allocation. Correctly captured baseline paths and owner correspondence are
premises. Real projection may introduce references, but those references cannot
create sample authority. Go tests must check every reachable annotation layer,
including absence, and separately preserve public naming. Independent review
confirmed all five path checks and the production repair. Final ticket review
awaits the complete generated-output comparison. Annotation-payload preservation here governs #574's schema-only
byte correction. Later intentional value-consumer projections need their own
annotation-value contract.

## Baseline acquisition before projection

`BaselineAcquisition.tla` addresses a premise of `AnnotationPath`: the acquired
baseline must already contain the correct occurrence's annotations. The
`SSEFieldPresenceDSL` generated comparison found that `x-loom-async` lost nested
defaults `event=default-event`, `id=default-id`, and `retry=1500`, despite unchanged
input defaults and successful generated Go builds. After service analysis named
`StreamDefaultedResult`, shared async analysis reused an ordinary cached
`StreamOptionalResponseBody` with the same structural fingerprint and absent
defaults. The prior flattened async baseline retained them. No Bytes occur in
this case, so byte projection cannot repair the incorrectly acquired baseline.
This weakens the production baseline-acquisition claim, not the earlier path
invariants under their correct-baseline premise.

The new model has three immutable occurrence inputs: an ordinary response and
two distinct named occurrences within one async message. Their structural shapes
and examples are identical. Each occurrence independently selects absence or
one of two default values for each of `event`, `id`, and `retry` (27 payloads per
occurrence). Retry values are opaque string labels for numeric defaults, not a
model of JSON numeric encoding. Existing memo payloads come from the ordinary
input independently of the requested async inputs. The cross-consumer scenario
seeds that memo; the same-message scenario starts empty. Both async occurrences
are acquired twice, interleaved, in either order. Identity is the stable index of
the materialized schema in the memo, not its name or its annotation value.

The structural policy fails `AuthoredPayloadPreserved` against immutable input:
the first cross-consumer acquisition can lose `retry=1500`; with a fresh memo,
the second named async occurrence can lose the same default. A fresh analyzer
per async message therefore does not alone resolve this acquisition rule.
Annotation-aware matching compares the complete modeled payload as well as shape.
Occurrence-owned matching uses the captured occurrence and consumer/materialization
usage (`http-response` versus `async-inline`). Both pass payload preservation,
repeated-occurrence identity reuse, and memo identity stability at these bounds.
The never-cache control preserves payloads but fails identity reuse on the third
request. Public-name labels and structural identity remain separate; neither
candidate encodes annotations into public names or changes the structural key.

This is a bounded acquisition model, not a Go implementation or schema-extraction
proof. Correct captured occurrence/usage, structural constraints, original
annotation extraction, and memo-entry materialization are assumptions. Only
nested defaults vary here; examples deliberately remain identical, and other
annotation kinds, recursive schemas, aliases, projection, public-name allocation,
and codec behavior are outside the model. An annotation-aware implementation
would need complete real annotation equivalence, including absence and every
relevant annotation kind; an occurrence-owned implementation would need correct
occurrence and usage capture. Correspondence tests must exercise the full
service-analysis-to-OpenAPI pipeline, both contamination scenarios, and stable
repeated acquisition. The relevant production seams are OpenAPI IR baseline
acquisition and async schema materialization; those Go repairs and tests remain
separate obligations. The existing three modules and 21 configurations are
unchanged.

Run each new configuration serially from this directory with TLC 2.19. For
example (substitute each table row's configuration and a unique output path):

```sh
/Library/Java/JavaVirtualMachines/amazon-corretto-25.jdk/Contents/Home/bin/java \
  -XX:+UseParallelGC -cp /tmp/loom-tla2tools.jar tlc2.TLC -workers 1 \
  -metadir /tmp/loom574-baseline-acquisition-model-annotation-aware/states \
  -config acquisition-annotation-aware.cfg BaselineAcquisition.tla
```

Independent exact-source review reran and confirmed all five configurations:

| Configuration | Exit / invariant | Distinct states | Depth |
| --- | --- | ---: | ---: |
| acquisition-structural-cross | 12 / AuthoredPayloadPreserved | 39,370 | 2 |
| acquisition-structural-message | 12 / AuthoredPayloadPreserved | 2,919 | 3 |
| acquisition-annotation-aware | 0 / all three invariants | 204,120 | 5 |
| acquisition-occurrence-owned | 0 / all three invariants | 204,120 | 5 |
| acquisition-never-cache | 12 / RepeatedOccurrenceReused | 4,375 | 4 |

Passing configurations enumerate 40,824 initial states and all four acquisitions;
the negative controls stop on the specified invariant, not a checker error.
Retain these sources, configurations and findings. Remove task-owned checker
states and logs after validation and independent review finish.

## Consumer dispatch before baseline acquisition

`ConsumerDispatch.tla` checks an earlier premise of `BaselineAcquisition`:
correct occurrence-owned caching cannot repair a shape already transformed by
another consumer's policy. The complete public pipeline comparison for
`StreamingPayloadResultCollectionWithExplicitViewDSL` found an unapproved
non-Bytes difference: parent async items contain `a,b,c`, while the candidate
contains only `a` and generated view descriptions. Input, paired plan, and
retained sampler still contain all three fields. `analyzeUserType` invokes
`analyzeProjectedResult` before async consumer dispatch; the parent async path
lowered inline named wrappers before that ordinary result-view policy applied.
The evidence covers OpenAPI alone, Service-to-OpenAPI, and
Service-to-Transport-to-OpenAPI. It does not change the ordinary response's
selected-view contract.

The model has three independently assigned named positions: an object, a
collection element, and a nested object. Each has an immutable role selected
from async-inline, ordinary-response, and retained-cut, plus an absent or authored
description. The selected view ranges over all seven nonempty subsets of
`{a,b,c}`, including the full set; it is shared across the three positions in
each run. Capture, the two dispatch/policy stages, and occurrence-owned memo
acquisition are separate transitions. All three positions are acquired twice,
with intervening acquisitions. Every policy uses the same correctly keyed cache.
The independent obligations require inline fields and original descriptions to
remain intact, ordinary and retained positions to apply their selected view,
and repeated occurrences to retain their memo identity. Ordinary policy adds a
generated-view description only where an authored description is absent.

The ordinary-first control reduces an inline object to `{a}` and adds a generated
description before its first cache insertion. The owner-first candidate selects
the consumer first, then applies ordinary policy only to ordinary responses and
retained cuts. It passes the original five view/identity invariants, but is only
partial evidence: dispatch order does not establish correct constructor policy.
The global-suppress control rejects
an overbroad repair: ordinary/retained positions still require view projection
and their ordinary annotation policy. None of these policies changes cache keys
or reconstructs fields after acquisition.

Two further public-DSL comparisons exposed constructor-policy contamination
after correct dispatch. An Int alias `Inner` with enum `{1,2}` and alias `Outer`
with enum `{2,3}` has parent async enum `{2,3}` and sample `3`; the new constructor
adds an inner `AllOf` constraint, narrowing the allowed values to `{2}` and
invalidating that sample. Separately, a named String alias with occurrence-level
`Nullable()` is a nonnullable string in the parent async inline contract; importing
ordinary nullability adds a string/null alternative. These are baseline changes,
not authorized Bytes corrections. Ordinary responses and retained cuts keep
their existing constraint and nullability policies; this model does not choose
the pending M3 value semantics.

The strengthened experiments independently vary base and use enum sets over all
seven nonempty subsets of `{1,2,3}` (49 pairs), and source/type nullability and
usage nullability over all four Boolean pairs. These inputs are shared across
positions to bound the state space. Construction happens after view dispatch and
before cache insertion. An inline consumer must retain the legacy use-site enum
override and source/type nullability; ordinary/retained consumers use enum
intersection and source-or-usage nullability. The constraint control correctly
dispatches, preserves inline nullability, but imports ordinary enum intersection;
the nullability control correctly dispatches and preserves inline enum override,
but imports ordinary nullability. They fail separate invariants, so an enum
failure cannot hide the nullability failure. `consumer-owned` selects both
constructor policies by consumer and checks all seven obligations together.

The original three configurations remain byte-identical and reproduce their
original counts with neutral enum sets `{1,2,3}` and both nullable flags false.
Their view-only passing result does not establish the strengthened contract.
`dispatch-consumer-owned` checks the full cross-product: 1,512 original inputs
times 49 enum pairs times four nullable pairs, with no reduced interaction suite.

This model assumes correct named-position, role, selected-view, and original
description extraction. Position labels cover object, collection, and nested
contexts, but do not model array traversal or prove recursive wrapper lowering.
Retained cuts are supplied roles, not a proof of recursive cut detection or
component registration. Enum sets abstract the allowed-value effect of alias
constraints, not `AllOf` syntax or sample synthesis. The null flags assume correct
capture of the underlying type and the outer occurrence; Go extraction remains
unproved. Byte-length projection and byte enum/null semantics are outside this
model and remain separate Lean/Go obligations. Other annotations remain separate
obligations. The existing
annotation-acquisition model and its five configurations remain unchanged.

Production correspondence belongs at `analyzeSchema`'s inline-role dispatch
before ordinary named-type handling, including result-view selection, validation
overlays and outer-occurrence nullability. Baseline acquisition and async
materialization preserve that role; ordinary and retained positions keep their
existing policies. Required Go regressions include exact public
Service/Transport-to-OpenAPI parent shape and annotations for the existing
explicit-view fixture; ordinary response projection; inline objects, collection
elements and nested named results; and retained/recursive cuts preserving
ordinary view and registration behavior. Actual view extraction, callback order,
runtime wire behavior, recursive termination, and final whole-document equality
remain Go test/comparison obligations, not conclusions of this bounded model.
Alias enum override and occurrence-nullability regressions must also compare the
parent async contract and ordinary/retained behavior through the public pipeline;
moving dispatch alone is not an adequate repair.

Run each configuration serially with TLC 2.19, using a distinct external output
directory, for example:

```sh
/Library/Java/JavaVirtualMachines/amazon-corretto-25.jdk/Contents/Home/bin/java \
  -XX:+UseParallelGC -cp /tmp/loom-tla2tools.jar tlc2.TLC -workers 1 \
  -metadir /tmp/loom574-consumer-dispatch-consumer-owned/states \
  -config dispatch-consumer-owned.cfg ConsumerDispatch.tla
```

Independently reviewed results:

| Configuration | Exit / invariant | Distinct states | Depth | Seconds |
| --- | --- | ---: | ---: | ---: |
| dispatch-ordinary-first | 12 / InlineShapeAuthority | 6,049 | 5 | 0.79 |
| dispatch-owner-first (partial view-only check) | 0 / original five invariants | 37,800 | 25 | 1.01 |
| dispatch-global-suppress | 12 / OrdinaryAndRetainedPolicy | 6,553 | 5 | 0.64 |
| dispatch-ordinary-inline-validation | 12 / ConsumerConstraintAuthority | 1,185,413 | 5 | 2.83 |
| dispatch-ordinary-inline-nullability | 12 / ConsumerNullabilityAuthority | 1,185,410 | 5 | 2.83 |
| dispatch-consumer-owned | 0 / all seven invariants | 7,408,800 | 25 | 52.62 |

The first three configurations enumerate 1,512 initial states; the strengthened
three enumerate 296,352. Each passing search covers all six acquisitions.
Negative controls stop on contract violations, not parser or resource errors.
Keep sources/configurations and these concise results;
remove owned checker states after validation and independent review, retaining
small logs only while a final review still needs them.

## Run and expected results

Use TLC 2.19, one configuration at a time, with output outside the repository:

```sh
java -XX:+UseParallelGC -cp /path/to/tla2tools.jar tlc2.TLC -workers 1 \
  -metadir /tmp/loom-schema-allocation-check -config checked.cfg SchemaDeclaration.tla
```

Use `AliasPairing.tla` for `pairing-*` and `AnnotationPath.tla` for `path-*`. Parser/resource
errors never count as expected failures. The current local runs produced:

| Configuration | Exit / invariant | Distinct states |
| --- | --- | ---: |
| byte-canonical | 12 / VariantOwnerPreserved | 248,834 |
| checked | 0 / all five invariants | 311,040 |
| child-cached | 12 / AcquiredOwnerPreserved | 248,869 |
| edge-checked | 12 / AcquiredOwnerPreserved | 248,977 |
| fingerprint-cached | 12 / VariantOwnerPreserved | 248,834 |
| legacy | 12 / DeclarationAuthority | 82,945 |
| mapping-confused | 12 / CapturedIdentityStable | 69,985 |
| mapping-lost | 12 / SemanticBindingStable | 69,985 |
| plan-only | 12 / AcquiredOwnerPreserved | 248,905 |
| source-fallback | 12 / DeclarationAuthority | 251,425 |
| target-only | 0 / all five invariants | 311,040 |
| unchecked-sharing | 12 / AcquiredOwnerPreserved | 248,833 |
| pairing-legacy | 12 / SourceCorrespondence | 130 |
| pairing-ancestry-only | 12 / SourceCorrespondence | 126 |
| pairing-terminal-ancestry | 12 / ExactRejection | 7,820 |
| pairing-checked | 0 / all three invariants | 26,510 |
| path-terminal-only (insufficient check) | 0 / terminal only | 3,528 |
| path-reconstruct | 12 / AnnotationPathPreserved | 1,837 |
| path-drop-authored | 12 / AnnotationPathPreserved | 1,885 |
| path-resample-absent | 12 / OriginalAbsencePreserved | 1,849 |
| path-checked | 0 / all four invariants | 3,528 |

Declaration runs reach depth five, except the provenance violations at depth two;
pairing's violations reach depth two and its checked search reaches depth six.
Independent review reran the sixteen declaration/pairing configurations and
confirmed their results, including separate provenance and structural binding. Actual wrapper
source-pointer and binding-propagation controls fail before repair and pass after;
the existing structural-flattening control remains supported.
These results support bounded candidate rules, not universal generated-code
validity. Acceptance also requires actual neutral-first and planned-first renderer
comparisons, authored aliases with and without Bytes, exact annotation callbacks,
and projection correspondence across shared transports. See the
[correspondence ledger](../../lean/value_projection/correspondence.md).
Keep only source/configuration/documentation; remove owned checker outputs after
validation and independent review finish.


## Prepared example and structural plan roles (#572)

`PreparedExampleRoles.tla` separates the authority to project schema structure
from the exact position used to attach a prepared example. A standalone schema
has no transport projection authority even when its examples were prepared with
a JSON plan. A transport occurrence has both roles. Both positions advance in
the existing recursion and are restored together when a scope exits, including
exception unwinding.

Run from this directory with the TLA+ tools JAR available:

```sh
java -cp "$TLA2TOOLS_JAR" tlc2.TLC -config prepared-roles-checked.cfg PreparedExampleRoles.tla
java -cp "$TLA2TOOLS_JAR" tlc2.TLC -config opaque-enums-checked.cfg OpaqueEnumWitnesses.tla
```

The paired policy passes all three invariants in 290 distinct states. The model
explores two consumer kinds, five builtin root codec categories, two child edges,
depth three and up to eight transitions.
`prepared-roles-conflated.cfg` reproduces assigning example preparation authority
to schema structure. `prepared-roles-clear-both.cfg` detects losing examples when
neutralizing schema projection. `prepared-roles-stale-child.cfg` detects failure
to advance the example position, and `prepared-roles-leaky-restore.cfg` detects
failure to restore it. `prepared-roles-inherited-wire.cfg` detects reusing the
runtime root codec for JSON-valued documentation examples. All five negative
configurations fail the relevant authority invariant. Runtime codec inputs remain
unchanged; this is documentation observation, not a runtime codec migration.

The model abstracts captured edges as exact handles. It does not establish Go
edge extraction, memoization, construction records, codec behavior or actual
exception handling. Node-local custom codecs, mapped SSE policies, text
formatters and raw-media serialization remain separate correspondence boundaries.
Direct tests must check neutral standalone schemas, actual
transport projections, distinct child examples, scope restoration, and existing
annotation ownership. The earlier acquisition and annotation-path models remain
applicable; this model does not replace their obligations.

## Opaque enum alternatives (#572)

`OpaqueEnumWitnesses.tla` checks conservative enum membership when a clause has
both semantically known and opaque alternatives. A known matching alternative can
establish membership. An all-opaque clause supplies no checked witness; retaining
that empty set of known witnesses prevents accidental acceptance. Clauses remain
intersected. This affects checked example projection only: the complete declared
enum remains in the rendered schema, and no custom codec runs during plan building.
Malformed enum values with independently checkable failures still reject the plan.

The checked configuration passes in 128 states over two values and two nonempty
declared clauses, with every subset of known witnesses. The `opaque-enums-drop-empty.cfg`
negative control violates sound membership by dropping a clause with no known
witness. These finite set properties do not prove the resolver identifies opaque
values correctly. Direct tests cover that correspondence, preserved malformed
input errors, mixed alternatives, clause intersection, supported siblings, and
codec call counts. An opaque alternative alone is not a builtin semantic witness;
the existing custom materialization boundary remains responsible for its codec.
