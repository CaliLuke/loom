# Typed Generator IR Pilot for OpenAPI v3

## Summary
Introduce a typed generator IR layer between `expr` and code generation, using OpenAPI v3 as the first vertical slice. The pilot will move naming, schema-shape, deduplication, discriminator, and example decisions out of ad hoc renderer helpers into explicit IR construction. Rendering for OpenAPI remains structured-object based. Future Go-emitter migrations are planned to target `jennifer` by default, but that decision is deferred from the pilot.

Success means:
- OpenAPI v3 generation still produces byte-for-byte equivalent specs or intentionally equivalent structured output with existing tests updated only where formatting requires it.
- `schemafy`-style logic is no longer the place where both analysis and rendering decisions are mixed.
- The new IR is reusable by at least one additional generator after the pilot without redesign.

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

### 3. Keep the first migration bounded to OpenAPI v3 body/schema generation
Do not migrate all of OpenAPI v3 at once. Limit the pilot to the code path currently centered on body/schema generation and component registration.

In scope for the pilot:
- Request/response body schema construction.
- Component schema registration and reuse.
- Union/discriminator and envelope schema generation.
- Example and validation projection as used by body schemas.

Out of scope for the pilot:
- Rewriting JSON/YAML serialization.
- Replacing non-OpenAPI Go code emitters.
- Converting template-heavy HTTP/gRPC/JSON-RPC Go generation.
- Introducing `jennifer` or raw AST in this pilot.

### 4. Preserve and formalize compatibility behavior
The pilot must preserve the current behavior for:
- Explicit OpenAPI typenames and canonical body component names.
- Structural deduplication of generated request/response body schemas.
- Distinct explicit names when schemas differ.
- Panic on conflicting explicit component names.
- Closed-object mode behavior for objects, maps, and unions.
- Stable union discriminator mappings and stable envelope refs.
- Current example generation/suppression semantics.

Where behavior is currently implicit, promote it into named IR policies/helpers so future generators do not need to rediscover it.

### 5. Define the staged migration path after the pilot
After OpenAPI v3 is stable on the IR:
1. Extend the same IR concepts to shared schema/body analysis used by other OpenAPI-related paths.
2. Add a second pilot for a Go emitter that currently has high template complexity, with HTTP server/client generation the preferred candidate.
3. For that later Go-emitter pilot, adopt `jennifer` as the default rendering backend for new code paths while keeping raw AST limited to post-processing or narrow syntax-sensitive cases.
4. Retire template/business-logic mixing incrementally, not in a single rewrite.

## Public API / Interface Additions
The plan should keep public DSL and generated surface behavior unchanged.

New internal interfaces/types to add:
- A typed IR model for OpenAPI generation.
- An analyzer interface or constructor that converts `expr` to IR.
- A renderer interface or package function that converts IR to OpenAPI v3 structs.

No user-facing DSL changes.
No new repo dependency in the pilot.
Do not add `jennifer` yet.

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

## Assumptions and Defaults
- The first pilot is OpenAPI v3 only.
- The plan includes both the pilot implementation shape and the staged migration path for the broader framework.
- The pilot is renderer-neutral and does not choose `jennifer` or raw AST.
- Future Go-emitter migrations should default to `jennifer`, not raw `go/ast`.
- Raw AST remains acceptable only for narrow cleanup/finalization cases, similar to the current import/format handling.
