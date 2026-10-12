# Framework correctness recovery plan

Implement the architectural direction accepted in the Codex discussion on 2026-10-12. Repair shared contracts and ownership boundaries; use review findings as regression witnesses. Authentication is resolved before transport execution, and results are recursively projected before transport encoding. This file is the recovery entry point and progress source of truth.

## Current state

- Twenty-four reports are delivered, pushed and confirmed closed: #670, #692, #708, #711, #712, #713, #714, #716, #722, #724, #725, #730, #731, #732, #741, #743, #755, #757, #779, #784, #788, #792, #794 and #795. Latest deliveries: union declarations `6194743c` and typed collection defaults `febb41bc`.
- #743 gRPC admission is delivered and closed at `d8ccf0ee`; its exact 44-file integration was independently approved. Six integrated modules / 71 artifacts match the reviewed candidate and independent repeat; owner and generated race tests pass.
- #681 readiness v3 is independently approved. `/root/exact_numeric_design` implements the shared exact owner in `/Users/luca/.codex/worktrees/exact-numeric-bounds/loom`, advanced to pushed `27da0247`. Existing finite hexadecimal DSL strings remain supported; source-specific JSON/YAML admission and runtime numeric matching share exact semantics. Sixty-six initial red regressions are retained. Exact bounds also expose authored default/enum projection bypasses: prepared annotations now retain checked wire materialization and declaration provenance, while external-codec outcomes remain explicit; bounded acceptance is in progress.
- #792 recursive imports delivered at `58916bc9`, independently reviewed, pushed and closed. The importer now plans constructor-sensitive reference occurrences once and rejects residual eager cycles before rendering.
- #794 Basic CLI defaults delivered at `8c7d8d71`, independently reviewed, pushed and closed. Shared argument records preserve effective defaults for both HTTP and JSON-RPC.
- `/root/basic_cli_defaults` implements #756/#729 in `/Users/luca/.codex/worktrees/effective-wire-bindings/loom` at `8c7d8d71`. Independent readiness review approved the complete source-tuple/codec binding owner: 17 direct reds, eight controls, generated request/response loss and 7,296 checked model states. Required/default selection remains M7.
- `/root/review_effective_repair` prepares #796 in `/Users/luca/.codex/worktrees/projected-union-validation/loom`. A required constrained untagged view branch compares a projected pointer as a value; parent reproduction is committed beside the union model. Assess the physical validation owner and classify #709 separately.
- Root prepares #762 in `/Users/luca/.codex/worktrees/multipart-decode-ownership/loom` at `febb41bc`. The custom multipart adapter still uses a separate request-elements template and error accumulator rule; readiness must establish shared decode ownership and real callback/validation behavior before implementation.
- Later accepted semantics: explicit empty collections survive encoding, and defaults fill absence without replacing explicit zero. Milestone 7 presence implementation remains outstanding.
- Latest committed tracker checkpoint before this update: `8b5d265c`. Inspect the Git log, dirty trackers and live agents on recovery. Subsequent deliveries through `febb41bc` are recorded below.
- CI at `56bc7eae` failed expression test isolation, an obsolete explicit-body declaration assertion and WebSocket closing behavior. The first two have independently reviewed repairs pushed as `b7e1b1f8` and `27da0247`; #795 is repaired at `2e4c8022`. No overall CI pass is claimed.

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

- 2026-10-12 — User explicitly set the goal to work through the entire tracked backlog without stopping at status updates. Root verified GitHub state: owning #724, #713, #788, #670 and #741 were already closed by their pushed commits; root closed resolved witnesses #725, #730 and #779 as completed. Recovery checkpoint `4c7bd8d7` is pushed. All eight delivered reports are now closed. #716 and #708 candidates are frozen for integration/review; #755 continues final validation. A historical JSON-package reference in #741 evidence triggered the Markdown lint scan; its wording is corrected without changing the evidence, and delivery/cleanup status is made current.

- 2026-10-12 — **Delivered #708:** commit `9880865a` on parent `26f252ef`, pushed to `origin/main`; GitHub confirms closure. `/root/review_artifacts` authorized the exact 12-file integrated diff (SHA-256 `20cfa227f31d7ebc41599a83bac44be89b1e8f0ca3d0e3a554dbd2915a2a566e`) with no findings. Root and reviewer independently passed affected race suites; vet, JSON lint and whitespace checks pass. Evidence: `http/tla/forwarded_client/README.md`. Runtime upgrade suffices. Task-owned scratch was removed after review/process check; managed worktree archival was refused as protected/pinned.

