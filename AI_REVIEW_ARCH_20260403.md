# AI Readiness Review - Loom Framework Architecture & Maintainability - 2026-04-03

## 1. Summary
The `loom` framework's core strength is its "design-first" approach, providing a single source of truth for APIs (the DSL). This naturally aligns with AI-assisted development by preventing transport-layer logic duplication. 

However, the framework currently still suffers from context saturation in parts of the DSL layer, structural coupling between the AST (`expr/`) and the generator (`codegen/`), and a lack of explicit observability primitives (like `slog`) in the core generation pipeline.

## 2. Issue Checklist

### Context Partitioning & Modularity
- [ ] **HIGH** - Massive File Bloat in Core DSL and Codegen
- [ ] **MEDIUM** - Mixed Domains in `expr/http_endpoint.go`

### DRY Code & Single Source of Truth
- [ ] **CRITICAL** - Generator "Shotgun Surgery" Risks
- [ ] **HIGH** - Hardcoded HTTP Methods and Magic Strings

### Test Velocity & Verification
- [ ] **HIGH** - Over-reliance on End-to-End Generation Tests

### Observability & Traceability
- [ ] **MEDIUM** - Lack of Structured Logging (`slog`) and Context Propagation

---

## 3. Detailed Findings & Roadmap

### Finding 1: Massive File Bloat in Core DSL and Codegen
- **Severity:** HIGH
- **Location:** `expr/http_endpoint.go` (~1,500 lines).
- **Agentic Failure Mode:** AI agents have limited context windows. When an agent is asked to modify HTTP endpoint preparation or validation behavior, it must ingest one very large file, leading to token waste, context truncation, and a higher chance of the AI forgetting earlier branches in the file.
- **Suggestion:**
  1. **Split `expr/http_endpoint.go`:** The `HTTPEndpointExpr` struct definition, validation logic (`Validate()`), and evaluation/preparation logic (`Prepare()`) should live in separate files (e.g., `http_endpoint_eval.go`, `http_endpoint_validate.go`). 
- **Status:** [ ] Pending

### Finding 2: Generator "Shotgun Surgery" Risks
- **Severity:** CRITICAL
- **Location:** `codegen/` package (e.g., `go_transform.go`, `service/convert.go`).
- **Agentic Failure Mode:** The codegen layer relies heavily on manually mapping `expr` AST nodes to generated strings. If a new type or attribute is added to the DSL, an AI (or human) must hunt down every `switch` statement or template string across the `codegen/` directory to support it. If one mapping is missed, the compiler won't catch it until the generated code fails to compile.
- **Suggestion:**
  1. **Extract Type Mappings:** Move all AST-to-Go type mappings into a centralized, single-source-of-truth helper registry (e.g., a `TypeMapper` interface). 
  2. **Template Interfaces:** Instead of generators manually inspecting deep nested structs like `expr.AttributeExpr.Type`, the AST nodes should implement interfaces that the templates consume directly (e.g., `IsCollection()`, `IsPrimitive()`).
- **Status:** [ ] Pending

### Finding 3: Hardcoded HTTP Methods and Magic Strings
- **Severity:** HIGH
- **Location:** `dsl/http.go` and `expr/http_endpoint.go`.
- **Agentic Failure Mode:** If HTTP methods (`"GET"`, `"POST"`) or built-in schema types are passed as raw string literals, the Go compiler cannot catch AI mistakes (e.g., the AI typos `"PUTT"`). The AI loses its compile-time self-healing loop.
- **Suggestion:**
  1. Enforce the use of standard library constants (e.g., `http.MethodGet` from `net/http`) or define strict internal constants (`const MethodGet = "GET"`) across the DSL and AST. 
  2. Replace raw string equality checks with constant comparisons.
- **Status:** [ ] Pending

