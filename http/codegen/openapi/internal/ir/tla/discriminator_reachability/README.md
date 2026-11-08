# Discriminator declaration reachability

This bounded design model separates discriminator declaration membership from
rendered wrapper shape, component names, and ordinary reference reachability.
It informed the shared IR dependency edges and the versioned serializer's
reference traversal. It is an opt-in design aid, not a build, test, hook, CI or
release requirement.

## Contract and counterexamples

Publishing a ParentHint declaration must retain its direct, reference-preserving
`allOf` children. A flat `Extend` alone does not add that implicit dependency;
an explicitly mapped flat child is retained by its mapping reference instead.
The child-to-parent `allOf` reference alone cannot retain an otherwise unused
child when the document only references the parent. Grandchildren of an
unhinted child and unrelated parent/child families must remain pruned.

Both explicit mapping and default-mapping references participate in alias
rewriting and reachability. Representation allocation must rename dependency
endpoints along with ordinary references. A hint belongs to its semantic named
declaration even when that declaration's rendered schema has an outer `allOf`
wrapper. It must not be rediscovered by looking only at a top-level rendered
`Discriminator` field.

The state machine allocates names and records edges, collapses the two-step
alias chain, and explores every order in which the pending component set can be
visited. The expected result is computed independently as a fixed-point closure
of semantic roots and required edges. Completion requires exactly that set, no
dangling retained hint targets, only direct semantic child edges, and eventual
termination. The model includes a structural reference cycle, a wrapped parent,
an unused family, flat extensions, explicit/default alias targets, and combined
dependencies. Every positive scenario runs with unchanged and reassigned names.

## Run explicitly

Use TLA+ tools **1.7.4** (TLC reports version 2.19). From this directory, set
`TLA_TOOLS_JAR` to the installed jar and choose a fresh task-owned output path:

```sh
java -XX:+UseParallelGC -cp "$TLA_TOOLS_JAR" tlc2.TLC -workers 1 \
  -config semantic.cfg -metadir /tmp/loom-discriminator-semantic \
  DiscriminatorReachability
```

Replace the configuration and output directory for each countercontrol below.
Do not run two checkers against the same output directory. No installation or
proof execution is added to any ordinary repository gate.

| Configuration | Expected result | Observed distinct states | Meaning |
| --- | --- | ---: | --- |
| `old-implicit.cfg` | Exit 12, `CompletePublication` | 9 | Reference-only pruning loses an implicit child. |
| `old-mapping.cfg` | Exit 12, `CompletePublication` | 9 | Old traversal omits explicit mapping targets. |
| `draft-wrapped.cfg` | Exit 12, `CompletePublication` | 11 | Top-level rendered hint lookup misses a composed parent. |
| `no-mapping-rewrite.cfg` | Exit 12, `CompletePublication` | 11 | Traversal follows an alias already deleted by cleanup. |
| `no-mapping-traversal.cfg` | Exit 12, `CompletePublication` | 9 | Rewritten mapping exists but its target is pruned. |
| `all-bases.cfg` | Exit 12, `NoExtraneousPublication` | 11 | Reversing flat composition edges publishes an unrelated implicit child. |
| `stale-names.cfg` | Exit 12, `CompletePublication` | 10 | Dependency endpoints retain names that allocation replaced. |
| `old-controls.cfg` | Exit 0, all invariants and termination | 44 | Existing flat, default-mapping, directly referenced cycle and unused-family behavior remains valid. |
| `semantic.cfg` | Exit 0, all invariants and termination | 100 | Eight concrete scenarios, both naming modes, all work-set visit orders. |

These are intentional invariant counterexamples, not tool errors. The positive
run generated 120 states from 16 initial states. State counts describe this
finite fixture set, not arbitrary schemas.

## Implementation correspondence

- IR `compositionReferenceBases` is shared by emitted composition references
  and dependency extraction. `Compositions` versus `AllOfEdges` models the
  authored identity versus actual reference marker distinction.
- `recordDiscriminatorChildren` combines those direct relationships with the
  named declaration's ParentHint ownership. `ParentHints` is a semantic fact;
  `WrappedHints` represents an outer composition wrapper that leaves that fact
  unchanged. The rejected draft removed those parents from its recorded set.
- `renameDiscriminatorChildren` and `installRepresentations` rename recorded
  endpoints. `Allocate` checks identity versus reassigned names. It does not
  model allocation collisions or schema-equivalence merging.
