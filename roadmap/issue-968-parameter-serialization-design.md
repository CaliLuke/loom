# OpenAPI parameter serialization styles and structured values (#968)

Design revision 5, full ticket scope. **Delivered by the #968 atomic change.**
Bytes text policy is settled below.
Revision 4 passed independent review.
Revision 5 applies the reviewer's resolutions of three design gaps found during
execution planning (DB-1 to DB-3 under Review Status) and records the
`expr.HTTPValueShape` placement. It is not an execution plan and does not
authorize implementation. Source citations are against `f0c7a2b8c`
(`v1.10.0-alpha.24`), newer than the `98131b0` revision cited in the issue.
Specification citations are to the OpenAPI 3.2.0 HTML at spec.openapis.org,
read 2026-11-05. Go behavior is from the toolchain pinned in `go.mod`
(`go 1.27.3`).

## Summary

Loom binds HTTP parameters with one codec per location: text scalars, repeated
or comma-list scalar arrays, and JSON named query maps. OpenAPI 3.2 defines
eight styles across four parameter locations, and the Encoding Object applies
the query style table to `application/x-www-form-urlencoded` properties. This
design makes `internal/httpencoding` the single owner of three decisions:

- the codec a binding uses (`Resolve`);
- the wire key namespace it occupies (`KeySet`);
- the per-branch codecs of a text-serialized union (`ResolveUnion`, consumed
  by #1014's two-domain matcher).

It admits every style, explode, location and value combination that OpenAPI
3.2 defines: label exploded collections, every cookie array and object form,
comma-bearing data in `form` lists, explicitly styled form fields, and
scalar-valued string-keyed maps (`ScalarMap`) under every style. It
rejects only:

- combinations the specification marks `n/a`;
- the cookie form Appendix D.1 calls wrong;
- values nested beyond one level, which no style can express;
- individual values whose spelling cannot round-trip.

Import, generated client and server code, and OpenAPI export derive from the
same resolution.

## Implementation discoveries

Validation is recorded beside the owner in
[parameter codec acceptance](../internal/httpencoding/parameter_codecs_validation.md).
The original baseline descriptions and line citations below are historical.
The implemented shape adapter retains existing primitive map-key carriers as
well as string keys, formatting and sorting their wire names through the same
native adapter. New styles still require scalar map values.

Bytes use raw text consistently in text parameters, with the selected location's
escaping and representability checks. This preserves byte values without an
implicit base64 conversion and keeps path, query, header, cookie and form
adapters consistent. JSON values retain their base64 contract. The user delegated
this choice to framework judgment; base64 text parameters were considered and
rejected because they would impose a different value spelling on every existing
text binding. For example, Bytes containing `hello` send `hello`, not `aGVsbG8=`;
a byte sequence outside a location's admitted alphabet is rejected rather than
silently recoded. Binary-schema importer behavior is unchanged.

Additional intended differences discovered by tests are required zero-key and
path representability preflight, retirement of text Any bindings, normalized
omission of redundant imported form metadata, and an HTTP-backed JSON-RPC
request envelope when its payload lives entirely in transport metadata.
Named and unnamed root collections use the same one-binding admission; response
constructors take their source from decoded response bindings. Form codec
metadata remains attached to each media variant, with callbacks scoped to the
current encode/decode operation.

## Problem Statement

Admission keeps a bounded subset and rejects valid documents with fatal
`parameter-shape` or `parameter-serialization` diagnostics (Current State).

Consumer evidence:

- The issue's `coverage.json` is synthetic.
- Stripe's published `spec3.json` (OpenAPI 3.0.0, fetched 2026-11-05) declares
  366 query parameters with `style: deepObject, explode: true`:
  - 21 typed `object` with scalar properties. Admitted here.
  - 63 untyped `anyOf`, for example `created` as a `{gt,gte,lt,lte}` range or
    an integer. These are unions; #1014 consumes `ResolveUnion`.
  - 282 typed `array` (`expand`). These are an `n/a` cell of §4.12.6 and stay
    rejected.
- No checked-in Loom fixture or exported design uses `matrix`, `label`,
  `spaceDelimited` or `pipeDelimited`. Those styles are a capability gap.
- The one request cookie in a real fixture, `Cookie("locale:locale")`
  (`http/integration_tests/fixtures/quality/design/design.go:89`), carries
  unreserved locale tags.

Verified defect. The exporter copies `openapi:style` into any parameter without
validation (`http/codegen/openapi/internal/ir/operation_metadata.go:213-221`),
and `validateURLFieldEncoding` checks only query and form fields
(`http/codegen/openapi/internal/ir/url_field_encoding.go:13-45`). A path
parameter with `Meta("openapi:style", "matrix")` exports `style: matrix` while
the client sends `simple`. The resolution model's `legacy` configuration
reproduces this class on a cookie.

## Goals

- One owner maps a binding to exactly one codec and one key namespace, or to a
  rejection that names the rule. Its inputs are the location, the value shape
  and the authored serialization.
- Generated encoders and decoders, response header codecs, OpenAPI export and
  import consume that owner. There are no transport-local style tables.
- Every defined OpenAPI 3.2 parameter and form-field serialization is
  supported, with independent client and server wire examples.
- Exact escaping, empty-value and presence domains per codec. Encoders reject
  only values that the specified decoding cannot carry.
- A published per-branch contract for #1014's text unions, and form-field
  styles for #1005.
- Updated import coverage ledger and HTTP guide.

## Non-Goals

These are outside the issue or undefined by the specification. None is a
deferral of an issue field.

- Nested values under any style, including object or array properties of a
  `deepObject`. RFC 6570 values are strings, lists or associative arrays of
  strings. OAS 3.2 §4.12.6 marks `deepObject` arrays `n/a`, and §4.12.7 says
  nested query formats are not supported.
- Partial-segment path templates such as `/report.{format}`. Loom wildcards are
  whole segments (`expr/http.go:45`, `http/mux.go:24-30`).
- `in: querystring` beyond the existing `MapParams()` export.
- Runtime `allowReserved`. It stays documentation metadata
  (`http/codegen/testdata/openapi_32_dsls.go:121-134`). This is
  decode-equivalent: Loom clients percent-encode reserved characters, which
  Appendix E calls always safe for decoding, and servers decode once.
- Whole-body and media presence. Present-empty form bodies and empty multipart
  envelopes belong to the body/media owner and #1014. Rule A below covers
  parameter and form-field key presence only.
- JSON-RPC and gRPC wire changes. JSON-RPC HTTP inherits these codecs through
  shared templates.
- OpenAPI 3.0 export. Loom emits 3.2.0 by default and 3.1.1 when
  `openapi:version` selects it (`http/codegen/openapi/v3/version.go:42,44`).
  Import continues to read 3.0, 3.1 and 3.2 documents.

## Current State

- Owner: `httpencoding.Select`/`Match` know `NamedQuery`, `FormField`,
  `WholeQuery`, `FormRoot` and modes `Text`, `Repeated`, `JSON`, `Root`
  (`internal/httpencoding/encoding.go:6-115`). `Match` rejects explicit
  style, explode or allowReserved on form-field composites (`:84-115`). Path,
  header and cookie are not locations
  (`http/codegen/internal/transportir/parameters_security.go:67-70`).
- Import admission:
  - query through `Select`;
  - path and header: primitive or primitive array;
  - cookie: primitive only
    (`internal/openapiimport/parameter_admission.go:11-34`,
    `url_serialization.go:38-50`).
  - Non-query styles other than `simple`/`form`/`cookie`, and any non-query
    `explode`, are rejected (`analyze.go:394-396`,
    `analyze_parameters.go:3-14`).
  - A `oneOf` has `urlShape` `Object` (`url_serialization.go:120-129`).
- DSL admission: path primitive or array; query also map; header primitive or
  array; cookie primitive (`expr/http_endpoint_mapping.go:103-182`).
- Form-field Encoding Objects: `expr` and import call `Match` with the
  authored `contentType`, `style`, `explode` and `allowReserved`
  (`expr/http_media_encoding.go:47-66`, `internal/openapiimport/url_serialization.go:14-35`).
  For named query parameters, a content media type combined with any style
  field is rejected, and a JSON-encoded query map is checked as content
  (`encoding.go:84-115`, `http/codegen/openapi/internal/ir/url_field_encoding.go:13-25`).
  So today `allowReserved` alone on a metadata-free query map is rejected at
  OpenAPI generation.
- Runtime:
  - Path builders escape every string path value with
    `http.EscapePathSegment`. That is `url.PathEscape`, except that a whole
    value `.` or `..` becomes `%2E` or `%2E%2E`, which the mux decodes back.
    Catch-all remainders use `EscapePathRemainder`, which applies the same rule
    to every `/`-separated segment (`http/path.go:9-35`, `http/codegen/paths.go:80-115`).
    Regressions cover both (`http/path_test.go:25-26,47-48,71-72`,
    `http/codegen/path_escaping_integration_test.go:132-133`).
  - Path arrays split the raw capture before unescaping
    (`http/codegen/paths.go:129-137`, `http/path.go:37-49`).
  - Header lists use `internal/headerlist`. It rejects empty, comma-bearing and
    whitespace-edged items, and drops empty members on decode per RFC 9110
    (`internal/headerlist/list.go:10-53`).
  - Query arrays repeat the key and are read from `url.Values`. Maps use JSON
    (`http/json_field.go:15-50`).
  - Cookies go through `http.AddRequestCookie`, which calls `Cookie.Valid` and
    then `req.AddCookie` (`http/request_cookie.go:12-19`). net/http's
    `AddCookie` wraps any value containing a space or comma in double quotes
    (`net/http/cookie.go:509-515`). Its value check admits bytes 0x20-0x7E
    except `"` `;` `\` (`cookie.go:517-519`).
- Ledger: "Fixed object query parameters remain unsupported". A named query
  map requires `content: application/json`; a map with `deepObject`, flattened
  `form` or an implicit style is a blocking mismatch
  (`docs/openapi-import-coverage.md:141-147`, `docs/http-guide.md:171`).
- Export: the 3.1.1 filter strips `style: cookie` and so describes raw cookies
  as `form`. It also strips non-query `allowReserved`
  (`http/codegen/openapi/v3/openapi31_filter.go:169-187`). Parameters never
  emit `explode`.
- Formal: `internal/httpencoding/tla/serialization` covers named query and form
  fields only; `http/tla/path_decoding` covers comma path arrays.

## Proposed Design

### 1. Shared owner and exact interface

`internal/httpencoding` replaces `Mode` with a resolved codec and adds key
namespaces and union resolution. Shape and `KeySet` names match the reviewed
#1014 revision 5, which consumes this API unchanged.

```go
type Location uint8 // NamedQuery, FormField, WholeQuery, FormRoot, Path, Header, Cookie
type Shape uint8    // Scalar, Sequence, Object, ScalarMap, Nested, Union, UnsupportedShape

// BranchShape describes any bound value (a whole binding or one union branch).
// It is produced by expr.HTTPValueShape over evaluated expr types and by the
// importer's urlShape over normalized schemas.
type BranchShape struct {
    Shape      Shape
    Properties []PropertyFact // Object only, in declaration order
    Keys       KeyDomain      // ScalarMap only: the key attribute's domain
    Domain     ValueDomain    // Scalar value, Sequence element or ScalarMap value domain
    Branches   []BranchShape  // Union only, in branch order
}
type PropertyFact struct {
    WireName string
    Required bool
    Kind     ScalarKind // Boolean, Integer, Number, String, Bytes
    Domain   ValueDomain // used only by the 3.1.1 cookie projection (section 7)
}
type Style uint8 // Simple, Matrix, Label, Form, SpaceDelimited, PipeDelimited, DeepObject, Cookie, JSON, Root
type Codec struct {
    Style   Style
    Explode bool  // resolved; false for DeepObject, JSON and Root
    Shape   Shape
}

// KeySet is the namespace a binding occupies: query keys, form-body field
// names or cookie names. Path and header bindings occupy only their own name.
type KeySet struct {
    Literals []string // exact keys: "color", "R", "filter[gt]"
    Prefixes []string // deepObject ScalarMap names: each claims every "name[...]"
    Residual bool     // exploded ScalarMap or MapParams: every key no sibling claims
}
type BranchCodec struct {
    Codec Codec
    Keys  KeySet
    Inner []BranchCodec // nested union branches
}

func Resolve(loc Location, name string, v BranchShape, src Serialization) (Codec, KeySet, error)
func ResolveUnion(loc Location, name string, branches []BranchShape, src Serialization) ([]BranchCodec, KeySet, error)
func CheckNamespaces(loc Location, bindings []Binding) error
```

`Binding` retains a diagnostic name and a `KeySet` for each independent claim.
Equal diagnostic names never merge claims; generated namespace ownership also
uses binding identity rather than those labels.

`ResolveUnion` takes the binding name because literal keys contain it;
#1014 revision 5 uses this signature.

Implementation refinement: `Resolve` replaces both `Select` and the old
parameter `Match`; multipart admission keeps its separate `MatchMultipart`
entry point. `Serialization` retains explicit-presence pointers.

`Domain`, `Keys`, and each property's `WireName` and `Domain` supply the
value and key facts for the 3.1.1 cookie projection (section 7). Both adapters
populate them; the resolved style and explode select the projection rule. `KeyDomain` and
`ValueDomain` distinguish values that are unreserved-only (integers, booleans,
`date` strings, and enums whose literals are all RFC 3986
unreserved) from every other value. UUID format alone does not establish that
domain: validation also accepts URN and brace spellings. An explicit enum of
unreserved UUID spellings can establish it. They are input facts, not policy.

Adapter placement. Expr validation calls `Resolve`, and `transportir` imports
`expr`, so the expr-type adapter lives in `expr` as
`expr.HTTPValueShape(*AttributeExpr) httpencoding.BranchShape`. It replaces
`transportir.URLShape` (`http/codegen/internal/transportir/field_encoding.go:9-23`)
and `expr.httpMediaFieldShape` (`expr/http_media_encoding.go:74-88`).
`transportir`, OpenAPI and the HTTP generators call it.

| Shape | Values |
|---|---|
| `Scalar` | Boolean, Integer, Number, String, Bytes (raw text); non-nullable, not `Any` |
| `Sequence` | array of `Scalar` |
| `Object` | fixed object whose properties are all `Scalar` |
| `ScalarMap` | string-keyed map whose values are `Scalar` |
| `Nested` | any map or object with array, object, map or union members |
| `Union` | `oneOf` branches; resolved by `ResolveUnion` |

The two shape adapters share no code. Their agreement is a proof obligation.
Physical carriers are unchanged: a `Object` keeps its struct, a `Sequence`
its slice, a `ScalarMap` its `map[string]T`. Encoders iterate map entries
in bytewise key order, so wire output is deterministic.

### 2. Resolution rules

OpenAPI defaults (§4.12.2.2) are `simple` for path and header and `form` for
query and cookie. `explode` defaults to `true` for `form` and `cookie`,
otherwise `false`. Loom's authored parameter defaults match OpenAPI except in
two places, both retained behavior:

- Cookies default to the `cookie` style (section 7).
- A whole named query binding of map shape with no `style` and no `explode`
  keeps the JSON content codec. An explicit `style` or `explode` on a
  `ScalarMap` selects the style table. `allowReserved` alone never switches
  this codec. Because a content codec cannot declare `allowReserved`
  (§4.12.2), such a binding stays rejected, exactly as at the baseline
  (Current State). The remedy is an explicit `style: form` or removing
  `allowReserved`.

Imported schema-based Parameters use OpenAPI defaults before `Resolve`,
not Loom's native defaults. The shared `internal/httpencoding` owner provides
`NormalizeOpenAPIParameter(location, source) Serialization`: when the source
has no content media type and no explicit style, it supplies `simple` for
path/header and `form` for query/cookie; it preserves explicit-presence fields.
Content parameters pass through unchanged so contradictory style fields still
fail. The importer calls this helper in `analyzer.parameter` and in the
planner's common parameter-serialization path in `url_serialization.go` for
public `Document` values constructed directly by callers, before shape
admission or `Resolve`. The resulting codec reaches `parameterMetadata` so
rendering preserves these defaults. Thus a schema-based query map with only
`allowReserved` resolves as form, while the same native Loom metadata remains
a content-codec contradiction. This parameter helper is never applied to
Encoding Objects or native DSL bindings.

Defaults are location-specific. The Encoding Object rule is not a Parameter
rule:

- **Parameters (§4.12.2).** A parameter has either `schema`, with the
  schema-only `style`, `explode` and `allowReserved`, or `content`, never both.
  A named query parameter with a content media type and any style field is
  rejected.
- **Form fields (§4.15.1.2).** On an `application/x-www-form-urlencoded`
  Encoding Object, an explicitly defined `style`, `explode` or `allowReserved`
  selects the query style table before `contentType` is considered.
  "Explicitly defined" is by presence: an explicit `false` counts. An explicit
  or default `contentType` is then ignored, not rejected. With only `explode`
  or `allowReserved` present, the style is `form`. Without any of the three,
  the `contentType` applies, explicit or defaulted by type (§4.15.1.1). The
  default is `application/json` for objects and maps, `text/plain` for
  scalars, and `application/octet-stream` for a typeless (union) property.

The effective style and explode are computed once per binding, before any
union branch is resolved. A union never applies the map JSON default to one
branch while a sibling branch uses `form`. A union declared with
`content: application/json` is a JSON-content union: one JSON codec matched by
#1014's JSON matcher, not a text union.

```text
resolve(loc, name, v, src):
  flags := src.Style != "" or src.Explode != nil or src.AllowReserved != nil  -- presence, any value
  WholeQuery | FormRoot -> Root                                        (unchanged)
  FormField:                                                           (§4.15.1.2, §4.15.1.1)
      flags                     -> contentType ignored; go to STYLE
      src.MediaType = application/json:
          v in {Object, ScalarMap, Nested, Union} -> JSON (Union: #1014's JSON matcher)
          otherwise                               -> reject            (unchanged)
      src.MediaType other       -> reject                              (unchanged)
      v in {Object, ScalarMap, Nested} -> JSON (default contentType)
      v = Union                 -> reject "a typeless form field defaults to
                                   application/octet-stream" (#1014 routes it)
      otherwise                 -> go to STYLE (default form)
  NamedQuery:                                                          (§4.12.2)
      src.MediaType set:
          flags                 -> reject "content-based parameters cannot declare
                                   style, explode or allowReserved"   (unchanged)
          application/json and v in {ScalarMap, Nested, Union} -> JSON
          otherwise             -> reject                              (unchanged)
      v in {ScalarMap, Nested} and src.Style = "" and src.Explode = nil:  (retained default)
          src.AllowReserved != nil -> reject, the same content-based reason (unchanged)
          otherwise             -> JSON
  Path | Header | Cookie:
      src.MediaType set         -> reject "content serialization is query-only"
STYLE:
  style   := src.Style or LoomDefault(loc)  (FormField: form)  -- once per binding
  explode := src.Explode or (style in {form, cookie})           -- once per binding
  if style not allowed in loc (§4.12.3; FormField uses the query row) -> reject
  v = Union  -> ResolveUnion with (style, explode)
  v = Nested -> reject "nested values have no style serialization"
  switch style:
    deepObject: Object | ScalarMap -> deepObject (explode ignored, normalized false)
                else -> reject "deepObject is defined only for objects (§4.12.6)"
    spaceDelimited | pipeDelimited:
                explode -> reject "n/a for explode: true"
                Sequence | Object | ScalarMap -> codec
                Scalar -> reject "n/a for primitives"
    form in Cookie:
                Scalar -> form
                explode = false -> form for Sequence, Object, ScalarMap
                explode = true, non-Scalar -> reject "form with explode uses the wrong
                  cookie delimiter (Appendix D.1)"
    otherwise (simple, matrix, label, form in query and form fields, cookie):
                Scalar | Sequence | Object | ScalarMap -> codec(style, explode)
  Scalar codecs normalize explode to the style default (it has no effect).
```

Form fields (#1005). An explicit `style`, `explode` or `allowReserved` on an
`application/x-www-form-urlencoded` Encoding Object now resolves through the
same table as query parameters (OAS 3.2 §4.15.1.2). Today `Match` rejects it,
and it rejects a content type combined with a style field. The form body is its
own `KeySet` namespace. A `deepObject` form field holding an `Object` or
`ScalarMap` is what #1005 needs for nested tagged form unions.

Export keeps the form-field codec. Without any style field, an Encoding Object
for an object, map or union would select its content type. So export always
emits an explicit `style` for a styled form-field object, map or union, even
when that style is OpenAPI's default `form` (the `form-flag-dropped` control).

Multipart parts are not the `FormField` location. OAS 3.2 §4.15.1.2 also lets
style fields apply to `multipart/form-data`, but RFC 6570 multipart part
encodings belong to #973. Explicit style fields on a multipart part keep
today's admission. They are accepted only on a `Scalar`, or on a `Sequence`
whose resolved codec is exploded `form`, which matches the existing text and
repeated part codecs. Every other combination is rejected, as today
(`expr/http_media_encoding.go:62-63`).

### 3. Escaping and delimiters

Three decoding regimes follow the specification:

- **Path captures** are raw (`mux.RawVars`) and split before unescaping
  (Appendix C.2: "first split values by delimiter before percent-decoding").
- **Query and form-body values of `form` with `explode: false`** are also read
  raw and split on literal `,` before percent-decoding. RFC 6570 and C.2 encode
  a data comma as `%2C`, so `?n=a%2Cb,c` is `["a,b", "c"]`. `url.Values`
  decodes in one pass and loses this distinction. These decoders therefore
  read `r.URL.RawQuery` and the raw form body, split pairs on `&` and `=`, and
  split values on literal delimiters before decoding.
- **`spaceDelimited` and `pipeDelimited`** are single-pass by specification.
  Appendix E.6 says the decoding is a single pass. Distinguishing a `+`-encoded
  data space from the `%20` delimiter would need a non-standard parse that is
  "not interoperable across all tools". A data `|` and the delimiter are both
  `%7C`. Their delimiter character is therefore not representable in data.

Cookie values are never percent-decoded by the transport.

| Codec | Escaping | Delimiters | Values the encoder rejects |
|---|---|---|---|
| path `simple` scalar and sequence | `http.EscapePathSegment` per value or element, byte for byte as today | `,` lists | an empty segment: the empty string, an empty list, or `[""]` |
| path `simple` object/map, `matrix` | `url.PathEscape` plus `=` → `%3D`, then dot-segment protection | `,` lists/objects; `=` pairs; `;` matrix | an empty segment (an empty `simple` object or map) |
| path `label` | same as the row above | `.` prefix; `,` (false); `.` (true) | a zero-member collection (no segment); with `explode: true`, any element, key or value containing `.` |
| query/form-field `form` | `url.QueryEscape` | `&`, `=`; literal `,` for `explode: false` | none beyond section 5 |
| query/form-field `deepObject` | same | key `name%5Bkey%5D` | keys containing `[` or `]` |
| query/form-field `spaceDelimited` | same | `%20` | elements, keys and values containing a space |
| query/form-field `pipeDelimited` | same | `%7C` | elements, keys and values containing `|` |
| header `simple` | none (§4.21.1.2) | `, ` lists/objects; `=` exploded pairs | list elements, unexploded keys and values that are empty, contain `,` or have edge whitespace; exploded values containing `,` or ending in whitespace; exploded keys that are empty, contain `,` or `=`, or start with whitespace |
| cookie `cookie` | none (§4.12.2.2) | `,` (false); `; ` pairs (true) | any element, key or value with a byte outside 0x20-0x7E, or containing `"`, `;`, `\` or `,` (non-ASCII such as `é` included); a serialized cookie value with edge whitespace; exploded keys that are not RFC 9110 tokens |
| cookie `form` | RFC 6570 unreserved set; space as `%20` | `,` (false) | none beyond section 5 |

Dot-segment protection (path). The encoder serializes the complete style
grammar first: the label prefix, the matrix `;name=` and the delimiters. It
then inspects the resulting segment. Only a segment that is exactly `.` or `..`
is replaced by `%2E` or `%2E%2E`. Every other segment keeps its structural
separators literal. The encoder emits uppercase hex (RFC 3986 §2.1).

The decoder first checks the raw capture. Only when the whole capture is a
protected spelling (`%2E`, `%2E%2E`, with hex digits matched
case-insensitively, so `%2e` and `%2E%2e` also qualify) does it replace the
capture with `.` or `..`. This happens before label-prefix removal and before
delimiter splitting. All other escapes stay raw until each token is decoded.
So the literal text `%2E`, sent as `%252E`, is decoded once and never becomes
a dot.

This is the spelling today's `EscapePathSegment` already produces for whole
scalar values. Existing `simple` scalar and sequence path builders, and
catch-all remainders, keep their current bytes.

| Codec, value | Serialized | Wire | Decoded |
|---|---|---|---|
| `simple` `.` / `..` | `.` / `..` | `%2E` / `%2E%2E` | `.` / `..` |
| `matrix` `.` | `;id=.` | `;id=.` (unchanged) | `.` |
| `matrix` `""` | `;id` | `;id` (unchanged) | `""` |
| `label` `""` | `.` | `%2E` | `""` |
| `label` `.` | `..` | `%2E%2E` | `.` |
| `label` `[""]`, either explode | `.` | `%2E` | `[""]` |
| `label` or `simple` `[]` | nothing | — | encoder error: no segment |
| `simple`, literal text `%2E` | `%252E` | `%252E` | `%2E` |
| `label`, literal text `%2E` | `.%252E` | `.%252E` | `%2E` |
| `matrix` name `id`, literal text `%2E` | `;id=%252E` | `;id=%252E` | `%2E` |

RFC 6570 §3.2.1 treats a list or associative array with zero members as
undefined: it expands to nothing, not even the label prefix. §3.2.5 keeps the
prefix for a defined empty string. So label `""` and `[""]` are representable,
and a zero-member collection cannot address a whole-segment route.

Exploded label collections still reject a data `.` in any element, key or
value. It collides with the `.` delimiter. Its only other spelling, `%2E`
inside the segment, changes element counts when an intermediary
percent-decodes unreserved characters (RFC 3986 §6.2.2.2; `label-pct-dot`
control). This is distinct from the label prefix dot. The design makes no
claim that protected dot segments survive intermediaries that remove dot
segments (RFC 3986 §5.2.4). They rely on the raw path being preserved, as
`EscapePathSegment` does today.

The `cookie` byte set is net/http's `validCookieValueByte` minus `,`. The `,`
exclusion follows the OAS 3.2 parameter example ("the comma … is forbidden in
cookie values") and RFC 6265 cookie-octets. It is the only rejection not
derived from a decoding failure. Interior spaces are allowed, as in that
example's `greeting=Hello%2C world!`. Loom writes cookie pairs verbatim:
`http.AddRequestCookie` appends `name=value` after the codec's domain check,
instead of calling net/http's quoting `AddCookie`.

### 4. Decoding grammar

Decoders split on the delimiter, preserving empty tokens, before decoding.
Then they validate by shape. Header decoders keep RFC 9110's rule that empty
list members are dropped; header encoders never emit them.

```text
Path, before anything else:
  raw capture exactly %2E or %2E%2E (hex case-insensitive) -> "." or ".."
  then remove the label prefix, then split, then decode each token once
Sequence:
  codecs where [] and [""] share the spelling "name=" (query/form-field form,
  space and pipe with explode false; cookie with explode false):
      whole value empty      -> present empty list
  path matrix ";name"         -> present empty list; ";name=" -> [""]
  otherwise split preserving empty tokens; each token is one element
Object / ScalarMap, unexploded (k,v,k,v):
  whole value empty           -> present, zero entries
  odd token count, duplicate key -> 400
  Object: undeclared key -> 400; empty value token -> property ""
Object / ScalarMap, exploded (k=v):
  pair without "="            -> 400 (path, header)
  duplicate key in one value  -> 400 (path segment, header field)
  query/form top-level keys, deepObject keys -> owned by KeySet (section 6);
      duplicate key -> 400; an undeclared Object key is not owned and is ignored
  cookie pairs -> first occurrence per name (RFC 6265 §5.4 orders by path);
      exploded arrays take every occurrence in order
Scalar repeated keys keep the existing first-value rule.
Repeated keys for a delimited sequence or unexploded object -> 400.
```

Every 400 is a `decode_payload` problem without service invocation. Its
`SafeDecodePayloadMessage` detail names the parameter, never its value.

### 5. Representable domain and key presence

Presence follows the scalar-binding rules (`docs/http-guide.md:555-569`).
Defaults apply only to absence, and required bindings reject absence.

| Codec | Empty scalar | Empty list / zero entries | List of one empty element |
|---|---|---|---|
| path `simple` | rejected (empty segment) | rejected (no segment) | rejected (empty segment) |
| path `label` | `.` sent as `%2E` | rejected (no segment, RFC 6570 §3.2.1) | `.` sent as `%2E` |
| path `matrix` scalar/list/unexploded object | `;name` | `;name` | `;name=` |
| path `matrix` exploded object | n/a | rejected: no marker | n/a |
| query/form-field `form` scalar | `name=` | n/a | n/a |
| query/form-field `form` exploded list | n/a | Rule A | `name=` |
| query/form-field `form`, space, pipe unexploded | n/a | `name=` | rejected (same spelling as empty) |
| query/form-field exploded object, `deepObject` | n/a | Rule A | n/a |
| header list, object | empty field (existing) | empty field | rejected |
| cookie scalar | `name=` | n/a | n/a |
| cookie unexploded list/object | n/a | `name=` | rejected (same spelling) |
| cookie exploded list | n/a | Rule A | `name=` |
| cookie exploded object | n/a | Rule A | n/a |

Rule A applies to key presence in a parameter namespace or form-field
namespace only. When a codec's empty collection produces no key, an optional
binding with a nonnil empty value is sent as absence and decodes as absence. A
required binding with such a value is a client encoder error before the
request is built; a request with no key is a 400 missing field.

Rule A does not apply to whole bodies, form roots or multipart envelopes. A
present empty form body or empty multipart envelope is a present value owned
by the body/media decoder; #1014 relies on that to match a closed
empty-object branch.

### 6. Key namespaces and collisions

| Binding | KeySet |
|---|---|
| named scalar, sequence, unexploded object, delimited, JSON | `Literals {name}` |
| exploded `Object` (`form` query/form field, `cookie` cookie) | `Literals {property wire names}` |
| `deepObject` `Object` | `Literals {name[p] …}` |
| `deepObject` `ScalarMap` | `Prefixes {name}` |
| exploded `ScalarMap` (`form` query/form field, `cookie` cookie), `MapParams` | `Residual` |
| union | union of its branches' sets; any number of `Residual` alternatives form one claim |
| query/cookie API-key credential, session cookie | `Literals {wire name}` |
| path, header | own wildcard or header name (existing uniqueness rules) |

Rejection is by endpoint binding; alternatives inside one union are not
siblings:

```text
CheckNamespaces(bindings):
  two bindings' literals intersect                     -> reject naming both
  a literal equals prefix[...] of another binding      -> reject
  two bindings with the same prefix                    -> reject
  more than one binding with Residual, or MapParams with any other binding -> reject
  a deepObject name or property contains "[" or "]"    -> reject
wire key ownership: literal owner, else prefix owner, else the Residual binding,
                    else ignored (unknown parameter, existing behavior)
dynamic encoder:    a map entry whose key another binding owns -> encoder error
inside one union:   every alternative reads the union's whole universe,
                    including keys declared by other alternatives; #1014's
                    matcher decides exact-one
```

Residual semantics (settled). An authored dynamic binding receives every key
that no sibling binding claims, including unrelated keys or browser cookies a
client sends. That is the contract the source states for an exploded
free-form object, and the guide must say so.

| Bindings | Wire | Outcome |
|---|---|---|
| `limit` scalar; `labels` ScalarMap `form` exploded | `limit=5&env=prod&tier=1` | `limit=5`, `labels={env:prod, tier:1}` |
| same, empty map, optional | `limit=5` | `labels` absent (Rule A) |
| same, empty map, required | — | client encoder error; server 400 for a missing field |
| same, `labels={limit:9}` | — | encoder error: key `limit` belongs to `limit` |
| same, foreign `utm_source=x` | `limit=5&utm_source=x` | `labels={utm_source:x}` |
| `filter` ScalarMap `deepObject`; `sort` scalar | `filter%5Bstatus%5D=open&sort=asc` | `filter={status:open}`; a literal sibling `filter[x]` is rejected at design time |
| union `id`: scalar or exploded ScalarMap of integers | `id=5` | both alternatives read `id`: ambiguous, rejected by #1014's matcher |
| union of two exploded ScalarMap alternatives | — | admitted: one dynamic claim; the matcher decides per request |
| two exploded ScalarMap bindings | — | rejected: two dynamic namespaces |
| ScalarMap exploded beside `MapParams` | — | rejected |
| cookie `prefs` ScalarMap `cookie` exploded; session cookie `sid` | `Cookie: sid=1; theme=dark` | `prefs={theme:dark}`; `sid` owned by the session binding |
| form field `pet` `deepObject` Object `{type, name}`; field `owner` | body `pet%5Btype%5D=cat&pet%5Bname%5D=Tom&owner=Ann` | `pet={type:cat,name:Tom}` (#1005) |
| ScalarMap of arrays, any style | — | rejected: nested values have no style serialization |
| header `X-Labels` ScalarMap `simple` exploded | `X-Labels: env=prod, tier=1` | `{env:prod, tier:1}` |

### 7. Cookies

Wire forms follow §4.12.6 and Appendix D.1:

| Style, explode | Scalar `a/b` | Array `[blue, black]` | Object or map `{R:100, G:200}` |
|---|---|---|---|
| `cookie`, false | `color=a/b` | `color=blue,black` | `color=R,100,G,200` |
| `cookie`, true (default) | `color=a/b` | `color=blue; color=black` | `R=100; G=200` |
| `form`, false | `color=a%2Fb` | `color=blue,black`; a data comma is `%2C` | `color=R,100,G,200` |
| `form`, true | `color=a%2Fb` | rejected (D.1 wrong delimiter) | rejected (D.1) |

Server decoding reads `r.Cookies()`, which also strips a surrounding quote
pair from legacy peers and drops pairs with invalid bytes. It splits before
percent-decoding `form` values.

Default. The authored default stays `cookie`. OAS 3.2 says `style: cookie`
SHOULD be used with `in: cookie`, and the `form` default exists "for
compatibility reasons". Appendix D.1 explains that `cookie` matches values set
through `Set-Cookie`, which Loom response cookies are. A `form` default would
percent-decode `%XX` in such values.

Export:

- 3.2.0: emits `style: cookie` for the default. Explicit `form` cookies export
  without a style.
- 3.1.1 has no `cookie` style. A `cookie`-style binding is described as `form`
  only when the two are wire-identical over every key and every value the
  binding can send:
  - The binding is a scalar, or uses `explode: false`.
  - Every value domain is unreserved-only: integers, booleans, enums whose
    literals are all RFC 3986 unreserved, or `date` strings. UUID format alone
    is insufficient because accepted URN and brace spellings contain reserved characters.
  - Every key is unreserved-only. That means fixed property wire names checked
    one by one, and a `ScalarMap` key attribute that is an integer or an
    enum of unreserved literals.
  - An unrestricted string-keyed map never qualifies. For example,
    `MapOf(String, Int)` with key `a/b` is `a/b` raw but `a%2Fb` under `form`.
- Otherwise 3.1.1 generation fails, naming the cookie. The remedies are the
  3.2.0 target, or `Meta("openapi:style", "form")`, which changes the wire to
  percent-encoding.

Import renders `openapi:style: form` for a source cookie with no style, and
`openapi:style: cookie` only for an explicit `cookie`.

Migration (observable):

- Values with interior spaces are sent unquoted (`n=a b` instead of `n="a b"`);
  net/http servers decode both identically.
- `cookie`-style values containing `,` or non-ASCII bytes are encoder errors.
  The remedy is `openapi:style: form`.
- Designs whose `cookie`-style bindings fail the 3.1.1 equivalence lose 3.1.1
  output. Their 3.2.0 output gains `style: cookie`.

Other observable changes, outside cookies:

- Path `.`/`..` values keep their current `%2E` spelling. Existing `simple`
  scalar and sequence builders, and catch-all remainders, are byte-identical.
- An empty `simple` path value (`""`, `[]` or `[""]`) is a client encoder
  error. Today the client sends an empty segment such as `/items/`. The design
  treats that as unable to carry a value through a whole-segment wildcard, and
  does not depend on how a muxer routes it.
- A form-field Encoding Object with a style field plus an explicit
  `contentType` is now accepted with the style codec, where `Match` rejected it
  before.
- Re-importing a document whose cookies have no style yields the `form` codec
  it declares.

### 8. Objects and maps in every location

`ScalarMap` uses the `Object` spellings with runtime keys:

| Location, style | Object `{R:100, G:200}` | ScalarMap `{env: prod}` |
|---|---|---|
| path `simple` false / true | `R,100,G,200` / `R=100,G=200` | `env,prod` / `env=prod` |
| path `matrix` false / true | `;color=R,100,G,200` / `;R=100;G=200` | `;color=env,prod` / `;env=prod` |
| path `label` false / true | `.R,100,G,200` / `.R=100.G=200` | `.env,prod` / `.env=prod` (true: no `.` in keys or values) |
| query/form-field `form` false | `color=R,100,G,200` | `color=env,prod` |
| query/form-field `form` true | `R=100&G=200` | `env=prod` (Residual) |
| query/form-field space / pipe | `color=R%20100%20G%20200` / `color=R%7C100%7CG%7C200` | `color=env%20prod` / `color=env%7Cprod` |
| query/form-field `deepObject` | `color%5BR%5D=100&color%5BG%5D=200` | `color%5Benv%5D=prod` (Prefix) |
| query, no metadata | `R=100&G=200` (OpenAPI default) | JSON `color=%7B%22env%22%3A%22prod%22%7D` (retained) |
| form field, no metadata | JSON (Encoding Object default) | JSON |
| header `simple` false / true | `R, 100, G, 200` / `R=100, G=200` | `env, prod` / `env=prod` |
| cookie | section 7 | section 7 |

Import changes:

- A source query map with no `content` and no style means `form` with
  `explode: true`. Import now admits it as a dynamic binding and renders
  `openapi:style: form`, replacing today's blocking mismatch.
- A source with `content: application/json` still imports as JSON.
- Nested maps keep JSON only.

### 9. Text unions (shared with #1014)

```text
ResolveUnion(loc, name, branches, src):
  (style, explode) := effective serialization of the binding (section 2), once
  for each branch b (nested unions recurse):
      codec_b := resolve b under (style, explode)
      error   -> reject naming the branch index and the reason
      keys_b  := KeySet of codec_b
  union KeySet := literals and prefixes of all branches; Residual if any branch is Residual
  return branch codecs and the union KeySet
```

The union KeySet takes part in `CheckNamespaces` like any binding.
Name-erasing alternatives (exploded objects, deepObject, dynamic maps) are
admitted; siblings that collide with them are rejected at design time.

Per-branch runtime contract. The matcher is #1014's (revision 5, section 3):
`internal/unionmatch.Select` over `Candidate{Schema, Decode}`, reached through
`loomhttp.MatchUntaggedText` and its `TextUnionBranch` adapters. #968
supplies each branch's callbacks:

| Callback | Contract |
|---|---|
| `Schema() (bool, error)` | Parses the branch's raw view with its codec into the JSON data model (exact numbers, no Go widths) and evaluates the branch schema and constraints. Pure. `false` means not a member. |
| `Decode() (bool, error)` | Typed Go decode and validation into a private candidate. `false` is an ordinary mismatch, including overflow, syntax, validation and "branch not sent". Writes no shared state. |
| error from either | Fatal (evaluator or configuration failure). It propagates unchanged and is never counted as a mismatch. |

#968's runtime seam is `loomhttp.TextBranch`, with methods
`Schema() (bool, error)` and `Decode() (bool, error)`. It holds four functions:

- `View`, which reads the branch's raw view with its codec;
- `Shape`, the branch's `ParamBranchShape`;
- `Member`, which evaluates the branch schema on the JSON data model;
- `Assign`, the typed conversion and validation into a private candidate.

`ParamValueJSON` converts a raw view into the JSON data model with exact
number tokens. #968 tests the seam with hand-written `Member` and `Assign`
functions. #1014's generated adapters supply them and do the counting. #968
adds no matcher.

The matcher visits every branch and keeps separate schema and decoder counts.
It commits only when both counts are one and name the same branch. So carrier
overflow never resolves a schema ambiguity: `9` against an int64 integer
branch and an int32-valued map branch matches both schemas and is rejected.
The client re-match runs the same callbacks over the encoder's own wire.

### 10. Authored DSL

```go
Payload(func() {
    Attribute("id", ArrayOf(Int), func() { Meta("openapi:style", "label"); Meta("openapi:explode", "true") })
    Attribute("range", Range)                   // Type("Range", func(){ Attribute("gt", Int); Attribute("lt", Int) })
    Attribute("labels", MapOf(String, String), func() { Meta("openapi:style", "deepObject") })
    Attribute("tags", ArrayOf(String), func() { Meta("openapi:style", "form"); Meta("openapi:explode", "false") })
    Attribute("prefs", MapOf(String, String))  // cookie: cookie style, exploded, Residual
})
HTTP(func() {
    GET("/items/{id}")
    Param("range", func() { Meta("openapi:style", "deepObject") })
    Param("labels")
    Param("tags")
    Cookie("prefs")
})
```

Metadata keys are `openapi:style`, `openapi:explode` and `openapi:allowReserved`
on parameters, and the existing
`openapi:encoding:<name>:{style,explode,allowReserved}` on form fields
(`expr/http_media_encoding.go:47-66`). They are read on the effective mapped
attribute, occurrence first. Expr validation calls `Resolve`/`ResolveUnion`
and `CheckNamespaces`. These are evaluation errors:

- an unknown style, or a style illegal for the location;
- a non-boolean explode;
- parameter metadata on a body field;
- a namespace overlap.

DSL admission widens to `Object` and `ScalarMap` in path, header and cookie.
Default-equivalent metadata is accepted and omitted on export. No new DSL
functions are added.

Unions are split by ticket order (#968, then #1014, then #1005, with no
cycle). #968 delivers:

- `ResolveUnion`;
- union KeySets in `CheckNamespaces`;
- `Union` shapes from both adapters, built over the existing untagged `OneOf`;
- the runtime branch seams of section 9.

#1014 removes today's location guards for unions
(`expr/http_endpoint_mapping.go:127-132,158,175`;
`internal/openapiimport/plan.go:195-197`, `schema-oneof-location`). It also
adds the matcher, the branch-declaration syntax and the generated adapters.
Until #1014 lands, unions at text locations stay rejected.

### 11. Projection surfaces

| Surface | Derivation |
|---|---|
| Client request encoders | `Codec` per `transportir.Parameter` replaces `Encoding`. `http` helpers cover path segments, raw query and form-body assembly with literal delimiters, deepObject keys, header objects, dynamic and prefix maps, and verbatim cookie pairs |
| Server request decoders | same `Codec`. Path reads `RawVars`. Query and form-body codecs that split read `r.URL.RawQuery` or the raw form body, with KeySet ownership. Cookies read `r.Cookies()` |
| Response headers | header `simple` codecs through the server response encoder (`http/codegen/template_sources_response.go:155-161`), the client response decoder (`template_sources_response_decoder.go:202-215`), the OpenAPI IR header builder (`http/codegen/openapi/internal/ir/document.go:460-468`, gains `Style`/`Explode`), import planning and render. An unrepresentable value fails before commit |
| OpenAPI export 3.2.0 | emits `style`/`explode` only where they differ from the OpenAPI default. `cookie` cookies emit `style: cookie`. `deepObject` never emits `explode`. Metadata-free query maps emit `content`. Form-field styles go to Encoding Objects. `validateURLFieldEncoding` covers every location |
| OpenAPI export 3.1.1 | section 7 cookie projection; everything else as 3.2.0 |
| Import analysis | `supportedParameterStyle` and the non-query explode rejection are deleted. `admitParameterShape`, `querySerialization` and `formSerialization` become `Resolve`/`ResolveUnion` calls, followed by `CheckNamespaces` per operation and per form body |
| Import render | `openapi:style`/`openapi:explode` only where the codec differs from Loom's default for the location and shape |
| CLI | unchanged; CLI JSON input is not a URL codec |
| Ledger and guide | the Parameter and Encoding rows, "Fixed object query parameters remain unsupported", and the named-map mismatch paragraph are replaced by sections 2-8 |

## Detailed Behavior

Invariant: on one endpoint every binding has exactly one `Codec` and one
`KeySet`. The client encoder, server decoder, exported document and re-imported
design agree on the codec. No two bindings own one wire key.

| Location, style, explode | Scalar `blue` | Array `[blue, black]` | Array `["a,b", c]` |
|---|---|---|---|
| path `simple` | `/blue` | `/blue,black` | `/a%2Cb,c` |
| path `matrix` false / true | `/;color=blue` | `/;color=blue,black` / `/;color=blue;color=black` | `/;color=a%2Cb,c` |
| path `label` false / true | `/.blue` | `/.blue,black` / `/.blue.black` | `/.a%2Cb,c` / `/.a%2Cb.c` |
| header `simple` | `blue` | `blue, black` | encoder error (`,`) |
| query `form` true | `color=blue` | `color=blue&color=black` | `color=a%2Cb&color=c` |
| query `form` false | `color=blue` | `color=blue,black` | `color=a%2Cb,c` |
| query space / pipe | `n/a` | `color=blue%20black` / `color=blue%7Cblack` | `color=a%2Cb%20c` / `color=a%2Cb%7Cc` |
| query `deepObject` | `n/a` | `n/a` | `n/a` |
| cookie `cookie` false | `color=blue` | `color=blue,black` | encoder error (`,`) |

Decoders accept both percent-encoded and literal `[`/`]` in deepObject keys,
because both spellings name the same key after the key is decoded.

Accepted imports:

- path `{id}` with `label`, `explode: true`, `integer` array;
- query `filter` `deepObject` object of strings and integers;
- query `labels` `deepObject` with `additionalProperties: {type: string}`;
- query `tags` `form`/`explode: false` with comma-bearing string elements;
- query `tags` `pipeDelimited`;
- header `X-Range` `simple` exploded flat object;
- cookie `prefs` `cookie` exploded string map;
- cookie `ids` `form`/`explode: false` integer array;
- a query map with no `content` or style (dynamic `form`);
- a form field with Encoding `style: deepObject` over a flat object;
- a form field object with `contentType: application/json` and
  `explode: false` (unexploded `form`; the content type is ignored);
- a form field object with only `allowReserved: false` (exploded `form`);
- explicit default `style: form, explode: true` on a scalar (normalized).

Rejected imports, fatal under `--allow-lossy`, with reasons:

- `deepObject` on `type: array`: "deepObject is defined only for objects
  (§4.12.6)".
- `deepObject` with an object or array property, or array-valued
  `additionalProperties`: "nested values have no style serialization".
- `spaceDelimited` with `explode: true`: "n/a for explode: true".
- `form` cookie array with default `explode`: "form with explode uses the wrong
  cookie delimiter (Appendix D.1)".
- `matrix` in query: "style matrix is not defined for in: query".
- query `content` plus `style`, `explode` or `allowReserved`: "content-based
  parameters cannot declare style, explode or allowReserved".
- Two exploded free-form query maps on one operation: "two dynamic
  namespaces".
- `nullable` or `Any` members.

Value-level encoder errors at runtime name the parameter, never the value:

- label exploded element `1.5`;
- `simple` path value `""`, `[]` or `[""]`, and a `label` or `simple` empty
  collection (no segment);
- `spaceDelimited` element `a b`;
- `cookie`-style value `a,b` or `é`;
- header element ` a`;
- a dynamic map entry whose key a sibling owns.

## Contract Matrix

| Capability | Supported | Rejected | Deferred | Default | Source of truth | Placement | Routing | Naming | Equivalence | Proof |
|---|---|---|---|---|---|---|---|---|---|---|
| Path styles | simple, matrix, label × both × Scalar, Sequence, Object, ScalarMap; `.`/`..` segments through `%2E` protection | nested; empty segments and zero-member collections; label-exploded data `.` | none | simple | `httpencoding.Resolve` | `http/path.go` helpers | raw capture, protected dots reversed first, split before unescape | whole wildcard | client = server = export; existing simple and catch-all bytes unchanged | `CodecRoundTrip` (`dot-reject`, `unprotect-after-prefix`, `dot-case-sensitive`, `unprotect-late`), `ParameterResolution` |
| Query and form-field styles | form (both), space/pipe (false), deepObject per table | `n/a` cells; nested; delimiter-bearing space/pipe values | none | form/true; metadata-free query maps and form-field objects JSON | same | query/form templates, `http` helpers | raw split for `form`/false; KeySet ownership | name, property names, `name[key]`, Residual | same | both models, `KeyNamespaces` |
| Header values (request and response) | simple × both × Sequence, Object, ScalarMap | nested; section 3 values | none | simple/false | same | `internal/headerlist`, templates, IR header builder | header field | header name | request = response codecs | `CodecRoundTrip` |
| Cookies | cookie × both × all shapes; form × Scalar and explode false | form explode true non-primitive (D.1); section 3 bytes; 3.1.1 non-equivalent cookie style | none | cookie/true | same | `http/request_cookie.go`, cookie templates | `r.Cookies()` | cookie name, property names, Residual | 3.2.0 names the style; 3.1.1 only when keys and values are wire-identical | `CodecRoundTrip`, `ParameterResolution` |
| Maps | ScalarMap in every defined style | Nested with a style | none | per location | same | runtime map helpers | Residual and prefix ownership | runtime keys | sorted keys | `KeyNamespaces`, `CodecRoundTrip` |
| Text unions | branches resolving under one style; any number of dynamic alternatives | a failing branch; namespace collisions | none | binding's effective style | `ResolveUnion` | #1014 matcher | `Schema` and `Decode`, two counts, same index | union KeySet | per-branch round trip | `BranchMatch` |
| Authored metadata | parameter and Encoding Object style fields; form-field style fields override `contentType` by presence | unknown or illegal values, overlaps; Parameter `content` with style fields; `allowReserved` alone on a retained-JSON query map | none | Loom default; location-specific | attribute metadata via `Resolve` | `expr` validation | n/a | existing keys | default-equivalent omitted; styled composite form fields always name their style | evaluation rejection tests; `ParameterResolution` (`reserved-switches-query`, `encoding-on-parameters`, `form-content-first`, `media-style-reject`, `form-explicit-false-ignored`, `form-flag-dropped`) |

## Acceptance Trace

| Issue acceptance item | Where met | Evidence obligation |
|---|---|---|
| Shared binding codec contract for supported OAS combinations | sections 1-2, 9 | `ParameterResolution` checked (122,242 distinct states) against the §4.12.2, §4.12.3, §4.12.6, §4.15.1.2 and D.1 data; per-cell and location-default table tests |
| Independent client/server wire examples | sections 7-8, Detailed Behavior | generated client and mounted server tests per row, plus a foreign peer per cookie row |
| Matrix and label paths | sections 3, 8 | `CodecRoundTrip` path rows; the `label-pct-dot`, `dot-reject`, `unprotect-after-prefix`, `dot-case-sensitive` and `unprotect-late` controls |
| Delimited queries | sections 3-5 | raw-split `form` (`single-pass-comma` control) and E.6 space/pipe (`no-delim-check`) |
| deepObject scalar properties | sections 2, 6, 8 | prefix and literal ownership; both bracket spellings |
| Permitted header and cookie structures | sections 3, 5, 7, 8 | header and cookie rows with both cookie peers; `cookie-nonascii` control |
| Explode variants | every table | `FullScope`; `reduced` control |
| Object/array values in path/header/cookie | section 8 | as above, including ScalarMap |
| Reject undefined nested deepObject | section 2 | `deep-arrays` control; import rejection tests |
| Focused regressions and ledger | Projection surfaces | ledger rows per Contract Matrix row |
| Existing OpenAPI validation and generated-code checks | Proof Obligations | libopenapi and pinned Vacuum on 3.2.0 and 3.1.1; compile and determinism |

## Alternatives Considered

- Keep `Mode` and add a mode per style. Rejected: `Codec` is the
  (style, explode) pair every consumer needs.
- New DSL functions. Rejected: metadata already carries these fields.
- Decode query `form` lists with `url.Values`. Rejected: it decodes before
  splitting and loses `%2C` data commas (`single-pass-comma`).
- An escape convention for space and pipe delimiters. Rejected by E.6 as
  non-interoperable.
- `%2E` for dots in label exploded elements. Rejected: normalization changes
  element counts (`label-pct-dot`).
- Reject whole `.`/`..` path segments (revision 4). Rejected: they are
  representable through the existing `%2E` protection, which Loom already
  ships for scalar path values (`dot-reject`).
- Reverse protection after stripping the label prefix, compare hex case
  exactly, or check the protected spelling after percent-decoding. Rejected
  (`unprotect-after-prefix`, `dot-case-sensitive`, `unprotect-late`).
- Let `allowReserved` alone switch a retained-JSON query map, apply the
  Encoding Object rule to Parameter `content`, prefer `contentType` over
  explicit form-field style fields, keep the blanket content-plus-style
  rejection for form fields, treat an explicit `false` as absent, or omit a
  styled composite form field's default style on export. Rejected
  (`reserved-switches-query`, `encoding-on-parameters`, `form-content-first`,
  `media-style-reject`, `form-explicit-false-ignored`, `form-flag-dropped`).
- Reject maps, or keep them JSON-only. Rejected: OpenAPI objects include
  free-form objects (`maps-json-only`, `keys-maps-rejected`).
- Admit only name-preserving union branches, or treat dynamic alternatives as
  conflicting. Rejected: KeySet composition with endpoint collisions is sound
  (`union-name-only`, `alternatives-conflict`, `branch-greedy-ownership`).
- Apply defaults per union branch. Rejected (`per-branch-default`).
- Cookie default `form`. Rejected: it breaks the Set-Cookie round trip D.1
  describes, and contradicts the §4.12.2.2 SHOULD.
- Describe `cookie`-style cookies as `form` on 3.1.1 by values alone, or
  always. Rejected (`form31-values-only`, `form31`).

## Risks And Tradeoffs

- Name-erasing objects and dynamic maps widen collision checks. Designs fail
  only when they adopt the new metadata or map/object bindings.
- A dynamic binding receives unrelated client keys and cookies. This is the
  source contract, and it is documented.
- Raw query and form-body parsing adds one parser per codec family. It must
  agree with `url.ParseQuery` on everything except split order.
- Cookie quoting, comma and non-ASCII rejection, and 3.1.1 withdrawals are
  breaking. Each is observable and has a one-line remedy.
- Rule A turns required empty no-key collections into encoder errors.

## Proof Obligations

Formal tools stay opt-in. Results are in
`internal/httpencoding/tla/parameter_codecs/README.md`.

- **Resolution, export and import.** Bounded: `ParameterResolution` checked
  passes, including `EncodingStyleFirst`, `ParameterContentExclusive` and
  `QueryReservedNeutral`. Fourteen controls fail: `legacy`, `reduced`,
  `deep-arrays`, `form31`, `form31-values-only`, `maps-json-only`,
  `union-name-only`, `per-branch-default`, and the six location-default
  controls listed under Alternatives. Implementation: per-cell tables;
  location-default tables covering query `allowReserved` alone, form-field
  explicit `false` flags, and an explicit `contentType` plus a flag; 3.2.0
  and 3.1.1 exporter-symmetry round trips with reserved map keys and property
  names.
- **Codec round trip and domain.** Bounded: `CodecRoundTrip` checked passes
  with `DomainMaximal`, relative to the specified decoders and to the stated
  raw-path assumption for protected dot segments. Twelve controls fail.
  Implementation: generated wire tests over the alphabet plus `"`, `\` and
  multi-byte UTF-8, with both cookie peers and a `%2E`-normalizing proxy.
  Foreign spellings must decode as specified: `R,,G,200`, `,R,100`, odd
  counts, duplicates, `;name=`, `n=a%2Cb,c`, `%2e`, `%2E%2e`, and `%252E`
  (the literal text `%2E`). The existing `http/path_test.go` and
  `path_escaping_integration_test.go` cases remain unchanged.
- **Namespace soundness.** Bounded: `KeyNamespaces` checked passes; six
  controls fail. Implementation: a rejection test per `CheckNamespaces` rule,
  and mounted tests with dynamic maps beside siblings and as union
  alternatives.
- **Union branch contract.** Bounded: `BranchMatch` checked passes, including
  a literal sibling beside the union's residual claim; seven controls fail.
  In that model `Decode` implies `Schema`, so a schema/decoder identity
  mismatch is unreachable and is not evidenced here; #1014's `UntaggedText`
  model owns that check. Implementation: `ResolveUnion` tables and #1014's
  `MatchUntaggedText` tests.
- **Shape adapter agreement.** `expr.HTTPValueShape` and the importer
  `urlShape` produce equal `BranchShape` values, including `Domain`,
  `PropertyFact.Domain` and `Keys`, for every exporter-symmetry parameter,
  header and form field.
- **Response header symmetry.**
- **Generated compilation and determinism.**
- **OpenAPI validity on both outputs.**

## Cross-Owner Requirements

- #1014 revision 5 consumes this interface as written: `ResolveUnion(loc,
  name, branches, src)`, shapes `Object` and `ScalarMap`, `KeySet{Literals,
  Prefixes, Residual}`, `CheckNamespaces`, and the per-branch `Schema` and
  `Decode` callbacks behind `internal/unionmatch.Candidate`. These are reached
  through `loomhttp.TextBranch` (section 9). #1014 owns the two-count matcher;
  #968 owns the raw carriers those callbacks read. `BranchShape.Domain` is an
  additive field that #1014 receives from the adapters. The expr adapter is
  `expr.HTTPValueShape`, not `transportir.URLShape`.
- #1014's form-root contract retains raw form pairs and independent body
  presence. Explicit `form`/`explode: false` fields split before decoding,
  through this owner's carrier (section 3); the cross-owner correction is
  reviewed and complete.
- #1005: `deepObject` and the rest of the query style table are available
  under `FormField` for `Object` and `ScalarMap` when explicitly authored.

## Review Status

Earlier reviews:

- distinct cookie codecs;
- representable-domain table;
- split-preserving grammar;
- response-header projection;
- form-field and root branches;
- collision helper;
- parallel shape adapters;
- no raw cookie described as `form`;
- Rule A.

The full-scope revision supported:

- label exploded collections;
- all cookie structures;
- maps under every style;
- union KeySets;
- RFC 9110 header empties.

This revision (final review):

1. HIGH: the union contract is #1014's two-count matcher. It has independent
   schema and decoder counts, the same branch, fatal precedence, and
   overflow-safe ambiguity. `BranchMatch` was rebuilt with decoder-only,
   schema-only, fatal-as-mismatch and alternatives-as-siblings controls.
2. HIGH: dynamic alternatives inside one union are one claim. Only multiple
   dynamic endpoint bindings conflict, and alternatives see each other's keys
   (`alternatives-conflict` control).
3. HIGH: `form`/`explode: false` data commas are representable through a raw
   carrier. The space and pipe rejections now rest on E.6. `DomainMaximal` is
   stated as relative to the specified decoders.
4. HIGH: the 3.1.1 cookie projection requires equivalence over keys and values
   (`form31-values-only` control).
5. MED: the exact cookie byte set is 0x20-0x7E minus `"` `;` `\` `,`;
   non-ASCII is rejected (`cookie-nonascii` control).
6. MED: the effective style is resolved once per binding. The JSON default
   applies only to whole map bindings, and JSON-content unions are separate
   (`per-branch-default` control).

Also: explicitly styled form fields for #1005, and dynamic-binding semantics
settled.

Revision 5. Execution planning found three gaps, and an independent reviewer
resolved them:

- **DB-1, path dot segments.** Revision 4 rejected `.`/`..` path segments, which
  contradicted the shipped `%2E` protection (`http/path.go:9-23`). Revision 5
  protects only a complete serialized segment that is exactly `.` or `..`. The
  decoder reverses only those complete spellings, case-insensitively, before
  removing the label prefix and splitting. The existing simple and catch-all
  bytes are preserved. Label `""` and `[""]` are representable, while
  zero-member collections are not (RFC 6570 §3.2.1, §3.2.5). The data-dot
  rejection in exploded label collections is kept. The model gained protected
  spellings, a lowercase-hex transport, the literal text `%2E`, and four
  controls.
- **DB-2, value domains.** The 3.1.1 projection needs the value domain of
  scalars, sequence elements and map values. `BranchShape.Domain` supplies this
  missing owner input, both adapters populate it, and adapter agreement checks
  it.
- **DB-3, location-specific defaults.** For named query maps, `allowReserved`
  alone never switches the retained JSON codec, and Parameter `content` with
  style fields stays rejected. For form fields, any explicit `style`,
  `explode` or `allowReserved`, by presence, selects the style table before
  `contentType`, which is then ignored (§4.15.1.2). With only `explode` or
  `allowReserved` present, the style is `form`. The pseudo-code ordering is
  exact. The model gained `allowReserved` and content-type inputs, three
  invariants and six controls. One control first passed because import read a
  form field with no style fields by the parameter default; the import oracle
  now follows §4.15.1.1.
- **Routine placement.** The expr-type adapter is `expr.HTTPValueShape`,
  because `expr` validation calls `Resolve` and cannot import `transportir`.

## Remaining Policy Choices

None. Raw Bytes text encoding, dynamic-binding semantics and the 3.1.1
projection are settled above. JSON Bytes remain base64 encoded.
