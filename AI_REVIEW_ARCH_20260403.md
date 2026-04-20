# AI Readiness Review - Loom Framework Architecture & Maintainability - 2026-04-20

## Summary

Loom is in better shape than this review said on 2026-04-03. The top-level
generation diagnostics gap is now closed, `generator.Generate` has a stable
debug contract, and the service-data refactor has already reduced one of the
previous bottlenecks by splitting analysis logic across focused files.

The previously identified maintainability bottlenecks from the 2026-04-03
review are now addressed. The remaining work should come from fresh review
passes against newly changed subsystems rather than continuing to chase the old
hotspots.

## What Improved Since The Last Review

- Top-level generation diagnostics are now stage-scoped and stable in:
  - `cmd/loom/main.go`
  - `cmd/loom/gen.go`
  - `codegen/generator/generate.go`
- The service-data layer is no longer concentrated in one file. The analysis
  work is already split across:
  - `codegen/service/service_data_analysis.go`
  - `codegen/service/service_data_methods.go`
  - `codegen/service/service_data_types.go`
  - `codegen/service/service_data_views.go`
  - `codegen/service/service_data_interceptors.go`
- SSE and mixed-transport behavior now has materially stronger regression
  coverage than before, especially in:
  - `http/codegen/sse_server_test.go`
  - `http/codegen/streaming_test.go`
  - `jsonrpc/codegen/sse_test.go`
  - `jsonrpc/codegen/sse_integration_test.go`
- The CLI generator is no longer concentrated in one large helper file. Its
  command shaping, payload loading, conversion, flag parsing, and usage logic
  are now partitioned across:
  - `codegen/cli/command_data.go`
  - `codegen/cli/conversion.go`
  - `codegen/cli/flag_parsing.go`
  - `codegen/cli/helpers.go`
  - `codegen/cli/payload.go`
  - `codegen/cli/usage.go`
- The CLI layer also now has direct seam coverage for previously hidden helper
  behavior in:
  - `codegen/cli/cli_test.go`
- The shared method/service analysis surface is no longer one flat bag of
  method state. `MethodData` is now partitioned into concern-based structs for:
  - payload metadata
  - result and viewed-result metadata
  - security and interceptor metadata
  - transport metadata
  - streaming metadata
  and those concerns are built and mutated through dedicated helpers in:
  - `codegen/service/service_data.go`
  - `codegen/service/service_data_methods.go`
  - `codegen/service/service_data_analysis.go`
  - `codegen/service/service_data_interceptors.go`
- The service-data refactor also now has direct characterization coverage in:
  - `codegen/service/service_data_refactor_test.go`
- The shared SSE lifecycle emission is no longer duplicated ad hoc across HTTP
  and JSON-RPC generators. Common header-init and write-and-flush behavior now
  lives in:
  - `internal/ssecodegen/ssecodegen.go`
  and is consumed by:
  - `http/codegen/misc_sections.go`
  - `jsonrpc/codegen/stream_sections.go`
- The shared SSE lifecycle helper now has direct verification in:
  - `internal/ssecodegen/ssecodegen_test.go`
- The validation generator is no longer concentrated in one mixed file. Its
  recursive traversal and per-rule emission logic are now split across:
  - `codegen/validation.go`
  - `codegen/validation_recurse.go`
  - `codegen/validation_render.go`
- The validation layer also now has direct seam coverage for newly explicit
  render-state behavior in:
  - `codegen/validation_test.go`

That means the old “instrument top-level generation first” recommendation is
done and should no longer drive prioritization.

## Issue Checklist

### DRY & Partitioning

- [x] Validation generation partitioning is in place.

## Detailed Findings

### 1. CLI Generation Partitioning And Seam Tests Are In Place

- **Severity**: RESOLVED
- **Location**:
  - `codegen/cli/command_data.go:1`
  - `codegen/cli/conversion.go:1`
  - `codegen/cli/flag_parsing.go:1`
  - `codegen/cli/payload.go:1`
  - `codegen/cli/usage.go:1`
  - `codegen/cli/cli_test.go:12`
