# Basic credential validation readiness

Status: implementation released by root after independent readiness approval. Owning report: #731. Worktree:
`/Users/luca/.codex/worktrees/basic-credential-validation/loom`.
Implementation parent: `433d2e741b43b35783d293d0de60cabeca0be6a2` (safe fast-forward from readiness parent
`651372ce6ff05ed1db495369fd51d48df0c0ec75`, preserving the three readiness files).
#732 is integrated; preserve its requirement-aware bearer extraction. No new public API or authentication policy is needed.

## Contract and cause

The design can constrain Username and Password exactly as ordinary string
attributes. A syntactically valid Basic header supplies both components, including
explicit empty strings. Validate the supplied values before any authentication or
service call. Missing/invalid Basic headers remain errors when either component
is Required; otherwise they leave optional credentials absent. Required stays a
payload constraint even when another security alternative is available.

`http/codegen/basic_auth.go` currently analyzes header presence only.
`service_data_payload.go` builds RequestDecodePlan without Basic components.
`template_sources_request_decoder.go` completes ordinary element validation and
payload construction, then parses and assigns Basic credentials without any
predicate emission. JSON-RPC POST uses this same HTTP request-decoder template.
The HTTP CLI already emits the constraints in `buildBasicAuthCLIArgs`, which is
why the existing constrained-credential regression passes despite the server bug.

The correction belongs in the shared HTTP request analysis and decoder. Use
`codegen.AttributeValidationCode` with the actual payload attribute occurrence,
which delegates to effective constraints; do not copy Pattern, Enum, length,
format or named-type inheritance rules into Basic analysis. gRPC metadata already
uses this owner in `extractMetadata` and emits predicates before construction;
its behavior is a control, not a second repair.

## Proposed implementation sequence

1. Extend BasicAuthData with analyzed component validation over the extracted
   physical string locals, retaining SchemeData's resolved service selectors,
   types and pointer semantics. Use the actual payload occurrences rather than
   only UsernameType/PasswordType, so occurrence refinements are retained. The
   validator context must describe physical strings rather than optional service
   pointers; a valid Basic header supplies both locals, regardless of which
   fields are Required.
2. Make Basic participation explicit in RequestDecodePlan before finalizing
   MustValidate and error-variable requirements. Share the same Basic analysis
   with the encoder; do not introduce a second presence decision. The current
   endpoint builder computes payload before security, so either order the
   analysis dependencies correctly or complete the request plan once after
   attaching the Basic analysis. Avoid mutable follow-on patches to several
   derived booleans.
3. Render extraction and recognized-header predicates before the common
   validation exit, then perform existing typed field assignment after successful
   validation and payload initialization. Delete the old post-validation parsing
   path. If custom multipart decoding already constructs the payload, Basic
   still passes through the same credential-validation barrier before returning;
   cover both custom and generated multipart branches' error-variable needs.
4. Preserve named-string casts through NameScope, renamed field selectors,
   Basic header Required policy and optional absence. Valid supplied empty
   components are validated as empty, never treated as absent or replaced by a
   default. Existing missing-input default initialization must be inspected and
   held stable; do not expand #731 into the separate defaults/presence policy
   repair. Explicit defaults do not authorize inventing a Basic header.
5. Extend tests first, then implementation. Regenerate affected output and
   independently review the exact final diff against the integrated #732 parent.
   One atomic #731 commit, push and verified issue closure conclude delivery.

## Bounded regression and output matrix

Reuse these existing meaningful fixtures:

| Fixture | Obligation |
| --- | --- |
| `codegen/service/testdata.ConstrainedCredentialsDSL` | Nested named constraints; required/optional Basic; independent user/pass length and pattern; valid and absent controls; gRPC metadata control. |
| `http/codegen/testdata.BasicPresenceDSL` | Four independent requiredness combinations and supplied empty versus absent; named conversion. |
| `codegen/service/testdata.NamedCredentialFieldsDSL` | Custom field names and all-scheme normal control. |
| #732 credential alternative fixture | Basic/Bearer both declaration orders, valid supplied Basic constrained value, absent Basic on Bearer input, global Required. |

Extend the constrained fixture (or its transport adapter) for a JSON-RPC POST
request and one occurrence Enum/MaxLength/Format refinement plus a valid default
control. Reuse an existing multipart fixture for the common barrier's alternate
control flow; avoid a broad exported-design audit.

HTTP and JSON-RPC wire tests must assert status/protocol problem code and detail,
zero authorizer/service calls for invalid supplied values, successful calls and
decoded values for valid inputs, and nil optional fields when the header is
absent. For combined violations assert the shared validator's merged error
contract, not an invented Basic-specific precedence. gRPC control uses generated
metadata decoding and verifies the existing constraint error identity.

