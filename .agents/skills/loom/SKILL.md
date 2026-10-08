---
name: loom
description: Use Loom from a consuming Go service. Covers authoring the design DSL, running loom gen, implementing services outside gen, and adopting generated HTTP, gRPC, JSON-RPC, OpenAPI, auth, streaming, and observability contracts. Do not use this skill to modify the Loom framework repository itself.
---

# Use Loom

Use this skill when an application consumes `github.com/CaliLuke/loom`: writing
or changing `design/*.go`, regenerating `gen/`, implementing service methods,
or wiring generated transports and runtime packages.

This is deliberately not a framework-maintenance guide. If the task changes
Loom's DSL implementation, expression model, generators, transports, OpenAPI
renderer, or framework tests, use the `loom-framework` skill.

## Core Workflow

1. Treat `design/*.go` as the source of truth.
2. Put validation, transport mappings, security, errors, and public contract
   metadata in the design.
3. Run `loom gen <module-import-path>/design` after every design change.
4. Implement business logic outside `gen/`.
5. Run `loom example <module-import-path>/design` only when scaffolding missing
   starter files. Each service stub returns `loom.Fault` until you implement
   it. The command does not overwrite existing `cmd/` files.
6. Run the consuming repository's formatting, tests, and integration checks.

Never edit `gen/` directly. `loom gen` deletes and recreates it transactionally,
so manual changes are both temporary and misleading.

Use Go import paths, not filesystem paths:

```bash
loom gen example.com/myapi/design
```

HTTP generation includes servers, per-service clients, and the aggregate
client CLI by default. For an HTTP server-only application, set API metadata
`Meta("http:generate", "server")`. Loom keeps generated service and HTTP server
packages and removes stale `gen/http/*/client/` and `gen/http/cli/` directories.
The gRPC client CLI takes the request message in `--message` as protocol
buffer JSON (`protojson`): set a `oneof` by the name of its selected `oneof`
field, not by the union attribute name.
Generated gRPC clients preserve local encoding and validation errors. Inspect
`grpc.ClientError` and `loom.ServiceError` with `errors.As`; remote failures use
the transport error mappings. Generic remote service errors retain the RPC cause
for `errors.Is`, `errors.As`, and `status.Code`. A remote cancellation alone does
not imply local context cancellation. Native remote errors without recognized
error details remain unchanged, including cancellation/deadline statuses. Service
context termination maps to its native gRPC status. Regenerate clients to adopt
this behavior.
HTTP and JSON-RPC decoders close consumed response bodies once and retain
cleanup errors with read/decode errors; use `errors.Is` to inspect each cause.
Body restoration is bounded and preserves captured bytes. Successful raw/file
responses stay open and caller-owned even with restoration enabled; close them
when done. SSE handshake failures use the same cleanup owner; successful
streams retain their open bodies, and JSON-RPC error-body diagnostics are bounded.
Regenerate clients to adopt this behavior.
Synthesized service and HTTP CLI examples are scoped by stable service and
method identity, so unrelated service edits or declaration reordering do not
churn their values. Implicit server service lists use stable service-name
ordering; an explicit `Server(... Services(...))` list preserves its authored
order.

HTTP and JSON-RPC CLI flags decoded as JSON retain the selected example's values
and union branch in the decoder's JSON representation, including base64 for
`Bytes`. Plain text flags retain their text syntax. An unavailable example omits
the hint; if a required flag has no usable example, Loom omits the sample
invocation while keeping the command available. Explicit JSON `null` remains
a valid example where the declared type permits it. JSON decoding failures in
generated CLI payload builders preserve their underlying cause for `errors.Is`
and `errors.As`, including when the message contains an example.

## Design Rules

- Prefer concrete types over `Any`, especially when gRPC generation matters.
- Put lengths, enums, formats, requiredness, and other validation in the DSL;
  do not duplicate it in service implementations.
- Constraints on a derived named type refine its base. A derived enum must be
  equal to or a subset of the ancestor enum. Bounds and required fields
  accumulate, pattern and format rules are conjoined, and an inherited default
  must satisfy the refined contract. A declared local default always replaces
  it and must be valid or design validation fails. For example, base `^a` plus
  derived `b$` admits `ab` and rejects `xb`.
- `Reference` copies selected object field templates. It does not create named
  type ancestry, and the copied fields keep their declared constraints.
- Use `FormatURI` for absolute URI contracts that include a scheme. Use
  `FormatURIReference` for relative paths and other URI references.
- `FormatHostname` accepts ASCII labels of 1–63 bytes, letters/digits at the
  edges and interior hyphens, up to 253 bytes excluding an optional terminal
  root dot. It validates syntax without DNS lookup or Unicode/IDNA conversion.
- Non-null array elements and map values reject JSON null during decoding,
  including viewed responses. HTTP clients report `decoding_error` with a JSON
  semantic cause; HTTP requests return 400 `decode_payload`. Declare `Nullable()` on
  elements when null is valid. Missing fields still follow the selected view.
- Do not rely on nil versus empty slices or maps to encode presence. Generated
  JSON uses `omitempty`, so both serialize as missing.
- For each non-`Extend` type, payload, or result, start literal field tags at
  `1` and increment within that definition. For definitions using `Extend`,
  start newly introduced fields at `100`.
- Give each object distinct generated Go field names. `foo_bar` and `fooBar`
  both become `FooBar`; rename one or set `Meta("struct:field:name", "OtherFooBar")`
  on it. The override preserves the design name and transport mapping.
- Error fields must also avoid generated methods `Error`, `LoomErrorName`, and
  `LoomErrorRemedy` when a `Remedy` is declared. Use `struct:field:name` to keep
  the wire name while choosing a different Go field name.
- Prefer a canonical `ResultType` with `View(...)` definitions over parallel
  hand-maintained DTOs for alternate public representations.
- Return a nonnil object satisfying the selected view's constraints for a
  successful viewed result. After regeneration, an unknown service-selected view
  or an invalid selected result returns a server fault, wrapping the validation
  cause. Excluded fields are not validated. Nil and empty result collections
  remain valid empty collections; non-nullable nil elements are rejected.
  Stream senders return the fault before emitting the invalid result.
- A type placed with `Meta("struct:pkg:path", "types")` keeps its Go name
  independently of service methods. A `Moved` type remains `types.Moved` even
  when a consuming service declares a `moved` method, including empty objects
  and aliases of primitives, arrays, maps or unions.
- Repeated `HTTP`, `GRPC`, or `JSONRPC` blocks in the same API, service, or
  method scope compose in declaration order. Use this to keep transport
  mappings near the errors or methods they describe; ordinary duplicate and
  conflict rules still apply to the combined contents.
