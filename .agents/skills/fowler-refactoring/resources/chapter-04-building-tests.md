# Chapter 4: Building Tests

Use this before meaningful refactoring work. This is the safety rail chapter.

## Rule 0: No meaningful refactoring without test feedback

- Refactoring is behavior-neutral only when verified often.
- Treat tests as your primary loop for speed and safety.

## Minimal baseline checklist

1. Identify current behavior contract (inputs/outputs at boundaries).
2. Add or update tests for the specific logic block you will touch.
3. Ensure baseline is green before first structural move.
4. Run tests after each meaningful step.

## What to prioritize

- Extract pure logic into testable units before changing structure.
- Decouple UI, transport, and external calls from business logic first.
- Focus tests on business logic and invariants, not incidental implementation details.

## Recommended refactoring sequence

- Add fixture/sample builders for repeatable object graphs.
- Build assertion checks for known invariants.
- Refactor one semantic unit at a time:
  - nested extraction
  - parameter cleanup
  - helper isolation
  - behavior-preserving rearrangement

## Practical cadence

- If any step changes output, stop and add/adjust tests first.
- If a test suite is slow, split it around edited modules and run focused tests often.
- Keep one 'green bar first' habit: no additional steps when tests are red.

## If tests do not exist

- Add a thin test harness before touching shared logic.
- Start with characterization tests if behavior is undocumented.
- If legacy module is hard to test, add seams gradually and keep edits conservative.

## Use in this workflow

- Review-First path: rely on tests to lock baseline understanding.
- Fix-First path: create at least one regression test before applying larger moves.
- Feature-Ready path: add tests for both current behavior and new behavior boundary.
