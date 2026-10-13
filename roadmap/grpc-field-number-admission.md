# Protobuf field number admission (#703)

Readiness parent: `e631f0a89`, including delivered #701. Initial readiness checkpoint; production edits began only after the corrected
703-R1 readiness received independent approval.

1. [x] Read the live report, DSL contract, selected-message owner and consumers.
2. [x] Reproduce invalid domains, normalized collisions and derived union edges;
   retain valid numeric boundaries, message scopes and excluded mappings.
3. [x] Model selection, derivation, normalization and admission alternatives.
4. [x] Freeze bounded acceptance and owner proposal for independent review.
5. [x] After approval, implement shared expression admission and aligned emitted-number consumers.
6. [x] Compare valid generated proto/Go artifacts, compile descriptors, exercise
   actual CLI rejection, and freeze exact final diff for fresh review.
7. [ ] Parent integrates one atomic reviewed commit, pushes and closes #703.

## Contract and owner

[Protobuf's field-number contract](https://protobuf.dev/programming-guides/proto3/#assigning)
requires emitted fields to use distinct numbers within each message, in
1..536870911 excluding 19000..19999. Loom's `Field` is general metadata sugar,
not a gRPC-only declaration. Admission belongs after gRPC has selected message
fields, not in `dsl.Field` or a generator panic recovery layer.

`validateMessageFieldTags` already selects request/stream/result/error messages,
excludes metadata and implicit credentials, and respects opaque protobuf mappings.
`validateRPCMessageTags` traverses reachable messages with cycle protection.
`validateRPCObjectTags` is the local presence/uniqueness owner. It currently uses
raw strings as identity; generator checks use parsed uint64, so `"01"` and `1`
can evade design admission while colliding in generated messages.

The intended owner admits numeric syntax/domain and compares normalized numeric
identity for selected emitted fields. Keep authored rpc:tag metadata unchanged.
Preserve decimal-string input accepted by the existing parser, including leading
zeros; do not reject Go argument kinds whose formatted value is an accepted integer.
Do not add raw field-tag checks to unrelated HTTP-only or unused declarations.

## Union derivation

`UnionFieldTags` owns explicit branch precedence and consecutive inherited tags;
three generator sites consume it too. Validate the actual emitted branch numbers.
An explicit branch number replaces the inherited slot; the enclosing seed is not
itself a protobuf field. Do not reject a harmless unused seed when all branch
numbers are explicitly valid. Numeric derivation must not wrap before admission:
an overflowing seed plus index can currently become a legal small number if early
branches have explicit numbers. Preserve a nonwrapping mathematical result (or an
explicit derivation error) at this shared owner, then apply the same numeric domain
admission. Never recompute tags separately in individual protobuf renderers.

## Alternatives and scope

- DSL argument filtering is too early and rejects non-gRPC metadata usage.
- Bounds-only admission leaves raw-string identity collisions unresolved.
- Generator-only checks fail late and do not protect the evaluated design.
- A global tag registry rejects reuse across distinct messages.
- Checking only a constructor union's base misses a reserved/over-limit later slot.
- Selected: shared emitted-number admission with local numeric identity, deriving
  union branches without overflow before checking the selected numbers.

#701 union logical identity and #702 map-key comparability remain separate.
This repair does not change protobuf field naming, request metadata keys, message
selection, security policy or codec semantics.

## Bounded acceptance

Direct cases cover zero, negative, nonnumeric, reserved endpoints, maximum+1,
uint64 limits/overflow, numeric spellings and duplicates; union derived reserved
and maximum crossings, explicit overrides and overflow masked by explicit slots.
Control numeric boundaries, independent message scopes, recursive/nested messages,
request/response/stream/error placement, metadata exclusions and opaque mappings.
An actual CLI invalid-number witness must fail at evaluation with source context,
not panic or reach protoc. Representative valid numeric-boundary and mixed-union
fixtures must generate identical parent/candidate output and compile with protoc;
repeat candidate generation in a separate process. Keep existing tag and union
fixtures unchanged. No full exported-design audit.

## Readiness evidence

The current owner admits 25 invalid cases: 17 direct number/union cases and 8
selected message-role cases. All 25 corresponding numeric/selection/placement
controls pass. Four actual CLI cases reach three generator panics (zero,
nonnumeric and equivalent numeric identity) and one protoc error (reserved
number), rather than evaluation admission. The final-assertion tests are red
at the intended boundary; original direct run 0.603s, CLI run 11.244s. After advancing to delivered
`e631f0a89`, the same cases reproduce in 0.148s and 11.357s; scoped expr/cmd
Go lint reports zero issues.

The initial bounded model had 14 designs and 56 owned states. Current, raw-identity
bounds-only, wrapped arithmetic, preselection admission and global identity
alternatives all produce the expected counterexamples. See the owning model's
README for assumptions and source/test correspondence.

Implementation acceptance will rerun these direct and actual CLI checks, the
existing union tag/nested-message tests and affected expr/dsl packages. Generate
one existing constructor-union specimen and one boundary/explicit-override
specimen at parent and candidate, compare all source bytes, run protoc and build
and vet each candidate module, and repeat generation in a separate process.
Previously generatable valid fixtures must remain byte-identical. Fully explicit
branches with malformed or overflowing unused seeds are newly generatable
controls and have no successful parent artifact set to compare. Their emitted
proto and generated Go must match equivalent numeric-seed controls (generation
manifest digests may reflect the different authored metadata). Generated service types and
wire numbers must retain the authored values; reject invalid designs instead of
repairing their numbers. No OpenAPI schema behavior is changed.

The implementation will retain the current decimal `ParseUint` syntax for
ordinary tags and union seeds. Numeric admission uses one parsed number for
range and per-message identity; diagnostics retain authored text and field
context. Shared union derivation will retain an exact decimal sum rather than
machine-wrap a seed plus branch index. Standard-library integer arithmetic can
represent this bounded sum without a new dependency or public API change.
An out-of-domain derived sum then fails the same admission as an explicit tag.
An unused malformed seed remains harmless when all emitted branches override it.

## Readiness review 703-R1: complete the emitted-number consumer boundary

The independent reviewer found that `writeProtoFieldShapes` still calls `rpcTag`
for every object member, including the non-emitted container of a oneof. An
unused `"abc"` seed with explicit branches 1 and 2 passes evaluation but panics
while registering message identity. Different legal unused seeds also produce
different shape identities despite emitting identical numbers. This contradicts
the accepted explicit-override contract and is part of this atomic owner repair.

All number consumers are inventoried: `protoBufObjectFieldLine` already sends
union fields directly to `protoBufOneofDef`; it parses only emitted branch tags.
`checkMessageFields` also separates ordinary fields from emitted union branches.
`writeProtoFieldShapes` must use the same distinction. Retain field name and
requiredness identity, record numeric tags only for emitted ordinary fields or
union branches, and preserve recursion/reference tracking. Do not parse or hash
an unused seed, rewrite authored metadata, or compensate in individual renderers.

Added direct shape tests reproduce two unused-seed parse panics and one numeric
seed identity mismatch, with same-seed and changed-emitted-number controls.
Actual generation reproduces the two panics; the numeric-seed control passes
protoc, full module build and vet. Original 25 admission reds / 25 controls and the
four invalid CLI rejection witnesses remain unchanged evidence. The model now
includes publication: container parsing fails PublishedWhenAccepted at 67 states;
owned passes 70 states. All six prior strategies were rerun after adding the phase.

The corrected implementation scope is `expr/grpc_field_tags.go`, shared union
arithmetic in `expr/attribute_required.go`, and the misaligned consumer in
`grpc/codegen/service_data_analysis.go`, plus tests/model/docs. R1 adds no policy
change. Final acceptance includes both newly generatable unused-seed variants,
message identity equivalence, and unchanged output for previously valid numeric
boundaries and ordinary constructor unions. No production change existed at that readiness checkpoint.

## Implementation and bounded acceptance

The shared claim function now parses each emitted tag once for protobuf range
and per-message numeric identity. Union derivation emits exact decimal sums;
message shape identity omits the non-emitted oneof container number. No metadata
is rewritten. The selected-message traversal and opaque/excluded fields retain
their existing owner.

Focused expression, union-tag and constructor/nested protobuf tests pass;
complete expr/dsl packages pass. Actual CLI checks now reject invalid numbers
at evaluation with source location. All three explicit-branch unused-seed modules
generate, compile and vet; every generated Go/proto byte matches the numeric-seed
control. Existing constructor-oneof proto golden remains unchanged.

The bounded output matrix uses the existing ConstructorOneOfFieldDSL and a
boundary/mixed-override specimen. Parent production is supplied through an exact
three-file Go overlay in this worktree; generator runtime and test harness
sources are unchanged. Each parent/candidate module generates through the actual
CLI/protoc, builds and vets. Independent-process repetition is generation-only.
No transport runtime or HTTP/OpenAPI path changes, so unrelated ticktock fixtures
are outside this field-number acceptance surface.

Final evidence: focused shared admission/shape and existing proto checks pass
(expr 0.802s, gRPC codegen 1.591s); actual invalid CLI plus all unused-seed cases
pass in 28.480s. The strengthened unused-seed Go/proto exact-byte comparison,
with generation/build/vet for each module, passes in 12.419s. Complete expr/dsl
packages pass (1.866s / 0.738s); scoped Go lint reports zero issues.
All 30 artifacts in each of parent, candidate and independent repeat match
exactly, including generation manifests. Both previously valid specimens pass
CLI/protoc, module build and vet. No checked-in generated artifact changes are
intended. The parent overlay contains exact production blobs from `e631f0a89`;
input designs, source mode and toolchain are recorded in the final packet.
Existing named-union direct/protoc checks and generated conversion round-trip
also pass in 4.168s, preserving named occurrence normalization and field reuse.
