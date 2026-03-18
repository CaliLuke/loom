---
name: fowler-refactoring
description: A practical, task-oriented implementation of Martin Fowler's refactoring approach, used when improving design without changing behavior.
source: /Users/luca/code/autok/auto-k-server/fowler/chapters
version: 4
---

# Fowler Refactoring

Use this skill when code is becoming difficult to change safely and you need to make design improvements without altering external behavior. The material is organized as a task-oriented map into chapter-based resources.

## Core mindset (start here every time)

1. Refactor and feature work are separate passes. Pick one hat, do it fully, and do not mix both goals in the same commit.
2. Refactor only to make future change cheaper, clearer, or less error-prone.
3. Stay in a tight safety loop: capture a baseline, apply one small step, and run the relevant checks before moving on.
4. Keep changes local and reversible. Use checkpoints so you can return quickly to the last known-good state.
5. Prefer clarity over cleverness: smaller, well-named units and explicit intent.

Chapter 1 is the baseline mindset anchor: it shows that substantial refactoring is built from small, verified steps.

## Decision matrix (fast path)

Use chapter numbers as quick aliases (for example, `3` means Bad Smells, `8` means Moving Features).

- Cleaning a specific defect: `4 (tests) -> 1 (foundation) -> 3 (smell map) -> 1 (one controlled refactor step)`.
- Reviewing for design health: `3 (smell map) -> 2 (principles) -> 1 (foundational process)`.
- Planning a near-term feature: `2 (principles) -> 4 (tests) -> 1 (foundational process) -> 5 (catalog)`.
- Starting from a raw smell without context: `3 (smell map) -> 1 (foundational process) -> 4 (tests)`.
- Quick cleanup before a planned edit: `4 (tests) -> 3 (smell map) -> 6 (first set) -> 1 (one controlled refactor step)`.
- Ownership/context misplaced: `3 (smell map) -> 8 (moving features) -> 1 (foundational process)`.
- Data ownership or value/reference decisions unclear: `3 (smell map) -> 9 (organizing data) -> 1 (foundational process)`.
- Branching complexity dominates: `3 (smell map) -> 10 (simplifying conditionals) -> 1 (foundational process)`.
- API boundaries or parameter shape is confusing: `3 (smell map) -> 11 (refactoring APIs) -> 1 (foundational process)`.
- Hierarchy drift is present: `3 (smell map) -> 12 (dealing with inheritance) -> 1 (foundational process)`.

## Choose your path

Use this before opening a file:

- If you are reviewing existing code for maintainability and risk, use the Review-First path.
- If a bug is already identified and you need a direct edit plan, use Fix-First.
- If you are preparing for a near-term feature, use Feature-Ready.
- If you need a concrete execution sequence, use Implementation Recipe.
- If the issue is specifically ownership/context, data organization, conditionals, API boundaries, or hierarchy, go directly to Chapters 8/9/10/11/12 after `3`/Chapter 3.

## Chapter-driven index

1. [Chapter 1: Refactoring: A First Example](resources/chapter-01-first-example.md)
   - Mindset and high-level process
   - Stepwise extraction and splitting patterns
   - Phase separation and behavior-preserving changes
   - Performance tradeoff stance while refactoring
2. [Chapter 2: Principles in Refactoring](resources/chapter-02-principles.md)
   - Scope, timing, and tradeoffs
   - How refactoring helps features, bugs, and teams
   - Planned vs opportunistic refactoring rules
3. [Chapter 3: Bad Smells in Code](resources/chapter-03-bad-smells.md)
   - Smell-to-refactoring lookup guide
   - Prioritization heuristics for cleanup work
4. [Chapter 4: Building Tests](resources/chapter-04-building-tests.md)
   - What to add before you start refactoring
   - Self-testing habits for safe iteration
   - Practical testing flow for refactoring sessions
5. [Chapter 5: Introducing the Catalog](resources/chapter-05-catalog.md)
   - How to use the catalog format under pressure
   - Where to find mechanics and quick examples
   - Choosing the right refactoring in the same session
6. [Chapter 6: A First Set of Refactorings](resources/chapter-06-first-set.md)
   - High-frequency refactoring families for quick structural gains
   - Practical move selector by symptom
   - Minimal reversible step guidance
7. [Chapter 7: Encapsulation](resources/chapter-07-encapsulation.md)
   - Hidden structure and mutation risks
   - Safe ownership boundary patterns
   - Safe sequencing for record/collection encapsulation
8. [Chapter 8: Moving Features](resources/chapter-08-moving-features.md)
   - Logic/data context shifts across functions, classes, and modules
   - Refactoring-driven function and field relocation
   - Statement and loop motion safety before major redesign
9. [Chapter 9: Organizing Data](resources/chapter-09-organizing-data.md)
   - Split mixed-use variables and clarify data responsibilities
   - Decide where value/reference semantics should live
   - Tighten names and data ownership
10. [Chapter 10: Simplifying Conditional Logic](resources/chapter-10-simplifying-conditional-logic.md)
   - Break apart complex conditional blocks
   - Consolidate repeated condition outcomes
   - Remove branching noise with guard clauses and polymorphism
