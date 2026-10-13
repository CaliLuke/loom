# JSON-RPC selected response projection — #853 readiness

Implementation parent: `eaa9c85065bc7f3510dd46780564f37feff08066`, advanced
with `git merge --ff-only` after readiness approval in the reused managed
worktree `import-credential-parent`. The parent includes delivered #854 physical
storage and #847 selected-field identity. Independent owner/model readiness was
approved against `d4b2905d8`; their ownership seams remain the same.

Reproduced source: pushed `1074b3cdf594f78d6a7c60db84ab21d4728a5561` via
`v1.10.0-alpha.9.0.20261013090112-1074b3cdf594`; standalone module has no replace.
Scope: successful result-to-wire projection before JSON-RPC framing. Keep this
separate from #847's source-field naming and the selected-view constructor
carrier mismatch. This packet implements the independently approved shared owner.

1. Preserve the ordinary-field standalone reproduction. Generation, build and vet
   pass; mounted selected strings (including empty) send the complete service
   object and an excluded sibling field. The generated client rejects that object.
   Whole-result and direct generated-encoder controls pass.
2. Independently review the exact internal report, then file it. Confirm the
   current accepted Body contract; no new product policy is proposed.
3. Trace response projection ownership across the unary handler, batch dispatch,
   notifications, response metadata capture and the existing WebSocket/SSE
   branches. Record which paths actually share the flawed selection rule and
   which already consume the correct projection. Do not infer affected behavior
   merely from a similarly named framing function.
4. Before implementation, model the current selection (metadata capture, body
   constructor, raw-result fallback) and reproduce the scalar-selection
   counterexample. Compare candidate single-owner designs. Invariants: framed
   result equals the selected transport body; excluded siblings never appear;
   metadata and selected view derive from the same response; notifications emit
   no response. State codec/buffering and protocol assumptions explicitly.
5. Make protocol framing consume the existing response projection contract rather
   than independently inferring it from whether a constructor exists. Prefer a
   shared projection seam for genuinely affected consumers; avoid another scalar
   exception or a header-dependent semantic fix. Evaluate reuse of the generated
   response encoder versus a shared typed projection before selecting the final
   design, including configured encoder behavior and allocation costs.
6. Add mounted wire/client regression tests for selected primitive and named
   scalar bodies, empty/zero values, arrays/objects where valid, ordinary whole
   bodies, headers/cookies, and selected views only after the separate carrier
   compile prerequisite is repaired. Cover notifications and a bounded mixed
   batch to preserve framing/ID behavior. Inspect streaming consumers and extend
   only branches whose projection owner changes. Verify excluded sibling values
   never reach the wire and service invocation remains correct.
7. Compare a bounded selection of existing designs before/after in independent
   processes. Document every intended output difference, compile/vet generated
   modules, run affected runtime/race checks and preserve metadata/error controls.
   Freeze the exact atomic packet for independent review before root integration.

Evidence: `/tmp/loom-selected-response-repro/` contains the pinned module, design
and mounted/direct/client tests; `/tmp/loom-selected-response-repro.log` records
passing controls and exact failing responses. The outgoing report is
`/tmp/loom-selected-response-issue.md`. `jsonrpc/codegen/handler_sections.go`
contains `writeJSONRPCResultSuccess` and the header/cookie-only capture gate;
the generated scalar EncodeCallResponse already selects the correct ordinary Go
field. This establishes a runtime framing/projection disagreement, not a new
field-selector or nullable-view-validator failure.

- [x] Reproduce on pushed source without custom fields or a local replacement.
- [x] Preserve exact wire mismatch, client failure and passing controls.
- [x] Independent report review and filing as #853.
- [x] Independent current owner/model readiness approval.
- [x] Model candidate shared owner and settle implementation scope.
- [x] Implement and validate the bounded atomic repair.
- [ ] Independent exact-final review, root integration and issue closure.

## Current ownership evidence and proposed design

At `d4b2905d8`, the saved mounted regression again fails for ordinary and empty
selected strings and the generated client rejects the object. The complete
module builds and vets; whole-result and direct response-encoder controls pass.
A supported client-streaming method with a selected final response reproduces
exactly the same leak over JSON-RPC WebSocket **and HTTP WebSocket**. Both
complete modules build/vet before their mounted wire assertions fail. These
are consumers of the same missing projection decision, not separate scalar
patches.

