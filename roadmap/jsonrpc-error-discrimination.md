# JSON-RPC typed error identity (#774, duplicate #775)

Status: implementation complete on `2537b97ac`; frozen final review pending.

## Contract and observed failures

`docs/error-handling.md` (JSON-RPC Error Codes) promises that an explicit
JSON-RPC response transmits its designed error body and that generated clients
validate that body and reconstruct the service error. It requires a required
`ErrorName` when two names share one custom type. `dsl/error.go` identifies the
marked field as the routing identity; `expr.ErrorExpr.Validate` enforces string
and required membership for that field. An ordinary field named `name` is not a
substitute. Supported scalar errors have generated nominal identity methods and
no discriminator member. The contract does not require all same-code alternatives
to have a common member or different JSON kinds.

The parent unary decoding groups by numeric code and always reads generic
`ErrorData.Name`. Actual generated round trips fail for `kind`, mapped
`kind:error_kind`, selected root `Body("kind")`, and explicitly selected object
bodies. A normal literal `name` and unmapped generic-error control pass. SSE drops
typed data into a formatted error; WebSocket generated clients return the raw
runtime error. Four generated client probes reproduce the two named alternatives
on each streaming transport. These are shared missing error-data identity and
decoding semantics, not three unrelated spelling bugs.

Actual generated server controls emit `"same"` versus `7` for same-code string
and integer alternatives, and identical `"same"` for two unconstrained string
alternatives. The new generated client matrix also checks two string alternatives
with disjoint `Enum("alpha")` and `Enum("beta")`: these remain distinguishable
without a new field. All four distinguishable cases fail reconstruction on the parent.
Identical wire representations cannot uniquely identify a declared error.

## Owning correction

1. Derive one effective error-data identity record from the finalized response
   body and existing binding provenance. Record the error name, numeric code,
   actual body and effective `ErrorName` selector when retained by the body.
   A selector may be the body root or an authored member; derive member spelling
   through `expr.JSONFieldName`, including mappings and full tag overrides.
   Do not rediscover the field from the Go name or hard-code `name`.
2. Project that record through the existing HTTP transport response IR and
   `ErrorData`/`ErrorGroupData`. Keep method-local alternative identity and order;
   cached file/import reanalysis must consume the same logical records. Existing
   server body projection remains authoritative and must agree with the record.
3. Emit one private method error decoder shared by unary, SSE and WebSocket
   adapters. Unknown/unmapped codes retain `*jsonrpc.RawErrorResponse`. Known
   codes evaluate effective selectors plus the existing typed body decoding and
   validation rules. Require exactly one successful alternative, with no
   declaration-order preference or default discriminator materialization.
4. Keep trial bodies, header/cookie bindings and constructor results private until
   uniqueness is established. Give each trial detached response/body state;
   never publish a partial result or mutate the caller's destination. Raw
   selectors check membership and string identity before decoder defaults.
   Malformed JSON, invalid designed bodies, unknown identity and ambiguous
   identity must produce clear errors, not a guessed service error. Preserve
   custom decoder errors; framework-owned state can be transactional, whereas
   application decoder callbacks remain application-owned and cannot be rolled
   back by Loom. Do not promise external callback side-effect rollback.
5. Unary passes its response metadata and configured decoder. SSE response,
   message and error event branches pass raw envelopes through that same decoder;
   WebSocket adapters translate runtime `RawErrorResponse` from both call and
   receive paths through it. Preserve existing stream cancellation, terminal
   markers, locks and close semantics. Runtime protocol routing remains neutral.

Trial failure precedence is explicit. Parse and check raw JSON before any trial.
A raw selector mismatch excludes the alternative before invoking a decoder.
For an eligible alternative, the generated validator's ordinary rejection is a
contract mismatch. Decoder errors are mismatches only when they satisfy the
existing `pkg.DecodeJSONCandidate` structured JSON conversion contract:
`json.SemanticError` with no underlying error, or the supported numeric/base64
conversion cause. Reuse/extract that classifier at a private shared internal
boundary, preserving existing untagged behavior; do not export a parent-package
helper solely for siblings. Every other decoder error is unexpected. Any
unexpected error from an eligible alternative wins even if another candidate
succeeded, and retains its cause through wrapping/joining. With no unexpected
error, accept exactly one success; zero successes retain useful decoding and
validation causes, while multiple successes report ambiguity. Do not infer
mismatch from error strings, discard a later failure, or publish the first result.

Classification is by the error contract, not proof of application callback intent:
a custom decoder may deliberately return the same structured conversion error.
Its ordinary external side effects cannot be undone. Each candidate gets fresh
framework-owned storage and response/body state; shared input metadata is not
mutated, and the selected result is returned only after all eligible trials finish.

