# Chapter 1: Refactoring - Practical Notes

## Why this chapter is the base mindset

Chapter 1 is the behavioral model for all future refactoring work:

- Start from real code that works.
- Ensure behavior is observable and locked by tests.
- Move in very small increments.
- Verify after each increment.
- Repeat.

When teams follow small, verified steps, refactoring becomes repeatable engineering.

## When to open this resource

Use this for:

- You know behavior changed from intended behavior and you want a safer design.
- You need to avoid duplicating logic for a new output/format.
- A function mixes multiple concerns and is risky to edit directly.

## Concrete workflow (named sequence)

### 1) Build the safety harness first

1. Create tests around current behavior before editing.
2. Keep assertions self-checking (expected output or contract checks).
3. Run tests to green.

### 2) Decompose the hot routine into intentions

1. Extract a meaningful chunk into a helper (for example amountFor).
2. Rename locals for clarity.
3. Replace temporary variables that are just lookups with query helpers.
4. Move lookups behind dedicated helpers and inline when possible.

### 3) Remove unnecessary locals and duplicated shape

1. Convert one-off stored values into functions or direct expressions.
2. For accumulation variables, create dedicated total functions.
3. Apply in small steps: Split Loop, Slide Statements, Extract Function, Inline Variable.

### 4) Split responsibilities (calculation vs formatting)

1. Separate calculation/data-building from rendering.
2. Introduce a data object for rendering and pass it to output functions.
3. Make renderers consume data only; avoid repeated direct dependency on source structures.

### 5) Move toward extensibility by design

1. Enrich each record with computed fields in one place.
2. Shift branching logic by type to owning classes/strategies when it appears in multiple places.
3. Keep behavior central and rendering independent.

## Repeatability rules to copy into agent behavior

- One named refactoring per edit batch.
- Compile/build after each action.
- Run tests after each action.
- Commit after each stable step.
- If a step fails, revert and reduce the step granularity.
- Do not mix refactoring and feature logic changes in the same pass.

## Performance note from the chapter

- Many refactoring changes have small performance impact.
- Favor clarity and structure first.
- Tune performance after refactoring if measurement shows a regression.

## Exit criteria for this chapter

- Top-level function is simpler and intent-first.
- Calculation is isolated from rendering.
- Helpers are smaller, named, and single-purpose.
- Adding additional output formats does not duplicate business logic.
- Team confidence in edit safety is higher than before.
