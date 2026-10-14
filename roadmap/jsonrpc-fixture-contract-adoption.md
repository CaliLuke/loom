# JSON-RPC SSE fixture contract adoption (#928)

1. [x] Reproduce exact-tree failures in ticktock and mixedtick on main. Preserve
   the parent log and inspect all generated differences, including newly retained
   blank interface assertions after the shared declaration owner correction.
2. [x] Regenerate both fixtures with the current local framework. Do not edit
   generated files or weaken exact-tree/provenance assertions.
3. [x] Compare every generated artifact with the checked-in parent and with a
   second independent generator process. Compile both apps and run existing SSE
   fixture/mixed-transport behavioral regressions with race checking.
4. [ ] Independently review the exact fixture diff, commit atomically, push,
   close the report and update the master correctness inventory.

The manifest intentionally fingerprints the complete evaluated design. Added
expression fields such as RequestContentType and TagBinding change its input
structure even when nil. Shared Go artifact composition also now preserves every
blank declaration occurrence, including repeated compile-time interface checks.
The fixtures must follow those shared owners; do not add per-fixture exceptions
or strip generated provenance from comparisons.

Parent evidence: `/tmp/loom-ci-fixture-digest-parent.log`. Scope is the two
existing JSON-RPC SSE fixture applications, not the exported-design corpus.

Acceptance: two manifests adopt the current complete evaluated-design digests;
three blank interface assertions are retained across the two merged stream files.
Those four files are the only differences among 35 generated artifacts; all 35
match an independent repeat exactly. Both fixture apps compile with `-race` and
`-mod=readonly`; module files remain unchanged after tidy. The existing fixture
regeneration and all mixed SSE tests pass with races (18.497s), retaining exact-tree
comparison. No generator/runtime change or suppression is included.

Evidence: `/tmp/loom-928-comparison.json`, `/tmp/loom-928-artifacts.diff`,
`/tmp/loom-928-generation-receipt.json`, `/tmp/loom-928-sse-final.log`, and
`/tmp/loom-928-{ticktock,mixedtick}-compile.log`.
