# Parameter codec acceptance (#968)

Parent: `f7a93576833c70eaa00d7a5bb9884f83b55ec15d`.
Checkout: `/Users/luca/.codex/worktrees/parameter-styles/loom` (initially clean).
Scratch: `/tmp/loom-968.GRZPUB`.

Design revision 5 and the complete execution plan were read before implementation.
The focused re-check confirms DB-1 complete dot protection, DB-2 key/value domains,
and DB-3 presence-based Encoding Object precedence. The independent readiness
review is recorded in the design. Implementation exposed a Bytes text-policy
conflict. The user subsequently delegated the choice: raw text is the selected
contract, with JSON Bytes retaining base64 encoding. Historical pending-policy
notes below describe the state when those checks ran. Integration onto the new
main parent is recorded separately before delivery.

## Workflow recital

1. M1: record every inventory disposition, parent outputs for twelve representative
   generated designs, six import controls and 21 acceptance/rejection reports.
2. M2: compile-safe owner and adapter stubs, behavioral red tests, implementation,
   then affected packages. Compile group A replaces Map across callers together.
3. M3: runtime stubs, failing domain/carrier tests, implementation and runtime gates.
4. M4: admission and generated tests first; compile group B moves IR and generator
   fields together, then codecs and HTTP/JSON-RPC mounted wire cases.
5. M5: structural and rendered export checks for both targets, libopenapi and pinned
   Vacuum v0.36.2. Original unrestricted-cookie inputs remain rejection controls.
6. M6: import acceptance/rejection, round trips and adapter agreement; compile group
   C retires Select/Mode after the last caller moves. Text unions await #1014.
7. M7: identical parent/change inputs must differ only by ID-1 through ID-4;
   independent-process byte comparisons, bounded compilation and integration-test.
8. M8: finalize docs, skills and model correspondence before make test, fmt, lint,
   lint-docs and affected race checks. Fresh exact-final-diff review resolves every
   finding before one atomic commit on main, followed by push and cleanup.

Commands follow those milestones: parent generation-determinism helper, bounded
test-testdata-compile, CLI build/import reports, owner go build and red/green tests,
affected packages, runtime tests, generated HTTP/JSON-RPC tests, OpenAPI contract
gates, importer round trips, retirement search, byte comparisons, integration-test,
final gates, staged review, commit and push. Every temporary-module command uses
explicit LOOM_DIR for the intended parent or change checkout. Determinism compares
exact output across processes. No edits follow the authorizing review.

No gen edits, proof tools in gates, forbidden Git cleanup/reset/checkout/stash,
dependency development or unrelated preparation edits. Files stay at most 1000
lines; type references use NameScope. Main's coordinator changes, .mcp.json and
other tickets remain untouched. Model evidence is reused within its assumptions;
implementation tests establish correspondence.

## Results

In progress. No implementation acceptance claimed. The parent copy has empty Git
metadata for source discovery and identical F-1/F-2 fixture changes. The existing
generation helper includes the twelve planned designs. Inventory: 252 matches,
with dispositions pending review. Raw artifacts remain outside the repository.

## Inventory dispositions

Every line in the recorded 252-match inventory is assigned below. Line numbers are at the parent.

