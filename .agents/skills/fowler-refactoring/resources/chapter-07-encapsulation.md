# Chapter 7: Encapsulation

Use this when hidden structure and uncontrolled mutation are making future changes risky.

## Core boundary moves

- Encapsulate Variable (132): control write points.
- Encapsulate Record (162): replace implicit field access with explicit methods.
- Encapsulate Collection (170): avoid uncontrolled direct mutations.
- Replace Primitive with Object (174): give domain meaning to raw values.
- Replace Temp with Query (178): reduce temp-variable coupling in long logic.
- Extract Class (182): pull cohesive data+behavior into one unit.
- Inline Class (186): remove unnecessary object indirection.
- Hide Delegate (189) and Remove Middle Man (192): choose when abstraction is too thin.

## Encapsulation decision checklist

- Is data changed in many places and hard to track? -> Encapsulate Variable/Record.
- Do values appear as plain records/arrays with hidden shape? -> Encapsulate Record/Collection.
- Is domain meaning being lost in raw primitives? -> Replace Primitive with Object.
- Are updates mixed with query logic and hard to read? -> Replace Temp with Query first.
- Are helper functions really operating as their own unit? -> Extract Class.

## Practical sequence

1. Add narrow accessors and replace direct raw writes.
2. Move write operations into one owner.
3. Convert callers to read/write through methods.
4. Fold raw containers into explicit types when stable.
5. Re-test after each phase.

## Tradeoff guidance

- Encapsulation increases safety but can add short-term verbosity.
- Use temporary indirection only when ownership boundaries are real.
- Reverse with Inline Class / Remove Middle Man when abstraction becomes noise.

## Fast use in this workflow

- Before moving behavior across modules, check if ownership boundaries are explicit.
- In legacy modules, use encapsulation first to create safe seams, then larger edits.
