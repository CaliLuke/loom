# Object default representation (#693)

The expression value owner resolves authored object defaults into declared member
identities. A map or plain struct input does not determine generated Go storage.
The former `defaultValueLiteral` handled arrays/maps but used `%#v` for objects,
putting a source map into a struct pointer or Nullable value. The shared
`AttributeContext.DefaultValue` now materializes the resolved tree using the
actual target context; HTTP/shared and gRPC callers retain their application gates.

`DefaultRepresentation.tla` models seven finite resolved trees, flattened by
path. Each node has a physical carrier and retained value. Cases cover pointer
fields, named object/field metadata, arrays/maps of objects, Nullable storage,
nested objects, unrepresentable extras and a custom codec boundary. The former
renderer and a root-only correction violate `Typed` in 10 states. Recursive
physical lowering passes all three invariants in 15 states. This supports a
shared recursive owner rather than shallow casts or transport-specific fixes.

The builtin route rejects unrepresented extras and does not guess storage for
opaque codecs. `rejected` in this model does not remove custom codec support;
the implementation delegates such values to their existing explicit target
representation. Ordinary incoming unknown-field acceptance is outside this model.
`NoSilentLoss` forbids silently dropping an authored default member to compile.

`DefaultExpansion.tla` models absent child defaults on a small type graph. A
shared child appears twice in a finite recipe; a second recipe cycles A→B→A.
Blind expansion violates `FinitePath` after 7 states. Checking the current active
expansion path passes `FinitePath` and `CorrectOutcome` in 10 states. This led to
`defaultRenderer.active`: mark an absent member occurrence only while expanding
its default and remove it on return. A global visited set would incorrectly
reject the finite shared-child case. This model covers recursive default recipes,
not arbitrary Go source graphs or an unbounded termination proof.

Run from this directory with Java 25 and TLA+ tools release 1.7.4 (JAR reports
TLC 2.19), SHA256
`936a262061c914694dfd669a543be24573c45d5aa0ff20a8b96b23d01e050e88`:

```sh
java -XX:+UseParallelGC -cp "$TLA_JAR" tlc2.TLC -workers 1 -config legacy.cfg -metadir /tmp/loom-693-tlc-legacy DefaultRepresentation.tla
java -XX:+UseParallelGC -cp "$TLA_JAR" tlc2.TLC -workers 1 -config shallow.cfg -metadir /tmp/loom-693-tlc-shallow DefaultRepresentation.tla
java -XX:+UseParallelGC -cp "$TLA_JAR" tlc2.TLC -workers 1 -config typed.cfg -metadir /tmp/loom-693-tlc-typed DefaultRepresentation.tla
java -XX:+UseParallelGC -cp "$TLA_JAR" tlc2.TLC -workers 1 -config expansion-blind.cfg -metadir /tmp/loom-693-expansion-blind DefaultExpansion.tla
java -XX:+UseParallelGC -cp "$TLA_JAR" tlc2.TLC -workers 1 -config expansion-checked.cfg -metadir /tmp/loom-693-expansion-checked DefaultExpansion.tla
```

Legacy/shallow must fail `Typed`; blind expansion must fail `FinitePath`. Typed
and checked modes must pass. These bounded models do not prove Go typing, source
admission, metadata/name allocation, general source immutability, application
gates or codec correctness. Carrier labels stand for facts supplied by
`AttributeContext`, `Attributor` and their presence helpers; the implementation
uses these owners rather than copying model labels.

## Correspondence and acceptance

- `expr.ValueContext.Resolve` and `ResolvedValue.Fields/Elements/Entries` own
  source interpretation, member identity and constraints. `RepresentationValue`
  retains opaque leaves only beside independently valid known data. `SourceValue`
  is for the selected custom/raw-JSON boundary; builtin containers are detached
  while opaque inputs retain the existing borrowing contract.
- `AttributeContext.DefaultValue` resolves the selected authored input then
  recursively chooses type/field names, object/primitive pointers and Optional/
  Nullable values. Registered named-type layouts come from the existing context.
  Unknown authored members without native fields report their path. The selected
  raw-JSON codec runs once and errors propagate; it never supplies guessed fields.
- `TestObjectDefaultCarrierRuntime` compiles and runs all three shared carriers,
  checking absent/default, explicit zero/override, nullable null, nested Nullable
  Any and a nested Nullable object's primitive pointer. These cases were red at
  the actual Go compile boundary before implementation.
- `TestObjectDefaultGenerated` now compiles eight formerly failing accepted HTTP
  shapes and two unchanged controls. `TestObjectDefaultsTransport` in HTTP,
  JSON-RPC and gRPC uses one located/named/renamed nested specimen and verifies
  decoded runtime values, explicit empty values and rejected invalid input.
  gRPC Nullable conversion is a separate no-default defect, excluded here.
- `TestObjectDefaultRejectsUnstoredMembers` and `TestObjectDefaultErrorPropagation`
  check no-loss diagnostics at nested paths and across all carriers.
  `TestObjectDefaultRejectsInfiniteNestedDefaults` exercises the active-path
  guard. `TestObjectDefaultRetainsCustomJSONBoundary` and
  `TestObjectDefaultReturnsCodecError` check explicit codec ownership.
