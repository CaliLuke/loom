# Sum-union branch validation carrier — #823

Base: `00fc9ba8bdebc8b77c42bd30f724707bbbea86a0`.

1. [x] Preserve parent constrained nested-union panic independently of #819.
2. [x] Identify physical branch representation and invocation owner; test
   pointer, explicit nullable, nil/non-nil collections and primitive controls.
3. [x] Model the old panic and candidate carrier-sensitive invocation without
   assuming generated-text shape. Keep nullable and collection rules explicit.
4. [x] Independent readiness review before production editing.
5. [x] Implement at the shared branch validator owner, then bounded direct and
   generated parent/current/repeat checks, compilation, runtime and lint.
6. [ ] Fresh final review, atomic integration; apply prerequisite to #819 and
   rerun only the changed combined boundary. Root owns commit/push/closure.

`renderUnionSumValidationCases` currently infers RequiresValue from whether
rendered code starts with `if actual != nil`. Nested named unions have pointer
storage but begin validation with a switch. Parent witness adds only MinLength(1)
to the existing nested-union fixture's Leaf.name; parent generation and original
round trips pass, then constructing a nil nested branch and invoking the public
validator panics at actual.Kind(). Wire null rejects before reaching the call.
Evidence: `/tmp/loom-819-nil-parent-generation.log`,
`/tmp/loom-819-nil-parent.log`, `/tmp/loom-819-nil-parent`.

Both service and HTTP sum-union declarations derive branch FieldType through
NameScope.GoTypeRef. That representation must drive safe invocation; text is
not a type oracle. Keep pointer absence distinct from an explicit nullable
carrier and from nil slice/map values. Collection value validators must still
check constraints when nil represents an empty value. No HTTP/JSON-RPC exception,
empty stub, or rendered-code string inspection is acceptable.

The related #819 candidate is frozen in its own worktree. Its eligibility repair
and evidence are separate and must remain unchanged during this prerequisite.

## Recovery readiness manifest

Recovered at unchanged parent `00fc9ba8bdebc8b77c42bd30f724707bbbea86a0`.
No production edits, commit, push or external communication have occurred.
The prior worker's `/tmp/loom-823*` artifacts remain intact, including the
rendered parent module. Recovery reruns are under `/tmp/loom-823-recovery-*`.
The durable model README records commands, bounds, state counts and limitations.

Readiness scope is these two regression files, five model/config files, the
model README and this plan. The direct regression now tests four enclosing
contexts, including an extracted object whose value fact must not leak into a
pointer-backed branch. The generated regression adds concrete invalid nested,
object and nullable controls and uses Go 1.27's standard JSON v2 package.

Commands already run at the parent with Go 1.27.2:

```sh
go test ./codegen -run '^TestUnionValidationUsesPhysicalBranchCarrier$' -count=1
LOOM_DIR="$PWD" go test ./codegen/generator -run '^TestUnionBranchCarrierValidation$' -count=1
```

The first fails 11 of 24 cases at the incorrect storage/guard decision. The
second generates and vets successfully, then fails exactly the nested-nil panic,
zero-minimum nil list/map acceptance, and positive-minimum nil list/map error
classification. All four model configurations reproduce their expected outcomes.
Production implementation may start only after independent readiness review.

## Implementation and acceptance plan

1. Resolve selected-branch representation through the same scope reference used
   for declarations. Use it for pointer presence and concrete-value invocation;
   delete rendered-validation prefix inference. Retain child field layout and
   the nullable carrier owner. Do not change named-validator eligibility.
2. Make the direct and generated regressions pass; run the full shared codegen
   package and affected service/HTTP generator tests. Preserve empty union,
   nullable absence/null, pointer nil rejection and nonnil constraint controls.
3. Compare parent, candidate and an independent candidate generation for the
   constrained existing nested-union design and the carrier matrix. Reuse the
   existing `TestUnionInUnionRoundTrips` and projected/untagged union controls
   for service, HTTP, JSON-RPC and gRPC boundaries. Select exact cases before
   running; the full exported-design corpus is not a ticket gate.
4. Enumerate every changed generated file and intended difference. Expected
   differences are pre-invocation missing-value guards for nested pointers and
   collection calls without the wrong missing-value/skip guards. Candidate and
   repeat output must agree byte for byte; compile/vet generated modules and
   run their runtime checks. Regenerate any affected checked-in fixtures through
   their normal generator only after identifying them.
5. Update owner documentation/model correspondence, run touched-source lint and
   diff checks, and request fresh independent review of the exact final diff.
   Root owns the single atomic commit, integration and dependent #819 rerun.

