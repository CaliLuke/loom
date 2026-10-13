# JSON-RPC mapped error representation — #817

Status: implementation and bounded acceptance complete; root independently approved
the candidate. Integrated guidance and the final main diff await fresh review.
Owner: `/root/review_uniform_json_corrections`.
Worktree: `/Users/luca/.codex/worktrees/jsonrpc-error-representation/loom`.
Parent: `0b5657422`.

## Contract and root cause

An explicit JSON-RPC response mapping uses its designed error body; unmapped
errors use `jsonrpc.ErrorData`. The existing built-in ErrorResult body has public
problem fields and must be projected from `http.ProblemResponse`, never read
as fields of `ServiceError`. This is an existing documented contract, not a new
JSON-RPC error schema. Preserve public-message selection, error codes, remedy
projection, notifications and custom typed bodies.

#797 moved public-source construction from the body constructor into HTTP
response emission. Shared response analysis records `NeedsProblemSource` and
constructor arguments referring to `problem`, but three JSON-RPC mapped-error
emitters omit normalization. The existing authorization integration test fails
on this parent with undefined problem at generated server.go:157; CI reproduces
it across unary, notification, SSE and WebSocket modules. Source commit is
`14ab5a758`. Local log: `/tmp/loom-jsonrpc-problem-red.log`; CI race job
115074353568 at285da6b0e.

The repair owner is one private JSON-RPC mapped-body emission helper consuming
its shared HTTP response plan. All three renderers derive normalization and
projection from that helper; transport dispatch and envelope emission remain
with their existing owners. Normalize the original error, preserving its outer
safe-message/remedy ownership, after existing typed-selection admission.

This is deterministic source materialization and reuses the established
`http/codegen/tla/problem_response_projection` field-projection model. No new
protocol state, lifecycle, ordering or public semantics are introduced, so a
new model would not add evidence. The existing model explicitly excludes Go
local binding; direct render tests and generated compilation cover that gap.
#713 disclosure model remains unchanged; mounted tests cover its correspondence.

## Ordered tasks

- [x] Read #713 and #797 owner records, inspect current analysis and all emitters.
- [x] Reproduce the original existing integration failure before production edits.
- [x] Add failing direct plan/render and mapped-disclosure generated tests.
- [x] Obtain independent readiness assessment of owner, scope and acceptance.
  Root approved manifest `aa1600ca633e27d3e416b8132af8bf3407ceebb939cf5650f59d04a6f86a7251`.
- [x] Implement shared mapped-body emission and remove duplicated constructor logic.
- [x] Run direct regressions and bounded generated authorization, notification,
  typed/custom/unmapped and mapped disclosure runtime tests. Cover unary, SSE,
  WebSocket, endpoint failure and stream SendError; explicit safe remedy on an
  outer wrapper must affect envelope message and body detail consistently.
- [x] Compare parent/candidate/independent-repeat output for mapped built-in
  disclosure design plus unchanged unmapped and custom-typed controls. Build/vet
  candidate modules. List every intentional byte difference; no full corpus.
- [x] Run affected owner tests and lint and freeze exact manifest/evidence.
- [x] Obtain fresh independent candidate implementation review.
- [ ] Review the integrated diff, then commit, push and close #817.
- [ ] Preserve concise durable receipts, clean task-owned temporary artifacts
  after integration review, and release the managed worktree through app tools.

Acceptance: all generated variants compile; designed error.data, safe detail,
code, id and optional retry_hint agree with the accepted owner; original errors
are not mutated. Custom/unmapped type and body-constructor output remains byte-identical.
Mapped builtin source materialization and shared SSE endpoint delegation are the
only semantic output changes; forwarding the caller context is mechanical.

Readiness evidence: `/tmp/loom-817-direct-red.log` records missing normalization
for all three transports; `/tmp/loom-817-mapped-red.log` records four undefined
source errors across unary, SSE endpoint, SSE service stream and WebSocket.
The original authorization witness remains unchanged. The extended existing
harness adds outer safe-remedy and preserves unknown/join disclosure controls.

## Expanded readiness: SSE endpoint-returned errors

The new mounted matrix exposes a second consumer bypass of the same mapped-body
plan. `writeJSONRPCSSEHandlerInitBody` handles endpoint-returned errors by
selecting generic protocol metadata directly. It never delegates designed
mapping to the endpoint stream's existing `mappedErrorResponse`/`SendError`.
This predates #797 and affects custom mapped errors too. The unchanged custom
`jsonRPCTypedStreamingErrorBodyDSL` generated with exact parent source compiles
and vets, then a mounted endpoint returning its ToolError emits -32603 instead
of 4403 and generic data instead of the authored body. Reproduction:
`/tmp/loom-817-parent-custom-endpoint-red.log` and the durable
`TestJSONRPCSSEEndpointUsesMappedErrorBody` harness. HTTP200, SSE message event,
request ID and service call assertions pass before the incorrect code assertion.

