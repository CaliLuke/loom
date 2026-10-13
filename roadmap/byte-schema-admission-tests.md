# Byte-schema contract tests after constraint admission

Status: test-only implementation and bounded acceptance complete; exact final independent review pending for [#833](https://github.com/CaliLuke/loom/issues/833). Worktree base: `bf7ae2246`.

## Contract and cause

The accepted #695 contract rejects negative authored lengths and contradictory inherited intervals before schema projection or value resolution. CI run `38344472643`, OpenAPI job `115122547452`, still expects `TestByteSchemaIndependentConstraints` to emit schemas for three invalid graphs. The first fatal assertion hides the next two failures. This is stale test admission, not a production defect or a reason to relax validation.

The mathematical byte-bound helper still supports empty and negative intervals as inputs. Its Lean correspondence is a separate boundary from admission of evaluated Loom designs. Inspect the adjacent alias correspondence test for the same stale expectation, retaining arithmetic evidence even when a design must be rejected.

## Sequence

- [x] Reproduce the exact enabled CI test and independently expose negative-lower and negative-upper failures without production edits.
- [x] Obtain root review of the issue draft and proposed test-only repair.
- [x] Make invalid inline-schema cases assert the precise design error and no emitted schema; keep legal redundant-overflow and nullable cases in the real Ajv instance matrix.
- [x] Reconcile the proven adjacent alias conformance expectation at the same admission boundary, preserving raw helper/Lean comparison and rejecting invalid source/target occurrence construction explicitly.
- [x] Update the owning byte-schema evidence record to distinguish design rejection from arithmetic projection.
- [x] Run the enabled focused Ajv gate, relevant ordinary tests and scoped lint; the two real Lean byte-reference groups pass. No generator source/output change or full corpus is required.
- [x] Freeze exact diff, evidence and manifest for an independent final review. Root filed #833 and owns integration and commit.

## Evidence and limits

- `/tmp/loom-byte-schema-admission-parent.log`: exact enabled test fails on contradictory inherited lengths.
- `/tmp/loom-byte-schema-admission/negative_lower.log`: isolated unchanged case fails on negative minimum.
- `/tmp/loom-byte-schema-admission/negative_upper.log`: isolated unchanged case fails on negative maximum.
- Existing `expr/TestInlineJSONSchemaByteEffectiveBoundsBeforeProjection` already records the accepted rejection behavior.

No new model is needed: this changes test expectations to match the delivered `expr/tla/constraint_domain` model and its admission contract. It does not alter interval arithmetic, codec semantics, schema construction or generated output. The original base64 malformed/pad-bit/newline/enum controls remain mandatory.

## Verified adjacent boundary

The real pinned Lean reference confirms four alias cases fail at source admission:
negative lower, negative upper, empty, and empty huge. An isolated continuation
confirms the same four target-plan failures. The repair preserves all nine raw
helper/reference comparisons and all five admitted-design resolution/projection
matrices. `/tmp/loom-byte-schema-admission/alias-parent.log` and
`target-parent.log` retain the failures. The reference build uses unchanged
`leanprover/lean4:v4.34.1`; no model or theorem source changes.

## Acceptance results

- Enabled Ajv gate: pass, 1.592s; two legal length schemas × 13 instances plus the three enum controls, all three invalid graphs assert rejection/no schema.
- Real Lean byte-alias and byte-schema groups: pass, 1.214s; all nine alias arithmetic cases, four source/target rejections and five valid design matrices, plus unchanged bound-projection reference controls.
- No production source or generated output changes. The existing test inputs and compiled test package supply the relevant validation boundary; no generated corpus is warranted.

- Full affected Go packages: `internal/valuecontract` 3.085s, `internal/byteschema` 0.523s. Scoped golangci-lint: 0 issues. Evidence: `/tmp/loom-byte-schema-admission/{packages-green,lint}.log`.
- Documentation gate and `git diff --check`: pass. Final review manifest: `/tmp/loom-833-final-manifest.json`; complete patch: `/tmp/loom-833-final.patch`.
