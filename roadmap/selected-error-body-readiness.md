# Built-in error response representation readiness (#797)

Parent: `66a9dc9090a06cd4136be1e5ebdaf26883ff90fa`.
Implementation worktree: `/Users/luca/.codex/worktrees/problem-projection-integration/loom`,
parent `25179e87b8b5d4e9cf5bb233d1818600469d060d`. The original readiness
worktree `/Users/luca/.codex/worktrees/selected-error-body/loom` is preserved.
Status: implementation and bounded validation complete; independent final review pending.

1. Reproduce the committed `Body("detail")` compile failure and inventory selected,
   omitted and mapped built-in problem fields. Keep custom service errors as controls.
2. Identify the owner of conversion from `ServiceError` to the public problem
   representation. Inspect body/header/cookie analysis, constructor arguments,
   safe-message handling, formatter overrides and client reconstruction.
3. Establish whether selection currently happens before representation conversion
   or assumes a complete body after fields have been mapped elsewhere. Model the
   required conversion/selection ordering if multiple paths share this disagreement.
4. Propose one response projection plan for complete, selected, pruned and absent
   problem bodies; dependent server/client adapters must derive field sources and
   native representations from that owner. Do not add a special case per field.
5. Independently review readiness and bounded acceptance before production: direct
   structural tests, real generated HTTP client/server compile and wire/error tests,
   no raw service-text disclosure, custom-error and ordinary full-problem controls,
   parent/candidate/repeated generation comparisons for each changed behavior.
6. Implement the accepted owner, remove superseded assumptions, validate, obtain
   fresh exact-diff review and deliver one atomic commit for #797.

## Initial evidence

`errorBuilder.buildResultInitCode` takes a built-in error shortcut before looking
at `origin:attribute`. `buildProblemClientResultTransformCode` then assumes every
nonempty body has Code, Detail, Instance and RetryHint members. Server body code
similarly constructs a complete problem object, while the generic no-constructor
response path selects the authored problem field directly from `ServiceError`,
whose physical fields and safe-message representation differ. These are candidate
representation-owner defects; tests must establish each claimed path.

## Readiness evidence

The initial ten-module matrix ran in 8.854s. The expanded thirteen-module
matrix explicitly used `LOOM_DIR` for this worktree and ran in 20.845s. The full built-in problem and a custom error
with selected `detail` compile. All seven built-in selected fields (`type`,
`title`, `status`, `detail`, `instance`, `code`, `retry_hint`) fail client and/or
server compilation. Mapping `code` to a header while leaving a pruned object body
also fails: the client reads nonexistent body.Code and the server writes an
unknown Code literal field. Log: `/tmp/loom-selected-error-body-matrix.log`.
The additional empty-body control compiles. Explicit detail-only object bodies
and code-to-cookie projections fail for the same full-body assumptions. The
source-pinned expanded evidence is `/tmp/loom-selected-error-body-local-matrix.log`;
its three controls pass and ten projected shapes fail. The earlier matrix used
the shared source-mode default for the generated runtime dependency, so use this
explicit-local run as the authoritative parent evidence.
The exact original detail reproduction is `/tmp/loom-selected-error-body-red.log`.

The bounded model checks all subsets of three representative problem members.
Legacy and server-only fixes fail at 10 and 18 states respectively; complete
source/body projection correspondence passes 24. Its README states abstraction,
implementation seams and exclusions. These results identify a shared public
problem representation owner, rather than seven field-specific corrections.

## Proposed owner and implementation boundary

Keep the HTTP problem adapter package-private in response analysis. It describes
the native `http.ProblemResponse` representation once, distinct from the service
`ServiceError` and from each prepared transport body. Reuse existing generic
field/type transforms, native presence rules and name allocation for projections.
Do not make a second validation, wire-location or error-disclosure policy.

On the server, materialize the existing `http.NewProblemResponse` once before
selecting built-in error body/header/cookie fields. All those projections read
that public representation. Complete and pruned object constructors and scalar
selections must consume this same source descriptor. Delete the handwritten
complete-body literal and direct ServiceError field-selection shortcut. Preserve
custom-error transformations and the existing explicit formatter override as an
external body boundary; it receives the whole service error, not one scalar.

