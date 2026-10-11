# Bounded OpenAPI semantic validation

Keep these obligations separate. A parser or compiler success does not establish
instance acceptance, and a golden snapshot does not establish runtime behavior.
Use the existing suites and fixtures; do not default to the exported-design corpus.

| Obligation | Existing owner and evidence |
| --- | --- |
| Type/null instance preservation | `internal/openapiimport/semantic_preservation_test.go` compares source and regenerated schemas against explicitly accepted and rejected values. The existing nullable `Record` specimen covers required null, optional null, object and array types, and rejected type/missing-property values. The existing unconstrained specimen covers null, scalars, arrays and objects for both empty and Boolean-true schemas. |
| Strict admission | `schema_admission_test.go` rejects false schemas, implicit object typing and constrained null branches before normalization. `cookie_test.go` checks primitive cookies evaluate/generate/compile and rejects arrays/objects, including component references. |
| Generated compilation | `requireRenderedDesignGenerates` compiles accepted imports. The bounded `OpenAPIProblemLinksAsyncDSL` and `OpenAPI32FeaturesDSL` compile specimens cover the changed export paths. |
| Runtime boundary agreement | `ir/semantic_bounds_test.go` validates emitted map cardinality and unsigned/floating numeric bounds with a schema engine, then compiles and executes generated Go validation on the same boundary values. `analyzer_unsigned_format_test.go` separately asserts format identity and actual decoder limits, including values above signed maxima. |
| Regex engine agreement | `ir/pattern_projection_test.go` compares Go regexp and unflagged Node ECMA matching on anchors, ASCII whitespace, literal classes/braces and quantified assertions. Unsupported syntax must return diagnostics. |
| Discriminator semantics | `ir/discriminator_metadata_test.go` checks admission and URI references; v3 discriminator tests cover both targets, required controls, nested/reusable occurrences and fallback-only graph edges. The bounded Lean model records the cross-field rule, not URI or engine correctness. |
| Independent-process determinism | `requireIndependentRegeneration` invokes a new CLI process and compares exact JSON/YAML bytes for the nullable import. Export fixture snapshots are likewise compared across separate generator processes. |
| Structural conformance | Libopenapi parses rendered JSON/YAML. Redocly 2.60.0 validates the selected 3.1 fixtures and portable 3.2 fixtures, but crashes with `Unknown type name found: string` on the existing 3.2 feature specimen. Retain this limitation and direct rule assertions; do not count the crash or parser model construction as full 3.2 conformance. |

The existing `gojsonschema` dependency is used for the specimens' shared
Draft 7/2020-12 vocabulary only (type, null, properties, required, anyOf and
bounds). These tests do not certify the entire OpenAPI 3.2 vocabulary or every
instance. There is no new test engine or dependency version.

Run the owning suites with:

```sh
go test ./internal/openapiimport ./http/codegen/openapi/internal/ir ./http/codegen/openapi/v3 -count=1
```

For faster iteration, select the changed tests. Keep runtime compilation,
external linting and process determinism explicit in the acceptance record.