- 2026-10-12 — #716 frozen 124-file patch integrated onto `9880865a` without conflicts. Source hashes match the agent candidate except the HTTP guide retains newer #708 text. Agent `/root/websocket_lifetime` is validating the three selected designs against this immediate parent before fresh independent review. Root owns shared guidance and final integration; the 19 pending #722 files remain excluded. Next authentication readiness is prepared by `/root/auth_readiness`; no production work is released.

- 2026-10-12 — #732 readiness independently accepted by root; implementation released to `/root/auth_readiness` in `/Users/luca/.codex/worktrees/auth-credential-alternatives/loom`, parent `9880865a1c2873cf455fbae979dc7de3a533071f`. Shared alternative-aware extraction must preserve payload-wide Required and malformed-matching-credential rejection; #731, #743, #729, #752 and #728 remain separate. Readiness: `roadmap/auth-credential-readiness.md`.

- 2026-10-12 - #716 current-parent comparison passes for all three selected designs (68 output files) and generated limit/error/projection controls. `/root/review_websocket` reviewed staged 128-file hash `2dd9446272195b7bb1ee85884864d8537bf954ae5cfb7933f55b5bcb02649e66` and found one medium cancellation-cause regression: later reads returned a low-level socket error instead of the documented closed sentinel. No commit authorized. `/root/websocket_lifetime` is repairing the common terminal transition in its worktree, with model and deterministic regressions before re-review.

- 2026-10-12 - #755/#757 candidate frozen in its recorded worktree: 43 files, manifest `/tmp/loom-effective-final-manifest.json` SHA-256 `dd83997cce82eb51e72d4d818379a82aaf2d0997a7913256a08271d840541bcc`, base `ce3f6a73`. Its six specimens compile/vet and 151 output files match independent repeats. Integration and current-parent comparison follow #716 and its required CI-maintenance follow-up; no delivery claimed.

- 2026-10-12 - Remote CI for `9880865a` failed its `ci` job with three maintainability-index violations (`codegen/generator.Generate`, `codegen/service.projectionConversion.typ`, `internal/artifactpolicy.Validate`) and two test-slice preallocation warnings. These are required follow-up repairs to earlier deliveries. Managed worktree `/Users/luca/.codex/worktrees/correctness-ci-gates/loom` at `9880865a` is owned by `/root/ci_gates`; private helper extraction and test preallocation are implemented and validating. Root full golangci on integrated #716 reproduced exactly those five baseline diagnostics and no new #716 diagnostics. #743 admission readiness investigation is assigned to `/root/grpc_admission`; source implementation remains unreleased; its readiness checkpoint is saved in `roadmap/grpc-stream-admission-readiness.md` and will resume after CI repair capacity is available.

- 2026-10-12 - #716 medium terminal-cause finding fixed and independently re-reviewed by `/root/review_websocket`; exact final 133-file diff SHA-256 `369ec8cbdd0c36340a251f8a3039abbdc41281cc66428d299299b929be204b8a` approved with no unresolved findings. Committed locally as `46b627ef`; push follows CI cleanup. Main runtime race tests and vet pass. CI cleanup seven-file candidate is staged for `/root/review_ci_cleanup` on this parent; integrated `make lint` is running. #732 frozen 23-file candidate is under independent review by `/root/review_auth_alternatives`.

- 2026-10-12 - #722 refreshed readiness assigned to `/root/websocket_lifecycle` in `/Users/luca/.codex/worktrees/websocket-lifecycle/loom` at `46b627ef`. Carry only the old lifecycle models/red witnesses from the previous worktree. Concrete buffer API and accounting must handle zero-length messages without unbounded queue metadata; production remains gated on refreshed readiness.

- 2026-10-12 - **Delivered #716:** `46b627ef` pushed with CI cleanup `651372ce`; GitHub confirms closure. CI cleanup independently approved by `/root/review_ci_cleanup`, exact seven-file staged hash `34102545cdf15df78a45cb10563c5c611e7761ddd1694edf52e53332b79923f8`. Full lint in the isolated source and bounded byte comparison passed; root integrated lint also exited zero, but overlapped the next integration and is not independent acceptance evidence. #755 integrated 43 frozen files with exact parent/candidate hash verification; `/root/effective_integration` owns six-specimen validation against `651372ce`.

- 2026-10-12 - #722 refreshed readiness approved for production after current-parent red witnesses and refined models. Explicit message count bounds empty frames; shared terminal owner covers attachment, handler completion/panic and bounded final writes. `/root/websocket_lifecycle` implements in its recorded new worktree. #755 integration exposed three maintainability-index failures; `/root/effective_integration` is performing cohesive private extractions and revalidating unchanged output before final review.

