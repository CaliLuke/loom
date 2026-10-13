# Protobuf collection element admission — #740

Status: #754 delivered; #740 implementation and bounded acceptance complete. Exact final packet is frozen for independent review; no #740 commit or push yet.
Implementation parent: `56c46446f99c510788931c597e78836244eee838`. Historical readiness parent: `fce66acb137cf7df0282bfe69446d9cceca1ceee`. Worktree:
`/Users/luca/.codex/worktrees/default-application-parent/loom`.

## Sequence

1. [x] Preserve #848 source packet and compact evidence; reuse an inactive managed parent checkout.
2. [x] Reproduce the recorded six conversion panics and four normal/empty controls against the current parent.
3. [x] Establish accepted element nullability and protobuf wire behavior from the DSL, docs, actual conversion/admission owners, and binary round trips.
4. [x] Compare structural candidates in a bounded model, including inline versus recursive helper conversion and client payload versus server result boundaries.
5. [x] Record the shared owner, intended behavior, rejected alternatives, normal/edge/rejected matrix and bounded implementation acceptance.
6. [x] Deliver #754 helper-local fallibility as an independently reviewed atomic prerequisite.
7. [x] Re-review the changed #740 readiness around that owner; approval includes the repaired helper ABI.
8. [x] Reconfirm current-parent nil-element failures after #754; add direct owner failures.
9. [x] Implement source-occurrence admission and extend helper-local fallibility.
10. [x] Convert runtime witnesses to rejection assertions; add mixed helper/physical carrier controls.
11. [x] Complete bounded generated comparison, compilation, wire/runtime and lint checks.
12. [x] Freeze exact final diff and evidence for fresh independent review.
13. [ ] Obtain final approval, integrate as one atomic commit, push and close #740.

The accepted empty/absence contract does not by itself make nil object collection
elements valid. Do not add per-loop guards before establishing admission and the
protobuf wire consequences. No user decision is requested unless the existing
public contract leaves a genuine semantic choice unresolved.

## Contract and current-parent evidence

`docs/dsl-reference.md` declares typed array elements and map values non-null by
default, separately from containing-field requiredness. `docs/grpc-guide.md`
rejects concrete Nullable at all gRPC positions, and even nullable Any at
collection positions. Any values retain their native JSON null domain. An
ordinary optional object field may remain nil; nil and empty containing
collections are not nil object elements. No unresolved product decision blocks
this repair.

At the original readiness checkpoint, the reused parent had no production edits. Go 1.27.2 with GOEXPERIMENT=jsonv2 and
LOOM_DIR explicitly pointing to this checkout produced:

- `TestNilCollectionConversionReadiness`: generation/build/vet pass, then six
  expected request/response conversion panics (outer object array, leaf array,
  nested map value). Four populated/empty controls pass; package 3.028s.
- `TestNilElementWireReadiness`: generation/build/vet pass, then four expected
  inverse conversion panics for native protobuf nil array/map objects. Binary
  probes prove nil repeated/map messages become nonnil empty messages. Recursive
  conversion preserves nil before wire loss; an empty object declaration creates
  a nonnil object without inspecting the nil source at all. Five optional-field,
  empty/nonempty-object and nested collection controls pass; package 4.267s.
- Existing generated Any conversion/null controls and constructor error paths
  pass (6.381s). Existing collection wrapper contracts, including present empty
  wrappers, authored length rejection and required wrapper rejection, pass
  (4.293s). Existing gRPC nullable design-admission tests pass (0.405s).

Logs are `/tmp/loom-740-current-parent-red.log`,
`/tmp/loom-740-wire-readiness.log`, `/tmp/loom-740-current-controls.log`,
`/tmp/loom-740-wrapper-controls.log`, and `/tmp/loom-740-rejected-contract.log`.
Those readiness tests were intentionally red at that checkpoint. They now require
conversion errors and zero results, preserving the negative logs as historical evidence.

## Shared owner and proposed change