| File | Lines | Disposition |
|---|---|---|
| `.agents/skills/loom-framework/SKILL.md` | 305, 1330, 1335, 1790, 1800 | M8: update public contract or owner/model guidance. |
| `.agents/skills/loom/SKILL.md` | 1075 | M8: update public contract or owner/model guidance. |
| `docs/dsl-reference.md` | 955 | M8: update public contract or owner/model guidance. |
| `docs/openapi-import-coverage.md` | 145, 160 | M8: update public contract or owner/model guidance. |
| `dsl/meta.go` | 267 | M8: update public contract or owner/model guidance. |
| `dsl/oneof_test.go` | 230 | Unchanged: existing scalar, simple-path, native JSON/root-form or union contract remains a regression control. |
| `dsl/untagged_maps_test.go` | 93, 96 | Unchanged: existing scalar, simple-path, native JSON/root-form or union contract remains a regression control. |
| `dsl/untagged_nested_test.go` | 80 | Unchanged: existing scalar, simple-path, native JSON/root-form or union contract remains a regression control. |
| `expr/http_endpoint_internal_test.go` | 105, 145 | M2 adapter/compile group A; M4 admission and affected message tests. |
| `expr/http_endpoint_mapping.go` | 81, 82, 106, 107, 112, 114, 128, 154, 159, 163, 172, 176 | M2 adapter/compile group A; M4 admission and affected message tests. |
| `expr/http_endpoint_test.go` | 378, 382, 386 | M2 adapter/compile group A; M4 admission and affected message tests. |
| `expr/http_endpoint_validate.go` | 346 | Unchanged: existing scalar, simple-path, native JSON/root-form or union contract remains a regression control. |
| `expr/http_header_collection.go` | 105 | M2 adapter/compile group A; M4 admission and affected message tests. |
| `expr/http_media_encoding.go` | 22, 49, 63, 74, 77, 79, 82, 84, 86 | M2 adapter/compile group A; M4 admission and affected message tests. |
| `expr/untagged_validation.go` | 44, 63, 92 | Unchanged: existing scalar, simple-path, native JSON/root-form or union contract remains a regression control. |
| `http/codegen/client_primitive_fields_test.go` | 88 | Unchanged: existing scalar, simple-path, native JSON/root-form or union contract remains a regression control. |
| `http/codegen/cookie_security_test.go` | 86, 87, 185, 186, 267, 290 | Unchanged: existing scalar, simple-path, native JSON/root-form or union contract remains a regression control. |
| `http/codegen/deep_object_readiness_test.go` | 77, 84 | Unchanged: existing scalar, simple-path, native JSON/root-form or union contract remains a regression control. |
| `http/codegen/internal/representation/value_carriers.go` | 31 | M4 compile group B and generated codec branches; preserve existing simple scalar/array bytes. |
| `http/codegen/internal/transportir/field_encoding.go` | 9, 11, 15, 17, 19, 21, 25, 34, 36 | M4 compile group B and generated codec branches; preserve existing simple scalar/array bytes. |
| `http/codegen/internal/transportir/media_variant.go` | 24, 25, 60 | M4 compile group B and generated codec branches; preserve existing simple scalar/array bytes. |
| `http/codegen/internal/transportir/model.go` | 153, 154, 264 | M4 compile group B and generated codec branches; preserve existing simple scalar/array bytes. |
| `http/codegen/internal/transportir/parameters_security.go` | 48, 67, 69 | M4 compile group B and generated codec branches; preserve existing simple scalar/array bytes. |
| `http/codegen/internal/transportir/request_response.go` | 54 | M4 compile group B and generated codec branches; preserve existing simple scalar/array bytes. |
| `http/codegen/json_field_carriers.go` | 24, 43, 64, 68 | M4 compile group B and generated codec branches; preserve existing simple scalar/array bytes. |
| `http/codegen/json_field_carriers_test.go` | 19 | M4 compile group B and generated codec branches; preserve existing simple scalar/array bytes. |
| `http/codegen/mapped_params_test.go` | 46 | Unchanged: existing scalar, simple-path, native JSON/root-form or union contract remains a regression control. |
| `http/codegen/openapi/internal/ir/document.go` | 197 | M4 compile group B projection inputs; M5 codec-derived export, version admission and rendered tests. |
| `http/codegen/openapi/internal/ir/media_variants.go` | 21 | M4 compile group B projection inputs; M5 codec-derived export, version admission and rendered tests. |
| `http/codegen/openapi/internal/ir/operation_metadata.go` | 67, 148, 220 | M4 compile group B projection inputs; M5 codec-derived export, version admission and rendered tests. |
| `http/codegen/openapi/internal/ir/url_field_encoding.go` | 13, 19, 22, 30, 31, 32, 34, 35, 36, 37, 40, 47, 48, 58, 62, 63 | M4 compile group B projection inputs; M5 codec-derived export, version admission and rendered tests. |
| `http/codegen/openapi/v3/deep_object_readiness_test.go` | 148 | M4 compile group B projection inputs; M5 codec-derived export, version admission and rendered tests. |
| `http/codegen/openapi/v3/openapi31_filter.go` | 59, 62, 125, 169 | M4 compile group B projection inputs; M5 codec-derived export, version admission and rendered tests. |
| `http/codegen/optional_object_request_body_test.go` | 68, 71 | Unchanged: existing scalar, simple-path, native JSON/root-form or union contract remains a regression control. |
| `http/codegen/path_escaping_integration_test.go` | 188, 239 | Unchanged: existing scalar, simple-path, native JSON/root-form or union contract remains a regression control. |
| `http/codegen/path_escaping_test.go` | 24, 29, 31, 37, 38, 39, 40 | Unchanged: existing scalar, simple-path, native JSON/root-form or union contract remains a regression control. |
| `http/codegen/paths.go` | 95 | M4 compile group B and generated codec branches; preserve existing simple scalar/array bytes. |
| `http/codegen/server_decode_test.go` | 327, 351 | Unchanged: existing scalar, simple-path, native JSON/root-form or union contract remains a regression control. |
| `http/codegen/service_data_payload.go` | 355 | M4 compile group B and generated codec branches; preserve existing simple scalar/array bytes. |
| `http/codegen/template_sources_request.go` | 60, 202 | M4 compile group B and generated codec branches; preserve existing simple scalar/array bytes. |
| `http/codegen/template_sources_request_decoder.go` | 76, 99 | M4 compile group B and generated codec branches; preserve existing simple scalar/array bytes. |
| `http/codegen/template_sources_request_decoder_partials.go` | 317 | M4 compile group B and generated codec branches; preserve existing simple scalar/array bytes. |
| `http/codegen/testdata/openapi_32_dsls.go` | 128 | M1/M4 identical F-2 fixture authoring; retain original rejection input in M5. |
| `http/codegen/union_request_body_test.go` | 62 | Unchanged: existing scalar, simple-path, native JSON/root-form or union contract remains a regression control. |
| `http/form.go` | 65, 67, 68, 72, 74, 85, 87, 121, 151, 553, 554, 558, 560, 565, 567, 570, 576, 578, 581, 583, 585 | M2 compile group A; M4 runtime form selection through Resolve, retaining JSON/root behavior. |
| `http/form_fuzz_test.go` | 30, 70, 75, 79 | Unchanged: existing scalar, simple-path, native JSON/root-form or union contract remains a regression control. |
| `http/form_root_map_test.go` | 30 | Unchanged: existing scalar, simple-path, native JSON/root-form or union contract remains a regression control. |
| `http/form_test.go` | 53, 63, 88, 90, 102, 106, 110, 112, 132, 170, 184, 189, 191, 279, 290, 322 | Unchanged: existing scalar, simple-path, native JSON/root-form or union contract remains a regression control. |
| `http/header.go` | 16, 27 | Unchanged: existing scalar, simple-path, native JSON/root-form or union contract remains a regression control. |
| `http/media_codec.go` | 242 | Unchanged: existing scalar, simple-path, native JSON/root-form or union contract remains a regression control. |
| `http/path.go` | 9, 15, 27, 32, 37, 39 | Unchanged: existing scalar, simple-path, native JSON/root-form or union contract remains a regression control. |
| `http/path_test.go` | 36, 56, 82, 94, 101, 102, 137, 147 | Unchanged: existing scalar, simple-path, native JSON/root-form or union contract remains a regression control. |
| `http/request_cookie.go` | 8, 12 | M3 verbatim cookie publication and byte-domain regression tests. |
| `http/request_cookie_test.go` | 11, 12, 41 | M3 verbatim cookie publication and byte-domain regression tests. |
| `http/testdata/form_comparison/comparison_test.go` | 52 | Unchanged: existing scalar, simple-path, native JSON/root-form or union contract remains a regression control. |
| `http/tla/path_decoding/README.md` | 63, 85 | Unchanged: existing scalar, simple-path, native JSON/root-form or union contract remains a regression control. |
| `http/uniform_json_readiness_test.go` | 52, 66, 104 | Unchanged: existing scalar, simple-path, native JSON/root-form or union contract remains a regression control. |
| `internal/httpencoding/encoding_test.go` | 24 | M2 compile group A; M6 retire the old selector and replace contract tests. |
| `internal/httpencoding/tla/serialization/README.md` | 4 | M8: update public contract or owner/model guidance. |
| `internal/openapiimport/analyze.go` | 394, 395, 427 | M2 independent shape adapter; M6 normalize serialization, move admission/render and migrate affected tests. |
| `internal/openapiimport/analyze_parameters.go` | 3 | M2 independent shape adapter; M6 normalize serialization, move admission/render and migrate affected tests. |
| `internal/openapiimport/cookie_test.go` | 33 | M2 independent shape adapter; M6 normalize serialization, move admission/render and migrate affected tests. |
| `internal/openapiimport/deep_object_readiness_test.go` | 33, 34, 74 | M2 independent shape adapter; M6 normalize serialization, move admission/render and migrate affected tests. |
| `internal/openapiimport/media_contract.go` | 19, 20, 28, 125, 184, 189 | M2 independent shape adapter; M6 normalize serialization, move admission/render and migrate affected tests. |
| `internal/openapiimport/model.go` | 243, 267, 355 | Unchanged: existing scalar, simple-path, native JSON/root-form or union contract remains a regression control. |
| `internal/openapiimport/parameter_admission.go` | 9, 12, 19, 22, 31 | M2 independent shape adapter; M6 normalize serialization, move admission/render and migrate affected tests. |
| `internal/openapiimport/parameter_admission_test.go` | 199 | M2 independent shape adapter; M6 normalize serialization, move admission/render and migrate affected tests. |
| `internal/openapiimport/plan.go` | 200, 201, 254, 287 | M2 independent shape adapter; M6 normalize serialization, move admission/render and migrate affected tests. |
| `internal/openapiimport/render.go` | 485 | M2 independent shape adapter; M6 normalize serialization, move admission/render and migrate affected tests. |
| `internal/openapiimport/symmetry_test.go` | 70, 71, 244 | M2 independent shape adapter; M6 normalize serialization, move admission/render and migrate affected tests. |
| `internal/openapiimport/url_serialization.go` | 14, 15, 23, 28, 38, 47, 48, 52, 65, 103, 106, 111, 113, 117, 120, 122, 124, 127, 129 | M2 independent shape adapter; M6 normalize serialization, move admission/render and migrate affected tests. |
| `jsonrpc/codegen/security_test.go` | 38, 64 | Unchanged: existing scalar, simple-path, native JSON/root-form or union contract remains a regression control. |

