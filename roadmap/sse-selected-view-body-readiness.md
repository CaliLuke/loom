# SSE selected-view body ownership (#727)

Base: `110d195214dc355c99e599f83114fb135f0e363f`.
Status: implementation and bounded acceptance complete; final integrated diff
awaiting independent review. Prerequisite #818 is delivered as `d172c4620`; the
full CLI fixture gate passes with that correction.

## Contract and owner

A successful send projects and validates one selected view, then converts that
same projection into that view's HTTP wire body. `SetView` affects subsequent
sends; the empty selection means default. Fixed method views and discriminator
projections retain their existing precedence. Unknown views and invalid values
must fail before any SSE event is written.

The defect is in HTTP response conversion: `SSEData.ResponseBody` retains only
`ServerBody[0]`, erasing the view-to-body mapping already prepared by HTTP type
analysis. The SSE renderer consequently serializes the default body even after
shared service projection selected another view. Unary HTTP and JSON-RPC use
the viewed result's `View`; WebSocket has its own switch over stream state.

Implemented owner: retain the eligible response's complete body family in SSE
analysis. One package-private HTTP streaming body-selection renderer consumes
that family and the validated viewed result, handles fixed and dynamic views,
and supplies a typed body to the transport payload assignment. SSE ordinary,
discriminator, and WebSocket consumers derive body selection from this owner;
remove the first-body-only and duplicated switch paths. Typed mapped-data
extraction must occur before assigning to `any`. Mixed-result streams must not
convert streaming events through unrelated unary response bodies. Primitive
and unconverted collection values keep their direct-encoding path.

This does not change the accepted view contract, framing, stream lifecycle,
client wire format, or concurrency guarantees. The model assumes sequential
service sends and SetView calls. A validated per-send view is the authority;
body conversion must not resample mutable stream selection.

## Sequence

- [x] Read issue, framework guidance, source owners, and existing fixtures.
- [x] Add a direct rendered regression and generated raw-wire/client regression.
- [x] Reproduce the current defect on the exact base with tests only.
- [x] Model first-body selection and candidate selected-view ownership; record
      old counterexample and candidate invariants.
- [x] Freeze readiness files, hashes, repro evidence and receive independent review.
- [x] Implement the shared HTTP streaming body owner and delete superseded paths.
- [x] Run focused regressions and affected packages; compare bounded generated
      output at parent/current and a process-isolated repeat, compile and vet.
- [ ] Freeze exact final diff for fresh independent review; root commits/pushes.
- [ ] Preserve compact evidence and remove unused temporary artifacts.

## Bounded acceptance

New viewed-SSE fixture: default → full → empty/default → full, exact raw frame
fields and generated client values; full-view collections; typed mapped raw-data
extraction in each selected view; fixed full; unknown view and nil value
fail before a frame. Existing discriminator fixture covers its precedence and
view-specific bodies. Existing SSE result-type and mixed-result fixtures cover
direct encoding, mapped data controls, pre-stream errors and unchanged lifecycle.
The existing viewed-WebSocket fixture covers dynamic/fixed object and collection
body conversion. Unary HTTP and JSON-RPC viewed output are unchanged controls.

Before final review select only these representative designs for parent/current
byte comparison and compile; document every intended output difference. Preserve
normal primitive, mapped, mixed-result, and fixed-view output byte for byte when
unaffected. No exported-design corpus audit is part of this ticket.

## Readiness evidence

At readiness, production files were unchanged. On the exact base with only the new tests:

- `LOOM_DIR="$PWD" go test ./http/codegen -run '^TestSSEViewedResult' -count=1`
  fails in dynamic object/collection/mapped rendered selection. Generated code
  builds and vets, then full-view raw frames and client values lose secret on
  sends 2 and 4; full-view collection elements also lose secret. Fixed full,
  mapped raw data, default omission, and unknown/nil pre-stream controls pass.
- Existing discriminator codegen/body/generated-module, mixed-result,
  optional-data codegen, absent-constructor, and pre/post-stream error renderer
  controls pass together (2.401s). These remain behavior controls after the fix.
- Pinned golangci-lint 2.14.0 over `./http/codegen/...` and its testdata package
  reports zero issues. Go is 1.27.2 on darwin/arm64.
- TLC first-body configuration exits 12 with the expected omission, 103/102
  generated/distinct states. Selected-body configuration exits 0, 1190/301
  states. See the owning model README for commands, pins and limitations.

Final validation reuses the expanded viewed-SSE design plus
`SSEVariantProjectionDSL`, `SSEResultTypesDSL`, existing optional-data and mixed
fixtures, and `webSocketViewedResultDSL`. Select normal unary and JSON-RPC viewed
controls without expanding their corpora. Compare only the affected generated
files and these representative modules, compile/vet each changed module, and
repeat generation in an independent process. Intended changes are dynamic SSE
body dispatch, discriminator/WS dispatch deriving from the validated view, and
any mechanically necessary typed body assignment; unchanged controls must
remain byte-identical. Regeneration is required for existing dynamic viewed SSE
services to receive the correction.

