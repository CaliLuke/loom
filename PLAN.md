# Typed Generator IR for OpenAPI v3

## Summary
The OpenAPI v3 generator now uses a typed IR layer between `expr` and code
generation. Naming, schema-shape, deduplication, discriminator, and example
decisions live in explicit IR construction instead of ad hoc renderer helpers.
Rendering for OpenAPI remains structured-object based.

Success means:
- OpenAPI v3 generation still produces byte-for-byte equivalent specs or intentionally equivalent structured output with existing tests updated only where formatting requires it.
- `schemafy`-style logic is no longer the place where both analysis and rendering decisions are mixed.
- The IR is reusable by adjacent generator code without redesign.

## Current Status

Current implementation:
- Typed OpenAPI IR package under `http/codegen/openapi/internal/ir` covering
  schemas, components, request bodies, responses, headers, parameters, media
  types, examples, and operation metadata.
- Analyzer-driven body/schema component construction plus renderer-driven
  `openapi.Schema` materialization.
- IR-backed OpenAPI document construction for request/response body/content
  data plus route-scoped operation metadata, parameter modeling, and reusable
  component hoisting.
- `http/codegen/openapi/v3` now consumes the IR for endpoint operation shape
  and component reuse instead of carrying separate legacy builders for
  parameters, responses, and post-render component identity.
- Go-source generator sections are implemented in Go through the shared
  section model (`codegen.Section`, `codegen.JenniferSection`,
  `codegen.RawSection`) rather than file-backed Go template assets.
- Non-Go text artifacts still use template rendering where it is the simplest
  fit, and those assets use neutral `.tmpl` filenames.

Remaining follow-up:
- Keep expanding typed IR reuse where it removes remaining ad hoc OpenAPI
  renderer decisions.
- Keep the Go-source generator architecture on the shared section model and use
  typed Go emission for new logic-heavy sections by default.

## Implementation Changes
### 1. Add a generator IR package for HTTP/OpenAPI schema and endpoint description
Create a new internal codegen model layer for OpenAPI-oriented generation, likely under `http/codegen/openapi/internal` or a new sibling package dedicated to generator models.

The IR must define explicit typed nodes for:
- API document model: paths, operations, request bodies, responses, content, components.
- Schema model: primitive, array, object, map/additional-properties, union/discriminator, reference, inline schema, examples, validations.
- Component identity model: canonical component name, structural hash, explicit-name claim, deduplication key, union branch key.
- Endpoint body model: request body plus response bodies by status/view.

The IR must be Go-typed and renderer-neutral:
- No `text/template` fields.
- No direct `openapi.Schema` values stored inside the IR.
- No implicit reuse of `expr.AttributeExpr` as the IR itself.

### 2. Split OpenAPI v3 generation into analysis and rendering passes
Refactor the current OpenAPI v3 builder into two phases:

Phase A: IR construction from `expr`
- Convert current `buildBodyTypes` and `schemafier` logic into an analyzer that walks `expr` and emits IR nodes.
- Centralize all component naming decisions here:
  - explicit `openapi:typename`
  - canonical name claims
  - structural deduplication
  - union envelope naming
  - `noref` and component registration policy
- Centralize all schema semantics here:
  - primitive/format mapping
  - object close/open behavior
  - map/additionalProperties policy
  - union discriminator/property mapping
  - example suppression and canonicalization
  - validation projection into schema fields

Phase B: IR rendering to current OpenAPI structs
- Add a pure renderer that converts IR nodes into `openapi.Schema`, `MediaType`, `RequestBodyRef`, responses, and the final document shape.
- Keep `http/codegen/openapi/v3/files.go` as a thin serialization layer over the final OpenAPI document object.
- Rendering must not decide names, dedupe, or inspect DSL metadata beyond what the IR already resolved.

### 3. Keep the OpenAPI IR responsibility bounded and explicit
Do not broaden the IR without a clear contract need. Keep the current split
centered on body/schema generation, operation metadata, and reusable component
registration.

In scope:
- Request/response body schema construction.
- Component schema registration and reuse.
- Union/discriminator and envelope schema generation.
- Example and validation projection as used by body schemas.

Out of scope:
- Rewriting JSON/YAML serialization.
- Replacing non-OpenAPI artifact generators that already have an adequate
  contract surface.
- Expanding the IR into unrelated transport/runtime concerns.

### 4. Preserve and formalize compatibility behavior
The implementation must preserve the current behavior for:
- Explicit OpenAPI typenames and canonical body component names.
- Structural deduplication of generated request/response body schemas.
- Distinct explicit names when schemas differ.
- Panic on conflicting explicit component names.
- Closed-object mode behavior for objects, maps, and unions.
- Stable union discriminator mappings and stable envelope refs.
- Current example generation/suppression semantics.

Where behavior is currently implicit, promote it into named IR policies/helpers so future generators do not need to rediscover it.

### 5. Keep the generator architecture consistent
For follow-up generator work:
1. Extend the same IR concepts to adjacent OpenAPI-related analysis where it
   removes duplicated contract logic.
2. Keep Go-source generation on the shared section model.
3. Use typed Go emission by default for new logic-heavy generator sections.
4. Keep raw AST limited to finalization or narrow syntax-sensitive cleanup.

## Public API / Interface Additions
The plan should keep public DSL and generated surface behavior unchanged.

New internal interfaces/types to add:
- A typed IR model for OpenAPI generation.
- An analyzer interface or constructor that converts `expr` to IR.
- A renderer interface or package function that converts IR to OpenAPI v3 structs.

No user-facing DSL changes.
Internal dependency used by typed Go emitters:
- `github.com/dave/jennifer/jen`

## Test Plan
Add direct tests for the IR analyzer and renderer split, while keeping current end-to-end coverage.

Required tests:
- Analyzer unit tests for schema identity, canonical naming, explicit name conflicts, and deduplication.
- Analyzer unit tests for union branch naming and discriminator mapping.
- Renderer unit tests proving IR object/array/map/union/reference nodes render to the expected OpenAPI structs.
- Existing OpenAPI v3 package tests must continue to pass, especially:
  - explicit typename reuse and conflict tests
  - request body structural dedup tests
  - repeated union envelope dedup tests
  - closed-object mode tests
  - example/discriminator tests
- Golden/spec tests remain the final compatibility check.

Acceptance criteria:
- `go test ./http/codegen/openapi/v3` passes throughout.
- Any new IR package has focused seam tests that do not depend on full DSL runs unless required.
- No generated spec regressions beyond formatting-only differences.
- The typed Go-emitter path has focused seam coverage and the relevant
  transport-generation packages pass direct section checks plus the
  temp-module/local-source loop where applicable.

## Assumptions and Defaults
- OpenAPI v3 remains the canonical IR-backed contract generator in this plan.
- The Go-source generator architecture is already standardized on the shared
  section model.
- The OpenAPI IR remains renderer-neutral.
- Future Go-source generator work should default to typed Go emission, not raw
  `go/ast`.
- Raw AST remains acceptable only for narrow cleanup/finalization cases, similar to the current import/format handling.
