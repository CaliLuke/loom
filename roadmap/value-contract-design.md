# Value meaning and transport projection

Status: accepted contract; implementation is in progress under the linked execution plan.
The independently reviewed #575 clarification settles partial-union selection and
the separately scoped #574 byte-schema correction before candidate implementation.

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

Raw built-in `Any` enum comparisons preserve the existing host-language
`reflect.DeepEqual` precheck before numeric, string-map, and slice fallbacks.
The finite reference model retains an adapter-owned equality-class identity at
every raw node of an immutable built-in snapshot. Identity equality means actual
host `DeepEqual`, not hash equality or equality after JSON conversion. Different
identities still use the established recursive fallbacks; concrete non-string map
and dynamic element types must not disappear before this check. This evidence
cannot bypass payload validity or map-name collision checks and has no effect on
target bytes. Extraction and equality-class assignment remain explicitly tested
host-library correspondence, not a theorem about Go reflection.

## Detailed behavior

### Union selection

Synthesized values retain the selected branch at generation time, including
nested unions with identical payload shapes.

An authored union is selected against its effective finalized service occurrence,
without consulting a target's field selection, visibility, names or wire encoding.
The resolver preserves the existing complete-candidate ranking and completeness
preference: validate each branch with the approved typed authored-value and coercion
semantics, prefer a unique complete candidate satisfying the existing preference
predicate, otherwise choose the sole complete candidate when none is preferred, and
report ambiguity in every other nonempty complete-candidate case. This preserves
the complete candidate set and selection order, subject to the already approved
typed-Bytes/String distinctions and coercion rules, including their explicitly
documented changes to authored ambiguity. A complete candidate must have all
required members and uniquely match every supplied nested union using complete
matching; an ambiguous nested union excludes that candidate from the complete
set, as it does today. Nested partial fallback does not promote an outer candidate
into the complete set. In particular, a complete branch continues to win over a
branch missing required fields or obstructed by nested ambiguity. An ambiguous nonempty complete set
never falls through to partial matching.

For object-shaped authored input, the existing preference predicate unwraps named
aliases. An object candidate is preferred when a supplied key matches one of its
authored or JSON field names. A compatible non-object candidate, including Any,
a map or a nested union, is also preferred. It is not necessary for that candidate
to declare a named object field. Preserve this behavior: a known-field object and
a compatible Any candidate can both be preferred and therefore ambiguous. For
non-object input, selection uses the complete-match count without this preference.

Only when there are no complete candidates may the resolver consider viable
partial or ambiguity-obstructed branches. Partial matching defers missing required object members, recursively;
it does not invent values or relax checks on supplied values. A supplied wrong
scalar type, incompatible null, failed enum/pattern/range/length constraint,
closed-object unknown member, or key collision disqualifies that branch. A
supplied empty collection, empty string or explicit null is a supplied value,
not an absent field. Optional absent fields stay absent. Constraints on a supplied
container are checked on that container; hypothetical extra members/elements
cannot repair a failed constraint. An absent child's own constraints are not run.

Apply the same preference predicate to the viable fallback branches: choose the
unique preferred candidate, or the sole viable candidate when none is preferred.
Multiple preferred candidates, or multiple viable candidates with none preferred,
are ambiguous. Zero viable candidates are invalid. Present nested unions use the
same complete-first rule. Within fallback only, a nested ambiguity with viable
interpretations is retained as an unresolved ambiguity, not relabeled a type
failure: an otherwise viable outer candidate can carry that obstruction even if
its own required fields are all present. A nested union with zero viable
interpretations still disqualifies the outer candidate. Selecting an obstructed
outer candidate does not resolve the nested branch. If it wins, the
whole example remains ambiguous; a target cannot remove the obstruction to choose
an interpretation. A resolved partial value records the selected identities and
missing fields. It can be projected without supplying or synthesizing those fields.

