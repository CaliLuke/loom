# JSON-RPC response body value reference

Worktree: `/Users/luca/.codex/worktrees/jsonrpc-response-value-reference/loom`.
Parent: pushed `39c33f48bc7f029cefbddf38791601d949e5e6aa`.
Issue: [#840](https://github.com/CaliLuke/loom/issues/840).
Status: implementation and bounded acceptance complete; frozen for final review.
Selected-view #834 remains
frozen with its complete intrinsic acceptance matrix.

## Contract and owner

A decoded body local must have the physical value reference already selected by
`http/codegen.TypeData.ValueRef`. `VarName` identifies its concrete declaration.
For nullable bodies those differ. `bodyValueRef` owns that distinction; the
JSON-RPC shared `writeSingleResponseDecode` must consume it for success and error
paths. HTTP response templates and WebSocket receive generation already do so.
No DSL policy, wrapper construction, validator or conversion policy changes.

A pushed-parent CLI reproduction with ordinary nullable `Type`, without views,
generates but fails compilation because the local concrete body cannot be passed
to nullable validation and conversion. This is independent of #834's selected-view
carrier implementation. Inputs and exact commands are preserved in the issue.

## Sequential delivery

1. Preserve the parent witness and #834 candidate. Search for duplicate reports;
   obtain independent editorial review before filing the self-contained issue.
2. Add direct source/value-reference correspondence tests across object, scalar,
   array and map bodies, both ordinary and nullable. Cover the shared error writer.
   Add a generated client compilation/runtime fixture before production edits.
3. Freeze readiness evidence for independent review. This is a single immutable
   metadata-consumer selection; existing declaration-identity models establish
   the distinction. No additional protocol state or ownership operation is added,
   so direct correspondence tests are more useful than a new state model.
4. Consume ValueRef in the shared writer. Adjust manually constructed direct-test
   metadata to satisfy the existing ValueRef contract; ordinary golden bytes must
   remain identical. Do not add an empty-ValueRef fallback.
5. Compare parent/current/process-isolated repeat outputs of the nullable and
   ordinary fixtures exactly. Compile/vet and exercise null, valid, invalid,
   missing/malformed response input, and ordinary mapped error decoding. Preserve
   validation errors, response status handling, and ordinary output. Run affected
   decoder/golden tests, then freeze the final exact diff for independent review.
6. Root integrates the approved atomic ticket; resume #834 only after delivery.

## Central consumer inventory

- `http/codegen/service_data_element_types.go`: TypeData separates declaration
  `VarName`, reference `Ref`, and semantic body-value `ValueRef`.
- `http/codegen/service_data_body_types.go`: bodyValueRef uses the shared Go body
  reference owner for nullable bodies and declaration name otherwise.
- `http/codegen/template_sources_response_decoder.go`: HTTP ClientBody.ValueRef.
- `http/codegen/stream_sections_websocket_recv.go`: WebSocket ClientBody.ValueRef.
- `jsonrpc/codegen/decoder_sections.go`: one wrong VarName local in the shared
  writer; success and error decoding both call it. Named-error dispatch and
  conversion helpers need no independent representation selection.

## Bounded acceptance

Actual DSL fixtures cover nullable and ordinary named object/string/array/map
results. Direct tests assert existing metadata and rendered local agree, then
cover error writer forwarding. Generated runtime acceptance uses the public
client decoder against protocol envelopes and inspects concrete/null state and
validation/decode errors. Existing response decoder goldens and mapped-error
controls establish unchanged ordinary behavior. #834's full intrinsic projected
fixture is a downstream integration check after this prerequisite is delivered.

## Readiness evidence

The direct matrix has two expected failures: actual nullable object response
local, and nullable metadata passed through the shared error writer. Eight other
checks pass: four ordinary shapes, three nullable scalar/array/map shapes whose
VarName already equals ValueRef, and ordinary error metadata. Do not infer that
all nullable shapes have different declaration and value names.

The ordinary generated module compiles/vets and passes 28 runtime cases (four
shapes × valid/null/invalid/absent/malformed/mapped-error/bad-status). The nullable
module generates but vet fails exactly at object body validation and conversion,
matching the independent CLI witness. The nullable runtime cases remain acceptance
obligations after the representation fix; they are not claimed as parent passes.

Evidence: `/tmp/loom-rpc-response-parent-direct.log`,
`/tmp/loom-rpc-response-parent-generated.log`, and the repeated parent generation
under `/tmp/loom-rpc-response-comparison/parent`. The helper overlay retains output
without changing generation behavior. Both comparison modules require pushed
`v1.10.0-alpha.9.0.20261013051529-2a8337700ffd` and replace Loom with this exact
worktree. The independent CLI witness uses that remote pin without replacement.
No production changes or new model claims are included in readiness.

## Candidate and final acceptance

Readiness was independently approved by root after source/evidence verification
and reproduction of the two intended direct failures. The implementation changes
only the shared decoder's local declaration from VarName to ValueRef. Three
fabricated bodies in the existing error-switch test now provide the required
ValueRef metadata; their golden output stays unchanged. Framework guidance records
that success and error decoding consume the shared owner. Applications need to
regenerate their clients; authored nullable contracts and ordinary decoding are
unchanged. Existing public Nullable documentation already describes this behavior.

Both ordinary and nullable generated modules compile/vet and pass all 56 runtime
cases. The final harness additionally checks error category and service/method
identity for invalid values, missing/malformed envelopes and bad status, plus the
concrete mapped error type/message. Direct regression and existing selected decoder
goldens pass (`/tmp/loom-840-candidate-direct.log`, 0.522s). Affected decoder,
return/header/tag/error writer checks pass (`/tmp/loom-840-affected.log`, 2.980s).
Final generated runtime passes (`/tmp/loom-840-candidate-runtime.log`, 4.758s).

Parent/current/process-isolated repeat comparisons cover ten generated artifacts
per specimen. All ten ordinary artifacts are unchanged. The nullable specimen has
exactly one changed line in `gen/jsonrpc/probe/client/encode_decode.go`: the object
body local becomes `loom.Nullable[ObjectResponseBody]`. Its other nine artifacts
are unchanged. All twenty current/repeat artifacts match exactly. The repeat
process only generates; compile/vet evidence applies to those identical bytes.
The strengthened runtime assertions do not change generation inputs or output.
Comparison receipt: `/tmp/loom-840-comparison.json`; complete intended output delta:
`/tmp/loom-840-generated.patch`. Gofmt and diff checks pass.

Status: implementation and bounded acceptance complete; frozen for independent
final review. No commit or integration is authorized by this plan. #834 source
remains separately frozen until this prerequisite is delivered.
