# Artifact write intent readiness

Status: implementation complete; candidate validation and independent review
precede integration. The user explicitly accepted rejection of create-only
artifacts inside `gen/` and support outside it.
Owning ticket: #670. Candidate parent: `eb5a169baa701891100c13790de54fe6fd8a8a36`.
Historical evidence baseline: `4535a403048510f055fc59ec3dfed41e3769ad58`.
Root integrates the independent authentication fix separately. Toolchain observed:
`go1.27.2 darwin/arm64`. Read the full live #670, #741 and #751 bodies and comments.

## Scope and acceptance

Preserve `codegen.File.SkipExist` from declaration to live installation. An
existing external destination must retain its bytes and filesystem identity;
an absent destination must be created; an external destination created after
staging must not be overwritten. Replaceable artifacts and whole-tree `gen/`
replacement retain their existing behavior. Only actually installed paths are
reported by the public CLI, one path per line. Installation failure restores
previous generated output and removes only artifacts installed by this attempt.

#741 is the same ownership principle but an independently defective declaration:
`ExampleInterceptorsFiles` performs cwd-relative `Stat` and never sets
`SkipExist`. Correcting #670 cannot repair an undeclared policy. Recommend a
separate atomic #741 follow-up: remove its two `Stat` guards and declare
`SkipExist: true`, reusing the writer fixed here. Its server/client fixtures are
useful unchanged compilation controls in this ticket; do not bundle the source
change into #670. The owner's live comment also identifies the shared writer as
policy owner. #751 changes staging filesystem placement; it is separable and
excluded. This task must not silently introduce a copy fallback for EXDEV.

## Current ownership and intended representation

1. `codegen/file.go: File.Render` checks `Stat` in its supplied directory, then
   unconditionally `WriteFile`s. Its check misses dangling symlinks and races.
2. `codegen/generator/generate.go: writeFiles`, `fileRenderResult`,
   `collectWriteResults`, `computeOutputs`, and `Generate` retain paths alone.
   `mergeFilesByPath` silently picks the first file's `SkipExist` policy.
3. `cmd/loom/gen.go: mainT` prints those paths; `Generator.Run` splits stdout.
   `cmd/loom/main.go: generate` passes the path slice to the transaction.
4. `cmd/loom/generation_transaction.go` validates staged files, swaps `gen/`,
   then replaces every external destination using rename and backup bookkeeping.
   It never receives the declaration, hence overwrites protected live files.
5. `cmd/loom/vet.go` uses the same `Generator.Run` stdout channel for its report.
   The helper/launcher protocol change must preserve this route explicitly.

Implemented narrow representation: `codegen.Artifact { Path string; SkipExist bool }`
as the result of `generator.Generate`; preserve policy in the existing render
work/result pipeline and output cleanup map. `File.Render` can retain its
existing return value because `writeFiles` still owns the corresponding File.
Serialize artifacts as a JSON array on the private helper stdout channel.
Have `Generator.Run` return raw bytes; generation decodes artifacts, while vet
decodes the same bytes as its report. The public CLI continues printing only
installed paths. No path-only compatibility fallback, second generator API,
plugin changes, or public installation API is needed.

Reject conflicting `SkipExist` policies for the same normalized output path
before any rendering; equal-policy contributors still merge sections/imports.
Validate policy before merging rather than change every test-only caller of
`mergeFilesByPath`. Normalize equivalent paths at the same owner to prevent
parallel writes from bypassing the conflict check.

Use exclusive creation for standalone `File.Render` create-only writes; use an
atomic no-replace install (hard link of the staged regular file) for transactional
external create-only output. Existing filesystem entries, including dangling
symlinks and directories, count as occupied and are skipped. Errors other than
existence fail. A successful link owns that created entry for rollback; a skip
must never be backed up, removed, or printed as installed. Preserve normal
replaceable-output checks and rollback behavior. Parent-directory symlink
handling and identity checks during rollback must be covered with real files;
this is not a claim of correctness against a hostile process continually
replacing ancestor directories.

