# gRPC stream authentication admission — #743

Status: implemented and ready for independent review in managed worktree, parent
`433d2e741b43b35783d293d0de60cabeca0be6a2`.

Sequence: preserve/recheck red and model; extract shared authentication execution;
wire metadata snapshot and stream admission; validate focused transport contracts;
freeze exact output comparison and final files for independent review.
No main-worktree file was modified. This replaces the earlier
checkpoint's exactly-once and available-alternative-first hypotheses.

## Owner and recommendation

Add a real authentication admission boundary before a gRPC stream-envelope
receive when **every authentication requirement is metadata-resolvable**.
Share the generated method authentication function with the ordinary endpoint.
After admission, decode and validate the complete request and enter the ordinary
endpoint, including its authentication and application access checks. Do not
cache successful authentication or bypass the complete-payload boundary.

Keep the existing endpoint function fields and middleware composition. A separate
endpoint object with direct/deferred entrypoints is unnecessary: it mainly avoids
an extra authentication pass, but no inspected public contract promises that.
`docs/authorization.md` (Invocation boundary, lines 137–157 at the examined source)
explicitly permits repeated authentication and requires synchronous repeatable
checks; `security.Protect` GoDoc requires read-only repeatable checks.
`security.Auth*Func` GoDocs specify their signatures and contain no exactly-once
promise. Document the newly distinct admission and complete-request boundaries
for auth-only streaming methods too. Application authentication must not consume
one-shot credentials or perform irreversible work per callback invocation.

This fixes the actual resource-admission defect without changing generated
endpoint field types, adding a success token, moving middleware to a different
position, or creating another transport implementation of AND/OR policy.

## Confirmed reproduction

Pinned source: `651372ce6ff05ed1db495369fd51d48df0c0ec75`, extracted with git archive
into `/tmp/loom-743-design/source`. This avoids changes to the concurrently
integrated main checkout. The existing `renderGRPCModule` helper rendered the
issue's API-key client-streaming envelope design into
`/tmp/loom-743-design/generated`.

The real grpc-go bufconn matrix reproduces:

- Wrong metadata, no first frame: client deadline expires; zero authentication
  callbacks and zero service calls; the server entered initial RecvMsg.
- Missing required metadata, no frame: same deadline and receive; no callback.
- Valid metadata, no frame: still zero authentication callbacks before deadline.
- Wrong metadata with a first frame: prompt failure, one callback, no service.
- Valid metadata with a first frame: success, one callback and one service call.

The first three cases fail the intended admission assertions; both first-frame
controls pass. Server receive entry is instrumented separately from deadline
observation. The one-second deadline bounds a request that intentionally never
sends or half-closes; it is not the only evidence for the ordering defect.

Repro input: `source/grpc/codegen/admission_readiness_test.go` and
`generated/admission_test.go` under the scratch directory. Complete wire log is
`/tmp/loom-743-design/wire.log`. Source and generated module are retained for the
implementation owner; no compiler output is proposed for commit.

## Eligibility and exact behavior

Eligibility is additionally limited to generated branches that would otherwise
receive an initial stream envelope before authentication. Unary, server-streaming
and no-initial-receive methods retain their existing phases.

The gRPC analysis owns wire availability. Compare the actual resolved credential
attribute identities with request metadata mappings, including both Basic
components, API-key/JWT/OAuth fields, aliases and mapped names. Every scheme in
every OR requirement must be available without the initial frame. Empty
anonymous requirements need no credentials. Do not infer availability from an
empty string, requiredness, Authorization header spelling, or the presence of
one successful ready branch. Unknown/transport-owned schemes require an explicit
availability fact, not an optimistic assumption.

AND and OR execution stays in the shared service authentication owner. Preserve
scheme and requirement declaration order, failed-branch context reset, successful
AND context chaining, optional absent credentials passed as empty strings, and
anonymous alternatives. Required and authored metadata validation remain global.
The #732 bearer extraction plan must be consumed, not copied or weakened.

Accepted cases:

- All-metadata API key, wrong: reject before Recv, no service/access/interceptor.
- All-metadata required key, missing or invalid: decode/validation error before
  Recv, no authentication if required-field decoding already rejects it.