For examples, requiredness is checked against the actual target after projection:
a missing field excluded by that target does not make that example incomplete;
a missing required retained field does. For enum/default declarations, partial
selection must never waive requiredness or another declaration constraint: an
incomplete contract value is rejected even if a target would drop the missing field.

Selection evidence supplied by synthesis or an existing representation-aware
internal boundary is authoritative only for the exact union occurrence and a
known branch. Validate its payload against that branch; do not try another branch
when it fails. An arbitrary DSL Example map is still an authored payload. Keys
that happen to equal the configured discriminator/value keys do not introduce a
new authored-envelope syntax. A discriminator represented by an ordinary branch
field/enum remains an ordinary constrained supplied field. At an existing wire
input boundary, the configured tag and payload members are instead parsed as a
wire envelope; unknown/missing tags, missing payloads and invalid selected payloads
fail that wire boundary without semantic branch fallback. Current public adapters
and custom projector boundaries retain their existing interpretations.

After source selection, each target projects only the retained branch. Untagged
wire validation independently checks every projected branch schema/decoder against
one identical final wire value. Emission requires exactly one wire match, and its
identity must equal the retained branch. Zero matches, multiple matches, or a sole
match for a different branch make that target unrepresentable; they never trigger
reselection or synthesis. Field loss can introduce or remove wire ambiguity, but
cannot resolve an already ambiguous authored source. Tagged/protobuf targets apply
their actual envelope/oneof and selected-payload validation with the same identity
requirement.

The public CanonicalizeExample compatibility adapter does not gain the partial
fallback or new failure behavior. Preserve its existing complete-selection results
and raw/pass-through outputs for partial, ambiguous, invalid and unsupported inputs,
including its nil handling. Built-in generators use the semantic route and cannot
use that fallback to bypass a failure. This is a behavioral compatibility requirement,
not permission for generators to maintain a second branch-selection implementation.

#### Union acceptance cases

All ordering-sensitive cases run with branch declarations reversed. Runtime/schema
checks use the actual target representation; semantic selection assertions inspect
retained identity before projection. No target may feed wire values back to selection.

| Case | Source result | Target / compatibility assertion |
| --- | --- | --- |
| Existing Reply requires thread_id; Resolve requires thread_id plus resolved_by; supplied {thread_id:T1} | Reply selected by the preserved complete-first ranking | Preserve current CanonicalizeExample test and generated example |
| A incomplete but recognizes a; B complete open object recognizes none | Existing complete B remains selected | Partial preference cannot supersede a complete winner |
| Two complete preferred branches plus a partial branch | ambiguous | No partial fallback or target-based tie-break |
| A requires a,h; B requires b; supplied {a:x}; both have no complete match | Partial A selected by unique known field | Target body excluding h emits a:x; target retaining required h is incomplete |
| Same partial A with a supplied h of the wrong type | A disqualified; apply the stated candidate rules to B | Do not treat wrong h as merely missing; explicit selected A fails rather than switching |
| Closed A requires a and closed B requires b; supplied invalid values for both | invalid | Path-qualified generation error, not omitted incomplete example |
| A requires a,h; B requires a,k; supplied {a:x} | ambiguous partial set | Removing h/k or a whole target field does not choose a branch |
| Outer A recognizes supplied a, all its required fields are present, but a has an ambiguous inner union; B is complete/open and recognizes no supplied field | Complete B selected; A is excluded from the complete set | Preserve existing complete branch winner; no fallback occurs |
| Same A; B requires missing b and recognizes no supplied field | No complete candidate; fallback prefers A, retaining its nested ambiguity | Overall ambiguous; excluding a in a target cannot resolve it |
| Same A; B is partial and recognizes another supplied field | Multiple preferred fallback candidates are ambiguous | Neither required-field presence alone nor target projection breaks the tie |
| Unique viable fallback candidate contains a present ambiguous nested union | ambiguous | Excluding that nested field cannot choose a semantic interpretation |
| Candidate contains a present nested union with zero viable interpretations | That outer candidate is invalid | A type-invalid nested value is not an ambiguity obstruction |
| Required field supplied as null/empty versus omitted | Distinct states under declared nullability and constraints | Null/empty are not treated as absent; omission equivalence is field-local |
| Invalid supplied enum/range/length/pattern value versus absent child | Invalid supplied value disqualifies candidate; absent child checks deferred | No synthetic filling; enum/default incomplete declarations still fail |
| Typed bytes in branch A; same field String in B | A selected semantically | Both receive the same final base64 string for wire matching; ambiguous untagged output omitted |
| Source-selected A/B distinguished by kind enum; target drops kind in both | Source A retained | New final-wire ambiguity omits example; contract values fail that target |
| Source uniquely selects Bytes A over String B; both closed branches initially accept the same encoded string; target hides the field only from B | A remains selected | Independent emitted-schema matching may remove B's match because its closed schema now rejects that member; source is never reselected |
| Source ambiguous but one target would admit only A | ambiguous | Target cannot rescue source ambiguity |
| Explicit retained B whose target wire matches only A | B retained but target unrepresentable | No branch substitution |
| Trusted selected identity unknown, wrong occurrence, or payload invalid for selected branch | invalid evidence/payload | Never fall back to another branch |
| Raw object containing type/value-like data keys | Existing authored payload interpretation | No accidental envelope recognition; adapter golden stays unchanged |
| Existing wire envelope with custom keys/tags, malformed tag, absent payload | Valid tag retains branch; malformed cases rejected at wire boundary | No changes to supported decoder contracts |
| Partial fallback cases through CanonicalizeExample | Legacy raw/pass-through shape | New built-in semantics may differ only where the clarification explicitly permits it |


