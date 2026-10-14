# Custom physical representation in service views (#926)

Implementation and bounded acceptance are complete. The combined source packet
awaits independent exact final review and atomic delivery. Steps 1–5 below are
complete; step 6 remains the delivery gate.

1. Preserve the published service-only UUID validation/default compiler failures
   and add direct owner regressions before production changes.
2. Bind semantic operations to physical representations in the shared codegen
   layer. Inspect ordinary custom aliases, opaque text types, raw JSON overrides,
   pointer/optional carriers, native nullable controls and both view conversion directions. Custom metadata combined with nullable remains a rejected design.
3. Model custom default success/failure and nested conversion propagation before
   implementing a fallible conversion plan. Obtain independent readiness review
   of the exact contract, affected signatures and bounded acceptance.
4. Correct the shared validation/default representation owners and make their
   consumers derive operations and failure propagation from that plan. Preserve
   supplied zeros, null and excluded view fields; do not panic or discard defaults.
5. Verify generated service-only and transport consumers, custom parser failures,
   native/named controls and recursive helpers. Compare a bounded selection of
   generated artifacts at the parent and candidate, repeat in separate processes,
   compile changed shapes, and list every intended difference.
6. Independent exact final review, atomic delivery and issue closure.

## Confirmed owners

`newValidationRenderData` derives String operations solely from the DSL kind and
passes custom physical values directly to rune counting, patterns, formats and
enums. `AttributeContext.DefaultValue` notices metadata but emits the authored
native literal without converting it into custom storage. `GoTransform` then
assigns that expression in the correctly selected absence branch. The presence
owner is already correct and must not be replaced.

The service-only UUID witnesses reproduce independently: a MinLength declaration
emits `utf8.RuneCountInString(*result.ID)`; a String default emits an assignment of
that string to `uuid.UUID`. Neither witness requires HTTP mappings. #930's stale
projected constructor package is a different owner and is excluded here.

## Accepted contract boundary

`Meta("struct:field:type", ...)` promises physical storage override, including
ordinary custom Go aliases and raw JSON types. The documented Stringer /
UnmarshalText contract applies to mapped HTTP String fields. Therefore merely
calling those methods on every metadata override would broaden the contract and
break existing plain custom aliases. Native/named String storage must retain
ordinary value semantics; opaque representations require an explicit supported
semantic projection/materialization operation. Do not guess struct fields or
assume Stringer text is accepted by an arbitrary parser.

The readiness comparison selected the first direction:

- Carry one representation capability through generic validation/default code,
  preserving native custom aliases and using authored codecs only where their
  contract is established. Unsupported opaque operations must fail clearly.
- JSON round-tripping all custom values is insufficient: a mapped HTTP text codec
  can intentionally differ from an authored JSON codec. The semantic owner must
  not silently select a transport's representation for all other consumers.
- Rejecting every custom default or skipping view validation would discard
  authored intent and retain the defect. Runtime panics are not acceptable error
  propagation.

## Fallibility boundary

The original generic conversion generator emitted infallible helper calls.
`TransformFunctionData.ErrorAware` already exists and is populated by gRPC's
conversion owner; generic GoTransform now derives it from the selected conversion graph. The
shared plan owns supplied representation crossings and absent custom defaults.

Generated `NewRecordFromRecordView` and recursive transform helpers currently
return a value only. The outer `NewRecord` already returns `(value, error)`;
`NewViewedRecord` likewise returns validation faults. A custom-default parser
failure must reach those boundaries. The implementation plan must enumerate
exact affected signatures/callers before changing them; globally converting
unrelated native-only transforms to fallible APIs is not justified by this bug.

## Bounded acceptance

- UUID and a non-UUID struct representation; ordinary custom String alias and
  native/named String controls; raw JSON override remains explicitly controlled.
- Minimum/maximum length, pattern, enum and format use semantic values while
  preserving required/optional guards and selected-view membership.
