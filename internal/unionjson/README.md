# Typed array/object JSON unions

## Summary

Design implemented for [#350](https://github.com/CaliLuke/loom/issues/350): extend the
existing `OneOf` / `Untagged` contract to concrete named arrays. Keep the current
typed Go sum API and require exactly one matching branch on both decode and
encode. Shared evaluated semantics own branch eligibility and constraints;
generated adapters supply typed operations to one handwritten matching runtime.

This capability requires no new DSL
function, query-dependent decoder, `Any` result, compatibility mode, or fallback.

## Problem statement

A successful JSON response can be a bare item array or a page object. Loom can
describe both types independently, but its untagged-union validation admits only
named object branches. Replacing that validation alone would leave object-only
decoders and schema/example matchers behind.

The original consumer need remains observable in Plane's Python cycle endpoint:
`apps/api/plane/api/views/cycle.py` returns a serialized list for `cycle_view=current`
and calls pagination for other views. Its work-item endpoint in
`apps/api/plane/api/views/issue.py` also selects single-object versus paginated
responses using external identifiers. These are application decisions about
which result to produce, not instructions for a framework decoder.

The current Go cycle implementation uses raw JSON response metadata and a reader;
its list SQL currently emits arrays. Consequently it is evidence of an escape
hatch, not proof that the Go port already preserves every original response mode.
Consumer migration must check the original contract independently.

## Goals and non-goals

Support typed mixed array/object JSON bodies consistently in service types,
HTTP server/client types, JSON-RPC JSON value codecs, and OpenAPI. Admit the same
subset through OpenAPI import. Preserve branch identity for empty arrays.

Do not implement query-dependent dispatch, automatically wrap arrays in objects,
broaden untagged unions to scalar/map branches, redesign pagination, or migrate
Plane as part of this framework ticket. Existing transport framing, protobuf
numbering, result-view selection, and error routing remain their owners' contracts.

## Baseline before this change

- `dsl/attribute_oneof.go` already supplies `OneOf` and `Untagged`. Constructor
  branches need stable identities; anonymous arrays must first become a `Type`.
- `expr/attribute_validate.go` rejects array branches. Object fields support
  primitives, concrete named objects, and arrays of those shapes.
- `codegen/service/sections.go` and `http/codegen/type_sections.go` emit separate
  object-only matching loops. They decode into temporary candidates, accept
  exactly one, and assign the destination only after success.
- Those marshalers validate the selected discriminant but do not check whether its
  final JSON uniquely identifies that branch. Thus an encoded object may be
  rejected by the corresponding decoder.
- `expr/value_projection_decode.go` already describes array decoding and
  exact-one untagged matching. The [value-contract design](../../roadmap/value-contract-design.md)
  requires emitted wire to match the retained identity uniquely. Its generation
  plans distinguish schema acceptance from actual decoder acceptance.
- OpenAPI IR has an object-only example matcher in
  `http/codegen/openapi/internal/ir/example_validation.go`. Import admission in
  `internal/openapiimport` likewise requires object components.

## Design

### Public authoring and generated API

```go
var Items = Type("Items", ArrayOf(Item))
var Page = Type("Page", func() {
    Attribute("items", ArrayOf(Item))
    Attribute("total", Int)
    Required("total")
})

// In a method:
Result(OneOf(Items, Page), func() {
    Untagged()
})
```

`Item` is an independently defined named object. `Page.items` is intentionally
optional so an empty collection remains valid under Loom's omission semantics.
Generated constructors, `Kind`, and `As...` accessors retain their current forms;
the array alternative contains its generated concrete slice type. Naming and
collision allocation remain with the existing union naming and `NameScope`
owners. No alternate names or aliases are introduced. Absent `Untagged()`, JSON
continues to use the existing discriminator/value envelope.

### Canonical semantics and placement

The evaluated `expr` graph, effective validation, and occurrence-specific value
plans are the semantic source of truth. Extend their supported-branch admission
to concrete named arrays whose elements use the currently admitted object-field
grammar: primitives, concrete named objects, or recursively arrays of these.
Apply the same structural restrictions recursively to named objects reached
through array elements; crossing an array must not bypass validation. Reject
maps, nested union fields, inline object leaves, and scalar root branches.
Existing primitive `Any` fields remain opaque leaves; they do not become an
implicit union branch or rescue a failed match.

Shared codegen analysis projects an immutable JSON matching description from
that evaluated contract and the actual emitted representation. The description
includes root shape, member names, presence/nullability, unknown-member policy,
array element shape, and effective constraints. HTTP supplies its finalized
layout and body/view selection; it does not select another semantic policy.
Service codegen supplies its own emitted layout through the same analysis.

The handwritten helper in `pkg` owns candidate orchestration and strict
wire-structure checking. Generated code supplies schema-membership predicates,
concrete decode/validate callbacks, and a final typed assignment. The shared
analysis derives both predicates from the occurrence-specific value plans:
schema membership uses the JSON contract, while decoding additionally applies
the concrete Go representation. These are separate predicates over the same wire,
not two competing owners of authored constraints. Service and HTTP generators must not
emit separate matching algorithms. The helper is public only because generated
consumer packages must call it; generator analysis remains internal. It must
not import `expr`, interpret a serialized design DSL, or use broad reflection.
Ordinary generated validators remain responsible for typed value predicates.

OpenAPI rendering and example preparation project the same evaluated contract.
Use the existing prepared value-plan path for schema/example branch decisions;
do not add an array case to a second independent semantic matcher. Schema and
runtime projections remain distinct where Go representability differs from
JSON Schema, especially numeric widths and external codecs.

For an HTTP body, schema membership means the exact projected body schema,
including view selection and schema visibility. For a service-only JSON type it
means that emitted type's JSON schema projection, without an HTTP mapping. A
transport adapter checks its own target, rather than using a service-layout
predicate on transport bytes. Metadata that makes a required predicate opaque
or inexpressible causes a generation error; it does not disable the check.

### Contract matrix

All rows derive from the evaluated DSL; there is no deployment switch or runtime
registration. Generated artifacts remain in their existing packages and OpenAPI
component locations, with existing occurrence-aware naming and collision rules.

| Surface | Supported and routing | Rejected / deferred | Equivalence and proof |
| --- | --- | --- | --- |
| DSL / service JSON | Named object and array alternatives; existing typed sum, shared matching runtime | Anonymous constructor arrays, scalar/map branches, unsupported nested shapes rejected during evaluation | Admission and generated matching agree on the supported grammar |
| HTTP JSON requests and success responses | Whole body or an existing legal nested union occurrence; server and client typed adapters use the shared runtime | Untagged string params, headers, cookies, forms and multipart remain rejected by transport validation | Corresponding sender/receiver agree on branch identity and value after declared normalization |
| JSON-RPC JSON values | Existing payload/result/event positions use the same generated JSON codecs | No new top-level RPC envelope or stream framing shape | Array selection is preserved inside the existing protocol envelope |
| OpenAPI output | Direct `oneOf` component refs including a `type: array` component; no discriminator; both supported OpenAPI versions | No invented discriminator or query-conditioned schema | Selected wire satisfies exactly one projected schema; runtime representability is checked separately |
| OpenAPI import | JSON request/success-body `oneOf` refs to supported named object/array components | Inline array branch schemas, unsupported element shapes, non-JSON locations and declared-error unions remain rejected | Re-export preserves branch constraints and exact-one meaning, not source formatting |
| Protobuf | Existing typed union projection with numbered oneof alternatives and collection message wrappers | No JSON shape inference on protobuf; existing field-number and placement restrictions still apply | Explicit protobuf branch identity survives independently of JSON ambiguity |
| Custom codecs / raw bodies | Existing separately authored raw-body contracts remain separate | Every reachable opaque codec replacement in any untagged alternative is rejected at generation, including existing object alternatives and object siblings of arrays; support is deferred | No admitted custom boundary may bypass exact matching |

JSON-specific matching does not run on protobuf messages. Merely marking a union
`Untagged` must not change its protobuf field numbers or wrappers. Tagged unions
remain outside the new matcher. No additional catalog/discovery entry is needed:
the existing method and named-type declarations expose the new alternative.

## Detailed behavior

### Decoding

Every candidate observes the identical original JSON value. Required membership
is checked before defaults, names are case-sensitive, and `null` is checked
against the occurrence's nullability before typed conversion. Unknown members
are rejected for explicitly closed objects and ignored for default-open objects;
ignored members cannot become evidence favoring that branch. Array length and
every element's constraints participate in matching. JSON syntax, duplicate
members, and trailing content follow Loom's strict JSON codec contract.

```text
match(wire):
    require one complete valid JSON value
    schemaMatches, decodedMatches = [], []
    for each branch:
        if root shape is incompatible: continue
        evaluate schema predicate on original wire
        decode and validate a new branch-local value from original wire
        propagate any fatal outcome
        record schema matches and decoded matches independently
    require exactly one schema match and one decoded match
    require their branch identities to agree
    return the decoded match

decode(wire, destination):
    candidate = match(wire)
    assign destination once, only after match succeeds
```

Root-shape filtering only skips impossible branches. It cannot choose between
two arrays or two objects. A failed decode leaves the old destination unchanged.
Branch order cannot alter success, selected identity, or rejection classification.
No match and ambiguity remain distinguishable by matched-count errors, qualified
by schema or decoder domain. Different unique identities produce an explicit
identity-mismatch error. Malformed JSON remains a syntax error.

Each callback returns one of three outcomes: match (with a value for typed
decoding), mismatch, or fatal failure. Wrong shape, missing required members,
disallowed null, ordinary constraint violations, and numeric overflow are
mismatches. Invalid matching descriptors, missing callbacks, and unsupported
predicate execution are fatal. Generated callbacks classify these explicitly;
arbitrary errors must not silently become mismatches. Neither a prior successful
candidate nor prior ambiguity suppresses a later fatal error.

### Encoding

```text
encode(selected identity, value):
    require a known selected identity and a valid selected payload
    serialize the selected payload with its emitted representation
    candidate = match(final wire)
    require candidate identity equal to selected identity
    return the serialized wire
```

This intentionally rejects ambiguous output that current object-union marshalers
can emit. It extends the value-contract design's existing identity rule to actual
generated codecs. Never try another branch or return partial JSON after failure.
The constructor remains a selection operation; marshaling is the wire-validity
boundary. Transport owners route the resulting error through their existing
encoding-error paths; this design introduces no alternative success body.

The dual check also rejects input that a concrete Go type would happen to
disambiguate but the public schema would not. For example, arrays of `Int32` and
`UInt32` currently both project to integer/int32 item schemas. `[-1]` has one Go
match but two schema matches and must fail. Authors can supply actual disjoint
constraints or use a tagged union. Numeric representation accidents cannot be
implicit discriminators. This is an intentional tightening for existing object
unions with the equivalent numeric-field pattern as well.

### Empty, null, and overlapping values

| Input / selected value | Result |
| --- | --- |
| `[]` with one array branch and one object branch | Array branch, if its minimum length allows zero |
| `[validItem]` | Array branch only if every element and array constraint passes |
| `[invalidItem]` | Rejected; no object or raw-value rescue |
| `{"total":0}` in the example | Page branch |
| `{}` in the example | Rejected because `total` is required |
| `[]` with two array branches that both admit empty arrays | Ambiguous; reject regardless of branch order |
| Two object branches both accepting the same object | Ambiguous on decode and encode |
| Selected nil slice | Existing JSON-v2 slice representation is `[]`; retain the selected array identity and apply the same uniqueness check |
| Selected nil object pointer | Reject as an invalid branch payload |
| `null` supplied to the union value codec | No object/array branch matches; reject |
| Absent optional body/field or an explicitly nullable outer wrapper | Handled by existing presence/nullability owner before invoking the union value codec; never selects an array branch |

Defaults cannot repair a missing required member to make a branch match. Normal
defaulting of optional members is permitted after structural checks. Round trips
preserve branch identity and the declared decoded value semantics, not discarded
unknown members or byte formatting. A union that has some ambiguous values remains
a valid design; those particular values fail at runtime. Proving arbitrary branch
disjointness at design time is not part of this change.

## Alternatives and tradeoffs

Allowing arrays only in the existing object-oriented loops would duplicate more
policy in service and HTTP generators. First-match decoding would make declaration
order affect meaning. Query-selected decoding would couple reusable types to
endpoint policy. A raw JSON result would remove the typed guarantee. None satisfies
the existing exact-one contract.

Checking JSON adds separate schema membership and typed candidate decoding work.
The schema predicates cover the supported projected contract; the runtime does
not parse OpenAPI documents or run a general JSON Schema interpreter. Shape
filtering bounds the common array-or-object case, but correctness takes precedence
over avoiding that work. Do not add a switch to bypass the check. Performance work
may optimize matching only if it preserves the same acceptance and identity.

## Proof obligations and bounded acceptance

The existing [Lean value model](../../expr/lean/value_projection/README.md) already
models arrays, exact-one decode, retained-identity projection, and ambiguous
emission rejection (`ProjectionControls.untaggedEqualPayloadBranchesRejected`).
It supports this direction but does not prove correspondence with newly generated
Go. The [TLA matching model](../../pkg/tla/untagged_json/README.md) separately
checks schema/decoder agreement, emission identity and transactional assignment.
It reproduces the old emission asymmetry and keeps actual predicate extraction
and typed-codec correspondence as Go test obligations.

Use a bounded extension of existing untagged fixtures: mixed array/page, overlapping
arrays (including empty), overlapping objects, nested invalid elements, effective
constraints, defaults, nullable elements, and explicit closed objects. Each distinct
changed behavior needs direct assertions plus generated compilation and actual
service/HTTP/JSON-RPC codec checks. Protobuf needs a focused identity/wrapper
control, not a new transport-wide matrix. Import needs one supported round trip
and rejected-shape controls.

Compare affected generated outputs against the base and between isolated generation
processes. Intended differences are the shared typed adapters/matcher calls and
new array branches/schema components; existing tagged controls must remain
byte-identical. Validate both rendered OpenAPI targets with libopenapi and Redocly.
No exhaustive exported-design corpus or new pairwise test system is required.

## Implementation and verification

`expr/untagged_validation.go` owns recursive admission. JSON-tag admission
uses the shared effective-tag lookup, including options embedded in
`struct:tag:json:name` and complete-tag precedence. Review exposed and regression
coverage closes the name-metadata case-folding bypass.
`expr/value_plan_json_shape.go` lowers captured effective predicates; `Plan` in
this package supplies finalized member names, visibility and HTTP closed-object
policy. `pkg/json_shape.go` evaluates those immutable descriptions, and
`pkg/json_union.go` requires independent schema/decoder agreement. The generated
`matchJSON` adapter retains branch-local typed values and assigns only on success.
Exact numeric evaluation propagates a fatal error if `math/big.Rat` cannot
represent an exponent; it must never reinterpret evaluator limits as a branch
mismatch and accidentally select another branch.

Validation for #350 used Go 1.27.1, protoc 35.1 and the dependencies in `go.mod`.
The base was `2195e43cc21cfb00ad3a2781ce4bd5846cd26d0e`. The original direct
mixed-array regression failed there because an array was not a named object.

The bounded generated comparison reused the `internal/testdatacompile` wrapper
in separate temporary modules, with `LOOM_DIR` set explicitly to each source.
It ran `loom gen`, `loom example`, `go mod tidy`, `go build ./...` and
`go vet ./...`. Generation used the same executable basename and module path;
complete `gen/` trees were compared byte for byte, including both OpenAPI files.
Every current specimen was also generated by a second independent process.

| Specimen | Obligation and observed differences |
| --- | --- |
| HTTP `MappedNamesDSL` | Existing object unions, bytes enums, service versus HTTP member names. Only `mappednames/service.go` and the client/server `types_unions.go` changed: selected-output matching, transactional decoding, and shared typed matching adapters. OpenAPI and all other files were identical. |
| HTTP `PayloadQueryBoolDSL` | Ordinary HTTP generation control; every generated byte identical to the base. |
| gRPC `ConstructorOneOfFieldDSL` | Tagged protobuf union control; every generated byte identical to the base. |
| HTTP `UntaggedArraysDSL` | Newly admitted arrays/pages, nullable members, empty slices, constrained elements, API closed objects, overlapping integer arrays and overlapping objects. Generated module and example compile/vet; isolated outputs identical. |

No checked-in integration fixture declares `Untagged`; none required regeneration.
The old nested-map untagged specimens exercised a validation loophole and were
replaced by explicit recursive rejection tests. Their tagged map and deterministic
serialization coverage remains in place.

Direct and package checks cover `pkg`, `expr`, `internal/unionjson`,
`codegen/service`, and OpenAPI IR. Focused HTTP tests cover union analysis,
mapped-name codecs, retained bytes examples, tagged boolean maps and determinism.
Generated HTTP checks assert status, problem code/detail, invocation and decoded
state for required/optional empty, whitespace, malformed, truncated, null, invalid
and valid bodies. JSON-RPC round trips and protobuf schema/wrapper compilation
cover their distinct boundaries. Import tests compile/re-export the mixed shape
and retain rejected-shape controls. Both OpenAPI targets pass libopenapi and
Redocly 2.46.1. Direct matcher/projection tests also pass with the race detector.
The TLC results and their assumptions are recorded beside the model.

This is bounded ticket validation, not an exhaustive exported-design audit.