`ValuePlan.JSONShape`/`MatchUntaggedJSON` own matching only for the admitted
untagged-union grammar, which excludes maps and opaque codecs. Do not apply that
admission to error bodies or build a second schema inference engine. Existing
normal body decoders/validation remain the matching contract for these errors.
The existing `pkg/tla/untagged_json` model supplies the bounded exactly-one,
order-independent and private-result reasoning. Its schema/decoder separation,
opaque-codec exclusion and transactional destination assumptions are limits,
not a proof that arbitrary error codecs have no external effects. No new graph,
protocol state or concurrency model is needed for this deterministic selection.

## Accepted and rejected wire examples

| Design/wire case | Result |
| --- | --- |
| Unique mapped code with valid designed body | Reconstruct its declared error |
| Shared type, `kind:error_kind`, unrelated `name` | Select from `error_kind`; preserve ordinary `name` |
| Different alternatives with different declared discriminator keys | Evaluate each selector; require one match |
| `Body("kind")`, wire `"forbidden"` | Select by the root string |
| Selected object body retaining `kind` | Select using that body's effective member name |
| Same code, string versus integer | Reconstruct the one valid body |
| Same code, disjoint string enum constraints | Reconstruct the one valid body |
| Same code, two unconstrained strings, wire `"same"` | Clear ambiguous-response error; never first declaration wins |
| Known code, absent/null/wrong/unknown required discriminator | Invalid response; no default or unrelated `name` fallback |
| Malformed body JSON | Decoding error, distinguishable from ambiguous identity |
| Unknown/unmapped code, generic or protocol data | Preserve raw error envelope |

