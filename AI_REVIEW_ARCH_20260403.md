# AI Readiness Review - Loom Framework Architecture & Maintainability - 2026-04-09

## Summary

Loom is in better shape than this review said on 2026-04-03. The top-level
generation diagnostics gap is now closed, `generator.Generate` has a stable
debug contract, and the service-data refactor has already reduced one of the
previous bottlenecks by splitting analysis logic across focused files.

The next obstacles to fast, bug-resistant development are narrower:

1. the CLI generator is still a single, mixed-responsibility file with only
   light seam coverage
2. the shared `service.Data` / `service.MethodData` model still centralizes too
   much transport-specific state, which keeps new feature work in shotgun
   surgery territory
3. SSE generation logic is still implemented in large, parallel string-building
   files across HTTP and JSON-RPC, which creates real drift risk
4. the validation generator remains one of the densest string-emission cores in
   the repo, though strong tests keep it from being the top priority

The right next move is not a rewrite. It is another round of targeted
partitioning and seam-test expansion in the places where a small behavior
change still requires too much whole-file reasoning.

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

That means the old “instrument top-level generation first” recommendation is
done and should no longer drive prioritization.

## Issue Checklist

### DRY & Partitioning

- [ ] HIGH - CLI generation is still a monolithic mixed-responsibility file.
- [ ] HIGH - `service.MethodData` still centralizes too much transport and
      streaming state.
- [ ] MEDIUM - SSE generation behavior still lives in parallel HTTP and
      JSON-RPC string builders.

### Test Velocity & Verification

- [ ] MEDIUM - CLI generation still lacks the kind of seam tests that make
      small generator changes cheap to verify.

### Mixed Emission Hotspots

- [ ] MEDIUM - Validation generation is still driven by recursive string/buffer
      composition.

## Detailed Findings

### 1. CLI Generation Is Still A Monolithic Mixed-Responsibility File

- **Severity**: HIGH
- **Location**:
  - `codegen/cli/cli.go:160`
  - `codegen/cli/cli.go:332`
  - `codegen/cli/cli.go:462`
  - `codegen/cli/cli.go:638`
  - `codegen/cli/cli_test.go:12`
- **Why it matters**: `codegen/cli/cli.go` is still about 950 lines and mixes:
  command metadata shaping, usage rendering, payload building, flag parsing,
  default handling, example generation, and scalar/JSON conversion rules.
- **Agentic Failure Mode**: An agent adds a new flag behavior or payload-loading
  rule, updates `FieldLoadCode` or `conversionCode`, and misses the matching
  usage, example, or error-path branch elsewhere in the same file. The change
  compiles, but the generated CLI becomes inconsistent across parsing, help
  text, and conversion errors.
- **Suggestion**:
  1. Split `cli.go` into focused files such as `command_data.go`,
     `usage_sections.go`, `flag_parsing.go`, and `conversion.go`.
  2. Add table-driven seam tests for `conversionCode`, `FieldLoadCode`,
     `FlagsCodeStatement`, and the error/default branches that currently hide in
     the monolith.
  3. Keep generated-section rendering tests, but make direct helper tests the
     fast inner loop.
- **Status**: [ ] Pending

### 2. `service.Data` And `service.MethodData` Still Centralize Too Much State

- **Severity**: HIGH
- **Location**:
  - `codegen/service/service_data.go:57`
  - `codegen/service/service_data.go:115`
  - `codegen/service/service_data_analysis.go:13`
- **Why it matters**: The analysis pipeline itself is now better partitioned,
  but the central output structs are still giant cross-domain bags of state.
  `MethodData` in particular mixes payload/result metadata, security, error
  locations, JSON-RPC classification, stream wiring, raw body bypass flags, and
  client endpoint-field naming.
- **Agentic Failure Mode**: An agent adds a new transport capability or method
  flag, threads it through some builders, but forgets one of the many places
  that consumes or initializes `MethodData`. The feature appears to work in one
  transport while silently missing a field, naming rule, or zero-value behavior
  in another.