## Parent evidence

Toolchain: go version go1.27.3 darwin/arm64. Isolated task cache is used for all Go commands.

`LOOM_GENERATION_DETERMINISM_HELPER_PATH=<scratch>/base-params.txt go test ./codegen/generator -run ^TestGeneratedFilesAreByteStableAcrossIndependentProcesses$ -count=1` passed (10.442s). Snapshot SHA-256: `647f449eba7bd5858d85722dac98086c9f6d7affbd6f3e56e3063527deb5affa`. This captures one parent dump; independent-process stability is separately checked at M7.

The plan’s base-import directory is not a Go identifier; outputs use `base-import/design` so the package is `design`. Both comparison sides will use that same package. Parent CLI build and all six imports passed.

| Fixture | SHA-256 |
|---|---|
| `supported.yaml` | `194abfd06b8388663fa764a50cf9a9d116a8f14d669863931fc7844c9ab5aaac` |
| `supported.json` | `194abfd06b8388663fa764a50cf9a9d116a8f14d669863931fc7844c9ab5aaac` |
| `symmetry.yaml` | `2b31c338717b9067a499b522b11cbed0bc6dee04a9545d5b82c9b22653e17214` |
| `encoded_component_reference.yaml` | `00509de63ef35f88adcbe023034f8fa393cfd10668103e8b4959127461989b4c` |
| `shared_status_different_schemas.yaml` | `3b2e2e1c810ba84ce507af8b0564d65d737d1d3963d053d57eedfd011aef51c8` |
| `object_field_names.yaml` | `7d9ebb03b06a7d0b9caee70f597e1bbc4325b3995bd4b25c23ed4f468c79fc71` |