For each distinct changed branch compare parent/current output byte for byte,
compile generated modules and generate candidate output in a second independent
process to establish exact-byte determinism. Expected changes are request decoder
extraction/validation/error-variable ordering and necessary imports. Service
payloads, endpoint authentication, client encoders, protobuf, OpenAPI and ordinary
unconstrained behavior should remain unchanged. Record actual differences rather
than assuming these expectations. Regenerate checked-in affected goldens through
the owning generator; run scoped formatter/linter and direct/affected package
regressions. No new dependency is needed.

## Existing model and design limits

`http/codegen/tla/basic_auth/README.md` and BasicPresence already model independent
requiredness and absent/empty/nonempty components. Preserve those transitions.
The proposed correction is a deterministic missing validation step using the
existing predicate owner, not a new authentication scheduler or conditional
constraint policy. A new concurrency model is not needed for this checkpoint.
If implementation moves or duplicates callback execution, changes presence, or
introduces a distinct validation state machine, stop and model that broader
proposal before production. Tests must prove generated Go sequencing; the
presence model does not prove constraints, default initialization or callbacks.

## Exclusions and migration

#743 streaming admission, #752 scope projection, #729 mapping admission and
#728 WebSocket credential placement remain separate owners. #732 controls foreign
Authorization framing; do not loosen strict matching-Bearer errors or make
constraints conditional on the winning authentication alternative.

Applications regenerate transports to receive the fix. Requests carrying Basic
credentials that violate authored constraints become validation failures before
authentication, matching other credential locations. Applications keep their
credential verification and authorization in their existing callbacks; they
should not duplicate these DSL predicates in authentication code.

## Checkpoint evidence

The tests added at this checkpoint use the existing constrained fixture and
existing render helpers. No production source, shared skill or master tracker is
changed. Readiness gates on Go 1.27.2 darwin/arm64:

- `LOOM_DIR=<worktree> go test ./http/codegen -run
  'TestBasicConstraintsParticipateInRequestPlan|TestConstrainedCredentialsGenerated'
  -count=1`: expected failure. Both analyzed-plan cases fail; ten generated wire
  cases return HTTP 204 instead of 400, with one authorizer and one service call
  each. Required missing-header, valid values and optional absence controls pass.
  Existing CLI and bearer controls also pass. The generated module compiled.
- `LOOM_DIR=<worktree> go test ./grpc/codegen -run
  TestConstrainedCredentialsGenerated -count=1`: passed as part of the combined
  command (4.975 seconds). Twelve required/optional metadata cases cover valid,
  short user, short password, user/password patterns and absent values. Generated
  aliases compile; optional absent fields remain nil.
- `git diff --check`: passed. All changed Go files were formatted.

Transient raw evidence: `/tmp/loom-basic-readiness-http-final.log` and
`/tmp/loom-basic-readiness-tests.log`. Test temporary modules have already been
removed by testing cleanup. Durable test sources preserve the reproductions;
remove these raw logs after final validation and review no longer need them.
The JSON-RPC POST, occurrence/default, independent Required and multipart cases
are implementation acceptance obligations, not yet claimed as passed evidence.

## Implementation and acceptance evidence

Root released production after `/root/review_basic_readiness` approved readiness
without findings. The worktree advanced safely to #732's pushed parent
`433d2e74` with the three readiness files preserved.

The implementation adds effective component validators to BasicAuthData and
supplies the actual payload occurrences to AttributeValidationCode using a
physical string context. RequestDecodePlan owns the outer decoder's validation
barrier and single error accumulator. The body and ordinary-element partials no
longer declare that accumulator. The custom multipart adapter retains ownership
of its own ordinary bindings; the outer decoder validates supplied Basic
components after the adapter returns. Typed assignment follows successful
validation. Missing required Basic headers merge into that same accumulator, retaining an
earlier ordinary-field validation error as the primary error. A dedicated
query-range plus missing-header witness first failed with `missing_field`
masking `invalid_range`; it now passes. No authentication callback logic or strict
bearer parsing changed.

Expanded red cases were run before this implementation: HTTP/JSON-RPC POST
accepted invalid values through authentication and service execution. Existing
CLI default defects also appeared in full-module compilation, as detailed below.
The newly constrained #732 fixture was generated with the parent implementation:
invalid supplied Basic credentials in both OR declaration orders incorrectly
returned no decode error in HTTP and JSON-RPC. Candidate tests reject both.

Focused generated wire gates pass with the race detector:

- Existing ConstrainedCredentialsDSL: ten previously accepted invalid requests
  reject before callbacks; valid, absent, CLI and bearer controls remain valid.
- BasicCredentialConstraintsDSL: 72 HTTP and 45 JSON-RPC POST cases cover named
  aliases, renamed Go selectors, inherited/occurrence length/enum/format,
  independent Required, Unicode length, absence, explicit empty strings and
  preserved missing-header defaults. HTTP covers generated and custom multipart.
  Errors assert status, code/detail or JSON-RPC -32602/message, and zero auth and
  service calls; successful cases assert decoded field values and presence.
- Existing BasicPresenceDSL and constrained #732 alternatives pass. The gRPC
  credential/alternative tests, including the new twelve-case metadata control,
  pass without a production gRPC change.

Seven parent/current specimens (144 generated/example files per pass) were
produced using separate CLI processes. Candidate output is byte-identical across
independent runs. Each specimen changes exactly its server `encode_decode.go`:

| Specimen | Intended difference |
| --- | --- |
| BasicCredentialConstraintsDSL HTTP | Parse Basic and emit effective predicates before construction; outer accumulator covers Basic-only and both multipart branches. |
| BasicCredentialConstraintsDSL JSON-RPC | Same shared decoder behavior for POST. |
| BasicPresenceDSL | Basic extraction precedes payload construction; assignment stays presence-guarded. |
| NamedCredentialFieldsDSL | Same extraction ordering and accumulator declaration movement, retaining casts and field selectors. |
| CredentialAlternativesDSL HTTP/JSON-RPC | Constrained supplied Basic components validate before the existing alternative callbacks; bearer extraction remains requirement-aware. |
| PayloadBodyQueryPathObjectValidateDSL | Only move `err` from the body variable block to decoder entry. |

All other generated files are identical, including service payloads, endpoints,
clients, protobuf absence/presence, specs, manifests and examples. Five ordinary
specimens compile/vet in full at parent/current; the two default witnesses
compile/vet their generated service and server packages, with the unchanged CLI
limitation explicitly retained. Bounded artifacts are in
`/tmp/loom-basic-731-comparison`; manifests and `limits.json` preserve exact files
and commands until independent review finishes.

Both checked-in ticktock fixtures were copied and regenerated with local source
replacement at parent, current and a second current process. All 35 generated
files repeat exactly. HTTP output is unchanged; JSON-RPC only moves the two
ordinary decoder `err` declarations. The generated JSON-RPC output was adopted
through regeneration. Parent/current fixture modules build and vet successfully.
The CLI invocation retains the fixture's existing `-o .` command identity.

Pinned golangci-lint passes for HTTP, JSON-RPC and gRPC codegen and the explicitly
named service testdata package. The full JSON-RPC codegen package suite passed (301.615 seconds). The optional
full HTTP codegen sweep hit the default ten-minute package timeout while
`TestOptionalObjectRequestBodyGeneratedIntegration` had been running for seven
seconds; this is not a full HTTP-suite pass. No assertion failure preceded the
timeout. The interrupted test and affected decoder seam tests were rerun alone;
see final gate results below. No complete repeat of the unrelated generated-app
sweep or exported-design corpus was run.

The final ordering correction refreshed all six Basic specimens at parent and
two independent candidate processes in `/tmp/loom-basic-731-final-comparison`.
Each again changes only its server decoder; the ordinary body specimen and both
ticktock fixtures have no Basic scheme, so their existing evidence remains valid.
The final frozen tests/harnesses passed HTTP, JSON-RPC and gRPC credential gates,
and pinned lint of those three codegen packages plus service testdata reports
zero issues. `make lint-filesize lint-gofmt lint-json-v2 lint-namescope` passed;
file-size output contains existing warnings only.

## Separate pre-existing defaulted Basic CLI defect

Both HTTP and JSON-RPC CLI builders fail to compile a non-required Basic
credential carrying Default: the generated CLI local is a string, while the
initializer tries to assign a pointer to it. Parent/current output is identical
and fails at the same lines. Root explicitly kept this out of #731.

Minimal authored shape (with an ordinary HTTP POST route or JSON-RPC POST):

```go
basic := BasicAuthSecurity("basic")
Service("basic", func() {
    Method("defaulted", func() {
        Security(basic)
        Payload(func() {
            Username("login", String, func() { Default("alice") })
            Password("secret", String, func() { Default("secret") })
        })
        HTTP(func() { POST("/defaulted") })
    })
})
```