11. [Chapter 11: Refactoring APIs](resources/chapter-11-refactoring-apis.md)
   - Separate query and mutation APIs
   - Clean constructor, parameter, and function surfaces
   - Move toward narrow, intention-revealing contracts
12. [Chapter 12: Dealing with Inheritance](resources/chapter-12-dealing-with-inheritance.md)
   - Resolve duplication and hierarchy mismatch
   - Pull up/push down fields and methods safely
   - Move from fragile inheritance to delegation where needed

## Path definitions

### Review-First path

Use this when evaluating existing code for readability, risk, and maintainability.

- Start with:
  - [Chapter 2](resources/chapter-02-principles.md) for refactoring intent.
  - [Chapter 3](resources/chapter-03-bad-smells.md) for concrete smell-to-refactoring mapping.
- Before proposing edits, state the smell and the future behavior change enabled by this refactor.
- Require a safety baseline from tests described in [Chapter 4](resources/chapter-04-building-tests.md).
- Expected output: a concise review plan with prioritized smells, target edits, and test checkpoints.

### Fix-First path

Use this when a known issue is already identified.

- Pick one issue and one function/cluster.
- Start by building behavior tests with [Chapter 4](resources/chapter-04-building-tests.md).
- Then isolate behavior with [Chapter 1](resources/chapter-01-first-example.md) extraction/splitting steps.
- Identify the smell in [Chapter 3](resources/chapter-03-bad-smells.md) and pick the matching refactoring.
- Refactor one named step at a time, and verify after each step.
- If any step fails, revert to the last green state and reduce step granularity.
- Expected output: minimal change diff + a short verification log of each successful step.

### Feature-Ready path

Use this when preparing for an upcoming feature.

- Do not implement feature behavior while refactoring.
- Keep preparation separate: create a clear refactor plan, then stop before feature code changes.
- Expected output: a minimal plan identifying reusable boundaries that reduce upcoming feature complexity.

### Implementation Recipe path

Use this when you need executable steps.

- Open the Chapter 1 resource.
- Execute in order:
  - Build baseline tests
  - Extract nested behavior into named helpers
  - Reduce temporary locals
  - Split accumulations into dedicated total functions
  - Split phase between data preparation and rendering
  - Add polymorphic structure only when type branching is repeated across functions
- If you are unsure which refactoring to apply, check the catalog conventions in
  [Chapter 5](resources/chapter-05-catalog.md).
- Expected output: step list with pass/fail check marks after each verification.

### New code smell discovered?

When you detect a smell, read:

- [Chapter 3](resources/chapter-03-bad-smells.md) first.
- [Chapter 6](resources/chapter-06-first-set.md) for move families when the smell is clear.
- [Chapter 8](resources/chapter-08-moving-features.md) when logic or data is in the wrong context.
- Use [Chapter 5](resources/chapter-05-catalog.md) entry conventions for exact mechanics.
- [Chapter 9](resources/chapter-09-organizing-data.md) when data ownership and assignments are unclear.
- [Chapter 10](resources/chapter-10-simplifying-conditional-logic.md) for conditional complexity.
- [Chapter 11](resources/chapter-11-refactoring-apis.md) for API/contract cleanup.
- [Chapter 12](resources/chapter-12-dealing-with-inheritance.md) when hierarchy drift is the smell.
- Execute and validate using [Chapter 1](resources/chapter-01-first-example.md) plus [Chapter 4](resources/chapter-04-building-tests.md).

## Immediate next read

When you are not sure where to start:

1. Read the Chapter 1 mindset in this skill.
2. Open [resources/chapter-01-first-example.md](resources/chapter-01-first-example.md).
3. Identify the smallest behavior block that can be decomposed in 2-5 refactoring steps.
4. Confirm task context with one follow-up chapter:
   - Fixing a known issue: [Chapter 4](resources/chapter-04-building-tests.md) and [Chapter 3](resources/chapter-03-bad-smells.md).
   - General quality pass: [Chapter 2](resources/chapter-02-principles.md) and [Chapter 3](resources/chapter-03-bad-smells.md).
   - Broad redesign planning: [Chapter 5](resources/chapter-05-catalog.md).
   - Safe cleanup before coding: [Chapter 4](resources/chapter-04-building-tests.md), then [Chapter 3](resources/chapter-03-bad-smells.md).
   - Feature-prep pass: [Chapter 4](resources/chapter-04-building-tests.md), [Chapter 3](resources/chapter-03-bad-smells.md), then [Chapter 5](resources/chapter-05-catalog.md).
   - Ownership moves: [Chapter 8](resources/chapter-08-moving-features.md).
   - Data cleanup: [Chapter 9](resources/chapter-09-organizing-data.md).
   - Conditional cleanup: [Chapter 10](resources/chapter-10-simplifying-conditional-logic.md).
   - API cleanup: [Chapter 11](resources/chapter-11-refactoring-apis.md).
   - Inheritance cleanup: [Chapter 12](resources/chapter-12-dealing-with-inheritance.md).
5. Run a green test and repeat.
