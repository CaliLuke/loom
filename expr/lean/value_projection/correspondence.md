# Proof and production correspondence ledger

Status: **M1 foundation and #569 candidate proofs reviewed and pushed; #570 registered production correspondence and full integration gates pass. Reviewed checkpoints are recorded below; atomic delivery requires final review of the frozen diff.** The reviewed contract is
[value-contract-design.md](../../../roadmap/value-contract-design.md).
The source-history reference is `f5b786b39675e2b5c1466f04f3301b7b337779b6`.
The word “proved” below applies to the Lean statement, not to handwritten Go,
generated programs, all JSON strings, or any codec implementation.

## Established statements

All names below are in `ValueContract.Legacy`. `required-theorems.txt` enumerates
the individual obligations, and the gate prints their transitive axiom sets.

| Theorem names | Status and domain | Production seam / additional evidence |
| --- | --- | --- |
| `erasedBranchesIndistinguishable`, `legacyBranchLoss` | **Proved**: two concrete selected branches erase to one payload; first-branch reconstruction changes the second identity | `expr/types_union.go` source selection and `expr/example_canonicalization.go` reconstruction; #566 nested-branch probe supplies independent generated evidence |
| `legacyByteReinterpretation`, `byteStringDecodes`, `byteEncodingWitness` | **Proved** for the concrete `hi`/`aGk=` specimen codec; literal `hi` does not decode as the original bytes | Historical OpenAPI byte conversion; current authored-byte CLI bug in `codegen/cli/conversion.go`; #566 actual generated builder/decoder probe |
| `byteTextWireCollision` | **Proved**: the same `Wire.text "aGk="` decodes as Bytes and String | Untagged OpenAPI matching in `http/codegen/openapi/internal/ir/document_examples.go`; candidate same-wire uniqueness is proved below; actual-decoder correspondence remains #570 |
| `legacyLengthRejectsValidBytes`, `legacyLengthAcceptsShortBytes` | **Proved** concrete mismatches between decoded byte count and ASCII wire-string length under `boundaryCodec`; neither direction can use copied length bounds | `expr/json_schema_inline.go`, OpenAPI IR `analyzer.go`, and generated decoded-length validation. #574 has independent generated-server/schema reproductions; repaired arithmetic and complete lexical equivalence remain pending |
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
| Copying byte-count bounds onto base64 string length rejects valid values and accepts short decoded values | Two registered concrete Lean counterexamples above; actual generated HTTP server and inline/OpenAPI discrepancy reproduced for #574 | #575 specifies the policy; #569 proves the candidate semantic relations under explicit codec boundaries; #574 implements shared schema arithmetic/grammar. Production correspondence remains pending and cannot assume the correction landed |
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
| Specialized traversal paths can bypass occurrence-owned constraints | Independent review found map-key enum/alias rules and documentary alias-collection length gates missing. Shared body/local-constraint/enum ordering repairs and the subsequent positive Any-key enum correction pass direct tests and independent re-review | **Pending #574/#571 extraction correspondence:** Go key-domain observation is separate from ordinary Any value observation and its fixes passed independent review. Before #574 relies on alias schema/length claims, translate alias-local schema gates. Before #571 relies on enum/default claims, translate cumulative alias/key constraints and key-domain enums. The existing graph adapter rejects unsupported shapes explicitly; these cases are not counted as passing differential coverage |
| Source graph traversal must preserve entries and terminate even for rejected input | Independent review reproduced a NaN map key silently erased by MapKeys/MapIndex, cyclic synthesized raw data traversed again without a guard, and custom defined-byte elements bypassing the raw child boundary | Shared MapRange key/value capture, cycle-aware owned copying and exact native-byte classification have red/green direct tests and passed independent re-review; finite builtin Lean inputs do not prove arbitrary Go graph traversal or custom callbacks |
| Raw Go `any` decoding is not Loom's generated Any carrier | [Actual JSONValue controls](../../../scripts/value_contract_any_test.go) preserve exact large numeric text through `loom.JSONValue`; typed bytes and literal base64 text still materialize to the same JSON string | Candidate `Materializes` and `jsonSnapshot` record target JSON meaning independently of raw-source equality. Universal materializer correspondence is checked in `MaterializationProofs`; candidate projection composition is checked; #570 carrier correspondence passes for the registered corpus; later consumer coverage remains required |
| Source candidate preference includes compatible non-object alternatives, and cross-member authored/wire aliases can conflict | [Preference/null controls](../../../scripts/value_contract_source_test.go) preserve current non-object ranking; [alias controls](../../../scripts/value_contract_alias_test.go) retain the actual wrong-type overlap witness | Candidate resolver validates every retained assignment and preserves supported name overlaps. Full resolver correspondence is checked; #570 source/alias regressions and registered correspondence pass; the public Go compatibility adapter remains a separately tested boundary |
| Schema acceptance, decoder unknown-member policy and semantic retention were conflated in one target flag | `TargetDeclaration` now separates schema openness, decoder rejection and target extra-member retention. `ProjectionControls` exercises a schema-unique wire with two runtime interpretations, rejection for runtime use and permitted documentation-only emission | #570 must derive all three policies from the actual occurrence/codec. The defaults match ordinary and generated tagged JSON decoding; no schema-only uniqueness inference is permitted |
| Map key constraints and lexical key decoding differ from emitted map schemas | Current IR `analyzeInlineMap` emits an object value schema without key constraints. Candidate schema follows that shape; runtime decoding separately checks key parser acceptance, constraints and decoded-key collisions | Numeric key formatting shares `NumericCodec`; key-parser functions have a separate lexical boundary from JSON number-token parsing. #570 must test both, including aliases; canonical encoder membership never substitutes for decoder acceptance |
| Logical service fields can share a JSON alias when their actual transport places them in different components | The required gRPC mapped-metadata matrix exposes the new occurrence builder imposing JSON wire uniqueness before a transport plan exists | Shared Go capture must preserve unique authored member identities; supplied duplicate wire aliases must be ambiguous, while existing cross-namespace authored/wire overlaps retain their validation and precedence. Target emitted-name uniqueness remains plan-owned. The Lean `WellFormedDeclarations` premise currently requires unique wire aliases: these additional Go graphs need explicit adapter rejection and direct/generated tests, not a false differential claim. #573 owns independent lowering, or an earlier migration if it needs that correspondence. `value_alias_ownership_test.go` verifies authored keys, ambiguous aliases, cross-namespace compatibility and target visibility; the actual adapter rejection subprocess and 16-service generated gRPC metadata round trips pass. Independent scoped review passed. The final parent/candidate comparison is byte-identical for mapped-metadata, and whole integration gates pass |
| A projected collection can inherit stale naming provenance from its source declaration | Required meal-planner output changed `RecipeResponseSummaryCollection` to `RecipeCollection` after copying source occurrence metadata | Shared view projection must preserve occurrence constraints while assigning naming provenance to the new derived declaration. Four direct view/canonical-name controls, race checks and unchanged meal-planner JSON/YAML goldens pass; independent scoped review passed. The final parent/candidate meal-planner comparison is byte-identical. This is DSL-to-plan naming evidence outside the semantic theorem, not grounds for a renderer exception or a new proof vocabulary |
| A named Bytes type shared by nullable and non-null result occurrences can corrupt view representation planning | The retained #570 nullable-view regression is outside the Lean table validator: the model receives explicit occurrence/target identities and does not construct them from shared Go attributes | #570 isolates effective occurrence presence and physical representation. Direct tests and generated nullable-view/child-view probes pass, with the original parent failure retained in the comparison record. This is tested adapter/plan derivation, not a semantic theorem |
| Initial Lean evaluator branch returns bypassed whole-node enum gates | Universal schema correspondence exposed a candidate defect before approval; independent spec rejected the empty enum while the initial array evaluator accepted. `ProjectionControls.legacyHoistedEnumCounterexample` retains the rejected do-block layering, and repaired controls cover array/object/map/union/Any gates | Schema, decoder and resolver now compute their bodies in separate helper functions before uniform enum validation. All affected component proofs/controls have been rebuilt; independent exact-diff review remains required; this is a candidate implementation repair, not an assumption narrowing representability |

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
Full integration gates and generation comparisons pass; final commit review is
required. Alias-local schema mapping is due
in #574 and cumulative key-constraint/enum mapping in #571. The old M1 gate alone does not establish candidate
correctness. `Resolution.missing` preserves an ordered list of member-identity
paths; `Failure` carries classification only. Production array-index/map-key
paths and rendered diagnostics remain separately tested adapter obligations.

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
intended differences and outcomes. Atomic delivery requires independent review
of the frozen final diff; this record does not grant that authorization.

**Consumer migrations:** extend the registered corpus for each newly translated
contract, discharge the explicitly assigned extraction boundaries above, validate
real rendered schema instances and actual decoders, and compare generated
build/vet results and bytes in isolated processes. This remains **tested
correspondence**, not a proved Go refinement. Neither the M1 witness gate nor
#570's fixed corpus establishes completion of these later obligations.
