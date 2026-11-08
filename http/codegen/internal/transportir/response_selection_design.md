# Status-Range, Fallback and Informational Response Contracts (#976)

> **Historical design record.** This is the accepted revision-4 design,
> moved unchanged from `roadmap/` when the #976 execution plan was retired. It
> keeps the rationale, rejected alternatives, proof obligations and review
> history. The public documentation linked from the
> [acceptance record](response_selection_validation.md) is normative for
> current behavior; the record also holds qualification evidence. Statements
> below about current source, prerequisites, delivery or "the plan" describe
> the state when they were written. The retired
> plan was created on 2026-11-05 from this design at `f0c7a2b8c`, and was
> reconciled against #1014 `4d5daba25d622d881a9ff1cdce35602880b5d0d4`. Its
> delegated decisions are recorded in sections 4, 7 and 8.

Status: independently reviewed design, revision 4: full ticket scope; all
findings resolved and ready for execution planning. Bounded TLA+ models, Lean proofs and a standalone
net/http scratch proof were run (see Proof Obligations). Not an execution plan
and distinct from its execution plan. This design-only status is historical;
the user's later completion direction authorizes the work, subject to the
prerequisite sequencing in that plan. Citations are to the working tree at
`f0c7a2b8c`. Go behavior is from the toolchain pinned in `go.mod`
(`go 1.27.3`).

## Current-source reconciliation proposal

Preparation only against committed #1014 source
`4d5daba25d622d881a9ff1cdce35602880b5d0d4` (2026-11-06). The revision-4
contract, examples and historical proof results below are retained. This
reconciliation changes implementation loci, not product semantics. #957 and
#1005 are not delivered. The execution plan now distinguishes a complete,
immutable, independently source-reviewed development baseline from the actual
Git parent required for delivery. Isolated implementation may use that verified
prospective composition after a fresh inventory; mutable or incomplete
prerequisites do not qualify. All four prerequisite commits and their publication
gates remain mandatory before final commit review. This changes sequencing only,
not the revision-4 contract, examples, validation scope or atomic delivery.

Current owner citations in this section and the adapted implementation text
refer to that commit. The original Current State and proof citations remain
historical. #957 API details below come from its separately reviewed candidate,
not committed main; confirm them on the selected frozen baseline and again on
the actual delivery parent.

Later source-backed preflight inventories (response binding, selector
import/export and response lifecycle) read an older frozen prospective #957 +
native/importer checkpoint. They found no product-contract change. Their
implementation obligations are folded into sections 3, 4 and 7-10, Proof
Obligations and the plan. They are historical preflight evidence, not source
readiness. Descriptions of "current" source behavior in those additions are
historical snippets from that checkpoint. Re-inventory them against the final
complete, source-approved composition and confirm them on the actual delivery
parent.

## Summary

OpenAPI response keys select a response three ways:

- an exact code (`404`);
- a class (`1XX` to `5XX`);
- the fallback `default`.

An exact key takes precedence over a range (OAS 3.2 §4.16). Loom stores one
concrete `StatusCode int` per response, and the importer rejects both
non-exact forms.

This design adds one shared selector value and one selection rule. The server
encoder, client decoder, SSE streams, OpenAPI projection, runtime contracts and
importer all use them. It also separates the three HTTP response phases:

| Phase | Codes | Rule |
|---|---|---|
| Final | `200`-`599`, `2XX`-`5XX`, `default` | Exactly one per exchange, selected once, decoded terminally |
| Informational | `100`, `102`-`199`, `1XX` | Zero or more interim responses before the final one (RFC 9110 §15.2). Header-only, emitted and observed through a separate typed channel |
| Upgrade | `101` | Terminal protocol switch, only on the existing WebSocket handshake |

Other decisions:

- One writer owner per exchange serializes interim publication against every
  final start: unary result, error, SSE open, upgrade.
- A non-exact response owns its runtime status through a `StatusField`
  transport source.
- Several success selectors on one operation are selected by that status.
- A fallback-only operation's `default` is its sole success.
- An error-only operation has no success response, and none is invented.
- SSE success statuses come from typed handshake metadata, never from events.

## Problem Statement

The ticket probe (`2XX` plus `default`) fails with `response-status` and
`default-response`. The probe is synthetic; no consumer document with range or
fallback keys was inspected. A valid OpenAPI contract class cannot be
expressed, and concretizing `2XX` to `200` or `default` to `500` would misroute
real statuses.

Inspection found existing defects the design must also fix:

- **Exact 1xx in the DSL.** `Response(StatusEarlyHints)` generates
  `w.WriteHeader(103)` as a final response
  (`http/codegen/template_sources_response.go:49,268`). net/http treats 1xx
  (except 101) as interim, keeps the header map, and later commits an implicit
  200 (`net/http/server.go:1223-1236`). The client transport consumes the 103,
  so the generated `switch` sees 200 (`net/http/transport.go:2711-2729`). The
  `phases-info-final` control reproduces this.
- **Exact 1xx on import.** The importer accepts exact `1xx` keys
  (`internal/openapiimport/analyze.go:99,281`). They are not in the success
  class (`plan.go:370-381`), so they render as errors such as `Status103`.
- **Staged raw-body writer.** It drops `WriteHeader` below 200
  (`http/handler_lifecycle.go:278-283`).
- **Mixed exact keys (reproduced).** A design with `Response(StatusNotFound)`
  and `Response("missing", StatusNotFound)` evaluates and generates. The
  generated client then fails to compile: `duplicate case http.StatusNotFound
  (constant 404 of type int) in expression switch`. The OpenAPI output merges
  both under one `anyOf`. Probe: `/tmp/loom-976-collision-probe` (`gen.log`,
  `build.log`).
- **Hardcoded SSE 200.** The SSE server writer commits a hardcoded 200
  (`http/sse_writer.go:274-283`). The SSE client accepts only 200
  (`http/codegen/client_jennifer.go:240-255`) and decodes each event through a
  fabricated `http.Response{StatusCode: 200}`
  (`http/codegen/stream_sections_sse_client.go:52-66,84,151`). An unopened
  stream with a nil service error falls back to net/http's implicit empty 200
  (`http/codegen/template_sources.go:126-138`).

## Goals

- Represent exact, class and fallback selectors in the DSL, and project them
  back to the same keys in both output targets.
- One precedence rule everywhere, independent of declaration order, with no
  fallthrough after selection.
- Runtime status ownership as a first-class transport source for unary
  responses, SSE handshakes and informational responses.
- Multi-success, fallback-only and error-only operations.
- Informational responses as declared, emitted, observed and validated interim
  responses, serialized against every final start.
- Import `Responses.default` and every `1XX`-`5XX` key, keeping schemas,
  headers, error identity and component provenance. Update the ledger.

## Non-Goals

Each of these is undefined or invalid, not deferred:

- Lowercase `2xx`, `20X`, `6XX` and `0XX`. They fall outside the OpenAPI
  response-key pattern and §4.16's list of allowed ranges.
- Content on informational responses. RFC 9110 §15.2: a 1xx response "cannot
  contain content or trailers".
- Importing `101`. OpenAPI does not describe the upgraded protocol. The DSL
  WebSocket path keeps exporting `101`
  (`http/codegen/openapi/internal/ir/document.go:238-244`).
- Non-exact selectors on JSON-RPC (transport status is always 200), gRPC
  (gRPC codes) and `FileResponse` (`ServeContent` owns its statuses).
- Non-exact WebSocket success. The handshake is always `101`.
- OpenAPI 3.0 output. Loom emits 3.2.0 and, on request, 3.1.1. Import reads
  3.0, 3.1 and 3.2 documents, all of which define `default` and the five
  ranges.

## Current State (verified at `f0c7a2b8c`)

- Importer:
  - `concreteHTTPStatus` (`analyze.go:99`).
  - `response-status` and `default-response` diagnostics (`analyze.go:281-292`).
  - Ledger rows: `Codes` conditional, `Default` rejected
    (`grammar_coverage_test.go:571-572`).
  - Exactly one success response is required (`render_media.go:169-198`).
  - Errors are named `"Status"+status`, with operation prefixes for shared
    names (`render_media.go:240`, `plan.go:104-118`).
- Expression:
  - `HTTPResponseExpr.StatusCode int` (`expr/http_response.go:88-90`).
  - No body for 1xx/204/304 (`:589-599`).
  - Success statuses are unique (`expr/http_endpoint_validate.go:113-127`).
  - No rule rejects success and error sharing a status (reproduced above).
- Tags and reconstruction:
  - `HTTPResponseTag` (`expr/http_response_tag.go:8-28`).
  - `OrderedResponses` (`expr/http_endpoint.go:166-182`).
  - The client assigns the tag (`template_sources_response_decoder.go:46,72`).
  - `responseReconstructionBindings` (`expr/http_reconstruction.go:43-62`).
- Server lifecycle:
  - Encoders call `WriteHeader` with a literal status.
  - `EncodeResponse` clones and restores headers before the error encoder.
  - `encodeError` retries once, then `handleFailure`
    (`http/handler_lifecycle.go:115-130,227-249`).
  - The unmapped status comes from `http/error.go:173-196`.
  - `CaptureResponse` treats 1xx except 101 as non-committing
    (`observability/transport/response_writer.go:36-44`).
- Client: `switch resp.StatusCode` with exact cases. `loom-error`
  disambiguates shared error statuses. Anything else is `ErrInvalidResponse`
  (`template_sources_response_decoder.go:33-136`).
- OpenAPI: keyed by `strconv.Itoa(StatusCode)`; same-key `anyOf` merge
  (`http/codegen/openapi/internal/ir/document.go:231-279`).
- Runtime contracts: `ResponseContractCase.StatusCode` checked by equality
  (`http/response_contract.go:35-36,163-164`).
- net/http, server side:
  - 1xx except 101 is written with the live header map and does not clear it.
  - It is not suppressed for HTTP/1.0 (`server.go:1223-1236`).
  - Writing `100` or any status of 200 or more disables the automatic
    `100 Continue`.
  - A body read under `Expect: 100-continue` sends one automatic `100` on the
    connection writer.
  - HTTP/2 treats 101 as informational (`internal/http2/server.go:2866-2885`).
