# Value contract model and proofs

This directory owns the formal source-value and target-projection model for the
[value-contract design](../../../roadmap/value-contract-design.md) and
[implementation plan](../../../roadmap/value-contract-plan.md). Start with the
[correspondence ledger](correspondence.md) for exact claims, assumptions,
counterexamples and production seams.

**Status:** The reviewed #569 candidate passed 483 required theorem audits,
fresh kernel replay and deliberate rejection controls. The #570 numeric and
map-identity checkpoint passed 522 theorem audits and fresh kernel replay, with
independent review. That checkpoint distinguishes canonical source keys from
native decoded-key identity and uses actual native-map codec correspondence.
The source-admission and native-byte-shape checkpoint passed 567 theorem audits
and fresh kernel replay, with independent review. Its production adapter passed
402 source comparisons. The subsequent no-callback control exposed reference
preflight requests for inadmissible numeric sources. The shared admission-aware
request collector now passes all 13 registered conformance assertion groups,
including those 402 comparisons and missing-codec negative controls. Full lint,
test, coverage and final generation comparisons passed. Final independent
review authorized the atomic #570 commit, pushed as `2f6bfbf7`.
Shared Go target-constraint and source-traversal repairs passed independent
re-review. Separate bounded adapters now check effective alias contracts, named
scalar key contracts and alias lengths; the general graph adapter continues to
reject unsupported shapes explicitly.
The #571 effective-alias checkpoint passed the 634-theorem audit, fresh kernel
replay and all five deliberate rejection controls. Its independent raw-layer
adapters and all 18 registered conformance groups pass. Final production
acceptance, including the 934-design retained comparison and its negative
controls, passed before #571 was committed and pushed as `1955dbc5`. The
[correspondence ledger](correspondence.md) records the exercised boundaries;
these checks do not certify arbitrary generated Go.
The gate targets the candidate `Proofs` entry point. #570 wires the same
registered-corpus proof/conformance gate into local checks and CI.
#574 delivered byte-alias length and schema lowering in `9e6b9121`; its
[production evidence](../../../internal/valuecontract/BYTE_SCHEMA.md) records
the generated comparisons and remaining representation boundaries. General
alias/key extraction obligations are tracked separately in the correspondence
ledger and must not be inferred from byte-length or effective-owner checks.

## Effective alias contracts: #571

[`NumericBounds.lean`](ValueContract/NumericBounds.lean) retains all four raw
numeric bounds and proves that choosing the tightest lower and upper endpoints,
with exclusive ties winning, is equivalent to their conjunction. Contradictory
bounds remain valid empty-set contracts. Its controls reproduce the former
inclusive/exclusive overwrite defects.

[`AliasContracts.lean`](ValueContract/AliasContracts.lean) models a finite raw
ancestry independently of the Go effective-constraint owner. Enumeration
acceptance is equivalent to the recursive refinement judgment: equal and proper
subsets succeed, while partial-overlap and disjoint declarations fail at their
first offending semantic member. On success, the effective enumeration is the
last explicit declaration. A present local default replaces an inherited one;
an absent local default preserves it; every selected default must satisfy the
effective enum and numeric contract. Required fields are unioned by finalized
identity, contain every raw identity, contain no duplicate identity and retain
the first provenance record in effective-to-ancestor order. Pattern and Format
clauses form a separate typed identity domain. They are stably de-duplicated in
current-to-base order with first/current provenance, and every remaining clause
is conjoined when admitting enum members and defaults. Every base-to-current
prefix validates an enum authored there under its modeled numeric and symbolic
predicate contract, while its selected default must satisfy the full effective
contract. A descendant cannot hide an invalid base declaration. An inherited
enum composes with later non-enum constraints as an intersection and is not a
new value assertion; a locally authored enum is. Valid explicit narrowing or
default replacement is not checked retroactively against constraints authored
later. The predicate IDs model
independent acceptance facts; Lean does not interpret regex or format syntax.

