# Conversion default admission

The implemented conversion owner uses an actual
physical carrier, an admitted state, a selected default, requiredness and a
target materialization decision. Native scalars cannot encode absence. Nil
pointers/collections/bytes, absent Optional and absent Nullable can; explicit
Nullable null is supplied. Required absence is rejected before conversion and
is therefore outside the admitted conversion states. The mounted readiness
harness checks that precondition; this model does not prove validation ordering.

`zero.cfg` models the parent Go transform native-zero substitution. It fails
PreserveSupplied. `bytes-kind.cfg` models protobuf primitive-kind classification
missing bytes nil absence and fails FillAbsence. `nil-only.cfg` and
`target-only.cfg` are rejected partial repairs: omitting wrapper states loses
Optional/Nullable defaults, and consulting only target materialization replaces
supplied values. `carrier.cfg` uses source-carrier absence and checks all three
invariants over 280 states. Deferred/defaultless conversions preserve input.

The input carrier is already the physical representation selected by the context
and any custom metadata; semantic Bytes alone cannot classify an opaque custom
replacement. Primitive aliases retain their physical facts. The model does not
prove metadata classification, generated Go compilation, helper closure, view
selection, recursive values, or wire codecs. Those are implementation/test
obligations. No JSON omission or protobuf collection carrier is modeled here.

## Implementation correspondence

- `codegen/go_transform_default_application.go:TransformAttrs.DefaultForField`
  combines selected target default, actual source carrier and target timing.
  `defaultSourceUsesNil` consults physical metadata before semantic kinds.
- `codegen/go_transform.go:transformObjectDefaultValueCode` uses that record only
  for a native nil-capable source; it emits no native-zero fallback.
- `codegen/go_transform_presence.go` consumes the same record for Optional/Nullable
  absence while retaining supplied/null conversion.
- `grpc/codegen/protobuf_transform.go:applyObjectFieldDefaults` consumes the same
  record, including protobuf bytes' nil absence. Requiredness is admitted earlier.

Direct correspondence: `codegen/go_transform_default_application_test.go` and
`go_transform_default_contract_test.go`, plus compiled bytes/metadata/Nullable
controls. `grpc/codegen/default_application_test.go` checks actual protobuf
request/response byte serialization. `codegen/generator/default_application_generated_test.go`
checks native service values, actual CLI constructor→wire values and raw absent/
required/null admission across HTTP/JSON-RPC; scalar gRPC and view controls retain
application timing. No JSON omission or ordinary protobuf collection presence
repair is claimed.

## Run

Use Corretto25.0.4.1 and official TLA+tools1.7.4 (TLC2.19 revision5a47802).
Set `TLA_JAR` to the installed JAR; expected SHA256 is
`936a262061c914694dfd669a543be24573c45d5aa0ff20a8b96b23d01e050e88`.
From this directory, substitute each config name below:

```sh
java -XX:+UseParallelGC -cp "$TLA_JAR" tlc2.TLC -config carrier.cfg \
  -metadir /tmp/loom-default-application-model-carrier DefaultApplication.tla
```

`zero`, `bytes-kind`, `nil-only`, `target-only` must exit12 with their stated
invariant counterexample; `carrier` exits0. Parsing/evaluation failures are not
successful negative controls. Preserve these inputs and concise results; delete
checker traces and raw logs only after validation and independent review.

## Delivered CLI producer prerequisite

The conversion model requires a source that still distinguishes absence.
#838 supplies that precondition for HTTP/JSON-RPC CLI JSON inputs through a
separate decoding representation. Its canonical model and implementation
correspondence live under
[`http/codegen/tla/cli_json_input`](../../../http/codegen/tla/cli_json_input/README.md).
The delivered constructor tests are reused, and this ticket adds composed CLI
constructor→wire controls. Do not duplicate that model here or infer that a
native encoder scalar retains an absence bit.

## Destination storage during materialization

`DefaultStorage.tla` closes a gap in the original model: deciding that a default
applies does not prove that its expression fits the destination representation.
It keeps native, Optional and Nullable destinations separate, including named
wrappers and collection element storage. `materialize` is the already-admitted
`DefaultForField` result; required/deferred/pointer contexts disable it.

Run `DefaultStorage.tla` with `storage-declared.cfg` using the same explicit TLC
command above. This checks 336 states. Negative controls are:

- `storage-old.cfg`: original native-to-Optional omission fails `FillAbsence`
  after 205 states.
- `storage-old-assignment.cfg`: checks the original Optional-to-Optional bare
  default assignment independently, failing `DeclaredStorage` after 269 states.
- `storage-wrap-only.cfg`: repairing assignment alone still omits native-source
  defaults (205 states).
- `storage-inferred.cfg`: generic inference loses named wrapper storage (207).
- `storage-raw-elements.cfg`: raw default members do not fit collection slots
  with Optional storage (174).

`presenceDefaultAssignment` consumes the existing admission record and uses the
declared wrapper's SetValue method; native targets receive native expressions.
Object default expressions are pointers, whereas field wrappers own object values.
The default renderer resolves each named array/map's own layout and wraps members
in its actual collection element type. Native nil-capable sources, including
Bytes aliases, retain an absent branch before default materialization; supplied
empty collections/bytes remain supplied.

`TestPresenceDefaultMaterialization` compiles 72 cases across scalar, named scalar,
Bytes, array, map and object values, source/destination representations, and enabled,
disabled and pointer-preserving default policy. It checks absence, supplied zero or
empty values, and explicit null. `TestCollectionDefaultPresenceStorage` compiles
array/map/nested/object slots and a JSON collection alias inside a native context.
The model abstracts concrete expressions and physical type classification; these
compiled tests establish the implementation correspondence, including named types.
