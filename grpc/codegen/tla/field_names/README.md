# Protobuf Go field names

The Go plugin allocates field names in descriptor order. It reserves message
methods and each field's getter, appending underscores when necessary. After
the first branch of an explicit or synthetic oneof, it allocates that oneof's
name. Its historical oneof rule does not reserve the getter and can clear an
existing getter reservation. A oneof wrapper also needs a separate type name
when its field name collides with an implicit map-entry message. The plugin
emits `ProtoReflect` without reserving that method name.

The model represents Go names as a getter-prefix depth, a stem, and a suffix.
It explores all 5,040 orders of eight declarations, keeping a oneof immediately
after its first branch. These include a reserved method, a field/getter pair,
a oneof/getter collision, a map-entry wrapper collision, and `ProtoReflect`.

- `legacy.cfg` reproduces Loom's unallocated field name disagreeing with the
  plugin's field name.
- `names_getter.cfg` reproduces invalid message selectors after fixing only
  the field-name lookup.
- `names_wrapper.cfg` reproduces the mismatch between a wrapper type suffix
  and its field selector.
- `names_reflect.cfg` reproduces the unreserved `ProtoReflect` method clash.
- `checked.cfg` uses allocated selectors and wrapper suffixes and retries with
  fresh wire names when the plugin would emit conflicting selectors. It checks
  agreement, selector safety, a bound on repairs, and termination: 95,760
  generated states and 90,720 distinct states, with no errors.

Run from this directory, using Java and the TLA+ tools jar:

```sh
java -cp /path/to/tla2tools.jar tlc2.TLC -config legacy.cfg FieldNames.tla
java -cp /path/to/tla2tools.jar tlc2.TLC -config names_getter.cfg FieldNames.tla
java -cp /path/to/tla2tools.jar tlc2.TLC -config names_wrapper.cfg FieldNames.tla
java -cp /path/to/tla2tools.jar tlc2.TLC -config names_reflect.cfg FieldNames.tla
java -cp /path/to/tla2tools.jar tlc2.TLC -config checked.cfg FieldNames.tla
```

The first four runs intentionally fail their respective invariants. The
checked run must pass. A suffix is chosen only outside the finite set of
reserved names; the individual suffix search therefore terminates when
unbounded suffixes are available. The bounded model checks the subsequent wire
repair loop for this declaration set, not for every possible design.

The model abstracts string normalization, wire-name collision allocation,
synthetic-oneof spelling, field types, and the concrete `_oneof`/`_field`
repair suffixes. It does not prove compatibility with future plugin versions
or arbitrary descriptor graphs. The Go tests compare actual protoc descriptors
and `protogen` names, including synthetic optional oneofs and map entries.
Generated applications compile and exercise request/response conversions,
validation, and CLI decoding with both union branches.

```sh
LOOM_DIR="$PWD" go test ./grpc/codegen -run 'ProtoGo|GeneratedProtoGoNames' -count=1
```

Run that command from the repository root with the supported protoc and Go
plugins on `PATH`. See [the supported toolchain](../../../../docs/grpc-guide.md).
The naming oracle is the pinned `google.golang.org/protobuf/compiler/protogen`
implementation. Protoc's synthetic-oneof naming is defined in
[`GenerateSyntheticOneofs`](https://github.com/protocolbuffers/protobuf/blob/v35.1/src/google/protobuf/compiler/parser.cc).