Parent report exit codes (2 means blocked; 0 means importable). The nullable-member specimen is accepted at the parent and is therefore a regression obligation for the new shape owner.

| Specimen | Exit |
|---|---|
| `cookie-form-list` | 2 |
| `cookie-map` | 2 |
| `form-content-explode` | 2 |
| `form-deep-object` | 2 |
| `form-reserved-false` | 2 |
| `header-object` | 2 |
| `path-label-integers` | 2 |
| `query-deep-map` | 2 |
| `query-deep-object` | 2 |
| `query-default-map` | 2 |
| `query-explicit-default` | 0 |
| `query-form-list` | 2 |
| `query-pipe-list` | 2 |
| `reject-content-style` | 2 |
| `reject-cookie-form-list` | 2 |
| `reject-deep-array` | 2 |
| `reject-deep-nested` | 2 |
| `reject-nullable` | 0 |
| `reject-query-matrix` | 2 |
| `reject-space-explode` | 2 |
| `reject-two-residual` | 2 |

Golden cookie inventory: meal-planner and async-session-security each have JSON/YAML API-key schemes, which stay unchanged. raw-request-bodies has one JSON/YAML cookie Parameter: identical F-1 authoring on both sides supplies form; the only implementation difference for that fixture is ID-3.

Compile-safe naming detail: Go cannot declare both a Location and Style constant named Cookie in one package. The location keeps Cookie; its style will be CookieStyle. JSON and Root retain their existing constant values during the temporary Select migration, then become Style constants when Mode is retired. These are internal identifier choices, with no wire or public policy change.

The bounded parent compilation passed (441.270s), covering the twelve planned designs. M1 is complete.

## M2 owner evidence

Compile group A and the owner/adapter stubs passed `go build ./...`. The direct
owner/adapter regression command then failed behaviorally in all three packages:
httpencoding 0.417s, expr 0.578s, importer 0.576s. There were no compilation failures.
The new shape tests classify arrays of objects as UnsupportedShape: the design's
Nested category describes maps/objects with nested members, not arbitrary arrays.
That prevents the retained query-map JSON default from admitting nested arrays.

The implemented direct M2 matrix passes: httpencoding 0.080s, expr 0.547s,
importer 0.297s. The affected-package gate runs against `<scratch>/m2-source`,
a frozen parent-plus-change copy with empty Git metadata. Manifest SHA-256:
`ff4294b84e7f532a7ecfa416581593446b8341da32ce4ab3b6e376f214beb857`.
Its explicit LOOM_DIR points to that snapshot, so subsequent runtime edits cannot
invalidate generated compilation evidence. The full package gate passed: HTTP codegen 1202.077s, importer 1203.840s, JSON-RPC codegen 682.577s; owner, expr, runtime and OpenAPI packages also passed (`m2-packages.log`). M2 is complete.

M3 interface completion: `ParamNamespace` must carry the current binding's
`OwnedLiterals` as well as sibling literals/prefixes. Nil means the binding has
no fixed allowlist; a nonnil list filters unknown fixed-object keys before
parsing. Otherwise the planned runtime cannot distinguish fixed objects from
residual maps (both use ParamPairs). Cookie publication also needs the sibling
namespace input to reject keys owned by credentials or other bindings before
mutating headers. These inputs implement the accepted KeySet rules; they do not
change wire policy. Update the design/runtime API correspondence before review.


## M3 runtime evidence

Compile-safe runtime stubs passed `go build ./...`. The planned direct runtime
matrix then failed behaviorally (`m3-red.log`, 0.439s). Implementation passes
that matrix (`m3-green.log`, 0.624s), including literal cookie publication,
raw delimiter splitting, path dot protection, associative duplicate rejection,
namespace collision rejection before mutation, bounded form parsing, and exact
JSON numeric membership independent of typed integer widths.

`go test -race ./http ./internal/headerlist -count=1` passed (6.653s and 1.202s).
A subsequent delimiter-alphabet correspondence test passed (0.606s), extending
the formal model alphabet with real UTF-8, quotes, backslashes and brackets.
It checks each admitted runtime encoding decodes to the same value across all
styles and explode settings. Exact expected-byte and rejection tables remain
independent controls, so round-trip success cannot silently narrow admission.

Plan correction: its M3 query checklist says to reject `R,,G,200`, but accepted
design section 4 explicitly permits empty object value tokens. The query test
therefore accepts `{R: "", G: "200"}`. Header framing drops empty list members,
so the same header spelling remains rejected for an odd token count. This is
alignment to the accepted contract, not a new public policy.

Private token framing owns pair parity, duplicate keys and once-only decoding;
location codecs own their separators and admitted data domains. Raw form capture
wraps the existing ParseFormWithLimit owner and retains body-close ownership.
No generator currently uses the new runtime APIs; M4 supplies that wiring.

