# Status selection, response phases and SSE handshakes (#976)

Design evidence for the accepted #976
[response-selection design](../../response_selection_design.md). The
implementation correspondence below names the Go regressions that check it.
These modules are separate from `../response_order`, whose
"default" is the untagged success response, not the OpenAPI `default` key.

## Modules

`StatusSelection.tla` explores one unary endpoint: key ownership validation
(operations may declare only error keys), the service choosing a designed
branch and status (or a success return on an error-only operation), the
encoder's selection check and publication origin, and the client's once-only
selection and decoding. `Select` is the design's membership-precedence
algorithm; `MostSpecific` is an independent max-specificity oracle; `Domain`
is the server runtime-domain formula written independently of `Select`.

`ResponsePhases.tla` explores one exchange with a concurrent informational
sender: a three-step send (snapshot and replace the header map, write the
interim status, restore), five final starts (unary result, error, SSE open,
WebSocket upgrade, redirect) as start → header preparation → commit, net/http's
automatic `100 Continue` written on its own connection writer, and the
client's in-order processing with an optional observer. The checked design
uses one writer owner: a send and a final start exclude each other, and the
final start closes informational admission before preparing any header.

`SSEOpen.tla` explores SSE status publication under the section 3 mode of the
endpoint's success set: a singleton exact success (`200` or `201`) publishes
its literal status, while exact `200`+`201` and `2XX` use status mode, where
typed handshake metadata must select a key. Handshake headers are none,
optional or required. Transitions cover `SetResponse` (only in status mode,
or for a singleton exact success with headers), every open trigger (`Open`,
`SendComment`, `Send`, `Complete` with or without a message, logical successful `end`,
handler return), errors before and after open, and the client's key
selection before stream construction.

## Reproduce

Toolchain: Amazon Corretto 25.0.5, `tla2tools.jar` 1.7.4 (TLC 2.19, rev
`5a47802`), SHA-256
`936a262061c914694dfd669a543be24573c45d5aa0ff20a8b96b23d01e050e88`. From this
directory:

```sh
for cfg in checked decl-order server-loose fallthrough mixed accept-invalid accept-invalid-blind publish-violation; do
  "$JAVA" -XX:+UseParallelGC -cp "$TLA_JAR" tlc2.TLC -workers 1 \
    -config "$cfg.cfg" -metadir "/tmp/loom-976-tlc/sel-$cfg" StatusSelection.tla
done
for cfg in phases-*.cfg; do
  "$JAVA" -XX:+UseParallelGC -cp "$TLA_JAR" tlc2.TLC -workers 1 \
    -config "$cfg" -metadir "/tmp/loom-976-tlc/${cfg%.cfg}" ResponsePhases.tla
done
for cfg in sse-*.cfg; do
  "$JAVA" -XX:+UseParallelGC -cp "$TLA_JAR" tlc2.TLC -workers 1 \
    -config "$cfg" -metadir "/tmp/loom-976-tlc/${cfg%.cfg}" SSEOpen.tla
done
```

`checked.cfg`, `accept-invalid-blind.cfg`, `phases-checked.cfg` and
`sse-checked.cfg` exit 0. All other configurations must exit 12 on the named
invariant.

## Recorded results (2026-11-05, source `f0c7a2b8c`, `-workers 1`)

