# AI Readiness Review - Loom Framework Architecture & Maintainability - 2026-04-03

## 1. Summary
The `loom` framework's core strength is its "design-first" approach, providing a single source of truth for APIs (the DSL). This naturally aligns with AI-assisted development by preventing transport-layer logic duplication. 

Recent refactoring efforts have successfully addressed the most critical architectural smells: massive "God Files" have been broken down, a `transportir` (Intermediate Representation) boundary was introduced across both HTTP and gRPC, and shared transport analysis capabilities were implemented. Furthermore, the core code generation engine has been almost entirely migrated from brittle string concatenation to safer AST-based generation (`jennifer`).

However, the framework still suffers from a few architectural smells that hinder maintainability, testability, and observability. These include pervasive global state in the generator and the lack of structured logging and context propagation.

## 2. Issue Checklist

### State Management & Observability
- [ ] **CRITICAL** - Global State in the Generator Engine
- [ ] **MEDIUM** - Lack of Structured Logging (`slog`) and Context Propagation

### Structural Coupling & Abstraction
- [~] **LOW** - Deeply Coupled Template Logic & String Concatenation (Migration to AST generation almost complete)

### Test Velocity & Verification
- [ ] **HIGH** - Over-reliance on End-to-End Generation Tests

---

## 3. Detailed Findings & Roadmap

### Finding 1: Global State in the Generator
- **Severity:** CRITICAL
- **Location:** `codegen/example/server_data.go` (`var Servers = make(ServersData)`), `codegen/plugin.go` (`var plugins []*plugin`), and others.
- **Architectural Failure Mode:** Generators should be pure functions (`Input AST -> Output Code`). Global state prevents safely running multiple generations in parallel (e.g., watch mode, language server) and causes tests to randomly pollute each other.
- **Suggestion:**
  1. **Encapsulate Variable / Parameterize Function:** State must be passed down via a `GeneratorContext` struct. Eliminate all global variables holding generation state.
- **Status:** [ ] Pending

### Finding 2: Deeply Coupled Template Logic
- **Severity:** LOW (Reduced from MEDIUM)
- **Location:** `codegen/cli/cli.go` (and legacy custom rendering functions).
- **Architectural Failure Mode:** Code generation previously relied entirely on massive string concatenations and deep nesting. Incredible progress has been made migrating generators to use the `jennifer` AST library. The core `go_transform.go` has been fully migrated to Jennifer and no longer relies on `fmt.Sprintf` for code assembly. The only remaining significant usage of string concatenation for code emission is inside the CLI generator (`cli.go`).
- **Suggestion:**
  1. **Replace Inline Code with Function Call:** Finish the final stretch of the migration by converting `cli.go` to use `jennifer`, completely eliminating `fmt.Sprintf` code assembly from the codebase.
- **Status:** [~] Almost Complete (Migration to `jennifer` mostly done, `cli.go` remains)

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

### 4.1 Introduce the Visitor Pattern for AST Traversal
- **The Problem:** The code generator relies heavily on deep type assertions and massive `switch` statements to traverse the syntax tree.
- **The Fix:** Implement an `ExprVisitor` interface inside the `expr` package. All code generators should implement this interface, allowing the Go compiler to enforce exhaustiveness.

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

### Recommendation: Eliminate Global State in the Generator Engine

- **Why this is best move now:** With the major structural abstractions (IR and AST generation) largely complete, the next biggest blocker for modernizing the framework (e.g., language server support, parallel generation, reliable testing) is the pervasive use of global state.
- **Concrete first slice:**
  1. Introduce a `GeneratorContext` or similar structure to hold state.
  2. Refactor the plugin registry (`codegen/plugin.go`) to be instance-based rather than a global slice.
  3. Plumb the context through the code generation pipeline to replace global variables like `var Servers` in the example generators.
- **Do not do first:** full plugin/IoC rewrite, repo-wide `slog` sweep.