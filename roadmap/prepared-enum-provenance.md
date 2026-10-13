# Prepared body enum provenance (#799)

Parent: `0dfab0ccd`. Owner: evaluated `expr` constraints and prepared HTTP bodies.

1. Re-run the exact durable issue design on current source; determine whether
   an already-delivered shared-owner repair resolves it before changing code.
2. If still failing, add direct mapped/prepared-body provenance controls and
   model the inherited declaration versus later refinement before implementation.
3. Preserve enum membership as a predicate at its declaring occurrence. A child
   minimum narrows values; inherited enum constants need not all satisfy the
   derived contract. Defaults remain checked against the complete final contract.
4. Validate accepted/rejected values and relevant request/response/explicit body
   consumers with bounded generated compilation, wire and OpenAPI checks.
5. Independently review exact changes or reconciliation evidence; commit/push
   a repair if needed, close only when the original design and contract pass.

No implementation is authorized by this note alone; investigation establishes
whether a defect remains and which shared owner loses provenance.

## Readiness investigation sequence

- [x] Original issue remains red on parent `0dfab0ccd`: prepared
  `ReadRequestBody` is incorrectly reported as the enum declaration.
- [x] Add shared mapped-attribute and actual request/response/explicit/streaming
  constructor probes; distinguish named aliases from composed object layers.
- [x] Test candidate representations in isolated test-only construction, including
  derived value membership, declaration identity and final-contract defaults.
- [x] Model current copying, predicate-only lowering and retained declaration
  candidates before selecting the owner; state projection/field-removal limits.
- [x] Freeze readiness with precise production seam, expected output changes,
  bounded generated/wire/spec acceptance and an independent review request.

No production source is changed during this investigation.

## Established cause and scoped proposal

The original source at parent `0dfab0ccd366d176d2f9f3c1294d0506d1b7a35b`
fails before generation. The enum is declared on EnumBase, yet preparation
reports the excluded n=1 constant as declared by ReadRequestBody. This is a
representation loss, not a conflict in the accepted DSL semantics (see
`docs/dsl-reference.md`, numeric constraint/default/enum contract).

`NewMappedAttributeExpr` copies the first inherited non-nil public Validation,
then `Remap` replaces a named type chain with its object. Composition's public
carrier already contains inherited predicates; its private declaration edges
are on the named attribute being removed. The new object no longer knows where
its enum was declared. A local Required additionally hides the enum altogether.
Request and response body wrappers expose this defective mapped object. Explicit
bodies and streaming keep their chain and pass the same probes.

A second observed failure appears after the test-only provenance repair:
`valuePlanBuilder.object` matches enum constants using complete field names.
The semantic member n does not match target n:wire_n, so it silently skips it and
projects each enum member as `{}`. Both boundaries belong to shared expr
representation: retain declaration context when flattening, then bind constants
by their logical member identity while using the target's wire spelling.

Proposed production scope (pending independent readiness approval):

1. Replace `inheritedAttributeValidation`'s first-carrier selection with retained
   named/occurrence constraint edges when a mapped object removes a named chain.
   Keep authored local constraints local; preserve their order relative to the
   eliminated named edge and existing Extend parents. Physical Required names
   continue to normalize through the existing mapping owner. Never treat a
   copied enum as newly authored at the transport body.
2. Let the existing effective-constraint and value-plan owners evaluate those
   retained edges. Do not call `EffectiveConstraintsFor` eagerly for every
   mapped constructor: mappings are also created during incomplete DSL assembly.
   The concrete test-only named-edge candidate passes both plain and local
   Required cases without eager evaluation. Finalized graphs and physical
   carriers must stay detached from mutable consumer copies.
3. Make inherited enum constant member binding use logical attribute identity
   under the existing controlled ancestor-clause mode, retaining the target's
   wire policy. Keep unrelated ordinary plan mappings rejected and detect
   ambiguous logical matches. Do not add HTTP-specific annotation workarounds.
4. Check prepared body wrappers consume the corrected record without selecting
   inherited clauses again. Current test-only wrapped candidates pass after
   source-edge retention; do not remove wrapper state mechanically without
   consumer evidence. Preserve normal source/explicit/streaming behavior.