- gRPC `Message`, `Metadata`, `Headers`, and `Trailers` select fields by
  attribute name, ignoring a service field's JSON suffix. For a field
  `"tok:t"`, use `Metadata(func() { Attribute("tok:x-token") })` to send
  `x-token` metadata; requiredness and defaults come from `"tok:t"`.
- Design files conventionally dot-import `github.com/CaliLuke/loom/dsl`.
  Top-level Go identifiers must not reuse exported DSL names. Examples include
  `Fault`, `Error`, `Result`, and `Type`. Use an application-specific variable
  name. The declaration string can keep the public name.
- Mount designed routes through generated server packages. Do not call
  `Handle` directly on a Loom mux for application endpoints.

## OpenAPI Contracts

Loom emits OpenAPI 3.2.0 by default at:

- `gen/http/openapi.json`
- `gen/http/openapi.yaml`

Both files are generated by default. Set API metadata
`Meta("openapi:output", "json")` or `Meta("openapi:output", "yaml")` to keep
only one format. Generation removes the stale sibling file when this setting
changes. JSON output is deterministically ordered, two-space indented, and
newline-terminated for reviewable diffs.

Automatic request-body, response, parameter and header components share complete
definitions only when their contents, including examples, agree. Changing an
example can move a definition inline without changing its underlying schema.
Do not depend on these automatic component references as stable identifiers.
Explicitly authored component names and schema names retain their contracts.
Review regenerated SDK and validator output when updating Loom; equivalent
OpenAPI schemas can still produce different client declarations. See the
[OpenAPI evolution guide](../../../docs/openapi-evolution.md).

Schema examples use JSON representations, including base64 for `Bytes`, in
both JSON and YAML, including byte fields inside untagged union branches.
Author text or byte values as usual; examples encode them once. Incomplete
object examples are omitted; explicit null examples are retained for nullable
schemas. These rules also apply to nested
parameter schemas, response-header schemas, and streaming message schemas.
Synthesized tagged-union examples retain every nested envelope, including when
branches share the same shape. Configured discriminator/value keys and branch
tags apply at each level; authored examples keep their existing interpretation.

For built-in JSON, `Bytes` schemas describe padded base64 strings, while
`MinLength` and `MaxLength` count decoded bytes. Generated constraints enforce
the grammar and decoded bounds; `contentEncoding` alone is only an annotation.
Raw/text bodies, multipart/form fields, custom codecs and explicit schema
overrides retain their contracts. See the [HTTP guide](../../../docs/http-guide.md).

OpenAPI security follows each endpoint's credential location. JWTs mapped to
query parameters, cookies, or custom headers use API-key schemes; Authorization
header JWTs use HTTP bearer. One scheme used at several locations gets separate
components. Path credentials and OAuth2 credentials outside Authorization cause
OpenAPI generation errors. Keep OAuth2 flows and scopes by using Authorization,
or exclude the endpoint with `Meta("openapi:generate", "false")` and document it
separately; the HTTP transport mappings remain supported.

To bootstrap a design from an existing OpenAPI 3.0, 3.1, or 3.2 JSON/YAML contract,
run `loom import openapi <input> -o design`. Import supports a strict subset:
it reports every unsupported construct it finds, writes no partial design or
TODO placeholders, and never overwrites an existing target. Schema `title`,
`default`, `example`, `examples`, `deprecated`, `readOnly`, and `writeOnly`
import without a flag. Unformatted integers and numbers also import without a
flag. String schemas with `format: byte` or `format: binary` import as `Bytes`.
Regeneration preserves the selected format.
Representable media examples map to `Example(...)`.
Use `docs/openapi-import-coverage.md` for the complete field and schema-keyword
matrix. It distinguishes preserved, conditional, lossy, and rejected input.

Path parameters retain their authored attribute names so route placeholders
and `Param(...)` mappings stay identical, including snake_case names. Generated
Go fields remain idiomatic. An operation uses its single 2xx response as the
primary success, or a single 3xx when no 2xx exists, so redirect-only contracts
do not need a synthetic 2xx.

An unconstrained OpenAPI schema `{}` imports as `Any` in each schema
location, including a named component. Regeneration preserves the empty schema.
Generated HTTP code accepts all JSON values. A direct `Any` uses
`loom.JSONValue`; arrays and maps use `[]loom.JSONValue` and
`map[string]loom.JSONValue`. This type preserves source JSON numbers without a
`float64` conversion. Decode stored JSON directly into a generated result.
To construct a field, use
`loom.NullableValue(loom.JSONValue(rawValue))`. Do not replace a typeless
schema that has constraints with `Any`. The importer rejects this contract
because `Any` discards its assertions. The equivalent
`anyOf: [{}, {type: "null"}]` form also imports as `Any`.

A two-member `oneOf` with a bare `null` branch and a local `$ref` to a schema
that explicitly excludes null imports as a nullable named type. Regeneration
uses the equivalent nullable `anyOf` form. A JSON request or success response
body `oneOf` with named object or array branches imports as an untagged typed
union. Recursive contents may be primitives, named objects, and arrays of these
shapes. Generated Go retains constructors and accessors; JSON uses the bare
selected array or object. Encoding and decoding require the same unique schema
and Go branch. Inline object branches become deterministic components; inline
array branches, inline nested objects, maps, nested unions, scalar unions, and
untagged unions in string-encoded transport locations remain blocked.

A scalar JSON Schema `const` without a sibling `enum` imports as the equivalent
one-value `Enum(...)` validation. Regenerated OpenAPI uses `enum` because Loom's
DSL has no separate scalar-constant construct. Structured constants and schemas
that combine `const` with `enum` remain blocked.

An OpenAPI free-form object with `type: object` and
`additionalProperties: true` imports as `MapOf(String, Any)`. Generated Go
uses `map[string]loom.JSONValue`, and regenerated OpenAPI preserves the object
boundary.
The importer rejects a schema that also declares object members because the
Loom DSL cannot preserve both shapes.

An OpenAPI form body with only schema-valued `additionalProperties` imports as
a typed map. Body-only methods keep the map payload and use
`OptionalRequestBody()` when the request body is optional. If transport fields
must coexist with the map, the importer selects a collision-free map attribute
with `Body(...)` and `FormRequest()`. Multi-media map bodies and optional
multipart maps remain raw request streams so the application owns codec and
content-negotiation policy. Optional form objects with required members also
remain raw so omission does not trigger nested required-field validation. Form
schemas with unconstrained values, including free-form maps, also remain raw
because typed form decoding cannot reconstruct JSON-valued data.

