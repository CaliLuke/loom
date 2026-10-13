# Generation staging and publication filesystems (#751)

Status: implementation on `688d94870`; readiness independently approved.

## Owner and contract

The CLI transaction owns render staging, destination publication and rollback.
Today it renders beside the module root to preserve relative `go.mod` replacements,
then assumes that stage shares every destination's filesystem. A module mounted
beneath a different filesystem breaks the gen-tree rename; nested plugin output
mounts also break create-only links and replaceable-file renames. Requiring the
module's parent to be writable is independently wrong.

Preserve the existing whole-gen replacement contract and external create-only
ownership. Do not add individual EXDEV fallbacks or change plugin policy. Prepare
all bytes before moving live data, and keep temporary candidates private until
publication. A mount directly on `gen/` cannot support whole-directory replacement;
report that limitation while preserving it, rather than silently switch to an
in-place update with weaker failure semantics.

## Proposed shared placement boundary

Render inside a hidden directory under the writable output root. Keep the staged
module-relative output suffix, so computed generated import paths retain the
original module prefix. Rewrite only local relative replacements in the detached
staged `go.mod` to their original absolute targets; never edit the caller's module
or workspace. Inspect actual helper/workspace behavior rather than assuming the
stage must participate in `go.work`.

Before changing live paths, prepare each publication unit beside its destination:
the complete gen tree beside live gen, and each external file in its destination
parent. This accommodates nested external mounts and keeps Link/Rename local.
Use one preparation/cleanup owner, preserving mode and fully written contents.
Reuse existing no-replace Link, replace Rename and rollback file identity rules;
create-only racing creators must still win. Admission of gen aliases runs before
creating external directories or preparing candidates. Cleanup removes owned temporary publication candidates on preparation,
validation, commit or rollback failures. Failed restoration has a separate backup
lifetime defect; this repair does not claim to preserve recovery backups when
rollback itself fails. The model assumes successful rollback filesystem calls.

External files are prepared unconditionally beside their destinations; gen is
already rendered on its destination filesystem. There is no per-consumer copy
fallback after a partially completed live publication.

## Sequence and bounded acceptance

1. [x] Reproduce module-parent writability and original cross-device commit;
   also cover nested output mount and external create-only/replaceable mounts.
2. [x] Establish ordinary and nested generated package identities and relative
   replacement resolution, including unchanged source go.mod and go.work.
3. [x] Extend the existing artifact-write-intent model with placement/preparation
   obligations: current cross-device failure, prepared candidates, preserved
   racing creator, preparation failure and existing rollback invariants. Keep
   mount stability and filesystem atomic-operation assumptions explicit.
4. [x] Freeze readiness tests/model and obtain independent approval.
5. [x] Implement the shared placement/publication owner, then run the direct
   transaction, policy/alias, failure-cleanup and bounded actual CLI controls.
   Compare affected output bytes; no generated semantics or broad corpus change.
6. [ ] Freeze exact final diff and evidence for independent review and atomic
   root delivery.

Consumer seams: `newGenerationTransaction`, `prepareModuleContext`, transaction
validation/commit/cleanup, external publication and rollback; helper module
context in `cmd/loom/gen.go`, generated package computation in
`codegen/generator/generate.go`, and `internal/artifactpolicy.Validate`. Existing
write-intent, alias, rollback, workspace and real CLI tests are retained controls.
The existing model covers content ownership but explicitly excludes arbitrary
mount layouts, so bounded placement modeling extends that boundary rather than
claiming the original model already validates cross-filesystem publication.

Cross-device tests can run a Linux test binary cross-compiled with the repository's
pinned host Go 1.27.2 in a disposable container with separate tmpfs mounts. The
container's installed Go is not used. No dependency/toolchain downgrade is needed.

## Parent evidence

The unwritable-parent test fails at stage creation with permission denied while
the module remains writable. Both ordinary and nested staged-module cases fail
placement assertions; unchanged relative-replace targets and module-relative
suffix checks retain the existing semantics as controls.

