# JSON-RPC bodyless result binding — #868

Parent: `56a78637a`. Scope: endpoint result consumption must follow the effective
response body plan, not the existence of a named service type.

1. [x] Reproduce existing relocated-empty design compile failure in both generator
   and JSON-RPC module tests; classify HTTP/gRPC stale snapshots separately.
2. [x] Add failing direct binding and generated named-empty tests, with local and
   relocated types, payload/no payload and ordinary/no-result/metadata controls.
3. [x] Model current Ref-based binding and shared-plan binding in the existing
   response-projection model family; record compile/source-use correspondence.
4. [x] Make endpoint invocation consume the existing framing HasBody owner;
   retain metadata capture, WebSocket forwarding and success/error rendering.
5. [x] Run mounted null/error/notification and codec/metadata checks, existing
   selected projection and WebSocket controls, bounded generation parity and lint.
6. [x] Freeze exact final packet for independent review.
7. [ ] Independent approval and root integration.

`writeJSONRPCEndpointInvoke` currently binds `res` whenever `Result.Ref` is
nonempty; `responseBodyInit` correctly uses `EncodePlan.HasBody`. Named empty
objects have a reference but no transport body, leaving an unused local. The
shared HTTP response plan already owns body consumption. JSON-RPC should derive
both binding and bodyless success from that plan; capture paths still consume the
result for headers/cookies, and WebSocket forwarding still returns it. Do not
silence an unused local or restore raw-result framing.

Existing repro commands and CI evidence are in the #868 report. The temporary
captured module `/tmp/loom-ci-866-usertype` proves that the unused variable is the
outer endpoint result, not the private projection binding. Model expansion is
limited to binding versus actual consumption; protocol timing remains unchanged.

Implementation refinement: existing success rendering already obeys HasBody.
Its two established paths differ in error-observation handling, so no success
routing change is needed or retained. Only endpoint invocation stops using the
competing Result.Ref rule. Initial test assumptions of notification status204 and
two metadata-only codec calls were corrected to the existing status200 and one
codec call: the bodyless HTTP encoder writes metadata without requesting a JSON
encoder. These controls preserve existing behavior rather than introduce policy.

Final bounded evidence: four direct parent failures and four generated unused
locals; candidate compiles/vets and passes 21 mounted cases under the race
checker. Existing user-type generator/module regressions and selected-body,
notification, metadata and WebSocket-forwarding controls pass (JSON-RPC 10.024s,
generator 2.880s). Parent/candidate/repeat each contain 12 generated files; the
only difference is four endpoint result bindings in one server file. Independent
repeat is byte-identical. No schemas, shared response expressions or physical
body declarations change. Scoped lint reports zero issues; structural gates pass.
The exact source/evidence manifest is `/tmp/loom-868-final-manifest.json`.
