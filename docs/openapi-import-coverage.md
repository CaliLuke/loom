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

## Object coverage

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
| Parameter | Identity, location, requiredness, schema, description, supported style, `allowReserved`, and supported `x-*` | `deprecated`, unsupported examples | `explode`, unsupported styles, and `content` |
| Request Body | `description`, `required`, supported content, and supported `x-*` | None | direct `$ref` plans that cannot retain component identity |
| Media Type | Supported schema and examples | OpenAPI 3.2 `description` | `itemSchema`, encoding fields, `prefixEncoding`, and `x-*` |
| Responses | Concrete status-code entries | None | `default` and `x-*` |
| Response | `summary`, `description`, supported headers and content, and supported `x-*` | None | Direct `$ref` plans and links |
| Header | `description`, `required`, schema, `style: simple`, and `allowReserved` | `deprecated`, unsupported examples | Other serialization, content, direct `$ref`, and `x-*` |
| Security Scheme | API keys, HTTP basic, HTTP bearer, and supported OAuth 2.0 flows and scopes | None | OpenID Connect, bearer formats, device authorization, and incompatible OAuth scope maps |
| Example | JSON-compatible inline, structured, and reusable component examples | Unrenderable examples | External values and incompatible value forms |

Path and operation extensions use a documented allowlist. Unknown extensions
at those supported scopes produce diagnostics. Document, schema, parameter,
request-body, and response extensions preserve JSON-compatible `x-*` values.

## Schema coverage

OpenAPI 3.0 defines a restricted Schema Object. OpenAPI 3.1 and 3.2 base the
Schema Object on JSON Schema 2020-12 and permit added vocabularies. Loom uses
the following closed import subset.

Loom preserves supported forms of:

- `type`, including OpenAPI 3.0 `nullable` and a two-member nullable type union
- `properties`, `required`, and supported `additionalProperties`
- `items`, `minItems`, and `maxItems`
- `minimum`, `maximum`, `exclusiveMinimum`, and `exclusiveMaximum`
- `minLength`, `maxLength`, and `pattern`
- `enum`
- `title`, `description`, `default`, `example`, `examples`, `deprecated`,
  `readOnly`, and `writeOnly`
- JSON-compatible `x-*` extensions

The following constructs are conditional:

- `format` supports known Loom formats. An unknown format requires
  `--allow-lossy`.
- A one-member `allOf` with a local reference is lossless.
- A `$ref` plus inline-object `allOf` requires `--allow-lossy`.
- An inline object used as array items requires `--allow-lossy`. Loom promotes
  it to a deterministic component.
- A two-member `anyOf` that only adds `null` maps to `Nullable()`.
- `additionalProperties: true` maps to `map[string]any` only when the object
  has no declared properties.

Loom rejects other composition and structural keywords, including `oneOf`,
unsupported `anyOf` and `allOf` forms, `not`, tuple and contains constraints,
conditional schemas, dependent schemas, pattern properties, unevaluated
constraints, property-count constraints, `const`, `multipleOf`,
`uniqueItems`, content annotations, XML, schema external documentation,
discriminators, schema dialect controls, anchors, dynamic references, and
local definitions.

A Schema Object with `$ref` siblings is rejected. Use a supported `allOf`
shape when the sibling constraint must remain part of the contract.

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