- 2026-10-12 - Completed remote race job at `9880865a` additionally failed custom-package projected-carrier references (`common.CardView` and framework-name variants). Root reproduced the existing generator witness on `651372ce` and prepared a separate ordering repair in `/Users/luca/.codex/worktrees/projection-carrier-packages/loom`; readiness and four-state failing/passing carrier-location model are saved there. Production remains gated on independent readiness review. This is separate from #733 and #784. CI maintenance scratch was cleaned after delivery/review; worktree archival was refused as protected/pinned.

- 2026-10-12 - #755 final staged 46-file diff `970388e34ca9eaa56ec482d65c54b2d225ff593977f49d77e62304fda2664f9d` is under `/root/review_effective_design`. Reviewer reproduced a medium blocker: mixed recursive field/inheritance paths are mistaken for inheritance cycles. No commit authorized; full findings and repair/re-review follow. Root custom-package follow-up is frozen as seven files, manifest `/tmp/loom-projection-packages-final-manifest.json`, SHA-256 `71a015da22ecda29218ca83a45c724aaa342c4495868c2e1f36077c67cdf9553`; 279 generated files match independent repeats and selected compilation/wire/lint checks pass. #732 low stale-readiness finding corrected in the root draft; final integration re-review still required.

- 2026-10-12 - #755 independent review completed with two medium findings: valid mixed recursive field/Extend graphs fail in either declaration order (false cycle or incomplete copied graph), and conflicting peer JSON identities evade admission when declared through different metadata keys. `/root/effective_integration` is repairing both shared owners with reviewer repros, model/test updates and focused revalidation; `/root/review_effective_design` will re-review. No commit authorized. #743 design now favors shared pre-frame authentication admission plus preserved normal endpoint rechecks; an exactly-once callback assumption was rejected because the public authorization contract permits repeats.

- 2026-10-12 - `/root/grpc_admission_design` approved the seven-file custom-package follow-up against `651372ce`, manifest `71a015da22ecda29218ca83a45c724aaa342c4495868c2e1f36077c67cdf9553`, with no findings. Independent direct/model/279-file byte checks passed; final main integration review still required. CI at `651372ce` clears lint but both ci/race jobs fail the same projected-package matrix; no other test failure was observed. Root approved #743 revised readiness and copied it into the main draft; assignment waits for #732. Completed #716 scratch and old lifecycle repro modules were removed after the #722 owner confirmed they were superseded and active-process checks passed.

- 2026-10-12 - #731 readiness/red-only preparation assigned to `/root/basic_credential_design` in `/Users/luca/.codex/worktrees/basic-credential-validation/loom` at `651372ce`. Coordinate with frozen #732 alternatives; no production changes until scoped readiness approval and a stable dependency parent.

- 2026-10-12 - During #755 repair root identified an additional metadata comparison concern: raw `attributeJSONTag` strings do not include the name-only tag default `omitempty` applied by the renderer. New compatibility controls currently invert optional full/name-only equivalence. Implementer must align the rule with the actual tag owner and inherited/effective requiredness before freezing; this is not authorization for a separate transport policy.


- 2026-10-12 - Root confirmed the #755 metadata repair uses effective rendered JSON identity/options, following the accepted readiness rule. Optional full `one` versus name-only `one` conflicts; optional full `one,omitempty` versus name-only `one` is compatible; effective Required makes full `one` and name-only `one` compatible. Normalization belongs in `internal/encodingmeta`, with unchanged codegen delegation and existing requiredness/presence facts. The authored-partial-contract alternative was rejected. Both peer orders and local/inherited comparisons require rendered controls before re-review.

- 2026-10-12 - #731 readiness checkpoint is frozen with ten failing HTTP wire cases and two failing request-plan cases; valid/absent HTTP controls and twelve gRPC metadata controls pass. `/root/review_basic_readiness` is independently reviewing the scoped proposal. Production remains dependent on #732 integration. #722 bounded comparison passes: five parent/candidate designs compile and vet, independent candidate generations are identical, and differences are confined to expected WebSocket adapters; adversarial validation continues.

- 2026-10-12 - #731 readiness independently approved by `/root/review_basic_readiness` with no findings; reviewer reproduced HTTP failures and passing gRPC controls. Implementation awaits #732 integration. #755 resolved-tag tests also exposed inherited Required disappearing after intermediate composition consumed Bases; the existing AllRequired owner now traverses retained parent constraints, with multilevel red/green controls. Final repair gates are running.


