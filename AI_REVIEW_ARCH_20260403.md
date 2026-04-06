# AI Readiness Review - Loom Framework Architecture & Maintainability - 2026-04-03

## 1. Summary
The `loom` framework's core strength is its "design-first" approach, providing a single source of truth for APIs (the DSL). This naturally aligns with AI-assisted development by preventing transport-layer logic duplication. 

Recent refactoring efforts have made significant progress in breaking down massive "God Files" into smaller, more focused files, and migrating code generation from brittle string concatenation towards safer AST-based generation (`jennifer`). 

However, this review needs one important correction: the repo is no longer uniformly missing an IR boundary. HTTP generation now has a real `transportir` layer, and JSON-RPC already reuses large portions of HTTP codegen data and rendering. The highest-value remaining architectural gap is now narrower and more actionable: gRPC still lacks the same IR boundary and still performs substantial endpoint shaping directly against `expr` structures, including in-place mutation of generated message state. That transport-analysis duplication is now the best target for improving development speed.

## 2. Issue Checklist

### Structural Coupling & Abstraction
- [~] **MEDIUM** - The "God File" & Missing Abstractions (HTTP improved; gRPC still lacks IR boundary)
- [ ] **CRITICAL** - Parallel Architecture Duplication (Primary gap now HTTP/JSON-RPC analysis model vs. gRPC)
- [~] **LOW** - Deeply Coupled Template Logic & String Concatenation (Migration to AST generation almost complete)

### State Management & Observability
- [ ] **CRITICAL** - Global State in the Generator Engine
- [ ] **MEDIUM** - Lack of Structured Logging (`slog`) and Context Propagation

### Test Velocity & Verification
- [ ] **HIGH** - Over-reliance on End-to-End Generation Tests

---

## 3. Detailed Findings & Roadmap

### Finding 1: The "God File" & Missing Abstractions
- **Severity:** MEDIUM
- **Location:** `http/codegen/service_data*.go`, `grpc/codegen/service_data*.go`.
- **Architectural Failure Mode:** These files used to be massive 1,800+ line God Files. Recent refactoring has physically split them into smaller, focused files, which is a major improvement. HTTP has progressed further than this document originally captured: `http/codegen/service_data_analysis.go` now starts from `http/codegen/internal/transportir`, and OpenAPI code also consumes that IR. The remaining abstraction gap is concentrated in gRPC, where `grpc/codegen/service_data_analysis.go` still mixes service naming, message conversion, request/response shaping, security partitioning, and stream handling directly against `expr.GRPCEndpointExpr`.
- **Suggestion:**
  1. **Split Phase / Extract Class:** Keep the HTTP/OpenAPI `transportir` direction. Do not restart this work generically.
  2. **Apply Same Boundary to gRPC:** Introduce `grpc/codegen/internal/transportir` and make gRPC analysis build endpoint data from IR rather than from live `expr` mutation.
- **Status:** [~] In Progress (HTTP/OpenAPI done; gRPC pending)

### Finding 2: Parallel Architecture Duplication
- **Severity:** CRITICAL
- **Location:** `http/codegen/...`, `grpc/codegen/...`, and `jsonrpc/codegen/...`.
- **Architectural Failure Mode:** This finding also needs refinement. JSON-RPC is not a fully separate parallel implementation anymore; it already leans heavily on `http/codegen` service data, section reuse, and source rewriting. The remaining expensive duplication is between the HTTP/JSON-RPC analysis path and the gRPC analysis path. Adding a capability that affects endpoint shaping, request/result mapping, or stream metadata still risks parallel work in two different architectural styles.
- **Suggestion:**
  1. **Do Not Start with Generic IoC:** A full plugin architecture is too broad for first move and would create high churn.
  2. **Create Shared Transport Analysis Kernel:** Standardize on `transportir`-style analysis first, then share capability builders across HTTP, OpenAPI, JSON-RPC, and gRPC where transport concepts actually match.
  3. **Top Refactor for Speed:** Prioritize gRPC adoption of the IR boundary. This is best leverage point for reducing repeated feature work.
- **Status:** [ ] Pending (problem narrowed; recommended first step changed)

