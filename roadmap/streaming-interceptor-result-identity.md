# Streaming interceptor result identity (#859)

Readiness independently approved; implementation complete, final review pending.
Readiness parent: `8f1240831c2df98d80c4b65f760925d805188375`.
Implementation parent: `b2b6f1480c77036d7afae324e972307aebf0047d`.
The clean tracked worktree advanced by fast-forward, preserving all readiness inputs.

## Sequence

- [x] Trace the accepted result boundary and affected consumers.
- [x] Reproduce rejected distinct fields and a compiled runtime type mismatch.
- [x] Model current selection and partial alternatives, then check shared ownership.
- [x] Freeze a bounded acceptance matrix and exact readiness evidence for review.
- [x] After readiness approval, implement the shared owner and regression tests.
- [x] Compare base/candidate/independent-repeat generation, compile and run selected clients.
- [x] Update documentation/model correspondence; freeze final diff for fresh review.
- [ ] Parent integrates one reviewed atomic commit and closes #859.

## Contract and owner

`ReadStreamingResult` and `WriteStreamingResult` describe the value crossing the
stream result boundary. For a distinct authored `StreamingResult`, fields,
optional-pointer semantics and nominal Go identity come from that attribute.
Ordinary result streaming still uses the existing result identity. HTTP/gRPC
client streams retain the single final result at SendAndClose/CloseAndRecv.
JSON-RPC streaming-payload-only request/reply methods retain their ordinary
result boundary. Existing rejection of ordinary result access on result streams
is unchanged; this ticket does not introduce dual unary interception for mixed
methods. No public policy remains unresolved.

The semantic attribute selection belongs to `expr.MethodExpr`, shared by
interceptor validation, accessor attribute analysis and service stream analysis.
Codegen must derive accessor nominal references from the existing canonical
stream operation data, not independently choose unary ResultRef. Preserve
MethodData.StreamingResultRef's existing mixed-result presence semantics.

Affected seams: expr/interceptor.go validation; codegen/service/
service_data_interceptors.go attribute collection and MethodInterceptorData;
service_data_methods.go canonical stream attribute analysis; generated accessor
structs and client/server accessor casts. Transport emitters already consume the
canonical stream result and should not need a second selection rule.

## Boundaries

#785 owns JSON-RPC server wrapper shape and request/reply endpoint carriers.
#726 owns gRPC operation context/lifetime. Neither is repaired here. The #720
operation/Close contract remains unchanged. Use client-only JSON-RPC mixed
interceptors to isolate #859; exercise HTTP server wrappers separately. Existing
first-method accessor signature aggregation and unrelated mapped-field naming
are not redesigned unless direct #859 evidence requires it.

## Concrete implementation sequence

1. Add a narrowly named MethodExpr resolver (`EffectiveStreamingResult`) that
   returns a nonempty explicit StreamingResult, otherwise Result. Its contract
   resolves value identity, not stream eligibility; existing IsResultStreaming
   and hasStreamedFinalResult remain the eligibility owner. Preserve prepared
   aliases and direct unprepared expr tests. Nil/Empty authored cases retain
   the existing service-analysis fallback; absent/non-object access still fails
   normal validation, never by a generator panic.
2. Route streaming accessor validation and attribute collection through that
   resolver. Route buildStreamingResultData and the sent-type identity selection
   through it as well, removing their independent selection predicates while
   preserving the distinct-result marker, example scope and ValueData plan.
3. In buildInterceptorMethodData, use the analyzed canonical client receive type
   for StreamingResultRef whenever there is a result stream. Both server and
   client accessor values use that concrete type. Keep server event-interface
   adjustments and existing wrapper operation data untouched. Do not populate
   MethodData.StreamingResultRef for ordinary streams merely to reuse that field:
   other consumers intentionally interpret its presence as a mixed result.
4. Extend direct tests, a bounded generated fixture and mounted harness, then
   document mixed stream accessor selection and regeneration. Update the
   framework owner guidance and model index/correspondence. Application code
   must regenerate and adapt incorrectly typed accessor implementations; there
   is no compatibility cast or fallback to unary fields.

## Readiness evidence

- Four direct read/write cases fail at the parent: distinct `chunk` is rejected;
  shared `value` selects string instead of int and Summary instead of Chunk.
- One generated module contains independent HTTP and JSON-RPC SSE services.
  `go mod tidy` and `go vet ./...` pass. Both mounted unintercepted controls pass;
  both typed intercepted streams panic in ClientStreamingResult on Chunk to
  Summary assertions. The runtime harness expects safe read/mutation and EOF.
- Existing expr result-boundary matrix passes (0.442s); selected interceptor
  goldens, rejected boundaries and custom-package stream controls pass (0.307s).
  Existing gRPC plain/viewed final-result modules and JSON-RPC request/reply
  unary interceptor module compile/vet/runtime controls pass (10.039s total).
- Six model configurations reproduce current and partial-repair failures, then
  check the coherent owner (21 states). See the durable model README.
- Source search in the local monorepo found application-facing streaming
  interceptors in the HTTP ticktock fixture. Loom-MCP progressive-discovery uses
  ordinary StreamingResult without these accessors. No external application
  using distinct Result + StreamingResult + accessors was found; do not claim
  production adoption. Existing HTTP MixedResultsDSL and JSON-RPC LocalBoth
  fixtures establish the accepted distinct-result contract that this witness
  combines with the existing interceptor capability.

## Bounded final acceptance

Run direct expr/service regressions and affected package tests. The selected
artifact matrix is finite; do not run the full exported-design corpus.

