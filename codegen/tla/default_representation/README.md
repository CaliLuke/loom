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
