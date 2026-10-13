# Optional JSON-RPC failure reporting (#723)

Implementation checkpoint: independent readiness review approved the shared
owner. Implementation started on delivered `5c6535b6d` plus the approved #682
candidate; final acceptance uses delivered `0277e0f36`, including #682 and the
separate #878 snapshot alignment. This owner
is independent of request completion.

1. [x] Reproduce nil callback failures at mounted and exported handler entries;
   preserve provided-callback and correlated-request controls.
2. [x] Inventory all generated invocations and runtime reporting consumers.
3. [x] Freeze the shared owner and bounded acceptance for independent review.
4. [x] After approval, implement and compare bounded generated output, compile,
   exercise transport controls and run affected lint.
5. [ ] Obtain fresh exact final review before parent integration and commit.

The public error callback is optional. Runtime `handleHTTPFailure` previously owned
nil-safe invocation across HTTP, mixed, SSE and WebSocket. Generated standard
handlers bypassed it at notification decode/endpoint failures and response encoding
failures. The generated standalone `encodeJSONRPCError` bypasses it too.
Normalizing only `server.New` misses public per-method HandlerInit constructors.
WebSocket HandlerInit has no error callback and returns typed errors to its
connection adapter; SSE HandlerInit returns errors to its runtime caller. Neither
should gain redundant diagnostic calls or change suppression/return semantics.

Approved owner: promote the existing runtime implementation to a documented
`ReportFailure(ctx, w, handler, err)` function, removing its private old name.
Generated callbacks and every existing runtime consumer call this same owner.
Nil handler is a no-op; a supplied handler is invoked synchronously once with the
same context, writer and error. Do not recover user callback panics, replace error
wrapping, write responses, change observations, or normalize per constructor.
The application owns its callback behavior. The protocol runtime continues to
supply notificationWriter when dispatching notifications.

This is a single conditional invocation with no new ownership, concurrency or
protocol ordering. A separate state model would merely mirror that conditional;
direct invocation identity tests and mounted protocol tests establish the useful
obligations. Existing protocol models remain unchanged. If implementation starts
changing callback timing or response framing, stop and reassess that boundary.

Bounded acceptance: nil/provided handlers through mounted HTTP, mixed HTTP and
exported direct HandlerInit; decode/endpoint notification failures; normal
correlated errors; success/error response encoder failures; result capture and
bodyless framing. Callback controls check exact cause and context, count, writer
identity where directly supplied, and notification response suppression. Existing
SSE/WebSocket runtime tests are unchanged controls. Generated output comparison
must show only shared reporting calls and documentation, no codec/body/lifetime
changes. No repair of callback policy for previously unreported SSE notifications.

## Historical readiness evidence and owner inventory

The generated matrix has 80 cases: two designs (ordinary and mixed SSE/HTTP),
two entry paths (mounted route and direct HandlerInit), nil/provided callback,
and ten failure/control modes. Parent produces 28 expected nil-pointer panics
and 52 passing controls. Each design compiles; all supplied-callback controls
prove exactly-once reporting and original wrapped cause/context. Notification
callback writes are suppressed by the mounted runtime's notification writer.
Direct constructor controls preserve the writer supplied by that caller.

The eight generator invocation sites are seven in `handler_sections.go`
(notification decode, notification endpoint, bodyless capture, bodyless envelope,
result capture, captured envelope, ordinary result envelope) and one in
`top_level_sections.go` (standalone error-envelope encoder). Bodyless metadata
capture shares the reporting rule; its ordinary header-only encoder has no
failure in this specimen. Do not manufacture a production exception for that
unreachable concrete branch. Preserve it structurally through the shared call.

Runtime consumers are `server.go` and `stream_server.go`: envelope/body/dispatch
failures, SSE handling, mixed negotiation, WebSocket upgrade/configuration/run/
close failures and standalone error encoding already invoke `handleHTTPFailure`.
Direct nil/provided tests prove its current identity-preserving contract. Existing
HTTP suppression, SSE, mixed negotiation/read failure and WebSocket lifetime
controls pass under race. The proposed exported owner relocates this policy,
without moving call sites, changing which failures are reported or adding a
second nil check in generated adapters.

Before final review, compare ordinary and mixed generated artifacts at parent,
candidate and an independent repeat. Add the direct shared-owner call assertions
and update affected goldens through their owning tests only. The new public
runtime helper needs GoDoc; document optional generated callbacks and preserve
SSE/WS suppression behavior. Independent readiness approval was recorded in
`/tmp/loom-723-independent-readiness-verdict.json` before production edits.

## Implementation acceptance

`ReportFailure` is the existing runtime callback invocation, promoted in place
with explicit GoDoc. Runtime consumers and all eight generated invocation sites
call it. The old private name and direct generated callback calls are removed.
No callback timing, observation, encoding, notification suppression or return
semantics changed. Generated constructor docs and the error-handling guide
explicitly state the optional callback contract and regeneration requirement.

All 80 mounted/direct cases pass. Shared-owner identity and existing HTTP, SSE,
mixed and WebSocket controls pass under race. The ordinary (11 files) and mixed
(13 files) designs change only their server/server.go reporting calls and docs;
independent-process generation is byte-identical. The checked-in JSON-RPC
ticktock fixture has 19 files, matches parent generation exactly, and changes only
server/server.go. Its candidate and independent repeat match exactly; parent and
candidate fixture apps pass race/test and vet.

Thirty direct JSON-RPC golden-owning tests were regenerated. They exposed three
pre-existing SSE snapshots from #790; those parent-generated alignments are
delivered separately as #878. After that alignment, this ticket's golden changes
contain only reporting calls and optional-callback documentation. Affected lint
includes the codegen testdata package and passes. No whole-corpus audit was run.

The final manifest records source/evidence hashes and every intended artifact
difference. Independent final review and parent integration remain open.
