# Generated value contract comparison

This test-only harness establishes the artifact baseline for the value-pipeline
repair (#566). It does not prove universal validity of generated Go. The manifest
records the common corpus, coverage, expected legacy failures and reviewed
artifact differences. A passing characterization reproduces a failure; it does
not fix it.

The initial run's source identities, outcomes and verification limits are recorded
in [BASELINE.md](BASELINE.md).

Run from a Loom checkout with Go 1.27 and `make depend` prerequisites:

```sh
go test ./internal/valuecontract ./internal/testdatacompile
LOOM_VALUE_BASE=f5b786b39675e2b5c1466f04f3301b7b337779b6 \
LOOM_VALUE_CANDIDATE="$PWD" \
LOOM_VALUE_RESULTS=/tmp/loom-value-contract/my-unique-run \
go test ./internal/valuecontract -run '^TestCompareRevisions$' -count=1 -timeout=90m
```

All three inputs are mandatory in enabled mode. The baseline must be a full
published commit ID exactly matching `baseline` in the selected manifest. The
candidate may be a local checkout or a full published commit ID. Published
sources use `go mod download` and must report the requested origin hash; local
sources use `internal/loomsource` validation. `LOOM_VALUE_MANIFEST` optionally
selects another reviewed manifest; the default is this package's `manifest.json`.
The selected manifest's exact original bytes are retained. A baseline mismatch
fails before generation. Expected outcomes are never derived automatically from
observed output. The results directory must not already exist.

## Source and input ownership

Local source is captured before building. The retained snapshot contains Git
tracked and nonignored untracked files, including dirty edits, new files and
file modes. Its deterministic hash manifest identifies the content independently
of HEAD. An inventory before and after copying rejects a torn capture; the
snapshot is checked again when the test ends. Both the generator binary and all
temporary-module replacements use this same snapshot. A later live-checkout edit
cannot change the exercised source. Internal relative symlinks retain their
semantics; escaping or absolute links are rejected. The explicitly excluded
`.claude/` agent pointers and Git metadata are not generator inputs; the actual
`.agents/` files remain captured. Snapshot bytes plus metadata reconstruct the
source exercised, without a local Git worktree or uncommitted diff being needed.
The selected manifest and executing checkout's harness source are retained too.
`internal/testprocess` owns every subprocess lifetime.

The executing checkout is also snapshotted as the common input source. New
probe designs and decoder templates come from this retained copy. Existing
specimen packages are copied in full, including supporting Go files, subpackages
and resources. Package-local import and metadata paths are rebound to the copied
`specimen/` subtree, so both revisions execute the same authored specimen rather
than importing different testdata from their respective framework versions.
Unit controls compile two fake revision trees with different authored values,
plus a package-local helper and embedded resource, to check this boundary.

Both checked-in ticktock apps are copied from the common snapshot, preserving
module identity and handwritten app files. Their old `gen/` is excluded;
`loom gen` regenerates it, followed by dependency tidying, build and vet of the
complete app. These existing apps do **not** run `loom example`: that command
adds a competing bootstrap to the handwritten JSON-RPC fixture. Fresh probe and
specimen modules still run `loom example` and compile the generated stubs.
Fixture generation is not a substitute for their transport integration suites.

Original design, specimen and handwritten fixture bytes are retained under
`inputs/` and hashed. Both repetitions and both revisions must share those
hashes. Generation must not change handwritten inputs; only `go.mod` and
`go.sum` may change for source replacement and dependency tidying.

## Execution and evidence

Each revision builds its own CLI and runs every probe twice, serially, in fresh
modules and independent generator processes. Phases are `gen`, `example` for
fresh modules, `tidy`, `build`, `vet`, and where declared `decoder`. Every phase
has its own retained output and status. Later phases do not run after failure.
Missing tools, source errors, timeouts, disk exhaustion, and unexpected phases
or diagnostics fail the test. An expected legacy failure needs an exact phase,
diagnostic substring and Loom issue. An unexpected pass also fails, requiring
removal of the stale expectation. Initial candidate expectations equal the
baseline; each migration changes only the outcomes it intentionally fixes.

Decoder probes execute generated `BuildSendPayload`, obtain its advertised
example from deliberately malformed input, and feed that exact example back
through the builder. Byte and gRPC probes also check decoded authored fields.
The result retains advertised JSON and the decoded service value. Legacy
failures remain real failing decoder tests characterized by the outer harness,
never skipped tests. Broader specimen probes establish generation, compilation,
vetting and artifact evidence; independent schema instance validation is a
separate milestone-4 requirement.

Each `base|candidate/<probe>/run-N/` contains `inputs/`, `artifacts/`, stage logs
and `result.json`. Artifacts retain exact bytes and relative paths; SHA-256
indexes make review compact. Source-dependent module files, unchanged common
inputs and the added decoder harness are excluded from generated artifacts.
Generated stubs, CLI help, Go/protobuf source, OpenAPI JSON/YAML and `gen/loom.json`
are retained without normalization. Repetitions must match exactly, including
decoder observations. Every cross-revision changed/added/removed path needs exact
before/after hashes and a reason in `intended_differences`; stale entries fail.
Unit negative controls cover changed bytes and paths, stale allowances, wrong
failure phases, handwritten input mutation, wrong baseline identity, source
mutation and revision-specific specimen leakage.

For a later generator commit, choose its parent as `LOOM_VALUE_BASE` and use a
reviewed manifest whose baseline and expected outcomes match that parent. Extend
the common corpus and enumerate every intended output difference. For the final
original-baseline comparison, select a separate reviewed manifest with the
original `f5b786b39675e2b5c1466f04f3301b7b337779b6` baseline, original failure
expectations, final candidate outcomes and the complete difference list. The
harness never silently switches expectations. Runtime/schema contradictions
return to the design; a broad failure substring is not an acceptable waiver.
