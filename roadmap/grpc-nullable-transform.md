# gRPC nullable admission — #804

Parent: `39f9ce17c`. Status: implemented; final independent review pending.

## Contract and owner

The public DSL reference rejects nullable gRPC shapes except unconstrained
`Any`, because ordinary protobuf messages cannot distinguish absent, explicit
null, and concrete values. #804 therefore requires early design rejection,
not a wrapper-to-pointer conversion that collapses states. The original witness
contains neither defaults nor enums.

The previous `GRPCEndpointExpr.validateNullableTransport` inspected transport
attributes before finalization fills implicit messages and inherited field
contracts from service payload/result definitions. `Finalize` then introduces
nullable occurrences after admission. The recursive presence predicate itself
already rejects nullable named objects when it sees them. Transport admission
must inspect the complete effective source contract and explicit mappings,
without independently recreating finalization policy.

## Sequence and acceptance

- [x] Read the issue, current admission/finalization owners and public contract.
- [x] Reproduce missing rejection across service and mapping boundaries; retain
  valid native and Any controls, including root Any normalization.
- [x] Model pre-finalization admission and corrected source admission. Demonstrate
  the current counterexample and the candidate invariant before implementation.
- [x] Independently review readiness and release implementation.
- [x] Implement shared effective presence admission; remove the superseded
  incomplete path. Keep transport representation and public semantics unchanged.
- [x] Run direct DSL/expr regressions and affected package tests. Compare bounded
  native/Any generated artifacts byte for byte against the parent, compile/vet
  them, and run existing Any presence roundtrip coverage. Rejected designs must
  fail before code generation. Do not run the full exported-design corpus.
- [ ] Freeze the exact diff and obtain independent final review.
- [ ] Root integrates one atomic #804 commit, pushes, closes the proven issue,
  updates shared guides/checkpoint, and removes task-owned temporary artifacts.

## Boundaries

Root Any is physically wrapped by protobuf normalization and has existing
three-state roundtrip tests. Preserve that accepted exception. String metadata
cannot inherit an Any exception merely because message Any has a dedicated
protobuf representation. Explicit field mappings must not erase nullable source
intent. Recursive named graphs need termination without hiding later nullable
occurrences. Root/source arrays, map values, named aliases, errors and both
streaming directions belong to the admission matrix.

## Implementation

Expression admission collects complete method payload, streaming
payload, result, streaming result and scope-correct mapped error sources, alongside
explicit transport attributes, in one presence-check plan. It reuses the existing
recursive named/collection/union traversal. It distinguishes message-root normalization
from nested positions: a root Any is wrapped by the existing protobuf owner and
must retain its accepted three-state behavior. Metadata checks resolve mapped
fields to their service occurrences before applying the existing non-JSON
presence restriction; do not infer a field's contract from its pre-finalization
placeholder. Do not call mutating Finalize during validation or move all gRPC
finalization ahead of structural validation.

The implementation retains explicit local presence checks and combines them
with source intent. A source-only shortcut would miss locally authored Nullable;
a transport-only shortcut is the observed defect. No downstream transform patch,
new protobuf envelope, or null-to-nil conversion is introduced.

Tests cover root/named alias and recursive graphs,
metadata Any classification, error inheritance, local declarations and malformed
mappings. Rejections name the affected transport position. Ordinary
native, Any field and normalized Any root are accepted controls. The old direct
transport predicate test now reflects supported root normalization while
retaining rejected nested nullable Any collections.

## Evidence

On parent `39f9ce17c`, Go1.27.2:

- `go test ./dsl -run TestGRPCPresenceAdmissionUses -count=1` fails all26 intended
  cases with “expected error but got nil” (0.482s),
  `/tmp/loom-804-admission-red.log`.
- `TestGRPCPresenceAdmissionPreservesSupportedShapes` passes native, Any root and
  Any field admission (0.383s), `/tmp/loom-804-controls.log`.
- Local-source `TestGeneratedNullableAnyRootPayloadRoundTrip` passes generated
  build, vet and absent/null/value/malformed JSON checks (3.705s),
  `/tmp/loom-804-any-wire.log`.
- Legacy TLC fails `NoLostPresence` after31 states: the source is nullable,
  an explicit mapping lacks local presence, admission accepts, then finalization
  exposes the incompatible occurrence. Candidate checks all36 states with
  `NoLostPresence`, `NativeAdmitted`, and `AnyAdmitted` passing. Logs are
  `/tmp/loom-804-model-{legacy,effective}.log`.

The original compile witness is retained in
`codegen/tla/default_representation/grpc_nullable_probe.go.txt`; main compiler
failure was independently recorded in `/tmp/loom-804-main.log`.


## Final validation and integration notes

Readiness was independently approved at manifest `e4b028f28d2c9f71075602da2170b4d42b0737387ac5e4b11c3404c45259341e` before implementation.

- Full expression and DSL packages pass (0.517s and 0.376s). Final admission,
  graph immutability, metadata/type and nested-Any controls pass again after
  narrowing duplicate diagnostics (expr 0.427s, DSL 0.189s). The added malformed
  mapping case retains its original missing-field diagnostic.
- The existing generated nullable Any root test passes build/vet and all four
  presence/malformed cases. `TestGeneratedLocalNullableAnyPresence` additionally
  proves local Nullable on a source Any field uses its real type rather than the
  String placeholder: build/vet and absent/null/concrete roundtrips pass (4.001s).
