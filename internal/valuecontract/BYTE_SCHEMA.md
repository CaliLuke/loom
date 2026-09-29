# Bytes schemas and representation ownership

This record belongs to #574. The repository gates, registered 72-probe comparison
and final-source 499-case catalog comparison passed. The final production capture
is `6d53d4d8`.
The accepted contract is in
[the value-contract design](../../roadmap/value-contract-design.md).

## Ownership

`internal/byteschema` intersects decoded-length bounds across an alias chain,
then projects that effective interval into three padded-base64 residue branches.
The helper owns checked arithmetic and grammar construction. It does not decide
whether a transport uses JSON, whether a field is present, or which declaration
owns a schema component.

`expr/json_schema_inline_bytes.go` adapts the helper for inline schemas. HTTP's
shared `http/codegen/internal/representation` preparation supplies the actual
codec and occurrence plan to both transport generation and OpenAPI analysis.
OpenAPI must finish representation analysis before sharing component names and
references. A media label alone does not establish the codec. Raw bodies,
multipart, locations and custom codecs retain their separate contracts.

Codec extraction is an explicit acceptance boundary. Actual-encoder probes
exposed raw mapped SSE Bytes and ordinary HTTP `text/plain` responses that the
draft plans incorrectly classified as JSON. The shared preparation follows
the existing codec dispatch for each direction and streaming mode. The finite
built-in codec matrix now passes against actual runtime functions and rendered
schemas. It checks production correspondence separately from the arithmetic
proof; it does not prove correct codec extraction for every design.

Use these entry points when maintaining the architecture:

| Responsibility | Owner |
| --- | --- |
| Select built-in HTTP codecs with the existing role-specific fallback rules | `internal/httpcodec/codec.go`, consumed by runtime `http/encoding.go` and representation `response_codec.go` |
| Capture transport occurrences and prepare plans before analysis | `http/codegen/internal/representation/prepare.go` |
| Bind referenced errors to their finalized declaration across method, service and API scopes | `http/codegen/internal/representation/error_binding.go` |
| Share emitted layout, tags and physical SSE field encoding | `http/codegen/internal/representation/{layout,tags,sse_encoding,stream_schema}.go` |
| Query captured structure without resolving values or mutating global registries | `expr/value_plan_schema.go` and `expr/value_occurrence.go` |
| Record actual baseline graph slots and project effective byte constraints without resampling | `http/codegen/openapi/internal/ir/analyzer_baseline.go` and `analyzer_representation.go` |
| Allocate complete component variants, including recursive and synthetic union graphs | `http/codegen/openapi/internal/ir/representation_registry.go` |
| Preserve the existing component example annotation independently of representation variants | `http/codegen/openapi/internal/ir/component_annotations.go` |
| Select inline async policy before ordinary named-type processing; retain occurrence-owned acquisition | `http/codegen/openapi/internal/ir/analyzer.go` and `async_baseline.go` |
| Inline async schemas after shared component allocation | `http/codegen/openapi/internal/ir/async_schema_inline.go` |
| Preserve occurrence-context sampling through materialization wrappers | `http/codegen/openapi/internal/ir/async_schema_examples.go` |
| Rewrite and retain references from ordinary and framework-owned async schema roots | `http/codegen/openapi/v3/schema_refs.go`, consumed by `schema_cleanup.go` |

Forced component registration is an inclusion requirement, not evidence of a raw
codec occurrence. Explicit naming conflicts between different authored shapes
remain errors. Only an actual representation conflict of the same declaration
permits the reviewed component split. Renderers consume this result; they must
not repair codec selection, bounds or component identity after analysis.

Static response content type controls the server encoder. A mapped response
header is assigned after encoder selection, so its advertised media enum cannot
establish that codec. Header-only media retain the previous schema under an
indeterminate boundary. The ordinary default JSON response schema describes the
default negotiation path, not every possible Accept header. Client request
encoding remains JSON; server request decoding has its own narrower media rules.

Controlled transport copies retain an immutable original declaration ID captured
in `expr`. That identity does not replace the separate structural comparison
used to reject incompatible explicit names. Imported explicit byte/binary formats
are retained as metadata by `internal/openapiimport/render_schema.go` rather than
inferred later from a changed Bytes default.

Error binding captures only finalized referenced declarations, preserving scope
and shadowing. A missing service carrier gets structural capture without extra
example synthesis. Component annotations likewise retain their existing source
and context; this compatibility owner does not claim to project examples through
every codec. The later value-consumer migrations still own that work.

## Proof and production checks