Authored enum and default values enter the model as independently resolved
semantic-class IDs plus optional exact decimals. The Lean theorem therefore
assumes correct declared-type resolution and equality; it does not compare raw
literals or serialized JSON. The Go conformance adapter under
[`internal/valuecontract`](../../../internal/valuecontract/) supplies an
independent normalizer for the exercised primitive, Bytes, Any, array, object,
map and nullable cases. It preserves raw declaration absence, duplicates and
provenance and compares the candidate only after constructing the reference
input. Exact signed/unsigned integers, Float32 normalization, Bytes text/byte
equivalence, recursive Any equality, collection equality and null are explicit
controls. Representative non-nil struct and pointer values are additional
raw semantic atoms for `Any`, compared independently by exact Go type and deep
equality through root, multi-hop and nested contracts. For declared Objects,
the adapter independently reflects codec-free structs and non-nil pointers over
the reviewed exported-field, JSON-name, anonymous-embedding, ambiguity,
snapshot and cycle subset, then applies normal child coercion. Recognized JSON
or text codec method sets stay opaque and no method is invoked. This Go host
materialization boundary is tested correspondence, not a Lean theorem and not
a claim about codec, serialization or projection correctness.

The Go adapter independently evaluates representative RE2-compatible patterns,
UUID and RFC3339 date-time values and compares the candidate's typed clause
order, exact text and original declaration provenance. This establishes the
named cases, not equivalence with every production regex or format validator.
Go graph extraction, DSL finalization and diagnostics, arbitrary custom values,
unions, codecs, float-to-decimal conversion, length and shape validation remain
tested boundaries rather than Lean theorems. Primitive field-template copying is also outside the
named-type ancestry judgment; a copied occurrence can author its own contract,
while an actual restricted named ancestry must satisfy refinement.

The `named-key-contracts` group independently extracts raw scalar key ancestry,
including enum absence and duplicates, default provenance, all four numeric
bounds, typed predicates and each length bound. It checks authored enum and
selected default admission at every ancestry prefix and compares whole-map
resolution. Source `Any` key equality retains Go host identity; canonical member
names and native decoded-key identity are checked separately. Constrained
projection controls compose independent key admission with the key codec
reference. The `effective-alias-lengths` group compares raw String, Array and Map
alias bounds with the existing length reference, effective constraints,
resolution and projection. Rune counting and collection cardinality remain
tested Go extraction boundaries.

The key-domain comparison exposed a capture defect: a present-empty authored
enum became absent and admitted a member. Shared capture now preserves its
presence. Direct controls check scalar roles, named inheritance, widening
rejection, map keys and detached snapshots. The existing model already
distinguishes empty domains from absent constraints; this repair changes the
implementation correspondence, not the accepted enum policy.

## Map collision diagnostics: #456

[`Issue456CollisionDiagnostics.lean`](ValueContract/Issue456CollisionDiagnostics.lean)
defines collision evidence as an unknown flag and a list of known witnesses.
Combining child evidence preserves every witness, including when another child
is opaque or recursive. Unknown evidence cannot justify rejecting a value, but
it cannot erase an independently established collision either.

The model identifies map collisions by duplicate declared member names after
key conversion. Its Float32 control reproduces a collision that comparison of
raw values or their original spellings misses. Its object control shows how
right-biased insertion loses duplicate names; rejection must precede insertion.
Length, requiredness and enum predicates do not participate in this diagnostic.

For an implicit union, rejection requires known eligibility, at least one
compatible alternative, and a collision in every compatible alternative.
Unknown eligibility or a compatible alternative without a known collision
prevents that universal claim. An explicitly selected branch remains
authoritative. Witnesses retain structural paths and branch identities; bounded
controls check presentation order without conflating dotted path components.

The 29 claims in [`required-theorems-456.txt`](required-theorems-456.txt) guide
the Go implementation; they do not prove its reflection or traversal code.
Recognizing custom codecs without invoking them, detecting host cycles,
extracting object members, and matching Go source compatibility remain tested
correspondence obligations. The compatibility model proves equivalence with its
finite judgment when supplied sufficient structural depth; it represents cycles
and opaque leaves explicitly as unknown.

The audited gate below checks the separate #456 manifest and replays
`ValueContract.Issue456CollisionDiagnostics` with a fresh kernel. Production
acceptance additionally requires the direct Go controls and generated-output
comparisons; a successful proof gate alone does not establish delivery.

## Byte-schema migration: #574

[`AliasLengthBounds.lean`](ValueContract/AliasLengthBounds.lean) proves that
intersecting every local decoded-length constraint on a finite alias chain
preserves their conjunction. [`ByteLengthProjection.lean`](ValueContract/ByteLengthProjection.lean)
proves residue bounds, clamping, empty-branch omission, encoded-length arithmetic
and its composition with an independently stated base64 grammar. The 19 new
statements brought the #574 checkpoint manifest to 586 theorems; the axiom audit
and fresh kernel replay passed. The current #571 manifest contains 634 theorems.