The root cause is lost occurrence admission at the protobuf conversion boundary.
`transformElement` selects inline conversion or a recursive object helper. Those
paths disagree about nil; the array/map renderers normalize wrapper attributes
before the shared element call, losing the physical source message distinction.
A pointer guard in each loop would avoid some panics but retain silent corruption.

Introduce one internal collection-element policy in the protobuf conversion
owner. It must retain the physical source occurrence before wrapper unwrapping,
distinguish an object/message element from a native collection value or Any, and
be consumed by both fallibility analysis and element conversion. Check the
source before field access, empty-object allocation or recursive helper dispatch.
Keep optional object-field helper behavior unchanged; nil rejection belongs to
the collection occurrence, not a globally reused object helper.

The conversion result must expose its fallibility alongside code/helpers so
`InitData.ErrorAware`, helper parameters/calls and emitted constructors derive
from the same decision. Remove the duplicated Any-only eligibility at
`protoBufTransform` and `buildInitData`; retain Any conversion failures. Reuse
existing error propagation in unary encoders/decoders, stream send/receive,
client CLI conversions and declared-error details. Do not add transport-local
nil scanners, panic recovery or a second general-purpose validator.

For invalid native object/message elements, return a regular conversion error
and no usable partial result. No request or stream item may be sent on failure.
For a server-produced invalid result/error detail, preserve existing conversion
fault classification and safe disclosure; do not report it as a client-authored
validation failure. Confirm exact status/problem behavior in the mounted test
against the existing Any conversion error path. This ticket does not redefine
all gRPC status codes.

Constructors whose graphs can encounter these invalid source elements will now
return `(value, error)` just as Any constructors already do. Generated adapters
and CLIs derive the signature; callers invoking constructors directly must handle
the error after regeneration. No protobuf schema or service interface change is
intended. Regenerate clients and servers; no compatibility shim.

## Alternatives tested

The bounded model under `grpc/codegen/tla/element_admission` reproduces inline
panics (433 states), guard-only invalid emission (865), and blanket nil rejection
of valid controls (901). The occurrence owner passes all three invariants over
1296 states. Post-wire validation cannot distinguish a normalized nil from an
actual valid empty message. Current recursive and empty-declaration probes prove
that information loss in generated code. The model assumes correct physical
classification and error propagation; generated compilation/runtime tests own
those obligations, not the model.

## Bounded implementation acceptance

| Obligation | Selected cases |
| --- | --- |
| Shared classifier and signatures | Direct source/target occurrence tests: ordinary named and anonymous object, empty object, recursive object, nested/named collection wrapper, native slice/map, scalar zero, Any; optional field versus array/map element; helper and constructor fallibility agree |
| Actual nil rejection | Original three nil positions in client request/server response; inverse native protobuf constructors; recursive helper and empty object; assert error and zero result, not merely absence of panic |
| Accepted values | Populated object, nonnil empty object, containing absent/empty collections, optional object field, current nested collection controls, scalar zero and Any null; required child fields and MinLength still reject independently |
| Real consumer boundary | One generated unary/bidirectional service plus declared error: invalid client input does not invoke/send; invalid server result/detail has safe fault behavior and emits no invalid body; stream failure blocks that send but preserves prior valid sends |
| Views and shared helpers | Reuse one selected-view result and a graph sharing an object between optional-field and element positions, in both declaration orders; no helper-policy contamination |
| Native protobuf source | Reject nil object and wrapper elements before unwrapping/dereference in direct converters; valid empty wrapper still produces empty contents; actual wire empty messages retain their ordinary validation semantics |
| Compilation and determinism | Bounded parent/candidate/repeat artifacts for the nil-item specimen, existing inline-object fixture, one recursive nested-map specimen and AnyErrorDSL; build/vet each distinct parent/candidate module; exact-byte independent repeat; list every changed artifact |
| Unchanged contracts | Existing nullable design rejection, Any/error paths, collection wrapper and protobuf scalar-default controls; affected direct tests and intended goldens only |