| Configuration | Initial | Generated / distinct | Result and witness |
| --- | --- | --- | --- |
| `checked` | 2,550 | 88,970 / 59,860 | pass: Precedence, SelectionMatchesOracle, Agreement, NoFallthrough, SideIntegrity, InvalidBodyRejected, AuthoredPublicationInDomain, UnmappedAliasing |
| `decl-order` | 2,550 | 34,863 / 33,133 | SelectionMatchesOracle: `{2XX, default}`, 200 selects `default` |
| `server-loose` | 2,550 | 35,261 / 33,531 | Agreement: `2XX` branch publishes 200 beside exact `200` |
| `fallthrough` | 2,550 | 34,863 / 33,133 | NoFallthrough: `2XX` body fails at 200 and `default` is returned |
| `mixed` | 2,550 | 121,083 / 121,083 | SideIntegrity: `2XX` owned by success and error; an error at 200 decodes as success |
| `accept-invalid` | 2,550 | 34,211 / 32,481 | InvalidBodyRejected: `{2XX}`, 200, invalid body returned as a `2XX` value |
| `accept-invalid-blind` | 2,550 | 88,970 / 59,860 | pass: the same mutant satisfies Precedence, SelectionMatchesOracle, Agreement, NoFallthrough and SideIntegrity |
| `publish-violation` | 2,550 | 34,214 / 32,484 | AuthoredPublicationInDomain: an out-of-domain authored status published as authored |
| `phases-checked` | 384 | 780,672 / 638,240 | pass: InterimBeforeFinal, InterimNeverTerminal, FinalAgreement, FinalHeadersIsolated, No1xxToHTTP10, NoAuthored100or101, EmitHonest, ClientBounded, LoomPeersNeverAbort, DeliveredSelected |
| `phases-unlocked` | 384 | 24,042 / 15,439 | FinalHeadersIsolated: send replaces the header map, the final start prepares and commits, and the final 200 carries the interim `Link` |
| `phases-late` | 384 | 37,682 / 30,143 | EmitHonest: admission left open after commit; net/http drops the write but the emitter reports sent |
| `phases-leak` | 384 | 37,520 / 29,580 | FinalHeadersIsolated: no restore after the interim write |
| `phases-http10` | 384 | 3,238 / 2,596 | No1xxToHTTP10 |
| `phases-unbounded` | 384 | 663,095 / 575,451 | ClientBounded: observed client accepts a foreign third interim response |
| `phases-shared-budget` | 384 | 538,723 / 464,354 | LoomPeersNeverAbort: two authored 103s plus the automatic 100 exceed a shared budget of 2 |
| `phases-info-final` | 384 | 34,467 / 27,018 | FinalAgreement: legacy exact-103 "response" yields 103 and a bare implicit 200 |
| `sse-checked` | 12 | 3,254 / 726 | pass: ExactLiteral, NoGuessedStatus, RequiredHeaderPresent, SingleFinal, ClientAgreement |
| `sse-status-200-special-exact` | 12 | 41 / 28 | ExactLiteral: the former "exact means 200" rule turns a singleton exact `201` stream into a framework error |
| `sse-status-200-special-multi` | 12 | 75 / 44 | NoGuessedStatus: `{200, 201}` treated as implicit 200; `Open` commits 200 with no selection |
| `sse-first-event-status` | 12 | 75 / 46 | NoGuessedStatus: `Open` before any event commits 200 for `{200, 201}` |
| `sse-mutable-after-open` | 12 | 1,306 / 399 | NoGuessedStatus: metadata changed to 201 after a 200 commit |
| `sse-end-unpublished` | 12 | 47 / 33 | ExactLiteral: unopened return publishes 200 for a singleton exact `201` |
| `sse-post-open-remap` | 12 | 182 / 93 | SingleFinal: an error after open published as a second status |
| `sse-client-200` | 12 | 280 / 129 | ClientAgreement: a client requiring 200 rejects a committed exact `201` stream with zero events |
| `sse-skip-header-check` | 12 | 33 / 25 | RequiredHeaderPresent: `Open` commits without a required handshake header |

## Assumptions and limits

`StatusSelection` covers six final statuses (200, 201, 204, 404, 409, 500),
keys from `{200, 2XX, 404, 4XX, default}` with at most three per endpoint, a
Boolean body outcome, and no 3XX, exact 500, 1xx or invalid statuses; named
error grouping, media, StatusField constraints and tags are abstracted. It
checks P1, P2 and P5 only within these bounds; the Lean module
`../../lean/status_selection` proves the universal selection claims for its
functions. `AuthoredPublicationInDomain` is classification only; commit timing,
header restoration and the lifecycle retry are runtime-test obligations.

`ResponsePhases` uses one final contract per final kind, informational keys
from `{103, 1XX}`, at most three emission attempts from `{100, 101, 102,
103}`, `Limit = 2` (the design value is 8), one retained sender, interleaved
atomic steps, and net/http behavior verified by the scratch proof
(`/tmp/loom-976-trace-proof`): 1xx writes use the live header map, writes after
the final header are dropped, and `Expect: 100-continue` adds one automatic
`100` after the handler's 103s. HTTP/2 framing, header byte budgets and
intermediary-generated 100s beyond one are outside it.

`SSEOpen` models status publication only. Bounds: success sets exact 200,
exact 201, exact 200+201 and `2XX` (domain `{200, 201}`); handshake headers
none, optional or required, as one Boolean; metadata statuses `{200, 201,
204}`; at most three sequential service steps; logical `end` denotes successful
completion, never Go deferred `End`; one mapped pre-open error
status. It does not model media types or `Content-Type` checks, header
values, event encoding, stream close or flush failures, `Last-Event-ID`, or
concurrency between service goroutines and the writer; those are
runtime-test obligations (the writer-ownership boundary is `ResponsePhases`).

