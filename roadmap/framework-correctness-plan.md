# Framework correctness recovery plan

Implement the architectural direction accepted in the Codex discussion on 2026-10-12. Repair shared contracts and ownership boundaries; use review findings as regression witnesses. Authentication is resolved before transport execution, and results are recursively projected before transport encoding. This file is the recovery entry point and progress source of truth.

## Current state

- Delivered and pushed: authentication scope #724 (#725 witness), public error disclosure #713, recursive projection #788 (#730/#779 witnesses), artifact write intent #670, and example interceptor ownership #741.
- Implementing: WebSocket message limits #716 and effective Extend composition #755 (#757 witness). Proxy-address trust #708 is implementing in its own worktree after independent readiness approval. Worktrees and owners are recorded below; these changes are not yet delivered.
- Next stream task: #722 connection lifetime after #716. The user approved closing and canceling on configurable buffer overload.
- Later accepted semantics: explicit empty collections survive encoding, and defaults fill absence without replacing explicit zero. Milestone 7 implementation remains outstanding.
- Recovery checkpoint after #670 delivery: `6183551e`. Inspect the latest Git log, uncommitted tracker updates and live agents before resuming.

## Delivery log

- 2026-10-12 — Baseline: `4535a403048510f055fc59ec3dfed41e3769ad58` on `main`; initial checkout clean. Plan drafted; independent readiness review pending. No implementation is complete.
- 2026-10-12 — Root orchestrates and owns this tracker. Authentication and projection agents may inventory isolated worktrees while readiness review runs. Implementation starts only after their milestone is ready.

- 2026-10-12 — Auth owner: agent `/root/auth`, worktree `/Users/luca/.codex/worktrees/auth-contract/loom`. Projection owner: agent `/root/projection`, worktree `/Users/luca/.codex/worktrees/recursive-projection/loom`. Both are inventory-only pending slice readiness. Root recited and adopted the test-first, model-first, bounded-evidence, independent-review and atomic-delivery workflow.
- 2026-10-12 — Inventory contains all 114 open reports in #679–#793 plus #670. Independent reviewer requested exact per-slice readiness records and full unary/stream error-disclosure inventory; corrections are being applied.

- 2026-10-12 — Planning artifacts independently reviewed by `/root/plan_review`, committed as `eb5a169b`, and pushed to `origin/main`. Auth readiness: `/Users/luca/.codex/worktrees/auth-contract/loom/roadmap/auth-contract-readiness.md`; projection readiness: `/Users/luca/.codex/worktrees/recursive-projection/loom/roadmap/recursive-projection-readiness.md`. Both await final scoped readiness approval.

- 2026-10-12 — Authentication #724 readiness approved independently; implementation released. #725 is its declaration-resolution witness. Decoder alternatives, constraints and stream admission remain separate tasks. Pristine baseline worktree: `/Users/luca/.codex/worktrees/correctness-baseline/loom`; do not mutate it. Error-disclosure preparation assigned to `/root/plan_review` in `/Users/luca/.codex/worktrees/safe-error-disclosure/loom`.

- 2026-10-12 — Projection #788 readiness approved independently and implementation released. Auth baseline tests and two legacy TLA controls reproduced #724/#725; corrected model checks 2048 states and direct expr/security/service suites pass during implementation. Final generated evidence and exact-diff review are still pending. Public decisions on Extend redeclaration and collection presence were requested asynchronously; dependent changes remain gated.

- 2026-10-12 — Error disclosure #713 readiness independently approved by root; implementation released. Legacy public-message model and direct tests reproduce leaks; corrected model and direct regressions pass during implementation. Generated unary/SSE/WebSocket evidence and final review remain pending. Auth generated security checks and bounded output/determinism comparisons pass; its final evidence record is being prepared.

- 2026-10-12 — **Delivered #724 / #725 witness:** commit `544c57f3` on parent `eb5a169b`, pushed to `origin/main`. `/root/review_auth` approved the exact 19-file integrated diff (SHA-256 `c40b9ae4c7279e9013b35811702995c441a6cfe28ce787dcccace4e89f79571e`) with no findings. Reviewer independently reran expr/security/service, focused three-transport security tests, both failing legacy models and the passing 2,048-state model. Paths, failing baseline, intended generated differences and commands are recorded in `roadmap/auth-contract-readiness.md` and `expr/tla/authentication_scope/README.md`. Regenerate consuming adapters. Remaining authentication tasks stay open.

- 2026-10-12 — Projection #788 passes generated HTTP/gRPC wire checks during implementation. #784 reproduces after the shared projection fix: HTTP client/server union declaration registries need separate ownership; it is excluded from #788, as is SSE body selection #727. Error #713 passes direct runtime and generated gRPC/HTTP/JSON-RPC unary/SSE/WebSocket regressions; bounded baseline/output comparisons remain in progress.