- net/http, client side:
  - 1xx except 101 is consumed and reported through `Got1xxResponse`.
  - A delivered interim response resets the header budget; the caller then
    bounds them (`transport.go:2711-2729`; `internal/http2/transport.go:2248-2276`).
  - `httptrace.WithClientTrace` composes hooks with `compose2to1`. That calls
    the new hook, discards its error, and returns the previous hook's result
    (`net/http/httptrace/trace.go:174-194,235-246`). This is open upstream
    issue golang/go#76669; the linked CL 741500 is documentation-only.

## Proposed Design

### 1. Shared selector and phases

`expr` owns the selector. `http` gets a dependency-neutral copy of the
selection functions for generated code and runtime contracts.

```go
type HTTPStatusClass int // Status1XX..Status5XX = 1..5; StatusFallback = -1
type HTTPStatusSelector struct { Code int; Class HTTPStatusClass } // exactly one set
type HTTPResponsePhase int // PhaseInformational, PhaseUpgrade, PhaseFinal

func (s HTTPStatusSelector) Exact() (int, bool)
func (s HTTPStatusSelector) Phase() HTTPResponsePhase
func (s HTTPStatusSelector) OpenAPIKey() string // "404" | "4XX" | "default"
func ParseOpenAPIStatusKey(key string) (HTTPStatusSelector, error)
func SelectFinal(keys []HTTPStatusSelector, code int) (HTTPStatusSelector, bool)
func SelectInformational(keys []HTTPStatusSelector, code int) (HTTPStatusSelector, bool)
```

| Selector | Phase | Client selection domain | Server authored emission domain |
|---|---|---|---|
| exact `100` | informational | `{100}` | none (`Expect: 100-continue` handling owns 100) |
| exact `102`-`199` | informational | `{code}` | `{code}` |
| `1XX` | informational | `100..199 \ {101} \ exact keys` | `102..199 \ exact keys` |
| exact `101` | upgrade | WebSocket handshake | WebSocket handshake |
| exact `200`-`599` | final | `{code}` | `{code}` |
| `NXX`, N = 2..5 | final | `N00..N99 \ exact keys` | same |
| `default` | final | `200..599 \ exact keys \ covered classes` | same |
| body-bearing final | | domain `\ {204, 304}` | same |

```text
SelectFinal(keys, c):         c not in 200..599 -> none
                              exact c -> exact; class(c) -> class; default -> default; else none
SelectInformational(keys, c): c not in 100..199 or c = 101 -> none
                              exact c -> exact; 1XX -> 1XX; else none
```

`HTTPResponseExpr.StatusCode` is replaced by `Status HTTPStatusSelector`, with
no parallel field.

### 2. Public DSL

```go
Response(StatusOK)
Response("missing", StatusNotFound)
Response("client", Status4XX, func() { StatusField("HTTPStatus") })
Response("unexpected", StatusFallback, func() { StatusField("HTTPStatus") })
Informational("hints", StatusEarlyHints, EarlyHints, func() { Header("links:Link") })
Informational("progress", Status1XX, Progress, func() { StatusField("Code") })
NoSuccessResponse() // error-only operation (section 3)
// SSE endpoint: typed handshake metadata, separate from the event type
Response(Status2XX, StreamResponse, func() { StatusField("HTTPStatus"); Header("trace:X-Trace") })
Response(StatusCreated, StreamHeaders, func() { Header("trace:X-Trace") }) // singleton exact: no StatusField
```

- `parseResponseArgs` accepts `HTTPStatusClass` beside `int`. `StatusFallback`
  projects to `default`.
- `StatusField(attr)` is valid in an HTTP `Response` or `Informational`.
- `Informational(name, status, type, fn)` declares an interim response with its
  own attribute type. Its DSL accepts `Header`, `StatusField` and
  `Description`.
- `NoSuccessResponse()` declares that the operation documents no success
  response. The method result must be empty.
- On an SSE endpoint, a `Response` may take a handshake type. That type
  carries the stream's mapped response headers and, in status mode only, the
  `StatusField`. It is separate from the event type (section 8).
- `Response` with an informational or upgrade status is an evaluation error
  that points to `Informational`. No consumer design uses an exact 1xx
  response; loom and loom-mcp were searched.

### 3. Final key ownership and success selection

Invariant: every final key belongs to exactly one side.

- Error keys: several errors may share a key. They are told apart by
  `loom-error` or by `ErrorNameFromStatus` counted per key.
- Success keys: at most one success response per key.
- Mixed keys are rejected for every selector kind, exact included. Today the
  generated client fails to compile for that case, so no compiling design
  changes behavior.

Success branch selection:

| Mode | When | Server picks the branch by | Client reconstructs |
|---|---|---|---|
| Tag (existing) | all success selectors exact; author uses `Tag` | first matching tag, untagged last | tag value |
| Status | any non-exact success selector, or several success selectors without tags (all imports) | `SelectFinal(successKeys, result.StatusField)` | `StatusField = resp.StatusCode` and the branch's members |
| Error-only | `NoSuccessResponse()` | none: a nil error is a framework violation (section 6) | final statuses outside error keys are `ErrInvalidResponse` |

In status mode every success response binds the same `StatusField`, exact
responses included. SSE endpoints apply these modes to their handshake type
instead of a result, without Tag mode (section 8). Tag combined with a non-exact success selector is
rejected; a shared `StatusField` covers that case.

Branch-owned members. A body attribute, header or cookie bound by only one
success branch is optional in the result type. If the source requires it, it
is recorded as branch-required in the wire-binding plan. The server rejects a
selected branch whose branch-required members are absent. The client rejects a
response missing them. Members of other branches are never set. This is new
ownership, not an existing facility: response finalization currently inherits
body, header and cookie requiredness directly from the source. Capture the
branch requirement before optionalizing the source member, and feed it to
both server validation and client wire-required decoding. Relaxing only
reconstruction admission, which currently requires every service-required
member on each branch, is insufficient.

### 4. StatusField as a transport source

`StatusField` adds a `status` location to a response-owned binding plan,
proposed as `httpResponseBindingPlan` in `expr/http_response_bindings.go`.
Endpoint `ensureWireBindings` resolves request payload/security mappings and
request body exclusions (`expr/http_wire_bindings.go:14-18,52-109`); response
sources must not enter it. The `httpWireBindingPlan` struct itself is already
reused as the response header collision cache (`HTTPResponseExpr.wireBindings`,
built by `HTTPResponseExpr.validateWireBindings`). Migrate that cache into the
response-owned plan instead of adding a second cache beside it. Keep header
name handling, the separate response cookie source/wire duplicate diagnostics,
and the `prepareErrorWireBindings` step, which projects non-selected built-in
Problem properties to `loom-attribute-*` headers before error collision checks.
Derive response body exclusion, mapping collisions, branch-required
members and reconstruction from the response occurrence, extending
`responseReconstructionBindings` (`expr/http_reconstruction.go:42-62`). Implicit
exclusion in `responseBodyObject`, per-view `buildHTTPResponseBody` and the
validation-time `responseBodySource` derives from that one plan, status included.
An explicit `Body` is collision-checked, not silently stripped. Reconstruction
coverage keeps its view filtering, first-response JSON-RPC isolation and
built-in Problem adapter exception. That exception does not admit a
`StatusField` without real native storage. `ErrorNameFromStatus` keeps its own
identity and reserved `loom-error` checks.
Success results, error carriers, informational values and SSE handshake values
retain separate source occurrences, including their views and media variants.

```go
func validateStatusField(e *HTTPEndpointExpr, r *HTTPResponseExpr, source *AttributeExpr) {
    required := !r.Status.IsExact() || e.statusSelectedSuccess(r) // exact informational: optional
    if r.StatusField == "" {
        require(!required, "non-exact or status-selected responses require StatusField")
        return
    }
    field := source.Find(r.StatusField)
    require(field != nil && field.Type == Int && source.IsRequired(r.StatusField) &&
        !field.Nullable && field.DefaultValue == nil, "StatusField must be a required non-nullable Int without a default")
    require(!mappedToBodyHeaderOrCookie(r, r.StatusField), "StatusField competes with an explicit mapping")
    require(r.StatusField != r.StatusErrorName)
    require(intersect(field.Validation, runtimeDomain(e, r)) != ∅, "StatusField constraints exclude every owned status")
    for _, view := range admittedViews(source) {
        require(view.Has(r.StatusField), "StatusField must survive every admitted view")
    }
}
```

Rules:

- Exact informational responses such as `103` need no `StatusField`; one is
  allowed with domain `{code}`.
- The field is excluded from the implicit body and from body schemas.
- One binding per response, shared by its media variants.
- The client appends a status `InitArgData` (`resp.StatusCode`, or the
  interim code) after body, header and cookie arguments and before any
  validation. Apply it to the actual native construction target. Current
  `errorBuilder.buildResultInitCode` returns the conversion and native target
  together (`http/codegen/service_data_response_errors.go:117-135`): a selected
  built-in Problem wire property reconstructs the complete native error with
  `SkipFieldInit`, while an ordinary selected field retains its own storage.
  Do not infer the native target from the selected wire body.
  `buildProblemClientResult` applies every field-bearing argument to an
  intermediate `ProblemResponse` before `ProblemErrorFromBody`. Appending the
  status argument to that list does not reach native storage. The status
  assignment must be part of the shared constructor contract
  (`newConstructorPlan`, `appendInitResult`, `fieldCode`). It must exist before
  constructor allocation finalizes, not be a template-local assignment. An
  InitArg alone creates no validation path: status and branch-required
  validation run from the response plan before construction and publication.
- The server checks domain and constraints before publication. Built-in
  `ProblemResponse.Status` is a transport-derived output, not authored
  `ServiceError` storage (`http/codegen/problem_representation.go:24-35`,
  `http/error.go:68`, `pkg/error.go:27`). Non-exact error responses still need
  a real admitted StatusField carrier; do not invent it from error flags or
  reuse the Problem document's status. Apply this rule to the final source:
  reject a non-exact built-in Problem error that lacks real native status
  storage. Add no native Status field and no blanket built-in special case.
  Custom native errors with a real carrier follow the ordinary rule. Imported
  `status` and transport `http_status` remain distinct. Generated errors keep
  `ResolveErrorResponse` first-owner selection. They never search inner causes
  for a `StatusField`. Server source-use accounting includes status demand, so
  a status-only response still extracts its source before encoding.

### 5. Client final decoding (unary)

