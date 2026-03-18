---
name: ai-readiness-review
description: Framework for auditing Go codebases for AI-assisted development readiness, with emphasis on DRY design, modularity, traceability, and fast verification.
---

# AI Readiness Review: Architecture for Safe & High-Velocity AI Development

This skill provides a framework for auditing a Go codebase to ensure it is optimized for AI-assisted development. It focuses on making the system **partitionable**, **traceable**, and **fast to verify**, allowing AI agents to work with high autonomy and minimal context drift.

## Core Pillars of AI Readiness

### 1. DRY Code & Single Source of Truth (CRITICAL)

**This is the highest-impact pillar.** Duplicated logic is the #1 source of bugs in AI-assisted codebases. When the same concept lives in 3-5 places, an AI agent will fix one and miss the rest — producing a codebase that is subtly inconsistent. Every fix becomes a scavenger hunt.

- **One Concept, One Location**: Business logic, validation rules, default values, error messages, and format strings must each live in exactly one place. If you find the same logic in multiple files, extract it into a shared function or constant.
- **Shotgun Surgery Detection**: If changing a single behavior (e.g., a default value, a validation rule, a display format) requires edits in more than one file, that is a **HIGH** severity finding. The fix is to extract the shared logic, not to document all the locations.
- **Constants over Inline Values**: Magic numbers, default strings, and threshold values repeated across files must be extracted to named constants. An AI agent cannot reliably grep for `"proposed"` across 10 files, but it can follow `graph.DefaultStatus` to a single definition.
- **Beware Structural Duplication**: Parallel switch statements, repeated type-to-X mappings, and copy-pasted handler patterns are especially dangerous. AI agents copy the pattern faithfully, including any bugs in the original.

### 2. Context Partitioning & Modularity (HIGH)

AI agents perform best when context is logically isolated.

- **File-Level Cohesion**: The unit of context is the **file**, not the struct. Large structs are acceptable if implementation is split across focused files (e.g., `search.go`, `task_deps.go`).
- **Bloat Threshold**: Files over **500 lines** mixing unrelated sub-domains are HIGH risk. They force agents to ingest irrelevant tokens and increase hallucination risk.
- **Idiomatic Boundaries**: Do not flag idiomatic Go patterns (e.g., single-implementation interfaces, receiver methods on large structs) as AI risks without a concrete failure scenario.

### 3. Test Velocity & Verification

AI velocity is limited by the latency of the feedback loop.

- **Verification Speed**: Can core logic be verified without external infrastructure (TypeDB/Postgres)? Pure unit tests enable 10x faster agent loops than "Steel Thread" integration tests.
- **Mockability without Boilerplate**: Interfaces should be small enough that an AI can generate a mock/fake without significant "not implemented" boilerplate.

### 4. Self-Healing Readiness

Code should allow agents to catch and fix their own mistakes early.

- **Compile-Time Safety**: Enforce schema concepts (types, relations) via **constants/enums**. Replacing `"task"` with `graph.TypeTask` allows the compiler to catch AI mistakes before a test even runs.
- **Error Specificity**: Errors should be descriptive enough for an AI to locate the root cause. "insert failed" (LOW) vs "insert persona PER-005: duplicate display_id" (HIGH).
- **Recovery Hints**: Use a `Recovery` field in error objects to suggest the immediate next step (e.g., "Use search_graph to verify IDs").
- **Stdlib over Hand-Rolled Logic**: AI agents frequently reimplement standard library functions (e.g., writing a manual substring search loop instead of `strings.Contains`, or a custom min/max instead of the `min`/`max` builtins). Flag any hand-rolled logic that duplicates `strings`, `slices`, `maps`, `sort`, or other stdlib packages. This is a self-reinforcing anti-pattern: once hand-rolled code exists, AI agents treat it as a local convention and propagate the pattern.

### 5. Contextual Traceability

Autonomous agents must be able to debug via logs.

- **Trace Propagation**: Ensure `request_id` and `project_id` propagate into background goroutines and workers.
- **Log Correlation**: If traces are lost ("Contextual Amnesia"), the AI cannot correlate a background crash with its own previous action.

---

## Audit Workflow: The "Plausible Failure" Rule

**MANDATE**: For every finding rated **MEDIUM** or **HIGH**, you MUST describe a specific, plausible **Agentic Failure Mode**. Abstract risks (e.g., "side effects are unpredictable") are not sufficient.

| Category          | Check                                        | AI Risk                                                                  |
| :---------------- | :------------------------------------------- | :----------------------------------------------------------------------- |
| **DRY**           | Same logic duplicated across 2+ files?       | **HIGH**: AI fixes one location, misses the rest — silent inconsistency. |
| **DRY**           | Behavior change requires edits in 3+ places? | **CRITICAL**: Shotgun surgery — every fix is a scavenger hunt.           |
| **Context**       | Files > 500 lines with mixed domains?        | **MEDIUM**: Token waste and context window saturation.                   |
| **Observability** | Context propagation in `go func()`?          | **HIGH**: Log-based debugging is broken for agents.                      |
| **Safety**        | String literals for schema concepts?         | **MEDIUM**: AI loses compile-time self-healing loop.                     |
| **Velocity**      | Logic requires infra (TypeDB) to test?       | **HIGH**: AI "Code-Test-Fix" loop is too slow.                           |
| **Observability** | Error specificity & Recovery hints?          | **MEDIUM**: AI stalls on errors instead of pivoting.                     |
| **Safety**        | Hand-rolled logic that duplicates stdlib?    | **MEDIUM**: AI reimplements instead of using the library.                |

---

## Output Template

Generate the report in the project root with a unique, descriptive filename (e.g., `AI_REVIEW_ARCH_20231027.md`).

### Structure

1. **Summary**: High-level status of AI-safety and velocity.
2. **Issue Checklist**: Grouped by issue type. Use `[ ] SEVERITY - Short Description`.
3. **Detailed Evidence & Roadmap**: For each checklist item, provide:
   - **Severity**: [CRITICAL/HIGH/MEDIUM/LOW]
   - **Location**: `path/to/file.go:line`
   - **Agentic Failure Mode**: Concrete scenario where an AI would fail or loop.
   - **Suggestion**: Actionable refactoring steps.

```markdown
# AI Readiness Review - [Description] - [Date]

## Summary

[Overview]

## Issue Checklist

### Context & Modularity

- [ ] MEDIUM - [Short Title]

### Test Velocity

- [ ] HIGH - [Short Title]

...

## Detailed Findings

### [Issue Title]

- **Severity**: [HIGH/MEDIUM]
- **Location**: `path/to/file.go:line`
- **Agentic Failure Mode**: [Concrete scenario, e.g., "The AI will enter a retry loop because this error lacks the duplicate ID"]
- **Suggestion**:
  1. [Step 1]
  2. [Step 2]
- **Status**: [ ] Pending

---
```