- 2026-10-12 — Priority refinement: promote P1 file-overwrite #670 from residual reconciliation to the next integration slot after the first wave, before Milestone 5. Agent `/root/artifact_ownership` owns readiness preparation in `/Users/luca/.codex/worktrees/artifact-write-intent/loom` at `eb5a169b`. The scope must preserve write intent through rendering and transactional installation; #741 is being reconciled as a consumer-policy witness, while cross-device #751 stays separate. No source implementation is released until readiness approval.

- 2026-10-12 — #724 validation scratch/model output was removed after review and delivery. Managed archive of `/Users/luca/.codex/worktrees/auth-contract/loom` was refused because the checkout is protected by a pinned task or workspace; it remains attached and is no longer needed for implementation. Durable evidence is committed.

- 2026-10-12 — Recovery update committed/pushed as `7d736078`, independently reviewed by `/root/review_auth`. #713 frozen 22-file candidate plus three root skill edits integrated/staged on that parent. Fresh reviewer `/root/review_errors` found one high-severity history disclosure: a nil/code-only outer remedy hides an intermediate SafeMessage from the separate history traversal. No commit authorized. Implementer `/root/plan_review` is repairing the common owner and adding the counterexample; re-review is required.

- 2026-10-12 — #670 readiness approved for external create-only artifacts and typed intent propagation. Root requested the public choice for create-only files inside wholly regenerated `gen/` (reject vs preserve); that dependent behavior remains gated. Agent may model/reproduce and implement independent external-file behavior. #741 is confirmed a separate declaration fix and will get its own atomic follow-up.

- 2026-10-12 — **Delivered #713:** commit `63e96099` on parent `7d736078`, pushed to `origin/main`. `/root/review_errors` confirmed the high-severity history finding fixed and authorized the exact 26-file diff (SHA-256 `182bd56beffe229363f32a882f173370f35da2e741b9e2a829e76a316a9b9974`). `pkg.ResolveErrorMessage` now supplies message and explicit-safe-override identity; gRPC history uses the same owner. Evidence/commands: `pkg/tla/error_disclosure/README.md`. Generated bytes unchanged; runtime upgrade suffices. Task temporary evidence was removed after review. Managed archive was refused as protected/pinned, so the completed error worktree remains attached.

- 2026-10-12 — **User accepted three public contracts explicitly:** Extend refines inherited constraints without weakening them; explicit empty collections remain distinct from absence; create-only declarations inside `gen/` are rejected and remain supported outside it. #670 agent notified; gen-tree rejection is unblocked. These are target contracts, not claims that later Extend/presence implementations have shipped.

- 2026-10-12 — #788 frozen 32-file candidate plus 3 shared skill edits integrated/staged on `63e96099`; `/root/review_projection` owns fresh final review. Six baseline wire probes fail and six candidate probes pass with `-race`; 30 bounded CLI generation/build/vet receipts pass, all 124 generated files match independent repeats. Exact differences and scope: `codegen/service/tla/recursive_projection/README.md`. No projection commit authorized yet.

- 2026-10-12 — Milestone 5 readiness assigned to `/root/websocket_lifetime` in `/Users/luca/.codex/worktrees/websocket-lifetime/loom` at `7d736078`. Source untouched pending root review. Existing runtime owner is incomplete; connection lifecycle, single-reader ownership and full-inbox behavior need explicit invariants.

- 2026-10-12 — **Delivered #788 / #730 and #779 witnesses:** commit `ce3f6a73` on parent `63e96099`, pushed to `origin/main`. `/root/review_projection` approved the exact 35-file diff (SHA-256 `e5843f8923c78f3f61bc284e3fb4482b25491b0b1eb476082376e4acac06fd67`) after a low stale-skill reference was corrected and re-reviewed. Reviewer reran full expr/codegen/service and HTTP projection/presence tests, verified 124 byte-identical repeats and 30 passing receipts. Regenerate adapters. #784 and #727 remain separate unresolved tasks.

- 2026-10-12 — WebSocket readiness approved for models/regressions only. Root requested the public overload contract (explicit close/cancel at configurable buffer bound, recommended, versus backpressure with delayed disconnect detection). #716 is a separately scoped proposed prerequisite using the existing HTTP 32 MiB individual-message bound and configurable policy; no lifecycle source edits until contract/dependency readiness is settled.