The user explicitly accepted the gen-tree boundary: create-only artifacts are
rejected inside the whole-replaced `gen/` tree, and supported outside it. The
single internal owner is `internal/artifactpolicy.Validate`; standalone rendering,
generator preflight and staged validation call that owner. Its diagnostic is
`create-only artifact <path> must be outside gen/`. This rejects the contradictory
ownership declaration before live mutation. Considered alternatives were copying
protected entries into the replacement tree (a larger live-copy/race contract)
and ignoring declared intent (which violates the protection guarantee).

## Planned files and checks

- `codegen/file.go`, new `codegen/file_write_intent_test.go`: absent/existing
  regular file, dangling/live symlink, directory, error, and racing creator.
  `codegen/artifact.go`: narrow documented result type only, as approved.
- `codegen/generator/generate.go`, existing
  `codegen/generator/generate_merge_test.go`, new
  `codegen/generator/artifact_write_intent_test.go`: policy survives rendering,
  cleaned path aliases, equal-policy merge, conflicting policies in both orders,
  cleanup removes the corresponding artifact record, deterministic ordering.
  Update `generate_cleanup_test.go` and other affected result consumers
  mechanically for the typed result.
- `cmd/loom/gen.go`, `cmd/loom/main.go`, `cmd/loom/vet.go`,
  `cmd/loom/generation_transaction.go`: private wire result and live owner.
  Existing `cmd/loom/generate_test.go` supplies fake runner, manifest, module
  setup, normal/failed transaction and CLI stdout helpers. Adapt fake output
  mechanically; add focused tests in
  `cmd/loom/artifact_write_intent_test.go` to keep the existing file below 1000
  lines. Registered plugin fixture runs the real built CLI and helper.
- `cmd/loom/tla/artifact_write_intent/ArtifactWriteIntent.tla`,
  `Current.cfg`, `Corrected.cfg`, `README.md`: durable bounded model,
  counterexample and implementation correspondence. No checker output committed.
- `docs/code-generation.md`: describe plugin create-only semantics, skipped
  stdout, typed programmatic result migration, and the accepted gen-tree policy.
  Root owns shared skill/index and tracker edits; send it the required deltas.

## Model and test-first sequence

1. Create a bounded model of stage, live output, intent loss, user creation
   between staging and installation, two external outputs, failure and rollback.
   `Current.cfg` must fail `PreserveUserContent` through intent loss before the
   corrected transition is implemented. A second unsafe check-then-rename
   control must fail when the user creates a destination after the check.
2. `Corrected.cfg` retains intent and makes create-only installation atomic.
   Check user bytes survive, skips never acquire rollback ownership, and a
   later installation failure restores generated state. Explicitly bound the
   paths/values and assume ordinary successful filesystem operations except
   the injected install failure; do not claim arbitrary crash recovery or a
   proof of Go/system-call refinement.
3. Add direct renderer/transaction/launcher regression tests before source
   changes; run them against the candidate parent and record actual assertion
   failures. Historical baseline remains read-only. For API-independent tests,
   use a temporary overlay/copy if historical confirmation is needed; never
   edit that baseline checkout.
4. Implement only after readiness approval and model/test counterexamples.
   Run direct regressions and affected packages. Preserve existing evidence
   after documentation-only edits.

TLC commands (run from the model directory with task-owned temporary metadir):

```sh
java -cp /Users/luca/.cache/loom-tools/tla2tools-1.7.4.jar tlc2.TLC -config Current.cfg -metadir "$MODEL_TMP/current" ArtifactWriteIntent
java -cp /Users/luca/.cache/loom-tools/tla2tools-1.7.4.jar tlc2.TLC -config Corrected.cfg -metadir "$MODEL_TMP/corrected" ArtifactWriteIntent
```

Expected old result is the named invariant violation, not parse/tool failure.
Record checker state counts and assumptions in the model README after running.

## Bounded acceptance matrix and commands

Accepted: plugin external create-only absent, pre-existing, created after stage;
replaceable plugin output; mixed create-only/replaceable outputs with later
failure; same-path equal policy contributors; normal `gen`, `example`,
`test-scaffold`, and `vet` helper routes. Rejected: conflicting write policies
in either order, malformed helper data, escaping/missing staged paths, invalid
manifest, replaceable symlink/directory destination, and the gen-tree create-only
case under the accepted policy. Preserve all original error/rollback cases.