- **Suggestion**:
  1. Keep the current split analysis files, but carve `MethodData` into nested
     substructures by concern: core method identity, transport metadata,
     streaming metadata, and client/server codegen hints.
  2. Add small constructor helpers so new fields are initialized in one place
     instead of being spread across the analysis pipeline.
  3. Prefer transport-specific attachments over adding more top-level booleans
     to `MethodData`.
- **Status**: [ ] Pending

### 3. SSE Generation Still Has Cross-Transport Drift Risk

- **Severity**: MEDIUM
- **Location**:
  - `http/codegen/stream_sections.go:30`
  - `http/codegen/stream_sections.go:109`
  - `http/codegen/stream_sections.go:146`
  - `jsonrpc/codegen/stream_sections.go:21`
  - `jsonrpc/codegen/stream_sections.go:77`
  - `jsonrpc/codegen/stream_sections.go:121`
  - `jsonrpc/codegen/stream_sections.go:193`
  - `jsonrpc/codegen/stream_sections.go:215`
- **Why it matters**: The tests around SSE behavior are strong now, but the
  implementation is still spread across two large, branchy, string-assembled
  generators with parallel lifecycle ideas: header init, stream open/flush,
  event encoding, error framing, and event parsing.
- **Agentic Failure Mode**: An agent fixes SSE framing, flush timing, or
  response/error behavior in the HTTP generator and misses the equivalent
  JSON-RPC path, or vice versa. The repo keeps compiling, one transport’s tests
  pass, and the other transport drifts until a broader integration loop catches
  it later.
- **Suggestion**:
  1. Extract shared SSE lifecycle primitives where the behavior is genuinely the
     same: header commit, event flush, buffer scanning, and common error/event
     framing helpers.
  2. Keep transport-specific protocol wrapping local to each package.
  3. Add a small cross-transport contract test matrix for the truly shared
     pieces so the next SSE behavior change does not rely on whole-file review.
- **Status**: [ ] Pending

### 4. Validation Generation Remains A Dense String-Emission Core

- **Severity**: MEDIUM
- **Location**:
  - `codegen/validation.go:1`
  - `codegen/validation.go:47`
  - `codegen/validation.go:138`
  - `codegen/validation.go:207`
  - `codegen/validation_test.go:12`
- **Why it matters**: This file is not the most urgent problem because it is
  well covered, but it is still one of the least mechanically friendly places
  to change. Behavior is built from recursive buffer composition and string
  stitching rather than smaller typed emitters.
- **Agentic Failure Mode**: An agent changes alias, pointer, union, or nested
  collection validation rules and keeps the common golden tests green, but
  breaks a shape combination that emerges from the recursive string assembly
  path rather than from an explicit typed intermediate representation.
- **Suggestion**:
  1. Do not prioritize a broad rewrite.
  2. When validation work is touched again, continue splitting object, array,
     map, and union emitters into narrower helpers with smaller contracts.
  3. Preserve the strong golden coverage while adding direct helper tests around
     the next changed branch.
- **Status**: [ ] Pending

## What Not To Prioritize Now

- another broad diagnostics push at the top-level generator boundary
- a wholesale rewrite of the service-data pipeline that ignores the progress
  already made
- converting every remaining string-based generator in one pass
- repo-wide logging abstractions unrelated to actual debug blind spots
- blanket new test suites without first targeting the remaining hot seams

## Recommended Execution Order

1. Decompose `codegen/cli/cli.go` and add direct seam tests around conversion,
   field loading, and flag parsing.
2. Partition `service.MethodData` and related shared transport state into
   smaller substructures with single-point initialization.
3. Extract the truly shared SSE lifecycle helpers and add cross-transport
   contract tests for them.
4. Keep validation-generator cleanup opportunistic and local to changed
   branches.

## Bottom Line

Loom no longer needs the top-level diagnostics work that dominated the last
review. The next maintenance wins are smaller and more concrete:

- reduce the CLI generator monolith
- shrink the shared method/service metadata surface
- remove the remaining SSE drift points across transports
- keep pushing mixed string-emission cores behind tighter seams

That work will improve agent autonomy and human development speed more than a
large architectural rewrite would.