A one-member `allOf` containing a local schema reference imports losslessly,
including numeric bounds and compatible scalar defaults on the wrapper.
Regenerated OpenAPI retains the component reference in `allOf` and those
sibling constraints on the occurrence. Other contract siblings remain
blocked.
With `--allow-lossy`, `allOf: [$ref, inline object]` renders with `Extend(...)`;
the regenerated schema is flattened. Inline object array items are promoted
to a deterministic component under the same flag so their fields and
validation remain renderable. Both structural changes are reported as lossy
warnings. Unsupported `allOf`, `oneOf`, `anyOf`, and `not` shapes remain blocked.
A Schema Object with `$ref` siblings is also blocked. Wrap the reference in a
supported `allOf` shape so the importer does not discard sibling constraints.

OpenAPI 3.2 Media Type `description` requires `--allow-lossy`.
`prefixEncoding` remains a strict import error.

If you explicitly accept omitting non-contract
metadata, the documented `allOf` flattening and inline array-item promotion,
unrecognized `format` values, or a parameter/header (not schema) `deprecated`
flag the HTTP DSL cannot express per-parameter, add
`--allow-lossy`; it warns on stderr for each omission but still refuses any
contract-affecting loss. Use `--skip-unrenderable` to retain renderable
operations. It reports skipped operations and omitted root members separately.
These root members include servers and unsupported info metadata. Tag metadata,
response summaries, supported serialization, structured examples, and reusable
examples import losslessly. API-key, HTTP basic, HTTP bearer, and supported
OAuth 2.0 schemes also import losslessly. OAuth 2.0 flows must use the same
scope map. Root and operation security requirements retain OAuth scopes and
AND or alternative semantics. An anonymous `{}`
alternative renders as `Security()` and an explicit operation `security: []`
renders as `NoSecurity()`. Other scheme kinds, bearer formats, incompatible
OAuth flow shapes, and invalid scope values fail closed. Exit `3` means partial output was written, exit `2`
means nothing was importable, and exit `1` is a command failure. Review the new
`design/design.go` before running `loom gen`.

The importer rejects a 3.2-only field when the source declares OpenAPI 3.0 or
3.1. Change the source version only when the complete contract is valid for 3.2.

Set API metadata `Meta("openapi:version", "3.1")` only when a downstream
consumer still requires OpenAPI 3.1.1. Selected output paths remain canonical
and the compatible surrounding contract is preserved.

Imported error responses preserve their authored descriptions verbatim. Loom
normally prefixes error descriptions with the error name; for hand-authored
designs, use `Meta("openapi:description:errorName", "false")` inside an HTTP
error `Response(...)` only when an external contract requires exact wording.
An imported schema-less error stays bodyless and gains no response headers.

For TypeScript clients, use the endorsed `@hey-api/openapi-ts` workflow in
`docs/typescript-clients.md`. Use the 3.1 compatibility target, pin the external
generator, and keep its entire output directory generator-owned.

For published OpenAPI compatibility, use the endorsed oasdiff workflow in
`docs/openapi-evolution.md`. Regenerate before comparison, block `ERR`, review
`WARN`, and keep policy files consumer-owned. Use the 3.1 compatibility target
until the project audits the required oasdiff 3.2 coverage.

OpenAPI 3.2 capabilities available from the DSL include:

- tag summaries, hierarchy, and kind
- QUERY and extension HTTP methods
- whole-query-string parameters
- `itemSchema` for sequential and streaming media
- reusable media types and nested encodings
- structured examples
- device authorization OAuth flows and OAuth metadata
- URI security schemes
- response summaries with optional descriptions
- XML node types
- optional discriminators and default mappings
- server names and `$self` document identity
- `allowReserved` for parameters and headers, plus cookie style

Use the metadata table and examples in `docs/dsl-reference.md` instead of
patching generated OpenAPI.

Other important OpenAPI usage rules:

- Use `Meta("openapi:typename", "...")` when a public schema component needs a
  stable explicit name.
- Use `Title(...)` inside a type or attribute block when a schema node needs a
  human-readable OpenAPI title distinct from its Loom name.
- Treat hash-suffixed fallback names as generated identities. Use explicit
  component-name metadata when downstream code requires a stable public name.
- Use `Meta("openapi:component:requestBody", "...")`,
  `Meta("openapi:component:parameter", "...")`,
  `Meta("openapi:component:response", "...")`, and
  `Meta("openapi:component:example", "...")` for stable reusable component
  names.
- Synthesized examples are deterministic in both OpenAPI outputs. Set
  `Meta("openapi:example", "false")` at API scope to omit synthesized examples
  while retaining explicit `Example(...)` values.
- Model workflow links with `Link(...)`, `LinkOperation(...)`,
  `LinkOperationRef(...)`, `LinkParam(...)`, and `LinkRequestBody(...)`.
- Use `Meta("openapi:readOnly", ...)` and
  `Meta("openapi:writeOnly", ...)` so request and response schemas split
  correctly when one domain type serves both directions.
- Use API metadata `Meta("openapi:closed-objects", "true")` when consumers need
  strict object contracts.
- Add vendor extensions with `Meta("openapi:extension:x-name", "<json>")`. At API
  scope that key lands on the `info` object; use
  `Meta("openapi:document:extension:x-name", "<json>")` for the document root.
  When one Loom attribute feeds two OpenAPI objects, name the target scope:
  `openapi:schema:extension:`, `openapi:parameter:extension:`,
  `openapi:requestBody:extension:`, `openapi:mediaType:extension:`,
  `openapi:response:extension:`, and `openapi:header:extension:`.
- Unreferenced component schemas are intentionally omitted.

## Errors and Remediation

Regenerate after upgrading the length-error API. `loom.InvalidLengthError`
accepts `(name, actualLength, bound, minimum)` and never the rejected value.
Length diagnostics retain field and bound information without echoing contents.

Always retain the return value of `loom.MergeErrors`: two nonnil operands
produce an independent `ServiceError`; neither input is updated. Nil returns
the other operand unchanged. `History()` returns detached original
contributions, including copied field/remedy metadata. Cause objects keep their
identity for `errors.Is` and `errors.As`. Use `errors.Join` for independent
failures rather than assigning one validation contribution's contract to all.
For gRPC joins, only a unanimous branch status is retained; conflicting codes
become `Unknown`. Details describe the whole failure without promoting one
branch's custom type or retry traits. To define an aggregate status or retry
policy, return an explicit outer service or designed error contract.

