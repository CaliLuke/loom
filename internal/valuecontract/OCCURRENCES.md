# Occurrence and representation comparison

This records the #570 generator checkpoint against published parent
`cd6d2fb03afa2381d42819a9bd50e364b419b3c6`. It is generated-code and
compatibility evidence, not a proof of the Go implementation. The complete,
reproducible corpus and exact before/after artifact hashes are in
[occurrence_manifest.json](occurrence_manifest.json).

The initial immutable candidate content ID was
`d9cdef1a2bfad0b554864b04cd33e548b019abc67323adab1cee3674f2e63351`.
The 62 common-input cases ran both revisions twice in separate processes:
248 result records, 1,368 parent artifacts and 1,390 candidate artifacts.
Every repeated generation was byte-identical. The phase totals were 242
successful generations, module tidy/build/vet runs, 234 successful example
runs, 12 passing decoder checks, 24 characterized decoder failures and six
characterized generation failures. There were no infrastructure failures.
The existing full fixture apps omit `loom example` because they already own
their bootstrap; their generation, tidy, build and vet phases all passed.

Of the 59 existing cases, 58 were byte-identical. The remaining existing case
changed only its design fingerprint. The three new cases cover nullable named
Bytes siblings, selected request bodies with authored headers, and two fields
of the same result declaration selecting different child views.

The initial run deliberately had no artifact allowlist. It exited nonzero on
exactly the four cases below; phase expectations and repeat comparisons passed.
The checked-in allowlist was added only after inspecting every changed path and
the attribution evidence. It accepts exact hashes, never path wildcards or
arbitrary fingerprint changes.

A second immutable checkpoint
`7f07e37db8bbafaddff86f4dddc4845337184c7b15f1fa4c5aa0a1afa9f1cf8d`
reran 29 affected cases at both revisions twice (116 records) using those exact
allowlisted hashes. All phase expectations, exact differences and repeat checks
passed. It includes both full integration fixtures and all three new probes.
After the common-owner constraint/traversal corrections, immutable checkpoint
`ad74f51beef1ede11e933fd35441d499ecf56898ba047f40e1b5b956b0965d32`
reran nine representative cases at both revisions twice (36 records). These
covered nested unions, map collisions, custom codecs, recursion, collection
enums, viewed JSON-RPC streams and all three new occurrence probes. All strict
allowlisted differences, repeat bytes and phase expectations passed. The
corrections introduced no additional generated changes. Later changes confined
to proof/reference and conformance-test adapters do not change the generator.

## Final integration regressions

The final immutable checkpoint
`5bce99fb3261264377a56205d95dbfbe2d1cc87e5bdb9fc0c38594fec05484f2`
reran the four changed-output cases above and two additional exported designs,
`MealPlannerDSL` and `MappedMetadataDSL`, at both revisions twice. All 24 records
passed: 22 successful generation/example/tidy/build/vet runs and two expected
parent generation failures for the nullable sibling view. Every repeat was
byte-identical. MealPlanner's 33 artifacts and mapped metadata's 230 artifacts
matched the parent exactly; the other four cases retained exactly the 28
allowlisted differences. The manifest now retains all 64 cases, including these
two additional designs; the initial 62-case totals above describe the initial run.

The full-suite failures exposed two ownership mistakes in the candidate:

- The occurrence graph rejected distinct authored members sharing a JSON alias,
  even when gRPC placed them in different message/header/trailer representations.
  `TestMappedMetadata` first reproduced the capture panic for `tok:json_name`.
  Source capture now preserves their distinct identities. Resolution rejects a
  supplied shared wire alias as ambiguous, while explicit authored keys work;
  each target still rejects duplicate visible emitted names. Unique wire aliases
  overlapping another authored name retain their existing behavior. The current
  Lean declaration domain requires unique wire aliases, so the differential
  adapter explicitly rejects this additional Go source shape rather than erasing
  its aliases. Its subprocess guard verifies that boundary; broader lowering is
  tracked in #573.
- Collection view projection copied stale generated `name:original` metadata
  from an unprojected body onto a new declaration. The original MealPlanner
  OpenAPI golden caught the resulting component-name change. The collection
  owner now drops only that derived name on its owned copy, retaining authored
  OpenAPI naming metadata and the original declaration. The direct regression
  covers default/tiny views with and without canonical authored names.

The regressions and boundary guards can be rerun without temporary evidence:

```sh
go test ./expr -run 'Test(ValueAlias|ValueUniqueAlias|ValueOccurrenceStillRejects|ProjectCollectionOwnsDerivedNamingProvenance)' -count=1
go test ./internal/valuecontract -run '^TestProductionConstraintGuardFailsClosed$' -count=1
LOOM_DIR="$PWD" go test ./grpc/codegen -run '^TestMappedMetadata' -count=1
```