Acceptance: every selected physical pointer branch fails safely when nil;
explicit nullable null remains valid and absence remains invalid; nil and empty
collections both receive their effective value constraints; valid branches and
existing wire-null rejection remain unchanged. This is a shared representation
correction, with no transport-local exception or source-text-shape inference.


## Final candidate evidence

Readiness was approved after its sole low-severity formatting finding was fixed
and confirmed. The production diff is limited to shared selected-branch analysis, its renderer
and its context comment: `Scope.Ref` supplies pointer storage, and
`validationValue` is reset for the selected branch. The existing nullable owner
and child field layout remain in use. Public docs/consumer guidance describe
constructed values and regeneration; framework guidance and the model index
record the owner.

The final carrier fixture replaces redundant `MinLength(0)` with `MaxLength(2)`
and adds oversized list/map controls. The former generated `len(value) < 0`,
which Staticcheck correctly rejected. This fixture-only refinement keeps an
empty collection valid and tests a meaningful upper-bound rejection. It does
not add a generator predicate optimization to this ticket. Parent/current/repeat
carrier checks were rerun after the refinement; other valid evidence was reused.

### Bounded generation comparison

The existing `runDesignHarness` generates service and transport files, tidies,
vets every generated package and runs the module harness. A temporary Go overlay
retains each module under an output directory instead of `t.TempDir()` and adds
only `MinLength(1)` to the existing nested-union design's `Leaf.name`. A third
comparison invokes that same helper with
`service/testdata.ProjectedUnionValidationDSL("array", true, true)` and an empty
root test package. The existing service test supplies its runtime coverage.
Parent generation overlays exactly `codegen/validation_recurse.go`,
`codegen/validation_render.go` and `codegen/transformer.go` with their `00fc9ba8b` contents; candidate and repeat
use current source. The checked-in new carrier test is identical in all three.
All runs select local source with `LOOM_DIR` and Go 1.27.2.

```sh
LOOM_DIR="$PWD" LOOM_823_OUTPUT=/tmp/loom-823-comparison/parent go test -overlay=/tmp/loom-823-comparison/parent-overlay.json ./codegen/generator -run '^(TestUnionBranchCarrierValidation|TestUnionInUnionRoundTrips|TestUnionCarrierProjectedComparison)$' -count=1
LOOM_DIR="$PWD" LOOM_823_OUTPUT=/tmp/loom-823-comparison/current go test -overlay=/tmp/loom-823-comparison/candidate-overlay.json ./codegen/generator -run '^(TestUnionBranchCarrierValidation|TestUnionInUnionRoundTrips|TestUnionCarrierProjectedComparison)$' -count=1
LOOM_DIR="$PWD" LOOM_823_OUTPUT=/tmp/loom-823-comparison/repeat go test -overlay=/tmp/loom-823-comparison/candidate-overlay.json ./codegen/generator -run '^(TestUnionBranchCarrierValidation|TestUnionInUnionRoundTrips|TestUnionCarrierProjectedComparison)$' -count=1
```

The final carrier-only reruns use the same commands with
`-run '^TestUnionBranchCarrierValidation$'`. These are independent Go test
processes; all rendered `gen/` files are compared as bytes, not decoded values.

| Design | Obligation | Files | Changed from parent |
| --- | --- | --- | --- |
| Carrier matrix | Pointer, explicit nullable, primitive and nil/empty/bounded collection behavior; service and HTTP compile | 21 | HTTP client `cli.go`, client/server `types_validation.go` |
| Existing nested union with constrained leaf | Nested/deep nested, collection-held union, HTTP/JSON-RPC round trips, gRPC conversion control | 53 | HTTP nest and JSON-RPC nestrpc client `cli.go`, client/server `types_validation.go` |
| Projected untagged array testdata | Selected-view collection invocation and ordinary service/untagged declaration control | 4 | `validation/views/view.go` |

All 78 candidate/repeat artifacts are byte-identical. Exactly ten files change
from the parent. Their only semantic differences are missing-value guards before
nested pointer invocation and removal of missing/skip guards around collection
value calls. Declarations, codecs, gRPC output and OpenAPI bytes are unchanged.
All nine generated modules compile/vet; the sole expected parent runtime failure
is the carrier regression. All candidate/repeat runtime checks pass.

The 51 tracked generated fixture artifacts contain no sum-union validation
switch. The checked-in HTTP/JSON-RPC ticktock and mixedtick fixture designs have
no union branches. No checked-in generated fixture changes are required.

### Gates and independent baseline failures