## Implementation correspondence

These textual correspondences do not assert a new model run; the recorded
results above are unchanged.

| Model claim | Owner and regression |
| --- | --- |
| Precedence and agreement | `expr.HTTPStatusSelector` and its runtime copy `http.SelectFinal`/`CheckFinalStatus`: `TestSelectFinalPrecedence`, `TestStatusSelectorMatchesExpr`, `TestHTTPStatusRuntimeDomain`; generated decoder switch and server domain check: `TestFinalStatusSelectionRendering`, `TestFinalStatusRoundTrip`, `TestFinalStatusErrorOwnership` |
| Terminal invalid body | generated client decoder for the selected key returns its error without trying another key: `TestFinalStatusNativeDecodeConstraint` |
| Publication origin | `HandlerLifecycle.EncodeResponse`/`encodeError` route `StatusViolationError` before commitment: `TestHandlerLifecycleStatusViolationRoutesToErrorEncoder`, `TestHandlerLifecycleErrorOnlyNilResultIsViolation`, `TestFinalStatusRejectsBeforeWrite` |
| Writer owner, header isolation, version gate, budgets | `http.ExchangeWriter` with `BeginFinal` at result, error, SSE open, upgrade, file, raw and redirect starts: `TestExchangeWriterConcurrentSendDuringFinalStart`, `TestExchangeWriterRestoresHeaderMap`, `TestExchangeWriterHTTP10Refused`, `TestExchangeWriterLimit`, `TestExchangeWriterRefusesAuthored100And101`, `TestHandlerLifecycleBeginsFinalBeforeHeaderClone`, `TestHandlerLifecycleExchangeFinalEntries`, `TestSSEHandshakeClosesInformationalAdmission`; generated emitters: `TestInformationalWire`, `TestInformationalGeneratedLifecycle`; client admission: `TestObserveInformationalBudgets`, `TestObserveInformationalWireExpectContinueBudget`, `TestObserveInformationalWireRedirectCountsPerExchange`, `TestHTTPTraceComposeDiscardsNewHookError` |
| SSE handshake | `SSEStreamWriter` open path, generated `SetResponse` and client stream construction: `TestSSEHandshakeOpenTriggers`, `TestSSEHandshakeSnapshot`, `TestSSEHandshakeAdmissionRace`, `TestSSEHandshakeFailureAdmission`, `TestSSEHandshakeGenerated`, `TestGeneratedSSEHandshakeContract`, `TestGeneratedFinalSelectorAdmission`, `TestImplicitHandshakeWire`, `TestImplicitHandshakeClient`, and the HTTP ticktock fixture's `TestGeneratedSSEResponseOwnership` |

`SSEOpen`'s logical `end`/`Return` is successful completion, implemented by
`SSEStreamWriter.FinalizeSuccess` on a nil handler return and by explicit server
stream `Close` with the request context (`TestSSETerminalSuccessAdmission`). It
is not the generated handler's deferred `End`, which stays cleanup-only, nor
JSON-RPC notification suppression, which uses `Suppress`
(`TestSSESuppressionAdmission`). The redirect final start is
`BeginFinal(FinalRedirect)` before `http.Redirect`, covered by
`TestRedirectHandlerContext` and `TestExchangeWriterConcurrentSendDuringFinalStart`.

Remove `/tmp/loom-976-tlc` and other task-owned metadirs after review.


## Runtime-owner exploration

`SSETerminal.tla` complements `SSEOpen` with two contending operations sharing
one terminal/metadata owner. See [TERMINAL.md](TERMINAL.md) for the pre-#976
baseline, adopted API correspondence, return-value distinctions, finite results
and limitations. Current mode includes the already committed `ClaimFailure`;
it does not resurrect the historical missing failure-admission bug. The current
silent Close/nil Complete/successful-return traces violate the adopted
success-publication obligation. Suppression and cleanup remain separate intent.

The updated `ResponsePhases` includes redirect status 302 and Location headers.
`phases-redirect-unlocked.cfg` is an intentionally defective prospective
candidate: excluding only redirect from shared writer ownership permits a
concurrent interim header map to contaminate the redirect. The adopted
`ExchangeWriter` closes admission for redirect like every other final start.

These amendments were adopted with the implementation described above. No model
execution belongs in ordinary builds, tests, hooks or CI.
