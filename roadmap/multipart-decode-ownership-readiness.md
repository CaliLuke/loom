# Custom multipart request decoding ownership (#762)

Status: readiness independently approved; implementation in progress.
Parent: `febb41bce2d2b80b36f73c67316f6c7d2cc9d226`.
Worktree: `/Users/luca/.codex/worktrees/multipart-decode-ownership/loom`.

1. Reproduce custom multipart compilation with path, query, header and cookie
   bindings, retaining unsupported body shapes and no-binding/generated controls.
2. Compare the ordinary and custom multipart request-element decoders, including
   error accumulation, return signatures, conversion and security framing.
3. Establish one element decode implementation and one shared analysis of its
   accumulator/barrier needs. Each enclosing decoder owns its signature and
   accumulator scope; custom callbacks continue to own body decoding.
4. Obtain independent readiness assessment before implementation. Preserve
   callback/validation order, safe decode errors and service invocation barriers.
   Add a bounded model if the selected design changes protocol ordering.
5. Run direct structural tests, generated build/vet, callback and actual HTTP
   adversarial cases, plus bounded parent/candidate byte comparison and separate
   process generation. Keep existing normal-body and generated multipart controls.
6. Freeze the exact final diff, independently review, push one atomic repair and
   reconcile #762 only from proven witnesses. Root owns shared guides and cleanup.

## Initial source evidence

`RequestDecodePlan` owns the outer request decoder's accumulator and validation
barrier. For custom multipart requests it deliberately excludes ordinary bound
fields, because the adapter validates them. The adapter uses a separate 440-line
request-elements template and 280-line conversion partial set. Its declaration
rule requires `MustValidate && !ServerBody`, although custom multipart always has
a body. Uses and final validation therefore refer to an undeclared `err`.

The copies have also drifted from current shared element semantics: ordinary
Authorization framing and custom-type decoding live in the ordinary partials,
while the multipart copy retains older branches. A one-line declaration repair
would preserve those two semantic owners.

Candidate to assess: remove multipart-specific element/conversion copies and
use the shared request-element decoder. Retain element-accumulator requirements
separately from the outer body/Basic accumulator requirements in the same decode
plan. Both wrappers declare their own variable and barrier from that owner.
Parameterize only the enclosing error-return signature (payload plus error
versus error only), not mapping/validation policy. Preserve callback errors and
opaque-body responsibility; do not redesign multipart parsing or resource limits.

## Readiness evidence and proposed implementation

At the recorded parent, five direct accumulator regressions and five full-module
compile regressions fail; the no-binding module compiles. The cookie diagnostic
also reports an unused cookie variable and an undefined `payload` in an error-only
function. The 3.379s run is `/tmp/loom-multipart-762-red.log`.

The ordinary request-element template has precisely three enclosing-function
returns: malformed query, malformed Authorization framing and unexpected cookie
error. Supply their return arity at `serverEncodeDecodeSections`, whose two
`transDecoderTmplFuncs` calls know whether the wrapper returns payload plus error
or only error. This is syntax adaptation, not a second decoding policy. The
custom adapter joins the existing `requestDecoderPartials`; delete its duplicated
440-line element body and 280-line conversion collection.

`RequestDecodePlan` will retain element validation needs independently from its
outer `MustValidate`/`NeedsErrorVar`. Ordinary wrappers derive their barrier from
those element facts plus body and Basic needs; custom adapters consume the same
element facts locally. The outer custom path retains only body callback error
handling and Basic validation. Preserve callback-before-elements ordering and
existing safe error conversion. No public semantic decision is unsettled.

The bounded model reproduces legacy declaration failure (65 states), rejects a
declaration-only fix because of the cookie return signature (73), and passes the
shared candidate (152). See `http/codegen/tla/multipart_decode/README.md` for
assumptions, run instructions and implementation correspondence.

Acceptance remains bounded: six custom binding fixtures plus existing generated
multipart and ordinary query/cookie controls; add one combined auth fixture and
one custom scalar control for known differences between the copied partials.
Prove callback failure, malformed multipart/query, missing/conversion/constraint
errors, invalid/valid Bearer framing and Basic validation block or permit the
service correctly. Compare parent/current artifacts and independent-process
output, compile/vet selected generated modules, and record all intended changes.
Do not absorb unrelated multipart parsing, file presence or resource-policy work.

Independent readiness assessment by `/root/review_effective_repair` confirmed
the five direct reds and shared owner. One low style finding (inline empty DSL
callback) was mechanically fixed and re-reviewed; no unresolved findings remain.

## Implemented owner and final acceptance

The shared request-element and conversion partials now serve ordinary and custom
multipart decoders. Removed both multipart copies (over 700 lines). The plan's
`ElementsMustValidate` fact retains element-local accumulator/barrier ownership
independently from the outer body/Basic requirements. Section construction supplies
the three shared error returns with each wrapper's signature. No mapping, framing,
conversion or validation decision is duplicated by that syntax adapter.

Nine custom fixtures now generate, compile, vet and run actual HTTP handler tests
under race detection. They cover all four locations, the unconstrained path,
no-binding, Basic, Bearer and a custom text scalar. Tests assert status, problem
code/detail, callback and service invocation counts and successful payload values.
Malformed multipart/query, callback-private and designed errors, missing fields,
conversion/constraint failures and invalid auth framing/constraints all preserve
the documented barriers. The first harness run exposed test assumptions (204
bodyless success, JSON v2 explicit field tags and generic validation detail); those
expectations were corrected against the existing public wire contract, without
changing production. Final selected direct/generated/control tests pass 28.365s;
all ordinary decoder goldens pass 0.798s. Pinned golangci-lint reports zero issues.

Twelve designs / 192 artifacts were generated with the real CLI at parent,
candidate and in another independent candidate process. All twelve candidate
modules build/vet. All candidate repeats match byte for byte. Only eight custom
server `encode_decode.go` files differ: local accumulator declaration, shared
current conversion/framing and correct cookie error return. No-binding, generated
multipart and ordinary query/cookie controls are parent-identical; the eight
corrected modules fail compilation at parent. No client, CLI, schema, manifest or
checked-in golden changed. Evidence and exact hashes remain in
`/tmp/loom-multipart-762-generated/report.json` through independent review.

The bounded model result is unchanged by implementation: legacy and declaration-
only counterexamples, corrected 152-state pass. The model abstracts conversion
failures and does not prove parser behavior, resource limits or arbitrary custom
callback behavior. The callback still owns body decoding and must construct the
payload on success. #762 alone is the delivery scope. Final review remains required.

## Integrated review boundary

Fresh reviewer `/root/review_multipart_final` approved the isolated 15-file
manifest `6832261e78228567e1e471500f059613eecd33ee3acc65313de8759a8277b79d`
without findings. It independently passed focused generated/race tests (24.271s),
optional cookie, array path and map query compile/vet probes (3.904s), checked all
576 physical artifact hashes and reran the 152-state model.

Root copied the source byte for byte onto `d0fefa59e`, with no overlapping files.
Public HTTP guidance, framework ownership and the model index now describe the
corrected adapter. Integrated custom multipart runtime/race, ordinary decoder
and gating checks pass 27.151s. The final exact diff remains subject to independent
integration review before commit.