No new design restriction, wire wrapper or compatibility mode is proposed.
Authors who need otherwise identical same-code bodies to round trip must expose
identity in their designed body or assign distinct codes. Required-field mapping
coverage (#780), service type deduplication and error metadata propagation are
separate owners. The scalar same-name/different-type dedup residual from initial
exploration is preserved separately and is not counted as a #774 failure.

## Consumer inventory and bounded acceptance

- Expr: effective full/selected body, original error identity, mapped/tagged
  selector and required membership. Direct tests cover root/member/omitted
  selectors and source nonmutation.
- HTTP transport IR and shared analysis: body/selector correspondence, same-code
  groups and import reanalysis; ordinary HTTP errors remain unchanged.
- Unary client: existing typed/raw round-trip harness plus renamed/mapped fields,
  unrelated `name` (including another declared error name), selected root/member,
  distinct scalar kinds and disjoint
  same-kind constraints. Reverse candidate order and verify the same result.
- SSE client: response/message/error branches, actual endpoint-returned and
  `SendError` mapped errors, raw protocol errors, invalid/ambiguous data and close
  behavior. Reuse existing stream-error and generated module seams.
- WebSocket client: call/request-reply and receive/server/bidi paths; actual
  mapped server error plus malformed/missing/unknown identity and raw controls.
  Existing success and stream lifecycle controls remain unchanged.
- Application decoder and custom-tag boundary: no shared target publication,
  custom sentinel errors before/after a successful candidate in both declaration
  orders, selector-ineligible callbacks never invoked, structured conversion
  mismatch controls and retained zero-match causes; exact
  full-tag wire key and missing required selector before defaults.
- Generate a bounded combined design covering the above shape classes at the
  recorded parent, candidate and independent repeat; compare all artifacts and
  compile/vet each distinct candidate module. Expected output differences are
  the shared error decoder and calls from existing adapters. Reuse current
  HTTP/JSON-RPC ticktock and JSON-RPC mixedtick fixture captures; regenerate any
  affected fixture through the CLI, never edit `gen/`.
- Run direct affected tests and selected mounted race checks, plus scoped lint.
  No full exported-design corpus or new generator harness.

## Sequence

1. [x] Reproduce renamed/mapped, selected, scalar and SSE/WS failures; retain
   literal-name, raw generic and actual wire-shape controls.
2. [x] Establish the public contract and common owner without new restrictions.
3. [x] Freeze source/evidence packet and obtain independent readiness review.
4. [x] Implement the shared owner and remove the three divergent client paths.
5. [ ] Compare bounded outputs, run affected gates, update public/consumer and
   framework guidance, then freeze the exact final diff for independent review.
6. [ ] Deliver one atomic commit; preserve concise receipts and clean inactive
   generated/model/build output after review and delivery.

## Readiness evidence

- `/tmp/loom-774-readiness-current.log`: direct key failures, selected root/member
  failures for both alternatives, six scalar identity/error-detail failures,
  four streaming client failures, and the passing literal-name/raw control.
- `/tmp/loom-774-readiness-helper-adoption.log`: after extracting the unchanged
  existing service fixture DSL, original typed/raw control and new literal-name
  control pass; renamed and mapped variants retain their expected failures.
- `/tmp/loom-774-parent-wire-shapes.log`: actual server wire discriminability
  versus indistinguishability, separate from client behavior.
- `/tmp/loom-774-readiness-lint.log`: scoped configured lint. Embedded test
  programs are gofmt-formatted; generated modules use the explicit local source.
- `/tmp/loom-774-readiness-manifest.json`: exact source, parent owner, inputs and
  evidence hashes, with retained generated artifact ledger. Earlier diagnostic
  logs remain evidence of exploration, not additional acceptance claims.

## Final implementation and bounded evidence

`HTTPErrorExpr.JSONRPCErrorSelector` derives the detached effective selector.
Transport IR and `ErrorData` carry it without renderer reconstruction. Endpoint
analysis records the private method decoder identity and constructor reservations
consume it. Unary, SSE and WebSocket paths call that decoder. The shared runtime
`DecodeErrorResponse` owns eligible private trials and exactly-one selection;
`internal/jsondecode.IsMismatch` extracts the existing union conversion classifier
without changing union admission or behavior. The old generic-name switch and
stream formatted/raw-only branches are removed.

- Nine generated modules cover four full-body spellings (literal, renamed,
  mapped and full-tag/default), selected root/member, scalar/enum/ambiguous
  bodies, and SSE/WS services. Their 115 artifacts repeat exactly in independent
  processes. Twelve changed artifacts contain only the method error decoder or
  adapter calls; declarations, server encoders and normal body validation remain
  byte-identical. Every candidate module passes vet and race tests.
- The three current fixture applications contain 55 generated artifacts. HTTP
  ticktock is unchanged; JSON-RPC ticktock and mixedtick each change only client
  `encode_decode.go` and `stream.go`. All repeat exactly; parent and candidate
  applications build and vet. Checked-in changes are copied from CLI output.
- Actual generated tests cover typed/raw unary round trips, missing/null/wrong/
  unknown identity before configured callbacks and defaults, ordinary `name`
  preservation, scalar/enum discrimination, ambiguity, and unexpected custom
  decoder failures before or after a successful trial. Mounted stream services
  cover endpoint-returned errors, `SendError`, WS call and queued receive paths.
  Existing SSE result/view controls assert structured raw protocol errors and
  retain their successful-result, close and view checks.
- Runtime tests cover different selector keys, reversed alternative order,
  malformed data, excluded callbacks, private body/header state, retained
  zero-match causes and opaque application failures. Expr checks selector
  provenance and nonmutation; existing untagged matching tests check extraction
  of the mismatch classifier.
- `/tmp/loom-774-generated-comparison.json`, `loom-774-artifact-ledger.json`,
  `loom-774-generated-output.diff`, `loom-774-generated-gates.json` and
  `loom-774-fixture-comparison.json` record exact output and gate results.
  `/tmp/loom-774-final-manifest.json` records source, inputs and evidence hashes.

The parent comparison uses the exact archived `2537b97ac` production source,
with only the test inputs overlaid. Its module-source resolver is adapted in the
scratch overlay to select that archive explicitly because an archive has no Git
metadata. The first scratch runs that ignored `/tmp` symlink canonicalization or
required Git metadata are invalid setup attempts, not behavioral evidence.
The scalar nominal identity assertion was corrected from nonexistent `ErrorName`
to the actual `LoomErrorName` before the final parent run. Corrected parent reds
remain; the original literal-name control passes. No new schema restrictions,
error mapping coverage repair (#780), or service-symbol deduplication change is
included. Runtime selector errors are checked centrally; the existing model's
narrower grammar and callback assumptions remain explicit above.

## Final review corrections

The independent review found three medium defects, corrected in the shared
owners without adapter exceptions:

- A selected body from another source attribute ends outer `ErrorName` lookup;
  a same-spelled ordinary member inside that selected object is not the outer
  discriminator. The direct unchanged reviewer probe and a generated client with
  `Body("details")` check this boundary. The latter uses distinct codes, a valid
  wire peer and the authored response header for the outer identity. It passes
  on the original parent, fails the pre-repair candidate, and passes the repair.
  The attempted full server round trip exposed separate missing mapped-error
  header emission; its failing log and inputs are preserved, not counted as a
  successful round trip or repaired under #774.
- Trial responses clone `TransferEncoding` as well as headers/trailers and the
  request. Immutable/external connection and context state is outside this
  ownership boundary; no general callback-state rollback is promised.
- A trial-body close failure is recorded as unexpected independently of decode
  mismatch classification. Joining it into a validation mismatch would hide its
  cause when another alternative succeeds. Direct tests check both candidate
  orders, and the unchanged reviewer probe passes.

Only the affected owner/probe and selected-body module checks rerun. Existing
output/fixture evidence remains valid: the added selected-object specimen adds
11 artifacts per stage; all prior generated hashes are unchanged. The final
ledger contains 510 hashes across parent, candidate and independent repeat.
`/tmp/loom-774-before-r1-r3/` retains the pre-repair exact source and packet;
`/tmp/loom-774-r1-r3.delta.patch` records the narrow correction for re-review.
