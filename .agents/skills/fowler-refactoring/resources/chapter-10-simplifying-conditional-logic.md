# Chapter 10: Simplifying Conditional Logic

Use this when conditionals are doing too much cognitive work and obscure intent.

## Core moves

- Decompose Conditional (260): isolate each branch condition and action into readable intent.
- Consolidate Conditional Expression (263): merge same-outcome checks into one clear expression.
- Replace Nested Conditional with Guard Clauses (266): return early for exceptional paths.
- Replace Conditional with Polymorphism (272): move type-specific branches into implementations.
- Introduce Special Case / Null Object (289): remove repetitive null/edge handling code.
- Separate Query from Modifier (306): avoid conditionals that both decide and mutate.
- Introduce Assertion (302): encode required invariants once and fail fast.
- Remove Flag Arguments (314): split behavior when flags create branching complexity.
- Substitute Algorithm (195): replace brittle conditional implementations with cleaner alternatives.

## Smell-to-move selector

1. If a conditional has many reasons for returning early:
   - Decompose Conditional, then apply Guard Clauses.
2. If many checks produce the same outcome:
   - Consolidate Conditional Expression, then extract condition.
3. If branching logic changes by type/strategy:
   - Replace Conditional with Polymorphism before adding more flags.
4. If special-case checks are repeated:
   - Introduce Special Case and prune duplicate callers.
5. If condition drives both read and update:
   - Separate Query from Modifier first.

## Practical sequence

- Decompose the conditional into named predicates.
- Extract each non-obvious predicate into functions.
- Collapse repeated outcomes into one guard.
- Keep invariants explicit with assertions where semantics are contract-like.
- Re-test each branch transition.

## Use in this skill

- If branching is the dominant blocker, move to this chapter after Chapter 3 smell mapping.
- For API-level condition cleanup combined with caller churn, bridge into Chapter 11.