Matching checks declared types before coercion: typed bytes are not reinterpreted
as an authored String value. Authored string `"hi"` for competing `Bytes
Enum("hi")` and `String Enum("hi")` branches is ambiguous and omitted with a
diagnostic; typed `[]byte("hi")` still selects Bytes. This changes advertised
examples, not runtime acceptance or supported untagged wire bodies.

JSON tagged projection retains existing discriminator/value keys and branch
tags. No new branch-selection DSL or untagged shape is introduced; public Go
constructors, accessors, supported bare JSON and wire tags remain unchanged.

### Coercion, presence, and naming

- Declared Bytes accepts the currently supported authored text and byte slices;
  semantic equality uses bytes, including the existing empty-byte equivalence.
  String and Any do not inherit that coercion.
- Primitive source admission precedes normalization and preserves the existing
  `Primitive.IsCompatible` concrete Go type matrix. Named scalars do not become
  admitted primitives merely because reflection exposes the same underlying
  kind or a formatter produces valid numeric text. `Int` and `Int64` retain
  their different source acceptance even on a 64-bit host. Any uses its separate
  finite raw-value contract. The reference must retain enough concrete type
  evidence to state this admission independently of resolver success.
- Sequence source shape and JSON representation are independent. Native byte
  slices and fixed arrays both serialize as base64, but only an unnamed native
  byte slice is admitted by declared Bytes. A named container of native bytes
  still encodes as base64; defined byte elements encode as an array. Declared
  Array checks the original elements, preserving their concrete source types;
  empty arrays retain the existing vacuous element check. Raw host evidence may
  not be reconstructed from a selected branch or target.
- Integer boundaries and floating-point rounding follow the declared type.
  Protobuf projection additionally honors the actual protobuf field range and
  protojson representation; it cannot synthesize a new number to make it fit.