The durable BasicCredentialConstraintsDSL fixture reproduces this with named
credentials. A generated HTTP CLI reports `cannot use &basicDefaultedLogin
(value of type *string) as string value in assignment`, and likewise for the
password and JSON-RPC CLI. `buildBasicAuthCLIArgs` omits default metadata from its
physical argument analysis. This is a follow-on diagnosis, not a repair claim.
The actual server currently delivers empty value fields when no Basic header is
supplied despite authored defaults; #731 preserves that behavior. A supplied
empty component remains empty and is rejected by MinLength rather than replaced
by the default.

## Proposed shared-skill updates for root

- Framework authorization ownership: BasicAuthData renders effective predicates
  from actual credential occurrences against physical extracted strings;
  RequestDecodePlan owns outer decoder validation and accumulator declaration.
  Ordinary custom-multipart binding remains in its adapter. HTTP and JSON-RPC
  consume the same decoder, without duplicating Basic constraint semantics.
- Consumer authentication guidance: regenerate servers to enforce authored
  Username/Password constraints before authentication; valid headers supply
  empty values too, and OR alternatives do not erase payload constraints.

No formal-model claim changes: the existing Basic presence model remains scoped
to presence, and generated tests establish concrete validation/callback ordering.


## Final gate ledger

- `/tmp/loom-basic-final-focused.log`: HTTP 23.728s, JSON-RPC 10.240s;
  Basic/credentials, all request-decoder goldens, analyzed request plan and
  builder tests pass after the ordering correction.
- `/tmp/loom-basic-final-harness.log`: HTTP 9.686s, JSON-RPC 7.225s, gRPC 6.365s;
  generated Basic/constrained credential wire harnesses pass (HTTP/JSON-RPC use
  race checks). The final expanded-block harness sources were rerun:
  `/tmp/loom-basic-frozen-harness.log`, HTTP3.754s and JSON-RPC3.986s, both pass.
- `/tmp/loom-basic-final-lint.log`: pinned lint, zero issues.
- `/tmp/loom-basic-final-compare.log`: six refreshed Basic specimens, all
  independent candidate comparisons exact; actual changed-file lists recorded.
- `/tmp/loom-basic-compare.log`: the ordinary-body specimen's still-valid
  parent/current comparison and independent repeat.
- `/tmp/loom-basic-fixtures.log`: checked-in HTTP/JSON-RPC regeneration,
  compile/vet and independent repeat pass.
- `/tmp/loom-basic-post-timeout.log`: interrupted optional-object integration
  and affected decoder seam rerun passed (7.074s).

Readiness logs, comparator scripts, parent archives, generated modules and CLI
binaries are temporary evidence. Keep them through independent exact-diff review,
then remove this task's paths after checking no validation/review process needs
them. Preserve this ledger, the test/fixture sources and the separate CLI repro
above. No shared caches or managed worktrees should be removed manually.


Frozen delivery: 202 paths (181 regenerated goldens, one regenerated checked-in
JSON-RPC fixture, and 20 source/test/readiness/documentation paths), with no
commits or changes to main, shared skills or the master tracker. The minimal
plain-String CLI default reproduction also failed as predicted at
`/tmp/loom-basic-cli-default-repro.log`; its generated output is retained until
review. Apply the delivery diff against its parent rather than overwriting files
changed by subsequent work, notably shared server-codegen files and the JSON-RPC
fixture. Root owns integration, independent exact-diff review, commit and push.

## Integrated delivery evidence

The repaired 202-path candidate integrated onto `b46345a0dc9ac818b3fd5a918b0716dde3444c53`
without a source conflict or merge. Root added two shared skill updates and this
appendix. The independent worktree review confirmed the lone low formatting
finding fixed and left no unresolved findings.

All seven specimens (six Basic variants and the ordinary-body control) produced
144 artifacts identical to the independently reviewed candidate and a fresh
process-isolated repeat. Existing parent/current compile/vet receipts and both
checked-in fixture comparisons remain valid; the documented defaulted-Basic CLI
limitation is unchanged. Integrated generated Basic and alternative wire/race
checks plus direct plan/predicate cases pass: HTTP8.120s, JSON-RPC8.315s.

The integrated comparison driver uses verified pushed module version
`v1.10.0-alpha.9.0.20261012181439-651372ce6ff0` with an explicit local replace and
LOOM_DIR pointing to main. `go mod download -json` confirms origin
`651372ce6ff05ed1db495369fd51d48df0c0ec75`. This replaces the old temporary
comparison driver's placeholder pin without changing generated output.
Temporary receipts: `/tmp/loom-basic-731-integrated-comparison`,
`/tmp/loom-basic-731-integrated-wire.log`, and
`/tmp/loom_basic_integrated_compare.py`. No broad suite or full corpus was repeated.