- 2026-10-12 — Recovery checkpoint committed and pushed as `096a793b`, independently reviewed by `/root/review_auth`. Completed projection scratch outputs were removed; its managed worktree could not be archived because it is protected by a pinned task or workspace.

- 2026-10-12 — #670 candidate integrated and staged on `096a793b`. `/root/review_artifacts` found a P1 in the exact 28-file diff: a stable live symlink such as `alias -> gen` bypasses lexical create-only admission, so replacing `gen/` can destroy the supposedly protected file. **No commit authorized.** `/root/artifact_ownership` is repairing shared live-path admission and adding the counterexample in its existing worktree. Re-review is required; root must preserve its integrated shared documentation deltas when copying repairs.

- 2026-10-12 — Milestone 6 readiness assigned to `/root/effective_design` in `/Users/luca/.codex/worktrees/effective-design-validation/loom` at `ce3f6a73`. Proposed owner #755 with #757 as a shared composition-order witness. Existing effective-constraint analysis must own refinement; multiple-base precedence and phase dependencies require concrete readiness review. No implementation released.

- 2026-10-12 — #716 prerequisite exploration now covers the decompressed message size as well as Gorilla's framing limit. Root accepted a configurable, symmetric 32 MiB default concept for HTTP/JSON-RPC clients and servers, with protocol close 1009 for oversized messages. Exact attachment/configuration API readiness remains pending before production edits. #722 overload semantics still await the user's answer; legacy models and direct/generated red regressions are available in the WebSocket worktree.

- 2026-10-12 — #716 concrete API approved: immutable `http.WebSocketConfig` owns the message cap and existing configuration hook; an unattached `WebSocketStream` configures and adopts the socket through one `SetConn` path. HTTP per-endpoint configuration and JSON-RPC configuration pass the same value. Context-carried limits were rejected during design because JSON-RPC shares connections across calls. Agent may implement this separate prerequisite after red tests/models, updating its readiness record first. #722 pump/overload behavior remains gated.

- 2026-10-12 — #755/#757 readiness independently reviewed by root; models and failing regressions released. Explicit nullable overlay provenance and identity/codec metadata classifications must be recorded before production changes. #741 follow-up readiness is drafted in `roadmap/example-interceptor-ownership-readiness.md`; root reproduced both server/client overwrites and cwd-only suppression with an overlay probe. That follow-up still awaits independent readiness approval and a post-#670 parent.

- 2026-10-12 — #755/#757 production implementation released after root reviewed the completed nullable/metadata readiness delta and model correspondence. Ten red cases reproduce the intended defects; three preservation controls pass. Four model controls fail as expected and the corrected bounded model passes five invariants across 246 states. These are design/reproduction results, not final implementation acceptance. Agent `/root/effective_design` continues in its recorded worktree.

- 2026-10-12 — **User accepted the WebSocket overload contract:** close the connection and cancel its context when the configurable buffer limit is exceeded. Agent `/root/websocket_lifetime` notified. #716 remains the separate message-size prerequisite; #722 follows with bounded raw-message accounting and explicit overload termination. The previous pending public-policy question is resolved; implementation and final validation are still outstanding.

- 2026-10-12 — #670 alias repair integrated and staged on `096a793b`; `/root/review_artifacts` owns re-review of the 31-file final diff (SHA-256 `5ae6031981d106005daa9579ae499a3e77212e99dde68ebff2b4f1349b2e78b6`). Original reproduction now passes; live parent resolution, absent descendants, filesystem identity and reserved case variants are covered. Agent validation and refined model pass, but commit remains gated on re-review. #741 readiness is being reviewed separately and is excluded from this staged candidate.

- 2026-10-12 — **Delivered #670:** commit `41b4a8a3` on parent `096a793b`, pushed to `origin/main`. `/root/review_artifacts` confirmed the P1 alias finding fixed and authorized the exact 31-file diff above with no unresolved findings. Independent original-reproduction, policy/renderer/generator/CLI, real-plugin, vet and whitespace checks pass; filesystem-identity tests ran without skips. Durable evidence: `cmd/loom/tla/artifact_write_intent/README.md`. Update the CLI and design-module dependency together. #741 and cross-device #751 remain separate.

- 2026-10-12 — #741 readiness independently approved by `/root/review_artifacts`. Follow-up worktree created at `41b4a8a3`: `/Users/luca/.codex/worktrees/example-interceptor-ownership/loom`. Its declaration-only correction reuses #670's ownership machinery; source implementation is next after this recovery checkpoint. Root retains the red overlay probe until its regression is incorporated.