- Parent/candidate/process-repeat comparison reuses `NamedMetadataDSL`,
  `BidirectionalStreamingRPCDSL` and the existing nullable Any root design.
  All46 accepted artifacts are byte-identical; parent and candidate build/vet.
  The original no-default nullable object produces15 uncompilable parent
  artifacts and now fails design evaluation before creating `gen/`.
- A local Nullable-on-Any field was incorrectly rejected by the parent. The
  corrected candidate produces15 compiling artifacts, identical across isolated
  processes. This removes a false rejection under the documented Any exception.
- Comparison receipts: `/tmp/loom-804-comparison/report.json` and
  `local-any-report.json`. After the final metadata admission refinement, both
  metadata generations still match all15 earlier files exactly; reuse their
  existing build/vet evidence (`final-metadata-repeat.json`). No checked-in
  generated fixture changes; the admission correction does not change accepted
  native or Any representation. No OpenAPI renderer changed.
- The full gRPC package run (268.359s) finds three existing failing tests:
  `TestGeneratedProtoGoNames` passes a newly variadic validator to an old
  fixed-arity harness; `TestNestedStructMetaNames` misses `ValidateA` and
  `ValidateChoice`; `TestGeneratedStructMetaRoundTrip` misses `ValidateChoice`.
  A parent-source overlay independently reproduces the same failures (7.451s).
  No whole-package pass is claimed. Logs are `/tmp/loom-804-grpc.log` and
  `/tmp/loom-804-parent-grpc-failures.log`; this ticket does not repair those
  separate validator identity/harness residuals.
- Scoped lint finds the existing unchecked `RunInvalidDSL` result in
  `dsl/numeric_value_admission_test.go:34`; changed-code lint has zero issues.
  Logs: `/tmp/loom-804-lint{,-focused}.log`. The untouched numeric test is also
  a known residual of the separate numeric source-admission work.

Root integration should update the formal model index and framework skill to
state that gRPC admission combines complete service contracts with local presence
and source-owned physical types, before finalization. Public docs already reject
unsupported nullable gRPC shapes; clarify that Any is a message-only exception,
including normalized message roots, while metadata rejects it. Application
migration is to remove unsupported Nullable or use message Any when explicit null
is needed; regenerating a formerly invalid design now returns a clear error.

Keep all task-owned comparison modules, checker state and raw logs until final
review and integration finish, then remove them after checking active use.


## Independent review correction

The first final diff had one MEDIUM finding: its shallow local-field scan missed
nested local nullable object members, array elements and scalar wrappers that the
parent correctly rejected. The review probes are retained as the expanded
`TestGRPCPresenceAdmissionChecksNestedLocalDeclarations` matrix, including map
values and Any controls. The fix replaces the shallow scan with one recursive
local/source pairing owner in `expr/grpc_presence_mapping.go`, used by message
and metadata admission. It retains positional Any semantics, scalar wrapping,
recursive graph termination and source/local immutability. The ordinary source
walk and paired walk share the same occurrence predicate.

The original ordering model assumed a complete local predicate. The new
`LocalPresence.tla` exposes the shallow omission (38 states), rejects a blanket
Any exception (36 states), and checks recursive positional admission (48 states).
Its README explicitly records this limitation and the implementation/test seams.
Focused expression/DSL cases and the exact reviewer overlay pass after repair.
Generated root/local Any build/vet and presence controls pass (4.642s). All61
accepted artifacts across metadata, streaming, Any root and Any field remain
byte-identical in both process-isolated generations; existing compilation
results therefore remain valid. Receipt:
`/tmp/loom-804-comparison/recursive-admission-repeat.json`. No broad suite rerun;
the earlier parent-reproduced validator failures remain separate.

The repaired diff awaits targeted re-review. Logs:
`/tmp/loom-804-rereview-{focused,generated,reviewer-probe,lint}.log` and
`/tmp/loom-804-model-local-{shallow,recursive,all-any}.log`.

## Main integration sequence

1. Integrate the approved recursive source/local admission owner, its tests and
   both model families without changing the reviewed implementation.
2. Clarify public message Any exceptions and rejected metadata/collection
   Nullable positions; update consumer/framework guidance and the model index.
3. Run complete DSL/expr suites and the two generated Any roundtrip controls,
   then scoped lint. Reuse unchanged accepted-output and model evidence.
4. Independently review the exact integrated diff before committing, pushing
   and closing #804. The three existing gRPC validator failures remain assigned
   to a separate shared-owner investigation.

The medium local traversal finding is confirmed fixed by independent re-review.
Approved source manifest: `250b84c17969633e54204ccdfebd5a9a7f778fbb704d72b221bdabc8c58cbb21`.

Integration against `7326becd4` preserves all 16 reviewed implementation/test/model
files exactly. Full DSL and expr suites pass (0.609s/0.977s); generated nullable
Any root and local-declaration roundtrips pass with module build/vet. Complete
DSL/expr lint reports zero issues, including the separately repaired #806 test.
Logs: `/tmp/loom-804-integrated-{owner,generated,lint}.log`. Prior 61-artifact
comparison and model receipts remain applicable; only shared/public guidance
and this integration record are added. Exact integrated review remains pending.
