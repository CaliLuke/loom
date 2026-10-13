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
so manual changes are both temporary and misleading. If rollback fails, Loom
retains the unrestored originals and reports each backup and intended destination
for manual recovery.
The output directory must be writable, but its parent need not be. Output and
plugin destinations may be on separate filesystems. Mount the output directory,
not `gen/` itself: regeneration must replace the whole generated tree. Relative
module replacements and workspace configuration keep their existing meaning.

Generated HTTP and JSON-RPC Server handler fields have their own names, which
can differ from service methods when they would collide with fixed members or
other handlers. Use the regenerated declarations when accessing fields directly;
service method names and wire routes stay unchanged. Regenerate after upgrading.

Transport client endpoint factories, request builders, Doer fields and mixed
stream factories can also receive suffixes when receiver members collide.
Regenerate and use the actual declarations when wiring custom Doers or calling
transport factories. Canonical service methods, CLI commands and wire names
remain unchanged. A service client's synthetic mixed-stream method can also gain
a suffix when an authored method occupies its usual name; use the regenerated
client declaration.

User-editable plugin files belong outside `gen/` and should declare create-only
behavior. Loom rejects create-only files inside the regenerated tree. Upgrade
the CLI launcher and the design module's Loom dependency together.

Generation, example, and vet helpers honor the active Go workspace's modules
and replacements. Set `GOWORK=off` to use only the design module's `go.mod`.

Use generated error-factory declarations rather than assuming `Make` plus a
normalized error name. Colliding names receive suffixes, shared with the service
package types. Regenerate and update handwritten references to renamed factories
or types; wire error names and mappings remain unchanged.

Declare each exact error name once per API, service or method scope. Duplicate
declarations fail design validation even when their types match; remove the
duplicate before regenerating. Method overrides of service errors and reuse in
separate scopes remain valid. Error names are case-sensitive.

Full built-in HTTP/JSON-RPC errors carry required `timeout`, `temporary` and
`fault` booleans. Generated clients preserve them independently of response status
or code. Explicitly pruned object bodies omit unselected properties (decoded as
false); scalar/empty Body selection retains automatic header mappings. Regenerate
clients and servers together. Handwritten `http.ProblemErrorFromBody` calls take a
complete `http.ProblemResponse` value with explicit properties.

HTTP errors may share a status and use distinct media types. OpenAPI retains
all representations and uses `anyOf` for overlapping body or header schemas.
Headers are required only across all alternatives. Keep response links,
extensions and explicit component names compatible; conflicting metadata
produces a generation diagnostic identifying the endpoint and status.

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
Generated gRPC imports escape collisions with local variables as well as
framework packages: service `md` can be imported as `mdsvc`, and service `req`'s
protobuf import as `reqpb2`. The service name, package path and wire contract
remain unchanged. Regenerate `gen/`; for existing create-only example servers,
compare a freshly generated example in a separate directory and update matching
import aliases and references while retaining your application changes.
gRPC methods with and without `Any` can share named types, including recursive
and union types. Regenerate adapters to pick up corrected conversion helpers for
these mixed designs; no design workaround is needed.
Typed object/message elements in gRPC arrays and maps must be nonnil. Regenerate
clients and servers to reject invalid elements before sending a message. Some
generated conversion constructors now return `(value, error)`; check that error
when calling them directly. Optional object fields and nonnil empty objects keep
their existing meaning.
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

HTTP and JSON-RPC example calls follow the generated `ParseEndpoint` signature:
streaming dependencies first, then multipart encoders and client interceptors
per hosted service. When combining multipart and interceptors across services,
update affected existing calls against a fresh example. `loom example` preserves
existing files; the generated parser signature remains unchanged.

HTTP and JSON-RPC CLI help displays descriptions and defaults literally, including
`${name}` and `$$`. Do not add template escaping to the design. Regenerate clients
to adopt this behavior; payload defaults and explicit flag values are unchanged.

HTTP and JSON-RPC CLI JSON body builders reject omitted required members before
applying defaults. Omitted optional members can receive defaults; supplied zero
and empty values remain supplied. Nullable members accept JSON null, ordinary
members reject it. Nested objects use the same rules. Regenerate clients to adopt
these checks; existing body and collection flag syntax is unchanged.

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
- `Pattern` keeps Go runtime semantics. OpenAPI exports a portable ECMA subset:
  ordinary BMP literals/classes, ASCII digit/word classes, boundaries, grouping
  and quantifiers; absolute anchors and ASCII whitespace are translated.
  Wildcards, broad Unicode/negated classes, flags and Go-only syntax without a
  faithful projection fail generation. See `docs/dsl-reference.md` before
  changing a rejected pattern; do not weaken application validation to export it.