- 2026-10-12 - Root began #681 inventory/red-only preparation in managed worktree `/Users/luca/.codex/worktrees/exact-numeric-bounds/loom` at `651372ce`. Twenty direct bound-preservation cases fail above float64's exact integer range; four small-number controls pass. The draft identifies DSL/effective representation as the shared owner and keeps #685 value-range admission and #692 target array literals separate. Exact representation, model evidence and readiness review remain outstanding; no production change is authorized.


- 2026-10-12 - #722 froze 145 implementation paths, then root added three shared guidance files in its isolated worktree. Combined 148-file manifest `/tmp/loom-ws-722-with-guidance-manifest.json`, SHA-256 `112cab2db6ea0a6971e3ab08c489e6e892d6b563a557dcab10844b79a335212a`, awaits `/root/review_effective_repair`'s separate verdict against `46b627ef`; final main integration remains pending. Independent runtime race, shutdown model and all fifteen retained generation manifests pass review so far. The full HTTP package timeout remains explicitly recorded.

- 2026-10-12 - #732 root guidance and the previously requested stale-readiness correction are now frozen in the auth worktree. New 26-file manifest `/tmp/loom-auth-with-guidance-manifest.json`, SHA-256 `de893d9595419757c6d180b3f7c70262b45dcb88924dc01d147e27a0aaeddcae`; original implementation is unchanged, but these documentation deltas require final integration review. #681 readiness/model investigation delegated to `/root/exact_numeric_design` in the recorded worktree, without production authorization.


- 2026-10-12 - `/root/review_effective_repair` approved the frozen #722 148-file worktree with no findings, independently passing runtime races, generated lifecycle wire checks and the 252-state shutdown model. The same reviewer separately approved the #732 documentation correction/new guides against its unchanged implementation and the carrier follow-up's one-paragraph guide addition; the latter is now eight files, manifest `/tmp/loom-projection-packages-with-guidance-manifest.json`, SHA-256 `7ae367742687c24fd2c6a20c3a2ce919a749a0a01b9b095f30e12d44ee257c45`. Each still needs final integrated-diff authorization before its own commit. #755 canonical named-owner binding now passes direct self/mutual recursion and snapshot mutation checks; final generated/schema comparison is running.


- 2026-10-12 - **Delivered #755 / #757 witness:** `4f7892a2b6b351786616a69fe7bbc3bee9202375` pushed to `origin/main`; GitHub confirms both closed (twelve delivered reports total). `/root/review_effective_repair` authorized the exact 59-file staged diff `b59d7c2abf66832bdb298623a450b8682e492a182550ecde9bde9aee5e5a8249`, final manifest `f138d283f484d04eb352da73b296c3afbff209c6a8a82a9f22ad5e5e0b4407c5`, with no unresolved findings. Original recursion and JSON identity findings, requiredness and structural binding follow-ups, and both low documentation/style corrections were confirmed. Durable evidence: `roadmap/effective-design-readiness.md` and `expr/tla/effective_design/README.md`. Completed task scratch removed after reviewer completion and active-process check; source/tests/models are committed.


- 2026-10-12 - **Delivered projected-package follow-up:** `c735c6360047dd964a3fe309de31ed3b3d065973` pushed to `origin/main`. `/root/review_effective_repair` authorized the exact eight-file diff `0c2eb151864e44e6b62220c1f227c5852feb63437342cd00f6b47fa3da3bba2b` against `4f7892a2`, with no findings. All 279 integrated generated files match both the previously compiled candidate and a new independent repeat; direct regression and three existing package-name compile/build/vet cases pass. This repairs the known CI failure family; remote CI outcome is not yet claimed. #732 has now integrated cleanly and is running three-transport wire and bounded output checks.


- 2026-10-12 - **Delivered #732:** `433d2e741b43b35783d293d0de60cabeca0be6a2` pushed to `origin/main`; GitHub confirms closure (thirteen reports total). `/root/review_effective_repair` authorized the exact 26-file diff `ec42f77572914b68257e97438d6f39cad2158d74ec510dc2d83f90e2c584fcff` against `c735c636`, no findings. All 108 artifacts match independent repeats; three-transport wire/race tests, affected units and scoped lint pass. Evidence: `roadmap/auth-credential-readiness.md`. #731 production released to its existing worktree after safe fast-forward to this parent. #722 all 148 frozen files integrated cleanly; its agent owns final bounded validation, with a verified pushed module pin replacing the old temporary v0.0.0 input. #743 managed worktree creation is pending operation `43a7a4ce-31e8-467e-a680-8b330be348d8` at this parent.

