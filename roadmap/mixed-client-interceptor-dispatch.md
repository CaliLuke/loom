# Mixed client interception boundaries (#867)

Status: implemented and validated; exact final packet awaits independent review.
Base: `c8515931c0f74cbdac1814f9ce0b9975c5bcc899`.

## Contract and evidence

A mixed method has two client endpoints: an ordinary result and a stream handle.
The constructor already retains that distinction, but both endpoints consume the
same method-level interceptor chain. A streaming interceptor therefore asserts
the ordinary result to the stream interface before any accessor executes.
The existing compiled HTTP and JSON-RPC service-client witness in issue #867
passes without an interceptor and panics with one. #859 corrected streamed field
and nominal identities; this failure remains a separate dispatch boundary.

The contract follows the existing interceptor guide: payload access intercepts
the opening call, streaming access intercepts message operations. A stream-only
interceptor has no operation to intercept on the ordinary endpoint. An interceptor
with both payload and stream access must retain its payload callback there.
Interceptors without streaming access retain their existing ordinary callback,
including interceptors with no access declarations. Existing chain order, nil
interceptor bypass, error propagation and application short circuits remain.
Ordinary result access on result-streaming methods remains rejected.

## Sequence

- [x] Preserve the compiled unary panic witness and delivered #859 inputs.
- [x] Inspect shared construction, wrapper rendering, naming and transport users.
- [x] Check current behavior and alternatives in the boundary model.
- [x] Freeze direct red/control evidence.
- [x] Obtain independent readiness review.
- [x] Implement one shared client endpoint interception plan.
- [x] Run the bounded matrix, update guidance and freeze the final review packet.
- [ ] Obtain independent final review; orchestrator integrates one atomic commit.

## Implemented owner

Analyze each client endpoint's role once: mixed primary is ordinary; mixed
secondary is streaming; a nonmixed method with a client stream has a streaming
primary; other primaries are ordinary. Build a plan containing the ordered
applicable interceptor bindings and whether each binding intercepts the opening
call, stream operations, or both. The constructor, public chain emitter and
private wrapper renderer consume that same plan. Never infer the role from a
runtime type assertion or independently in a transport.

For an ordinary endpoint, exclude stream-only bindings and retain the ordinary
part of combined bindings. For a streaming endpoint, preserve current opening
payload and message behavior. An empty chain passes the endpoint through.
Do not duplicate `MethodInterceptorData` or its accessor declarations to obtain
two wrappers: endpoint bindings refer to the existing method/interceptor record.
Stream operation references continue to come from canonical `StreamData`.

Early evaluated-expression analysis determines role and access capabilities
before `serviceSymbols` allocates package declarations. Reuse that result in
later service analysis. The existing package NameScope reserves every emitted
wrapper before nominal types; consumers use the recorded name. Ordinary and
nonmixed wrapper names stay unchanged. Mixed primary retains
`WrapWatchClientEndpoint` when it has applicable bindings; mixed secondary uses
`WrapWatchStreamClientEndpoint`. If no interceptor has message access, both
endpoint roles reuse the identical
opening-only chain rather than emit duplicate wrappers. Private mixed bindings
with distinct behavior analogously distinguish
`wrapClientWatchAudit` and `wrapClientWatchStreamAudit`. Empty primary chains
emit no unused wrapper. Accessor and wrapped-stream identities remain shared.
Generated aliases or compatibility modes are unnecessary; regenerate clients.

