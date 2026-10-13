# Shared interceptor accessor coverage (#782)

## Contract and owner

An interceptor has one generated Info/accessor declaration per service. Its
accessors must cover every method where either server or client applies that
interceptor. Side-specific endpoint chains must still contain only the methods
and callbacks authored for that side. A shared method is declared once.

Current analysis produces independent server/client InterceptorData, while
interceptors.go drops the client declaration by name. The retained server
Methods slice therefore becomes an incomplete declaration contract. A renderer
must not decide semantic coverage by choosing a side.

## Sequence

- [x] Read issue, current owners, dispatch model and accepted interceptor contracts.
- [x] Reproduce direct declaration coverage loss, preserving side-local controls.
- [x] Reproduce actual generated accessor failures in ordinary and streaming calls.
- [x] Model current side selection, an unsafe union of invocation sets, and a
  shared declaration union with independent side-local invocation sets.
- [x] Freeze readiness source/evidence and obtain independent architecture review.
- [x] Implement the reviewed shared declaration owner; remove renderer-side
  name filtering and derive declaration emission from analysis.
- [x] Verify focused direct, generated, determinism, compilation and runtime cases;
  update owner guidance.
- [ ] Obtain independent exact-final review.
- [ ] Integrate as one reviewed commit, push, close issue and clean temporary evidence.

## Proposed representation

Build one shared declaration record per interceptor from the server/client
analysis. Union method identities in service declaration order and reuse their
canonical MethodInterceptorData accessor names/type references. Keep each
side's InterceptorData.Methods and #867 client bindings unchanged for wrappers,
stream adapter collection and endpoint applicability. Deduplicate before
rendering private accessor types/methods and Info switches. Analysis records
which existing output file owns the shared declaration (server when present,
otherwise client), preserving public symbols and normal file layout.

Shared payload accessors must accept both a server endpoint carrier and the
canonical client payload where a streaming method has a carrier; this follows
the method's canonical ServerStream metadata, independently of which file emits
its declaration. Streaming accessors already distinguish send/receive through
their public operation methods and must share the same complete method union.
No new runtime fallback or per-transport coverage policy is needed.

Keep this declaration owner package-private on service Data. Construct it during
service analysis after side-local records exist, so fileData/WithPackageNames
reanalysis rebuilds references under each file's import aliases. Do not cache
rendered type strings across scopes. Existing serviceSymbols already allocates
one method/interceptor accessor identity across sides; consume those names
without reserving again. Preserve current field-interface metadata precedence
(server record when present, otherwise client); #767 owns its validation and
per-method representation limitations. The union order affects method dispatch
and declaration order only, not interceptor chain order.

