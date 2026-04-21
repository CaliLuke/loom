# Chapter 9: Organizing Data

Use this when one variable, parameter, field, or object shape is being used for multiple meanings.

## Core themes

- Split Variable (240): separate mixed responsibilities in one variable.
- Rename Variable (137) and Rename Field (244): make roles explicit.
- Replace Derived Variable with Query (248): remove unnecessary cached values in favor of source-of-truth computation.
- Change Reference to Value (252) and Change Value to Reference (256): align mutation and identity semantics.
- Encapsulate Record patterns in service of stable ownership.

## How to apply

- I spot one variable updated for different meanings:
  - Split Variable, then Rename Variable on each role.
- I see data used as derived cache:
  - Replace Derived Variable with Query.
- I need to reduce accidental complexity from pass-through data:
  - Prefer Replace Value with Reference or Reference with Value depending on change frequency.
- I see unclear ownership in records:
  - Improve with encapsulation and name alignment first, then move data ownership only when stable.

## Practical sequence

- Start from the narrowest declaration and split outward.
- Make values immutable where meaning is stable.
- Convert one ambiguous shape at a time.
- Retest after each declaration or ownership boundary change.

## Safety checks

- If replacing a value with a reference, ensure equality semantics are still correct.
- If changing references to values, check mutation paths are explicit.
- If changing cached fields to queries, verify performance assumptions for hot paths.

This chapter applies to object-oriented, procedural, and functional code; use the same decisions even if your language does not expose explicit class fields.
