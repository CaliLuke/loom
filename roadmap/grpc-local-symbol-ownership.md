# gRPC local symbol ownership (#691)

Parent: `8ce69db06302f98bc9f131071f9c42bd4b55f6f4` (pushed main).
Status: implementation and bounded acceptance complete; final independent review pending.
The exact readiness packet passed independent review before production changes. The later #694 commit corrects service
package/member identities, not these import bindings; integration must record
its parent delta rather than assuming the readiness parent changed.

## Scope and accepted contract

Authored service names remain valid regardless of fixed generated Go locals.
gRPC analysis assigns transport import aliases; every generated reference uses
that allocated identity. Preserve neutral service package declarations, package
paths, protobuf packages/messages/field numbers and authored method names.
The `req` witness concerns the import alias `reqpb`, not protobuf message naming.

HTTP/JSON-RPC already reserve fixed locals in their transport import owner;
HTTP example files also retain file-specific service/server import bindings.
They are existing architecture controls, not another repair site. Keep #694
service declarations, #700 transport members, #736 protobuf messages, and
unrelated interceptor/error families outside this repair. The cross-transport
Go `init` restriction is independently reproduced and tracked by #856, extracted
from the comment on #691. Do not add a gRPC-only `init` exception.

## Sequence

- [x] Reproduce parent generated compilation failures and passing controls.
- [x] Inventory affected import and local scopes, including downstream example
  file allocation and custom type metadata locals.