- 2026-10-12 - #681 readiness froze eleven non-production files in its worktree, manifest `/tmp/loom-exact-numeric-readiness-manifest.json`, SHA-256 `065c1ac3379e2a1e35d063be4b2a8aa0bbd919d0fec12a7faf32bfe8f202d769`. Thirty-two DSL/import precision cases reproduce; source-admission/raw-node controls and nine bounded Lean claims pass. Root requested the genuinely unresolved public numeric-string syntax decision: exact decimal strings with hexadecimal strings rejected (recommended), or preserving hexadecimal strings with further cost/limit investigation. This question remains pending; no dependent parser policy or production is authorized.


- 2026-10-12 — **Delivered #722 / #711, #712 and #714 witnesses:** `56bc7eae` pushed to `origin/main`; all four reports are confirmed closed (seventeen total). `/root/review_effective_repair` authorized the exact 148-file integrated diff `1fe7645447eb6b09672e7f08f8ccbfe3b551aac8d4af43a5b30f822ccf19e8b8` against `433d2e74`, with no unresolved findings. Five designs produce 114 files identical to the reviewed candidate and independent repeat; focused runtime race and generated real-socket checks pass. Evidence: `roadmap/websocket-lifetime-readiness.md` and `http/tla/websocket_lifetime/README.md`. Regenerate adapters for the new shared configuration and context attachment contract. #739 and other protocol-specific findings remain open.

- 2026-10-12 — Completed #732 and #722 temporary comparison/checker outputs were removed after final review and active-process checks; durable evidence is committed. Managed archival of both worktrees was refused because they are protected by a pinned task or workspace. Their source remains attached; no manual worktree deletion was attempted.

- 2026-10-12 — #743 managed worktree creation completed at `/Users/luca/.codex/worktrees/grpc-stream-admission/loom`, parent `433d2e74`; approved production implementation released to `/root/grpc_admission_design`. Metadata snapshots and the shared authentication runner must preserve cancellation, callback context, declaration ordering and normal endpoint rechecks. #731 implementation has expanded its red HTTP/JSON-RPC cases; an existing defaulted-Basic CLI compile defect is retained separately, outside this decoder-validation repair.

- 2026-10-12 — #681 policy response: the user emphasized cross-language interoperability and did not accept decimal-only strings. Root withdrew the Go-literal justification and resumed readiness investigation with existing hexadecimal-string support retained as the candidate. No parser narrowing, new numeric complexity limit or production implementation is authorized. The earlier pending choice is superseded by this investigation.

- 2026-10-12 — Recovery checkpoint independently reviewed by `/root/exact_numeric_design`, committed and pushed as `b46345a0`; the exact two-file diff was `5a209b03d34b3c75f325867077cb10b537402b8e2c53836ab56f027c41b4ecde`. Root began #792 inventory/red/model preparation in `/Users/luca/.codex/worktrees/recursive-import-references/loom` at `56bc7eae`. Three recursive component shapes fail Go initialization; matching narrow named-lookup controls evaluate successfully. The bounded reference model reproduces the failure and checks the candidate over 19,683 classified graphs. Production remains gated on full reference-context readiness and independent review. #706/#719/#793 serialization was inspected only and remains separate.

- 2026-10-12 — **Delivered #731:** `601a4338b` pushed to `origin/main`; GitHub confirms closure (eighteen reports total). `/root/review_effective_repair` approved the isolated implementation after a low formatting correction; `/root/websocket_lifecycle` authorized the exact 204-file integrated diff `8c8d304eaa62f2240535c42f8296715e4c3035ddf99ca1ce1eab79c00a26b2e4` against `b46345a0`, no findings. Seven designs / 144 artifacts match the reviewed candidate and independent repeat; integrated HTTP/JSON-RPC wire/race and direct gates pass. Existing bounded compilation/fixture evidence is reused. Evidence: `roadmap/basic-credential-validation-readiness.md`. The separate, reproduced CLI default representation defect is filed as #794 and remains open.

- 2026-10-12 — #681 readiness v2 is frozen as seventeen non-production files, manifest `/tmp/loom-exact-numeric-readiness-v2-manifest.json`, SHA-256 `a5d3db15e3fef936dffe8cce2aa4c329435b41a86a250f2d32546599e4acd8ee`. Forty-four precision regressions reproduce across DSL, decimal import and YAML radix import; sixteen scoped Lean claims and bounded interval/protocol probes pass. Root released independent readiness review to `/root/review_effective_repair`; no production is authorized. Existing finite hexadecimal strings are preserved in the candidate, with source-specific JSON/YAML admission. Runtime JSON-shape numeric consumers and their existing capacity-failure contract require explicit reconciliation.

