# Response selection validation (#976)

This record covers status-range, fallback, multi-success, error-only and
informational responses, the `StatusField` response source, the shared
exchange writer, SSE handshake selection and their importer, OpenAPI and
runtime-contract projections. The acceptance rule is one selection rule shared
by expression validation, server publication, client decoding, import and
export: exact keys outrank classes, classes outrank the fallback, declaration
order never matters, a published status must lie in its key's domain before
any header is written, and the client never retries another key. The accepted
[design record](response_selection_design.md) keeps the rationale, rejected
alternatives and proof obligations.

## Status

Implemented on parent `cadba28e0e551401dd94015822f6b9d467e6ffb3` (2026-11-07). The
qualified checkpoints below are retained history. The final local qualification
records the implementation source; exact review and normal delivery gates remain
separate from publication.

## Public contract

| Contract | Normative documentation |
| --- | --- |
| Selectors, precedence, `StatusField`, multi-success, error-only | [Status-selected Responses](../../../../docs/http-guide.md#status-selected-responses), [Response Selection](../../../../docs/dsl-reference.md#response-selection) |
| Informational emitters, observers, budgets, exact 100 observation-only | [Informational Responses](../../../../docs/http-guide.md#informational-responses) |
| SSE handshake modes, `SetResponse`, `Close` as successful completion, client media check | [SSE Handshake Status and Completion](../../../../docs/http-guide.md#sse-handshake-status-and-completion) |
| Runtime contract `Selector`/`Domain` | [Response Contract Checks](../../../../docs/http-guide.md#response-contract-checks) |
| Import classification and SSE `itemSchema` inverse | [Code generation import](../../../../docs/code-generation.md#import-openapi-loom-import-openapi), [import coverage](../../../../docs/openapi-import-coverage.md), [SSE response import](../../../../internal/openapiimport/sse_inverse.md) |

The derived `x-loom-async` handshake `statuses` list (canonical selector keys
in ascending byte order; WebSocket `["101"]`) is owned by
`asyncHandshakeStatuses` in `http/codegen/openapi/internal/ir`. Its reserved
import contract is owned by [SSE response import](../../../../internal/openapiimport/sse_inverse.md).

## Owners and durable checks

| Contract | Owner | Durable checks |
| --- | --- | --- |
| Selector parsing, precedence, runtime domain | `expr.HTTPStatusSelector`; `http.SelectFinal`, `CheckFinalStatus`, `SelectInformational` | `TestSelectFinalPrecedence`, `TestHTTPStatusRuntimeDomain`, `TestHTTPStatusSelectorAdmission`, `TestStatusSelectorMatchesExpr` |
| `StatusField` binding, view survival, branch requiredness | response-owned binding plan in `expr` | `TestStatusFieldWireBinding`, `TestCapturedInformationalDomains`, generated `TestSelectedFinalPreparation` |
| Generated decoding and server domain check | HTTP codegen final-status templates | `TestFinalStatusSelectionRendering`, `TestFinalStatusRoundTrip`, `TestFinalStatusNativeDecodeConstraint`, `TestFinalStatusRejectsBeforeWrite` |
| Violation routing before commitment | `HandlerLifecycle.EncodeResponse`/`encodeError` | `TestHandlerLifecycleStatusViolationRoutesToErrorEncoder`, `TestHandlerLifecycleErrorOnlyNilResultIsViolation` |
| Interim/final writer ownership, every final start including redirect | `http.ExchangeWriter`, `BeginFinal(FinalKind)` | `TestExchangeWriterConcurrentSendDuringFinalStart`, `TestExchangeWriterRefusesAuthored100And101`, `TestExchangeWriterLimit`, `TestRedirectHandlerContext` |
| Client observation, per-exchange budgets, golang/go#76669 workaround | `http.ObserveInformational` | `TestObserveInformationalBudgets`, `TestObserveInformationalWireRedirectCountsPerExchange`, `TestHTTPTraceComposeDiscardsNewHookError` |
| Exact 100: declared, imported, exported, observed; never emitted | `expr` observation domain; shared outgoing literal owner | `TestInformationalContinueAdmission`, `TestDeclaredContinueCannotEmit`, `TestDeclaredContinueNativeWire`, `TestImportContinueDeclaration`, `TestInformationalContinueArtifacts` |
| SSE handshake publication, success completion, suppression | `SSEStreamWriter` terminal owner | `TestSSEHandshakeOpenTriggers`, `TestSSETerminalSuccessAdmission`, `TestSSESuppressionAdmission`, `TestSSEHandshakeGenerated`, `TestGeneratedSSEResponseOwnership` |
| Nullable native status source captured once | nullable response projection | `TestNullableStatusCompositionGenerated` |
| Import classification and rejections | `internal/openapiimport` status selector planner | `TestImportStatusSelectorAnalyze`, `TestImportStatusSelectorRender`, `TestImportStatusSelectorRejections`, `TestImportSSEInverseRequired`, `TestImportSSEReservedContract` |
| OpenAPI keys and informational documentation | `http/codegen/openapi/internal/ir` | `TestOpenAPIResponseSelectorKeys`, `TestInformationalResponsesAreDocumented` |

The client abort path's [golang/go#76669](https://github.com/golang/go/issues/76669)
workaround can be removed when the pinned Go release propagates the newer
composed hook's error and the maintained
`TestHTTPTraceComposeDiscardsNewHookError` reproduction confirms the correction.
A failing test alone does not establish that removal condition.

## Design models

Bounded TLA+ and Lean models are opt-in design aids, never gates. Their recorded
results and Go correspondence are kept with the models:

- [status selection, phases and SSE handshakes](tla/status_selection/README.md),
  with [runtime owner](tla/status_selection/RUNTIME-OWNER.md),
  [terminal](tla/status_selection/TERMINAL.md) and
  [mixed response](tla/status_selection/MIXED_RESPONSE.md) notes;
- [selector proofs](lean/status_selection/README.md);
- [status bindings](../../../../expr/tla/http_status_bindings/README.md);
- [informational client and declarations](../../../tla/informational_client/README.md);
- [SSE import admission](../../../../internal/openapiimport/tla/sse_import/README.md).

## Qualified checkpoints

Each entry is retained evidence for the named source inventory. None was rerun
for this record. Commands pin Go 1.27.3, `GOWORK=off`, physical `LOOM_DIR` and
`GOMAXPROCS=1` with `GOFLAGS=-p=1` unless stated.

1. **Final-parent composition**, tree `a392cb1c819e3330a2b5af04f096ada2fbec251b`
   (8,544 providers, inventory `8c79a089bc995caf1426f027765aec9f3b0777335ac456dda9f34431bc628ccb`).
   The nullable `StatusField` compile red was reproduced, corrected through
   the existing nullable projection, and its generated regression passed.
   Service/HTTP/JSON-RPC compilation, 105 direct test nodes and scoped lint
   passed.
2. **Exact 100 correction**, tree `b034f0ddd36e09ea14ddc185df27f2471a189c16`
   (8,552 providers, inventory `7986756b43c1c97dcf4198467f4eb9dd5a0dd53a001581896289f6d833d97c84`).
   Actual red against the old owners, 25 green nodes, 95 focused and 8 final
   direct passes; two generated informational modules compiled and ran.
   Independent generation matched all 42 Go files; rendered JSON/YAML were
   byte-identical and passed libopenapi 0.42.4, pinned Vacuum 0.39.3 and strict
   re-import. Package lint, fixture lint and vet passed (`GOMAXPROCS=2`). The `Declarations`
   TLA+ extension checked 3,456 distinct states. Independent source-stage
   review approved with one layout-only harness override.
3. **Generated runtime renewal**, pre-M9 inventory
   `cedae54f3ec3b1a217d751f117150634959a23ffd3276bd2927509ed741cfd8c`.
   `TestSSEHandshakeGenerated`, `TestNullableStatusCompositionGenerated` and
   mapped `TestGeneratedErrorDisclosure` ran with retained modules. In run A,
   the nullable and RPC modules passed. The typed module failed two stale
   `TestSelectedFinalPreparation` absence rows, which were fixed by an
   independently reviewed test-only oracle correction. The corrected typed
   module test and vet then passed, and run B passed on the corrected oracle.
   The A and B runs matched all 117 retained Go files: typed 64, nullable 14,
   RPC 39.
4. **Existing-design snapshot**, inventory
   `eb7e152663f189a78aa89496b1133a77f8d6526d434be2152c12a22abb5bd7df`
   (8,552 providers). Of 1,104 retained sections compared to the accepted
   baseline, 979 are exact and 125 are classified intended changes, with none
   added or removed. This is prior-prefix comparison history, not final
   determinism.
5. **Changed saved OpenAPI specs**, same inventory. All eight passed the
   maintained libopenapi 0.42.4 helper and unchanged Vacuum rules at the
   approved correctness pin `v0.39.4-0.20261107034229-b6c4e203d2ea`.
6. **CLI importer rerenders**, held M9 inventory
   `fa7f31f712caed3ba9f6c399511ed5db343ef4129fca381e14476e6443491159`.
   One CLI build reproduced all six accepted importer outputs byte for byte.

Earlier milestone evidence, recorded model runs and reviews predate these
inventories and remain historical. Unchanged providers keep their original
attribution. Changed providers require the renewal listed under
[Before the commit](#before-the-commit).

## Delivery binding

### Before the commit

The reviewed M9 registrations, normal five-process determinism check and three
checked-in fixture renewals are complete, as recorded below. The final source
inherits #1005 parent `cadba28e0e551401dd94015822f6b9d467e6ffb3`,
which descends from #957 `3d386572c2c6a5b41596f121327e129d67b2704d`
and the reviewed main prerequisites.
Final exact-parent review authorizes one atomic commit; the normal commit hook
runs on that unchanged diff. Documentation and exact generated-postimage
composition do not invalidate the recorded runtime or model checks.

### Publication

The normal push gates then run on the corrected batch, followed by the CI
release. The #957 and #1005 publication gates, including Vacuum #1056, belong
to this step. Publication results are not recorded in this file.

## Design and plan

The execution plan was retired from `roadmap/`. The accepted design moved here
as the [design record](response_selection_design.md). It has a historical
banner and is otherwise unchanged. Model notes cite it for rationale and
placement.

## Final local qualification

The normal `TestGeneratedFilesAreByteStableAcrossIndependentProcesses` passed
across all 34 registered designs and five independent child processes, 29.347s
wall time. It compared both raw sections and rendered bytes on held source tree
`104e75e30d746ce5c6f5600cbd89938e46d59efd` (8,559 providers, inventory
`1cba32036f8dd3a95eac61b35d01614be5bca5f032e5cb3aa6fe51a2b5bc705f`).
The full source inventory remained unchanged. This was the normal test, without
helper/snapshot mode. The three new maintained fixtures cover the four planned
shapes: range and multiple exact unary success, informational responses, and
multiple exact typed handshake responses.

The earlier 31-design comparison retains its own source binding: 1,104 raw and
rendered sections, 979 exact and 125 classified intended changes, with no added
or removed positions. Eight changed saved specifications passed libopenapi and
the unchanged pinned Vacuum rules. Six actual CLI imports matched the accepted
M1 output hashes. These checks were not relabeled as new executions on the later
prefix; the later corrections affect transport lowering and documentation, and
are independently reviewed and qualified. Complete corrected-source determinism
is established by the 34-design run.

Normal regeneration of HTTP ticktock, JSON-RPC ticktock and JSON-RPC mixedtick
produced 55 byte-identical file pairs across independent processes. All three
modules compiled and passed `go vet`; HTTP and JSON-RPC ticktock executed their
root package tests, while mixedtick has no tests and supplies compilation
coverage here. Exactly five checked-in generated files changed: one HTTP file
updates three `Close` comments to describe successful completion, and four RPC
server/stream files consume the inherited resolved `ServiceError` capability.
The other 50 generated files, including every emitted specification, are exact.
These postimages were generated normally; no generated source was hand-edited.

The final nullable-status interaction module separately proves status-only
formatted errors: valid native 400/401 permits replacement of an absent selected
body; null, absent or invalid native status fails before formatter invocation or
publication. Incoming named nullable error construction preserves its checked
payload and validates status constraints. Its genuine generated compile failure
was corrected at the shared constructed-status owner; actual module tests/vet
pass and all 14 raw output/harness pairs match independent generation. Existing
success exact/class, mapped header/cookie and incoming constraint controls remain.

These are completed local acceptance checks, not a claim of publication. Final
review binds the exact source/documentation/fixture successor and intended
parent; commit hooks and normal push remain separate delivery steps. Formal
models remain opt-in design evidence, with no new proof execution required for
this documentation or exact generated-postimage composition.
