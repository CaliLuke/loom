# Request cookie publication (#715)

Readiness independently approved; implementation on `18ae28c9e` is complete; final independent review pending.

1. [x] Trace every request-cookie consumer and the response grammar owner.
2. [x] Reproduce invalid-value publication in generated HTTP and JSON-RPC clients;
   retain ordinary, comma, space, empty, absent, default and native-zero controls.
3. [x] Freeze the shared runtime publication proposal and bounded evidence for review.
4. [x] After approval, implement one request-cookie publication function in `http`:
   validate the actual name/value with `net/http.Cookie.Valid`, then call AddCookie.
   Generated requests use this owner after shared typed formatting and propagate
   an encoding error before the HTTP doer runs. No escaping or decoded-value change.
5. [x] Verify direct runtime rejection has no header mutation, generated consumer
   behavior, affected exact snapshots, representative output parity/compilation,
   then obtain independent final review before the atomic ticket commit.
6. [ ] Independently review the exact final diff, deliver one atomic commit and
   preserve compact evidence before removing inactive validation artifacts.

The response owner already calls Cookie.Valid. Request validation should validate
only request wire fields (Name and Value), not response-only attributes that
AddCookie never sends. This avoids importing response deployment/prefix policy.
No parallel grammar or policy: the standard library owns both directions.
The helper delegates valid additions to standard `Request.AddCookie`, retaining
its existing publication behavior (including its handling of multiple Cookie
lines); it leaves all header values unchanged on rejection. The accepted
preservation control uses a preexisting single Cookie line. Generated encoders
may have prepared earlier fields, but must return before the doer. Nil/empty/default
selection and physical scalar conversion remain their existing owners.

No concurrency/state model is needed: the corrected operation is a synchronous
validate-before-mutate boundary with no retries or ownership transfer. Direct
mutation/no-mutation checks and generated no-send tests establish correspondence.
#710 cookie-name design admission remains a distinct evaluator issue.

## Observed owner and result

On 53849ce33, both actual generated modules compiled under Go1.27.2/jsonv2
and ran with race instrumentation. Each module has 12 failing invalid-value
cases (ordinary, optional, API-key and inferred session cookies × semicolon,
quote and Unicode), plus 23 passing controls. The captured HTTP request proves
`a;b` and `a"b` become `ab`, and `café` becomes `caf`, with no client error.
The existing response owner rejects all the same invalid values before writing.
Comma, space, plain, supplied empty, omitted pointers, default-bearing native
empty string and Boolean false all keep their accepted wire behavior.

`requestEncoderSource` is the only production AddCookie call. JSON-RPC calls
HTTP ClientEncodeDecodeFile, retaining its request encoder. HTTP ordinary/SSE/
WebSocket clients call that encoder before Do/DialContext. JSON-RPC ordinary/SSE
calls it before Do. JSON-RPC WebSocket shared connection bootstrap does not call
this encoder and dials with nil headers; that path does not publish modeled
request cookies and is not silently claimed covered by this repair. Transport-
owned browser session cookies (empty field name) are deliberately omitted from
payload encoder analysis; their cookie jar/application boundary is unchanged.

Use `AddRequestCookie(req *http.Request, name, value string) error` as the narrow
runtime publication API: construct Name/Value only, use Cookie.Valid, then
AddCookie. These are the fields actually serialized in requests. Delete the
request template's transient Cookie literal/response-only attribute branches;
response-only DSL setters cannot author them in a request. Keep scalar formatting
and pointer admission untouched; wrap failure with the existing ErrEncodingError
at the generated context boundary. That helper's existing message says request
body even for other request encoding failures; changing global diagnostics is
not required for this ticket.

Before final review add direct helper tests for preexisting Cookie header
preservation, unchanged header on invalid input, invalid name, empty/space/comma,
backslash/control bytes, and ordinary valid additions. Strengthen the generated
witness to count Doer calls explicitly (current witness captures actual network
publication), assert encoding_error/service/method, and retain all controls.
Reuse existing primitive/custom-text cookie fixtures for formatter composition;
add bounded HTTP stream and RPC SSE no-dial/no-send checks if structural owner
assertions do not establish their call ordering. No new streaming lifecycle model.
Regenerate only affected request-cookie goldens and compare both modules against
preserved parent outputs plus one ordinary no-cookie design. Repeated process
outputs must match, compile once per distinct output. No OpenAPI contract changes
are intended; no unrelated full corpus or stale-golden maintenance.

