# Loom Roadmap

This file contains only unresolved, framework-owned work. Completed work belongs
in the implementation, tests, user documentation, release notes, and Git
history—not in the roadmap.

## Decision Rules

Add roadmap work only when it is unresolved, framework-owned, and backed by a
concrete defect, maintenance cost, or downstream consumer need. Remove an item
as soon as the implementation, tests, and durable documentation are complete.

Do not add compatibility work solely to preserve historical upstream behavior,
runtime security policy better owned by applications, or speculative DSL
surface without a current consumer.

### Generator verification scope

Grow determinism coverage around distinct ordering risks, such as import
collection, emitted file order, and nested map serialization. Extend an existing
representative regression when a reproduced defect or a new generator path
exposes an uncovered risk. Compare exact output bytes across independent
processes; reuse compilation evidence when those bytes and build inputs agree.

Use the existing exported-design compile CI job for protoc validity. A repeated
determinism scan of the entire design catalog is not a standing deliverable.
Choose broader audits explicitly after major generator changes, with a named
risk, bounded inputs, and a cost budget. The remaining testing proposals must
justify their own scope against this rule.

## Active Designs

- [Value meaning and transport projection](value-contract-design.md) defines the
  root-cause repair for inconsistent examples, enums, and defaults. The
  [execution plan](value-contract-plan.md) sequences the work against a
  [consumer inventory](value-contract-inventory.md).
- [Generated transport runtime boundary](codegen-runtime-boundary.md) tracks
  the staged work from issue #267.