| Consumer | Current projection owner/use | Proposed change |
| --- | --- | --- |
| HTTP unary encoder | Shared ResponseData: constructor arguments, selected ResultAttr and projected source | Emit complete value expressions from its ResponseEncodePlan |
| JSON-RPC HTTP unary | Metadata invokes HTTP encoder; otherwise constructor or raw res | Consume the same plan for nonmetadata paths; retain existing metadata encoder capture |
| JSON-RPC batch | Calls unary handler through existing batch item writer | Inherits corrected projection, preserves framing and metadata handling |
| Incoming HTTP notification | Invokes endpoint then returns before projection | Preserve no codec calls and no response |
| JSON-RPC WebSocket response/notification | writeStreamResultBodyInit chooses constructor or raw result | Bind canonical source and consume plan, preserving IDs/view framing |
| HTTP WebSocket final result | writeServerWebSocketResponseBody returns false without Init, sending raw res | Consume plan even when no constructor is needed |
| JSON-RPC SSE mixed events | Separate StreamingResult; sseResponseBodies deliberately omitted | Preserve event source; never apply unary Result Body selection |
| Ordinary SSE projected events | Existing response-body constructors and view-family conversion | Preserve output; consume shared expression where the existing family applies |

Pure `StreamingResult` plus `Body("value")` currently fails during DSL
execution because the named Body resolves Result before finalization. Result
without streaming plus SSE also rejects, as does a nonstreaming JSON-RPC GET.
The supported SSE inventory therefore uses mixed Result/StreamingResult; its
separate string event must remain a string even though the unary result selects
an object field. Those rejected designs are not additional implementation work.
The supported WebSocket inventory uses StreamingPayload(String) with a final
object Result and Body("value"). Its compiled mounted request proves the leak.

The owning data structure is **ResponseEncodePlan**, built by the private
`newResponseEncodePlan` in `http/codegen/service_data_response.go`. Extend that
plan with complete per-body/per-view value expressions. Build each expression
once from the already resolved source variable, projected-storage flag,
ResultAttr and constructor ServerArgs. An absent body is explicit. A selected
field without a constructor is a complete projection, not a reason to return the
service result. Do not introduce another selected-field binding: #847 resolves
that identity upstream; this plan consumes the existing ResultAttr and #854
constructor storage semantics.

Consumers bind the canonical source name expected by the plan and select the
appropriate view variant, then frame or encode the resulting value. Existing
constructor calls must use every ServerArg rather than assume one service
argument. Keep helper visibility private; metadata consumed by both transport
generators belongs alongside the existing public generator data types, not in
an exported pass-through rendering helper. Remove the superseded raw-result
fallbacks; do not leave separate copies of selection policy in each transport.

Keep the current configured-encoder boundary: nonmetadata JSON-RPC HTTP calls
encode the envelope once; metadata calls retain their existing body capture and
then envelope encoding. All these calls use the same projection plan. Universal
body serialization/capture would invoke custom encoders an extra time for
formerly single-encode responses and allocate a full extra body buffer. A
capture-only repair is therefore rejected. This work does not redesign codec
selection, error formatting or metadata buffering. Transport write failures and
view metadata remain associated with the same projected response.

## Implementation sequence after independent readiness approval

1. Add direct plan tests for selected native/named/nullable/collection/object
   values, constructor ServerArgs, whole values, bodyless values and view variants.
   Existing mounted readiness tests must fail first and become durable regressions.
2. Populate complete expressions in ResponseEncodePlan and have the HTTP response
   template consume them. Preserve valid HTTP unary output byte for byte.
3. Make both WebSocket projection paths and JSON-RPC nonmetadata unary framing
   consume that owner. Preserve existing view projection construction, notification
   suppression, metadata capture and encoding errors; remove obsolete branches.
4. Exercise the bounded matrix below, update architectural guidance and regression
   plan status, freeze exact source/evidence/output hashes, obtain independent
   final review and let root integrate atomically.