- Numeric meaning and representation identity are separate. Equal exact numeric
  values held as Float32 and Float64 can have different JSON/key spellings; raw
  values retain that precision, and decoding precision belongs to each target
  occurrence. Numeric enum equality ignores representation identity. Map-key
  source collision checks use the contract's actual key conversion and canonical
  member names, not enum equality. Native runtime maps separately require unique
  decoded keys under the actual target type's equality: opposite signed zeros
  collide even when their JSON member names differ. Key admission uses the actual
  JSON map codec, not a standalone numeric parser. Exact reference coefficients must support arbitrary
  precision; encoding a Float32 as its short JSON decimal must not replace its
  exact numeric meaning. Declared floating-point roles apply the established
  authored-literal formatting and destination parsing before validation or branch
  ranking, after source admission. Formatting is an independent boundary and
  cannot admit an otherwise unsupported named source. Numeric source formatting
  methods retain separate literal provenance; normalized scalars lose that
  source-specific formatter. JSON spelling and enum
  numeric equality do not inherit literal-formatting identity.
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

### Byte constraints in JSON schemas

Keep Bytes MinLength/MaxLength semantics as decoded byte counts, and preserve the
existing generated runtime acceptance. For an ordinary Bytes occurrence carried
by Loom's built-in JSON codec, project those semantic bounds to a JSON Schema
constraint on the actual base64 string. Apply the same projection in OpenAPI
3.1/3.2 JSON schemas and InlineJSONSchema, including named wrappers, nullable
wrappers, object members, collection items and inherited/effective bounds. Keep
ArrayOf min/max item counts unchanged; its Bytes element has a separate length
constraint. Do not replace Bytes with ArrayOf(Byte).

This policy explicitly includes unconstrained built-in JSON Bytes: absent lower
and upper bounds mean L=0 and U=infinity, so emit the full accepted base64 grammar
and contentEncoding:base64, not annotation alone. This intentionally tightens
those emitted string schemas to the codec language without changing runtime
acceptance. It is an intended schema output difference even where no length DSL
appears and must be included in the fixture comparison. Other enum/default/null
and presence obligations are not strengthened by this length/grammar theorem.

The schema must not copy decoded byte bounds directly to string minLength and
maxLength. Nor may it use only rounded encoded bounds: 1, 2 and 3 decoded bytes
all occupy four encoded characters. Removing the bounds or omitting valid
examples is not a resolution of this blocker.

The narrow equivalence obligation is: for a JSON string s under the built-in
Bytes codec, the generated Bytes-length constraint accepts s exactly when that
codec decodes s successfully and the decoded byte count satisfies the effective
length bounds. It is not a theorem about all other field constraints, null or
absence, arbitrary custom codecs, or the compiled Go program. Their existing
constraints and presence rules remain conjoined and independently checked.

#### Exact finite schema construction

Let A be [A-Za-z0-9+/]. For decoded length n=3q+r, r in {0,1,2}:

| r | String grammar | Encoded length |
| --- | --- | --- |
| 0 | (A{4})* | 4q |
| 1 | (A{4})* A{2} == | 4q+4 |
| 2 | (A{4})* A{3} = | 4q+4 |

For semantic lower bound L (zero if absent, otherwise clamped to at least zero)
and upper bound U (unbounded if absent), each residue branch has q >= max(0, ceil((L-r)/3)) and, when U exists,
q <= floor((U-r)/3). Omit an empty branch. Add the corresponding encoded
minLength/maxLength to that branch, with an anchored grammar pattern. Combine
the at most three branches with anyOf (or equivalent). This keeps the grammar
constant-size rather than expanding a repetition count proportional to U.
Use checked integer arithmetic; a schema bound that cannot be represented safely
must cause a path-qualified target-generation error, never wrap or silently relax.
A negative lower bound is vacuous and projects as zero. A negative upper bound
accepts no decoded byte string and projects as a standard unsatisfiable schema
(e.g. `not: {}`), never a negative JSON Schema length keyword. Preserve existing
DSL rejection of explicitly inconsistent MinLength > MaxLength declarations. If
an effective/inherited range reaches projection with no possible nonnegative
byte count, project an unsatisfiable constraint. Do not add new declaration
errors for currently accepted negative bounds and do not emit an empty anyOf.
This fixes the semantic disposition; the exact equivalent schema spelling can
follow the existing representations.

