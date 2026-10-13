# Shared legal Go package identity (#856)

Status: implemented and validated; independent final review pending. Parent: `9a4f1c2f3414ec7667c5ed18bbf9e4bec6e8d1f7`.

The existing generated-module helper reproduces four compiler failures: a service
named `init` under HTTP, gRPC and JSON-RPC, and an API named `init` with an
ordinary HTTP service. The ordinary API/service control builds and vets. The
failures are illegal import identifiers, not local-variable shadowing. The
helper runs actual generation, example generation, tidy and compilation; the
readiness overlay adds only its test source and does not replace production.

## Owner and consumers

`codegen/service.PackageBaseName` already derives the common Go package/import
base from authored API and service names. At the parent it emitted `init`, which Go
cannot bind as an import identifier. Make that base legal (`init` becomes
`init_`) before allocation by existing NameScopes. Keep `Goify`, ServiceDir,
authored names and protocol identities unchanged. Do not add transport reserved
names or another allocator.

- `servicePackageName`, via `newServiceScopes`, allocates Data.PkgName and the
  derived view package identity; service declarations and neutral examples use it.
- HTTP `newServiceImportAliases` starts from service.PackageName. JSON-RPC
  consumes HTTP service analysis. gRPC `newImportAliases` starts from the same
  shared service identity and retains independent protobuf import identities.
- `ExampleServiceFiles` derives the root API package from PackageBaseName and
  its NameScope. `codegen/example`, HTTP and gRPC example server/client helpers
  use the same base; JSON-RPC fallback uses it too.
- Existing AliasClashingImports and local NameScopes remain responsible for
  file-specific import collisions. Directory and wire-name normalization stay
  independent; no compatibility alias is needed for uncompilable output.

Reserving `init` only in newServiceNameScope is insufficient: API example
packages bypass it. Changing generic Goify would unnecessarily alter field,
method and directory names. No public policy choice is unresolved: accepted
service/API names must produce compilable code without losing authored identity.

## Bounded implementation and acceptance

1. [x] Add direct regressions to existing package-name tests for init/case variants,
   neutral PackageName versus analyzed Data.PkgName agreement, stable paths and
   existing NameScope collision allocation. Preserve main, Unicode and normal
   controls. Add the compiler regression through existing example-module helpers.
2. [x] Correct only the shared package-base owner. Update its GoDoc to separate legal
   Go identity from generated directory identity; remove no existing independent
   file-level allocation. There is no superseded fallback or duplicate code path.
3. [x] Compile generated service/transport/example modules for HTTP, gRPC, JSON-RPC,
   API-only init, API+service init (allocation crossing), and ordinary control.
   Compare parent/candidate output bytes; enumerate intended identifier changes,
   keep authored service/route/protobuf/spec identities, and require ordinary
   output unchanged. Repeat generation in a separate process, not repeated builds
   of identical output. No unrelated corpus or runtime transport matrix needed:
   this correction changes identifiers, not codecs or transport behavior.
4. [x] Run affected direct tests and scoped lint. Reuse existing NameScope allocation
   controls; no new formal model is needed for one pure base-name legalization
   rule with unchanged allocation/ordering algorithms. Preserve exact diff and
   evidence for independent review before any commit.
5. [ ] Obtain independent final review and deliver.

Existing consumer applications using affected names must regenerate; no DSL
change or application workaround is required. The example API boundary is the
same shared identity owner and belongs in this repair rather than a second
transport-specific fix.

## Validation receipt

Parent direct tests fail for three case variants and the shared analysis/allocation
identity; parent generated tests fail all five init cases and pass the ordinary
control (7.731s). The added API+service case proves that legalizing only transport
aliases would miss the root example namespace. No production overlay was used:
the exact parent owner was still unchanged when parent capture ran.

Candidate direct/allocator/example-alias controls pass under race (1.683s).
All six actual generated modules build and vet (20.549s). Existing generator
helpers own generation, example output and compilation. A test-only overlay
copies the already-rendered output before compilation, without changing generation
or production; a second process repeats generation and skips redundant builds.
All123 artifacts match that repeat exactly. The ordinary specimen's21 artifacts,
all OpenAPI/protobuf files and all generated paths remain unchanged.

Exactly54 of123 artifacts change. Every changed byte is explained by the existing
identifier derivations: `init`→`init_`, `initapi`→`init_api`, `initsvr`→`init_svr`,
`initjssvr`→`init_jssvr`, and `initc`→`init_c`. A bounded byte comparison reverses
only those identifier substitutions and recovers every parent file exactly.
The full per-path list is in `/tmp/loom-856-artifact-comparison.json`, with all
three-stage hashes in `/tmp/loom-856-artifact-hashes.json`. The output diff is
`/tmp/loom-856-artifact-output.patch`. The existing checked-in HTTP/JSON-RPC
transport fixtures have no service/API name normalizing to init, so no fixture
regeneration is required for this base-name-only change.

Scoped golangci-lint for service/generator reports zero issues; diffcheck and
formatting pass. No broad corpus, new model or full-CI claim is made. Evidence
is `/tmp/loom-856-{direct-parent,direct-candidate,generated-parent,generated-candidate,generated-repeat,lint}.log`.
