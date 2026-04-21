# Chapter 5: Introducing the Catalog

This chapter is the operational map for selecting and executing concrete refactorings.

## What the catalog format gives you

- Each refactoring entry includes: name, motivation, mechanics, and example.
- For work sessions, you use only the first three parts (name/motivation/mechanics) and defer examples unless ambiguity remains.

## How to use it during a live refactor

1. Identify smell / pressure point.
2. Open matching catalog entry.
3. Apply mechanics as small atomic steps.
4. Test after each step.
5. Move to next entry only when this one is verified.

## Mechanics discipline

- Take the smallest safe step available.
- If the path is unclear, pause and revert to simpler extraction or naming step first.
- Prefer reversible edits first (rename/extract/move over broad API-level rewrites).

## Common starting entries for this skill

- **Extract Function**: for readability and test-targeting.
- **Split Phase**: separate setup/calculation/formatting concerns.
- **Extract Class / Introduce Parameter Object**: for cohesion and signature clarity.
- **Move Function / Move Field**: for ownership alignment and coupling reduction.
- **Encapsulate Variable / Remove Setting Method**: for mutable state control.
- **Replace Conditional with Polymorphism**: when repeated branching indicates unstable behavior growth.

## Family lookup by pressure

- I need to remove duplicate behavior quickly:
  - `Duplicate Code` + `Long Function` -> `Extract Function`, then `Replace Temp with Query`, then `Slide Statements`.
- I need safer ownership boundaries:
  - `Feature Envy` / `Message Chains` / `Middle Man` -> `Move Function`, `Hide Delegate`, `Extract Class`.
- I need less mutation risk:
  - `Global Data` / `Mutable Data` -> `Encapsulate Variable`, `Split Variable`, `Change Function Declaration`, `Replace Derived Variable with Query`.
- I need easier extension:
  - repeated type branching -> `Replace Conditional with Polymorphism` (only when intent is stable and duplication is real).

## Minimal catalog entry workflow

1. Confirm smell from chapter 3.
2. Pick one entry family from pressure lookup.
3. Verify with one micro-test before first structural step.
4. Apply only one mechanics chain.
5. Run tests, then repeat or switch families.

## Reading strategy in this environment

- Use the catalog as a lookup, not a prescription.
- Stop when refactoring goal is satisfied for immediate task.
- Return only if the same smell blocks the next change.

## Quick anti-pattern check

If an entry would require huge speculative edits without direct test coverage, pause and:

1. add tests first,
2. create a narrow seam,
3. then continue with smaller steps.