The complete runtime race gate including the alphabet passed (6.774s, 1.076s).
A bounded independent M3 review found malformed UTF-8 from valid percent-encoded
input was incorrectly fatal during JSON conversion. Five red regressions cover
scalar string, bytes, sequence element, map value and dynamic map key. UTF-8
admission now returns ordinary non-membership before marshaling. The affected
race selection passed (1.822s), and focused independent re-review confirmed the
fix with no unresolved M3 findings. This slice review is not commit authorization.

M4's initial expr admission matrix was written before implementation and fails
behaviorally (`m4-expr-red.log`, 0.530s). It includes the new location/shape cells,
invalid metadata, body-field misuse, delimiter restrictions and namespace clashes.

M4 expr admission passes (0.765s), including form flag precedence. The full expr
suite passes (1.022s) after migrating formerly rejected scalar maps/objects and
retaining nested/union rejection controls. API-key and transport-only session
cookie namespace collisions pass direct DSL checks (0.509s).

Compile group B adds resolved Codec/Keys to transport IR, FormFieldCodecs to
requests/media variants, and codec/namespace descriptors to generator elements
and styled form fields. `go build ./...` and compilation of HTTP/OpenAPI/JSON-RPC
package tests pass. The existing transport IR suite passes (0.141s); the new
resolved-codec matrix passes (0.412s). Metadata-free query maps retain JSON.

Generated HTTP and JSON-RPC matrices initially failed DSL admission (0.697s and
0.564s). After B, HTTP reaches its expected next red seam: the old scalar
formatter cannot serialize ParameterRange (0.372s). No generated implementation
is claimed complete. The shared native-value adapter was introduced as a stub,
then its scalar/list/map/object tests failed behaviorally (0.628s), before its
implementation. Template wiring, typed assignment and generated compilation are
still pending.

### M4 generated wiring and review

The shared native text adapters now feed path, query, header, cookie, response
header and styled form codecs. `PathFiles`/`ServerPathFiles` take `genpkg`, like
other transport file generators, so structured path arguments import their actual
service declarations. Only newly styled path constructors return `(string,error)`;
ordinary simple constructors retain their signatures. Required zero-key values
are checked in BuildRequest before HTTP request allocation. Raw form and raw query
carriers are independently named.

The generated matrix first reached compilation, then exposed four missing
preflight guards and the unwired form branch. After those were implemented,
`TestParameterStylesGeneratedHTTP` passed (2.701s). The JSON-RPC case exposed an
existing metadata-only request omission: a request with no params body and no
mapped ID sent no envelope. Its explicit nonnil-body control failed before the
shared encoder was corrected to emit the protocol envelope (green 1.811s).
This is an additional intended difference from the parent.

Independent slice review reported six medium findings: optional defaults,
form/query local collision, projected header pointer layout, optional nil Bytes,
custom text metadata/raw validation, and indexed path local collisions. The
mixed query/form compile regression failed at the duplicate `:=`; the native
carrier regression failed for projected members, UUID carriers and authored
`parameterValue0`/`parameterSegment1`. Optional cookie/list/property defaults and
nil-byte round-trip cases cover the value behavior. Corrections use the actual
AttributeContext and shared DefaultValue owner, preserve required precedence,
and allocate path locals through the existing name scope. The reviewer confirmed
all six fixes with no remaining finding. This is not commit authorization.

Current focused command:

```sh
GOCACHE=/tmp/loom-957-build-cache.r6al1q \
LOOM_DIR=/Users/luca/.codex/worktrees/parameter-styles/loom \
go test ./http/codegen ./jsonrpc/codegen \
  -run '^TestParameter(StylesGenerated|NativeCarriersGenerated|Value)' \
  -count=1 -timeout 10m
```

Result: HTTP 4.359s, JSON-RPC 2.628s, both pass. This includes exploded form
objects alongside JSON default fields, exact form replay, mixed query/form,
strict unquoted cookie peer, invalid exploded-header keys, viewed header object
compilation, custom UUID matrix with a non-format constraint, and local-name
collisions. Source output and logs are in `m4-review-fixes.log`,
`m4-review-controls-red.log`, and `m4-native-red.log` beneath the scratch directory.
The earlier frozen `m4-source` broad gate remains an intermediate snapshot and
does not cover these subsequent review fixes or the not-yet-implemented exporter.

### M5 export evidence

Focused projection and 3.1 cookie admission tests pass (IR 0.672s, renderer
0.613s). The full OpenAPI package suite passes (IR 6.747s, renderer 8.562s).
The F-1 raw upload cookie explicitly uses form and F-2 preferences uses the
unreserved compact/full enum, matching the parent fixture preparation. Original
unrestricted inputs remain inline 3.1 rejection controls. TestFiles added only
the two parameter-styles goldens; existing goldens are byte-unchanged.

Frozen `m5-source` passed `make build-loom openapi-contract`: rendered/parser and
pinned Vacuum checks 32.379s (including both parameter-style versions), semantic
import round trip 10.109s, generated HTTP Bytes 4.659s, JSON-RPC Bytes 6.093s,
value-contract 1.354s, and IR byte constraints 1.387s. The compatibility pass
checks cookie equivalence before dropping 3.2-only constructs. An independent
M5 review reported no findings; its focused tests passed (IR 0.658s, renderer
0.624s). This clears the slice, not the ticket's final commit review.