Select one recursive nested-map case, not the entire exported-design corpus.
Expected artifact changes are constructor/helper signatures, admission branches
and derived error handling at generated callers. Protobuf schemas, service type
layout and valid-value wire bytes should remain unchanged. If implementation
reveals an unmodeled physical wrapper or helper-sharing case, update readiness
and the model before depending on that design.

## Adjacent finding, separate scope

Native `ValidateEchoRequest` rejects a nil array object element but currently
accepts a nil map object value. `renderMapValidationCode` / `renderMapValidation`
in shared codegen lack the non-null object check used for arrays; the check is
currently limited to JSON presence wrappers. This is distinct validator ownership.
The wire probe records the behavior without claiming it correct. #740 must reject
invalid map object values at conversion regardless, but must not fold a shared
validator repair into its atomic change. A separate report can reuse the same
minimal design and `ValidateEchoRequest(&pb.EchoRequest{ByKey:
map[string]*pb.Leaf{"k": nil}})` failing admission assertion.

## Recovery

#848 remains frozen in its original worktree and exact final patch. Its cleanup
receipt is `/tmp/loom-848-cleanup-receipt.json`: 58 inactive generated/checker/log
paths removed after process/open-file checks, with compact evidence preserved.
Five untracked #835 parent inputs were verified byte-identical to delivered
sources before reuse; `/tmp/loom-740-worktree-reuse-receipt.json` records hashes.
At the original recovery checkpoint only readiness tests, this plan and model
sources/configurations had been added. Production implementation began after the
subsequent owner approvals below; no #740 commit or push has occurred.

## Review finding and revised dependency (#754)

Independent reviewer `review_json_normalization` found a medium blocker in the
proposed fallibility design. Helpers are shared by source/target pair while their
`ErrorAware` flag and calls inherit root-transform fallibility. A pure Leaf used
inside both a plain root and a root with an unrelated Any field fails compilation
in either declaration order (too many versus too few arguments). The review's
exact reproducer and log are `/tmp/loom-740-review-helper-abi_test.go` and
`/tmp/loom-740-review-helper-abi.log`. This is the existing reported #754 root
cause, not a new issue.

#754 must land first as one atomic fix: helper declaration, helper body, nested
helper calls and every external call derive error parameters from the helper's
own conversion contract. Root constructors derive fallibility from their own
conversion graph. A fallible caller may invoke an infallible helper without an
error argument. An infallible caller cannot invoke a fallible helper: transitive
conversion analysis must mark that caller fallible. Declaration order must not
change signatures or error propagation. Do not encode root fallibility into
helper names or make all pure conversions fallible.

#740 then adds element-admission failure to that shared conversion analysis.
An object's own nil admission remains occurrence-specific: a plain Leaf helper
can preserve optional-field nil without acquiring an error parameter solely
because some caller uses Leaf as an array element. That caller checks the element
before helper dispatch and is fallible. A Node helper whose body itself contains
object collections is fallible because those nested occurrences can fail. Both
cases require mixed-parent and both-order generation/compile controls. The
current 1296-state element model establishes admission/order only; it explicitly
does not establish helper ABI consistency. The #754 helper-contract model and
compiled witnesses provide that missing obligation.

The original frozen manifest remains `/tmp/loom-740-readiness-manifest.json`;
its initial plan bytes are retained at
`/tmp/loom-740-readiness-plan-pre-helper-review.md`. Revised scope/evidence will
be frozen separately, without overwriting the reviewed negative evidence.

## Implementation checkpoint after #754

#754 was delivered as `56c46446f`. The worktree advanced without forbidden Git
operations after archiving its exact reviewed source packet; receipt:
`/tmp/loom-754-advance-receipt.json`. #740 inputs were retained. On this parent,
the same six outgoing and four inverse nil-object panics reproduce (5.158s),
while normal controls still pass: `/tmp/loom-740-after-754-red.log`. Direct owner
cases require object-array, object-map and incoming wrapper conversions to be
fallible; outgoing native nested scalar collections remain infallible.