No enum filtering, permissive default admission, global same-name ancestry
matching, transport-specific exceptions or public-policy changes are proposed.
No gRPC or JSON-RPC production expansion is justified by this witness; existing
shared mapping/presence controls cover their reliance on this constructor.

## Red and candidate evidence

- `go test ./http/codegen -run '^TestPreparedEnumProvenanceOriginal$' -count=1`
  fails with the exact original admission error; log
  `/tmp/loom-799-original-red.log`. No generated compile success is claimed.
- `go test ./expr -run '^TestPreparedEnumProvenance' -count=1 -v` records twelve
  constructor reds and fifteen healthy controls, then four representation
  comparisons and two named-edge candidate controls. All healthy assertions
  check declaration EnumBase, acceptance n=2, rejection n=1/n=3, and exact enum
  JSON `{n:1}`/`{n:2}`. The additional mapped-name projection red checks exact
  `{wire_n:1}` but receives `{}`. Log `/tmp/loom-799-constructors-red.log`.
- Lowering alone is an explicitly rejected candidate: its test expects the
  real declared-type error. A retained source edge preserves all predicates
  and provenance; a simpler named-edge retention additionally accommodates
  pre-finalization DSL construction. These remain nonproduction probes.
- Seven bounded model configurations reproduce the current copy, first-local
  and full-name failures, reject lowering-only, borrowed-state and wrong-default
  candidates, and check the retained design's six invariants over 192 states.
  Sources/run instructions are in `expr/tla/prepared_enum_provenance/README.md`.

## Bounded implementation acceptance

- Direct mapped/remapped/duplicated construction must retain enum declaration
  and full membership through direct composed types, aliases and local Required.
  Test normal mappings with unfinished fields, inherited requiredness, remapped
  suffixes, nullable objects and detached mutable copies using existing tests.
- Keep source, explicit request/response and streaming constructor controls.
  Check inherited enum constants under mapping names in the shared projection;
  reject unrelated and ambiguous member bindings. Verify retained constants,
  schema matching and final default admission separately.
- Generate the original implicit request and a representative implicit response
  from the same enum/refinement design. Exercise n=2 and reject n=1/n=3 through
  mounted request validation, client response validation where generated, and
  direct generated validators. Include an explicit-body or streaming control
  and a mapping-name case; compile/vet complete generated modules and inspect
  generated validation. Do not claim a service's invalid result is rejected by
  a server unless its emitted response path actually validates it.
- Required schema enum annotations must remain `{n:1}|{n:2}` conjoined with
  minimum 2, or the corresponding mapped wire member. Default n=2 is valid;
  defaults n=1/n=3 are rejected by the complete final contract. Render and parse
  representative specs with libopenapi and pinned Vacuum.
- Compare a bounded original-red, response, mapped-name and healthy explicit/
  streaming selection at the pushed parent and candidate, with separate-process
  repeat. List every changed artifact. Pin a verified pushed Loom revision in
  each external module and explicitly select this worktree for candidate code.
  Reuse unaffected evidence; no full exported-design corpus is an acceptance gate.
- After focused gates, freeze the exact patch and manifest for independent
  final review. Root owns main integration, atomic commit, push and issue closure.

Readiness artifacts contain tests, the model and this plan only. Production
implementation is still awaiting the root's release after readiness review.


## Implementation checkpoint

Independent readiness review approved manifest `de2af67eab3b0541a051e9cbe685dcabb9481b66237fbf65d07d1858bd71f1d5`
without findings; root released implementation. Production changes retain declaration edges in mapped flattening and use
controlled ancestor-enum logical member matching. Acceptance exposed the shared
OpenAPI enum consumer boundary described below; it now consumes the same checked
annotation owner for emission and schema identity. Final independent review is
pending.

## Implementation and final review handoff

- [x] Replace inherited public-carrier selection with a detached retained named
  declaration edge in `Remap`; keep local constraints and incomplete DSL assembly.
- [x] Bind captured ancestor enum members by logical identity, preserving target
  wire names, ambiguity rejection and ordinary ancestry checks.
