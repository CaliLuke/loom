# Shared CLI argument ordering (#697)

Readiness starts at pushed `214d83799bfe82287c21704124f8a09e25f72a82`.
The completed #782 worktree was archived, its exact reviewed changes reversed,
and the clean checkout fast-forwarded. Independent readiness review approved the
shared plan on this parent; implementation now follows that approval. #698 flag
quoting remains separate.

## Sequence

- [x] Read the report and both declaration/call owners.
- [x] Reproduce the two-service multipart/interceptor mismatch directly and in
  a generated example module; retain ordinary and one-feature controls.
- [x] Check streaming configuration, server selection/order and JSON-RPC reuse.
- [x] Model the competing iteration orders and one canonical ordered plan.
- [x] Freeze readiness sources, evidence and acceptance for independent review.
- [x] After approval, implement one private ordered argument plan shared by the
  parser declaration and example call, deleting duplicate order decisions.
- [x] Validate direct/rendered output, bounded independent-process generation,
  example compilation and unchanged controls.
- [x] Independent exact final and integration reviews approved.
- [x] Deliver atomic `af21e5098`, push and confirm #697 closed.
- [ ] Clean inactive temporary evidence after preserving compact receipts.

## Current owner disagreement

`appendParseEndpointStreamingParams` emits a global dialer, then one configurer
per selected streaming service. `appendParseEndpointCommandParams` emits each
service's multipart encoders followed immediately by that service's client
interceptors. `renderExampleCLIEnd` independently emits all multipart encoders
across services, then all interceptors. With two combined services the call is
`multipart(alpha), multipart(beta), interceptors(alpha), interceptors(beta)`;
the declaration expects `multipart(alpha), interceptors(alpha), multipart(beta),
interceptors(beta)`.

The shared owner belongs in `http/codegen`, where both HTTP and JSON-RPC already
share command and example generation. Its ordered entries must retain semantic
identity (kind, service, endpoint when relevant), rather than infer identity from
rendered text. Declaration and example renderers project names/types/expressions
from the same sequence after their own import allocation. Preserve the existing
parser signature order: global dialer, selected-service streaming configurers,
then selected-service endpoint encoders and interceptors. Service selection and
order must come from the same server's endpoint-bearing transport services.

Do not change public transport configuration callbacks or couple example import
aliases to parser import aliases. Keep package references under existing naming
owners. Constructor wrappers/CLI flags and interceptor invocation roles are not
part of this ordering repair.

## Bounded acceptance

Direct rendered declaration/call comparison covers no extras, multipart only,
interceptors only, one combined service, two combined services, streaming mixed
extras, and explicit server subset/reversed ordering. Generate representative
HTTP and JSON-RPC examples, compile the actual generated `cmd` packages, and
reuse those specimens for parent/candidate/repeat artifact comparison after
implementation. Every intended output change must be a corrected call argument
position or documented removal of an unhosted/endpoint-free argument. Existing
parser signatures and unrelated transport/service artifacts should stay exact.

A small ordering model will reproduce independent kind-grouped versus
service-grouped emission and check that a shared sequence preserves declaration
and call identity at each position. It abstracts Go types/imports and protocol
admission; direct AST assertions and generated compilation cover those seams.

## Concrete implementation boundary

Add a package-private plan builder that accepts the server expression and shared
`ServicesData`. Select endpoint-bearing transport services in `svr.Services`
order once. Its entries identify dialer, service configuration, endpoint
multipart encoder, or service interceptors, retaining exact service/method
identity. Neither entry stores a consumer's import alias or rendered type text.

`endpointParser` resolves the entries against its allocated `commandData` and
`subcommandData` after `allocateHTTPCommandIdentifiers`. `exampleCLIWithCache`
resolves the same entry kinds/identities against its example service imports,
API package alias, interceptor locals and `StreamingConfigDefault`. Replace
`appendParseEndpointStreamingParams`/`appendParseEndpointCommandParams` and the
independent loops in `renderExampleCLIEnd` with plan iteration. Keep the first
six fixed arguments unchanged. Derive whether the example needs its dialer from
the plan rather than adding another feature scan. Preserve the existing
transport callbacks; JSON-RPC supplies its distinct configuration type/default.

No parser signature change is intended. Existing handwritten callers remain
valid. `loom example` remains create-only: existing application examples are not
overwritten; users can reorder affected calls or generate a fresh example for
comparison. The framework/consumer guidance should explain this when delivered.

## Readiness evidence

