# Chapter 8: Moving Features

Use this when behavior or data is in the wrong place and context boundaries are becoming muddy.

## Core moves

- Move Function (198): move behavior to the context where it belongs.
- Move Field (207): move data ownership to the object that actually maintains it.
- Move Statements into Function (213): move duplicated pre-call code into callee.
- Move Statements to Callers (217): move varying code back into caller sites.
- Replace Inline Code with Function Call (222): replace repeated inline behavior with existing function.
- Slide Statements (223): reorder safe, small statement movements to improve locality.
- Split Loop (227): separate loops with mixed responsibilities.
- Replace Loop with Pipeline (231): prefer pipeline style when logic is transform/filter/map.
- Remove Dead Code (237): delete now-unused code when behavior is no longer referenced.

## Use this chapter to choose a move

1. I have logic spread across class/function boundaries.
   - Start with Move Function, then Move Field only after function and caller points are clear.
2. I see duplication that can be removed once call sites are aligned.
   - Try Move Statements into Function, then Replace Inline Code with Function Call.
3. Callers need slightly different prep/side behavior.
   - Start with Move Statements to Callers.
4. A loop does two unrelated jobs.
   - Start with Split Loop.
5. I want to move from index-style loops to declarative data flow.
   - Use Replace Loop with Pipeline in small steps and keep tests after each stage.
6. A code block became a constant nuisance during review.
   - Verify it is dead and use Remove Dead Code.

## Practical sequence

- Copy first, then move in small chunks.
- Reuse tests after each statement/class move.
- If a move feels awkward, step back and re-choose destination context.
- Remove delegating shells only after all callers can use the target directly.
- Prefer reversibility over one-shot refactorings.

## Safety gates

- Test before a move and immediately after each move.
- Validate that side effects stay attached to their intended context.
- If a move touches ordering or control flow, pause and reduce scope before continuing.

## Fast route in this skill

- Chapter 8 is usually for tasks about "where does this belong?" not "how does this work?".
- Use it after identifying the smell in Chapter 3 and the move families in Chapter 6.