- **Why it mattered**: The old CLI generator mixed command metadata shaping,
  usage rendering, payload building, flag parsing, default handling, example
  generation, and scalar/JSON conversion rules in one large file, which made
  small changes expensive to reason about.
- **What changed**:
  1. The CLI generator helpers are now split across focused files by concern.
  2. Direct seam tests now cover `conversionCode`, `FieldLoadCode`, and
     `FlagsCodeStatement`.
  3. Generated-section rendering coverage remains in place, but helper tests
     are now part of the fast inner loop.
- **Status**: [x] Complete

### 2. `service.MethodData` Partitioning And Centralized Mutation Are In Place

- **Severity**: RESOLVED
- **Location**:
  - `codegen/service/service_data.go:116`
  - `codegen/service/service_data_methods.go:12`
  - `codegen/service/service_data_analysis.go:89`
  - `codegen/service/service_data_interceptors.go:108`
- **Why it mattered**: The method analysis output used to mix payload/result
  metadata, error and auth state, transport flags, stream wiring, and client
  endpoint-field naming in one flat struct and one large construction path.
- **What changed**:
  1. `MethodData` is now partitioned into embedded concern structs for payload,
     result, security, transport, and streaming state.
  2. Method construction is now split into small builders by concern instead of
     one cross-domain literal.
  3. Remaining mutation points for viewed results, interceptor registration,
     and endpoint-field assignment now flow through method-owned helpers.
  4. Direct characterization tests now lock the grouped state and mutation
     points through analyzed service fixtures.
- **Status**: [x] Complete

### 3. Shared SSE Lifecycle Extraction Is In Place

- **Severity**: RESOLVED
- **Location**:
  - `internal/ssecodegen/ssecodegen.go:1`
  - `http/codegen/misc_sections.go:219`
  - `jsonrpc/codegen/stream_sections.go:21`
  - `internal/ssecodegen/ssecodegen_test.go:1`
- **Why it mattered**: HTTP SSE and JSON-RPC SSE each carried their own
  header-init and write-and-flush lifecycle emission, which meant a transport
  behavior change could silently be applied in one path and not the other.
- **What changed**:
  1. Shared SSE header initialization and write-and-flush emission now live in a
     neutral helper package.
  2. HTTP and JSON-RPC generators both consume that shared lifecycle helper
     while keeping protocol-specific event framing local.
  3. The shared helper has direct tests, and the transport-specific generators
     still retain their own SSE characterization suites.
- **Status**: [x] Complete

### 4. Validation Generation Partitioning And Typed Render State Are In Place

- **Severity**: RESOLVED
- **Location**:
  - `codegen/validation.go:1`
  - `codegen/validation_recurse.go:1`
  - `codegen/validation_render.go:1`
  - `codegen/validation_test.go:12`
- **Why it mattered**: Validation behavior previously lived in one dense file
  that mixed recursive traversal, render-state mutation, and rule-specific
  string emission. Small changes required too much whole-file reasoning.
- **What changed**:
  1. Recursive traversal and branch assembly now live in a dedicated recurse
     file, separate from the public entrypoints and shared helpers.
  2. Rule-specific rendering now lives behind a typed render-state struct
     rather than an unstructured `map[string]any`.
  3. Direct seam coverage now locks alias/pointer state handling and the
     existing exclusive-range generation contract alongside the broader golden
     suite.
- **Status**: [x] Complete

## What Not To Prioritize Now

- another broad diagnostics push at the top-level generator boundary
- a wholesale rewrite of the service-data pipeline that ignores the progress
  already made
- converting every remaining string-based generator in one pass
- repo-wide logging abstractions unrelated to actual debug blind spots
- blanket new test suites without first targeting the remaining hot seams

## Bottom Line

Loom no longer needs the top-level diagnostics work that dominated the last
review, and the concrete maintainability targets identified there are now
closed. The next useful improvements should come from new evidence-backed
reviews of whatever subsystem changes next rather than from reopening this same
checklist.
