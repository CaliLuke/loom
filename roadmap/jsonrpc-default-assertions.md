# JSON-RPC effective-default assertion ownership — #845

Parent: `03e38fd16e0eace6b0911f28981b808ab06e9608`. Repair the test expectations
for the accepted absence-only default contract. Production generation and the
existing effective-constraints design stay unchanged. The server payload
constructor owns admission defaults; outgoing native values remain authored.

## Status

- Plan created; CI's obsolete client-body assignment expectation is reproduced.
- A diagnostic overlay already passes mounted absent/explicit/invalid cases plus
  existing numeric/required runtime checks; permanent acceptance remains below.

## Milestones

### Milestone 1: Restore contract-owned assertions and CI coverage

Goal

Make `TestEffectiveConstraintsGeneratedJSONRPCModule` assert the actual server
admission owner and prove defaults without hiding failures behind stale strings.

Acceptance Criteria

- The fixture DSL is preserved; parent/candidate/repeat generated files are
  byte-identical and parent/candidate modules build and vet.
- Mounted JSON-RPC requests prove absent inherited/local defaults, explicit
  nonzero values, explicit empty validation, nullable absence/value/null, and
  inherited required and numeric validation already covered by the harness.
- Named constructor assertions distinguish client encoding from server admission;
  no production code changes. Focused tests, lint and exact independent review pass.

Checklist

- [x] Read this plan and loom-framework/execution-plans skills; recite milestone,
  test-first, validation, independent review and root commit/push handoff before edits.
- [x] Record the parent failure with `GOEXPERIMENT=jsonv2 LOOM_DIR=$PWD go test
  ./jsonrpc/codegen -run '^TestEffectiveConstraintsGeneratedJSONRPCModule$' -count=1`.
- [x] Preserve the existing DSL in a shared test helper in
  `jsonrpc/codegen/effective_constraints_compile_test.go`; compare the extracted
  body with its parent text and capture parent generated output through a temp overlay.
- [x] Replace obsolete all-file client-default checks with named constructor
  assertions; add mounted absence/explicit/nullable/invalid admission cases.
- [x] Add process-isolated artifact capture using the same DSL and existing
  `renderJSONRPCModule`/`runGoJSONRPCTestCommand` helpers, with a pushed module pin.
- [x] Run the direct generated test, parent/candidate/repeat artifact comparison,
  module build/vet and scoped JSON-RPC codegen lint; save concise receipts under /tmp.
- [ ] Freeze exact source/evidence hashes and obtain a fresh independent review.
- [ ] Hand off the reviewed atomic diff for root commit/push and ticket closure;
  remove inactive temporary generation artifacts after preserving a compact receipt.


## Acceptance receipt

The original named test fails at the obsolete `body.Inherited = "base"`
assertion (`/tmp/loom-jsonrpc-default-assertions-parent-red.log`). No production
file changes. Extracted DSL text is identical after removing only its outer
indentation; the original snapshot is `/tmp/loom-jsonrpc-default-original-dsl.txt`.

The corrected direct test passes in 2.016s and its outer generator race run in
4.143s (`/tmp/loom-845-contract.log`, `/tmp/loom-845-race.log`). Generated module
build/vet and 32 runtime cases pass: 18 numeric and four inherited-required cases
are preserved, eight admission cases cover defaults/value/null/invalid inputs,
and two client-construction cases prove native values are not replaced by defaults.
These child-module runtime tests use the normal test tier; the outer race flag
covers the generator process. No runtime concurrency behavior changes here.

Ten generated artifacts are byte-identical across parent, candidate and a second
process-isolated candidate run (`/tmp/loom-845-artifacts.json`). Both parent and
candidate modules build and vet; the repeated bytes need no duplicate compilation.
The parent capture uses the original test and DSL with only a temporary output
path/early-return overlay. All compared runs use unchanged production source from
the parent. Test manifests pin pushed `063198a2f427` with explicit local source
replacement; these manifests and tidy-produced sums are not generated artifacts.
No generated difference is intended. Scoped JSON-RPC codegen lint reports zero
issues (`/tmp/loom-845-lint.log`); `git diff --check` passes.

Issue: https://github.com/CaliLuke/loom/issues/845. Independent exact final review,
root integration and atomic delivery remain pending. No new formal model is needed:
this corrects test ownership for already accepted/default-tested behavior.
