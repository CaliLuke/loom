# Proof and production correspondence ledger

Status: **M1, #569, #570 and #574 are independently reviewed and pushed. The #571 effective-alias implementation has passed its model and tested production-correspondence gates within the boundaries recorded below.** The reviewed contract is
[value-contract-design.md](../../../roadmap/value-contract-design.md).
The source-history reference is `f5b786b39675e2b5c1466f04f3301b7b337779b6`.
The word “proved” below applies to the Lean statement, not to handwritten Go,
generated programs, all JSON strings, or any codec implementation.

The detailed #574 investigation checkpoints below preserve what each local
review established. Their pending whole-ticket gates were subsequently completed
in `9e6b9121`; the [final #574 evidence](../../../internal/valuecontract/BYTE_SCHEMA.md)
records the repository gates, registered 72-probe comparison and final-source
499-case comparison. That delivery does not discharge separately listed general
alias/key extraction obligations or turn tested Go correspondence into a theorem.

## Established statements

All names below are in `ValueContract.Legacy`. `required-theorems.txt` enumerates
the individual obligations, and the gate prints their transitive axiom sets.

| Theorem names | Status and domain | Production seam / additional evidence |
| --- | --- | --- |
| `erasedBranchesIndistinguishable`, `legacyBranchLoss` | **Proved**: two concrete selected branches erase to one payload; first-branch reconstruction changes the second identity | `expr/types_union.go` source selection and `expr/example_canonicalization.go` reconstruction; #566 nested-branch probe supplies independent generated evidence |
| `legacyByteReinterpretation`, `byteStringDecodes`, `byteEncodingWitness` | **Proved** for the concrete `hi`/`aGk=` specimen codec; literal `hi` does not decode as the original bytes | Historical OpenAPI byte conversion; current authored-byte CLI bug in `codegen/cli/conversion.go`; #566 actual generated builder/decoder probe |
| `byteTextWireCollision` | **Proved**: the same `Wire.text "aGk="` decodes as Bytes and String | Untagged OpenAPI matching in `http/codegen/openapi/internal/ir/document_examples.go`; candidate same-wire uniqueness is proved below; actual-decoder correspondence remains #570 |
| `legacyLengthRejectsValidBytes`, `legacyLengthAcceptsShortBytes` | **Proved** concrete mismatches between decoded byte count and ASCII wire-string length under `boundaryCodec`; neither direction can use copied length bounds | `expr/json_schema_inline.go`, OpenAPI IR `analyzer.go`, and generated decoded-length validation. #574 delivered the repaired arithmetic, grammar and tested generated-server/schema correspondence; see the final byte-schema evidence below. This is not a theorem about arbitrary runtime codecs |
| `canonicalByteEnumAgreement`, `aliasSchemaDecoderMismatch` | **Proved**: canonical `aGk=` meets both enum predicates, while String `aGl=` fails the canonical byte-schema enum but decodes to the allowed byte value `hi` | #569 separately requires emitted-schema validity and actual decoder branch/observation preservation. #570 tests real codecs and #574 corrects the length/grammar schema; no enum broadening or String normalization follows from this witness |
| `legacyAuthoredReplacement`, `legacySourceReplacement` | **Proved**: concrete unconditional synthesis replaces value and source | `grpc/codegen/client_cli_example.go`, `service_data_analysis.go` and transport-copy source selection; #566 probes with and without unions |
| `selectedBodyObservation`, `selectedBodyRepresentable`, `observationIsNotWholeService` | **Proved**: a body selection drops its service header, retains bytes, and remains representable under the specimen codec | `expr/http_body_types.go` and HTTP target plans; actual plan extraction remains a tested implementation boundary |
| `visibleBranchRetained` | **Proved**: a visible second branch is retained by the independent observation judgment | `expr` semantic identity and transport plans; candidate nested preservation is proved below; production mapping remains #570 |
| `nullDistinctFromAbsent`, `nullObservation`, `absentObservation`, `omissionIsFieldLocal` | **Proved**: null/absence are distinct, and only the selected field omission rule collapses empty collections | `expr` and generated Optional/Nullable/omitempty contracts; candidate presence relations are proved below; production JSON/protobuf matrix remains #570 |
| `duplicateEntriesRetained` | **Proved**: the example raw entry list retains both integer `1` and string `1` keys | `internal/jsonkey`, enum/default normalization and #456; spelling/collision rejection is not proved here |
| `finiteRecursiveValue` | **Proved**: a recursive declaration has a finite null-terminated structural inhabitant | `expr` recursive type/example handling; candidate resolution and cycle outcomes are proved below; production correspondence remains #570 |

## Foundation vocabulary and its limits

This table describes the retained M1 model only. The candidate relations and
checked executable correspondence are listed separately below.

| Model owner | Production owner | Current boundary |
| --- | --- | --- |
| `RawValue`, `Scalar`, `Role`, `Source`, `Supplied` | `expr` authored inputs, example generator, enum/default values and future semantic API | Finite vocabulary; no Go adapter yet. Ordered raw field/map entries preserve collisions; typed bytes differ from authored strings |
| `Value`, `Identity` | Future `expr` resolved values and generation-local carrier identity | No public Go API or storage implementation is certified |
| `Outcome` | Future resolver/projector outcomes and diagnostics | Foundation constructors only; candidate failure classification is proved separately below |
| `Shape`, `Declarations`, `HasType` | Finalized effective `expr` occurrence | Foundation structural completeness only; candidate optional fields, constraints and resolution use separate relations below |
| `Plan`, `Field`, `Branch` | HTTP/OpenAPI visibility, body/view selection and allocated protobuf mapping | Explicit inputs. Producing valid plans from DSL identities is not proved. foundation `required` and protobuf styles alone establish no target behavior |
| `Observe` | Target-observable service meaning | Independent of a projector. Expresses field/visibility loss and retained branch identities. the foundation alone has no full numeric, object openness, nil-container or protobuf presence rules |
| `Wire`, `DecodeScalar`, `WireTyped`, `Decode`, `Representable` | Structured JSON/protobuf projection and existing runtime codecs | Foundation wire judgments cover scalars, nullable, arrays and selected bodies only. Separate candidate judgments below cover full finite shapes |

Depth-indexed judgments use an unbounded existential depth, not a finite TLC-like
search bound. Successful derivations are finite. The foundation by itself does not prove that every complete finite value
resolves, projects or decodes.

## Weaknesses discovered after the initial foundation

| Weakness | Model/evidence now | Remaining owner and acceptance boundary |
| --- | --- | --- |
| Copying byte-count bounds onto base64 string length rejects valid values and accepts short decoded values | Two registered concrete Lean counterexamples above; actual generated HTTP server and inline/OpenAPI discrepancy reproduced for #574 | #575 specifies the policy; #569 proves the candidate semantic relations under explicit codec boundaries; #574 delivered shared schema arithmetic/grammar and the tested production correspondence recorded in BYTE_SCHEMA.md; the Lean witnesses alone do not establish that correspondence |
| Schema enum membership can differ from actual decoder acceptance of a noncanonical pad-bit alias | Registered alias counterexample plus canonical control use a finite, explicit specimen codec. This is not a universal codec axiom or a proof about generated union routing | Candidate schema and decoder predicates are independent, and representability/preservation require both. #570 and consumer gates test canonical/alias/malformed values with actual schema validators and decoders; never infer runtime uniqueness from schema-only uniqueness |
| Missing required fields and ambiguous nested unions affect complete-first branch ranking differently | M1 has no resolver or optional-field typing theorem. Existing Go complete matching excludes nested ambiguity; no M1 theorem establishes a broader candidate algorithm | #575 settles complete-first ranking, zero-complete fallback and ambiguity obstruction. Candidate full-outcome correspondence and competing controls now establish that rule; #570 preserves the public compatibility adapter |
| One named schema/cache entry can be shared by incompatible JSON, multipart or location representations | M1 `Plan` is explicit vocabulary, not a derived representation identity or a schema-cache model. The shared `Blob` reference probe reproduces this integration weakness; current Lean witnesses make no naming/cache claim | #575 defines public-name/reference compatibility; #570 supplies durable representation/occurrence plans; #574 schema analysis consumes them before component registration. Reversed traversal, recursive references and excluded-context equivalence require independent generated tests |
| Named-Bytes pointer conversion generates uncompilable Go (#576) | Concrete generated compilation failure outside the Lean semantic domain; not a defect in the statements proved here | #576 repair committed in `8714f0d2`; [compiled-layout and generated-runtime evidence](../../../codegen/bytes_validation.md) establishes the tested cases. This remains tested generated-Go correspondence, never a Lean theorem about Go; broader integration stays pending |
| Native and named Bytes defaults generate illegal slice equality in shared transformations (#577) | Generated compilation exposes a failure that an accepted source golden did not detect. This is a separate generated-Go boundary, not a Lean semantic theorem | #577 repair committed in `8f30cdc8`; [compiled transformation and HTTP/JSON-RPC evidence](../../../codegen/bytes_defaults.md) records the completed tests. This closes that concrete compiler regression, not a universal generated-Go correctness claim; broader integration stays pending |
| Separate integer/decimal JSON wire constructors hide real branch collisions | `scripts/value_contract_numeric_test.go` verifies that integer 1 and Float64 1 both emit exactly `{"n":1}` and decode into either destination. Lexical `1.0`/`1e0` still decode as Float64 but fail the integer decoder | The candidate now uses one `Wire.number` lexical domain before schema/decoder matching. `CandidateControls` retains shared-wire and lexical-alias witnesses; numerical formatting/parsing remain narrow, explicit tested boundaries. Affected scalar proofs, the integrated audit and independent #569 review passed |
| Exact numeric equality does not determine raw numeric spelling or target precision | [Actual mixed-width controls](../../../scripts/value_contract_numeric_representation_test.go) use Float32(0.1) and Float64(Float32(0.1)): both equal 13421773/134217728, but their JSON/key spellings differ. Declared Map(Any,String) retains both keys, Any arrays retain both spellings, and scalar enums accept their equal numeric meaning. The same wire `0.1` decodes to different exact values under Float32/Float64 | **Reviewed #570 numeric checkpoint:** candidate scalars now retain width and signed zero independently of exact meaning and host DeepEqual identity. `NumericRepresentationControls` proves equal-meaning/distinct-wire and key-name controls and retains the old numeric-key-uniqueness counterexample. Encoding tables use the full representation identity; the Go reference adapter retains arbitrary-precision coefficients. The extended library builds. Declared source precision is now modeled before validation/selection and checked against Go for examples, enums and defaults. Target decoder precision is explicit and checked by the target-policy corpus below; canonical source map-key identity has passed the 522-theorem checkpoint and independent review. The registered 13-group production corpus now passes. DSL extraction outside those cases remains a tested boundary, not a universal correspondence claim. |
| Numeric formatting does not establish source eligibility | The first `numeric_literal_conformance_test.go` checkpoint exercised the draft reflective matcher. Independent legacy checks then showed that named numeric values do not pass `Primitive.IsCompatible`; treating their formatting result as an admitted Float64 source widened union candidates | `NumericLiteral` and `Coerces` retain formatting provenance as an explicit normalization boundary, but those proofs do not authorize source admission. `expr/value_source_eligibility_test.go` checks 321 host/contract pairs against the legacy matcher and public branch selection, then all three new resolver roles. The shared Go matcher now checks primitive eligibility before normalization. `SourceScalar` now retains concrete raw tags in scalar inputs and map keys. `SourceAdmission` and `SourceCoerces` independently require admission before normalization; canonical values remain unchanged. The integrated 567-theorem gate, fresh replay and independent source-admission/byte-shape review passed. Direct Go/reference checks retain both concrete named-value rejection and the explicitly abstract formatting counterexample. The byte-shape checkpoint passed 402 source comparisons. The subsequent no-callback correction preserves concrete source tags in reference preflight, requests literal conversion evidence only after fixed source admission, and does not recoerce prepared enums. All 13 registered assertion groups pass, including 402 source comparisons, missing admitted scalar/map-key rows, missing Any numeric spelling and the panicking-Stringer invocation counter. Full integration gates and generation comparisons pass; final commit review is required |
| Destination width and signedness change decoder acceptance independently of schema and source meaning | `internal/valuecontract/production_target_numeric_test.go` initially checked 420 scalar/map/untagged cases using JSON-v2 scalar and standalone numeric parsers. Independent native-map checks showed that standalone parsing is insufficient: numeric map keys `+1` and `01` are rejected and opposite signed-zero keys collide | Per-target integer and float policies, exact result/sign comparison, and fail-closed protocol controls passed the 510-theorem checkpoint and independent review. The corpus now uses actual typed-map decoding; 132 additional direct cases check native widths and cardinality. This correction and map identity passed the 522-theorem checkpoint and independent review. This correction changes codec evidence, not schema acceptance; the earlier standalone-parser checkpoint is not a generated-map guarantee |
| Raw sequence shape, primitive eligibility and JSON encoding are independent | `expr/value_resolve_test.go:TestValueResolveByteHostShapes` and `expr/value_source_eligibility_test.go` cover native byte slices/arrays, named containers and defined byte elements. A named slice of native bytes encodes as base64 but fails declared Bytes admission; defined byte elements encode as an array and fail nonempty numeric-array admission; ArrayOf(Any) retains them | **Reviewed #570 checkpoint:** `SourceBytes` retains native slice/array/named-container evidence before branch matching; defined element types remain ordinary child inputs. Sequence children retain original host evidence; Any materialization preserves binary meaning, while declared Array consumes the original children. Empty arrays retain vacuous element compatibility. `SourceByteControls` retains the old scalar-erasure ambiguity counterexample. The integrated 567-theorem gate, fresh replay and 402 production source comparisons passed independent review |
| Numerical equality is not source map-key identity, and source identity is not native decoded-key identity | Mixed-width equal-valued keys can have distinct canonical names; the new source differential comparison now retains both. Native typed numeric maps reject equal decoded keys, including opposite signed zeros, even when their wire names differ | `SameKey`/`keyEqual` and `AdmissibleKeys` now own source/enum map identity through the canonical encoder, with the same codec threaded through typing and strict equality. Raw Any equality stays separate. Runtime `mapDecoded` keeps homogeneous native numerical uniqueness. `MapIdentityControls` covers swapped values, heterogeneous collisions, raw-host separation, rounding collisions and signed-zero rejection. Decoded numeric spelling requests close the external boundary for postdecode enum/observation comparisons; the complete 522-theorem checkpoint and fresh replay passed independent review |
| Specialized traversal paths can bypass occurrence-owned constraints | Independent review found map-key enum/alias rules and documentary alias-collection length gates missing. Shared body/local-constraint/enum ordering repairs and the subsequent positive Any-key enum correction pass direct tests and independent re-review. The raw-key comparison subsequently exposed capture erasing a present-empty authored enum; preserving the empty snapshot repairs the shared path | **Tested #571 correspondence:** `named-key-contracts` independently extracts raw scalar key ancestry, checks enum/default admission at every prefix, and compares whole-map resolution. Constrained projection composes independent key admission with the key codec reference, keeping raw Any host identity separate from canonical names and native decoded keys. `effective-alias-lengths` compares String/Array/Map raw lengths, effective bounds, resolution and projection. All 18 groups pass independent review. These bounded adapters do not remove the general graph adapter's unsupported-shape guards or prove arbitrary Go extraction. #574's byte-schema evidence remains separately recorded |
| Source graph traversal must preserve entries and terminate even for rejected input | Independent review reproduced a NaN map key silently erased by MapKeys/MapIndex, cyclic synthesized raw data traversed again without a guard, and custom defined-byte elements bypassing the raw child boundary | Shared MapRange key/value capture, cycle-aware owned copying and exact native-byte classification have red/green direct tests and passed independent re-review; finite builtin Lean inputs do not prove arbitrary Go graph traversal or custom callbacks |
| Raw Go `any` decoding is not Loom's generated Any carrier | [Actual JSONValue controls](../../../scripts/value_contract_any_test.go) preserve exact large numeric text through `loom.JSONValue`; typed bytes and literal base64 text still materialize to the same JSON string | Candidate `Materializes` and `jsonSnapshot` record target JSON meaning independently of raw-source equality. Universal materializer correspondence is checked in `MaterializationProofs`; candidate projection composition is checked; #570 carrier correspondence passes for the registered corpus; later consumer coverage remains required |
| Source candidate preference includes compatible non-object alternatives, and cross-member authored/wire aliases can conflict | [Preference/null controls](../../../scripts/value_contract_source_test.go) preserve current non-object ranking; [alias controls](../../../scripts/value_contract_alias_test.go) retain the actual wrong-type overlap witness | Candidate resolver validates every retained assignment and preserves supported name overlaps. Full resolver correspondence is checked; #570 source/alias regressions and registered correspondence pass; the public Go compatibility adapter remains a separately tested boundary |
| Schema acceptance, decoder unknown-member policy and semantic retention were conflated in one target flag | `TargetDeclaration` now separates schema openness, decoder rejection and target extra-member retention. `ProjectionControls` exercises a schema-unique wire with two runtime interpretations, rejection for runtime use and permitted documentation-only emission | #570 must derive all three policies from the actual occurrence/codec. The defaults match ordinary and generated tagged JSON decoding; no schema-only uniqueness inference is permitted |
| Map key constraints and lexical key decoding differ from emitted map schemas | Current IR `analyzeInlineMap` emits an object value schema without key constraints. Candidate schema follows that shape; runtime decoding separately checks key parser acceptance, constraints and decoded-key collisions | Numeric key formatting shares `NumericCodec`; key-parser functions have a separate lexical boundary from JSON number-token parsing. #570 must test both, including aliases; canonical encoder membership never substitutes for decoder acceptance |
| Logical service fields can share a JSON alias when their actual transport places them in different components | The required gRPC mapped-metadata matrix exposes the new occurrence builder imposing JSON wire uniqueness before a transport plan exists | Shared Go capture must preserve unique authored member identities; supplied duplicate wire aliases must be ambiguous, while existing cross-namespace authored/wire overlaps retain their validation and precedence. Target emitted-name uniqueness remains plan-owned. The Lean `WellFormedDeclarations` premise currently requires unique wire aliases: these additional Go graphs need explicit adapter rejection and direct/generated tests, not a false differential claim. #573 owns independent lowering, or an earlier migration if it needs that correspondence. `value_alias_ownership_test.go` verifies authored keys, ambiguous aliases, cross-namespace compatibility and target visibility; the actual adapter rejection subprocess and 16-service generated gRPC metadata round trips pass. Independent scoped review passed. The final parent/candidate comparison is byte-identical for mapped-metadata, and whole integration gates pass |
| A projected collection can inherit stale naming provenance from its source declaration | Required meal-planner output changed `RecipeResponseSummaryCollection` to `RecipeCollection` after copying source occurrence metadata | Shared view projection must preserve occurrence constraints while assigning naming provenance to the new derived declaration. Four direct view/canonical-name controls, race checks and unchanged meal-planner JSON/YAML goldens pass; independent scoped review passed. The final parent/candidate meal-planner comparison is byte-identical. This is DSL-to-plan naming evidence outside the semantic theorem, not grounds for a renderer exception or a new proof vocabulary |
| A named Bytes type shared by nullable and non-null result occurrences can corrupt view representation planning | The retained #570 nullable-view regression is outside the Lean table validator: the model receives explicit occurrence/target identities and does not construct them from shared Go attributes | #570 isolates effective occurrence presence and physical representation. Direct tests and generated nullable-view/child-view probes pass, with the original parent failure retained in the comparison record. This is tested adapter/plan derivation, not a semantic theorem |
| Initial Lean evaluator branch returns bypassed whole-node enum gates | Universal schema correspondence exposed a candidate defect before approval; independent spec rejected the empty enum while the initial array evaluator accepted. `ProjectionControls.legacyHoistedEnumCounterexample` retains the rejected do-block layering, and repaired controls cover array/object/map/union/Any gates | Schema, decoder and resolver now compute their bodies in separate helper functions before uniform enum validation. All affected component proofs/controls have been rebuilt; independent exact-diff review remains required; this is a candidate implementation repair, not an assumption narrowing representability |
| Alias enum overrides disagree across existing consumers | An admitted Inner Int enum `{1,2}` with bounds `[0,9]`, then Outer enum `{2,3}` with bounds `[1,5]`, produces a validator accepting `3`; its actual OpenAPI schema and resolver example role reject `3`. The emitted example is also `3`. Parent `2f6bfbf7` emits the same schema. | **Accepted #571 policy:** an explicit descendant enum must equal or refine the effective ancestor enum. Any member outside it is a design error; no override or silent intersection is permitted. `AliasContracts` proves the raw-ancestry acceptance judgment and positive/rejected controls. The Go adapter independently normalizes exercised authored values before comparing the production snapshot. DSL extraction, diagnostics, codecs and unmodeled value shapes remain tested boundaries. |
| Effective Pattern/Format selection can disagree with ordinary alias resolution and all-of schemas | The first effective owner retained only the nearest Pattern and Format, so a derived enum or default could pass construction while normal alias resolution rejected it against an ancestor clause. | **Accepted #571 policy:** exact typed clauses conjoin across the whole named ancestry. The owner reports stable current-to-base clauses, de-duplicates exact `(kind, value)` identities with first/current provenance, and admission checks every clause. `AliasContracts` proves this abstract conjunction for full `(kind, predicate)` identities and retains separate nearest-only Pattern and Format counterexamples. The Go adapter independently evaluates representative regex/UUID/date-time cases; arbitrary engine equivalence remains external. Direct Go controls require example synthesis to filter inherited enum candidates through every effective Format clause. Runtime and documentation plans retain every enum clause member in its declared shape; later predicates still govern projection and decoding. Generator choice, plan extraction and the concrete format engine remain tested correspondence boundaries, not Lean claims. |
| A correct effective owner does not protect a sampler that bypasses it | The three-transport probe retained the inherited enum and derived UUID format in both service and protobuf attributes, but `AttributeExpr.Example` inspected only local validation and generated a UUID outside the enum. The generated gRPC validator rejected that example. Merely copying effective predicates still let an inherited object enum supply an example missing a newly required field: sampling must use the owner's filtered candidates. Re-capturing an already prepared synthesis graph also suppressed named union examples because the graph contains a private branch-selection adapter. | **#571 adapter obligation:** ordinary generated examples consume the complete effective constraints after authored-source and suppression selection. Enum sampling uses `EnumCandidates`, including its present-empty domain; runtime and schema validation retain the unfiltered clauses. `ValueContext.Synthesize` samples its already materialized graph without re-admission: only exact attribute pointers prepared by that graph may skip materialization, recorded privately on its own generator. Direct controls cover enum/format intersections, required-field narrowing, empty domains, inherited non-enum predicates and retained named-union choices; generated transport controls must accept the emitted example. This closes an implementation correspondence gap under the existing conjunction and source-selection design; it adds no theorem about Go sampling or codecs. |

The same example re-admission defect also affects OpenAPI's private
`selectedExampleUnion` wrapper. Its #571 generated-example repair uses shared
source selection and synthesis, then `DeclaredJSONValue` as a semantic carrier into the existing
OpenAPI conversion path. That carrier preserves declared bytes and precision and
the selected tagged envelope; it is not checked target wire output. Carrying the
original service result through target plans and removing the remaining OpenAPI
re-synthesis and interpretation remain #572 obligations.

These rows invalidate any broader inference from the old foundation approval;
the original concrete lemmas remain unchanged. Counterexample/model and ledger
updates, audited gates and independent re-review precede reliance on a repaired
claim. A documented pending row is not a substitute for the later required proof
or production test.

## Candidate proof and production mapping

The candidate modules use `ValueContract.Candidate`; they do not replace or enlarge
the statements of the original M1 witnesses. `Canonical`, `SchemaEvaluation`,
`RuntimeEvaluation`, `Observe` and `Representable` are separate relations. Runtime
representability requires the same canonical wire to satisfy the emitted schema
and actual decoder preservation; documentation-only targets have no decoder claim.

`SourceSelection`, `GraphValidation`, scalar correspondence and the materializer
have checked universal component statements. In particular,
`MaterializationProofs.materialize_iff_Materializes` and
`jsonValid_iff_JSONValid` use unbounded finite derivations, not a fixed input cap.
Their axiom audit includes only `propext`, `Classical.choice` and `Quot.sound`;
no executable function uses a choice operator to manufacture its output.

The following candidate component statements have been checked. The integrated manifest/axiom audit, fresh replay and actual rejection controls
have passed. Independent exact-diff review remains required; neither a component
row nor the combined gate establishes the Go adapter.

| Owner and principal statements | Proved finite-domain obligation | Production boundary |
| --- | --- | --- |
| `SourceSelection`: `selectedFromInput`, `selectedProvenance`, `authoredSuppressionException`, `synthesisEligibility`, `contractNoReplacement` | Authored input and provenance survive source selection; only an absent example request can authorize synthesis | Existing `ExtractUserExamples` precedence, reachability and occurrence extraction must map faithfully to the supplied lists/roles. No random generator or synthesis quality theorem |
| `SourcePublicOutcomeProofs`: `resolve_outcome_iff`, `sourceOutcome_total`, `sourceOutcome_unique`, `sourceOutcome_typed`, `sourceOutcome_complete_missing`; `SourcePreferenceProofs`: `prefersObjectInput_iff` | Complete-first and zero-complete fallback outcomes, including all failures, branch identities and ordered missing paths, equal independent recursive judgments. Derived structural/rank budgets terminate without malformed exhaustion on well-formed declared roots | Actual authored/wire alias extraction, source provenance, host evidence classes, key codecs and diagnostics require Go correspondence; no target participates in source ranking |
| `GraphValidation`: `validateDeclarations_iff`, `validateTargets_iff` | Executable graph guards equal independent identity/edge/rank predicates; consuming recursion permits arbitrary finite inhabitants | DSL graph/target-plan extraction and assigned identities/ranks remain tested implementation work |
| `MaterializationProofs`: `materialize_iff_Materializes`, `jsonValid_iff_JSONValid`; `MaterializationValidity`: `Materializes_JSONValid` | Built-in finite Any materialization equals independent canonical meaning and produces valid structured JSON | Actual `loom.JSONValue`, scalar codecs, host snapshots and deterministic serialization are separately tested; opaque/custom values are excluded external codec boundaries |
| `EqualityBudgetProofs`, `StrictEqualityBudgetProofs` | Wire/target/strict finite equality decisions match unbounded relations, including negative nonmembership; snapshots contribute wire height to strict depth | JSON enum numeric equality differs from exact snapshot preservation and raw host equality; adapters must preserve these distinctions |
| `SchemaProofs`: `schema_iff_SchemaEvaluation`; `RuntimeProofs`: `decode_iff_RuntimeEvaluation` | Derived structural/rank budgets implement independent complete schema and actual decoder judgments, including rejected alternatives | Emitted schema constraints and runtime decoder policies must be derived separately from each target occurrence; scalar library behavior is parameterized, not a proved Go implementation |
| `EmptyProofs`: `emptyObserved_true_iff`, `emptyObserved_false_iff`, `observePresence_iff` | Both omission decisions equal the unbounded field-local predicate; exhaustion never means nonempty | Requiredness, nullability, omitempty/default and target visibility must map to the correct occurrence plan |
| `ObservationExecutionProofs`: `observeValue_iff` | Every finite independently observed value is produced by the executable phase, and every successful output has that observation; only well-formed graph/root lookup is required | This phase preserves visible union identity and legitimately drops excluded data. It does not derive target plans from mutable `expr` graphs |
| `CanonicalConstructionProofs`: `construct_some_iff`, `construct_none_iff` | Canonical construction and explicit absence agree with the independent relation at a graph/value-derived budget | Required wire fields are checked after field-local observation and parent omission; codecs and plan extraction remain external |
| `ProjectionCorrectness`: `project_sound`, `project_progress`, `project_runtime_preservation` | Every emitted wire has independent canonical/schema meaning; every independently representable finite value emits; runtime-backed targets decode that same wire to the target observation | No premise assumes projector success or preservation of excluded service fields. This proves the reference functions, not production Go or generated decoders |
| `NumericBounds`: `effectiveNumericBounds_iff`; `AliasContracts`: `effectiveEnumeration_iff_raw_refinement`, `effectiveEnumeration_returns_last_explicit`, `validateAliasPrefixes_iff`, `evaluated_current_authored_enumeration_valid`, `evaluated_default_valid`, `numeric_layers_conjunction`, `predicate_clauses_conjunction`, `effectivePredicates_contains_raw`, `effectivePredicates_identities_nodup`, `effectivePredicates_first_identity`, `effectiveRequired_contains_raw`, `effectiveRequired_identities_nodup`, `effectiveRequired_first_identity` | Every finite raw alias ancestry enforces enum refinement, validates each locally authored enum under the non-enum contract at its declaration, validates the selected default at every base-to-current prefix under that prefix's full modeled contract, selects the final explicit enum/default, conjoins all four numeric bounds, stably unions exact typed Pattern/Format identities current-to-base with first provenance, and unions required identities with effective-first provenance. An inherited enum composes with later non-enum clauses as an intersection; it is not re-authored at the descendant | Semantic IDs and predicate acceptance facts are independent adapter inputs. Go extraction, exact float conversion, regex/format implementation, finalization/diagnostics, length/shape validation and codecs are executable boundaries. Primitive copied field templates are not named-type ancestry |

The full `project` function is executable and its controls include lost fields,
partial-source selected bodies, second equal-payload branches, finite recursive
arrays, nil/null/absence, heterogeneous map collisions, JSON numeric and byte
aliases, and schema/runtime unknown-member differences. The controls supplement
the universal target theorems above. Source-outcome and target composition are
now checked. The reviewed #569 integrated gate passed all 483 registered theorem
audits, fresh kernel replay and real rejection controls. The #570
numeric and map-identity checkpoint passed 522 audits, fresh kernel replay and
independent review. The integrated source-admission and byte-shape checkpoint passed 567 audits,
fresh replay and independent review, together with 402 production source
comparisons. The later no-callback control exposed over-eager reference codec
requests; the corrected request collector passes all 13 registered assertion
groups without failed or skipped events. These are groups, not input counts.
That #570 checkpoint was delivered in `2f6bfbf7`. #574 subsequently delivered
byte-alias schema mapping. #571's 18-group comparison adds bounded raw
scalar-key and String/Array/Map alias-length correspondence. Its integrated proof
gate passes 634 registered claim audits, fresh kernel replay and five rejection
controls. The parent/candidate corpus acceptance covers all 934 registered
designs, with exact reviewed output differences, source-scoped evidence reuse
and explicit linked pre-existing failures. The final retained-evidence check
and its negative controls pass; the failures remain recorded as failures.
Eight standalone fixtures also pass their paired generation and output checks.
These checks establish the exercised production correspondence, not a theorem
about arbitrary generated Go. The old M1 gate alone does not establish candidate
correctness. `Resolution.missing`
preserves an ordered list of member-identity
paths; `Failure` carries classification only. Production array-index/map-key
paths and rendered diagnostics remain separately tested adapter obligations.

## Map collision diagnostics: #456

The [diagnostic model](README.md#map-collision-diagnostics-456) guides rejection
through known collision witnesses and explicit unknown evidence. Unknown or
cyclic siblings preserve known witnesses. An implicit union is rejected only
when eligibility is known, at least one alternative is compatible, and every
compatible alternative has a known collision. An explicitly selected branch
remains authoritative. The separate 29-claim audit and fresh kernel replay pass;
Go reflection, declaration capture, compatibility and traversal remain tested
implementation boundaries.

Direct controls cover declared-key conversion, duplicate object spellings,
finite compatibility, cycles, opaque codecs without invocation, independent
design errors and deterministic structural diagnostic ordering. Integrated lint
and tests pass. Two process-isolated generator attempts per revision reproduce
the reported `Any`-key `1`/`"1"` collision: the parent fails during generation,
while the candidate reports the collision during design validation and emits no
artifacts. A collision-free `MapOf(String, String)` companion has identical
parent/candidate artifacts and passes generation, example generation, module
tidying, build and vet in both attempts.

The collision-free `MapOf(Any, String)` companion also has identical artifacts,
but both revisions generate an invalid `map[loom.JSONValue]string` key type and
fail compilation; vet is skipped. This pre-existing transport defect remains
unresolved. The successful concrete-key control does not establish compilation
of arbitrary map key types.

## Explicit request-body ownership: #581

HTTP finalization owns the correspondence between an explicitly declared body
and its selected payload. Bind the endpoint's copied body and its immediate
object members using the existing HTTP member mapping. Preserve the shared
authored declaration, body type name and transport annotations. Unmatched
members remain unrelated; value planning must continue to reject them.

[`BodyMemberBindings.tla`](../../tla/BodyMemberBindings.tla) compares root-only
binding, mutation of the shared declaration and binding each endpoint's copy.
The first two fail member correspondence; the endpoint-copy rule preserves
correspondence and the authored declaration across both finalization orders.
The finite model assumes the member mapping. It does not establish Go copying,
name normalization or recursive member inference.

Direct Go controls fail on the parent and pass with the repair. They exercise
two endpoints, two independently mapped members, unchanged authored ownership,
selected payloads, retained annotations and rejection of unrelated members.
Affected package, race, vet and configured staticcheck checks pass, as do the
full package tests, lint, generated-code quality, OpenAPI contracts and HTTP and
JSON-RPC integration gates. Parent/candidate ticktock and HTTP quality fixtures
regenerate, compile and vet with identical generated bytes. All six coverage
thresholds pass.

The 45 affected catalog designs were compared against parent `06e47031` with two
process-isolated attempts per revision. Forty-four retain their prior outcomes
and artifact bytes, including intentional invalid designs. For
`EndpointBodyAsUserType`, both parent attempts reproduce the unrelated-member
panic. Both repaired attempts pass generation, example generation, module
tidying, build and vet and produce the same 21 artifacts. These newly emitted
files are the only output difference. The rendered contract retains `id` as a
path parameter and `EntityData` as the named request body.

## Trust and evidence boundaries

- **Proved:** the registered M1 witnesses and candidate component statements
  above, checked by Lean 4.34.1 and the combined fresh kernel replay.
  Allowed logical axioms are exactly `propext`,
  `Classical.choice`, `Quot.sound`; the per-theorem report is authoritative.
  `sorryAx`, custom correctness axioms and native-evaluation trust are rejected.
- **Tested:** gate orchestration and actual transitive rejection of an admitted
  dependency/custom axiom by `scripts/value_contract_proof_test.go`. Enabled
  external tests fail on missing tools. #566 owns actual generator probes;
  their results must be linked to the baseline record, not inferred from Lean.
- **Bounded-checked:** the separate M1 TLA model, owned by #568, addresses
  phase/cache/source ordering only. No TLA result is asserted by this gate.
- **Assumed or external:** the Lean kernel/toolchain and operating environment;
  correct DSL-to-plan mapping; scalar runtime precision/ranges; lexical codec
  behavior; JSON/protojson libraries; Go compiler; custom/opaque Any inputs.
  `ByteCodec` has explicit functions, not an axiom claiming arbitrary codecs
  correct. A future theorem requiring `ByteCodec.RoundTrip` must list that
  hypothesis. Its actual base64 implementation needs separate tests or proof.
  Both Bytes and String use one `Wire.text` domain before matching; byte/string
  collisions remain visible.
  `boundaryCodec` intentionally recognizes only the concrete empty, one-byte,
  two-byte and alias specimens; no universal round-trip or decoder-language law
  is claimed for it. Its string lengths in the witnesses are ASCII lengths.

## Raw host-equality evidence

Early #569 review found that erasing concrete Go map/key/value types made a
non-string-key `Any` map fail its own enum. Widening structural key equality
would incorrectly accept different concrete Go maps. `Input.host` and
`Value.host` now retain an adapter-owned `DeepEqual` equivalence-class identity
at every recursive raw node. Equal identities authorize only the legacy raw
comparison precheck; fallback comparison still runs for unequal identities.
Finite payload validation and canonical key-collision checks remain mandatory.
Materialization erases the evidence before producing the shared wire value.

The boundary requires identity equality **iff** actual `reflect.DeepEqual` on
immutable finite built-in snapshots; hashes alone cannot satisfy it. This is a
host adapter obligation, not a proof of reflection or candidate correctness.
[`TestValueContractAnyHostEquality`](../../../scripts/value_contract_source_test.go)
checks native int/int64/bool-key maps, concrete/dynamic type differences, nested
maps, insertion order, and the preserved numeric fallback against the actual
legacy adapter. Lean host controls now check same/different class behavior, payload rejection
and target erasure; all affected correspondence components have been rebuilt.
The integrated audit passed; independent exact-diff review remains required.

## Required-field canonical representation

The #569 construction proof exposed a gap in the first independent canonical
relation: its object fragment rule allowed omission of required fields. In a
documentation-only untagged union, an empty wire could then satisfy another
branch's schema while the selected branch remained incomplete. The old
observation/canonical/schema witnesses and builder rejection were checked before
the repair. The canonical relation now independently requires `required=false`
for every omitted fragment, including implicit-default omission.
[`PresenceControls.lean`](ValueContract/PresenceControls.lean) retains the rejected
fragment rule, a universal required-singleton countercheck, required/default
negative controls and optional/default positives. Existing selected-body controls
continue to accept legitimate loss of unrelated service fields. Representability
still does not mention builder or projector success.

## Observation before canonical construction

A second checked #569 progress counterexample had a full independent runtime
representability witness: a parent optional `omitEmpty` field contained an object
with an absent required child. Observation legitimately removed the empty parent,
and the resulting `{}` passed schema and actual decoding. The first combined
builder instead checked the child's required field before parent omission and
returned incomplete. [`LegacyProjectionBuild.lean`](ValueContract/LegacyProjectionBuild.lean)
retains that rejected executable for the durable witness in
[`OmissionControls.lean`](ValueContract/OmissionControls.lean).

The candidate now executes `observeValue` before `construct`. The former owns
visibility, materialized Any meaning and field-local presence; the latter checks
required wire fields only on the retained observation. The same child remains
incomplete when its parent uses explicit presence. Representability remains the
unchanged independent observation/canonical/schema/decoder specification. General
emptiness decision adequacy now has positive/negative unbounded equivalence in
`EmptyProofs`, and `observeValue_iff` establishes the full observation phase.
`construct_some_iff` and `project_progress` now establish canonical construction
and combined progress against the unchanged independent representability
relation. Final independent exact-diff review remains required.

## Pending acceptance obligations

**#569:** the complete candidate functions and universal component/composition
proofs are checked. The final static manifest covers every public theorem;
integrated fresh replay, actual negative controls and independent exact-diff
review passed. The atomic #569 commit is `cd6d2fb0`; #570 amendments have their
own validation and review obligations.

**#570:** the executable reference calls the proved functions. Independent
Go/reference adapters retain map entries, source identity and presence; the
registered differential corpus and its negative controls pass. Full lint, test,
coverage-ratchet and final generation comparisons pass. The [durable comparison
record](../../../internal/valuecontract/OCCURRENCES.md) preserves the inputs,
intended differences and outcomes. Final independent review authorized the frozen diff; the atomic commit
`2f6bfbf7` was pushed. The completed task's temporary results and compiled Lean
outputs were removed after review, with durable inputs retained in Git.

**#574 checkpoint:** `AliasLengthBounds` proves decoded-bound intersection for
arbitrary finite alias chains. `ByteLengthProjection` proves residue/encoded
interval equivalence, actual clamp/omission behavior, complete residue coverage,
grammar-length composition and the checked multiplication limit. These 19
statements passed the 586-theorem aggregate audit and fresh kernel replay.
The complete conformance target also passed its actual rejected-proof controls
and all 15 executed comparison groups.
The `byte-alias-lengths` and `byte-schema-bounds` executable groups compare the
actual shared helper and Go source/target behavior against these functions.
They do not discharge general alias enums, key constraints or DSL lowering.

**#571 checkpoint:** `NumericBounds` and `AliasContracts` consume raw authored
layers, never a production-lowered effective interval or requiredness oracle.
The universal statements cover the four-bound conjunction, open ties, valid
empty intervals, enum refinement and selected declaration, validation of each
enum where authored, selected-default validation at every base-to-current
prefix, default replacement, current-to-base typed Pattern/Format clause
conjunction and first provenance, and required identity inclusion, uniqueness
and first provenance. Inherited enum members compose with later non-enum clauses
as an intersection; they are not reasserted as locally authored values.
The deliberate rejection files retain the old numeric-overwrite, closed-tie and
enum-override defects plus nearest-only Pattern and Format selection as false
obligations.
The aggregate axiom audit covers 634 required theorem statements and admits
only `propext`, `Classical.choice` and `Quot.sound`.

The registered `effective-alias-contracts` Go group extracts raw ancestry,
duplicates, absence and provenance independently. Its local semantic normalizer
does not call `NewValueContext`, `NewOccurrence`, `Resolve`, `IsRequired`, or
the production effective owner. The candidate owner is called only on the
comparison side. Controls exercise exact large signed/unsigned integers,
Float32 declared precision, Bytes text/byte normalization, recursive Any
equality, arrays, maps and nullable enum members. Codec-free structs and non-nil
pointers for declared Objects use an independent reflection adapter for the
reviewed exported-field/JSON-name/embedding/ambiguity/snapshot/cycle subset;
their children use the same independent declared coercions. Structs and
pointers supplied to `Any`, plus recognized JSON/text codec method sets, retain
raw host identity; method sets are detected without invocation. This host
materialization is tested Go correspondence, not part of the Lean theorem.
These controls establish the named cases, not a universal Go refinement. Union
values, custom callbacks, general Go JSON reflection, all DSL graph finalization
and diagnostic text, arbitrary float-to-decimal extraction, codecs, and default
length/shape checks remain explicit Go/tested boundaries.
The Pattern/Format adapter independently evaluates representative regular
expressions, UUIDs and RFC3339 date-times, and compares exact typed clause text,
current-to-base order and original provenance. It does not prove equivalence to
the complete production regex or format engines.

A read-only finalized-design audit registered default roots before DSL
evaluation in both retained consumers. Auto-K at its pinned Loom v1.9.0 observed
100,391 attributes including localized copies, with 100 Pattern and 570 Format
layer observations; Drum at its pinned v1.9.0-alpha.15 observed 7,475 attributes,
with 40 Pattern and 22 Format observations. Neither graph contained a named
ancestry with Pattern or Format on more than one layer. The new conjunction is
therefore required framework behavior, while these current consumer designs do
not predict a Pattern/Format output delta. This audit does not replace their
post-freeze generation and compilation probes.

Independent review caught a production composition error: projecting an inner
bound before intersecting the outer alias could overflow a redundant huge bound,
or reject an effective empty range instead of emitting an unsatisfiable schema.
Both adapters must collect the complete decoded-bound conjunction before the
single shared projection. This is a production-to-model mismatch; the proved
intersection statement itself did not require weakening. The direct and Ajv
regressions retain both cases. Independent scoped review confirmed the repair;
final whole-ticket implementation review remains required.

Schema preparation also needs structural plans without example/enum resolution.
A custom Any enum is valid existing schema input but need not resolve through a
built-in value projector. `ValuePlanSchema` keeps that distinction explicit and
cannot authorize `ProjectJSON`. Independent review also found that the initial
`ValuePlanNode.Attribute` query used `DupAtt`, which appended copied result types
to the global generated-result registry. The query now uses the shared occurrence
copy owner in captured-source mode: it preserves the already captured example
value and null flag instead of rereading the mutable authored origin. Repeated
queries, recursive graphs and caller mutation have direct and race regressions;
independent review closed both schema-mode findings. These are tested capture
and query boundaries, not additional semantic theorems.

The unchanged TLA+ checked configuration passes all 47,712 states, while the
missing-target cache control still fails `CacheOwnership` as expected. Neither
configuration models concrete component naming. The [byte-schema evidence
record](../../../internal/valuecontract/BYTE_SCHEMA.md) links the production
owners and reproduction commands. Complete schema registration,
per-representation naming, generated comparisons and all final #574 gates remain
open.

The registry review reproduced a remaining target-ownership violation in
synthetic tagged-union components: a cache keyed only by the authored branch
fingerprint reused a raw schema for JSON, or the reverse, depending on traversal
order. The existing TLA+ target-cache counterexample describes this ownership
class; it does not itself find the concrete Go cache. Both-order Go regressions
must pass after synthetic components enter the same representation-aware owner.
The same entry-point audit found unprepared WebSocket and SSE schema analysis;
their retained failures require migration through the shared representation
factory before #574 can close. These findings extend the production acceptance
matrix without claiming a new Lean theorem about component allocation.

The next registry review also rejected a naming exception based only on shared
source identity and the presence of a plan. A two-field String object and its
one-field projection could then silently claim one explicit public name even
without Bytes. Registration must retain structural identity separately from the
representation-specific byte projection: only the latter conflict authorizes a
variant. The no-byte case and a byte-containing sibling case are required
negative controls. This is another concrete allocator obligation outside Lean's
explicit target-plan domain.

Actual codec selection remains a separate extraction obligation. The physical
SSE check reproduced a false JSON classification for a native Bytes field mapped
to event data: `EncodeSSEData` writes its raw text, whereas named slices,
Nullable wrappers and whole-event values use JSON. A second runtime probe found
the same class of failure in an ordinary `ContentType("text/plain")` HTTP
response: `ResponseEncoder` writes `hi`, but the draft schema required base64.
These are failures to supply the model's codec input correctly, not failures of
the proved byte arithmetic. The repair must derive representation plans from
the actual built-in codec owner before schema registration. Request encoding,
request decoding, response encoding and SSE dispatch have distinct selection
rules; a media label cannot substitute for those rules. The finite built-in
codec matrix now passes against actual runtime functions and rendered schemas;
independent scoped review confirms the shared selector's behavior and ownership.
Whole-ticket comparison and final review remain pending. Custom runtime-injected
codecs stay outside this correspondence claim. `internal/httpcodec/codec.go`
owns the role-specific selectors consumed by runtime `http/encoding.go` and
HTTP representation preparation; SSE retains its separate physical-type owner.

The selector audit also found that a mapped response `Content-Type` header is
written after the generated code chooses its encoder. Header enum values
therefore do not establish a codec. A static response content type can establish
the encoder context; a header-only media declaration must retain its old schema
as an indeterminate codec boundary unless a separate runtime link establishes
the choice. This preserves existing behavior without claiming JSON equivalence
from an advertised label or introducing a new unsupported-target error.

The first full repository/comparison checkpoint exposed additional production
boundaries. A copied documentation body reported its transport wrapper's type ID
instead of the original declaration ID, so an equivalent imported schema was
rejected as a public-name collision. Original identity must be captured from
controlled ancestry before later mutations; the independent legacy structural
comparison still rejects genuinely different shapes. A referenced API error
also lacks a method-local carrier even though its finalized transport error
retains the correct declaration. Preparation must bind that exact declaration,
including inherited mapping scope, without inheriting every API error or using
the projected response body as its source.

Separately, splitting a pre-registered component into representations sampled a
new request copy and changed its existing example annotation. Legacy component
annotation authority must stay separate from representation-specific schema
identity, with source/context isolation, absence retention and copied results.
This is not permission to share projected wire values across codecs, nor does it
complete the #572 example migration. The importer must also retain explicit
`format: binary` metadata rather than relying on the old implicit Bytes format.
Independent scoped review has confirmed these repairs, including incompatible
String/Bytes shape rejection, inherited error shadowing, annotation isolation
and actual importer generation. Full comparison and final ticket gates remain
pending. The existing Lean statements take correct source/plan extraction as
an explicit boundary; these concrete tests do not turn it into a proved Go
refinement.

The isolated comparison also exposed an async reference-shape boundary: the
legacy inline traversal retains a named reference when a declaration hash is
already on its active path, including copied wrappers with the same hash. The
new traversal expanded an acyclic component reference instead, changing both
the schema shape and the example sampling context at a non-Bytes location.
This remains an open comparison finding. Any repair must preserve the existing
reference cuts while analyzing the intact constraint and representation graph;
flattened constraints must not become the semantic authority again. The Lean
byte-length statements do not establish reference-shape or sampling equivalence.
The same materialization boundary includes synthetic tagged-union envelope
references created during schema analysis: expanding them also changes existing
async contracts even when no byte constraint is involved. Acceptance must cover
those generated references as well as authored named declarations.
An annotation-only sibling of a reference also triggered an unnecessary
`allOf` wrapper. This hid inline properties from occurrence-context sampling
and exposed different component-context samples. Materialization must distinguish
annotations from assertion siblings: preserve ordinary inline shape for the
former and retain conjunction for the latter, without discarding byte or other
constraints to match an old example.
The retained-reference assertion control also found that async contract
extraction discarded active siblings through an unconditional reference-only
shortcut. TestAsyncMaterializationCutKeepsAssertionSiblings covers this boundary:
only a pure reference may take that shortcut; assertion siblings must reach
the shared schema renderer intact.
Independent review then found that a legitimate Required overlay retained its
assertion but hid inline child properties beneath `allOf`, skipping a child's
example callback. Constraint retention alone is insufficient: sampling must
follow the source/schema correspondence through conjunction wrappers without
flattening assertions or repeating callbacks. A combined rendered-document check
also found a stale async component reference after allocation. The final alias
collapse and reachability passes visited ordinary schema roots but omitted
framework-owned async message schemas. Both passes must traverse those typed
schema roots while leaving example/default values and arbitrary extensions as
data. Pre-resolving aliases in the async materializer would duplicate the later
cleanup policy instead of closing its missing edge ownership. Reference closure
and paired sampling remain separate production obligations under this repair;
neither follows from the byte arithmetic or graph fingerprint model.
Collect the complete comparison inventory before changing this shared policy.

The mixed representation probe also exposes a pre-existing named Bytes response
root-validator mismatch: the generated signature takes a value, but the body
and call assume a pointer. Parent and candidate inputs, non-OpenAPI artifacts
and compiler diagnostics match exactly. Earlier field-layout regressions do not
cover root declaration/body/call agreement. This is a tested counterexample at
the generated-Go boundary, not a failure of the Lean byte arithmetic. Milestone
5 / #565 must derive all three uses from the same physical body plan and make
the retained probe build, vet and pass accepted/rejected runtime validation.
Until then it remains an explicit known failure, never successful build evidence.

Complete recursive schema identity is another allocation obligation outside the
Lean value arithmetic. The mixed probe retained two identical completed schemas
because the old DFS fingerprint distinguished a self-cycle from an equivalent
prefix into that cycle. The [representation-equivalence model](../../../http/codegen/openapi/internal/ir/tla/representation_equivalence/README.md)
reproduces this false distinction and an overmerge counterexample. Its proposed
partition-refinement and quotient-fingerprint algorithm passes two bounded
configurations, including successive refinement steps. Exact schema/annotation
extraction, ordered reference paths, public-name reservations and Go
implementation correspondence still require direct tests and independent review;
this model does not establish universal JSON Schema semantic equivalence.
The first Go quotient implementation also changed fingerprint serialization.
A two-envelope acyclic control showed that this alone can switch the canonical
name winner. The model used identical structural tuples for legacy and quotient
fingerprints, so its passing equivalence checks did not cover that change.
AcyclicFingerprintsPreserved now states abstract compatibility explicitly and
passed both bounded configurations and independent re-review;
exact legacy JSON serialization, reference-slot substitution and public-name
stability remain Go regression and artifact-comparison obligations.

Response semantic interning and public naming also require distinct keys.
`UnionResponseBodyDSL` retained identical rendered responses, but preserving
preexisting reference-sibling examples changed discriminator hash preimages and
a public suffix. The shared response owner now retains complete equality while
using the historical serialization only for naming. Its complete prior pass
reserves slots before semantic eligibility filtering, preserving original
representatives and per-use public aliases; split classes cannot steal retired
or authored names. The [response allocation model](../../../http/codegen/openapi/internal/ir/tla/representation_equivalence/README.md#response-public-identities)
checks this bounded policy with independent historical/semantic keys and rejected
partial repairs. Exact serialization, recursive normalization and literal parent
names remain Go extraction and artifact-comparison obligations, not Lean claims.

The next complete-comparison checkpoint exposed a remaining source-origin
boundary in TypeIdentityDSL. Both target wrapper levels correctly capture the
original declaration, but the schema query reads only the independently advancing
source cursor. At the inner wrapper that cursor is already an unnamed object;
the IR fallback uses the renamed target and a different example context.
Their non-annotation schemas are identical, but
the newly retained async reference makes the inner component and its different
examples observable. The registry correctly distinguishes those complete
schemas; weakening annotation equivalence would hide the upstream ownership
error. The schema query must use captured target declaration authority, and
compatible existing public components must retain their annotation context.
The [schema declaration model](../../tla/schema_declaration/README.md) records
this boundary and rejected partial repairs. The candidate now stores captured
target authority on each plan node and exposes it through `TargetDeclarationID`;
The then-current IR inferred public annotation reuse from matching declaration
and baseline; later counterexamples below invalidate that premise. The target
identity extraction remains independently checked. Scoped direct tests passed:
literal authored alias sequences (not the extraction helper as oracle), field/
container/branch/recursive positions, distinct equal-shaped declarations and
exact authored annotation-owner/context/position counts. Two fresh TypeIdentity
generations match both parent runs byte for byte, with successful build/vet.
The supplemental alias probe, final full comparison and repository gates remain
pending. This is tested query-to-consumer correspondence: the bounded model still
assumes correctly captured labels and does not prove arbitrary Go extraction,
graph pairing or sampling behavior.

The supplemental `DeclarationAuthorityDSL` then invalidated the annotation
sharing claim beyond TypeIdentity: two authored alias collection components with
the same declaration and baseline retain different legacy example contexts.
The candidate merges them into the canonical component. All four generated apps
build and vet, but the two OpenAPI artifacts differ without any Bytes contract
change. This is a production correspondence failure, not an approved difference.
The earlier declaration model assumed a derived target component and omitted
neutral registration and independent annotation-owner provenance. Its passing
result did not justify the production sharing rule.
The corrected planned-first comparison confirms one sampled Base component is
lost and AliasOne's reference changes: a plan-valid guard is insufficient. The
earlier parent-side overlay did not apply because `/tmp` and `/private/tmp` were
not canonicalized consistently; that run is invalid. Both corrected runs confirm
the intended prepass override actually executed. The complete 66-probe inventory
reproduces the same loss
in moved-object `Item` examples. Actual ancestry traces locate an earlier error:
the builder advances source aliases on inserted target-only wrapper edges.
The intermediate declaration model rejected plan-only and shared-child-role
repairs but still assumed representation edges could select context. The final
allocation-owner correction below removes that assumption; exact original
annotation binding remains required for byte variants. The
paired-alias model reproduces blind advancement on a valid inserted wrapper and
checks nearest-ancestry pairing over bounded valid and invalid target chains.
Both original candidate configurations passed independent reruns. A subsequent
extraction check confirms supported structural flattening retains outer ancestry
but still requires exact underlying semantic IDs. The pairing model now separates
that descent from alias-edge matching and rejects an ancestry-only repair; its
expanded candidate passed independent review. Actual source-pointer
controls reproduce the wrapper failure while structural projection still passes.
The projection conformance adapter now compares one/two controlled wrappers
against the same independent Lean outcomes, including named unions and objects.
Independent execution passed 112 wrapper checks without skips; these compatibility
controls also pass before repair and do not prove opaque source identity.
Focused Go repair tests pass; final correspondence and review remain pending.
Source-origin capture, recursive
termination and baseline annotation binding are explicit assumptions, not proved
Go refinements. No universal annotation-context claim follows from these models.
An object-Bytes probe further shows that repeating a global baseline fingerprint
lookup loses an authored `noRef` position. The implementation must carry the exact
baseline binding along the paired traversal. The annotation model now distinguishes
two authored allocated contexts and rejects a projected memo hit that overwrites a
correct binding with another same-declaration/baseline context. That extension
passed independent review. An explicit `Body(Extend(...))` then exposed conflated
provenance: semantic transport binding overwrote copy/declaration ancestry. The
candidate separates those roles and propagates both through copies/constructors;
new controls reject overwriting target authority or dropping the semantic binding.
The pairing model also preserves structurally valid explicit body mappings without
requiring their structural definition in the source alias chain; named transitions
remain strict. Its structural-check predicate abstracts actual kind/member/branch
checks, whose extraction remains a Go obligation. Independent review reran all
sixteen configurations: both candidates passed and fourteen negative controls
failed on their intended invariants. `value_source_binding_test.go` checks that
copies and real streaming wrappers preserve separate declaration/source bindings,
including cycle rejection. `component_annotation_edges_test.go` checks explicit
body identity and parent-derived ordinary/Bytes annotations in neutral-first and
planned-first paths. `analyzer_baseline_test.go` checks exact structural routes,
scope restoration and cycle termination. Full expr/IR/v3 tests and scoped race
and lint checks pass; final generated comparisons and production review remain.
The parent-derived method-order oracle also rejects an overstrong test premise:
reordering authored methods already changes anonymous component names and samples.
The repair must preserve each input's baseline, stable canonical declarations and
same-input process determinism, not impose order invariance the parent lacks.

The next four-case generated comparison retained exact TypeIdentity, ordinary
aliases and moved-object outputs; all sixteen apps built and vetted. Bytes still
introduced reachable alias examples absent from the parent. The terminal Base
samples remained correct, exposing a gap in both the scalar context model and
the direct test's terminal-only oracle. Parent IR inspection shows pure alias
references without examples; the renderer normally collapses them. Projected
analysis resampled those references, preventing collapse. Complete-schema
allocation then correctly distinguished them and displaced canonical Base. This
is not an approved grammar difference or a reason to weaken schema equivalence.
`AnnotationPath.tla` now models the missing ordered annotation/absence
obligation. The old terminal-only check passes reconstruction; full-path
checking rejects it and an annotation-erasure control. The candidate permits new
references without new sample authority. Independent review confirmed all five
configurations, including the added resample-absent control: the terminal owner
stays correct while an existing annotation-free layer acquires a sample, violating
`OriginalAbsencePreserved`. The candidate passes all four invariants over 3,528
states. Actual baseline-path extraction, recursive traversal and naming remain
Go obligations.

The reviewed implementation records actual construction slots in
`analyzer_baseline.go`; `analyzer_representation.go` projects that baseline graph
without reconstructing aliases or sampling again. Direct tests cover member,
element, union and wrapper mapping, baseline immutability and full rendered
annotation paths. The independent four-case comparison at `a6fcc679` preserves
all parent names, references, samples and absence; only 72 byte-schema keyword
paths differ. Complete ordinary/async schemas pass 156 independent Ajv assertions,
now required by `make openapi-contract`. Capture `1ac7e9b0` passes strict
66-case comparison with exactly 18 reviewed schema artifacts changed and no
other artifact drift. All repository gates pass, including the 586-theorem audit,
rejection controls and 15 production assertion groups. The 527-design compile corpus
also passes; full affected-output comparison coverage and final ticket review remain
pending. See the [evidence record](../../../internal/valuecontract/BYTE_SCHEMA.md)
for retained baseline failures and the additional bounded alias-length check.
Expanded comparison then exposed defaults lost before projection: after service
generation names an async result, structural component reuse selects an ordinary
baseline without its defaults. The path model assumes a correct baseline; its Go
oracle checked examples but missed this acquisition boundary.
[`BaselineAcquisition.tla`](../../tla/schema_declaration/BaselineAcquisition.tla)
now reproduces cross-consumer and same-message reuse with identical examples but
different default presence. Independent review confirmed all five configurations:
annotation-aware and occurrence-owned acquisition preserve payloads and repeated
identity; structural reuse and never-caching fail their respective invariants.
Extraction and routing remain Go obligations: [IR controls](../../../http/codegen/openapi/internal/ir/async_baseline_authority_test.go)
check occurrence reuse, branch-tag correspondence, recursive cuts and synthetic
ownership; [generator controls](../../../codegen/generator/async_annotation_authority_test.go)
check defaults, sibling annotations, explicit false overrides and retained names.
These tests do not extend the model's proved domain. Expanded comparison also
found ordinary named-type policies applied before inline consumer ownership.
[`ConsumerDispatch.tla`](../../tla/schema_declaration/ConsumerDispatch.tla)
checks view, enum-set and nullability ownership together, including repeated
identity; faulty ordinary-first, global-suppression and policy-import controls
fail their respective invariants. Its roles and extracted inputs are assumptions,
not proofs of Go traversal, recursive cuts, byte constraints or runtime behavior.
The production boundary is `analyzeSchema` before named-type processing and
outer-occurrence nullability. [Public pipeline controls](../../../codegen/generator/async_view_authority_test.go)
compare parent scalar/object/collection and combined policies; [retained-cut controls](../../../http/codegen/openapi/internal/ir/async_view_authority_test.go)
keep ordinary view selection. The original plan still supplies effective byte
bounds without changing baseline null policy. Independent repair review, all six
repository gates and the registered 68-probe comparison pass. The broader 499-case
comparison and final ticket review remain pending under #574.

**Baseline allocation boundary:** the full catalog's `ResultBodyUserRequiredDSL`
changed a non-Bytes component name and both sample levels. `componentNaming`
overrode the actual allocated baseline context from matching declaration, shape
and representation edge. Removing that inference preserves allocation; actual
memo reuse and explicit canonical naming keep their existing owners. The
strengthened `SchemaDeclaration` varies actual fresh/reuse decisions independently;
old edge inference fails acquisition ownership, while byte memo controls fail only
projection ownership. Parent-backed generator and IR controls preserve the selected
body and TypeIdentity's ordinary suffix plus retained canonical cut. Three external
public pipeline comparisons match parent bytes; full generated verification remains
required. Source pairing, captured declaration identity and byte projection are
unchanged. No total callback-parity claim is added.

**M3 constraint-lowering gap:** a public DSL occurrence with `Minimum(5)` and
`ExclusiveMinimum(1)` resolves value `3`, although generated validation and
schemas retain both restrictions. `checkMinMaxValue` drops the inclusive bound;
`productionRules` repeats that overwrite, so agreement with the reference can
hide the defect. Independent lowering must preserve the complete effective
interval. Alias-required checks also share `IsRequired`; test declaration-derived
requiredness independently. A constructed alias loses outer required fields,
but the tested public DSL route finalizes their union correctly. Pattern/format
alias discrepancies are likewise internal-graph witnesses; the tested DSL route
rejects them. #571 now owns the accepted enum-refinement policy and these
effective-contract acceptance cases.

**Consumer migrations:** extend the registered corpus for each newly translated
contract, discharge the explicitly assigned extraction boundaries above, validate
real rendered schema instances and actual decoders, and compare generated
build/vet results and bytes in isolated processes. This remains **tested
correspondence**, not a proved Go refinement. Neither the M1 witness gate nor
#570's fixed corpus establishes completion of these later obligations.


## #572 prepared OpenAPI examples (implementation in progress)

The ordered-source extension is part of the existing audited `Proofs` entry point.
Its ten required claims preserve the first nonempty authored group, its order and
metadata, agreement with the singular last-member selector, reachability and
suppression. The flatten-all-groups counterexample distinguishes a different
policy. The source-selection dependency matches the separately reviewed model.
The integrated theorem audit and fresh kernel replay pass with 644 main claims
and the unchanged 29 collision-diagnostic claims. The opt-in negative-test wrapper
reached its ten-minute timeout during the final
collision-audit group, after earlier checks completed without failure. That group
passes separately with an explicit thirty-minute timeout; the Makefile now gives
the opt-in group that bounded budget. Production acceptance is still pending;
passing these proof checks does not complete the migration.

The Go correspondence seams are `ExampleSelection.Entries`, per-entry retained
occurrence/source/result records, and `ValuePlan.ForOccurrence`. Each subplan query
must match the captured source occurrence and target node, preserving its original
selection and policy. Foreign nodes, mismatches and genuine ambiguity must fail.
Metadata detachment, alias-effective ownership and transport filtering remain
executable checks, rather than conclusions of source-selection algebra.

The refreshed prototype's ten independent IR failures exposed structural and
example-plan roles being conflated. `PreparedExampleRoles.tla` checks separate
roles through one bounded recursion: neutral standalone structure, actual
transport projection authority, exact child example position and paired scope
restoration. The paired policy passes; conflated, cleared, stale-child and leaky
policies fail. Its root-codec extension also rejects inheriting a non-JSON runtime
codec for a JSON-valued documentation observation; all three checked invariants
pass over 290 states. Runtime codecs and node-local custom/SSE policies remain
separate from this root observation choice. Existing construction/annotation and
memo ownership obligations
remain in force. The model assumes correct captured edges; it does not prove Go
traversal, schema construction records or cache keys.

`OpaqueEnumWitnesses.tla` checks the conservative projection boundary for opaque
enum alternatives. Known members can witness membership. Every clause remains,
including one with no semantically known candidates, so an opaque clause cannot
silently disappear from the conjunction. The complete authored enum remains in
schema output; plan building does not run its custom codecs. Invalid declarations
still error. The bounded model passes, while deleting empty witness sets admits a
counterexample. Direct tests must establish the resolver boundary, preservation
of supported siblings, clause intersection and codec call counts.

The ten retained IR regressions pass after separating these roles. A direct
target-rebuild regression checks that new plan nodes receive new attachments
while source, anchor and resolved results remain identical and preparation
consumes no further random values. The rendered-instance matrix passes for
JSON and YAML in OpenAPI 3.1 and 3.2, including named groups, async child bytes
and explicit nulls, exact numeric values and rejection of a mutated tagged-union
discriminator. These are concrete correspondence checks, not universal claims
about all schemas or serializers.

The complete expr, representation and IR suites and their vet checks pass after
two further correspondence repairs. A target-only alias wrapper now reuses its
source precisely when the captured structural source is the same node; consuming
an authored alias remains a distinct edge. A retained whole-anchor result stays
paired with the whole-anchor plan, whose selection identifies the target member.
The context-owned `DeclaredJSONValue` observation follows those captured member
identities before materializing declared values. This preserves cookie text and
byte values without serializing the entire result object or running a JSON codec.
Direct controls cover selected members, absence/null, ownership rejection and
the existing opaque-value boundary. These changes implement the existing
selection/edge assumptions; they do not add a theorem about Go traversal.

The integration pass also exposed a traversal-domain mismatch: OpenAPI omits
security credentials from ordinary query, header and cookie parameters, but
example preparation tried to bind a synthetic session credential to a payload
member. Preparation and emission must share the same exact location/name
classification. Ordinary parameter binding errors must still fail; this is not
permission to ignore missing source ownership. The finite role model starts
after a documentation surface is selected, so it does not establish this
classification. A direct synthetic-session regression and the existing security
contract tests pass for the correction, along with the representation,
transport-IR and OpenAPI-IR suites and vet checks. The endpoint-preparation
helpers were moved to their own file to retain the repository's file-size limit;
the affected suite and vet checks pass again after that mechanical split. These
checks establish the exercised classification boundary, not a universal claim
about all security designs.

The accepted component reuse decision keeps occurrence-owned examples intact.
Existing complete-content comparison may stop sharing automatic request-body,
response, parameter and header definitions when their examples differ. No source
selection, projection or reuse algorithm changes for this decision. The models
do not prove component allocation or downstream client compatibility; exact
rendered comparisons, explicit-name checks and consumer generation cover those
implementation boundaries.

The AutoK comparison exposed a renderer boundary failure despite intact retained
metadata: the single-example fast path emitted a bare value even when its metadata
requested an explicitly named reusable component. The repaired guard keeps such
examples on the structured path. Direct IR cases cover named, unnamed and
whitespace-only names; `TestRenderedSingletonExplicitExampleNameCreatesComponent`
checks the component, reference, summary, description and value in OpenAPI 3.1/3.2
JSON and YAML. The ordered-group claims preserve the metadata entering this
boundary; they do not prove that the renderer uses it. No source-selection model
or theorem changed for this guard restoration.

The final repository batch also exposed an importer producer error. For a shared
object error schema, the importer emitted `Error(name, func() { Extend(base) })`.
With no explicit type, this selected the built-in problem type as the service
source, while HTTP body construction merged the imported object's fields. The
strict plan correctly rejected the unrelated member. Tracing confirmed a source
with problem fields and a target with the imported `message` field, rather than
lost copy ancestry. The repair gives each error an explicit object type clone,
with `Extend` inside `Type`, while keeping its canonical `OpenAPIBody` reference.
This restores the finalized-source-shape precondition at the importer. It does
not weaken ancestry checks or add implicit base flattening to captured
occurrences. Evaluated source-shape and generated-program tests must establish
this producer correspondence; the alias-pairing model assumes those shapes are
captured correctly and does not prove importer DSL emission.

See the [model run instructions and bounds](../../tla/schema_declaration/README.md).
Deterministic generation, fixture compilation, repository gates and independent
final-diff review remain required before #572 delivery.

## #565 CLI projection correspondence

The existing `Legacy.legacyByteReinterpretation` witness reproduces the relevant
design defect: text `hi` is not the JSON representation of bytes `[104, 105]`.
`byteStringDecodes` and `byteEncodingWitness` establish the concrete `aGk=`
control under the specimen codec. The accepted repair uses the existing retained
source result and runtime projection; it does not add a new source-selection or
wire-format policy.

The same obligation applies to direct JSON body flags that do not need a payload
builder. A collection routed through a query or header remains a location flag;
its JSON-shaped CLI input alone does not make it a request-body projection.

`ProjectionCorrectness.project_runtime_preservation` applies within the model
when the target plan describes the actual decoder. For #565, the production
boundary is the client body `TypeData.Value`: its plan is built after the emitted
layout is known. Routing its `Source.Example` through that plan must preserve
the selected branch and target-observable bytes. Generated HTTP and JSON-RPC
builder tests must check the advertised text and decoded service value; the
theorem does not prove emitted Go or command-line formatting.

This routing applies to flags actually decoded as JSON. A native byte flag can
use the plain `[]byte(text)` CLI conversion before the HTTP encoder runs; its
text contract must not be replaced with a base64 JSON literal solely because
the eventual HTTP body uses JSON. Missing or mismatched retained plans are
generation errors, not grounds for silently omitting the hint.

An unavailable serialized hint is distinct from a successful JSON `null` value.
Omission must reach both JSON diagnostic constructors, individual and aggregate
help, and the top-level example heading. Optional unavailable flags may be left
out of sample invocations; an unavailable required flag prevents advertising a
complete invocation. Commands and parsers remain available. These are consumer
correspondence obligations checked by focused tests, not new semantic theorems.
The generated HTTP and JSON-RPC regressions reproduce the previous rejection of
`{"bytes":"hi"}` and now accept the advertised `{"bytes":"aGk="}`, retaining the
`Data` branch and decoding `[]byte("hi")`. The affected CLI, example, HTTP and
JSON-RPC package suites pass, including optional-body and collection-default
controls. Direct tests cover omission, explicit null, numeric precision, both
diagnostic constructors and help surfaces. Existing semantic proofs are reused;
this adapter change does not alter their model.

## #434 gRPC CLI source preservation

`Legacy.legacyAuthoredReplacement` and `Legacy.legacySourceReplacement`
establish the existing concrete source-replacement counterexamples. The bounded
TLA+ `authored-replacement.cfg` fails `NoAuthoredResynthesis`; `checked.cfg`
preserves source, selected branch and observed presence under its stated
assumptions. These existing results support consuming the retained service
example instead of synthesizing again from the protobuf message schema.

The implementation obligation is to map `MethodData.PayloadValue` into the
allocated protobuf request message. Semantic member and branch identities
determine the retained value. Protobuf field names, oneof alternatives, message
wrappers and metadata visibility determine its target representation. An absent
optional union remains absent; its presence in the schema does not authorize
inventing either its branch or a sibling field's value.

This transport adapter is not proved by the shared JSON projector: that
projector does not implement `ValueCodecProtoJSON`. Actual generated CLI builder
and service conversion checks must establish field, branch, byte and numeric
preservation for the exercised protobuf cases. Incomplete, ambiguous or
unsupported examples require a diagnostic and omission, without removing the
command or advertising `null`; invalid authored values remain generation errors.
These are production correspondence obligations, not new theorem claims.

The explicit-null probe exposed an existing gRPC converter limitation:
an `Any` payload with `Nullable()` and `Example(Null())` generates assignments between
`loom.JSONValue` and `loom.Nullable[loom.JSONValue]` and compares the nullable
wrapper with `nil`. Those converters do not compile. #434 does not add nullable
Any transport support; its example adapter must diagnose and omit that unsupported
hint. Empty selected collection branches remain a separate supported case and
must retain their oneof selection through a generated round trip. Typed-nil
authored collection examples are rejected by existing DSL validation and do not
establish an executable protobuf acceptance case.

Focused #434 production checks pass: retained type/payload source precedence,
absent and selected unions, renamed fields, bytes, exact Int64 values,
unavailable-hint diagnostics and command availability. Existing generated
oneof fixtures and a selected empty-collection wrapper pass compilation and
CLI replay. The existing isolated-process CLI determinism check passes.

The parent `4857a3f3` and candidate comparison reuses `CLIProtoJSONDSL` and the
four existing `grpc-type-union`, `grpc-type-plain`, `grpc-payload-union` and
`grpc-payload-plain` probes. All four reproduce authored-value replacement at
the parent and decode exactly `{"id":"authored-id"}` with the candidate.
Build and vet pass at both revisions. Literal comparisons change only embedded
CLI example/help text; generated protobuf schemas, converters, types, endpoints
and service/example application files remain byte-identical. The selected
fixture's temporary module replacement path is the only non-generated metadata
difference. Four checked-in CLI goldens record changed sample values and Int64
spelling. These bounded results establish tested adapter correspondence, not
arbitrary protobuf equivalence or a full-corpus audit.
