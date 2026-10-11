# Discriminator metadata admission

Run `./check.sh` with Lean 4.34.1. The theorem manifest audits transitive axioms.
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

The active contract gate now uses Vacuum v0.33.0 and requires the full 3.2
specimen to lint successfully after correcting its separate nested-encoding
conflict. The historical Redocly results above remain evidence for the original
change. See the framework skill's OpenAPI validation reference for current scope
and validator limitations.
