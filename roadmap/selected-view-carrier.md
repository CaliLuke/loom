# Selected-view nullable carrier handling — #834

Parent: `0bc139d5e45da0d49d3f77a09fb7d5a8a7497bb5` (pushed #840).
Managed worktree: `selected-view-carrier/loom`.
Status: implementation and bounded acceptance complete; final independent review pending.

## Sequence

1. Preserve #834's exact parent reproduction and repeat it after the declaration
   identity fix. Trace selected-view nested-field and collection-element calls.
2. Add direct red carrier/call controls before production. Evaluate actual DSL
   admission for occurrence and intrinsic Nullable, including collection elements;
   do not infer public support from manually constructed expression graphs.
3. Build a bounded model of selected/excluded fields, requiredness, absent/null/
   concrete states and physical call arguments. Reproduce current invalid pointer
   operations, reject guard-only or null-as-absence repairs, and check the shared
   proposed carrier operation before implementing it.
4. Reuse the minimal ordinary Child/Parent witness; extend bounded generated
   coverage for default/tiny/nested/collection views, ordinary controls, and shared
   service/HTTP/JSON-RPC consumers. Keep independent conversion defects separate.
5. Freeze exact readiness sources, red/control/model evidence and proposed owner
   for root review. Implement only after release; then compare bounded parent,
   current and independent-process repeat outputs, compile/vet and runtime checks.
6. Fresh independent exact-diff review before root's single atomic commit.

## Initial owner evidence

`buildViewObjectValidation` resolves the correct canonical helper name after
#829, but its field record carries only name/helper/requiredness.
`renderValidateTypeCode` unconditionally uses nil guards and passes the stored
field directly. `buildValidations` similarly reduces collection elements to
NilableElement/NullableElement flags; rendering compares nullable wrappers to nil
and passes them to a concrete helper. The shared service view validation owner
must retain occurrence carrier facts and derive presence admission, unwrap and
helper argument operations together. HTTP/JSON-RPC consume this generated owner;
no transport-local guard is proposed.

Ordinary nullable properties distinguish absent and explicit null. Required
nullable properties reject absence and accept explicit null. Only concrete
values run their selected child view. Excluded fields must not be validated.
Collection positions have no omitted element: nullable null is valid, ordinary
null is rejected, and constructed absent wrappers need explicit consideration
against the existing collection validation contract. Canonical helper identity
remains owned by NamedValidationFunction; physical arguments must agree with its
reference while preserving occurrence metadata and declaration storage.

Preserved pre-#829 CLI witness:
`/tmp/loom-829-result-carrier-probe/standalone/design/design.go`; pushed bf7ae
vet reports invalid Nullable nil comparison plus the now-fixed helper identifier.
The follow-up issue is https://github.com/CaliLuke/loom/issues/834.

## Expanded parent evidence and prerequisite freeze

Occurrence optional/required direct tests fail only nullable pointer operations;
ordinary fields and excluded tiny views pass. `Attributes(func(){ Nullable() })`
is valid inside ResultType, unlike a bare Nullable call directly in ResultType.
This exposes intrinsic root validation selectors, viewed envelope references,
projection entry nil guards and CollectionOf element guards in the same physical
carrier owner. Preserve the metadata envelope independently of nullable payload.
Correct the earlier #829 wording during delivery where it could suggest that all
intrinsic nullable ResultType declarations are rejected.

The three-carrier generated specimen has default/tiny/nested views, ordinary and
required fields, and Child/Parent collections. Ordinary HTTP/JSON-RPC compiles.
Occurrence fields fail with the expected shared-view guard/call errors. Intrinsic
service-only generation reaches the corresponding root/element failures.
Its transport variant instead fails earlier: expr.valueViewPlan.project builds
an object declaration attribute without Nullable. A direct expr.Project probe
reproduces true-to-false effective nullability without any Go generator.

Root classified semantic preservation as a separate atomic prerequisite. Freeze
this carrier candidate until that owner is repaired, then restore full intrinsic
HTTP/JSON-RPC generation and runtime acceptance. Do not reduce the promised
matrix or introduce a transport workaround. Preserved full witness is
`/tmp/loom-834-intrinsic-transport-witness_test.go`; direct projection witness is
`/tmp/loom-834-semantic-projection-probe_test.go`. No production changes exist.

Model old/guard-only/null-as-missing/unwrap-only fail at163/167/176/132 states;
checked passes256. Logs and configs are under `/tmp/loom-834-model` and the durable
model directory. Full readiness review remains pending, including root nullable
selection/admission and physical view-envelope facts discovered during generation.

## Resumed readiness after #837

The original ten-file freeze hashes were verified before safe fast-forward of
this managed checkout. #837 now preserves semantic intrinsic nullability.
Restore the full intrinsic HTTP/JSON-RPC fixture; add direct tests for root
presence admission, physical envelope references and projection entry guards.
Extend the existing model with root construction/projection/envelope operations
and rejected metadata-envelope wrapping. No production changes precede review.

The proposed shared private selected-view carrier record binds presence,
concrete access and the canonical named-validator call target. Root validator
bodies, nested fields and collection elements derive their operations from it.
Constructors admit required roots by actual carrier presence. Projection entry
guards apply only to physical pointers. The viewed metadata envelope has its own
non-null declaration; its Projected payload retains intrinsic Nullable. Preserve
existing ordinary object-pointer and collection-value envelope references.

Expanded direct parent evidence: seven physical-carrier subtests fail and five
ordinary/excluded controls pass. Root metadata envelope, required presence and
both-direction projection entry tests now reproduce their exact failures.
The restored generated matrix has one passing ordinary design and two nullable
compile failures after successful complete service/HTTP/JSON-RPC generation.
No intrinsic transport specimen remains excluded. Logs retain actual emitted Go.

The expanded model checks 216 states and six rejected alternatives. Required root
absence, required nullable field absence and constructed absent collection
elements reject; explicit null passes without child predicates. Runtime acceptance
will exercise direct constructors/validators and mounted HTTP/JSON-RPC paths for
default/tiny/nested selections, required/optional null/absent/valid/invalid values,
collection element admission, unknown view and ordinary nil controls. Compare
all three designs on parent/current/isolated repeat; ordinary bytes must remain
unchanged, nullable differences must correspond to the reviewed shared owner.

## Implementation checkpoint and independent decoder prerequisite

The approved shared carrier implementation now passes direct controls (.546s).
The ordinary generated fixture remains byte-identical and the occurrence fixture
compiles. Intrinsic service/HTTP code advances past the carrier errors, but the
full module finds an independent JSON-RPC decoder mismatch: its local body uses
ClientBody.VarName instead of the shared ValueRef. An unchanged pushed 2a8337700
remote module with an ordinary nullable Type and no result views reproduces the
same validation/conversion argument errors. No JSON-RPC production edits made.

Preserve this candidate and its entire promised intrinsic matrix while that
shared decoded-body-reference prerequisite is reviewed and repaired. Runtime
matrix completion and final output comparisons remain outstanding. Do not claim
full #834 acceptance or omit the intrinsic JSON-RPC client from compilation.

One readiness assertion was corrected against the existing shared converter:
nullable projection preserves absence through IsNull()/Value(), not a literal
Present() call. Generated diagnostic paths after root unwrapping now retain the
caller's validationPath rather than leaking the local variable name `actual`.

## Resumed after #840

Verified all nineteen frozen source hashes before advancing to pushed #840. The
fast-forward had no overlapping production files. The shared decoder now consumes
ValueRef; all three complete fixtures compile. Continue the approved full runtime
matrix and retain the original red/model evidence. No carrier scope was removed.

## Method-local root occurrence completion

A fourth bounded specimen follows the completed three-carrier matrix. On pushed
0bc139d5e, a named ordinary Type with method-local Result Nullable generates and
vets, while the equivalent viewed ResultType fails transformation. The shared
viewed envelope rebuilds Projected from Type and loses occurrence Nullable.

1. Add direct envelope/reference and canonical-call regressions before edits.
2. Retain the projected occurrence on the metadata envelope; adapt canonical
   projection and validation calls at this shared boundary in both directions.
3. Extend the model with occurrence preservation and round-trip call state.
4. Keep the three previous specimens; add root-occurrence runtime and exact
   parent/candidate/process-isolated repeat comparison before final review.

## Final acceptance after root-occurrence completion

The shared viewed envelope preserves local Nullable through method projection,
metadata construction and cache identity. Ordinary and nullable methods sharing
one named result have separate envelope/constructor signatures in either visit
order; canonical projected declarations and helper bodies remain shared.
Constructors adapt occurrence object/collection wrappers in both directions.
Envelope validation rejects absent nullable roots independently of reusable
intrinsic helpers. HTTP/WebSocket/JSON-RPC calls consume the recorded validator
name; SSE already followed this ownership contract. No transport-local policy.

Four final generated specimens pass 620 service/HTTP/JSON-RPC cases plus 28
helper/envelope cases. Current and process-isolated repeat match all 120 files.
The ordinary 30-file fixture is unchanged. Field-occurrence changes two view
files; intrinsic changes seven service/view/server files, solely references,
nullable operations and now-unused imports. Exact root-occurrence parent
fails generation before files exist; its 30 candidate files compare exactly
with repeat and differ from the ordinary control in eighteen declared carrier,
constructor, validation and transport-reference files. The full intended delta
is retained in `/tmp/loom-834-complete-generated.patch`; artifact hashes are in
`/tmp/loom-834-complete-comparison.json`.

The extended model checks 432 states and rejects ten alternatives. Shared
metadata reuse, canonical call arguments and round-trip presence supplement
original validation/admission invariants. Direct tests cover both reuse orders;
remote-only pushed-parent ordinary Type versus viewed ResultType proves this
extra method-occurrence boundary belongs to this ticket. Earlier prerequisites
#837 and #840 remain delivered separately and are not included in this diff.

Final review receives exact source/evidence hashes, bounded generated differences,
parent/current/repeat logs, full service package results, affected HTTP/JSON-RPC
fixture controls, adjacent union runtime controls, lints and model correspondence.
No commit or publication before fresh independent exact-diff approval.

Final checks: full service package 36.148s; generated four-fixture acceptance
6.954s; adjacent generated runtime controls 19.246s; affected HTTP/JSON-RPC
decoder/writer/WebSocket goldens 1.059s/0.532s. Scoped golangci reports zero issues.
Formatting, NameScope, filesize and documentation gates pass (existing large-file
warnings only). All ordinary golden expectations remain unmodified.