### M6 importer integration and media review

OpenAPI parameter defaults now normalize before `Resolve`; headers and form
encodings retain explicit-presence metadata. The old `Select`/`Mode` projection
is deleted, including transport IR consumers. Multipart uses its own admission
entry point and keeps its existing codecs. The two shape adapters agree on the
nontrivial exported parameter fixture (M6 shape test 0.660s).

Thirteen newly admitted import specimens generate and compile (the combined
migrated importer selection passed in 63.066s). Eight rejected specimens assert
their specific reason and remain excluded in lossy partial imports (0.565s).
Credential namespace controls and overlapping residual fields in a multimedium
form branch are covered; the latter failed before shared namespace checking was
added. Three imports regenerate OpenAPI JSON/YAML byte-identically across
independent processes (combined namespace/regeneration command 17.811s).
The existing semantic and symmetry CLI roundtrips and the new parameter fixture
pass with comparison negative controls (12.879s). The old symmetry negative
control pointed at a nonexistent example path and failed on unchanged main;
it now changes the actual `examples/inline/dataValue/id` value.

Independent review found two integration defects: form encoding metadata was
projected onto the enclosing media union, and generated media variants dropped
the resolved field codecs. Both were reproduced before correction. Encoding
metadata now stays on its branch; default media factories consume typed,
occurrence-owned callbacks emitted by the shared form-field renderer. Custom
factories retain codec ownership. Generated tests check exact request/response
form bytes, replay, empty optional forms, malformed duplicate fields and required
empty values before request publication or response status commitment. The latter
also failed before the shared required-key check and eager form preparation.
The combined runtime/generator/importer media tests passed (0.474s/9.598s/8.779s).

Re-review found a callback lifetime defect when an exported request encoder reused
a request for a different form variant. The new styled-to-ordinary control first
sent the old body, then passed after generated code restored the original request
context at call exit (3.502s). Restoration preserves Body/GetBody/header mutations.
All M6 review findings are confirmed fixed; this is slice review, not final
commit authorization.

The frozen `m6-source` full importer run completed in 560.927s. Besides the fixed
media projection, failures were old expectations for text Any headers, explicit
form defaults and unexploded query arrays. Those cases now use a concrete header,
assert default omission, and reject the still-undefined exploded space grammar;
focused migrated regressions pass (12.372s). Text Any has explicit admission
regressions and seven exact exported-design rejection expectations. Arbitrary
JSON numeric fidelity remains covered through a generated JSON body roundtrip.
The ordinary form decoder's generated output was restored to its parent bytes.

### Additional intended generator differences

In addition to the initial ID-1–ID-4 inventory, four existing path BuildRequest
snapshots gain the design's pre-publication delimiter/absence checks. Four raw
JSON-RPC section snapshots gain the `sort` import used by deterministic associative
parameters; unused imports are removed by the ordinary file finalizer. Fifteen
text-Any golden fragments are retired in favor of rejection tests; JSON body Any
coverage remains. Raw request-cookie publication tests now distinguish the new
verbatim-data alphabet from the unchanged response-cookie contract (HTTP 3.839s,
JSON-RPC 3.334s). These are explicit contract/test migrations, not weakened gates.

Scratch evidence: `m6-*`, `m7-media-*`, `m7-import-migration.log`,
`m7-golden-before.log`, and `m7-golden-update.log` beneath the scratch root.
The Bytes text-policy question remains unresolved; no byte policy is inferred
from these passing slices. Broader M7 validation and final commit review remain.

### M7 root bindings and body metadata

The root-collection matrix first failed for named collections, map paths and
map header/cookie bindings, then exposed constructor sources taken from request
fields instead of decoded response bindings. `validateCollectionPayloadTransport`
now enforces one payload-owning binding uniformly; actual cookie bindings feed
payload constructors and actual response arguments feed response transforms.
Transport-only session cookies remain outside payload ownership. Eight generated
map/list exchanges across all four locations pass with exact wire assertions
(2.403s); related constructor/admission controls pass (2.772s). Full expr tests
pass (0.914s), including the new success/error effective-body metadata rejection
controls that failed before validation moved to those body occurrences.
Independent slice review found no issues and reran expr and generated exchanges
(0.174s/2.109s). This is not final commit authorization.

The frozen M7 package run passed HTTP codegen (766.648s) and JSON-RPC codegen
(437.804s). Importer checks (756.163s) isolated an unused form-values local in
three all-styled form specimens and their regeneration. The shared renderer now
emits that local only when an ordinary field needs it; the existing compile
regressions reproduce the defect. Focused rerun results follow below.

### M7 baseline comparison

All eighteen original/control designs were compared as raw sections and rendered
files against the M1 parent dump (six existing determinism controls plus twelve
parameter controls). Every hunk was inspected. Differences are:

- raw section imports of sort and the service package, removed when unused by
  the normal finalizer; rendered files otherwise retain their original imports;
- required zero-key checks in query-array and form-array BuildRequest, plus path
  representability preflight in path-array and raw-upload BuildRequest;
- raw cookie style explicitly emitted for PayloadCookieStringDSL (ID-1);
- the identically authored raw-upload form cookie uses its actual percent codec,
  and default form style is omitted from export (ID-3).

