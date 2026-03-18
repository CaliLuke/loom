# Chapter 11: Refactoring APIs

Use this when module/adapter boundaries, function signatures, or call patterns are unclear or too coupled.

## Core moves

- Separate Query from Modifier (306): split read-only computation from mutation.
- Parameterize Function (310): collapse duplicated function variants with arguments.
- Remove Flag Argument (314): split behavior instead of overloading one function signature.
- Preserve Whole Object (319): keep coherent object passing instead of unpacked fields.
- Replace Parameter with Query (324): move parameter derivation into callee.
- Replace Query with Parameter (327): move repeated query logic back to caller when needed.
- Remove Setting Method (331): reduce unnecessary mutators.
- Replace Constructor with Factory Function (334): expand construction flexibility.
- Replace Function with Command (337): isolate command-like parameter-heavy orchestration.
- Replace Command with Function (344): return to function when command object no longer needed.

## API-focused diagnosis

1. A method both computes and mutates.
   - Separate Query from Modifier.
2. Similar methods differ only in literals.
   - Parameterize Function.
3. Callers pass a lot of redundant extracted fields.
   - Preserve Whole Object and replace Parameter with Query where appropriate.
4. A setter exists only to initialize or pass-through.
   - Remove Setting Method.
5. Construction needs branching based on context.
   - Replace Constructor with Factory Function.
6. A call path has too many ungrouped parameters.
   - Extract command object, then move back to function once reduced.

## Practical sequence

- Pick one boundary and make behavior/query split first.
- Keep query methods side-effect free and easy to test.
- Move behavior out of signatures before removing arguments.
- Remove dead intermediate APIs only after all call sites are migrated.
- Verify through the same tests expected by your existing integration contract.