- [x] Verify final default admission and detached copies; all 27 constructor
  cases, mapped projection and bounded negative controls pass.
- [x] Correct the affected shared OpenAPI consumer boundary: apply checked enum
  annotations even when local `Validation` is nil. Schema deduplication supplies
  those same checked projections to the canonical encoder; raw structural hashes
  used by example seeds remain structural and do not perform admission.
- [x] Full `expr` and OpenAPI IR packages pass. Actual component collision tests
  distinguish retained enums both with no local validation and local Required.
- [x] Original design builds. Implicit, mapped and explicit generated modules
  pass mounted request and client response n=1/2/3 checks, full build and vet.
  Requests assert status, problem code/detail, service invocation and payload;
  responses assert invalid values rejected and n=2 decoded correctly.
- [x] JSON/YAML specs parse through libopenapi and pass pinned Vacuum, including
  the existing enum duplicate/large-number rejection controls. Schema assertions
  retain both declaration constants conjoined with the final minimum.
- [x] Bounded CLI generation and example scaffolding at parent/candidate, plus
  separate-process candidate repeat. All candidate output bytes repeat exactly.
- [x] Independent final review of the frozen exact patch.
- [x] Root integration, atomic commit/push, issue reconciliation and scratch cleanup.

### Additional consumer evidence

The original readiness scope exposed a downstream annotation consumer mismatch:
`applySchemaValidation` returned on nil local `Validation` before consuming the
already prepared enum clauses. The structural fingerprint made the same local
carrier assumption and conflated different inherited enums. A direct IR red
(`/tmp/loom-799-ir-red.log`) proved both missing-local and local-Required identity
collisions before the correction. The implementation does not select or filter
enums in IR; `PrepareJSONSchemaAnnotations` remains the common owner. Raw invalid
structural utility inputs remain supported by the separate raw encoder path.
This is a required prepared-contract consumer repair, not a new enum policy.

### Exact bounded output comparison

Inputs: `PreparedEnumProvenanceDSL`, `PreparedEnumMappedDSL`,
`PreparedEnumExplicitDSL`, and existing `EffectiveCompositionDSL`. The first two
include both request and response plus a named alias. The last specimen retains
ordinary composed validations across HTTP/gRPC. External modules pin verified
pushed `v1.10.0-alpha.9.0.20261012232630-0dfab0ccd366`; every run explicitly replaces
that source with the intended parent or candidate and sets `LOOM_DIR`.

Parent: `0dfab0ccd366d176d2f9f3c1294d0506d1b7a35b`. Comparison source lives in the
managed `prepared-enum-parent` worktree. Go 1.27.2 darwin/arm64. All three runs use
identical module, design and command arguments (`loom gen` / `loom example`).

Intended differences:

1. Implicit and mapped parent specimens fail design admission for a base enum
   member incorrectly attributed to ExchangeRequestBody/ExchangeResponseBody.
   Candidate generates 21 files per specimen and builds/vets the complete module.
2. Explicit-body Go/scaffold/manifest bytes are unchanged. Only OpenAPI JSON/YAML
   add the inherited enum clause beside the EnumAlias reference. The reference
   target already contains that enum; the additional clause preserves the same
   contract and is now consumed consistently from the shared annotation owner.
3. Ordinary EffectiveComposition generated Go, protobuf, scaffold and OpenAPI
   bytes are unchanged. Only `gen/loom.json` changes its design digest because the
   prepared expression graph no longer carries inherited predicates as local
   declarations. The digest remains deterministic; its package tests pass.
4. Separate-process candidate/repeat comparisons have zero changed files for
   all four specimens. No output normalization was used. An initial probe used
   different CLI basenames and was corrected by rerunning generation with equal
   argv; the final comparison is the regenerated-results manifest below.

Evidence (temporary, retained through independent review):

- `/tmp/loom-799-owner-packages.log`: full expr and IR packages.
- `/tmp/loom-799-ir-identity.log`: component identity and emitted enum checks.
- `/tmp/loom-799-generated.log`: original admission/build and generated matrix.
- `/tmp/loom-799-wire-final.log`: final exact problem-detail/wire assertions,
  generated builds/vet and required schema assertions.
