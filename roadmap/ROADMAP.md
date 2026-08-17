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

## Active Designs

- [Generated transport runtime boundary](codegen-runtime-boundary.md) tracks
  the staged work from issue #267.
