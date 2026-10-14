# Error response ownership (#908)

Status: implementation and bounded acceptance complete; original final review
approved, refreshed exact diff awaiting narrow review.
Readiness was independently approved in
`/tmp/loom-908-independent-readiness-verdict.json`. The preserved readiness parent
was `b9e51df2804339fca3f67339e8bc2639536fb0b7`; implementation was safely refreshed
to `c80c5b3595c772018ad9ba3607e36dd985bdc398`, including delivered #905.
The original generated JSON-RPC witness still fails on that refreshed parent.
Final review approved that candidate without findings. A subsequent preserved
exact-patch inverse/clean fast-forward/reapply moved it to
`16749ea5bb4d52539f2acafc7875c48e8275996a`, which delivers #911/#912.

## Contract and shared cause

`docs/error-handling.md` defines independent joins as complete failures, not a
license to publish one contribution's contract. Single-cause wrappers are
transparent. An explicit outer named error owns its response, including when it
wraps a join. `ResolveErrorMessage` already follows this boundary; HTTP and
JSON-RPC code/body/metadata selection instead performs separate tree-wide
`errors.As` searches. The original generated JSON-RPC witness returns code 4403
and a private typed body from one joined branch while the message is generic.

Correct one wire-response ownership decision in `pkg`, consumed by default
runtime encoders and generated HTTP/JSON-RPC adapters. Keep diagnostic extraction
helpers (`ExtractErrorRemedy`, `ErrorStatusCode`) and application error trees
unchanged. They have public standalone uses; changing their general traversal
is not required to fix wire selection. Generated code must inspect the resolved
owner directly, not run `errors.As` on that owner and rediscover a different
contract beneath it.

A safe-message-only wrapper remains a message override and does not suppress the
valid typed body of its single underlying error. Public message selection and
body ownership are distinct results of one bounded single-cause inspection.
Explicit wrapper remedy metadata may describe the complete failure; independent
branch remedy metadata may not. The first explicit remedy on the permitted spine
retains its current nil-shadowing behavior. An explicit named owner ends body
and trait lookup: no inner ServiceError ID, retry trait or concrete typed body is
borrowed when the outer owner lacks it.

## Owner and consumers

| Boundary | Correction / preserved contract |
| --- | --- |
| `pkg/error_remedy.go` public message resolution | Reuse its single-cause/independent-join rule in the response owner; retain SafeMessage override and history flag. Keep general extraction helpers separate. |
| `http/error.go` default ProblemResponse | Read ID, code, traits and retry metadata from the resolved response contract; unowned joins use ordinary generic fault/status. Explicit caller type/title/status overrides remain authoritative. |
| `http/codegen/template_sources_response.go` | Resolve once before mapped name/body/header/cookie selection; direct concrete-owner assertion. Keep original error for logging/formatter hooks. |
| `jsonrpc/error_data.go` | Emit only metadata owned by the complete response; unowned joins have no child ErrorData. |
| `jsonrpc/codegen/handler_sections.go` | Unary and pre-stream failure mappings use the resolved name/body; original error retained for ReportFailure. |
| `jsonrpc/codegen/stream_sections_websocket.go` | Shared stream mapped-error writer and raw stream error source use the same owner, covering SSE and WebSocket request/reply/bidi branches. |
| `grpc/error_contract.go` and `grpc/error.go` | Preserve existing unanimous aggregate code, explicit GRPCStatus, context termination and mapped conversion policy. Existing resolver is the reference for join ordering. Any adoption must retain native recognition before cause traversal; do not substitute an HTTP policy. |

The intended public semantic result exposes the selected concrete named owner
and response metadata/message, rather than a general graph visitor. Keep traversal
mechanics private unless the existing gRPC classifier genuinely needs the same
single-cause split primitive; do not export a callback walker for sibling reuse.
Concretely, a shared semantic response result records the first direct named
owner encountered along the single-cause spine and the explicit remedy observed
on that permitted spine (including a present nil remedy, so it does not fall
through to a cause). Traversal ends at that owner, an unknown leaf, or a join with
multiple nonnil causes. No independent branch is visited. Message selection
retains its existing separate SafeMessage precedence: a message-only wrapper does
not become the typed owner. A generic result has no concrete owner to extract.
The final API should expose only these wire semantics, not graph traversal.
HTTP and JSON-RPC do not need `ErrorStatusCode`: the caller inventory finds no
transport use, so its standalone extraction behavior remains unchanged.

The existing gRPC classifier already resolves its native status and aggregate
code before detail emission. Keep it as a bounded regression consumer rather
than rewriting its policy. Its generated mapper still uses concrete `errors.As`;
the actual generated explicit-outer/same-name case reproduced InvalidArgument
where the outer fault requires Internal. The generated mapper now directly
asserts the shared resolved owner, rather than borrowing its descendant body.
The existing native/context/aggregate classifier and public mapper callback input
are unchanged; the corrected mounted case and all existing controls pass.

Implementation must remove the superseded name/body/ServiceError wire searches,
not add an early join check in each renderer. No schema or design rejection,
wire wrapper, compatibility mode, or client decoder change is needed.

## Concrete acceptance

- A typed error, an ordinary `%w` wrapper and a one-nonnil-child join retain their
  designed code/body and the configured public message.