- v3 `rewriteSchemaRefs` must rewrite `Discriminator.Mapping` as well as
  `DefaultMapping`. `CollapseAliases` removes alias components and resolves
  those target names before pruning.
- v3 `collectSchemaRefs` includes both mapping kinds, and
  `collectReachableComponentSchemas` follows IR child dependencies from each
  reachable parent. `Visit` is a work-set abstraction of that queue. It allows
  every next-node order while marking nodes before their later traversal.

The wrapped-parent counterexample also has an executed native DSL witness:
`Base`, a hinted `Parent` that extends `Base` with the allOf-reference marker,
and a direct reference-preserving `Child`. The draft produced Parent.AllOf but
no Parent-to-Child dependency. The maintained regression belongs beside
`TestDiscriminatorParentHintChildren`; the code fix must preserve the semantic
owner rather than scan arbitrary nested rendered discriminators.

## Bounds, assumptions, and remaining tests

This model assumes source admission and declaration identity are already
correct. It uses nine component identities, a two-hop terminating pure-reference
alias chain, injective name reassignment, and the eight listed graphs. The
structural cycle is valid reference recursion; cycles made only of alias nodes
are outside the model. Full alias validation, arbitrary recursive schemas,
representation-equivalence merging, hash collisions, declaration rollback,
nullable-wrapper construction, and external reference resolution remain
implementation obligations.

Captured implicit dependency endpoints are non-pure-ref declaration components:
the child emits an allOf body, while the parent owns a hint (possibly under its
own composition wrapper). This is an explicit construction assumption, not an
assertion that arbitrary auxiliary dependency maps survive alias deletion.
Tests must establish this correspondence for aliases and representation
allocation. If the implementation can emit a dependency endpoint that is itself
a deleted pure alias, extend the model and canonicalize that edge at the shared
alias boundary before relying on these results.

Direct Go controls must cover plain/composed/nullable ParentHints, aliases and
renamed components, direct children versus grandchildren/flat extensions,
explicit and default target aliases, unrelated unused families, and source
isolation. Rendered JSON/YAML must preserve the references, parse successfully,
and retain deterministic exact bytes across independent generations. Actual
generated imports must compile; this model proves neither Go generation nor
discriminator validation semantics. Preserve existing runtime value contracts:
these publication edges do not add schema validation branches or mapping keys.

Keep only this source, configurations and concise findings in the repository.
Checker state, verbose logs, draft-source snapshots and temporary Go probes stay
outside it and can be removed after independent review and the last dependent
task finish.

## Representation extension

`RepresentationPublication.tla` extends the declaration model with byte
representations. A declaration has a raw registration representation and a
JSON representation; the allocation quotient later merges equal component
bodies. The model keeps the occurrence codec context separate from that
quotient.

Observed failures:

- **R2** lost the child of a JSON byte-projected parent and kept raw hint
  targets.
- **R3** projected a declaration only when its *own* body changed. A byte-free
  JSON parent therefore published its byte-bearing child, mapping or default
  target, and a union's byte-bearing fallback in the raw representation. The R3
  oracle (`Expected`) used the same own-body rule and accepted that result.
- **R4** resolved the context only from an occurrence node's physical named
  type when that type was a declaration root. It read composition contributors
  only from named types' attributes. HTTP lowering hides the declaration in
  body copies, in nested fields, array elements and map values (renamed copies
  whose IDs are not roots), and in flattened unions. Inline composed fields
  carry their contributor on an unnamed attribute. All of those shapes lost the
  context. An inline composed root kept it, because the named body copy's own
  attribute carries the contributor.
- **v4** analyzed form and multipart bodies from value-plan copies. An inline
  composition inside a JSON-encoded field lost its allOf reference, the
  requiredness its parent imposes, and the parent's children. A JSON-encoded
  field also keeps the form plan codec, so the family it reaches is raw.

The corrected design keeps every publication edge inside one representation:

- D1: a composition contributor is projected in the base declaration's own
  representation, not through the child's occurrence plan (`ChildPlanBase`).
- D2: mapping and default references of a projected discriminator follow the
  projected alternative or the referenced declaration's representation
  (`RawHintRefs`).
- D3: projecting a hinted parent root publishes its direct children in that
  representation (`RawDependenciesOnly`).
- D4: registration arbitration follows dependency edges like references
  (`IRFollowsDependencies`).
