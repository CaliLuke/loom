# OpenAPI 3.2 untagged `oneOf`: branches, locations and symmetry (#1014)

Status: delivered. This retained revision 5 contract covers the full ticket,
including text locations, map roots, form and multipart. The integration note
below accounts for #1013 landing after independent design review. The
[acceptance record](../internal/openapiimport/oneof_import_validation.md)
records the actual implementation parent, corrected owners, generated-output
comparisons and verification evidence. Historical source citations below refer
to `f0c7a2b8c`; they describe the design baseline, not current line locations.
The formal evidence below was checked with the pinned tools. It is design
evidence, not proof of the Go implementation.

## Summary

Before this change, import admitted an untagged `oneOf` only when every branch
was a named object or array component and the occurrence was a JSON request
body, a JSON success body or a component. The delivered contract makes two
decisions independently:

- which branch schemas are legal;
- where a union may appear.

Scalar, array, object, map, inline and nested branches are admitted at parameter,
header, media and error locations whenever their codecs support the shape and
the runtime can select exactly one branch. Exporter symmetry holds for every
admitted case.

This design separates three contracts previously fused in
`internal/openapiimport/plan.go`:

1. **Shape grammar.** Which branch schemas form a legal untagged union. The
   owner is `expr`. Import projects that grammar onto normalized schemas and
   must agree with it.
2. **Occurrence codec.** How one occurrence is read and written: JSON, text
   parameter codecs, form, multipart or documentation-only. The owner is the
   transport owner for that occurrence. Text codecs come from `#968`'s shared
   `internal/httpencoding` resolution, **per branch**.
3. **Exact-one selection.** For every codec, one shared rule with **two
   accounting domains**, as `MatchUntaggedJSON` already has:
   - The *schema domain*: each branch's codec parses the transport-native
     input into the JSON data model, and the projected schema validates it.
     This domain has no Go widths.
   - The *carrier domain*: the typed Go decode, followed by validation.
   - An input is accepted only when exactly one branch matches in the schema
     domain, exactly one matches in the carrier domain, and it is the same
     branch. Assignment happens only then.
   - When encoding, encode the selected branch, then run the same matcher
     over the output. Refuse output that is not attributed to the selected
     branch.

   Nothing is coerced into JSON, and no branch wins by order. A carrier
   overflow never selects a branch: text or JSON `2147483648` against
   `OneOf(Int, Int32)` stays ambiguous in the schema domain and is rejected.

OAS 3.2 settles the text case. §4.24.4.2 says implementations "MAY inspect
subschemas ... such as `oneOf` ... but MUST NOT attempt to resolve
ambiguities". Serializers with validated data "MUST inspect the data". Exact-one
selection is exactly that rule, so text locations are not deferred.

Representations remain honestly rejected only in these cases:

- the specification leaves them undefined (for example `deepObject` on a scalar,
  or a typeless form property, which defaults to `application/octet-stream`);
- another ticket owns the missing representation (null-only domains, general
  `allOf`/`anyOf`, `type` arrays and composite defaults). Each such case names
  its owner below.

## Problem Statement

What fails today:

- `TestUntaggedResponseOneOfKeepsUnsupportedShapesRejected`
  (`internal/openapiimport/issue_302_test.go:330-343`). `oneOf: [{type: string},
  {type: integer}]` fails with `schema-composition`, which does not mention
  unions.
- The parameter probe `TestUntaggedOneOfRejectsStringEncodedParameter`
  (`:505-529`, a form-exploded query object union) and the error probe
  (`:531-557`) fail with `schema-oneof-location`.
- The nested inline object probe (`:475-503`) fails with `schema-oneof-branch`.

Three diagnostic codes describe one fused decision. Separately, the planner's
branch rule (`plan.go:409-425`) requires `$ref` object or array components, and
the runtime root rule (`expr/untagged_validation.go:12-16`) requires a concrete
named object or array.

Exporter evidence was rendered from the current tree into scratch: the untagged
specimen exports `properties.v.oneOf: [$ref A, $ref B]` with an `example`
sibling and no `type`. Importing it succeeds today. The tagged and form
specimens are covered by #1005.

Consumer evidence:

- No checked-in OpenAPI document in `loom` or `loom-mcp` has a scalar,
  parameter, header, form or error-location `oneOf`.
