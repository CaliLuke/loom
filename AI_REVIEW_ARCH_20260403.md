# AI Readiness Review - Loom Framework Architecture & Maintainability - 2026-04-03

## 1. Summary
The `loom` framework's core strength is its "design-first" approach, providing a single source of truth for APIs (the DSL). This naturally aligns with AI-assisted development by preventing transport-layer logic duplication. 

Recent refactoring efforts have successfully addressed the most critical architectural smells: massive "God Files" have been broken down, a `transportir` (Intermediate Representation) boundary was introduced across both HTTP and gRPC, shared transport analysis capabilities were implemented, and pervasive global state in the generator and DSL runtime has been eliminated. Furthermore, the core code generation engine has been almost entirely migrated from brittle string concatenation to safer AST-based generation (`jennifer`).

However, the framework still suffers from a few architectural smells that hinder maintainability and observability. These include pockets of legacy string/buffer-based code emission in a few generator paths and the lack of structured logging and context propagation across the top-level generation pipeline.

## 2. Issue Checklist

### State Management & Observability
- [x] **LOW** - Global State in the Generator Engine (State has been refactored and encapsulated)
- [ ] **MEDIUM** - Lack of Structured Logging (`slog`) and Context Propagation

### Structural Coupling & Abstraction
- [~] **LOW** - Deeply Coupled Template Logic & String Concatenation (Migration to AST generation almost complete)

### Test Velocity & Verification
- [ ] **HIGH** - Over-reliance on End-to-End Generation Tests

---

## 3. Detailed Findings & Roadmap

### Finding 2: Deeply Coupled Template Logic
- **Severity:** LOW (Reduced from MEDIUM)
- **Location:** Remaining legacy generator helpers such as `grpc/codegen/codec_sections.go`, `jsonrpc/codegen/stream_sections.go`, `codegen/service/sections.go`, and parts of `codegen/validation.go`.
- **Architectural Failure Mode:** Code generation previously relied entirely on massive string concatenations and deep nesting. Incredible progress has been made migrating generators to use the `jennifer` AST library, and the CLI generator is no longer the primary concern. The remaining maintainability risk is now concentrated in a smaller set of helpers that still assemble emitted code through strings, buffers, or mixed rendering styles, which makes targeted refactors and regression diagnosis harder than in the Jennifer-based paths.
- **Suggestion:**
  1. Continue opportunistic migration of the remaining string-emission hotspots to structured AST generation when touching those areas for real feature or bug work.
  2. Avoid broad rewrites for their own sake; prioritize the hotspots with the worst branch density or lowest test clarity.
- **Status:** [~] Almost Complete (Most generators are Jennifer-based; a few targeted hotspots remain)

### Finding 3: Lack of Structured Logging (`slog`) and Context Propagation
- **Severity:** MEDIUM
- **Location:** Project-wide.
- **Architectural Failure Mode:** If the code generation pipeline fails deep inside a nested template or AST traversal, the error returned is often just a string. Without structured logs or `context.Context` propagating a request/correlation ID, an autonomous agent (or human) cannot trace the failure back to the specific DSL line that caused it.
- **Suggestion:**
  1. Audit the `eval` and `codegen` packages to ensure `context.Context` is passed through all major pipeline boundaries.
  2. Integrate `log/slog` for structured logging, especially in the CLI entry points and codegen pipeline.
- **Status:** [ ] Pending

---

## 4. Pure Architecture Recommendations (Fowler Refactoring)

Treating the framework as a compiler (Frontend -> AST -> Backend), these pure architecture recommendations will dramatically improve modularity:

### 4.1 Consolidate Attribute Traversal and Type Dispatch
- **The Problem:** Parts of the generator still repeat ad hoc recursion and `switch` logic over `expr.AttributeExpr.Type` (`Object`, `Array`, `Map`, `Union`, `UserType`), which scatters traversal rules across packages.
- **The Fix:** Do not introduce a classic visitor hierarchy in `expr`. Loom's type model is intentionally open via interfaces such as `expr.DataType` and `expr.UserType`, and the repo already has walker-style helpers (`codegen.Walk`, `codegen.WalkMappedAttr`, `expr.AsObject` / `AsArray` / `AsMap` / `AsUnion`). Instead, strengthen those shared traversal utilities and extract focused dispatch helpers for recurring structural cases so generators share one recursion model without forcing a closed AST visitor pattern.

### 4.2 Inversion of Control (IoC) via a Plugin Architecture
- **The Problem:** The core generation engine is tightly coupled to specific target outputs (HTTP, gRPC, OpenAPI).
- **The Fix:** Define a clean `Generator` interface (e.g., `Generate(ast *expr.RootExpr) ([]*File, error)`). Implement HTTP, gRPC, and OpenAPI generators as entirely decoupled plugins.

### 4.3 Separate Semantic Analysis from AST Definitions
- **The Problem:** The `expr` package violates the Single Responsibility Principle by defining both AST data shape and complex validation/evaluation logic.
- **The Fix:** Treat the AST purely as a data structure. Move all semantic analysis into a dedicated `analyzer` or `validator` package.

### 4.4 Encapsulate Magic with the Strategy Pattern
- **The Problem:** Complex rules for type mapping and formatting (e.g., Go types to OpenAPI types) are scattered across templates or buried in massive utility files (`go_transform.go`).
- **The Fix:** Implement the Strategy Pattern via a `TypeTransformer` interface. Concrete strategies encapsulate specific transformation rules, allowing templates to delegate complex logic.

---

## 5. Updated Highest-Value Refactor

### Recommendation: Fix Top-Level Logging and Context Propagation

- **Why this is best move now:** The framework's current debugging surface is still weak at the orchestration layer where generator failures are discovered. The CLI and generation pipeline already emit ad hoc timing output, which provides a natural insertion point for structured `slog` logging and request-scoped `context.Context`. This is a higher-leverage next step than further architectural abstraction because it improves diagnosis immediately without requiring a broad rewrite.
- **Concrete first slice:**
  1. Start at the top-level generation boundaries:
     - `cmd/loom/main.go`
     - `cmd/loom/gen.go`
     - `codegen/generator/generate.go`
  2. Replace ad hoc stderr timing prints with structured `log/slog` events that record:
     - generation stage name
     - duration
     - generator identity
     - file counts / output paths where relevant
     - wrapped errors with stage metadata
  3. Introduce a generation-scoped `context.Context` and thread it through the orchestration path before attempting package-wide propagation into every helper.
  4. After the top-level pipeline is instrumented, expand inward only where logs show persistent blind spots.
- **Do not do first:** repo-wide logger plumbing across every generator helper, plugin/IoC rewrites, or broad package moves in `expr`. The first step should be a narrow orchestration-layer observability pass.
