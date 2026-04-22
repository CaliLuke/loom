# Architecture Review: Loom Framework

## 1. Findings

### CR-001 (S2): DSL Evaluation Engine Thread-Safety (Global State)

**Impact:** The DSL evaluation phase is not safe for concurrent independent runs because it relies on process-global state. This is primarily an architectural limitation for embedders, tests, and future long-running/programmatic use rather than an immediate problem for the one-shot `loom gen` CLI workflow.
**Evidence:** `eval/context.go` explicitly declares a global variable `var Context *DSLContext`. The `eval.RunDSL()` function reads from and mutates this single global instance without any `sync.Mutex` or other synchronization primitives.
**Recommendation:** Treat removal of the global `Context` as cleanup to enable isolated evaluation contexts when Loom needs concurrent or embedded execution.
**Confidence:** High

### CR-006 (S2): Import Alias Collisions in Code Generation

**Impact:** Import alias conflicts remain a generator-quality risk in projects with overlapping package names. When alias selection is not deterministic and readable, generated code becomes harder to trust and debug.
**Evidence:** Extensive test fixtures (e.g., `codegen/import_alias_safety_test.go`) show this has been a recurring issue. The framework relies on tracking "seen" imports, which can become non-deterministic depending on the AST traversal order.
**Recommendation:** Continue hardening alias generation toward deterministic, readable names that stay collision-free across packages.
**Confidence:** Medium

---

## 2. Architecture Assessment Summary

Loom acts as a declarative API generator, translating a custom Go-based DSL into executable AST representations (`eval`/`expr`), which are then converted into server, client, and transport code (`codegen`).

- **DSL Evaluation (`eval/` & `dsl/`)**: Uses global state that is acceptable for the current CLI flow but constrains concurrent or embedded use.
- **Import Management (`codegen/`)**: Import alias handling has active regression coverage, but deterministic naming remains an area worth continuing to harden.

---

## 3. Test/Verification Gaps

- **Concurrent Compilation:** There are no tests verifying the thread-safety of the evaluation engine.
- **Generator Robustness Inputs:** The codebase lacks targeted fuzzing or equivalent hostile-input coverage for identifiers, names, and edge-case DSL shapes that stress code generation.

---

## 4. Prioritized Remediation Plan

1. **P2 (Import Alias Determinism):** Continue hardening import alias selection so generated files remain deterministic and collision-free in projects with overlapping package names. Prefer stable, readable aliases over opaque hash-heavy naming.
2. **P2 (Evaluation Context Cleanup):** Treat the global `eval.Context` as architectural cleanup for future embeddability and concurrent/programmatic use, not as an immediate production blocker for the one-shot CLI workflow.
3. **P3 (Targeted Verification):** Add focused coverage for evaluator concurrency assumptions and generator robustness on hostile identifiers and edge-case DSL shapes.

---

## 5. Open Questions / Assumptions

- **Global Context Deprecation:** How disruptive would it be to the community (and external plugin authors) to modify the signature of `eval.Register()` and `eval.RunDSL()` to require a context object?
- **Import Alias Strategy:** What readable deterministic aliasing policy best balances collision avoidance with stable generated output?
