# AI Readiness Review - Loom Framework Architecture & Maintainability - 2026-04-03

## 1. Summary
The `loom` framework's core strength is its "design-first" approach, providing a single source of truth for APIs (the DSL). This naturally aligns with AI-assisted development by preventing transport-layer logic duplication. 

Recent refactoring efforts have made significant progress in breaking down massive "God Files" into smaller, more focused files, and migrating code generation from brittle string concatenation towards safer AST-based generation (`jennifer`). 

Additionally, the framework has successfully introduced a `transportir` (Intermediate Representation) boundary across both HTTP and gRPC transport layers! This separates the AST extraction from the code shaping, drastically narrowing the architectural gap. The next highest-value architectural target is sharing transport analysis capabilities across these IR-driven boundaries.

## 2. Issue Checklist

### Structural Coupling & Abstraction
- [x] **LOW** - The "God File" & Missing Abstractions (IR boundaries implemented across HTTP and gRPC)
- [~] **HIGH** - Parallel Architecture Duplication (Shared transport analysis kernel implemented; transport-specific shaping intentionally remains local)
- [~] **LOW** - Deeply Coupled Template Logic & String Concatenation (Migration to AST generation almost complete)

### State Management & Observability
- [ ] **CRITICAL** - Global State in the Generator Engine
- [ ] **MEDIUM** - Lack of Structured Logging (`slog`) and Context Propagation

### Test Velocity & Verification
- [ ] **HIGH** - Over-reliance on End-to-End Generation Tests

---

## 3. Detailed Findings & Roadmap

### Finding 1: The "God File" & Missing Abstractions
- **Severity:** LOW (Reduced from CRITICAL)
- **Location:** `http/codegen/service_data*.go`, `grpc/codegen/service_data*.go`.
- **Architectural Failure Mode:** These files used to be massive 1,800+ line God Files. Recent refactoring has physically split them into smaller, focused files, which is a major improvement. Furthermore, both HTTP and gRPC now utilize a unified `transportir` (Intermediate Representation) boundary to separate AST extraction from Go-specific template shaping. 
- **Suggestion:**
  1. **Split Phase / Extract Class:** The extraction of the AST into an Intermediate Representation (IR) is now fully separated from the code that shapes it for Go templates across major transport layers.
- **Status:** [x] Complete

### Finding 2: Parallel Architecture Duplication
- **Severity:** HIGH (Reduced from CRITICAL)
- **Location:** `http/codegen/...`, `grpc/codegen/...`, and `jsonrpc/codegen/...`.
- **Architectural Failure Mode:** JSON-RPC already leans heavily on `http/codegen` service data, section reuse, and source rewriting. The expensive duplication used to be capability selection logic spread across HTTP/JSON-RPC and gRPC analysis. That duplication is now materially reduced: both transports consume shared service-layer descriptors for package selection, payload/result resolution, viewed-result projection, error type resolution, stream capability facts, and request/response wrapper capability checks. The remaining duplication is now mostly transport-specific by design: HTTP body/status/header/cookie shaping and gRPC protobuf/metadata shaping.
- **Suggestion:**
  1. **Do Not Start with Generic IoC:** A full plugin architecture is still too broad and still wrong as a first move.
  2. **Keep Growing the Shared Transport Analysis Kernel:** Future cross-transport capability work should start in `codegen/service/transport_descriptors.go` and only fall back to transport-specific code where the concepts truly diverge.
  3. **Leave Transport-Specific Shaping Local:** HTTP response/body/status logic and gRPC protobuf/metadata logic should remain local unless a second concrete consumer appears.
- **Status:** [~] In Progress (shared kernel implemented and used by HTTP/gRPC; transport-specific shaping remains intentionally local)

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
- **The Fix:** Both HTTP and gRPC now implement this boundary via `transportir`, and the shared capability logic now lives in `codegen/service` over `service.Data`, `service.MethodData`, and concrete attrs rather than transport-specific `expr` traversal.

### 4.3 Inversion of Control (IoC) via a Plugin Architecture
- **The Problem:** The core generation engine is tightly coupled to specific target outputs (HTTP, gRPC, OpenAPI).
- **The Fix:** Keep this as a later-stage cleanup, not first refactor. Immediate speed gains come from converging transports onto shared analysis structures before broadening the execution model.

### 4.4 Separate Semantic Analysis from AST Definitions
- **The Problem:** The `expr` package violates the Single Responsibility Principle by defining both AST data shape and complex validation/evaluation logic.
- **The Fix:** Treat the AST purely as a data structure. Move all semantic analysis into a dedicated `analyzer` or `validator` package.

### 4.5 Encapsulate Magic with the Strategy Pattern
- **The Problem:** Complex rules for type mapping and formatting (e.g., Go types to OpenAPI types) are scattered across templates or buried in massive utility files (`go_transform.go`).
- **The Fix:** Implement the Strategy Pattern via a `TypeTransformer` interface. Concrete strategies encapsulate specific transformation rules, allowing templates to delegate complex logic.

---

## 5. Updated Highest-Value Refactor

### Recommendation: Share transport analysis helpers across `transportir` boundaries

- **Why this is best move now:** Both HTTP and gRPC have successfully adopted the `transportir` intermediate representation, and they now share a real analysis kernel in `codegen/service`.
- **Implemented shared entrypoints:**
  1. `service.DefaultPackageName`
  2. `service.BuildPayloadDescriptor`
  3. `service.BuildResultDescriptor`
  4. `service.BuildErrorDescriptor`
  5. `service.BuildStreamDescriptor`
  6. `service.DescribeStream`
  7. `service.DescribeMethodCapabilities`
- **Current payoff:** New cross-transport capability work starts in one shared place for package/type/view/stream/error/wrapper decisions before fanning out into HTTP or gRPC rendering.
- **Deliberate survivors:** HTTP body/status shaping, SSE/WebSocket envelope details, and gRPC protobuf/metadata shaping remain local.
- **Do not do next:** full plugin/IoC rewrite, repo-wide `slog` sweep, or global-state purge.
