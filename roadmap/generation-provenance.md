# Complete evaluated-design provenance

## Status

Readiness independently approved for #753 implementation and #742 reconciliation. Current implementation parent `2a98e4123`; original readiness parent `be89c93b5` preserved in its manifest. The accepted architectural direction is one evaluated-design identity consumed by generation, manifest emission and vet. This document settles the bounded representation proposal before an execution checklist.

## Contract and owner

`docs/code-generation.md` promises that `loom vet PACKAGE` detects generated output that differs from its evaluated design. The framework skill requires deterministic maps, output-affecting generation context, and exclusion of runtime-only evaluator state and `loom:vet:*` metadata. `TestGenerateManifestFingerprintsDesignBeforeGeneratorsMutateIt` establishes the existing boundary: after DSL evaluation, before generator preparation or emission mutates roots. Preserve that boundary. Fingerprinting is not a hash of arbitrary generator code, external files or nondeterministic plugin behavior.

Current `Generate` has the complete dependency-ordered `eval.Context.Roots()` result but passes only `expr.Root` to `internal/designfingerprint.Digest`. `codegen.ManifestFile` records no originating package. Vet walks every module manifest and compares it to its one supplied root. A plugin-only change goes undetected, while another design's current tree is falsely stale.

Proposed owner: an explicit provenance input at `internal/designfingerprint`, containing originating design package, complete ordered evaluated roots, generation command, generated package and design version. The existing deterministic graph encoder remains the only serializer. Generator and vet must pass the same input; manifest origin is recorded by generation and used for attribution before digest comparison. CLI `Generator.DesignPath` is the authoritative supplied import path; do not infer origin from output directory, API name or source scanning. Public entry points must accept explicit origin/root context rather than make vet digest a different ambient global graph. Proposed signatures are `generator.Generate(dir, command, designPackage string, debug bool)` and `vet.Analyze(roots []eval.Root, moduleDir, designPackage string)`. Vet derives its core `*expr.RootExpr` from that same root list for existing semantic analysis. The temporary CLI program obtains Roots once for vet after evaluation and passes its design import path to both branches. Reject absent/invalid origin when creating or checking provenance; tests and direct callers supply their actual specimen package, without an origin-free compatibility wrapper.

## Behavior

- Capture the full root list before preparation plugins or generators. Include each unique `EvalName` plus the existing encoded root value. `eval.Register` already rejects duplicate names and Roots rejects dependency cycles. Preserve evaluated order because generators receive that order; do not sort away observable ordering. Repeated identical ordered input is deterministic; a changed order is an input change.
- Existing graph semantics remain: cycles and shared nodes are supported; private fields, funcs/channels/unsafe runtime state, ExampleGenerator and vet metadata retain exclusions. Plugins must expose evaluated semantic state in the same encodable graph; this change does not inspect closures or private implementation internals.
- Record `design_package` in `loom.json`, and include it in the fingerprint. Generation and vet use the supplied canonical import path. Relative package spellings need canonicalization through the existing package loader if accepted by the CLI; they must not become distinct origins.
- Vet compares a digest only when the manifest's design origin matches the selected design. Version-skew inspection remains module-wide. Missing provenance is an explicit regeneration diagnostic, consistent with current missing-digest behavior; never silently attribute an old manifest to the selected design. A mismatched known origin is skipped for design-skew only.
- Changing the fingerprint format requires regeneration. Do not retain a second legacy digest implementation or infer matches by trying multiple roots. #751 staging/filesystem ownership is out of scope.

## Consumer inventory and deletion boundary

`cmd/loom/gen.go` owns both temporary generation and vet programs and knows DesignPath. `codegen/generator/generate.go` captures roots before preparation; `codegen/header.go` emits the manifest. `internal/designfingerprint` owns encoding and the shared manifest data record, also consumed by `cmd/loom/generation_transaction.go` validation. Delete its duplicate manifest struct as well as the vet/emitter copies; require attribution on newly staged generation rather than permit creation of an unverifiable tree. `vet/analyze.go` and `vet/source.go` carry explicit design provenance through module inspection. Direct generator/vet callers and manifest tests must adopt that same input; generated service code and wire behavior should not change. Remove expr-only fingerprint calls and all-manifest same-design inference; no transport-specific correction.

## Proof obligations

Direct regressions must demonstrate changed plugin output with unchanged parent digest and incorrect A/B attribution. Existing deterministic map, cyclic/shared graph and pre-mutation tests remain controls. New cases cover ordered plugin roots, plugin cyclic state, origin changes, absent provenance, API suppression and unrelated-manifest version skew. This is a deterministic input/identity correction; no new concurrent protocol or independent formal state model is proposed.

Bounded actual CLI acceptance uses two design packages with separate generated trees in one consuming module, plus a registered plugin root whose exported value affects a generated artifact. Both current trees must pass design-skew analysis; mutate only plugin A, observe only A stale, regenerate A and clear it. Compile both trees. Repeat generation in an independent process and compare all bytes, listing intended manifest changes and plugin artifact changes only. Keep the existing meaningful generation fixture as the normal control; no exported-design sweep.

After readiness review, implement #753 at the shared owner. If complete provenance necessarily fixes #742, use its separate original witness for reconciliation rather than introducing an artificial second implementation. Root retains atomic commit and fresh review authority.

## Parent evidence