- Numeric/length bounds are checked as one effective contract, including named
  ancestry and `Extend`. A base minimum of 10 plus a derived maximum of 5 rejects
  before generation. Equal inclusive endpoints and zero lengths are valid;
  negative authored lengths and equal exclusive endpoints are invalid.
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
- Automatic JSON omission distinguishes absent values from supplied empties.
  Nonnil empty pointers, byte slices, arrays and maps stay on the wire; native
  `false`, `0` and `""` values are supplied, including fields with defaults.
  Required empty collections remain valid unless constrained by length. Custom
  Go replacement types retain their codec contract. Full `struct:tag:json`
  overrides keep their authored options in service and HTTP/JSON-RPC encoders;
  `omitempty` explicitly discards encoded empties. Explicit presence and JSON-RPC ID
  normalization still apply. Both full and name-only `"-"` mean exclusion;
  optional omitted HTTP/JSON-RPC body fields remain invalid. Regenerate after
  upgrading to apply presence and authored-tag handling.
- For each non-`Extend` type, payload, or result, start literal field tags at
  `1` and increment within that definition. For definitions using `Extend`,
  start newly introduced fields at `100`.
- Give each object distinct generated Go field names. `foo_bar` and `fooBar`
  both become `FooBar`; rename one or set `Meta("struct:field:name", "OtherFooBar")`
  on it. The override preserves the design name and transport mapping.
- Package types avoid emitted framework names with deterministic numeric suffixes;
  service methods have a separate member namespace. A domain type `Foo` no longer
  forces method `foo` to become `FooEndpoint`. Regenerate clients and servers and
  update handwritten references after upgrading; affected gRPC RPC names/full
  method paths change with the corrected service method name. HTTP routes,
  JSON-RPC method strings and authored schema names are unchanged.
- Native gRPC handler fields avoid RPC method and embedding names. Regenerate
  after upgrading and update any handwritten handler-field references from the
  generated declarations. Service methods and protobuf wire names stay unchanged.
- HTTP/JSON-RPC conversion constructors use distinct names when source types or
  selected fields differ, even if the target body type is shared. Regenerate
  clients and servers and update handwritten constructor calls; read generated
  declarations instead of predicting `New` plus the target type name.
- Error fields must also avoid generated methods `Error`, `LoomErrorName`, and
  `LoomErrorRemedy` when a `Remedy` is declared. Use `struct:field:name` to keep
  the wire name while choosing a different Go field name.
- Prefer a canonical `ResultType` with `View(...)` definitions over parallel
  hand-maintained DTOs for alternate public representations.
- Return a nonnil object satisfying the selected view's constraints for a
  successful non-nullable viewed result. Declare intrinsic result `Nullable()`
  inside `Attributes`, or use method-local `Nullable()` inside `Result` for only
  that method. Return `loom.NullValue[T]()` or `loom.NullableValue(value)`
  for such a result, never its absent zero value. Required nullable fields reject
  absence and accept explicit null, as do nullable collection positions; only
  concrete values run child-view constraints. After regeneration, an unknown service-selected view
  or an invalid selected result returns a server fault, wrapping the validation
  cause. Excluded fields are not validated. Ordinary nil and empty result collections
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
- gRPC message field numbers must be unique per message and within
  `1..536870911`, excluding `19000..19999`. Invalid numbers fail design evaluation.
  Constructor unions inherit consecutive numbers; explicit branch tags override
  their slots. Correct the design and regenerate instead of editing generated code.
- Binary gRPC metadata (`Bytes`, named byte types and arrays of byte values)
  derives a lowercase `-bin` wire key. Mapping `raw:x-raw` uses `x-raw-bin`;
  an existing suffix is preserved. Distinct fields cannot share the resulting
  key in one channel. Regenerate both sides and use the derived keys in
  handwritten peers with their runtime's binary metadata API.
- Distinct gRPC credentials must also use distinct metadata keys. If session
  bearer and cookie fields both default to `authorization`, explicitly map the
  cookie field with `Attribute("cookie:x-session")` inside `Metadata`.
  Use that same key in handwritten gRPC clients.
- gRPC metadata arrays may use named primitive elements, including types placed
  in another generated package. Keep the named service field; generated codecs
  perform the conversion and enforce inherited constraints and defaults.
  Regenerate both sides when adopting named metadata elements.
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

Automatic request-body, parameter and header components share complete
definitions only when their contents, including examples, agree. Responses share
by HTTP status and complete shape except the top-level description: in both 3.1
and 3.2, operation descriptions override a neutral component description beside
`$ref`. Description-only changes do not split or rename components. Other
metadata and examples still distinguish responses; explicit names stay separate.
Read the reference's description override before the component description. Changing an
example can move a definition inline without changing its underlying schema.
Do not depend on these automatic component references as stable identifiers.
Explicitly authored component names and schema names retain their contracts.
Keep `openapi:operationId` values unique across the document. Use the same
parameter names when multiple methods share one path template: `/items/{id}`
and `/items/{slug}` collide, including templates with the same static suffix.
The default `{routeIndex}` distinguishes emitted paths when a method expands
through multiple service base paths; static operation IDs remain literal and
generation rejects duplicates.
In 3.2, reuse an explicit `openapi:component:mediaType` name only for identical
projected media definitions, including examples and encodings. Conflicts cause
OpenAPI generation errors.
At each media or nested encoding object, named `openapi:encoding` entries cannot
coexist with positional `openapi:prefixEncoding` or `openapi:itemEncoding`
entries. Prefix and item entries can coexist. Generation rejects conflicts for
both targets; move independent choices into separate child objects.
Review regenerated SDK and validator output when updating Loom; equivalent
OpenAPI schemas can still produce different client declarations. See the
[OpenAPI evolution guide](../../../docs/openapi-evolution.md).