## Bounded final acceptance

Bound artifact comparison to **11 complete modules** across these existing or
focused fixture families. Compile/vet each candidate module once; repeat only
generation for determinism.

| Fixture family | Modules | Obligation |
| --- | ---: | --- |
| Current selected-response readiness fixture, extended with array/object and batch cases | 1 JSON-RPC | Constructor-free selection and framed wire/client behavior |
| SelectedBodyDefaultsDSL (HTTP); SelectedBodyFieldsDSL("jsonrpc", "response", "defaulted", false) | 2 HTTP/JSON-RPC | Existing request/default and selected native/named controls |
| SelectedResponseStorageDSL | 1 mixed | Required/optional view, nullable and named storage from #854 |
| Current client-stream final-response fixtures | 2 HTTP/JSON-RPC WS | Proven shared leak and response/notification projection |
| Existing response_metadata_test.go fixture | 1 JSON-RPC | Headers/cookies and unchanged configured-codec boundary |
| jsonrpcViewedUnaryResultDSL | 1 JSON-RPC | Constructor-backed views and invalid view |
| Existing viewed_stream_result_test.go fixture | 1 mixed streaming | Default/narrow stream views and metadata correspondence |
| jsonrpcSuccessResultMemberDSL | 1 mixed | Empty/bodyless result framing preserved |
| Existing sse_mixed_result_module_test.go fixture | 1 SSE | Distinct unary and streaming types preserved |

No exported-design corpus sweep is required. Existing mounted test fixtures can
supply byte manifests through capture hooks; do not invent duplicate designs
merely to collect files.

- HTTP and JSON-RPC unary body cases: ordinary/named selected scalar, empty/zero,
  absent nullable versus null, selected array/object, whole-body and bodyless.
  Required viewed and optional viewed cases reuse the delivered #854 storage
  gate; include two different valid views and an invalid view control.
- Mounted JSON-RPC wire/client: selected output and no excluded sentinel, exact
  request ID, service invocation, headers/cookies, custom encoder observed input
  and invocation count, a mixed batch with one notification, and notification-only
  no-response/no-encoder behavior. Encoding failure remains observable and does
  not send a success with the whole result.
- Mounted HTTP/JSON-RPC WebSocket final results: selected scalar, zero/empty and
  constructor-backed/viewed controls; JSON-RPC outgoing notifications retain
  selected params, responses retain IDs and projection/view agreement.
- SSE control: mixed unary/streaming results retain distinct types and event
  payloads; existing projected SSE event/view checks remain green. Do not apply
  unary selection to the event, and do not claim to repair the rejected DSL cases.
- Compare parent/candidate artifacts, then independent-process repeat. Compile/vet
  each changed complete module once and run selected mounted race checks. Intended
  wire differences are only previously bypassed selected bodies; emitted code
  may centralize equivalent projection expressions. Metadata codec invocation
  counts, proto/OpenAPI artifacts, ordinary whole-body outputs and unaffected
  SSE event payloads remain unchanged. Any additional difference must be explained.

Current evidence: `/tmp/loom-853-final-readiness-tests.log`,
`/tmp/loom-853-websocket-runtime.log`, `/tmp/loom-853-http-websocket-runtime.log`,
`/tmp/loom-853-stream-inventory.log`, and
`/tmp/loom-853-http-stream-inventory.log`. Readiness model and correspondence:
[`response_projection/README.md`](../http/codegen/tla/response_projection/README.md).

## Implementation evidence and intended output differences

The shared `ResponseEncodePlan.BodyValues` map contains the complete expression
for each actual body variant. Both transport generators derive projection from
it. No new selected-field binding, per-scalar fallback, universal serialization
capture or lifecycle owner was introduced. HTTP unary templates consume the map
without changing their generated bytes. JSON-RPC binds the service/viewed source
in a private scope, preserving constructors, view selection and metadata.

