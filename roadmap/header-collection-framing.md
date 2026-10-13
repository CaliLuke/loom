# HTTP collection header framing (#689)

Status: delivered as `7ed1bfc63`, independently reviewed, pushed and confirmed closed.
The user accepted standard comma lists with clear errors, explicit empty presence
and no JSON-header feature on 2026-10-13. Root approved readiness before implementation.

## Sequence

- [x] Reconcile original report and all four encode/decode consumers.
- [x] Preserve #752 evidence and safely reuse its clean worktree at the recorded base.
- [x] Reproduce combined/repeated string and integer collections through existing
  generated-module seams; retain ordinary scalar/single-item controls.
- [x] Record current nil/empty behavior without treating it as policy; add accepted
  contract assertions and authored-default admission regressions.
- [x] User accepted alternative A; arbitrary text lists can use a JSON body.
- [x] Implement the accepted shared framing owner; remove independent framing.
- [x] Validate bounded consumers, exact output differences and independent repeat.
- [x] Independent final review and atomic delivery.
- [ ] Temporary-output cleanup after preserving compact evidence and #887 witness.

## Owner and consumers

At the recorded parent, the framework exposes schema-based header arrays. Generated responses join
logical items with comma-space, whereas request and response decoders treat
physical header lines as items. Requests emit repeated fields. Intermediaries can
combine ordinary list field lines, so changing response Set to Add alone cannot
repair the contract. Strings change cardinality and combined numbers fail parsing.

One HTTP header framing owner should translate logical items to/from physical
field lines; existing scalar conversion and DSL validation remain separate.
Both encoders and decoders must consume it. Record framing in shared HeaderData
only if multiple admitted modes need a choice; do not reuse URL encoding policy.

| Boundary | Existing owner / consumer |
| --- | --- |
| Authored type admission | `dsl/http.go`, `expr/http_endpoint_mapping.go`, `expr/http_response.go`, `expr/http_error.go` admit scalar/primitive arrays |
| Shared transport analysis | `http/codegen/service_data_payload.go:buildHeaders`, `service_data_transport_helpers.go:extractHeaders`, `HeaderData` and `internal/transportir` |
| Request encoder | `template_sources_request.go`: per-item Add, existing scalar formatting |
| Request decoder | `template_sources_request_decoder_elements.go`: raw Header lines, then element conversion, required/default/validation |
| Success/error response encoder | `template_sources_response.go`: shared responseHeaders/header_conversion; defaults also use `server.go:printValue` |
| Success/error response decoder | `template_sources_response_decoder.go`: raw lines then element conversion/default/validation |
| JSON-RPC metadata | Reuses HTTP request encoding/server response encoding; `decoder_sections.go:writeResponseHeaderDecode` replaces client response decoding and must consume the same owner; `handler_sections.go` capture copies encoded metadata |
| Stream opening | Existing HTTP request codec; no stream lifetime/message-body change |
| OpenAPI | `openapi/internal/ir/operation_metadata.go` request schema parameters; `document.go:headersFromIR` response schema headers |
| Import | `internal/openapiimport/analyze_parameters.go` admits simple headers; `analyze.go` rejects content-based response headers |

Cookies share a conversion partial today but have their own DSL/wire semantics;
separate header framing from that partial rather than changing cookies. No HTTP
Trailer DSL or generated trailer mapping was found; gRPC Trailers is a distinct
repeated-metadata owner. Authentication scalar headers remain unchanged.

## Existing guarantees versus unsettled behavior

OpenAPI schema-based arrays use default `simple` comma-list serialization.
`docs/code-generation.md` promises preservation of supported simple headers.
OpenAPI does not authorize automatic percent encoding or automatic quoting for
schema header values. HTTP's field grammar, including quoted syntax, is specific
to each named field. Repeated fields can combine in order; Set-Cookie is an
explicit exception and must not be treated as an ordinary list.

The existing explicit-empty guarantee in `docs/http-guide.md` is scoped to
Body/JSON/JSON-RPC params. It does not settle header presence. The recorded parent
omits empty request arrays (zero Add calls), emits empty response arrays as a
present empty field (Join then Set), and decodes that field as `[""]`. Required
and default checks distinguish nil physical lines from present ones. Therefore
neither current asymmetry nor JSON policy alone authorizes a header change.

## Concrete policy alternatives