- 2026-10-12 — Recovery checkpoint independently reviewed, committed and pushed as `6183551e`. Agent `/root/artifact_ownership` is implementing #741 in the new worktree above; the old #670 worktree is frozen. Completed #670 comparison, checker, integration and reviewer scratch directories were removed after checking active processes. Managed archival was refused because the old worktree is protected by a pinned task or workspace; its source remains attached.

- 2026-10-12 — #741 is frozen: direct/service tests and unchanged goldens pass; 26 generated/example files match its immediate parent and an independent repeat; real CLI probes preserve edited files and no longer suppress fresh output because of cwd files. Validation exposed a separate inherited JSON v1 lint violation in the recursive projection wire-test harness. Root reproduced the lint failure, switched only that harness to JSON v2 with explicit envelope tags and raw-value retention, and reran lint plus both generated wire cases successfully. `/root/review_artifacts` is reviewing that one-file maintenance change before #741 integration. #708 proxy-trust readiness is separately drafted for review; source is untouched.

- 2026-10-12 — Test-maintenance correction independently reviewed with no findings and delivered as `93f513be`, pushed to `origin/main`. `/root/review_artifacts` approved exact one-file diff `c057e2e3727be1c85cc3d640161625d6f6e1e491e68a23be646cc9645b5a72a2`; JSON v2 lint and both recursive projection wire cases pass. No production behavior or generated output changed.

- 2026-10-12 — **Delivered #741:** commit `23f2af5d` on parent `93f513be`, pushed to `origin/main`. `/root/review_artifacts` approved the exact four-file diff (SHA-256 `2ce00171da55e3602d5f42a029ad2dced8c0d8c0b4bccee1997a3500e1b641db`) with no findings, independently checked regressions and all 26 output comparisons/CLI receipts. Evidence: `roadmap/example-interceptor-ownership-readiness.md`. Example interceptor declarations now use the shared create-only writer. #708 readiness was also independently approved; its implementation is the next available assignment.

- 2026-10-12 — #708 implementation assigned to `/root/forwarded_client` in `/Users/luca/.codex/worktrees/forwarded-client-trust/loom`, parent `23f2af5d2e4ad9c9b659309839af4f3b25b3f8e3`. Independently approved readiness is copied into that worktree. The shared request-metadata owner must stop at invalid forwarding hops; test/model-first implementation is released. No generator sweep is required for this handwritten runtime correction.

- 2026-10-12 — Completed #741 comparison and readiness-probe scratch was removed after independent review, delivery and an active-process check. Managed archive of `/Users/luca/.codex/worktrees/example-interceptor-ownership/loom` was refused as protected by a pinned task or workspace; its source stays attached. Durable reproduction and results remain in the committed readiness record.

## Recovery and delivery

Read this entire file, repository `AGENTS.md`, `.agents/skills/loom-framework/SKILL.md`, and the applicable model README before resuming. Inspect `git status`, `git log`, attached managed worktrees, and agent status before assigning work. Never assume a running agent or uncommitted diff is abandoned. Do not use git clean, stash, reset, or checkout.

Milestones are ordered by priority and integration dependency. Milestones 2 and 3 may be implemented concurrently in isolated managed worktrees; integration remains sequential. Milestone 4 can start when a slot becomes available. Later milestones require a bounded readiness review of their concrete inventory and acceptance matrix before implementation. Do not turn the whole issue list into an undifferentiated rewrite.

One structural task has one owning ticket and one atomic commit. Before implementing a family, root selects one existing owning structural ticket and records related reports as regression witnesses in a readiness record; do not combine unrelated tickets in one commit. Record the ticket, parent SHA, worktree, changed paths, failing baseline, passing commands, intended generated-output differences, review verdict, commit SHA and push status in this file's Status section. Findings are not complete merely because code is written. A checked delivery item means reviewed, committed, pushed, and integrated.

Before any implementation edit, the task owner writes a readiness record under `roadmap/` naming the owning ticket, intended parent, included and excluded reports, exact file/test/model loci and checker commands, accepted/rejected fixture cases, baseline/current byte-comparison method, compile commands and intended output differences. An independent reviewer must approve that concrete slice. The broad milestone checklist alone never releases implementation. Record deeper public-policy blockers separately and continue independent ready tasks.

Before every commit, an agent that did not implement the change reviews the exact final diff against its intended parent. State the acceptance rule and run its checks first. Resolve every finding according to AGENTS.md, obtain re-review, and make no changes between authorization and commit. Integration that changes a reviewed diff requires re-review. Root alone updates this tracker and `roadmap/ROADMAP.md`; implementation agents hand back documentation deltas affecting shared skills for coordinated integration.

