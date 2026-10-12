# Projected carrier package ownership follow-up

Parent: `651372ce6ff05ed1db495369fd51d48df0c0ec75`.
Worktree: `/Users/luca/.codex/worktrees/projection-carrier-packages/loom`.
Owner: root; readiness approved by `/root/grpc_admission_design` before implementation.
The reviewer required lowering before initial name allocation and explicit nested/recursive carrier coverage; both are included.

Remote race job `114936923404` at `9880865a` fails the existing
`TestStructPkgPathFrameworkImportNamesCompile` matrix: generated service helpers
refer to `common.CardView` and similarly aliased custom packages, although view
carriers are declared in the service's views package. This is a regression in
conversion graph snapshot ordering following recursive projection delivery.
It is separate from #733's collection namescope allocation and #784's HTTP
union declaration registry. No previously closed ticket is re-closed here.

1. Reproduce with the existing generator `common` case and a direct service
   analysis test covering both conversion directions, nested carriers, package
   aliases, and unchanged canonical service locations.
2. Model the current order: build conversion snapshot, lower carrier location,
   emit references. Confirm the snapshot retains an obsolete location. Compare
   location lowering before snapshot construction, with the canonical graph
   still retaining its authored package.
3. Correct the shared projected-carrier preparation boundary before conversion
   or reference consumers. Do not strip metadata from each conversion helper,
   add special package names, or modify NameScope to ignore authored locations.
4. Run full service tests, the selected generated package cases (ordinary,
   framework-name collision, and views-package collision), and recursive
   projection regressions. Compare parent/candidate outputs for the same
   bounded designs, list intended reference/helper differences, compile and vet
   candidate modules, and compare fresh-process repeats exactly. Parent build
   failure is expected for regression cases; preserve its diagnostics.
5. Update the owning model README and framework guidance, freeze explicit file
   hashes, obtain independent exact-diff review, then commit and push separately.

Acceptance: every projected carrier reference names its declaring views
package; canonical type references retain custom packages/aliases; selected
view fields remain unchanged. No public API, schema, wire behavior, dependencies
or design-validation policy should change. Existing unaffected generated
fixtures must remain identical. The model checks finite ordering/location
invariants; actual Go naming, aliases, recursion and compilation remain test
obligations.

## Baseline and model checkpoint

The existing generator `common` case fails at both `common.CardView` references
(31.674 seconds). New direct service cases reproduce stale root carrier locations
for ordinary and framework-name packages; explicit alias reanalysis is included.
The bounded `CarrierLocation.tla` legacy control fails
`ReferencesNameDeclaration` in four states. Location lowering before snapshot
construction passes both invariants and termination (five generated/four
distinct states). It models three finite carriers, not the Go graph walker.
The concrete candidate is to run the existing projected-graph `removeMeta`
operation before `buildTypeInits`/`buildProjections` in `buildProjectedType`,
instead of afterward. Earlier projection reused mutable metadata so late cleanup
incidentally reached conversions; detached snapshots exposed the ordering error.
Canonical graph metadata must remain untouched.

## Final bounded evidence

Go 1.27.2 darwin/arm64, pinned golangci-lint 2.14.0. No dependencies changed.
Full `go test ./codegen/service -count=1` passes (9.662s), including the new
recursive/package alias test. The existing generator cases `common`, `context`
and `shopviews` pass generation, build and vet (24.454s). Scoped pinned
`golangci-lint run ./codegen/service/...` reports zero issues.

Generated recursive projection wire checks pass for HTTP, gRPC and JSON-RPC
(10.067s, 10.676s, 9.998s). Their existing fixtures preserve recursive selection,
excluded-secret safety and compilation across the affected conversion owner.

Retained parent/candidate/repeat generation uses the existing generator fixture
and module helpers in separate `go test` processes. Three existing custom-package
inputs cover ordinary, framework import and views-package collision locations.
A fourth reuses `ResultWithRecursiveCollectionOfResultTypeDSL` with custom package
placement on its result types; the fifth uses that fixture unchanged as a control.
Each standalone module pins pushed parent
`v1.10.0-alpha.9.0.20261012181439-651372ce6ff0` and explicitly replaces it with the
selected source. No corpus expansion or checked-in fixture edits are needed.

All 279 generated files match independent candidate repeats exactly. Differences
from parent are limited to four service files: `gen/shop/service.go` in each of
the three package cases replaces two stale `CardView` construction references
with the actual views package. The recursive specimen's service file corrects
view carrier references and corresponding private helper identifiers/calls.
Canonical references, selection logic, manifests, schemas, transport files and
example files are unchanged. The seven-file unaffected control is byte-identical.
All five retained candidate modules pass tidy/build/vet; parent recursive output
fails on the same undeclared custom-package view names as the existing common
baseline. Parent control shares the verified identical generated source.

Comparison artifacts and exact diffs are in
`/tmp/loom-projection-packages-comparison`; original direct/generator failures and
model results are in the task-prefixed `/tmp/loom-projection-package*` and
`/tmp/loom-carrier-location-*` logs. Temporary validation driver removed from the
candidate checkout. Preserve evidence through independent review, then remove
only task-owned scratch after active-process checks.

Root-owned framework guidance delta: projected carrier package placement must be
lowered before any name allocation or detached conversion snapshot; canonical
service locations remain untouched. Keep CarrierLocation model, direct nested
and recursive references, and custom-package compilation coverage aligned.

## Main integration evidence

Integrated onto pushed `4f7892a2` after the Extend composition repair. The seven
implementation/model/test files match the independently approved candidate;
framework guidance merges without changing either task's ownership rule.
All 279 files from the five selected designs match the reviewed candidate exactly,
and a fresh independent generation also matches all 279 bytes. Thus the recorded
parent differences and compiled-module evidence remain applicable. The existing
three custom-package generator cases compile/build/vet again, and the direct
projected-carrier package regression passes on integrated source. No additional
output difference is introduced by Extend composition.

Comparison evidence is retained at `/tmp/loom-carrier-integrated` through final
integration review; generator helper removed from the checkout. This appendix
and the merged framework paragraph are the integration review delta.