No other rendered differences appeared. The five-process byte-stability test
now covers all those controls plus ParameterStylesDSL, ParameterStyles31DSL and
StyledMediaContractsDSL; it passed in 13.455s. This run precedes the subsequent
all-styled-local correction; that affected generated output will be rechecked.

The all-styled fix passes all thirteen accepted import specimens and independent
regeneration (61.519s). The bounded M7 compile tier passed (89.242s), covering the
same twelve parent controls, both new parameter style fixtures, styled media and
the seven intended text-Any rejections. The 21 CLI reports now match the design:
thirteen accepted, eight rejected. Five legacy import files retain identical
hashes; symmetry.yaml differs only by omission of its equivalent explicit
`openapi:style: form` metadata. Its semantic roundtrip and negative controls are
covered by M6. This normal form change is an additional intended difference from
the plan's initial byte-identity assumption.

Frozen `m8-source` runtime race checks pass: http 20.529s, httpencoding 1.429s.
The source manifest is stored in scratch. Full tests and transport integration
are running; lint found render.go just beyond its 700-line repository threshold,
so parameter rendering is being extracted without behavior changes.

`make integration-test` passes on the frozen M8 source: JSON-RPC framework
1.209s, harness 20.434s, mounted tests 174.657s; HTTP harness 10.494s, mounted tests
17.890s. The metadata-renderer move preserves its function byte-for-byte and
passes direct metadata/symmetry controls (4.693s).

Five-process generation byte stability passes again after the all-styled decoder
fix (74.660s on the frozen M8 source). Documentation slice re-review confirms all
wording findings fixed, including exact deepObject ownership, allowReserved's
codec-neutral meaning and explicit future #1014 obligations. Final ticket
review and the Bytes policy decision remain outstanding.

### M8 final review fixes and gates

The frozen `m8-source` full `make test` run completed successfully (80 passing
package reports). Its earlier runtime race, transport integration and
independent-process determinism results remain valid for unchanged paths.

The final integration review found that structured text parameters inherited
JSON field names/omission in OpenAPI and examples. `BuildValuePlan` now owns
normalization of text members to DSL names, including an explicit normalized
name when other struct tags remain. Styled form fields receive separate copies
so a shared declaration's ordinary JSON form occurrence keeps its JSON tags.
The schema analyzer consumes the captured target and retains original
component baselines by plan position; canonical-name collision checks remain
strict. Red regressions demonstrated both the JSON-hidden-member panic and
`Count` versus `count` examples when an XML tag survived JSON-tag removal.
The independent reviewer confirmed both fixes.

`ParameterPropertyNamesDSL` now covers a canonical named record shared by query,
response header, styled form, ordinary JSON form and request/response JSON.
Structural checks assert properties, required fields and examples in both
registration orders; a separate text-only record retains a JSON-hidden member.
The mounted generated client/server test verifies exact query/header names,
JSON and form values, decoded service payloads and the returned result.
The example-free import companion verifies strict export/import/regeneration
and compilation without conflating wire names with annotation placement.

The import probe found libopenapi v0.42.3 incorrectly promotes `schema.example`
to an absent `Header.example`. An independent Go reproduction confirmed it and
was published as https://github.com/pb33f/libopenapi/issues/695 after checking
for an existing report and verifying latest stable release metadata. No
production dependency workaround was added. Example projection remains covered
by the IR test; the wire-only import fixture intentionally has no examples.

Lint findings were resolved through private function extractions, an unused
helper removal, bounded preallocation and style corrections. Independent review
confirmed preserved validation order, errors, generated variable allocation and
wire behavior. The two fixture-function splits preserve DSL call order. All 552
previously compared generated sections remain byte-identical; adding the styled
and property-name specimens produces 738 sections, also byte-identical across
the subsequent fixture extractions. No extra wire difference was introduced.

Frozen candidate evidence uses Go 1.27.3 and explicit `LOOM_DIR`:

- Full affected owners: representation 0.561s, OpenAPI IR 11.126s, OpenAPI v3
  12.814s, HTTP runtime 3.837s and encoding policy 0.230s.
- Generated parameter/media/root and direct renderer checks: 10.423s; importer
  parameter/media checks: 18.608s. Matrix preallocation controls: 0.775s.
- Five-process determinism passes with the new property-name specimen.
- `make lint lint-docs` passes, including testdata and integration modules.
  The later CLI/specimen edits pass scoped golangci-lint with zero issues.
- New property-name specimen passes libopenapi and pinned Vacuum v0.36.2
  (16.731s). Final strict CLI round trip and compiled regeneration pass (6.409s).
  The final IR names/examples/canonical-order regression passes (0.696s).

The first post-extraction generated-test attempt accidentally used the old
pinned runtime and failed on missing new APIs. It is not acceptance evidence;
the recorded successful rerun explicitly selects the frozen candidate source.
The temporary wire harness also consumed the request body during observation;
its corrected version restores the unchanged bytes before calling the server.

The current source manifests and logs are retained in scratch pending final
acceptance. The Bytes text-policy decision and exact-final-diff commit review
are still required before delivery; these gate results do not settle policy.

### Complete-candidate review

A fresh independent review inspected the complete candidate against parent
`f7a93576833c70eaa00d7a5bb9884f83b55ec15d` (patch SHA-256
`25ab7db967d84a5ab88c00a9377f41e03018a9961ebb4d90fedac423c9726bd2`,
tree `1d60206e78baf8e27c13412d677564a0c61c8757`). It found one medium defect:
OpenAPI lost inherited `allowReserved` on query parameters, response headers and
ordinary named form bodies. The existing runtime/admission owner already
resolves this metadata through aliases and preserves occurrence overrides.

