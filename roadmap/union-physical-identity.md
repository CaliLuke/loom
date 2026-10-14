# Preserve authored union branch representation (#936)

## Contract and reproduced defect

`struct:field:type` selects a field's concrete Go representation. A primitive
union branch is also a field. Its declared type and methods must survive DSL
normalization. Union discriminator identity remains independent of physical Go
type identity: two tags may select the same authored type.

At main `9f16cc3dc167022cb575c3cf54b6367744069288`, the service-only generated
regression compiles but fails at runtime: a String branch authored as
`scalar.Text` becomes `type ValueText scalar.Text`. It is neither the authored
type nor a Stringer/TextUnmarshaler. This reproduction does not use #926 or any
transport conversion. The cause is `dsl.appendUnionBranch`, which synthesizes a
nominal type for every inline branch, including explicitly represented scalar
fields. Generic conversion correctly rejects the subsequently missing methods.

## Owner and design

The DSL promotion boundary should retain an explicitly represented primitive
branch directly. Its metadata, validations, default, wire tag and logical type
remain on the original AttributeExpr. Existing NameScope helpers already resolve
primitive metadata to the authored type, and union emission already supports
primitive branches without a synthesized user type.

Keep promotion for uncustomized primitives and anonymous structural branches,
which still require existing generated identity and references. Keep authored
user types unchanged. Do not synthesize Go forwarding methods or make
transport-specific representation choices. Do not change named DSL Type
semantics, primitive codecs, nullability admission or wire tags.

Alternatives rejected: defining an alias only in service emission would leave
normalization and other reference consumers believing in a generated nominal
identity; suppressing generic capability errors would lose authored behavior.
A general no-promotion change would alter unrelated native generated APIs.

This is a static representation rule with no lifecycle or concurrent state.
Existing nominal-identity/cycle tests cover recursive structural promotion; a
new formal state model would add no independent ordering invariant here.

## Sequence and acceptance

- [x] Reproduce service-level identity/method loss on published main without #926.
- [x] Review the bounded owner change before implementation.
- [x] Add direct DSL coverage for custom String/Int/Bytes primitives, shared
      authored type under distinct tags, retained metadata/constraints/default,
      native promotion and recursive inline object controls.
- [x] Make the promotion boundary preserve custom primitive occurrences.
- [x] Extend generated service acceptance for struct and defined-string custom
      types, pointer storage, text methods, native controls and branch tags;
      compile projected ResultType views and mount the generated HTTP client/server.
- [x] Compare parent/candidate and independent-process generated artifacts.
      Intended differences are removal of unnecessary custom primitive nominal
      declarations and direct authored references; native artifacts stay exact.
- [x] Layer the repair over frozen #926 in a separate source copy and run the
      original custom gRPC union conversion witness, including parser error and
      publication round trip. Do not edit the #926 delivery source.
- [x] Run affected DSL/service checks and record regeneration effects.
- [ ] Obtain fresh independent review of the frozen diff before one atomic commit.

## Untagged admission boundary

Untagged unions admit only named object/array branches and reject custom Go
representation overrides inside those branches. Both rules live in
`expr/untagged_validation.go`. Rejected scalar and object-leaf custom String
controls preserve this boundary, including authored constraints/defaults.
`unionBranchValidationCode` therefore still receives admitted UserType branches;
there is no need to generalize it or silently introduce opaque untagged codecs.

## Any admission boundary

`expr.AllowsNull` makes Any intrinsically null-admitting, including occurrences
with `openapi:nullable` set to false. `validatePresence` therefore rejects custom
field representation metadata for Any. Direct service/view native Any emission
and rejected custom Any DSL controls preserve this contract. A hand-built raw
Any expression bypassing admission is not evidence for expanding this repair;
`primitiveAliasGoType` remains unchanged.

## Final bounded evidence

The delivery parent is `194ab48c693e41084d828aea794f97676a0bc03b` (#926).
Go 1.27.2 with GOEXPERIMENT=jsonv2 was used. The original service-only method-set
failure also reproduces on `9f16cc3dc167022cb575c3cf54b6367744069288`, before #926.

- Direct DSL tests cover String/Int/Bytes metadata, duplicate physical types with
  distinct discriminator tags, native promotion, constraints/defaults, and
  rejected Any/untagged contracts. Existing recursive inline-object and union
  identity/collision tests pass (DSL 2.700s, expr 1.569s, service 11.336s).
- The final generated fixture covers native String plus struct, defined-string
  and pointer-valued custom String. Each design generates a service, a projected
  ResultType view, HTTP server and client. All compile and run with the race
  detector (17.740s); custom identity, methods, parser errors, JSON round trips,
  view construction and an actual mounted HTTP round trip pass. Native and
  custom tags remain distinct. Final rejected Any/native Any checks pass 1.279s.
- The original raw-primitive gRPC witness now tests both native and custom
  generated modules, empty/nonempty request-to-response conversion and authored
  sentinel parser failure. It passes on the delivery parent plus this repair
  (7.691s). The earlier frozen-#926 overlay was checked independently as well.
- Four representative designs produced 44 files, byte-identical across two
  independent generation processes. All 11 native files match the parent. For
  each custom shape, exactly six files change: service, views, HTTP client codec
  and types, HTTP server codec and types. Unnecessary nominal declarations are
  removed, fields/constructors use authored types, and same-storage conversions
  become direct copies with no artificial parsing/error channel. Empty
  validators for removed nominal declarations disappear. No discriminator or
  unrelated native wire contract changes. The prior generated custom view fails
  compilation at its artificial fallible projection; the corrected one passes.
- Targeted golangci-lint over DSL, service and gRPC codegen reports zero issues.
  No checked-in generated golden contains these custom primitive union shapes;
  the existing union controls and four explicit generated designs bound this
  change. Regeneration removes the unnecessary custom branch type names and may
  remove error results from conversion constructors that now only copy values.

Exact hashes, generated byte differences, reproduction inputs and the final
independent verdict are preserved in the delivery receipt. No raw logs or
compiled/generated temporary outputs belong in this repository.
