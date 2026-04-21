# Chapter 6: A First Set of Refactorings

Use this when you need a small, high-confidence move set that appears repeatedly in real work.

## Core refactoring families

- Extract Function (106): split intent from mechanism.
- Inline Function (115): collapse unnecessary indirection.
- Extract/Inline Variable (119/123): isolate expressions.
- Change Function Declaration (124): rename and reshape contracts.
- Rename Variable (137): clarify meaning.
- Replace Temp with Query (178): reduce temp pressure.
- Introduce Parameter Object (140): keep related args together.
- Combine Functions into Class (144) / Combine Functions into Transform (149): group helper behavior.
- Split Phase (154): split pipeline stages.

## Use this chapter as a move selector

1. I need clearer intent in a block of logic.
   - Start with Extract Function, then add small helper names.
2. Same block appears in multiple places.
   - Extract Function, then Replace Inline Code with Function Call.
3. Too many local values or parameters.
   - Replace Temp with Query, then Introduce Parameter Object.
4. Function does more than one conceptual stage.
   - Split Phase after stable helper boundaries.
5. Too many tiny utilities around one structure.
   - Combine Functions into Class or Transform.

## Practical sequence (recommended)

- Pick one smallest behavior slice.
- Refactor with one technique only.
- Re-run tests.
- Confirm intent is clearer (name communicates goal).
- Repeat.

## Safety rules from chapter 6

- Start with naming/structure that makes behavior explicit.
- If a move adds confusion, stop and inline it back.
- Short, reversible steps beat wide structural moves.

## When to use with this skill

- In Fix-First and Review-First, after deciding intent from chapter 3.
- In Implementation Recipe, use as default starter list before deeper catalog work.

## Quick anti-pattern check

If you can't describe the behavior that the new function/class adds in one sentence, do not extract yet.
