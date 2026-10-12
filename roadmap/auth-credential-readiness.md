# Authentication credential readiness

Status: independently reviewed and accepted by root on 2026-10-12.
Root releases #732 test/model-first implementation from `9880865a`; record the
assigned isolated worktree and full parent before source edits.
Implementation worktree: `/Users/luca/.codex/worktrees/auth-credential-alternatives/loom`.
Parent: `9880865a1c2873cf455fbae979dc7de3a533071f`. The other
reconciled authentication reports remain separate, unreleased follow-ons. All six reports below
were still open when read through `gh issue view`.

## Slices and order

| Owner | Cause confirmed in current source | Scope |
| --- | --- | --- |
| #732 | Transport extraction flattens OR requirements; a bearer decoder rejects Basic framing before endpoint alternatives run. | Restore requirement-aware credential extraction, starting with optional Basic/Bearer and separately mapped API-key alternatives. Preserve explicit payload Required. |
| #731 | Basic fields are assigned after general request validation, without emitting their attribute predicates. | Feed Basic components through the existing validation owner; separate commit, using #732's extraction plan if delivered first. |
| #743 | gRPC stream-envelope decode calls `Recv` before endpoint authentication, although metadata already holds credentials. | Separate stream admission owner; higher resource-exhaustion priority, but not a simpler decoder reorder. Prepare in parallel when capacity permits. |
| #729 | Implicit mappings put distinct simultaneously required credentials in the same Authorization location, without representability admission. | Separate mapping-admission commit. Reject impossible AND bindings; do not reject documented distinguishable Basic/Bearer OR sharing. |
| #752 | Every callback receives requirement-wide scopes, including scopes absent from that scheme. | Separate per-scheme scope projection in expr/shared analysis and OpenAPI; current expr already rejects scopes absent from every scheme. |
| #728 | JSON-RPC streaming normalization changes the payload before credential mapping; unsupported locations reach finalization/codegen. | Separate transport-placement admission. Do not silently invent WebSocket handshake or message authentication semantics. |

#732 is the next bounded extraction repair. #731 is not another instance of
bearer parsing: it bypasses validation even on a Basic-only method. Do not combine
these reports merely because they mention credentials. #743 should not wait for
all other auth work, but must have its own ordering model and admission design.

## Existing contracts and accepted examples

`dsl.Security` and `docs/dsl-reference.md` specify AND within one call, OR between
calls, and context isolation between failed alternatives. `Security()` is an
anonymous alternative. `Required` describes the payload, not a conditional
security branch. The Basic presence documentation explicitly requires a valid
header whenever either component is Required. Root confirmed preserving this
contract for #732; there is no pending Required-policy question.

| Design/request | Required behavior |
| --- | --- |
| Optional user/password/token; Basic OR JWT; valid Basic header | Decode Basic components; leave token absent; let existing OR callbacks authenticate. No bearer-framing decode failure. |
| Same design; valid Bearer header | Decode token; leave optional Basic fields absent; preserve callback order and context isolation. |
| Optional bearer OR explicitly mapped API key; valid key and unrelated `Authorization: Basic ...` | Foreign scheme must not make bearer extraction abort the request. API-key alternative remains usable. |
| Basic OR JWT plus `Required("user")`; Bearer-only request | Continue rejecting missing required Basic header. OR does not erase the authored payload constraint. Do not reject the design wholesale. |
| JWT-only method; Basic or raw token in Authorization | Preserve strict invalid-bearer decode error. Do not globally loosen `security.DecodeBearer`. |
| Any method; validly framed credential violates an authored constraint | Preserve validation and do not pass that invalid value to authentication. General payload constraints are not silently conditionalized. |
| AND credentials at distinct locations | Every scheme must pass; successful callback contexts chain within that requirement. |
| Failed first OR callback returns a modified context | Later alternative starts with the original context; only the winner reaches the service. |