## Implementation findings and final acceptance

Independent readiness approval covered the seven-file manifest
`22c02f354257fef0b972aba14a3378165b604aebe269dfbedede80e53419c186`.
The implementation exposed mapped fields omitted by a view: the parent optional
String control compiles and emits no data, while the initial selected-body
candidate failed compilation by referencing the absent Go member. The extended
parent fixture also reproduces a nil dereference when its default-body
constructor reaches an omitted required member. The repair uses finalized
selected-body membership plus the documented codec omission contract; no
fallback to the default body, re-projection, or new DSL rejection remains.
Root approved this completion within the same owner and accepted contract.

The final model adds tiny-view and mapped-data states. Original first-body and
missing-member controls exit 12 (277/276 and 265/264 generated/distinct states).
The final candidate exits 0 (3976/923 states). The model README records
implementation correspondence and limitations.

Bounded output comparison used nine designs: new viewed SSE; discriminator
SSEVariantProjectionDSL; SSEResultTypesDSL; SSEOptionalDataDSL; MixedResultsDSL;
webSocketViewedResultDSL; HTTP ResultBodyMultipleViewsDSL; JSON-RPC
ViewedStreamResultDSL; and jsonrpcViewedUnaryResultDSL. All 207 artifacts match
an independent-process repeat. Exactly seven files change versus the parent:

- Viewed SSE server `sse.go`: dynamic object/collection dispatch and mapped
  field omission; fixed Full method stays identical.
- Discriminator SSE server `sse.go`: dispatch on validated `res.View`, with
  typed body assignment and the complete prepared view family.
- Five dynamic WebSocket server files: `switch s.view` becomes `switch res.View`.
  The fixed-view WebSocket service is unchanged.

All other output is byte-identical, including unary HTTP, JSON-RPC, primitive,
unconverted collection, optional mapped data, mixed-result and fixed-view
controls. All 18 parent/current modules build and vet (10.801s/10.623s); repeat
output reuses that compilation evidence. Six checked-in streaming expected-code
sections receive only the same switch-expression change.

Final focused HTTP tests pass (19.211s), including dynamic/fixed/mapped/discriminator
SSE server and client behavior, selected-body goldens, primitive and optional
codecs, viewed WebSockets, and the viewed-union test interrupted by the package
run's aggregate timeout. JSON-RPC viewed-stream/unary controls pass (10.615s).
HTTP checked-in SSE fixture regeneration, exact tree equality, build, external
client and pre-stream rejection pass (15.472s). JSON-RPC POST SSE fixture
regeneration/tree equality/build and external protocol-error event compatibility
pass (5.538s). The raw GET-listener branch and lifecycle code are unchanged.
Pinned affected golangci-lint 2.14.0 reports zero issues.

The broad package attempt is not green: eight unrelated HTTP test names fail
identically at the exact parent (mapped-name/nullable assertions, optional union
compile, map-query scope, duplicate nullable validation, local names and reused
union validation), and three JSON-RPC test names reproduce parent `undefined:
problem` errors. Focused parent reproductions confirm these independently.
The HTTP package also reaches its aggregate ten-minute deadline, after the
then-running viewed-union case had run four seconds; that case passes in the
final focused run. No unrelated test or production changes are included.

Temporary parent/current/repeat modules and concise comparison data live under
`/tmp/loom-727-comparison`; exact final hashes and log references are frozen in
the handoff manifest. Remove them only after validation and independent review,
retaining this durable evidence and a compact cleanup receipt. Root owns final
review, atomic commit, push, ticket closure, and managed-worktree cleanup.

### Separate CLI blocker

The exact new exported design was also checked with
`LOOM_DIR="$PWD" make test-testdata-compile TESTDATA_RUN='TestDesigns/http/codegen/SSEViewedResultDSL$'`
(one design, not a corpus audit). Its mapped defaulted String exposes an
independent pre-existing annotation-preparation defect: `loom gen` rejects
`Default("fallback")` because its actual SSE codec is raw text, while required
JSON-schema annotation preparation demands the built-in JSON codec. Direct
rendered Go compiles/vets/runs, but complete CLI generation reaches this later
example preparation boundary. The exact parent reproduction and logs are
retained separately. No default, fixture, assertion, or expectation was removed
to hide the defect. Root is tracking the shared annotation owner with the
custom text cookie-default CI failure before integrating this delivery.

## Integration receipt

Root independently reviewed the frozen 21-file candidate and reran selected-view
SSE, WebSocket and bounded model checks. No unresolved findings remain.
The full new fixture gen/example/tidy/build/vet pipeline passed with prerequisite
#818, now delivered on main in `d172c4620`. Integrated guidance records selected
body ownership and the existing codec behavior for omitted mapped fields.

Comparison modules now require the verified pushed parent
`v1.10.0-alpha.9.0.20261013005718-110d195214dc` while retaining their explicit
local source replacements. The pin correction changes no source or generated
artifacts; all module tidies pass. Existing generation/build evidence is reused.
Receipt: `/tmp/loom-727-comparison/pin-receipt.json`.
