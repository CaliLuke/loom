# Effective design composition readiness

Status: root approved the model/red checkpoint and scoped production implementation. The uncommitted candidate is ready for root integration and fresh independent review after the final bounded evidence below. Earlier proposal and red-checkpoint sections are historical decision records; the delivery section records the actual implementation and checks.

Inspected source: `ce3f6a733f19de30b50986f86e9ce4cef06e48cb` in `/Users/luca/.codex/worktrees/effective-design-validation/loom`. Historical correctness baseline: `4535a403048510f055fc59ec3dfed41e3769ad58` (managed, read-only baseline). Live issue bodies were read on 2026-10-12. Focused red regressions and model checks were executed after root approval; results and exact correspondence are in `expr/tla/effective_design/README.md`. The delivery section below records production acceptance evidence separately.

## One structural ticket

Use **#755** as owner: compose inherited object declarations into an owned effective contract before validating their consumers. **#757** is a second witness of that same boundary: inherited protobuf numbers are absent when their uniqueness is checked. Deliver one atomic #755 commit with the #757 regression witness; do not combine unrelated admission rules merely because they live in `expr`.

The risk is accepted authored constraints disappearing, or an accepted design reaching gRPC codegen with a number collision. The framework owns composition, constraint refinement, occurrence ownership and rejection before generation. Applications own their field choices and constraint values. Existing framework designs include `ExtendedMethodTypesDSL`, `ExtendedAliasedMethodTypesDSL`, `ExtendedResultMethodTypesDSL`, `PayloadExtendedValidateDSL`, and `GRPCEndpointWithExtendedTypes`; these are the initial compatibility controls. Additional local consumer evidence is recorded below and must remain internal.

| Issue | Actual owner | Disposition |
| --- | --- | --- |
| #755 | `AttributeExpr.Finalize` / `Merge`, effective constraint ancestry | Selected owner |
| #757 | Complete effective object supplied to `validateRPCObjectTags` | Same-cause witness |
| #756 | HTTP request/response wire-name namespace, including implicit credentials | Separate mapping admission ticket; moving Extend cannot detect two locally authored duplicate mappings |
| #701 | Union branch-name admission in `validateChildTypes` / union validation | Separate; duplicate names already exist before inheritance |
| #702 | Service/map key representability | Separate; arrays and Bytes are invalid native Go map keys without any inheritance |
| #703 | Protobuf field-number value domain | Separate from #757 uniqueness; negative/reserved/overflow numbers remain invalid even on a one-field object |
| #699 | Uniqueness within service/method error declaration lists | Separate error-scope owner |
| #710 | HTTP header/cookie token syntax | Separate mapping-name admission; potentially shared work with #756 only after its own readiness review |
| #760 | HTTP result `Tag` selector/type contract | Separate response admission owner |
| #780 | Complete HTTP payload/error binding coverage | Separate transport representability policy |
| #781 | Effective route namespace across services and JSON-RPC mounts | Separate full-route owner |

## Evidence and phase dependencies

1. `eval/eval.go:RunDSL` and `PrepareValidateFinalize` execute prepare, validate, then finalize. Preserve that evaluator contract.
2. `expr/root.go:WalkSets` visits types/result types, services, methods, HTTP/JSON-RPC and gRPC. `expr/union_branch_names.go:RootExpr.Prepare` currently names promoted branches, but returns immediately when its branch sequence is unchanged. Composition must not hide behind that conditional.
3. `expr/attribute_validate.go:AttributeExpr.Prepare` resolves references, normalizes presence and prepares package metadata. `AttributeExpr.Validate` runs effective value checks and recursive child checks before finalization.
4. `expr/method.go:MethodExpr.Prepare` initializes empty and streaming positions. `MethodExpr.Validate` injects selected session credential fields before payload validation. A composition pass must tolerate absent pre-Prepare positions and leave this deliberate injection ordering intact; transport mapping preparation must see the owned effective shape.
5. `expr/http_endpoint.go:HTTPEndpointExpr.Prepare` creates/inherits mappings, infers route params and prepares responses; gRPC preparation initializes requests, responses and error mappings. These depend on named identities and method fields. Their finalizers perform transport body projection, naming and bindings that must remain after admission.
6. `expr/attribute.go:AttributeExpr.Finalize` currently calls `Inherit` for references, then `Merge` for bases, clears `Bases`, then finalizes children. `Merge` calls `left.Set(name, base.Attribute)` and therefore discards local redeclarations and aliases mutable base fields. Object validation is merged separately.
7. `expr/grpc_field_tags.go:rpcMessageAttribute` selects fields using `AsObject`; `validateRPCObjectTags` checks the selected object's numbers. It never composes bases itself. Supply the completed object to this existing owner; do not implement another tag checker in a generator.
8. `expr/go_field_names.go:effectiveGoFields` independently reconstructs the old last-base-wins field set. Replace its reconstruction with consumption of the prepared effective object. Its second consumer is error Go-field validation in `expr/service.go`; retain the `LoomErrorName` collision checks.
9. `expr/result_type.go:Finalize` creates automatic default views only after underlying inheritance, while explicit views retain authored selection. Early composition must preserve this distinction and recursive projection identity.
10. `dsl/attribute.go:Attribute` first calls `DupAtt(parent.FindAttribute(name))` when it finds an inherited field, then applies local type/description and executes local DSL. `Minimum`, `Default` and metadata can therefore operate on a copied base field before composition. Preserve the explicitly authored overlay and its source separately; do not infer author intent from a copied final value or allow Extend-before/after-Attribute order to choose different defaults.
11. `expr/dup.go`, `dsl/attribute_copy.go`, method type copies and value origins preserve different identities. Do not treat declaration identity as permission to share mutable validation, metadata, examples or defaults across occurrences.

Proposed seam: an unconditional, cycle-aware **expr-owned preparation operation**, invoked from root preparation after promoted branch naming, traverses authored types, result types, method positions and errors, resolves structural inheritance, and records detached effective field contracts. It must also cover transport-authored body shapes before their consumers inspect them. The exact roots must be enumerated in direct tests and programmatic `expr.PrepareValidateFinalize` controls. Do not globally move `Finalize` before `Validate`, invoke endpoint finalizers from preparation, or add a generator-time catch-all validation pass.