Missing optional credentials currently reach callbacks as empty strings; a
transport-owned session cookie intentionally uses that callback convention.
Preserve it in this bounded fix. Do not equate an empty callback string with an
invalid parser result, or introduce blanket callback skipping that breaks cookie
authentication. Application authentication decisions remain callback-owned.

## Error classes and remaining policy boundary

Keep three cases explicit: absent input, a syntactically recognized foreign
Authorization scheme, and malformed input claiming the expected scheme.
`DecodeBearer` is a strict single-credential parser: raw input, empty Bearer and
whitespace within a token fail. Scheme classification belongs to the plan that
knows alternatives; it must not change this parser's success predicate.

For #732's first slice, only foreign-scheme framing with an available declared
alternative changes from a global decode error to absent optional bearer input.
Basic-only/JWT-only rejection stays unchanged. Add explicit controls for malformed
`Bearer`, `Bearer ` and `Bearer a b`, including an otherwise valid independent
API key. Preserve the current malformed-matching-credential rejection unless a
broader policy is accepted. The docs do not explicitly decide whether malformed
optional credentials should invalidate the request or only their OR candidate.
The latter requires carrying per-candidate failures to authentication; it is not
authorized by treating parser failure as absence. Record it as a separate policy
boundary rather than expanding this slice silently.

## Owning seams

- `codegen/service/security_projection.go` retains `RequirementsData`; the
  `ExpandRequirementSchemes` / `PartitionSchemesByIn` path in
  `security_partition.go` flattens it for transports. Preserve alternative
  membership in a shared analyzed extraction plan rather than reconstructing it
  separately in HTTP, gRPC and JSON-RPC templates.
- `http/codegen/service_data_analysis.go` assigns a single `HeaderData.Bearer`
  boolean from flattened header schemes. `template_sources_request_decoder_elements.go`
  then returns immediately when `DecodeBearer` fails. JSON-RPC uses the same HTTP
  request decoder; verify POST transport separately, without extending WebSocket
  security as part of #732.
- `grpc/codegen/codec_metadata.go` has the analogous eager return. Basic gRPC
  credentials have their own metadata fields; use API-key versus bearer cases
  where protocol framing actually applies, not an invented HTTP Basic header.
- `codegen/service/endpoint_method_section.go` already owns OR/AND callback and
  context sequencing. Keep that behavior, including direct endpoints and
  authorization-wrapped endpoints. No per-transport authentication policy fork.
- For #731, `http/codegen/basic_auth.go` knows presence but not constraint
  predicates; `template_sources_request_decoder.go` assigns `user` and `pass`
  after the normal `MustValidate` exit. Use the existing effective attribute
  validator and named-string conversion rules. gRPC's current constrained
  credential test compiles output but does not exercise invalid wire values.

## Test/model-first sequence for #732

1. Add a reusable DSL fixture alongside existing credential fixtures in
   `codegen/service/testdata`. Include optional Basic/JWT OR, bearer/API-key OR
   with distinct mappings, an AND control, a singleton JWT control, anonymous OR
   and explicit Required controls. Assert analyzed requirement membership before
   rendering. Reproduce valid Basic returning decode-payload before changing code.
2. Add a bounded TLA+ model beside the shared security analysis. Model absent,
   foreign, valid and malformed framing separately; OR/AND topology; required
   payload constraints; callback success/failure and modified contexts. The old
   global-parse-failure transition must fail optional-alternative usability.
   Corrected model must retain singleton strictness, AND failure, required-field
   rejection and candidate-context isolation. State that it abstracts parsers,
   callback implementation and generated Go.
3. Implement the smallest shared extraction-plan correction supported by these
   invariants. Derive HTTP/JSON-RPC and gRPC behavior from that owner, retaining
   the strict parser. Delete superseded scheme-independent decisions for changed
   paths. Avoid a catch-all parser-error suppression branch.
