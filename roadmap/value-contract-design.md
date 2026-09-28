# Value meaning and transport projection

Status: proposed contract for the root-cause repair; implementation has not started.

## Summary

Resolve the meaning of examples, enum members, and defaults once against the
evaluated design. Preserve presence, declared scalar type, authored provenance,
and every selected union branch until a transport projects the value. Projectors
may change representation or omit an unusable documentation example; they may
not invent values, select another branch, or silently repair a contract.

This repairs the value pipeline. It does not replace Loom's DSL, service types,
transport runtime, or supported JSON contracts.

## Problem and evidence

The reference revision is `f5b786b39675e2b5c1466f04f3301b7b337779b6`.
The fork point is Goa `aa13905210321b63421d929afda6e8acdb4864b6` (March 17).
Upstream was inspected at `e82f6c1132d09c4777a029d669866396b7d3c9b2`.

- At the fork point, `expr.Union.Example` discarded the selected branch and
  returned only its payload. Goa later preserved canonical envelopes in
  [86484276](https://github.com/goadesign/goa/commit/86484276). Loom's
  `expr/example_canonicalization.go` instead inferred the branch afterward;
  `http/codegen/openapi/internal/ir/example_generation.go` now retains selections
  through a private copied graph for OpenAPI only.
- Loom introduced byte-to-text conversion in the OpenAPI IR in
  [534127ef](https://github.com/CaliLuke/loom/commit/534127ef). The fork-point
  generator encoded actual byte slices correctly. Recent repairs correct that
  behavior but leave different paths for CLI, schema, enum, and default values.
- HTTP CLI examples authored as text for `Bytes` fail their advertised decoder
  at the fork point and in current Loom ([#565](https://github.com/CaliLuke/loom/issues/565)).
  Current Goa also fails the authored-text case, with a generation panic.
- `grpc/codegen/client_cli_example.go` can synthesize a replacement when a type
  contains a union. [#434](https://github.com/CaliLuke/loom/issues/434) tracks
  authored-value loss. Comparative probes also found authored-value loss without
  a union; do not attribute all gRPC example problems to that helper alone.
- Untagged unions were added for [#302](https://github.com/CaliLuke/loom/issues/302)
  to retain existing bare-object response bodies. The initial flat-object scope
  expanded to named nested objects and arrays in `7d3e0bcd`. Removing their
  encoding would break those API contracts.

These are selected reproductions and source-history findings, not an audit of
all Goa or Loom behavior. Upstream supplies useful design precedents, not a
blanket correctness oracle.

## Scope

V1 covers generation-time example selection, scalar coercion, field and map-key
identity, union selection, enum/default value projection, and their consumers:
inline JSON Schema in `expr`, OpenAPI 3.1/3.2, HTTP and JSON-RPC JSON bodies and
CLI examples/defaults, shared collection-enum validation, and protobuf CLI JSON.
Payload, result, designed error, and streaming-message values use the same rules
where the owning transport already exposes those surfaces. Headers, parameters,
cookies, forms, and raw byte flags keep their current transport codecs.

V1 does not add union shapes, change service signatures or protocol runtime,
rewrite validation generation, change nullability, migrate to Goa, or remove
supported custom codecs. Mixed array/object unions (#350) and general naming or
streaming fixes remain separate work. No new DSL configuration is introduced.

## Current ownership

| Owner | Current decision | Required disposition |
| --- | --- | --- |
| `expr/example.go`, `expr/types.go`, `expr/random.go` | Authored precedence, synthesis, recursion/memo behavior | Resolve provenance and selections at this source |
| `expr/example_canonicalization.go` | Field naming, map keys, branch inference | One semantic resolver; preserve public compatibility adapters |
| `internal/enumvalue/value.go` | Coercions and another union traversal | Move semantic decisions into the resolver; remove duplicate selection |
| `expr/json_schema_inline.go` | Inline schema examples/defaults/enums | Consume the same resolved values and JSON projection |
| OpenAPI IR `document*.go`, `example*.go`, `analyzer.go` | Visibility, coercion, completeness, branch matching | Keep visibility and final schema validation; remove semantic reselection |
| `http/codegen/service_data_payload.go`, `codegen/cli` | CLI body examples/defaults and serialization | Consume JSON-body projection; retain plain-flag codecs |
| `codegen/service/service_data*`, `grpc/codegen/service_data_analysis.go` | Transport copies and example provenance | Carry selected authored origin across copies |
| `grpc/codegen/client_cli_example.go` | Synthesis and protobuf mapping | Keep protobuf mapping only; consume shared semantic selections |
| `codegen/validation_render.go`, `internal/jsonkey` | Enum comparison and map-key spelling | Share typed scalar rules and key spelling; preserve runtime acceptance |

## Proposed design

### Semantic values

The resolver belongs in `expr`, which already owns evaluated type semantics and
example selection. Its implementation remains package-private. A minimal,
documented semantic query API exposes an opaque resolved value and outcome to
generators; it does not export traversal helpers merely to share implementation.
The resolved value API belongs to `expr` rather than exposing an `internal` type
in a public signature. This is a new semantic query, not new authoring syntax.

The query has three required operations: select an authored source (or record
that none exists) for an effective occurrence; resolve an explicitly supplied
value with its provenance and role; and synthesize a resolved example only when
selection found no authored source. Resolution never invokes synthesis. JSON
projection accepts only a resolved value plus a target-shape plan and returns a
wire value plus an outcome. Exact Go signatures are an implementation detail;
these responsibilities and outcomes are not optional.

The semantic authority is the **effective service occurrence after DSL
finalization**, including method-local type copies and validation overrides.
A component analyzed without a method has its own finalized type occurrence.
The authored origin only selects the source value; it cannot override that
occurrence's constraints. Resolve the service occurrence once and carry the
result into its transports. A view, selected HTTP Body, parameter, or protobuf
wrapper is a projection plan referencing the service field/branch identities;
it specifies selected fields, target names, requiredness, and codec constraints.
It never resolves the raw value again. Missing fields remain recorded in the
semantic graph; completeness is evaluated for the projected target, so a missing
header does not invalidate an otherwise complete body example. A copied type
with changed semantics is a new effective occurrence and resolves independently,
while retaining the authored source identity. Memo keys include that occurrence,
source and role. This separates reuse of an authored value from reuse of a
semantic result.

Documentation-only `OpenAPIRequestBody` and `OpenAPIBody` attributes are
independent finalized semantic occurrences: their schema and examples need not
correspond to any service payload/result. Resolve their own authored values and
validate against the emitted schema and declared media representation; do not
invent a runtime conversion or require a generated decoder. An explicit
transport-local example overrides the inherited service example for that target
using the current `ExtractUserExamples` precedence. Resolve that override once
against the effective target occurrence and retain a distinct source/cache key;
other transports continue using their selected service source. Inherited copies
without an explicit override reuse the service resolution and projection plan.

The proposed resolved graph has private storage and read-only access for:

- presence: absent, explicit null, or present;
- scalar kind and value: bytes remain bytes, numbers retain declared precision;
- object fields identified by the evaluated field, with authored and wire names
  in the projection plan, rather than repeatedly renaming map keys;
- collection elements and map entries before member-name conversion;
- a union occurrence identity, selected branch identity, and resolved payload;
- provenance: authored example, synthesized example, enum member, or default.

Branch identity is tied to the evaluated occurrence and declaration, never to a
payload hash. Copies retain authored identity but own mutable analysis state.
Read access to built-in byte slices, maps, and collections cannot mutate the
design or another consumer; accessors return copies or immutable views. The graph is generation-local, never a new runtime representation.

Existing public `AttributeExpr.Example` and custom OpenAPI projector callbacks
retain their raw-value contracts. They are compatibility boundaries, not the
route used by built-in generators after migration. `CanonicalizeExample` retains
its documented JSON-shape contract through a thin adapter. On incomplete values
it preserves available fields as today. On ambiguous, invalid, or unsupported
values it returns the same raw/pass-through shape as the legacy implementation;
its `any` signature gains neither errors nor panic behavior. Suppression is not
owned by that API. Built-in consumers cannot use this compatibility fallback to
avoid the new outcomes. Golden adapter tests pin these cases separately. The new semantic
query is the only built-in entry for type-directed value resolution.

### One-way projections

```text
DSL and evaluated attribute
        |
 authored selection / synthesis
        |
 semantic resolution (type, presence, branch, provenance)
        |
        +-- Loom JSON projection -- OpenAPI visibility/schema check -- JSON/YAML
        |                      +-- HTTP/JSON-RPC JSON CLI and body defaults
        |                      +-- inline JSON Schema
        +-- protobuf projection -- protojson CLI examples
        +-- existing plain-flag codecs
```

JSON projection belongs with expression JSON semantics and produces a distinct
wire-value type. OpenAPI-specific visibility and schema checks stay in its IR.
Protobuf projection stays in `grpc/codegen` and uses allocated protobuf fields
and wrappers. No projector calls random generation, guesses a selected branch,
or feeds its wire output back to the semantic resolver. Bytes become base64
once for JSON; raw byte flags continue to accept their existing literal text.

Enum members, defaults, and examples share scalar coercion and equality rules.
Their *use policies* remain distinct: a documentation example can be omitted;
an invalid enum/default cannot silently disappear from the contract.

### Resolution and failure policy

Outcomes distinguish `resolved`, `incomplete`, `ambiguous`, `invalid`,
`unsupported`, and `suppressed`. Null is a value, never a failure sentinel.
Diagnostics name the service/type, attribute path, surface, and reason.

| Input or outcome | Examples | Enum/default contract |
| --- | --- | --- |
| Valid authored value | Preserve supplied fields/values; never replace with random data | Preserve normalized value |
| No authored value | Synthesize with recorded branch choices | No synthetic enum/default |
| Optional field absent | Leave absent; do not fill it because its type contains a union | Existing absence/default semantics apply |
| Required field absent | Preserve partial resolved information; omit executable/schema example with a diagnostic | Reject invalid declaration |
| Ambiguous authored branch | Omit example with a diagnostic; never choose first branch | Reject ambiguous contract value |
| Wrong type, map-key collision, cycle in authored data | Path-qualified generation error | Path-qualified generation error |
| Suppressed examples | Emit no example and no missing-example warning | No effect on enum/default |
| Supported value unrepresentable on a target | Omit example with target diagnostic | Fail that target's generation |

The processing order is: existing DSL validation; target reachability and
visibility selection; existing example suppression/precedence rules; source
selection; semantic resolution; target projection; target validation; emission
or diagnostic. Suppression wins over newly introduced example checks: an
example that the current suppression rules omit is not resolved or serialized
and cannot newly fail generation. Existing DSL errors still fail normally.
New target checks do not inspect unused or target-excluded types. Enum/default
checks on reachable contracts run independently of example suppression.
Suppression respects current authored-example exceptions; this is not a new
blanket interpretation of `openapi:example=false`.

For retained custom OpenAPI callbacks, the caller continues receiving raw
examples and owning its returned representation. Built-in guarantees do not
silently rewrite custom output. Custom codec behavior is covered by explicit
compatibility probes; it is never inferred from ordinary byte/string rules.

## Detailed behavior

### Union selection

Synthesized values retain the selected branch at generation time, including
nested unions with identical payload shapes. Authored raw values resolve once
using current authored matching rules, including the preference for a unique
branch with a known object field. Matching checks declared types before
coercion: typed bytes are not reinterpreted as an authored String value.
Primitive coercion and enum equality occur against that candidate's declared
semantic type. More than one preferred candidate is ambiguous. This deliberately changes one
existing example fallback: authored string `"hi"` for otherwise competing
`Bytes Enum("hi")` and `String Enum("hi")` branches is omitted with an ambiguity
diagnostic, rather than allowing late wire matching to select the String branch.
Typed `[]byte("hi")` still resolves to Bytes. This affects advertised examples,
not runtime acceptance, service signatures, or untagged wire bodies. No new
branch-selection DSL is part of V1; an ambiguous example may remain omitted.

JSON tagged projection emits existing configured discriminator/value keys and
branch tags. Untagged projection emits the selected object directly. For an
untagged *wire value*, every candidate validates the same final JSON value;
exactly one must match and it must correspond to the retained selected branch.
Zero/multiple matches cause an example omission or contract-value error under
the table above. Validation here confirms representability; it cannot select a
replacement. The same requirement applies after OpenAPI visibility projection.

No untagged shape currently rejected becomes supported. Valid bare JSON remains
bare; public Go constructors, accessors, and wire tags remain unchanged.

### Coercion, presence, and naming

- Declared Bytes accepts the currently supported authored text and byte slices;
  semantic equality uses bytes, including the existing empty-byte equivalence.
  String and Any do not inherit that coercion.
- Integer boundaries and floating-point rounding follow the declared type.
  Protobuf projection additionally honors the actual protobuf field range and
  protojson representation; it cannot synthesize a new number to make it fit.
- Requiredness, nullability, nil/empty serialization, and absent optional fields
  follow their existing evaluated and codec contracts. No new equivalence
  between absent, null, and empty is introduced.
- JSON object names use `ElementName`/`JSONFieldName`; protobuf names use its
  allocation plan. Existing accepted authored-name/wire-name aliases and
  precedence remain. Two distinct map keys that become the same JSON member
  name fail before insertion; map iteration order cannot pick a winner (#456).
- Unknown object members retain existing open/closed-object behavior. Any and
  custom representations retain their codec-owned JSON semantics under the
  explicit ownership rules below.

### Presence and opaque values

| Input | Resolved meaning | JSON example projection | Enum/default projection |
| --- | --- | --- | --- |
| No source example | Absent source | Synthesis permitted unless suppressed | No synthesis |
| Missing object key | Absent field | Remains absent; target-required field makes that target incomplete | Existing default application is not an example operation |
| Authored explicit null (`ExampleExpr.ExplicitNull`) | Null, with source presence retained | Emit null only when target admits null; otherwise invalid | Preserve accepted nullable null; reject non-nullable declaration |
| Typed nil array/map used as an example | Present nil-container marker | Omit as incomplete, preserving current OpenAPI omission; CLI also omits an unusable nil-container hint | Non-null collections use existing empty `[]`/`{}` JSON coercion; nullable null semantics remain as declared |
| Typed nil object pointer | Null-like value | Nullable target may emit null; non-nullable target is invalid | Follow declared nullability; never turn into an empty object |
| Empty concrete array/map | Present empty collection | Emit collection unless generated field omission removes it | Preserve existing collection defaults and enums |
| Declared Bytes, empty string/slice or nil byte slice | Present empty bytes | JSON string `""`; a field with `omitempty` may omit it | Existing byte equality treats these as equal |
| Any containing `jsontext.Value("null")` | Present codec-owned JSON null | Preserve null when the target permits it | Preserve JSON null |
| Any containing typed nil bytes or a custom value | Opaque codec input | Materialize using its actual JSON codec, e.g. JSON v2 nil bytes become `""` | Same codec semantics; no declared-Bytes coercion applied to Any |

These rules intentionally permit advertised-example changes for typed nil
containers/pointers when old CLI formatting disagrees with the schema. They do
not change runtime codecs. Target round-trip equivalence is defined after the
existing generated encoder's omission/default rules: absent and empty may be
equivalent only for a field whose existing `omitempty` behavior collapses them;
absent and null remain distinct for explicit presence wrappers. Required empty
collections that cannot survive their existing wire contract are not advertised
as valid examples. Protobuf equivalence uses service conversion and protobuf
presence/default rules rather than JSON field-presence identity. Each such
collapse must be represented in the proof matrix, never a global equality rule.

Opaque custom values are borrowed compatibility inputs. The framework must not
mutate them, but cannot promise that a user marshaler is pure or deterministic.
Each target materializes a borrowed value once per occurrence into an owned
snapshot; subsequent rendering reads that snapshot. A custom marshaler's side
effects remain user-owned. Cross-process determinism and source immutability
claims apply to built-in values and pure custom codecs only. A materialization
error is a path-qualified target-generation error for authored examples,
enums/defaults, and an omission diagnostic for synthesized examples. Raw custom
OpenAPI projector callbacks retain their existing acceptance/failure contract
outside this materialization route.

### Synthesis and recursion

Authored examples retain `ExtractUserExamples` precedence. Transport copies
carry the selected source example instead of falling back to synthesis.
Synthesis records branch choice and terminates at recursive edges using the
existing recursion rules. A required recursive edge with no finite valid
example yields `incomplete`; generation of an otherwise valid service remains
possible. Memoization cannot mix raw, resolved, and projected values. Each cache
entry is owned by its representation and occurrence; projected envelopes are
never cached under only a leaf type. A broader redesign of random identities
is deferred: changing unrelated examples is not necessary for this repair.

## Surface contract

The source of truth for every row is the evaluated DSL plus selected authored
example/enum/default. Default routing is automatic; no metadata enables the
repair and no deployment registration changes. Public schema/component/Go names
and aliases stay unchanged. Unsupported shapes are rejected by existing design
validation; widening them is deferred.

| Surface | Placement and routing | Supported/default behavior | Rejection/omission | Equivalence proof |
| --- | --- | --- | --- | --- |
| OpenAPI bodies, parameters, headers, errors, async messages | Shared JSON value into existing IR visibility/schema checks, then 3.1/3.2 renderer | Existing supported examples, explicit nullable null, projected enums/defaults | Failure policy above; existing external/serialized example metadata remains opaque | Every emitted ordinary example validates against its actual emitted schema; JSON/YAML agree |
| Documentation-only HTTP bodies | Independent finalized `OpenAPIRequestBody`/`OpenAPIBody` occurrence into OpenAPI IR | Own schema, authored precedence and media representation | Same example failure policy; no runtime routing | Emitted example validates against emitted schema/media; no generated decoder exists |
| HTTP/JSON-RPC JSON CLI bodies | Same JSON projection into existing CLI formatter and builder | Authored branch, fields and bytes retained | Omit unusable example/hint; keep command available | Advertised JSON passes generated builder and preserves semantic value |
| Plain flags / non-JSON locations | Existing location codec | Existing text, byte, numeric and collection syntax | Existing rejection behavior | Prior flag/default matrix unchanged |
| gRPC CLI | Resolved message through existing protobuf allocation/wrapper plan | protojson spelling and representation, authored values retained | Omit unusable hint with diagnostic; never replace authored data | Actual builder accepts sample and service-side conversion preserves supplied fields/branch |
| Inline JSON Schema and shared enum validation | JSON schema projector and typed enum comparison | Same coercion/equality as semantic resolution | Invalid contract declaration fails | Runtime validation and schema enum values agree for supported contracts |
| Custom example projector / custom codec | Existing extension boundary | Preserve raw callback input and codec-owned representation | Existing custom failure behavior | Dedicated probes preserve callback calls, inputs and returned values |

## Compatibility and alternatives

Unaffected generated Go, protobuf schemas, names, and protocol bytes must remain
byte-for-byte identical. Enumerate changes to examples, help diagnostics, and
new path-qualified rejection of invalid values. OpenAPI differences outside
example/default/enum values require a separate design decision, not a golden
update. A runtime/schema disagreement discovered by the proof matrix is a
Design Blocker: do not silently broaden acceptance or normalize it away.

Adopt the upstream branch-retention principle selectively. Copying current Goa
wholesale would discard Loom-specific contracts and still retain observed
upstream failures. Removing untagged support would break existing imported API
shapes. Adding another transport-specific normalization helper perpetuates the
root cause. A single universal wire format would also be wrong: JSON and
protobuf have different representations of the same semantic value.

## Proof obligations

1. Projection preserves supplied values and selected branch, and preserves presence
   under the target-specific equivalence defined above, through
   the actual generated decoder/conversion path for every runtime-backed surface.
   Documentation-only bodies instead satisfy emitted schema/media validation.
2. Every emitted ordinary example validates against the emitted schema;
   validation uses an independent parser/validator, not only the resolver.
3. Authored values are never replaced by synthesis; absent optional fields stay
   absent. Failed examples have an explicit outcome, not arbitrary null/text.
4. Built-in inputs are immutable; process-isolated generation is byte deterministic
   for built-in values and pure custom codecs, with borrowed opaque values tested
   under the explicit ownership limits above.
5. Old counterexamples fail the old pipeline and pass the replacement, including
   nested identical branches, byte/text enums, binary/empty bytes, field mapping,
   recursive types, optional/null values, map collisions and protobuf ranges.
6. A bounded model checks phase ordering, branch/presence preservation and cache
   ownership; its abstraction mapping and excluded cases are documented. Go
   property tests and generated round trips discharge the implementation seams.
7. Migration is complete only when the inventory has no built-in bypass that
   independently reselects branches or applies scalar coercion.
