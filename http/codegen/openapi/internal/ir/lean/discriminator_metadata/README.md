# Discriminator metadata admission

Run `./check.sh` with Lean 4.35.0. The theorem manifest audits transitive axioms.
The Boolean model reproduces admission of an optional discriminator without a
fallback, then checks the cross-field rule and the 3.1 required-property policy.
All five theorems pass without axioms. A valid fallback is an assumption in this
model: Go tests establish syntax admission, local reference resolution, graph
retention, and rendering. This does not prove URI dereferencing or engine behavior.

`discriminator_metadata.go` owns metadata decoding before schema mutation.
`Analyzer.Err` checks local targets after analysis; `finalizeRepresentations`
checks before allocation. `mapSchemaReferences` includes the fallback graph edge,
and the v3 reference walker retains it during pruning. The existing version
router removes defaultMapping and requires the property in 3.1. Tests cover
nested and reusable unions, invalid input, required controls and external URI
preservation; external documents are not fetched or validated.

The design requires an authored fallback, never an inferred branch. Bare values
mean component names; local fragments must name emitted components. Explicit
relative paths and absolute URI references remain authored external references.
This keeps metadata meaning independent of the selected output version.

Validation: shared IR and v3 suites pass; the existing OpenAPI32FeaturesDSL
compiles after generation. Its 3.1/3.2 JSON and YAML bytes are unchanged from the
baseline and identical across independent generations. The original Redocly
2.60.0 run validated the 3.1 specimen. Rechecking with 2.62.1 still crashes on
the unchanged 3.2 feature specimen with
`Unknown type name found: string`. Libopenapi parses both; direct tests assert
the changed discriminator rules. Neither parser construction nor the known lint
failure is treated as full 3.2 conformance certification.

The active contract gate now uses Vacuum v0.36.2 and requires the full 3.2
specimen to lint successfully after correcting its separate nested-encoding
conflict. The historical Redocly results above remain evidence for the original
change. See the framework skill's OpenAPI validation reference for current scope
and validator limitations.

## Mapping regeneration and version projection (#1005 design)

`DiscriminatorMapping.lean` is audited by the same `./check.sh`. The historical
module keeps its manifest and its "no axioms" requirement. The new module's
manifest, `mapping-required-theorems.txt`, accepts only Lean's standard axioms
(`propext`, `Classical.choice`, `Quot.sound`). Any other axiom fails the
script, as does a missing theorem; both negative checks were run in scratch.

Contents:

- **Effective mapping, per OAS 3.2 §4.25.3 and Appendix G.3.**
  - An explicit entry wins.
  - Any other value is an entry-document component name.
  - URI values resolve through an abstract resolver, which is #957's canonical
    target.
- **Admission.** Every explicit value must resolve to a listed branch.
- **Regeneration.** The exporter emits the retained explicit entries as
  canonical local references.

`regenerated_effective_mapping` proves, for arbitrary component lists,
explicit mappings and resolvers, that the regenerated effective mapping equals
the source mapping for every key that is not the name of a component added by
import. That requires the regenerated resolver to map each canonical reference
back to its target. Three counterexamples support it:

- `prior_implicit_rule_counterexample`: revision 2's "implicit key only for
  unmapped branches" contradicts §4.25.3;
- `derived_keys_counterexample`: deriving exported keys from Loom branch names
  invents hints;
- `added_component_name_becomes_implicit_hint`: promoted or bundled component
  names add implicit hints, so the hypothesis is necessary.

Version rules (`version_rules_by_source_version`) take the importer source
version:

- 3.2: an optional property needs `defaultMapping` (MUST).
- 3.1.1: optional is allowed (SHOULD).
- 3.0.3: optional is rejected (MUST, §4.7.24.2.1).
- Only 3.2 has `defaultMapping`.
- Export targets are 3.2.0 and 3.1.1 only.

For the 3.1 projection:

- `union_required_is_redundant_for_required_branches` proves that adding the
  property to the union's `required` does not change validation when every
  branch requires it (tagged envelopes).
- `union_required_narrows_untagged_hint` shows that the same filter narrows a
  hinted untagged union, so it must not apply there.

Revision 5 added these theorems:

- `explicit_key_shadows_implicit`: with `oneOf: [Cat, Dog]` and
  `mapping: {Cat: Dog}`, the explicit key shadows the implicit name, and
  regeneration keeps it.
- `branch_name_validation_counterexample`: a validator that compares explicit
  keys with generated branch names rejects that valid document, so full-object
  and ParentHint keys are checked only against keys.
- `duplicate_alternatives_mapping`: `oneOf: [A, A, B]` with `a → A` is
  admitted, and regeneration emits one entry.
- ParentHint: `inherited_requiredness` (a child requires the property when its
  parent does), `inline_only_requiredness_counterexample` (deriving optional
  from inline child schemas misclassifies the typical `Cat`) and
  `parent_hint_version_rules`.

Exact-one runtime behavior for duplicate alternatives is #1014's matcher
obligation; it is not modeled here.

Observed with Lean 4.35.0 via `check.sh` (from a scratch copy):

- the build passes with `warningAsError`;
- 17 mapping theorems are audited; their axioms are drawn from `propext` and
  `Quot.sound`;
- the existing five theorems stay axiom-free.

Payloads are abstracted to two Booleans, and resolution is an abstract
function. The module does not prove URI normalization, #957 bundling, Go
rendering or exporter serialization; those are direct and round-trip test
obligations.