- Absent custom defaults materialize; supplied zero values remain supplied. Native nullable controls retain explicit null, and custom-plus-nullable designs remain rejected. Invalid custom defaults return an error without publishing a
  partially converted result. Authored parser errors retain causal wrapping.
- Nested objects, collection elements and recursive helpers propagate the same
  fallibility; native-only outputs and signatures remain unchanged.
- Direct service generation plus representative HTTP/JSON-RPC callers prove
  boundary adoption. No exhaustive exported-design sweep is required.

## Implemented representation contract

Introduce one semantic String capability at the runtime/codegen boundary:

- A native or defined Go string has native String semantics. A custom type's
  authored text codec takes precedence where it supplies the operation:
  `fmt.Stringer` for a semantic value and `encoding.TextUnmarshaler` for authored
  text materialization. Do not assume the two operations are inverse; validation
  checks the actual projected text and materialization checks the parser result.
- An opaque custom storage type without the required operation returns a clear
  representation error when that operation is needed. Do not guess its fields,
  format an arbitrary struct with `%v`, or switch silently to its JSON codec.
- Keep the explicit existing raw-JSON representation owner distinct. Native and
  named values without overrides retain generated bytes and signatures. A plain
  custom string alias remains valid without implementing text methods.
- Capability dispatch may use the runtime Go kind for defined string storage;
  it must not inspect or infer arbitrary object layouts. Shared runtime helpers
  need public names only because generated consumer modules must call them.

Carry materialization as an owned code plan, not an expression pretending all
values are literals. The default renderer must compose native literals and
fallible custom materialization through nested defaults. The transform plan must
carry code, helper declarations and fallibility together. Reuse
`TransformFunctionData.ErrorAware` for helper declarations and derive parent
fallibility from selected conversion dependencies, including recursive types.
Applied defaults and supplied values crossing physical representations can add failure; absent fields excluded by a selected view
must not affect behavior or signatures.

### Affected public signatures and callers

- Generated view-to-service helpers `NewTFromTView` return `(T, error)` only when
  their selected conversion can materialize a custom String default or cross a
  physical String representation, directly
  or through a helper. `NewT` already returns an error and propagates it.
- Service-to-view `ProjectT` does not materialize defaults in pointer targets;
  its signature remains unchanged. `NewViewedT` already returns validation
  faults and can propagate semantic String projection errors through that path.
- Generated HTTP body/result constructors and service ConvertTo/CreateFrom
  helpers whose conversion plan is fallible gain an error result. Their call
  sites derive propagation from that plan. HTTP decoders/encoders, CLI builders,
  and JSON-RPC transport adapters already own error-returning outer boundaries.
- gRPC has an existing fallibility owner; its uses of the shared default renderer
  must consume the same materialization plan and extend its existing error
  classification when custom defaults apply. Do not introduce a second error
  channel or change unrelated native-only signatures.
- Framework contributor APIs `AttributeContext.DefaultValue` and `GoTransform`
  return a structured result containing code and fallibility.
  Update their bounded set of call sites together rather than add pass-through
  compatibility wrappers. Generated application code must be regenerated.

The actual production consumers are generic GoTransform object and presence
branches, service projection and ConvertTo/CreateFrom builders, HTTP shared
transform/body/field/constructor owners, and the gRPC default/metadata owners.
The implementation must keep raw JSON and native controls and exercise any
changed consumer; this is a single representation/fallibility correction, not
separate patches that each reinterpret custom storage.

The bounded model under `codegen/tla/custom_representation` rejects the existing
native-literal assignment, a panic alternative, and swallowed nested failures.
It supports explicit conversion-error propagation (40 states and completion).
It assumes correct capability selection and parser behavior; implementation tests
must prove those correspondences. Independent readiness review approved this boundary before implementation.
The final fixed-point graph model passes 35 generated / 26 distinct states and
completion, including supplied crossings and default-disabled contexts.


### Readiness review correction: codec method sets and graph ownership