- `TestObjectDefaultAdmission` covers closed extras, invalid/required/constrained
  fields; an open-object source retains its extra. `TestObjectDefaultProjectionBoundary`
  demonstrates that JSON runtime observation may drop an extra and is therefore
  not a lossless native-default projection.
- Existing `expr/lean/value_projection` proofs cover source resolution and wire
  observation under explicit assumptions and exclude arbitrary generated Go
  correctness. This native-target supplement does not change those proofs or
  claim to establish their Go extraction.

## Independent no-default probes

`anonymous_nested_probe.go.txt` preserves [#803](https://github.com/CaliLuke/loom/issues/803):
Optional unwrapping produces an anonymous struct value but validation compares it
to nil. Overlay it onto `http/codegen/anonymous_nested_probe_test.go` and run:

```sh
LOOM_DIR=<repo> go test -overlay=<overlay.json> ./http/codegen -run '^TestAnonymousNestedObjectWithoutDefaultProbe$' -count=1
```

`grpc_nullable_probe.go.txt` preserves a separate conversion failure without any
Default. The generated client compares `Nullable[Leaf]` to nil and accesses `X`
on its carrier; the server returns nil or a pointer where a value is expected.
Overlay it onto `grpc/codegen/nullable_object_probe_test.go` and run:

```sh
LOOM_DIR=<repo> go test -overlay=<overlay.json> ./grpc/codegen -run '^TestGRPCNullableObjectWithoutDefaultProbe$' -count=1
```

These are evidence for separate shared carrier/validation work, not #693 repairs.
Keep checker state, raw logs and temporary modules outside the repository.
Remove task-owned temporary outputs after independent validation and review;
retain reusable toolchains and these durable inputs/results.


## Union and opaque default contract correction

This is a standalone correction to native default admission and materialization,
not complete untagged oneOf #1014, discriminator contracts #1005, or imported
annotation preservation #1006. Imported default annotations are unchanged.

The accepted policy rejects `Default` attached directly or effectively to a
union occurrence, including aliases and late-bound Param/Header types. It keeps
ordinary branch defaults and container defaults containing valid, uniquely
selected union values. Empty containers and omitted optional union fields remain
valid. Alternatives considered were enabling whole-union defaults, banning every
container containing a union, or reselecting from JSON; all are rejected. The
first changes the DSL contract, the second removes admitted values, and the third
loses selected branch identity. Default application remains absence-only.

`UnionDefaults.tla` models finalized admission, retained selected branch and
allocated declaration identity, recursive pointer/value/Nullable storage, and
no loss of supplied payload. Its current-admission configuration reproduces the
late-binding gap in 3 states; current-renderer reproduces rejection of a valid
container in 4 states. Branch reselection, ignoring allocated identity and
value-only storage each fail in 4 states. Blanket container rejection fails in
3 states. Checked mode passes all five invariants in 31 states. The model was
run before production implementation; its initial negative control exposed an
incorrectly parenthesized action, which was fixed and all configurations rerun.

The common implementation is `expr.resolveEffectiveDefault` plus
`AttributeContext.DefaultValue`. The renderer consumes `ResolvedValue.Union`
and `ValueOccurrence.Branches`, uses `unionnames.PackageDeclarations` from the
owning scope's allocated primary, and derives pointer layout from shared field
and branch contexts. Actual service declaration allocation precedes qualified
references; raw external references do not allocate local names. Direct tests
in `expr/effective_union_default_test.go`, `dsl/union_default_contract_test.go`
and `codegen/union_default_representation_test.go` cover these correspondences.
The model abstracts branch selection and assumes the resolver has already
rejected ambiguous or invalid values; tests exercise those rejected cases.

For opaque composites, the accepted policy requires an existing materialization
capability or proof of a constructible physical Go literal. Exact typed
`url.Values` inputs and their import aliases remain supported. An unnamed
`map[string][]string` is rejected even though it can be Go-assignable to
`url.Values`: metadata alone cannot prove the opaque definition. The migration
is `Default(url.Values{...})`. Reflection here checks an already-authored Go
value for literal constructibility; it does not convert it into the target.
No package discovery or codec-based conversion is introduced. Unknown aliases,
inaccessible fields, cyclic state, custom Go formatting and nonliteral pointer
addresses fail with the default's member path. Explicit builtin aliases are
provable; existing String semantic/text and raw JSON operations keep ownership.
Scalar override behavior is outside this composite clarification.

`OpaqueDefaults.tla` supplements the original custom-boundary abstraction with
proved identity and constructibility. Current mode emits unsupported literals and violates its safety invariant in
15 states; checked mode preserves supported literals and explicit capabilities
in 20 states. This
supplement was checked after the initial literal implementation and before its
review; it does not prove reflection, Go parsing, imports or arbitrary codecs.
`TestOpaqueCompositeDefaultMaterialization` supplies direct implementation
controls. Native generated compilation and transport application require the
separate bounded acceptance specimen; these finite models cannot establish them.

All commands remain opt-in design tools. From this directory:

```sh
for cfg in union-current-admission union-current-renderer union-reselect union-unqualified union-value-only union-reject-containers union-checked; do
  java -XX:+UseParallelGC -cp "$TLA_JAR" tlc2.TLC -workers 1 -config "$cfg.cfg" -metadir "/tmp/loom-defaults-$cfg" UnionDefaults.tla
done
for cfg in opaque-current opaque-checked; do
  java -XX:+UseParallelGC -cp "$TLA_JAR" tlc2.TLC -workers 1 -config "$cfg.cfg" -metadir "/tmp/loom-defaults-$cfg" OpaqueDefaults.tla
done
```

Only the `checked` configurations should pass; the others are negative controls.
Remove those temporary state directories after inspection. No proof execution
or installation is part of normal tests, hooks, CI or release eligibility.


### Review revision: structural identity and diagnostic ownership

The first reviewed opaque implementation only established an imported qualifier
when the source root itself was named. F1 showed that `[]url.Values`,
`map[string]url.Values`, `[1]url.Values` and their declared aliases are already
provable physical types, including empty/nil containers. The identity owner now
walks explicit structural source components and checks every named component
against the declared import path. Source and target type syntax are then compared
using that same qualifier proof; native literal rewriting consumes the result.
This does not infer an opaque named target from an unnamed source. Tests cover
populated, empty, nil, alias and mismatched-import cases. `OpaqueDefaults.tla`
now adds four structural cases plus a root-only negative mode. The earlier
15/20-state results above refer to the initial ten-case model; the expanded
model now reproduces unsupported emission in 19 states and root-only identity
loss in 16 states; checked mode preserves both invariants across 28 states.
Direct tests and compiled opaque literals cover imported structural types,
aliases, empty/nil containers and unsupported inputs. The finite model does not
prove Go parsing or arbitrary custom type semantics.

F2 showed that attributed analysis panics had no corresponding normal diagnostic
boundary. `DefaultDiagnostics.tla` was prepared and checked before implementing
the correction: the old boundary violates ExpectedDiagnostic in 7 states;
catching all panics violates ProgrammerPanic in 9; global preflight violates
ActualInvocation in 6. Checked mode passes all four invariants in 10 states.
The implementation labels normal `AttributeContext.DefaultValue` failures with
`DefaultValueError`; `generator.Generate` recognizes only that error chain when
an analysis caller panics. Attribution survives wrapping, while unexpected
programmer panics (including attributed ones) retain panic behavior. Existing
returned errors continue through their ordinary path. Invocation gates remain
with the transport and shared transform owners. A required or unused default
is not preflighted merely because it exists in the design.

`TestGenerateDefaultMaterializationDiagnostic` exercises an optional HTTP body
materialization and a required-field control through Generate;
`TestGeneratePreservesProgrammerPanics` covers ordinary and attributed programmer
panics. The direct generator diagnostic and programmer-panic tests pass after
the attribution correction below. Actual CLI checks report the authored
`design/design.go:12` and service/method without a panic stack; required unused
default controls succeed. The old CLI witness reproduces the expected
materialization panic and its missing source location. The shared predicate-facts
split under `expr/tla/constraint_facts` also allows the unchanged nested-union
gRPC default specimen to generate, compile and preserve runtime defaults.
HTTP/gRPC ordinary controls remain byte-identical, and 59 candidate artifacts
are byte-identical across independent generations. Query callers
must retain the error with `%w` and supply their physical
attribute to `codegen.NewError`; runtime emitted failure callbacks are separate.

```sh
java -XX:+UseParallelGC -cp "$TLA_JAR" tlc2.TLC -workers 1 -config opaque-root-only.cfg -metadir /tmp/loom-defaults-root-only OpaqueDefaults.tla
for cfg in diagnostics-current diagnostics-catch-all diagnostics-preflight diagnostics-checked; do
  java -XX:+UseParallelGC -cp "$TLA_JAR" tlc2.TLC -workers 1 -config "$cfg.cfg" -metadir "/tmp/loom-defaults-$cfg" DefaultDiagnostics.tla
done
```

The root-only and three diagnostic mutants must fail their designated invariant;
checked mode must pass. Keep execution opt-in and remove checker state afterward.

### Source attribution after materialization recovery

The exact-parent acceptance run exposed an attribution gap not represented by
this model's abstract `attributed` flag: a transport wrapped the expected error
with a synthesized body that had no DSL source. Recovery supplied the authored
field, but `NewError` retained the unlocated body. The shared error owner now
uses a located fallback only when its existing expression cannot supply a source
location. An already located owner wins even when the fallback has another
location; an unlocated fallback never replaces a nonnil owner. Existing
service/method scope, wrapped causes and expected-error identity remain intact.
`TestNewErrorRetainsUsableAttribution` covers these choices and chain identity;
the generator diagnostic and programmer-panic controls exercise the boundary.
This is a deterministic attribution selection, without a new protocol state or
public default-policy change; the existing diagnostic model remains unchanged.