- All-metadata valid key: authenticate at admission, then wait for the first
  envelope. No payload-dependent access, interceptor or service runs yet.
- All-metadata OR/AND and anonymous alternatives: execute the same declared
  requirement list as the full endpoint. Anonymous success does not suppress
  unconditional metadata validation.
- Valid admission followed by a wrong-kind envelope, missing initial payload,
  invalid complete payload or cancellation: reject, no service invocation.
- Valid admission followed by full endpoint authentication failure (for example
  a revoked credential while waiting): reject at the complete boundary.
- Fully decoded endpoint middleware retains its existing positions and checks.
  Authorization-enabled continuation rechecks still reject mutated payloads.
- Successful admission contexts flow into decoding and the normal endpoint,
  like the successful contexts flowing between existing protected boundaries.
  The complete endpoint re-evaluates authentication from that incoming context.

Still rejected / intentionally not promised:

- A message credential cannot be authenticated before its frame exists.
- A mixed OR list waits for the initial frame in either declaration order.
  Selecting a ready later metadata branch would change the declared winner;
  a message alternative can still authenticate when all ready alternatives fail,
  so that optimization does not remove the worst-case unauthenticated wait.
- Authentication is not exactly once. A successful metadata admission and normal
  endpoint authentication are distinct checks; middleware can add more checks.
- Admission does not authorize payload-dependent application access, individual
  stream messages, or continued access after revocation.
- Raw service calls or externally replaced endpoint fields retain the documented
  application-owned escape-hatch status.

## Implemented API and ownership

The implementation retains the approved contracts without adding DSL or a
dependency. No endpoint-object migration is needed.

1. Extract the context/cancellation execution currently inside
   `security.protectedEndpoint` into one shared runtime operation:

   ```go
   func RunCheck(ctx context.Context, request any, check EndpointCheck) (context.Context, error)
   ```

   It retains the original request context through the existing private origin
   key, checks original and supplied cancellation before/after the callback,
   rejects a successful nil context, and returns the callback context with the
   original marker restored. `Protect` delegates its existing check execution to
   this owner. Preserve its current error precedence and middleware markers.
   This is execution ownership, not an authentication-success marker or cache.

2. For secured streaming methods, generate one private authentication factory
   from the existing `buildEndpointAuth` requirement renderer. Its
   `security.EndpointCheck` consumes the typed **payload**, not the stream input
   wrapper, and performs only authentication. Both the ordinary method endpoint
   and any authorization-aware complete check invoke this same factory-produced
   function. The aggregate `Endpoints` retains a private bound check and exposes
   a method such as:

   ```go
   func (e *Endpoints) AuthenticateUpload(ctx context.Context, payload any) (context.Context, error)
   ```

   This public transport entrypoint runs that check through `security.RunCheck`.
   Its GoDoc explicitly says it performs authentication admission only, not
   complete payload validation or application authorization. It does not replace
   `Upload`, whose type remains `loom.Endpoint`. Allocate the new identifiers in
   the existing generated scope; do not introduce collision-prone string names.
   Raw authentication callbacks are bound at construction, not looked up by
   scheme name at request time. No gRPC requirement executor is generated.

3. Factor the gRPC metadata decoder so the admission and complete decoders share
   extraction, framing, validation and typed conversion. A generated admission
   decoder returns a typed service payload containing only metadata fields.
   Only the authentication check sees this partial value. The full decoder still
   requires and validates the initial envelope and constructs the complete input.
   Do not call the ordinary full decoder with a nil frame and suppress its errors.