- A typed error joined with an unrelated error, in either order and under a
  transparent wrapper, emits generic HTTP/JSON-RPC failure data. No branch code,
  ID, retry flags, headers/cookies or typed body escapes.
- An explicit named outer owner over a join wins. An unknown outer name may not
  select a recognized inner name. An outer named custom error may not acquire
  inner ServiceError metadata. A same-name but different concrete outer owner
  cannot satisfy its body mapping by finding a descendant type.
- SafeMessage-only outer wrappers retain the underlying singular typed body;
  safe metadata inside an unowned join is not promoted. Custom `As` methods
  cannot synthesize wire ownership by impersonating a descendant contract.
- Existing gRPC same-code/mixed-code joins, native status, conversion failure,
  context termination, wrappers and explicit owners remain unchanged.

## Sequence and bounded validation

1. [x] Preserve original generated unary witness and add direct runtime red cases
   at existing HTTP ProblemResponse and JSON-RPC ErrorData seams, with ordinary,
   single-cause and explicit-owner controls.
2. [x] Extend the existing `pkg/tla/error_disclosure` model with response owner,
   mapped body and metadata selection. Check current first-branch selection fails;
   checked ownership preserves branch-order independence and outer precedence.
   Reuse the gRPC error-contract model; no concurrency/message-context model is
   required for this deterministic ownership correction.
3. [x] Independent readiness review before production.
4. [x] Implement shared response resolution and remove independent wire-selection
   paths. Coordinate renderer composition with #905.
5. [x] Run direct package regressions and existing generated HTTP disclosure and
   JSON-RPC typed-error/stream harnesses. Cover HTTP, unary RPC, SSE and WS actual
   mapped-error paths, full/selected bodies and mapped metadata; reuse existing
   fixtures. Compare ordinary outputs, list intended resolver substitutions,
   independently repeat generation, compile/vet bounded generated modules.
6. [x] Regenerate only affected checked-in fixtures, preserve public guidance and
   migration (regenerate clients/servers where emitted server code changes).
   No full corpus.
7. [x] Freeze exact source/evidence for independent final review; original
   candidate approved without findings.
8. [ ] Narrow review of refreshed parent, plan and regenerated API-primitive
   encoder golden; delivery remains owned by the orchestrator.

The model assumes finite acyclic error graphs and abstracts opaque application
methods. Tests check callbacks, actual concrete extraction and serialized wire
fields. Error causes remain available for diagnostics; default wire ownership
never promises to sanitize a custom formatter deliberately supplied by an app.


## Readiness evidence

Current direct race checks reproduce generic HTTP 503/branch traits and RPC
branch metadata for both join orders and transparent outer wrappers. Direct,
wrapped and one-child controls pass; a custom named outer error also wrongly
borrows the nested ServiceError metadata today. The generated HTTP design returns
403 instead of 500 for both join orders (1.759s); ordinary mapped controls pass.
The original generated RPC witness independently reproduces code 4403 plus typed
branch data on this worktree (1.945s). These are preserved parent results, before production changes.

The response model reproduces current selection (43 distinct states), rejects
blanket wrapper rejection (39 distinct states), and checks the proposed owner
(64 distinct states). Its two branch orders represent distinguishable public
branch owners; the runtime repro also checks a public branch plus an unknown
cleanup failure. Evidence hashes and commands are in the readiness manifest.

## Current implementation evidence

The shared `pkg.ResolveErrorResponse` result records the concrete named owner,
name and permitted remedy; private `singleErrorCause` is also used by message
resolution. HTTP, JSON-RPC unary/SSE/WebSocket and the proven generated gRPC
mapper consume this result with direct concrete assertions. The focused runtime
and generated transport race checks pass, including branch-order controls.
Selected-body/header/cookie controls, exact generated-output and fixture
comparisons, guidance and lint are complete. Narrow parent-refresh review remains. JSON-RPC
keeps its delivered body-only response admission; mapped header/cookie acceptance
is HTTP-only.

## Bounded final comparison and limitations

The parent/candidate/process-repeat matrix contains 12 actual generated modules
and 219 generated artifacts per stage. All candidate/repeat bytes match; 22 files
change only server error-owner selection, direct concrete extraction, original
error formatting/problem normalization, and now-unused imports. Ordinary client,
service, types, codecs and unrelated server output remain byte-identical. The
matrix includes custom/full/selected error bodies, problem headers/cookies,
mapped/unmapped unary/SSE/WebSocket paths, and the gRPC explicit-outer case.
The three checked-in transport fixtures also repeat exactly, with only five
server files changing. Their parent and candidate modules build and vet.
All four affected runtime packages pass race checks; scoped configured lint has
zero issues. A separate mounted assertion checks formatter input identity.

Two preserved original-parent failures were independently repaired by #911/#912:
`TestSelectedProblemBodyConstruction/explicit_detail_object` rejects a documented
pruned built-in problem body, and `TestEncodeError/api-primitive-error-response`
panics before generation. Both reproduce on archived parent c80c5b359. The former
is a reconstruction-model regression, not an invalid fixture; neither was changed
or counted as passing in the original final packet. After their separate delivery,
the unchanged explicit-object mounted check passes and the API-primitive golden
is regenerated with only the same reviewed owner/formatter substitutions. Both
previously blocked rows pass on the refreshed candidate. The other 20 problem
selections and 13 encoder golden cases retain their earlier passing evidence.
No unaffected artifact or fixture comparison was repeated.