Loom's default HTTP errors are RFC 9457-style
`application/problem+json` documents with a stable `code` field.
After routes are mounted, the default muxer's unmatched-path response carries
HTTP/body status `404` and code `not_found` in JSON, XML, and Gob. That routing
fallback retains `ResponseEncoder` negotiation (JSON by default). Text formats
receive status `404` and the public message `404 page not found`.

- Use `ProblemResult` when explicitly modeling the same public document shape.
- Use `ProblemType(...)` and `ProblemTitle(...)` for public error overrides.
- Use `AuthErrorResponses()` for standard 401/403 mappings. Inherited API HTTP
  mappings retain their custom error types; return the generated custom type
  to use that mapping. Unused API error definitions are not generated.
- Use `Remedy(...)` for structured remediation metadata. Put
  `RemedyCode(...)`, `SafeMessage(...)`, and `RetryHint(...)` inside its
  callback. See the structured-remediation example in
  `docs/error-handling.md`.
- An error must have a concrete type. For a bodyless HTTP error, omit the
  explicit type and map `Body(Empty)` on its response.
- For a shared custom error type, use `ErrorName` for routing. Map that field
  with `Header("field:loom-error")` on each HTTP error response when routing
  metadata must stay out of the response body and OpenAPI schema.

Do not duplicate these contracts in handwritten transport code.

Generated HTTP server packages expose per-method response contract functions
and a service-wide `ResponseContractCases()` manifest. Loom supports unary,
non-streaming flat multipart, server SSE, and plain HTTP WebSocket endpoints. After `loom gen`, run
`loom test-scaffold <design-package>` once to create non-overwriting provider
tests under `internal/contracttest/`. Fill each callback with a real
generated-transport request. Unary callbacks return the response. Multipart
callbacks receive the designed content type, parts, media types, and
requiredness. The application owns its multipart codecs and fixtures. SSE
callbacks return an `loomhttp.SSEResponseContractObservation` containing the
handshake, parsed events, and terminal error. Declared pre-stream errors remain
unary cases. WebSocket callbacks receive the stream contract. They return an
`loomhttp.WebSocketResponseContractObservation` with the handshake, outbound
JSON messages, and terminal error. Declared pre-upgrade errors remain unary
cases. Missing callbacks fail individually. The matching Loom validator checks
implemented callbacks. The application remains responsible for payloads, fakes,
and state that make every declared response reachable.

Generated gRPC server packages also expose per-method and service-wide
`ResponseContractCases()` manifests. The manifest covers unary and
server-streaming success messages, declared status codes, typed status details,
and required headers and trailers. Run `loom test-scaffold` once, then make each
callback call the generated gRPC client and return a
`loomgrpc.ResponseContractObservation`. Server streams must end with clean EOF.
Client-streaming and bidirectional completion contracts are reported as
unsupported generation diagnostics.

Generated JSON-RPC server packages expose the same per-method and service-wide
manifest pattern. Cases cover success results, declared error codes, typed
error data, and response suppression for ID-less notifications. Call each real
generated handler from the scaffold and return a
`jsonrpc.ResponseContractObservation`. Server-SSE calls with an ID end with a
final response. ID-less streams suppress that response. Other streaming
completion shapes are explicit generation limitations.

## JSON Presence and Nullability

- Loom and generated transports use Go 1.27's `encoding/json/v2`. Decoding is
  case-sensitive and rejects duplicate object names and invalid UTF-8.
  Generated `Any` fields use `loom.JSONValue`, an alias for
  `encoding/json/jsontext.Value`. Do not use the legacy raw-message type.
- Treat requiredness and nullability as independent. `Required("field")`
  requires presence; `Nullable()` allows a present field to contain JSON null.
- Generated nullable fields use `loom.Nullable[T]`. Its zero value is absent;
  use `loom.NullValue[T]()` for explicit null and `loom.NullableValue(value)`
  for a concrete value. Inspect it with `Present`, `IsNull`, and `Value`.
- Use `Example("name", Null())` to author an explicit null example. `Null()`
  is not valid as a default. Concrete defaults apply only to absent inputs.
- `Nullable()` is supported for JSON HTTP bodies, JSON-RPC values, array
  elements, and map values. Do not use it for map keys, string-encoded HTTP
  parameters or metadata, forms, multipart bodies, or gRPC message fields.
- Array elements reject JSON null by default. Put `Nullable()` on the element
  definition to accept null members; array-field requiredness is independent.
- HTTP WebSocket `StreamingPayload` roots cannot be nullable: a root JSON `null`
  frame ends client input. Put nullable values inside a non-null payload object.
  Nested nullable fields and collection members remain supported; this request
  framing restriction does not apply to JSON-RPC or WebSocket responses.
- OpenAPI 3.1/3.2 value-or-null unions and OpenAPI 3.0 `nullable: true` import
  to this same DSL contract.

## Unions, Views, and Projections

- `OneOf(...)` works as both a named union declaration and a type constructor.
- Block and branch names can repeat in different objects. Loom keeps their
  types distinct and suffixes generated Go names without changing wire tags.
- Named union branches can refer to types declared later. Recursive unions
  must pass through an object field; union branch cycles, including cycles
  through arrays or maps, are rejected during design validation.
- For gRPC, `Field(n, "name", OneOf(A, B))` gives the branches the numbers n
  and n+1, so leave those numbers free. Use the block form with a `Field` for
  each branch when the branches can be reordered or removed. A named union
  `Type("U", OneOf(A, B))` passed to `Field(n, ...)` numbers its branches the
  same way.
- On gRPC, the fields, `oneof` names and `oneof` fields of a message share one
  namespace. A field that is not a union normally keeps its name. A `oneof` that
  collides takes `_oneof` suffixes, and a branch that collides with an earlier
  name takes the union field name as a prefix, such as `b_int64`. Protocol
  buffer clients see the prefixed names; the service type keeps the branch
  names. Generated protobuf Go selectors can also take underscores to avoid
  methods and getters; wrapper type names can differ from their fields. Use
  the emitted Go declarations when accessing protobuf values directly. Loom
  adds further `_oneof` suffixes if protoc would emit conflicting oneof
  getters, and `_field` suffixes for a field that would clash with
  `ProtoReflect`. Service attribute names and field numbers stay unchanged.
- gRPC transport files alias service imports that collide with framework
  imports, such as `protojson` or `strconv`. Keep the service name; its package
  path and protocol buffer names do not change.