### A. Schema simple lists with an explicit admissible item domain (accepted)

Preserve the existing OpenAPI contract. Decode repeated and combined list fields
identically in wire order. Trim separator optional whitespace. In accordance
with RFC9110 section 5.6.1, senders reject empty logical items while receivers
ignore empty list members. An empty present field means a nonnil empty collection;
absence means nil, so defaults apply only to absence and required empty remains
valid unless MinLength rejects it. This presence rule is now an explicitly
accepted header contract, not inferred from JSON.

Reject authored defaults/enums/examples that cannot be represented and reject
outgoing runtime items containing comma, empty text, or leading/trailing SP/HTAB.
Do not silently mutate authored item values. Incoming `a,,b` becomes `["a","b"]`;
incoming `a, b` and separate lines `a` / `b` are equivalent. Ignore reasonable
empty members subject to existing request-size bounds. Quoting does not make an
item with an embedded comma representable under this generic item contract.
Do not claim to parse arbitrary standard-header grammars. For a header with
quoted commas or non-list grammar, use a scalar carrying its complete authored
wire syntax; generic Set-Cookie collection mapping must be rejected from
this generic list mode, with the cookie DSL as its existing owner.

| Logical or wire example | Accepted result |
| --- | --- |
| Outgoing `["a","b"]` | `X-Tags: a, b` |
| Incoming lines `a` / `b, c` | `["a","b","c"]` |
| Outgoing `[1,2]`, incoming `1, 2` | Numeric items round-trip through existing parsers |
| Outgoing nil | Header absent |
| Outgoing `[]`, incoming present empty header | Explicit nonnil `[]` |
| Outgoing `[""]`, `["a,b"]`, `[" a"]` | Error; no lossy coercion |
| Incoming `a,,b,` | `["a","b"]`, tolerant list reception |
| Scalar `Accept: text/plain, application/json` | Unchanged scalar wire value |
| Scalar field containing quoted commas | Unchanged opaque field; no generic list parser |
| `Set-Cookie` array | Outside generic list mode; retain separate cookie DSL |

Quotes in this restricted simple mode are ordinary characters, not a CSV escape
mechanism: incoming `"a,b"` splits into `["\"a","b\""]`, not one logical item.
A standard header whose own grammar gives quoted commas special meaning must
use scalar mapping to retain that complete syntax. Automatic CSV parsing or a
new blanket quote rejection would introduce a third, unsupported policy.

### B. Explicit JSON content for custom collection headers (not adopted)

An explicitly selected custom-header content mode could encode
`X-Tags: ["a,b",""]`, preserving arbitrary item text and `[]`. OpenAPI must then
emit Header/Parameter `content: application/json` rather than a simple schema.
This requires a new public selection contract and importer support; it cannot
silently replace schema simple arrays or standard fields. Standard headers such
as Accept and Set-Cookie are excluded from JSON framing. Repeated JSON field lines
must be rejected or have a separately specified combination rule, since generic
comma-coalescing does not preserve one JSON array document.

This mode is useful for explicitly authored custom protocols, but introduces
surface and adoption work beyond repairing the established simple-list owner.
The user selected A and no JSON-header feature; use a JSON body for arbitrary lists.

