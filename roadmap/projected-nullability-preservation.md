# Projected result nullability (#837)

Worktree: `projected-nullability-preservation/loom`. Parent:
`9669d281584d0086a56394236238e168521f722e` (pushed). Status: implementation complete, final review pending. Dependent #834 remains frozen in its own worktree.

## Contract and owner

Selecting a result view changes visible fields and requiredness, not authored
intrinsic nullability. `expr.valueViewPlan.project` constructs object declarations
without Nullable; collection projection consequently loses intrinsic element
nullability too. Occurrence copying already retains local Nullable. Preserve the
source declaration's Nullable in the shared projected declaration constructor;
keep occurrence attributes and the existing source/selection/view cache identity.
Do not promote a nullable occurrence into its shared declaration or copy all
source validation into a selected view.

## Sequence

1. Preserve the actual pushed-parent DSL/generation failure and focused issue.
2. Add direct root/nested/collection semantic null controls, source immutability,
   both occurrence orders, same-view reuse and distinct-view identity tests.
3. Model current declaration loss, occurrence promotion and view conflation;
   check the proposed declaration copy against bounded identity/admission rules.
4. Freeze exact readiness evidence for independent review before production.
5. After approval, repair the shared constructor and rerun affected expr checks.
6. Compare representative ordinary/occurrence generated controls at parent,
   candidate and isolated repeat; compile/vet those representable controls.
   For intrinsic transport witness, assert removal of semantic transform failure
   and preserve #834's remaining generated physical-carrier compiler failure.
7. Update guidance/model correspondence, freeze exact final diff for review.

## Acceptance boundary

Root null, nested null and nullable collection elements retain semantic admission
under default/tiny selection. Concrete values project only selected fields;
ordinary non-null declarations continue rejecting null. Same selected declaration
is reused, different views remain distinct, and no source attributes mutate.
Generated physical nullable selected-view handling remains #834: #837 does not
claim that intrinsic generated Go compiles before that independent repair.

## Parent evidence

Issue: https://github.com/CaliLuke/loom/issues/837. The self-contained module at
`/tmp/loom-project-nullability-repro` requires pushed
`v1.10.0-alpha.9.0.20261013043423-a8783ef651e7` without replacement. Nullable inside
`ResultType`'s `Attributes` is valid DSL. Generation fails with `cannot transform
nullable field child to a non-nullable representation`; direct expr.Project
also loses IsNullable. The approved issue body and generation log are retained
outside the checkout until this prerequisite and #834 acceptance finish.

## Frozen readiness evidence

Focused direct matrix: 13 intended red cases and 21 passing controls on the exact
parent. Semantic null loss fails as `object target requires an object`;
concrete selected values and ordinary null rejection pass. Collection-root
Nullable already survives copying and remains a passing control. The public DSL
root default/tiny tests fail at effective projected nullability. Existing
projection/recursive controls pass unchanged. Model checked mode passes 2048
states; four rejected alternatives produce the documented counterexamples.
At readiness, no production files had changed. Toolchain Go 1.27.2, Corretto 25.0.4.1, TLC 1.7.4.

## Implementation and bounded acceptance

The shared object-result constructor now copies `rt.Nullable`; no downstream
transport or validation workaround was added. Declaration identity, occurrence
copying and selected-view validation remain with their existing owners. Root
approved readiness before the production edit. The public guide distinguishes
semantic preservation from #834's separate generated-carrier compilation defect.

Full expr suite passes (1.627s), including all 34 direct matrix leaves. Existing
`TestViewedNilResultBoundary` object/collection designs and
`TestNullableObjectUnionTransformCarrier` retain 39 byte-identical generated
artifacts across parent/current/process-isolated repeat. Each initial generation
was vetted and its runtime harness passed; the comparison helper retains a
pushed a8783ef65 module requirement with an exact local source replacement.

The approved intrinsic DSL witness now generates successfully, instead of
panicking because a nullable child becomes non-nullable. Its 17 generated files
are byte-identical across two processes. Vet still rejects `result.Child != nil`
on `loom.Nullable[ChildView]`, the exact independent #834 physical-carrier defect.
No claim of intrinsic selected-view compilation is made here. That complete
matrix remains frozen for #834 to resume after integration of this prerequisite.

Evidence: `/tmp/loom-837-comparison/summary.json` records all output hashes;
parent/current/repeat logs record generation/vet/runtime outcomes. The actual
remote parent witness and readiness model logs remain retained until #834 no
longer depends on them. No checked-in generated fixtures require changes.

Final affected checks: service view/projection/nullable cases pass (28.545s);
HTTP cases pass (148.329s). `make lint-gofmt lint-namescope lint-docs` and
`git diff --check` pass. Comparison runs: parent 5.242s, candidate 6.567s,
isolated repeat 5.609s. No source changed after these checks except documentation.
Final manifest and patch are retained under `/tmp/loom-837-final-*` for a fresh
independent exact-diff review before root integration/commit. #834 stays frozen.