- `struct:name:proto` requires exactly one name and applies to every message
  generated for a named type,
  including nested references and scalar, collection and union wrappers. Scalar
  fields and inlined union fields still produce no separate message. Names are
  authoritative: conflicting shapes or protobuf-to-Go names fail generation,
  with no fallback names or compatibility mode. Shared messages require the
  same fields, numbers, requiredness and referenced message names. Use distinct
  explicit names for distinct contracts. Regenerate clients and servers and
  update direct protobuf references, descriptors and `Any` type URLs after
  migrating from the former root-only naming behavior.
- gRPC scalar message fields retain explicit presence, including required
  fields and scalar payload/result wrappers. Omitted required values fail in
  requests, responses, and stream items, even when the field has a default.
  Explicit zero values are accepted subject to constraints. Defaults fill
  absent optional fields; encoding preserves the supplied service value.
  Regenerate both clients and servers. Direct protobuf Go callers must set
  scalar pointers (for example `proto.String("")`); old clients omit required
  zeros and cannot satisfy this contract. Bytes remain slices and scalar
  oneof branches retain their wrapper-based presence.
- gRPC map keys must be `Boolean`, `String`, or integer types, including
  aliases. `Any` is supported as a map value, but cannot be a map key.
- gRPC rejects a union used as an array element or map value. Wrap the
  union in a type with one `Field` and use that type as the element or value.
  Wrapping a union does not make it a valid map key.
- On gRPC, a union used as a branch of another union, a named array such as
  `Type("Tags", ArrayOf(String))` and a named map such as
  `Type("Index", MapOf(String, Int))` are each a message that wraps the
  `oneof`, the repeated `field` or the map `field`. Named maps and arrays
  can be union branches; `OneOf` blocks name inline collections automatically.
  A selected nil or empty map keeps its branch through protobuf serialization;
  nil and empty map entries are equivalent on the wire.
- Add `Untagged()` in the union attribute, payload, or result block only when
  JSON must encode a concrete named object or array branch directly. Both encoding
  and decoding require the same unique schema and Go branch; overlapping schemas
  fail even when Go numeric widths distinguish them. Nil slices encode as `[]`,
  which is ambiguous if multiple array branches admit empty arrays. Put nullability
  on the enclosing union, not a branch. Opaque codecs and nested maps are rejected.
- Explicit discriminator tags control wire values independently of schema and
  Go type names.
- Optional object unions generate as pointers; required unions remain values.
- Missing and explicit JSON `null` are both rejected for required unions.
- Result views inherit canonical requiredness. Use `ViewRequired(...)` and
  `ViewOptional(...)` for deliberate overrides. Method-specific `Required(...)`
  customization applies to included view fields while explicit view overrides
  take precedence; the original result type and other methods are unchanged.
- Generated projection helpers convert between canonical results and view
  types. Use them instead of maintaining app-local conversion copies.
- `Extend(...)` inside a named `Payload`, `Result`, `StreamingPayload`, or
  `StreamingResult` customization creates a method-specific type with the
  inherited fields and requiredness. Plain uses of the named type are unchanged.
  Automatic result views include inherited fields; explicit views keep their
  authored field selection.
- For typed SSE projections, use `SSEProjection(eventType, view)` with
  `SSEEventType(...)`.

## HTTP Bodies and Parameters

HTTP methods require one untagged application response. Use `Tag` on every
additional response; multiple untagged declarations are rejected instead of
silently ignored. Designed errors and FileResponse protocol statuses remain
separate. Update ambiguous designs before regenerating. Tagged responses retain
authored first-match priority; default placement never changes it. Regenerate
servers to adopt this ordering correction.

- A method may share its name with a type nested in its object or union body.
  Loom allocates distinct transport type names, including WebSocket bodies;
  a numeric suffix on a generated body type does not change schema names or
  the wire format.
- An explicit `Body(func() { ... })` inherits requiredness from both inline
  and named payloads for the fields it lists. A body `Required(...)` must agree
  with payload requiredness; parameter-only fields stay outside the body.
- Generated service payloads use `*string` for optional string query fields.
  A nil pointer means omitted. A nonnil pointer preserves an empty or nonempty
  value. Map the field with `Param(...)`. No additional DSL is necessary.
  Generated clients emit the key for each nonnil pointer. Defaults apply only
  when the key is absent.
- Use `FormRequest()` for typed object, map, or constructor-union
  `application/x-www-form-urlencoded` payloads. Put a map in an object payload
  and select it with `Body(...)` when other transport fields must coexist.
- Use `MultipartRequest()` for supported multipart object payloads.
- Use `OptionalRequestBody()` for optional JSON object bodies and optional
  typed form object/map bodies. An object or union selected with `Body("name")`
  from an optional payload attribute is optional either way: a nil attribute
  sends no body, and an empty body decodes to a nil attribute. So is an
  optional primitive, array, map, or `Bytes` attribute, except a primitive
  with a default value, which is always sent. For an optional non-nullable defaulted
  primitive selected with `Body`, an absent body or JSON-RPC `params` and an
  omitted CLI body flag use the declared default. Explicit values, including
  zero, false, and empty strings, override it. A required body or CLI flag
  must still be present. A nullable or
  `Any` attribute selected with `Body("name")` keeps absent, null, and concrete
  states: an absent optional value sends no body, and an empty body decodes to
  an absent value. Non-nullable JSON bodies reject root `null` with
  `decode_payload`, whether required or optional; JSON-RPC returns `-32602`.
- Omitted optional collection CLI flags use their declared defaults, including
  array and map attributes selected with `Body`. Explicit empty collections
  override the default. Collection flags accept JSON; boolean map keys must be
  the member names `"true"` or `"false"`, including in nested collections.
- HTTP JSON bodies, generated union codecs, and optional/nullable wrappers use
  the same boolean map-key rules. Boolean values keep JSON boolean syntax and
  custom JSON/text codecs keep their representation. Custom JSON integrations
  can pass `loom.JSONOptions()` to the JSON v2 encoder or decoder; deterministic
  ordering still requires `json.Deterministic(true)`.
- Use `OpenAPIRequestBody(...)` with `SkipRequestBodyEncodeDecode()` when a raw
  request stream needs a documentation-only OpenAPI contract.
- Use `OpenAPIRequestBodyTypes(...)` when one raw request schema accepts
  multiple media types. Inspect the content type and decode the stream in the
  service.
- Use `SkipResponseBodyEncodeDecode()` when the service returns a raw response
  stream. Declare `ContentType(...)` on the response. The generated server
  applies that static media type before it streams the body. Use
  `OpenAPIBody(...)` only to document the schema.
- String-backed path, query, header, and cookie fields with
  `Meta("struct:field:type", ...)` decode through `encoding.TextUnmarshaler`.
  CLI flags accept raw text through the same parser; omitted optional fields
  remain absent and declared defaults apply. Implement `fmt.Stringer` on the
  custom type to supply the wire text used by generated HTTP clients.
