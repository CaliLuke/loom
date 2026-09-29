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
test, coverage and final generation comparisons pass. Commit authorization
requires final independent review of the frozen diff and this evidence.
Shared Go target-constraint and source-traversal repairs passed independent
re-review. Alias-local and map-key constraints still lack complete formal
correspondence; the adapter rejects unsupported shapes explicitly.
The gate targets the candidate `Proofs` entry point. #570 wires the same
registered-corpus proof/conformance gate into local checks and CI.
Alias-local schema lowering is due in #574; cumulative alias/key-constraint and
key-enum lowering is due in #571, before those consumers rely on it.

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
```

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