ECMA-262 `$` can match before a final line terminator. Require that the entire
string contain only the base64 alphabet and padding, e.g.
`not: {pattern: "[^A-Za-z0-9+/=]"}`, in addition to the anchored branch pattern.
Do not depend on a Go-only end anchor or a validator-specific regex extension.
The exact schema shape is an implementation choice subject to the same language
and dialect checks; supporting `not` in existing schema representations may be
necessary. This is not permission to add a second independent schema owner.

For exactly two bytes, one valid candidate is:

```json
{
  "type": "string",
  "contentEncoding": "base64",
  "not": {"pattern": "[^A-Za-z0-9+/=]"},
  "anyOf": [{
    "pattern": "^([A-Za-z0-9+/]{4})*[A-Za-z0-9+/]{3}=$",
    "minLength": 4,
    "maxLength": 4
  }]
}
```

Unused pad bits remain unrestricted. The actual server accepts aGl=, aGm= and
aGn= as the same two bytes as aGk=. Canonical-bit regexes would narrow acceptance.
Conversely, the actual JSON v2 decoder rejects CR/LF even though standalone
encoding/base64.StdEncoding accepts them. It also rejects missing/extra padding,
URL-alphabet substitutions, spaces, tabs and NUL. The grammar follows the JSON
codec, not an assumed generic base64 codec. Empty string decodes to zero bytes.
JSON escaping is removed before this string constraint and decoder operate.

#### Representation and other constraints

For JSON-carried bytes, type:string plus contentEncoding:base64 is the appropriate
3.1/3.2 metadata. `format:binary` currently appears on the generated JSON field,
but it does not mean base64 and does not change string-length semantics.
contentEncoding is an annotation; it cannot by itself enforce decoded lengths.
The reviewed #575 decision authorizes JSON-position metadata correction together
with constraint projection in the separate #574 fix. Enumerate the exact intended
output differences. Do not mechanically rewrite every Bytes format in the analyzer:
raw HTTP binary bodies, multipart parts, non-JSON parameters, documentation-only
media schemas, and custom codecs need their own actual representation context.
Preserve those unrelated targets' dereferenced schema constraints, metadata and
codec behavior, and exclude them from this built-in JSON theorem. The shared-name
projection rule below permits only the necessary reference/component-name split
when such an occurrence shares a declaration with corrected JSON bytes. Missing correspondence
evidence is not permission to introduce an unsupported-target rejection. Any new
rejection needs its own reproduction and separately approved design disposition.
This boundary must be explicit and tested; it cannot reject already supported
ordinary JSON bodies either.

Bytes DSL Pattern and Format are rejected as non-string validations; there is no
ordinary authored Bytes Pattern to overwrite. Semantic enum/default/example
values and their existing JSON projection must remain intact, conjoined with the
new length predicate. In particular, length projection must not alter enum
membership to hide a length failure. Canonical enum strings and the decoder's
noncanonical pad-bit aliases are distinct wire strings; an assertion of full
runtime/schema acceptance equivalence for Bytes enums needs its own evidence and
must not be inferred from the length theorem. Nullable null and missing optional
fields retain their existing outer handling; this string rule only governs the
present non-null Bytes branch. Explicit metadata/custom representation cannot be
silently overwritten: preserve its existing behavior and keep it outside this
built-in codec theorem unless explicit correspondence proves it compatible. A
new metadata-conflict diagnostic would need a separately authorized disposition.