The Lean modules
[`AliasLengthBounds.lean`](../../expr/lean/value_projection/ValueContract/AliasLengthBounds.lean)
and
[`ByteLengthProjection.lean`](../../expr/lean/value_projection/ValueContract/ByteLengthProjection.lean)
establish alias-bound intersection and encoded-length/residue properties.
The executable reference calls those proved functions. The
[correspondence ledger](../../expr/lean/value_projection/correspondence.md)
records their assumptions and limits.

The [recursive representation model](../../http/codegen/openapi/internal/ir/tla/representation_equivalence/README.md)
checks partition refinement and rooted quotient fingerprints for bounded
component graphs. It reproduces the old false distinction between a self-cycle
and an equivalent extra unfolding, and rejects a candidate that merges different
graphs. Its acyclic invariant also checks abstract compatibility with the old
fingerprint, assuming unchanged serialization. Direct Go naming regressions
also check exact legacy fingerprint serialization: an equivalent graph algorithm
can still change allocated names if its serialized bytes change. This complements the Lean byte
arithmetic; exact Go schema extraction, annotation preservation, naming guards
and artifact compatibility remain tested boundaries.

The [alias pairing and declaration models](../../expr/tla/schema_declaration/README.md)
reproduce source advancement through an inserted target wrapper and annotation
sharing that collapses distinct authored contexts. The previous declaration/
baseline sharing rule is now a failing control, as are plan-only, shared-child
role and canonical-byte-owner shortcuts. Independently checked candidate rules
pair by ancestry and preserve the actual allocation owner and each byte
variant's selected annotation binding. Incoming edges guide pairing, not sample
ownership; matching declaration or shape cannot replace a baseline allocation. Production extraction, recursive identity
and neutral-first/planned-first renderer correspondence remain required.

`byte_alias_conformance_test.go` independently extracts supported authored Bytes
alias chains and compares resolution, projection and the helper's effective
bounds with the reference. Unsupported non-length constraints and custom codecs
fail extraction explicitly. `byte_schema_conformance_test.go` compares emitted
branches and overflow failures with arbitrary-precision reference results.

The generated HTTP test exercises 324 requests across 12 body designs. It checks
decoded bytes, service invocation, status and problem details. In enabled
contract mode it compares those actual outcomes with inline schemas and complete
OpenAPI documents using Ajv's JSON Schema 2020-12 implementation and JavaScript
regular expressions. The JSON-RPC test performs the same kind of comparison for
its existing named, nullable, collection and default fixture. The validator is
pinned under `internal/schematest`; dependencies are installed only in a test
temporary directory.

Cases include each base64 residue, empty values, noncanonical unused pad bits,
malformed padding, alternate alphabets and rejected whitespace/line terminators.
Additional instance checks cover nullable values, negative and empty bounds,
and an enormous inner alias bound narrowed by an outer bound before projection.
The enum control deliberately demonstrates that a byte decoder can accept a
pad-bit alias that the schema's canonical enum rejects. Length equivalence does
not establish enum equivalence or unique union decoding.

These are proofs of the specified Lean functions plus tested production
correspondence. They do not prove Go code generation, the codec library, regular
expression execution, or schema-name allocation correct for all inputs.

## Reproduction

Run from this checkout with the repository's Go, protoc, Lean and Node/npm
prerequisites. Enabled external gates fail when required tools are absent.

```sh
export LOOM_DIR="$PWD"
go test ./internal/byteschema ./expr
make value-contract-conformance
LOOM_OPENAPI_CONTRACT=1 go test ./internal/valuecontract \
  -run '^TestByteSchemaIndependentConstraints$' -count=1
LOOM_OPENAPI_CONTRACT=1 go test ./http/codegen \
  -run '^TestBytesSchemaGeneratedHTTP$' -count=1
LOOM_OPENAPI_CONTRACT=1 go test ./jsonrpc/codegen \
  -run '^TestBytesValidationGeneratedJSONRPC$' -count=1
make openapi-contract
```

The revision-comparison inputs are in `byte_schema_manifest.json`, based on
`2f6bfbf7758f4de56eb14569a49a645355ca228f`. The 28 reviewed artifact differences are
recorded there with exact before/after hashes and reasons. Follow [README.md](README.md) to run the manifest
in independent processes at both revisions. Runtime Go/protobuf artifacts must
stay unchanged; every intended schema and reference difference needs an explicit
record. Temporary outputs and compiled proofs are removed after validation and
independent review, following `AGENTS.md`.

The mixed `byte-representation-ownership` probe exposes an existing generated
HTTP client root-validator defect. Its named Bytes response validator signature
takes a value, while its body and call assume a pointer. Parent `2f6bfbf7` and
the #574 candidate use identical probe inputs and emit identical non-OpenAPI
artifacts and build errors. The build diagnostic SHA-256 is
`13482562680f9942f5524499b3952fe4f9c7e357de2a57b7543322839b1f8476`.
The responsible response-analysis, type-reference and validator-section
functions are unchanged from the parent. Keep this case as a precise known
failure while comparing #574 schemas; it does not supply build or runtime
success evidence. Milestone 5 / #565 owns alignment of the validator declaration,
body and call with the shared physical body plan. That milestone cannot close
until this probe builds, vets and passes its validation behavior matrix, and
its failure expectation is removed. The existing pointer-field conversion
repair remains required.