The interface field contract stays unchanged. Incompatible accessed-field
representations across methods are #767 and are not silently normalized here.
JSON-RPC server carrier shape (#785), native stream lifetime (#726), endpoint
role applicability (#867) and streaming result identity (#859) remain separate
owners; this repair consumes their existing records.

## Bounded acceptance

Direct analysis/render checks cover server-only, client-only, identical,
partially overlapping and disjoint method sets, including a multiple-server
set, ordinary payload/result and all streaming accessor roles. They require
one declaration per method and unchanged side-local wrapper membership.

A small generated specimen uses distinct nominal payload/result/message types
with the same valid accessed field contract. Actual client-only method calls
must read and mutate their own fields without panicking or returning nil; server
and overlap controls must retain behavior. Include the one-server direct-cast
and multiple-server switch cases, plus bidirectional streaming operations and
server-carrier versus client-payload access. Missing attributes and invalid
stream access remain rejected by existing expr validation controls.

Final output comparison is bounded to the changed specimen and existing
ordinary/streaming interceptor fixtures. Compare base/candidate/repeat exact
bytes, enumerate intended declaration additions and changed dispatch switches,
and compile the generated modules. No full exported-design corpus audit.

## Readiness evidence

Parent: `cdc581066` (pushed #867, including #726).

- Direct `TestInterceptorAccessorCoverage`: six failing union coverage cases
  (overlap, single-server disjoint and multiple-server disjoint for unary and
  streaming); six single-side/identical controls pass. Side-local MethodData and
  InterceptorData membership remain exact.
- Four generated HTTP/service modules (unary/streaming by single/multiple
  server methods) generate, tidy and vet successfully. Ten client-only accessor
  calls fail at runtime: unary payload/result and streaming opening/send/recv,
  each under both server cardinalities. The single-server failure is a wrong
  nominal type assertion; the multiple-server failure is a nil accessor.
  Twelve overlap/server controls pass, including read/write mutation, native
  payload versus endpoint carrier, bidirectional operations and Close forwarding.
  These are actual generated typed-client/endpoint runtime checks, not mounted
  network transport claims.
- TLC `current` fails Coverage (73 distinct states), `append` fails
  UniqueDeclarations (72), `broaden` fails SideLocalCalls (64); `owned` passes
  126 distinct states across all 63 input pairs. All counterexamples exit 12,
  and the owned model exits 0.
- Existing TestInvalidInterceptors retains rejected unary-result access on a
  client stream and streaming-result mismatch controls.

Reproduction commands (with `GOEXPERIMENT=jsonv2`):

```sh
go test ./codegen/service -run '^TestInterceptorAccessorCoverage$' -count=1
LOOM_DIR="$PWD" go test ./codegen/generator -run '^TestGeneratedInterceptorAccessorCoverage$' -count=1
go test ./codegen/service -run '^TestInvalidInterceptors$' -count=1
```

At readiness, the new regression assertions were red and production code was
unchanged. Generated modules, source/input hashes and evidence are
recorded in `/tmp/loom-782-readiness-manifest.json` for independent review.

## Implementation and final acceptance

`interceptor_declarations.go` now constructs the private declaration owner during
service analysis. It shares immutable canonical accessor records, unions exact
interceptor identities in service method order, and leaves both invocation lists
untouched. `interceptors.go` consumes the recorded output-file placement; its
name-based client declaration filter is deleted. File import reanalysis rebuilds
the owner rather than copying stale type references.

- The direct declaration matrix now passes, including reversed side order and
  exact side-local invocation membership. An imported payload and accessed field
  in `types/context` verify fileData reanalysis produces `context2` references
  without mutating the original analysis.
- All five generated runtime specimens pass (four original shapes plus imported
  types). Actual NewEndpoints/NewClient nonapplication controls confirm sharing
  accessor declarations does not install callbacks on the other side. Existing
  server carrier/client raw-payload, send/receive mutation and Close controls pass.
- Seven bounded specimens produce 121 generated files at base, candidate and a
  separate-process repeat: all 121 candidate/repeat files match exactly. Only
  five `service_interceptors.go` files change. Each adds the missing client-only
  method's private accessors and its public dispatch cases; single-method direct
  casts become switches. Streaming payload cases retain canonical carrier/raw
  handling. Imported field and payload references keep the file's chosen alias.
  No file is added or removed. Existing ordinary and streaming controls are
  byte-identical, as are client constructors, wrapper chains, transports, CLIs,
  OpenAPI and all other artifacts. Existing checked-in interceptor goldens pass
  without modification.
- All fourteen base/candidate modules tidy, vet and compile; candidate modules
  pass race checks. The final harness still fails against base in all five
  affected specimens (eleven wrong-cast/nil-accessor witnesses, including the
  imported type). Original controls retain their success.
- Full service package tests and scoped service/generator lint pass. The model
  inputs/configurations remain byte-identical to approved readiness; its checker
  evidence remains valid. No exhaustive corpus or mounted-network claim is made.

Exact sources, checks, inputs, output hashes and intended changes are recorded
in `/tmp/loom-782-final-manifest.json` and its artifact ledger. Production and
new regression source will remain frozen during independent review; no commit
is authorized until that review resolves every finding.