On the client, derive body, header and cookie member inputs from the evaluated
response bindings and actual prepared carrier. Assemble the public problem
representation through the shared transform/init machinery, then call the
existing `ProblemErrorFromBody` conversion. HTTP status retains its existing role
in error classification. Allocate local identifiers in the constructor's scope;
do not hardcode `code`, `detail`, `instance` and `retryHint` over bound arguments.
The current `hasBody` switch is insufficient: replace it with resolved field
sources, without re-inferring authored wire mappings.

Keep the runtime safe-message and retry-hint helpers authoritative. A service
error's raw message is not a valid substitute for the public detail. Selected
scalar and absent bodies keep the media rules derived from their actual body;
complete/pruned problem objects retain their problem result identity. Explicit
ContentType and custom formatters retain their declared boundaries.

## Bounded implementation acceptance

1. Direct analysis verifies one normalized source shared by built-in body/header/
   cookie projections and precise client reconstruction for complete, pruned,
   scalar and absent bodies. Exercise string, int and optional retry hint, plus
   mapped argument name collisions and custom errors as controls.
2. Reuse the thirteen-module matrix for compiler coverage and extend it with
   optional hint absence/presence runtime controls.
   Group compatible method specimens into a generated runtime app where practical.
3. Generated actual HTTP server/client checks assert status, Content-Type, selected
   scalar/object wire shape, mapped headers/cookies, reconstructed error code/id/
   detail/retry hint, no raw service text, and full-error formatter invocation.
   Include explicitly safe and unsafe error messages and type/title overrides.
4. Compare a bounded full-problem/custom-error normal control and each distinct
   selected/pruned/absent behavior against this parent. Generate the candidate
   twice in independent processes; compare exact bytes and build/vet all outputs.
   List intended differences and validate rendered OpenAPI JSON/YAML and pinned
   Vacuum for changed contract specimens. Reuse prior unchanged evidence.
5. Run affected HTTP codegen/runtime and custom error projection/security checks,
   the bounded model and selected lint. Obtain a fresh exact-diff review after
   all intended changes are frozen. No full exported-design sweep is required.

The readiness gate was independently approved before implementation; its receipt is recorded below.

## Adoption inspection

The adjacent Loom-MCP registry design uses built-in ErrorResult errors over gRPC,
so this HTTP projection correction does not require changing its design. Existing
HTTP testdata includes mapped error fields and the public Body DSL explicitly
admits scalar selection and shaped objects. The repair belongs to HTTP response
representation analysis, not an application-specific error adapter.

## Implementation and acceptance evidence

Readiness was independently approved on seven-file manifest
`ed599d7d7f82cb3b46638a5112ee603af31ed2b6b0cb664b1130caabb1d20d0f`.
Implementation lives in the package-private public-representation adapter:
`problemRepresentation` defines the runtime shape; `responseBodySource` selects
it before prepared-body projection. Existing generic transforms and field
initialization derive server/client mappings. The handwritten complete-body
constructor and client shortcut are deleted. One public ProblemResponse is
materialized before all built-in body/header/cookie reads; the existing formatter
receives the whole error for object and scalar bodies. Generated local `problem`
is reserved by the existing import/local-name owner.

The selected matrix now has sixteen modules: the original thirteen plus full,
type and title metadata overrides. Every built-in shape tests mounted HTTP and
imported generated clients across absent/present retry guidance, safe-message
selection and custom formatter use. It asserts status, media type, selected body,
explicit/automatic headers, cookie and reconstructed code/detail/instance/remedy.
The custom selected-error control and existing custom-media/disclosure cases pass.
Complete-body, projected scalar and explicit-shaped body media retain their
existing contracts; scalar/shape bodies are application/json, not problem+json.

Final focused commands (explicit current worktree LOOM_DIR):

```sh
go test ./http/codegen -run '^(TestSelectedProblemBodyConstruction|TestProblemResponseUsesNativeRepresentation|TestEncodeError|TestClientTypes|TestClientTypeFiles|TestServerTypes|TestErrorBodyProjectionEncoders)$' -count=1
go test ./http/codegen -run '^(TestTransportLocalNameReservationsCoverGeneratedLocals|TestEncodeError|TestClientTypes|TestClientTypeFiles|TestServerTypes|TestErrorBodyProjectionEncoders|TestGeneratedErrorDisclosure|TestErrorMediaGenerated|TestInheritedAPIErrorGeneratedModule)$' -count=1
$(go env GOPATH)/bin/golangci-lint run ./http/codegen ./http --timeout=5m
```