4. Generated wire tests must assert decoded pointer presence, exact error
   classification, callback invocation/order/context and service invocation.
   Cover both OR declaration orders, case-insensitive scheme names, absent and
   empty values, malformed matching input, explicit Required and named strings.
   Reuse `renderHTTPModule`, `renderGRPCResponseContractModule` and
   `renderJSONRPCModule`; use real transport requests for changed framing paths.
5. Compare base/candidate output for the new OR fixture in all three transports,
   existing HTTP `PayloadJWTAuthorizationHeaderDSL`, named credential fixtures
   and one session bearer/cookie fixture. Reuse Basic presence fixtures as the
   Required/pointer control. Each distinct behavior needs one representative,
   not every exported design. Render accepted inputs in isolated processes,
   compile generated modules, list every intended byte difference, and repeat
   candidate generation in another process for exact-byte determinism.
6. Run direct security and shared-service tests plus bounded transport tests:
   `go test ./security ./codegen/service -count=1`, then affected credential,
   Basic-presence, session-security and new alternative tests in the three
   codegen packages. Set local source mode/`LOOM_DIR` to the implementing
   worktree. Run affected formatting/lint checks. Existing goldens and checked-in
   fixtures only change through regeneration. Independent exact-diff review,
   one #732 commit, push, and ticket-state verification conclude the slice.

## Independent follow-on readiness obligations

#743 must admit metadata-authenticated streams before their first frame while
preserving complete-payload authorization and the existing intentional endpoint
rechecks. Earlier no-repeat and mixed-alternative eager-success proposals were
hypotheses, not accepted contracts. `docs/authorization.md` explicitly permits
repeated authentication; avoid adding a success cache or changing public endpoint
representation solely to eliminate such repeats. The current design investigation
limits eager admission to methods whose complete alternatives are metadata
resolvable, preserving declaration order for mixed message/metadata alternatives.
Shared authentication analysis must own the check, preserve original cancellation
and isolate failed OR contexts. Capture transport metadata independently of any
authentication callback's replacement context. Test missing/wrong credentials
without a frame, delayed valid frames, cancellation, EOF, malformed first envelopes
and message-carried credentials. Do not invoke services or interceptors before
complete payload validation. Concrete API/model readiness remains separately gated.

#729 requires a single effective location owner before validation; current
`finalizeRequirementScheme` resolves implicit defaults after validation. Header
identity is case-insensitive. The validator must consider simultaneous AND use,
same credential reused by multiple schemes, and distinguishable OR framing.
Raw API-key/Bearer overlap and client selection when both OR inputs are supplied
need explicit representability rules; a method-wide duplicate-name check is
wrong for Basic/Bearer OR.

#752's current expr rule admits a scope if any scheme declares it; project each
scheme's required subset from that established contract. OpenAPI OAuth2
requirements currently copy the entire requirement scope list, so fix that
consumer with the shared owner. #728 can reject an unsupported placement before
mutation, but adding a supported WebSocket credential transport needs its own
public contract and lifecycle design.

This investigation read current source and live reports. It does not claim
generated transport validation or final implementation acceptance.

Bounded outside-repository section probe on main
`9880865a1c2873cf455fbae979dc7de3a533071f`, Go 1.27.2:

- Basic-only `Password("pass", String, func() { MinLength(12) })`, both
  components Required: the assertion that server decoder sections contain
  `InvalidLengthError` fails. This reproduces the missing predicate, not a wire
  test or complete #731 acceptance matrix.
- Optional Basic/JWT OR with only username Required: shared method data retains
  two requirements, Basic `HeaderRequired` is true, and the rendered decoder
  contains the unconditional invalid-bearer return. These structural controls
  pass; they do not independently execute #732's wire reproduction.
- JWT declaring `read` AND a key declaring no scopes: rendered endpoint source
  confirms the requirement-wide scope is passed to the API-key callback.