- 2026-10-12 — #792 readiness investigation delegated to `/root/websocket_lifecycle` in the existing recursive-import worktree. Root's red/model checkpoint remains the starting point; constructor-specific symbolic reference support, eager cycles and error/union paths need a concrete shared plan before production.

- 2026-10-12 — Completed #731 cleanup: removed 44 task-owned scratch paths after review and durable evidence preservation. Managed worktree archival was rejected because it is protected by a pinned task/workspace; checkout retained unchanged.

- 2026-10-12 — CI `38296412526` exposed three distinct regression causes. Root fixed expression test context isolation (exact reviewed diff `2c4e527b35b576392d962091fd1b36a22a02677600c824ea8dcd995a730852d6`, pushed `b7e1b1f8`; same shuffled full expr race suite passes 5.966s), then corrected the explicit-body seam assertion to distinguish declaration and physical value carrier (exact reviewed diff `4644ac78e68266ba6b70f1f533c30ccdfce924adad7e9d37bfbc4c1fb002540e`, pushed `27da0247`; focused nullable/body gates pass 0.833s). `/root/review_effective_repair` approved each atomic commit separately. No production was changed by either maintenance repair.

- 2026-10-12 — Filed #795 for confirmed generated WebSocket close cancellation before queued data/null consumption. Three union wire runs fail on `601a4338`; nested collection CI also fails. `/root/websocket_lifecycle` paused #792 readiness, preserved its model/reference-context checkpoint, and reproduced the missing lifecycle sequence in TLA+. Candidate bounded peer wait is under contract/seam investigation; no test weakening is authorized.

- 2026-10-12 — #681 v3 readiness independently approved by `/root/review_effective_repair`: nineteen-file manifest SHA-256 `e369c61af21823c15d07f7570cfe55fa18246f61fc906dad44ee2e8ad9e58f3b`. Runtime `pkg` shape matching and prepared schema membership now belong to the proposed common exact owner; evaluator errors must propagate fatally, and unrenderable imported overflow requires a precise diagnostic. Production released after the worktree safely advanced to `27da0247`, all nineteen readiness hashes preserved. This is implementation authorization, not delivery evidence.

- 2026-10-12 — #743 isolated 41-file implementation independently approved after both low findings were repaired. Final pinned comparison repeated all eighteen modules and six builds/vets; all 71 artifacts and intended differences match. Root integrated onto `27da0247`, added three owning guides/index entries, reran owners and generated races (21.183s), and repeated integrated output/compilation. Exact staged 44-file diff `d329325650f2067fe05471eb9a485d521423871f71c3239beb9c0be4c7e20ce3` is under final integration review. Trackers are excluded from that commit.

- 2026-10-12 — **Delivered #743:** exact final 44-file diff `d329325650f2067fe05471eb9a485d521423871f71c3239beb9c0be4c7e20ce3` approved by `/root/review_effective_repair`, committed and pushed as `d8ccf0ee`; GitHub confirms closure (nineteen reports total). Gofmt/filesize checks pass. Removed 39 completed task scratch paths after durable evidence preservation; managed archival remains blocked by protected-worktree status.

- 2026-10-12 — Root approved concrete #795 readiness after inspecting the new counterexample and bounded peer-wait model. Production released to `/root/websocket_lifecycle` in `/Users/luca/.codex/worktrees/websocket-close-order/loom`. One terminal deadline covers writer acquisition, final JSON and physical-peer wait; raw Close and half-close remain distinct. Model correspondence and synchronized socket regressions are mandatory before final review.

- 2026-10-12 — Recovery checkpoint reviewed after two low presentation fixes, committed and pushed `214bab44`; exact two-file diff `4a5f08ce25f4816eee99bcf4f0dcf974c0a7461911d53a729ac922a5ac74cf3a`. Inventory contains 117 reports including #794/#795.

- 2026-10-12 — #792 readiness independently approved, twelve-file manifest `be7d7efb2eb79856fceb4645d708e07194c74ec18c0391f9cc52e0b19275a2f6`. Root assigned implementation to `/root/review_effective_repair` (a different final reviewer will be required). Worktree advanced to `214bab44` with all readiness hashes preserved. Direct planner reds establish per-occurrence ownership and mandatory-cycle diagnostics before production; coordinated numeric/importer file boundaries with #681.