The executable reference exposes these same functions through `aliasLengths`
and `byteLengthSchema`. The new Go conformance groups compare the real shared
`internal/byteschema` helper, source resolution and target projection with the
reference. General non-length alias/key constraints remain explicitly outside
this lowering; their existing rejection controls remain required.

Actual JSON codec acceptance, ECMA-262 regex execution, schema adapter traversal,
component identity and emitted Go are still tested boundaries. The generated
HTTP/JSON-RPC and Ajv checks run in `make openapi-contract`. #574's repository
gates, registered 72-probe comparison, final-source 499-case comparison and
independent review passed before `9e6b9121` was pushed. That tested production
evidence is separate from the arithmetic proof; neither establishes universal
correctness of emitted schemas.

## Architecture and navigation

The pipeline preserves an authored source and its selected branch, resolves a
semantic value, observes what a target exposes, constructs its canonical wire
value, then checks that same wire against the schema and runtime decoder.
Observation finishes before construction: an omitted optional child must not
fail a required-field check for a wire object that will never be emitted.

All module names below refer to files under [`ValueContract/`](ValueContract/).
The independent judgments describe the accepted behavior without assuming that
the candidate evaluator succeeds.

| Stage | Executable or data model | Independent specification | Proofs |
| --- | --- | --- | --- |
| Source precedence and provenance | `SourceSelection` | Eligibility and source-order predicates in the same module | Theorems in `SourceSelection` |
| Declarations and target plans | `CandidateModel`, `GraphValidation` | Well-formed declaration and target predicates | Theorems in `GraphValidation` |
| Concrete source primitive admission | `SourceAdmission`, `SourceScalar`, `coerceSourceScalar` | `SourceAdmits`, `SourceCoerces` | Universal admission/coercion equivalence and `SourceAdmissionControls` |
| Native byte source shape | `SourceBytes`, `NativeByteSequence.inputs`, `arrayInput` | `NativeByteOctets`, `NativeBytesAdmitted`, `NativeByteValue`, `ArrayInputEntries` | Source/raw correspondence plus `SourceByteControls` |
| Source interpretation and typing | `Resolve` | `ResolutionSpec`, `Typing`, `SourceBodySpec`, `SourceUnionSpec` | `ResolverProofs`, `SourceResolutionProofs`, `SourceBodyProofs`, `SourceUnionProofs`, `SourceMissingProofs` |
| Complete source outcomes | `Resolve` | `SourceFullOutcomeSpec`, composed from the `Source*OutcomeSpec` relations | `SourceFullOutcomeProofs`, `SourcePublicOutcomeProofs`, supported by `Source*OutcomeProofs` |
| Source depth and recursive graph budgets | `Resolve`, `ValueDepth` | Input depth, expansion ranks and independent raw outcomes | `SourceRawProofs`, `SourceNullProofs`, `SourceBudgetProofs`, `SourceGraphBudgetProofs`, `SourceStabilityProofs`, `SourceEqualityBudgetProofs`, `SourcePreferenceProofs` |
| Visibility and field presence | `ProjectionObservation` | `Observation`: `Observe`, `FieldPresence`, `EmptyObserved` | `ObservationExecutionProofs`, `ObservationProofs`, `EmptyProofs`, `ValueBudgetProofs` |
| Canonical wire construction | `CanonicalConstruction` | `Canonical` | `CanonicalConstructionProofs`, `CanonicalProofs` |
| Schema admission | `SchemaValidation` | `SchemaSemantics` | `SchemaProofs` |
| Runtime decoding | `Decoding`, `KeyDecoding` | `DecoderSemantics` | `RuntimeProofs` |
| End-to-end projection | `ProjectionBuild`, `Projection` | `Representation`: `Representable`, `RuntimeObligation` | `ProjectionProofs`, `ProjectionCorrectness` |
| Raw JSON materialization | `Materialization`, `MapKeys` | Materialization and JSON-validity relations in `Materialization` | `MaterializationProofs`, `MaterializationValidity` |
| Value and wire equality | `ValueEquality`, `TargetEquality`, `WireEquality`, `StrictValueDepth` | Independent equality relations in those modules | `EqualityBudgetProofs`, `StrictEqualityBudgetProofs`, `SourceEqualityBudgetProofs` |

