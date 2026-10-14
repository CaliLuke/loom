# Shared generated declaration ownership — #675

Parent: a231863c0a27e921cd21137719816f584feadfac. Status: candidate implemented and validated; final independent review pending.

1. [x] Reproduce category-based loss through the merger and real FooBAR/FooBar shared-package generation.
2. [x] Model declaration coverage, consistent duplicate admission and occurrence preservation; compare category keys with declaration identity.
3. [x] Obtain independent readiness review of owner design and bounded acceptance.
4. [x] Replace category deduplication with private rendered-Go declaration ownership at same-path artifact composition.
5. [x] Check direct adversarial matrix, generated shared-package/union controls, exact base/candidate bytes, independent process determinism and generated compilation.
6. [ ] Independent final review, atomic integration, published-source verification and issue closure.

## Shared owner

SectionName identifies a renderer category for inspection, not a Go declaration. A shared file may contain several declarations from one section and several sections of one category. Render all contributors, then merge named Go declarations by Go scope identity and their normalized definitions at the shared artifact boundary. Existing File finalization still owns formatting, unused imports and compiled pattern lowering. A private merged section implements Section without adding public plugin metadata or exposing a helper from codegen.

Top-level types, variables, constants and functions share the package namespace; methods use receiver base plus method name, ignoring receiver pointer spelling. Identical named definitions are emitted once. Conflicting definitions are rejected with the same path/symbol diagnostic regardless of service order. Type groups can be split into independent specs. Value specs with several names remain atomic; const groups remain atomic because iota and omitted expressions depend on preceding specs. Do not silently splice overlapping const groups. init and blank-only declarations remain separate occurrences, even when their text matches. Mixed blank/named atomic value groups require explicit treatment so duplicate removal cannot erase intentional effects.

The repair is limited to same-path generated Go artifacts; it must not start suppressing erroneous duplicates within ordinary unmerged files. It must preserve header imports and avoid dropping a non-header first section. Non-Go merge behavior requires a direct control and a deliberate unchanged boundary, not Go parsing.

## Bounded acceptance

Direct merged artifact cases cover same category/distinct type names, methods on different receivers, overlapping multi-declaration union sections, identical named declarations under different categories, conflicting type/function/method definitions in both contribution orders, preserved init and blank effects, whole const-group deduplication and overlapping const-group rejection. Include real FooBAR/FooBar shared-package design, one-service and shared-union controls, header import union and malformed input rejection.

Compare the real collision specimen and existing shared-union/shared-package specimens at parent and candidate. List intended changed files; compile every distinct emitted package. Repeat candidate generation in independent processes and compare exact bytes. No full exported-design audit is part of this ticket.

## Readiness evidence

The real accepted FooBAR/FooBar design produces only FooBAR in the shared file
on the parent. The direct ten-case rendered matrix fails every expected defect:
same-category declarations/effects vanish, differently categorized identical names
are duplicated, conflicts are silently accepted, and const groups lose ownership.
The first test attempt had a require.Error return-value compile mistake; it was
corrected before the recorded behavior run. `/tmp/loom-675-red.log` is the actual
executed red witness (0.580 s), not the preliminary test compile failure.

TLC 1.7.4 passes the identity design: 20 generated / 18 distinct states. The
category configuration fails declaration coverage; a separate first run also
reproduced silently admitted conflicting definitions. Logs remain temporary.

### Implementation constraints for independent review

- Only coalesced Go files get the declaration pass. Preserve the normal File
  finalizer and single-file behavior. A private merged section renders its
  constituent sections once, performs declaration composition, and writes the
  resulting source for ordinary finalization; it needs no new public metadata.
- Resolve package-level identity across type/var/const/function kinds. Methods
  normalize pointer and generic receiver forms to the declared receiver name.
- Compare normalized definitions excluding ordinary documentation, but retain
  compiler directives in equivalence checks. Preserve the first occurrence’s
  documentation and source order; do not sort variable initializations.
- Type specs are independently mergeable. Multi-name var specs and whole const
  groups stay atomic. Reject partial overlaps; exact duplicate const groups are
  removed as a whole so iota/implicit-expression meaning cannot change. Blank-only
  declarations and init functions remain occurrences. A mixed named/blank var spec
  with a repeated named definition must reject rather than erase its blank effect.
- Existing header import union remains, but only actual source headers may be
  skipped. Check explicit conflicting import bindings before comparing definitions
  so identical selector text cannot hide distinct imports. Leave established
  non-Go merge semantics unchanged, with a control proving that boundary.
- Conflict diagnostics name relative output path and sorted conflicting symbol
  identities only; absolute temporary paths and contributor order must not leak
  into their stable text. Parser errors retain normal syntax diagnostics.

Further direct controls to add with implementation: group atomicity and effect
preservation, namespace cross-kind collision, generic receivers, import union and
explicit binding conflict, malformed syntax, comments/directives, first non-header
section preservation, and non-Go/single-file unchanged behavior. Generated
acceptance reuses the existing shared-union integration and a selected shared-package
case; no unrelated exported design corpus is planned.

## Candidate acceptance

Independent readiness approved the shared artifact owner, with explicit package
and import-binding rejection controls added. Production is private to generator
composition; the public Section documentation now distinguishes inspection
categories from Go declaration identity. Three contributors, unnamed first
sections, ordinary docs/directives, tuple overlaps and declaration kind conflicts
are checked directly. A compiled standalone artifact verifies two blank effects,
two init functions, const iota values and imports contributed by both files.

Go 1.27.2, jsonv2, candidate direct and existing shared-union races pass (3.309 s).
All three generated specimens compile: collision, shared union and ordinary
shared packages (5.781 s under the race-enabled harness). The identical specimen
on the parent fails compiling common.FooBar, while its ordinary and shared-union
controls compile. Across 25 generated artifacts, the only changed file is
collision/gen/types/common/foo_bar.go, which now includes FooBar. The other 24
files match the parent exactly. A separate candidate generation process emits
all 25 byte-identical artifacts. Package lint reports zero issues. These receipts
are /tmp/loom-675-comparison.json, /tmp/loom-675-specimens-{base,candidate,repeat}.log
and /tmp/loom-675-lint.log; raw outputs remain temporary.

TLC identity ownership passes 20 generated / 18 distinct states; the retained
legacy configuration reproduces DeclarationCoverage failure. The model does not
prove the parser or runtime compiler; concrete tests cover their correspondence.
No exported-design corpus sweep was started. Final independent exact-diff review
and integration remain pending.