- D6: a JSON occurrence projects whenever the byte identity of its
  *representation context* is non-empty: discriminator publication targets and
  allOf contributors, transitively (`Design`). A context without contributors
  (`noContributorContext`) leaves a byte-free child extending the raw parent
  while its JSON parent differs.
- D7: the allocation quotient compares dependency targets as well as bodies, so
  the JSON and raw representations of a byte-free parent stay distinct when they
  publish different children (`bodyQuotient`).
- D8: every occurrence node resolves its declaration through its value plan's
  target declaration, not its physical type (`noTargetDeclaration`). Every
  attribute contributes its own composition contributors, named or inline
  (`noInlineContributors`). `OccurrenceNodes` lists each occurrence shape's
  nodes with their physical named kind (`decl`, `copy` or `none`), plan target
  and own contributors, and `Resolved` applies a design's resolution to them.
- D9: projection decides at the occurrence root from the occurrence attribute
  graph, but at every deeper member from the value-plan node. There the plan
  target survives, but the owned attribute copy carries no composition
  contributors. This is the intended capture contract, not a capture defect:
  value capture copies only the constraint-local value contract
  (`constraintLocal`) and drops declaration edges (`Type`, `Bases`,
  `References`). Composition-reference identity is an OpenAPI rendering fact
  owned by the graph attribute (`compositionReferenceBases`) and the
  construction record (`schemaSlot.base`), so the IR reads contributors from
  those owners. Earlier reports called the missing `compositionBases` "the
  value-capture defect itself"; that wording was inaccurate and is superseded
  here. `OccurrencePath` orders each shape's nodes from root to leaf, and
  `ContextSeen` requires the root and every deeper member to see the context.
  - P1 (`SlotScope`): a member consults the contributors that constructions
    record (`constructionOnly` alone misses the holder of a deep inline field).
  - P2 (`DeclarationGraphContext`): a member that denotes a declaration sees the
    publication neighbors of that declaration's whole graph
    (`declarationGraphOnly` alone misses the inline composition member).
- D10: P1 reads the contributors recorded at every *descendant* JSON position
  of a member, not only at its own value position (`SlotScope = "descendant"`).
  v3 read only the member's own position (`"position"`). An intermediate that
  denotes no declaration, such as an inline holder, an inline composition whose
  body holds the composed attribute, or an inline tagged-union branch, has
  neither a plan target for P2 nor a contributor at its own position, so v3 left
  it raw. Contributors count only at JSON codec nodes (`CodecAllows`); the
  `codecBlind` countercontrol also counts them at non-JSON nodes and wrongly
  gives the form occurrence (`FormPath`) of `mixedInline` JSON context.
- D11: a text, form or multipart body is analyzed from its value-plan copy,
  which keeps the occurrence metadata but not its composition bases. The
  composition contributors of every node below such a root are read from the
  original graph attribute paired with the node's plan position
  (`GraphAcquisition`). Without it (`v4`, `jsonBoundaryOnly`) the authored
  composition reference is never emitted (`AuthoredComposition`). The form
  scenarios distinguish the root's form codec from the wire codec of each
  field (`wire`). The captured `HTTPFormBodyEncoding` and
  `HTTPMultipartBodyEncoding` contracts decide it. Object values default to
  JSON content; array content follows the captured Value or Items grammar.
  Scalar items default to text but can use explicit JSON content; multipart
  byte items stay binary even with a JSON media label. An explicit style keeps
  supported scalar-member objects and unions as form text.
- D12: the plan codec of a JSON-encoded descendant of a form or multipart root
  is JSON (`JSONBoundary`). Without it the descendant inherits the form plan
  codec, so a composition reference acquired by D11 publishes the raw parent
  and its raw child although the wire carries JSON (`graphAcquisition`).
  Under item application only each item is JSON content, and the array keeps
  the form codec (`formJSONItems`). A boundary belongs to the normalized
  occurrence that one field, branch or map value owns privately. Attached to a
  declaration that a form occurrence of P shares (`formMixed`), it would leak
  the JSON codec into that occurrence (`sharedBoundary`). `FormPath` is that
  form occurrence: a form body that is P itself, whose fields are form fields.
  A P-typed field of a form body is JSON content, like `formJSONField`.

`Expected` is computed with the full design and full occurrence resolution,
independently of `Mode`. `JSONContextEncoding` is a design-independent oracle:
every byte-bearing declaration published from a JSON occurrence, by reference
or dependency, is in its JSON representation. `ConsistentDependencies` checks
that after the quotient each component's implicit children share one
representation.

