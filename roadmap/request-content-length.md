# Request Content-Length publication — #766

Status: independently reviewed, pushed and confirmed closed as `d5919ce6e`.
Final and integration reviews approve the exact change. Inactive output cleanup
is complete; `/tmp/loom-766-cleanup-receipt.json` records preserved evidence.
Initial probes used 8375ce604 plus importer-only #789; the implementation parent
was 7685ae527, including #787 media encoding.

## Contract and evidence

The DSL's SkipRequestBodyEncodeDecode example explicitly maps
`Header("length:Content-Length")`. The raw client accepts io.ReadCloser and must
stream without loading the whole body. RFC9110§8.6 defines Content-Length as a
nonnegative decimal octet count; it cannot be forwarded when known incorrect.
https://www.rfc-editor.org/rfc/rfc9110.html#section-8.6

The original issue reproduction runs unchanged and fails: the actual generated
client/server sends three bytes but the service's mapped Length is nil. Log
`parent.log`, package race2.368s. The four-case expanded module also fails
positive-reader and explicit-zero/empty-reader; zero/http.NoBody and absent
length/nonempty-reader controls pass (`cases.log`,2.374s). This is not a mocked
Doer assertion: both use a real httptest server and net/http transport.

Go1.27.2 `http.Request.outgoingLength` treats a nonnil, non-NoBody reader with
ContentLength0 as unknown. Thus merely assigning the dedicated request field
fixes positive lengths but cannot repair the explicit-zero reader case.

## Shared owner and consumers

- `http/codegen/template_sources_request.go`: one typed header encoder currently
  stores every scalar/list value in Header. It encodes body afterward, including
  form, multipart and configured encoder factories. Keep typed formatting here.
- `http/codegen/misc_sections.go:renderRequestCreation`: attaches raw Body before
  header encoding. Do not add a raw-only framing exception here.
- `http/encoding.go:RequestEncoder` and `http/form.go:SetFormRequest`: may replace
  Body/GetBody and/or ContentLength. Framing publication must run after these.
- `http/codegen/client_jennifer.go`: ordinary HTTP and SSE call the shared request
  encoder before Do; HTTP WebSocket passes Header to DialContext instead of the
  constructed Request. Raw request DSL rejects streaming/gRPC combinations.
- `jsonrpc/codegen/client.go`: reuses HTTP request encoder; unary and SSE setup
  then publishes through Do. WebSocket messages use their independent JSON-RPC
  connection path. Do not claim Request.ContentLength fixes WS frame semantics.
- Server header decoding is already correct when the field reaches HTTP wire.
  Response Content-Length, cookies, trailers and OpenAPI names remain separate.

Implemented correction: shared runtime request-framing finalization, called once
from the shared generated request encoder after body encoding when a designed
Content-Length mapping exists. Parse the effective formatted length, reconcile
known body/framing state, and populate Request.ContentLength; keep absence as a
no-op. The request-plan completion bit records raw ownership or mapped length so
ordinary encoders retain their output. No duplicated raw/form/RPC length policy.
Do not rely solely on the Int64 carrier: String/named integer mappings still
format through the existing typed owner. Invalid syntax, overflow, conflicting
framing and known body mismatch return contextual encoding errors before Do.
For positive unknown streams, retain streaming and let net/http enforce actual
length while writing. Do not silently override caller TransferEncoding.

## Chosen zero and failed-preparation ownership

Root reviewed the technical alternatives and directed bounded EOF verification,
not a new opaque-reader restriction. Go1.27.2 Request.Body explicitly requires
concurrent Read/Close and Close unblocking Read (request.go186-188). Use that
existing contract; do not spawn a detached read/probe goroutine.

One runtime completion owner runs after the shared generated encoder, including
failed preparation. Generated shape is a named error result and deferred
`FinalizeRequestEncoding(req, service, method, err)`. Emit it for raw request
ownership or an actual Content-Length mapping; unchanged ordinary encoders need
no new call. Existing formatted header logic and body encoders remain owners of
their values. The completion owner preserves an existing encoding error identity,
joins cleanup causes, and wraps newly detected framing errors as the existing
contextual encoding error. No independent client endpoint wrappers or duplicated
HTTP/JSON-RPC policy.

A private preparation state captures the current body. Its close-once operation
owns the Close result; the synchronous EOF probe and context.AfterFunc callback
share it. An arbitrary opaque reader is probed for at most one byte (bounded
memory); any byte is a length mismatch, never a successful empty publication.
Wait for Close completion before reading its error or returning, stop cancellation
registration, and retain read/close/context errors together. On success only,
replace Body with http.NoBody and GetBody with an empty replay; preserve positive
and absent GetBody unchanged. A cancellation arriving after preparation still
uses the existing context-bearing net/http Do gate. Do not claim an atomic custom
network operation or bounded behavior for a ReadCloser violating Go's contract.

The same runtime completion state closes an attached raw body on earlier encoding
failure, before Do transfers ownership to the transport. This is an actual existing
failure, not speculative expansion: an invalid X-Tags list causes no request but
Body.Close count0 (`ownership.log`,2.462s). The positive/zero-reader failures also
remain on this refreshed source; NoBody/absent controls still pass. Keep one close
owner so a failed zero probe is not closed a second time by error cleanup. Never
overwrite TransferEncoding to force a declared length; diagnose conflicting
framing before changing request state.