Composition follows inheritance dependencies before copying fields, with separate active/done state for inheritance edges versus ordinary recursive field references. Publishing the prepared shape is idempotent. `Finalize` consumes that shape; it no longer reselects Extend fields. Keep Reference's selected-template semantics rather than turning it into named ancestry.

## Accepted semantics and exceptions requiring a decision

The user accepted same-type Extend redeclarations as refinements: base minimum 0 plus local maximum 100 means 0..100; local declarations cannot weaken inherited requiredness. Incompatible types and contradictory constraints are rejected. This replaces the issue's original open override-or-reject alternative.

`docs/dsl-reference.md` under “Constraints on Derived Named Types” and `expr/effective_constraints.go` already own conjunction, enum refinement and nearest-default selection. `effectiveConstraintLayers` currently follows a linear `declaration.alias` chain. Extend fields need provenance-preserving additional declaration inputs to **that owner**, not another bound/enum/pattern engine and not lossy `ValidationExpr.Merge`. `EffectiveValidation.Lowered` is a detached predicate carrier, not a replacement for original ancestry/provenance.

| Case | Acceptance rule / proposed handling |
| --- | --- |
| Same scalar type, base minimum 0, local maximum 100 | Accept; enforce both limits in shared values, HTTP, gRPC and schema |
| Local minimum -10 over inherited minimum 0 | Effective minimum remains 0; the added weaker predicate does not remove the inherited one |
| Base required `id`, local omits `Required(id)` | `id` remains required |
| Base String, local Int64 (including #755 original sample) | Reject with field, both declarations and incompatible types |
| Base pattern `^a`, local `b$` | Accept conjunction; `ab` accepted, `xb` and `ax` rejected |
| Base enum `{ab, ax}`, local `{ab}` | Accept; local `{ab, xb}` rejected by existing enum refinement owner |
| Local default 5 replacing base default 2, effective range 0..10 | Select 5 and validate it using existing nearest-default semantics |
| No local default, base default 2, local minimum 3 | Reject effective default; do not silently drop it or synthesize a replacement |
| Local default outside any inherited predicate | Reject |
| Base minimum 10, local maximum 5; analogous length bounds or exclusive equal endpoints | Reject this Extend composition under the user's accepted contradiction rule |
| Existing named-alias ancestry with empty bounds, no Extend | Root-directed scope: preserve existing admission in this ticket. The effective constraint owner computes conjunction unchanged; the new structural Extend composition admission rejects detectable contradictory redeclarations. Record the cross-syntax coherence question as unresolved follow-up, not an unapproved global behavior change |
| Arbitrary patterns/formats whose intersection may be empty | Conjoin and apply existing admission checks; no general regex intersection solver is proposed. Clarify whether “contradictory” requires more than statically decidable bounds/enums before claiming that broader guarantee |
| Diamond that repeats the same ancestor declaration | Deduplicate its inherited guarantees by declaration provenance; accept; no duplicated field or predicate |
| Recursive object field (`Node.next` references Node) | Accept finite graph and preserve recursion |
| Inheritance-only cycle (`A Extend B`, `B Extend A`) | Proposed rejection with cycle path, not recursion truncation; the declaration does not define a finite inheritance order |
| Named descendant used to refine its named base field | Proposed acceptance when the explicit ancestry preserves the same physical representation and all inherited constraints/codec obligations; identical nominal identity is unnecessarily strict |
| Unrelated named types with the same underlying scalar | Reject as incompatible unless explicit named ancestry proves refinement. Representation equality alone is not semantic compatibility; root confirmed this scope |
| Nullable widening from non-null inherited field | Reject an explicitly local Nullable that weakens a non-null base. An omitted Nullable inherits; it is not an authored false/narrowing. No new narrowing DSL is introduced |
| Same base field number reused by a different local/inherited field | Reject during DSL evaluation with both field paths and declaring context |
| Same field redeclared with a different `rpc:tag`, JSON wire name or custom Go type | Proposed identity-metadata conflict rejection, not last-writer selection |
| Local description/title/docs/examples and ordinary metadata versus one base | Preserve local authored value; inherit missing values as detached copies. Constraint metadata must keep the existing owning interpretation |
| Two unrelated bases provide conflicting defaults for one field | Root-approved: an explicitly local default resolves the conflict if valid under the full conjunction; otherwise diagnose. Semantically identical inherited defaults coalesce while retaining provenance |
| Multiple-base metadata differences | Classify documentation overlays versus identity/codec metadata. Local documentation may override; do not reject every differing key. Conflicting representation/wire identities require a diagnosed resolution under their existing key owner |

Root accepted explicit local default disambiguation, identical-value coalescing, named descendant refinement, rejection of unrelated nominal substitutions, and preservation of existing named-alias empty-domain admission for this ticket. The concrete nullable/metadata representation delta appears below; no production change has begun. Defaults for the ordinary one-base case already follow the documented owner and require no new policy. Metadata is not covered by `EffectiveConstraintsFor`; trace each semantic key to its existing owner before selecting or rejecting it. Two different Go field names that collide after an otherwise valid refinement remain an existing Go-name validation error. Duplicate HTTP mapped names remain #756, even when fields came from Extend.

## Concrete adoption and policy evidence

The focused repository search found 27 `Extend` calls under DSL tests and the expr/HTTP/gRPC/JSON-RPC/service fixture directories. `expr/testdata/method_validate_dsls.go:ValidSecuritySchemesExtendDSL` is an actual two-base example: `Common.version` plus `Security.user/pass`, with no shared field. Preserve that accepted additive composition in either order. The imported-allOf fixture in `internal/openapiimport/render_composition_test.go:TestRenderSpringAllOfInheritance` uses one referenced base plus an inline object with disjoint properties; `analyze_schema_composition.go` currently restricts that special import path to exactly that shape. It supplies no peer-default precedence contract.

Internal local adoption check: `/Users/luca/code/loom-mono/loom-mcp/registry/design/design.go` contains `DrainProviderPayload`, `RetryToolPayload`, and `PublishToolOutputDeltaPayload`, each extending one existing payload and adding fields numbered 100 or 101. None redeclares inherited fields or settles overlapping defaults/metadata. The same patterns in the neighboring s17a checkout are copies, not independent evidence. Do not disclose these names or paths in external project communication. No broad consumer search or consumer regeneration was run during readiness.

Examples that distinguish nominal identity, representation and refinement:

```go
number := Type("Number", Int, func() { Minimum(0) })
positive := Type("Positive", number, func() { Minimum(1) })
base := Type("Base", func() { Attribute("n", number) })
// Explicit named ancestry supports this refinement; exact ID equality would reject it.
derived := Type("Derived", func() { Extend(base); Attribute("n", positive) })
```

`Type("Temperature", Int)` substituted for an unrelated `Type("UserID", Int)` has the same Go scalar representation but no declared refinement relation. A `String` alias carrying `struct:field:type` can further have a codec/Go representation different from another String alias. Do not use only `Kind()`/unaliasing to call these compatible. The existing type/alias ownership must select the representation while the existing constraint owner retains both authored guarantees; add explicit related-alias, unrelated-alias and custom-codec controls.

For defaults, bases A and B both declaring `n Int` with defaults 1 and 2 are equally near. Current `Merge` happens to pick B; that is the defective replacement path, not a documented priority. Recommend a design error naming A.n and B.n unless a local default disambiguates and satisfies every inherited constraint. Equal defaults require semantic declared-type equality, not raw reflected representation equality. For metadata, `Meta` appends values and different readers use first/last/value-list semantics. A blanket map merge or generic “local wins” rule can corrupt tags, custom codecs or ordered examples. Preserve explicit overlay provenance and apply the already owning key contract; reject unresolved identity conflicts with both sources.

The empty-domain policy is deliberate existing behavior, not an inference: docs explicitly admit empty inherited bounds; `AliasContracts.lean` near `numericCompositionControls` includes contradictory intervals as accepted contracts, while `TestEffectiveValidationConjoinsPatternAndFormatClauses` and `TestNamedPatternAndFormatClausesUseConjunction` accept conjunctions such as Email plus Hostname with no supplied value. Root limits this ticket to **statically detectable contradictions in Extend redeclarations** while preserving existing named-alias admission. This is an explicit structural composition admission boundary, not a different conjunction algorithm. Reuse the owner's computed bounds and enum membership; do not claim general pattern/format satisfiability or treat every empty enum representative set as proof of an impossible contract. Cross-syntax consistency remains a recorded policy follow-up. If the existing owner cannot preserve this boundary cleanly, report that before changing alias semantics.

## Planned files and implementation order

Current changes are this readiness file, `expr/attribute_composition_test.go`, and `expr/tla/effective_design/` only. After the next root approval:

1. Add failing tables in `expr/attribute_composition_test.go` (new) and `expr/grpc_nested_tags_dsl_test.go`; extend `dsl/effective_constraints_test.go`, `expr/method_extended_type_test.go` and `expr/root_runner_test.go` for DSL and lifecycle ownership. Run against unchanged source and record exact failures for #755 and #757.
2. Add `expr/tla/effective_design/EffectiveDesign.tla`, legacy-overwrite, legacy-late-validation, shared-base-mutation and corrected configurations, plus README correspondence. First reproduce current failures, then revise the model. State bounds: small field/type domain, one diamond, ordinary recursion distinct from inheritance cycles, two method consumers in either order, and explicit local overlay versus inherited copied value in both DSL declaration orders. Invariants: authored refinement retained, inherited guarantee preserved, base/other occurrence unchanged, complete shape validated before generation, same completed result under repeated preparation. It proves only that finite lifecycle model.
3. Implement explicit-overlay capture at `dsl/attribute.go` / `dsl/attribute_copy.go`, then `expr/attribute_composition.go` (new), adapt `expr/attribute.go`, `expr/attribute_validate.go` and `expr/union_branch_names.go` (or move Root.Prepare into a focused root-preparation file). Remove the superseded Extend selection in finalization and `effectiveGoFields`; update `expr/go_field_names.go` / `expr/service.go` consumers.
4. Extend the existing effective constraint graph in `expr/effective_constraints.go` / `expr/value_occurrence.go` and relevant copy/source files **only as required** to retain refinement input provenance. No synthetic public named types or alternative semantic engine. Exercise immutable snapshot/copy behavior. If existing Lean named-ancestry statements need to change, update `expr/lean/value_projection/ValueContract/AliasContracts.lean` and its concrete dependent modules, README, theorem manifest and correspondence ledger before relying on the revised claim; do not assert old linear-chain proofs cover a new DAG automatically.
5. Add or extend rendered coverage in `http/codegen/testdata/extended_method_types_dsl.go`, `http/codegen/extended_method_types_test.go`, `http/codegen/effective_constraints_integration_test.go`, and `grpc/codegen/effective_constraints_compile_test.go`. Reuse `expr/testdata/endpoint_dsls_03.go:GRPCEndpointWithExtendedTypes` for numbered inheritance. Add a focused JSON-RPC shared-contract control only if its effective-value extraction seam differs. Do not patch transport renderers to restore constraints.
6. Update `dsl/result_type.go` GoDoc and `docs/dsl-reference.md` with accepted/rejected examples, regeneration implications and the explicitly approved empty-domain policy. Send consumer/framework skill and tracker deltas to root. Preserve source provenance in `eval.ValidationErrors`; fields without a DSL callback require a containing type/method fallback, not a made-up field line.

File list is a bounded estimate, not approval to expand into separate validation tickets. Unexpected representation changes or conflicting owners return to root before broadening scope.

## Direct acceptance matrix

Use table-driven tests for one/two bases, one/two inheritance depths, a diamond, local-before/after-Extend DSL order, both method traversal orders, all four method positions, plain versus customized use, explicit versus automatic result views, an unrelated Reference control, named aliases and recursive field references. Assertions must cover values and ownership: distinct mutable field pointers, validation slices, metadata slices, defaults/example snapshots, base shape before/after, occurrence origins, repeat preparation and immutable EffectiveConstraints snapshots.

For #757 cover ordinary/streaming payload, result and error message roots; inherited nested object/array/map/union fields; a number reused in different messages is valid; metadata-excluded fields are not protobuf members. Check errors have a real source filename and positive line, field-number text, both field paths and declaration context, with no generator panic. Do not claim exact Field-call attribution where the evaluator only retains the containing DSL callback.

Mapped-name evidence in this slice is structural identity: refined inherited `struct:field:name` / JSON field names feed the existing shared validators and produce source-attributed errors when two effective fields collide. The #756 request/response header/query/cookie matrix is expressly deferred to its separate owner.

## Bounded generated evidence and commands

Run new direct regressions first on unchanged source. The implementation acceptance commands are:

```sh
go test ./eval ./expr ./dsl -count=1
go test ./expr -run 'TestEffectiveConstraints|TestExtended|TestGRPCNestedTag|TestRecursiveRPCFieldTags|TestPrepareValidateFinalize' -count=1
go test ./codegen/service -count=1
go test ./internal/openapiimport -run 'TestRenderSpringAllOfInheritance' -count=1
go test ./http/codegen -run 'TestExtendedMethodTypesGenerated|TestEffectiveConstraintsGeneratedHTTPModule' -count=1
go test ./grpc/codegen -run TestEffectiveConstraintsGeneratedGRPCModule -count=1
LOOM_DIR="$PWD" make test-testdata-compile TESTDATA_RUN='TestDesigns/(http/codegen/(ExtendedMethodTypesDSL|ExtendedAliasedMethodTypesDSL|ExtendedResultMethodTypesDSL|PayloadExtendedValidateDSL)|expr/GRPCEndpointWithExtendedTypes)$'
git diff --check
```

`internal/testdatacompile/catalog_test.go` confirms IDs are `<domain>/<exported-name>`; verify those selected names still exist before executing the bounded compile regex; unmatched tests are not evidence. Preserve exact Go 1.27.2 and Makefile-pinned protoc/plugin versions. Source selection must point to this unpushed worktree. If affected OpenAPI output changes, run structural assertions, `libopenapi` parsing and the pinned Vacuum specimen gate (`make openapi-contract`). The full exported-design corpus is not an acceptance gate.

For source comparison, use the managed historical baseline to reproduce original defects, but compare final intended changes against the **immediate integration parent** as well; earlier delivered auth/error/projection changes are not this ticket's differences. Record each revision and dirty-diff hash. Use the existing temp-module generation pattern in `internal/testdatacompile/compile_test.go`, not a new corpus harness. A task-local driver may preserve the selected outputs that the test harness normally removes:

- Same module path `example.com/specimen`, identical frozen wrapper/design inputs, dependency versions, generator flags and toolchain in separate temporary directories.
- Generate each accepted selected design once with the immediate parent, once with candidate, and a second time with candidate in a **fresh process**. Include one checked-in fixture only if its design uses the changed composition; otherwise record that none is affected. Do not regenerate unrelated SSE fixtures.
- For each module run `loom gen example.com/specimen/design`, `loom example example.com/specimen/design`, `go mod tidy`, `go build ./...` and `go vet ./...`. `loom gen` must include protoc for the numbered gRPC case. Use `internal/testprocess` ownership for driver subprocesses.
- Compare complete generated file path sets and bytes (`diff -ru` / per-file SHA-256), including schema files. Do not normalize body text, ordering, imports or schemas. Compare and record `gen/loom.json` separately only if its version/design fingerprint legitimately differs across revisions; do not erase it from the evidence. Two candidate processes must be fully byte-identical, including manifest.
- For new refinement fixtures, run the **same fixture source** against parent and candidate: expected differences are retained local description/default/example/metadata, conjunctions in validators/schema, inherited requiredness and corresponding default behavior. No field/type-name churn, order drift or lost base constraints is intended. Existing no-redeclaration controls should be byte-identical. List actual file-by-file differences rather than pre-approving all changes in these categories.
- For rejected fixtures, use generation only to verify design-error exit before generated output and exact stable diagnostic content; compiling invalid output is not an acceptance target. Historical collision should demonstrate acceptance then codegen failure, corrected collision must fail evaluation.
- HTTP generated runtime tests send below/inside/above numeric values and missing inherited-required fields, checking rejection before service invocation and acceptance payload state; gRPC generated code compiles and validators enforce the same refinements. Default-view and plain-method controls check no cross-occurrence widening.

Run the new TLA configs using the pinned existing TLA tools described in `expr/tla/value_projection/README.md`, with each `-metadir` outside the checkout. Legacy controls must fail specifically the intended invariant; corrected configuration must pass with recorded bounds/state counts. If Lean claims change, run `make value-contract-proof` and the affected `internal/valuecontract` comparison groups per their README, keeping test/proof claims distinct.

## Reviewed readiness delta: authored state and metadata owners

`dsl/presence.go:Nullable` sets only `AttributeExpr.Nullable=true`; there is no public non-null reset DSL. `expr/presence.go:IsNullable` follows named ancestry; `AllowsNull` also includes intrinsic Any nullability. The local-overlay descriptor must retain **whether Nullable was authored**, separately from the effective bool copied from a base. Omission inherits the complete base contract; local true cannot widen an inherited non-null guarantee. When peer bases disagree, their accepted-value conjunction retains non-null (the stronger guarantee); an explicit local Nullable cannot override that. Named aliases retain their existing semantics. Legacy presence metadata is normalized by `AttributeExpr.Prepare`; do not mistake an inherited normalized true for a local opt-in.

Representation proposal: capture an explicit attribute overlay during DSL execution, including local default presence, Nullable opt-in, metadata append/removal operations and the source callback. Its inherited template is a separate input. The effective output remains an ordinary owned AttributeExpr plus provenance consumed by the existing effective-constraint owner. Do not infer authored state by comparing final values: a local `Default(1)` equal to an inherited 1 still matters when another base supplies 2; `RemoveMeta` also needs a removal marker. Do not change Reference's selected-template behavior. This is an internal DSL/expr seam, with no new consumer DSL.

Metadata decisions are per existing key owner, not one generic map overwrite. All effective slices are detached. The bounded policy inventory is:

| Keys / data | Current owner semantics | Composition rule |
| --- | --- | --- |
| Description, Title, Docs, authored examples | Attribute fields; example selection retains authored ordering/source | Explicit local documentation replaces inherited documentation; missing values inherit. Conflicting equally near peer documentation with no local selection must retain source evidence and diagnose rather than choose base order |
| `struct:field:name` | `internal/naming.GoifyAttribute` uses first value | A local setting can supply an otherwise absent explicit override; differing explicitly inherited overrides conflict. Compare selected field identity, not appended slice order; shared Go-name validation checks distinct fields |
| `struct:field:type` | `codegen.GetMetaType`: positional Go type, import path, import alias | Preserve as an atomic codec/representation tuple; incompatible explicit tuples or removal of inherited representation reject |
| `struct:field:proto` | `grpc/codegen/protobuf.go`: positional proto type/import/Go name/Go import; expr treats it as opaque for field traversal | Atomic representation tuple; incompatible overrides reject before gRPC analysis; no generic slice append across layers |
| `rpc:tag` | `AttributeExpr.FieldTag` selects last value; union tag expansion derives consecutive tags | Same-field inherited identity must agree; different effective fields are checked by existing per-message uniqueness owner |
| `struct:tag:json`, `struct:tag:json:name` | `attributeJSONTag` joins supplied pieces with comma; full tag takes precedence over name-only, then shared tag/element helpers interpret it | Compare effective wire identity/options as one contract. Never append unrelated layer tags; do not let a local name-only key silently lose to inherited full tag |
| `proto:tag:json` | `grpc/codegen/protobuf.go` uses first nonempty value for protobuf JSON name | Preserve compatible explicit wire identity; incompatible explicit values reject |
| `struct:type:name`, `struct:pkg:path`, `struct:name:proto`, `oneof:typename` | `UserTypeExpr.Name` first value; `NameScope`/attribute package preparation last package; protobuf name validation requires one name; union DSL uses first explicit typename | Declaration-owned identity, not field-documentation overlay. Retain named declaration owner; no synthetic renaming to hide conflicts |
| `oneof:type:field`, `oneof:value:field` | `dsl/attribute_oneof.go` validates and installs discriminator/value keys on the union | Semantic wire identity; retain original owner and reject conflicting inherited mappings |
| `openapi:nullable` | Legacy normalization into semantic presence in Attribute.Prepare | Interpret through the presence owner before composition; never treat it as documentation |
| `openapi:additionalProperties` | Expr value resolution and schema target use last value for closed/open objects | Contract predicate; cannot reopen an inherited closed object by metadata overlay; keep its existing semantic owner |
| `openapi:contentEncoding`, `openapi:contentMediaType` | Expr schema/value representation and untagged validation; OpenAPI schema owner | Representation annotations, not prose. Differing explicit inherited representations diagnose; do not silently change the byte/string codec contract |
| `struct:error:name`, `security:username`, `security:password`, `security:token`, `security:accesstoken`, `security:apikey:<scheme>` | Error/security expr selectors interpret marker presence | Semantic role markers must survive inheritance; removal/reassignment cannot be treated as documentation cleanup |
| Unknown extension metadata | `Meta` appends ordered values; `RemoveMeta` removes a key; consumers own interpretation | Preserve a single inherited source or identical peer values; explicit local append/removal retains existing key operations. Conflicting peers without local disambiguation diagnose; do not assume arbitrary keys are harmless documentation |

This inventory fixes scope and comparison obligations; it does not authorize a new cross-framework metadata policy engine. If a key requires semantics outside that owner or a public override would weaken an inherited guarantee, report the precise example before changing it.

## Red/model evidence checkpoint

The final focused command `go test ./expr -run '^TestExtendComposition' -count=1 -v` returns exit 1 against unchanged production source: 10 intended failing leaf cases, 3 passing preservation controls. All tests state their obligation; this is not a Cartesian transport matrix. The bounded TLA model has four expected-failing configurations and one passing corrected configuration (246 distinct states). See `expr/tla/effective_design/README.md` for exact counts, invariant names, limits and commands. Copied-overlay is a rejected-repair control, not a claim about current last-base-wins selection. An initial parallel Java temp-file parsing race was rerun sequentially; it is explicitly excluded from model evidence.

Before production, root reviews this delta and evidence. After implementation, full package gates are eval/expr/dsl/service, with the selected affected transport tests and bounded generation comparisons above; all transport package suites are not additionally required merely because the shared owner changes.

## Handoff and cleanup

After validation, freeze candidate source and SHA-256 of the exact diff. Keep everything uncommitted for root transfer and a fresh independent review against the intended main parent. Fixes after review require affected checks and re-review. Summarize commands, input hashes, intended output differences and limitations durably, then remove only this task's temporary modules, TLC state/logs and compiled Lean output after confirming no active process depends on them. Preserve shared toolchains/caches and all unrelated work.


## Implemented scope and delivery evidence

The owner is #755, with #757 as the concrete late-validation witness. The
production seam is unconditional `RootExpr.Prepare` composition, after promoted
union naming and before consumers prepare transport mappings or validate field
sets. Finalization still owns its existing dependent defaults/views/transport
work; the evaluator phases were not globally reordered. The superseded late
Extend merge, interceptor merge, body merge fallback and separate Go-field
simulation are removed. Reference template behavior stays in its existing owner.

`dsl.Attribute` keeps authored overlays separate from Extend templates and
retains the original declaration callback for errors. `expr/attribute_composition.go`
owns a detached field graph; `value_occurrence.go` carries local and inherited
constraint inputs into the existing EffectiveConstraints owner. Default
selection/equality lives in `effective_default.go`: equivalent admitted values
coalesce in the effective occurrence, explicit local defaults resolve peer
choices, and all selected defaults satisfy the complete contract. Identity and
codec comparisons share actual naming/tag accessors and the tuple parser moved
to `internal/encodingmeta`; generators use that same parser.

Concrete accepted/rejected boundaries are in the public Extend documentation.
Omitted Nullable inherits; an explicit widening or nullable named descendant of
an inherited non-null type rejects, naming the participating source, because the
existing carrier cannot express that intersection. Explicit nominal ancestry is
required for named substitution. Only statically decidable Extend bounds, length
and finite-enum contradictions are rejected; general pattern satisfiability and
ordinary named-alias empty-domain policy are unchanged. Cross-syntax empty-domain
coherence remains a separate follow-up rather than an implicit policy expansion.

Two implementation counterexamples supplement the model: an Extend-only inline
payload can initially hold the shared Empty singleton, and retained constraints
can otherwise borrow mutable original base data. The direct regressions cover
both Empty method orders and later source mutation. The composer allocates an
owned object for Empty and snapshots retained parent inputs. The TLA README
records these correspondence limits; no claim extends the bounded model to Go
cloning or the older linear-alias Lean theorems.

Direct coverage combines the new named-obligation tests with existing package
controls, rather than multiplying every obligation across transports:

- `TestExtendComposition*`: both typed DSL overlay orders, inherited requiredness,
  invalid/refined defaults, nullable inheritance/widening, related/unrelated named
  types, nested collection constraints, enum intersection, RPC collisions with
  source locations, peer metadata/default resolution, diamond/recursive-member/
  inheritance-cycle behavior, Empty and base ownership, and idempotent repeated
  preparation.
- Existing `TestExtendedMethodTypesOwnTheirShape`, `TestExtendedResultViewSelection`,
  `TestPrepareValidateFinalize`, Go-field-name Extend/Reference cases, recursive
  RPC tag and message-role tests retain all four method positions, plain-use/view
  isolation, lifecycle, mapped structural names, nested traversal and metadata
  exclusions. #756 transport mapping policy remains out of scope.
- One meaningful `EffectiveCompositionDSL` specimen supplies shared HTTP/gRPC and
  OpenAPI tests: count 0..100, required count/label, retries 0..10 and default 5.
  HTTP checks missing/invalid fields before service invocation and accepted
  payload state; gRPC checks compiled validator bounds/requiredness and default
  construction; OpenAPI checks JSON/YAML structure, libopenapi and pinned Vacuum.

Final acceptance uses Go 1.27.2 darwin/arm64, protoc 36.2,
protoc-gen-go 1.36.12 and protoc-gen-go-grpc 1.6.2. Commands:

```sh
go test ./eval ./expr ./dsl ./codegen/service ./internal/encodingmeta -count=1
go test ./internal/openapiimport -run '^TestRenderSpringAllOfInheritance$' -count=1
go test ./codegen -run '^Test(GetMetaTypeImports|GatherAttributeImports)$' -count=1
LOOM_DIR="$PWD" go test ./http/codegen ./grpc/codegen -run '^(TestExtendedMethodTypesGenerated|TestEffectiveConstraintsGeneratedHTTPModule|TestEffectiveCompositionGeneratedHTTP|TestEffectiveConstraintsGeneratedGRPCModule|TestEffectiveCompositionGeneratedGRPC|TestProtoMetaMessageName|TestProtoStructNameUsesMetaMessageName|TestProtoMetaGoTypeName|TestProtoFilesStructMeta|TestStructMetaProtoTypeRefs)$' -count=1
LOOM_OPENAPI_CONTRACT=1 go test ./http/codegen/openapi/v3 -run '^TestEffectiveCompositionOpenAPI$' -count=1
make lint-filesize lint-gofmt lint-namescope lint-json-v2
git diff --check
```

The full four transport codegen suites and full exported-design corpus were not
run. Checked-in SSE fixture designs contain no Extend usage, so no checked-in
fixture regeneration was required. The pre-existing JSON-v1 import in
`jsonrpc/codegen/recursive_projection_test.go` is repaired separately on main;
this worktree does not duplicate that unrelated repair.

### Exact output comparison

The task-local driver follows `internal/testdatacompile` temp-module construction
and uses `internal/testprocess` for all subprocesses. It builds a matching source
CLI for each revision, uses the identical executable basename `loom`, and runs
`loom gen`, `loom example`, module tidy, build and vet. Parent and candidate each
compile once; the independent candidate repeat only regenerates identical inputs.
No output normalization is used. The six specimens and obligations are:

| Specimen | Obligation |
| --- | --- |
| ExtendedMethodTypesDSL | Ordinary customized method ownership |
| ExtendedAliasedMethodTypesDSL | Named alias ownership |
| ExtendedResultMethodTypesDSL | Result type/default view ownership |
| PayloadExtendedValidateDSL | Existing inherited validators |
| GRPCEndpointWithExtendedTypes | Existing numbered gRPC inheritance/protoc |
| EffectiveCompositionDSL | Newly retained constraints/defaults across HTTP, gRPC and OpenAPI |

The baseline here is the clean immediate worktree parent `ce3f6a733f19de30b50986f86e9ce4cef06e48cb`,
extracted by `git archive`; it includes earlier delivered auth/error/projection
work. The historical `4535a403` correctness baseline remains read-only and is not
mixed into output deltas. Root must repeat the bounded parent comparison against
the current integration parent, using matching source/CLI (main now includes the
#670 artifact-return API). The driver calls the CLI and assumes no generator
return type.

The five existing controls preserve all generated code, schema and example
bytes. Only `gen/loom.json` design digests differ because composition now records
an effective validation carrier before analysis; those differences are retained
in the comparison. The new specimen changes exactly these files:

- `gen/grpc/cli/composition/cli.go`
- `gen/grpc/composition/client/cli.go`
- `gen/grpc/composition/client/types.go`
- `gen/grpc/composition/server/types.go`
- `gen/http/cli/composition/cli.go`
- `gen/http/composition/client/cli.go`
- `gen/http/composition/client/types.go`
- `gen/http/composition/server/types.go`
- `gen/http/openapi.json`
- `gen/http/openapi.yaml`
- `gen/loom.json`

These retain maximum 100 for count, maximum 10 for retries and local default 5
instead of inherited 2 in validators, constructors, CLI examples/help and schema
constraints/examples. Service declarations and protobuf shape are unchanged.
Every candidate/repeat output, including manifest, must match byte for byte.
An initial comparison using different CLI basenames was rejected as mismatched
inputs and rerun; its command-comment differences are not acceptance evidence.

Task-owned output/driver/hash records stay outside the deliverable checkout until
root transfer and independent review finish, then are removed with no active
process depending on them. Root receives the exact path/hash manifest, binary
diff hash and driver with the uncommitted delivery. Source changes after this
freeze invalidate only their affected checks and require re-review.


### Gate results at candidate freeze

All named package and targeted transport commands above passed. The final full
package run was eval 0.489s, expr 1.279s, DSL 0.302s, service 9.854s and encodingmeta
0.163s. The targeted HTTP/gRPC run passed in 26.924s/10.455s. The OpenAPI specimen
passed libopenapi plus pinned Vacuum in 88.842s; the subsequent default-owner
change preserved those specimen bytes. The importer and import/metadata helper
controls passed. The added repeated-preparation regression passed separately.

The first file-size run found the expanded effective-constraint file above the
repository's 700-line gate. Moving default selection to `effective_default.go`
reduced it to 679 lines; the final filesize, gofmt and NameScope gates passed.
The JSON-v2 lint reports only the previously identified baseline import in
`jsonrpc/codegen/recursive_projection_test.go`, already fixed on main. No unrelated
repair is part of this delivery. `git diff --check` passes.

The final source comparison and independent generation repeat preserve the file
counts 23/29/24/21/20/34 for the six specimens in table order. All 151 files match
between candidate and repeat, including manifests; the five existing manifests
and the eleven enumerated new-specimen files are the only parent differences.
The compiled parent/candidate outputs and per-file SHA-256 comparison are kept
for root's fresh review. The helper and raw logs are task-owned temporary
artifacts, not committed framework infrastructure.

### Integrated maintainability repair plan

The immediate integration parent is `651372ce6ff05ed1db495369fd51d48df0c0ec75`.
Integrated pinned golangci found three maintainability-index failures: `Attribute`
(34), `designComposer.object` (33), and `composeAttributeMetadata` (33). These
are acceptance blockers for this same repair, not reasons to suppress the gate.
After the frozen six-specimen comparison completes, the validation agent will:

1. Extract private attribute construction from parent admission and insertion.
2. Extract ordered inherited-field collection from object composition.
3. Extract one inherited metadata-key decision from traversal and publication.
4. Rerun the affected package and transport checks and pinned scoped lint.
5. Regenerate the six specimens with the revised candidate and compare every
   output byte to the already compiled frozen candidate, retaining parent
   evidence. No policy, accepted/rejected contract or generated-output change
   is intended. The final integrated diff requires fresh independent review.

### Integration validation on current main

The integrated parent is `651372ce6ff05ed1db495369fd51d48df0c0ec75`.
All 43 files first matched the frozen delivery hashes. Parent and candidate
used the same six specimen inputs, module name, CLI basename and toolchain:
Go 1.27.2 darwin/arm64, protoc 36.2, protoc-gen-go 1.36.12 and
protoc-gen-go-grpc 1.6.2. The reused driver now pins the pushed module version
`v1.10.0-alpha.9.0.20261012181439-651372ce6ff0`, with explicit local replacement
and `LOOM_DIR` selecting each compared revision. Parent source came from a
read-only `git archive`; no old generator return API was assumed.

Each parent and frozen candidate specimen generated, tidied, built and vetted.
All 151 candidate/repeat files matched exactly. The five existing controls
still differ from parent only in `gen/loom.json`; the new composition specimen
has exactly the eleven intended differences listed above. No normalization
removed code, schema, examples or manifests from comparison.

The integrated pinned linter required three private responsibility extractions:
`attributeTemplate` selects the Reference template or Extend type hint;
`inheritedCompositionFields` gathers inherited members in their existing order;
`composeInheritedMetadata` owns one inherited metadata-key decision. These
preserve policy, ordering, mutation and diagnostic behavior. An initial broader
attribute-construction extraction was rejected by `TestTooManyArgError`: adding
a stack frame changed a security-field diagnostic from its DSL name to `Field`.
The final extraction keeps argument parsing and DSL execution at their original
call depth; the complete eval suite verifies the corrected diagnostic contract.

After these extractions, all 151 generated files matched the already compiled
frozen candidate byte for byte, including manifests and OpenAPI. A new isolated
repeat also matched all 151 files. Parent and identical generated-module builds
were reused; only invalidated generation and affected behavior gates were rerun.
The prior libopenapi and pinned Vacuum result remains applicable to the identical
new-specimen JSON/YAML bytes. Model sources/configurations were unchanged, so
the earlier bounded model evidence remains applicable within its stated limits.

Final full package results: eval 0.594s, expr 0.843s, DSL 0.436s, service 6.187s,
encodingmeta 0.153s. The targeted HTTP/gRPC command listed above passed again
in 17.567s/5.523s. Importer and metadata-import controls passed. Integrated
OpenAPI structural/libopenapi/Vacuum checks passed in 14.873s. Scoped pinned
`golangci-lint run ./eval ./expr ./dsl ./codegen ./grpc/codegen
./internal/encodingmeta ./http/codegen/testdata` reports zero issues. File-size, gofmt, NameScope and JSON-v2 gates pass
(the size gate reports warnings only). `git diff --check` passes, including the
three root-owned shared guidance changes.

Temporary retained review evidence:

- `/tmp/loom-effective-integrated-output`: compiled parent/frozen-candidate and
  first independent repeat.
- `/tmp/loom-effective-integrated-repaired-output/comparison.json`: final
  parent/current/repeat SHA-256 `30ddb86c42f56ad802303fe6689856cfcc90e73250c58ebc1f2c6fff08fa99e0`.
- `/tmp/loom-effective-integrated-repaired-output/frozen-comparison.json`:
  output identity after extraction, SHA-256
  `7ae452096c2de1f934b3524da4cda6680ad7d17d19b3e6447499695a36220574`.
- `/tmp/loom-effective-integrated-maintenance.patch`: three-file extraction
  against the frozen source, SHA-256
  `631ed32a041c4605f18b470d35155e5e0637724ac5b416a3b1fd3f9d3d37de23`.
- `/tmp/loom-effective-integrated-driver.go`: retained driver, SHA-256
  `09650a98406e262514bb8b3d6a29ebb56b1ce7faeca5bf341202b5a1392d8f0f`.

The temporary driver was removed from the checkout. Raw evidence stays outside
it until independent review finishes, then task-owned scratch must be removed
after checking for active users. No broader corpus or unrelated fixture sweep
was run. The final exact integrated diff still requires independent review.

### Independent review repair plan

Independent review found two medium defects before commit. A mixed graph where
Base.next references Node and Node extends Base is legitimate recursion, but
composition confused it with an inheritance cycle or copied an unfinished Node,
depending on declaration order. Peer full JSON tags and name-only tags also
bypassed per-key comparison. The repair sequence is:

1. Reproduce both recursive declaration orders and both peer tag orders directly.
2. Model graph preparation versus snapshot ordering, including actual inheritance
   cycles and the rejected repair that merely suppresses cycle diagnostics.
3. Separate inheritance-only cycle checks from member traversal. Retain named
   references during composition, then detach snapshots only after every named
   shape is complete. Keep owned occurrence metadata throughout.
4. Compare complete resolved JSON tag identity/options across peers through the
   shared tag accessor; retain compatible and rejected cases in both orders.
5. Run direct packages and bounded generated compilation for the new recursive
   specimens, repeat affected output comparisons, and obtain independent re-review.

### Independent review repairs and final integrated evidence

Both medium findings are fixed at their owners. Graph discovery resolves named
references and checks only inheritance edges for cycles. Composition keeps named
references live until all effective shapes are complete, then prepares every
replacement before publishing any. Structural copies bind recursive back edges
to the enclosing named declaration; retained local and parent constraint inputs
are fully detached. This distinction preserves named identity and occurrence
ownership together. Merely ignoring a mixed-cycle diagnostic, copying early,
or making another same-ID node were rejected repairs.

The original reviewer reproductions failed before the repair in
`/tmp/loom-effective-review-red.log`. Direct tests now cover Base-first and
Node-first self and mutual recursion, completed fields, actual Extend-cycle
rejection, separate derived metadata, and retained constraint inputs after
mutating the canonical named field. The extra identity regression failed before
the owner-binding change (`/tmp/loom-effective-recursive-identity-red.log`). The
bounded recursive model reproduces mixed-cycle rejection (3 states), unfinished
snapshot (7 states) and complete-but-unbound structural identity (7 states), all
with exit 12. The corrected model passes three invariants over 7 states. Its
binding flag abstracts identity; concrete pointer/mutation behavior remains a
Go-test obligation. The existing EffectiveDesign model was unchanged.

JSON identity/options now resolve through `internal/encodingmeta.JSONTag`,
which owns the existing renderer normalization; codegen delegates unchanged.
The shared explicit-presence representation facts also live in that package.
Composition compares peer and local tags under effective containing requiredness
and representation, after representation metadata merges. Optional full `one`
and name-only `one` conflict; optional full `one,omitempty` and name-only `one`
agree. Required `one` agrees with name-only `one`, while required
`one,omitempty` conflicts. Nullable optional carriers normalize to `omitzero`.
These are the accepted resolved-contract rules, not an authored partial-contract
policy. The existing requiredness owner now traverses retained Extend parents,
including a grandparent whose Bases have already been consumed. The missing
case failed eight rendered-tag controls before this correction. The 24-case
rendered matrix covers both peer orders and local/inherited comparison, exact
rendered bytes, nullable representation and inherited requiredness.

Final source used the same parent, pushed module pin, explicit local replacement
and toolchain recorded above. Seven specimens generated in independent candidate
and repeat processes: the original six plus RecursiveEffectiveCompositionDSL.
All 185 files match exactly across those two processes. The recursive specimen's
34 files match parent except `gen/loom.json`, including schema names, examples,
self references and all generated Go. Five original controls retain their
manifest-only changes except ExtendedResultMethodTypesDSL, which also changes
`gen/fields/service.go`: show/watch view-to-service conversion now directly
assigns the inherited required `Left` field. Its existing view validators reject
nil before conversion. The composition specimen retains exactly the earlier
eleven intended parent differences. No output normalization was applied.

The final result-type specimen was tidied, built and vetted again. Other
identical generated Go reuses the compiled parent/candidate evidence; independent
repeat compilation would prove no additional obligation. The recursive HTTP
compile test passes, and the bounded driver had compiled its HTTP/gRPC module
before the binding repair; all its Go bytes remain identical. Both composition
OpenAPI specimens pass structural assertions, libopenapi parsing and pinned
Vacuum checks (28.118s); the recursive test asserts the exact RecursiveNode and
ReverseNode component names and self references.

Final affected full packages pass: eval 0.458s, expr 0.565s, DSL 0.220s, shared
codegen 18.599s, service 10.275s, encodingmeta 0.143s. The targeted HTTP/gRPC
command above, including recursive compilation and extended result runtime
coverage, passes in 19.843s/5.922s. Scoped golangci-lint reports zero issues;
file-size, gofmt, NameScope and JSON-v2 gates pass (size warnings only).
`git diff --check` passes. No exhaustive corpus was run.

Retained final review evidence:

- `/tmp/loom-effective-owner-output/comparison.json`: complete parent/current/
  repeat file hashes, SHA-256
  `afd0ed521a7c4eac15526182db612765d1c7fab3c9c7470ea255777c82327972`.
- `/tmp/loom-effective-owner-output/compiled-candidate-differences.json`: exact
  differences from already compiled candidates, SHA-256
  `79e366af3f3b3520c6edcf08ead67f1148bb536cfdeb208c1df6c9bf64bf3eda`.
- `/tmp/loom-effective-recursive-owner-tlc`: final checked model logs and state.
- `/tmp/loom-effective-owner-{packages,transports,openapi,lint,fastlint}.log`:
  final scoped gate results.
- `/tmp/loom-effective-review-driver.go`: the reused seven-specimen driver.

The temporary driver has been removed from the checkout. All checker state,
generated specimen modules and raw logs stay outside the repository. Retain them
until independent re-review finishes, then remove task-owned scratch once no
process or reviewer uses it. Root-owned shared guidance is included in the final
path manifest; unrelated master trackers and auth/gRPC drafts are excluded.