Implementation sequence:

1. Extend `conversionCanFail` with the existing source AttributeContext and one
   collection-element admission predicate. Examine the physical source graph:
   target protobuf wrappers alone must not make outgoing native scalar collections
   fallible. Any conversion fallibility remains unchanged.
2. Preserve source element attributes before array/map wrapper normalization;
   `transformElement` emits admission before inline conversion or helper dispatch.
   The source policy includes protobuf union message pointers while service union
   collection elements are values. Empty-object loops retain the source variable
   for admission. Optional object helper calls keep their existing nil behavior.
3. Return regular conversion errors through the established error channel; the
   generated root returns its zero result. Do not add a transport policy or shared
   validator repair. Supply the generated standard-library error import centrally.
4. Check ordinary and internally fallible recursive helpers in both method orders,
   incoming wrappers, Any/native collection/zero controls and a selected view.
   Then run the recorded bounded acceptance and freeze the final packet.

This is a physical-classifier refinement within the approved model's nil-message
category, not a new public rule or altered state machine. Model assumptions stay
explicit: actual Go carrier classification and generated error propagation are
established by direct/generated tests. #754 resolved the sole medium readiness
finding; its separate helper model supplies the transitive ABI obligation.

## Historical fresh owner checkpoint

After reconfirming parent failures, the provisional shared owner passed the direct
three-red regression and generated rejection witnesses. The source-context
refinement was frozen for the requested fresh owner review before further
production work. The review packet is `/tmp/loom-740-owner-readiness-manifest.json`.
This is not final acceptance or commit authorization.

- Source occurrence classifier: `elementNeedsAdmission` consumes the physical
  attribute and source context. Native service union values, arrays, maps, bytes
  and Any receive no nil-message check; object elements and protobuf union/message
  carriers do. Source wrappers are checked before `Field` access, even though
  normalized conversion attributes describe the underlying collection.
- Fallibility: `conversionCanFail` retains Any analysis on the pair and adds
  source-graph occurrence analysis. A target wrapper alone does not make outgoing
  native nested scalar collections fallible. The #754 detached helper context and
  each called helper use their own source context and graph.
- Emission: one `transformElement` admission branch precedes inline conversion,
  recursive helper dispatch and empty-object allocation. It records a regular
  conversion error and skips that element. Existing root constructors return a
  zero result after any recorded error; no partially converted result is usable.
- Reconfirmation: three direct owner reds and all original ten panics were
  observed at parent. Provisional direct checks, ten helper compilation cases,
  nil rejection/control modules and five artifact builds/vets pass. Exact logs
  are in the owner manifest. Golden/harness changes caused by new fallibility
  signatures remain unfinished and are not final evidence.
- Model correspondence is unchanged: the nil-message category describes a
  physical source pointer, not a logical collection or a service union value.
  The 1296-state admission model assumes that classifier; the new direct carrier
  table supplies the correspondence check. #754's transitive helper model remains
  a separate prerequisite; no new model invariant or public rule is introduced.

At that checkpoint, the mounted transport probe had not passed final acceptance: invalid client input
already returns an error without service invocation, but the initial test wrongly
expected every server conversion error to have the same message as the existing
Any failure. The current Any path exposes its raw parse diagnostic; nil admission
returns the static `gRPC conversion: nil message collection element` diagnostic.
This is recorded as an adjacent existing disclosure concern, not repaired in #740.
Final transport assertions must preserve nil rejection, no invalid message writes,
normal preceding sends and the actual intended safe static diagnostic/code
contract, without treating the existing Any diagnostic as a new desired policy.

## Approved owner and final acceptance progress

Root independently approved the source-context owner after checking the exact
21-source/12-evidence manifest and rerunning direct and original nil-rejection
witnesses (8.871s). Model/configuration sources are unchanged; their recorded
1296-state owner result remains applicable, with direct carrier tests proving the
refined source-context correspondence.

