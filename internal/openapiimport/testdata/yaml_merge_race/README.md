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
Upstream v0.41.8 includes that repair but still fails local component references
supplied through aliases or merges, tracked in
[libopenapi #672](https://github.com/pb33f/libopenapi/issues/672) and fixed by
[PR #673](https://github.com/pb33f/libopenapi/pull/673).
The released-library acceptance used `github.com/pb33f/libopenapi v0.42.3`;
the root `go.mod` owns the current exact pin. Upstream cherry-picked PR #673 as
`f4fb28d55d879425d0d07ca7d0efe07ee17455bf`, first included in v0.42.0.
Its patch is identical to the submitted correction even though the PR remains
open. The temporary fork module and self-import renaming are no longer needed.
Component indexing and model construction
read the same effective mappings; repeated aliases retain each semantic occurrence,
while active-path mapping and sequence cycles terminate. The corresponding model
and focused acceptance are in `../../tla/yaml_index_owner/README.md`.

To verify the corrected public API, use the original reproduction import and
change its module requirement to `github.com/pb33f/libopenapi` at the version in the root
`go.mod`, then repeat the two commands above. Framework regression owners are
`TestAnalyzeYAMLNumericSourceNormalization`, `TestAnalyzeYAMLAliasedMergeSequence`
and the invalid-merge tests in `numeric_source_test.go`, plus
`TestImportedYAMLComponentIndex` in `yaml_index_test.go`. They preserve exact
bounds, local overrides and merge precedence while race detection checks the
library's internally concurrent builders.

The importer is internal; returning to the upstream module does not change the
framework's public Go types. That acceptance used Vacuum v0.36.2 with released
libopenapi v0.42.3, including the enum fix in upstream PR #656; no tool-module
override was needed then. The current contract gate builds Vacuum with the
framework's selected parser and verifies the binary dependency version, as
described in the [validation guide](../../../../.agents/skills/loom-framework/references/openapi-validation.md).