4. Capture incoming metadata once before admission and preserve that snapshot
   across callback context replacement. `metadata.FromIncomingContext` after a
   callback returning `context.Background()` would otherwise lose credentials.
   The snapshot is transport input, not authenticated evidence. Pass the successful
   authentication context onward; do not overwrite it with the original context.

   The runtime API adds optional `Admit func(context.Context) (context.Context,
   error)` to `StreamServerSpec`, and:

   ```go
   func AdmitStream(ctx context.Context, decode RequestDecoder, check security.EndpointCheck) (context.Context, error)
   ```

   `AdmitStream` captures the original incoming metadata, calls the generated
   partial decoder and authentication check, then restores metadata onto the
   successful callback context. The existing decoder-error normalization is
   shared by both phases. Metadata failures mark decode failure; authentication
   failures retain normal endpoint error mapping and handler-failure observation.
   RequestObserver remains first-failure-wins.

   `StreamHandler`, `NewStreamHandler`, and generated handler constructor
   signatures stay unchanged. The generated server stores the allocated bound
   authentication accessor and supplies an Admit closure only for eligible methods.
   `ServeStream` owns the request observer and service/method identity; it restores
   those after callback context replacement before Decode and Handle. Failed
   admission never invokes Decode. Physical Recv retains the original gRPC stream
   context, so replacement cannot detach the receive from connection cancellation.
   Shared checks preserve the original cancellation marker before service execution.

5. Regenerate the service and gRPC packages together. HTTP and JSON-RPC still
   consume the same endpoint fields and constructor signatures. Secured streaming
   endpoint authentication moves to a shared factory in all transports; only
   eligible gRPC methods gain an extra phase. No mutable admission-success state,
   alternate legacy API, or transport copy of requirement policy is introduced.

## Model evidence and limits

Durable model: `security/tla/stream_admission/StreamAdmission.tla`.
It enumerates two OR alternatives, metadata-only versus mixed availability,
success/failure outcomes, valid/invalid first frames, optional middleware,
credential mutation at continuation, callback context replacement (modeled as
still live) and cancellation of the original request. Complete decode validation
precedes the normal authentication boundary. It admits client silence as a wait
state; a safety invariant forbids entering that wait without required admission.

Actual required invariants are prompt metadata admission before waiting,
declaration-order winner selection, complete payload before access/interceptor/
service, original cancellation before service, and middleware rechecks. The
separate `AuthenticationBranches.tla` model checks two AND schemes in each OR
branch, declaration order and reset of a failed branch's returned context.

`admission-checked.cfg` is the recommended two-boundary model. `checked.cfg`
represents the more expensive endpoint-object candidate. Both satisfy the actual
safety invariants. The `exactly-once-hypothesis.cfg` result only demonstrates that a hypothetical
exactly-once requirement would reject the recommended two-boundary design; that
requirement has been withdrawn and is not an acceptance criterion. It remains named as a hypothesis to prevent treating it as a required invariant.

The other negative configurations deliberately omit a real invariant: receive
before admission; choosing a later available OR branch; losing original
cancellation; omitting middleware rechecks; leaking a failed branch context.
They produce bounded counterexamples. These models do not prove arbitrary Go
callbacks terminate, credentials remain valid while waiting, TLS/HTTP2 resource
limits, parsing correctness, actual context-key propagation, or generator output
compilation. Exact model counts and toolchain hashes are in `security/tla/stream_admission/README.md`.

## Implementation sequence and acceptance

1. Preserve the generated red and expand it to bidirectional envelopes and
   metadata-only/mixed alternatives; commit only durable inputs and summaries.
2. Extract shared check execution and generated stream authentication ownership.
   Keep Protect tests green and add context replacement/nil/cancellation tests.
3. Add metadata snapshot/partial decoding and optional stream admission, then run
   the red against the candidate. Assert RPC status/detail, receive entry,
   authentication phase sequence, access/interceptor/service counts and payload.
4. Cover both OR orders, AND context chaining, failed-branch isolation, anonymous
   OR, optional absent values, Required/constraints/aliases, #732 framing behavior,
   EOF/wrong first envelope, late credential revocation, original cancellation,
   auth callbacks replacing context, and existing middleware short circuits and
   mutations. Add direct endpoints plus unary and server-streaming controls.
5. Compare parent/candidate generated bytes and isolated repeats for the envelope
   fixture, ordinary secured stream fixture and authorization/middleware fixture.
   Compile/vet generated candidates. Record only intended admission/helper/wiring
   changes and preserve normal HTTP/JSON-RPC wire contracts. Run affected shared
   security/service and gRPC tests; no exhaustive corpus required.