## Verification checkpoint

Candidate generation used production capture `beca914b`; the later manifest
registration and this record do not change generator behavior. The repair keeps
three decisions separate: declaration provenance, occurrence-owned baseline
acquisition, and representation-specific byte projection. Recorded member,
element, branch and wrapper edges preserve annotations and their absence without
resampling. Alias pairing follows ancestry through inserted transport wrappers.

Async inline ownership is selected in `analyzeSchema` before ordinary view,
validation-overlay and outer-occurrence null handling. Inline construction uses
its existing consumer policy; ordinary schemas and retained cuts keep theirs.
The original plan supplies the effective byte interval.
The 156 complete-schema Ajv assertions check those bounds, including empty
intersections, without importing ordinary null policy into async construction.
General alias enum/default semantics remain assigned to #571.

The [schema ownership models](../../expr/tla/schema_declaration/README.md) now
include `BaselineAcquisition` and `ConsumerDispatch`. Their negative controls
reproduce incompatible memo reuse and ordinary-policy leakage into inline
consumers. The strengthened dispatch candidate checks all seven invariants over
7,408,800 states, including enum and nullability combinations. These bounded
models assume correctly extracted roles and inputs; actual Go traversal,
recursive cuts, codec selection and naming still require production tests.

Public-pipeline regressions preserve parent views, enum overrides,
nullability and combined cases, with ordinary/retained-cut controls. The separate
API-error, SSE-default and explicit-view comparison passes all 60 phases across
12 isolated runs with byte-identical parent/candidate artifacts and no allowances.

The registered comparison has 68 probes and 272 parent/candidate run records:

| Group | Common input capture | Result |
| --- | --- | --- |
| Original 66 probes | `02cdce4d` | 264 records verified; exactly 18 reviewed OpenAPI artifacts across nine cases change. |
| Two declaration-ownership probes | `a6fcc679` | Eight records verified; scalar artifacts are exact; Bytes changes only JSON/YAML at 18 schema sites (72 keyword paths). |

Keep the input captures distinct. Their fixture trees differ only by the inert
declaration-probe file: four functions, no initialization or existing callers.
The manifest binds all 20 changes with hashes and reasons; other artifacts,
including generated Go/protobuf, match. Supplemental whole-path checks preserve
annotations, absence, references and component names; output matches the prior
reviewed candidate byte for byte.

Sixty probes have successful outcomes. Eight preserve known failures: six decoder,
one generation and one build failure. These are unchanged
failures, not successful generation or runtime acceptance. In particular, all four
#565 compiler outputs retain the exact diagnostic hash recorded above.

Repository lint, root-module tests, coverage floors, enabled OpenAPI contracts,
generated-code quality and value-contract conformance pass on production capture
`6d53d4d8`. The latter audits 586 Lean theorems, passes real rejection controls
and executes 15 production assertion groups. Compiled proof outputs were removed
after verification; durable proof sources, configurations and toolchain pins remain.

The final paired catalog comparison used common input capture `1ac7e9b0` and
production capture `6d53d4d8`: 498 valid designs plus one intentional invalid-DSL
rejection. An interrupted run completed 439 pairs. A separately reviewed bounded
recovery generated both attempts for the exact remaining 60 pairs from the same
frozen source. The retained verifier visited all 499 cases, checked repeated output
and parent input equality, and accepted only the registered differences. All 60
recovered difference records are empty.

Four independently reviewed catalog cases have differences: MixedPayloadInBodyDSL,
MultiDSL, SelectedBodyDefaultsDSL and SSEResultTypesDSL. Their eight exact JSON/YAML
allowances change only built-in JSON byte grammar. The manifest has 72 probes and
28 rows; the original 68 probes and 20 rows are unchanged.

An additional bounded audit executed emitted validators for three internal alias
chains: inner maximum 2/outer maximum 4, inner minimum 2/outer minimum 1, and inner
minimum 2/outer maximum 3. Constraints were installed during DSL evaluation before
transport copying. For byte lengths 0–5, current complete schemas (locked Ajv),
validators and the resolver's example role agree on all 18 outcomes; parent
schemas incorrectly measure encoded length. This is not full HTTP evidence or a
public-admission claim: direct named-alias outer length DSL calls reject in both
versions. Supported `Reference` plus primitive-field override instead finalizes
as one primitive constraint, not a retained alias intersection.
