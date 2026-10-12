# Projected union validation readiness (#796)

Status: implementation complete; frozen for independent final review.
Root inspected the owner, callers, fixture tests and model and repeated the four
expected direct failures plus optional control before releasing implementation.
Independent evidence: `/tmp/loom-projected-union-796-root-readiness.log`.
Parent: `6194743c9485dd242762d230fabc05519ecc1765` (pushed).

## Readiness sequence (completed)

1. Preserve the delivered required-field reproduction and inspect ordinary and
   projected union validation alongside projected type declarations.
2. Add direct representation-context failures and bounded generated controls for
   required/optional primitives, constraints, nested objects and collections.
3. Classify #709 recursive optional validation as shared or independent ownership;
   do not bundle it based only on both reports touching validation.
4. Explore context choices with the existing projected-presence model where
   applicable, documenting assumptions and implementation correspondence.
5. Propose a single physical-context owner and bounded implementation acceptance;
   freeze readiness for independent assessment before production.

## Initial ownership evidence

`buildUnionTypeData` and `buildViewUnionTypeData` both call
`unionBranchValidationCode`, which always constructs `typeContext(scope)` and
passes `view=false`. Projected object fields instead use pointer storage even
when required. Existing `buildValidations` already uses
`projectedTypeContext("", !expr.IsPrimitive(projected.Type), scope)` and
`view=true`. Branch validation must use the same physical declaration contract.
The committed reproduction remains at
`http/codegen/tla/union_declarations/required_untagged_probe.go.txt`.

#709 concerns a separate recursion-owner decision: optional presence wrappers
inline a named type into a shared in-progress buffer rather than invoking its
named validator. Context selection alone cannot repair that truncation. A
bounded witness will establish this boundary before final readiness.

## Selected repair owner

Keep the ordinary and view builders as consumers of one branch-validation
function. Make its physical context explicit instead of constructing ordinary
`typeContext` inside it. Ordinary callers retain the existing service context;
view callers use the existing `projectedTypeContext` used by `buildValidations`,
including its view-validation mode. The helper delegates predicate rendering to
`codegen.ValidationCode`; it must not hand-write dereferences, nil guards,
requiredness checks or per-constraint exceptions. Keep ordinary output stable.

The schema/runtime branch matcher still owns candidate selection, and the
selected branch's validation still checks the same authored constraints. Preserve
`unionjson.Analyze`, candidate error classification and match ambiguity behavior.
Do not replace inline branch validation with a default-view named validator
without establishing that its selected-view contract is identical: result-type
branches may carry views, and blindly calling a default validator can change the
constraints applied. This ticket is physical-context correction, not view policy.

No new public semantics are needed. A projected required field can be absent
before validation; its pointer must be checked before content validation. A
projected defaulted field remains pointer-shaped too. Defaults/presence behavior
and M7 are not changed. The correction requires regenerating affected code.

## Evidence and boundaries

`TestProjectedUnionValidationReadiness` has four direct reds: required range,
pattern, enum and length all use `v.Value` despite projected pointer storage.
Optional range is a passing control; ordinary storage assertions precede each
view assertion and pass. `TestProjectedUnionValidationGeneratedReadiness` covers
eight modules: ordinary required, view required, view optional, view default,
view string constraints, nested object, array-valued field and named array branch.
Required numeric, optional defaulted numeric and required string view cases fail
compilation; the other five build and vet. All modules use verified pushed pin
`v1.10.0-alpha.9.0.20261012201303-8c7d8d711b9e` and explicit local replacement.
The earlier durable #796 reproduction from #784 is unchanged.

An exploratory map-valued field is rejected by the current untagged DSL contract;
this is not a validator regression or a supported control. Existing DSL
`TestUntaggedRejectsScalarBranches`, `TestUntaggedRejectsNestedBranchFields` and
`TestUntaggedAcceptsNamedObjectBranchFieldsAndArrays` pass (0.466s). The supported
collection controls use arrays. Do not expand supported union shapes here.

The context model reproduces current representation disagreement (41 states),
rejects a dereference-only patch that reads absent required fields (45 states),
and passes the physical-context candidate (52 states). It complements, rather
than extends the claims of, the existing projected-presence model. See
`codegen/service/tla/union_validation_context/README.md` for limitations and the
invalid initial scratch-run correction.

The #709 probe passes both field orders with native pointers and fails both with
JSON Optional presence. Its output lacks the recursive named call and repeats a
partial validation buffer. This failure remains when the physical pointer
context is correct. Its durable input is retained as `.go.txt` alongside the
model; it is a separate traversal-owner repair and is not claimed resolved.

## Bounded production acceptance after release

1. Replace unconditional context selection in the shared branch-validation owner;
   retain the existing projected and ordinary context constructors. Direct reds
   become green. No changes to `validation_recurse.go` or Optional recursion.
2. All eight generated modules compile and vet. Add real untagged decode tests in
   ordinary and views packages for absent/null, zero/out-of-range, valid bounds,
   wrong pattern/enum/length, defaulted missing versus supplied values, nested
   invalid values, and arrays. Assert rejected candidates remain rejected without
   panic and valid branches preserve values. Use race checks on those harnesses.
3. Exercise a result-type branch/selected-view control so the context change does
   not silently choose a different default view. Preserve existing ambiguity and
   malformed-JSON contracts using the relevant existing untagged-union tests.
4. Compare parent/candidate and independent-process output for the eight bounded
   fixtures and the existing #784 included-untagged specimen. Ordinary service
   output should remain byte-identical. Expected differences are view branch
   validation using the correct pointer guards/dereferences and required-field
   checks; list every actual changed file. Do not regenerate the entire corpus.