6. Update docs/grpc-guide.md, docs/authorization.md and callback GoDocs with the
   genuine two-boundary/repeatable contract, metadata-only limitation and required
   regeneration. Root owns shared skill/index updates. Retain durable models by
   the shared security owner and link exact regressions and limitations.
7. Freeze exact files/evidence for independent review, then one #743 commit/push
   and issue reconciliation. Root approved this readiness and assigned implementation on the stable
   post-#732 parent. Commit, push and issue reconciliation remain root-owned.

## Implemented acceptance evidence

- [x] Reproduced the original no-frame bug on the integration parent `433d2e74`
  before production changes: wrong, missing and valid credentials enter initial
  receive with zero callbacks; wrong/valid first-frame controls complete.
- [x] Rechecked current and corrected admission models plus AND/OR context models.
  Expected legacy/failed-context counterexamples reproduce; corrected models pass.
- [x] Full `codegen/service`, `security`, and `grpc` package tests pass.
- [x] Generated bufconn tests pass with the race detector for client-streaming,
  bidirectional and authorization-enabled cases. They assert initial receive
  entry, public RPC status/problem details, callback/service/access counts,
  winning contexts and complete service payloads. Context replacement, nil
  contexts, late revocation, invalid/EOF/wrong envelopes and original cancellation
  are covered. Middleware short circuit, credential mutation and payload mutation
  retain their protected boundaries.
- [x] Ordered metadata OR in both orders, failed-branch reset, AND, anonymous,
  named aliases, Bearer/foreign-scheme/absent alternatives, mixed OR in both orders
  and generated accessor name collisions compile and pass generated race tests.
  Both Basic credential components must be metadata-mapped; direct eligibility
  checks reject either mixed component mapping.
  Unary, server-streaming and metadata-only stream controls receive no admission.
- [x] Six bounded designs render at the parent, candidate and independent repeat:
  client, bidirectional, authorized, alternatives, non-initial-receive controls
  and existing `UnaryRPCNoPayloadDSL`. All 71 artifacts match exactly between
  independent candidate processes; all six candidate modules build and vet.
  Intended parent differences are three server/service files for client, bidi
  and alternatives; those plus `authorization.go` for authorized; endpoint
  authentication extraction only for secure controls; zero for the existing
  unsecured unary control. Clients, protocol buffers, request/response shapes,
  response scaffolds and authorization manifests remain unchanged.
- [x] Selected HTTP WebSocket/session/authorization-stream and JSON-RPC
  session/authorization integration checks pass after shared owner extraction.
- [x] Existing gRPC streaming, request decoder, client/server, security-scope,
  session and metadata-partition generator checks pass. No checked-in golden or
  ticktock fixture output changes were required.
- [x] Scoped golangci-lint reports zero issues; repository gofmt and NameScope
  lints and `git diff --check` pass.

The broad gRPC package run started during development reached its default
10-minute aggregate timeout while progressing through unrelated generated tests;
it is not claimed as passing or repeated as an acceptance requirement. The
focused tests above establish this repair's selected contracts. Development
scratch output and logs remain outside the repository through independent review;
root will remove them after review. No exhaustive exported-design audit was run.

## Integration review evidence

The isolated review found no medium or high issues. Its two low findings were
mechanical test-block formatting and placeholder module versions in comparison
scratch. The formatting correction passed the affected generated race tests
again (19.441s). All eighteen parent/candidate/repeat modules were regenerated
using pushed pin `v1.10.0-alpha.9.0.20261012181439-651372ce6ff0` and an explicit
local replacement for the source being checked. All 71 artifacts match the
previously reviewed candidate byte for byte and the independent repeat; all six
candidate modules build and vet. Parent differences are exactly those listed
above. The corrected 41-file manifest SHA-256 is
`63294943178f0ba6bf69a6b6422019ac435231f1eda15dca06177d7af30ac8d3`.

Root integrated these exact files onto `27da0247`, with no overlapping delivered
source changes, and added the owning framework/consumer guidance and model index.
Integrated owner packages pass (`security`, `grpc`, `codegen/service`); all
generated admission race cases pass (21.183s). Two isolated integrated
generations reproduce all 71 reviewed artifacts byte for byte, and all six
modules build and vet with the same pushed pin and explicit local source.
Independent final review remains pending.