| Configuration | Expected result | Observed distinct states | Meaning |
| --- | --- | ---: | --- |
| `rep-r4-occurrence-roots.cfg` | Exit 12, `CompletePublication` | 17 | R4 loses the context for body copies, flattened unions, nested copies and inline fields. |
| `rep-r4-json-oracle.cfg` | Exit 12, `JSONContextEncoding` | 17 | The independent oracle alone rejects those R4 shapes. |
| `rep-no-target-declaration.cfg` | Exit 12, `CompletePublication` | 13 | Inline contributors without plan targets still miss copies and flattened unions. |
| `rep-no-inline-contributors.cfg` | Exit 12, `CompletePublication` | 5 | Plan targets without inline contributors miss an inline composed field. |
| `rep-r3-plain-byte-child.cfg` | Exit 12, `CompletePublication` | 5 | R3 publishes the raw child of a byte-free JSON parent. |
| `rep-r3-plain-targets.cfg` | Exit 12, `CompletePublication` | 17 | R3 keeps raw mapping/default/union-fallback targets. |
| `rep-r3-json-oracle.cfg` | Exit 12, `JSONContextEncoding` | 19 | The independent oracle alone rejects R3. |
| `rep-body-quotient.cfg` | Exit 12, `ConsistentDependencies` | 11 | A body-only quotient merges JSON and form parents with different children. |
| `rep-no-contributor-context.cfg` | Exit 12, `CompletePublication` | 21 | Publication-only context: a byte-free child extends a duplicate raw parent. |
| `rep-r2-implicit.cfg` | Exit 12, `CompletePublication` | 4 | Raw-only dependencies lose the JSON parent's child. |
| `rep-r2-mapped.cfg` | Exit 12, `CompletePublication` | 5 | A projected parent keeps a raw mapping target. |
| `rep-child-plan-base.cfg` | Exit 12, `NoExtraneousPublication` | 6 | A refining child's contributor is a second parent variant. |
| `rep-raw-hint-refs.cfg` | Exit 12, `CompletePublication` | 13 | Mapping/default targets stay raw under a JSON parent. |
| `rep-no-ir-deps.cfg` | Exit 12, `CompletePublication` | 14 | The raw child of a form parent is discarded when the child has its own JSON occurrence. |
| `rep-r2-controls.cfg`, `rep-r3-controls.cfg` | Exit 0 | 5 / 54 | Plain and byte-parent behavior already handled by R2/R3. |
| `rep-r4-controls.cfg` | Exit 0, all invariants and termination | 94 | R4 scenarios plus the inline composed root, which R4 already handled. |
| `rep-r5-inline-slot.cfg` | Exit 12, `CompletePublication` | 9 | R5 member decisions miss inline and deep inline composed fields. |
| `rep-r5-json-oracle.cfg` | Exit 12, `JSONContextEncoding` | 9 | The independent oracle alone rejects those R5 shapes. |
| `rep-construction-only.cfg` | Exit 12, `CompletePublication` | 5 | P1 without P2 misses the holder member of a deep inline field. |
| `rep-declaration-graph-only.cfg` | Exit 12, `CompletePublication` | 9 | P2 without P1 misses the inline composition member. |
| `rep-r5-controls.cfg` | Exit 0, all invariants and termination | 109 | R5 handles every other occurrence shape. |
| `rep-v3-inline-intermediate.cfg` | Exit 12, `CompletePublication` | 17 | Position-only P1 (v3) misses inline holders, nested inline compositions, inline union branches and the JSON side of `mixedInline`. |
| `rep-v3-json-oracle.cfg` | Exit 12, `JSONContextEncoding` | 17 | The independent oracle alone rejects those v3 shapes. |
| `rep-codec-blind.cfg` | Exit 12, `CompletePublication` | 5 | Counting contributors at non-JSON nodes gives the form occurrence JSON context. |
| `rep-v3-controls.cfg` | Exit 0, all invariants and termination | 119 | v3 handles the eighteen earlier scenarios. |
| `rep-v4-form-json.cfg` | Exit 12, `CompletePublication` | 7 | v4 loses the composition reference below a form root. |
| `rep-v4-authored-oracle.cfg` | Exit 12, `AuthoredComposition` | 7 | The independent oracle alone rejects v4. |
| `rep-graph-acquisition-only.cfg` | Exit 12, `CompletePublication` | 13 | D11 without D12 publishes the raw parent and child from a JSON-encoded field. |
| `rep-graph-acquisition-json-oracle.cfg` | Exit 12, `JSONContextEncoding` | 13 | The independent JSON oracle alone rejects D11 without D12. |
| `rep-json-boundary-only.cfg` | Exit 12, `CompletePublication` | 7 | D12 without D11 still loses the composition reference. |
| `rep-v4-controls.cfg` | Exit 0, all invariants and termination | 145 | v4 handles the twenty-two earlier scenarios. |
| `rep-v5-partial-form.cfg` | Exit 12, `CompletePublication` | 9 | D11 without D12 (the v5 partial source) also fails item-applied and mixed form occurrences. |
| `rep-shared-boundary.cfg` | Exit 12, `CompletePublication` | 5 | A JSON boundary attached to a shared declaration gives the form root occurrence JSON context. |
| `rep-representation.cfg` | Exit 0, all invariants and termination | 176 | Twenty-seven scenarios with D1–D12. |