```go
func decode(resp) (any, error) {
    if resp.StatusCode == 101 { return nil, ErrInvalidResponse }   // unary: never an upgrade
    key, ok := SelectFinal(keys, resp.StatusCode)
    if !ok { return nil, ErrInvalidResponse }
    branch, err := key.branch(resp)                                   // success; or error by loom-error / status name
    if err != nil { return nil, ErrInvalidResponse }
    if branch.HasBody() && (resp.StatusCode == 204 || resp.StatusCode == 304) { return nil, ErrInvalidResponse }
    // media, body, branch-required members, init args, validation:
    // any failure is returned for this key; no other key or branch is tried.
}
```

The generated decoder is a tagless `switch`: exact cases, then class ranges,
then `default`.

### 6. Server publication and violations

```text
encode(branch b, value):            -- runs after beginFinal (section 7)
  v := exact code, or value.StatusField
  if v not in runtimeDomain(b), constraints fail, or a branch-required member is absent:
      return violation error        -- before WriteHeader; publication origin = framework
  WriteHeader(v)                    -- publication origin = authored
error-only operation, nil error:    violation error (no designed success response)
```

A violation is an unmapped framework encoding error, routed through the
existing lifecycle:

- `EncodeResponse` restores headers and calls the error encoder.
- The error encoder retries once with the violation error, then calls
  `handleFailure`.
- A violation carries no designed error name, so the retry cannot re-select a
  designed contract.
- If a declared `5XX` or `default` covers the framework status, the client
  decodes it terminally. This aliasing is pinned, not prevented.

### 7. Exchange writer ownership and informational phase

One per-request owner, `http.ExchangeWriter`, held by `HandlerLifecycle`,
serializes every write to the `ResponseWriter` header map and status:

```text
state: phase in {interim, final}; mutex
Send(code, headers):       -- typed informational emitter
  lock
  phase = final                            -> ErrInformationalClosed (not sent)
  request is HTTP/1.0                      -> ErrInformationalUnsupported (RFC 9110 §15.2)
  authored sends >= MaxInformationalResponses -> ErrInformationalLimit
  code outside the authored emission domain, or constraints fail -> error
  snapshot := header.Clone(); replace header with exactly the declared headers
  WriteHeader(code); restore header := snapshot
  unlock
beginFinal(kind):          -- kind in {result, error, sse-open, upgrade, file, raw}
  lock; phase = final; unlock           -- before any final header is read or written
```

`beginFinal` is called at the start of `EncodeResponse`, before
`initialHeaders` is cloned. It is also called at the start of `encodeError`
and `HandlerFailed`, by the SSE writer's first open trigger, before the
WebSocket upgrade, in `ServeFile` and `WriteRawBody`, and on handler return.
Placement details:

- `HandlerFailed` closes admission before it chooses between late failure and
  the error encoder.
- `WriteRawBody` closes it before its optional first-byte peek and failure
  branch.
- The generated WebSocket path closes it inside its lazy upgrade `sync.Once`,
  before header preparation.
- The generated `Redirect` branch writes its final status directly through
  `http.Redirect`. It is also a final start: it enters final phase before that
  write and is covered by the same coordination as the kinds listed above.
  This adds no admission rule for `Informational` on `Redirect` endpoints.

A retained service goroutine calling `Send` concurrently is therefore either
fully published before the final phase or refused. Its snapshot and restore
can never erase or leak final headers. `MaxInformationalResponses = 8`
authored interim responses per exchange.

Final phase closes interim admission; it is not proof that bytes have committed.
Keep `CaptureResponse` as the physical-commit owner
(`observability/transport/response_writer.go:24-42`). An encoding failure after
`beginFinal` but before physical commit still restores headers and uses the
existing error fallback while phase remains final
(`http/handler_lifecycle.go:115-133`). Preserve raw staging and file protocol
statuses. Do not replace the captured writer or create a second request observer.

Writer locking must preserve current callback ownership: stream observation
callbacks drain after releasing the SSE write lock
(`http/sse_writer.go:342-346`), whereas `Complete` preparation runs under its
terminal write ownership and must not reenter the writer (`:126-149`). New
exchange coordination must not move observer callbacks under either write lock
or turn an interim observation into request completion. Terminal state changes
and close recording stay under write ownership. Only the close and failure
observer callbacks drain after unlocking (historical `close`,
`finishObservation`). Keep the per-call cancellation
behavior: a frame buffered before cancellation is never resent, and the
connection deadline is restored before the next operation. The trace admission
state below is a distinct per-Do owner, not the exchange writer lock.

Server emitters. Each `Informational` gets a typed emitter in the HTTP server
package, allocated by the package-symbol owner, for example
`SendGetItemHints(ctx context.Context, v *svc.EarlyHints) error`. Authored
emission never sends `100` or `101`. `100` belongs to net/http's
`Expect: 100-continue` handling, which sends at most one automatic `100` on its
own connection writer. A declared `100` key is still imported, exported and
observable.

Client observers. Each `Informational` gets a context option, for example
`WithGetItemHints(ctx, func(context.Context, *svc.EarlyHints, error) error)`.
The trace hook is installed only when an observer is registered.

```text
budgets (per transport exchange, see Counting below): at most
         MaxInformationalResponses non-100 interim responses, plus at most one 100;
         exceeding either aborts the request. Intermediary interim
         responses count against the same budgets; peers and intermediaries
         that need more are outside the contract.
per interim response: SelectInformational; undeclared -> ignore (RFC 9110 §15.2);
         declared -> decode headers and StatusField -> observer(value, decodeErr);
         a non-nil observer return aborts the exchange
```