- Real array/object evidence exists (`codegen/unionjson/README.md:21-31`).
- MCP `RequestId` is a scalar union, but it is spelled `type: [string, integer]`
  (#1000), not `oneOf`.
- Per the accepted rule, a valid contract class that cannot be expressed
  justifies the work. Consumers validate ergonomics; they do not gate it.

## Goals

- One shape grammar, owned by `expr` and used unchanged by import.
- One exact-one selection rule for every occurrence codec, in both directions.
- Branch-indexed text codecs shared with `#968`, with no union-specific style
  table.
- All roots the specification defines: primitive, named scalar alias, array,
  string-keyed map, object, nested untagged union and nullable branches. Inline
  branches keep their inline schema when that is representable.
- Parameter, header, cookie, path, response header, form, multipart, error and
  media locations.
- Exporter symmetry: import → render → evaluate → export → reimport gives the
  same normalized schema and the same codec, for every admitted case.

## Non-Goals (owned elsewhere)

| Construct | Owner | Behavior here |
|---|---|---|
| `type: [string, integer]` | #1000 | unchanged (`schema-type-union`) |
| `anyOf` beyond the nullable pair | #1019 | `schema-composition` |
| `allOf` with ≥2 members, and **non-redundant** assertion siblings beside `oneOf` | #1018 | `schema-composition`, naming the sibling |
| `unevaluatedProperties` other than `false` beside `oneOf` | #993 | `advanced-schema` |
| Null-only domains and typeless *constrained* branches (`{minimum: 0}`) | #1010 | `schema-oneof-branch`, naming #1010's class |
| Composite union defaults | #1006 | existing default rules |
| Content-based parameters other than JSON query | #970 | that owner's diagnostic |
| SSE and sequential `itemSchema` | #972 | union adds no gate; item codec owner decides |
| Named, positional and nested multipart encodings | #973 | that owner's diagnostic |
| `discriminator` | #1005 | consumes this design |

## Current State

| Concern | Owner today | Behavior |
| --- | --- | --- |
| Analyzer branch admission | `analyze_schema_composition.go:75-92` | ≥2 branches, object (inline or ref) or array (ref); any sibling (`hasDirectAllOfSchemaShape`, `:220-226`) → `unsupportedComposition` |
| Location gate | `plan.go:165-167, 195-197, 244-246, 282-284, 312-314` | Pre-walk `schemaContainsUntaggedOneOf` (`plan.go:318-352`) emits `schema-oneof-location` |
| Planner branch rule | `plan.go:409-425`, `schemaIsUntaggedShape` `:494-534` | `$ref` to object/array component; rejects maps, nested unions, `Bases` |
| Promotion | `analyze_schema_promotion.go:96-152` | Inline object branch → `<Owner>Variant` or `<Owner><FirstRequired>` |
| Rendering | `render_schema.go:98-111, 356-360` | `OneOf(a, b)` constructor plus `Untagged()`; inline object branch is an error |
| Runtime root rule | `expr/untagged_validation.go:12-20` | Concrete named object or array; branch may not admit null |
| Runtime leaf rule | `expr/attribute_validate.go:342-356` | Primitives (named scalar aliases included, because `IsPrimitive` sees through them), named objects/arrays/maps, untagged unions, arrays and string-keyed maps of these |
| Branch identity | `expr/union_branch_admission.go:8-27` | Unique logical name and tag. Constructor names come from `DerivedUnionVariantNames` (`expr/union_naming.go:26-44`), so `OneOf(String, String)` evaluates as `String`/`String2` (scratch probe) |
| Text locations | `expr/http_endpoint_mapping.go:128-133, 156-160, 173-177` | Unions forbidden in path, query, header, cookie |
| Form/multipart | `expr/http_endpoint_validate.go:449-457`; `expr/http_media_encoding.go:41` | Untagged unions forbidden |
| Text codecs | `internal/httpencoding/encoding.go:6-115` | `Select`/`Match` for named query and form fields; #968 proposes `Resolve` for every location |
| JSON matcher | `pkg/json_union.go:31-71`; `codegen/unionjson/union.go:116-125, 161-178` | Exact-one schema match plus exact-one decoder match, with the same identity. `Marshal` re-matches its own output (`matched != selected` is an error) |
| JSON shapes | `pkg/json_shape.go:18-35, 187-230` | `any`, scalar, `alias`, array, map, object and nested `union` kinds already exist |
| Errors | `expr/service.go:126-134`; `render_response.go:124-128` | Error types cannot be unions. Import wraps unconstrained and media-union errors and selects them with `Body(...)` |
| Exporter | `ir/analyzer.go:412-424`; `ir/analyzer_naming.go:36-50` | Untagged → `oneOf` of branch schemas, sorted by branch name |

## Integration with typed additional members

#1013 landed at `66765668` before this ticket's implementation parent. Its
`expr.ObjectOpenness` and physical-member APIs own object policy. The accepted
named-object grammar includes its JSON representations: the schema and carrier
domains must both validate typed undeclared members, including synthesized
required members. Lower the additional-member schema and its presence through
the prepared ValuePlan and JSONShape owners; do not infer the schema domain
from successful Go conversion. The selected branch evaluates every object
member when additionalProperties is present, exactly as the existing closure
model specifies. A missing policy evaluates only declared members.

For example, a branch with integer `x` and integer additional members admits
`{"x":1,"extra":2}` and rejects `{"x":1,"extra":"bad"}` in both domains.
Outer closure must not reject the admitted extra member. A closed source or a
transport-omitted member remains rejected according to the object owner's
captured policy. Text and form codecs continue to reject typed records at their
existing owner boundary; admitting their JSON shape does not authorize a new
wire codec. No model transition changes: the Additional closure profile already
models presence, while new prepared/lowered/generated tests must establish
correspondence for #1013's concrete carriers.

## HTTP error presentation

Generated JSON body decoders retain the established SafeDecodePayloadMessage
contract: invalid union bodies return HTTP 400 with code `decode_payload` and
detail `invalid request body`. The exact-one matcher and typed JSON codecs retain
the precise schema/decoder match counts for callers and diagnostic tests. The
behavior-table count messages below describe those codec errors; they do not
replace the HTTP body's existing safe error projection. Text binding errors
continue through the parameter codec owner's error contract. This preserves the
framework's existing public boundary while expanding union admission.

## Proposed Design

### 1. Shape grammar (owner: `expr`)

```text
root(branch):
    primitive: boolean | string | bytes | integer | number | Any      -- Any NEW at root
  | named scalar alias                                                -- NEW
  | array<leaf>, named or declared inline on the branch               -- inline NEW
  | map<string, leaf>, named or declared inline on the branch         -- NEW
  | named object (structured; open, closed, or typed additional members)
  | untagged union, named or nested anonymous                         -- NEW (nested composition)
  with optional branch-level null admission                           -- NEW
  rejected: tagged union, opaque codec / schema override, non-string map key,
            descriptor cycle without consuming a member or element (existing)
leaf(field | element | map value): existing leaf rule, unchanged
union: len(branches) >= 2, distinct identities (existing), no shape-based pair rule
```

- **Overlap is legal.** Ambiguity is decided per value by exact-one matching
  (`codegen/unionjson/README.md:248-250`). `oneOf: [A: string, B: string,
  C: integer]` imports: integers select `C`, and strings fail with "matched 2
  branches". `OneOf(Int, Int32)`, `OneOf(String, String)` (`String`/`String2`)
  and `OneOf(String, Any)` are legal designs.
  - `OneOf(String, Any)` accepts exactly the non-string values, which is the
    meaning of `oneOf: [{type: string}, {}]`.
  - Revision 2 rejected `{}` branches; that was a static overlap rule and is
    withdrawn.
- **Null is an ordinary value.** A branch admits null in any of these cases:
  - its occurrence declares `Nullable()` (from `type: [T, null]`);
  - its named type is nullable;
  - it is `Any` (`{}` and `true` admit null intrinsically);
  - it is a nested untagged union that admits null. That means its occurrence
    is nullable, or exactly one of its own alternatives admits null, counted
    recursively by these same rules.

  Exact-one counts null like any other value:
  - The JSON shape gate runs before the typed decoder
    (`codegen/unionjson/union.go:170-173`), so null reaches only a shape whose
    `Nullable` is set.
  - A nullable branch must use the existing null-preserving carrier; expr
    rejects a nullable branch without one. The carriers are:
    - explicit or inherited nullable occurrences: the existing `loom.Nullable[T]` carrier selected by NameScope;
    - `Any`: the existing JSON-value carrier, preserving JSON null;
    - a nested union: the carrier of its unique null-admitting alternative, or
      the nested occurrence's nullable carrier.
  - This keeps the reviewed invariant that null never collapses to a scalar
    zero value.
  - The current rule "branch cannot admit null" (`untagged_validation.go:17-19`)
    is replaced by this rule.
- **Tagged unions as untagged branches** stay rejected. A tagged branch would
  need a tagged JSON shape kind. Import never produces one: under an untagged
  branch, #1005 selects its full-object representation.
- **gRPC.** The protobuf projection keeps its existing admission. Any branch
  kind it cannot carry is a gRPC evaluation error; nothing is dropped
  silently. Imported designs declare no gRPC transport.

### 2. Occurrence codecs (owner: the transport owner of the occurrence)

```text
codec(occurrence):
  JSON body | JSON component | JSON-RPC params/result        -> json
  parameter/header with content: application/json           -> json, where the content owner resolves one
                                                                (named query today; #970 for other locations)
  parameter (path, query, header, cookie) | response header  -> text: Resolve per branch (#968, section 4)
  form-urlencoded root                                       -> form-root: per-branch FormRoot object/map codec
  form or multipart field (Encoding Object)                  -> by contentType/style (section 5)
  multipart root                                             -> multipart-root: per-branch object decoding
  raw non-JSON body (existing documentation-only mode)       -> documentation: OpenAPIBody, schema-only grammar
  SSE/sequential item                                        -> item codec owner (#972)
```

Rules:

- `schemaContainsUntaggedOneOf` and all five `schema-oneof-location` emitters
  are deleted.
- A location rejects a union only when its codec owner cannot resolve every
  branch. The diagnostic names the branch index and the owner's rule.
- Branch legality never depends on location. Location legality never depends
  on branch shape, except through the codec owner's own per-shape table.

### 3. Exact-one selection (one contract, two accounting domains)

**Owner.** The counting core moves to a new, dependency-neutral package,
`internal/unionmatch`. It imports only the standard library. Both public entry
points delegate to it:

- `pkg` (`loom.MatchUntaggedJSON`, unchanged signature);
- `http` (`loomhttp.MatchUntaggedText`, new).

This is cycle-free: `pkg` imports only internal leaf packages (`exactnum`,
`identifier`, `jsondecode`, `jsonkey`), and `http` already imports `pkg`, so
both can import `internal/unionmatch`. A private `pkg` function could not be
called from `http`, and exporting it would add public API for an internal
contract.

`unionmatch` also owns the union-contract mismatch error. That error is the
existing `jsonUnionMismatch`, moved out of `pkg`. `DecodeJSONCandidate` and the
text adapters therefore classify nested-union mismatches through one type,
next to `internal/jsondecode.IsMismatch` for conversion mismatches. Generated
code keeps calling only the public entry points, because it cannot import
`internal`.

```go
// internal/unionmatch
type Candidate struct {
    Schema func() (bool, error) // schema domain: codec grammar + projected shape, no Go widths
    Decode func() (bool, error) // carrier domain: typed decode + validation into a private value
}
func Select(name string, candidates []Candidate) (int, error)
type Mismatch struct{ /* message */ } // ordinary contract mismatch; anything else is fatal

// pkg: JSONUnionBranch{Schema *JSONShape, Decode} adapts to Candidate (unchanged behavior)
func MatchUntaggedJSON(name string, data []byte, branches []JSONUnionBranch) (int, error)
// http: one candidate per #968 BranchCodec (section 4)
func MatchUntaggedText(name string, branches []TextUnionBranch) (int, error)
```

`Select` keeps the two counters that `MatchUntaggedJSON` has today
(`pkg/json_union.go:41-70`):

```text
Select(candidates):
  for each candidate i, in declaration order, without stopping early:
      s_i := Schema_i()          // fatal error -> return it
      d_i := Decode_i()          // fatal error -> return it
  if count(s) != 1 -> Mismatch "matched N branches in schema"
  if count(d) != 1 -> Mismatch "matched N branches in decoder"
  if index(s) != index(d) -> Mismatch "schema and decoder identities differ"
  return the index; the caller assigns only now
```

Rules:

- A carrier overflow can never select a branch. Consider `2147483648` against
  `OneOf(Int, Int32)`. The schema domain matches both branches, and only `Int`
  decodes. The value is rejected, in JSON (unchanged behavior) and in text
  (`UntaggedText`, `TextCarrierOnly` control).
- An input that no branch decodes is rejected. The destination is unchanged in
  every state: it is never written during candidate evaluation.
- `Select` visits every candidate, so declaration order cannot hide a failure.
  The model checks two branch orders; the existing order-reversal tests move
  with the core.
- Fatal callback errors (evaluator or configuration failures) propagate
  unchanged and are never counted as mismatches. This classification is
  outside the model and is the existing contract.

Encoding rule:

```text
encode(u):
  wire := Encode_{u.kind}(u.value)                    // branch codec
  i    := Select(candidates over wire)                // same two domains, local view
  if i != u.kind -> encoding error before the request is built or the response commits
  publish wire
```

This is today's JSON `Marshal` check (`union.go:119-124`) extended to text:

- Client encoders run the check before publication.
- Server response-header encoders run it before commit.
- The generated CLI parses flag text for a text-union binding with the same
  matcher. A query flag is one raw parameter value when every branch owns only
  one occurrence of the binding's literal key; for example `hello&peer=1` is scalar data and
  `a%2Cb,c` retains the escaped comma within a list item. If any branch owns
  repeated sequence keys, flattened object keys, a deepObject namespace or
  residual map keys (including inside a nested alternative), the flag
  is a complete query fragment, such as `data=x`, `value[data]=x` or `x=1`.
  Cookie flags use the same single-value versus complete-fragment distinction,
  with semicolon-separated cookie pairs such as `x=1; y=2` for a map or
  `value=a; value=b` for an exploded sequence. This grammar is fixed by the
  resolved binding, never inferred from flag text.

### 4. Text-location unions (consuming #968's `internal/httpencoding`)

#968 owns every text-codec decision: style and explode resolution, shapes
(`Scalar`, `Sequence`, `Object`, `ScalarMap`, nested values), escaping and
delimiters, the decoding grammar, the representable domain, Rule A, key
namespaces (`KeySet`: `Literals`, `Prefixes`, `Residual`) and
`CheckNamespaces`. This design states none of those rules again. It consumes
#968's resolved outputs:

| #968 output | How the union uses it |
|---|---|
| `ResolveUnion(loc, name, branches, src)` | style and explode resolved **once** from the binding's serialization, then each branch's `Codec` and `KeySet`, plus the union's combined `KeySet`; any branch error rejects the union, naming the branch and #968's reason |
| union `KeySet` in `CheckNamespaces` | the union is one binding: collisions with sibling bindings are design-time errors; several `Residual` alternatives inside one union are one binding's claim |
| per-branch schema membership on the raw text | `Candidate.Schema` |
| per-branch typed decode and validation | `Candidate.Decode`, called for **every** branch so both domains are counted independently |
| per-branch encoder with #968's domain rules | the encode rule in section 3 |
| key-presence rule | union presence: any key of the union's `KeySet` present (always for path; header field present). Absence applies requiredness and default before matching |

Rules owned by the union:

- **Branch not sent (parameter context).** Inside a shared parameter
  namespace, a branch whose codec finds none of its own keys is a mismatch,
  never an empty value (`UntaggedPresence` `PresenceChecked`; the
  empty-is-value candidate fails). Root bodies follow section 5.
- **Alternatives are not siblings.** A `Residual` or `Prefixes` alternative
  reads the wire keys of the other alternatives in the same union. For
  example, `?id=5` is read by both a scalar `id` branch and an exploded map
  branch, so it is ambiguous and is rejected by counting, not by ownership.
- **No coercion.** Text is parsed as JSON only by a branch whose #968 codec is
  `JSON` (named query `content: application/json`).

Examples, all query unless noted. Outcomes follow from #968's outputs plus the
rules above.

| Source branches and serialization | Wire or value | Outcome |
|---|---|---|
| `string`, `integer`; `form` | `id=abc` | string |
| `string`, `integer`; `form` | value `"42"` or `42` | encoder error; a peer's `id=42` gets 400 "matched 2 branches in schema" |
| pattern `^[a-z]+$` string, `integer`; `form` | `id=42` | integer |
| `integer`, `array<integer>`; `form`, explode false | `id=4,5` | array; `[4]` and `4` are encoder errors |
| `{data!}`, `{error!}` objects; `form` exploded (the existing probe) | `data=x` | first branch |
| scalar `id`, all-optional object `{x, y}`; `form` exploded | value `{}` | no keys: optional → absence (Rule A), required → encoder error |
| `integer`, `ScalarMap` string→integer; `form` exploded, sibling `page` | `a=1&page=2` | map `{a:1}`; `page` stays with the sibling; map `{page:1}` and `{id:1}` are encoder errors (sibling key; ambiguous with the integer branch) |
| object `{x}`, `ScalarMap` string→integer; `deepObject` | `f[x]=1` | ambiguous: both decode `{x:1}`; `f[y]=1` → map |
| `integer`, `ScalarMap`; `deepObject` | — | **rejected**: `deepObject` is defined only for objects (§4.12.6), so the scalar branch fails `ResolveUnion` |
| `integer` (int64), `integer` `format: int32`; `form` | `id=2147483648` | 400 "matched 2 branches in schema"; only `Int` decodes, but the schema domain is ambiguous |
| path `oneOf: [integer, array<integer>]`; `label` exploded | `/.4.5` | array; an element containing `.` is an encoder error (#968 element domain) |
| cookie `oneOf: [string, ScalarMap]`; `cookie` exploded | `Cookie: theme=dark` | map; a sibling session cookie stays with its binding |

### 5. Form and multipart

**Root-body presence belongs to the media owner,** not to key presence. A root
body is one value. The request decoder (`http` request decoding with
`OptionalRequestBody`) decides whether it is present, before any branch runs:

| Body | Required | Optional |
|---|---|---|
| form, zero length | present: the empty value set | absent (the existing "form decoders accept an empty form" rule, made explicit) |
| form, non-empty | present | present |
| multipart, no body | missing (400) | absent |
| multipart envelope with zero parts | present: the empty part set | present: the empty part set |

When the body is present, every branch decodes the **whole** value set, empty
included, and then validates. The parameter "branch not sent" rule does not
apply. An empty set is a legal value of every branch that accepts it: a closed
empty object (`additionalProperties: false`, no properties) or an all-optional
object.

So a present empty form body uniquely selects closed-empty `B` beside an `A`
that requires `x`. The one coincidence is owned by the media owner: an optional
zero-length form body is absence. A union value whose branch encodes to the
empty set is therefore sent as absence for an optional form body, which is
Rule A at the media owner. Multipart can spell it as an empty envelope. Both
contexts are checked by `UntaggedPresence` (`BodyCheckedEmptyValue`,
`BodyCheckedEmptyAbsent`). The candidate that applies the parameter rule to
bodies fails (`BodyParamRule`).

- **Form root** (`application/x-www-form-urlencoded`). Every branch must be an
  object or map (§4.15.3: form encoding serializes named properties). A
  non-object root branch is undefined and is rejected naming the branch. Each
  branch decodes the present body through #968's raw form carrier. Parse the
  body once under the existing size limit, retaining raw pair spelling and
  independent presence for every branch. A field codec splits its literal
  delimiters before percent-decoding; `url.Values` alone cannot distinguish
  `n=a%2Cb,c` from delimiter data. Keep the FormRoot decoder's undeclared-key
  policy for each branch's openness. Schema and carrier domains are counted
  as in section 3; body presence still belongs to this media owner.
- **Multipart root.** Admitted when every branch passes the existing multipart
  object admission. The parsed `multipart.Form` is read once, under the
  existing body limit, and decoded per branch. Any other multipart shape keeps
  that owner's diagnostic (#973).
- **Authored body identity.** An object remains an object even when it has
  exactly one union field. That field keeps its name and Encoding Object.
  Flat tagged forms (including OAuth grants) must use an actual root union,
  such as `Type("Grant", OneOf(func() { ... }))`. The previous shape-based
  flattening silently changed authored object contracts and is removed;
  synthetic wrapper methods and their metadata are deleted. Regenerate after
  expressing flat forms as root unions; service calls use the generated union
  directly instead of an enclosing one-field struct. The bounded
  `http/codegen/tla/form_body_identity` model demonstrates the old collision.
- **Encoding application and codec are independent.** This corrects the earlier
  whole-array JSON default and multipart content/style-conflict claims against
  [OAS 3.2 §4.14.5.1 and §4.15.1](https://spec.openapis.org/oas/v3.2.0.html#encoding-object).
  Named array properties apply content encoding to each item; other properties
  apply it to the whole value. Item arrays remain whole JSON values. Presence of
  any authored `style`, `explode`, or `allowReserved` selects the parameter-style
  path and suppresses implicit and explicit contentType in both media.
- **Capture source facts before normalization.** Preserve the effective source
  type after `$ref`/`allOf` inspection; do not infer an absent outer type from
  `oneOf` alternatives. In particular, `type: array` plus array alternatives
  applies content to items, while a typeless union of those same alternatives
  with explicit JSON applies content to the whole union value. These remain
  different representations after their normalized value schemas coincide.
- **Shared representation.** The existing `BranchShape` graph retains
  `SourceType` and an `Items` edge, including non-scalar and nullable items.
  Empty `SourceType` means a native design whose concrete shape supplies the
  default; `absent` explicitly records missing source type. Other values are
  `array`, `object`, `string`, `integer`, `number`, or `boolean`. Imported
  attribute-local `Meta("http:encoding:sourceType", value)` preserves a source
  fact when native inference after normalization would differ. It follows
  occurrence-first aliases but is not inherited from a root union into its
  alternatives. Authored Encoding Objects retain their separate existing slots
  and continue to propagate occurrence-first through media branches.
- **Resolved application.** The shared codec records `Value` or `Items` beside
  its existing style/kind. In content mode the field's source type chooses this
  once; `Items` removes exactly one array level. Each candidate branch resolves
  the content/default for its item schema without applying another outer-item
  expansion. Defaults therefore belong to the encoded value's source type,
  including distinct item types in array alternatives. Style mode uses its
  existing shape-dependent parameter grammar. The superseded
  `DefaultMediaType` and `http:encoding:<field>:defaultContentType` representation
  is removed, rather than retaining two competing provenance systems.

  | Source field and Encoding Object | Application and wire values |
  | --- | --- |
  | string, no encoding | Value, text |
  | object, no encoding | Value, one JSON object |
  | array of strings, no encoding | Items, repeated text |
  | array of objects, no encoding | Items, repeated JSON objects |
  | array of arrays, no encoding | Items, repeated JSON arrays |
  | array of strings, explicit JSON | Items, repeated JSON strings |
  | `type: array` plus string-array/integer-array oneOf | Items; each branch parses the repeated text independently; `1` is ambiguous, `word` selects strings |
  | `type: array` plus string-array/object-array oneOf | Items; each branch retains its item codec, text or JSON respectively |
  | typeless array-alternative oneOf, explicit JSON | Value, one complete JSON array |
  | typeless oneOf, no encoding | Octet default cannot represent a typed union; existing declare-contentType-or-style diagnostic |
  | array of typeless union items, explicit JSON | Items, each JSON item passes its nested exact-one matcher |
  | JSON contentType plus explicit `allowReserved: false` | Style mode; contentType is ignored |

- **Import, runtime and export share these facts.** Import admission uses the
  same shape/application resolver as evaluated DSL. Rendered DSL must retain
  source application through generation and round trip. Export restores
  justified redundant source types beside normalized unions and preserves
  branch-local item schemas. It must not force one Encoding Object label onto
  branches with different implicit codecs. Both incoming branch membership and
  outgoing consistency use the captured application, before typed Go-carrier
  conversion. A successful match of one repeated item does not select the whole
  array branch; the entire candidate array must pass its schema.
- `expr` lifts its blanket untagged form/multipart prohibitions. It admits a
  union exactly when this contract resolves every branch. This specification
  correction changes implementation obligations, not the accepted ticket scope.

### 6. Normalization (owner: import analysis, before promotion)

**Single branch.** A `oneOf` with one branch and only annotation siblings
normalizes to that branch.

**Null.** Normalization runs bottom-up: nested unions first, then the outer
union. Let `b` be the number of bare `{type: null}` branches without
annotations. Each other branch has a null kind, computed per section 1:

- `excluded`: it does not admit null.
- `occurrence`: its null can be removed without changing its non-null domain.
  That is an inline `type: [T, null]`, or a nested union whose only null comes
  from a removable inner null or occurrence flag.
- `declared`: it admits null that cannot be removed here. That is a nullable
  component, `Any`, or a nested union whose unique null-admitting alternative
  is declared.

Let `k` be the number of branches whose kind is not `excluded`. Branch null
admission is kept as authored, and bare null branches are removed by this rule:

```text
b = 0                        -> keep branches; null counted per branch
b = 1, k = 0                 -> occurrence Nullable()          (revision 2 rule, unchanged)
b >= 2, k = 0                -> occurrence not nullable (null matched b >= 2 branches: rejected)
k >= 2                       -> keep branches; null still matches >= 2 branches: rejected
b >= 1, k = 1, occurrence    -> remove that branch's removable null (recursively in a nested
                                union); null rejected
b >= 1, k = 1, declared      -> reject: the kept branch must stay nullable while null must be
                                rejected; needs a null-only branch (#1010). `oneOf: [{}, string,
                                null]` is this case: `{}` cannot drop null without `not` (#1020)
no ordinary branch           -> null-only domain, reject (#1010)
annotated or constrained null branch -> reject: no slot keeps its annotation (#1010 owns
                                        constrained null-only domains)
```

Evidence:

- **Flat rule.** `normalize_sound` proves it for arbitrary branch lists, given
  the kinds.
- **Classification.** Kinds, including `Any` and nested unions, are checked
  over bounded nested trees by `OneOfNull.tla`. Both rejected classifiers fail
  there:
  - leaf flags only: inner `oneOf: [string, {}, null]` accepts null twice;
  - nested occurrence flag only: the inner nested union's local null is missed.

**Outer siblings.** These rules are shared with #1005.

```text
annotation siblings: title, description, examples/example, deprecated,
                     readOnly, writeOnly, x-*            -> union occurrence annotation slots
type: T          redundant iff every branch's type set ⊆ T (integer ⊆ number)     -> dropped
required: [p..]  redundant iff every branch is an object requiring each p         -> dropped
properties.p: S  redundant iff S has no annotations and every branch either
                 declares p with a schema whose assertions include S's with
                 equal values, or is closed and does not declare p               -> dropped
unevaluatedProperties: false -> union occurrence closure (below)
default          -> reject on a surviving union; branch defaults keep existing rules
anything else    -> schema-composition naming the sibling (#1018)
```

"Dropped" means provably redundant. Each dropped fact is implied by every
branch, so the accepted domain is unchanged. No authored annotation is
discarded.

Whole-union defaults have no representable `Default` contract, even when exactly
one branch accepts the value. For example, a sibling `default: "chosen"` on
`oneOf: [{type: string}, {type: integer}]` is rejected; a `default: "chosen"`
inside the string branch retains that branch's existing scalar-default contract.
Moving the outer default into a branch would change authored scope and is not a
valid normalization. Supporting outer defaults would require an explicit union
default representation and absence semantics across generators; this ticket
retains the constructor boundary described in §7. Composite defaults remain
owned by #1006.


**Closure** (`unevaluatedProperties: false` beside `oneOf`). This follows JSON
Schema 2020-12 Core §11.3 (`unevaluatedProperties`), which runs after the
in-place applicators, and only annotations from the passing `oneOf` branch
count. The closure is applied **after** counting:

```text
closed_match(v):
  i := match(original branches, v)            // section 3, branches unmodified
  if v is an object: require members(v) ⊆ evaluated(branch_i, v)
evaluated(object, v) = members(v)             if additionalProperties is present (true, schema or false)
                     = members(v) ∩ declared  if additionalProperties is absent
evaluated(map, v)    = members(v)
evaluated(Any, v)    = ∅                      // `{}` produces no annotations
evaluated(union, v)  = evaluated(selected inner alternative, v)
evaluated($ref, v)   = evaluated(target, v)
```

Rules:

- Closing branches *before* counting is wrong. With `A` an open object
  requiring `x` and `B` an open object with optional `y`, the value `{x: 1}`
  matches both, so the source rejects it. Closing `B` first would wrongly
  accept `A` (`UntaggedClosure` `CloseFirstOpenPair`).
- Declared field-name lists are wrong for maps, nested unions and objects with
  `additionalProperties`; three controls fail.
- Runtime: `JSONShape` matching returns the evaluated member set of the
  selected branch, and the union occurrence checks it. This applies to every
  codec that yields an object value: JSON, form root, multipart root and text
  object codecs.
- Tagged envelopes evaluate exactly `{typeKey, valueKey}`. The generated tagged
  decoder enforces the closure when the occurrence carries it.
- The evaluated set depends on whether `additionalProperties` is absent or
  explicitly present. Preserve this fact through the object owner and prepared
  JSON shape, including #1013 typed additional members.

Occurrence identity is resolved after service and transport finalization.
A local closure selects a private copy of the union's alias chain, with a
distinct reserved name and declaration ID; repeated occurrences of the same
closed declaration share that variant. Raw-union hashes include the policy.
The branch occurrences are already finalized and remain shared: closing an
outer union does not close its branches or recursive uses of the original type.
A declaration's own closure continues to govern every reference to that
declaration without requiring a second declaration. Existing names and IDs are
reserved before variant allocation, so an authored `ChoiceClosed` cannot be
overwritten by a generated variant of `Choice`. The opt-in model under
`expr/tla/union_closure` rejects both shared mutation and policy-erasing identity;
direct tests cover name collisions and copies outside its assumptions.

### 7. Inline branches, branch declaration and naming

Inline primitives with constraints, inline arrays, inline maps and nested
inline unions are rendered **inline**. They use a branch-declaration
constructor that works wherever a type is accepted:

```go
// NEW constructor form (shared with #1005): branch attributes, any position.
Result(OneOf(func() {
    Untagged()
    Attribute("String", String, func() { Pattern("^[a-z]+$") })
    Attribute("Array", ArrayOf(Int))
    Attribute("Map", MapOf(String, Int))
    Attribute("OneOf", OneOf(func() { Untagged(); Attribute("Int", Int); Attribute("Boolean", Boolean) }))
}))
```

The constructor returns a `DataType`, so its root accepts only branch
declarations and state represented by `expr.Union`: `Untagged`, the three
`oneof:typename`/`oneof:type:field`/`oneof:value:field` keys, and union closure.
The temporary attribute is checked before construction returns; other metadata
and attribute-level nullability, annotations, examples, validation or composition
are errors rather than silently discarded state. Branch attribute DSLs remain
complete. For example, `Type("Choice", OneOf(func() { ... }), Nullable)` is
accepted, while `OneOf(func() { Nullable(); ... })` is rejected. Enclosing
`Attribute`, `Payload` and `Result` use the same placement. Whole-union defaults
remain rejected by their existing contract; branch defaults are unchanged.

This deliberately keeps the existing data-type return contract. The alternative
of returning an attribute-bearing wrapper would introduce a new type carrier
through every consumer merely to retain settings already represented by enclosing
attributes. Silently dropping them is not an acceptable alternative. The
constructor-boundary rejection preserves author intent through an actionable
diagnostic and keeps the complete ticket scope available through normal DSL
placement. `TestOneOfConstructorRejectsDiscardedDirectives`,
`TestOneOfConstructorRetainsSupportedSettings` and
`TestOneOfConstructorOccurrenceDirectives` cover the boundary and accepted forms.

Naming:

- A `$ref` branch uses the existing constructor-derived name
  (`Title(UnionVariantPublicName)`).
- An inline primitive is named by its Goified Loom type (`String`, `Int64`,
  `Int32`, `Float64`, `Boolean`, `Bytes`, `Any`).
- An inline array, map or union is named `Array`, `Map` or `OneOf`.
- Collisions use the existing `UniqueStableNames` suffixes (`String2`).
- When every branch is a `$ref` or a bare primitive, the existing
  `OneOf(A, B)` constructor form is kept, byte for byte.

Generated Go member names have a separate shared allocation step. Storage fields
must not collide with fixed union methods or any branch's `As*`/`Set*` methods.
Distinct logical names that sanitize to the same Go identifier receive stable
numeric suffixes for accessors and constructors. Allocation reserves authored
sibling names before assigning suffixes, so a branch named `Kind` does not steal
an authored `Kind2` field. All declaration, transform, validation and transport
generators consume that allocation. Wire names and discriminator values retain
their authored spelling. Consumers regenerate and use the generated accessors;
no compatibility aliases preserve previously uncompilable declarations.

Package-level allocation is a separate shared contract. Before a local type is
referenced, its package owner reserves the primary name together with validators,
union companions and other emitted declarations. Service, view and HTTP packages
each supply their actual inventory; declarations and references consume the same
allocated identity. View metadata maps and physical viewed-result envelopes own
distinct roles. Logical view names retain their authored strings even when their
Go helper suffixes need disambiguation. Consumers regenerate and use the new Go
names; the correction does not change branch matching or wire values. The
[acceptance record](../internal/openapiimport/oneof_import_validation.md#package-declaration-ownership-correction)
records the bounded source, compilation and independent-process byte evidence.

Rendering inline adds no synthetic component. That matters for #1005: every
component name is an implicit discriminator key. This is proved by
`added_component_name_becomes_implicit_hint`.

Inline **object** branches and populated inline object fields under a union
branch are still promoted. The runtime needs named objects. The traversal
carries an "under untagged branch" flag through fields, array items and map
values:

- Branch names are `<Owner>Variant` or `<Owner><FirstRequired>`; fields use
  `<Owner><GoifiedField>`.
- Suffixes are allocated by the existing allocator, depth first, inner before
  outer.
- Structurally equal inline objects are not deduplicated.
- Outside union branches, the empty-record-only rule is unchanged.
- The promotion is silent, following the branch precedent. The existing lossy
  diagnostic for promoted inline array items is unchanged.

### 8. Error locations

Loom forbids union error types. A JSON error body `oneOf` imports as a concrete
envelope. Its single required attribute holds the union, and `Body("body")`
selects it, through the existing wrapping path (`render_response.go:124-128`).
The wire stays the bare branch.

```go
var ImportedGetItemBadRequest = Type("GetItemBadRequest", func() {
    Attribute("body", OneOf(ImportedFirst, ImportedSecond), func() { Untagged() })
    Required("body")
})
// Error("GetItemBadRequest", ImportedGetItemBadRequest); Response("GetItemBadRequest", 400, func() { Body("body") })
```

- Error response headers with unions use the text codec.
- Shared-status names keep `disambiguateSharedErrorStatuses` (`plan.go:104-119`).
- Response selection before body decoding belongs to #976. A union body failing
  exact-one is a decode failure of the selected response. It never falls back
  to another response.

### 9. Exporter symmetry (owner: `http/codegen/openapi/internal/ir`)

| Construct | Export |
|---|---|
| Union occurrence | `oneOf` of branch schemas; inline branch attributes render inline; named branches `$ref`; no discriminator |
| Nullable branch | branch schema admits null (3.1/3.2 type array or nullable wrapper) |
| Occurrence nullability | existing nullable wrapper |
| Text-location union | the parameter or header `schema` is the `oneOf`. `style`/`explode` come from the same `#968` codec and are emitted only when they differ from the location default |
| Form/multipart union field | preserve captured source application/type and item schemas; emit authored encoding or an equivalent resolved encoding only when it does not change branch-local defaults |
| Union closure | `unevaluatedProperties: false` on the union schema |
| Inline JSON Schema (`expr/json_schema_inline.go`) | the same closure contract: an untagged union occurrence carrying the closure emits `unevaluatedProperties: false` beside its `oneOf`, inside the nullable wrapper when the occurrence or its named type is nullable. A closed `OneOf(Any, String)` therefore rejects `{"x": 1}` in the inline schema, as it does at runtime |
| Error envelope | the selected body exports the bare `oneOf` |

The analyzer treats retained source type as structural component identity, not
as a reference annotation. A source-absent singleton keeps its normalized type
assertion inside a singleton `oneOf`; deleting the assertion would widen JSON
membership, and an `allOf` would expose the type to source-type inspection again.
Concrete source types on normalized unions are redundant assertions. Expression
validation checks their non-null domain, including effective finite enum and
exact numeric point refinements. Null acceptance still follows original exact-one
branch counting, before source assertions are restored.

Encoding export walks the captured form or multipart branch tree and groups by
physical field name. It tests authored encoding candidates against every captured
leaf through the shared resolver; defaults remain implicit when one explicit label
would change a branch codec. Ordinary homogeneous JSON form defaults may retain an
explicit equivalent JSON label. Explicit style flags suppress contentType; a lone
`allowReserved: false` also emits `style: form` so omission of false by serializers
does not reactivate content encoding. Multipart labels remain exact because they
are actual part headers. If no one Encoding Object describes all occurrences of
a physical name, export reports the conflicting field instead of selecting a
branch's contract. For example, one root branch requiring JSON string parts and
another requiring plain text parts for the same name cannot share a single media
Encoding Object. This is an OpenAPI representation limit, not reduced runtime
union admission; distinct implicit item defaults remain representable in schemas.

Branch order is not semantic in JSON Schema. The exporter keeps its name sort
(`analyzer_naming.go:36-50`). Equivalence compares branch multisets.

## API / Contract Changes

1. `Untagged()` admits the roots in section 1, including nullable branches,
   `Any`, scalar aliases, maps and nested unions. Update the `Untagged`
   GoDoc (`dsl/attribute_oneof.go:58-77`).
2. New constructor form `OneOf(func() { ...Attribute... })`, valid wherever a
   type is accepted and copied by `namedUnionCopy`. #1005 uses the same form.
3. HTTP mappings admit untagged unions in path, query, header, cookie, response
   header, form and multipart positions, exactly when section 2 resolves every
   branch.
4. Text locations consume #968's `ResolveUnion`, `KeySet`, `CheckNamespaces`
   and per-branch schema, decode and encode operations (section 4).
5. New runtime `http.MatchUntaggedText`. It shares the
   `internal/unionmatch` counting core and mismatch type with
   `loom.MatchUntaggedJSON`.
6. Import diagnostics:
   - `schema-oneof-branch` for shape;
   - the codec owner's code (`parameter-shape`, `parameter-serialization`,
     `media-type`) for occurrences;
   - `schema-composition` for routed siblings.
   - `schema-oneof-location` is removed.
7. Accepted examples:

```go
var Strings = Type("Strings", ArrayOf(String))
var Lookup  = Type("Lookup", OneOf(String, Strings), Untagged)                 // string or list
var Range   = Type("Range", MapOf(String, Int))
var Filter  = Type("Filter", OneOf(Range, Document), Untagged)                 // map root
var Wide    = Type("Wide", OneOf(Int, Int32), Untagged)                        // every integer ambiguous
Payload(func() { Attribute("id", OneOf(Int, Slug), func() { Untagged() }) })  // Slug: pattern alias
HTTP(func() { GET("/items/{id}") })                                            // path union
// Rejected: OneOf(Tagged, Doc), Untagged (tagged branch); MapOf(Int, X) branch (non-string key)
```

## Detailed Behavior

Invariant: an untagged union occurrence is admitted exactly when its branches
satisfy the grammar and the occurrence codec owner resolves every branch.
Admitted values are exactly those with one schema-domain match and one
carrier-domain match on the same branch, followed by closure when present.
Encoded output is exactly what that same matcher attributes to the selected
branch.

| Source (JSON success body unless noted) | Outcome | Owner / reason |
| --- | --- | --- |
| `oneOf: [string, integer]` | `OneOf(String, Int)` | grammar |
| `oneOf: [string, {type: array, items: string}]` | branch form; inline `Array` | section 7 |
| `oneOf: [{type: string, const: a}, {type: string, const: b}]` | branch form; inline enums | section 7 |
| `oneOf: [integer, number]` | accepted; integral values ambiguous | overlap |
| `oneOf: [{$ref: A}, {$ref: B}, integer]`, A and B bare strings | accepted; integers select the third | overlap counterexample |
| `oneOf: [string, {}]` | `OneOf(String, Any)`; accepts non-strings | grammar |
| `oneOf: [{type: [string, null]}, integer]` | nullable `String` branch; null selects it | null rule `b = 0` |
| `oneOf: [string, null]` | `String` + `Nullable()` | `b = 1, k = 0` |
| `oneOf: [A, null, null]` | `A`, not nullable | `b ≥ 2` |
| `oneOf: [{type: [string, null]}, integer, null]` | `OneOf(String, Int)`, not nullable | `k = 1` local |
| `oneOf: [$ref N (nullable), integer, null]` | reject → #1010 | `k = 1` declared |
| `oneOf: [{}, string]` | `OneOf(Any, String)`; null selects `Any` | `Any` admits null, `b = 0` |
| `oneOf: [{}, string, null]` | reject → #1010/#1020 | `k = 1`, `Any` is declared |
| `oneOf: [oneOf: [string, {type: [string, null]}], integer, null]` | inner local null removed; null rejected | nested `occurrence` kind |
| `oneOf: [integer (int64), {type: integer, format: int32}]`, value `2147483648` | accepted design; that value is 400 in the schema domain | section 3 |
| `oneOf: [A, {type: null, description: x}]` | reject → #1010 | annotated null |
| `oneOf: [$ref A]` | `ImportedA` | single branch |
| `oneOf: [$ref A, $ref StringMap]` | accepted (map root) | grammar |
| `oneOf: [A, oneOf: [B, C]]` | nested inline union branch | grammar; not flattened, because flattening changes `A∧B∧C`-style counts |
| `A.data` populated inline object | promoted `AData` | section 7 |
| `type: object` beside object branches | sibling dropped | redundant |
| `type: object` beside `[string, A]` | reject `schema-composition` → #1018 | not redundant |
| `minProperties` beside `oneOf` | reject → #1018 | assertion sibling |
| Error 400 body `oneOf: [A, B]` | envelope + `Body("body")` | section 8 |
| Query `oneOf: [{data!}, {error!}]` (existing probe) | accepted, form-exploded | section 4 |
| Query `deepObject` `oneOf: [integer, {x}]` | reject `parameter-serialization`, branch 0 | #968 table |
| Query `deepObject` `oneOf: [{x}, map string→integer]` | accepted; `f[x]=1` ambiguous, `f[y]=1` map | #968 `ScalarMap` + `Prefixes` |
| Query `deepObject` `oneOf: [integer, map string→integer]` | reject `parameter-serialization`, branch 0 | #968: `deepObject` is objects only |
| Query form-exploded map branch beside another dynamic-key binding | reject at design time | dynamic claim rule |
| Query map branch with object values | reject: nested object serialization undefined | §4.12.6 |
| Header `oneOf: [integer, array<integer>]` | accepted | simple codec |
| Cookie `oneOf: [string, integer]` | accepted | #968 cookie codec |
| Form field `oneOf`, no encoding | reject: default `application/octet-stream` | §4.15.1.1 |
| Form field `oneOf`, `contentType: application/json` | JSON matcher | section 5 |
| Form root `oneOf: [Cat, Dog]` objects | per-branch FormRoot | section 5 |
| Form root `oneOf: [string, Cat]` | reject: non-object form root branch | §4.15.3 |
| Required form root `oneOf: [A{x!}, B closed empty]`, zero-length body | `B` | body presence, section 5 |
| Optional form root, same branches, zero-length body | absence; `B` sent as absence | media owner coincidence |
| Multipart root, same branches, envelope with zero parts | `B` | body presence |
| Form field with sibling `type: object` beside `oneOf`, no encoding | JSON codec captured, then `type` dropped; exported with explicit `contentType` | section 5 capture |
| `unevaluatedProperties: false` beside `[A{x!} open, B{y?} open]`, value `{x: 1}` | 400: both branches match before closure | section 6 closure |
| Raw `text/plain` response `oneOf: [string, integer]` | `OpenAPIBody(..., Untagged)` | documentation mode |
| Component union referencing itself as a direct branch | reject | descriptor cycle |

Runtime outcomes:

```text
"x"  vs OneOf(String, Int)                 -> String
1    vs OneOf(Int, Float64)                -> 400 "matched 2 branches in schema"
null vs OneOf(String|null, Int)            -> String branch, null value
null vs OneOf(String, Int) not nullable    -> 400 "matched 0 branches"
"x"  vs OneOf(String, Any)                 -> 400 "matched 2 branches"; true -> Any
2147483648 vs OneOf(Int, Int32)            -> 400 "matched 2 branches in schema" (JSON and text)
query id=42 vs OneOf(String, Int)          -> 400 "matched 2 branches"
encode Int(1) in OneOf(Int, Float64)       -> encoding error, no partial body
encode String("42") as query in OneOf(String, Int) -> encoding error before the request
```

Projection surfaces:

| Surface | Behavior |
| --- | --- |
| Service types | existing sum type; branch fields hold primitive, alias, slice, map, nested union or nullable carrier |
| HTTP JSON | `MatchUntaggedJSON` adapters; error envelope uses the same adapter |
| HTTP text/form/multipart | generated `TextUnionBranch` adapters over #968 codecs; client and response encoders run the re-decode check |
| CLI | text-union flags decode with the same matcher; JSON bodies unchanged |
| JSON-RPC | service JSON codecs; text unions do not occur |
| gRPC | existing protobuf oneof projection; unsupported kinds rejected at evaluation |
| OpenAPI | section 9 |
| Inline JSON Schema | section 9 row; same closure contract as OpenAPI and runtime |
| Import diagnostics | section 2 and the codes in API change 6 |

## Contract Matrix

| Capability | Supported | Rejected | Deferred | Default | Source of truth | Placement | Routing | Naming | Equivalence | Proof |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| Root branches | primitives, `Any`, scalar aliases, arrays, string maps, objects, nested untagged unions, nullable branches | tagged branch, opaque codec, non-string key, unconsumed cycle | none | tagged unless `Untagged()` | `expr` grammar | existing union types | matcher per codec | constructor names; branch form names (section 7) | import admits ⇔ design evaluates | grammar agreement corpus |
| Overlap | any overlap; decided per value | none statically | none | n/a | runtime exact-one | adapters | shared core | n/a | — | `UntaggedText` (ambiguous text never accepted or sent) |
| Null | per-branch null incl. `Any` and nested unions; bare-null normalization | declared null (component, `Any`, nested) beside bare null; annotated/constrained null; null-only (#1010) | none | not nullable | import normalization, bottom-up | branch or occurrence | shape gate before decode | n/a | normalized domain equal | `normalize_sound` (Lean), `OneOfNull` (TLA) |
| Closure | `unevaluatedProperties: false` beside `oneOf`, every object-yielding codec | other values (#993) | none | open | normalization → union occurrence | metadata | count first, then evaluated annotations | n/a | regenerated identically | `UntaggedClosure` |
| Inline branches | inline scalar/array/map/union in branch form; inline objects promoted | none new | dedupe of equal inline objects | n/a | promotion traversal | `components.schemas` only for objects | reference plan | section 7 | regenerated schema equal modulo inline-object `$ref` | deterministic isolated renders |
| Outer siblings | annotations; redundant `type`/`required`/`properties`; closure | non-redundant assertions (#1018) | none | n/a | normalization | union occurrence | n/a | n/a | domain unchanged | redundancy table tests |
| JSON locations | request, success, error (envelope), component, media branch, JSON form field and part, content-JSON query | none new | none | n/a | `httpcodec` | existing | `MatchUntaggedJSON` | existing | bare branch on the wire | error envelope round trip |
| Text locations | path, query, header, cookie, response header; scalar, sequence, flat-object and scalar-valued map branches | branches #968 cannot resolve; a second dynamic-key binding | none | location default style | `httpencoding.ResolveUnion` | `http` runtime helpers | `MatchUntaggedText`, two domains | parameter name; literal, prefix and dynamic key sets | client text ⇔ server selection | `UntaggedText`, `UntaggedPresence` (parameter) |
| Form/multipart | form root objects/maps with media-owner body presence; fields per Encoding Object; multipart root objects | typeless field without encoding; non-object form root branch | named, positional and nested multipart encodings (#973) | per §4.15.1.1, captured from the source before normalization | `httpencoding` + media owner | form/multipart templates | text or JSON matcher | field names | explicit encoding exported | `UntaggedPresence` (body); form round trip; octet-stream rejection |
| Documentation bodies | schema-only grammar | none | none | n/a | `OpenAPIBody` | docs | none | n/a | regenerated `oneOf` | evaluates |
| Exporter | section 9 | none new | none | n/a | evaluated design | IR | n/a | sorted branch names | order-insensitive branch multiset | symmetry fixture on the 3.2.0 and 3.1.1 targets |

## Acceptance Coverage

| Issue expectation | Covered by | Residual |
|---|---|---|
| Branch selection and composition independent of occurrence | sections 1-3; diagnostics split | none |
| Admit legal branch schemas when the runtime can select exactly one | section 1 (all roots, `Any`, null, overlap), section 6 | declared null (component, `Any`, nested) beside a bare null, and annotated null (#1010); typeless constrained branches (#1010); `type` arrays (#1000) |
| Ambiguity | shared core; text re-decode check | none |
| No match | `matched 0 branches` on every codec | none |
| Scalar, array, object (and map) | section 1 | none |
| Parameter locations | section 4, including `ScalarMap` branches | style/shape cells the specification leaves undefined (#968's `n/a` cells, nested values) |
| Media locations | sections 2 and 5, with body presence at the media owner | multipart encodings (#973), item schemas (#972) |
| Error locations | section 8 | none |
| Exporter symmetry | section 9; proof obligation 7 | the inline-object promotion changes the spelling to `$ref` only |
| Minimal schema `oneOf: [string, integer]` | grammar row 1 | none |
| Parameter/header `oneOf` | section 4 table | none |
| Named object branches with nested union or map leaves | existing runtime leaf rule now projected by import | none |

## Alternatives and Tradeoffs

- **Store source defaults as authored contentType slots.** Rejected. The value schema and its item application are lost, and a single captured
  label cannot represent branch-specific defaults. Source type and item edges
  remain explicit until the shared resolver selects application and codec.
- **Materialize one root label or clone schema graphs to remove defaults.**
  Rejected. One field name can resolve to repeated text in one alternative and
  binary octets in another; a common label changes a branch's wire contract.
  Specialized union declarations add identity and naming churn. Source
  type facts preserve both contracts without either transformation.
- **Defer text locations (revision 2).** Withdrawn. §4.24.4.2 defines the
  required behavior, and the per-branch codec reuses #968 without new style
  rules.
- **Coerce text to JSON, then use the JSON matcher.** Rejected. Text `5` is a
  valid string, and coercion makes it an integer. `TextCoerce` shows the server
  then accepts text that two branches decode.
- **First successful branch.** Rejected. `TextFirstMatch` shows it resolves an
  ambiguity the specification forbids resolving.
- **Server-side check only.** Rejected. `TextNoRoundtrip` shows a client sending
  `"5"` as a string, which every server rejects.
- **Static disjointness or pair rules.** Rejected. They would discard the legal
  third branch in `[string, string, integer]`.
- **Promote constrained scalars and inline arrays (revision 2).** Replaced by
  the branch-declaration form. Promotion adds components, and every component
  name is an implicit discriminator key.
- **Inject null into another branch** to represent `[N nullable, I, null]`.
  It is domain equal (`null_injection_alternative_is_domain_equal`), but it
  rewrites an authored non-null branch. Rejected in favor of #1010's null-only
  branch.
- **Lift the union-error prohibition.** The envelope reaches the same wire
  through existing owners.

Risks:

- Same-kind overlaps (for example `string`/`integer` in a query) make many
  values unsendable. Generated errors name the matched count, and the HTTP
  guide must state the rule.
- Removing the location pre-walk moves diagnostic paths. `--skip-unrenderable`
  must still see an operation-level blocker.

## Proof Obligations

Formal tools stay opt-in design aids. They are never gates.

1. **Grammar agreement.** For every normalized import schema that renders,
   import admits iff the rendered design evaluates. The bounded corpus covers
   every row of the behavior table.
2. **Exact-one on every codec.** A JSON or text input is accepted iff exactly
   one branch matches in the schema domain, exactly one in the carrier domain,
   and both are the same branch. The destination is unchanged in every state
   until acceptance, and no-match input is rejected. Model: `UntaggedText`, with
   four checked profiles including widths and reversed order. Controls: first
   match, coercion, eager assignment, carrier-only counting. Witnesses: zero
   match (`?`) and width split (`big`).
3. **Encode attribution.** Every published value is attributed by the same
   matcher to the selected branch, with the same value. Model: `UntaggedText`
   (`Agreement`, `NoAmbiguousSend`); control: no round-trip check.
4. **Presence.** Parameter context:
   - a branch with none of its keys is not sent;
   - optional empty name-erasing values travel as absence, and required ones
     fail before publication.

   Body context:
   - presence is body presence, so a present empty body is a value;
   - an optional zero-length form body is absence;
   - an empty multipart envelope is a value.

   Model: `UntaggedPresence`, both contexts. Controls: empty-is-value, no Rule
   A, and the parameter rule applied to bodies.
5. **Null normalization soundness.** For every source, an admitted
   representation accepts exactly the source domain.
   - The flat rule is proved for arbitrary branch lists by `normalize_sound`.
   - The kinds of `Any` and nested unions are checked bottom-up over bounded
     trees by `OneOfNull`, which also shows that every rejection is a declared
     null beside a bare null.
   - Reproduced counterexamples: the naive collapse, dropping bare nulls,
     `Any` treated as excluded, and nested unions classified by their flag only.
5a. **Closure.** Under `unevaluatedProperties: false`, acceptance equals JSON
   Schema 2020-12: exact-one over the original branches, then members ⊆ the
   selected branch's evaluated annotations. Model: `UntaggedClosure`, six
   profiles. Controls: close-first (three counterexamples) and name lists
   (three counterexamples).
5b. **Codec capture.** The media codec of every form or multipart field is
   resolved from the source schema before redundant siblings are dropped.
   Regeneration emits it explicitly, and reimport resolves the same codec.
6. **No null-to-zero collapse.** The generated decoder never assigns a
   non-nullable carrier from JSON null. This is a direct and generated test
   obligation.
7. **Exporter symmetry.** Each accepted row, imported, compiled and re-exported,
   reimports to the same normalized schema (order-insensitive branches) and
   resolves the same codec. The 3.2.0 and 3.1.1 export targets (there is no
   3.0 export) validate with libopenapi and the pinned Vacuum rules. 3.0.x
   sources are importer inputs only.
8. **Owner-routed diagnostics.** Each rejected row yields one diagnostic at the
   occurrence path, from the named owner. Partial import still blocks the
   operation.
9. **Determinism.** Promoted names, branch names and rendered designs are
   identical across isolated generation processes.
10. **Transport controls.** JSON-RPC and protobuf fixtures with new branch kinds
    keep envelope framing and field identity, or are rejected at evaluation.

## Formal Evidence (design-time, opt-in)

Toolchains:

- Java 25.0.5 (Corretto) with TLC 1.7.4 (`tla2tools`, sha256
  `936a2620…0e88`);
- Lean 4.35.0.

Commands, assumptions and limits live in each model README.

| Model | Configuration | Result |
|---|---|---|
| `pkg/tla/untagged_json/UntaggedText.tla` | `TextChecked` (string/int/list) | pass, 114 states |
| | `TextCheckedPattern` (pattern string; `?` matches none) | pass, 92 states |
| | `TextCheckedWidths` (`Int`, `Int32`, pattern) | pass, 70 states |
| | `TextCheckedReversed` (reversed order) | pass, 114 states |
| | `TextFirstMatch` | `ServerSound` violated: `5` selected the first of three |
| | `TextNoRoundtrip` | `NoAmbiguousSend` violated: client sent string `"5"` |
| | `TextCoerce` | `ServerSound` violated: coerced `5` accepted as integer |
| | `TextEager` | `Transactional` violated: destination written mid-loop (input `a`) |
| | `TextCarrierOnly` | `ServerSound` violated: `big` accepted as `Int` although the schema domain matched two |
| | `TextZeroWitness` / `TextWidthWitness` | expected witness violations: `?` and `big` reached |
| `pkg/tla/untagged_json/UntaggedPresence.tla` | `PresenceChecked` (parameter) | pass, 162 states |
| | `BodyCheckedEmptyValue` / `BodyCheckedEmptyAbsent` | pass, 55 states each |
| | `PresenceEmptyIsValue` | `NoFalseRejection` violated: `A` unsendable |
| | `PresenceNoRuleA` | `NoFalseRejection` violated: optional empty object refused |
| | `BodyParamRule` | `NoFalseRejection` violated: closed-empty body branch unsendable |
| `pkg/tla/untagged_json/UntaggedClosure.tla` | six `ClosureChecked*` profiles | pass, 68 states each |
| | `ClosureCloseFirstOpenPair` | `Agreement` violated: `{x}` accepted as `A` |
| | `ClosureCloseFirstDiscriminated` | `Agreement` violated: `{kind, x}` accepted |
| | `ClosureCloseFirstAny` | `Agreement` violated: scalar rejected after closing `Any` |
| | `ClosureNamesListMap` / `Nested` / `Additional` | `Agreement` violated: map, nested and `additionalProperties` members wrongly closed or opened |
| `internal/openapiimport/tla/oneof_null/OneOfNull.tla` | `checked` | pass, 126 144 states (31 536 trees), `NullAgreement` and `RejectionIsNecessary` |
| | `flag-only` | `NullAgreement` violated: inner `oneOf: [string, {}, null]` |
| | `shallow` | `NullAgreement` violated: nested union's inner local null missed |
| `internal/openapiimport/lean/schema_normalization/OneOfNormalization.lean` | 16 theorems | built; `normalize_sound` uses `propext`, `Classical.choice`, `Quot.sound`; no `sorryAx` |

Design refinements from the models:

- The client re-decode check is required. Without it, a sent value can be
  rejected by the server.
- A branch with none of its keys present must report a mismatch inside a
  parameter namespace, and must not in a root body.
- Text matching needs the separate schema and carrier domains.
- Closure must be applied after counting, over evaluated annotations.
- `Any` and nested unions take part in null counting.
- The `k = 1` declared-null case is not representable by the kept family (theorem
  `declared_null_beside_bare_null_has_no_kept_representation`).

## Settled Decisions and Cross-Owner Alignment

The user's full-scope instruction authorizes these. They are settled design
decisions, each with its evidence:

- **Branch-declaration constructor** `OneOf(func(){...})`, valid at every type
  position. Inline constrained scalar, array, map and nested-union branches
  stay inline, with no synthetic components. A synthetic component name would
  become an implicit discriminator key; `added_component_name_becomes_implicit_hint`
  proves this. #1005 uses the same constructor.
- **Redundant-sibling elimination** (section 6). Only facts implied by every
  branch are dropped, so the domain is unchanged. Media codecs are captured
  before the drop. This admits the exporter's own tagged output and the
  common outer `required: [petType]` pattern without waiting for #1018.

Cross-owner alignment, coordinated by the parent; no public policy is open
here:

1. **#968.** Its revision publishes `ResolveUnion`, style resolved once per
   union, `ScalarMap` codecs, `KeySet` claims and the two-domain per-branch
   contract. Three points must match section 4:
   - the per-branch typed decode is callable for every branch, so the carrier
     domain is counted, not called only for the unique schema member;
   - several `Residual` or `Prefixes` alternatives inside one union form one
     binding's claim;
   - inside one union, alternatives read each other's keys, and counting
     resolves them; `CheckNamespaces` applies between bindings only.
2. **Shared core.** `internal/unionmatch` replaces the private counting in
   `pkg/json_union.go`. That is an internal refactor with unchanged JSON
   behavior.

## Review History

Revision 2 corrections retained:

- Duplicate-shape branches still admit values through a third branch.
- Null never collapses to a scalar zero value.
- The collapse that accepted null beside a nullable residual is never applied
  (`naive_collapse_counterexample`). A local residual null is now removed, so
  null stays rejected. Annotated null stays rejected.

Revision 5 corrections (focused re-review):

- Section 4 consumes #968's resolved outputs instead of restating its tables.
  The stale "exploded `label` and cookie objects" exceptions are gone (#968
  defines those variants).
- The invalid example "scalar plus `ScalarMap` under `deepObject` accepted" is
  now a rejection: the scalar branch is undefined under `deepObject`.
- The counting core moves to the dependency-neutral `internal/unionmatch`, so
  `http` can share it without a private `pkg` call.
- Settled decisions replace the "needs acceptance" list.

Revision 4 corrections (independent review):

- Closure is applied after counting the original branches, over JSON Schema
  evaluated annotations. Closing first changed exact-one.
- Root-body presence is the media owner's. Present empty bodies are values.
- Both JSON and text matching keep separate schema and carrier accounting, so
  carrier overflow never selects a branch.
- Null counting includes intrinsic `Any` and recursive nested unions.
- Scalar-valued maps are specification-defined object values in text
  locations, with dynamic key claims that never take sibling keys.
- Form and multipart codecs are captured before redundant `type` removal and
  exported explicitly.
- Export targets are 3.2.0 and 3.1.1; 3.0 is only an import source.
- The model inputs now include zero-match text, and destination preservation
  is checked in every state.

Revision 3 corrections:

- `OneOf(String, String)` is not rejected by branch identity. Constructor names
  are `String`/`String2`, verified by a scratch probe.
- `{}` branches are admitted; their rejection was a static overlap rule.
- Map roots, text locations and form/multipart are designed, not deferred.
- Inline constrained scalars and arrays render inline rather than as promoted
  components.
