# Loom Roadmap

This file contains only unresolved, framework-owned work. Completed work belongs
in the implementation, tests, user documentation, release notes, and Git
history—not in the roadmap.

## Priority 1: Verification

- Prove representative downstream generation in temporary modules pinned to an
  exact pushed Loom commit, then run compile and contract smoke tests without
  modifying the consumer repository.

## Decision Rules

Add roadmap work only when it is unresolved, framework-owned, and backed by a
concrete defect, maintenance cost, or downstream consumer need. Remove an item
as soon as the implementation, tests, and durable documentation are complete.

Do not add compatibility work solely to preserve historical upstream behavior,
runtime security policy better owned by applications, or speculative DSL
surface without a current consumer.
