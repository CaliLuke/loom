# libopenapi YAML merge race reproduction

This standalone diagnostic uses libopenapi directly; it does not import framework
code. Copy `merge_race_test.go.txt` to `merge_race_test.go` in a temporary directory
and create `go.mod` with:

```text
module example.com/mergerace

go 1.27.2

require github.com/pb33f/libopenapi v0.41.6
```

Then run:

```sh
go mod tidy
go test -run '^TestMergedComponents$' -count=1 -v
go test -race -run '^TestMergedComponents$' -count=1 -v
```

With Go 1.27.2 darwin/arm64 and libopenapi v0.41.6, the ordinary control passes
and preserves merged minimum 5/maximum 10; race detection reports a write in
`utils.CheckForMergeNodes` at utils.go:1296 against a read in
`utils.mergedNodeContent` at utils.go:444. Both goroutines originate inside
`datamodel/low/v3.createDocument` at create_document.go:349.

The final race-enabled reproduction failed in 0.616s; the ordinary control passed
in 0.498s. As with any race detector witness, scheduling may affect manifestation;
the identified accesses have no synchronization. The latest upstream tag and HEAD
were `c4c4be74abe32f0201f6f3086a951178975b00e3` on 2026-10-13. No application
concurrency is needed.

The reviewed report is [libopenapi #657](https://github.com/pb33f/libopenapi/issues/657),
and the ownership repair is [PR #658](https://github.com/pb33f/libopenapi/pull/658).
The framework adopts that repair through the ordinary module
`github.com/CaliLuke/libopenapi`, pinned in the root `go.mod`. Its source is
libopenapi v0.41.6 plus the reviewed immutable merge-view change and mechanical
module/self-import renaming. Consumers need no `replace` directive.

To verify the corrected public API, change the reproduction import and module
requirement to `github.com/CaliLuke/libopenapi` at the version in the root
`go.mod`, then repeat the two commands above. Framework regression owners are
`TestAnalyzeYAMLNumericSourceNormalization`, `TestAnalyzeYAMLAliasedMergeSequence`
and the invalid-merge tests in `numeric_source_test.go`. They preserve exact
bounds, local overrides and merge precedence while race detection checks the
library's internally concurrent builders.

Return to the original module path when a stable upstream release includes the
immutable merge-view fix and passes these regressions plus upstream metadata,
render/reload and alias/merge controls. Rename all importer and test imports
together, update `go.mod`/`go.sum`, and verify a separate consuming module can
fetch and compile without replacement directives. The importer is internal;
this temporary module identity does not change the framework's public Go types.
Vacuum is installed into an independent tool module and retains its separate
pin for the enum fix in upstream PR #656; its Go types never cross this boundary.