- `go test ./codegen -count=1` passes, including all 24 direct carrier cases.
- `LOOM_DIR="$PWD" go test ./codegen/service ./http/codegen -run '(Union|union)' -count=1`
  passes the service union selection, including nine generated projected/
  untagged contexts. HTTP's union selection has exactly two existing structural
  test failures: `TestGeneratedOptionalUnionObjectValidationCompiles` and
  `TestTypeFilesNamedUnionUsedTwiceValidatesBothFields`. Both assert old literal
  diagnostic paths (and former local names). The same failures reproduce under
  the exact parent production overlay; no candidate-only failure was observed.
  These assertions are already corrected by #821, commit
  `0edf714f8977e304283c22af5d027a8080d6c3e3`; they are not a new backlog item.
- The unmodified `TestUnionInUnionRoundTrips` still reproduces #819's undefined
  `ValidatePeerResponseBody` in HTTP and JSON-RPC clients. The constrained
  comparison above isolates #823 and passes. Eligibility remains untouched.
- Golangci-lint 2.14.0 (Go 1.27.2) passes `./codegen ./codegen/generator` and the
  generated source of all three candidate modules with `.golangci-generated.yml`.
- Repository gofmt, name-scope, filesize, JSON-v2 and documentation checks pass.
  Filesize reports only existing warning-threshold files, not hard-cap failures.

Evidence remains in `/tmp/loom-823-comparison/`,
`/tmp/loom-823-codegen.log`, `/tmp/loom-823-union-packages.log`,
`/tmp/loom-823-http-parent-controls.log` and `/tmp/loom-823-lint-source.log`.
Original readiness/model evidence is preserved. Root owns fresh exact-diff
review, the atomic commit and #819 integration. Cleanup is deferred until the
reviewer and dependent #819 task no longer need these task-owned outputs.

Final integration control: an overlay takes only
`http/codegen/optional_validation_compile_test.go` and
`http/codegen/union_reuse_validation_test.go` from #821's exact commit above,
leaving #823 production source intact. This focused command passes both tests,
including generated compilation and runtime checks:

```sh
LOOM_DIR="$PWD" go test -overlay=/tmp/loom-823-comparison/821-tests-overlay.json ./http/codegen -run '^(TestGeneratedOptionalUnionObjectValidationCompiles|TestTypeFilesNamedUnionUsedTwiceValidatesBothFields)$' -count=1
```

Evidence: `/tmp/loom-823-http-821-controls.log`. The two #821 files remain
unchanged in this worktree and must retain their main-branch versions during
integration. No additional correctness finding remains within #823's scope.


## Targeted final-review correction

The first final review found a medium-severity admission gap: a physical pointer
branch with no value predicates still disappeared under `val == ""`. A selected
nil unconstrained object therefore passed validation. The earlier candidate's
broader pointer-admission claim was withdrawn pending a direct/generated
reproduction and model extension; the existing constrained-branch evidence
remains valid within its narrower scope.

Before the repair, the permanent `unconstrained_object` direct case failed in
all four contexts and generated `empty_object_nil` returned no error.
`empty_object_valid` remained valid. The model now independently varies whether
a pointer branch has predicates: `SkipEmpty.cfg` reproduces the incomplete
candidate (`Contract`, 70 states), while `Physical.cfg` passes 84 states.
The original three expected-negative checks retain their 66/27/52 state counts.
These checker results were obtained before changing production admission.

The shared owner now admits a case for either pointer presence or nonempty
value validation. The renderer omits `break` only when a presence-only case has
no subsequent invocation to skip. A direct rendered regression and generated
Staticcheck cover this control-flow detail; it introduces no new carrier rule.
The constrained nested fixture now additionally emits presence-only cases for
its unconstrained `Other` and `Tags` branches, including branches nested inside
union elements. This is an intended consequence of the corrected admission
contract, not a named-helper eligibility change.

Rechecked evidence:

- Full `./codegen` tests pass, including 28 direct carrier contexts.
- The generated matrix passes all 26 constructed cases, collection wire
  round trips and non-nullable null rejection; nil and nonnil unconstrained
  object values are explicit controls.
- Parent carrier generation was repeated with the expanded input; it fails its
  original cases and the newly added empty-object nil case. Current and a fresh
  independent repeat regenerate all three bounded designs, compile/vet and run
  their harnesses. All 78 candidate/repeat artifacts match byte for byte.
- The same ten generated files change from the parent; intended differences now
  also include presence-only branch cases/switches, beyond nested-pointer guards
  and collection guard removal. Declaration/codec/gRPC/OpenAPI bytes stay equal.
- Touched-source and all three generated-source lint selections pass. The
  service union selection and the focused #821 assertion-overlay controls pass.

Logs are `/tmp/loom-823-empty-{direct-red,generated-red,break-red,codegen-green,lint-source,service-unions,http-821-controls}.log`,
`/tmp/loom-823-empty-model-*.log` and comparison `*-empty-final.log` /
`empty-lint-*.log`. Temporary outputs remain for the targeted independent
re-review and dependent #819 task. No commit or integration has occurred.
