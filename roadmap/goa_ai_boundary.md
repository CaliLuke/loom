# Goa-AI Boundary Cleanup

## Goal

Move generic API-contract and transport semantics into `goa-light`, keep MCP/agent/runtime concerns in `goa-ai`, and remove application-specific leaks from generic framework generators.

## Status

### Completed

- move generic union discriminator tag handling into `goa-light`
- move generic OpenAPI contract stability policy into `goa-light`
- move generic auth/session transport modeling into `goa-light`
- move temporary root preparation/finalization helpers into `goa-light`
- move synthesized JSON-RPC HTTP service construction into `goa-light`
- move deterministic temporary user-type registration into `goa-light`
- move remediation-aware error metadata into `goa-light`
- move inline JSON Schema generation into `goa-light`
- move generic attribute import gathering into `goa-light`

### Next

1. Rewire `goa-ai` to consume the core helpers already moved into `goa-light`.
2. Delete or collapse duplicated helpers in `goa-ai/codegen/shared`.
3. Verify the remaining `goa-ai` shared code is MCP-specific only.
4. Remove direct app-specific framework coupling once the generic Goa-core hooks exist.
5. Leave MCP annotations, tool runtime, planners, and registries in `goa-ai`.

For the full end-to-end finish gate, see [Finish Checklist](./finish_checklist.md).

## Migration Plan

### 1. Union Discriminator and Schema Semantics

- Move any remaining generic union wire-tag behavior out of `goa-ai/codegen/shared`.
- The canonical source of truth for variant wire tags, discriminator values, and union example normalization should live in `goa-light`.
- `goa-ai` should only add MCP-specific wrapping on top of Goa’s generic union model.
- Goal: no MCP generator should need its own fallback copy of generic union-tag logic.

### 2. JSON Schema Example Canonicalization

- Move generic example canonicalization for unions and nested Goa types into `goa-light` where it can benefit HTTP, JSON-RPC, OpenAPI, and any future generators.
- Keep only MCP-specific final shaping in `goa-ai` when the protocol requires a different outer envelope.
- Goal: example correctness is a Goa property, not an MCP plugin property.

### 3. OpenAPI Contract Stability Policy

- Keep stable `operationId`, schema naming, response modeling, and other machine-consumable contract choices in `goa-light`.
- Do not let `goa-ai` carry its own naming or contract-stability conventions.
- Goal: there is one contract policy for generated APIs, not separate policies for AI and non-AI generators.

### 4. Generic Auth/Session Transport Modeling

- Keep bearer-or-cookie session auth, cookie transport binding, standard auth responses, and related OpenAPI/security emission in `goa-light`.
- `goa-ai` should consume those contracts instead of inventing its own auth transport shortcuts.
- Goal: MCP and ordinary APIs share one security model wherever the transport semantics are the same.

### 5. Generic Remediation-Aware Error Modeling

- Add first-class remediation/error contract primitives in `goa-light`.
- Support stable error code, safe message, retryability, hint, and optional structured fields as design-level concepts.
- Generate these consistently across HTTP, JSON-RPC, OpenAPI, and any transport that Goa owns directly.
- Goal: tool-style actionable failures come from the Goa design model, not from app-specific template hooks.

### 6. JSON-RPC Contract Semantics That Are Not MCP-Specific

- Audit `goa-ai` for any JSON-RPC generator behavior that is generic transport semantics rather than MCP behavior.
- Move anything that improves plain JSON-RPC correctness or stability into `goa-light/jsonrpc`.
- Goal: `goa-ai` should rely on Goa’s JSON-RPC transport, not patch around it.

### 7. Generic Transport/Runtime Metadata Hooks

- If `goa-ai` needs structured method metadata beyond raw `Meta(...)` for things that are not MCP-specific, add proper extension points to `goa-light` instead of duplicating metadata interpretation in plugin code.
- Goal: generic framework hooks live in Goa; protocol-specific metadata stays in plugins.

### 8. Error Projection Into OpenAPI and JSON Schema

- Once remediation-aware errors exist in `goa-light`, ensure OpenAPI 3.1 output exposes them in a machine-usable way with stable schemas and examples.
- `goa-ai` should then map tool/MCP failure behavior onto the same underlying contract instead of bypassing it.
- Goal: one error contract across normal APIs and tool-style APIs.

## Keep In Goa-AI

- MCP DSL and code generation
- agent and toolset DSL/runtime behavior
- planner/runtime structured tool execution features
- MCP-specific annotations and protocol-specific metadata

## Push Out Of The Frameworks

- application-specific special-casing such as direct framework dependence on a single app-owned error package