`docs/error-handling.md` explicit mappings select their designed data schema;
there is no exception for endpoint-returned SSE errors. Reuse the endpoint
stream's existing mapped terminal owner instead of implementing another
projection at its caller. Keep decode/protocol InvalidParams/MethodNotFound
branches generic and preserve request-ID suppression, terminal ordering,
failed-write error wrapping and existing SSE media/event timing. The terminal
state algorithm is unchanged; existing SSE lifecycle tests cover correspondence.

Bounded expected output expands: mapped builtin source initialization plus SSE
endpoint failure delegation change; custom/unmapped SSE endpoint handler bodies
also change mechanically to call the existing terminal owner. Their type/schema
output stays byte-identical; unmapped runtime behavior stays unchanged. The
custom mapped wire failure must become green without weakening its assertions.
Root requested this common-owner scope rather than a separate residual ticket;
root approved expanded readiness manifest `36f4f9f838b2f727b4c30ae7348fe1f9d78ec522eaca961cc7c60b9c921e031e` before the endpoint path changed.

## Final acceptance evidence

The private `mappedErrorBody` owns source preparation plus existing body
projection for all three JSON-RPC renderers. It consumes `NeedsProblemSource`
and constructor argument metadata without re-deriving transport policy.
It normalizes the original `err`, preserving outer safe remedy ownership.
SSE endpoint-returned service errors call the existing `SendError` terminal
owner; the two special protocol branches remain byte-identical. The caller
context now reaches `mappedErrorResponse`. No lifecycle state machine changes.

- Original unchanged authorization witness: parent fails with undefined problem.
- Direct source/plan plus mapped/unmapped disclosure and custom SSE endpoint
  integration: pass12.373s (`/tmp/loom-817-implementation.log`). The later helper
  whitespace cleanup is covered by final generated modules below.
- Selected JSON-RPC custom/unmapped/mapped error, notification, service-name and
  generated SSE result-type controls: all behavioral cases passed in30.642s;
  one expected stale golden was updated and its exact affected tests rerun.
  Five goldens now record SSE delegation and caller-context forwarding.
  Final focused golden/terminal/lazy/GET checks pass0.580s
  (`/tmp/loom-817-goldens-green.log`). Do not describe the initial stale-golden
  run as a full pass or this selection as the complete package suite.
- HTTP problem owner/disclosure controls pass2.797s; production HTTP is unchanged.
- Regenerated ticktock terminal-response race test passes14.077s, including
  concurrent sends, closed-stream rejection, call context and preservation of
  original endpoint errors after terminal response. Checked-in fixture was
  regenerated with Loom; its resulting module vets successfully.
- Pinned golangci-lint on `./jsonrpc/codegen` reports0issues; name-scope check and
  `git diff --check` pass. No nested testdata Go fixture was edited. No full
  corpus, unrelated package sweep or proof rebuild was performed.

`/tmp/loom-817-comparison/report.json` records83 generated artifacts across
three existing design inputs. Parent generation uses an exact0b5657422 source
archive. Candidate and independent-repeat generations match every byte.
All three candidate modules build, vet and execute race-enabled mounted tests.
The require pin is pushed source
`v1.10.0-alpha.9.0.20261013012419-0b565742284e`, with explicit local replacements
for the parent/candidate source under test. The test-only capture adapter does
not alter generation semantics. Parent builtin module has four undefined-source
compile failures; parent custom module compiles/vets then fails its mounted
4403 code assertion with actual-32603. Unmapped parent behavior passes.

Every intended base/candidate difference (9files):

1. Mapped builtin design (35artifacts): unary server.go, SSE sse.go and
   WebSocket websocket.go materialize the public source; SSE stream.go does so
   and forwards call context; SSE server.go delegates endpoint errors.
2. Unmapped design (35artifacts): SSE server.go delegates errors to the existing
   mapping owner, preserving its generic wire behavior; stream.go forwards ctx.
3. Custom typed SSE design (13artifacts): server.go delegates endpoint errors,
   correcting the previously ignored mapping; stream.go forwards ctx. Its custom
   body constructor, service-level SSE emitter, types and client are unchanged.

`/tmp/loom-817-fixture/report.json` compares checked-in JSON-RPC ticktock with
exact parent regeneration (zero differences), then candidate regeneration
(exactly clock/server/server.go delegation and clock/server/stream.go ctx).
The two corresponding checked-in generated files were regenerated in place;
no generated file was hand-edited and unrelated module dependency changes from
the generator command were discarded. Neither HTTP nor JSON-RPC public schemas
change. Root owns shared-guide integration, exact final review and delivery.

## Integration receipt

Root approved the exact 15-file candidate with no findings, after inspecting all
production/test/fixture changes and bounded comparison evidence. Independent
source-plan, mapped/unmapped disclosure, custom SSE endpoint and terminal tests
passed in 9.500s (`/tmp/loom-817-independent.log`). The candidate patch applies
without conflict after #727; its response-body family selection remains intact.
Public and framework guidance now record shared endpoint/SendError mapping.