- 2026-10-12 — **Delivered #795:** fresh reviewer `/root/review_ws_close_order` approved exact 13-file integrated diff `ec13e6f6ac898172c5f66793f17e0ee87a9760adad511033c7b48def128b0740` against `214bab44`, no findings. Committed/pushed `2e4c8022`; GitHub confirms closure (twenty reports total). Integrated HTTP/JSON-RPC runtime race checks pass 7.606s/2.091s and original generated regressions pass 13.445s. Independent positive/negative model reruns agree. Durable evidence is `roadmap/websocket-close-order-readiness.md`; completed task scratch removed and protected managed checkout retained when archive was rejected.

- 2026-10-12 — Assigned #794 readiness to `/root/basic_cli_defaults`: shared Basic CLI argument metadata must agree with physical presence/default representation; no emitted-code workaround or unrelated M7 policy rewrite. Root retains production release and final delivery ownership.

- 2026-10-12 — **Delivered #794:** `8c7d8d711b9e2fe54fbff1dabb9b49fd361ed170` pushed to `origin/main`; GitHub confirms closure (21 reports). `/root/review_basic_cli_defaults` approved the exact seven-file integrated diff `abcdab8399a328bcfd0ab1773be527a1a6b188bdc59c1008109b90990bb39bb3` against `2e4c8022`, no findings. Four shared metadata additions restore each Basic credential’s effective CLI default. HTTP/JSON-RPC full generated CLI build/vet/race and default/override wire tests pass; 67 compared artifacts show exactly six intended CLI changes and zero repeat differences. Root integrated focused tests passed (6.998s/6.996s). Evidence: `roadmap/basic-cli-defaults-readiness.md`. M7 presence policy remains separate. All 18 completed `/tmp/loom-basic-cli-794*` scratch paths were deleted after review; managed worktree archival was refused as protected/pinned, so its checkout remains attached. The implementer now owns readiness-only #756/#729 in `/Users/luca/.codex/worktrees/effective-wire-bindings/loom`; seven current-parent reds and six controls precede the shared binding proposal.

- 2026-10-12 — **CI assertion maintenance:** `589e96b1` pushed after `/root/review_effective_repair` approved the exact two-file test-only diff `09e8afc9f656d5c1d9acba82ca9b87f05bf0553e10e4c51e7288fa994a6bc5e6` against `8c7d8d71`, no findings. Three assertions expected the shared decoder error variable inside the body declaration block; they now check both declarations separately and preserve EOF, validation and presence assertions. Current-parent reds reproduced and the focused optional object/value body tests passed (0.857s). CI run `38300194426` also failed on the already-repaired #795 and a Go proxy download error; this is not an overall CI pass.

- 2026-10-12 — **Delivered #792:** `58916bc96d8057874242c6fc330f44bc1c684274` pushed to `origin/main`; GitHub confirms closure (22 reports). `/root/review_recursive_import` approved isolated 22-file source and exact 25-file integrated diff `828d47c2ee300eb706c569403950e1b7ef424e002a0c5b89b3f24f901d742180` against `8b5d265c`, no findings. Independent focused acceptance passed 104.206s; root current-main direct/evaluated/generated acceptance passed 110.129s. Eighteen evaluated cases, nine durable generated cases, 192 exact-repeat artifacts and 24 Vacuum specs cover constructor-sensitive recursive references; 32 parent acyclic and 64 parent mechanism-control artifacts remain identical. Evidence: `roadmap/recursive-import-readiness.md` and `internal/openapiimport/tla/schema_references/README.md`. All 29 completed #792 scratch paths were removed after review; managed archival was refused as protected/pinned. The implementer now continues readiness-only #784 in the recorded view-union worktree; main/shared guide changes remain root-owned.

