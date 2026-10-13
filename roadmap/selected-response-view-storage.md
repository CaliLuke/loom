# Selected response field storage — #854

Status: implemented and locally validated; awaiting independent exact-diff review
and atomic integration. Parent: `fce66acb137cf7df0282bfe69446d9cceca1ceee`.
The original ordinary required-view reproduction remains unchanged. Both HTTP
and JSON-RPC failed to compile it; optional view controls did not replace it.

The shared boundary is the physical storage of a selected field before it is
detached from its enclosing declaration. Service requiredness alone does not
describe projected-view pointers or explicit Nullable storage. Target
reconstruction and source conversion now derive their native primitive pointer
fact from `expr.ServiceFieldLayout` with the actual service/view context through
`serviceFieldPrimitivePointer`. No public requiredness, default or nullable
admission policy changes.

1. [x] Reproduce the ordinary-field failure with a remotely pinned pushed module;
   preserve whole-body and optional-scalar controls. Independently review and
   file #854, separate from #847 field naming and #834 view validators.
2. [x] Reproduce the target mismatch on current parent, inspect the declaration
   owner introduced by #848, and independently review the owner/model design.
3. [x] Model current semantic-requiredness addressing and reject blanket context
   pointers. The canonical declaration-layout candidate passes. Implement the
   target repair without changing transportFieldBinding or shared context policy.
4. [x] Let the generated named-view gate expose source-side loss of enclosing
   storage. Model current value-only reads and reject unguarded dereferencing;
   independently review the presence-preserving extension before implementation.
5. [x] Use the same occurrence query in source conversion. Request construction
   keeps its existing admission guard. Response constructors preserve native
   pointer absence, convert only present dereferenced values and return the
   resulting pointer. Body references and initializer return references agree.
   Nullable/custom-presence wrappers remain values; nonprimitive paths retain
   their own conversion semantics. Do not add a parallel selected-field binding;
   root's #847 identity repair must retain this storage owner when integrated.
6. [x] Add direct target/source/context controls and extend the existing
   selected-body fixture file. Use complete service/transport generator entry
   points for a mixed HTTP/JSON-RPC module, compile/vet it and run its harness
   under race detection. Preserve required/optional/defaulted/named/custom
   scalars, nullable null/value, empty strings, native zero, arrays/bytes,
   ordinary service fields and whole-body controls. Required-view validation
   remains asserted. Mounted HTTP covers optional named nil/empty/value for both
   service and view sources; both transports' client constructors are checked.
7. [x] Compare the pushed parent and candidate byte for byte, then regenerate
   candidate in an independent process. Preserve bounded default-body controls,
   run affected body/response tests and lint, and record remaining independent
   defects without claiming they are fixed.
8. [ ] Obtain independent review of the exact final atomic diff, resolve findings,
   then let root integrate, push and close #854. No commit has been made here.

## Evidence and acceptance

The target model has eleven declaration inputs. Current service-requiredness
fails after12 states, blanket context pointers after18; layout exhausts22 states.
The source model has seven initial states. Current root conversion fails after8
states, naive dereferencing after9; presence-preserving conversion exhausts14.
All negative checks fail their named invariant, not parsing. These models trust
canonical layout and transform facts as inputs; they do not prove arbitrary
codecs, validation, field identity or JSON-RPC envelope projection. Sources,
configurations, assumptions and commands live beside the HTTP owner in
`http/codegen/tla/response_field_storage/`.

Final direct/generator race gates pass (`http/codegen`1.834s,
`codegen/generator`7.262s). The generated harness itself builds/vets the complete
module and runs under race detection. The existing body/response boundary tests
pass (HTTP72.528s, JSON-RPC1.754s); focused owner controls pass0.352s. Pinned lint
covers HTTP codegen, generator tests and the handwritten HTTP testdata package:
zero issues. This is bounded ticket acceptance, not a claim that all framework
CI jobs are green.

The exact comparison includes16 result shapes in both transports and the
existing SelectedBodyDefaultsDSL. Storage artifacts:31 files per generation;
only the two client types.go files and two server types.go files change, plus
loom.json's expected released-version versus `(devel)` provenance. Client
changes address required/defaulted/native/named/custom viewed values and stop
addressing already-nullable wrappers. Server changes preserve pointer-shaped
named source absence and dereference only present values. Default fixture:
17 files per generation; only loom.json version provenance differs. Design
digests, OpenAPI JSON/YAML, other source and request-body artifacts are unchanged.
All48 candidate artifacts match independent repeated generation exactly.
The parent mixed module fails at the expected client and server assignments;
the candidate complete module builds/vets. The existing default fixture also
builds/vets on the parent. Its candidate already passed generated runtime/vet
coverage. The checked-in ticktock designs contain no selected Body mapping, so
this selection-only change does not invalidate their generated fixtures.

Comparison sources are the pushed parent module
`v1.10.0-alpha.9.0.20261013090738-fce66acb137c` and the local candidate under
Go1.27.2 with jsonv2. Exact inputs are under `/tmp/loom-854-exact-{base,candidate,repeat}`
and `/tmp/loom-854-defaults-{base,candidate,repeat}`. Hashes and enumerated changes
are `/tmp/loom-854-final-artifact-hashes.json`,
`/tmp/loom-854-final-artifact-summary.json` and
`/tmp/loom-854-final-artifacts.patch`. An earlier outer-Go-overlay comparison was
discarded because the real CLI deliberately clears GOFLAGS for its generator
build; it is not acceptance evidence. Raw temporary artifacts stay outside Git
and are removed after final review with a compact receipt.

## Independent residuals

- #735: two selected named results sharing one body alias currently reuse one
  server constructor despite different source-view parameter types. Both HTTP
  and JSON-RPC fail. Preserve `/tmp/loom-854-shared-alias-repro-dsl.go` and
  `/tmp/loom-854-fixture-final-build.log` as additional constructor-identity
  acceptance evidence. The final storage fixture uses distinct aliases while
  retaining required, optional and nil/empty/value controls. The original
  shared-alias aggregate is not claimed to compile after #854.
- The manual renderJSONRPCModule/renderMixedHTTPJSONRPCModule helpers omit
  ViewsFile. Root is repairing their shared assembly separately. The durable
  regression uses existing complete generator entry points; no caller-specific
  Views workaround was added. Failure: `/tmp/loom-854-focused.log`.
- #853 still owns mounted JSON-RPC success-envelope projection for selected
  scalar bodies. Constructor correctness and HTTP mounted results here do not
  imply that separate JSON-RPC handler behavior is fixed.