5. Run affected service/codegen direct tests, existing projected-view and untagged
   regressions, pinned scoped lint, and the context model. Preserve #709 evidence
   as classification only. Freeze the final diff for a different agent's review.

The readiness checkpoint contained no production changes. After independent
release, production changed only the shared branch-validation context boundary.
Main/master trackers, shared skills, commits and pushes remain root-owned.


Final-source readiness run: Go 1.27.2, 17.949s, four direct failures plus
three generated compilation failures as described above; all five generated
controls and the optional direct control pass. Log:
`/tmp/loom-projected-union-796-final-red.log`. Separate #709 classification log:
`/tmp/loom-projected-union-796-recursion-classification.log`. Model evidence:
`/tmp/loom-projected-union-796-{legacy2,dereference2,checked2}.log`.

Pinned golangci-lint v2.14.0 reports zero issues for `./codegen/service`
(`/tmp/loom-projected-union-796-readiness-lint.log`). Gofmt and diff checks pass.


## Implementation and final acceptance

The production delta is confined to `codegen/service/service_data_types.go`.
`unionBranchValidationCode` now receives an explicit `AttributeContext` and view
mode. The ordinary union builder supplies `typeContext(scope), false`; the view
builder supplies the existing `projectedTypeContext` and `true`. All predicate,
requiredness and nil-guard rendering stays in `codegen.ValidationCode`. No matcher,
conversion, Optional recursion or view-selection policy changed.

The final direct regression is `TestProjectedUnionValidationContext`; the final
module regression is `TestProjectedUnionValidationGenerated`. The latter has nine
specimens: the original eight plus a named ResultType branch with default/full
views. It builds and vets each module, then runs actual untagged decoding under
race detection. Cases cover absent/null/malformed input, numeric zero and bounds,
string pattern/length/enum failures, missing versus supplied defaulted fields,
nested objects and arrays. Successful candidates retain the branch and decoded
value. Missing optional/defaulted projected fields remain absent; the change does
not introduce default materialization.

The ResultType control verifies two distinct existing contracts: the untagged
branch checks all authored required/constraint fields, while its default-view
validator excludes `hidden` and its full-view validator requires and validates
it. This prevents a future substitution of the default-view validator for branch
validation. Initial exploratory assertions incorrectly expected the excluded
field to be optional in union decoding and guessed the branch tag from TypeName;
the final assertions use the actual authored graph and identifier-derived
`UnionBranch` tag. Those exploratory failures are not defect evidence.

Parent/candidate/independent-process generation used the exact parent archive,
Go 1.27.2 and pushed module pin
`v1.10.0-alpha.9.0.20261012201303-8c7d8d711b9e` with an explicit local replacement
for each source. Ten specimens produce 30 generated files: nine service/view
fixtures plus the delivered #784 included-untagged HTTP fixture. Candidate and
repeat bytes match exactly. All ten candidate modules build and vet. The parent
fails compilation for required numeric, defaulted numeric, required strings and
the ResultType branch; the remaining six controls build and vet.

The complete intended difference list is `gen/validation/views/view.go` in these
eight fixture directories: `array-true-true`, `collections-true-true`,
`default-false-true`, `nested-true-true`, `number-false-true`, `number-true-true`,
`selected-true-true`, and `strings-true-true`. Changes add physical pointer guards,
dereferences and required-field checks. The array/nested/collection and optional
numeric specimens differ only in the required boolean `Other` branch's missing
field check. All ordinary service files and all 13 files in the existing HTTP
fixture are byte-identical to the parent. No checked-in generated fixture changes
are required, and the complete service package (including existing goldens) passes.

Evidence retained until final review:

- `/tmp/loom-projected-union-796-runtime-final.log`: all direct and nine generated
  runtime/build/vet regressions pass, 31.547s.
- `/tmp/loom-796-owner-packages.log`: full `codegen/service`, `codegen`,
  `internal/unionjson`, `pkg` pass, 34.719s / 13.751s / 0.354s / 0.366s; includes
  existing ambiguity, malformed JSON and projected validation contracts.
- `/tmp/loom-projected-union-796-outputs/comparison.json`: exact generated hashes,
  parent differences and repeated-process parity; sibling `diffs/` has every
  changed file's unified diff and `build-results.json` has module outcomes.
- `/tmp/loom-796-final-lint.log`: pinned golangci-lint v2.14.0, zero issues for
  `./codegen/service`. Gofmt and `git diff --check` pass.
- `/tmp/loom-796-final-model-{legacy,dereference-only,checked}.log`: repeat the two
  intended counterexamples and passing candidate; sources and correspondence
  remain beside the owner. #709 remains explicitly unresolved and separate.

Temporary capture drivers were removed from the worktree and preserved only with
the temporary comparison evidence. Parent archive and generated output must stay
available through independent review, then root owns cleanup after delivery.

## Integrated review boundary

Fresh reviewer `/root/review_projected_validation_final` approved the isolated
12-file manifest `45d4ba0f9becf9499775fecebb7e3046c885898eaf8a86a5f14c45b4c668ef48`
without findings. It independently ran all nine generated runtime/race modules
(30.498s), verified all 90 artifact hashes and eight intended differences, and
rechecked all three model configurations.

Root copied those files byte for byte onto `f40d1677c`, with no overlapping
source changes, and added the regeneration note, framework owner and model index.
Integrated projected-validator and included-untagged HTTP checks pass 24.293s and
6.118s. The final integrated diff requires its own authorizing review before commit.