Capability precedence is explicit. For semantic String projection, use the
value's authored `fmt.Stringer`, then its addressable pointer method set, then a
native/defined string's underlying string value. For materialization, use the
fresh target's `encoding.TextUnmarshaler` method set, then native/defined string
assignment. Neither operation invokes JSON codecs as an implicit fallback.
Unsupported opaque storage errors only if a validation/default semantic operation
is actually required; merely storing, copying or projecting unconstrained custom
values requires no new codec. Semantic projection of a nil custom pointer must produce a checked
representation error where an operation is needed, not panic. Fresh default
materialization may allocate the required concrete parser receiver. Explicit pointer
metadata must preserve allocation/absence ownership; the capability must not
recursively allocate an unbounded pointer-type cycle. Native raw JSON retains
its existing explicit owner and tests.

The dependency plan keys nodes by selected source/target conversion pair and
physical contexts, including whether defaults can apply. Seed a node only when
its own emitted conversion can fail; propagate that fact backward through helper
dependencies to a fixed point. Never cache an unvisited node as permanently
infallible, or merge two view selections merely because their Go declarations
have the same names. This makes cycles and traversal order explicit rather than
relying on incidental recursion order.

The additional `FallibilityGraph.tla` model includes a full-view A/B cycle with
reachable fallible leaf, a second branch, a native-only cycle, a tiny-view cycle
that excludes the default-bearing edge, and enabled/disabled default contexts.
The once-visited strategy misses a reachable failure depending on traversal
order. A physical-name-only key contaminates the excluded view. The fixed-point
strategy retains selection/default context and passes all classifications and
completion. These are finite graph checks; concrete pair extraction remains a
test obligation.

## Final implementation and acceptance evidence

- `pkg/string_representation.go` owns semantic projection and fresh parsing;
  `internal/representation` classifies actual metadata without transport policy.
  Unicode named strings, pointer method sets, nil/cyclic pointers, parser causes
  and supplied-value immutability have direct runtime regressions.
- `GoTransform` carries code, helpers and `ErrorAware` together. Its fixed-point
  key includes semantic pair, physical references and presence/default contexts.
  Both sibling orders and recursive selected/excluded pairs are tested.
- Service views, ConvertTo/CreateFrom, HTTP/JSON-RPC constructors and gRPC body
  and metadata consumers derive failure propagation from these plans. A failed
  parser returns an error before publishing a partial result. Native-only
  signatures remain unchanged; applications must regenerate affected code.
- Generated service fixtures compile and execute UUID, non-UUID text, defined
  String, opaque no-operation, selected/default-disabled and recursive cases.
  Only custom validation/default service artifacts change; native tiny-view and
  ordinary projection output stays byte-identical to the parent.
- Public ConvertTo/CreateFrom methods compile and execute both directions; a
  rejected parse preserves the receiver and causal error. Only the conversion
  artifact changes in that fixture, with exact independent-process repetition.
- HTTP/JSON-RPC fixtures compile unary and streaming constructors and execute
  valid/invalid absent defaults through server, client and CLI boundaries.
  Native checked-in transport goldens remain unchanged. Six of 15 HTTP and
  eight of 27 JSON-RPC artifacts change only custom default/error plumbing;
  independent process repeats are byte-identical.
- gRPC fixtures execute provided values, arrays/maps, recursive defaults, parser
  failures and metadata/CLI paths. Four custom body goldens intentionally change
  casts to shared codecs. Native/named metadata output and 20 of 24 bounded body
  artifacts remain byte-identical; separate-process repeats are exact.

The union control covers custom String storage inside a selected object branch,
including imports and actual protobuf representation crossings. A separately
confirmed primitive-branch promotion defect drops authored Go method sets by
emitting a new defined wrapper type; #936 owns that declaration correction.
This ticket does not claim the raw primitive custom-branch shape is repaired.

The model proves only its finite ownership/classification assumptions. Compiler,
wire and runtime regressions check the concrete extraction and generated callers.
No full exported-design audit is part of this ticket. Raw logs, generated temp
modules and checker state remain outside the repository.