```sh
go test ./codegen -run 'TestFile.*(WriteIntent|SkipExist)' -count=1
go test ./codegen/generator -run 'Test.*(ArtifactWriteIntent|GenerateMerges|GenerateParallel|GenerateRemoves|GenerateSkips|GenerateRejects)' -count=1
go test ./cmd/loom -run 'Test.*(ArtifactWriteIntent|GenerateTransaction|GenerateStdout|GenerateRealBinary|Vet|GeneratorUsesCallerWorkspace)' -count=1
LOOM_DIR=$PWD go test ./codegen/generator -run '^TestExampleInterceptorsCompile$/service-package$' -count=1
go test ./codegen ./cmd/loom -count=1
```

For generated-byte comparison use existing HTTP ticktock checked-in fixture
(`http/integration_tests/fixtures/ticktock`) plus
`codegen/service/testdata.ServerInterceptorExampleDSL` as bounded meaningful
controls, alongside the registered-plugin fixture. Build baseline and candidate
CLIs once, then invoke identically named binaries with identical arguments in
independent temporary copies/modules. Rewrite each temporary local `replace`
to the revision under test and keep GOWORK off. Do not edit checked-in `gen/`.
Generate at baseline and candidate; record a sorted relative-path/SHA-256
manifest and compare every byte. Generate the candidate twice in separate
processes with the same source/input/toolchain to establish determinism. Compile
(`go mod tidy`, `go build ./...`, `go vet ./...`) one generated output per distinct
input/revision; repeating identical builds is unnecessary. Reuse the existing
interceptor compilation helper and `internal/testingx` process wrappers.

Intended byte differences: none for ordinary fresh generated source or
interceptor example source; plugin create-only destinations retain the exact
user bytes on a repeat run instead of template bytes; newly absent create-only
and replaceable files retain their existing emitted template bytes. Private
helper stdout changes from paths to JSON records; public CLI stdout stays paths
and excludes skipped destinations. Conflicting policy diagnostics and the
approved gen-tree policy are the only new rejections. No service wire-protocol/schema changes
and no exhaustive exported-design sweep are in scope.

## Handoff and review

All candidate files stay uncommitted. Record final file hashes and exact gates,
source/input identities, old failures, intended/observed output differences,
model findings and unresolved limitations. Root copies the frozen manifest to
main and requests fresh exact integrated-diff review before its atomic #670
commit. No changes after that review without re-review. After validation and
review, remove only this task's confirmed-unused model states, probe modules,
binaries and comparison snapshots; preserve model sources and concise evidence.


## Evidence collected during implementation

Before implementation, the direct transaction and registered-plugin CLI tests
both read `template` instead of expected `user`. Renderer tests failed for a
creator during rendering and a dangling symlink. Conflicting same-path policies
were accepted in both orders. TLC results are recorded in the model README.

Passed outside-gen gates with Go 1.27.2:

- `go test ./codegen ./cmd/loom -count=1` (43.341s / 94.221s).
- Focused artifact, merge, parallel-render and output-removal tests in generator,
  renderer, and launcher packages.
- Existing interceptor compile/build/vet control selected with
  `LOOM_DIR=$PWD go test ./codegen/generator -run
  '^TestExampleInterceptorsCompile$/service-package$|TestArtifactWriteIntent'
  -count=1`.
- Existing real launcher, transaction, stdout, vet, and workspace controls.
- Parent `eb5a169b` versus candidate: zero differing bytes across all 27 files in
  an HTTP ticktock fixture copy and all 24 generated/example files from
  `ServerInterceptorExampleDSL`. A second independent candidate process for each
  input also produced zero differences. Both parent and candidate modules passed
  tidy, build and vet. The parent source was a temporary `git archive` of the exact
  candidate parent; historical `4535a403` remained untouched. Both CLI basenames
  were `loom`, commands used `-o .`, GOWORK was off, and each temporary module
  replaced Loom with its matching source. No checked-in fixture changed.

Task-owned scratch retained until review: `/tmp/loom-artifact-model.wgGIiT`
and `/tmp/loom-artifact-compare.BowYbJ`. These contain checker logs, source archive,
binaries, exact-byte comparison manifests and generated temporary modules; none
are commit inputs. Generated-output comparison code is task validation only,
not a new repository harness.


