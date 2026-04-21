# Chapter 2: Principles in Refactoring

Use this file when the agent needs policy guidance on *whether* to refactor and *how to sequence* it.

## 1) Use the exact definition in practice

- Refactoring = behavior-preserving internal restructuring.
- Every suggested step should be selected to reduce future cost, not just to make code look cleaner.
- If a behavior change is needed, handle it outside the refactoring loop and document it separately.

## 2) Two hats approach is a control loop

- **Refactor hat**: restructure only, avoid feature additions.
- **Feature hat**: implement new behavior only.
- Do not blend goals in one step.

## 3) Why refactor (decision support)

- Keep design from decaying as requirements evolve.
- Make change cheaper by reducing duplication and improving structure.
- Improve comprehension so future maintainers (you included) can see intent quickly.
- Reduce bug risk by understanding and clarifying invariants.
- Increase long-term feature speed even if short-term seems slower.

Use these as the justification when challenged by urgency.

## 4) What to refactor first

1. **Opportunistically while adding features/bug fixes**.
2. **Preparatory refactoring** before a known change gets easier.
3. **Comprehension refactoring** when behavior is understood but difficult to reason about.
4. **Litter-pickup refactoring** when small cleanup is immediate and low effort.

Reserve large dedicated refactoring windows for true hotspots, not routine work.

## 5) Refactoring frequency and team rhythm

- Prefer frequent, small edits over isolated big cleanups.
- Refactoring belongs in normal workflow, not as a detached phase.
- Do not overuse branches that drift far from mainline; small frequent integration keeps semantic changes cheaper.

## 6) When to delay

Avoid refactoring when:

- You do not need to touch or understand the area.
- The code is stable API surface and changing it now increases risk.
- Rework cost exceeds expected near-term benefit and a simpler route (small isolated change) is clearer.

In these cases, log the smell and return later.

## 7) When not to refactor at all

- If code is ugly API-only and only called externally, do only what protects callers.
- If rewrite is clearly lower cost than safe incremental change, consider it, but only with evidence.
- In legacy systems without tests, add minimal seams first before broad design changes.

## 8) Required safety baseline

- If tests are missing, treat refactoring as high risk.
- Prefer green regression checks before and after every meaningful step.
- If confidence drops, checkpoint, reduce the change size, and continue.

## 9) Review workflow rule for this chapter

When reviewing:

1. Identify the pressure point (duplication, complexity, wrong ownership, poor modifiability).
2. State expected future behavior benefit of each proposed refactor.
3. Check whether the edit is behavior-neutral.
4. Validate with tests or compile+behavior checks.