- `/tmp/loom-799-spec.log`: libopenapi and pinned Vacuum JSON/YAML plus controls.
- `/tmp/loom-799-http-generated-controls.log`: existing alias body generated
  integration with race/vet and body-name OpenAPI control.
- `/tmp/loom-799-design-fingerprint.log`: design-fingerprint package.
- `/tmp/loom-799-compare.py`, `/tmp/loom-799-regenerate.py`: bounded CLI harnesses.
- `/tmp/loom-799-comparison/results.json`: parent/candidate full build/vet phases.
- `/tmp/loom-799-comparison/regenerated-results.json`: final exact output hashes.
- `/tmp/loom-799-comparison/comparison.json`: exhaustive changed paths within
  the selected four specimens, and zero differences on isolated repeat.

One unrelated baseline control remains red: `TestAliasRequestBodyValidation`
expects validators without the previously delivered `validationContext` argument.
Eight string/int/array/map assertions fail identically on parent0df and candidate;
union controls pass. This is a stale assertion from #709, not changed generated
behavior. Exact parent reproduction is
`go test ./http/codegen -run '^TestAliasRequestBodyValidation$' -count=1` in the
managed parent worktree, with logs `/tmp/loom-799-parent-alias-control.log` and
`/tmp/loom-799-http-controls.log`. Its generated integration passes. No unrelated
assertions were weakened or modified in this ticket.

Source overlap with main `a9285e5ec83cbfeabd1f6c5414645e101474c91a`: among this
candidate's tracked files, only `docs/dsl-reference.md` changed since the parent.
The new paragraph must be applied without replacing main's numeric documentation.
No production owner file overlaps those delivered changes.

## Main integration sequence

1. Preserve the independently reviewed production/test/model files and the
   current numeric guide. Apply the analyzer annotation change at its newly split
   `schema_annotations.go` owner, retaining #808's source-size correction.
2. Update framework/consumer guidance and the formal model index.
3. Rerun affected owner and generated enum/spec checks after integration, compare
   moved helper bodies with the reviewed candidate, and run scoped lint/size gates.
4. Obtain independent review of this exact integrated diff before one atomic
   commit, push and issue closure. Reuse unchanged four-specimen comparison
   receipts; no full exported-design audit is required.

Original 23-file semantic implementation independently approved by
`/root/review_uniform_json_corrections`, manifest
`3eaf3e3cc4574eb906ee0626505982d51a6369b673a425314800c382d0fee8ad`.

Integration against `55cd4ca77` preserves all reviewed implementation bodies,
including the annotation helper now in `schema_annotations.go`; the numeric
`uint` guide remains intact. Full expr/IR suites pass (0.855s/6.956s), generated
HTTP enum matrix passes (12.035s), and OpenAPI JSON/YAML parsing plus pinned Vacuum
checks pass (21.666s). File-size and scoped expr/IR lint pass with zero lint
issues. Logs: `/tmp/loom-799-integrated-{owner,generated,spec,size,lint}.log`.
The reviewed four-specimen generated comparison remains applicable: the only
integration code movement is byte-identical and #808 separately proved unchanged
ordinary generated output. Final integrated review passed without findings.

## Delivery

Delivered as `7326becd4`, pushed to main and #799 confirmed closed. Exact
27-file integration independently approved by `/root/review_uniform_json_corrections`:
manifest `df28a219e8f4043e706b50a3b187f1f42b235c27beaf8351565c7e1739adaf88`,
patch `b7128239b46a4ad5f2b4abf59e3b2f809925862b1dfdb09810976c2a41bb38d3`.
Temporary-output cleanup is complete: combined #799/#804/#808 receipt
`/tmp/loom-delivered-799-804-808-cleanup-receipt.json`, SHA
`a307fbf7b3c17e74fd5fab00db8cef67ea5df9fa45e11861364811052fb829b7`.
Comparison JSONs and copied command logs remain; generated modules and checker
state are removed. #804 raw logs are condensed into
`/tmp/loom-804-validation-summary.json`; review/integration manifests and patches
remain. Managed worktrees remain because archival was refused.
