# Projected nullable declaration identity — #829

Parent: `bf7ae22464aad50a60d5f14de45d8d6c4f13114e`.
Managed worktree: `nullable-declaration-identity-completion/loom`.
Status: implementation and bounded acceptance complete; awaiting independent
final review. Shared declaration/helper identity preserves occurrence carriers.

1. Preserve the parent generator failure and locate the shared declaration owner.
2. Add direct red identity/carrier/order controls and generated projection cases.
3. Model declaration identity independently from occurrence carrier and test
   rejected alternatives before selecting the repair.
4. Obtain independent readiness review of the exact model/tests and bounded plan.
5. Implement at shared ownership, rerun affected model/direct/generated checks,
   compare representative parent/current/repeat artifacts and compile them.
6. Fresh independent exact-diff review, then one atomic root-owned commit.

The minimal issue reproduction fails while formatting a view declaration:
`type loom.Nullable[NullableTextView] string`. Nullable objects emit the same
invalid declaration pattern; nullable nested unions emit a validator named
`Validateloom.Nullable[NestedView]`. All fail before nullable wire decoding.

Initial owner evidence: `buildProjectedType` uses occurrence `GoTypeName` as its
`Name`/`VarName`, though that helper intentionally preserves Nullable wrappers.
`buildValidations` separately concatenates the same wrapped name instead of using
its already-resolved `NamedValidationFunction.Name`. The latter's parameter Ref
also receives the first encountered occurrence, so tests must distinguish
intrinsic declaration nullability from occurrence-only wrapping and visit order.
`GoValueTypeName`, physical named validation identity and union declaration
collection already supply relevant shared contracts; do not add transport-local
renamers or strip wrapper strings downstream.

Bounded acceptance must cover nullable primitive/object/nested-union view types,
ordinary controls, intrinsic and occurrence nullability, reused named declarations
in both visit orders, and pointer/value validator parameter agreement. Generated
coverage must restore #824's blocked projected Nullable null/value round trips,
missing/nonnullable rejection and selected-view validation. Retain ordinary
projected, recursive and physical declaration-name controls. No exported-design
corpus or new dependency is needed.

## Readiness evidence and supported repair

The direct matrix reproduces illegal declaration/helper identifiers in every
nullable shape, and wrong parameter/reference storage when a nullable occurrence
of an ordinary named type is collected first. A concrete-name-only repair is
therefore insufficient. The generated #824 specimen is extended in place:
projected results now inherit its whole branch contract, with nullable object and
nested-union branches added. Ordinary service/HTTP/JSON-RPC and projected targets
retain absent/null/concrete/wrong-type controls. Parent generation still fails
at `type loom.Nullable[BranchNullableView] string`; no production changes exist.

Model: old fails identity, names-only fails parameter agreement, erase-presence
fails occurrence preservation (each 3 generated / 3 distinct states). Checked
passes all invariants and termination (34 / 16). Inputs and pinned run instructions
are under `codegen/service/tla/projected_declaration_identity`; logs remain in
`/tmp/loom-829-model`. Direct/generated red logs are `/tmp/loom-829-direct-red.log`
and `/tmp/loom-829-generated-red.log`.

Proposed shared owner: projected named declaration construction must normalize
its input to the intrinsic named declaration for definition identity/parameters,
without mutating the occurrence graph. Use `GoValueTypeName` for allocated
concrete declaration identifiers. Use `NamedValidationFunction` for the complete
validator name and physical Ref, instead of separately reconstructing its name
from a nullable reference. `collectViewUnionTypes` must use the same concrete
identity when emitting intrinsically nullable named unions. Keep occurrence
wrappers on struct fields, sum-type branches and wire admission; preserve name
reservations and projected package placement. Do not change NameScope's existing
nullable type-reference semantics or add per-transport name substitutions.

After readiness approval, bound generation to the extended #824 transport/view
specimen plus existing ordinary projected-union and recursive-view controls.
Compare parent/current/separate-process repeat exact artifacts and compile/vet;
the malformed nullable parent is the expected negative control. Use a verified
pushed source requirement and exact local replacements for temporary modules.
Run service projection/union declaration tests, shared named validation identity
controls, and selected-view runtime checks. Any additional implementation seam
exposed during compilation must be explained within this identity owner or
separated as a new defect. No full corpus is part of acceptance.

## Revised readiness: ordinary service declarations

The frozen production candidate remains unchanged. #832's broader carrier matrix
exposed `type loom.Nullable[NullableNested] struct` in ordinary service output,
without ResultType or views. This is the same declaration/reference confusion:
`collectUnionTypes` calls `GoTypeName` for a named union declaration, just as the
original `collectViewUnionTypes` did. Both collectors already key declarations
by the referenced named hash; retain that allocation owner and ask its existing
`GoValueTypeName` operation for the concrete declaration identifier. No new
transport naming policy, wrapper stripping, or union identity algorithm is needed.

`TestNullableUnionDeclarationIdentity` covers the service and view collectors,
collecting intrinsically nullable named unions through both occurrence orders.
It requires one concrete named declaration, retained `loom.Nullable[Named]` use
references and unchanged intrinsic/occurrence flags. Exact-parent overlay fails
both collectors; the frozen candidate fixes views but still fails service in both
orders. Existing raw/named alias identity permutations remain unchanged controls.