The direct plan test failed before population of BodyValues. Mounted ordinary
and empty selected responses failed on the parent for JSON-RPC HTTP, JSON-RPC
WebSocket and HTTP WebSocket; complete generated modules built and vetted first.
Current mounted tests cover selected wire/client responses, empty and zero
values, named/custom/nullable storage, absent optional views, arrays/objects,
metadata headers/cookies, encoder invocation counts and both encode failures,
request IDs, batches and notifications. Existing invalid-view and default/narrow
stream controls remain unchanged behaviorally. Any results use the concrete
`loom.JSONValue` carrier and retain their existing stream regression.

The bounded comparison uses the actual existing renderers through Go compiler
source overlays against the exact implementation parent; the mixed storage
fixture invokes the full generator. Parent, candidate and process-isolated
repeat each contain 182 generated files. Thirteen server files change:

- `http-websocket`: selected final result uses `res.Value`.
- `rpc-defaults`, `rpc-selected`, `websocket` and `storage`: framing consumes the
  selected body, including constructor-free native fields and physical storage.
- `rpc-viewed-unary`, `rpc-viewed-stream` and `success-members`: equivalent
  projections bind the canonical source before framing; view selection and
  bodyless null framing remain covered by mounted controls.

There are no added or removed generated files. HTTP unary, metadata capture and
mixed SSE modules remain byte-identical. Every candidate module builds and vets;
the independent generation repeat matches every candidate byte. No client,
OpenAPI or protobuf file changes. Applications must regenerate affected servers
to receive the corrected existing Body selection contract; no DSL change is
required.

Validation records retained for review: `/tmp/loom-853-plan-before.log`,
`/tmp/loom-853-plan-after.log`, `/tmp/loom-853-final-mounted.log`,
`/tmp/loom-853-final-stream-controls.log`, `/tmp/loom-853-view-controls.log`,
`/tmp/loom-853-boundary-controls.log`, `/tmp/loom-853-build-results.json`,
`/tmp/loom-853-artifact-summary.json` and `/tmp/loom-853-lint.log`. The final
manifest records exact sources, inputs, toolchain, outputs and review parent.
The model README's readiness table separator finding is mechanically corrected.

The six directly affected JSON-RPC snapshots were regenerated through their
owning tests: unary handler init, WebSocket send/full server, SSE object endpoint,
SSE service stream, and the raw aggregate SSE server snapshot. Their only changes
are canonical source/body bindings and an equivalent removed trailing constructor
comma. The SSE primitive/user-type count assertion retains its 2 constructor and
14 identity branches, with the constructor source now named `res`. All owning
source/golden tests pass (`/tmp/loom-853-golden-after.log`).

Checked-in ticktock fixtures were separately copied and generated from the exact
parent, candidate and independent repeat: 35 files per stage, no checked-in/base
drift, HTTP byte-identical, and only JSON-RPC server `sse.go`/`stream.go` gaining
the equivalent shared projection binding. Both candidate apps build, vet and
pass their existing race suites (HTTP 1.470s, JSON-RPC 1.493s). Exact comparisons
are recorded in `/tmp/loom-853-fixture-{summary,hashes}.json`.
The checked-in trees are intentionally not copied into this patch: main now has
the independently delivered #720 fixture design/interface changes. Root must
apply the reviewed production change to that main, regenerate its current
fixture designs and independently review those integration differences before
committing. No old fixture output may overwrite the newer designs.

## Current-main integration evidence

Integrated onto `8f1240831` after independent original-packet approval. Current
#720 ticktock designs were regenerated directly, then generated again in separate
processes. All 39 generated files repeat byte for byte; HTTP output is unchanged,
and only JSON-RPC server `sse.go` and `stream.go` acquire equivalent shared body
bindings. Both fixture modules build and vet, and race tests pass (HTTP 1.947s,
JSON-RPC 2.025s). Generator-only checksum additions were removed from fixture
`go.sum`; no runtime dependency changes are required.

Exact baseline/candidate/repeat hashes are retained in
`/tmp/loom-853-current-fixtures-{base,candidate,repeat}.json`. Integrated mounted
projection tests pass in HTTP, JSON-RPC and the mixed generator fixture; evidence
is `/tmp/loom-853-integration-tests.log`. Final integration review covers this
updated plan, the two regenerated fixture files, and preservation of the newer
stream contracts. The original reviewed production diff remains unchanged.