References: [OpenAPI Header Object](https://spec.openapis.org/oas/v3.2.0.html#header-object),
[OpenAPI styles](https://spec.openapis.org/oas/v3.2.0.html#style-values),
[RFC9110 field order](https://www.rfc-editor.org/rfc/rfc9110.html#section-5.3),
[RFC9110 list syntax](https://www.rfc-editor.org/rfc/rfc9110.html#section-5.6.1).

## Bounded acceptance

Reuse `renderHTTPModule` and existing header-array fixtures. No new harness
framework or formal model is needed for pure framing. Readiness adds one generated
module with string/integer request and response fields: combined/repeated forms,
generated client/server round trip, single item/scalar controls, explicit
presence and outbound rejection. Readiness also records authored-default admission.
Final acceptance adds malformed/ambiguous input, defaults/required/length checks
and alias conversion. Reuse existing root-array,
view and error header fixtures plus one JSON-RPC response-metadata case where the
consumer differs. Cookies and scalar standard fields are unchanged controls.
Compare affected output at the exact parent/candidate, compile it and repeat
independent generation. Check schema/header contract output with existing OpenAPI
validation and import tests; fixtures without collection headers should be byte
identical. Do not expand into an exhaustive transport corpus.

## Implementation direction

Place list splitting/joining and item admissibility in one small internal HTTP
header-value owner. Expr admission consumes its predicate for authored values;
HTTP runtime adapters add header identity/error context and read/write the actual
Header map, allowing generated external modules to use the owner without importing
Loom internal packages. This is not a new configurable encoding mode or allocator.
Do not copy comma splitting or admissibility into each template. Existing scalar
conversion runs before outgoing framing and after incoming framing; nil arrays
must remain nil through conversion, while nonnil empty arrays remain present.

Expr admission traverses mapped request, success and error headers through one
shared check (including root arrays), following effective named type values. Check
defaults, enumerated values and authored examples at the same boundary; body-only
attributes do not acquire header restrictions. Array Set-Cookie has non-list HTTP
semantics and must not enter generic comma framing; retain scalar and cookie DSL
behavior. No registry of every standard HTTP header grammar is proposed.

Error-returning runtime adapters must fail before committing an outgoing response;
request failure must not call the service. Header names/context belong in errors,
not secret-bearing header contents. Existing decode validation and required/default
logic consume logical presence, not the raw number of physical lines. All default
response array literals must flow through the same owner rather than printValue's
independent Join path. Keep cookie conversion separate and unchanged.

## Readiness evidence

At the recorded parent, the original generated module builds and vets, then
reproduces seven framing failures while repeated/single-item/scalar controls pass.
After recording the accepted policy, the same module also reproduces explicit
empty presence and outbound rejection failures. Six authored-default admission
regressions fail; ordinary and empty defaults remain accepted. Exact source and
logs are recorded in `/tmp/loom-689-readiness-manifest.json`. The first log retains
the pre-policy presence observations; the final tests assert the accepted rule.

## Implementation progress

- Shared `internal/headerlist` framing and HTTP Header adapters implemented.
- Expr checks effective mapped values using detached `initAttr` binding before
  response finalization; HTTP and JSON-RPC codecs consume the same runtime owner.
- Original and accepted contract regressions pass. Added defaults, required/min
  length, non-view alias, plain-array view/error/bodyless and JSON-RPC metadata
  checks; nil/default and empty paths remain explicit.
- A combined named-slice/view specimen exposes an independent pre-existing
  physical binding bug tracked as [#887](https://github.com/CaliLuke/loom/issues/887):
  AttributeContext.Pointer marks a projected slice as a pointer. Preserve `/tmp/loom-689-alias-view-witness` and archived-parent compiler
  evidence separately; do not claim that combined specimen passes framing review.

## Final validation

Four actual CLI specimens compare the archived parent with the candidate: wire,
shapes, JSON-RPC metadata and an ordinary scalar control. All build and vet; the
three contract specimens reproduce parent runtime failures and pass candidate
race checks. Of 63 artifacts per stage, only six transport `encode_decode.go`
files change, using the shared framing owner and logical presence checks. All
OpenAPI artifacts and the ordinary scalar control remain byte-identical. Separate
repeat and final generation processes reproduce every candidate byte. The final
mounted checks also cover rejection without service invocation or response commit.

All 55 generated files from HTTP ticktock, JSON-RPC ticktock and mixedtick remain
byte-identical to their checked-in versions in parent, candidate and repeat runs;
parent and candidate fixture builds and vet pass. Focused final race checks pass
for the internal owner, HTTP runtime, Expr, HTTP codegen and JSON-RPC codegen.
Affected golden groups, focused OpenAPI/import checks, source gates and scoped
static analysis pass. Exact source, inputs, commands and artifact hashes are in
`/tmp/loom-689-final-manifest.json`.

An optional full HTTP/JSON-RPC generator-package sweep was stopped because it
expanded into unrelated generated modules. It is incomplete and is not a passing
gate. No full-corpus or CI-green claim is made. The unchanged OpenAPI outputs do
not require a new schema migration. The combined alias/view compiler failure
remains independently tracked as #887; the separate valid alias and view controls
pass. Temporary outputs are being cleaned after completed review and delivery.
Final production review: `/tmp/loom-689-root-final-verdict.json`; root-authored
guide composition review: `/tmp/loom-689-integration-composition-verdict.json`.
The integrated Expr/HTTP/JSON-RPC focused race checks pass (1.621/14.157/12.071s).
