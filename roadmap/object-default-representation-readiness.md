# Object default representation readiness (#693)

Parent: `30cabf6338a6e1aa0a98b5d4f89e63f2df76d27d`.
Worktree: `/Users/luca/.codex/worktrees/object-default-representation/loom`.
Status: readiness independently approved; implementation and bounded acceptance
complete in this isolated worktree; exact-diff review and integration pending.

1. [x] Reproduce the original accepted object default generating an invalid map-to-
   struct assignment. Check anonymous/named objects and collection elements,
   alongside scalar and typed collection defaults already repaired by #692.
2. [x] Reconcile the original unknown-key allegation with the current shared value
   admission owner delivered by #681/#685. Distinguish already-correct design
   rejection from the remaining target representation defect.
3. [x] Trace the shared literal renderer and its Native/Optional/Nullable callers.
   Derive object members from resolved values and target context, including field
   aliases, primitive pointers, nested objects and collections. Do not add a
   second map/struct interpretation policy or transport-specific literals.
4. [x] Check existing value projection proofs and default-presence tests; model any
   additional representation/ordering obligation before implementation. Preserve
   explicit zero/empty/null and absent-value boundaries. Scalar absence policy
   changes remain a separate M7 task.
5. [x] Obtain scoped independent readiness review with concrete red cases, accepted
   design and bounded acceptance: direct admission/representation tests, compiled
   generated Native/Optional/Nullable conversions, representative HTTP/JSON-RPC/
   gRPC output/build checks, independent-process byte comparison and affected
   model/proof checks. Include valid and rejected designs and ordinary controls.
6. [x] Implement the shared owner, delete superseded paths, run affected checks
   and preserve the exact output comparison.
7. [ ] Obtain fresh review of the exact final diff, integrate with current main
   (including its registered named-layout changes), update shared guides, and
   deliver one atomic #693 commit. Push and close only after final verification.

The accepted contract treats defaults as authored values checked against the
complete effective contract. Existing `Default` rejects union and null defaults,
while structured values are admitted. Supporting admitted object values through
the shared target representation is the preferred direction; blanket rejection
would discard currently authored intent. No new public policy is proposed here.

Initial source evidence: `codegen/defaultValueLiteral` handles arrays/maps, then
uses `%#v` for other values. Its callers already own when defaults apply. Current
`expr` value admission resolves required defaults recursively; the original issue
predates that owner and must be reproduced against this parent before assuming
its entire report still applies. The reproduced results and remaining acceptance obligations are recorded below.

## Readiness findings

At the parent revision, the eight accepted object-default shapes (anonymous, named, array elements, map
values, plain struct source, nullable object, field metadata, nested named
container) fail generated HTTP compilation. Both scalar and typed-array controls
compile. The shared transform independently fails for Native, Optional and
Nullable because source map literals are assigned to a target struct pointer or
passed to `SetValue` expecting a concrete struct. These are representations of
the same admitted value, not three transport policies.

The unknown-key allegation is narrower than initially assumed. The initial
unknown-key test lacked required `x`, so it established only missing-required
rejection. With `x` supplied, an open object admits and retains `zzz`; an explicit
`openapi:additionalProperties=false` rejects it. The separate runtime JSON target
probe resolves `{x:1,extra:2}` but emits `{x:1}` under `PreserveAdditional=false`.
That is allowed target observation, not a proof that the complete authored
default can be materialized in a generated struct.

Recommendation: retain ordinary open-object admission, but reject a required
native default that has a supplied member with no target storage, with a clear
member-path generation diagnostic. Do not silently drop it and do not globally
close object schemas. This follows the accepted preserve-authored-intent rule;
no new public policy choice is needed. A custom representation may have an owner
that can retain the value, and must be handled by that existing boundary rather
than by reflecting the builtin object's guessed fields.