### Finding 3: Global State in the Generator
- **Severity:** CRITICAL
- **Location:** `codegen/example/server_data.go` (`var Servers = make(ServersData)`), `codegen/plugin.go` (`var plugins []*plugin`), and others.
- **Architectural Failure Mode:** Generators should be pure functions (`Input AST -> Output Code`). Global state prevents safely running multiple generations in parallel (e.g., watch mode, language server) and causes tests to randomly pollute each other.
- **Suggestion:**
  1. **Encapsulate Variable / Parameterize Function:** State must be passed down via a `GeneratorContext` struct. Eliminate all global variables holding generation state.
- **Status:** [ ] Pending
- **Priority Note:** This remains real technical debt, but it is not first place for development-speed payoff. Compared with transport-analysis duplication, current global state is more contained and often reset explicitly in tests.

### Finding 4: Deeply Coupled Template Logic
- **Severity:** LOW (Reduced from MEDIUM)
- **Location:** `codegen/cli/cli.go` (and legacy custom rendering functions).
- **Architectural Failure Mode:** Code generation previously relied entirely on massive string concatenations and deep nesting. Incredible progress has been made migrating generators to use the `jennifer` AST library. The core `go_transform.go` has been fully migrated to Jennifer and no longer relies on `fmt.Sprintf` for code assembly. The only remaining significant usage of string concatenation for code emission is inside the CLI generator (`cli.go`).
- **Suggestion:**
  1. **Replace Inline Code with Function Call:** Finish the final stretch of the migration by converting `cli.go` to use `jennifer`, completely eliminating `fmt.Sprintf` code assembly from the codebase.
- **Status:** [~] Almost Complete (Migration to `jennifer` mostly done, `cli.go` remains)

### Finding 5: Lack of Structured Logging (`slog`) and Context Propagation
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

### 4.1 Introduce the Visitor Pattern for AST Traversal
- **The Problem:** The code generator relies heavily on deep type assertions and massive `switch` statements to traverse the syntax tree.
- **The Fix:** Implement an `ExprVisitor` interface inside the `expr` package. All code generators should implement this interface, allowing the Go compiler to enforce exhaustiveness.

### 4.2 Isolate the Intermediate Representation (IR)
- **The Problem:** Generator data structures attempt to hold broad cross-output state, creating deep coupling (Feature Envy) between templates and the global AST.
- **The Fix:** Continue current direction. HTTP already has this boundary via `transportir`; next step is to extend same pattern to gRPC and make shared capability logic depend on IR, not on transport-specific `expr` mutation.

### 4.3 Inversion of Control (IoC) via a Plugin Architecture
- **The Problem:** The core generation engine is tightly coupled to specific target outputs (HTTP, gRPC, OpenAPI).
- **The Fix:** Keep this as a later-stage cleanup, not first refactor. Current generator/plugin state has debt, but immediate speed gains come from converging transports onto shared analysis structures before broadening the execution model.

### 4.4 Separate Semantic Analysis from AST Definitions
- **The Problem:** The `expr` package violates the Single Responsibility Principle by defining both AST data shape and complex validation/evaluation logic.
- **The Fix:** Treat the AST purely as a data structure. Move all semantic analysis into a dedicated `analyzer` or `validator` package.

### 4.5 Encapsulate Magic with the Strategy Pattern
- **The Problem:** Complex rules for type mapping and formatting (e.g., Go types to OpenAPI types) are scattered across templates or buried in massive utility files (`go_transform.go`).
- **The Fix:** Implement the Strategy Pattern via a `TypeTransformer` interface. Concrete strategies encapsulate specific transformation rules, allowing templates to delegate complex logic.

---

## 5. Updated Highest-Value Refactor

### Recommendation: Bring gRPC onto `transportir`, then share transport analysis helpers

- **Why this is best move now:** HTTP has already proven the pattern. JSON-RPC already benefits from HTTP reuse. gRPC is remaining outlier.
- **Expected speed gain:** New transport-level features stop requiring one implementation in IR-driven HTTP/OpenAPI/JSON-RPC style and another in direct-`expr` gRPC style.
- **Concrete first slice:**
  1. Add `grpc/codegen/internal/transportir` with only fields gRPC needs.
  2. Replace in-place endpoint message mutation with IR construction.
  3. Refactor `grpc/codegen/service_data_analysis.go` to `BuildServiceIR -> buildEndpointDataFromIR -> render`.
  4. Add seam tests around IR builders before changing broad golden coverage.
- **Do not do first:** full plugin/IoC rewrite, repo-wide `slog` sweep, or global-state purge.
