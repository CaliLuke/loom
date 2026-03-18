# Chapter 12: Dealing with Inheritance

Use this when inheritance has become the wrong abstraction and behavior keeps leaking between levels.
If your codebase is not inheritance-heavy, treat these moves as a template for composition/delegation or interface-level extraction checks.

## Core moves

- Pull Up Method (350): move duplicated methods to superclass.
- Pull Up Field (353): promote shared data into common parent.
- Pull Up Constructor Body (355): centralize initialization.
- Push Down Method (359): keep subclass-specific behavior at leaf level.
- Push Down Field (361): move data to the subclasses that own it.
- Replace Type Code with Subclasses (362): replace coded behavior switches.
- Remove Subclass (369): delete dead subclass branches.
- Extract Superclass (375): extract shared behavior into a new abstraction.
- Collapse Hierarchy (380): remove unnecessary inheritance layers.
- Replace Subclass with Delegate (381): replace fragile subtype branching.
- Replace Superclass with Delegate (399): invert inheritance toward composition/delegation.

## Smell selector

- Duplicated logic appears in sibling classes:
  - Pull Up Method/Field.
- Subclasses diverge heavily from parent responsibility:
  - Push Down Method/Field.
- New behavior types appear as flags/type codes:
  - Replace Type Code with Subclasses.
- Inheritance adds complexity but no reuse:
  - Consider Replace Subclass/Superclass with Delegate or Collapse Hierarchy.

## Practical sequence

- Start with duplicate detection between siblings.
- Resolve signature differences with Change Function Declaration as needed.
- Move fields/methods in smallest safe steps with existing coverage.
- Remove dead or redundant hierarchy classes only after all call sites are stable.

## Safety checks

- Keep external behavior unchanged until each extraction is complete.
- Validate all abstract contracts at each boundary.
- Prefer delegation over inheritance when behavior ownership is changing.