A distinct anonymous nested-object compile failure occurs with **no default**:
an unwrapped Optional value struct is compared against nil in its validator.
The durable `codegen/tla/default_representation/anonymous_nested_probe.go.txt`
is separate evidence, tracked as [#803](https://github.com/CaliLuke/loom/issues/803).
The #693 nested test uses a named container so fixing the
literal owner does not implicitly absorb that validator bug.

## Accepted owner and implementation sequence

1. At the shared transform/default boundary, retain the effective semantic
   occurrence and resolve its selected default with `expr.ValueContext` once.
   Consume `ResolvedValue` identities, declared names, scalar normalization and
   presence. Do not use raw map iteration to match struct fields; do not add
   another map-key, wire-alias, struct-reflection or constraint policy.
2. Lower the resolved tree into its actual Go target using the existing
   `AttributeContext`, `Attributor` and presence helpers. Use allocated type and
   field names (including metadata/package placement), concrete wrapper values,
   native object/primitive pointers and registered named-type field layouts.
   Select a common recursive literal path for objects nested in collections.
   Preserve current scalar/Any/custom representation paths unless the new owner
   can establish equivalent semantics; never invoke an opaque codec to infer a
   builtin struct layout. Do not convert a target through serialized JSON.
3. Check lossless default representability before emission. Unknown supplied
   members without storage require an error; unresolved custom storage must stay
   with its explicit representation owner. Reuse existing error propagation and
   contextual diagnostics, rather than generating malformed Go or suppressing a
   default. Ordinary selected-body/example projection remains unchanged.
4. Keep the existing default-application gates in `go_transform.go` and
   `go_transform_presence.go`: this ticket changes the chosen value's physical
   literal, not when an existing value is treated as absent. Preserve #692's
   32-case gate matrix, required fields, pointer targets, and Nullable null.
   Within an authored object default, preserve supplied zero/empty values and
   use the same owning default rule for genuinely absent nested fields. The
   separate M7 scalar zero-as-absence repair must not be hidden in this change.
5. Add the missing direct/generation acceptance cases below, review the exact
   final diff independently, integrate and deliver one atomic #693 commit.

The model compares legacy source literals, a shallow root-only fix and recursive
physical lowering. Legacy and shallow modes both fail `Typed` (10 states); typed
lowering passes (15). The abstraction makes no universal Go compiler claim.
The existing value-contract Lean README expressly excludes generated Go typing;
the existing Lean contracts are unchanged by this native-target supplement.

## Bounded final acceptance rule

| Obligation | Selected evidence |
| --- | --- |
| Actual source admission | Closed extras, invalid/required/constrained fields; open extras retained before target checks; plain struct source; alias/field metadata; existing codec snapshot controls |
| Physical literals | Native/Optional/Nullable compiled shared transforms; anonymous and named objects; primitive pointers, explicit zero, metadata names/package location, nested objects/arrays/maps, nullable children, omitted child with its own default |
| Existing semantics | #692 typed collection literal/runtime and 32-case application gates; Any custom representation and Bytes metadata/default controls; null/union default rejection remains unchanged |
| Runtime | Carrier test verifies absent default, explicit zero/override and nullable null; generated HTTP mounted handler receives expected values; JSON-RPC request decoder/service sees same semantics; gRPC native conversion compiles and exercises default-capable field path without changing proto scalar presence policy |
| Rejection | Extra authored member without native target slot gives a clear error before malformed Go; valid declared defaults still generate; explicit custom representation controls remain at their owner |
| Output and build | Compare parent/candidate and two independent candidate processes for one representative nested-object transport specimen per HTTP/JSON-RPC/gRPC, plus one scalar/typed-collection control; build/vet candidate modules, record all intended file changes; do not run the exported-design corpus |
| Checked-in fixtures | Regenerate HTTP and JSON-RPC ticktock fixtures against base/candidate; expect no object-default-related changes since these designs do not exercise this repair; distinguish any pre-existing regeneration drift |
| Schemas | Object default values/field names remain consistent with generated values; if rendered specs change, inspect exact changes and run existing libopenapi/Vacuum gates on the selected specimens |
| Models | Repeat the two expected failures and passing typed model; maintain implementation correspondence, assumptions and excluded custom/validation boundaries |

Inspecting `loom-mcp`'s actual checked-in design files found scalar defaults
(assistant pagination 50/10), not object defaults; those are useful unchanged
consumer controls. The existing framework `codegen/testdata/validation_types_dsl.go`
also contains a plain-struct default on a named object. Reuse it where practical
for final source/output coverage; do not add application-specific exceptions.

Readiness checks use Go 1.27.2, parent `30cabf6338a6e1aa0a98b5d4f89e63f2df76d27d`,
and explicit `LOOM_DIR` to this worktree. External modules pin the pushed
`v1.10.0-alpha.9.0.20261012214204-66a9dc9090a0` before local replacement.
Raw readiness logs are `/tmp/loom-693-final-readiness.log`,
`/tmp/loom-693-carriers.log` (final tagged carrier harness),
`/tmp/loom-693-boundary.log`, `/tmp/loom-693-model-{legacy,shallow,typed}.log`.
The readiness checkpoint was independently approved before production edits.

## Implementation and final evidence

`AttributeContext.DefaultValue(att, authored)` is the shared physical owner. It
resolves the authored input under the complete effective occurrence, traverses
resolved member identities and emits the context's allocated type/field names,
pointers and presence carriers. HTTP/shared transforms and gRPC use that owner;
the previous duplicate array/map/default renderers are removed. Callers retain
their existing application gates. Missing nested fields can use their own
defaults; explicitly supplied zero, empty and nullable values remain present.
An active expansion-path guard rejects infinitely recursive nested defaults while
allowing finite reuse of the same child type in separate branches.

`ValueResult.RepresentationValue` exposes a structurally retained tree, including
opaque leaves only when independently checkable siblings have no failure.
`ResolvedValue.SourceValue` serves only the explicit custom/raw-JSON target
boundary: builtin containers are copied; opaque values keep `SupplyValue`'s
borrowing contract. The renderer does not serialize or reflect a custom value to
guess builtin fields. The selected raw-JSON codec runs once and reports errors.
Neither accessor changes design admission or ordinary wire observation.

Final commands use explicit local source and Go 1.27.2:

```sh
LOOM_DIR="$PWD" go test ./expr ./codegen ./dsl
LOOM_DIR="$PWD" go test ./expr ./codegen ./http/codegen ./jsonrpc/codegen ./grpc/codegen -run '^TestObjectDefault|^TestResolvedRepresentationBoundary$|^TestTypedCollectionDefault|^TestTransformAnyDefault|^TestBytesDefaultTransformLayouts$|^TestDefaultValueStorageMetadata$|^TestProtoBufTransform|^TestClientTypes$|^TestServerTypes$' -count=1
"$(go env GOPATH)/bin/golangci-lint" run ./codegen/... ./expr/... ./grpc/codegen/... ./http/codegen/... ./jsonrpc/codegen/...
git diff --check
```

All pass. The first command covers complete affected semantic/shared packages;
the second covers the bounded transport and golden surfaces. Generated carrier,
HTTP and JSON-RPC modules run `go test -race ./...` and `go vet ./...`. The gRPC
module runs its actual runtime harness with `go test -race .`, then builds and
vets every generated package; the generated contract-test skeleton has no user
scenarios and is not treated as an executable application suite.

The comparison harness uses exact parent `30cabf633`, candidate, and two
independent candidate generator processes with the same command basename and
identical designs. The nested object specimen combines field/package metadata,
child defaults, objects in arrays/maps, and HTTP/JSON-RPC Nullable. Each candidate
module builds and vets. All three parent object modules fail map-to-struct
compilation; the parent typed-collection control builds and vets.

Only these eight generated files differ:

- HTTP `gen/http/defaults/{client/cli.go,client/types.go,server/types.go}`.
- JSON-RPC `gen/jsonrpc/defaults/{client/cli.go,client/types.go,server/types.go}`.
- gRPC `gen/grpc/defaults/{client/cli.go,server/types.go}`.

Every difference replaces a source map/struct literal with the allocated native
object literal, recursively including collection elements, renamed `Count`, the
omitted child `Label` default, and concrete Nullable values. There are no type
schema, OpenAPI JSON/YAML, protobuf or application-gate changes. The scalar/typed
collection control is byte-identical. All 65 generated files across the four
specimens are byte-identical between independent candidate processes. HTTP and
JSON-RPC ticktock regeneration respectively produces 18 and 17 files, identical
between parent/candidate, with both copied fixture suites passing. No checked-in
`gen/` file is edited. Unchanged schema bytes do not require a new schema gate.

An actual CLI negative design with `value={x:1,extra:2}` passes open-object source
admission, then exits with the existing generator error path:
`service defaults, method send: build HTTP request body transform: default.extra:
authored default member has no native Go field`. The error propagates through
all three direct carrier transforms. No member is silently discarded and no
ordinary wire object is made closed.

Model results: legacy and shallow renderers fail `Typed` after 10 states; recursive
physical lowering passes 15. Blind nested-default expansion violates finite paths
after 7 states; the active-path guard passes both finite/shared and cyclic recipes
in 10 states. See the model README for abstraction limits and correspondence.

Separate no-default failures are preserved as durable probes: anonymous Optional
validation is #803; gRPC Nullable object conversion compares the carrier to nil
and accesses missing fields. The gRPC specimen deliberately excludes Nullable
because that independent conversion failure occurs before default semantics can
be observed. HTTP/JSON-RPC and the shared compiled carrier cover Nullable here.

Raw local evidence (temporary, remove after review):
`/tmp/loom-693-final-packages.log`, `/tmp/loom-693-affected.log`,
`/tmp/loom-693-final-literals.log`, `/tmp/loom-693-final-transports.log`,
`/tmp/loom-693-final-lint.log`,
`/tmp/loom-693-comparison/{report.json,*-objects-*.log}`,
`/tmp/loom-693-fixtures/report.json`, `/tmp/loom-693-fixtures.log`,
`/tmp/loom-693-unrepresented-default.log`, and the model logs listed above plus
`/tmp/loom-693-expansion-{blind,checked}.log`. Reusable test inputs live in the
new direct tests and `http/codegen/testdata/object_defaults_dsl.go`.

## Integration documentation text

Application guidance for the shared public/consumer guide: object defaults are
checked against the full declared contract and generated in the service's native
representation, including objects in arrays/maps. Missing nested members use their
own defaults when that target applies defaults; supplied values remain explicit.
Every authored member must have declared native storage; unknown default members
fail generation with a member path. This does not change acceptance of unknown
members on incoming wire objects. Regenerate affected clients and servers; no
handwritten replacement literal or JSON round trip is required.

Framework guide: `AttributeContext.DefaultValue` owns native default
materialization from `expr`'s resolved graph and target metadata. Callers own
application gates. Keep raw/custom codecs at their existing target boundary;
never reinterpret their source as guessed builtin fields. Keep the two bounded
TLA models and regression cases aligned when changing carrier or expansion rules.


## Main integration

The independent final review approved the exact 30-file diff against
`30cabf6338a6e1aa0a98b5d4f89e63f2df76d27d`, manifest
`1221c6b2db9ab1e1deb8715e4959bc418de10b5b16ade77184f6a7f5f8db6f95`,
without findings. Integration targets main `6486dfb45`. Three-way merges preserve
uniform-JSON registered layout handling in `codegen/go_transform.go` and the
separately delivered #803 validation carrier in `codegen/validation_render.go`.
Other production bytes match the reviewed candidate. Public/consumer guidance
clarifies that this changes literal representation, not default application gates.
#803 is delivered separately as `090427558`; #804 gRPC nullable conversion remains
open and excluded. Focused integration verification and exact re-review follow.

Integration checks pass: selected expr/defaults 0.459s, shared/defaults plus carrier
controls 14.074s, DSL 0.220s; generated defaults, anonymous validation and uniform
JSON carrier controls pass HTTP 27.471s, JSON-RPC 19.648s and gRPC 6.130s. Scoped lint
reports zero issues. Fresh comparison against main `6486dfb45` repeats 65 artifacts
exactly in separate generator processes; the same eight default-literal Go files
change, while the typed-collection control and schemas/protobuf stay unchanged.
Parent object builds/vet fail; candidate objects and controls build/vet. Repeated
candidate runs generate only; identical output is not rebuilt for determinism.
Comparison receipt: `/tmp/loom-693-integrated-comparison/report.json`. Only the
previously identified source merges and four owning-guide additions need
integration re-review; final delivery is pending that approval.