[`RequestFraming.tla`](../http/tla/request_framing/RequestFraming.tla) explores probe/cancel/close scheduling. Current framing,
separate closers, and dropped close-error variants fail distinct invariants. The
owned variant checks close once, proven EOF, retained error causes, safe
publication and cancellation release under explicit fairness/Body assumptions.
The model abstracts only zero-length preparation; runtime/direct/generated tests
cover typed parsing, streaming positives, GetBody and transport integration.

## Bounded acceptance

1. Direct runtime seam: absent unchanged; valid positive, zero/nil and NoBody;
   zero opaque EOF; nonempty zero, invalid/negative/overflow/count syntax;
   known mismatch and transfer-encoding conflict; read/close failure and
   cancellation behavior for the chosen zero policy.
2. Actual generated raw module: original optional Int64 plus required and named
   scalar/string/case-insensitive mapping, nonempty/empty/absent paths; actual
   service sees declared length and exact bytes; errors are observable.
3. One ordinary encoded and one form body control establishes finalization after
   body publication. One JSON-RPC HTTP consumer covers shared header encoder;
   unframed ordinary/SSE controls remain unchanged. WS remains its own carrier,
   no new WS/raw admission policy bundled.
4. Compare bounded parent/candidate artifacts and process-isolated repeat;
   build/vet generated modules, direct/package race checks, scoped lint.
   Reuse existing harnesses and the approved zero-body lifetime model; parsing
   remains deterministic and needs no separate model.

Prior related tickets:710 header token grammar,151 raw OpenAPI annotation,
206/217 inbound request limits,186 response file framing. None fixes this owner.

## Implementation sequence

- [x] Preserve parent mounted failures and passing controls; independently review
  owner proposal and current/rejected/owned lifetime model.
- [x] Implement shared runtime preparation completion with one captured-body close
  owner, retained errors and bounded synchronous EOF verification.
- [x] Record completion need during HTTP request analysis and emit one deferred
  adapter in the shared HTTP/JSON-RPC request encoder.
- [x] Add direct cancellation/progress/failure controls and compiled raw, named,
  required, form/custom/ordinary body and JSON-RPC consumers.
- [x] Compare bounded generated outputs and checked-in transport fixtures against
  archived parent, repeat generation independently and finish affected gates.
- [x] Independent exact final review, atomic delivery, compact evidence cleanup.

## Final bounded evidence

- Full HTTP runtime race suite before the deterministic method-admission extension:
  8.809s. The final focused runtime race suite passes 1.492s, including cancellation
  before read, during read, after EOF, callback completion, read/close causes,
  literal versus joined EOF, and repeated zero-byte successful reads.
- Compiled generated HTTP and JSON-RPC acceptance plus direct analysis/renderer
  controls initially passed race 5.137s / 4.256s. After method admission, all three
  packages pass 1.492s / 8.507s / 5.146s. Generated modules run race and vet;
  the archived parent fails the expected mounted length and cleanup assertions.
- Of 37 bounded generated files, 34 are byte-identical to the archived parent.
  The three intended changes are client `encode_decode.go`: fourteen request
  encoders gain a named error return and deferred completion call. All 37 files
  repeat byte-identically across separate generation processes.
- HTTP ticktock, JSON-RPC ticktock and mixedtick: all 55 generated files match
  parent/candidate/repeat exactly. Candidate fixture modules build and vet.
  Each checked-in `loom.json` already differs in design_digest on the parent;
  no generated Go changes or ticket-specific digest drift occur. These outputs
  remain valid after the runtime-only method admission correction. Keep this
  inherited provenance adoption separate from the framing change.
- Scoped configured lint: zero issues. Model sources/configurations are unchanged
  from the independently approved readiness run (owned 206 generated/137 distinct
  states). Direct runtime tests check correspondence and progress beyond the
  model's finite scheduling assumptions.

Recovery evidence is frozen in `/tmp/loom-766-final-manifest.json`, including
parent inputs, generated artifact hashes, command logs and the exact final patch.

## Method representation decision

A generated-client witness against the intermediate candidate reproduced silent
zero-header loss for GET, HEAD, DELETE and OPTIONS over both real HTTP/1 and
negotiated HTTP/2 (8 failures). The other 34 states—positive/absent lengths and
POST/PUT/PATCH zero—passed. Pinned Go's HTTP/1 transferWriter and HTTP/2 shared
header encoder both emit zero only for POST/PUT/PATCH in the standard framing
configuration. Native Request.Write controls agree.

The shared finalizer now rejects an explicitly supplied zero on other methods
before probing or Do, retaining one-close failure cleanup. Empty method has Go's
default GET semantics; method spelling is case-sensitive. Absence and positive
lengths retain their existing behavior. This prevents silently discarding authored
presence. No custom transport, header bypass, server inference or new DSL
admission rule is introduced. Applications should make the mapping optional and
omit its value for an empty body on those methods; regenerate clients. The
tradeoff is an actionable encoding error for a previously lossy request rather
than an unfaithful successful send. This deterministic admission needs no new
concurrency model; the existing zero-probe model applies after method admission.

The original intermediate packet is preserved as
`/tmp/loom-766-before-method-manifest.json`; it is not final approval evidence.