Use direct regressions and affected packages, then bounded generated comparisons against the baseline: compare exact bytes, list intended differences, compile generated Go, and repeat generation in independent processes for determinism. Use existing harnesses and representative fixtures; the full exported-design corpus is not a task gate. Set `LOOM_DIR` to the implementing worktree for local temp-module tests. Preserve model sources and summarized evidence; remove task-owned temporary output only after review and after checking active users.

## Accepted contracts and decision boundaries

- Nearest scope declaring `Security`, `SessionSecurity`, or `NoSecurity` owns the complete authentication declaration. Same-scope alternatives retain documented OR/AND composition; contradictory anonymous-only and authenticated declarations fail validation.
- Projection recursively preserves only the selected view through objects, collections, unions, and supported recursive references. Excluded-field changes cannot alter serialized output. Transport encoders consume this projection.
- Unknown service errors disclose a generic public message while retaining their cause for server-side diagnosis. Designed public errors and native protocol termination keep their contracts.
- Defaults fill absence and never replace explicit zero. Presence is independent of value. Explicit empty scalars survive encoding. Explicitly present empty arrays/maps must survive as empty values, distinct from absence. This supersedes the old nil/empty policy when Milestone 7 is implemented; model presence explicitly rather than infer it from length.
- Runtime owners control resource lifetime; generators provide typed adapters. Worker publication must not expose an unready worker. Lost replies require reconciliation rather than assuming rollback.
- WebSocket buffer overload is terminal: close the connection and cancel its context when its configured buffer budget is exceeded. Do not silently discard a message and continue. Bound raw-message storage explicitly and document memory outside that bound; the independently configured per-message limit remains a separate check.
- Local scheduler cancellation is provisionally intended to stop local participation, not delete the global schedule. Consumer evidence and a recorded contract decision are required before changing that behavior.
- `Extend` same-type redeclarations refine inherited constraints without weakening guarantees. For example, an inherited minimum 0 plus derived maximum 100 admits 0–100; incompatible types, contradictory constraints, or weakening inherited requiredness fail design validation. Preserve authored intent and document the exact composition of defaults/metadata in slice readiness.
- `gen/` is wholly framework-owned and regenerated. Reject create-only (`SkipExist`) declarations inside it before live mutation. Outside it, create-only files preserve any existing destination and create an absent destination atomically.

## Milestones

### Milestone 1: Durable inventory and execution readiness

Goal: Establish recoverable ownership and a reviewed first implementation wave.

Acceptance Criteria

- An independent reviewer approves this plan's first-wave milestones against current code; later work remains explicitly gated on its scoped readiness review.
- Every open code-review finding #679–#793, excluding closed duplicate #775, has a recorded family or residual disposition in `roadmap/framework-correctness-inventory.md`. Include referenced older #670. Dispositions do not assert unverified fixes.
- Worktree paths and owners are recorded in Status before implementation.

Checklist

- [x] Read this plan and the framework skill; recite milestone order, exit criteria, named commands, test-first procedure, independent-review gate, commit/push handoff, and repository constraints before editing implementation.
- [x] Fetch reports with `gh issue list --repo CaliLuke/loom --state open --label code-review --limit 200 --json number,title,body,labels,url`; reconcile their owners in `roadmap/framework-correctness-inventory.md`.
- [x] Review first-wave loci with `rg -n 'EffectiveRequirements|inheritedRequirements|inheritedSessionAuths|buildConstructorCode|ErrorSafeMessage|newErrorResponse' expr codegen security http grpc jsonrpc pkg` and map every relevant consumer to acceptance coverage.
- [x] Obtain independent readiness and exact-diff review of the planning artifact; run `git diff --check`; commit and push the initial recovery plan.

### Milestone 2: One effective authentication contract

Goal: Resolve authentication once and preserve it across extraction, validation and execution.

Acceptance Criteria

- `expr/method_test.go` and security tests prove API/service/method overrides in both session/ordinary directions, same-scope alternatives, anonymous alternatives, and rejection of contradictory NoSecurity declarations.
- Generated HTTP, gRPC and JSON-RPC security tests prove usable Basic/Bearer alternatives, credential constraints and matching effective OpenAPI requirements. Metadata-only authentication runs before blocking gRPC reads; message-carried credentials retain explicit ordering constraints.
- Models reproduce the old scope/ordering failures and check the corrected invariants; exact baseline/output comparisons cover selected security fixtures and compile emitted adapters.

Checklist