Affected owners: `service_data.go`, `service_data_interceptors.go`,
`package_symbols.go`, `client_interception.go`, `client_jennifer.go`, `interceptors.go` and
`interceptor_sections.go`. Stream method rendering is a consumer to verify, not
a second policy owner. HTTP/JSON-RPC constructors receive these shared generated
clients; their transport templates need no dispatch policy. Server wrappers and
endpoint carriers remain unchanged (#785). Shared accessor declaration coverage
(#782) and shared attribute analysis (#767)
remain separate: endpoint bindings reference existing method/interceptor data,
without changing which side emits Info/accessor types. No dependency on either
repair is needed for client-only or already-covered shared methods.
Context/lifetime work (#726), stream
attribute identity (#859), and arbitrary member naming (#700) remain separate.

Rejected alternatives: skipping every interceptor on mixed ordinary calls loses
payload callbacks; always using ordinary wrappers loses stream callbacks;
duplicating whole method analysis duplicates accessors and naming policy. A
runtime cast guard can hide the panic but leaves invocation identity implicit.

## Bounded acceptance

| Case | Obligation |
| --- | --- |
| Mixed HTTP + JSON-RPC, stream-only and combined payload/stream profiles | Typed ordinary result retained; zero stream-only ordinary callbacks; combined ordinary callback exactly once; stream read/write accessors use Chunk, never Summary |
| Same mixed methods, payload-only, no-access and nil interceptor controls | Existing callback behavior, ordering and bypass unchanged |
| Opening failure, payload short circuit, stream receive failure | Preserve values/errors; no premature stream callback; short-circuit success must obey endpoint result contract |
| Two applicable interceptors | Preserve existing execution order at opening and message operations; mutate payload/results and verify next layer sees changes |
| Ordinary unary and nonmixed result stream | Existing output bytes and runtime behavior unchanged |
| Existing HTTP/gRPC final client-stream result | Keep final receive operation interception and original context/Close delegation |
| Authored type colliding with new mixed wrapper base | Package declaration owner allocates all names/references consistently; generated module builds |
| HTTP mounted ordinary + stream; JSON-RPC mounted stream | Real transport-backed client callbacks, mutation, EOF and early cleanup |

JSON-RPC's SSE transport endpoint opens a stream; its ordinary service-client
boundary control supplies a valid ordinary endpoint directly. This ticket does
not add unary negotiation to that transport or claim a mounted RPC unary path.
Inspect actual existing ticktock consumers and reuse their stream controls.

Use at most four generated specimen groups: mixed profiles (both transports),
mixed collision, ordinary unary/nonmixed stream controls, existing final-result
controls. Capture base/candidate/process-isolated repeat artifacts; compare exact
bytes and compile/vet candidate modules. Run mounted race checks only where the
changed boundary executes. Intended differences are mixed constructor bindings,
their public/private wrapper declarations and the corresponding mixed golden;
new wrapper-name reservations may rename colliding authored types and their
references. No transport/protocol identity or stream accessor change is intended.
Enumerate actual changed paths before final review. Run direct and affected
service/generator checks plus scoped pinned lint; do not sweep the full corpus.

The adjacent `client_interceptor_dispatch` model explores boundary selection,
ordered callbacks and failed opens. It does not prove Go compilation, naming,
I/O lifetime, arbitrary application short circuits or transport negotiation.
Generated tests supply those implementation checks within the bounded scope.

## Readiness results

On the base above, `TestClientInterceptorDispatchReadiness` fails for stream-only
and combined profiles; payload-only and generic profiles pass. A fresh generated
shared-result specimen builds and vets, then reproduces both ordinary service
client panics (HTTP and JSON-RPC, 0.394s). Its four mounted stream cases (both
transports, plain/intercepted) pass in 0.135s, preserving #859's delivered fix.
The original #867 issue witness remains archived unchanged. At readiness, new
code was limited to a test, this plan and the model; production was unchanged.

The corrected model checks 272 concrete combinations and 545 distinct states.
Current dispatch fails `OrdinaryResultRetained`; skipping ordinary interception
fails `OpeningCallbacks`; ordinary-only wrappers fail `MessageCallbacks`. The
owned plan passes, including deadlock detection (finished states self-loop).
Initial model exploration exposed an unparenthesized boolean assignment that
accidentally constrained transitions; it was corrected before these results and
deadlock checking was enabled. This is model-development evidence, not an
implementation result. A first mounted-test command matched no tests; it is
explicitly excluded from acceptance and replaced by the actual four-case run.

Pinned tools: Go 1.27.2, TLC jar 1.7.4 (TLC 2.19), Corretto 25.0.4.1. Exact inputs,
source/owner/artifact hashes and command results are frozen in the temporary
readiness manifest for independent review. Keep model sources/configuration and
concise results; remove temporary generated modules/checker states after final
validation and review, preserving the issue's durable reproduction inputs.

Implementation parent advanced by clean ff-only to `f64fc2d1e`; the eight frozen
readiness inputs were preserved. The approved readiness packet remains historical
evidence against c851. No shared interception owner changed during this advance.


## Implementation and final evidence

The early `clientInterception` plan now owns the ordered endpoint bindings.
`serviceSymbols` allocates its public and private declarations before nominal
user types. `MethodData.ClientInterceptorWrapper(stream)` exposes the recorded
name (or absence) without repeating applicability. Constructors and wrapper
emission consume the same plan. The single accessor record is unchanged, and
nonmixed or identical opening-only chains retain their existing declarations.

Implementation tracing found one necessary consumer omitted from the readiness
inventory: generated CLIs call service interceptor wrappers directly. The
orchestrator approved including this correspondence. Shared command data now
accepts the selected allocated wrapper, including absence. HTTP CLI supplies the
ordinary mixed endpoint; JSON-RPC SSE CLI supplies its stream-opening endpoint;
gRPC CLI supplies the role of its actual endpoint. No CLI layer decides which
callbacks apply. Existing gRPC CLI golden output is unchanged and the mounted
HTTP/RPC specimen invokes both stream-only and combined CLI commands.

Four specimen groups produced 159 files each at base/candidate/repeat (477
hashes); final candidate and independent repeat match byte-for-byte. Ordinary
unary/nonmixed and final-result groups are unchanged. The mixed group changes
exactly eight files: `client.go`, `client_interceptors.go`, and
`interceptor_wrappers.go` for each service, plus HTTP and JSON-RPC CLI parsers.
They contain the endpoint-specific chains and references only. The collision
group changes those same eight files plus these twelve physical-name consumers:

- Each service's `service.go` (two files).
- Each transport client's payload CLI builder (two files).
- Each transport's client/server `encode_decode.go` (four files).
- Each transport's client/server `types_helpers.go` (four files).

Those twelve changes are exactly the authored type
`WrapCombinedStreamClientEndpoint` becoming `WrapCombinedStreamClientEndpoint2`,
its references and derived constructor names, with formatting/comment wrapping
caused by the longer name. Normalization checks both code and comment content;
exact-byte artifact hashes remain the primary record. Only the existing mixed
constructor golden changes among the checked-in service fixtures.

All eight base/candidate modules tidy, vet and compile. The mixed/collision
candidate harnesses pass runtime and race checks: six profiles per transport,
nil/installed interceptors, ordered opening and message callbacks, payload and
message mutation, endpoint errors, typed short circuits, receive errors/EOF,
opening/explicit operation contexts and Close delegation. Mounted HTTP ordinary
and stream calls, mounted JSON-RPC stream calls, both CLIs and early client Close
pass. The final harness reproduces both combined unary panics against the base
in 0.444s. JSON-RPC's supplied unary service endpoint is tested directly; no
mounted unary RPC transport support is claimed.

Additional controls pass: complete service/CLI packages (35.783s/0.405s), four
HTTP final-result cases (16.197s), gRPC final and the four #859 identity cases
(13.678s), JSON-RPC request/reply (8.42s), unchanged gRPC CLI golden (0.303s), and
regenerated HTTP/JSON-RPC ticktock fixtures (14.266s/13.428s). The latter compare
checked-in output exactly and run typed opening-cancel, operation-cancel,
deadline, EOF and early-close controls. No exhaustive corpus was run.

Model/configuration bytes are unchanged from approved readiness; its counterexamples
and 545-state owned result remain valid. The README and model index now name the
implemented seams. Scoped pinned lint reports zero issues. Formatting, naming,
file-size, documentation and diff checks complete the final gate receipt.

During harness construction, invalid placement of JSON-RPC ID metadata was
corrected to the payload declaration, test names were capitalized, and the HTTP
service was made to return its synchronous result while sending to the provided
discard stream. Those were fixture mistakes, not additional framework repairs.
A maintainability lint finding was resolved by extracting chain-section emission
from the existing file generator without changing emitted bytes. Temporary
comparison/checker outputs are retained for independent review and must be
cleaned afterward, preserving exact inputs, hashes and compact results.