`resolve_outcome_iff` covers the full source result, including failures, selected
branches and ordered missing-member paths. Source outcomes have adequate finite
derivations and are unique across adequate depth/rank choices. Successful
enum/default resolution retains no missing required-member paths.
`project_emitted_iff` and `project_progress` establish target correspondence and
progress from independent representability, including legitimate field loss.
Component checks are not a substitute for the final audited manifest and review.

Public depth-indexed judgments quantify over finite derivations without a fixed
maximum value depth. Internal indexed source judgments expose exhausted runs;
their public relations admit only structurally adequate derivations. Budget
adequacy and result stability cover negative outcomes as well as success.

Source selection consumes already extracted source groups: local attributes,
ordered references, ordered bases, then the type. It selects the last example
within the first nonempty group. Authored examples take precedence over synthesis
suppression; raw null and empty values remain authored values. Extraction from
the actual DSL graph is a separate production boundary.

Source equality, target equivalence and exact snapshot equality serve different
contracts. In particular, numeric wire equivalence cannot stand in for exact
lexical snapshot equality. Raw `Any` values can carry adapter-assigned host
identity classes that preserve Go `reflect.DeepEqual` distinctions. These IDs
never reach the wire and cannot bypass invalid-input or map-collision checks.

## Counterexamples and controls

[`Model.lean`](ValueContract/Model.lean) and
[`Legacy.lean`](ValueContract/Legacy.lean) retain the initial foundation and
proved abstract counterexamples: selected-branch erasure, equal-shaped branch
misidentification, literal bytes sent to a base64 decoder, and authored value or
provenance replacement. They are not a simulation proof of historical Go.

The wire model has one string constructor. Bytes `hi` and string `aGk=` can have
the same wire string, so a private type tag cannot establish untagged branch
uniqueness. The original finite specimen codecs cover only their named witness
strings. They do not prove the complete base64 grammar or length projection.
The witnesses include both directions of the decoded-byte versus encoded-text
length mismatch and a noncanonical base64 spelling accepted by a decoder but
rejected by a canonical schema enum.

Candidate controls are grouped by responsibility:

- `NumericRepresentationControls`: exact numeric equality with distinct float
  widths and signed-zero spellings, plus the rejected numeric-equality key
  uniqueness rule. The reviewed map-identity amendment uses canonical names.
- `NumericCoercionControls`: declared precision before validation and selection,
  all-role normalization, normalized-key collisions, and source-specific numeric
  literal origins for named formatting methods. Formatting/parsing callbacks
  remain an explicit production boundary. Normalization does not establish
  concrete Go source-type admission; that separate check precedes coercion.
- `NumericTargetControls`: destination-owned integer width/signedness and float
  precision, distinct scalar/member-name parsers, signed zero, same-wire union
  matching and independent schema acceptance. Codec implementations remain
  external; the Go target corpus checks 420 real-codec cases.
- `MapIdentityControls`: canonical source/enum keys, native decoded-key
  uniqueness, mixed-width keys, signed zero, and raw Any host separation.
- `SourceAdmissionControls`: concrete host eligibility before normalization,
  map-key and object-name evidence, and explicit abstract-control boundaries.
- `SourceByteControls`: byte/array competing candidates, slice/array/container
  admission, original child host evidence, nil presence, raw binary meaning and
  the rejected scalar-erasure counterexample. The integrated 567-theorem
  checkpoint and 402 production source comparisons passed independent review.
- `ResolutionControls`: authored matching, nullability, source equality and
  whole-node enum admission.
- `CandidateControls` and `ProjectionControls`: target behavior, schema/runtime
  distinctions and the rejected enum-gate placement.
- `HostMaterializationControls`: preservation of host equality evidence without
  weakening raw-input validation or leaking evidence into wire values.
- `PresenceControls`: required versus optional field construction, including
  a schema alternative that must not excuse a missing selected-branch field.
- `OmissionControls` and `LegacyProjectionBuild`: a representable optional
  omitted child that the former combined observation/construction pass rejected,
  plus successful candidate and retained-child rejection controls.

Keep the rejected behavior and the corrected behavior distinguishable. A witness
can establish a defect without establishing its repair.

## Run the audited gate

Lean **4.34.1** is pinned in [`lean-toolchain`](lean-toolchain). It includes
`lake` and `leanchecker`; no proof-library or generated-service dependency is
added. With elan installed:

