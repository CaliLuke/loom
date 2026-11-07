---
title: OpenAPI Import Coverage
weight: 4
description: "The OpenAPI 3.0, 3.1, and 3.2 constructs that Loom preserves, conditionally imports, omits, or rejects."
llm_optimized: true
aliases:
---

# OpenAPI Import Coverage

Loom does not import the full OpenAPI grammar. It imports a strict subset and
fails when it cannot preserve a contract. This page defines that boundary for
OpenAPI 3.0, 3.1, and 3.2.

The coverage contract uses four terms:

- **Preserved** means Loom imports the construct and regenerates its contract.
- **Conditional** means support depends on the value, location, or related
  fields. Loom reports unsupported variants.
- **Lossy** means Loom imports only with `--allow-lossy` and reports the
  omitted or normalized detail.
- **Rejected** means Loom reports a blocking diagnostic. It does not silently
  discard the construct.

If Loom rejects a parent object, fields inside that object are outside the
import boundary. For example, Loom rejects callbacks, so it does not claim
field-level import coverage for Callback Objects.

## Specification baseline

The importer accepts documents from these feature lines:

- [OpenAPI 3.0.4](https://spec.openapis.org/oas/v3.0.4.html)
- [OpenAPI 3.1.1](https://spec.openapis.org/oas/v3.1.1.html)
- [OpenAPI 3.2.0](https://spec.openapis.org/oas/v3.2.0.html)

The test ledger records the union of fixed fields from the published
[3.0 schema](https://spec.openapis.org/oas/3.0/schema/2024-10-18),
[3.1 schema](https://spec.openapis.org/oas/3.1/schema/2025-11-23), and
[3.2 schema](https://spec.openapis.org/oas/3.2/schema/2025-11-23).

Loom also checks version boundaries. A 3.0 or 3.1 document cannot use fields
that the OpenAPI specification first defines in 3.2.

YAML aliases and merge keys on the `components` and `components.schemas`
collections preserve local `$ref` lookup and the effective component names.
Local fields override inherited fields, and the first merge source wins when
sources overlap. Import keeps the original source locations while reading those
effective values. Re-import documents previously rejected because an inherited
component could not be found, then regenerate the design.

Schema-level aliases and merge keys obey the same effective-value rules.
Equivalent explicit, aliased and merged schemas produce the same design,
including null-only alternatives and aliased boolean schemas. Inherited unknown
keywords and constrained or annotated null alternatives remain fatal errors,
even with `--allow-lossy`. Re-import and regenerate designs affected by these
previously rejected forms or by an aliased `false` schema that escaped rejection.

## Object coverage

Imported schema and operation declarations, security credentials, and HTTP
parameter/header mappings receive distinct Go identifiers when their names
collide, including collisions with existing numbered names. These allocations
retain original schema references and wire names. Re-import and regenerate a
design affected by duplicate declarations or mapped fields in these locations.

Object properties also receive distinct Go field names when keys such as `@type`
and `type`, or `foo-bar` and `fooBar`, normalize to the same identifier. The
imported DSL records these names with `struct:field:name`; JSON keys remain
unchanged. Allocation includes occupied numeric suffixes, inherited properties,
and reserved error methods. A derived error's override stays on that declaration
so other uses of the base keep their field names and constraints. Re-import and
regenerate affected designs, then use the generated Go selectors in service code.
This does not assign protobuf tags or change protobuf field-name admission.

Multiple request and response media types retain independent schemas, examples,
and encoding contracts. They render as typed service unions with HTTP
`MediaType(media, branch)` mappings, including equal schemas under different
media types. Exact types outrank subtype and global ranges. Ranges expose a mapped
concrete Content-Type field. The generated client/server select a codec for the
selected branch; unknown codecs require an application factory. Single-media
behavior is unchanged. See [HTTP media contracts](http-media-contracts.md).

The following table summarizes fixed fields on objects that the importer
reads. Specification extensions are patterns, not fixed fields.

| Object | Preserved or conditional | Lossy | Rejected |
|---|---|---|---|
| OpenAPI | `openapi`, `info`, `paths`, supported `components`, `security`, `tags`, supported `x-*` | `externalDocs` | `servers`, `jsonSchemaDialect`, `$self`, `webhooks` |
| Info | `title`, `description`, `version`, supported `x-*` | `summary`, `termsOfService`, `contact`, `license` | None |
| Tag | `name`, `summary`, `description`, `parent`, `kind`, supported `externalDocs`, and `x-*` | None | Extensions inside `externalDocs` |
| Paths | Path entries | None | `x-*` |
| Path Item | Standard methods, `QUERY`, additional methods, `parameters` | `summary`, `description` | `$ref`, `servers`, `x-*` |
| Operation | `tags`, `summary`, `description`, `operationId`, `parameters`, `requestBody`, `responses`, `deprecated`, `security`, supported `x-*` | `externalDocs` | `callbacks`, `servers` |
| Components | Supported schemas, parameters, request bodies, responses, headers, security schemes, and examples | None | Links, callbacks, path items, media types, and `x-*` component entries |
| Parameter | Identity, location, requiredness, schema, description, supported style and `explode` combinations, `allowReserved`, named query-map JSON `content`, and supported `x-*` | `deprecated`, unsupported examples | Incompatible serialization or content |
| Request Body | `description`, `required`, supported content, and supported `x-*` | None | unresolved, cyclic, or wrong-kind `$ref` plans |
| Media Type | Distinct per-media schemas and examples, ranges, admitted styled and JSON form field encoding, and multipart part content types | OpenAPI 3.2 `description` | `itemSchema`, unsupported encoding fields, `prefixEncoding`, and `x-*` |
| Responses | Concrete status-code entries | None | `default` and `x-*` |
| Response | `summary`, `description`, supported headers and content, and supported `x-*` | None | Unresolved, cyclic, or wrong-kind `$ref` plans and links |
| Header | `description`, `required`, schema, `style: simple` with either `explode` value, and `allowReserved` | `deprecated`, unsupported examples | Other serialization, content, unresolved/cyclic/wrong-kind `$ref`, and `x-*` |
| Security Scheme | API keys, HTTP basic, HTTP bearer, and supported OAuth 2.0 flows and scopes | None | OpenID Connect, bearer formats, device authorization |
| Example | JSON-compatible inline, structured, and reusable component examples | Unrenderable examples | External values and incompatible value forms |


Security schemes that use the same credential location and encoding share one
payload field after import. For example, HTTP bearer and OAuth alternatives both
read the same bearer token from `Authorization`; their authentication checks and
OAuth scopes remain separate. Basic authentication retains its username and
password fields. API-key schemes share a field only when their location and wire
name match (header names ignore case; query and cookie names do not). Re-import
the OpenAPI document, regenerate transports, and use the single credential field
in application code. Duplicate credentials no longer become unrelated request
body fields.

Each security requirement object is a conjunction; separate objects remain
alternatives. Import preserves each scheme's independent array, including empty
arrays and overlapping OAuth scope inventories. For example,
`[{Bearer: [admin], OAuth: [read]}]` requires both the application's `admin`
role and the OAuth `read` scope. Import renders `ScopeFor` for OAuth scopes and
`RoleFor` for non-OAuth role names. Roles have no protocol-defined inventory;
unknown OAuth scopes still fail closed. Non-OAuth role arrays require OpenAPI
3.1 or 3.2; OpenAPI 3.0 requires those arrays to be empty. Anonymous alternatives and explicit
operation overrides remain distinct. Partial import skips operations that
inherit an invalid root requirement while retaining public or secured overrides.
The retained design includes root security only when a retained operation
inherits it. `security_per_scheme_test.go` and the generated credential fixture
cover per-scheme fidelity and runtime enforcement.

OAuth flows retain their own scope names and descriptions, including disjoint,
overlapping and empty inventories. Import emits a flow-local Scope block for
each flow. Regeneration preserves the exact flow maps; requirement validation
checks names across the effective inventories without assigning token grants.
`security_flow_scopes_test.go` covers the imported and regenerated contract.

Path and operation extensions use a documented allowlist. Unknown extensions
at those supported scopes produce diagnostics. Document, schema, parameter,
request-body, and response extensions preserve JSON-compatible `x-*` values,
and regenerated OpenAPI emits each value at the scope it came from.

Parameter import uses the same codec admission as generated transports. Paths
support `simple`, `label` and `matrix`; query and URL-encoded form fields support
`form`, `spaceDelimited`, `pipeDelimited` and `deepObject`; request and response
headers support `simple`. These grammars admit scalars, scalar arrays, fixed
objects with scalar fields and scalar maps. References use the same rules.
`deepObject` requires an object or map. Space/pipe grammars reject scalars and
`explode: true`. Nested style values, nullable text values and text `Any` remain
blocking errors. Text unions are not yet admitted.

OpenAPI defaults normalize before codec selection. Thus a schema-based query map
with omitted style imports as an exploded `form` map, while a named map using
`content: {application/json: {schema: ...}}` retains JSON. JSON map content also
supports nested maps and arrays inside maps. Regenerated DSL emits only the
metadata needed to distinguish the selected codec from Loom's defaults.
Content-based parameters cannot also declare style, explode or allowReserved.

Named object/map properties of URL-encoded forms default to JSON. Explicit
`contentType: application/json` has the same effect when no encoding flags are
present. Any explicit `style`, `explode` or `allowReserved` selects the style
rules, overriding contentType. This rule applies separately within multi-media
branches. Root form maps and raw-body imports retain their own contracts.

Cookie parameters support raw OpenAPI 3.2 `cookie` style and percent-encoded
`form` style. Form collections require `explode: false`; raw cookies can also
explode arrays and objects into cookie pairs. Unsupported raw-cookie exports to
3.1 fail rather than silently changing their encoding. See the
[HTTP guide](http-guide.md#parameter-serialization) for the complete wire rules.

Namespace admission rejects overlapping exploded properties, overlapping
`deepObject` prefixes and multiple maps claiming remaining names. Credential
bindings participate in those checks. Invalid shapes, styles and namespaces
block design publication, including with `--allow-lossy`; direct `Render`
performs the same admission. Partial imports can exclude invalid operations.

For JSON query maps, omitted `additionalProperties` on a memberless object and
explicit `additionalProperties: true` select the same map carrier. A closed
memberless record (`additionalProperties: false`) remains a fixed object;
it is not silently converted to a JSON query map.

In OpenAPI 3.1 and 3.2, a parameter reference's `description` overrides the
component description for that occurrence, including an explicitly empty
string. Other occurrences retain the component description. Regeneration may
use separate parameter components to preserve these effective descriptions.

Reusable request bodies, responses, and headers retain their component names and
forward alias references. In OpenAPI 3.1 and 3.2, Reference Object descriptions
and summaries remain occurrence annotations, including explicitly empty strings;
shared definitions are unchanged. Summaries have an effect only on target object
kinds that support them. OpenAPI 3.0 ignores Reference Object siblings.

Referenced definitions must fit the same media, schema, and header subset as
inline objects. Unused definitions, cyclic aliases, unresolved references, and
references to the wrong component kind remain blocking errors. Regenerate an
imported design to obtain preserved component identities and annotation edges.

An extension at an unsupported scope is a `vendor-extension` diagnostic. Strict
import fails before it writes output. With `--allow-lossy`, the importer omits
the extension and reports it as a warning.

Schema descriptions are independent of parameter and header descriptions. Loom
preserves both when the source supplies both, including descriptions on array
items and map values. Imported location attributes use ordinary `Description(...)`
for the parameter or header and `Meta("openapi:description:schema", ...)` for the
schema; an empty metadata value keeps the schema description empty. Other schema
attributes use `Description(...)` directly. A supported reference overlay retains
its local annotation without changing the referenced component. Re-import and
regenerate to restore descriptions dropped by previously generated designs.

## Schema coverage

OpenAPI 3.0 defines a restricted Schema Object. OpenAPI 3.1 and 3.2 base the
Schema Object on JSON Schema 2020-12 and permit added vocabularies. Loom uses
the following closed import subset.

Loom preserves supported forms of:

- explicit `type`, including OpenAPI 3.0 `nullable` and an OpenAPI 3.1 or 3.2
  two-member nullable type union
- unconstrained `{}` schemas and OpenAPI 3.1 or 3.2 boolean `true` schemas
- `properties`, `required`, and supported `additionalProperties`
- `items`, `minItems`, and `maxItems`
- `minimum`, `maximum`, `exclusiveMinimum`, and `exclusiveMaximum`
- `minLength`, `maxLength`, and `pattern`
- `enum`
- `title`, `description`, `default`, `example`, `examples`, `deprecated`,
  `readOnly`, and `writeOnly`
- JSON-compatible `x-*` extensions

An example is kept only when its value belongs to the complete normalized
schema that carries it: every local assertion, the referenced target, each
inherited object base and exactly one `oneOf` branch. This holds for schema,
media and referenced component examples regardless of component declaration
order, including recursive and promoted object schemas. Reference siblings such
as `minLength` beside `$ref` constrain the example in addition to the target.
Null belongs only through a nullable type, a nullable wrapper such as
`anyOf: [{$ref: ...}, {type: 'null'}]`, a nullable target, or exactly one
`oneOf` branch. An example keeps the domain of the schema it was authored on
when normalization flattens that schema: `example: null` on the string branch
of `anyOf: [{type: string}, {type: 'null'}]` is rejected, while the same
example on the wrapper is kept. A closed `oneOf` (`unevaluatedProperties:
false`) keeps an object example only when the single matching branch evaluated
every member: its declared properties, explicit `additionalProperties`, or the
properties of its references and inherited bases. A property declared only by a
non-matching branch does not count. Other examples receive a lossy `examples`
diagnostic at their authored location and are omitted with `--allow-lossy`.

Imported patterns must preserve ECMAScript acceptance in Loom's Go validators.
The portable subset supports ordinary BMP Unicode literals, positive character
classes, groups, alternatives, quantifiers, anchors, ASCII digit/word classes and
word boundaries. Unicode escapes and ECMAScript whitespace classes are translated
without changing their accepted values: `\s` includes NBSP and vertical tab.
Escaped metacharacters remain literals, including inside character classes.

Lookaround, backreferences, flags, named captures, wildcard/complement classes,
non-BMP/surrogate characters and unsupported escapes receive a fatal `schema-pattern`
diagnostic at the source pattern. Ambiguous class ranges, empty classes and
leading-zero repetition bounds are also outside this subset. `--allow-lossy`
cannot remove a pattern constraint. Re-import and regenerate affected designs;
previous imports could fail design evaluation or silently use different whitespace
semantics. Authored Go DSL patterns retain their existing Go semantics.

Numeric bounds retain the original JSON numeric tokens and YAML numeric scalar
values, including hexadecimal, octal and binary integer spellings. Quoted
numbers are not numeric constraints. Imported bounds are normalized exactly into
DSL string arguments so compiling a design cannot round them through an untyped
Go floating-point constant. A valid source number outside the DSL's finite
admission range, such as `1e309`, produces a diagnostic at the bound's source
path; `--allow-lossy` does not drop that constraint. Native floating-point
precision remains separate from the exact schema number (see
[numeric validations](dsl-reference.md#numeric-validations)).

Finite scalar domains intersect `enum` or `const` with the declared type, including
nullability. A nullable string with `enum: [a, null]` retains null; the same type
with `enum: [a]` does not. Members excluded by the source type are removed, not
coerced. Empty or null-only intersections are blocked because the DSL cannot
represent them. An authored empty enum is never treated as absent validation.
Numeric members retain source identity before carrier conversion: a fractional
member of an integer enum is excluded even if floating-point rounding would make
it integral. An included member that cannot retain its numeric value in the DSL
carrier produces a blocking diagnostic rather than rounding or disappearing.

Re-import and regenerate byte-formatted schemas to adopt the corrected string
representation. Imported application fields change from `[]byte` to `string`;
use `encoding/base64` when the application needs decoded bytes. Native authored
Loom `Bytes` continues to use decoded-byte constraints and its existing JSON codec.

Local component references decode URI percent escapes before JSON Pointer token
escapes. For example, `#/components/schemas/%49temName` resolves to the legal
component key `ItemName`, just like its unencoded spelling. Regeneration uses
canonical references. This does not relax OpenAPI's component-key naming rules
or add external-document reference support.

The following constructs are conditional:

- Recursive local component references through object fields, arrays, and maps
  retain their named targets. This includes aliases anchored in an object graph
  and recursive schemas reused by error responses. Import rejects a cycle made
  entirely from immediate type constructors, such as an array whose items refer
  to that same array, with `schema-reference-initialization`. The diagnostic is
  blocking even with `--allow-lossy`; it describes the current import subset,
  not a restriction on JSON Schema itself. Pure alias cycles remain rejected
  by reference resolution.
- `format` supports known Loom formats. An unknown format requires
  `--allow-lossy`. A string with `format: byte` imports as `String` with
  `Format(FormatBase64)`. Lengths, patterns, defaults, enums and examples apply
  to the encoded text. Standard padding is required; whitespace and invalid
  padding are rejected. Decodable alternate pad bits are accepted, but enum
  equality still compares exact text: an enum containing `"aGk="` rejects
  `"aGl="`, even though they decode to the same bytes.
- `format: binary` retains `Bytes` only for a whole raw response body or whole
  multipart part. JSON values, text parameters and headers cannot represent raw
  octets and receive a blocking `schema-binary-location` diagnostic. Whole raw
  request bodies remain outside the existing import subset.
- A one-member `allOf` with a local reference is lossless. Numeric bounds and
  compatible scalar defaults are preserved on the reference occurrence; other
  contract siblings remain rejected.
- A local `$ref` plus an inline object in `allOf` preserves the reference and
  the object intersection. It does not require `--allow-lossy`.
- An inline object used as array items requires `--allow-lossy`. Loom promotes
  it to a deterministic component.
- A two-member `anyOf` with a bare `{type: null}` branch maps to `Nullable()`.
  Assertions or annotations on the null branch remain unsupported. When the
  value branch is `{}`, it normalizes to `Any` because the empty schema already
  accepts null. Outer nonempty titles and descriptions take precedence over
  branch labels. Positive `deprecated`, `readOnly` and `writeOnly` annotations
  survive from either layer; an outer false does not cancel a positive branch
  annotation. Vendor extensions retain branch keys, with outer keys taking
  precedence. Re-import and regenerate affected designs to recover annotations
  lost by earlier importer versions.
- `oneOf` normalization counts null admission across its alternatives. It does
  not turn multiple null matches into an accepted null. Bare-null alternatives
  can use the existing nullable representation when that preserves the accepted
  domain; annotated or constrained null-only schemas remain unsupported.
- A scalar `const` without a sibling `enum` maps to a one-member enum.
  Regeneration uses the semantically equivalent `enum` representation.
- Supported `oneOf` schemas map to untagged typed unions. Branches may be
  primitives, unconstrained values, aliases, arrays, string-keyed maps, objects,
  or nested untagged unions, including nullable alternatives. Inline object
  shapes receive concrete named declarations where required; scalar and
  collection branches retain their constraints without synthetic wrappers.
  Overlapping alternatives are legal. Each input or emitted value must have
  exactly one schema match and one valid Go candidate, with the same identity.
  Zero or multiple matches report their count and domain; different unique
  matches report an identity mismatch. Go numeric widths cannot disambiguate an
  overlapping public schema.
- JSON bodies, nested fields and error envelopes use the JSON union contract.
  Path/query/header/cookie bindings and form or multipart fields additionally
  require a codec for every alternative at that location. The parameter style
  contract remains owned by [#968](https://github.com/CaliLuke/loom/issues/968).
  Form roots admit supported object/map alternatives; multipart union roots
  require objects. Text cannot encode nullable values; JSON-valued fields can.
  Discriminator-driven import remains separate work in
  [#1005](https://github.com/CaliLuke/loom/issues/1005).
- `unevaluatedProperties: false` beside a supported union closes the selected
  value after unique schema/decoder selection. It cannot make overlapping
  alternatives unambiguous by removing unknown properties first. Closure uses
  the selected schema's evaluated-member annotations, including nested unions
  and additional-member policies. Other unevaluated-property policies and
  general assertion intersections remain unsupported.
- A memberless `type: object` maps to `map[string]loom.JSONValue` when
  `additionalProperties` is absent, true or an empty schema. Closed empty objects
  (`additionalProperties: false`) retain their object type and accept only `{}`,
  plus null when separately nullable. Inline closed-empty fields and structured
  object map values receive deterministic named types so every DSL constructor
  has a concrete type. Existing named objects retain their names. Reimport and
  regenerate to replace earlier String fields or incomplete map constructors.
- An object that declares `properties` and also has schema-valued or `true`
  `additionalProperties` maps to a type with `AdditionalAttributes`. A
  `required` name that `properties` does not declare is supported when
  `additionalProperties` describes its value, including an object with only
  `required` and `additionalProperties`; an open object without
  `additionalProperties` uses `AdditionalAttributes(Any)` for it only when it
  has no `allOf` base. Beside an `allOf` base, an undeclared required name needs
  an `additionalProperties` policy of its own or one inherited from a base,
  including `true` or an empty schema. A form body with such an object, or
  with an object that inherits such a policy through `allOf`, stays a raw body
  (`SkipRequestBodyEncodeDecode()`), because typed form decoding has no carrier
  for undeclared members. URL parameters and URL-serialized form fields do not
  support these objects either.
- A property whose Go field name is `AdditionalAttributes`, the field that
  carries undeclared members, keeps its JSON name and receives a distinct Go
  field name through `struct:field:name`. Undeclared required names remain
  literal keys in the additional-members map, so names such as `foo`, `Foo`,
  `a:b`, `a,b`, and `AdditionalAttributes` do not compete for Go field names.
- A form request whose object has only schema-valued `additionalProperties`
  maps to a typed root map. A body-only optional form also emits
  `OptionalRequestBody()`.
- A form map combined with transport parameters uses an object payload and a
  selected `Body(...)` attribute. Optional single-media multipart maps use
  the documented raw-body contract. Multi-media multipart variants require
  structured objects; form variants use the supported form-root codec.
- A single-media form schema containing an unconstrained value, including a free-form
  `additionalProperties: true` map, stays raw because typed form decoding
  cannot reconstruct JSON-valued data.
- An optional single-media form object with required members also stays raw so an absent
  body does not spuriously fail nested required-field validation.

Loom rejects boolean `false` schemas, constrained schemas without an explicit
type (including typeless `properties`), and the legacy `nullable` keyword in
OpenAPI 3.1 or 3.2. These are blocking diagnostics even with `--allow-lossy`.
Analysis and rendering retain the same admission rules for selected operations
and their component closure.

Loom rejects other composition and structural keywords, including unsupported
`oneOf` locations or branch shapes, unsupported `anyOf` and `allOf` forms,
`not`, tuple and contains constraints,
conditional schemas, dependent schemas, pattern properties, unevaluated
constraints other than the union closure described above, property-count constraints, `multipleOf`,
`uniqueItems`, content annotations, XML, schema external documentation,
discriminators, schema dialect controls, anchors, dynamic references, and
local definitions.

Structured `const` values and schemas that combine `const` with `enum` are also
rejected.

OpenAPI 3.1 and 3.2 Schema Objects can place supported scalar assertions,
array length bounds, required property names and annotations beside a local
`$ref`. Loom retains the reference and occurrence constraints independently:
`{$ref: '#/components/schemas/Name', minLength: 2}` enforces both the Name
contract and the extra minimum, without changing other uses of Name. A
required name must be a property of the referenced object or of its `allOf`
bases, unless the referenced object has `additionalProperties` that is a
schema or `true`. A policy inherited from an `allOf` base also qualifies,
including `true` or an empty schema, which Extend keeps as an `Any` carrier. A
required name is rejected when any source of the object sets
`additionalProperties: false`. Regenerate imported designs to pick
up corrected validation of optional fields required by a named occurrence.

Object `properties` beside a reference refine the inherited fields and can add
fields to an open object. Omitted property types use the inherited carrier.
Regeneration retains the base reference in `allOf` alongside the composed
object. A closed base cannot gain fields, and a closed sibling must declare all
inherited property names; these checks prevent changing the accepted key set.
A sibling with schema-valued `additionalProperties` constrains the inherited
properties it does not declare: the importer redeclares each of them with that
schema as a refinement and gives the object `AdditionalAttributes`. Over a
closed base, that policy can only apply to keys the base already rejects, so
the importer keeps the redeclarations, drops the policy, and the object
inherits the closure. Schema-valued `additionalProperties` on the base
constrains every property that only the sibling declares: such a property
takes its omitted type and format from that policy, and a property whose type
or numeric format differs from the policy is reported. The same rule applies
to `allOf` with one reference and one inline object. A referenced property, or
an array property with its own `items`, keeps its own type, so that type must
be the policy's type or a named alias of it. When both the sibling and the
base have schema-valued `additionalProperties`, one policy type must refine
the other: the sibling's policy must be the base policy's type or a named
alias of it, and it cannot admit null when the base policy does not. A
non-nullable sibling policy over a nullable base policy is accepted. Policies
whose JSON kinds are disjoint, such as integer and
string, admit no undeclared member, so the object is closed. Overlapping
policies where neither type refines the other, such as integer and number,
`int32` and `int64`, or two different inline objects, are reported. A
policy of `true` or an empty schema constrains nothing and passes these
checks. These diagnostics use the `schema-reference-siblings` code.

Reference-level `items`, typed `additionalProperties` on property
refinements, arbitrary composition,
type changes, removal of inherited nullability and changes to numeric carrier
formats remain blocked. `--allow-lossy` cannot bypass these boundaries. OpenAPI
3.0 direct reference siblings remain unsupported.

OpenAPI 3.1 and 3.2 permit custom JSON Schema vocabularies. Loom rejects every
unknown non-`x-*` schema keyword because it cannot prove that the keyword is
only an annotation. This rule prevents silent loss when a contract uses a
vocabulary that the parser or importer does not know.

## How coverage stays current

Importer tests enforce the boundary in several ways:

- A fixed-field ledger classifies every field exposed by the parser as
  preserved, conditional, lossy, rejected, or parser-only.
- A second ledger compares parser fields with the official OpenAPI 3.0, 3.1,
  and 3.2 schema fields. A standard field missing from the parser needs an
  explicit raw-source guard.
- Raw-source guards currently cover the OpenAPI 3.2 Media Type fields that the
  parser does not expose.
- A schema-keyword matrix requires every known unsupported keyword to produce
  a diagnostic. Unknown non-`x-*` keywords also fail closed.
- JSON and YAML canonical fixtures cover the same supported contract.
- Location matrices exercise reusable and inline schemas in every supported
  component and operation location.
- Cross-product tests cover response selection, path-parameter identity,
  reference wrappers, and every renderable schema kind.
- An exporter-symmetry fixture imports every supported exporter shape. It then
  compiles the generated service and compares the regenerated OpenAPI contract.

This gives Loom full **classification coverage** for the declared import
boundary. It does not mean that Loom imports every OpenAPI construct.
