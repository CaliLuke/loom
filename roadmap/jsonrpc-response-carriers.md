# Uniform JSON-RPC response bodies (#905)

The user accepted one response contract: application values are carried in
`result` or `error.data`. Authored JSON-RPC response Header/Cookie mappings are
rejected, including inherited errors and streaming endpoints. This replaces
working single-call HTTP metadata mappings because batch envelopes, SSE events
and multiplexed WebSocket messages cannot carry independent HTTP header sets.
Migrate those mappings to the body and regenerate both ends. Ordinary HTTP
response mappings and JSON-RPC request metadata retain their existing contracts.
Framework-owned view identity, CORS and middleware/session HTTP headers remain
separately owned; this rule does not prohibit physical HTTP metadata itself.

Delivered and confirmed closed as `4df3a5ff0`; independent final and integration
reviews approve the exact change with no unresolved findings.

## Sequence

1. [x] Preserve current mounted error failure and add direct parent-red admission
   cases for success/error, inherited API/service declarations, SSE and WS.
   Keep valid HTTP response, RPC request and body-only controls.
2. [x] Enforce the capability once at evaluated JSON-RPC endpoint response
   admission, after inherited declarations are prepared. Every renderer derives
   from the admitted plan; no per-renderer exceptions or envelope extension.
3. [x] Delete unreachable response metadata capture and decoding paths. Preserve
   body projection, safe errors, notifications and framework view transport.
4. [x] Migrate existing positive metadata specimens to body mappings while
   retaining typed reconstruction and projection assertions. HTTP-owned header
   formatting controls remain at the HTTP owner.
5. [x] Run affected expression/generator checks and bounded mounted unary,
   batch, SSE, WS, safe-error/notification and HTTP/request controls. Compare
   representative parent/candidate/independent-repeat output; compile outputs.
   Regenerate affected checked-in fixtures through the generator only.
6. [x] Obtain independent exact-final review before one atomic commit/delivery.

The original failure is retained in `/tmp/loom-774-mapped-header-repro/` and the
current-main reproduction/owner ledger in
`/tmp/loom-jsonrpc-response-metadata-readiness.json`. Closed #817 fixed body source
preparation; #774 fixed discriminator selection. This ticket closes the remaining
carrier mismatch rather than repeating either repair.

Rejected alternative: add an explicit HTTP-unary-only mode and reject batches
for those endpoints. It would preserve old metadata integrations at the cost of
another public transport mode and weaker compositional behavior. Silently moving
authored headers into bodies, merging batch values, or inventing envelope fields
would discard author intent or create an undocumented protocol.

This is static carrier admission and generator deletion, not a new concurrent
protocol. Existing framing/observation owners remain unchanged; no additional
formal state model is needed unless implementation reveals an ordering change.

## Completed implementation and bounded evidence

The endpoint's existing JSON-RPC validator owns admission after inherited errors
have been prepared. It rejects every success and error response mapping before
transport analysis. The removed paths are the generated response capture type,
metadata-copy/filter loop, metadata-only result binding, response Header/Cookie
decoders and their private scalar conversion helpers. Shared HTTP response body
encoders remain because their projection/codec API is still valid; no separate
JSON-RPC body formatter was added.

Existing metadata specimens now carry named collections, defaults, error
identities and details in bodies. Mounted batch requests preserve per-envelope
values while unrelated middleware headers/cookies survive. Dynamic/fixed views
retain `loom-view`; CORS and request SessionSecurity controls remain green.
The selected projection fixture's pre-existing required/excluded-field failure
was delivered separately as #907 and is part of this patch's parent.

- 24 parent-red admission cases cover unary, SSE and WebSocket success and
  method/service/API errors; ten HTTP/request controls remain admitted.
- Actual CLI generation accepts both original mapped-header/mapped-cookie designs
  at the parent and rejects them before publishing `gen` with this change.
- Five representative modules contain 58 generated files: parent, candidate and
  independent-process repeat are byte-identical using the same admitted designs.
  Builds and vet pass; mounted collection, bodyless, codec-failure, selected
  projection, typed-error, view, batch and notification checks pass.
- Three checked-in fixture designs produce 55 files with identical parent,
  candidate and repeat output; parent and candidate build/vet pass. Each fixture's
  checked-in `gen/loom.json` already has a different design digest from current
  parent regeneration. This change introduces no additional drift and does not
  adopt unrelated manifest changes.
- Full expression/DSL/JSON-RPC runtime races pass (7.583/1.445/2.049 seconds).
  HTTP response-metadata controls pass (3.992 seconds); scoped lint reports no
  issues. Current projection and source-binding models pass with 97 and 17
  distinct states; the historical current/owned configurations retain their
  expected negative/positive results.

Evidence is frozen with the final source packet at
`/tmp/loom-905-final-manifest.json`; artifact ledgers record exact hashes and
commands. No full design corpus was run. Joined-error ownership (#908), dynamic
view protocol redesign and middleware-owned physical metadata are separate
owners; this repair does not claim to resolve them.
