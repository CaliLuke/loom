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
| Structural conformance | Libopenapi parses rendered JSON/YAML. The exactly pinned Vacuum build lints both formats for the representative specimens, including the full 3.2 fixture. The gate requires success and verifies rejection of malformed documents; no crash is counted as a pass. |
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

`make openapi-contract` installs the official Vacuum module
`v0.39.4-0.20261107034229-b6c4e203d2ea`, exact commit
[`b6c4e203d2eae41934087140a4d809dd1cce6d82`](https://github.com/daveshanley/vacuum/commit/b6c4e203d2eae41934087140a4d809dd1cce6d82)
into a temporary module with the repository Go toolchain. That module selects
the framework's `github.com/pb33f/libopenapi` version from the root module, and
the gate checks the installed binary's build information to confirm the version.
The current pin is [libopenapi v0.42.4](https://github.com/pb33f/libopenapi/releases/tag/v0.42.4),
whose tag is commit `f6e97f4e9e2ee7e2f0404064d50ffd67fcd0b182`.
It preserves absolute external reference origins when an OpenAPI document supplies
`$self` ([upstream #699](https://github.com/pb33f/libopenapi/issues/699)), and includes
[libopenapi PR #656](https://github.com/pb33f/libopenapi/pull/656).
This fixes enum value comparison for arrays, objects and numeric representations.
The parser uses its released version. Neither dependency uses a fork, source patch
or replacement directive.
The Vacuum pin lives in `v3/contract_vacuum_test.go`; this is a contributor
tool, not a runtime dependency. Run the existing gate so its CLI uses the same
parser version as the framework.
`v3/testdata/vacuum.yaml` extends the recommended rules. API descriptions are
optional in fixtures. The duplicate-enum rule is enabled; JSON Schema recommends
unique enum elements but does not require them. Enum controls verify acceptance
of distinct arrays, objects, scalar types and large numbers, and rejection of
equal values, including equivalent numeric representations. Literal properties
named `enum` inside an enum value remain data. The existing collection-enum
specimen must still pass, and the framework's direct collection/byte enum semantic
tests remain in place. No structural schema rule is disabled.
Other controls remove required document and device-flow fields, introduce an
invalid parameter location or schema type, and reintroduce conflicting nested
encodings. Assertions require the relevant rule diagnostic, not just process failure.

The Vacuum pin is a required-unreleased-correctness exception. Stable v0.39.3
rejects valid document-prefixed and external OpenAPI 3.2 Security Requirement
URIs ([upstream #1053](https://github.com/daveshanley/vacuum/issues/1053),
[upstream #1056](https://github.com/daveshanley/vacuum/issues/1056)). The pinned
head belongs to official draft [PR #1029](https://github.com/daveshanley/vacuum/pull/1029)
and includes dedicated external-security fix
[`c8d7f10d3472db624b17892eebf679471c922c47`](https://github.com/daveshanley/vacuum/commit/c8d7f10d3472db624b17892eebf679471c922c47).
The draft also contains broader autofix work; this gate only invokes linting,
never repair or write operations.

Return to the latest stable Vacuum release once it includes both security URI
fixes and the maintained same-document/external positives and missing-target
negatives pass with the selected parser. Keep the authored URIs, admitted-resource
boundary and security rule intact. A successful ordinary lint run alone does not
satisfy the generated external-reference identity specimen; its zero-error check
remains mandatory. No rule or specimen is disabled to adopt the pin.

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