The temporary probe used `RunHTTPDSL`, `CreateHTTPServices`,
`ServerEncodeDecodeFile` and `codegen.SectionCode` on non-header sections; the
scope case used `EndpointFile`. It did not modify production files or add tests
to main. Its temporary directory was removed after root accepted readiness; the
#732 reproduction is now retained by the new generated transport fixture.


## Implementation handoff (#732 only)

The implementing worktree is based on the full parent recorded above. No commits
or pushes were made by the implementer. Root owns integration, shared guidance,
independent exact-diff review and ticket closure. Other auth slices remain open.

`internal/securitygen.PlanBearer` now retains whether a requirement omits the
same payload credential, and whether its field is Required, alongside existing
wire framing classification. HTTP/JSON-RPC and gRPC use this shared plan. The
new runtime `security.IsOtherAuthorizationScheme` classifies a foreign ASCII
scheme token followed by a space and nonempty credential. `DecodeBearer` is
unchanged. Optional fields remain absent for foreign schemes only when another
requirement can omit them; malformed matching values, singleton requirements,
AND-only credentials and explicit Required retain strict rejection.

Acceptance evidence is in
`internal/securitygen/tla/credential_alternatives/README.md`. Model old control
fails and corrected model passes; generated HTTP/JSON-RPC/gRPC wire tests run
with the race detector. Nine bounded designs produce 108 artifacts; all repeat
bytes match and only six server decoder files change. Parent and candidate
modules compile and vet. No checked-in generated file requires editing;
checked-in ticktock designs have no security declarations, while the quality
fixture was regenerated in temporary copies and its OpenAPI output is unchanged.

The pinned golangci-lint 2.14.0 check covers the changed packages and the service
testdata package. The candidate has no new diagnostics. At parent `9880865a`,
`codegen/service/projection_conversion.go:89` and
`codegen/service/testdata/recursive_projection_dsl.go:11` still fail maintidx;
root's separate CI-repair agent owns those baseline failures. Do not claim this
branch's aggregate linter exit is green before those changes integrate.

Shared guidance proposals for root integration:

- Framework skill: replace the `internal/securitygen.IsBearer` paragraph with
  `PlanBearer` ownership, optional independent-alternative membership, unchanged
  strict `DecodeBearer`, and `IsOtherAuthorizationScheme` framing classification.
  Keep field Required and malformed/constraint errors unconditional.
- Consumer skill: explain that Basic/Bearer or bearer/key alternatives can omit
  optional credentials; authored Required still applies to the whole payload,
  and regeneration is required. Do not introduce conditional Required syntax.
- Formal model index: add the shared extraction model and its parser/context
  assumptions, linking the three generated transport regressions.

## Main integration acceptance

Integrated onto pushed `c735c636` after effective composition and projected-package
repairs. The implementation files match the independently reviewed candidate;
DSL and shared guidance merge without changing either neighboring contract.
The stale #743 hypotheses above were corrected and independently re-reviewed.

All 108 generated artifacts across the eight seam specimens and quality fixture
match a fresh process-isolated repeat. The eight seam specimens' 91 generated
files (plus the retained harness input) match the reviewed candidate exactly.
The quality fixture's 17 artifacts differ from the old candidate only in its
design digest, which matches the current parent. A current-parent generation
confirms the quality fixture's only behavioral output difference remains the
already intended accounts server bearer-extraction change. The other five
intended decoder differences retain their original exact output. Existing
compiled-module evidence therefore remains applicable.

Integrated HTTP, gRPC and JSON-RPC generated wire tests pass under the race
detector (5.995s, 5.972s, 6.139s). Full securitygen, security and service package
tests pass (0.558s, 0.491s, 6.162s). Pinned golangci-lint across those owners and
all three generator packages reports zero issues; previous baseline diagnostics
are resolved by the separate CI correction. Temporary snapshot probes were
removed from the checkout. Exact comparisons and logs remain under
`/tmp/loom-auth-integrated` until final review and delivery; no full corpus ran.