- [x] Deliver #724 nearest-scope ownership with #725 contradiction witness; independent review, commit `544c57f3`, push and integration complete. Evidence: `expr/tla/authentication_scope/README.md`.
- [ ] Inventory `rg -n 'EffectiveRequirements|SessionAuths|Requirements|DecodeBearer|BasicAuth|writeGRPCStreamInputDecode' expr security codegen/service http/codegen grpc/codegen jsonrpc/codegen`; reconcile #724, #725, #729, #731, #732, #743, #752 and #728 by exact cause, splitting distinct structural tasks before implementation.
- [ ] Add failing direct regressions at `expr/method_test.go` and relevant transport security tests; record the failing commands and scope-selection model before implementation.
- [x] Replace independent security inheritance in `expr/method.go` with one effective owner; derive consumer behavior from it and delete superseded selection paths.
- [ ] Run `go test ./expr ./security ./codegen/service -count=1`; run bounded security tests in `./http/codegen`, `./grpc/codegen`, `./jsonrpc/codegen`, recording exact filters and generated fixtures.
- [ ] Complete credential-alternative and stream-admission tasks only after their own test-first, model, generated comparison and independent-review obligations are ready; keep payload-dependent authorization at its valid boundary.
- [ ] Update `docs/dsl-reference.md`, relevant authentication guides and consumer skill; obtain exact-diff review, commit each owning structural ticket atomically, push and integrate.

### Milestone 3: Recursive view projection

Goal: Make selected views authoritative for every nested output shape and transport.

Acceptance Criteria

- Direct service conversion tests prove arrays, maps, unions, adjacent result fields and recursive supported objects use the same selected-view semantics in both conversion directions.
- Generated HTTP, JSON-RPC and gRPC tests prove excluded secrets never appear, including streaming results; declaration order cannot change projection or produce inconsistent body declarations.
- Selected normal and adversarial fixtures generate deterministically and compile; excluded-field perturbation leaves wire output unchanged.

Checklist

- [x] Deliver #788 recursive selection and conversion ownership with #730/#779 witnesses; independent review, commit `ce3f6a73`, push and integration complete. Evidence: `codegen/service/tla/recursive_projection/README.md`.
- [ ] Inventory `rg -n 'buildConstructorCode|ViewScope|Project|viewedResultBodyInit|collectHTTPUnionTypes' codegen/service http/codegen grpc/codegen jsonrpc/codegen`; reconcile #730, #779, #788, #784 and #727, recording distinct causes before grouping.
- [ ] Add failing direct tests beside `codegen/service/views_test.go` and generated wire regressions using existing view fixtures; record baseline failures and a projection ownership model before changing conversions.
- [ ] Replace direct-field special handling and generic semantic fallback in `codegen/service/service_data_views_init.go` with one recursive view-aware conversion owner; align view/body declaration collection with the selected projection.
- [ ] Run `go test ./codegen/service -count=1`; run bounded viewed-result tests in `./http/codegen`, `./grpc/codegen` and `./jsonrpc/codegen`, recording exact filters and fixtures.
- [ ] Compare selected baseline/current generated bytes, compile them, repeat generation in separate processes and record every intended difference in durable evidence beside the owner.
- [ ] Update `docs/code-generation.md` and affected consumer guidance; obtain exact-diff review, commit the structural ticket atomically, push and integrate.

### Milestone 4: Safe public error disclosure

Goal: Apply one public-disclosure policy while preserving each protocol's error classification.

Acceptance Criteria

- A plain internal error containing a sentinel secret never exposes that text in HTTP bodies, gRPC status/details or JSON-RPC envelopes; the server retains the cause.
- Designed errors, safe-message overrides, wrapped errors, joins, context cancellation and deadlines retain their explicitly tested classification and public contracts. Unary and stream paths are covered separately where their renderers differ. `grpc/error_contract.go` retains classification ownership; tree-wide errors.As must not select one joined branch as aggregate disclosure policy.

Checklist

- [x] Inventory `rg -n 'ErrorSafeMessage|newErrorResponse|EncodeServerError|InternalError|err.Error' pkg/error_remedy.go pkg/error* http/error* http/codegen/error* http/codegen/service_data_response_errors.go grpc/error.go grpc/error_contract.go grpc/error* jsonrpc/codegen/handler_sections.go jsonrpc/codegen/stream_sections_websocket.go`; reconcile #713 without replacing protocol classification with a generic status policy.
- [x] Add failing disclosure regressions to runtime error tests and generated transport tests; inspect existing error-ownership models before implementing the common public-message decision.
- [x] Implement the shared disclosure owner in `pkg` and make the three transport error boundaries consume it.
- [x] Run `go test ./pkg ./http ./grpc ./jsonrpc -count=1` and bounded generated error tests in `./http/codegen`, `./grpc/codegen`, `./jsonrpc/codegen`; record exact filters, baseline comparison and compile evidence.
- [x] Update `docs/error-handling.md`; obtain exact-diff review, commit #713 atomically, push and integrate.