### Finding 4: Over-reliance on End-to-End Generation Tests
- **Severity:** HIGH
- **Location:** `codegen/` and `expr/` test suites.
- **Agentic Failure Mode:** If an AI proposes a refactor to a generator template, and the only way to verify it is to run the entire `loom gen` command against a dummy project, the "Code-Test-Fix" feedback loop is too slow and fragile.
- **Suggestion:**
  1. Write table-driven tests for specific generator functions (e.g., testing `go_transform.go` by passing a mocked `AttributeExpr` and asserting the output string) without invoking the full CLI pipeline.
- **Status:** [ ] Pending

### Finding 5: Lack of Structured Logging (`slog`) and Context Propagation
- **Severity:** MEDIUM
- **Location:** Project-wide.
- **Agentic Failure Mode:** If the code generation pipeline fails deep inside a nested template or AST traversal, the error returned is often just a string. Without structured logs or `context.Context` propagating a request/correlation ID, an autonomous agent cannot trace the failure back to the specific DSL line that caused it.
- **Suggestion:**
  1. Audit the `eval` and `codegen` packages to ensure `context.Context` is passed through all major pipeline boundaries.
  2. Integrate `log/slog` for structured logging, especially in the CLI entry points and codegen pipeline, to provide rich, easily parsable output for debugging and AI context retrieval. Error messages should include actionable recovery hints (e.g., `"failed to map type: unknown type 'uuid', use Type() to define custom types"`).
- **Status:** [ ] Pending

---

## 4. Pure Architecture & Maintainability Enhancements

Treating the framework as a compiler (Frontend -> AST -> Backend), the following pure architecture recommendations will dramatically improve modularity and maintainability:

### 4.1 Introduce the Visitor Pattern for AST Traversal
- **The Problem:** The code generator relies heavily on deep type assertions and massive `switch` statements to traverse the syntax tree in the `expr` package. This prevents the compiler from verifying that all nodes are handled when a new node type is added.
- **The Fix:** Implement an `ExprVisitor` interface inside the `expr` package that requires a `Visit` method for every node type. All code generators should implement this interface, allowing the Go compiler to enforce exhaustiveness.

### 4.2 Isolate the Intermediate Representation (IR)
- **The Problem:** Some generator data structures still attempt to hold broad cross-output state, creating deep coupling (Feature Envy) between templates and the global AST.
- **The Fix:** Decouple the AST from templates via strict, context-bounded IR structs. Map the `expr` AST into small, focused structs (e.g., an `HTTPHandlerScope`) just once at the boundary, and pass only what is needed to the rendering engine.

### 4.3 Inversion of Control (IoC) via a Plugin Architecture
- **The Problem:** The core generation engine is tightly coupled to specific target outputs (HTTP, gRPC, OpenAPI). Adding a new transport or language output requires modifying core engine files.
- **The Fix:** Define a clean `Generator` interface (e.g., `Generate(ast *expr.RootExpr) ([]*File, error)`). Implement HTTP, gRPC, and OpenAPI generators as entirely decoupled plugins that register themselves with the engine.

### 4.4 Separate Semantic Analysis from AST Definitions
- **The Problem:** The `expr` package violates the Single Responsibility Principle by defining both the shape of the AST data and the complex business logic needed to validate and evaluate it (e.g., `Validate()` and `Prepare()`).
- **The Fix:** Treat the AST purely as a data structure. Move all semantic analysis (validation, default resolution, conflict detection) into a dedicated `analyzer` or `validator` package, creating a clean pipeline: `DSL Execution -> Raw AST -> Analyzer/Validator -> Validated AST -> Code Generators`.

### 4.5 Encapsulate Magic with the Strategy Pattern
- **The Problem:** Complex rules for type mapping and formatting (e.g., converting Go types to OpenAPI types) are scattered across templates or buried in massive utility files like `go_transform.go`.
- **The Fix:** Implement the Strategy Pattern via a `TypeTransformer` interface. Concrete strategies (e.g., `OpenAPITransformer`, `GoStructTransformer`) encapsulate specific transformation rules, allowing templates to delegate complex type formatting logic instead of housing it inline.