Abort propagation (workaround for golang/go#76669; upstream evidence:
https://github.com/golang/go/issues/76669#issuecomment-6675485882).

One admission owner, `loomhttp.ObserveInformational`, is called once per
`Doer.Do` by the generated request path. Observer options found in the context
select the declared keys and callbacks; all state below is created per `Do`
and is never stored in, or read back from, the context. A context that
already carries hooks, including a Loom hook from an earlier call, is treated
as the caller's chain.

```text
ObserveInformational(req, observers) (req', finish):
  rctx, cancel := context.WithCancel(req.Context())        -- owned by this Do
  st := {mu; loomErr; exchange counters}
  hooks (one ClientTrace, composed once):
    GotFirstResponseByte: lock; reset per-exchange counters; unlock -- before response 1xx loop
    Got1xxResponse: lock
                    err := st.loomErr
                    if err == nil {              -- no new admission after the first Loom error
                        err = admit(code, header) -- budgets, selection, decode, observer
                        if err != nil { st.loomErr = err }   -- store first ...
                    }
                    unlock
                    if err != nil { cancel(); return err }                     -- ... then cancel
                    return nil
  finish(resp, doErr) (resp, error):
    lock; loomErr := st.loomErr; unlock                       -- synchronized read after Do
    if loomErr != nil:                                         -- Loom error preferred deliberately
        errs := [loomErr]
        if doErr != nil: errs += doErr                         -- whatever Do returned, possibly context.Canceled
        if resp != nil: errs += resp.Body.Close()              -- close error identity preserved
        cancel(); return nil, errors.Join(errs...)
    if doErr != nil: cancel(); return nil, doErr
    resp.Body = exchangeBody{resp.Body, cancelOnce}            -- cancel retained, never deferred here
    return resp, nil
exchangeBody: Read returning io.EOF -> cancelOnce; Close -> inner.Close() returned unchanged, then cancelOnce
no-body responses (http.NoBody, HEAD, 204, 304): the generated decoder's existing
    body cleanup closes the wrapper, which cancels; SSE and raw streams cancel on
    stream Close or EOF
```

Post-abort observer admission. Once `st.loomErr` is retained, later Loom hook
calls return that error without counting, decoding or invoking another Loom
observer. Use the existing monotonic first-error record, not another lifecycle
flag. A callback already running is not forcibly interrupted. Go still owns
composition: the pre-existing caller hook runs once and may complete after Do
or finish has returned; Loom must not invoke it manually. Late hook completion
never changes the result already returned by finish.

This chooses short-circuiting over continuing observer side effects for buffered
interims after Loom abort. Accepted: observerA fails, a buffered103 selected for
observerB arrives, B is not called, and the caller hook still runs through Go's
composition. Accepted: an already-running caller hook completes after Do.
Rejected: invoke a new Loom observer after A is retained, clear A on an exchange
reset, or replay the caller hook manually. The choice tightens this new feature's
abort contract before implementation; it does not alter the caller's own hook
contract or forcibly stop an executing callback.

Counting. Budgets apply per transport exchange. The first response byte of the initial
exchange, each redirect and each transport retry fires `GotFirstResponseByte`
before any 1xx callback and resets the counts: at most `MaxInformationalResponses` non-100 interim responses and
one 100 per exchange. Interim responses of every exchange in a `Do` chain are
delivered in order. Redirects are transparent to interim observation, as they
already are to the final response. The `Do` chain itself is bounded by the
`Doer`'s redirect policy (`http.Client` defaults to 10).

The reset uses native response-reader ordering, not request-write completion.
On HTTP/2, `WroteHeaders` can run after interim responses have arrived; resetting
there could exceed the same exchange's strict limit. Late write notifications
must leave counters unchanged. Go1.27.3 HTTP/1 invokes first-byte before its
internal 1xx loop, and HTTP/2 does so once per fresh stream before handling 1xx;
its per-header state reset is distinct from its first-byte state. This also
covers direct ClientConn calls. Arbitrary custom transport trace order is not
promised. The first retained Loom error is never cleared by either boundary.

Guarantees and non-guarantees:

- The caller's pre-existing `Got1xxResponse` hook runs exactly once per
  interim response. net/http calls Loom's hook first and the caller's second,
  and Loom never calls the caller's hook itself.
- Returned errors. When a Loom failure is recorded, the result is
  `errors.Join` of three things, nil ones omitted: the first Loom error
  (deliberately first), the error `Do` returned, and the response body's close
  error. When only the caller's hook fails, net/http propagates that error
  natively and Loom returns it unchanged. When both fail, the caller-hook
  error's identity is not guaranteed: Loom's cancellation may make `Do`
  return `context.Canceled` instead.
- Loom composes once per `Do`. It does not repair defects in a chain the
  caller built, and does not promise to interrupt a caller hook that never
  returns. Cancellation aborts the transport, and Loom's error is returned
  when `Do` settles.
- Cancellation is never deferred right after a successful `Do`. That would
  abort streaming and raw bodies, as the scratch proof shows.
- The workaround stays linked to golang/go#76669 in code. It is removed, and
  the hook simply returns its error, when the pinned Go propagates a composed
  hook's error. Upstream evidence is already published at the link above.

Scratch proof `/tmp/loom-976-trace-proof` (Go 1.27.3, HTTP/1.1 httptest,
`proof-output.txt`; not production):

| Case | Observation |
|---|---|
| no caller hook | 3 interim seen; aborted with Loom's error |
| caller hook, plain return (bug) | 3 seen, caller hook called 3 times; 200, no error |
| caller hook, cancel workaround | 3 seen, caller hook called 3 times; Loom's error |
| within limit | 2 seen; 200 |
| `Expect: 100-continue`, 8 authored 103s | 9 interim (`103`×8, `100`) |
| 307 redirect, interim responses on both exchanges | `GotFirstResponseByte 103 GotFirstResponseByte 103 103`; per-exchange response boundary observed |
| streaming body, cancel retained | late event read |
| streaming body, cancel right after `Do` | `context canceled` on read |
| `errors.Join(loomErr, closeErr)` | both identities match `errors.Is` |

Runtime-test obligations:

- caller hook nil, returning nil, and returning an error;
- Loom and caller hooks both failing (asserting the Loom error and the error
  `Do` returned, not the caller-hook identity);
- an observed SSE stream and a raw body read to completion after `Do`;
- body `Close` racing `EOF` with exactly one cancel;
- redirect and retry counting;
- close errors preserved;
- HTTP/1.1 and HTTP/2 transports.

### 8. SSE success handshake

An SSE endpoint applies section 3 using its typed handshake, separate from the
event carrier, as the one source of success status and response headers.
There is no second selector. In mixed methods, section 8.1 preserves the whole
ordinary Result separately from this shared metadata source.

When no success Response is authored, resolve endpoint, service and API SSE
inheritance before selecting the default. An HTTP SSE success is body-bearing,
including a mixed `Result(Empty)` plus `StreamingResult(Event)` method and a
successful stream that sends no events. Its implicit default is therefore 200,
not 204; 200 also serves the empty ordinary branch of that mixed method. The same
response wire-body owner vetoes a body-forbidden default and determines
exact-status body admission and captured final runtime domains. Default policy
still makes 204 eligible only for native Empty results (excluding raw-body and
FileResponse); a supplied native result retains default 200 even if its implicit
body is empty or its members are all carried in headers. Explicit SSE 204/304
remain invalid. The existing raw-body, FileResponse and explicit Redirect default
policies remain unchanged.

| Success responses of the SSE endpoint (section 3 mode) | Status at open | `StatusField` | `SetResponse` |
|---|---|---|---|
| one exact success, any code (`200`, `201`, …) | that literal code | none; never invented | generated only when the handshake declares headers |
| several successes (for example exact `200` and `201`), or any non-exact success (`2XX`, `default`): status mode | `SelectFinal(successKeys, meta.StatusField)` | required on the handshake type | always generated |
| Tag mode | not available: no result value exists when a stream opens. Today's rule of at most one SSE success response (`expr/http_endpoint_validate.go:134-136`) is relaxed only for status mode | | |

```text
SetResponse(meta):                     -- generated server stream method
  stream opened -> ErrSSEResponseCommitted (immutable after open)
  status mode: validate StatusField against the success runtime domains and constraints -> error
  validate mapped header values -> error
  store meta
first open trigger, under beginFinal(sse-open):
  triggers: Open, SendComment, Send/WriteEvent, successful Complete with a message,
            explicit successful Complete(nil), explicit server stream Close,
            successful zero-event handler finalization
  deferred End is cleanup only, never a success-publication trigger
  status := literal code (singleton exact) | SelectFinal over meta (status mode)
  status mode, metadata missing or selecting no success key -> pre-commit framework error
  a required handshake header not set by SetResponse         -> pre-commit framework error
  an optional handshake header not set                       -> header omitted
  (both errors use the ordinary error path; a status is never guessed)
  write handshake headers and Content-Type, WriteHeader(status), flush
service error before open -> ordinary error mapping (any declared error key)
service error after open  -> terminal stream failure (existing)
```

Every successful response of one SSE endpoint uses the same native handshake
carrier type. Expression validation establishes this before code generation:
named types use their declaration identity (including copied wrappers), and
anonymous types use their actual source type identity. Equal field shapes or
Go-friendly names do not establish a shared type. This is the single metadata source
for the one typed `SetResponse` / `Response` API; the ordinary Result in section
8.1 remains separate. Code generation must not choose the first carrier or
synthesize a union.

| Authored carriers across success responses | Admission |
| --- | --- |
| Separate response wrappers of the same named declaration | accepted |
| Copies retaining the same native declaration ID | accepted |
| The same actual anonymous object source | accepted |
| Separately named object types with identical fields | rejected |
| Separate anonymous object sources with identical fields | rejected |
| A separately declared object-backed alias and its source type | rejected as distinct native declarations |
| A singleton exact typed handshake or a no-metadata singleton | existing admission preserved |

Type identity does not merge response occurrences. Each branch retains its own
examples, header/cookie bindings and captured requirements. Informational and
error carriers may be different types; they are not included in this success
identity check. Existing StatusField requirements still govern multi-success
handshakes, so a primitive alias does not become a status-bearing object.

Header requiredness comes from the handshake type's own `Required`, exactly
as response headers derive it from the result today.

#### Mixed unary and streaming representations

For a method with distinct unary `Result` and `StreamingResult`, a typed
handshake is the single per-exchange status/header/cookie metadata source for
whichever successful representation content negotiation selects. The unary body
is the whole `Result`; SSE events remain `StreamingResult`. Handshake bindings
never subtract same-named fields from the unary body, suppress a primitive or
collection Result, or populate Result from headers/cookies/status on the client.
Neither body is the metadata carrier. Independent authored documentation keeps
its existing authority; export must describe both selected body representations.

The service is invoked once. Its typed `SetResponse` reaches one shared runtime
response-metadata admission owner through the actual SSE stream adapter or the
unary discard adapter. This owner holds the metadata lock, immutable prepared
snapshot and sealed state. Unary capture and SSE opening both capture and seal
atomically under that same owner; the owner is extracted from the current SSE
metadata admission boundary, not reimplemented by the discard adapter. Once
capture/seal starts, concurrent or later setters cannot replace the captured
snapshot and return `ErrSSEResponseCommitted`. The model's `held` phase represents
this critical section; it excludes setters until publication completes. This
shared seal is required and is not a separate discard-stream publication state
machine. In SSE mode this is the writer's admission lock: every terminal admission,
including pre-open failure, suppression, completion and Close, seals the shared
metadata owner under that same lock even when nothing was published. Unary failed
capture and handler error likewise close admission atomically. There is no
unlocked terminal check followed by a separate metadata update. Existing
preparation callbacks retain their documented non-reentry contract; observation
and other reentrant callbacks are dispatched after releasing admission ownership.
Existing terminal/error ownership remains with the selected response.

Every exposed discard-stream operation is non-publishing and non-sealing:
`Send`, `SendWithContext`, `Close`, and any exposed open/comment/completion
operation discard their stream effect. They do not finalize or close the actual
HTTP exchange or its shared metadata owner. Consequently `Send(e); SetResponse(m)`
and `Close(); SetResponse(m)` may succeed in unary mode until unary capture,
while the same code in SSE mode rejects the setter after the first operation
commits or completes the stream. This representation-dependent result is
intentional: actual capture or terminal admission seals metadata; discarded
operations do neither.

`SetResponse` validates and snapshots its supplied value before replacing the
prior snapshot. Invalid native status, an error key used as success, or an invalid
mapped binding returns an error to the service and preserves the previous
metadata. Missing required metadata fails at first successful capture, before
any physical write, through the ordinary framework error path. Replacement after
capture is rejected at the setter. Optional singleton metadata may be absent,
using its literal status and omitted optional fields. A supplied native scalar
zero is not absence. Existing callback completion, first-error, cancellation and
terminal-winner assumptions remain unchanged.

The generated client exposes both successful values explicitly. For a typed
mixed method, its ordinary client method returns `(UnaryResult, HandshakeRef,
error)`, where `HandshakeRef` is the existing generated native metadata reference
(for an object Opening this is normally `*Opening`). Its stream method returns
the stream whose `Response()` exposes that same `HandshakeRef`. Every admitted
generated client endpoint of the typed mixed method returns one shared client
response carrier containing
separate Result and Metadata fields; the shared client-return and interceptor
owners unwrap/preserve that carrier. `SelectFinal` determines one response:
its StatusField and header/cookie bindings reconstruct Opening, and its unary
body decoder reconstructs the whole Result independently. Both must validate
before either is returned. An error returns the zero Result and zero `HandshakeRef` (nil for the object
carrier).
The server service method retains its existing Result/error signature and
supplied stream setter. No extra endpoint invocation or context callback/sink is
introduced. Untyped/default native client returns remain unchanged. Regeneration
changes the typed mixed client's return arity; callers, generated client
interceptors and CLI client adapters must consume the explicit pair rather than
silently discarding metadata.

This does not introduce a new transport admission rule. Typed handshakes are
already HTTP-SSE-only; authored StreamingResult makes the method streaming, and
the existing shared-method check rejects simultaneous HTTP/JSON-RPC streaming
mappings in either declaration order. Existing gRPC admission rejects distinct
Result/StreamingResult mixed methods. Therefore the typed mixed carrier currently
has HTTP client endpoints only. Standalone JSON-RPC mixed methods remain admitted
without HandshakeRef and retain their existing result/envelope and CLI contracts;
HTTP typed mixed and separate JSON-RPC methods may coexist in one service. No
transport fabricates absent HTTP metadata. Any future admission expansion must
provide the same shared client carrier and an explicit metadata wire contract
before it can use the transport-neutral client/interceptor return owner.

Accepted: `Result(Unary)` plus `StreamingResult(Event)` and typed Opening
responses; the service sets Opening once and returns Unary or sends Event as
negotiated. A Unary.session body member remains present even when Opening.session
maps to a cookie/header; a string or array Result remains its whole body. A
required missing Opening fails before write. An invalid SetResponse returns an
error without destroying a previous valid Opening; a late setter is rejected.
Rejected alternatives are casting Unary to Opening, dropping metadata on the
unary client, hiding it behind an optional callback, erasing Result fields,
reinvoking the endpoint, or rejecting the otherwise admitted mixed design.

OpenAPI export must retain unary Result and SSE event media/body representations
with this same metadata owner. It must not describe both as an event-only schema
or infer one body from the other. Existing independent OpenAPIBody authority and
event correspondence eligibility still apply.

The bounded `MixedResponse.tla` companion explores this role/capture boundary;
SSEOpen/SSETerminal retain terminal/publication assumptions. It does not prove Go
compilation, arbitrary callbacks/codecs, client API generation or schema graphs.
Generated correspondence must cover both representations, Send/Close followed
by SetResponse, same-named and non-object Result bodies, client/interceptor/CLI
pair handling, zero-event SSE, and missing/invalid/late metadata before write.
Exact terminal-setter controls must reject with `ErrSSEResponseCommitted` after
SSE pre-open failure, suppression and Close, and after unary failed capture or
handler error; include a concurrent setter/terminal race with one atomic winner.
Transport controls compile and exercise one service with a typed HTTP mixed
method plus a separate JSON-RPC method through its shared client/interceptors,
preserve the standalone JSON-RPC mixed module, and retain both-order dual-stream
and gRPC-mixed admission rejection. No new rejection replaces those controls.

`SSEStreamWriter.initHeaders` (`http/sse_writer.go:274-283`) takes the
selected status instead of `http.StatusOK`. An unopened stream with a nil
error publishes its handshake with zero events, instead of the implicit empty
200. Successful finalization is a fallible path before deferred cleanup; a
missing handshake or required header routes through the ordinary lifecycle
without publishing success. Generated handlers currently defer `End` before
calling the endpoint and route errors through `HandlerFailed`
(`http/codegen/template_sources.go:129-138`). Making that defer open would emit
success after a pre-open error or during panic unwinding. Keep `End` cleanup-only
and preserve its original-panic precedence (`http/sse_writer.go:152-160`).

The model's logical `end` trigger means successful completion: `Trigger` and
`Return` require `~failed` (`SSEOpen.tla:62-80`). It is not permission to open
from unconditional Go cleanup. Retain one terminal winner and current Complete
preparation ownership; failed preparation leaves the stream usable. Tests must
cover successful zero-event publication, pre-open errors, failure/panic cleanup,
and concurrent terminal operations separately.

The HTTP open triggers above do not redefine every nil completion. JSON-RPC
`CompleteSSEStream` calls the shared `Complete(ctx, nil)` to suppress a
notification's response, and records suppression only after admission. That
stream must stay unstarted, with no SSE headers. Globally making nil preparation
publish a handshake would change that protocol path.

Decision: the one `SSEStreamWriter` terminal owner offers two explicit
terminal operations, successful publication and protocol suppression. Both share
its single terminal state and winner. Public `Complete(nil)` and successful
zero-event handler finalization are successful publication. `CompleteSSEStream`
calls the explicit suppression operation, so JSON-RPC notifications stay silent
and suppression is still recorded only after admission. Wrappers hold no
terminal state, and there is no per-consumer patch. Method spelling is a routine
implementation choice. Rejected alternative: inferring intent from writer
configuration such as a handshake option. That is context-dependent and leaves
publication intent implicit. Suppression is not `End`: `End` stays cleanup-only
and panic-preserving. The callback, lock and cancellation rules above apply to
both operations. Both generated HTTP branches finalize only on
nil error: the mixed-result SSE branch and the dedicated SSE branch. A
callback encoding failure is not a pre-open error, because `writeEventLocked`
initializes headers before it invokes the callback. Keep it separate from the
missing-metadata and missing-header pre-commit errors.

Explicit `Close` is successful completion. This is a deliberate public behavior
correction. In the historical preflight source, the generated HTTP server
stream `Close` called `SSEStreamWriter.Close`, which ended an unopened stream
without publishing anything. Now `Close` is a successful-publication trigger on
the same terminal owner as `Complete(nil)`.

- **Context and validation.** `Close` has no context argument, so it uses the
  writer's request-owned context. Admission and cancellation timing match the
  other successful triggers. On an unopened stream it publishes the configured
  handshake before closing, under the same status, required-header, failure and
  pre-commit error rules. It never invents a default status.
- **Already terminal.** `Close` on a stream that is already terminal keeps its
  idempotent no-op behavior. It never opens the stream or overwrites an earlier
  suppression, failure or publication. Its return value follows the writer's
  existing terminal-failure contract and is fixed by the model and tests, not by
  the generated wrapper.
- **Handler return and later failure.** A normal handler return after a
  successful `Close` observes that outcome: no second publication and no
  spurious error. A later error or panic cannot replace a handshake that is
  already committed.
- **Cleanup.** Error and panic cleanup is the generated handler's deferred
  `End`, which stays silent and never opens. Applications must not use `Close`
  for that cleanup. The generated `Close` documentation states these semantics.
- **No new branches.** There is no compatibility mode and no per-protocol
  runtime branch. In that source, JSON-RPC paths did not call
  `SSEStreamWriter.Close`.

Rejected alternative: a "closed for writes" phase that postpones success until
handler return. It would add a second terminal-like state only to defer
publication, and it would split the single terminal winner.

The generated SSE client:

1. Applies `SelectFinal` over the endpoint's keys before constructing a stream.
   Dispatch with and without designed errors derives from that one selection.
2. Decodes error keys as errors.
3. For a success key, checks `text/event-stream` and decodes handshake
   metadata, including the actual status, then constructs the stream. The
   stream exposes `Response()` even when no event ever arrives. The check
   parses Content-Type with standard MIME parsing and requires the essence
   `text/event-stream` with valid parameters. Accepted: `text/event-stream`
   and `text/event-stream; charset=utf-8`. Rejected: a missing header,
   `application/json`, `text/event-stream-extra`, and a malformed value.
   `mime.ParseMediaType` returns the essence even with
   `ErrInvalidMediaParameter`; that result is still rejected. This replaces
   the current prefix match, which also admits a missing header, as a protocol
   correction under the accepted check. The WHATWG SSE processing model fails
   the connection when the response Content-Type is not `text/event-stream`
   (https://html.spec.whatwg.org/multipage/server-sent-events.html#processing-model).
   Loom takes only this media rule from it. Status selection stays unchanged,
   with no EventSource 200-only restriction or reconnection behavior, and no
   new charset or event-codec policy. Body ownership is unchanged:
   `DecodeResponse` transfers or closes the body exactly once, and the stream
   closes it on EOF or cancellation.
4. Decodes JSON event data through the central `SSEEventResponse` adapter, which
   detaches `Request` and sets `Content-Type: application/json`. It does not
   inherit request-media context or reselect the handshake for each event.
   Apply the adapter to every JSON-event path that fabricates a response
   today: whole structured, primitive/collection, projected and optional/default
   data. Explicit raw mapped string/bytes/int event field codecs remain intact.

The hardcoded 200 check (`client_jennifer.go:240-255`) and the fabricated
`http.Response{StatusCode: 200}` in event decoding
(`stream_sections_sse_client.go:52-66,84,151`) are removed. No event is ever
re-selected.

Browser `EventSource` "fails the connection" unless the status is 200 (WHATWG
HTML §9.2.2, constructor step 15.3). Generated native clients support the full
success domain. APIs whose consumers are browsers should declare exact `200`;
the guide says so.

### 9. Projection surfaces

| Surface | Rule |
|---|---|
| Server encoders | `beginFinal`, then an exact literal or section 6 |
| Client decoder | section 5; status mode assigns `StatusField` and branch members |
| Informational emitters and observers | section 7 |
| SSE | section 8 server and client |
| OpenAPI IR/v3 (3.2.0 and 3.1.1) | key = `OpenAPIKey()`. Informational responses are entries with headers and no content. Same-key error alternatives merge through `anyOf`; different keys never merge. Keys sorted. `StatusField` and handshake fields are absent from body schemas. Error-only operations emit only their error keys. Integer-keyed response body indexes, their IR/render/v3 adapters and the media contexts migrate together, preserving per-occurrence identity. Final v3 maps already take string keys. Non-exact keys need a selector description, not `http.StatusText` |
| Body type names | suffix from the selector (`StatusText`, `Status4XX`, `StatusDefault`); today `httpResponseBody` uses `StatusText` (`expr/http_body_types.go:236-241`) |
| Runtime contracts | `ResponseContractCase` replaces `StatusCode` with `Selector` and `Domain []int`, and adds an informational kind. Public API change |
| Docs | `docs/http-guide.md`, `docs/dsl-reference.md`, `docs/code-generation.md`, the consumer `loom` skill and the framework skill |

### 9.1 Derived async handshake and message examples

`x-loom-async` is reserved derived output, never an independent authored async
contract. Its handshake response uses one nonempty, duplicate-free `statuses`
array of canonical `OpenAPIKey()` strings, sorted by ascending lexical byte
order. Use that same ordering for the equivalent-success representative in
section 10.1. Exact 201 is `["201"]`, exact 200 plus 2XX is `["200", "2XX"]`,
and exact 200/201 is `["200", "201"]`. WebSocket uses `["101"]`; the current
projection obtains 101 from `expr.StatusSwitchingProtocols`, and the prospective
shared upgrade response plan must retain that fact. This does not make 101 a
final-success selector. Error and informational response selectors are excluded
from the SSE success-handshake list.

The shared prepared stream response plan supplies the selector occurrences.
Exporters serialize its keys; they do not infer selectors from a first response,
HTTP 200, examples or extension metadata. Migrate the scalar projection and its
consumers together, without a scalar/list union or fallback alias. Regeneration
changes derived extension format for existing exact SSE and WebSocket operations;
it does not change their wire status.

Importer admission checks any operation-level `x-loom-async` before the generic
opaque-extension rendering path. In the admitted itemSchema SSE branch it checks
and consumes the complete derived contract: transport/direction, route/method,
content types, selector list, message schema including examples, and mappings.
Every occurrence outside that branch, including whole-schema SSE, non-streaming
operations and extension-only operations, receives a targeted reserved-derived-
extension diagnostic requiring regeneration or a supported source contract.
Legacy scalar status, scalar plus list, unknown contract members and conflicting
facts reject. Never preserve this reserved key as opaque Meta, silently strip it,
or allow it to select SSE. Unrelated extensions retain their existing behavior.

The recognized operation extension has only two typed schema roots:
`messages.inbound.schema` and `messages.outbound.schema`, mirroring the exporter's
`visitAsyncMessageSchemas` traversal. Source recognition follows the actual
itemSchema SSE branch through reusable response references, before successful
semantic admission. The initial subset rejects inbound messages. These roots
participate in ordinary resolution, dialect checks, selected closure and naming;
examples/defaults, arbitrary extension members and their `$ref`-looking data
remain literal. An unsupported reserved extension elsewhere is diagnosed, not
interpreted by a generic schema search.

The single event-value owner is `endpoint.Stream.ResponseValue`, captured from
`MethodExpr.EffectiveStreamingResult()`. It is not an arbitrary response body or
the first status occurrence. Determine eligibility before synthesis. Require a
pure server SSE stream, `!MethodExpr.HasMixedResults()`, no independent success
documentation body, no event projections and all five requestID/data/id/event/
retry mappings null. Then establish each success's event correspondence by the
following source-owned rule. All conditions true means the derived branch;
otherwise retain the existing async branch. The new typed-handshake event
documentation rule below applies in either branch. This is not an implementation
option.

Result and StreamingResult are separately captured by `captureMethod`, even when
expr declares one event contract. Their ValueOccurrence IDs and ValuePlanNode
handles need not be equal. The expr `!HasMixedResults()` predicate is the semantic
bridge for the legacy result-derived path; do not replace it with equality of
capture IDs, attribute pointers, public names or generated samples, and do not
refactor captureMethod globally to manufacture one identity. This means the exact
existing !HasMixedResults predicate, not generalized structural or semantic type
equivalence between separately authored results.

The shared stream response plan distinguishes two response roles and preserves
the independent documentation override:

- **Result-derived response:** the implicit response source is the captured
  Method.Result occurrence / ResultValue, bridged to the effective stream event
  by the expr predicate. For an implicit whole body require empty BodyOrigin and
  target Selection, no independent OpenAPIBody, and no actual event member
  removal. HTTPBodyProjection supplies Excluded and declaration-level
  composition/hint provenance; the actual target supplies selection. Its Source
  is not an occurrence-annotation authority. A result-derived header/cookie/status
  binding that removes an event member makes this path ineligible. Primitive
  bodies need no projection record: exclusions are vacuous, but a primitive
  lowered to Empty by bindings is ineligible. A synthetic wrapper/name is not a
  different event. Authored Body presence must be retained before lowering;
  never infer it from `resp.Body == nil` after body synthesis.
- **Typed SSE handshake:** the binding source is the separate Handshake type.
  `responseBodySource` and `httpResponseBody` intentionally produce no metadata
  body. This Empty value is neither a missing event nor a projection of it.
  For every new typed-handshake endpoint, eligible or not, prepare/analyze the
  actual event from ResponseValue once in an endpoint/event context independent
  of status and visit order. Every success's implicit payload documentation
  consumes that completed event outcome, retaining the source-bound SSE field mappings and
  projection/selection semantics. The shared event analysis may own several
  actual projection/declaration positions; it is not permission to substitute an
  unprojected whole event or promise one callback across distinct owners.
  Handshake status/header/cookie records remain separate. Same-name handshake
  members never remove event members; only source-bound event projections count.
  Eligibility decides whether async documentation derives from the completed
  event or retains its existing independent acquisition/sampling branch.
- **Independent documentation override:** an explicit OpenAPIBody retains its
  existing independent documentation projection and authored annotations. It
  takes precedence over implicit event payload documentation, including on a
  typed handshake, and always makes derived-async eligibility false. On a new
  typed handshake the actual event remains ResponseValue-owned and is prepared
  once; the independent documentation body is neither that event nor the Empty
  handshake body.
  Expression binding validation must apply that same source distinction:
  handshake StatusField/header/cookie competition is checked within the
  handshake carrier's actual bindings. A same-named member of independent
  OpenAPIBody event documentation is not a competing handshake binding. Correct
  the current validateStatusMappings OpenAPIBody check at that shared owner;
  preserve ordinary response competition checks and genuine duplicate mappings
  within the handshake. This is a required expression correction, not a claim
  that the current candidate already admits the override.
  Never erase the override to obtain roundtrip eligibility. Invalid handshake
  Body remains rejected by the existing DSL owner.

For result-derived payload correspondence, compare three role-specific operands:
(1) the effective StreamingResult's authored occurrence and completed event
target plan; (2) the implicit Method.Result occurrence / ResultValue, or the
original explicit Body occurrence with its local UserExamples and annotations;
(3) the actual finalized response target plan. The expr bridge justifies the
implicit result source; it does not waive complete authored/target comparison.
For primitives use those same occurrence operands, without inventing a recorded
whole-primitive projection source. An explicit Body with the same complete
contract and representation is deterministically eligible when comparison
succeeds. Missing association, unsupported comparison or mismatch is ineligible.

The original explicit Body occurrence is a required retained fact at the existing
pre-finalization response-plan capture boundary. Current finalizeBody duplicates,
initializes and wraps Body before cloneExplicitHTTPBody records its projection;
that later projection argument is not a pristine authored snapshot. Preserve
its authored presence/source and local UserExamples before those transformations,
using the existing capture/composition ownership, not a second example authority.
Current ResultValue capture already provides the implicit occurrence. These are
implementation obligations, not claims that a new provenance API or comparator
already exists. HTTPBodyProjection.Source may assist declaration composition,
never recover lost occurrence-local annotations.

Implement the comparison callback-free before example synthesis; the following
traversal is a required algorithm, not an already available equality API. Traverse
paired typed nodes with a pair guard through named/alias edges, object members,
array/map elements and keys, additional attributes, union alternatives/discriminator
bindings and recursive cuts. Compare the actual selection and effective view;
kind, codec, member spelling and wire name, visibility/presence, requiredness and
openness; all effective constraints and default presence/value; schema title,
description, documentation, nullable/read/write/deprecation and authored metadata;
and ordered authored example values, scopes, summaries, descriptions and metadata
at each occurrence/declaration position. Use the captured declaration/selection
associations and existing immutable node queries (`Members`, `Branches`,
`Underlying`, `Element`, `Additional`, `Key`, `Attribute`, `TargetDeclarationID`)
to read the actual graph, not attributes recopied to infer identity. Compare
supported literal values structurally without invoking custom projectors/codecs;
unsupported opaque comparisons fail closed. Generated samples never participate.

Only known compiler body wrappers, names and provenance bookkeeping are excluded
from authored-contract equality, through their existing owner records. Compare
original authored sources before those additions rather than stripping arbitrary
metadata from finalized attributes. Preserve real public declaration/naming claims
through the existing acquisition owner; equality is not permission to erase a
canonical explicit Body claim. No shape-only equivalence or blanket metadata
normalization is permitted. No new public equality API is required by this rule.

A Body selecting a field/subset or changing a constraint, default, view, local
Example or annotation is ineligible, even if every status repeats that same
changed contract. Do not promote those native Body facts onto StreamingResult.
Existing non-handshake ineligible paths, including mixed results, independent
documentation bodies and mapped/projected SSE, keep their current behavior.
New typed handshakes follow the event/override rules above even when ineligible;
their async branch retains its independent sampling and authored intent. This
adds no generic DSL rejection. The strict importer may still reject output
outside its one-event inverse.

For eligible endpoints, use the same once-prepared event rule with the existing
stream target and endpoint/event context independent of success key or visit
order. Typed handshakes have already selected that event analysis regardless of
eligibility; do not perform it again for the derived branch. Use the effective
StreamingResult's ordinary authored/synthesized example selection and actual
declaration annotation owners. Every success itemSchema's contentSchema and the derived outbound message consume this same completed
schema/outcome. Status and headers remain independent handshake records; choosing
201 instead of 200 cannot choose a different event schema or sample. A native
unauthored 200/201 design therefore publishes equal per-status payload siblings
and can satisfy the importer common-payload rule. Do not analyze/synthesize each
response first and then overwrite its result with a representative.

The eligible branch does no independent async baseline acquisition, example
preparation or sampling. Retain the completed event graph through component
finalization and reference rewriting, then use the existing typed materialization
correspondence for the outbound view. There is no reserved async-example record,
imported metadata key, second prepared example set or representation capture
owner. Preparing one existing event target is not permission to invent another
value authority beside ResponseValue or to mutate shared declarations.

The event occurrence supplies the root's effective schema example owner: retain
a local owner selected by ordinary analysis, and a declaration owner where no
local owner exists. A bare reference with an empty Example field is not proof
of suppression. Expanded children use their actual acquired declaration or inline
occurrence positions; nested reference-local overlays stay local. Preserve
complete assertions, defaults, annotations, nullable/assertion wrappers, union
branch identity and recursive/synthetic envelope cuts. An explicitly empty or
suppressed selected outcome replaces the entire group, without declaration
fallback; this does not erase independently owned child examples. Schema and
media scopes remain distinct. Copy through existing callback-free snapshots,
never by patching serialized JSON or looking up a guessed component name.

`WithExampleValue` remains a projector over the original attribute and raw value
at the existing event/declaration owner. It runs once per ordinary owned input,
not again per success response, expansion, or async message. Copy its completed
transformed values, ordered groups, representative choice, explicit null/empty,
filtered entries and suppression. Filter-all cannot trigger fresh synthesis.
Existing custom-output responsibility and opaque-value copy boundaries remain.
Callback counts must be tied to actual owners; there is no universal one-call
promise. Old non-handshake ineligible paths retain their current callback
behavior. New typed-handshake implicit documentation reuses the completed event
outcome across statuses; an ineligible async branch and an explicit independent
documentation override retain their own existing owners and callback behavior.

Removing async acquisition can remove async-only registrations or change final
allocation even without changing assertions. Preserve the event's real component
closure, including rootless recursive cuts, before finalization. Do not retain
obsolete acquisition just to preserve incidental names. Conversely, do not call
any name/inventory drift harmless: M9 requires every actual removed async-only
registration and consequent naming/reference change to be classified separately
from sample changes, with unchanged ordinary declarations and reachability proven.

Import compares the complete derived message to the common event graph, including
examples and presence at corresponding nodes. A legacy scalar-status format
receives a targeted regeneration diagnostic. A current statuses-format message
that differs from the payload receives an unsupported event/payload divergence
diagnostic naming both schema paths; regenerating an intentionally ineligible
current design does not cure that mismatch. Neither diagnostic permits sample
stripping or shadow metadata. The equal complete imported payload-occurrence
facts attach to the StreamingResult occurrence itself; status/header records
stay per response. They cannot remain only in parallel Body annotations that
would make the exporter's event owner differ. This normalization is justified
only after all source occurrences compare equal, with all provenance retained.
It does not authorize promoting differing native Body intent to StreamingResult.

### 10. Importer mapping

```text
analyze(operation):
  each key -> ParseOpenAPIStatusKey, or response-status ("HTTP code, uppercase 1XX-5XX, or default")
  Responses.Default -> fallback selector
  101 -> response-status ("101 describes a protocol upgrade OpenAPI cannot define")
  1xx exact or 1XX with content -> response-status ("1xx responses cannot contain content")
classify:
  informational := exact 1xx (not 101) and 1XX
  success := class-2 keys if any; else class-3 keys if any; else {default} if present
  errors  := every other final key
  success empty and errors non-empty -> NoSuccessResponse() (OAS §4.16 SHOULD, not MUST)
  no final keys at all -> blocking diagnostic: Responses must contain a response code
```

| Shape | Rendering |
|---|---|
| one exact success | unchanged |
| one non-exact success | result gains `http_status` (`HTTPStatus`) as `StatusField` |
| several success keys | status mode: envelope `{http_status, status_200, status_2xx, …}`, `Body("<branch attr>")`, branch-required members |
| `default` as sole success | status mode, one branch |
| error-only | method without result; `NoSuccessResponse()`; errors as below |
| non-exact error | error type gains `http_status`; non-object or union bodies use `{http_status, body}` with `Body("body")`; schema-less errors use `Body(Empty)` |
| informational | `Informational("<Op>Status103"\|"<Op>Status1XX", …)` with a header-only type; `1XX` gains `StatusField("http_status")` |
| SSE with several or non-exact successes | status mode handshake type `{http_status, headers…}` |
| SSE with one exact success and response headers | handshake type `{headers…}`, literal status |

Declaration names must use the delivered #957 unified `allocateDesignNames`
reservation owner, including local/external error clones. Occurrence-local
transport fields still use collision-free occupancy; a field allocator is not
a declaration allocator. Preserve original declaration origin and reference
provenance when adding envelopes. Errors are named `Status4XX` and
`StatusDefault` under the existing shared-status disambiguation. Integrate
status envelopes with `responseMappingBody`'s existing union/Any selected-body
wrapping (`internal/openapiimport/render_response.go:130-157`), preserving media
variants and captured source encoding facts; do not add an independent wrapper.
Extend the existing `responseType` body/envelope decision rather than a
renderer-local path.

One operation response plan owns classification for every current caller:
`documentPlanner.operation`, `operationResponseBodyMode` and
`renderer.operationResponses`. That plan keeps the exact-200 file protocol
conditions, so range and `default` never become `FileResponse`. The renderer's
singular success becomes success and informational collections plus
error-only state. One selector-derived name and spelling serves error naming,
shared-status disambiguation and `responseMappingPrefix`. A status mapping
needs a mapping block even when the response has no schema or headers.
`errorCloneInventory` calls `operationResponses` during name allocation. Selector
classification must therefore precede naming without recursion: it must not
require already-allocated envelope names. `Responses.Default` is appended as a
real response occurrence, so normalized selection closure retains its schemas
and headers. Range and `default` references flow through the existing
resolver. #1005 discriminator occurrence and declaration identities on body
schemas are preserved.

The reviewed #957 candidate retains `Analyze([]byte)` but changes
`AnalyzeSelected` to `(Source, Selection)` and `AnalyzePartial` to
`(Source, Selection, bool)`. `Source{Bytes: ...}` has no implicit filesystem
access; retrieval paths, reference roots and URI mappings stay explicit.
Confirm those APIs and allocation names on #957's actual committed source
before execution. In the ledger, `Codes` and `Default` become conditional and
`Extensions` stays rejected.

### 10.1 Required SSE item-schema inverse

The required SSE 2XX, singleton 201 and 200/201 imports use the OAS 3.2 parsed-event
envelope, not a whole-body string renamed as an event. Only a single normalized
text/event-stream media entry with itemSchema on a successful response selects
this path. itemSchema in 3.0/3.1 receives a versioned-field diagnostic before
SSE admission. Whole-schema-only raw/file import is unchanged when it has no
reserved derived extension. Whole schema plus itemSchema, unsupported sequential
media, multiple media choices and mixed typed-stream/raw successes reject when
the full contract cannot be preserved. Extension-only operations do not select
SSE; section 9.1 applies to their reserved extension.

The inverse consumes an inline closed object with only required data: string,
contentMediaType application/json and contentSchema containing the decoded
payload. Envelope/data annotations, openness and constraints outside that fixed
inverse receive targeted diagnostics. A named envelope or data component is
unsupported. The dedicated SSE owner checks and consumes the envelope and data
nodes without routing either through `p.schema`, `p.content` or `schemaEncoding`;
only the contentSchema payload enters `p.schema` normally. Unconditionally reject
the normalized content keywords in generic `schemaAdmission`, independently of
its object-constructor role. Do not add ContentSchema as a generic admission
child or give objectDeclaration special SSE permission. Every retained ordinary
schema, including unreferenced declarations and nested objectField children,
therefore retains the same strict content-keyword rejection. Ordinary referenced
payload declarations without those keywords stay supported.

Multiple successful handshakes must resolve their payload roots to the same
source declaration identity, then have equal complete normalized occurrence-
local siblings compared to each other, not compared against that declaration.
This admits separate inline envelopes whose contentSchema occurrences both
reference StreamEvent and carry the same sibling example even if StreamEvent
has no root example. Compare assertions, default presence, annotations and
ordered examples with their metadata. Distinct provenance does not imply unequal
semantics; equal pointers, names or samples do not prove equality. Preserve
numeric precision and presence with the existing normalized reference/value
owners. Use a deterministic pair guard for recursive comparisons and fail closed
on unsupported normalized fields. Retain every occurrence and diagnostic path.

Multiple unrelated inline payload roots are unsupported in this slice; a single
success may use an admitted inline payload. Sort canonical selector-key strings
by ascending lexical byte order, as in section 9.1, and choose the first only
after all payload comparisons pass. Keep all original branch occurrences,
assertions, examples, headers and source paths; the representative never replaces
those records. Different local samples/defaults/assertions or targets reject,
without first-wins selection.
The equal complete payload facts become the single StreamingResult occurrence
contract, including local root examples and schema annotations. Keep separate
source provenance and handshake records; do not render a second Body-only example
authority. The lexical representative selects an equivalent source record, not
a sampling context or an override of the event value.

When present, the known extension must equal the complete derived contract from
section 9.1, including examples and their presence at the corresponding root,
child and recursive-reference positions. Pair inline expansion with the resolved
payload graph using typed edges and a recursive pair guard, rather than requiring
textually equal `$ref` placement or interpreting literal `$ref` data. This is
conservative comparison of admitted contracts, not general schema equivalence.
Non-null mappings and projections remain unsupported; no wrapper/data-field
inversion or independent imported async annotation owner is introduced.

## Accepted and Rejected Examples

| Input `responses` / DSL | Result |
|---|---|
| `200`, `4XX` (object E), `default` (no schema) | result at 200; `Status4XX{…E, HTTPStatus}`; `StatusDefault{HTTPStatus}` with `Body(Empty)`; regenerated keys `200, 4XX, default` |
| `2XX`, `default` (ticket probe) | success `HTTPStatus`; 201 decodes as success; 404 as `StatusDefault` |
| `200`, `2XX` | status mode; 200 → `200` branch, 201 → `2XX` branch |
| `default` only | sole success; every final status decodes as the result |
| `404`, `default` | `default` success, domain 200-599 minus 404; `404` error |
| `4XX` only | error-only: `NoSuccessResponse()`; a 200 is `ErrInvalidResponse`; a nil service error is a framework violation |
| `200`, `404`, `4XX` | 404 → `404` error; 400 → `Status4XX`; publishing `Status4XX{HTTPStatus: 404}` is a violation |
| `2XX` with a body; service picks 204 | server violation; a client receiving 204 returns a terminal error |
| `5XX` body fails to decode | terminal error, not retried as `default` |
| `103` with `Link`, `200` | emitter and observer; no `StatusField`; never a result |
| `1XX` with a header | informational with `StatusField`; emitting 100 or 101 is refused |
| SSE `2XX` with `text/event-stream` | status mode; `SetResponse(201)` then `Open` → 201 stream with zero events visible to the client |
| SSE `2XX`, `Open` without `SetResponse` | pre-commit framework error |
| SSE singleton exact `201` | `Open` → 201; no `StatusField`; a required handshake header not set → pre-commit error; an optional one is omitted |
| SSE exact `200` and exact `201` | status mode: `Open` without `SetResponse` is a pre-commit error, not an implicit 200 |
| SSE with `Tag` | rejected: no result value exists at open |
| SSE handshake Content-Type `text/event-stream` or `text/event-stream; charset=utf-8` | accepted |
| SSE handshake Content-Type missing, `application/json`, `text/event-stream-extra` or malformed | client rejects before stream construction |
| JSON-RPC SSE notification completion | explicit suppression: no stream opened, no SSE headers |
| HTTP SSE `Close` before any event, exact `201` or status-selected with valid metadata | publishes the handshake and zero events once; a later nil handler return adds nothing |
| HTTP SSE `Close` in status mode without metadata | pre-commit framework error; nothing published |
| HTTP SSE `Close` after failure, suppression or publication, or repeated | no opening; earlier outcome kept |
| Non-exact built-in Problem error without native status storage | rejected; `ProblemResponse.Status` never qualifies |
| `103` or `1XX` with `content`; `101` | `response-status` |
| `2xx`, `20X`, `6XX` | `response-status` |
| DSL `Response(Status4XX)` without `StatusField` | rejected |
| DSL success and error on one key; `Tag` with `2XX`; `Response(StatusEarlyHints)` | rejected |
| JSON-RPC with `Status5XX`; file response with `2XX` | rejected |

## Contract Matrix

| Capability | Supported | Rejected | Deferred | Default | Source of truth | Placement | Routing | Naming | Equivalence | Proof |
|---|---|---|---|---|---|---|---|---|---|---|
| Final class/fallback | 2XX-5XX, default on unary, raw-body and SSE | JSON-RPC, gRPC, file, WebSocket, mixed keys | none | absent → `StatusOK` | `HTTPResponseExpr.Status` | expr → transportir → codegen/IR | `SelectFinal` | `Status4XX`, `default` | server domain = client selection | Lean `domain_iff_select`; `StatusSelection` |
| Multi-success | status mode; Tag for exact | Tag with non-exact | none | Tag when authored | expr | result envelope | `SelectFinal(successKeys)` | `status_<key>` | branch = selection of carried status | `StatusBindings` |
| Fallback-only, error-only | sole-success default; `NoSuccessResponse()` | no final key | none | n/a | importer classification | result / none | as above | `HTTPStatus` | no invented key | `StatusSelection` checked domain |
| Informational | exact 100-199 (not 101), 1XX; emitters, observers | content; authored 100/101; after final; HTTP/1.0 | none | not observed | `Informational` | `ExchangeWriter`; client trace hook | sink; `Got1xxResponse` | `Send<Op><Name>`, `With<Op><Name>` | emitted ⇒ observable before final | `ResponsePhases`; scratch proof |
| SSE handshake | singleton exact: literal status; several or non-exact successes: status mode via `SetResponse` | missing selection, missing required header, changes after open, Tag mode | none | literal status of a singleton exact success | handshake type (section 3 mode) | `SSEStreamWriter`, generated stream | first open trigger | `<Method>StreamResponse` | client status = committed status | `SSEOpen` |
| StatusField | required non-null Int with constraints; optional on exact | missing on non-exact, competing mapping, view loss, empty intersection | none | n/a | response-owned binding plan | result/error/informational/handshake | init arg last | `http_status` | field = wire status | `StatusBindings` |
| Violations | lifecycle error path | re-selecting a designed contract | none | n/a | handler lifecycle | `http/handler_lifecycle.go` | `EncodeResponse`/`encodeError` | unmapped | authored publication only in domain | `StatusSelection`; runtime tests |
| Importer | every key above | invalid keys, 101, 1xx content, no final key | none | exact path unchanged | document | Analyze + planner | n/a | `Status`+key | identical key set, refs kept | exporter symmetry |

## HTTP, JSON-RPC and gRPC

HTTP owns selectors and phases. JSON-RPC errors are RPC codes over HTTP 200,
so non-exact selectors, `Informational` and `NoSuccessResponse` are rejected
there. `GRPCResponseExpr.StatusCode` is untouched.

## Alternatives Considered

- Concretize on import. Rejected: it invents statuses.
- Exact 1xx as final responses (today). Rejected (`phases-info-final`).
- Informational values from the method result. Rejected: interim responses
  precede the result.
- Close informational admission at `WriteHeader`, or skip the owner lock.
  Rejected: a concurrent send's snapshot and restore races final header
  preparation (`phases-unlocked`).
- One budget for authored interim responses and the automatic 100. Rejected:
  a Loom server at its limit with `Expect: 100-continue` trips a Loom client
  (`phases-shared-budget`).
- Rely on `Got1xxResponse` error propagation. Rejected: Go discards it when a
  caller hook exists (golang/go#76669; scratch proof).
- SSE status from the first event. Rejected: `Open`, comments and empty
  completion commit before any event (`sse-first-event-status`,
  `sse-end-unpublished`).
- Reject error-only operations. Rejected: OAS §4.16 makes a success response a
  SHOULD, and inventing one misrepresents the source.
- Tag-only multi-success for imports. Rejected: tag and status could disagree.

## Proof Obligations

Commands and historical results are in the model READMEs; reconciliation did
not rerun them. Their `end`/return publication obligation applies to successful
completion, and their response status source is response-owned as clarified
above. Current callbacks, native Problem construction and raw/file physical
commit behavior remain implementation correspondence obligations. Two shared
interactions were newly identified: the explicit successful-publication and
protocol-suppression operations at one SSE terminal owner, including explicit
`Close` as successful publication (section 8), and the redirect final start.
Before implementing either, update the owning `SSEOpen`/`ResponsePhases` model
or its correspondence and run it opt-in, outside every gate. `SSEOpen` models
sequential publication, not close/flush failure, event encoding or
service-goroutine concurrency. `ResponsePhases` does not establish mutex order
or callback reentrancy. Direct race and runtime tests own those concerns.

Future OpenAPI checks use the repository's current Vacuum pin (v0.39.3 at
this reconciliation, `http/codegen/openapi/v3/contract_vacuum_test.go:20`),
not a historical tool version. #957's document-prefixed external-security URI
gate, tracked separately as Vacuum #1056 in its current prerequisite packet,
is not solved or waived by status-selection work. Its unresolved disposition
continues to block delivery of that prerequisite; ordinary green specimens are
not substitute evidence.

- **P1 Precedence.** Lean (unbounded, for its functions): membership
  invariance, covering, maximal specificity, and the declaration-order
  counterexample. TLC `StatusSelection` (bounded).
- **P2 Agreement.** Lean `domain_iff_select`; TLC `Agreement` (bounded).
  Runtime: compiled tests per accepted example, both declaration orders, a
  view, two media variants.
- **P3 No invented status.** Key sets and per-key schemas round-trip,
  including error-only and informational keys. libopenapi and pinned Vacuum
  on both outputs.
- **P4 Contracts.** `ValidateResponseContract` accepts exactly the domain.
- **P5 Ownership and no fallthrough.** TLC `SideIntegrity`, `NoFallthrough`
  and `InvalidBodyRejected`. The `accept-invalid` mutant passes the other
  invariants, which is why `InvalidBodyRejected` is required.
- **P6 Status-source isolation.** TLC `StatusBindings`. Runtime: body schemas
  exclude the field.
- **P7 Violations.** TLC classification only (`AuthoredPublicationInDomain`).
  Runtime: the violating status is never committed, headers are restored, the
  retry order holds, and error-only nil returns are violations.
- **P8 Phases and writer ownership.** Lean `phases_disjoint` and
  `switching_protocols_never_interim`. TLC `ResponsePhases` (bounded): final
  starts exclude concurrent sends, headers are isolated, HTTP/1.0 gate,
  separate 100 budget, honest emitter results. Runtime: race-detector tests
  with a goroutine sending during each final start, and HTTP/1.0, HTTP/1.1 and
  HTTP/2 servers.
- **P9 Client abort propagation.** Scratch proof (section 7). Runtime: the
  obligations listed in section 7.
- **P10 SSE handshake.** TLC `SSEOpen` (bounded): success sets exact `200`,
  exact `201`, exact `200`+`201` and `2XX`; required, optional or no handshake
  header. It checks the literal status for singleton exact successes, no
  guessed status in status mode, required header presence, a single final
  status and client agreement. Two controls reproduce the former 200 special
  case: exact `201` fails, and `{200, 201}` commits an unselected 200.
  Publication only, not media, close or concurrency. Runtime: every open trigger
  for exact `201`, exact `200`+`201` and `2XX` endpoints; required versus
  optional handshake headers; zero-event streams; pre-open errors.
- **P11 Admission parity and determinism.**

Model bounds: `StatusSelection` covers six final statuses, keys from
`{200, 2XX, 404, 4XX, default}` (at most three per endpoint), and a Boolean
body outcome. It has no 3XX, exact 500, 1xx or invalid statuses; named errors,
media, constraints and tags are abstracted. It does not establish P1, P2, P5
or P7 in general. `ResponsePhases` uses `Limit = 2`; the design value is 8.
`SSEOpen` covers four success sets (exact 200, exact 201, exact 200 plus 201,
and 2XX) and three service steps.

## Acceptance Trace

| Issue acceptance item | Where met | Evidence |
|---|---|---|
| Preserve fallback/range contracts alongside concrete responses | sections 1, 3, 10 | P1-P3 |
| Concrete-over-range precedence | section 1 | Lean P1/P2; `decl-order`, `server-loose` |
| Error schemas | sections 3, 10 | envelope rows; error-only rows |
| Runtime status ownership | sections 4, 6-8 | P2, P6-P10 |
| Round-trip the status map without inventing a status | sections 9-10 | P3, including `1XX`, `default`, error-only |
| Verify actual response selection | sections 5, 7, 8 | P2, P5, P8-P10 runtime tests |
| `1XX`-`5XX` and `Responses.default` | sections 1, 7, 10 | informational rows; ledger |
| Focused regressions and ledger | Projection surfaces | ledger rows |
| Existing OpenAPI validation and generated-code checks | P3, P11 | libopenapi, Vacuum, compile, determinism |

## Missing Evidence

No real consumer document with range, fallback or informational responses
was inspected.

## Review Status

Earlier reviews applied: ownership, StatusField source, domains, order,
204/304, violation routing, publication origin, and `InvalidBodyRejected`.

This revision (final review):

1. HIGH: client abort propagation uses a per-`Do` admission owner. The first
   Loom error is stored under a lock before cancel, read under the lock after
   `Do`, and preferred deliberately. Close errors are joined. Cancel is
   retained until EOF or Close. Budgets are counted per transport exchange.
   Caller-hook guarantees and limits are stated. Upstream evidence is linked
   (golang/go#76669); the scratch proof was extended.
2. HIGH: one exchange writer owner. `beginFinal` precedes all final header
   work for result, error, SSE, upgrade, file and raw starts. The
   `ResponsePhases` concurrency boundary and the `unlocked` control were added.
3. HIGH: authored emission excludes 100. The client has a separate budget of
   one 100, and the scratch proof shows `103`×8 plus `100`
   (`shared-budget` control).
4. HIGH: the SSE handshake follows the section 3 mode. A singleton exact
   success (any code) publishes its literal status with no `StatusField`.
   Several or non-exact successes use status mode through `SetResponse`
   before every open trigger. Tag mode is unavailable for SSE. Required
   handshake headers fail before commit; optional ones are omitted. Metadata is
   immutable after open. The client selects before stream construction, and
   hardcoded 200 paths are removed (`SSEOpen`, including two
   200-special-case regressions). Browser `EventSource` is documented as
   200-only.
5. MED: exact informational responses do not require `StatusField`.
6. MED: error-only operations are supported through `NoSuccessResponse()`.

Also: the mixed exact-key compile failure is reproduced, and
`MaxInformationalResponses = 8` is settled.

## Remaining Policy Choices

None. Routine choices are settled under the full-scope direction.
