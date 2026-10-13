# Multipart request source budget — #759

Implementation on `d3cb9ad26` after independent readiness approval.

1. [x] Trace both generated entries, reader API, existing body policy and public
   limits. The accepted contract bounds complete request bytes, not part values.
2. [x] Reproduce empty-part overhead and post-boundary/early-callback bypasses;
   preserve complete-wire exact admission, malformed/callback errors and body
   ownership. Extend existing request-limit and custom-multipart harnesses.
3. [x] Model source-budget/completion ordering: current value budget, raw budget
   without completion, and bounded completion before successful dispatch.
4. [x] Freeze compact owner proposal and bounded evidence for readiness review.
5. [x] After approval, implement one source owner shared by built-in collection
   and generated custom callbacks; remove value bookkeeping and reader API.
6. [x] Run focused runtime/generated checks and compare representative parent,
   candidate and independent-repeat output before final review and integration.

## Owner and contract

`ReadMultipartForm` and `ReadMultipartFormWithLimit` must take the original
`*http.Request`, since an opaque `*multipart.Reader` cannot expose raw headers,
boundaries or buffered/remaining bytes. One shared callback operation owns
request validation, positive limit, raw budget, parser creation and successful
completion. The built-in collector and generated custom adapter call it; the
custom decoder signature `func(*multipart.Reader,*Payload) error` stays intact.
The old reader-taking API is replaced, without a misleading compatibility path.

Reuse `requestBodyLimitReadCloser` as the byte owner. Known oversized length
rejects before callback invocation. Unknown/chunked length is counted identically.
Multipart EOF is only a final boundary: successful parsing or an early successful
callback must consume the remaining raw source within the same budget before
service dispatch. Count bytes already read ahead once, retain the one-byte
probe for exact limits, and include epilogue. Completion must preserve physical
read errors already observed alongside data rather than letting buffered parser
success hide them. No whole-body buffering, per-part estimates or new allocator.

Callback/parse errors do not trigger extra draining. Preserve errors and normalize
observed size failure to the existing 413/request_too_large contract. The server
retains Close ownership; no helper closes or transfers the request body. Existing
request cancellation/deadline policy remains unchanged. The custom callback owns
payload decoding, while the shared entry owns admission before success.

The original report's four-byte value-budget example must become a full encoded
wire-length boundary test: multipart framing itself consumes bytes. Valid forms,
files, repeated and unnamed fields remain supported. Direct runtime API users
pass their request instead of an already-created parser; applications regenerate
server adapters. No client callback API change or new public policy is needed.

## Bounded acceptance

Reuse runtime multipart tests and existing HTTP request-limit/custom-binding
modules. Direct small-body cases cover overhead, exact/+1, known/unknown,
epilogue, data-plus-error and Close ownership. Existing generated limit tests
retain JSON/text/form/raw controls; add default overhead and mounted custom
completion cases, asserting 413 problem detail/code and zero dispatch. Keep
existing custom payload/metadata controls. Compare only affected multipart
renderings and generated modules; ticktock streaming fixtures have no multipart
contracts. No new broad transport harness or corpus sweep.

## Approved implementation

Use `DecodeMultipartRequestWithLimit(req, maxBytes, decode)` as the single public
callback operation. `ReadMultipartFormWithLimit(req,maxBytes)` collects a form
through it; `ReadMultipartForm(req)` chooses RequestBodyLimit from the context.
The generated built-in path passes its request directly; the custom adapter calls
the same operation with a closure binding the existing payload argument. No
consumer initializes its own parser or guesses byte usage afterward.

Reuse an installed policy counter when its immutable limit matches; direct
readers establish the same counter once. An explicitly different limit layers
onto any existing policy, preserving both constraints. Retain the first physical
read error alongside the existing overflow state in
`requestBodyLimitReadCloser`, including errors returned with data and the exact
limit probe. Recording this observation need not change its existing Read return
semantics for raw-policy users. The multipart owner checks that observation on
callback success before trying further reads; EOF is successful completion,
other physical errors reject. A callback failure is returned without more reads,
except an already-observed budget error remains the size classification. The
counter and callback operation own these decisions once, not each template.

Readiness evidence on the unchanged production parent: six raw-size cases and
one buffered source-error case fail as expected; both exact-wire runtime controls
pass. Mounted default empty-part overhead invokes the built-in endpoint and
returns 204 instead of 413. Custom unknown/configured and default epilogues also
return 204 and invoke the endpoint; known-length oversize returns 413 but invokes
the callback before rejecting. Existing malformed/callback/payload and exact
configured controls remain passing, including explicit limits above the 32 MiB
default. A callback that consumes the final multipart boundary still accepts an
oversized epilogue; callback failure with an unread oversized tail retains its
original failure and does not invoke the service. The bounded model's current value owner
fails Admission at 14 distinct states; raw-only owner fails at 15; complete owner
passes 20 generated/16 distinct states. These checks establish the owner gap and
completion requirement without expanding the request-decoder contract.

Historical readiness was frozen in `/tmp/loom-759-readiness-manifest.json` and
approved by `/tmp/loom-759-root-readiness-verdict.json` before production edits. Parent generated output is retained from
the two existing modules. Its capture overlay changes only output directories;
no production source is overlaid. Later extra callback/budget test cases do not
change either DSL or the generated-output baseline.

## Final bounded validation

Full HTTP runtime race tests pass (6.845s). The existing generated request-budget
and custom callback modules pass after installed-policy reuse (7.166s); earlier
query/custom-text binding controls pass with the same generator (13.381s combined
run). Eight owning snapshots are regenerated; their tests plus five existing
multipart compiler designs pass (4.970s). Scoped lint reports zero issues.
Focused controls retain raw-reader retry behavior, reject a wrapped EOF as a
physical failure, prevent a callback from hiding either its own or a stricter
installed policy's size error, and check callback failure reads/closes nothing.

The two saved parent modules contain 26 generated artifacts. Candidate and
process-isolated repeat agree exactly; only the built-in and custom server
`encode_decode.go` files differ from parent. Those are the intended shared-owner
call changes (including the custom closure's multipart import). All other bytes
are unchanged. Parent modules compiled during the expected failing regression
runs; candidate generated tests and explicit vet pass. Capture overlays modify
only existing test output paths and skip duplicate test/vet during candidate
and repeat capture; production is never overlaid. The approved model is unchanged.

Delivered and pushed as `6a549820f`; #759 is confirmed closed. Independent
final and integration reviews left no findings. Root runtime race and mounted
built-in/custom controls passed; integration retained the exact functional diff.
Cleanup preserved reproduction inputs, compact evidence and artifact hashes in
`/tmp/loom-759-cleanup-receipt.json`; safe worktree reuse is recorded separately.