- Let generated clients and routers handle path escaping exactly once. Do not
  add app-local `url.PathEscape` or `url.PathUnescape` layers. Generated path
  builders return escaped paths: a value such as `a/b` stays in its segment,
  and a catch-all value keeps its `/` separators. Build a URL from a path
  builder result with `loomhttp.RequestURL`, not `url.URL{Path: ...}`, which
  escapes it twice.
- Custom `loomhttp.Muxer` implementations must provide `RawVars` alongside
  decoded `Vars`. Preserve escaped captures directly from the router so array
  elements containing commas remain distinct from comma separators. Do not
  reconstruct raw values by escaping decoded ones. The built-in `NewMuxer`
  already implements both methods; regenerate servers to adopt array decoding
  that preserves escaped commas.
- Percent-encode reserved query delimiters, such as `;`, in manually built
  URLs. Generated clients encode them. Generated servers return a
  `decode_payload` response with status 400 for malformed query strings.
  Repeated query keys retain their order.
- Prefer modeled response cookies over raw `Set-Cookie` header bags.
- When deployment configuration owns modeled response-cookie attributes, wrap
  the generated method with `loomhttp.ResponseCookiePolicy.Handler`. The policy
  may set `Domain`, `Secure`, and `Expires` but may not change the designed
  name, result-derived value, path, lifetime, or browser security attributes.
- For a deployment-specific request limit, construct
  `loomhttp.NewRequestBodyPolicy` at startup and apply its `Handler` to the
  generated method field. Generated JSON, text, form, and multipart decoders
  preserve the standard `request_too_large` 413; raw-body services receive the
  same service error while reading.
- Use `loomhttp.NewResponseNegotiationPolicy` on a generated method when an
  incompatible `Accept` header must produce a 406 instead of the default JSON
  fallback.
- Mount an ordinary unary GET companion with `loomhttp.MountDerivedHead`.
  Keep explicit designed `HEAD` routes for `FileResponse`, and do not derive
  HEAD for streaming methods.
- Use `FileResponse()` for seekable HTTP downloads that need range and
  conditional request handling. Implement the generated service method with a
  `*loomhttp.FileResponse`, set `Name`, `ModTime`, and `Content`, and declare
  `HEAD(...)` explicitly when needed. The generated client returns an
  `io.ReadCloser`; the caller must close it. Map every modeled result metadata
  attribute to a response header or cookie because FileResponse owns the body.

If a body shape is unsupported, use the documented custom encoder/decoder seam
rather than modifying generated files.

For generated JSON-RPC servers, route requests through `Server.ServeHTTP`.
Middleware installed with `Server.Use` is honored whether installed before or
after `Mount`. Complete configuration before requests start; do not change the
chain concurrently with requests. Regenerate clients and servers when adopting
this corrected dispatch behavior.

## Authentication and Sessions

Declare credentials with plain attribute names: `Token("token", String)`,
then map them in the transport DSL, such as `Header("token:Authorization")`.
Mapping suffixes on `Token`, `AccessToken`, `APIKey`, `Username`, `Password`,
or their numbered `Field` declarations are rejected during validation.
Use `Meta("struct:field:name", "AuthToken")` on a credential to rename its Go
field; authentication and transport generation use that same field. Transport
header/query names remain independently authored. Regenerate after upgrading
to pick up corrected credential selectors and named-string conversions.
Credentials may use `String` or a named string type. Their payload fields retain
that type; authentication callbacks receive strings. Non-string, nullable and
`struct:field:type` credential overrides are rejected during DSL validation.
Basic-auth CLI values are plain strings, including for named credential types.

Basic-auth header presence is independent of its two components. The client
sends the header if either component is required or present; an absent optional
counterpart becomes an empty component. When both are optional, a missing or
invalid Basic header leaves their pointers nil. A valid header supplies both
components, including empty values. If either is required, a missing or invalid
header fails decoding. Regenerate clients and servers to adopt this behavior.


Prefer Loom's first-class session DSL:

- `SessionAuth(name, fn)`
- `BearerTransport(scheme, fieldName, fn...)`
- `CookieTransport(scheme, fieldName, fn...)`
- `CookieName(name)`
- `SessionSecurity(contract)`
- `SessionCookie(...)`
- `CookieInsecure()` for plain-HTTP local development only

`CookieTransport(scheme, "", fn...)` is the transport-owned browser-cookie
mode. It emits the security contract without synthesizing a payload field or
CLI flag, allowing the application to resolve the cookie from request metadata.

`SessionCookie(...)` remains secure by default. A response may call
`CookieInsecure()` immediately afterward for plain-HTTP local development, but
must not do so in production or for `__Host-`/`__Secure-` names or
`SameSite=None` cookies.

Alternative security requirements are isolated. Context returned by a failed
alternative does not leak into the next one; schemes within one successful
requirement retain AND semantics.

## Application Authorization

- Declare stable typed requirements with `Authorization("document.edit", DocumentRef)`;
  input is a named object or `Empty` for context-only decisions.
- Apply with `Authorize(requirement, func() { Bind("id", "document_id") })`.
  Bind every input field to a typed payload path. Named types retain identity;
  nullable access paths and nullable or unconstrained input fields are rejected.
- Enable `StrictAuthorization()` on an API or service to reject unclassified
  methods. `NoAccessCheck("reason")` preserves authentication; pair it with
  `NoSecurity()` only for intentionally anonymous methods.
- Use `AuthorizeBy("value", ...)` and exhaustive `AuthorizationCase` declarations
  for unions or string enums. Union bindings traverse the selected branch.
- Supply the generated `AccessAuthorizer` to `NewEndpoints(service, access, ...)`.
  Missing implementations panic at construction. Every evaluator error denies
  execution; map declared errors using each transport's existing DSL.
- Evaluators and authentication hooks must be read-only and repeatable. Protected
  middleware runs after checks, and checks repeat before service execution to
  catch changed targets. Use `Endpoints.Use` or the protected method constructor's
  middleware arguments to preserve this boundary.
- Route in-process, MCP, and agent-tool adapters through protected generated
  endpoints. Direct raw-service calls are not automatically protected.
- Applications own current policy facts, authorized queries, transaction checks,
  and per-message stream authorization. Initial invocation access does not imply
  continuing stream access.
- Inspect `gen/<service>/authorization.json` for static requirements and explicit
  exemptions. It does not describe per-user capabilities.
- Read `docs/authorization.md` for the complete contract.

## Streaming