They pass: 24.516s / 10.062s; pinned lint reports zero issues. Logs are
`/tmp/loom-797-extraction-checks.log`, `/tmp/loom-selected-error-body-focused.log`
and `/tmp/loom-797-lint.log`. A whole HTTP codegen run at the initial implementation
source took 512.814s and found only intended golden differences and the missing
`problem` reservation. Those failures are corrected and their affected gates
rerun above. It is not claimed as a whole-suite pass; no exhaustive corpus audit
was run. A maintainability warning was resolved by extracting the cohesive source
selection owner; no generated output changed after that extraction.

`/tmp/loom-797-comparison/report.json` records three bounded designs, 48 artifacts
per parent/candidate/repeat, and exact independent-repeat equality. Durable DSL
inputs are beside the model as `*_design.go.txt`; all use a pushed framework pin
and explicit local replacement. Parent selected/pruned shapes fail compilation;
all candidate modules build/vet. Full/custom controls compile at both sources.
The initial comparison setup failed before generation because its temporary
module needed resolution; it produced no acceptance evidence. The recorded final
run uses explicit module resolution. At generation time main's generator source
matched `25179e87b`; pending Pulse-only edits did not enter the generator graph.

Intended differences are bounded to nine files: projected service client/types,
server/types and encoder use public representation and field membership; server
and aggregate CLI alias the service named `problem` against the new local.
Full problem client/types, server/types and encoder use the same shared conversion.
Custom selected scalar error encoder now honors the whole-error formatter.
All JSON/YAML OpenAPI bytes are unchanged. A post-extraction generation of every
candidate confirms all 48 hashes unchanged (`post-extraction.json`). These exact
six specs pass libopenapi and pinned Vacuum rules in 9.325s; the temporary
`TestSelectedProblemSpecs` overlay and `/tmp/loom-797-spec-validation.log` preserve
that result. No schema-policy change is claimed.

Both maintained SSE fixtures were regenerated from their real designs using the
local source and the original `-o .` invocation. Eighteen HTTP and seventeen
JSON-RPC generated files were compared against fresh parent generation. Only the
HTTP client/types, server/types and server encoder differ due to this repair;
JSON-RPC output is unchanged. Both parent and candidate fixture apps pass their
Go tests. The checked-in HTTP refresh also incorporates the already-delivered
#709 validator signatures and the current design fingerprint; these are explicit
baseline regeneration differences, not new wire or semantic changes. Temporary
CLI module-resolution/tidy edits to fixture go.mod/go.sum were restored; no
module dependency change is included. Receipt:
`/tmp/loom-797-fixture-comparison.json` and fixture generation/test logs.

The model was rerun after implementation with unchanged inputs: legacy fails
Renderable at 10 states, source-only at 18, checked passes 24 states. This proves
only the stated bounded representation/membership property. Go typing, codecs,
public disclosure and formatter behavior are established by the concrete tests.
Final independent review must approve this exact scope before delivery.

Malformed response checks additionally pass for all fifteen built-in modules:
truncated JSON and wrong JSON kinds return the client's decoding error; object
bodies missing required members return its validation error before reconstruction.
The initial test incorrectly excluded ServiceError anywhere in the error chain;
validation errors legitimately wrap one. The corrected assertion checks the outer
ClientError category. Final run: 22.160s, /tmp/loom-797-malformed-wire.log.

## Integration review receipt

Independent final review approved all forty files on parent `25179e87b` with no
findings (manifest `48204f13f27c831297db54b359d9b785521977be8d34235e403c5431618ca242`).
The reviewer independently reran the generated wire/malformed matrix and direct
owner/reservation/projection checks in 25.960s. Integration onto `30cabf633`
retains all production and generated files byte-for-byte; the three shared skill
and model-index guides merge the already-delivered reader cursor documentation.
The implementation worktree remains frozen. This integrated receipt and guide
merge require a scoped exact-diff review before the atomic commit.