Inherited object enums remain predicates when a derived field adds constraints.
HTTP bodies and OpenAPI retain the original declaration; mapped field names
change wire spelling without changing enum membership. Regenerate transports
and OpenAPI after updating to adopt this correction.

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

For maps, `MinLength` and `MaxLength` count entries and export as
`minProperties` and `maxProperties`, including named map types. Regenerate
contracts after upgrading so schema validators enforce the declared map bounds.

Unsigned integer schemas use `uint32` for `UInt32` and `uint64` for `UInt64`
and `UInt`. Regenerate after upgrading and review generated clients. These
format annotations describe the domain; validator format support is optional.
The JSON representation and authored numeric bounds stay the same.

Numeric bounds retain exact integer and string values. Use decimal strings such
as `Minimum("0.1")` for decimal contracts; finite hexadecimal strings are also
accepted. Native Go floats keep their actual binary value, which can differ from
that decimal. Defaults and enum members must project into the required generated
schema without widening it. Regenerate clients, servers and OpenAPI after
upgrading. See the [numeric validation contract](../../../docs/dsl-reference.md#numeric-validations)
for native decoding, exact JSON numbers and inherited enum refinements.

Numeric defaults and enum members must fit their declared type, including nested
values: an Int32 default above 2147483647 and a negative unsigned enum member
fail design evaluation. Float32 values can round or underflow, but cannot overflow
or be nonfinite. A wider bound such as `Maximum(1e12)` on Int32 remains valid;
constraints describe allowed values rather than stored numeric fields.

Conversion defaults fill actual absence. A native Go scalar has no absence bit:
its `false`, `0` or empty string is supplied and survives generated conversion,
including when a struct literal leaves it at zero. Optional pointers/wrappers and
nil-capable values can carry absence; explicit nullable null is retained. Missing
required input is rejected before defaults. Regenerate clients and servers to
adopt this correction; do not depend on implicit native-zero substitution.

Object defaults, including objects inside arrays and maps, use the generated
native type. Within an authored default, missing nested members use applicable
member defaults while supplied members keep their values. Every authored member
needs declared native storage; unknown default members fail generation with a
member path. This does not change unknown-member admission on incoming wire
objects or the rules for when defaults apply. Regenerate affected clients and
servers; no handwritten literal replacement or JSON round trip is needed.

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
Local component references accept equivalent percent-encoded URI spellings, such
as `#/components/schemas/%49temName` for `ItemName`. Regeneration emits canonical
references; component keys must still obey OpenAPI's naming rules.
Cookie parameters require primitive schemas. Array and object cookies are
blocked before rendering, including schemas reached through references.
Parameter reference description overrides in OpenAPI 3.1 and 3.2 survive
import and regeneration independently of the shared component description.
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

OpenAPI 3.1 and 3.2 boolean `true` schemas also import as `Any`. Boolean
`false` schemas and typeless `properties` are blocked because the DSL cannot
preserve their accepted values. Use an explicit object type only when the
source contract intends to reject non-object values. The legacy `nullable`
keyword is supported only in OpenAPI 3.0; modern documents must use a supported
null type union or a bare null composition branch. Assertions or annotations
on that branch remain blocked, including with `--allow-lossy`.

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

Imported enum/const membership intersects the declared type and nullability.
A nullable string enum must include null to retain `Nullable()`; null members
excluded by the source type are removed. Empty or null-only domains and included
numeric members that cannot retain their value in the DSL carrier are blocked,
including with `--allow-lossy`. Reimport the source contract to adopt this repair.

A scalar JSON Schema `const` without a sibling `enum` imports as the equivalent
one-value `Enum(...)` validation. Regenerated OpenAPI uses `enum` because Loom's
DSL has no separate scalar-constant construct. Structured constants and schemas
that combine `const` with `enum` remain blocked.

A memberless OpenAPI `type: object` with absent, true or empty-schema
`additionalProperties` imports as `MapOf(String, Any)`. Generated Go uses
`map[string]loom.JSONValue`. Closed empty objects retain the `{}` contract and
separate nullable intent. Inline closed-empty fields and structured object map
values become deterministic named types. Reimport and regenerate to replace
previous String fields or incomplete map constructors. Combining declared
members with free-form additional properties remains unsupported.

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

Imported schema descriptions and titles survive on scalar fields, collection
containers and elements, named types and supported reference overlays. Parameter
and header descriptions remain separate from their schema descriptions; generated
DSL uses `Meta("openapi:description:schema", ...)` for that distinction. Re-import
and regenerate to recover descriptions omitted by older generated designs.

Ordinary recursive component references import through named DSL references,
including recursion through object fields and their arrays or maps. Cycles made
only of eager alias, array or map constructors remain outside the import subset
and receive a blocking diagnostic before a design is written. See
[OpenAPI import coverage](../../../docs/openapi-import-coverage.md) for supported
recursive shapes and limitations.

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
AND or alternative semantics. Schemes using the same wire credential share one
payload field while retaining each authentication check. A conjunction whose
schemes require different scope sets is rejected; equal scope sets and separate
alternatives remain supported. Partial import omits unused root security when
every retained operation explicitly overrides it. An anonymous `{}`
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

Optional discriminator metadata requires an explicit fallback: a known component
name or local component reference, or an external URI reference (use `./` for a
relative path). Invalid combinations fail export for both targets. The 3.1
projection requires the discriminator property and removes the fallback field.

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

Unknown service errors return `internal server error`. Explicit `ServiceError`
and generated error messages are public unless `SafeMessage` overrides them.
Single-cause wrappers keep their diagnostic text private; independent joins use
the generic message unless an explicit outer error owns the public contract.
Designed error bodies remain public authored data. Native gRPC cancellation
and deadline errors use canonical context messages. This message policy takes
effect on runtime upgrade without regenerating adapters.

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
- Use `Body("field")` on a custom HTTP error response to encode that field as
  the entire body and restore it in generated clients. String fields support
  `text/plain` and `text/html`; JSON `Any` and map fields have no extra envelope.
  Regenerate both transports after updating Loom for selected error body fixes.

Built-in HTTP errors also support `Body("detail")` and other public problem
fields, explicit shaped bodies, and header/cookie mappings. They expose the
public safe-message and retry-hint representation. Generated clients reconstruct
only fields available in those bindings; unmapped fields cannot be recovered.
Custom formatters receive the whole service error even for a selected scalar.
Regenerate server and client transports when adopting this projection repair.

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

Request `ID()` fields bind only to the JSON-RPC envelope and are excluded from
params. Regenerate clients and servers together after this change. Required IDs
reject absence before defaults; optional defaults affect service data without
turning a notification into a request. Explicit null requires `Nullable()` and
never defaults. Named String constraints and compatible custom codecs apply to
the normalized envelope value; original tokens remain the response correlation.
Use a separate generated CLI ID flag rather than embedding ID in the body flag.
See [ID semantics](../../../jsonrpc/README.md#id-semantics).

Generated JSON-RPC server packages expose the same per-method and service-wide
manifest pattern. Cases cover success results, declared error codes, typed
error data, and response suppression for ID-less notifications. Call each real
generated handler from the scaffold and return a
`jsonrpc.ResponseContractObservation`. Server-SSE calls with an ID end with a
final response. ID-less streams suppress that response. Other streaming
completion shapes are explicit generation limitations.

## Map Keys

Native `Bytes`, array and map keys, including named aliases, fail design
validation because their generated Go storage cannot be a map key. Scalar,
object-pointer and `Any` keys retain their existing representations. Custom Go
key types must be comparable; transport codecs can impose narrower restrictions.
Correct the key design and regenerate instead of editing generated declarations.

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
  parameters or metadata, forms, or multipart bodies. gRPC rejects nullable
  concrete shapes before generation. Its message `Any` fields and message-root
  `Any` support explicit null; this exception does not permit `Nullable()` on
  gRPC collection members or union branches. Regenerate adapters after correcting
  previously accepted unsupported designs.
- Array elements reject JSON null by default. Put `Nullable()` on the element
  definition to accept null members; array-field requiredness is independent.
- HTTP WebSocket `StreamingPayload` roots cannot be nullable: a root JSON `null`
  frame ends client input. Put nullable values inside a non-null payload object.
  Nested nullable fields and collection members remain supported; this request
  framing restriction does not apply to JSON-RPC or WebSocket responses.
- OpenAPI 3.1/3.2 value-or-null unions and OpenAPI 3.0 `nullable: true` import
  to this same DSL contract.

## HTTP Collection Headers

`Header("tags:X-Tags")` on an array uses standard comma lists. Repeated and
combined lines decode equivalently. Nil is absent; a nonnil empty collection
sends an empty header, and defaults apply only to absence. Required empty arrays
remain valid unless a length constraint rejects them. Items containing a comma,
empty text, or leading/trailing space or tab are rejected in authored values and
by outgoing encoders. Use a JSON body for arbitrary text lists. Scalar headers
keep their own wire grammar; quotes do not escape collection delimiters. Use the
cookie DSL instead of an array `Set-Cookie` mapping. Regenerate clients and servers
together; these rules also cover JSON-RPC HTTP metadata, not gRPC metadata.

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
- A present gRPC collection wrapper can contain an empty array or map, including
  an empty inner collection in a nested array/map. `MinLength` still applies,
  and a required wrapper cannot be omitted. Regenerate clients and servers to
  adopt corrected empty-content validation. Ordinary unwrapped repeated/map
  fields still collapse absence and empty on protobuf wire.
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
- Generated union validators reject selected nil object/nested-union pointers.
  Selected nil slices/maps are empty values and still receive their length and
  element constraints. A constructed nullable branch accepts explicit null but
  rejects absence. Regenerate service and transport code to receive these checks.
- A tagged union envelope always requires its value property. Explicit null
  is accepted for a selected nullable or `Any` branch, including inherited
  nullability, and rejected for other branches. This is separate from absence
  or null of the enclosing union attribute. Regenerate service and transport
  packages when adopting this decoder correction.
- Result views inherit canonical requiredness. Use `ViewRequired(...)` and
  `ViewOptional(...)` for deliberate overrides. Method-specific `Required(...)`
  customization applies to included view fields while explicit view overrides
  take precedence; the original result type and other methods are unchanged.
- Generated projection helpers convert between canonical results and view
  types. Use them instead of maintaining app-local conversion copies.
- Nested result types use their explicitly selected child view or their own
  default view, including through arrays, maps, objects and union branches.
  A parent's non-default view does not select the same view on its children.
  Regenerate adapters after upgrading to apply recursive projection fixes.
- `Extend(...)` inside a named `Payload`, `Result`, `StreamingPayload`, or
  `StreamingResult` customization creates a method-specific type with the
  inherited fields and requiredness. Plain uses of the named type are unchanged.
  Automatic result views include inherited fields; explicit views keep their
  authored field selection.
- `Extend` redeclarations refine inherited contracts. Base `Minimum(0)` plus
  local `Maximum(100)` admits 0 through 100; inherited requiredness and
  non-nullability remain enforced. Incompatible types, contradictory constraints
  and conflicting explicit wire identities fail design validation. Named types
  must retain the inherited nominal ancestry. A local default may select among
  inherited defaults only if it satisfies the complete refined contract; unequal
  peer defaults without a local selection are rejected. Omitted `Nullable`
  inherits the base contract. Regenerate adapters after upgrading.
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
- An explicitly selected zero-field object response remains `{}` in JSON.
  `Body(Empty)` declares no body; `OpenAPIBody` preserves the same schema
  distinction without changing encoding. Regenerate affected clients and servers.
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
- Named query maps and named form object/map fields encode one JSON document,
  for example `filter={"color":"red"}` before URL escaping. This also applies
  to shallow maps. Scalars and repeated scalar arrays retain their formats;
  whole-query maps and root-form maps retain their unwrapped outer layout.
  Fixed-object query parameters and composite form arrays remain unsupported.
  Regenerate clients and servers together and update handwritten bracket-style
  requests when adopting this contract. OpenAPI query parameters describe JSON
  through `content`; form properties use `encoding.contentType: application/json`.
  Import rejects incompatible deepObject/style/explode contracts, including with
  `--allow-lossy`. JSON-compatible form serialization and JSON query map content
  can round-trip through import and generation.
- OpenAPI import admits path parameters and headers only as primitives or
  primitive arrays, and cookies only as primitives, following schema references.
  Unsupported bindings fail before design publication, including with
  `--allow-lossy`. JSON query maps remain supported; closed empty objects are
  records, while memberless open objects are maps.
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
- Start each path parameter at the beginning of its segment, using ASCII letters,
  digits, or underscores in its name. `/{prefix}/{number}` and `/{name}.json`
  are supported; `/{prefix}-{number}` and `/item-{name}` fail DSL validation.
  Use a single parameter for a composite value, or split the parameters into
  separate segments. A `{*name}` catch-all must terminate the path.
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
  same service error while reading. Multipart limits count the complete encoded
  source, including headers, boundaries and epilogue. Successful custom callbacks
  finish within that budget before service invocation; do not retain a live parser
  or part. Regenerate adapters. Direct `ReadMultipartForm[WithLimit]` calls now
  take the original request; the default entry honors the configured policy.
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
HTTP and JSON-RPC CLIs use one direct flag for each payload field, even when Basic
credentials or ordinary fields feed several explicit transport mappings. The
single payload value supplies every mapping. Body flags remain separate; direct
field flags keep their existing precedence. Regenerate to remove duplicate flags.

HTTP mappings reject different fields sharing a header, query or cookie name;
headers compare case-insensitively. This includes inferred Authorization mappings.
Map independent credentials to distinct locations, for example
`Header("key:X-API-Key")` beside a JWT in Authorization. Basic OR Bearer may share
Authorization with optional payload credentials; their framing distinguishes them.
The same source and encoding may be reused. Correct collisions and regenerate.
Only selected security credentials are implicitly excluded from the request body.

For JSON-RPC WebSockets, explicitly select message credentials in `Body`.
Header/query/cookie credentials, Basic and implicit session bindings are rejected.
A separate `StreamingPayload` must retain each selected credential's name,
security tag and supported string type. An untagged field with the same name
cannot replace a `Token`, `APIKey` or `AccessToken`. Authentication runs per
message, including each authenticated alternative, rather than on the shared
handshake. Correct rejected designs before regeneration.

Basic-auth header presence is independent of its two components. The client
sends the header if either component is required or present; an absent optional
counterpart becomes an empty component. When both are optional, a missing or
invalid Basic header leaves their pointers nil. A valid header supplies both
components, including empty values. If either is required, a missing or invalid
header fails decoding. Generated HTTP and JSON-RPC servers enforce authored
Username and Password constraints before authentication, including on supplied
empty values. Security alternatives do not remove payload constraints.
Regenerate clients and servers to adopt this behavior.

Optional bearer credentials can coexist with Basic or other security alternatives.
A foreign Authorization scheme leaves an optional bearer field absent only when
another requirement does not use it. Authentication callbacks decide which
alternative succeeds. Payload Required, malformed Bearer framing and supplied
credential constraints still apply unconditionally. This applies to HTTP,
JSON-RPC POST and gRPC metadata; regenerate transports after upgrading.

Prefer Loom's first-class session DSL:

- `SessionAuth(name, fn)`
- `BearerTransport(scheme, fieldName, fn...)`
- `CookieTransport(scheme, fieldName, fn...)`
- `CookieName(name)`
- `SessionSecurity(contract)`
- `SessionCookie(...)`
- `CookieInsecure()` for plain-HTTP local development only

The nearest API, service or method scope declaring `Security`, `SessionSecurity`,
or method `NoSecurity` supplies the complete authentication declaration. Local
ordinary security replaces inherited sessions; local sessions replace inherited
ordinary security. Declare both at the same scope to accept both. Combining
`NoSecurity` with authenticated `Security` or `SessionSecurity` at one method is
rejected. Use `Security()` to add an anonymous alternative. Regenerate adapters
after upgrading to apply this scope rule.

A compound `Security` requires every listed scheme. Its required scopes are
projected onto each scheme's declared scopes, including Basic and API keys.
Callbacks can use `scheme.Validate(grantedScopes)` directly: an unscoped key
alongside a scoped JWT receives no JWT scopes. Shared scopes apply to each scheme
that declares them; undeclared scopes fail design validation. Separate `Security`
alternatives retain their own scope sets. Regenerate endpoints and OpenAPI to
adopt this correction.

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

- Viewed HTTP SSE results use the selected view for each event and collection
  element. An omitted mapped data field stays omitted even if it is required
  or defaulted in the canonical result: raw text is empty, and JSON data is
  `null`. Regenerate peers after the selected-view body correction.
- For gRPC client-streaming or bidirectional methods with an initial payload
  frame, Loom authenticates before waiting only when all declared alternatives
  use metadata credentials. It authenticates again after complete decoding, so
  hooks must be read-only and safe to repeat. Mixed metadata/message alternatives
  wait for the initial frame in declared order. Application access checks and
  middleware receive complete payloads. Regenerate service and gRPC packages
  together; endpoint fields and constructor signatures stay unchanged. See
  [gRPC streaming](../../../docs/grpc-guide.md#streaming-patterns).
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
- HTTP and JSON-RPC WebSocket clients and servers default to 32 MiB per
  decompressed message, a 64 MiB raw buffer budget, and 256 queued/decoding
  messages. Use `loomhttp.NewWebSocketConfig(loomhttp.WebSocketLimits{...},
  configure)` to choose limits and a hook; zero fields select defaults. The
  byte budget must hold at least two maximum messages. Decoded objects,
  allocation overhead and raw bytes retained by callers are outside that budget.
  Oversize messages close with 1009; buffer overload closes with 1013 and
  cancels the connection context. Empty messages count toward the message limit.
  Regenerate adapters and replace the old numeric constructor argument with
  `WebSocketLimits{MaxMessageBytes: maxBytes}`. HTTP uses generated
  `NewConnConfigurer(config)` and per-method fields; JSON-RPC passes config
  directly. A replacement hook owns releasing its input without invalidating
  the replacement. Direct runtime callers allocate `NewWebSocketStream(config)`
  and check `SetConn(ctx, conn, cancel)`.
- The shared reader detects disconnects even while the service is not receiving.
  Generated HTTP handlers close attached connections when the service returns
  (1000 on success, generic 1011 on failure or panic). No upgrade is forced for
  a handler that never uses its stream. `CloseAndRecv` still half-closes input
  and receives the result; it does not terminally close before the response.
  HTTP client `Close` gives the peer a best-effort interval to finish after its
  final null marker. One second bounds the write and wait together; return does
  not acknowledge service processing. Direct runtime `Close` remains immediate.
- Connection read failures are terminal. Live-context readers may consume
  previously admitted messages before the recorded natural read error; explicit
  close and overload discard queued work. Return from the receive loop on a
  terminal error. Cancellation during an operation closes the connection;
  already-done contexts return their error without touching the stream. A decode
  error, including `io.ErrUnexpectedEOF` for empty input, leaves it readable.
  Messages use strict `encoding/json/v2`. JSON-RPC maps normal closure to io.EOF
  and answers invalid JSON with a Parse error frame. A canceled service context
  may return context.Canceled instead of the connection's recorded cause.
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
- With a trusted peer, the nearest untrusted `X-Forwarded-For` IP selects the
  client; an invalid or empty entry encountered first keeps the direct peer.
  Proxy entries must be bare IP addresses.
- Use `loomhttp.EffectiveClientAddress` instead of interpreting forwarding
  headers in application code.

## Observability and Debugging

`loomotel.New` leaves existing global telemetry settings intact on construction
failure and attempts reverse-order cleanup with an uncanceled five-second context.
Handle its joined construction/cleanup error. After successful setup, call
`Runtime.Shutdown` with the application's shutdown deadline.

Prefer framework packages over repeated bootstrap glue:

- `github.com/CaliLuke/loom/observability/otel`
- `github.com/CaliLuke/loom/http/middleware/otel`
- `github.com/CaliLuke/loom/grpc/middleware/otel`
- `github.com/CaliLuke/loom/observability/transport`

For HTTP clients, wrap `*http.Client` with `otel.WrapHTTPClient(...)`. For HTTP
servers, use `loomhttp.NewMuxer()` with `otel.HTTPMiddleware(...)`. For gRPC,
use `otel.GRPCServerOption(...)` and `otel.GRPCClientOption(...)`.
`otel.TraceConfig.SampleRatio` controls root traces; valid local and remote
parents retain their sampling decision. Explicit new-root spans use the root
ratio. A supplied custom provider owns its own sampling policy.

For Clue logging, derive a child with `log.With` before changing its options
with `log.Context`. Derived loggers retain independent configuration snapshots;
updating one does not reconfigure its parent or siblings. `WithContext` joins
an existing logger instead. Derived loggers share one scope buffer: flushing
from any descendant emits all queued entries once and disables buffering for
that scope. Entries keep their original destinations and formats; configuration
changes apply to future entries. `WithOutputs` copies output slice elements when
applied; shared writers and callbacks remain the application's responsibility.

Clue HTTP middleware and gRPC server interceptors give each handled request a
fresh log buffer, even with request IDs or access logging disabled. Root fields
and options are inherited; root buffered entries and flush state are not.
Configured Debug and buffering callbacks continue to control immediate output.

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
Regenerate after security/mapping changes. Pass the credential-bearing service's
`gen/http/<service>` or `gen/jsonrpc/<service>` package `Fields()` to handwritten
`NewDebugDoer` and `middleware.Debug` calls; generated example clients do this
automatically. These constructors copy credential metadata, mask actual custom
wire names as well as conventional names, and conservatively combine policies.
Scalar credential bodies and credential paths are omitted, including route vars.
Ordinary opaque previews may still contain unmodeled secrets.
For access logging, `log.HTTP(ctx, log.WithURLCredentials(serviceDiagnostics.Fields()))`
and `log.Client(transport, log.WithClientURLCredentials(serviceDiagnostics.Fields()))`
use the same URL policy. Conventional query credentials and userinfo are masked
by default; ordinary query values stay visible. Options snapshot their inputs and
compose when reused. New generated server examples select metadata for their own
HTTP/JSON-RPC services. Existing example files are create-only: update their
logging calls when adopting this behavior.
Its response capture follows caller reads without eager consumption. Automatic
output occurs once at observed EOF or caller Close; keep closing bodies normally.
`Fprint` is a snapshot, so before EOF it may be incomplete. Incomplete structured
previews are omitted, and an unread/unclosed response does not print automatically.
Server `middleware.Debug` preserves SSE flushing and response writer capabilities.
It captures at most 64 KiB per request or response preview; larger bodies produce
an omission marker without limiting application I/O. Request capture follows
downstream reads and preserves their bytes, errors and Close behavior. It never
drains an unread request. A preview without observed EOF is explicitly marked
incomplete. The request-before-response summary is written when the handler
returns; streaming does not wait for the upload to complete.
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
without design provenance also requires `loom gen`. Each generated tree records
its originating design package and a fingerprint of all evaluated roots, including
plugins. In a
module with several designs, vet checks design skew only for the selected
package; version checks still cover all trees. Regenerate existing trees after
upgrading this manifest format. Source
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

## Generated SSE Client Streams

Use the service client with either HTTP or JSON-RPC SSE. `Recv()` retains the
opening context; `RecvWithContext(ctx)` supplies a per-operation context. Client
streaming interceptors preserve both operations and `Close()`. Close a stream
when stopping before EOF. After regeneration, migrate direct SSE `Recv(ctx)`
calls to `RecvWithContext(ctx)` and pass the opening context explicitly to a
direct HTTP `New...Stream(ctx, response, decoder)` constructor. Generated
endpoints do this automatically. For a shared SSE/gRPC service interface,
gRPC Close retains CloseSend semantics; cancel the original RPC context to
terminate reception.

## gRPC Stream Operation Contexts

Use `SendWithContext`, `RecvWithContext`, `SendAndCloseWithContext` or
`CloseAndRecvWithContext` for operation deadlines. An already-ended context
performs no I/O and leaves a healthy stream usable. Cancellation during active
I/O terminates the whole RPC; server service code must return after that error
so native work can finish. Canceling an old, completed operation does not cancel
later operations. Context-free methods use the opening context. `CloseSend`
keeps reception alive; receive final status or cancel the opening context when
abandoning it. Regenerate gRPC packages to adopt the correction. See
[stream operation contexts](../../../docs/grpc-guide.md#stream-operation-contexts)
for data ownership, recovery and custom-hook limits.

## Shared Interceptor Accessors

One interceptor may be applied to different server and client methods. Its
shared Info/accessor types cover both method sets while invocation remains
limited to each side's declarations. Regenerate services and clients after
upgrading to obtain complete accessor coverage; no interceptor implementation
change is needed when accessed fields share a compatible type contract.
Optional/required storage and custom Go field names remain method-local. Shared
accessors require one logical Go signature; conflicting primitive, named or
nullable types fail generation with both methods identified. Reuse one design
type for fields intended to share a typed accessor; regenerate to adopt corrected
bindings.

## Interceptor Result Boundaries

Use `ReadResult` and `WriteResult` for ordinary endpoint results. An HTTP or
gRPC method with a streaming payload sends its final response through
`SendAndClose`; use
`ReadStreamingResult` or `WriteStreamingResult` for it. Server send callbacks
access `info.ServerStreamingResult()` before `next`; client receive callbacks
access `info.ClientStreamingResult(result)` after a successful `next` call.
Branch on `info.CallType()` when also accessing ordinary or streaming payloads.
Stream wrappers preserve `SetView` and expose canonical service result types.
When a method declares distinct `Result` and `StreamingResult` types,
`ReadStreamingResult` and `WriteStreamingResult` select fields from
`StreamingResult`. Their server and client accessors read or modify that
streamed value, including its optional fields; unary-only fields are rejected.
Regenerate services and clients, and update interceptor implementations if they
previously relied on a field type from the unary result. Ordinary result access
remains unavailable on a result-streaming method.

Mixed ordinary client calls retain the unary result when stream interceptors
are installed. Stream-only callbacks run only on stream operations; combined
payload/message interceptors still intercept the ordinary payload once and the
stream opening once. Regenerate clients and CLIs to adopt the separate endpoint
chains. HTTP mixed commands use the ordinary endpoint; JSON-RPC SSE commands
open streams. Successful interceptor short circuits must return the selected
endpoint's result type. `Close` continues to delegate cleanup directly.

JSON-RPC server message interceptors follow the generated service operations:
SSE `Send`/`SendAndClose` and WebSocket `SendNotification`/`SendResponse` use
streaming-result access with the supplied context. For decoded WebSocket incoming
messages, call `next`, then use `ServerStreamingPayload` on the returned message
before service dispatch. Request/reply methods retain ordinary `ReadResult` and
`WriteResult`; final-response streaming accessors do not apply to that boundary.
Error sends and cleanup delegate directly. Regenerate to adopt these adapters.

## Installation and Commands

The repository skill tracks Loom `main`; a copy read from a release tag
describes that tagged snapshot. Use the current recommended release for
consuming services unless intentionally testing an unreleased checkout. Loom's
documentation recommendation is curated separately from daily alpha publication.
Select an explicit published tag from the release notes when adopting a newer
alpha, and pin the CLI and module together. `loom version` reports the installed
Go module version; local checkouts and local replacements report `(devel)`.

`loom vet` is available in `v1.9.0-alpha.1` and later releases.

```bash
go install github.com/CaliLuke/loom/cmd/loom@v1.10.0-alpha.5
loom version
loom import openapi openapi.yaml -o design
loom gen <module-import-path>/design
loom example <module-import-path>/design
loom vet <module-import-path>/design
```

## Canonical Guides

For Pulse applications, read `docs/pulse.md`. Use the latest stable Redis
release, currently 8.10.2; older releases are unsupported. Pool protocol 3 requires ending old schedules with `ErrScheduleStop`,
draining/stopping work and completing old-binary pool `Shutdown` before upgrade.
`Close` alone leaves pending old operations. Recreate schedules/jobs afterward;
mixed versions and rolling upgrades are unsupported. `Node.Health(ctx)` reports incompatible live members. `Job.Epoch`
identifies ownership; external stores must enforce fencing themselves. Pool
shutdown retains epoch counters. Scheduler intent survives uncertain admission replies;
`StopAll` selects only that producer's requested keys except the current Start keys.
Queued stops remain responsibility until confirmed; retries retain their original
cutoff so they cannot stop later admitted work. A later authored Start remains a
new command even for the same key/payload; empty plans do not restart completed
admissions. `StopJob` confirms enqueue only. Workers stop handlers after three quarters
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
Failed `AddWorker` calls do not start handlers. Private cleanup can outlive caller
cancellation within the worker shutdown TTL; inspect the returned error for cleanup
failures. Node Close joins in-flight admissions. Existing event expiry still bounds
recovery after uncertain Redis outcomes; see `docs/pulse.md`.
`rmap.Map.Subscribe` coalesces pending invalidations by Reset, Delete, then Change
precedence. Read current map contents rather than counting notifications; a later
change cannot hide an unread reset. Unsubscribe when finished.
A default Reader snapshots the newest cursor when `NewReader` succeeds; events
added after that snapshot remain eligible even before `Subscribe`. `AddStream`
snapshots each new stream independently. These operations can return Redis
lookup errors. Explicit oldest, timestamp and event-ID options retain their
positions without a new lookup.
Removing and re-adding a Reader stream starts a new registration at the chosen
position. Earlier reads cannot move that new cursor. Events already admitted
before removal can still be delivered; removal does not drain subscriptions.
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