- gRPC clients retain the stream when sending its initial payload returns EOF.
  Use `CloseAndRecv` for client streaming or `Recv` for bidirectional streaming
  to read the final server status or response; EOF from the opening send alone
  does not establish success. Regenerate clients to adopt this behavior.
- SSE endpoints use normal HTTP success responses with
  `text/event-stream`. Regenerate HTTP and JSON-RPC clients to use the shared
  library reader. An incomplete final event is discarded at EOF. Always close
  the stream; canceling an active receive closes it too. Direct runtime callers
  receive an `SSEEvent` from `SSEStreamReader.ReadEvent`, rather than raw bytes.
  Mapped event IDs retain the last ID when a later event omits `id:`; an
  explicit empty `id:` resets it. Put a per-event identifier in the payload.
- Generated HTTP and JSON-RPC SSE streams expose `loomhttp.SSEControl`.
  Use `Open(ctx)` for explicit readiness and `SendComment(ctx, text)` for
  heartbeat frames.
- Configure bounded stream writes with `loomhttp.NewStreamWritePolicy`.
- Read `Last-Event-ID` through `loomhttp.LastEventIDKey`.
  For HTTP SSE payload binding, pass the DSL attribute name to `SSERequestID`;
  generated Go field names and `struct:field:name` overrides are resolved
  automatically.
- Do not recover or write the raw response writer to work around streaming
  behavior.
- Generated WebSocket streams use `loomhttp.WebSocketStream`; use the generated
  interface rather than adding parallel socket lifecycle code. `Close` is
  terminal, even before the lazy server upgrade: later `Send`/`Recv` return
  `loomhttp.ErrWebSocketStreamClosed`. Call `Recv` from one goroutine only.
- A failed WebSocket connection read is terminal: later `Recv` calls return
  the same error, so return from the receive loop. A context canceled during a
  read closes the stream, so later calls return
  `loomhttp.ErrWebSocketStreamClosed`; an already-done context returns its
  error without reading. A decode error (including
  `io.ErrUnexpectedEOF` for an empty message) leaves the stream readable.
  Messages decode with strict `encoding/json/v2`. JSON-RPC WebSocket `Recv`
  returns `io.EOF` on a normal closure and answers invalid JSON with a Parse
  error frame itself.
- JSON-RPC WebSocket client streams opened on one generated client share its
  connection safely: each stream receives only its own responses. Closing a
  stream or canceling its context ends that stream alone; the connection
  closes with its last stream or with the client. Client `Close` is permanent:
  later stream calls fail without dialing. Create a new client to reconnect.
  Match explicit stream or client closure with
  `errors.Is(err, jsonrpc.ErrStreamClosed)`, including blocked receives and
  attempts to open a stream after client closure. Independent cancellation
  and network failures keep their own errors. An interrupted write also
  preserves its socket error; a response already available may win a race
  with closure.
  Do not add a separate
  reader or a client per stream to work around sharing.

Loom also emits the framework-owned `x-loom-async` OpenAPI extension for richer
SSE and WebSocket handshake/message contracts.

## JSON-RPC

JSON-RPC is a first-class transport, not an HTTP behavior alias.

- Omitted `params` decode as `{}`; ordinary required-field validation still
  applies. Nullable or `Any` params are the exception: omitted params leave an
  optional attribute absent and fail a required one, or a nullable or `Any`
  payload, with `-32602` `missing_payload`.
- Only an explicit effective JSON-RPC `Response(...)` mapping creates a typed
  `error.data` contract. HTTP mappings never apply to JSON-RPC.
- Explicit mappings project the concrete service error into the designed body.
  Use a required `ErrorName` field when multiple names share one custom type.
- Unmapped service errors use generic `jsonrpc.ErrorData`. Generated clients
  return the raw `*jsonrpc.RawErrorResponse`, including its code, message, and
  data.
- Add remedy fields to a mapped custom error type when its public body must
  expose retry guidance. Generic errors carry Loom's nested remedy metadata.
- SSE notifications, final responses, and protocol errors use the generated
  stream contract.
- For viewed JSON-RPC stream results, call `SetView(name)` on a dynamic method
  stream over WebSocket or SSE; fixed-view methods have no setter.
  Connection-level sends use the method's fixed or default view. WebSocket
  methods that receive a streaming payload and return one result also use
  the fixed or default view. Generated peers carry `loom_view` per message
  and validate the selected view before returning the canonical
  type. Missing markers fall back to the fixed or default view. Regenerate both
  peers for dynamic views; keep this Loom extension when using a custom client.
- Set intermediate notification names with
  `SSENotificationMethod(...)` when the default namespaced method is unsuitable.
- A raw `GET /rpc` events listener is ID-less and suppresses final responses;
  send every value that must reach that listener with `Send`.
- For mixed HTTP/SSE services, only designed SSE methods route to streams even
  if the client advertises `Accept: text/event-stream`.
- Mount and wrap the generated public `ServeHTTP` handler so middleware and
  transport policy are preserved.

## CORS and Request Metadata

- Model browser access with `CORS` in the HTTP or JSON-RPC design.
- Use `RuntimeCORS()` when origins come from deployment configuration, then
  pass a validated `loomhttp.RuntimeCORSPolicy` snapshot to the generated
  constructor.
- An authored HTTP `OPTIONS` endpoint may share a path with CORS preflight.
  Generated dispatch uses the presence of `Access-Control-Request-Method`.
- Apply `loomhttp.RequestMetadataMiddleware` through the generated server's
  `Use` method and read `loomhttp.RequestMetadataFromContext`.
- Configure retained headers and trusted proxies with
  `NewRequestMetadataPolicy`. Sensitive headers require explicit opt-in.
- Use `loomhttp.EffectiveClientAddress` instead of interpreting forwarding
  headers in application code.

## Observability and Debugging

Prefer framework packages over repeated bootstrap glue:

- `github.com/CaliLuke/loom/observability/otel`
- `github.com/CaliLuke/loom/http/middleware/otel`
- `github.com/CaliLuke/loom/grpc/middleware/otel`
- `github.com/CaliLuke/loom/observability/transport`

For HTTP clients, wrap `*http.Client` with `otel.WrapHTTPClient(...)`. For HTTP
servers, use `loomhttp.NewMuxer()` with `otel.HTTPMiddleware(...)`. For gRPC,
use `otel.GRPCServerOption(...)` and `otel.GRPCClientOption(...)`.

Transport observation and clue HTTP logging preserve supported writer interfaces
for streaming, upgrades and optimized I/O. Captured HTTP status ignores interim
responses such as 103, retains terminal 101, and records implicit 200 on writes
or flushes. Before commit the status is zero; hijacked connection I/O is outside
the capture.