Run it like the base model with `RepresentationPublication` as the module and a
fresh `-metadir` per configuration.

- `rep-no-contributor-context.cfg` omits the `Terminates` property: with TLC
  2.19 its liveness evaluation did not finish within 90 seconds for that
  countercontrol, while each of its invariants terminates and reports the
  intended violations.
- The D8 to D12 countercontrols omit `Terminates` for the same reason;
  they exist for their safety counterexamples.
- Termination of the corrected design is checked by `rep-representation.cfg`.

Implementation correspondence:

- D1: `schemaSlot.base` and `declarationRepresentation`.
- D2: `projectDiscriminatorReferences`.
- D3: `projectDiscriminatorChildren`, called from `projectBaselineComponent`.
- D4: the dependency-following reach in `discardRegistrationOnlyVariants`.
- D6: `representationIdentity` and `projectionIdentity` with
  `occurrenceContext`, `declarationClosure` and `publicationDependencies`.
- D7: `completeSchemaFingerprints` with dependency classes.
- D8: `projectionIdentity` consults its context at every JSON node.
  `occurrenceDeclaration` resolves `ValuePlanNode.TargetDeclarationID` through
  `declarationTypes`. `attributePublicationNeighbors` is the one neighbor
  contract for occurrence attributes (`occurrenceContext`) and declaration
  graphs (`declarationNeighbors`).
- D9/D10 P1: `constructionContext`, read by `projectBaseline`, pairs the
  baseline's construction with its plan node and walks every recorded slot with
  the plan node `projectBaselineChildren` projects it at. Both use
  `schemaConstruction.structuralPlan` and `schemaSlot.plan`; `positional`
  names the slots that keep their parent's position. The walk follows local
  component references as `projectBaselineComponent` does, uses the
  `(schema, plan)` pair (`schemaPosition`) as its cycle key, and adds the
  `declarationClosure` of each `schemaSlot.base` whose composition sits at a
  JSON plan node (`CodecAllows`).
- D9 P2: `occurrenceContext` adds the neighbors of `declarationNeighbors` for
  the resolved declaration to the attribute's own neighbors.
- `constructionContexts`, `declarationNeighborCache` and
  `publicationDependencyCache` memoize these reads on the analyzer, with the
  lifetime of `publicationContexts`: they are filled after declarations and
  discriminator families are recorded and read before representation
  allocation renames components. `constructionContexts` is keyed by
  `schemaPosition`, whose plan node carries the codec. The declaration caches
  hold codec-independent declaration facts. They return clipped slices that
  callers borrow read-only: clipping keeps an append from writing into the
  cache's spare capacity, but it is not a clone, so callers must not assign
  or reorder elements.
- D11: `analyzeCompositionReferences` reads the composition bases of a
  position from `parameterSources[a.plan]`, the original graph attribute that
  `captureParameterSources` pairs with the plan copy. The copy's own metadata
  still decides whether reference identity is requested. Outside text, form
  and multipart acquisition the attribute is its own owner.
