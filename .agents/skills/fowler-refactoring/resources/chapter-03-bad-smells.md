# Chapter 3: Bad Smells in Code

Use this file as a smell atlas: identify what is stinking first, then pick the matching refactoring.

## High-priority smell triage

- **Duplicated Code** -> extract shared method(s) first (`Extract Function`), then deduplicate flows.
- **Long Function / Long Parameter List** -> split responsibility, replace temps/fields with focused helpers.
- **Mysterious Name** -> rename first, then reassess design.
- **Feature Envy / Message Chains / Middle Man** -> move behavior toward owning data (`Move Function`, `Hide Delegate`, avoid unnecessary delegation).
- **Data Class / Primitive Obsession / Data Clumps** -> introduce value objects or small aggregates.
- **Global Data / Mutable Data** -> encapsulate, reduce mutable scope, isolate side effects.

### Priority order (review triage)

- Must-fix before touching behavior:
  - Global Data / Mutable Data
  - Divergent Change / Shotgun Surgery
  - Feature Envy
  - Message Chains / Insider Trading
- High-value cleanup:
  - Duplicated Code
  - Long Function / Long Parameter List
  - Mysterious Name
  - Data Clumps
- Optional cleanup if budget allows:
  - Data Class
  - Lazy Element
  - Speculative Generality
  - Comments smell (when purely explanatory)

## Smell to action sequence (practical)

1. Confirm the smell and where it appears.
2. Confirm behavior target is unchanged.
3. Apply smallest safe extraction/move first.
4. Consolidate and rerun checks.
5. Repeat until intent is clear and structure matches usage.

## Detailed smell map

### Mysterious Name
- If you need to read code twice to guess intent, rename.
- Rename is often a design amplifier, not just style.

### Duplicated Code
- Duplicate expressions or fragments in same/similar context should be collapsed.
- Keep small extraction granularity so behavior risk is low.

### Long Function
- Treat comments as indicators of extract-worthy blocks.
- If too many params/temps block extraction, use temp query, parameter object, or transform helpers first.

### Large Class
- Split by usage subsets and cohesive subsets.
- Prefer extracting coherent subsets before broad structural changes.

### Data Clumps
- If values repeatedly travel together, group them.
- Convert related value tuples into dedicated object/class and slim signatures.

### Primitive Obsession
- If business domain meaning is lost in primitive strings/numbers, wrap into domain types/objects.

### Feature Envy
- If function spends more time on another module's data than own data, move it.

### Divergent Change / Shotgun Surgery
- Divergent: one module changes in many contexts -> split phase or modules.
- Shotgun: one change touches many modules -> centralize behavior.

### Temporary Field
- Move temporary/conditional fields into dedicated object/class.

### Lazy Element
- Delete unnecessary indirection if element only delegates or duplicates behavior.

### Speculative Generality
- Remove unused hooks, flags, parameters, dead wrappers.

### Message Chains / Insider Trading
- Reduce over-navigation and tight coupling; hide delegate and narrow dependencies.

### Comments / Documentation smell behavior
- If a comment explains how code works, prefer extraction and naming to clarify intent.

## When to stop

- Refactoring should continue until the original issue is eliminated and intent is visible in code structure.
- Stop when further cleanup adds low value for current risks.

## Decision rule

If you can name a refactoring and describe how it reduces risk/effort for future changes, you are ready to proceed.
