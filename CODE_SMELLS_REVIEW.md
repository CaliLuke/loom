# Codebase Smell Review

The previous pass mixed actionable issues with expected invariants, fixture-only references, and style complaints that do not justify code churn. The high-confidence items from that list have been fixed.

Resolved in code:

- silent cleanup and parse errors in generator and CLI paths
- ignored errors in test utilities and integration helpers
- repo-owned legacy naming in JSON-RPC framework aliases and DSL spec sample data

Removed from the checklist as non-actionable or too weakly justified:

- fixture and example imports that intentionally exercise external packages
- `panic` sites that enforce internal invariants or constructor preconditions
- mutation-only `codegen.Walk` calls whose callbacks cannot fail
- `Flush()` calls on interfaces that do not return an error to the caller

If a new smell review is opened, it should only include issues that are both:

1. concretely fixable without changing public contracts or intended fixture coverage
2. clearly worse than the surrounding code after repo context is considered