- [x] Model the current failures and proposed ownership in both directions.
- [x] Preserve and separately file the `init` defect (#856), following root review.
- [x] Freeze exact readiness inputs/evidence.
- [x] Obtain independent readiness approval.
- [x] Implement the approved owners and direct regressions.
- [x] Compare the bounded parent/candidate/process-isolated repeat artifacts,
  compile/vet generated modules, run affected direct tests and models.
- [x] Update public regeneration guidance, consumer/framework guidance and model
  correspondence.
- [x] Freeze the final exact diff.
- [ ] Obtain final independent review.
- [ ] Parent delivers one reviewed atomic commit and closes #691; remove inactive
  scratch outputs after preserving compact receipts.

## Evidence and consumer inventory

One marked design covers unary request metadata, response headers/trailers,
union conversion and CLI decoding, typed errors, secure stream admission,
viewed results, validation and all three streaming directions.

| Boundary | Current owner and observed failure | Intended owner |
| --- | --- | --- |
| Service/protobuf aliases | `newImportAliases` reserves fixed imports and receiver `c` only. Services `md`, `ctx`, `v`, `result`, `err`, `hdr`, `trlr`, `message`, `p`, `s`, `stream` fail; `req` fails through alias `reqpb`. | One private gRPC fixed-name scope factory seeds both import families before allocation. |
| Metadata locals | `metadataVarScope` separately reserves 13 locals, actual service/view/protobuf aliases and walked custom package aliases. Its policy cannot stop an import from shadowing a fixed local. | The same fixed-name scope factory, then actual imports, then metadata names. |
| Per-file custom aliases | `ServiceData.FileImports` and `ServicesData.WithPackageNames` reanalyze file-specific relocated type aliases. gRPC `fileData` consumes this owner. | Preserve this owner and feed its actual aliases into metadata allocation. The new md/mdsvc relocated-package fixture checks the newly exposed suffix binding. |
| Client/server code | `client.go`, `server.go`, `client_types.go`, `type_files.go` and encode/decode sections use retained service/protobuf identities. `ClientCLIFiles` also renders with `fileData`. | Keep these consumers deriving from analysis; no template-local escaping or text substitution. |
| Example server file | `exampleServer` allocates each server/service/protobuf import in its header, but `jennifer_example_server.go` reconstructs references from transport aliases. `ordinary` + `ordinarysvr` already fails. | Private file data retains each role's final alias; header and every rendered qualifier use it. |
| Aggregate regression | Parent `p` + `psvcsvr`, with simple unary string payload/result, compiles/vets. Escaping p to psvc exposes the existing example mismatch. | Preserve this accepted control through the file-specific owner, not by exempting p. |
| CLI/other examples | Aggregate CLI has its own recorded command identifier owner. Neutral service stubs/main files have separate import scopes. | Reuse them; compile complete examples to verify propagation. |

`testingx.ServicePackageShadowNames` observes 18 fixed identifiers in scopes
where the marked service/views qualifiers can be hidden:
`ctx err hdr md message name ok p proj reqpb res result s stream trlr v val vres`.
The proposed single gRPC local inventory is their union with the existing
metadata reservations and receiver:
`c ctx err hdr md message name ok p payload proj reqpb res result s stream trlr v val vals value vres`.
The AST helper checks service/views scopes, not every protobuf selector; the
separate `req` compile and direct alias contract establish the protobuf case.
Authored marker names are excluded. Future renderer changes must update this
inventory through the same structural coverage test; this is a bounded sample,
not an exhaustive Go scope proof.

## Concrete allocation sequence

1. A private gRPC name-scope factory reserves `transportGeneratedImportNames`
   and one `transportGeneratedLocalNames` inventory. Preserve receiver c's
   reservation through this inventory; remove the separate one-off policy.
2. `newImportAliases` uses it, allocates all protobuf aliases, then all service
   aliases in existing declaration order. Keep NameScope suffix behavior:
   md -> mdsvc, ctx -> ctxsvc, req retains service alias req but its protobuf
   alias reqpb -> reqpb2. Store both aliases in existing transport data.
3. `metadataVarScope` uses that same factory, idempotently reserves actual
   service/view/protobuf/custom aliases and allocates request metadata or the
   shared header/trailer sequence. Never mutate neutral nominal type scopes.
4. gRPC example-server import construction retains existing role allocation
   order (server, service, protobuf) to avoid gratuitous output changes. Store
   final names in a private file-specific record containing `data *ServiceData`,
   `serviceImport`, `serverImport`, `protobufImport`, analogous to the existing
   HTTP private example record. Pass these records to rendering; do not mutate
   cached service data or rebuild imports after rendering.
5. Parameter endpoint types use serviceImport; server declarations/constructors
   use serverImport; registration uses protobufImport. Capability checks still
   use actual ServiceData. Preserve host selection and ordinary CLI behavior.

No public policy choice is pending. This removes invalid generated Go without
changing the DSL or wire contract. Consumers regenerate the coherent output;
no aliases or compatibility switches are added for defective spellings.

## Models

`grpc/codegen/tla/import_aliases/FixedLocals.tla` explores all 120 orders of five
service requests and metadata names, including service suffix competition,
reqpb and a custom import. Current reservations fail ImportsVisible (217
states); reserving imports for locals but not locals for imports is insufficient.
Owned reservations satisfy ImportsVisible, MetadataSafe and eventual completion
(2,400 states). Dropping actual imports from metadata allocation independently
fails MetadataSafe (1,561 states).

`ExampleBindings.tla` demonstrates why fixing only the header is insufficient:
recomputing references fails ReferencesAgree after six states; retaining the
allocated role binding passes uniqueness/reference agreement/completion over
seven states. Formal runs use Corretto 25.0.4.1 and the pinned TLC 1.7.4 jar.
See the adjacent README for assumptions, commands and correspondence.

## Parent receipts

Go 1.27.2 darwin/arm64, local source explicitly set to this worktree.

- Expanded generated modules: ordinary build/vet passes; all 12 named collision
  cases fail compilation as above (11.29 seconds total).
- Complete ordinary example build/vet passes (2.99s); complete md example fails
  in the expected client/server metadata qualifiers (0.84s).
- Durable aggregate fixtures: p/psvcsvr build/vet passes (1.76s);
  ordinary/ordinarysvr fails at the example parameter qualifier (1.06s).
- The custom md + relocated mdsvc package fixture fails at md-qualified payload
  construction (0.50s). Existing metadata relocation behavior is retained as a
  candidate regression obligation, not claimed fixed on this parent.
- Structural scope test reproduces all 18 missing reservations; direct contracts
  reproduce md/ctx/req allocation and suffix-order failures. Ordinary identity
  passes. Production remains unchanged; these readiness tests intentionally fail.
- Separate minimal HTTP init fixture fails (0.34s); initial gRPC fixture also
  fails, including neutral example imports. Tracked as #856; not in acceptance.

Logs and exact source/generated hashes are indexed by
`/tmp/loom-691-readiness-manifest.json`. Initial probes and discarded checker
attempts are historical evidence only; final model runs are the `ready` logs.

## Bounded implementation acceptance

| Case | Obligation and expected output differences |
| --- | --- |
| Marked ordinary module and complete ordinary example | Byte-identical generated output, build/vet; relevant normal behavior remains unchanged. |
| 12 fixed-name modules | Escaped service/protobuf import qualifiers, associated transform helper spellings and metadata local bindings only; no service/protobuf declarations or wire identity changes. Build/vet all. |
| md custom relocation | Per-file custom import suffixes and metadata locals agree with the actual qualifiers; build/vet and retain request/header/trailer mappings. |
| Both aggregate fixtures | Example file aliases and references agree, including newly escaped p; whole examples build/vet. Changes outside affected gRPC imports/qualifiers require explanation. |
| Alias suffix orders | Direct four-order edge matrix; generated combined md/mdsvc/mdsvc2 and req/reqpb/reqpb2 cases once each for cross-service consumers. Declaration-order output need not be identical; each order is internally consistent. |
| Existing fixed import, custom type and metadata aliases fixtures | Preserve existing successful allocation/custom-codec/CLI and request/header/trailer behavior. Use existing direct/generated tests; no full exported corpus. |
| Host isolation | Two server hosts and differing service subsets render stable recorded aliases without mutating cached transport data. Direct check plus one complete generated example. |

Compare parent/candidate and process-isolated candidate/repeat Go/proto/example
files byte for byte. Independent repeat need not rebuild identical output.
Every intended difference must be listed; checked-in affected golden changes
must derive from regeneration. Existing framework import controls and HTTP/RPC
fixed-owner tests remain passing. Run focused gRPC and generator tests, relevant
lint, both new model pairs and existing import-alias model configurations.
Public docs and consumer guidance describe regeneration only; contributor
instructions describe the shared scope and file-binding ownership boundary.

## Final implementation receipt

The approved owners are implemented. `newTransportNameScope` replaces the
separate receiver and metadata-local reservation paths. `exampleServerImports`
retains the file's service/server/protobuf aliases in private records; every
example qualifier consumes its recorded role. The import owner was extracted
from the example-file builder to keep the changed function within required lint
limits. No production change is made to shared service identities or #856.

The final bounded comparison contains 479 artifacts across 21 cases, with 169
intended differences and no added/removed artifacts. Every candidate artifact
matches the process-isolated repeat. All protobuf source/generated Go, neutral
service output and both ordinary controls are unchanged. Changed surfaces are:

- Per-service gRPC client/server imports, qualifiers, conversion helper names,
  associated documentation wrapping and metadata variables.
- Aggregate client CLI imports and their existing recorded references.
- Fresh response-contract scaffolds' server import aliases and names derived
  from the service alias. Existing create-only scaffolds remain untouched.
- gRPC example server parameter types, server constructors and registration
  qualifiers, including the prior ordinary/ordinarysvr mismatch and preserved
  p/psvcsvr control. File-specific aliases never mutate cached transport data.

All 21 candidate modules build and vet. Eighteen parent cases reuse identical
readiness artifacts; three additional suffix/host cases reproduce expected
parent build failures. Final candidate generation/compilation took 25.104s in
gRPC tests and 21.820s in generator tests. After private-field naming and import
owner extraction, a new process regenerated all 479 bytes identically; direct
all-role and host-order checks passed on that exact source.

Affected existing encoder/decoder/type/metadata/CLI/example golden tests pass
(3.411s), with no checked-in golden changes. Existing fixed-import/custom-type,
metadata wire round-trip and independent HTTP/RPC owner controls pass (gRPC
24.690s, HTTP 1.104s, JSON-RPC 0.372s). Scoped golangci reports zero issues;
gofmt/filesize/namescope/docs checks pass with existing filesize warnings only.
All six existing and five new model configurations match their expected results.
This is ticket acceptance, not a claim that the full repository CI is green.

Final exact source, patch, artifact hashes, parent overlay, receipts and logs
are indexed in `/tmp/loom-691-final-manifest.json`. The parent overlay supplies
only the four changed Go files from the exact parent and suppresses the new
private-record-only test when generating the parent; runtime dependencies used
by generated modules are unchanged by this patch. No checkout/reset/stash or
source swapping was used. The base/repeat generate-only mode intentionally skips
redundant build/vet; candidate acceptance runs with that mode unset.