The generated source witness is
`/tmp/loom-832-intrinsic-union-identity-witness_test.go`; failure is preserved in
`/tmp/loom-832-intrinsic-union-identity.log`. Restore this previously unrepresentable
case to the delivered #832 specimen during #829 acceptance, alongside its ordinary
and occurrence-nullable controls. Keep #829's entire projected specimen, adding
selected-view rejection/omission and collection-of-view runtime controls. Inspect
related validator invocation names in `buildViewObjectValidation`, collection
`buildValidations`, and projection helper naming; use the shared physical validator
identity owner for declaration/call agreement. Do not repair any distinct carrier
admission or conversion defect by renaming it.

The existing projected identity model applies equally to both named declaration
collectors: allocation is intrinsic, occurrence storage is independent. No model
state or invariant changes are needed; its prior 34/16 checked result is reusable.
The pre-existing union declaration model remains responsible for raw/named hash
identity and declaration cardinality. Update correspondence, not the model's scope
claims. Final acceptance stays bounded to the expanded nullable transport/views
specimens plus ordinary required-union views and recursive declaration controls.
No full corpus is required.

After #832 delivery, create or advance a suitable managed worktree without losing
this frozen candidate, applying it on the delivered parent. Record the new parent
and source hashes before implementation. Only the service collector's declaration
identifier is newly proposed production scope at this readiness step; any further
name/reference seam exposed by compilation must be assessed against this owner.

Resumption: root approved expanded readiness after #832 delivery. The prior
frozen candidate is preserved in its original worktree and applied here from
`/tmp/loom-829-expanded-candidate.patch` (SHA48bc16f1bcbb6235272fad8e769b7a2024c3062513bc0bdff5fe90830bf92552).
Only the reviewed identity owner is authorized; unrelated carrier defects remain separate.


## Final implementation and acceptance

The shared identity correction now covers service, projected and HTTP named union
collectors. The HTTP collector had the same direct red in both occurrence orders
and both transport object modes; root approved it within this owner. All retain
existing named hash allocation, declaration cardinality and transport membership.
Projected declaration construction uses intrinsic local input copies; it does
not clear occurrence nullability. Concrete identifier fragments come from
`GoValueTypeName`. Named validation definitions and selected-view/collection calls
use `NamedValidationFunction`, with the intrinsic declaration's physical parameter.
Projection initializer names use the same concrete identifier operation.
JSON-RPC consumes these shared owners; no local naming substitutions were added.

The 20-context direct projection matrix covers five shapes, intrinsic nullability
and both occurrence visit orders. Service/view and HTTP named-union direct tests
preserve hash/cardinality and wrapped use references. A corrected selected-view
helper test uses occurrence Nullable on an ordinary Child ResultType; intrinsic
Nullable directly in ResultType is rejected DSL and is not claimed as a supported
case. The valid helper test fails on exact parent and passes with this change.

Generated acceptance reuses the #824 and #832 fixtures. The transport/view fixture
now covers nullable primitive, object, occurrence nested union, intrinsically
nullable named nested union, Any and ordinary branches, with missing/null/value/
wrong-type decoding controls. HTTP and JSON-RPC projected constructors check
constraints in default views, omission in tiny views, unknown-view rejection,
collection views and nested selected views. The #832 fixture restores its
intrinsic nested-union case: all 15 shape/presence contexts now compile and run.

Bounded comparison uses four designs and the exact parent production overlays:

| Specimen | Parent | Current/repeat | Obligation |
| --- | --- | --- | --- |
| Expanded null-admission transport/views | Expected render failure, zero artifacts | 44 identical artifacts | Formerly invalid declarations; selected-view runtime contracts |
| Expanded nullable transform carriers | Expected render failure, zero artifacts | 21 identical artifacts | Intrinsic nullable named nested-union identity with all carrier controls |
| Existing required-union views | 14 artifacts | 14 unchanged, repeat identical | Ordinary projected declarations and physical presence |
| Existing recursive projection | 4 artifacts | 4 unchanged, repeat identical | Recursive declaration reuse |

All 83 current artifacts match independent-process repeats. Parent versions of
the two expanded fixtures also compile/run on parent and current, with 36 + 21
artifacts byte-identical. Together these preserve 75 previously supported parent
artifacts. Successful modules are vetted (compiling all generated packages) and
run their harnesses. Modules require pushed
`v1.10.0-alpha.9.0.20261013040829-bf7ae22464aa` and use exact local replacements;
parent generation overlays all five changed production files from `bf7ae2246`.
The output inventory and hashes are in `/tmp/loom-829-comparison/summary.json`.

Affected checks: full `codegen/service` package passes (53.305s); HTTP union/view/
nullable declaration selection passes (92.608s); focused identity/allocation
checks pass; golangci-lint on service/generator/HTTP codegen reports zero issues.
The existing model's 34/16 checked result and three 3/3 negative counterexamples
remain valid: implementation did not change the modeled state or invariants.

### Separate carrier limitation

A valid ordinary Child ResultType, used by Parent first as plain `a` then Nullable
`z`, exposes selected-view code comparing `loom.Nullable[ChildView]` against nil.
The exact pushed parent also emits that comparison, alongside its old helper-name
error; the candidate fixes the name while leaving the guard/argument mismatch.
A standalone pinned-parent CLI reproduction is retained under
`/tmp/loom-829-result-carrier-probe`. This is a separate carrier owner, excluded
from #829 acceptance and tracked as [#834](https://github.com/CaliLuke/loom/issues/834)
rather than repaired by name changes.
The promised nullable union branch matrix remains fully enabled; no previously
accepted generated case was removed. Nullable ResultType member carrier support
is not established by these identity tests or the bounded model.

### Review and cleanup

Root owns fresh independent exact-diff review and one atomic commit. Keep #832
and #829 comparison evidence until that review and integration finish. Then retain
source hashes, concise check receipts and durable model/reproduction inputs before
removing inactive generated modules and checker outputs.