`codegen/generator/provenance_readiness_test.go` changes registered plugin state and actual emitted plugin.txt; both parent digests remain equal. `vet/provenance_readiness_test.go` keeps independently current A/B manifests with explicit origin fields; parent ignores attribution and reports B stale. Exact output is `/tmp/loom-provenance-parent-red.log`. Existing graph/map/cycle, pre-generator mutation, missing-manifest and version-skew controls pass under race in `/tmp/loom-provenance-parent-controls.log` (fingerprint1.607s, generator1.667s, vet1.389s).

The external actual-CLI specimen is `/tmp/loom-provenance-cli-parent`, with command/input runner `/tmp/loom-provenance-cli-probe.py` and results `/tmp/loom-provenance-cli-parent-results.json`. It uses explicit local source on this unchanged production parent and two design packages generating A/B trees in one module. It is bounded acceptance input for the implementation, not a new reusable harness.

Actual CLI evidence: before mutation, vet A reports only B and vet B reports only A. After changing only plugin A, vet A still reports only B and misses its own changed output. Both generated trees pass `go build ./...` and `go vet ./...`; regeneration changes plugin.txt but leaves the parent manifest byte-identical. No GOFLAGS overlay or alternative generator source was used.

## Accepted execution

### Milestone 1: Shared provenance owner

Goal: Generator, staged installation and vet consume one explicit input and manifest record.

Acceptance Criteria

- Original plugin and A/B regressions pass without output-directory inference.
- Invalid origins and missing/nil/duplicate roots produce diagnostics, never panics.
- Existing graph, API suppression, version and pre-mutation controls retain their contracts.

Checklist

- [x] Recited test-first shared-owner migration, focused controls, actual CLI comparison, independent final review and root commit handoff; inherited AGENTS forbids manual generated edits and destructive git operations.
- [x] Preserve parent failures and record all current consumers in this document.
- [x] Replace expr-only digest and duplicate manifest records in generator, CLI staging and vet.
- [x] Migrate direct generator/vet call sites to explicit origin/root inputs.
- [x] Run owning fingerprint/vet packages and selected generator/CLI provenance controls under race; record exact source and commands.

### Milestone 2: Bounded adoption and final review

Goal: Actual CLI output confirms deterministic provenance and no unrelated generated changes.

Acceptance Criteria

- A/B origins match independently; only plugin A becomes stale after its root changes; regeneration clears it.
- Relative and canonical import-path invocation produce the same origin and bytes.
- Three checked-in fixture regenerations differ only in loom.json; parent/candidate build and vet, independent repeats match exactly.

Checklist

- [x] Compare two CLI designs and plugin mutation against physical archived parent; compile both trees and compare independent generation bytes.
- [x] Regenerate HTTP ticktock, JSON-RPC ticktock and mixedtick with the actual CLI; preserve all other artifact bytes.
- [x] Update public generation documentation and framework guidance; record required regeneration and programmatic API adoption.
- [ ] Freeze exact final source/evidence for an independent review; root owns atomic #753 delivery and separate #742 witness reconciliation.

## Final bounded evidence

- Shared owner implemented against `2a98e4123`. Sources retain the approved pre-Prepare boundary, explicit ordered roots and canonical package. Missing origins are rejected before output allocation for both gen and example. Staged manifests, emitter and vet use one record; no alternate old digest remains.
- `/tmp/loom-provenance-owning-packages.log`: full fingerprint and vet packages pass race1.336s/8.507s. `/tmp/loom-provenance-owner-consumers.log`: selected CLI transaction/origin/version and generator regressions pass2.062s/1.512s. Final admission check `/tmp/loom-provenance-final-direct.log` passes generator1.794s/CLI1.738s. `/tmp/loom-provenance-lint-exact.log`: zero issues across changed owning packages.
- Actual CLI physical parent is an exact `git archive` of `2a98e4123`, never a GOFLAGS overlay. `/tmp/loom-provenance-cli-comparison.json` records34 artifacts across A/B trees: only the two manifests change on adoption; candidate repeated canonical generation is byte-identical. Plugin mutation changes only A's plugin.txt and manifest, vet reports only A, and regeneration clears it. Parent instead reports the other design and misses its own plugin change. Both modules build/vet.
- Relative/import-path invocations produce identical manifest bytes. Generated command comments intentionally preserve invocation spelling; an initial comparison incorrectly expected those comments to match too. The preserved first-attempt logs record that harness assertion. Corrected acceptance compares manifest identity across spellings and all bytes across repeated identical canonical invocations; no production normalization workaround was added.
- `/tmp/loom-provenance-fixture-ledger.json`:55 artifacts per parent/candidate/independent-repeat stage across HTTP ticktock, JSON-RPC ticktock and mixedtick. Exactly three manifests change; every other artifact equals the checked-in parent, and repeats are exact. Parent/candidate build/vet pass. Checked-in manifests are copied from the actual CLI output after hash comparison; never hand-edited.
- A final pre-allocation validation addition does not change accepted encoding or manifest input. `/tmp/loom-provenance-final-cli-admission.json` recompiles final CLI source, regenerates A and vets it; all34 artifact bytes remain exact. Earlier three-fixture comparisons remain valid for these accepted inputs.
- Original #742 A/B witness now passes using attributable manifests through the same #753 owner. Root may reconcile that report separately; no second production change is needed.
- Scope excludes plugin implementation/source hashing, private/runtime-only state and external file contents; those existing encoder limits are now explicit. Final exact independent review and root delivery remain pending.