- 2026-10-12 — **Delivered #784:** `6194743c9485dd242762d230fabc05519ecc1765` pushed and closed. `/root/review_view_union_final` independently authorized the exact 25-file diff `721467ca5aad4483bc000fde33e7b0d7f12c845996d3ec4add3af115d30e256d` against `58916bc9`, no findings. Prepared body roots own package declarations and validators; one allocator retains union identity. Eleven modules / 267 artifacts compile/vet and repeat exactly, with five intended server-only removals. Independent focused tests passed 12.228s/10.276s; integrated tests 18.873s/13.880s. The corrected untagged controls exposed a separate required-pointer validation bug, filed as #796 with a durable parent reproduction.
- 2026-10-12 — **Delivered #692:** `febb41bce2d2b80b36f73c67316f6c7d2cc9d226` pushed and closed. `/root/review_typed_defaults_final` authorized exact nine-file diff `c0b83dc6e6407371dfdf1f64ff5fc6e012051f477aace483377b1c5bd83d2a7b` against `6194743c`, after confirming a mechanical low-severity import grouping fix. Shared recursive target-typed collection literals replace competing assignment/alias formatters. Forty representation cases, 32 application gates, native/Optional/Nullable race/vet checks and HTTP/JSON-RPC compilation pass. Seven specimens / 109 artifacts repeat exactly; six intended default-assignment files change. Integrated focused checks passed 5.023s/2.763s/2.301s. #685, #693 and M7 policy remain separate.
- 2026-10-12 — After final review and delivery, removed all 90 completed #784/#692 scratch paths, including generated trees, manifests, checker output and reviewer overlays. Durable source/probes and summarized evidence are committed in their readiness/model records. Managed archival refused both worktrees as protected/pinned; their attached checkouts remain intact. Active numeric, wire and #796 evidence is retained.


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

- [x] Deliver #731 Basic credential validation at `601a4338b`; independently reviewed, pushed and closed. Evidence: `roadmap/basic-credential-validation-readiness.md`. Separate CLI default compile issue #794 remains open.
- [x] Deliver #732 credential-alternative extraction through shared security analysis; reviewed, pushed and closed at `433d2e74`. Evidence: `roadmap/auth-credential-readiness.md`.
- [x] Deliver #724 nearest-scope ownership with #725 contradiction witness; independent review, commit `544c57f3`, push and integration complete. Evidence: `expr/tla/authentication_scope/README.md`.
- [ ] Inventory `rg -n 'EffectiveRequirements|SessionAuths|Requirements|DecodeBearer|BasicAuth|writeGRPCStreamInputDecode' expr security codegen/service http/codegen grpc/codegen jsonrpc/codegen`; reconcile #724, #725, #729, #731, #732, #743, #752 and #728 by exact cause, splitting distinct structural tasks before implementation.

- Read-only follow-up reconciliation: #729's blanket suggested rejection conflicts with the delivered, supported Basic-or-Bearer alternative contract (#732). #756 establishes explicit duplicate wire-name loss; #729 adds implicit security mappings after validation. Investigate one effective wire-binding owner before validation, but distinguish a Basic composite credential and disjoint authorization framing alternatives from two independently authored values overwriting one slot. Preserve HTTP header case-insensitivity, query/cookie case sensitivity, explicit/implicit/inherited mappings, and valid same-name mappings in different locations. Do not close both issues from a generic duplicate-name check without these controls.
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

- [x] Deliver projected-package ownership follow-up `c735c636`; reviewed and pushed, with 279 deterministic generated artifacts and existing package-name compilation controls. Evidence: `roadmap/projection-carrier-packages-readiness.md`.
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

- [x] Deliver #716 message admission at `46b627ef` and #722 lifetime ownership at `56bc7eae`, with #711/#712/#714 witnesses; independently reviewed, pushed and closed. Evidence: `http/tla/websocket_message_limit/README.md` and `http/tla/websocket_lifetime/README.md`.
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

- [x] Deliver #755 effective Extend composition with #757 witness at `4f7892a2`; independently reviewed, pushed and closed. Evidence: `roadmap/effective-design-readiness.md`. Remaining validation reports require separate scope reconciliation.
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

- [x] Deliver #708 proxy-address trust at `9880865a`; independently reviewed, pushed and closed. Evidence: `http/tla/forwarded_client/README.md`.
- [x] Deliver promoted #670 artifact write-intent ownership; independent review, commit `41b4a8a3`, push and integration complete. Evidence: `cmd/loom/tla/artifact_write_intent/README.md`.
- [x] Deliver #741 example interceptor declarations through that shared writer; independent review, commit `23f2af5d`, push and integration complete. Evidence: `roadmap/example-interceptor-ownership-readiness.md`.
- [ ] Reconcile inventory residuals, including #670/#741 file ownership and #708 proxy trust, against final integrated source; promote each supported defect to a separately reviewed bounded task.
- [ ] Run targeted adversarial review of repaired owners and their consumers; record new counterexamples and update task acceptance before implementation.
- [ ] Run `make lint`, `make test` and integration checks selected by actual changed behavior; record exact source and outstanding blockers. Do not start a full exported-design audit without a named purpose and cost budget.
- [ ] Confirm every delivery commit was independently reviewed and pushed; update `roadmap/ROADMAP.md` and this tracker to reflect remaining active work.
- [ ] Preserve durable evidence and archive completed managed worktrees; remove task-owned model/compiler output after checking active processes.