### Milestone 5: Complete stream lifetime ownership

Goal: Make runtime owners authoritative for connection closure, cancellation and serialized operations.

Acceptance Criteria

- WebSocket close always releases the socket even when protocol writes fail; peer failure and handler exit terminate owned resources under the documented policy.
- Bounded message handling and disconnect detection have explicit backpressure and cancellation contracts; no second reader is introduced accidentally.
- gRPC send serialization and operation-context behavior are proven in their own protocol owner and compiled adapters.

Checklist

- [ ] Inventory `rg -n 'WriteClose|ReadMessage|SetReadLimit|WithContext|WebSocketStream|conn.Close' http jsonrpc grpc/codegen`; reconcile #711, #712, #714, #716, #722, #726, #758 and #739 into protocol-specific structural tickets.
- [ ] Record exact lifetime states, resource owners and bounded acceptance cases beside `http/websocket.go`; obtain scoped readiness review before implementation.
- [ ] Extend the owning TLA models with current-failure controls; run their README commands; add failing socket/race and generated-handler regressions.
- [ ] Move remaining generated lifecycle decisions into `http.WebSocketStream` and the appropriate JSON-RPC/gRPC runtime owners; delete duplicated protocol execution.
- [ ] Run `go test -race ./http ./jsonrpc ./grpc -count=1` plus bounded generated stream tests and compile comparisons; record selected cases and exact commands.
- [ ] Update stream guides and consumer skill; review, commit, push and integrate each structural ticket separately.

### Milestone 6: Validate the effective design

Goal: Ensure generators receive a fully resolved, validated semantic contract.

Acceptance Criteria

- Inherited field-number and mapped-name collisions fail as source-attributed design errors, before generator panics or silent overwrites.
- Valid effective shapes retain deterministic output; validation does not silently discard authored declarations or mutate shared bases.

Checklist

- [ ] Inventory `rg -n 'PrepareValidateFinalize|Finalize|Merge|validateRPCObjectTags|ElemName' eval expr`; reconcile #755, #757, #756, #701, #702, #703, #699, #710, #760, #780 and #781 by actual validation owner.
- [ ] Record phase dependencies and the accepted constraint-refinement contract for Extend redeclarations in the owning design documentation; obtain scoped readiness review before implementation.
- [ ] Add failing effective-shape tests and model existing phase-order counterexamples; move composition before the checks that require it without globally swapping unsafe evaluator phases.
- [ ] Run `go test ./eval ./expr ./dsl -count=1`, affected generator packages and bounded generated accepted/rejected fixtures; record byte differences and compilation results.
- [ ] Update DSL documentation; review, commit, push and integrate each owning structural ticket separately.

### Milestone 7: Exact values and explicit presence

Goal: Preserve authored numeric meaning and explicit values through shared conversion contracts.

Acceptance Criteria

- Bounds above 2^53 and integer extrema preserve their exact meaning; invalid defaults and enum members fail at design time; accepted collection literals compile using their declared types.
- Absent/default/zero/empty/null cases follow the accepted scalar contract across supported transports. Explicitly present empty collections survive as empty values, distinct from absence, under the accepted public decision.

Checklist

- [ ] Inventory `rg -n 'Minimum|Maximum|typedDefaultValueLiteral|UseDefault|omitempty|omitzero' expr codegen http/codegen grpc/codegen jsonrpc/codegen`; reconcile #681, #685, #692, #693, #695, #696, #690 and existing `roadmap/value-contract-inventory.md` obligations.
- [ ] Record the exact numeric domain and presence matrix beside the existing value-contract design; obtain scoped readiness review before implementation.
- [ ] Add failing exact-bound, target-literal and scalar presence regressions; update relevant Lean/TLA source and correspondence before replacing lossy representations.
- [ ] Run `go test ./expr ./codegen ./codegen/service -count=1` and selected transport presence/default tests; run the affected proof README commands and bounded generated comparisons.
- [ ] Document regeneration and scalar contract changes; review, commit, push and integrate each structural ticket separately.

### Milestone 8: Generated identities share real scopes

Goal: Allocate generated declarations and references together without hiding authored identity.

Acceptance Criteria

- Adversarial authored names cannot collide with framework declarations, helper names or imports; references use the allocated declaration identity.
- Declaration ordering and existing numbered suffixes cannot change correctness; supported names compile across the affected generators.

Checklist