The mounted transport harness now passes, asserting Unknown plus a static nil
conversion diagnostic for invalid unary results and `internal server error` for
invalid error details/stream results. Invalid client input never invokes the
service, and a valid prior stream item remains observable. Existing Any disclosure
is recorded separately; no status-message policy was changed.

New fallible constructors require mechanical migration of existing test harness
calls. Every adaptation keeps its original semantic assertions and explicitly
checks each conversion error. Selected wrapper, aliases, recursive containers,
inline objects, selected-view and CLI tests cover these newly affected calls.
These checks are required signature migration, not a whole exported-design audit.
The CLI renderer has a separate import list; its compile regression exposed and
fixed the required standard `errors` import there, without a second admission rule.

Final artifact matrix includes six specimens: nil-items, physical-carriers,
inline, recursive-map, Any and transport, including client CLI rendering. Parent
and candidate modules compile/vet; independent candidate generation is compared
byte for byte. Intended differences are error imports, admission branches,
constructor/helper signatures and generated error propagation. Proto schemas,
service interfaces and the Any-only specimen must remain byte-identical.

## Final validation receipt

Source parent: `56c46446f99c510788931c597e78836244eee838`; Go 1.27.2,
`GOEXPERIMENT=jsonv2`, local-source `LOOM_DIR` explicitly set to this worktree.
No dependency, protobuf descriptor or service signature changes.

- Direct owner, physical carriers, original nil rejection, both-order shared
  helpers, mounted transport, Any/local Nullable and scalar-default controls:
  pass (33.118s), `/tmp/loom-740-final-regressions.log`.
- All changed constructor-call fixtures, including aliases, recursive containers,
  optional/named/nested unions, mapped fields, defaults, inline objects, selected
  views and wrapper contract ordering: pass (97.168s),
  `/tmp/loom-740-final-migrations.log`. Original assertions remain; each newly
  fallible conversion is checked explicitly.
- CLI generated runtime plus exact CLI goldens: pass (12.920s),
  `/tmp/loom-740-cli-final.log`. Import inventory, errors-named service including
  CLI build/vet, adapter goldens and nested message declarations: pass (5.235s),
  `/tmp/loom-740-import-adapter-final.log`.
- Six specimen parent build/vet pass (7.209s); candidate build/vet pass (8.184s).
  Independent repeat generation passes (2.119s). The comparison covers 78 files,
  with 27 intended changes: type/CLI conversion error imports and guards,
  constructor/helper signatures and derived adapter/stream error propagation.
  All 78 repeated candidate files match exactly; descriptors, service files and
  the Any-only specimen are unchanged. File-by-file hashes and exact differences:
  `/tmp/loom-740-artifact-comparison.json`, `/tmp/loom-740-generated.diff`.
- 21 affected checked-in goldens regenerated through their owning tests: eight
  conversion snippets, four client/server type files, three CLI files and six
  encoder/decoder adapters. No generated fixture source was edited by hand.
- Scoped `golangci-lint run ./grpc/codegen/...`: zero issues. Repository format,
  size, JSON-v2, NameScope and documentation checks pass; pre-existing size warnings
  remain below the hard limit. `git diff --check` passes.
- TLA+ source/configurations unchanged from the approved owner checkpoint;
  recorded owner1296 success and three counterexamples reused. Generated union
  collections remain unsupported; only physical classification is directly
  tested. Existing Any disclosure and shared native map-element validation remain
  separate findings. No expanded exported-design audit was run.

The full import owner inventory is `grpcTypeImports` for both type files and
`payloadBuilders` for CLI conversion bodies. Other generated adapters call their
constructors. Existing `transportGeneratedImportNames` already reserves `errors`;
no parallel alias policy was introduced. Exact source/evidence manifest and patch
are `/tmp/loom-740-final-manifest.json` and `/tmp/loom-740-final.patch`.
