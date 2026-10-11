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
| Structural conformance | Libopenapi parses rendered JSON/YAML. Vacuum v0.33.0 lints both formats for all 20 representative specimens, including the full 3.2 fixture. The gate requires success and verifies rejection of malformed documents; no crash is counted as a pass. |
| Encoding admission | Shared IR rejects named `encoding` combined with positional `prefixEncoding` or `itemEncoding` at the same object. Direct metadata tests cover nested scopes and index aliases; rendered tests cover requests, successful responses and errors in both targets. |

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

## External lint gate

`make openapi-contract` installs [Vacuum v0.33.0](https://github.com/daveshanley/vacuum/releases/tag/v0.33.0)
into a temporary directory with the repository Go toolchain. The module pin lives
in `v3/contract_vacuum_test.go`; this is a contributor tool, not a runtime dependency.
`v3/testdata/vacuum.yaml` extends the recommended rules. API descriptions are
optional in fixtures. The advisory duplicate-enum rule is disabled because Vacuum's scalar-value
comparison incorrectly reports distinct array enum values as duplicates.
JSON Schema recommends unique enum elements but does not require them. The
existing collection-enum specimen must still pass, and the framework's direct
collection/byte enum semantic tests remain in place. No structural schema rule
is disabled.
Other controls remove required document and device-flow fields, introduce an
invalid parameter location or schema type, and reintroduce conflicting nested
encodings. Assertions require the relevant rule diagnostic, not just process failure.

Redocly 2.62.1 crashed on the full 3.2 specimen before linting. The replacement
exposed an invalid combination of named and positional nested encodings in that
specimen; its named encoding now occupies a separate prefix object. This keeps
all feature families exercised. Generator comparisons cover both targets and
formats; only that fixture's encoding metadata changes.

The Python `openapi-spec-validator` 0.9.0 trial passed 19 of 20 original specimens.
Besides the encoding conflict, it rejected `allowReserved` placements and
`defaultMapping` under the advertised dialect. It is not part of this gate:
the normative 3.2 prose identifies the 3.1 dialect URI, while the Python tool
uses different 3.2 schema resources. Do not change authored contracts just to
match a validator's interpretation. Vacuum shares the libopenapi ecosystem, so
these checks supplement direct semantic and runtime tests; they are not an
independent proof of all OpenAPI semantics.