This is the explicit schema correction for [#574](https://github.com/CaliLuke/loom/issues/574).
The generated server accepts the two-byte `hi` value while the old schema rejects
its four-character base64 string; the reverse mismatch occurs for a one-byte
value under MinLength(2). Candidate arithmetic and projection may be modeled
before production changes. Codec correspondence, emitted dialect validity and
actual generated-server behavior remain executable verification obligations,
not assumed consequences of a Lean theorem.

The rule follows [JSON Schema 2020-12 validation](https://json-schema.org/draft/2020-12/json-schema-validation)
and [OpenAPI binary-data representation](https://spec.openapis.org/oas/v3.2.0.html#working-with-binary-data):
string length applies to encoded characters; contentEncoding alone is an
annotation. One shared owner converts effective byte bounds into these JSON
constraints; inline-schema and OpenAPI projectors adapt that result rather than
implementing independent bound arithmetic.


#### Complete schemas and representation identity

A reusable schema component denotes a complete wire contract. Do not weaken it
into neutral structure and require every consumer to rediscover its constraints.
Derive each occurrence's actual encoding and shape plan before registering
components. The plan describes the codec owner, selected fields, wire names,
requiredness and visibility; media labels alone do not identify runtime codecs.
Schema caches and recursive graph identities include the semantic occurrence and
schema-relevant representation plan. An example-context string or traversal
order is not a representation identity. Request/response media entries with
incompatible plans cannot share one mutable schema or MediaType record.

The JSON-byte correction can make two occurrences of one logical declaration
require different complete schemas. Only for this representation conflict, the
public canonical name anchors the JSON projection when JSON is actually present.
With no JSON occurrence, keep the prior canonical component. Equivalent projected
schemas still share. Preserve excluded occurrences through equivalent complete
schema graphs: inline only when safe, and allocate deterministic derived variant
components where graph identity or recursion requires them. This is one
representation-aware analyzer consuming the shared target plan; no renderer pass
or second legacy normalization pipeline may repair its output afterward.

This is an explicit compatibility exception for #574: non-JSON operation references
and derived component paths may change in mixed-use graphs, and downstream SDK
type names can consequently change. Their dereferenced schema constraints,
metadata, codec behavior and Loom service/wire contracts remain equivalent.
Canonical JSON names and unaffected projection names/schema bytes remain stable.
Do not rename every component, silently drop constraints, or reject a previously
supported raw/custom occurrence merely because it shares a declaration with JSON.

Reserve explicit public names before allocating derived names. Allocation must be
stable when declaration, operation and media traversal orders are reversed.
Representation variants of one declaration have generated internal identities;
they do not independently claim the public canonical name. Existing rejection of
incompatible authored declarations or projected shapes claiming one explicit name
remains in force; this exception does not turn those errors into automatic
renaming. Documentation-only schemas remain independent occurrences under their
own declared media/codec ownership.

Proof obligations include mixed JSON/multipart/query use of named Bytes; nested
objects, arrays, maps and unions; recursive graphs; nullable occurrences;
explicit names that collide with a would-be derived name; reversed traversal;
and unchanged raw/custom contracts. Actual schema/decoder correspondence is
required separately from reference allocation. Both emitted-schema validation
and runtime decoder branch preservation must hold: canonical Bytes enum strings
alone cannot establish runtime uniqueness when a decoder also accepts alternate
pad-bit spellings. These are independent checks against one final wire value,
never a reason to rewrite an authored String or broaden an enum.

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
update. The byte grammar, length constraints and JSON metadata above are the
explicit #575 decision for #574; other schema differences remain outside that
authorization. A runtime/schema disagreement discovered by the proof matrix is a
Design Blocker: do not silently broaden acceptance or normalize it away.

Adopt the upstream branch-retention principle selectively. Copying current Goa
wholesale would discard Loom-specific contracts and still retain observed
upstream failures. Removing untagged support would break existing imported API
shapes. Adding another transport-specific normalization helper perpetuates the
root cause. A single universal wire format would also be wrong: JSON and
protobuf have different representations of the same semantic value.

## Proof obligations

### Lean scope and connection to production

Lean is required for the shared value semantics. For every finite built-in
value in the modeled domain, successful resolution/projection must produce a
well-typed target value whose modeled decoding preserves the target-observable
semantic value, including retained branch identity and target-specific presence
equivalence. Define `Observe(plan, value)` independently of the projector: it
selects fields and visibility from the plan, retains branch identities of unions
that remain visible, and applies the documented target presence equivalence.
For example, observing an HTTP body excludes its service payload's header
fields. Preservation compares decoding with this observation, not with the
entire service value. Legitimate field/visibility loss remains in the progress
theorem's domain; it cannot be excluded by a representability premise. The domain
includes scalars, bytes, objects, arrays, maps and nested unions. Recursive type
declarations may describe finite values; cyclic input graphs and incomplete
recursive examples have explicit failure outcomes. These are general theorems
over the domain, not enumeration of a bounded example set.

Typing, representability, decoding and semantic equivalence must be defined
independently of the projector. A success-soundness theorem alone is insufficient:
prove that a finite, complete, unambiguous, well-typed value representable under
the target plan succeeds. An implementation that always omits examples must not
satisfy the contract. Failure theorems distinguish invalid/ambiguous/incomplete
inputs, and source-selection theorems exclude replacement of authored values by
synthesis. Representability includes target ranges, naming uniqueness, visibility
and the unique wire match required by untagged unions.

The proved target is a structured JSON/protobuf value, not rendered Go source
or arbitrary serialized text. Target plans are inputs satisfying explicit
well-formedness conditions; deriving those plans from the evaluated DSL and
allocating real protobuf fields remains an implementation obligation. Numeric
formatting, external codecs, opaque Any/custom values, runtime libraries and the
Go compiler are outside the core theorem. Each boundary must be recorded rather
than hidden in a premise that assumes the conclusion.

V1 keeps the production resolver in Go and provides an executable Lean reference
for differential checking. A correspondence record maps each modeled constructor,
operation and outcome to its Go owner and executable contract tests. Tests compare
Go and Lean on the same independently encoded inputs and target plans, then check
actual emitted examples with real decoders/schema validators. Negative controls
must detect wrong bytes, branch changes, lost presence and incorrect omission.
The input adapter must preserve duplicate map entries, source identity and
absence/null distinctions so it cannot erase the bug before comparison.

This is a **proved semantic reference with tested implementation correspondence**.
V1 does not contain a machine-checked refinement proof of the Go implementation
or a verified Go renderer. Neither differential testing nor a passing build is
described as such a proof. Certifying every generated Go program for every
accepted design would require that additional verified generation pipeline and
is deferred beyond this repair; actual generated compilation remains mandatory.

Proof acceptance includes independent review of theorem statements, non-vacuous
preconditions and definitions, plus kernel checking and a transitive axiom audit.
No `sorry`, admitted obligations, custom correctness axioms or native-evaluation
trust may discharge the required theorems. Standard Lean logical axioms are
listed explicitly. The proof checks follow Lean's
[validation guidance](https://lean-lang.org/doc/reference/latest/ValidatingProofs/).

| Evidence | Required claim | Limit |
| --- | --- | --- |
| Lean | Universal soundness, progress and preservation for the defined finite semantic domain | Does not certify handwritten Go, external codecs or rendered source |
| TLA+ | Bounded exploration of ordering and cache/source/occurrence ownership | Not an unbounded type-preservation proof |
| Go/Lean differential tests | Production behavior agrees with the executable reference on exercised inputs | Tested correspondence, not universal refinement |
| Generated compilation and decoder/schema tests | Actual artifacts compile and preserve the tested contracts | Tested designs and environments only |
| Assumptions ledger | Explicit codec, toolchain, library and specification boundaries | No excluded boundary may be reported as proved |

### Required properties

1. Projection preserves the supplied target-observable values and retained branch,
   with the target-specific presence equivalence defined above, through
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
6. Lean proves the semantic obligations above; a bounded TLA+ model checks phase
   ordering and cache ownership. The correspondence record and assumptions ledger
   distinguish proved properties, bounded checks and implementation test evidence.
   Neither formal model replaces Go property tests and generated round trips.
7. Migration is complete only when the inventory has no built-in bypass that
   independently reselects branches or applies scalar coercion.