A Linux arm64 test binary built with host Go 1.27.2 reproduces four actual
cross-device failures in a disposable container: module-root gen rename, a nested
output alias into a mounted directory, external create-only Link, and external
replace Rename. Docker's existing image supplies only the runtime filesystem;
its Go compiler is not invoked. The nested output test intentionally uses a
stable directory alias into the mount, also exercising existing alias policy.
The failed container is removed after capturing its output and exit code.

The placement model rejects original staging (3/3 states), unwritable module
parent (2/2) and output-only staging (5/5). Destination-local preparation checks
placement, rollback and terminal cleanup over 9/8 states. Existing create-only
ownership modeling remains separate and applicable.

Use unconditional destination-local preparation for external files, avoiding a
platform-specific device probe and copy-on-EXDEV fallback after publication.
Rendering already inside output makes the gen tree local; it can be renamed
directly without a redundant second full-tree copy. Both are prepared publication
units with cleanup ownership before live mutation. This is the smallest shared
owner change that satisfies all four mount cases and parent writability.

Readiness controls: existing transaction success/rollback/validation tests,
create-only preservation/rollback cases, actual CLI nested module-qualified
output compilation, and workspace replace/use generator cases pass in 8.195 s.
Configured CLI-package lint reports zero issues. Final acceptance adds candidate
passes for the four mounted cases and unwritable parent, preservation on an
unrenamable mounted gen tree, external preparation failures before live mutation,
mode/content fidelity and cleanup of every destination-local temporary file.
Existing concurrent creator, replacement-owner rollback and alias rejection
controls must remain unchanged. Compare one meaningful generated fixture through
ordinary/nested CLI output with parent/candidate/repeat bytes and compilation;
there is no transport code generation semantic change requiring a wider matrix.

## Implementation and acceptance

`newGenerationTransaction` now creates its private render root inside output.
`prepareModuleContext` retains the module-relative output suffix and makes only
local relative replace targets absolute in the detached copy. Versioned and
already-absolute replacements remain unchanged. `prepareExternalOutputs` validates
all policies first, skips existing create-only destinations, and delegates complete
private copies to `preparePublicationFile`. Candidates are registered for cleanup
at creation, closed before publication, and retain file permissions. Existing
Link/Rename and rollback ownership consume those prepared candidates unchanged.

The final focused transaction/policy/alias/publication race selection passes in
13.138 s; existing nested CLI compilation and workspace helper controls pass in
6.648 s. Configured CLI lint reports zero issues. All four actual cross-device
cases now pass using a host-Go-1.27.2-built Linux binary. A fifth case mounts gen
itself and checks refusal with the original inode/content preserved. Preparation
failure leaves the old generation intact; candidate content/mode and recursive
temporary cleanup are checked separately. An invalid output that is a regular
file now fails in Prepare before invoking the helper, rather than in Commit.

The bounded CLI comparison uses the checked-in JSON-RPC ticktock fixture with
ordinary and nested output, a relative Loom replacement and an active caller
workspace. Parent, candidate and separate-process repeat produce identical bytes
for each output shape (19 files per run). Ordinary output also matches all 19
checked-in generated files. Parent/candidate ordinary and nested packages build
and vet successfully. Caller go.mod and go.work bytes remain unchanged. Nested
output intentionally differs from the root fixture in command/import/package
context; it is compared against the same nested parent invocation, not treated
as an accidental output change. No output difference is intended by this fix.

The existing failed-restoration backup lifetime defect is preserved as a separate
reproduction: after rollback reports an error, cleanup may delete the only
remaining original backup. Candidate cleanup does not repair or strengthen that
backup lifecycle. The placement model assumes rollback/cleanup succeed; it does
not establish recovery guarantees for that separate defect.

Integration caught one old success-test assertion still expecting staging beside
the output directory. Its expected parent now matches output-local placement;
helper removal and debug/write/compile/run assertions are unchanged. The exact
success/debug/compile-failure/stage-failure race controls pass in 1.775 s. This
one-line test correction does not invalidate mounted or generated-byte evidence.
