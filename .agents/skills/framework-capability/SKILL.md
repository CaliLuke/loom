---
name: framework-capability
description: Add a new `goa-light` framework capability end to end. Use this when implementing generator, DSL, transport, OpenAPI, auth/session, or other framework-level behavior that should remove real application glue. Covers scoping the capability, raising the testing bar, validating repo cleanliness, updating roadmap/docs including the Goa skill, and finishing with commit and push.
---
# Framework Capability

Use this skill when the task is to add or complete a real `goa-light` framework capability, not just adopt existing behavior in an application.

## Trigger Conditions

Use this skill when the work involves any of:

- new Goa DSL surface area
- codegen or transport behavior changes
- OpenAPI or JSON Schema contract changes
- auth/session framework semantics
- multipart, form, JSON decode, or other request/response transport capabilities
- removal of repeated app-local glue that exists because the framework stops short

Do not use this skill for:

- pure app-level adoption work
- one-off local patches in a consumer repo
- docs-only roadmap updates without framework implementation

## Core Rule

The framework only earns new capability when it removes real glue or real risk in consuming apps.

Start by naming:

1. the repeated app-local workaround
2. the generic framework behavior that should replace it
3. the concrete consumer proving the need

If you cannot state all three clearly, slow down before adding framework surface.

## Delivery Standard

A framework capability is not done when code compiles. It is done only when:

- the framework behavior is implemented at the right layer
- the tests are thorough enough for a maintained fork
- the roadmap and plan docs reflect reality
- the Goa skill reflects the new contract or workflow
- the branch is committed and pushed

## Workflow

### 1. Scope the capability

Write down, at least mentally:

- current app-local glue
- why that glue exists
- what layer owns the fix:
  - DSL
  - `expr`
  - codegen
  - HTTP transport
  - gRPC transport
  - JSON-RPC transport
  - OpenAPI generation
- what should remain app-owned after the framework change

Prefer the narrowest framework change that removes the repeated workaround cleanly.

### 2. Read before editing

Inspect:

- the motivating consumer usage or workaround
- the existing framework path that almost solves it
- nearby tests and goldens
- the Goa skill if the change affects contract semantics or recommended usage

Do not guess where generation stops short.

### 3. Implement at the root cause

Rules:

- do not patch generated output
- do not add app-facing seams when the framework should own the behavior
- do not special-case one app protocol flow if a generic transport capability is the real fix
- preserve clear boundaries between framework semantics and app runtime policy

If the change affects multiple generators or transports, decide explicitly which ones must change now and which only need regression coverage.

## Testing Bar

Happy-path-only testing is not acceptable for framework capability work in this fork.

### Required test layers

Add the layers that apply:

1. focused unit tests for the changed logic
2. generator/output tests when output shape matters
3. regression tests for the motivating bug or workaround
4. broader package or transport suite runs to catch collateral regressions

### Required coverage mindset

Aim for MECE-style coverage where practical:

- normal path
- edge cases
- invalid input / malformed shape
- ambiguity boundaries
- “should still reject” cases
- transport-specific or content-type-specific branches

For contract-shaping or output-shaping changes, include:

- direct structural assertions
- rendered/golden assertions where the serialized shape matters
- spec or parser validation where relevant, for example `libopenapi`

For transport capability changes, include:

- decoding success path
- invalid decode path
- validation-after-decode path
- “old custom workaround no longer needed” regression

### Transport verification

If the framework change is in one transport but could affect others, run the relevant surrounding suites.

Typical examples:

- OpenAPI / HTTP contract work:
  - `go test ./http/codegen/...`
  - `go test ./http/codegen/openapi/...`
  - `go test ./http/codegen/openapi/v3`
- shared expression or DSL work:
  - `go test ./expr/...`
- gRPC regression gate:
  - `go test ./grpc/...`
- JSON-RPC regression gate:
  - `go test ./jsonrpc/...`

If a broader suite is red, do not leave it red. Fix the source issue or update the stale fixture if the framework output intentionally changed.

## Cleanliness Check

Before calling the work done:

1. run formatting for touched files
2. run the relevant targeted tests
3. run the broader transport/package suites that should remain green
4. check `git status --short`
5. make sure there are no unexplained red tests
6. make sure goldens or fixtures reflect intended behavior, not stale expectations

The repo should be clean before the final handoff unless the user explicitly asked to stop short.

## Documentation Duties

Framework work must update docs where behavior changed.

### Roadmap

Update the roadmap when needed:

- add the capability if it was not already captured
- move status from planned to completed when finished
- update or add the specific plan doc if the implementation changed the plan

At minimum inspect:

- `roadmap/ROADMAP.md`
- the specific capability plan doc

### Goa skill

If the capability changes how an agent should think about Goa-light behavior, update the Goa skill directly.

Check:

- `.agents/skills/goa/SKILL.md`
- `.agents/skills/goa/references/repo-map.md` if navigation guidance changed

Do not hide important repo-specific framework semantics in a hacky sidecar note when they belong in the main skill instructions.

Update the Goa skill when the change affects:

- contract expectations
- transport semantics
- DSL usage guidance
- OpenAPI output shape
- auth/session modeling
- union/discriminator semantics
- request decoding behavior developers need to know about

## Finish Sequence

Unless the user explicitly says otherwise:

1. update roadmap docs
2. update Goa skill docs where needed
3. verify tests are green
4. commit with a concrete message
5. push the branch

Do not stop at “implemented locally” if the natural next step is commit and push.

## Final Report Checklist

Your final summary should state:

- what framework capability was added
- what app-local glue it replaces
- which files carry the implementation
- which docs were updated
- which test suites were run
- whether the branch was committed and pushed

If anything is intentionally partial, say so plainly.