Eight direct AST cases give four failures (combined, split features, combined
with WebSockets, reversed/subset server) and four controls (plain, single,
multipart only, interceptors only). The nine actual generated example modules
give the same four HTTP compilation failures and five passing controls,
including JSON-RPC unary and two-service WebSocket configurations. The error is
the reported function/interface position swap, not an unrelated build failure.
The final source and module evidence is `/tmp/loom-697-readiness-frozen.log`.
Scoped lint over HTTP codegen, its explicit testdata and generator tests passes.

The report understates one boundary: even services that each use only one
feature can fail when an earlier service needs interceptors and a later service
needs multipart. The split-feature specimen and model counterexample prove it.

The JSON-RPC multipart combination has a separate earlier defect: its accepted
design emits references to undefined encoder/decoder types, so it cannot yet be
an end-to-end compiler witness for this CLI ordering ticket. That exploratory
probe is retained in `/tmp/loom-697-readiness-rpc.log` and
`/tmp/loom-697-readiness-rpc/jsonrpc/combined`; reproduce with
`CLIArgumentOrderDSL("jsonrpc", "combined")`. It is excluded from the nine-case
acceptance matrix. Do not repair or silently reject that unrelated transport
contract as part of #697. Supported JSON-RPC controls still exercise both shared
renderers and distinct configuration default/types.

`http/codegen/tla/cli_arguments` explores 256 selections/feature combinations.
Current emission fails `MatchingIdentity` with an interceptor on service 1 and
multipart on service 2. The rejected unfiltered strategy fails when service 2
needs streaming but the server hosts only service 1. Owned emission passes all
256 distinct states (512 generated). The model is unchanged by physical
renderer details; final tests must compare actual emitted arguments and compile
all selected modules, then compare exact generated bytes in fresh processes.

Initial exploratory logs included a test-key spelling mismatch and an invalid
JSON-RPC route design. They are not acceptance evidence; the frozen tests use
semantic dependency identities and valid service-level JSON-RPC routes.

## Implementation

The shared plan now selects endpoint-bearing transport services and retains
service/endpoint pointers before either consumer renders names. Parser command
copies preserve those identities through allocation; the example reads each
ordered entry rather than running separate multipart/interceptor scans. Dialer
selection derives from the same plan. Endpoint-free services no longer contribute
example interceptor locals or arguments; parser signatures already excluded them.
Direct tests cover this explicit boundary and aliased per-service configuration.
Public guidance explains updating existing create-only example calls.

Readiness approval: `/tmp/loom-697-independent-readiness-verdict.json`.
Final bounded evidence, independent final review and main integration review
passed. Delivered `af21e5098`, pushed and #697 confirmed closed.

## Final acceptance evidence

- Nine emitted ordering cases and explicit selected service/endpoint identity
  tests pass (0.920s). The identity test deliberately renames a client import and
  asserts exact per-service configurer types after allocation; nil call values
  alone are not treated as identity evidence.
- Existing HTTP/JSON-RPC CLI declarations, example sections, server isolation,
  file-only parsers and transport configuration controls pass (0.608s/0.708s).
- The original nine generated modules build and vet (15.691s), including single
  combined service, each isolated feature, mixed features, streaming, explicit
  subset/reversal, JSON-RPC unary and JSON-RPC WebSocket controls.
- A tenth real file-only-service specimen reproduces an extra interceptor
  argument on the parent (4.254s expected compiler failure). Candidate build/vet
  passes (5.602s). Its parent generator used a recorded source overlay restoring
  only four production files and replacing the new owner file with an empty
  package; runtime/dependencies are unchanged. Two initial invalid fixture
  spellings were corrected before these final results and are not evidence.
- All 400 generated/example files across ten modules match a fresh process's
  output byte for byte. Parent comparisons show only five changed example call
  files: four interceptor argument moves and removal of the file-only service's
  unused import/local/initialization/argument. All parser signatures and all
  other artifacts remain exact. See `/tmp/loom-697-final-artifacts.json` and
  `/tmp/loom-697-generated-delta.patch`.
- Both checked-in ticktock fixtures regenerate byte-identically and compile:
  HTTP 10.667s, JSON-RPC 10.400s. No fixture edits are needed.
- Scoped lint passes with zero issues; formatting and diff checks pass. The
  unchanged owned model's 256-state readiness result remains applicable; only
  its implementation correspondence documentation changed.

Candidate runtime/codegen source stays on the reviewed readiness parent
`214d83799`. Main integration preserved #790 CLI body analysis in
`client_cli.go` and received independent narrow review before delivery.
The separate JSON-RPC multipart codec defect remains excluded as documented.