- D12: the representation owner, not the exporter, records the boundaries.
  - `normalizeParameterTarget` and its form and multipart normalizers already
    give every field, branch and form map value a private `TargetGraph` copy.
    They also collect each normalized occurrence that the captured contract
    encodes as JSON content:
    - a Value-applied JSON field, value-key branch or map value;
    - the array element of an Items-applied JSON codec;
    - per-branch codecs of source-array unions;
    - multipart `JSONPart`, with `StyleCodec` delegated to the form grammar;
    - `BinaryPart` and omitted fields record nothing.
  - `BuildValuePlan` appends one `ValueCodecPolicy{JSON}` per collected target
    that no custom or SSE policy already owns (`appendJSONContentCodecs`).
  - The logical `FormJSONShape` and `MultipartJSONShape` pass no collector.
- **Additional positions.** `captureParameterSources` does not pair
  `AdditionalAttributes` with `plan.Additional()`. Text, form and multipart
  acquisitions are the only ones that fill `parameterSources`, and none of
  them admits a typed object: the form and multipart constructors reject it,
  and parameter shapes report it unsupported. So no admitted design reaches
  that position, and the model has no additional-position scenario.

A diagnostic probe executed on R4 (`export_r5/probe-v2`) established the node
facts that `OccurrenceNodes` encodes:

- Body copy and nested copy: a non-root physical type whose plan target is
  `Parent`.
- Flattened union: the plan target is `Choice`.
- Inline field: an unnamed attribute with contributor `Parent`.
- Inline root: a non-root body copy whose own attribute has contributor
  `Parent`.

The R5 probe (`export_r5/probe-v3`) established the D9 facts. The inline and
deep inline fields' plan-node attribute copies carry no contributors, while
their graph attributes do. The deep holder's plan target remains `Holder`.

Maintained tests:

- `TestDiscriminatorPublicationCodecContext`
- `TestDiscriminatorPublicationMixedCodecContext`
- `TestDiscriminatorUnionHintFallbackCodecContext`
- `TestDiscriminatorOccurrenceCodecContext` (nested field, array element, map
  value, inline field, deep inline field, inline root)
- `TestDiscriminatorPlainParentPublishesJSONChild`
- `TestOccurrenceDeclarationFollowsPlanTarget`
- `TestDeclarationNeighborsTraverseAttributes`
- `TestDiscriminatorInlineIntermediateCodecContext` (inline holder, inline
  composition holding a composed member, tagged union branch holder, recursive
  declaration with an inline holder, array of a named holder)
- `TestDiscriminatorInlineIntermediateMixedCodec`: the default JSON form
  field `pet` shares the JSON parent, and a form root that is the parent keeps
  the raw parent and child.
- `TestDiscriminatorParentHintDistinctRepresentations` and
  `TestDiscriminatorPublicationMixedCodecContext`: the form occurrence is a
  form root, not a default JSON field.
- `TestDiscriminatorJSONFieldComposition` (inline holder, inline field and
  named holder, each in form and multipart; D11 and D12, including the JSON
  child's decoded-byte gate)
- `TestFormJSONContentCodecs` (named array items JSON vs text, the same named
  union styled vs JSON, multipart binary items, form map root values)
- `TestFormBodyPlanCodecs`, `TestFormRootPlanCodecs` (D12 plan codecs at each
  field, item, branch, value key, map value, custom and omitted target)
- `TestDiscriminatorFormRootKeepsFormCodec` (form and multipart roots keep the
  form codec)
- `TestParameterCompositionReadsGraphSource` (D11 owner seam)
- `TestConstructionContextReadsRecordedContributors` (descendant, wrapper,
  reference, inline holder and recursive positions; memoized per position)
- `TestOccurrenceContextReadsDeclarationGraph`
- `TestDeclarationGraphMemoization`
- `TestRepresentationEquivalenceComparesDependencies`
- the R3 representation tests

Bounds: five published declarations plus the occurrence-path labels `H`, `W`,
`X` and `B` (holder, wrapper, union and inline composition base), two codec
contexts (JSON and one non-JSON occurrence), a form wire codec, and sixteen
occurrence shapes (the deep inline field, the four inline intermediates and
five form-rooted shapes included). Untagged union branch fields and
array or map elements cannot be inline objects in the DSL, so those
intermediates are not modeled.
`OccurrencePath` abstracts HTTP lowering and value capture; it does not model
`ValuePlan` construction. Implicit keys are the
allocated child component names, including allocation suffixes of non-public
representations.

The model does not check:

- allocator collisions;
- arbitrary declaration graphs, cycles or array/map nesting beyond the listed
  shapes;
- nested JSON fields under non-JSON roots;
- per-field custom codecs;
- operation-specific plans whose projection of a parent differs from the
  parent's standalone JSON representation.