The manifest reproduction below also checks both exported designs' complete
emitted output, including the MealPlanner OpenAPI documents.

## Attribution and observable behavior

- **JSON-RPC viewed stream:** the complete exported evaluated-design comparison
  differs only in empty `Bases` slices of projected field/view occurrences.
  The new local copy preserves nil rather than making an empty slice.
  Normalizing only empty `Bases` in a diagnostic fingerprint encoder produces
  the same hash for both revisions
  (`4fda0bc5e82da0aa6f171847fe0e7529b6ad96d693526e18397365e5c10e7f5e`).
  No generator/fingerprint normalization workaround was added.
- **Selected body:** the parent HTTP body preparation deletes the authored
  `trace` header value from the shared service/type example map. The candidate
  filters its own copy. Removing only that preserved key diagnostically restores
  the exact parent fingerprint
  (`abfa490e9246f82a31e82c59b5b3acad4c3471e0c63437f8f4550bfc0d5b911c`).
  All emitted code and OpenAPI bytes remain identical.
- **Nullable sibling view:** the parent fails generation with
  `cannot transform nullable field value to a non-nullable representation`.
  The candidate emits the complete 22-file module and passes build/vet.
  Generated server JSON → generated client decoding → service-view conversion
  also passes the race detector for default/tiny orderings and absent/default,
  explicit null and concrete Bytes states. Direct occurrence tests exercise
  reversed field order and default/view combinations.
- **Distinct child views:** only four HTTP type/conversion files change.
  `Full` retains the default child declaration; `Small` receives the Tiny child
  declaration without `Detail`. A generated server-body wire assertion fails
  against the parent because it emits `detail` under `small`; the candidate
  passes the same assertion with the race detector. OpenAPI and service files
  are unchanged.

## Exact intended paths

| Case | Path relative to generated module | Change |
| --- | --- | --- |
| `jsonrpc-viewed-stream-result` | `gen/loom.json` | changed |
| `occurrence-view` | `cmd/svc-cli/http.go` | added |
| `occurrence-view` | `cmd/svc-cli/main.go` | added |
| `occurrence-view` | `cmd/svc/http.go` | added |
| `occurrence-view` | `cmd/svc/main.go` | added |
| `occurrence-view` | `gen/http/cli/svc/cli.go` | added |
| `occurrence-view` | `gen/http/openapi.json` | added |
| `occurrence-view` | `gen/http/openapi.yaml` | added |
| `occurrence-view` | `gen/http/svc/client/cli.go` | added |
| `occurrence-view` | `gen/http/svc/client/client.go` | added |
| `occurrence-view` | `gen/http/svc/client/encode_decode.go` | added |
| `occurrence-view` | `gen/http/svc/client/paths.go` | added |
| `occurrence-view` | `gen/http/svc/client/types.go` | added |
| `occurrence-view` | `gen/http/svc/server/encode_decode.go` | added |
| `occurrence-view` | `gen/http/svc/server/paths.go` | added |
| `occurrence-view` | `gen/http/svc/server/server.go` | added |
| `occurrence-view` | `gen/http/svc/server/types.go` | added |
| `occurrence-view` | `gen/loom.json` | added |
| `occurrence-view` | `gen/svc/client.go` | added |
| `occurrence-view` | `gen/svc/endpoints.go` | added |
| `occurrence-view` | `gen/svc/service.go` | added |
| `occurrence-view` | `gen/svc/views/view.go` | added |
| `occurrence-view` | `svc.go` | added |
| `occurrence-body` | `gen/loom.json` | changed |
| `occurrence-child-views` | `gen/http/svc/client/encode_decode.go` | changed |
| `occurrence-child-views` | `gen/http/svc/client/types.go` | changed |
| `occurrence-child-views` | `gen/http/svc/server/encode_decode.go` | changed |
| `occurrence-child-views` | `gen/http/svc/server/types.go` | changed |

## Reproduce

Use Go 1.27 and the repository generation prerequisites. The harness pins the
published parent and captures an immutable candidate/common source snapshot;
it never compares revision-specific DSL inputs.

```sh
LOOM_VALUE_BASE=cd6d2fb03afa2381d42819a9bd50e364b419b3c6 \
LOOM_VALUE_CANDIDATE="$PWD" \
LOOM_VALUE_MANIFEST="$PWD/internal/valuecontract/occurrence_manifest.json" \
LOOM_VALUE_RESULTS=/tmp/loom-occurrences-new-run \
go test ./internal/valuecontract -run '^TestCompareRevisions$' -count=1 -timeout=90m
```

Use a results path that does not already exist. Temporary snapshots, compiler
outputs, generated modules and diagnostic logs remain through independent review,
then are removed manually under `AGENTS.md`. Retain this record, the manifest,
and the probe source templates; regenerate evidence for future changes.