`observability/transport.Event.Reason` is a stable, low-cardinality value for
metrics and routing. Handle these values rather than parsing error messages:

- `ok`
- `request_decode_failed`
- `invalid_jsonrpc_envelope`
- `invalid_jsonrpc_batch`
- `invalid_jsonrpc_method`
- `invalid_jsonrpc_params`
- `unsupported_method`
- `missing_credentials`
- `invalid_credentials`
- `permission_rejected`
- `principal_mismatch`
- `handler_error`
- `panic`
- `response_write_failed`
- `stream_write_failed`
- `stream_flush_failed`
- `stream_write_timeout`
- `stream_flush_timeout`
- `stream_final_response_suppressed`
- `mcp_session_missing`
- `mcp_session_not_found`
- `mcp_session_principal_mismatch`
- `mcp_events_stream_write_failed`

Use `loomhttp.NewDebugDoer` only for bounded, redacted development diagnostics.
Set `DEBUG_LOOM=1` while generating when you need DSL/codegen decision traces.

`loom vet <module-import-path>/design` evaluates the composed design. Run it
periodically to find APIs that need better modeling; do not add it to the
routine generation or local check path.

The command checks incomplete target-module source analysis, direct Loom mux
routes, exact duplicate manual routes, exact manual/design route conflicts,
generated-version differences, generated output that no longer matches the
evaluated design, missing HTTP error metadata, and descriptions that imply
missing validation or a concrete scalar hidden behind `Any`. The
`untyped-semantic-attribute` warning is conservative: timestamp-style names or
unambiguous scalar descriptions can trigger it, but `Any` by itself and generic
metadata, configuration, preference, or document fields do not. A manifest
created before design digests were introduced also requires `loom gen`. Source
analysis follows the active packages selected by the current Go build
environment and excludes dependency modules.

Applications can opt into missing-mount analysis at API scope:

```go
API("inventory", func() {
    Meta("loom:vet:http-entrypoint", "./cmd/api", "./cmd/worker")
})
```

Each value identifies a package whose consumer-owned files contain generated
HTTP server `Mount` calls. Conditional calls count. Add every package that owns
mounts; Loom does not prove runtime branch reachability. Library modules stay
opted out by omitting this metadata. Suppress an intentionally unhosted service
with `Meta("loom:vet:ignore", "service-not-mounted")` on that service.

Use `--format=json` or `--format=sarif` to archive or share audit results.

Use `Meta("loom:vet:ignore", "<rule>")` to suppress an intentional design
warning. Put `Meta("loom:vet:ignore", "generated-design-skew")` on the API only
when generated output is intentionally managed outside the current evaluated
design. Put `//loom:vet ignore route-outside-design -- <reason>` immediately
before an intentional direct mux route. The same source-comment form accepts
`duplicate-route-registration` and `route-conflict-with-design`; Loom does not
infer conflicts from dynamic method or path expressions.

## Interceptor Result Boundaries

Use `ReadResult` and `WriteResult` for ordinary endpoint results. An HTTP or
gRPC method with a streaming payload sends its final response through
`SendAndClose`; use
`ReadStreamingResult` or `WriteStreamingResult` for it. Server send callbacks
access `info.ServerStreamingResult()` before `next`; client receive callbacks
access `info.ClientStreamingResult(result)` after a successful `next` call.
Branch on `info.CallType()` when also accessing ordinary or streaming payloads.
Stream wrappers preserve `SetView` and expose canonical service result types.
JSON-RPC WebSocket streaming-payload methods retain their ordinary server
result boundary; the new final-response streaming accessors do not apply.

## Installation and Commands

The repository skill tracks Loom `main`; a copy read from a release tag
describes that tagged snapshot. Use the current recommended release for
consuming services unless intentionally testing an unreleased checkout. Loom's
release workflow stamps this recommendation and the public installation guides
together so their version pins remain aligned.

`loom vet` is available in `v1.9.0-alpha.1` and later releases.

```bash
go install github.com/CaliLuke/loom/cmd/loom@v1.10.0-alpha.4
loom version
loom import openapi openapi.yaml -o design
loom gen <module-import-path>/design
loom example <module-import-path>/design
loom vet <module-import-path>/design
```

## Canonical Guides

For Pulse applications, read `docs/pulse.md`. Use the latest stable Redis
release, currently 8.10.2; older releases are unsupported. Pool protocol 2 requires closing
all protocol 1 nodes before upgrading or rolling back; mixed versions are not
supported. `Node.Health(ctx)` reports incompatible live members. `Job.Epoch`
identifies ownership; external stores must enforce fencing themselves. Pool
shutdown retains epoch counters. Workers stop handlers after three quarters
of the TTL without a confirmed renewal, then resume only the same owner and
epoch. Callbacks must finish within the remaining fence margin; external
writes still need store-side epoch checks. Use `Worker.CheckOwnership` in the
job work loop as a point-in-time check; abort the
operation on any error. Ownership mismatches queue an asynchronous stop so
`Stop` can join the checking loop. It cannot replace atomic epoch enforcement at the
destination store. After rebalance releases a job,
a failed Redis requeue reply does not restart it on the old worker because
the write may already have succeeded. If it did not, orphan recovery waits
for `max(2 * workerTTL, ackGracePeriod)` without a worker and a subsequent
sweep. Account for that recovery delay when choosing these settings.
Retry failed `Sink.RemoveStream` calls: membership errors retain the local
stream, while group-cleanup errors stop polling and retain cleanup ownership.
`AddStream` can resume a stream with pending group cleanup; `Close` releases all
retained map subscriptions. Consumer rotation covers all active streams and
preserves pending entries for idle recovery. A sink with no streams waits;
adding a stream resumes polling.

- `docs/quickstart.md`
- `docs/dsl-reference.md`
- `docs/code-generation.md`
- `docs/typescript-clients.md`
- `docs/http-guide.md`
- `docs/grpc-guide.md`
- `docs/error-handling.md`
- `docs/interceptors.md`
- `docs/production.md`
- `jsonrpc/README.md`

Open the guide closest to the task before searching framework source. If using
Loom correctly still leaves repeated application glue, report the boundary and
route a separate framework task through `loom-framework`.

JWT/OAuth service payloads contain the token without `Bearer `. HTTP and gRPC
Authorization mappings add/remove bearer framing before applying token constraints.
Malformed Authorization is rejected, including a raw token or another scheme.
Optional credentials may be absent. Custom mappings and API keys carry raw values;
do not add a prefix to custom headers. See the credential mapping contract in
`docs/dsl-reference.md`.