The red regression reproduced the loss. The exporter now reads effective
metadata through `HTTPMediaMetadata` at all three projection sites, matching
admission and media-variant export. `ParameterMetadataDSL` covers nested scalar
aliases, inherited true flags, explicit false overrides and a named form's map
encoding. This adds the intended export difference for formerly lost inherited
flags; it changes no wire grammar or escaping behavior.

The fix passes full affected exporter packages (IR 14.991s, v3 13.410s), bounded
new-specimen generation/compilation (35.620s), libopenapi plus pinned Vacuum
(20.999s), scoped lint with zero issues and five-process determinism (25.388s).
All 738 existing compared output sections remain byte-identical; the additional
metadata specimen contributes 29 sections (767 total). The complete review
reported no other actionable finding. Delta re-review is required for this
correction; the Bytes policy decision remains explicitly unresolved.


## Final integration with object openness and AdditionalAttributes

The user delegated the remaining framework choices. Text Bytes preserve raw
bytes through the location codec; JSON Bytes retain base64. The public guide,
consumer/framework skills, design and model correspondence now state that
contract. The earlier pending-policy notes are historical.

The final commit parent is `66765668a608934c22135eb4a24385807976bfcc`, incorporating
object-openness ownership and typed AdditionalAttributes. Three-way integration
preserves those changes. `urlShape` uses the current single-result object
classifier. Both shape adapters reject typed records in text bindings, including
aliases and inherited policies, rather than silently dropping their undeclared
members. Existing JSON body support and form-body rejection remain intact.
The four new shape controls failed before that correction. They and DSL controls
for path/query/header/cookie/form rejection and JSON body acceptance pass after it.

The source snapshot is `main-integration/source` under the original task scratch;
`source-manifest.json` identifies its inputs. All checks use Go 1.27.3 and the
shared task cache; generated modules select the snapshot with explicit LOOM_DIR.

- Direct shape and AdditionalAttributes selection: expr 0.588s, importer 0.608s.
- Full affected owner packages: expr 0.808s, httpencoding 0.302s, HTTP 4.213s,
  representation 0.467s, transport IR 0.469s, OpenAPI IR 5.836s, v3 6.954s.
- Generated typed-member JSON bodies, property names, styled media and root
  collections: 9.413s. Parameter styles/native carriers: HTTP 2.983s,
  HTTP-backed JSON-RPC 2.107s.
- Affected importer/CLI parameter, additional-member and media round trips:
  130.833s / 3.939s.
- Four parameter specimen families through libopenapi and the pinned Vacuum
  contract rules: 27.959s, zero errors.
- Five-process exact-byte determinism: 16.905s.
- `make lint lint-docs`: passes, including every lint module. Existing size
  warnings remain warnings; no suppressions added.

The matching parent/candidate output harness adds AdditionalAttributesTypedDSL
to the original 18 specimens: 593 raw/rendered sections on each side, no sections
added or removed, 81 changed. Every one of the 552 sections shared with the prior
reviewed #968 dump is byte-identical to that reviewed result, so its detailed
intended-difference inventory remains applicable. Of the 41 additional-member
sections, five differ: four raw import inventories plus the rendered client
BuildBoundRequest preflight (including its fmt import). Body types, JSON codecs,
validation and conversions are byte-identical to the new parent. The mounted
additional-member test compiles and exercises the generated code. Full diff and
section comparison are retained in `main-integration/output-comparison.*` until
independent review is complete. Prior full-suite, race and integration evidence
is reused for unchanged paths; this integration adds the affected checks above.

The final bounded compile tier also passes (27.561s): the existing query-map,
path-array and form-body specimens, ParameterStylesDSL and the new parent's
AdditionalAttributesTypedDSL. Each generated module is built, vetted and tested.
Final documentation lint and whitespace checks pass after recording this evidence.

## UUID raw-cookie domain correction

UUID format validation accepts canonical, URN and brace spellings. The latter
two contain characters that form encoding percent-escapes, so the shared
`expr.httpValueDomain` must not claim an unreserved alphabet from UUID format
alone. Explicit enums still establish a narrower unreserved domain. Runtime
UUID validation and the raw-cookie codec retain their existing behavior.

Direct shape and cookie-equivalence tests failed before removing the unsound
format inference. Runtime witnesses show that raw/form cookie spellings differ
for URN and brace UUIDs and that reading the form spelling as raw fails UUID
validation. Export regressions cover both 3.1 and 3.2 targets, explicit form
encoding and a canonical UUID enum. Full expr, OpenAPI IR and v3 package tests
pass. Accepted JSON/YAML artifacts parse with libopenapi and pass pinned Vacuum
v0.36.2 rules. Parent/candidate comparison preserves all ten accepted artifacts
byte for byte; only the unrestricted raw UUID 3.1 case changes from two emitted
artifacts to a named rejection. Two independent candidate processes agree.
The existing 25-design representative output remains unchanged; its prior
mounted compilation evidence remains applicable. No checked-in fixture changes.
Scratch reproduction inputs and logs: `/tmp/loom-fresh-audit/uuid-cookie`.