Final policy validation also passed the same focused generator/launcher/vet and
workspace set after adding shared gen-tree admission (codegen 0.602s, generator
3.592s, launcher 29.547s), `go vet ./codegen ./codegen/generator ./cmd/loom`, and
`make lint-json-v2 lint-namescope lint-gofmt`. The final admitted-path code was
regenerated twice for both bounded specimens against the retained exact parent
outputs: again zero differences across 27 and 24 files, with candidate tidy,
build and vet passing. Baseline generation/build evidence was reused.

The accepted gen-tree rule has its own `GenPolicy` model and expected-failure
control, documented beside the external-installation model. Direct tests cover
`gen`, `gen/custom.txt`, and a cleaned alias; `genesis/` and a nested `gen/`
within an external directory remain accepted. A real registered-plugin failure
preserves the previous generation manifest. No #741 or #751 source was changed.

Final real-plugin CLI test passed after adding the scaffold route and rejected
gen-tree plugin: `go test ./cmd/loom -run
'^TestArtifactWriteIntentRealPlugin$' -count=1` (10.130s). The scaffold's generated
tests compile with `go test -run '^$' ./...`. Final renderer controls also pass
for `genesis/` and an external directory containing a nested `gen/`.


## Integrated-review repair

The first frozen candidate was superseded after an independently confirmed P1:
a stable live directory alias into `gen/` passed lexical admission and let the
whole-tree replacement erase the protected file before installation. The exact
reviewer overlay and new alias regressions reproduced the failure before repair.
The updated GenPolicy model first reproduced the lexical-only counterexample,
then checked actual ownership classification. Model findings and correspondence
are recorded in its README. Root accepted case-insensitive reservation of the
top-level `gen` component as the portable ownership boundary.

The repair changes only shared path admission and its call sites: stable live
parent aliases and absent suffixes are resolved before mutation; canonical
containment and filesystem identity identify the generated owner. The entire
transaction batch is admitted before any output directory creation. Leaf
symlinks remain occupied external entries. No claim is made against hostile
ancestor swaps between admission and installation. Existing rendering bytes,
helper protocol, no-replace installation and rollback are unchanged, so the
prior generated-byte/determinism evidence remains valid for accepted controls.

Repair validation, Go 1.27.2 on the same filesystem and source worktree:

- The exact review overlay reproduced the overwrite before repair and passed
  afterward: `go test -overlay
  /tmp/loom-artifact-compare.BowYbJ/alias-overlay.json ./cmd/loom -run
  '^TestReviewCreateOnlyStaticGenAlias$' -count=1` (0.531s).
- `go test ./internal/artifactpolicy ./codegen ./codegen/generator ./cmd/loom
  -run 'TestValidate|TestFileSkipExist|TestArtifactWriteIntent' -count=1` passed
  (0.183s / 0.628s / 0.704s / 13.164s), including the registered-plugin route,
  generated-app compilation, test scaffolds, and rejected policy behavior.
- Additional red-first cases caught symlink targets containing `link/..` and a
  case alias of the output root leading to absent `GEN`. Both now reject. A
  verbose policy run confirmed that both filesystem-identity tests executed
  and passed; neither was skipped on this filesystem.
- `LOOM_DIR=$PWD go test ./codegen/generator -run
  '^TestExampleInterceptorsCompile$/service-package$|^TestGenerate' -count=1`
  passed (5.454s); the generated examples compile and the normal controls
  exercise accepted artifacts. `go test ./cmd/loom -run
  '^TestGenerateTransaction|^TestArtifactWriteIntent' -count=1` passed (11.367s).
- `go vet ./internal/artifactpolicy ./codegen ./codegen/generator ./cmd/loom`,
  `make lint-json-v2 lint-namescope lint-gofmt`, and `git diff --check` passed.
  The refined admission model has two expected-failure controls and a passing
  corrected configuration; exact counts and commands are in its README.

Only the admission model and affected implementation checks were invalidated;
the retained 51-file comparison and independent-process determinism results
remain valid because rendering and admitted output bytes did not change. No
new external protocol, installer, or rollback behavior was introduced by the
review repair. No #741 or #751 implementation is included.