- [ ] Inventory `rg -n 'NameScope|Unique|Goify|TypeName|ImportName' internal/naming codegen/service http/codegen grpc/codegen internal/openapiimport`; reconcile #691, #694, #700, #733, #734, #735, #736, #770, #786 and #688 to real output scopes.
- [ ] Record each scope's declaration/reference owner and collision fixture; obtain readiness review and add failing tests plus bounded allocation-model controls.
- [ ] Route each selected scope through its shared allocator; remove recomputed names and suffix-based references.
- [ ] Run `go test ./internal/naming ./codegen ./codegen/service -count=1` and scoped generator/importer tests; compare and compile adversarial generated fixtures across independent generations.
- [ ] Review, commit, push and integrate each structural ticket separately.

### Milestone 9: Shared parameter wire contracts

Goal: Make runtime codecs and OpenAPI admission/export agree on accepted parameter serialization.

Acceptance Criteria

- Accepted location/style/explode/shape combinations round-trip through import, generation and real request decoding without changed meaning.
- Unsupported combinations fail with source-specific diagnostics; recursive component imports compile and preserve graph references under a separately scoped importer task.

Checklist

- [ ] Inventory `rg -n 'deepObject|supportedParameterStyle|Explode|Style|Encoding' internal/openapiimport http/codegen http`; reconcile #706, #719, #793, #791 and separate importer recursion #792.
- [ ] Record the shared serialization capability table and recursion rendering scope; obtain readiness review and add failing import/export/runtime fixtures.
- [ ] Implement the common parameter contract and make import, export and codecs derive from it; implement recursive reference rendering only under its own reviewed task.
- [ ] Run `go test ./internal/openapiimport ./http/codegen/openapi/... -count=1`, selected HTTP codec tests, `make openapi-contract`, and bounded import-to-generated compile/semantic round trips.
- [ ] Update OpenAPI/HTTP guides; review, commit, push and integrate each structural ticket separately.

### Milestone 10: Transactional resource construction and publication

Goal: Prevent failed construction from leaking resources or publishing unusable distributed participants.

Acceptance Criteria

- Every injected local acquisition failure releases resources once; retries do not accumulate subscriptions or goroutines.
- Only ready workers receive jobs; lost publication replies, retries and cleanup converge under explicit ownership rules on real Redis.
- Scheduler cancellation and job bookkeeping match the recorded local/global lifetime decision.

Checklist

- [ ] Inventory `rg -n 'AddNode|newWorker|SetAndWait|initialWorkerHeartbeat|Schedule|ErrJobExists' pulse/pool`; reconcile #687, #704, #738 and #737, keeping observability constructor rollback #763 as a separate owner.
- [ ] Record acquisition/publication states and scheduler consumer evidence beside owning code; obtain readiness review before changing lifecycle contracts.
- [ ] Model failure and lost-reply counterexamples; add table-driven fault injection and real-Redis regressions before implementing local cleanup and distributed publication changes.
- [ ] Run `go test -race ./pulse/pool ./pulse/rmap ./pulse/streaming -count=1`, affected model README commands and `make test-pulse-redis`; distinguish miniredis evidence from real Redis behavior.
- [ ] Update Pulse guidance; review, commit, push and integrate each structural ticket separately.

### Milestone 11: Residual findings and final reconciliation

Goal: Leave every report with an evidence-backed disposition and no hidden unfinished architectural work.

Acceptance Criteria

- The inventory distinguishes delivered fixes, reproduced residual defects, duplicates, unsupported cases and public decisions; no issue is closed solely by similarity to a repaired family.
- File overwrite safety, proxy-address trust, logging/debug wrappers and uncovered protocol paths have explicit owners and bounded next actions.
- Integrated validation passes and temporary worktrees/output are cleaned only after all active users finish.

Checklist

- [x] Deliver promoted #670 artifact write-intent ownership; independent review, commit `41b4a8a3`, push and integration complete. Evidence: `cmd/loom/tla/artifact_write_intent/README.md`.
- [x] Deliver #741 example interceptor declarations through that shared writer; independent review, commit `23f2af5d`, push and integration complete. Evidence: `roadmap/example-interceptor-ownership-readiness.md`.
- [ ] Reconcile inventory residuals, including #670/#741 file ownership and #708 proxy trust, against final integrated source; promote each supported defect to a separately reviewed bounded task.
- [ ] Run targeted adversarial review of repaired owners and their consumers; record new counterexamples and update task acceptance before implementation.
- [ ] Run `make lint`, `make test` and integration checks selected by actual changed behavior; record exact source and outstanding blockers. Do not start a full exported-design audit without a named purpose and cost budget.
- [ ] Confirm every delivery commit was independently reviewed and pushed; update `roadmap/ROADMAP.md` and this tracker to reflect remaining active work.
- [ ] Preserve durable evidence and archive completed managed worktrees; remove task-owned model/compiler output after checking active processes.
