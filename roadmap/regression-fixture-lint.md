# Regression fixture lint repair (#815)

Parent: `abeb200e9f200ec93d330a2040e5f30b1dba5737`.
Scope: three fixture builders only. Preserve authored DSL declarations, evaluation
order, types, metadata, constraints, defaults, views and transport mappings. No
production changes, lint suppressions or threshold changes.

1. [x] Read the issue, three fixtures, their bounded regression matrices and
   owning lint script. Create the managed isolated worktree.
2. [x] Reproduce pinned lint failures and snapshot representative parent outputs.
3. [x] Replace the multipart value if-chain with its equivalent switch. Extract
   private named branch/selected-result and viewed shape/item/wrapper construction
   helpers, invoked at the same points inside the returned DSL closures.
4. [x] Run pinned lint for both affected testdata packages, existing generated
   regression matrices, and bounded parent/candidate/repeat exact-byte output
   comparisons with build/vet. Use verified pushed pin and explicit local source.
5. [x] Freeze exact diff/evidence for fresh independent review. Root owns atomic
   integration, commit/push and closure; do not edit shared guides or main.

This mechanical extraction has no new state/protocol decisions; no formal model
or public policy choice is needed. Intended generated output differences: none.

## Bounded evidence

Pinned golangci-lint 2.14.0 with Go 1.27.2 reproduces the three original failures
(`/tmp/loom-815-parent-lint.log`) and now reports zero issues for
`./http/codegen/testdata ./codegen/service/testdata`
(`/tmp/loom-815-candidate-lint.log`). The root `.golangci.yml` is unchanged.

Nine representative output inputs cover the changed construction responsibilities:

- Multipart custom query text metadata, path string, and constrained integer query.
- Projected union default, selected result branch, and named array branch.
- Viewed HTTP omitted union; HTTP included untagged/map/request-reuse union; and
  JSON-RPC omitted union.

Each parent and candidate module is generated with the same command and built and
vetted. Candidate generation repeats in an independent process; identical repeats
reuse build/vet evidence. All temporary modules use verified pushed pin
`v1.10.0-alpha.9.0.20261013010152-abeb200e9f20` plus explicit local replacement and
`LOOM_DIR`. Snapshot driver/input and evidence live in `/tmp/loom-815-artifacts/`.
The generated trees contain 112 files across these nine cases.

Existing runtime acceptance is the nine-case projected union validation matrix,
nine multipart locations, and selected-view declaration/membership/wire controls.
These retain normal, optional, required, default, nested, collection, string,
selected-view and custom codec branches without adding a redundant test suite.
Logs: `/tmp/loom-815-projected-matrix.log` and `/tmp/loom-815-http-matrices.log`.

## Final acceptance

All listed checks passed. The nine-case projected runtime matrix completed in
43.154s; the HTTP multipart and selected-view matrices passed, including race
checks in their generated harnesses. Parent and candidate generated modules
build/vet; all 112 generated files are byte-identical across parent, candidate and
independent repeat. `/tmp/loom-815-artifacts/comparison.json` records per-case
counts and empty difference sets. No output changes are intended or observed.

Reproduce the affected lint and existing matrices from this worktree:

```sh
$(go env GOPATH)/bin/golangci-lint run --config .golangci.yml ./http/codegen/testdata ./codegen/service/testdata
LOOM_DIR="$PWD" go test ./codegen/service -run '^TestProjectedUnionValidationGenerated$' -count=1
LOOM_DIR="$PWD" go test ./http/codegen -run '^Test(CustomMultipartBindingAccumulator|CustomMultipartBindingsGenerated|ViewedUnionDeclarations|ViewedUnionDeclarationsGenerated|ViewedUnionPackageControls|ViewedUnionPackageMembership|ViewedUntaggedUnionValidatorDependencies|ViewedUnionWire)$' -count=1
```

The final diff contains only three fixture source files and this plan. The
extracted helpers run at the same points within returned DSL closures, and the
unused original Branch declaration is retained for array/selected alternatives
so declaration registration and allocation order remain identical. No public
surface, generator, runtime, lint policy, shared guide or master plan changed.

Status: frozen for fresh independent final review; not committed or pushed.
Keep task-owned temporary evidence until review/delivery finishes, then remove
it after checking for active dependents. Root owns delivery and issue closure.