```sh
export PATH="$HOME/.elan/bin:$PATH"
elan toolchain install leanprover/lean4:v4.34.1
# From the Loom repository root:
make value-contract-proof
```

The Make target runs `scripts/check_value_contract_proof.sh`, then the gate tests
with `LOOM_VALUE_PROOF_TEST=1` so real rejection controls cannot be skipped.
The script checks the actual toolchain version, runs `lake build`, then runs
these commands from this directory:

```sh
lake env lean -DwarningAsError=true ValueContract/AxiomAudit.lean
lake env leanchecker --fresh ValueContract.Proofs
lake build ValueContract.Issue456CollisionDiagnostics
lake env lean -DwarningAsError=true ValueContract/AxiomAudit456.lean
lake env leanchecker --fresh ValueContract.Issue456CollisionDiagnostics
```

The #571 legacy controls are expected to fail because each asserts a rejected
behavior as if it were valid:

```sh
lake env lean NegativeNumericOverwrite.lean
lake env lean NegativeNumericClosedTie.lean
lake env lean NegativeEnumOverride.lean
lake env lean NegativeNearestPattern.lean
lake env lean NegativeNearestFormat.lean
```

Each command must exit nonzero at its stated false obligation. A successful
compile would mean the corresponding overwrite, closed-tie, enum-override or
nearest-only predicate defect had returned.

[`required-theorems.txt`](required-theorems.txt) is the reviewed claim manifest.
`AxiomAudit` uses Lean's `collectAxioms` for each required theorem, including
imported dependencies. Only `propext`, `Classical.choice` and `Quot.sound` are
allowed. The gate rejects missing tools, command failures, warnings, missing
required theorem reports, empty audits and forbidden transitive axioms.
Fresh kernel replay supplements the audit: a kernel can accept a declaration
that depends on a declared axiom, so replay alone is insufficient.

Ordinary Go tests check orchestration with controlled tool outputs. The enabled
external-tool tier builds temporary copies containing an admitted intermediate
theorem, a custom axiom behind an intermediate theorem, or a missing required
theorem. The real gate must reject each. Enabled mode fails if Lean is absent;
ordinary Go test success does not claim to have run Lean.

The mandatory production gate is:

```sh
make value-contract-conformance
```

It runs the proof dependency, builds `value_contract_reference` from the proved
functions, runs enabled differential tests and verifies both Go test events and
a fresh JSON report. Report counts are executed assertion groups, not corpus
input counts. Missing tools, failed/skipped tests, empty reports and absent owner
coverage fail the gate. CI uses the same target and pinned toolchain. Unsupported
DSL translations fail explicitly in adapter controls; they are not silently
skipped or counted as differential coverage. Each consumer migration must extend
the registered corpus before relying on a newly translated contract.
A direct enabled test invocation must also set `LOOM_VALUE_CONFORMANCE_REPORT`
to an absolute report path; the Make target supplies and cleans its own path.

## Production boundaries and maintenance

The model does **not** prove that arbitrary generated Go always compiles. These
boundaries require executable conformance and compiler evidence in #570 and
subsequent migrations:

- DSL extraction, occurrence identity and derivation of target/representation
  plans, including copied attributes, views and selected bodies;
- Go host identity assignment and numeric coercion/precision;
- rendered path-qualified diagnostics, including array indices and map keys;
- actual lexical codecs, regex checks, JSON/protojson behavior and schema
  validators;
- generated Go types, pointer conversions, defaults and transport compilation.

The actual Go controls live in `scripts/value_contract_*_test.go`. The
[correspondence ledger](correspondence.md) maps these boundaries and records
remaining assumptions. Documentation-only targets have no runtime-decoder claim;
runtime-backed targets must check the same canonical wire against the actual
modeled decoder and preserve its observed value.

Source missing paths are ordered lists of member identities, while failures
carry semantic categories. Exact correspondence for those modeled outputs does
not prove the text or collection locations of production diagnostics.

When a weakness is found, update its independent specification or counterexample,
the affected correspondence claims and the production acceptance cases before
relying on the proof again. Run the audited gate and obtain independent review
of the changed scope. Follow [AGENTS.md](../../../AGENTS.md) for manual cleanup:
keep durable proof/model sources, configuration, manifests, toolchain pins and
concise findings; remove task-owned compiled proofs, traces, logs and probes
when validation and review finish and no active task needs them. Installed
toolchains and shared caches are not task-owned output.