| Generated selection | Obligation and expected base/candidate difference |
| --- | --- |
| Mixed shared-field HTTP + JSON-RPC SSE, client interceptors (current module) | Base builds, mounted accessor panics; candidate preserves stream types/wire artifacts and changes only accessor stored/cast type to Chunk. Mounted read/write, EOF and Close pass. |
| Mixed disjoint fields and optional scalar (same two transports, one module) | Base rejects valid streamed fields; candidate accessor fields, pointer handling and references come from Chunk; reject unary-only and unknown streamed fields. |
| Mixed HTTP SSE, server + client interceptors (one module) | Actual ServerStreamingResult and ClientStreamingResult mutate the streamed Chunk; keep unary Summary shape, operation methods and lifetime unchanged. JSON-RPC server role remains #785. |
| Mixed external-package streamed type + ordinary streamed method sharing one interceptor (one module) | Package-qualified refs and multi-method dispatch use their own nominal values; accessor field types consistent across methods. Base/candidate only mixed accessor identity changes. |
| Existing HTTP/gRPC final-result plain, dynamic/fixed-view fixtures | Preserve canonical final result and SetView; selected interceptor goldens byte-identical. Existing generated HTTP final-result runtime fixture and the two gRPC compile modules stay green. |
| Existing ordinary streaming and unary/request-reply controls | Selected service interceptor goldens byte-identical; JSON-RPC client-stream unary module remains green. Existing ticktock typed-client tests remain unchanged and pass. |

Compare complete affected generated artifacts at the parent and candidate where
parent generation succeeds. For rejected parent designs, preserve exact failure
and compare two independent candidate generations. Compile/vet each distinct
candidate module; an independent-process repeat needs byte comparison, not
repeated build/vet. Inventory every generated difference. Direct tests also
cover no stream, primitive result rejection, read/write symmetry, Empty/absent
fallback and ordinary Result access remaining rejected on result streams.
Run six models, focused scoped lint, and diff checking before final review.

## Separate residual: mixed unary wrapper dispatch

Generated mixed NewClient wraps both ordinary and streaming endpoints with the
streaming wrapper. A compiled unary endpoint returning Summary then panics on
its WatchClientStream assertion before any accessor executes; unintercepted
calls pass. This affects both HTTP and JSON-RPC generated service clients.
Retain `/tmp/loom-859-unary-dispatch-harness_test.go` and its log for root review.
#785 specifically reports JSON-RPC server operation/interface and endpoint
carrier defects; it does not describe this mixed unary client dispatch.
Do not claim it fixed by #859 or silently bundle it. Root independently reproduced and filed it as [#867](https://github.com/CaliLuke/loom/issues/867).

## Final implementation and checks

`MethodExpr.EffectiveStreamingResult` now owns the shared attribute selection.
Validation and field analysis use it; service analysis uses the same attribute
for nominal result collection and send identity. Interceptor nominal casts use
canonical `ClientStream.RecvTypeRef`. Stream eligibility, unary ResultRef,
MethodData's mixed-result marker, JSON-RPC server event interfaces, endpoint
wrapper dispatch and lifetime behavior remain unchanged.

All four original direct cases now pass. Extended direct tests cover optional
fields, read/write symmetry, unary-only and unknown field rejection, primitive
rejection, unprepared/empty fallback and absent-result validation. Full expr and
service package suites passed in 0.732s and 33.964s. A final helper-only test
adjustment reused `expr.RunInvalidDSL` (the API returning just the needed error),
and the affected validation test passed again in 0.459s.

The generated matrix contains six designs: shared mixed HTTP/JSON-RPC clients,
disjoint optional mixed clients, HTTP server/client wrappers, external-package
mixed and ordinary method dispatch, existing HTTP final-result views, and
ordinary HTTP/JSON-RPC streaming. Five parent modules compiled/vetted; the
sixth was rejected for the expected four missing-field diagnostics. All six
candidate modules compiled/vetted; four execute their new harnesses under the
race detector. There were 33 successful tidy/vet/test commands total.

Exact artifact inventories: parent 125, candidate 160, independent repeat 160.
Candidate and repeat match byte for byte. The only changes to artifacts the
parent could generate are:

- `shared/gen/events/client_interceptors.go`: stored result and both accessor
  casts change from Summary to Chunk.
- `shared/gen/rpcevents/client_interceptors.go`: the same concrete cast repair.
- `server/gen/events/service_interceptors.go`: both roles select Chunk.
- `external/gen/events/client_interceptors.go`: the mixed method selects
  `chunks.Chunk` and imports its owning package; the ordinary method is unchanged.

The optional design adds 35 candidate artifacts because its parent rejects the
authored fields. Its accessor getters read nil as zero and setters allocate the
optional value; both generated typed clients preserve the actual Chunk pointer.
All ordinary, final-result and transport artifacts are otherwise byte-identical.
Existing checked-in interceptor goldens pass without edits.

Additional controls: HTTP final-result runtime 15.269s; gRPC plain/viewed final
results and JSON-RPC request/reply boundary 8.881s; HTTP and JSON-RPC ticktock
typed-client controls 0.831s and 0.774s. A harness initially compared exported
method names to lowercase specimen values; correcting those test values left
production unchanged. The external harness passed in the captured race suite
and through its owning test in 5.508s. Six model configurations retained their
expected failures/pass; scoped pinned golangci-lint v2.14.0 reports zero issues.

The final packet records exact source, generated artifact, toolchain, command
result and model hashes. No broad corpus audit was run. Final independent review,
one atomic commit, push and issue closure remain with the orchestrator.