## Reproduction commands and remaining gates

Run from numeric-comparison with GOEXPERIMENT=jsonv2 and shared GOCACHE:
`LOOM_DIR=$PWD go test -overlay /tmp/loom-715-readiness/overlay.json ./http/codegen ./jsonrpc/codegen -run '^TestRequestCookiePublicationReadiness$' -count=1`.
This writes generator-owned outputs in httpcookies and rpccookies. Copy the
matching `http_cookie_harness_test.go` / `rpc_cookie_harness_test.go` to that
module's `cookie_test.go` (the existing files already match), run `go mod tidy`,
then `go test -race -v ./...`. Current expected exit is 1 with 12 leaf failures
and 23 leaf passes per module. Harness inputs and every generated file hash are
in the manifest. Generation uses existing package module renderers, not a new
CLI harness. Frozen #686 source hashes are unchanged.

Readiness approved in `/tmp/loom-715-independent-readiness-verdict.json`.
The scratch witnesses and parent outputs remain preserved; exact final review
is required before delivery.

## Implementation and final acceptance

The runtime `AddRequestCookie` constructs only Name/Value, validates them with
`Cookie.Valid`, and then delegates to `Request.AddCookie`. The shared generated
request encoder calls it after existing formatting/pointer checks and wraps an
error with the service/method `encoding_error` before networking. The old literal
and all response-only branches have been removed. Existing auth-binding and
primitive-encoder assertions now reference the shared publication call rather
than the removed literal; their absence/duplicate controls are retained.

Direct runtime tests cover preexisting single-cookie-line preservation, invalid name, empty,
space/comma, quote/semicolon/backslash/control/non-ASCII rejection without any
header mutation (including multiple preexisting Cookie lines on rejection), and request use of a secure-prefix name without importing
response deployment requirements. The actual generated modules retain all 35
ordinary/presence cases per transport and add HTTP SSE/WebSocket plus JSON-RPC
SSE invalid-cookie controls. Explicit doer/dialer counters prove rejection before
network publication, and errors retain encoding category plus service/method.
The original parent and strengthened current-parent runs fail as expected.

The three-module comparison includes those two HTTP/RPC consumers and existing
`PayloadQueryStringDSL` as an unrelated control. At parent `18ae28c9e`, in-process
generation restores only the exact parent request template through a Go overlay;
subprocesses compile and execute already-rendered files. No child CLI clears or
reinterprets that overlay. There are 45 artifacts per stage: only the two cookie
client `encode_decode.go` files change, and all 45 candidate files exactly match
an independent process repeat. Candidate modules compile, run mounted race tests
and pass vet (HTTP 8.157s, RPC 6.822s, including the ordinary query control).
The repeat only regenerates already-validated output; it does not rerun builds.

Seven affected cookie goldens were regenerated with the owning `TestClientEncode`
`-update` path. Their only differences replace unvalidated Cookie literals with
the shared call and contextual error propagation. Existing native primitive and
custom-text cookie wire tests, auth/session controls and exact goldens pass under
race (HTTP runtime 2.759s, HTTP codegen 15.079s, RPC codegen 1.634s). The final direct runtime check passes (1.554s), and browser-owned/session
omission controls pass under race (2.287s). Final scoped lint reports zero issues. All runs use Go 1.27.2 with JSON v2 and explicit local
Loom source. Checked-in ticktock/mixedtick designs have no request-cookie mappings;
no fixture regeneration or OpenAPI difference is needed.

The first lint attempt found six blank-line formatting issues in transferred
prototype test code; those were corrected without changing DSL or harness
behavior. Final lint and independent generation cover that correction. Source,
commands, original failures and artifact hashes are retained in
`/tmp/loom-715-final-manifest.json`. No general concurrency model, transport
lifecycle rewrite or JSON-RPC WebSocket bootstrap change is part of this repair.
